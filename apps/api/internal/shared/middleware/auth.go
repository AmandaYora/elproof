// Package middleware holds cross-cutting HTTP middleware. Verifying an access
// token's signature/expiry is treated as a technical utility (crypto operation),
// not domain logic — issuing tokens (password checks, rotation) stays inside the
// identity module. See ADR-0005.
package middleware

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/golang-jwt/jwt/v5"

	"elproof/internal/shared/response"
)

// Claims is the shared shape both the identity module (issuer) and this
// middleware (verifier) agree on for the JWT access token.
type Claims struct {
	PrincipalType string `json:"principal_type"`
	PrincipalID   string `json:"principal_id"`
	TenantID      string `json:"tenant_id,omitempty"`
	Role          string `json:"role"`
	jwt.RegisteredClaims
}

// TenantIDInt parses the string tenant_id claim to int64 — every
// tenant-scoped module (staff, clients, vendors, projects) needs this to
// scope its own queries; platform_admin principals have no tenant_id, so ok
// is false for them.
func (c *Claims) TenantIDInt() (int64, bool) {
	if c.TenantID == "" {
		return 0, false
	}
	id, err := strconv.ParseInt(c.TenantID, 10, 64)
	if err != nil {
		return 0, false
	}
	return id, true
}

type contextKey string

const claimsContextKey contextKey = "elproof.auth.claims"

// RequireAuth verifies the Authorization: Bearer <token> header against secret,
// and injects the parsed Claims into the request context on success.
func RequireAuth(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			token, ok := strings.CutPrefix(header, "Bearer ")
			if !ok || token == "" {
				response.Error(w, http.StatusUnauthorized, "Token otorisasi tidak ditemukan", nil)
				return
			}

			claims := &Claims{}
			parsed, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (interface{}, error) {
				return []byte(secret), nil
			})
			if err != nil || !parsed.Valid {
				response.Error(w, http.StatusUnauthorized, "Token tidak valid atau sudah kedaluwarsa", nil)
				return
			}

			ctx := context.WithValue(r.Context(), claimsContextKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// FromContext extracts the authenticated Claims injected by RequireAuth.
func FromContext(ctx context.Context) (*Claims, bool) {
	claims, ok := ctx.Value(claimsContextKey).(*Claims)
	return claims, ok
}
