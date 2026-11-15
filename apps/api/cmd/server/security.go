package main

import (
	"log"
	"net/http"
	"time"

	"elproof/internal/modules/identity"
	"elproof/internal/shared/config"
	"elproof/internal/shared/middleware"
)

// maxProductionReuseGrace caps REFRESH_REUSE_GRACE outside development: a
// long grace keeps a stolen, already-rotated refresh token usable (ADR-0044).
const maxProductionReuseGrace = 2 * time.Minute

// sessionOptions configures the refresh-token cookie (EP-07, ADR-0044).
func sessionOptions(cfg config.Config) identity.SessionOptions {
	dev := cfg.AppEnv == "development"
	grace := cfg.RefreshReuseGrace
	if !dev && grace > maxProductionReuseGrace {
		log.Printf("REFRESH_REUSE_GRACE=%s is only for tests; using %s", grace, maxProductionReuseGrace)
		grace = maxProductionReuseGrace
	}
	return identity.SessionOptions{Development: dev, ReuseGrace: grace}
}

// securityHeaders wraps the whole server with the browser security headers
// (EP-08); the Content-Security-Policy names the object storage that
// presigned file links point at.
func securityHeaders(cfg config.Config) func(http.Handler) http.Handler {
	return middleware.SecurityHeaders(middleware.SecurityHeadersConfig{
		Development:     cfg.AppEnv == "development",
		StorageEndpoint: cfg.S3Endpoint,
		StorageUseSSL:   cfg.S3UseSSL,
		StoragePresign:  cfg.S3Presign,
	})
}
