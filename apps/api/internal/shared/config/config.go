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

	// AppTokenTTL is how long an external payment App's access token (Fase
	// 10, `POST /auth/app/token`) is valid — no refresh token exists for
	// this principal type, it simply re-exchanges appId+secret once expired.
	AppTokenTTL time.Duration

	S3Endpoint  string
	S3Bucket    string
	S3AccessKey string
	S3SecretKey string
	S3UseSSL    bool
	S3Region    string
	// S3Presign: hand browsers direct, short-lived file links (the storage
	// endpoint must be reachable from them).
	S3Presign bool

	// PaymentEncryptionKey derives the AES-256-GCM key `payment` uses to
	// encrypt gateway credentials at rest — see MODULE_PAYMENT.md §8.
	PaymentEncryptionKey string

	// PaymentReconcileInterval is how often `payment` re-checks charges whose
	// webhook was never received against the gateway directly — the safety
	// net described in knowledge/MODULE_PAYMENT.md.
	PaymentReconcileInterval time.Duration

	// SubscriptionReconcileInterval is how often `billing`'s own reconciler
	// (A16) re-checks subscription intents whose webhook was never received —
	// separate from PaymentReconcileInterval since it's a different
	// reconciler polling a different table (subscription_app_charges, not
	// payment_charge_dispatch). See PLAN.md A32: the two reconcilers call
	// the gateway independently by design.
	SubscriptionReconcileInterval time.Duration
	// SubscriptionChargeMaxAge is how long a subscription charge can stay
	// `unpaid` before the reconciler force-resolves it as `expired`.
	SubscriptionChargeMaxAge time.Duration

	// PurgeInterval is how often the server permanently removes what has stayed
	// deleted (payments, vendor engagements, clients) for longer than 24 hours
	// (ADR-0034). 0 switches the in-process ticker off (the `api purge` command
	// still works, e.g. from cron).
	PurgeInterval time.Duration

	// GoogleClientID is the OAuth web client ID for Sign in with Google
	// (public: it is also shipped to the browser via /public/config).
	GoogleClientID string
	// GoogleJWKSURL overrides Google's signing-key URL. Only honoured
	// outside production (end-to-end tests sign their own tokens).
	GoogleJWKSURL string
}

// DefaultGoogleClientID is ElProof's own OAuth web client (decision K9). It
// is public by design (Google Identity Services shows it in every page), so
// it ships as the default; GOOGLE_CLIENT_ID overrides it, e.g. for a
// staging client with different authorised origins.
const DefaultGoogleClientID = "145222647838-e9nhkpkaoiupcd0ume68qg2bqmlhcq58.apps.googleusercontent.com"

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
		AppTokenTTL:   getDuration("APP_TOKEN_TTL", time.Hour),

		S3Endpoint:  getEnv("S3_ENDPOINT", ""),
		S3Bucket:    getEnv("S3_BUCKET", ""),
		S3AccessKey: getEnv("S3_ACCESS_KEY", ""),
		S3SecretKey: getEnv("S3_SECRET_KEY", ""),
		S3UseSSL:    getEnv("S3_USE_SSL", "true") == "true",
		S3Region:    getEnv("S3_REGION", ""),
		S3Presign:   getEnv("S3_PRESIGN", "true") == "true",

		PaymentEncryptionKey:     getEnv("PAYMENT_ENCRYPTION_KEY", ""),
		PaymentReconcileInterval: getDuration("PAYMENT_RECONCILE_INTERVAL", 3*time.Minute),

		SubscriptionReconcileInterval: getDuration("SUBSCRIPTION_RECONCILE_INTERVAL", 3*time.Minute),
		SubscriptionChargeMaxAge:      getDuration("SUBSCRIPTION_CHARGE_MAX_AGE", 24*time.Hour),

		PurgeInterval: getDuration("PURGE_INTERVAL", 30*time.Minute),

		GoogleClientID: getEnv("GOOGLE_CLIENT_ID", DefaultGoogleClientID),
		GoogleJWKSURL:  getEnv("GOOGLE_JWKS_URL", ""),
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
