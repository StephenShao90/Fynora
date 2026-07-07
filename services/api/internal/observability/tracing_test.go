package observability

import (
	"context"
	"testing"
)

func TestTracingConfigOnOff(t *testing.T) {
	off := New(false, "svc", "test")
	ctx, span := off.Start(context.Background(), "op", nil)
	if span.TraceID != "" || TraceID(ctx) != "" {
		t.Fatal("expected tracing disabled to avoid trace id")
	}
	on := New(true, "svc", "test")
	ctx, span = on.Start(context.Background(), "op", nil)
	if span.TraceID == "" || TraceID(ctx) == "" {
		t.Fatal("expected tracing enabled to create trace id")
	}
	if SpanID(ctx) == "" {
		t.Fatal("expected tracing enabled to create span id")
	}
	if err := on.Shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown failed: %v", err)
	}
}

func TestNewWithConfigRejectsInvalidProtocol(t *testing.T) {
	_, err := NewWithConfig(context.Background(), Config{
		Enabled:     true,
		ServiceName: "svc",
		Environment: "test",
		Endpoint:    "localhost:4317",
		Protocol:    "zipkin",
		SampleRatio: 1,
	})
	if err == nil {
		t.Fatal("expected invalid protocol error")
	}
}

func TestTracePropagationMapCarrier(t *testing.T) {
	tracer, err := NewWithConfig(context.Background(), Config{Enabled: true, ServiceName: "svc", Environment: "test", SampleRatio: 1})
	if err != nil {
		t.Fatalf("tracer setup failed: %v", err)
	}
	defer func() {
		if err := tracer.Shutdown(context.Background()); err != nil {
			t.Fatalf("shutdown failed: %v", err)
		}
	}()
	ctx, span := tracer.Start(context.Background(), "producer", nil)
	defer span.End()
	headers := map[string]string{}
	tracer.Inject(ctx, headers)
	if headers["traceparent"] == "" {
		t.Fatal("expected W3C traceparent header")
	}
	next := tracer.Extract(context.Background(), headers)
	next, consumer := tracer.Start(next, "consumer", nil)
	defer consumer.End()
	if TraceID(next) != span.TraceID {
		t.Fatalf("expected propagated trace id %s, got %s", span.TraceID, TraceID(next))
	}
}

func TestSampleRatioBounds(t *testing.T) {
	if got := SampleRatio("0.25"); got != 0.25 {
		t.Fatalf("expected explicit ratio, got %v", got)
	}
	for _, value := range []string{"", "0", "-1", "2", "bad"} {
		if got := SampleRatio(value); got != 1 {
			t.Fatalf("expected invalid ratio %q to default to 1, got %v", value, got)
		}
	}
}
