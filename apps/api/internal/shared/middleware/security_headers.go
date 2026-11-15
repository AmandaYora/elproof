package middleware

import (
	"net"
	"net/http"
	"net/url"
	"strings"
)

// SecurityHeadersConfig feeds SecurityHeaders. Everything that differs per
// deployment comes from configuration, so the policy always names the real
// object-storage host.
type SecurityHeadersConfig struct {
	// Development: local http, so no Strict-Transport-Security.
	Development bool
	// StorageEndpoint / StorageUseSSL: the S3 endpoint (host[:port]) that
	// presigned file links point at, when browsers get direct links.
	StorageEndpoint string
	StorageUseSSL   bool
	StoragePresign  bool
}

// Google Identity Services (Sign in with Google, ADR-0024): the sources its
// documentation lists for a Content-Security-Policy.
const (
	gsiScript = "https://accounts.google.com/gsi/client"
	gsiStyle  = "https://accounts.google.com/gsi/style"
	gsiOrigin = "https://accounts.google.com/gsi/"
	// The payment gateway's QRIS image is shown on the subscription page.
	tripayOrigin = "https://tripay.co.id"
)

// APIContentSecurityPolicy is sent with every /api/ response: JSON and file
// streams never need to load anything or be framed.
const APIContentSecurityPolicy = "default-src 'none'; frame-ancestors 'none'"

// storageSources are the CSP sources for presigned object-storage links:
// the endpoint itself (path-style URLs) and, for a DNS name, its
// subdomains (virtual-hosted-style URLs, bucket.endpoint).
func storageSources(cfg SecurityHeadersConfig) []string {
	endpoint := strings.TrimSpace(cfg.StorageEndpoint)
	if !cfg.StoragePresign || endpoint == "" {
		return nil
	}
	scheme := "http"
	if cfg.StorageUseSSL {
		scheme = "https"
	}
	u, err := url.Parse(scheme + "://" + endpoint)
	if err != nil || u.Host == "" || strings.ContainsAny(u.Host, " ;,'\"") {
		return nil
	}
	sources := []string{scheme + "://" + u.Host}
	host := u.Hostname()
	if net.ParseIP(host) == nil && strings.Contains(host, ".") {
		wildcard := "*." + host
		if p := u.Port(); p != "" {
			wildcard += ":" + p
		}
		sources = append(sources, scheme+"://"+wildcard)
	}
	return sources
}

// AppContentSecurityPolicy is the policy for the single-page app and its
// static files. Why each source is there:
//
//   - script-src: the app's own bundles plus Google's sign-in script; no
//     inline scripts and no eval.
//   - style-src 'unsafe-inline': the UI kit (Base UI, the toaster) injects
//     <style> elements at runtime; CSS cannot run script, and 'unsafe-inline'
//     never applies to scripts.
//   - img-src data: blob:: generated QR codes and previews of picked or
//     compressed images (canvas), and files fetched through the API as
//     blobs (logos, proof images, login photos).
//   - frame-src blob:: PDF proof shown in an iframe from a blob URL.
//   - storage sources (img/frame/connect): presigned links to view a file
//     and to upload one directly (PUT).
//   - worker-src 'self': the Client Portal's service worker (/sw.js).
func AppContentSecurityPolicy(cfg SecurityHeadersConfig) string {
	storage := strings.Join(storageSources(cfg), " ")
	join := func(parts ...string) string {
		out := make([]string, 0, len(parts))
		for _, p := range parts {
			if p != "" {
				out = append(out, p)
			}
		}
		return strings.Join(out, " ")
	}
	directives := []string{
		"default-src 'self'",
		join("script-src 'self'", gsiScript),
		join("style-src 'self' 'unsafe-inline'", gsiStyle),
		join("img-src 'self' data: blob:", tripayOrigin, storage),
		"font-src 'self' data:",
		join("connect-src 'self'", gsiOrigin, storage),
		join("frame-src 'self' blob:", gsiOrigin, storage),
		"media-src 'self' blob:",
		"worker-src 'self'",
		"manifest-src 'self'",
		"object-src 'none'",
		"base-uri 'self'",
		"form-action 'self'",
		"frame-ancestors 'none'",
	}
	return strings.Join(directives, "; ")
}

// PermissionsPolicy turns off powerful features ElProof never uses (photos
// are picked with a file input, which needs none of them).
const PermissionsPolicy = "camera=(), microphone=(), geolocation=(), payment=(), usb=(), serial=(), bluetooth=(), browsing-topics=()"

// SecurityHeaders sets the browser security headers on every response
// (EP-08): no MIME sniffing, no framing, a strict referrer, a
// Content-Security-Policy (strict for the API, fitted to the app for
// everything else) and, outside development, HSTS. The policy strings are
// built once.
func SecurityHeaders(cfg SecurityHeadersConfig) func(http.Handler) http.Handler {
	appPolicy := AppContentSecurityPolicy(cfg)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()
			h.Set("X-Content-Type-Options", "nosniff")
			h.Set("X-Frame-Options", "DENY")
			h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
			h.Set("Permissions-Policy", PermissionsPolicy)
			h.Set("Cross-Origin-Opener-Policy", "same-origin-allow-popups")
			if strings.HasPrefix(r.URL.Path, "/api/") {
				h.Set("Content-Security-Policy", APIContentSecurityPolicy)
			} else {
				h.Set("Content-Security-Policy", appPolicy)
			}
			if !cfg.Development {
				h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			}
			next.ServeHTTP(w, r)
		})
	}
}
