package config

import (
	"strings"
	"testing"
)

func productionConfig() Config {
	return Config{
		AppEnv:                     "production",
		DatabaseURL:                "postgres://postgres:postgres@localhost:5432/fynora?sslmode=disable",
		JWTSecret:                  "production-secret-with-enough-length",
		AllowedOrigins:             "https://app.example.com",
		ProviderTokenEncryptionKey: "0123456789abcdef0123456789abcdef",
		OTELExporterOTLPProtocol:   "grpc",
	}
}

func TestValidateProductionAllowsDisabledOTEL(t *testing.T) {
	cfg := productionConfig()
	cfg.OTELEnabled = "false"
	if err := cfg.ValidateProduction(); err != nil {
		t.Fatalf("expected valid production config, got %v", err)
	}
}

func TestValidateProductionRequiresOTELServiceName(t *testing.T) {
	cfg := productionConfig()
	cfg.OTELEnabled = "true"
	cfg.OTELServiceName = ""
	cfg.OTELExporterOTLPEndpoint = "localhost:4317"
	if err := cfg.ValidateProduction(); err == nil || !strings.Contains(err.Error(), "OTEL_SERVICE_NAME") {
		t.Fatalf("expected OTEL_SERVICE_NAME validation error, got %v", err)
	}
}

func TestValidateProductionRequiresOTLPEndpointWhenEnabled(t *testing.T) {
	cfg := productionConfig()
	cfg.OTELEnabled = "true"
	cfg.OTELServiceName = "clearflow-api"
	cfg.OTELExporterOTLPEndpoint = ""
	if err := cfg.ValidateProduction(); err == nil || !strings.Contains(err.Error(), "OTEL_EXPORTER_OTLP_ENDPOINT") {
		t.Fatalf("expected OTEL_EXPORTER_OTLP_ENDPOINT validation error, got %v", err)
	}
}

func TestValidateProductionRejectsUnsupportedOTLPProtocol(t *testing.T) {
	cfg := productionConfig()
	cfg.OTELEnabled = "true"
	cfg.OTELServiceName = "clearflow-api"
	cfg.OTELExporterOTLPEndpoint = "localhost:4317"
	cfg.OTELExporterOTLPProtocol = "jaeger"
	if err := cfg.ValidateProduction(); err == nil || !strings.Contains(err.Error(), "OTEL_EXPORTER_OTLP_PROTOCOL") {
		t.Fatalf("expected OTEL_EXPORTER_OTLP_PROTOCOL validation error, got %v", err)
	}
}
