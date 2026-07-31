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

type Contracts interface {
	ProjectExists(ctx context.Context, tenantID, projectID int64) (bool, error)
	// ProjectPICStaffID resolves a project's current PIC — used by `clients`
	// to scope a Wedding Planner's client-read access to only projects
	// they're PIC of (see PLAN.md's RBAC section).
	ProjectPICStaffID(ctx context.Context, tenantID, projectID int64) (int64, error)
	ListVendorEngagementHistory(ctx context.Context, tenantID, vendorID int64) ([]VendorEngagementHistoryItem, error)
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

func (c *impl) SeedDefaultMilestoneTemplate(ctx context.Context, tenantID int64) error {
	return c.milestoneTemplates.SeedDefaults(ctx, tenantID)
}
