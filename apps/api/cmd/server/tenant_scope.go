package main

import (
	"net/http"
	"strings"
)

// staffAllowedWhenRestricted lists what a WO's staff can still do once the
// tenant's subscription is read-only or billing-only (middleware
// TenantAccessGate): see who they are, see the plan catalog and their own
// subscription, pay to reactivate, and export their data. Data is never
// deleted automatically (product decision K10), so exports stay available.
func staffAllowedWhenRestricted(method, path string) bool {
	p := strings.TrimSuffix(path, "/")
	switch method {
	case http.MethodGet, http.MethodHead:
		switch {
		case p == "/api/v1/auth/me",
			p == "/api/v1/tenants/me",
			strings.HasPrefix(p, "/api/v1/tenants/me/"),
			p == "/api/v1/plans",
			p == "/api/v1/subscription-transactions",
			p == "/api/v1/subscriptions/pending-charge",
			strings.HasSuffix(p, "/export"):
			return true
		}
	case http.MethodPost:
		switch p {
		case "/api/v1/subscriptions/pay", "/api/v1/subscriptions/pending-charge/cancel":
			return true
		}
	}
	return false
}
