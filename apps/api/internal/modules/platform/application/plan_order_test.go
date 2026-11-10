package application

import (
	"reflect"
	"testing"

	billingcontracts "elproof/internal/modules/billing/contracts"
)

func TestPlanIDsByName(t *testing.T) {
	plans := []billingcontracts.Plan{{ID: 3, Name: "Pro"}, {ID: 1, Name: "basic"}, {ID: 2, Name: "Business"}, {ID: 5, Name: "PRO"}}
	got := planIDsByName(plans)
	// basic < Business < Pro/PRO (tie: lower id first)
	if want := []int64{1, 2, 3, 5}; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	if plans[0].ID != 3 {
		t.Error("the caller's slice must not be reordered")
	}
	if len(planIDsByName(nil)) != 0 {
		t.Error("no plans, no ranking")
	}
}
