package domain

import (
	"reflect"
	"testing"

	"elproof/internal/shared/apperror"
)

func TestNormalizeBulkIDs(t *testing.T) {
	got, err := NormalizeBulkIDs([]int64{5, 3, 5, 9, 3})
	if err != nil || !reflect.DeepEqual(got, []int64{5, 3, 9}) {
		t.Fatalf("dedupe keeping first appearance: %v %v", got, err)
	}
	over := make([]int64, MaxBulkIDs+1)
	for i := range over {
		over[i] = int64(i + 1)
	}
	for name, ids := range map[string][]int64{"empty": {}, "nil": nil, "too many": over, "zero": {1, 0}, "negative": {-4}} {
		_, err := NormalizeBulkIDs(ids)
		appErr, ok := apperror.As(err)
		if !ok || appErr.Kind != apperror.KindValidation || len(appErr.Fields["ids"]) != 1 {
			t.Errorf("%s: want a validation error keyed ids, got %v", name, err)
		}
	}
	exactly := over[:MaxBulkIDs]
	if got, err := NormalizeBulkIDs(exactly); err != nil || len(got) != MaxBulkIDs {
		t.Errorf("100 ids are allowed: %d %v", len(got), err)
	}
	dup := make([]int64, MaxBulkIDs+1)
	for i := range dup {
		dup[i] = 7
	}
	if _, err := NormalizeBulkIDs(dup); err == nil {
		t.Error("101 repeated ids are still over the limit as sent")
	}
}

func TestParseBulkAction(t *testing.T) {
	if a, ok := ParseBulkAction("activate"); !ok || !a.Target() {
		t.Error("activate sets active")
	}
	if a, ok := ParseBulkAction("deactivate"); !ok || a.Target() {
		t.Error("deactivate clears active")
	}
	for _, bad := range []string{"", "Activate", "delete", "toggle", "activate "} {
		if _, ok := ParseBulkAction(bad); ok {
			t.Errorf("%q must be rejected", bad)
		}
	}
}

func TestPlanBulk(t *testing.T) {
	current := map[int64]bool{1: true, 2: false, 3: true, 4: false}
	up, same, missing := PlanBulk([]int64{4, 1, 99, 2, 3, 100}, current, true)
	if !reflect.DeepEqual(up, []int64{4, 2}) || !reflect.DeepEqual(same, []int64{1, 3}) || !reflect.DeepEqual(missing, []int64{99, 100}) {
		t.Errorf("activate: %v %v %v", up, same, missing)
	}
	up, same, missing = PlanBulk([]int64{1, 2}, current, false)
	if !reflect.DeepEqual(up, []int64{1}) || !reflect.DeepEqual(same, []int64{2}) || len(missing) != 0 {
		t.Errorf("deactivate: %v %v %v", up, same, missing)
	}
	up, same, missing = PlanBulk([]int64{7}, map[int64]bool{}, true)
	if up == nil || same == nil || missing == nil {
		t.Error("lists are never nil (they encode as [])")
	}
}
