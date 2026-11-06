package middleware

import (
	"context"
	"net/http"

	"elproof/internal/shared/response"
)

// Grants is what the signed-in staff member or client may do in their
// workspace, resolved per request from their role (ADR-0026). Permission
// keys are checked by the owning modules; this type only carries them.
type Grants struct {
	RoleID   int64
	RoleName string
	// Owner: the workspace's system role (every permission).
	Owner bool
	// AllProjects: project permissions reach every project; otherwise only
	// projects where the person is PIC or PIC Sales.
	AllProjects bool
	Permissions map[string]bool
}

// Can reports whether a permission is granted (nil Grants grant nothing).
func (g *Grants) Can(permission string) bool {
	return g != nil && g.Permissions[permission]
}

// Covers reports whether these grants include every permission listed and,
// when allProjects is set, the every-project scope. Someone may only hand
// out (or edit) a role their own role covers, so managing the team or roles
// never becomes a way to give oneself more access (ADR-0026). The Owner
// covers everything.
func (g *Grants) Covers(permissions []string, allProjects bool) bool {
	if g == nil {
		return false
	}
	if g.Owner {
		return true
	}
	if allProjects && !g.AllProjects {
		return false
	}
	for _, p := range permissions {
		if !g.Permissions[p] {
			return false
		}
	}
	return true
}

// GrantsResolver loads a role's grants (implemented by the access module).
// A nil result means the role no longer exists.
type GrantsResolver interface {
	ResolveGrants(ctx context.Context, tenantID, roleID int64) (*Grants, error)
}

const grantsKey contextKey = "grants"

// WithGrants stores grants in a context (tests and LoadGrants).
func WithGrants(ctx context.Context, g *Grants) context.Context {
	return context.WithValue(ctx, grantsKey, g)
}

// GrantsFrom returns the request's grants; nil for principals without a
// workspace role (platform admins, service tokens).
func GrantsFrom(ctx context.Context) *Grants {
	g, _ := ctx.Value(grantsKey).(*Grants)
	return g
}

// LoadGrants resolves the grants of staff and client principals once per
// request. A token without a role id predates ADR-0026: it is refused as an
// expired session so the client refreshes into a token that carries one.
// The resolver failing closes access (503) rather than guessing.
type LoadGrants struct {
	resolver GrantsResolver
}

// SetResolver completes two-phase wiring (the access module is built after
// the authed middleware chain is assembled).
func (l *LoadGrants) SetResolver(r GrantsResolver) {
	l.resolver = r
}

func (l *LoadGrants) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := FromContext(r.Context())
		if !ok || (claims.PrincipalType != "staff" && claims.PrincipalType != "client") {
			next.ServeHTTP(w, r)
			return
		}
		tenantID, ok := claims.TenantIDInt()
		if !ok {
			next.ServeHTTP(w, r)
			return
		}
		if claims.RoleID == 0 {
			response.Error(w, http.StatusUnauthorized, "Sesi perlu diperbarui, silakan masuk kembali",
				map[string]string{"code": "session_revoked"})
			return
		}
		if l.resolver == nil {
			response.Error(w, http.StatusServiceUnavailable, "Hak akses belum siap, coba lagi sebentar", nil)
			return
		}
		g, err := l.resolver.ResolveGrants(r.Context(), tenantID, claims.RoleID)
		if err != nil {
			response.Error(w, http.StatusServiceUnavailable, "Hak akses tidak dapat diperiksa, coba lagi sebentar", nil)
			return
		}
		if g == nil {
			response.Error(w, http.StatusForbidden, "Peran Anda sudah tidak ada. Hubungi Owner WO Anda.",
				map[string]string{"code": "role_missing"})
			return
		}
		next.ServeHTTP(w, r.WithContext(WithGrants(r.Context(), g)))
	})
}
