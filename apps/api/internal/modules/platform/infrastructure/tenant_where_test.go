package infrastructure

import (
	"reflect"
	"testing"

	"elproof/internal/modules/platform/domain"
)

// EP-09: the status filter of the tenant list (and its CSV export) never puts
// a frozen tenant under a lifecycle status; frozen tenants have their own value.
func TestTenantWhereStatusAndFrozen(t *testing.T) {
	cases := []struct {
		name  string
		f     domain.TenantFilter
		where string
		args  []interface{}
	}{
		{"no filter", domain.TenantFilter{}, "", nil},
		{"active excludes frozen", domain.TenantFilter{Status: "active"},
			` WHERE subscription_status = ? AND is_suspended = FALSE`, []interface{}{"active"}},
		{"frozen only", domain.TenantFilter{Status: domain.FilterFrozen},
			` WHERE is_suspended = TRUE`, nil},
		{"frozen with a plan", domain.TenantFilter{Status: domain.FilterFrozen, PlanID: ptr(int64(3))},
			` WHERE is_suspended = TRUE AND plan_id = ?`, []interface{}{int64(3)}},
	}
	for _, c := range cases {
		where, args := tenantWhere(c.f)
		if where != c.where || !reflect.DeepEqual(args, c.args) {
			t.Errorf("%s: got %q %v, want %q %v", c.name, where, args, c.where, c.args)
		}
	}
}

func ptr[T any](v T) *T { return &v }
