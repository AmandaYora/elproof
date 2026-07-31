package application

import (
	"context"
	"time"

	"elproof/internal/modules/projects/domain"
)

// seedDefaultMilestones seeds a newly created project's Timeline tab from the
// tenant's own configurable template (PLAN.md's "Timeline Default Template")
// instead of a hardcoded list shared by every tenant -- see
// MilestoneTemplateService. DaysBeforeEvent is clamped to PrepStartDate for
// short-notice projects, same as before this became configurable. An empty
// template is not an error (PLAN.md's decision): the loop simply does nothing
// and the project's Timeline tab starts empty.
func (s *ProjectService) seedDefaultMilestones(ctx context.Context, tenantID, projectID int64, eventDate, prepStartDate time.Time) error {
	templates, err := s.milestoneTemplates.List(ctx, tenantID)
	if err != nil {
		return err
	}
	for i, tmpl := range templates {
		target := eventDate.AddDate(0, 0, -tmpl.DaysBeforeEvent)
		if target.Before(prepStartDate) {
			target = prepStartDate
		}
		m := &domain.ProjectMilestone{
			ProjectID: projectID, SortOrder: i + 1, Name: tmpl.Name,
			Status: domain.MilestoneNotStarted, TargetDate: target,
		}
		if err := s.milestones.Create(ctx, m); err != nil {
			return err
		}
	}
	return nil
}
