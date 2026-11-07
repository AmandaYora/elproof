package middleware

import (
	"context"
	"net/http"
	"sync"
	"time"

	"elproof/internal/shared/response"
)

// SupportSessionValidator confirms a support session is still allowed:
// open, consented to by the Owner, run by an admin who may still do it.
// Implemented by the platform module.
type SupportSessionValidator interface {
	CheckSupportSession(ctx context.Context, tenantID, sessionID int64, adminID string, now time.Time) error
}

// RequireLiveSupportSession ends a support token the moment its session is
// ended, consent withdrawn or admin deactivated, instead of when the token
// expires. Bound late because platform is built after the authed chain.
type RequireLiveSupportSession struct {
	mu        sync.RWMutex
	validator SupportSessionValidator
}

func (l *RequireLiveSupportSession) SetValidator(v SupportSessionValidator) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.validator = v
}

func (l *RequireLiveSupportSession) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := FromContext(r.Context())
		if !ok || claims.Impersonator == "" {
			next.ServeHTTP(w, r)
			return
		}
		l.mu.RLock()
		v := l.validator
		l.mu.RUnlock()
		tenantID, tenantOK := claims.TenantIDInt()
		// A support token that cannot be checked is refused, never trusted.
		if v == nil || !tenantOK || claims.ImpersonationID == 0 {
			response.Error(w, http.StatusUnauthorized, "Sesi dukungan sudah berakhir", map[string]string{"code": "support_session_ended"})
			return
		}
		if err := v.CheckSupportSession(r.Context(), tenantID, claims.ImpersonationID, claims.Impersonator, time.Now()); err != nil {
			response.Error(w, http.StatusUnauthorized, "Sesi dukungan sudah berakhir", map[string]string{"code": "support_session_ended"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RestrictImpersonation makes a support session read-only: while a platform
// admin looks at a workspace as its Owner, anything that is not a read is
// refused (support access is for looking, FLOW-IDEAL 1.5). Must run after
// RequireAuth.
func RestrictImpersonation(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if claims, ok := FromContext(r.Context()); ok && claims.Impersonator != "" {
			switch r.Method {
			case http.MethodGet, http.MethodHead, http.MethodOptions:
			default:
				response.Error(w, http.StatusForbidden, "Sesi dukungan hanya untuk melihat. Akhiri sesi untuk mengubah data.", map[string]string{"code": "impersonation_read_only"})
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
