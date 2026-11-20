package application

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	offerscontracts "elproof/internal/modules/offers/contracts"
	"elproof/internal/modules/projects/domain"
	"elproof/internal/shared/apperror"
	"elproof/internal/shared/logger"
)

// QuotationSource is what "Jadikan proyek" (PLAN §5.4) needs of `offers`:
// offers/contracts.Quotations, wired in main.go (SetQuotations). Every call is
// scoped to the caller's tenant; another workspace's quotation answers 404.
type QuotationSource interface {
	AcceptedQuotation(ctx context.Context, tenantID, quotationID int64) (offerscontracts.AcceptedQuotation, error)
	MarkConverted(ctx context.Context, tenantID, quotationID, projectID int64) error
	MarkProjectDeleted(ctx context.Context, tenantID, projectID int64) error
}

// ProjectQuotationLookup finds the workspace's project made from a quotation
// (the MySQL project repository; nil, nil when there is none).
type ProjectQuotationLookup interface {
	FindByQuotation(ctx context.Context, tenantID, quotationID int64) (*domain.Project, error)
}

// Messages of the conversion.
const (
	msgQuotationNotFound      = "Penawaran tidak ditemukan"
	msgQuotationConverted     = "Penawaran ini sudah dijadikan proyek oleh anggota tim lain."
	codeQuotationConverted    = "quotation_already_converted"
	msgNoAgreement            = "Proyek ini tidak dibuat dari penawaran"
	msgConversionUnavailable  = "Konversi penawaran belum tersedia"
	msgEventDateRequired      = "Tanggal acara wajib diisi"
	msgNoEvents               = "Penawaran ini belum punya acara. Tambahkan acara di penawaran dulu."
	msgEventCount             = "Jumlah acara harus sama dengan acara di penawaran"
	msgPrepStartRequired      = "Tanggal mulai persiapan wajib diisi"
	msgDownPaymentDate        = "Tanggal tanda jadi wajib diisi"
	msgDownPaymentAboveValue  = "Tanda jadi tidak boleh melebihi nilai kontrak"
	capacityLeadConversion    = "Proyek belum bisa dibuat. "
	downPaymentFieldPrefix    = "downPayment."
	conversionActivityMessage = "Proyek baru dibuat dari penawaran "
)

// ConversionInput is what the "Jadikan proyek" page adds to the accepted
// revision (DESAIN §6.5).
type ConversionInput struct {
	// Name, BrideName, GroomName: empty = from the quotation's couple names
	// ("Ayu & Bima" → name "Ayu & Bima", bride "Ayu", groom "Bima").
	Name, BrideName, GroomName string
	// EventDates is the exact date of each event of the accepted revision, by
	// position; a zero date (or a missing one) keeps the revision's date, and
	// an event with neither is a 422 "events.N.eventDate".
	EventDates    []time.Time
	PrepStartDate time.Time
	PICStaffID    int64
	// PICSalesStaffID nil = the quotation's PIC.
	PICSalesStaffID *int64
	Description     string
	// DownPayment is the "tanda jadi" already received (optional): recorded as
	// the project's first client payment (type DP), with its proof.
	DownPayment *DownPaymentInput
	// FieldErrors are problems the handler already found (a bad date format).
	FieldErrors map[string][]string
}

// DownPaymentInput is the down payment received before the project existed.
// Its proof is uploaded with POST /projects/from-quotation/{id}/uploads.
type DownPaymentInput struct {
	Amount          int64
	PaymentDate     time.Time
	Method          string
	ReferenceNumber string
	Notes           string
	Evidence        []ProofItem
}

// ConversionResult is the project made (or found) for a quotation.
type ConversionResult struct {
	Project *domain.Project
	// Created is false when the quotation had been converted already (the
	// endpoint is idempotent per quotation).
	Created bool
	// DownPayment is the first client payment (nil without one).
	DownPayment *ClientPaymentCreated
}

// ConversionPreview is "Periksa sebelum membuat": what the project will be,
// computed without writing anything.
type ConversionPreview struct {
	Quotation     offerscontracts.AcceptedQuotation
	EventDate     time.Time
	ContractTotal int64
	Terms         []domain.PaymentTerm
	Events        int
	Suppliers     int
	Lines         int
	HasTermsText  bool
	// ExistingProjectID is set when the quotation is a project already.
	ExistingProjectID *int64
}

// ConversionService turns an Accepted quotation into a project.
type ConversionService struct {
	projects *ProjectService
	lines    *LineService
	payments *ClientPaymentService
	lookup   ProjectQuotationLookup
}

func NewConversionService(projects *ProjectService, lines *LineService, payments *ClientPaymentService, lookup ProjectQuotationLookup) *ConversionService {
	return &ConversionService{projects: projects, lines: lines, payments: payments, lookup: lookup}
}

// conversionPlan is a checked conversion, ready to write.
type conversionPlan struct {
	q       offerscontracts.AcceptedQuotation
	project domain.Project
	// events are the revision's events with their exact dates, by position.
	events      []domain.ProjectEvent
	mainIndex   int
	downPayment *ClientPaymentInput
	proofs      []ResolvedProof
}

func (s *ConversionService) source() (QuotationSource, error) {
	if s == nil || s.projects == nil || s.projects.quotations == nil || s.lookup == nil {
		return nil, apperror.Internal(msgConversionUnavailable)
	}
	return s.projects.quotations, nil
}

// Convert makes the project of an Accepted quotation of the caller's
// workspace (PLAN §5.4). Everything is checked first and every problem is
// returned together as one 422; then, in ONE projects transaction, the
// project (with quotation_id, the revision's PKP flag) is created with copies
// of the revision's events and sessions, vendors & venues (payer, show name,
// capacities, agreed cost; back to "Planned"), lines (rule copies, parents,
// events and suppliers remapped), the payment terms (amounts from the
// project's own contract total, due dates from the exact dates), the default
// Timeline, and the optional down payment with its proof. After the commit
// `offers` is told (MarkConverted). A quotation already converted answers its
// project (Created false) and repairs the mark; two concurrent requests give
// one project (the unique key decides, the loser reads the winner's).
func (s *ConversionService) Convert(ctx context.Context, tenantID int64, actor ProjectActor, quotationID int64, input ConversionInput, today time.Time) (*ConversionResult, error) {
	src, err := s.source()
	if err != nil {
		return nil, err
	}
	plan, existing, err := s.prepare(ctx, tenantID, actor, quotationID, input)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return s.found(ctx, src, tenantID, actor, quotationID, existing)
	}
	var out ConversionResult
	err = s.projects.txOrDirect().InTx(ctx, func(ctx context.Context) error {
		p, dp, err := s.write(ctx, tenantID, actor.StaffID, plan, today)
		if err != nil {
			return err
		}
		out = ConversionResult{Project: p, Created: true, DownPayment: dp}
		return nil
	})
	if errors.Is(err, domain.ErrQuotationConverted) {
		existing, ferr := s.lookup.FindByQuotation(ctx, tenantID, quotationID)
		if ferr != nil {
			return nil, ferr
		}
		if existing == nil {
			return nil, err
		}
		return s.found(ctx, src, tenantID, actor, quotationID, existing)
	}
	if err != nil {
		return nil, err
	}
	if err := src.MarkConverted(ctx, tenantID, quotationID, out.Project.ID); err != nil {
		// The project is the source of truth; the mark is a cache the next
		// call of this endpoint repairs (PLAN §5.4).
		logger.Error("quotation %d converted to project %d but MarkConverted failed: %v", quotationID, out.Project.ID, err)
	}
	return &out, nil
}

// found answers the project a quotation was already converted to, and repairs
// the mark in `offers`. A person limited to their own projects who holds
// neither PIC slot of it learns only that it exists.
func (s *ConversionService) found(ctx context.Context, src QuotationSource, tenantID int64, actor ProjectActor, quotationID int64, p *domain.Project) (*ConversionResult, error) {
	if actor.AssignedOnly && p.PICStaffID != actor.StaffID && p.PICSalesStaffID != actor.StaffID {
		return nil, apperror.Conflict(msgQuotationConverted).WithCode(codeQuotationConverted)
	}
	if err := src.MarkConverted(ctx, tenantID, quotationID, p.ID); err != nil {
		logger.Error("repairing the conversion mark of quotation %d (project %d) failed: %v", quotationID, p.ID, err)
	}
	return &ConversionResult{Project: p, Created: false}, nil
}

// Preview checks a conversion like Convert and answers what it would create,
// writing nothing ("Periksa sebelum membuat").
func (s *ConversionService) Preview(ctx context.Context, tenantID int64, actor ProjectActor, quotationID int64, input ConversionInput) (*ConversionPreview, error) {
	if _, err := s.source(); err != nil {
		return nil, err
	}
	plan, existing, err := s.prepare(ctx, tenantID, actor, quotationID, input)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		id := existing.ID
		q, err := s.projects.quotations.AcceptedQuotation(ctx, tenantID, quotationID)
		if err != nil {
			return nil, err
		}
		return &ConversionPreview{Quotation: q, EventDate: existing.EventDate, ContractTotal: existing.ContractTotal, ExistingProjectID: &id,
			Terms: []domain.PaymentTerm{}}, nil
	}
	total := plan.q.Revision.ContractTotal
	return &ConversionPreview{
		Quotation: plan.q, EventDate: plan.project.EventDate, ContractTotal: total,
		Terms:  conversionTerms(plan.q, plan.events, plan.mainIndex, total),
		Events: len(plan.q.Content.Events), Suppliers: len(plan.q.Content.Suppliers), Lines: len(plan.q.Content.Lines),
		HasTermsText: strings.TrimSpace(plan.q.Content.TermsText) != "",
	}, nil
}

// prepare reads the accepted revision and checks the input. existing is the
// project already made from the quotation (then plan is nil).
func (s *ConversionService) prepare(ctx context.Context, tenantID int64, actor ProjectActor, quotationID int64, input ConversionInput) (*conversionPlan, *domain.Project, error) {
	src := s.projects.quotations
	q, err := src.AcceptedQuotation(ctx, tenantID, quotationID)
	if err != nil {
		return nil, nil, err
	}
	// A person limited to their own work reaches only the quotations they are
	// PIC of (as in `offers`); any other answers like a missing one.
	if actor.AssignedOnly && q.PICStaffID != actor.StaffID {
		return nil, nil, apperror.NotFound(msgQuotationNotFound)
	}
	existing, err := s.lookup.FindByQuotation(ctx, tenantID, quotationID)
	if err != nil {
		return nil, nil, err
	}
	if existing != nil {
		return nil, existing, nil
	}

	fe := newFieldErrors()
	fe.mergeFields(input.FieldErrors)

	// Exact dates.
	content := q.Content
	if len(content.Events) == 0 {
		fe.add(msgNoEvents, "events")
	}
	if len(input.EventDates) > len(content.Events) {
		fe.add(msgEventCount, "events")
	}
	events := make([]domain.ProjectEvent, len(content.Events))
	mainIndex := -1
	for i, se := range content.Events {
		var date time.Time
		if i < len(input.EventDates) && !input.EventDates[i].IsZero() {
			date = input.EventDates[i]
		} else if se.EventDate != nil {
			if d, err := time.Parse(dateLayoutISO, *se.EventDate); err == nil {
				date = d
			}
		}
		key := "events." + strconv.Itoa(i) + ".eventDate"
		if date.IsZero() && len(input.FieldErrors[key]) == 0 {
			fe.add(msgEventDateRequired, key)
		}
		events[i] = snapshotEvent(se, date)
		if !date.IsZero() && (mainIndex < 0 || date.Before(events[mainIndex].EventDate)) {
			mainIndex = i
		}
	}
	if input.PrepStartDate.IsZero() && len(input.FieldErrors["prepStartDate"]) == 0 {
		fe.add(msgPrepStartRequired, "prepStartDate")
	}

	// PIC and PIC Sales (by default the quotation's PIC). A person limited to
	// their own projects stays able to reach the one they create.
	picSales := q.PICStaffID
	if input.PICSalesStaffID != nil {
		picSales = *input.PICSalesStaffID
	}
	if actor.AssignedOnly && input.PICStaffID != actor.StaffID {
		picSales = actor.StaffID
	}
	if err := s.projects.checkPICs(ctx, tenantID, input.PICStaffID, picSales); err != nil && !fe.mergeErr(err) {
		return nil, nil, err
	}

	// Capacity, against the capacities the revision copied (PLAN §4.1).
	if err := s.projects.eventService.capacityRefusal(capacityLeadConversion, venueEvents(events, content.Events), snapshotVenues(content.Suppliers)); err != nil && !fe.mergeErr(err) {
		return nil, nil, err
	}

	// The down payment and its proof (uploaded before the project existed).
	var dp *ClientPaymentInput
	var proofs []ResolvedProof
	if d := input.DownPayment; d != nil {
		dp = &ClientPaymentInput{Type: domain.PaymentDP, Amount: d.Amount, PaymentDate: d.PaymentDate,
			Method: d.Method, ReferenceNumber: d.ReferenceNumber, Notes: d.Notes}
		if d.PaymentDate.IsZero() && len(input.FieldErrors[downPaymentFieldPrefix+"paymentDate"]) == 0 {
			fe.add(msgDownPaymentDate, downPaymentFieldPrefix+"paymentDate")
		}
		if d.Amount > q.Revision.ContractTotal {
			fe.add(msgDownPaymentAboveValue, downPaymentFieldPrefix+"amount")
		}
		if s.payments == nil {
			return nil, nil, apperror.Internal(msgConversionUnavailable)
		}
		resolved, err := s.payments.prepareWithProof(ctx, tenantID, 0, *dp, d.Evidence)
		if err != nil {
			ae, ok := apperror.As(err)
			if !ok || ae.Kind != apperror.KindValidation {
				return nil, nil, err
			}
			fe.mergeFields(prefixFields(ae.Fields, downPaymentFieldPrefix))
		}
		proofs = resolved
	}
	if err := fe.err(); err != nil {
		return nil, nil, err
	}
	if err := s.projects.checkActiveProjectQuota(ctx, tenantID); err != nil {
		return nil, nil, err
	}

	name, bride, groom := coupleNames(q.CoupleNames)
	if v := strings.TrimSpace(input.Name); v != "" {
		name = v
	}
	if v := strings.TrimSpace(input.BrideName); v != "" {
		bride = v
	}
	if v := strings.TrimSpace(input.GroomName); v != "" {
		groom = v
	}
	if name == "" {
		name = q.Number
	}
	qid := q.ID
	return &conversionPlan{
		q: q,
		project: domain.Project{
			TenantID: tenantID, Name: name, BrideName: bride, GroomName: groom,
			EventDate: events[mainIndex].EventDate, PrepStartDate: input.PrepStartDate, IsPKP: q.IsPKP,
			Status: domain.StatusDraft, PICStaffID: input.PICStaffID, PICSalesStaffID: picSales, Description: input.Description,
			QuotationID: &qid, QuotationNumber: q.Number, QuotationRevisionN: q.Revision.N,
		},
		events: events, mainIndex: mainIndex, downPayment: dp, proofs: proofs,
	}, nil, nil
}

// write creates the project and its copies on the transaction ctx carries.
func (s *ConversionService) write(ctx context.Context, tenantID, actorStaffID int64, plan *conversionPlan, today time.Time) (*domain.Project, *ClientPaymentCreated, error) {
	ps := s.projects
	content := plan.q.Content
	p := plan.project
	if err := ps.repo.Create(ctx, &p); err != nil {
		return nil, nil, err
	}

	supplierIDs := make([]int64, len(content.Suppliers))
	for i, ss := range content.Suppliers {
		sup := snapshotSupplier(ss)
		sup.TenantID, sup.ProjectID = tenantID, p.ID
		if err := ps.suppliers.Create(ctx, &sup); err != nil {
			return nil, nil, err
		}
		supplierIDs[i] = sup.ID
	}

	eventIDs := make([]int64, len(plan.events))
	for i, e := range plan.events {
		e.TenantID, e.ProjectID, e.IsMain, e.SortOrder = tenantID, p.ID, i == plan.mainIndex, i
		if vi := content.Events[i].VenueIndex; vi != nil && *vi >= 0 && *vi < len(supplierIDs) {
			id := supplierIDs[*vi]
			e.VenueSupplierID = &id
		}
		if err := ps.events.Create(ctx, &e); err != nil {
			return nil, nil, err
		}
		eventIDs[i] = e.ID
	}

	if err := s.copyLines(ctx, tenantID, p.ID, actorStaffID, content.Lines, eventIDs, supplierIDs); err != nil {
		return nil, nil, err
	}
	saved, err := ps.repo.FindByID(ctx, tenantID, p.ID)
	if err != nil {
		return nil, nil, err
	}
	if saved == nil {
		return nil, nil, apperror.NotFound("Proyek tidak ditemukan")
	}

	if terms := conversionTerms(plan.q, plan.events, plan.mainIndex, saved.ContractTotal); len(terms) > 0 && saved.ContractTotal > 0 {
		if ps.terms == nil {
			return nil, nil, apperror.Internal(msgConversionUnavailable)
		}
		for i := range terms {
			terms[i].ProjectID = p.ID
		}
		if err := ps.terms.ReplaceAll(ctx, p.ID, terms); err != nil {
			return nil, nil, err
		}
	}
	if err := ps.seedDefaultMilestones(ctx, tenantID, p.ID, p.EventDate, p.PrepStartDate); err != nil {
		return nil, nil, err
	}
	ps.activity.Record(ctx, &p.ID, domain.ActivityProjectCreated, actorStaffID, "project", formatID(p.ID), p.Name,
		conversionActivityMessage+plan.q.Number+" Revisi "+strconv.Itoa(plan.q.Revision.N))

	var dp *ClientPaymentCreated
	if plan.downPayment != nil {
		if dp, err = s.payments.recordWithProof(ctx, tenantID, p.ID, actorStaffID, *plan.downPayment, plan.proofs, today); err != nil {
			return nil, nil, err
		}
	}
	return saved, dp, nil
}

// copyLines inserts the revision's lines in its order (parents first, so each
// child finds its package's copy), then recomputes the extra-guest charges
// for the copied events and writes the contract cache.
func (s *ConversionService) copyLines(ctx context.Context, tenantID, projectID, actorStaffID int64, lines []offerscontracts.SnapshotLine, eventIDs, supplierIDs []int64) error {
	if s.lines == nil {
		return apperror.Internal(msgConversionUnavailable)
	}
	at := func(ix *int, ids []int64) *int64 {
		if ix == nil || *ix < 0 || *ix >= len(ids) {
			return nil
		}
		id := ids[*ix]
		return &id
	}
	lineIDs := make([]int64, len(lines))
	for pass := 0; pass < 2; pass++ {
		for i, sl := range lines {
			if (pass == 0) != (sl.ParentIndex == nil) {
				continue
			}
			l := snapshotLine(sl)
			l.TenantID, l.ProjectID, l.SortOrder = tenantID, projectID, i
			l.EventID, l.SupplierRef = at(sl.EventIndex, eventIDs), at(sl.SupplierIndex, supplierIDs)
			if sl.ParentIndex != nil {
				l.ParentLineID = at(sl.ParentIndex, lineIDs)
				if l.ParentLineID == nil || *l.ParentLineID == 0 {
					continue
				}
			}
			if err := s.lines.repo.Insert(ctx, &l); err != nil {
				return err
			}
			lineIDs[i] = l.ID
		}
	}
	if s.lines.offers == nil {
		return s.lines.repo.RefreshContract(ctx, tenantID, projectID, true)
	}
	return s.lines.recompute(ctx, tenantID, projectID, actorStaffID, true)
}

// Agreement is "Kesepakatan awal": the accepted revision a project was made
// from, read from `offers` (404 for a project made without a quotation).
func (s *ConversionService) Agreement(ctx context.Context, tenantID, projectID int64) (*offerscontracts.AcceptedQuotation, error) {
	src, err := s.source()
	if err != nil {
		return nil, err
	}
	p, err := s.projects.Get(ctx, tenantID, projectID)
	if err != nil {
		return nil, err
	}
	if p.QuotationID == nil {
		return nil, apperror.NotFound(msgNoAgreement)
	}
	q, err := src.AcceptedQuotation(ctx, tenantID, *p.QuotationID)
	if err != nil {
		return nil, err
	}
	return &q, nil
}

// ---- copies of the frozen revision -------------------------------------------

const dateLayoutISO = "2006-01-02"

func snapshotEvent(se offerscontracts.SnapshotEvent, date time.Time) domain.ProjectEvent {
	e := domain.ProjectEvent{
		EventKindID: se.EventKindID, KindName: se.KindName, EventDate: date,
		StartTime: se.StartTime, EndTime: se.EndTime, Layout: domain.Layout(se.Layout),
	}
	for i, ss := range se.Sessions {
		e.Sessions = append(e.Sessions, domain.EventSession{Label: ss.Label, StartTime: ss.StartTime, EndTime: ss.EndTime, Guests: ss.Guests, SortOrder: i})
	}
	return e
}

func snapshotSupplier(ss offerscontracts.SnapshotSupplier) domain.ProjectSupplier {
	sup := domain.ProjectSupplier{
		Kind: domain.SupplierKind(ss.Kind), SupplierID: ss.SupplierID, Name: ss.Name, PaidBy: domain.Payer(ss.PaidBy),
		ShowName: ss.ShowName, CapacityStanding: ss.CapacityStanding, CapacitySeated: ss.CapacitySeated,
		AgreedCost: ss.AgreedCost, EngagementStatus: domain.EngagementPlanned, Notes: ss.Notes,
	}
	if ss.CategoryID != nil {
		sup.CategoryID = *ss.CategoryID
	}
	return sup
}

// snapshotLine is a frozen line as a project line. A project line has no
// "details" text: it stays readable in Kesepakatan awal.
func snapshotLine(sl offerscontracts.SnapshotLine) domain.ProjectLine {
	l := domain.ProjectLine{
		Kind: domain.LineKind(sl.Kind), Description: sl.Description, CategoryID: sl.CategoryID,
		Quantity: sl.Quantity, UnitPrice: sl.UnitPrice, Amount: sl.Amount, IsBonus: sl.IsBonus,
	}
	if sl.Unit != nil {
		l.Unit = *sl.Unit
	}
	if sl.PackageRule != nil {
		r := fromOfferRule(*sl.PackageRule)
		l.PackageRule = &r
	}
	return l
}

// venueEvents are the events as the capacity rule reads them: each at its
// venue's position in the revision (position + 1 stands in for an id).
func venueEvents(events []domain.ProjectEvent, frozen []offerscontracts.SnapshotEvent) []domain.ProjectEvent {
	out := make([]domain.ProjectEvent, len(events))
	for i, e := range events {
		out[i] = e
		if vi := frozen[i].VenueIndex; vi != nil {
			id := int64(*vi) + 1
			out[i].VenueSupplierID = &id
		}
	}
	return out
}

// snapshotVenues are the revision's venues with their copied capacities,
// keyed like venueEvents.
func snapshotVenues(suppliers []offerscontracts.SnapshotSupplier) map[int64]domain.ProjectSupplier {
	out := map[int64]domain.ProjectSupplier{}
	for i, ss := range suppliers {
		if ss.Kind != offerscontracts.SupplierVenue {
			continue
		}
		sup := snapshotSupplier(ss)
		sup.ID = int64(i) + 1
		out[sup.ID] = sup
	}
	return out
}

// conversionTerms is the project's payment schedule: the revision's terms,
// with what each asks for of the project's contract total (the quotation's
// rule) and a due date from the exact dates: "saat diterima" counts from the
// day of acceptance, an event term from its event (the main event when none
// is named).
func conversionTerms(q offerscontracts.AcceptedQuotation, events []domain.ProjectEvent, mainIndex int, total int64) []domain.PaymentTerm {
	frozen := q.Content.Terms
	amounts, _ := offerscontracts.TermAmountsOf(frozen, total)
	accepted := q.Revision.CreatedAt
	if q.AcceptedOn != nil {
		accepted = *q.AcceptedOn
	}
	accepted = civilDay(accepted)
	out := make([]domain.PaymentTerm, 0, len(frozen))
	for i, t := range frozen {
		base := accepted
		if t.DueBasis == offerscontracts.DueOnEvent && mainIndex >= 0 {
			ix := mainIndex
			if t.DueEventIndex != nil && *t.DueEventIndex >= 0 && *t.DueEventIndex < len(events) && !events[*t.DueEventIndex].EventDate.IsZero() {
				ix = *t.DueEventIndex
			}
			base = civilDay(events[ix].EventDate)
		}
		out = append(out, domain.PaymentTerm{Label: t.Label, Amount: amounts[i], DueDate: base.AddDate(0, 0, t.DueOffsetDays)})
	}
	return out
}

// civilDay is the calendar day of t (in its own location) at midnight UTC,
// the way dates are stored.
func civilDay(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// coupleNames splits "Ayu & Bima" (or "Ayu dan Bima") into the project's name,
// bride and groom.
func coupleNames(couple string) (name, bride, groom string) {
	name = strings.TrimSpace(couple)
	for _, sep := range []string{"&", " dan "} {
		if parts := strings.SplitN(name, sep, 2); len(parts) == 2 {
			return name, strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
		}
	}
	return name, "", ""
}

// prefixFields puts every key of a validation error under prefix
// ("amount" → "downPayment.amount").
func prefixFields(fields map[string][]string, prefix string) map[string][]string {
	out := make(map[string][]string, len(fields))
	for k, v := range fields {
		out[prefix+k] = v
	}
	return out
}

// CheckQuotation confirms the caller may convert the quotation (it is an
// Accepted quotation of the workspace they reach): what the down payment's
// upload needs before it signs anything.
func (s *ConversionService) CheckQuotation(ctx context.Context, tenantID int64, actor ProjectActor, quotationID int64) error {
	src, err := s.source()
	if err != nil {
		return err
	}
	q, err := src.AcceptedQuotation(ctx, tenantID, quotationID)
	if err != nil {
		return err
	}
	if actor.AssignedOnly && q.PICStaffID != actor.StaffID {
		return apperror.NotFound(msgQuotationNotFound)
	}
	return nil
}
