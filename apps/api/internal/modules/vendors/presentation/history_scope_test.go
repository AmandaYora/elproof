package presentation

import (
	"context"
	"testing"

	projectscontracts "elproof/internal/modules/projects/contracts"
	"elproof/internal/shared/middleware"
)

// fakeProjects answers only what the project history needs; any other call panics.
type fakeProjects struct {
	projectscontracts.Contracts
	history  []projectscontracts.VendorEngagementHistoryItem
	pic      map[int64]int64
	picSales map[int64]int64
}

func (f fakeProjects) ListVendorEngagementHistory(context.Context, int64, int64) ([]projectscontracts.VendorEngagementHistoryItem, error) {
	return f.history, nil
}
func (f fakeProjects) ProjectPICStaffID(_ context.Context, _, projectID int64) (int64, error) {
	return f.pic[projectID], nil
}
func (f fakeProjects) ProjectPICSalesStaffID(_ context.Context, _, projectID int64) (int64, error) {
	return f.picSales[projectID], nil
}

func staffCtx(staffID string, allProjects bool) context.Context {
	ctx := middleware.NewContextWithClaims(context.Background(), &middleware.Claims{PrincipalType: "staff", PrincipalID: staffID, TenantID: "1"})
	return middleware.WithGrants(ctx, &middleware.Grants{AllProjects: allProjects, Permissions: map[string]bool{"vendor.view": true}})
}

// T-20261110-12: a role limited to the projects it holds never sees tenant-wide
// lists (ADR-0026). A vendor's "Lihat proyek" history is one: with vendor.view on
// an "assigned" role it listed every project the vendor worked on.
func TestVendorHistoryFollowsTheProjectScope(t *testing.T) {
	h := &VendorHandler{projects: fakeProjects{
		history:  []projectscontracts.VendorEngagementHistoryItem{{ProjectID: 1, ProjectName: "Milik saya"}, {ProjectID: 2, ProjectName: "Orang lain"}, {ProjectID: 3, ProjectName: "Saya PIC Sales"}},
		pic:      map[int64]int64{1: 7, 2: 9, 3: 9},
		picSales: map[int64]int64{3: 7},
	}}
	got, err := h.visibleHistory(staffCtx("7", false), 1, 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].ProjectID != 1 || got[1].ProjectID != 3 {
		t.Errorf("assigned scope: got %+v, want only projects 1 and 3", got)
	}
	all, err := h.visibleHistory(staffCtx("7", true), 1, 5)
	if err != nil || len(all) != 3 {
		t.Errorf("every-project scope keeps the whole history: %+v %v", all, err)
	}
}

func (f fakeProjects) ListAffectedProjectsForVendor(context.Context, int64, int64) ([]projectscontracts.ProjectRef, error) {
	return f.refs(), nil
}

func (f fakeProjects) refs() []projectscontracts.ProjectRef {
	out := make([]projectscontracts.ProjectRef, 0, len(f.history))
	for _, h := range f.history {
		out = append(out, projectscontracts.ProjectRef{ID: h.ProjectID, Name: h.ProjectName})
	}
	return out
}

// The hard-delete warning is the same kind of list: an "assigned" role (e.g. a
// custom one holding vendor.delete) is told how many other projects are affected,
// never their names (review round 1, T-20261110-12).
func TestDeleteImpactNamesOnlyProjectsInScope(t *testing.T) {
	f := fakeProjects{
		history:  []projectscontracts.VendorEngagementHistoryItem{{ProjectID: 1, ProjectName: "Milik saya"}, {ProjectID: 2, ProjectName: "Orang lain"}, {ProjectID: 3, ProjectName: "Orang lain juga"}},
		pic:      map[int64]int64{1: 7, 2: 9, 3: 9},
		picSales: map[int64]int64{},
	}
	got, err := scopedImpact(staffCtx("7", false), f, 1, f.refs())
	if err != nil {
		t.Fatal(err)
	}
	if len(got.AffectedProjects) != 1 || got.AffectedProjects[0].Name != "Milik saya" || got.HiddenProjectCount != 2 {
		t.Errorf("assigned scope: got %+v, want only 'Milik saya' named and 2 hidden", got)
	}
	all, err := scopedImpact(staffCtx("7", true), f, 1, f.refs())
	if err != nil || len(all.AffectedProjects) != 3 || all.HiddenProjectCount != 0 {
		t.Errorf("every-project scope names every project: %+v %v", all, err)
	}
}
