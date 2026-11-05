package application

import "testing"

func TestActionLabel(t *testing.T) {
	cases := map[string]string{
		"/api/v1/tenants/12/toggle-suspension":               "POST /tenants/{id}/toggle-suspension",
		"/api/v1/payment/apps/app_02ef90c52704/reset-secret": "POST /payment/apps/{id}/reset-secret",
		"/api/v1/plans": "POST /plans",
	}
	for path, want := range cases {
		if got := ActionLabel("POST", path); got != want {
			t.Errorf("%s: got %q, want %q", path, got, want)
		}
	}
}
