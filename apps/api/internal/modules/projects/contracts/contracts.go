// Package contracts is the ONLY package other modules may import from
// projects — used by `clients` (validate a project_id belongs to the
// caller's tenant) and `vendors` (resolve a vendor's cross-project
// engagement history for "Lihat Project", since `project_vendors` is owned
// by `projects`, not `vendors`).
package contracts

import (
	"context"
	"time"

	"elproof/internal/modules/projects/application"
)

// VendorEngagementHistoryItem is the cross-module-safe shape of one history
// row — a primitive projection of domain.VendorEngagementHistoryRow, never
// the domain type itself.
type VendorEngagementHistoryItem struct {
	ProjectID        int64
	ProjectName      string
	EventDate        time.Time
	Venue            string
	EngagementStatus string
}

// ProjectRef is a minimal {id, name} projection of a project — a
// cross-module-safe copy of domain.ProjectRef, same convention as
// VendorEngagementHistoryItem mirroring VendorEngagementHistoryRow above.
type ProjectRef struct {
	ID   int64
	Name string
}

type Contracts interface {
	ProjectExists(ctx context.Context, tenantID, projectID int64) (bool, error)
	// ProjectPICStaffID resolves a project's current PIC — used by `clients`
	// to scope a Wedding Planner's client-read access to only projects
	// they're PIC of (see PLAN.md's RBAC section).
	ProjectPICStaffID(ctx context.Context, tenantID, projectID int64) (int64, error)
	// ProjectPICSalesStaffID resolves a project's current PIC Sales — used by
	// `clients` to scope a Sales staff member's client read/write access to
	// only projects they're PIC Sales of (PLAN.md
	// revisi-timeline-vendor-role-sales).
	ProjectPICSalesStaffID(ctx context.Context, tenantID, projectID int64) (int64, error)
	ListVendorEngagementHistory(ctx context.Context, tenantID, vendorID int64) ([]VendorEngagementHistoryItem, error)
	// ListAffectedProjectsForVendor/Venue/Staff back the hard-delete "impact"
	// endpoints (PLAN.md's Vendor/Venue/Staff hard delete) -- every project
	// that would lose a data source if the given vendor/venue/staff row were
	// hard-deleted, named so the frontend's confirmation dialog can say
	// exactly what's affected instead of a generic warning.
	ListAffectedProjectsForVendor(ctx context.Context, tenantID, vendorID int64) ([]ProjectRef, error)
	ListAffectedProjectsForVenue(ctx context.Context, tenantID, venueID int64) ([]ProjectRef, error)
	ListAffectedProjectsForStaff(ctx context.Context, tenantID, staffID int64) ([]ProjectRef, error)
	// SeedDefaultMilestoneTemplate gives a newly registered tenant a starting
	// Timeline Default template (PLAN.md) — called by `platform`'s tenant
	// registration flow, the same moment `vendors.SeedDefaultCategories` is.
	SeedDefaultMilestoneTemplate(ctx context.Context, tenantID int64) error
}

type impl struct {
	projects           *application.ProjectService
	vendorEngagements  *application.VendorEngagementService
	milestoneTemplates *application.MilestoneTemplateService
}

func New(projects *application.ProjectService, vendorEngagements *application.VendorEngagementService, milestoneTemplates *application.MilestoneTemplateService) Contracts {
	return &impl{projects: projects, vendorEngagements: vendorEngagements, milestoneTemplates: milestoneTemplates}
}

func (c *impl) ProjectExists(ctx context.Context, tenantID, projectID int64) (bool, error) {
	return c.projects.ExistsForTenant(ctx, tenantID, projectID)
}

func (c *impl) ProjectPICStaffID(ctx context.Context, tenantID, projectID int64) (int64, error) {
	p, err := c.projects.Get(ctx, tenantID, projectID)
	if err != nil {
		return 0, err
	}
	return p.PICStaffID, nil
}

func (c *impl) ProjectPICSalesStaffID(ctx context.Context, tenantID, projectID int64) (int64, error) {
	p, err := c.projects.Get(ctx, tenantID, projectID)
	if err != nil {
		return 0, err
	}
	return p.PICSalesStaffID, nil
}

func (c *impl) ListVendorEngagementHistory(ctx context.Context, tenantID, vendorID int64) ([]VendorEngagementHistoryItem, error) {
	rows, err := c.vendorEngagements.ListHistoryForVendor(ctx, tenantID, vendorID)
	if err != nil {
		return nil, err
	}
	items := make([]VendorEngagementHistoryItem, 0, len(rows))
	for _, r := range rows {
		items = append(items, VendorEngagementHistoryItem{
			ProjectID: r.ProjectID, ProjectName: r.ProjectName, EventDate: r.EventDate,
			Venue: r.Venue, EngagementStatus: string(r.EngagementStatus),
		})
	}
	return items, nil
}

// ListAffectedProjectsForVendor reuses ListVendorEngagementHistory's own data
// (project_vendors joined to projects) -- the exact same set the existing
// "Lihat Project" feature already surfaces, just projected down to {id, name}
// and de-duplicated (a vendor could in principle have more than one
// project_vendors row against the same project across its history).
func (c *impl) ListAffectedProjectsForVendor(ctx context.Context, tenantID, vendorID int64) ([]ProjectRef, error) {
	rows, err := c.vendorEngagements.ListHistoryForVendor(ctx, tenantID, vendorID)
	if err != nil {
		return nil, err
	}
	seen := make(map[int64]bool, len(rows))
	var refs []ProjectRef
	for _, r := range rows {
		if seen[r.ProjectID] {
			continue
		}
		seen[r.ProjectID] = true
		refs = append(refs, ProjectRef{ID: r.ProjectID, Name: r.ProjectName})
	}
	return refs, nil
}

func (c *impl) ListAffectedProjectsForVenue(ctx context.Context, tenantID, venueID int64) ([]ProjectRef, error) {
	rows, err := c.projects.ListAffectedProjectsForVenue(ctx, tenantID, venueID)
	if err != nil {
		return nil, err
	}
	refs := make([]ProjectRef, 0, len(rows))
	for _, r := range rows {
		refs = append(refs, ProjectRef{ID: r.ID, Name: r.Name})
	}
	return refs, nil
}

func (c *impl) ListAffectedProjectsForStaff(ctx context.Context, tenantID, staffID int64) ([]ProjectRef, error) {
	rows, err := c.projects.ListAffectedProjectsForStaff(ctx, tenantID, staffID)
	if err != nil {
		return nil, err
	}
	refs := make([]ProjectRef, 0, len(rows))
	for _, r := range rows {
		refs = append(refs, ProjectRef{ID: r.ID, Name: r.Name})
	}
	return refs, nil
}

func (c *impl) SeedDefaultMilestoneTemplate(ctx context.Context, tenantID int64) error {
	return c.milestoneTemplates.SeedDefaults(ctx, tenantID)
}
