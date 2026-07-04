package config

import "os"

type Config struct {
	Port            string
	DatabaseURL     string
	JWTSecret       string
	StorageDriver   string
	LocalStorageDir string
	AWSRegion       string
	AWSS3Bucket     string
	OpenAIAPIKey    string
	OpenAIModel     string
	MarketProvider  string
	AppEnv          string
}

func Load() Config {
	return Config{
		Port:            env("PORT", "8080"),
		DatabaseURL:     env("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/fynora?sslmode=disable"),
		JWTSecret:       env("JWT_SECRET", "dev-secret"),
		StorageDriver:   env("STORAGE_DRIVER", "local"),
		LocalStorageDir: env("LOCAL_STORAGE_DIR", "./data/raw-events"),
		AWSRegion:       os.Getenv("AWS_REGION"),
		AWSS3Bucket:     os.Getenv("AWS_S3_BUCKET"),
		OpenAIAPIKey:    os.Getenv("OPENAI_API_KEY"),
		OpenAIModel:     env("OPENAI_MODEL", "gpt-4o-mini"),
		MarketProvider:  env("MARKET_DATA_PROVIDER", "mock"),
		AppEnv:          env("APP_ENV", "development"),
	}
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
