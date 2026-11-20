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
	permPortalTasks    = "portal.tasks.complete"
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
	lines          *LineService
	supplierBills  *SupplierBillService
}

// SetSupplierBills wires the bills of the vendors the client pays directly
// (Paket WO Tahap 4): the home's next vendor bill and its action.
func (s *ClientHomeService) SetSupplierBills(b *SupplierBillService) { s.supplierBills = b }

// SetLines wires the Rincian (the package names the home shows).
func (s *ClientHomeService) SetLines(l *LineService) { s.lines = l }

func NewClientHomeService(projects *ProjectService, clientPayments ClientPaymentRepository, terms *PaymentTermService, evidence *EvidenceService, issues IssueRepository) *ClientHomeService {
	return &ClientHomeService{projects: projects, clientPayments: clientPayments, terms: terms, evidence: evidence, issues: issues}
}

// SetTenantContact completes the two-phase wiring with `platform`.
func (s *ClientHomeService) SetTenantContact(c TenantContactReader) { s.contact = c }

type HomeProject struct {
	ID        int64
	Name      string
	BrideName string
	GroomName string
	EventDate time.Time
	// MainEventKind, Location and EventsLine describe the events (the main one's
	// kind, where they happen, the others); empty without portal.timeline.view.
	MainEventKind string
	Location      string
	EventsLine    string
	// PackageNames are the project's packages (Rincian), only with
	// portal.vendor.view (PortalPolicy.Home().Services).
	PackageNames []string
	Status       string
	PICName      string
	PICPhone     string
	// CoverPhotoPath/CoverUpdatedAt: the hero photo (ADR-0047); nil without one.
	CoverPhotoPath *string
	CoverUpdatedAt *time.Time
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
	// ContractValue is the contract total (subtotal + PPN); ContractTax the PPN
	// in it (0 when the project is not PKP).
	ContractValue int64
	ContractTax   int64
	Paid          int64
	Remaining     int64
	NextTerm      *HomeNextTerm
	OverdueCount  int
	// NextSupplierBill is the earliest bill not fully paid of a vendor the
	// client pays directly ("tagihan berikutnya", Paket WO Tahap 4); nil when
	// none. It is outside the contract: Paid and Remaining above never include it.
	NextSupplierBill *HomeSupplierBill
}

// HomeSupplierBill is one bill of a vendor the client pays directly, with the
// vendor's real name (the client pays it, so it is always named).
type HomeSupplierBill struct {
	BillTermID        int64
	ProjectSupplierID int64
	SupplierName      string
	Label             string
	Amount            int64
	DueDate           time.Time
	Status            string
	Remaining         int64
	// Pending is what the client already sent for this bill and the WO has
	// not checked yet.
	Pending int64
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
	// ClientTasks: the steps flagged for the couple (domain.SelectClientTasks);
	// empty when the role sees neither the timeline nor may complete tasks.
	ClientTasks []domain.ProjectMilestone
}

// describeEvents fills the main event's kind, the location line and the other
// events of the project (venues are always named to the client).
func (s *ClientHomeService) describeEvents(ctx context.Context, projectID int64, hp *HomeProject) error {
	events, err := s.projects.events.ListByProject(ctx, projectID)
	if err != nil {
		return err
	}
	suppliers, err := s.projects.suppliers.ListByProject(ctx, projectID)
	if err != nil {
		return err
	}
	names := map[int64]string{}
	for _, sup := range suppliers {
		if sup.Kind == domain.SupplierVenue {
			names[sup.ID] = sup.Name
		}
	}
	if m := domain.MainEvent(events); m != nil {
		hp.MainEventKind = m.KindName
	}
	hp.Location = domain.LocationLine(events, names)
	hp.EventsLine = domain.EventsLine(events)
	return nil
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
	policy := domain.NewPortalPolicy(can)
	canFinance, canTimeline, canIssues := can(permPortalFinance), can(permPortalTimeline), can(permPortalIssues)
	loc := now.Location()

	picName := s.projects.ResolvePICName(ctx, tenantID, p.PICStaffID)
	picPhone := s.projects.ResolvePICPhone(ctx, tenantID, p.PICStaffID)
	home := &ClientHome{
		Project: HomeProject{
			ID: p.ID, Name: p.Name, BrideName: p.BrideName, GroomName: p.GroomName, EventDate: p.EventDate,
			PackageNames: []string{}, Status: string(p.Status), PICName: picName, PICPhone: picPhone,
			CoverPhotoPath: p.CoverPhotoPath, CoverUpdatedAt: p.CoverUpdatedAt,
		},
		DaysToEvent: domain.DaysUntil(now, p.EventDate),
		Actions:     []domain.HomeAction{},
		ClientTasks: []domain.ProjectMilestone{},
		PICName:     picName,
		PICPhone:    picPhone,
	}

	if policy.Home().Services {
		if names := s.lines.PackageNames(ctx, tenantID, []int64{projectID})[projectID]; names != nil {
			home.Project.PackageNames = names
		}
	}
	if policy.Events() {
		if err := s.describeEvents(ctx, projectID, &home.Project); err != nil {
			return nil, err
		}
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
	var supplierBillsDue []domain.SupplierBillDue
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
			ContractValue: p.ContractTotal,
			ContractTax:   p.ContractTax,
			Paid:          domain.PaidOnContract(applied),
			Remaining:     domain.RemainingToPay(p.ContractTotal, applied),
			OverdueCount:  domain.OverdueTermCount(termProgress),
		}
		if t := domain.NextOpenTerm(termProgress); t != nil {
			fin.NextTerm = &HomeNextTerm{
				ID: t.Term.ID, Label: t.Term.Label, Amount: t.Term.Amount, DueDate: t.Term.DueDate,
				Status: string(t.Status), Remaining: t.Remaining,
			}
		}
		if s.supplierBills != nil {
			bills, err := s.supplierBills.Bills(ctx, tenantID, projectID, now)
			if err != nil {
				return nil, err
			}
			var visible []domain.SupplierBill
			for _, b := range bills {
				if policy.BillTerm(b.Supplier) {
					visible = append(visible, b)
				}
			}
			supplierBillsDue = domain.OpenSupplierBills(visible)
			if next := domain.NextSupplierBill(visible); next != nil {
				t := next.Bill
				fin.NextSupplierBill = &HomeSupplierBill{
					BillTermID: t.Term.ID, ProjectSupplierID: t.Term.ProjectSupplierID, SupplierName: next.SupplierName,
					Label: t.Term.Label, Amount: t.Term.Amount, DueDate: t.Term.DueDate, Status: string(t.Status),
					Remaining: t.Remaining, Pending: t.Pending,
				}
			}
		}
		home.Finance = fin
		for _, pay := range payments {
			if !pay.IsVerified() {
				continue
			}
			updates = append(updates, domain.RecentUpdate{
				Kind: domain.UpdatePaymentVerified, Title: domain.PaymentVerifiedTitle(pay.Type), At: *pay.VerifiedAt,
				Proof:             domain.FirstVisibleProof(evidences, domain.RelatedClientPayment, pay.ID, policy),
				VerifiedByStaffID: pay.VerifiedByStaffID, SourceID: pay.ID,
			})
		}
	}

	var milestones []domain.ProjectMilestone
	if canTimeline || can(permPortalTasks) {
		if milestones, err = s.projects.ListMilestones(ctx, tenantID, projectID); err != nil {
			return nil, err
		}
		home.ClientTasks = domain.SelectClientTasks(milestones)
	}

	if canTimeline {
		progress, err := s.projects.ComputeClientProgressBatch(ctx, tenantID, []int64{projectID}, now)
		if err != nil {
			return nil, err
		}
		pr := progress[projectID]
		home.Progress = &HomeProgress{OverallPercent: pr.OverallPercent, Condition: string(pr.Condition)}
		for _, m := range milestones {
			if m.Status != domain.MilestoneCompleted || m.CompletedDate == nil {
				continue
			}
			updates = append(updates, domain.RecentUpdate{
				Kind: domain.UpdateMilestoneDone, Title: m.Name + " selesai", At: calendarDay(*m.CompletedDate, loc),
				Proof: domain.FirstVisibleProof(evidences, domain.RelatedProjectMilestone, m.ID, policy), SourceID: m.ID,
			})
		}
	}

	openIssues := 0
	if canIssues {
		issues, err := s.issues.ListByProject(ctx, projectID)
		if err != nil {
			return nil, err
		}
		// Only what staff shared with the client counts: a hidden kendala is neither
		// counted, nor an action, nor "sudah teratasi" news.
		for _, i := range domain.ClientVisibleIssues(issues) {
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
					Proof: domain.FirstVisibleProof(evidences, domain.RelatedIssue, i.ID, policy), SourceID: i.ID,
				})
			}
		}
		home.OpenIssueCount = &openIssues
	}

	home.Actions = domain.SelectHomeActions(domain.HomeActionInput{
		Terms: termProgress, Today: now, OpenIssues: openIssues,
		ShowTerms: canFinance && p.Status != domain.StatusCancelled, ShowIssues: canIssues,
		SupplierBills: supplierBillsDue,
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

	// contact.whatsapp: the PIC's phone first, then the business WhatsApp of the
	// workspace profile (ADR-0047); the section is absent when both are empty.
	businessWhatsApp := ""
	if s.contact != nil {
		if whatsapp, bank, number, holder, err := s.contact.PortalContact(ctx, tenantID); err == nil {
			businessWhatsApp = whatsapp
			if canFinance && bank != "" && number != "" {
				home.PayTo = &HomePayTo{BankName: bank, AccountNumber: number, AccountHolder: holder}
			}
		}
	}
	home.WhatsApp = domain.PortalWhatsApp(picPhone, businessWhatsApp)
	return home, nil
}
