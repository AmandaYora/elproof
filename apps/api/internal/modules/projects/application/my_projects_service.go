package application

import (
	"context"
	"sort"
	"time"

	"elproof/internal/modules/projects/domain"
)

// maxMyProjects bounds the "Proyek saya" page: a person holding more open
// projects than this is not looking at a card list anyway.
const maxMyProjects = 30

const dateLayout = "2006-01-02"

// MyProject is one project the signed-in staff member holds (as PIC or PIC
// Sales), with what needs their attention.
type MyProject struct {
	Project          domain.Project
	ProgressPercent  int
	OpenIssues       int
	OverdueTerms     int
	OverdueTotal     int64
	NextTerm         *domain.TermProgress
	TasksDueThisWeek []domain.ProjectMilestone
}

// MyProjectsService backs the landing page for roles limited to their own
// projects (the full dashboard aggregates every project, so they cannot have it).
type MyProjectsService struct {
	projects *ProjectService
	terms    *PaymentTermService
}

func NewMyProjectsService(projects *ProjectService, terms *PaymentTermService) *MyProjectsService {
	return &MyProjectsService{projects: projects, terms: terms}
}

// Get returns the person's open projects, soonest event first.
func (s *MyProjectsService) Get(ctx context.Context, tenantID, staffID int64, today time.Time) ([]MyProject, error) {
	all, err := s.projects.List(ctx, tenantID, &staffID, &staffID)
	if err != nil {
		return nil, err
	}
	var open []domain.Project
	for _, p := range all {
		if isActiveProject(p.Status, p.IsArchived) {
			open = append(open, p)
		}
	}
	// Upcoming events first (soonest first), then events that already passed
	// but were never completed; the cap must cut those, not what is coming up.
	todayStr := today.Format(dateLayout)
	sort.Slice(open, func(i, j int) bool {
		iPast, jPast := open[i].EventDate.Format(dateLayout) < todayStr, open[j].EventDate.Format(dateLayout) < todayStr
		if iPast != jPast {
			return !iPast
		}
		return open[i].EventDate.Before(open[j].EventDate)
	})
	if len(open) > maxMyProjects {
		open = open[:maxMyProjects]
	}
	ids := make([]int64, len(open))
	for i, p := range open {
		ids[i] = p.ID
	}
	progress, err := s.projects.ComputeProgressBatch(ctx, tenantID, ids, today)
	if err != nil {
		return nil, err
	}

	// Dates are compared as calendar days (YYYY-MM-DD), so neither the server's
	// zone nor the zone dates were parsed in can shift a day.
	fromDay, toDay := todayStr, today.AddDate(0, 0, 7).Format(dateLayout)
	out := make([]MyProject, 0, len(open))
	for _, p := range open {
		mp := MyProject{Project: p, ProgressPercent: progress[p.ID].OverallPercent, OpenIssues: progress[p.ID].OpenIssueCount}

		terms, err := s.terms.List(ctx, tenantID, p.ID, today)
		if err != nil {
			return nil, err
		}
		for _, t := range terms {
			if t.Status == domain.TermPaid {
				continue
			}
			if t.Status == domain.TermOverdue {
				mp.OverdueTerms++
				mp.OverdueTotal += t.Remaining
			}
			if mp.NextTerm == nil {
				next := t
				mp.NextTerm = &next
			}
		}

		milestones, err := s.projects.ListMilestones(ctx, tenantID, p.ID)
		if err != nil {
			return nil, err
		}
		for _, m := range milestones {
			if m.Status == domain.MilestoneCompleted || m.Status == domain.MilestoneCancelled {
				continue
			}
			if day := m.TargetDate.Format(dateLayout); day >= fromDay && day <= toDay {
				mp.TasksDueThisWeek = append(mp.TasksDueThisWeek, m)
			}
		}
		out = append(out, mp)
	}
	return out, nil
}
