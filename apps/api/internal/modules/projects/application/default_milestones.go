package application

import (
	"context"
	"time"

	"elproof/internal/modules/projects/domain"
)

// defaultMilestoneTemplate is the checklist a new project starts with, so the
// Milestone tab isn't blank by default — the common WO flow from first venue
// survey through hari-H. DaysBeforeEvent is clamped to PrepStartDate in
// seedDefaultMilestones for short-notice projects.
var defaultMilestoneTemplate = []struct {
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

func (s *ProjectService) seedDefaultMilestones(ctx context.Context, projectID int64, eventDate, prepStartDate time.Time) error {
	for i, tmpl := range defaultMilestoneTemplate {
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
