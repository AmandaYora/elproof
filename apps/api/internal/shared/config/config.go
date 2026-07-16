// Package config loads environment configuration for the API process.
package config

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Config struct {
	AppEnv  string
	AppPort string
	AppName string

	DatabaseURL string

	JWTSecret     string
	JWTAccessTTL  time.Duration
	JWTRefreshTTL time.Duration

	S3Endpoint  string
	S3Bucket    string
	S3AccessKey string
	S3SecretKey string
	S3UseSSL    bool

	// PaymentEncryptionKey derives the AES-256-GCM key `payment` uses to
	// encrypt gateway credentials at rest — see MODULE_PAYMENT.md §8.
	PaymentEncryptionKey string
}

// Load reads .env from the repo root (or the current directory) into the process
// environment, without overriding variables already set by the real OS environment,
// then builds a Config from it.
func Load() Config {
	loadDotEnv(".env")
	loadDotEnv(filepath.Join("..", "..", ".env"))

	return Config{
		AppEnv:  getEnv("APP_ENV", "development"),
		AppPort: getEnv("APP_PORT", "8080"),
		AppName: getEnv("APP_NAME", "ElProof"),

		DatabaseURL: getEnv("DATABASE_URL", ""),

		JWTSecret:     getEnv("JWT_SECRET", ""),
		JWTAccessTTL:  getDuration("JWT_ACCESS_TTL", 30*time.Minute),
		JWTRefreshTTL: getDuration("JWT_REFRESH_TTL", 168*time.Hour),

		S3Endpoint:  getEnv("S3_ENDPOINT", ""),
		S3Bucket:    getEnv("S3_BUCKET", ""),
		S3AccessKey: getEnv("S3_ACCESS_KEY", ""),
		S3SecretKey: getEnv("S3_SECRET_KEY", ""),
		S3UseSSL:    getEnv("S3_USE_SSL", "true") == "true",

		PaymentEncryptionKey: getEnv("PAYMENT_ENCRYPTION_KEY", ""),
	}
}

func loadDotEnv(path string) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if _, alreadySet := os.LookupEnv(key); !alreadySet {
			_ = os.Setenv(key, value)
		}
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value
	}
	return fallback
}

func getDuration(key string, fallback time.Duration) time.Duration {
	value, ok := os.LookupEnv(key)
	if !ok || value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}
