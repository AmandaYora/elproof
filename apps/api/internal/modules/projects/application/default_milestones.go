package application

import (
	"context"
	"time"

	"elproof/internal/modules/projects/domain"
)

// seedDefaultMilestones seeds a newly created project's Timeline tab from the
// tenant's own configurable template (PLAN.md's "Timeline Default Template")
// instead of a hardcoded list shared by every tenant -- see
// MilestoneTemplateService. Each step's target is domain.MilestoneTarget (event
// date minus the offset, pulled to the booking date -- prepStartDate, shown as
// "Tanggal Booking" -- only for a short-notice project), and the step keeps its
// offset so the target follows later changes of the event or booking date (EP-06,
// see shiftMilestones). An empty template is not an error (PLAN.md's decision):
// the loop simply does nothing and the project's Timeline tab starts empty.
func (s *ProjectService) seedDefaultMilestones(ctx context.Context, tenantID, projectID int64, eventDate, prepStartDate time.Time) error {
	templates, err := s.milestoneTemplates.List(ctx, tenantID)
	if err != nil {
		return err
	}
	for i, tmpl := range templates {
		offset := tmpl.DaysBeforeEvent
		m := &domain.ProjectMilestone{
			ProjectID: projectID, SortOrder: i + 1, Name: tmpl.Name,
			Status:     domain.MilestoneNotStarted,
			TargetDate: domain.MilestoneTarget(eventDate, prepStartDate, offset), DaysBeforeEvent: &offset,
		}
		if err := s.milestones.Create(ctx, m); err != nil {
			return err
		}
	}
	return nil
}

// shiftMilestones moves the project's Timeline steps that follow the event date
// (domain.ShiftMilestones) to the targets the new event and booking dates give.
// Steps dated by hand and finished or cancelled ones stay where they are. It
// returns how many steps moved.
func (s *ProjectService) shiftMilestones(ctx context.Context, projectID int64, eventDate, prepStartDate time.Time) (int, error) {
	milestones, err := s.milestones.ListByProject(ctx, projectID)
	if err != nil {
		return 0, err
	}
	moved := domain.ShiftMilestones(milestones, eventDate, prepStartDate)
	for i := range moved {
		if _, err := s.milestones.Patch(ctx, &moved[i], []string{"targetDate"}, nil); err != nil {
			return 0, err
		}
	}
	return len(moved), nil
}
