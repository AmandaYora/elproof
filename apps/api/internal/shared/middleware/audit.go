package middleware

import (
	"context"
	"net/http"

	"elproof/internal/shared/logger"
)

// AuditRecord is what AuditTrail hands to the recorder. It deliberately has
// no request or response body, so no password, secret or key is ever logged.
type AuditRecord struct {
	ActorType  string
	ActorID    string
	ActorRole  string
	Method     string
	Path       string
	StatusCode int
	IP         string
	UserAgent  string
}

// AuditRecorder persists audit records (implemented by the audit module).
type AuditRecorder interface {
	Record(ctx context.Context, rec AuditRecord) error
}

// AuditTrail records every state-changing request (anything but GET, HEAD and
// OPTIONS) made by one of the given principal types — successful or not,
// since a refused attempt is as interesting as an applied change. Must run
// after RequireAuth (it reads the claims).
func AuditTrail(recorder AuditRecorder, principalTypes ...string) func(http.Handler) http.Handler {
	audited := make(map[string]bool, len(principalTypes))
	for _, p := range principalTypes {
		audited[p] = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := FromContext(r.Context())
			if !ok || !audited[claims.PrincipalType] ||
				r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
				next.ServeHTTP(w, r)
				return
			}
			sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(sw, r)
			rec := AuditRecord{
				ActorType: claims.PrincipalType, ActorID: claims.PrincipalID, ActorRole: claims.Role,
				Method: r.Method, Path: r.URL.Path, StatusCode: sw.status,
				IP: ClientIP(r), UserAgent: r.UserAgent(),
			}
			// A fresh context: the request's may already be cancelled once
			// the response is written, and the record must still land.
			if err := recorder.Record(context.WithoutCancel(r.Context()), rec); err != nil {
				logger.Error("audit: failed to record %s %s by %s %s: %v", rec.Method, rec.Path, rec.ActorType, rec.ActorID, err)
			}
		})
	}
}

type statusWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (s *statusWriter) WriteHeader(code int) {
	if !s.wroteHeader {
		s.status, s.wroteHeader = code, true
	}
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusWriter) Write(b []byte) (int, error) {
	if !s.wroteHeader {
		s.wroteHeader = true
	}
	return s.ResponseWriter.Write(b)
}
