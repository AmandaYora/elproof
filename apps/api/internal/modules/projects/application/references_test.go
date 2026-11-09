package application

import (
	"context"
	"testing"

	"elproof/internal/modules/projects/domain"
	"elproof/internal/shared/apperror"
)

// staffByTenant knows which staff ids exist in which workspace.
type staffByTenant map[int64]map[int64]string

func (s staffByTenant) GetName(_ context.Context, tenantID, staffID int64) (string, bool, error) {
	name, ok := s[tenantID][staffID]
	return name, ok, nil
}
func (s staffByTenant) GetPhone(context.Context, int64, int64) (string, bool, error) {
	return "", false, nil
}

func (s staffByTenant) StaffNames(_ context.Context, tenantID int64, ids []int64) (map[int64]string, error) {
	out := map[int64]string{}
	for _, id := range ids {
		if name, ok := s[tenantID][id]; ok {
			out[id] = name
		}
	}
	return out, nil
}

type vendorsByTenant struct{ vendors, categories map[int64]map[int64]bool }

func (v vendorsByTenant) VendorExists(_ context.Context, t, id int64) (bool, error) {
	return v.vendors[t][id], nil
}
func (v vendorsByTenant) CategoryExists(_ context.Context, t, id int64) (bool, error) {
	return v.categories[t][id], nil
}

func (v vendorsByTenant) VendorNames(context.Context, int64, []int64) (map[int64]string, error) {
	return map[int64]string{}, nil
}
func (v vendorsByTenant) CategoryNames(context.Context, int64, []int64) (map[int64]string, error) {
	return map[int64]string{}, nil
}

func TestReferencesOnlyAcceptTheCallersOwnWorkspace(t *testing.T) {
	ctx := context.Background()
	refs := NewReferences(staffByTenant{1: {10: "Ani"}, 2: {20: "Budi"}})
	refs.SetVendors(vendorsByTenant{
		vendors:    map[int64]map[int64]bool{1: {5: true}, 2: {6: true}},
		categories: map[int64]map[int64]bool{1: {7: true}, 2: {8: true}},
	})

	if err := refs.Staff(ctx, 1, "picStaffId", 10, 0); err != nil {
		t.Errorf("own staff (and 0 = none) must pass: %v", err)
	}
	for name, err := range map[string]error{
		"another workspace's staff":    refs.Staff(ctx, 1, "picStaffId", 20),
		"a staff id nobody has":        refs.Staff(ctx, 1, "picStaffId", 999),
		"one bad id among good":        refs.Staff(ctx, 1, "picStaffId", 10, 20),
		"another workspace's vendor":   refs.Vendor(ctx, 1, "vendorId", 6),
		"another workspace's category": refs.Category(ctx, 1, "categoryId", 8),
	} {
		ae, ok := apperror.As(err)
		if err == nil || !ok || ae.Kind != apperror.KindValidation {
			t.Errorf("%s must be refused with a validation error, got %v", name, err)
		}
	}
	if err := refs.Vendor(ctx, 1, "vendorId", 5); err != nil {
		t.Errorf("own vendor: %v", err)
	}
	if err := refs.Category(ctx, 1, "categoryId", 7); err != nil {
		t.Errorf("own category: %v", err)
	}
}

type onePV struct {
	VendorEngagementRepository
	byProject map[int64]map[int64]bool
}

func (r onePV) FindByID(_ context.Context, projectID, id int64) (*domain.ProjectVendor, error) {
	if r.byProject[projectID][id] {
		return &domain.ProjectVendor{ID: id, ProjectID: projectID}, nil
	}
	return nil, nil
}

func TestIssueMustUseAnEngagementOfTheSameProject(t *testing.T) {
	s := &IssueService{engagements: onePV{byProject: map[int64]map[int64]bool{5: {1: true}, 6: {2: true}}}}
	if err := s.validateEngagement(context.Background(), 5, 1); err != nil {
		t.Errorf("own engagement: %v", err)
	}
	for _, c := range []struct{ project, pv int64 }{{5, 2}, {6, 1}, {5, 0}, {5, 99}} {
		if err := s.validateEngagement(context.Background(), c.project, c.pv); err == nil {
			t.Errorf("project %d with engagement %d must be refused", c.project, c.pv)
		}
	}
}
