package application

import (
	"context"
	"time"

	"elproof/internal/modules/projects/domain"
)

// CalendarQuery is one request for the cross-project calendar. Every query of
// the repository reads only this module's tables (projects, project_milestones,
// client_payment_terms, client_payments) and applies the same project filter:
// the workspace, not archived, not cancelled, and the scope below.
type CalendarQuery struct {
	TenantID int64
	From, To time.Time
	// AssignedStaffID limits the projects to those this person holds (PIC or
	// PIC Sales): set for a role that does not reach every project.
	AssignedStaffID *int64
	// PICStaffID is the optional "PIC" filter (projects whose PIC is this person).
	PICStaffID *int64
}

// CalendarRepository holds the calendar's three reads.
type CalendarRepository interface {
	// ListEventDays returns the projects whose event date falls in the range.
	ListEventDays(ctx context.Context, q CalendarQuery) ([]domain.CalendarItem, error)
	// ListOpenMilestones returns the schedule steps due in the range that are
	// not Completed or Cancelled.
	ListOpenMilestones(ctx context.Context, q CalendarQuery) ([]domain.CalendarItem, error)
	// ListTermsTouching returns every installment of the projects that have at
	// least one installment due in the range (the whole schedule: payments are
	// allocated over it), with what each of those projects' clients has paid
	// against the contract.
	ListTermsTouching(ctx context.Context, q CalendarQuery) ([]domain.TermWithProject, map[int64]int64, error)
}

// CalendarService assembles GET /calendar.
type CalendarService struct {
	repo CalendarRepository
}

func NewCalendarService(repo CalendarRepository) *CalendarService {
	return &CalendarService{repo: repo}
}

// List returns the entries of the kinds asked for, ordered by day. `asOf` is
// the instant whose WIB day decides whether an installment is overdue.
func (s *CalendarService) List(ctx context.Context, q CalendarQuery, kinds domain.CalendarKinds, asOf time.Time) ([]domain.CalendarItem, error) {
	items := []domain.CalendarItem{}
	if kinds.Has(domain.CalendarEvent) {
		events, err := s.repo.ListEventDays(ctx, q)
		if err != nil {
			return nil, err
		}
		items = append(items, events...)
	}
	if kinds.Has(domain.CalendarMilestone) {
		steps, err := s.repo.ListOpenMilestones(ctx, q)
		if err != nil {
			return nil, err
		}
		items = append(items, steps...)
	}
	if kinds.Has(domain.CalendarPayment) {
		terms, applied, err := s.repo.ListTermsTouching(ctx, q)
		if err != nil {
			return nil, err
		}
		items = append(items, domain.CalendarPaymentItems(terms, applied, q.From, q.To, domain.TodayWIB(asOf))...)
	}
	domain.SortCalendarItems(items)
	return items, nil
}
