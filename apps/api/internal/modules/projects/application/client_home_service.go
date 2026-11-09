package application

import (
	"context"
	"time"

	"elproof/internal/modules/projects/domain"
)

// TenantContactReader is what the home needs from `platform` (declared here,
// bridged in cmd/server like the other ports): the business WhatsApp number and
// the bank account clients pay into. Primitive-typed so projects never imports
// platform. Empty strings mean "not filled in".
type TenantContactReader interface {
	PortalContact(ctx context.Context, tenantID int64) (whatsapp, bankName, accountNumber, accountHolder string, err error)
}

// Portal permissions that decide which sections of the home exist.
const (
	permPortalFinance  = "portal.finance.view"
	permPortalTimeline = "portal.timeline.view"
	permPortalIssues   = "portal.issues.view"
)

// ClientHomeService assembles GET /projects/me/home: everything the Client
// Portal "Beranda" shows, in one pass, limited to the sections the client's
// role may see (and computing nothing for the others).
type ClientHomeService struct {
	projects       *ProjectService
	clientPayments ClientPaymentRepository
	terms          *PaymentTermService
	evidence       *EvidenceService
	issues         IssueRepository
	contact        TenantContactReader
}

func NewClientHomeService(projects *ProjectService, clientPayments ClientPaymentRepository, terms *PaymentTermService, evidence *EvidenceService, issues IssueRepository) *ClientHomeService {
	return &ClientHomeService{projects: projects, clientPayments: clientPayments, terms: terms, evidence: evidence, issues: issues}
}

// SetTenantContact completes the two-phase wiring with `platform`.
func (s *ClientHomeService) SetTenantContact(c TenantContactReader) { s.contact = c }

type HomeProject struct {
	ID          int64
	Name        string
	BrideName   string
	GroomName   string
	EventDate   time.Time
	Venue       string
	PackageName string
	Status      string
	PICName     string
	PICPhone    string
}

type HomeProgress struct {
	OverallPercent int
	Condition      string
}

type HomeNextTerm struct {
	ID        int64
	Label     string
	Amount    int64
	DueDate   time.Time
	Status    string
	Remaining int64
}

type HomeFinance struct {
	ContractValue int64
	Paid          int64
	Remaining     int64
	NextTerm      *HomeNextTerm
	OverdueCount  int
}

type HomePayTo struct {
	BankName      string
	AccountNumber string
	AccountHolder string
}

// HomeUpdate is one "Kabar terbaru" line with names already resolved.
type HomeUpdate struct {
	Kind       domain.RecentUpdateKind
	Title      string
	At         time.Time
	Proof      *domain.UpdateProof
	VerifiedBy string
}

// ClientHome is the whole answer. A nil pointer / empty string means the
// section does not exist for this client.
type ClientHome struct {
	Project        HomeProject
	DaysToEvent    int
	Progress       *HomeProgress
	Finance        *HomeFinance
	PayTo          *HomePayTo
	WhatsApp       string
	Actions        []domain.HomeAction
	OpenIssueCount *int
	RecentUpdates  []HomeUpdate
	PICName        string
	PICPhone       string
}

// calendarDay is the instant a stored date (no time of day) counts as for
// ordering news: the start of that day in Jakarta, whatever zone it was parsed in.
func calendarDay(d time.Time, loc *time.Location) time.Time {
	return time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, loc)
}

// Get builds the home for the client's own project. `now` carries the Jakarta
// zone ("today" is the Indonesian calendar day, ADR-0007); `can` is the
// client's portal permission check.
func (s *ClientHomeService) Get(ctx context.Context, tenantID, projectID int64, can func(string) bool, now time.Time) (*ClientHome, error) {
	p, err := s.projects.Get(ctx, tenantID, projectID)
	if err != nil {
		return nil, err
	}
	canFinance, canTimeline, canIssues := can(permPortalFinance), can(permPortalTimeline), can(permPortalIssues)
	loc := now.Location()

	picName := s.projects.ResolvePICName(ctx, tenantID, p.PICStaffID)
	picPhone := s.projects.ResolvePICPhone(ctx, tenantID, p.PICStaffID)
	home := &ClientHome{
		Project: HomeProject{
			ID: p.ID, Name: p.Name, BrideName: p.BrideName, GroomName: p.GroomName, EventDate: p.EventDate, Venue: p.Venue,
			PackageName: p.PackageName, Status: string(p.Status), PICName: picName, PICPhone: picPhone,
		},
		DaysToEvent: domain.DaysUntil(now, p.EventDate),
		Actions:     []domain.HomeAction{},
		PICName:     picName,
		PICPhone:    picPhone,
	}

	// Each source is read at most once, and only when some section needs it.
	var (
		evidences []domain.Evidence
		updates   []domain.RecentUpdate
	)
	if canFinance || canTimeline || canIssues {
		if evidences, err = s.evidence.List(ctx, projectID); err != nil {
			return nil, err
		}
	}

	var termProgress []domain.TermProgress
	if canFinance {
		payments, err := s.clientPayments.ListByProject(ctx, projectID)
		if err != nil {
			return nil, err
		}
		if termProgress, err = s.terms.progressWith(ctx, projectID, payments, now); err != nil {
			return nil, err
		}
		applied := appliedFromPayments(payments)
		fin := &HomeFinance{
			ContractValue: p.ContractValue,
			Paid:          domain.PaidOnContract(applied),
			Remaining:     domain.RemainingToPay(p.ContractValue, applied),
			OverdueCount:  domain.OverdueTermCount(termProgress),
		}
		if t := domain.NextOpenTerm(termProgress); t != nil {
			fin.NextTerm = &HomeNextTerm{
				ID: t.Term.ID, Label: t.Term.Label, Amount: t.Term.Amount, DueDate: t.Term.DueDate,
				Status: string(t.Status), Remaining: t.Remaining,
			}
		}
		home.Finance = fin
		for _, pay := range payments {
			if !pay.IsVerified() {
				continue
			}
			updates = append(updates, domain.RecentUpdate{
				Kind: domain.UpdatePaymentVerified, Title: domain.PaymentVerifiedTitle(pay.Type), At: *pay.VerifiedAt,
				Proof:             domain.FirstVisibleProof(evidences, domain.RelatedClientPayment, pay.ID, can),
				VerifiedByStaffID: pay.VerifiedByStaffID, SourceID: pay.ID,
			})
		}
	}

	if canTimeline {
		progress, err := s.projects.ComputeProgressBatch(ctx, tenantID, []int64{projectID}, now)
		if err != nil {
			return nil, err
		}
		pr := progress[projectID]
		home.Progress = &HomeProgress{OverallPercent: pr.OverallPercent, Condition: string(pr.Condition)}
		milestones, err := s.projects.ListMilestones(ctx, tenantID, projectID)
		if err != nil {
			return nil, err
		}
		for _, m := range milestones {
			if m.Status != domain.MilestoneCompleted || m.CompletedDate == nil {
				continue
			}
			updates = append(updates, domain.RecentUpdate{
				Kind: domain.UpdateMilestoneDone, Title: m.Name + " selesai", At: calendarDay(*m.CompletedDate, loc),
				Proof: domain.FirstVisibleProof(evidences, domain.RelatedProjectMilestone, m.ID, can), SourceID: m.ID,
			})
		}
	}

	openIssues := 0
	if canIssues {
		issues, err := s.issues.ListByProject(ctx, projectID)
		if err != nil {
			return nil, err
		}
		for _, i := range issues {
			if i.Status.IsOpen() {
				openIssues++
				continue
			}
			if i.Status == domain.IssueResolved || i.Status == domain.IssueClosed {
				if i.ResolvedDate == nil {
					continue
				}
				updates = append(updates, domain.RecentUpdate{
					Kind: domain.UpdateIssueResolved, Title: i.Title + " sudah teratasi", At: calendarDay(*i.ResolvedDate, loc),
					Proof: domain.FirstVisibleProof(evidences, domain.RelatedIssue, i.ID, can), SourceID: i.ID,
				})
			}
		}
		home.OpenIssueCount = &openIssues
	}

	home.Actions = domain.SelectHomeActions(domain.HomeActionInput{
		Terms: termProgress, Today: now, OpenIssues: openIssues,
		ShowTerms: canFinance && p.Status != domain.StatusCancelled, ShowIssues: canIssues,
	})

	// Newest five, then one batched lookup for the names of whoever verified.
	top := domain.OrderRecentUpdates(updates)
	var verifierIDs []int64
	for _, u := range top {
		if u.Kind == domain.UpdatePaymentVerified {
			verifierIDs = append(verifierIDs, u.VerifiedByStaffID)
		}
	}
	names := s.projects.StaffNames(ctx, tenantID, verifierIDs)
	home.RecentUpdates = make([]HomeUpdate, 0, len(top))
	for _, u := range top {
		hu := HomeUpdate{Kind: u.Kind, Title: u.Title, At: u.At.In(loc), Proof: u.Proof}
		if u.Kind == domain.UpdatePaymentVerified {
			hu.VerifiedBy = names[u.VerifiedByStaffID]
		}
		home.RecentUpdates = append(home.RecentUpdates, hu)
	}

	if s.contact != nil {
		if whatsapp, bank, number, holder, err := s.contact.PortalContact(ctx, tenantID); err == nil {
			home.WhatsApp = whatsapp
			if canFinance && bank != "" && number != "" {
				home.PayTo = &HomePayTo{BankName: bank, AccountNumber: number, AccountHolder: holder}
			}
		}
	}
	return home, nil
}
