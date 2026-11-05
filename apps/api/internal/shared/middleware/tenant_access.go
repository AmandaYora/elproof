package middleware

import (
	"context"
	"net/http"
	"sync"

	"elproof/internal/shared/logger"
	"elproof/internal/shared/response"
)

// Tenant access levels, as produced by the platform module's subscription
// lifecycle. Kept as plain strings here so this technical gate never imports
// a domain package.
const (
	TenantAccessFull        = "full"
	TenantAccessReadOnly    = "read_only"
	TenantAccessBillingOnly = "billing_only"
	TenantAccessBlocked     = "blocked"
)

// TenantAccessResolver returns a tenant's current access level.
type TenantAccessResolver interface {
	TenantAccessLevel(ctx context.Context, tenantID int64) (string, error)
}

// TenantAccessGate enforces a tenant's subscription state on every
// authenticated request from a tenant-scoped principal (docs/plan/
// registrasi-wo-self-service/FLOW-IDEAL.md §1.4):
//
//   - blocked (frozen by the platform team): every staff and client request
//     is refused.
//   - read_only: staff may read (GET/HEAD) but not write, except the routes
//     restrictedAllow permits (paying the subscription).
//   - billing_only: staff may only use routes restrictedAllow permits.
//   - client principals are never restricted by billing state: the portal
//     is read-only already and the couple must not lose access because the
//     WO is late paying (product decision K4).
//
// The resolver is bound late (SetResolver) because the platform module that
// implements it is built after the authed middleware chain is assembled.
type TenantAccessGate struct {
	mu              sync.RWMutex
	resolver        TenantAccessResolver
	restrictedAllow RouteAllowFunc
}

func NewTenantAccessGate(restrictedAllow RouteAllowFunc) *TenantAccessGate {
	return &TenantAccessGate{restrictedAllow: restrictedAllow}
}

func (g *TenantAccessGate) SetResolver(resolver TenantAccessResolver) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.resolver = resolver
}

func (g *TenantAccessGate) Middleware(next http.Handler) http.Handler {
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
		g.mu.RLock()
		resolver := g.resolver
		g.mu.RUnlock()
		if resolver == nil {
			next.ServeHTTP(w, r)
			return
		}

		level, err := resolver.TenantAccessLevel(r.Context(), tenantID)
		if err != nil {
			logger.Error("tenant access check failed for tenant %d: %v", tenantID, err)
			response.Error(w, http.StatusServiceUnavailable, "Gagal memeriksa status langganan, coba lagi sebentar lagi", nil)
			return
		}

		if allowed, code, msg := decideTenantAccess(level, claims.PrincipalType, r.Method, r.URL.Path, g.restrictedAllow); !allowed {
			response.Error(w, http.StatusForbidden, msg, map[string]string{"code": code})
			return
		}
		next.ServeHTTP(w, r)
	})
}

// decideTenantAccess is the pure decision behind the gate (unit-tested).
func decideTenantAccess(level, principalType, method, path string, restrictedAllow RouteAllowFunc) (allowed bool, code, message string) {
	if level == TenantAccessBlocked {
		return false, "tenant_frozen", "Akun WO ini sedang dibekukan oleh ElProof. Hubungi tim ElProof untuk bantuan."
	}
	if principalType != "staff" {
		return true, "", ""
	}
	isRead := method == http.MethodGet || method == http.MethodHead
	switch level {
	case TenantAccessReadOnly:
		if isRead || restrictedAllow(method, path) {
			return true, "", ""
		}
		return false, "subscription_read_only",
			"Masa langganan WO sudah berakhir, sehingga data hanya bisa dilihat. Perpanjang langganan untuk kembali mengubah data."
	case TenantAccessBillingOnly:
		if restrictedAllow(method, path) {
			return true, "", ""
		}
		return false, "subscription_suspended",
			"Langganan WO sedang ditangguhkan karena belum diperpanjang. Data Anda tetap aman; perpanjang langganan untuk membukanya kembali."
	default:
		return true, "", ""
	}
}
