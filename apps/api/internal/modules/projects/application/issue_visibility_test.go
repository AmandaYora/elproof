package application

import (
	"context"
	"testing"
	"time"

	"elproof/internal/modules/projects/domain"
	"elproof/internal/shared/apperror"
)

type fakeIssues struct {
	IssueRepository
	rows []domain.VendorIssue
	next int64
}

func (f *fakeIssues) Create(_ context.Context, i *domain.VendorIssue) error {
	f.next++
	i.ID = f.next
	f.rows = append(f.rows, *i)
	return nil
}
func (f *fakeIssues) FindByID(_ context.Context, _, id int64) (*domain.VendorIssue, error) {
	for _, r := range f.rows {
		if r.ID == id {
			i := r
			return &i, nil
		}
	}
	return nil, nil
}
func (f *fakeIssues) Patch(_ context.Context, i *domain.VendorIssue, _ []string, _ *time.Time) (bool, error) {
	for k := range f.rows {
		if f.rows[k].ID == i.ID {
			f.rows[k] = *i
		}
	}
	return true, nil
}

func yes(b bool) *bool { return &b }

func TestAKendalaWithoutAVendorWorksAndDefaultsToShared(t *testing.T) {
	repo := &fakeIssues{}
	svc := &IssueService{repo: repo, engagements: onePV{byProject: map[int64]map[int64]bool{5: {1: true}}}}
	ctx := context.Background()
	in := IssueInput{Title: "Hujan", Impact: domain.ImpactLow, FoundDate: time.Now()}

	got, err := svc.Create(ctx, 5, 9, in)
	if err != nil {
		t.Fatalf("an issue about the event in general needs no vendor: %v", err)
	}
	if got.HasVendor() || !got.IsClientVisible {
		t.Errorf("no vendor, shared by default: %+v", got)
	}
	in.IsClientVisible = yes(false)
	if got, err = svc.Create(ctx, 5, 9, in); err != nil || got.IsClientVisible {
		t.Errorf("isClientVisible=false must be honoured: %+v %v", got, err)
	}
	// A deliverable belongs to a vendor.
	ms := int64(3)
	in.VendorMilestoneID = &ms
	if _, err := svc.Create(ctx, 5, 9, in); apperror.HTTPStatus(err) != 422 {
		t.Errorf("a vendor milestone without a vendor: want 422, got %v", err)
	}
	// A vendor of another project is still refused.
	in.VendorMilestoneID, in.ProjectVendorID = nil, 1
	if _, err := svc.Create(ctx, 6, 9, in); apperror.HTTPStatus(err) != 422 {
		t.Errorf("another project's engagement: want 422, got %v", err)
	}
}

func TestIssuePatchKeepsVisibilityUnlessSent(t *testing.T) {
	repo := &fakeIssues{rows: []domain.VendorIssue{{ID: 1, ProjectID: 5, Title: "x", IsClientVisible: false, Status: domain.IssueOpen}}}
	svc := &IssueService{repo: repo, engagements: onePV{byProject: map[int64]map[int64]bool{}}}
	upd := IssuePatch{Title: Some("y"), Impact: Some(domain.ImpactLow)}
	got, err := svc.Patch(context.Background(), 5, 1, 9, upd, time.Now(), nil)
	if err != nil || got.IsClientVisible {
		t.Fatalf("omitted keeps hidden: %+v %v", got, err)
	}
	upd.IsClientVisible = Some(true)
	if got, err = svc.Patch(context.Background(), 5, 1, 9, upd, time.Now(), nil); err != nil || !got.IsClientVisible {
		t.Fatalf("sent true shares it: %+v %v", got, err)
	}
}

// A kendala staff did not share with the client must not leak through the
// project's progress: not in the open count, not in the condition.
func TestClientProgressIgnoresHiddenIssues(t *testing.T) {
	data := &progressData{
		milestones: map[int64][]domain.ProjectMilestone{}, engagements: map[int64][]domain.ProjectVendor{},
		vendorMilestones: map[int64][]domain.VendorMilestone{}, payments: map[int64][]domain.VendorPayment{},
		venuePayments: map[int64][]domain.VenuePayment{}, evidence: map[int64][]domain.Evidence{},
		issues: map[int64][]domain.VendorIssue{1: {
			{ID: 1, ProjectID: 1, Impact: domain.ImpactCritical, Status: domain.IssueOpen, IsClientVisible: false},
			{ID: 2, ProjectID: 1, Impact: domain.ImpactLow, Status: domain.IssueOpen, IsClientVisible: true},
		}},
	}
	asOf := time.Date(2026, 11, 10, 0, 0, 0, 0, time.UTC)
	staff := data.progressFor(1, asOf, false)
	client := data.progressFor(1, asOf, true)
	if staff.OpenIssueCount != 2 || staff.Condition != domain.ConditionAtRisk {
		t.Errorf("staff see everything: %+v", staff)
	}
	if client.OpenIssueCount != 1 || client.CriticalOrHighOpenCount != 0 || client.Condition != domain.ConditionAttention {
		t.Errorf("a client sees only the shared kendala: %+v", client)
	}
	data.issues[1] = data.issues[1][:1] // only the hidden one
	if c := data.progressFor(1, asOf, true); c.OpenIssueCount != 0 || c.Condition != domain.ConditionOnTrack {
		t.Errorf("a hidden critical kendala must not push the client's condition: %+v", c)
	}
}
