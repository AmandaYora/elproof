package application

import (
	"context"
	"strconv"
	"strings"
	"unicode/utf8"

	"elproof/internal/modules/projects/domain"
	"elproof/internal/shared/apperror"
)

type MilestoneTemplateRepository interface {
	List(ctx context.Context, tenantID int64) ([]domain.ProjectMilestoneTemplate, error)
	FindByID(ctx context.Context, tenantID, id int64) (*domain.ProjectMilestoneTemplate, error)
	Create(ctx context.Context, t *domain.ProjectMilestoneTemplate) error
	Update(ctx context.Context, t *domain.ProjectMilestoneTemplate) error
	Delete(ctx context.Context, tenantID, id int64) error
	NextSortOrder(ctx context.Context, tenantID int64) (int, error)
	Reorder(ctx context.Context, tenantID int64, orderedIDs []int64) error
}

type MilestoneTemplateService struct {
	repo MilestoneTemplateRepository
}

func NewMilestoneTemplateService(repo MilestoneTemplateRepository) *MilestoneTemplateService {
	return &MilestoneTemplateService{repo: repo}
}

func (s *MilestoneTemplateService) List(ctx context.Context, tenantID int64) ([]domain.ProjectMilestoneTemplate, error) {
	return s.repo.List(ctx, tenantID)
}

func (s *MilestoneTemplateService) Get(ctx context.Context, tenantID, id int64) (*domain.ProjectMilestoneTemplate, error) {
	t, err := s.repo.FindByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, apperror.NotFound("Tahapan jadwal bawaan tidak ditemukan")
	}
	return t, nil
}

type MilestoneTemplateInput struct {
	Name            string
	DaysBeforeEvent int
}

// MaxTemplateDays bounds a template's offset either side of the event (FQ-04):
// two years before (H-730) to two years after (H+730). A project's own
// Timeline offsets are copied only from templates, so this one bound covers
// both.
const MaxTemplateDays = 730

// Column width of project_milestone_templates.name (migration 000024) and the
// form's minimum (milestone-template.schema.ts).
const (
	minTemplateNameLen = 3
	maxTemplateNameLen = 255
)

// validateTemplate trims the name in place and checks the name and the offset.
func validateTemplate(input *MilestoneTemplateInput) error {
	f := newFieldErrors()
	input.Name = strings.TrimSpace(input.Name)
	switch n := utf8.RuneCountInString(input.Name); {
	case n < minTemplateNameLen:
		f.add("Isi nama tahapan, minimal "+strconv.Itoa(minTemplateNameLen)+" karakter", "name")
	case n > maxTemplateNameLen:
		f.add("Nama tahapan maksimal "+strconv.Itoa(maxTemplateNameLen)+" karakter", "name")
	}
	switch {
	case input.DaysBeforeEvent > MaxTemplateDays:
		f.add("Jumlah hari maksimal "+strconv.Itoa(MaxTemplateDays)+" (2 tahun) sebelum acara.", "daysBeforeEvent")
	case input.DaysBeforeEvent < -MaxTemplateDays:
		f.add("Jumlah hari maksimal "+strconv.Itoa(MaxTemplateDays)+" (2 tahun) setelah acara.", "daysBeforeEvent")
	}
	return f.err()
}

// Create accepts a negative DaysBeforeEvent as a deliberate "H+" (Setelah
// Acara) offset -- default_milestones.go's AddDate(0, 0, -DaysBeforeEvent)
// already resolves this correctly to eventDate+N with no change needed
// there (PLAN.md "Timeline Default: dukung H+"). Either way the offset is at
// most MaxTemplateDays.
func (s *MilestoneTemplateService) Create(ctx context.Context, tenantID int64, input MilestoneTemplateInput) (*domain.ProjectMilestoneTemplate, error) {
	if err := validateTemplate(&input); err != nil {
		return nil, err
	}
	order, err := s.repo.NextSortOrder(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	t := &domain.ProjectMilestoneTemplate{TenantID: tenantID, SortOrder: order, Name: input.Name, DaysBeforeEvent: input.DaysBeforeEvent}
	if err := s.repo.Create(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

// Update -- see Create's doc comment re: negative DaysBeforeEvent (H+).
func (s *MilestoneTemplateService) Update(ctx context.Context, tenantID, id int64, input MilestoneTemplateInput) (*domain.ProjectMilestoneTemplate, error) {
	if err := validateTemplate(&input); err != nil {
		return nil, err
	}
	t, err := s.Get(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	t.Name = input.Name
	t.DaysBeforeEvent = input.DaysBeforeEvent
	if err := s.repo.Update(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

// Delete is a hard delete (PLAN.md's decision) -- unlike a vendor category, a
// template row is only ever copied into a project's own milestones at
// creation time, so it's never referenced afterward and there's nothing to
// orphan by removing it outright.
func (s *MilestoneTemplateService) Delete(ctx context.Context, tenantID, id int64) error {
	if _, err := s.Get(ctx, tenantID, id); err != nil {
		return err
	}
	return s.repo.Delete(ctx, tenantID, id)
}

// Reorder mirrors ProjectService.ReorderMilestones' own validation shape:
// orderedIDs must be an exact permutation of this tenant's current template
// IDs, or the whole request is rejected.
func (s *MilestoneTemplateService) Reorder(ctx context.Context, tenantID int64, orderedIDs []int64) error {
	existing, err := s.repo.List(ctx, tenantID)
	if err != nil {
		return err
	}
	if len(orderedIDs) != len(existing) {
		return apperror.Validation("Urutan jadwal bawaan tidak valid", nil)
	}
	existingIDs := make(map[int64]bool, len(existing))
	for _, t := range existing {
		existingIDs[t.ID] = true
	}
	seen := make(map[int64]bool, len(orderedIDs))
	for _, id := range orderedIDs {
		if !existingIDs[id] || seen[id] {
			return apperror.Validation("Urutan jadwal bawaan tidak valid", nil)
		}
		seen[id] = true
	}
	return s.repo.Reorder(ctx, tenantID, orderedIDs)
}

// defaultMilestoneTemplateSeed is the starter checklist a newly registered
// tenant gets (moved here from default_milestones.go, same names/offsets as
// before this became configurable) -- the common WO flow from first venue
// survey through hari-H. Already-registered tenants were backfilled with the
// exact same rows directly by migration 000024, not by this code path.
var defaultMilestoneTemplateSeed = []struct {
	Name            string
	DaysBeforeEvent int
}{
	{"Survei Venue & Vendor", 90},
	{"DP / Tanda Jadi ke Vendor", 60},
	{"Technical Meeting", 14},
	{"Pelunasan Vendor", 7},
	{"Gladi Resik", 1},
	{"Hari-H Pernikahan", 0},
}

// SeedDefaults is called once, from `platform`'s tenant registration flow
// (via projects/contracts), to give a newly registered tenant a starting
// Timeline Default template instead of an empty one.
//
// Idempotent: a retried workspace setup only adds the steps still missing.
func (s *MilestoneTemplateService) SeedDefaults(ctx context.Context, tenantID int64) error {
	existing, err := s.repo.List(ctx, tenantID)
	if err != nil {
		return err
	}
	have := make(map[string]bool, len(existing))
	for _, t := range existing {
		have[t.Name] = true
	}
	for i, tmpl := range defaultMilestoneTemplateSeed {
		if have[tmpl.Name] {
			continue
		}
		t := &domain.ProjectMilestoneTemplate{TenantID: tenantID, SortOrder: i + 1, Name: tmpl.Name, DaysBeforeEvent: tmpl.DaysBeforeEvent}
		if err := s.repo.Create(ctx, t); err != nil {
			return err
		}
	}
	return nil
}
