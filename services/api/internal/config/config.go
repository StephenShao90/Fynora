package config

import (
	"fmt"
	"os"
)

type Config struct {
	Port                       string
	DatabaseURL                string
	JWTSecret                  string
	StorageDriver              string
	LocalStorageDir            string
	AWSRegion                  string
	AWSS3Bucket                string
	OpenAIAPIKey               string
	OpenAIModel                string
	MarketProvider             string
	PlaidClientID              string
	PlaidSecret                string
	PlaidEnv                   string
	PlaidProducts              string
	PlaidCountries             string
	PlaidWebhookVerification   string
	StripeClientID             string
	StripeSecretKey            string
	StripeWebhookSecret        string
	StripeRedirectURL          string
	ProviderTokenEncryptionKey string
	RedisURL                   string
	RedisEnabled               string
	RedisTLS                   string
	OTELEnabled                string
	OTELServiceName            string
	OTELExporterOTLPEndpoint   string
	OTELEnvironment            string
	OTELExporterOTLPProtocol   string
	OTELExporterOTLPHeaders    string
	OTELSampleRatio            string
	AppEnv                     string
	AllowedOrigins             string
	MaxBodyBytes               int64
	WorkerPollMS               int
	WorkerID                   string
}

func Load() Config {
	return Config{
		Port:                       env("PORT", "8080"),
		DatabaseURL:                env("DATABASE_URL", "postgres://postgres:postgres@localhost:5433/clearflow?sslmode=disable"),
		JWTSecret:                  env("JWT_SECRET", "dev-secret"),
		StorageDriver:              env("STORAGE_DRIVER", "local"),
		LocalStorageDir:            env("LOCAL_STORAGE_DIR", "./data/raw-events"),
		AWSRegion:                  os.Getenv("AWS_REGION"),
		AWSS3Bucket:                os.Getenv("AWS_S3_BUCKET"),
		OpenAIAPIKey:               os.Getenv("OPENAI_API_KEY"),
		OpenAIModel:                env("OPENAI_MODEL", "gpt-4o-mini"),
		MarketProvider:             env("MARKET_DATA_PROVIDER", "mock"),
		PlaidClientID:              os.Getenv("PLAID_CLIENT_ID"),
		PlaidSecret:                os.Getenv("PLAID_SECRET"),
		PlaidEnv:                   env("PLAID_ENV", "sandbox"),
		PlaidProducts:              env("PLAID_PRODUCTS", "transactions"),
		PlaidCountries:             env("PLAID_COUNTRY_CODES", "US,CA"),
		PlaidWebhookVerification:   env("PLAID_WEBHOOK_VERIFICATION", "false"),
		StripeClientID:             os.Getenv("STRIPE_CLIENT_ID"),
		StripeSecretKey:            os.Getenv("STRIPE_SECRET_KEY"),
		StripeWebhookSecret:        os.Getenv("STRIPE_WEBHOOK_SECRET"),
		StripeRedirectURL:          env("STRIPE_REDIRECT_URL", "http://localhost:8080/api/v1/integrations/stripe/callback"),
		ProviderTokenEncryptionKey: os.Getenv("PROVIDER_TOKEN_ENCRYPTION_KEY"),
		RedisURL:                   env("REDIS_URL", "redis://localhost:6379/0"),
		RedisEnabled:               env("REDIS_ENABLED", "false"),
		RedisTLS:                   env("REDIS_TLS", "false"),
		OTELEnabled:                env("OTEL_ENABLED", "false"),
		OTELServiceName:            env("OTEL_SERVICE_NAME", "clearflow-api"),
		OTELExporterOTLPEndpoint:   os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
		OTELEnvironment:            env("OTEL_ENVIRONMENT", env("APP_ENV", "development")),
		OTELExporterOTLPProtocol:   env("OTEL_EXPORTER_OTLP_PROTOCOL", "grpc"),
		OTELExporterOTLPHeaders:    os.Getenv("OTEL_EXPORTER_OTLP_HEADERS"),
		OTELSampleRatio:            env("OTEL_SAMPLE_RATIO", "1.0"),
		AppEnv:                     env("APP_ENV", "development"),
		AllowedOrigins:             env("ALLOWED_ORIGINS", "*"),
		MaxBodyBytes:               envInt64("MAX_BODY_BYTES", 1<<20),
		WorkerPollMS:               int(envInt64("WORKER_POLL_MS", 10000)),
		WorkerID:                   env("WORKER_ID", ""),
	}
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envInt64(key string, fallback int64) int64 {
	if value := os.Getenv(key); value != "" {
		var out int64
		for _, ch := range value {
			if ch < '0' || ch > '9' {
				return fallback
			}
			out = out*10 + int64(ch-'0')
		}
		return out
	}
	return fallback
}

func (c Config) ValidateProduction() error {
	if c.AppEnv != "production" {
		return nil
	}
	if c.JWTSecret == "" || c.JWTSecret == "dev-secret" {
		return fmt.Errorf("JWT_SECRET must be set to a production secret")
	}
	if c.DatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL is required in production")
	}
	if c.AllowedOrigins == "" || c.AllowedOrigins == "*" {
		return fmt.Errorf("ALLOWED_ORIGINS cannot be wildcard in production")
	}
	if c.PlaidEnv == "production" && (c.PlaidClientID == "" || c.PlaidSecret == "") {
		return fmt.Errorf("Plaid production mode requires PLAID_CLIENT_ID and PLAID_SECRET")
	}
	if c.ProviderTokenEncryptionKey == "" {
		return fmt.Errorf("PROVIDER_TOKEN_ENCRYPTION_KEY is required in production")
	}
	if boolEnv(c.RedisEnabled) && c.RedisURL == "" {
		return fmt.Errorf("REDIS_URL is required when REDIS_ENABLED=true")
	}
	if boolEnv(c.OTELEnabled) {
		if c.OTELServiceName == "" {
			return fmt.Errorf("OTEL_SERVICE_NAME is required when OTEL_ENABLED=true")
		}
		if c.OTELExporterOTLPProtocol != "grpc" && c.OTELExporterOTLPProtocol != "http/protobuf" {
			return fmt.Errorf("OTEL_EXPORTER_OTLP_PROTOCOL must be grpc or http/protobuf")
		}
		if c.AppEnv == "production" && c.OTELExporterOTLPEndpoint == "" {
			return fmt.Errorf("OTEL_EXPORTER_OTLP_ENDPOINT is required in production when OTEL_ENABLED=true")
		}
	}
	return nil
}

func boolEnv(value string) bool {
	switch value {
	case "1", "true", "TRUE", "yes", "YES", "enabled", "ENABLED":
		return true
	default:
		return false
	}
}
