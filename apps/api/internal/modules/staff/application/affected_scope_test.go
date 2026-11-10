package application

import (
	"context"
	"testing"

	projectscontracts "elproof/internal/modules/projects/contracts"
)

type fakeLookup struct {
	refs          []projectscontracts.ProjectRef
	pic, picSales map[int64]int64
}

func (f fakeLookup) ListAffectedProjectsForStaff(context.Context, int64, int64) ([]projectscontracts.ProjectRef, error) {
	return f.refs, nil
}
func (f fakeLookup) ProjectPICStaffID(_ context.Context, _, id int64) (int64, error) {
	return f.pic[id], nil
}
func (f fakeLookup) ProjectPICSalesStaffID(_ context.Context, _, id int64) (int64, error) {
	return f.picSales[id], nil
}

// T-20261110-12 (review round 1): a team manager whose role reaches only the
// projects they hold is told how many other projects the member is PIC of, never
// their names (ADR-0026: an "assigned" role never sees a tenant-wide project list).
func TestAffectedProjectsNamesOnlyTheViewersOwn(t *testing.T) {
	s := &StaffService{}
	s.SetProjectReferenceLookup(fakeLookup{
		refs:     []projectscontracts.ProjectRef{{ID: 1, Name: "Dipegang saya"}, {ID: 2, Name: "Bukan milik saya"}, {ID: 3, Name: "Saya PIC Sales"}},
		pic:      map[int64]int64{1: 7, 2: 9, 3: 9},
		picSales: map[int64]int64{3: 7},
	})
	named, hidden, err := s.AffectedProjectsFor(context.Background(), 1, 9, 7, false)
	if err != nil || len(named) != 2 || named[0].ID != 1 || named[1].ID != 3 || hidden != 1 {
		t.Errorf("assigned viewer: named %+v hidden %d err %v; want 1 and 3 named, 1 hidden", named, hidden, err)
	}
	named, hidden, err = s.AffectedProjectsFor(context.Background(), 1, 9, 7, true)
	if err != nil || len(named) != 3 || hidden != 0 {
		t.Errorf("every-project viewer: named %+v hidden %d err %v", named, hidden, err)
	}
}
