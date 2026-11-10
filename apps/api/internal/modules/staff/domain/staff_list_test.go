package domain

import (
	"reflect"
	"testing"
)

func TestSortRoleIDsByName(t *testing.T) {
	names := map[int64]string{1: "Admin", 2: "wedding planner", 3: "Sales", 4: "admin"}
	got := SortRoleIDsByName(names)
	// Admin(1) and admin(4) tie ignoring case: by id; then Sales, then wedding planner.
	if want := []int64{1, 4, 3, 2}; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	if len(SortRoleIDsByName(nil)) != 0 {
		t.Error("no roles, no ranking")
	}
}
