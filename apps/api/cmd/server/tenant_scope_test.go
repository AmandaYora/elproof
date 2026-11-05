package main

import (
	"net/http"
	"testing"
)

func TestStaffAllowedWhenRestricted(t *testing.T) {
	allowed := []struct{ method, path string }{
		{http.MethodGet, "/api/v1/auth/me"},
		{http.MethodGet, "/api/v1/tenants/me"},
		{http.MethodGet, "/api/v1/tenants/me/subscription"},
		{http.MethodGet, "/api/v1/plans"},
		{http.MethodGet, "/api/v1/subscriptions/pending-charge"},
		{http.MethodPost, "/api/v1/subscriptions/pay"},
		{http.MethodPost, "/api/v1/subscriptions/pending-charge/cancel"},
		{http.MethodGet, "/api/v1/vendors/export"},
	}
	for _, c := range allowed {
		if !staffAllowedWhenRestricted(c.method, c.path) {
			t.Errorf("%s %s should stay available", c.method, c.path)
		}
	}
	denied := []struct{ method, path string }{
		{http.MethodGet, "/api/v1/projects"},
		{http.MethodPost, "/api/v1/projects"},
		{http.MethodPost, "/api/v1/plans"},
		{http.MethodPatch, "/api/v1/tenants/me"},
		{http.MethodDelete, "/api/v1/vendors/3"},
	}
	for _, c := range denied {
		if staffAllowedWhenRestricted(c.method, c.path) {
			t.Errorf("%s %s must not bypass the restriction", c.method, c.path)
		}
	}
}
