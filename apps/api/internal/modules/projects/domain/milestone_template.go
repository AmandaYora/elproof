package domain

import "time"

// ProjectMilestoneTemplate is one row of a tenant's configurable Timeline
// Default (PLAN.md) -- copied into a new project's ProjectMilestone rows at
// creation time, never referenced again afterward.
type ProjectMilestoneTemplate struct {
	ID              int64
	TenantID        int64
	SortOrder       int
	Name            string
	DaysBeforeEvent int
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
