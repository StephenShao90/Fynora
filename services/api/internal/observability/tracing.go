package observability

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

type contextKey struct{}

type Config struct {
	Enabled     bool
	ServiceName string
	Environment string
	Endpoint    string
	Protocol    string
	Headers     string
	SampleRatio float64
}

type Span struct {
	TraceID string
	SpanID  string
	Name    string
	Start   time.Time
	attrs   map[string]string
	span    trace.Span
}

type Tracer struct {
	Enabled     bool
	ServiceName string
	Environment string
	tracer      trace.Tracer
	provider    *sdktrace.TracerProvider
}

func New(enabled bool, serviceName, environment string) Tracer {
	t, _ := NewWithConfig(context.Background(), Config{Enabled: enabled, ServiceName: serviceName, Environment: environment, SampleRatio: 1})
	return t
}

func NewWithConfig(ctx context.Context, cfg Config) (Tracer, error) {
	if !cfg.Enabled {
		return Tracer{Enabled: false, ServiceName: cfg.ServiceName, Environment: cfg.Environment}, nil
	}
	if cfg.ServiceName == "" {
		cfg.ServiceName = "clearflow-api"
	}
	if cfg.Environment == "" {
		cfg.Environment = "development"
	}
	if cfg.Protocol == "" {
		cfg.Protocol = "grpc"
	}
	if cfg.SampleRatio <= 0 || cfg.SampleRatio > 1 {
		cfg.SampleRatio = 1
	}
	var exporter *otlptrace.Exporter
	var err error
	if cfg.Endpoint != "" {
		headers := parseHeaders(cfg.Headers)
		switch cfg.Protocol {
		case "http/protobuf":
			opts := []otlptracehttp.Option{otlptracehttp.WithEndpointURL(cfg.Endpoint)}
			if len(headers) > 0 {
				opts = append(opts, otlptracehttp.WithHeaders(headers))
			}
			exporter, err = otlptracehttp.New(ctx, opts...)
		case "grpc":
			endpoint := strings.TrimPrefix(strings.TrimPrefix(cfg.Endpoint, "http://"), "https://")
			opts := []otlptracegrpc.Option{otlptracegrpc.WithEndpoint(endpoint), otlptracegrpc.WithInsecure()}
			if strings.HasPrefix(cfg.Endpoint, "https://") {
				opts = []otlptracegrpc.Option{otlptracegrpc.WithEndpoint(endpoint)}
			}
			if len(headers) > 0 {
				opts = append(opts, otlptracegrpc.WithHeaders(headers))
			}
			exporter, err = otlptracegrpc.New(ctx, opts...)
		default:
			return Tracer{}, fmt.Errorf("unsupported OTLP protocol %s", cfg.Protocol)
		}
		if err != nil {
			return Tracer{}, err
		}
	}
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
			attribute.String("deployment.environment", cfg.Environment),
		),
	)
	if err != nil {
		return Tracer{}, err
	}
	options := []sdktrace.TracerProviderOption{
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(cfg.SampleRatio)),
	}
	if exporter != nil {
		options = append(options, sdktrace.WithBatcher(exporter))
	}
	provider := sdktrace.NewTracerProvider(options...)
	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	return Tracer{Enabled: true, ServiceName: cfg.ServiceName, Environment: cfg.Environment, tracer: provider.Tracer(cfg.ServiceName), provider: provider}, nil
}

func (t Tracer) Start(ctx context.Context, name string, attrs map[string]string) (context.Context, Span) {
	if !t.Enabled {
		return ctx, Span{}
	}
	if t.tracer == nil {
		t.tracer = otel.Tracer(firstNonEmpty(t.ServiceName, "clearflow-api"))
	}
	otelAttrs := make([]attribute.KeyValue, 0, len(attrs))
	for key, value := range attrs {
		otelAttrs = append(otelAttrs, attribute.String(key, value))
	}
	ctx, sp := t.tracer.Start(ctx, name, trace.WithAttributes(otelAttrs...))
	sc := sp.SpanContext()
	traceID := sc.TraceID().String()
	spanID := sc.SpanID().String()
	if traceID == "00000000000000000000000000000000" {
		traceID = newTraceID()
	}
	span := Span{TraceID: traceID, SpanID: spanID, Name: name, Start: time.Now().UTC(), attrs: attrs, span: sp}
	return context.WithValue(context.WithValue(ctx, contextKey{}, traceID), spanContextKey{}, spanID), span
}

func (s Span) End() {
	if s.span != nil {
		s.span.End()
	}
}

func (t Tracer) Middleware(next http.Handler, onStart func()) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !t.Enabled {
			next.ServeHTTP(w, r)
			return
		}
		if onStart != nil {
			onStart()
		}
		ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))
		ctx, span := t.Start(ctx, "http.request", map[string]string{"path": r.URL.Path, "method": r.Method})
		defer span.End()
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (t Tracer) Inject(ctx context.Context, headers map[string]string) {
	if !t.Enabled {
		return
	}
	carrier := propagation.MapCarrier(headers)
	otel.GetTextMapPropagator().Inject(ctx, carrier)
}

func (t Tracer) Extract(ctx context.Context, headers map[string]string) context.Context {
	if !t.Enabled {
		return ctx
	}
	return otel.GetTextMapPropagator().Extract(ctx, propagation.MapCarrier(headers))
}

func (t Tracer) Shutdown(ctx context.Context) error {
	if !t.Enabled || t.provider == nil {
		return nil
	}
	return t.provider.Shutdown(ctx)
}

type spanContextKey struct{}

func TraceID(ctx context.Context) string {
	if value, ok := ctx.Value(contextKey{}).(string); ok {
		return value
	}
	if sc := trace.SpanContextFromContext(ctx); sc.IsValid() {
		return sc.TraceID().String()
	}
	return ""
}

func SpanID(ctx context.Context) string {
	if value, ok := ctx.Value(spanContextKey{}).(string); ok {
		return value
	}
	if sc := trace.SpanContextFromContext(ctx); sc.IsValid() {
		return sc.SpanID().String()
	}
	return ""
}

func newTraceID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return ""
	}
	return hex.EncodeToString(buf)
}

func Enabled(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "enabled":
		return true
	default:
		return false
	}
}

func SampleRatio(value string) float64 {
	if value == "" {
		return 1
	}
	out, err := strconv.ParseFloat(value, 64)
	if err != nil || out <= 0 || out > 1 {
		return 1
	}
	return out
}

func parseHeaders(value string) map[string]string {
	out := map[string]string{}
	for _, pair := range strings.Split(value, ",") {
		key, val, ok := strings.Cut(strings.TrimSpace(pair), "=")
		if ok && key != "" {
			out[key] = val
		}
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
