package domain

import "testing"

// EP-12: one seat definition. With an active Owner and one active staff member
// the Pengguna tab shows two active people but one seat used — the same number
// the Pemakaian tab and the trial cap count.
func TestCountsTowardSeats(t *testing.T) {
	members := []StaffMember{
		{Name: "Owner", Role: RoleOwner, IsActive: true},
		{Name: "Admin aktif", Role: "Admin", IsActive: true},
		{Name: "Staf nonaktif", Role: "Staff", IsActive: false},
		{Name: "Owner nonaktif (tidak terjadi, tetap bukan kursi)", Role: RoleOwner, IsActive: false},
	}
	seats, active := 0, 0
	for _, m := range members {
		if m.CountsTowardSeats() {
			seats++
		}
		if m.IsActive {
			active++
		}
	}
	if seats != 1 || active != 2 {
		t.Fatalf("seats %d active %d, want 1 and 2", seats, active)
	}
}
