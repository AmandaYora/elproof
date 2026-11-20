package application

import (
	"context"
	"testing"
	"time"

	offerscontracts "elproof/internal/modules/offers/contracts"
	"elproof/internal/modules/projects/domain"
	"elproof/internal/shared/apperror"
)

// "Jadikan proyek" (PLAN §5.4): an Accepted quotation of workspace 1 becomes a
// project whose events, sessions, vendors & venues, lines and terms equal the
// accepted revision.

// ---- fakes ----

// fakeQuotations is offers/contracts.Quotations as `offers` answers it: an
// accepted revision per workspace, 422 quotation_not_accepted for any other
// status, 404 for a missing one or another workspace's.
type fakeQuotations struct {
	byTenant map[int64]map[int64]offerscontracts.AcceptedQuotation
	status   map[int64]string // quotation id → status (default Accepted)
	marks    []int64          // project ids passed to MarkConverted
	deleted  []int64
}

func (f *fakeQuotations) AcceptedQuotation(_ context.Context, tenantID, id int64) (offerscontracts.AcceptedQuotation, error) {
	q, ok := f.byTenant[tenantID][id]
	if !ok {
		return offerscontracts.AcceptedQuotation{}, apperror.NotFound("Penawaran tidak ditemukan")
	}
	if st := f.status[id]; st != "" && st != "Accepted" {
		msg := "Penawaran ini belum diterima. Tandai diterima dulu sebelum dijadikan proyek."
		return offerscontracts.AcceptedQuotation{}, apperror.Validation(msg, map[string][]string{"status": {msg}}).WithCode("quotation_not_accepted")
	}
	return q, nil
}
func (f *fakeQuotations) MarkConverted(_ context.Context, _, _, projectID int64) error {
	f.marks = append(f.marks, projectID)
	return nil
}
func (f *fakeQuotations) MarkProjectDeleted(_ context.Context, _, projectID int64) error {
	f.deleted = append(f.deleted, projectID)
	return nil
}

// convProjects is the follow project store with the unique key
// (tenant_id, quotation_id): a second project for the same quotation is refused.
type convProjects struct {
	followProjects
	creates int
}

func (r *convProjects) Create(_ context.Context, p *domain.Project) error {
	cur := r.db.project
	if p.QuotationID != nil && cur.ID != 0 && cur.QuotationID != nil && *cur.QuotationID == *p.QuotationID && cur.TenantID == p.TenantID {
		return domain.ErrQuotationConverted
	}
	r.creates++
	p.ID = 1
	r.db.project = *p
	return nil
}

// convLookup finds the project of a quotation; stale > 0 answers "none" that
// many times (the other request has not committed yet: the race).
type convLookup struct {
	db    *followDB
	stale int
}

func (l *convLookup) FindByQuotation(_ context.Context, tenantID, quotationID int64) (*domain.Project, error) {
	if l.stale > 0 {
		l.stale--
		return nil, nil
	}
	p := l.db.project
	if p.ID == 0 || p.TenantID != tenantID || p.QuotationID == nil || *p.QuotationID != quotationID {
		return nil, nil
	}
	return &p, nil
}

type convPayments struct {
	ClientPaymentRepository
	db *followDB
}

func (r convPayments) Create(_ context.Context, p *domain.ClientPayment) error {
	p.ID = int64(len(r.db.payments) + 1)
	r.db.payments = append(r.db.payments, *p)
	return nil
}
func (r convPayments) ListByProject(_ context.Context, projectID int64) ([]domain.ClientPayment, error) {
	var out []domain.ClientPayment
	for _, p := range r.db.payments {
		if p.ProjectID == projectID {
			out = append(out, p)
		}
	}
	return out, nil
}

type convContracts struct{ db *followDB }

func (c convContracts) ProjectContractValue(context.Context, int64, int64) (int64, error) {
	return c.db.project.ContractTotal, nil
}
func (convContracts) Supplier(context.Context, int64, int64) (*domain.ProjectSupplier, error) {
	return nil, apperror.NotFound("Vendor tidak ditemukan")
}

type convFixture struct {
	svc      *ConversionService
	projects *ProjectService
	db       *followDB
	repo     *convProjects
	lookup   *convLookup
	quotes   *fakeQuotations
	activity *memActivity
}

const (
	quotationA int64 = 501 // Accepted, workspace 1
	quotationB int64 = 777 // Accepted, workspace 2
	staffRina  int64 = 11  // the quotation's PIC
	vendorKat  int64 = 41
	categoryKt int64 = 5
)

func newConvFixture(t *testing.T) *convFixture {
	t.Helper()
	db := &followDB{events: &memEvents{}, suppliers: &memSuppliers{pays: &memSupplierPayments{}}}
	db.tx = &followTx{db: db}
	mem := &memActivity{}
	act := NewActivityService(activityRepoAdapter{mem})
	repo := &convProjects{followProjects: followProjects{db: db}}
	projects := NewProjectService(repo, followMilestones{db: db}, followTemplates{}, db.suppliers, nil, nil, nil, db.events, nil, act,
		staffByTenant{1: {10: "Ani", staffRina: "Rina"}, 2: {20: "Budi"}})
	projects.SetAtomicSupport(db.tx, followTerms{db: db})
	db.eventSvc = wireEventsWith(projects, db.events, db.suppliers, db.tx, act)
	db.lines = &memLines{db: db}
	lines := NewLineService(db.lines, projects, db.eventSvc, db.events, db.suppliers, followTerms{db: db}, nil, db.tx, act)
	lines.SetOffers(testOffers())
	db.eventSvc.SetLines(lines)
	projects.SetLines(lines)
	payments := NewClientPaymentService(convPayments{db: db}, nil, act, convContracts{db: db})
	payments.SetAtomicSupport(NewProofSupport(db.tx, nil, nil), followTerms{db: db})
	quotes := &fakeQuotations{byTenant: map[int64]map[int64]offerscontracts.AcceptedQuotation{
		1: {quotationA: acceptedQuotation()},
		2: {quotationB: otherWorkspaceQuotation()},
	}, status: map[int64]string{}}
	projects.SetQuotations(quotes)
	lookup := &convLookup{db: db}
	return &convFixture{svc: NewConversionService(projects, lines, payments, lookup), projects: projects, db: db, repo: repo,
		lookup: lookup, quotes: quotes, activity: mem}
}

func strp(s string) *string { return &s }
func ip(n int) *int         { return &n }

// acceptedQuotation is MO-PNW-261119-001 Revisi 2: an Akad at the mosque
// (120 seated, dated), a Resepsi at the hall the client pays (550 standing in
// two sessions, no date yet), catering the WO pays; Paket Gold with a
// component and its extra-guest charge, an item from the caterer, a discount;
// 30/40/30 terms. Contract Rp160.000.000, not PKP.
func acceptedQuotation() offerscontracts.AcceptedQuotation {
	accepted := ymd("2026-11-20")
	rule := offerscontracts.PackageRule{PackageID: pkgGold, Name: "Paket Gold", Price: 150_000_000, EventKindIDs: []int64{kindResepsi},
		IncludedGuests: intp(500), ExtraGuestStep: intp(50), ExtraGuestStepPrice: i64p(4_000_000)}
	return offerscontracts.AcceptedQuotation{
		ID: quotationA, Number: "MO-PNW-261119-001", CoupleNames: "Ayu & Bima", PICStaffID: staffRina, IsPKP: false,
		AcceptedOn: &accepted, AcceptedChannel: "WhatsApp",
		Revision: offerscontracts.AcceptedRevision{ID: 9001, N: 2, ContractTotal: 160_000_000, CreatedAt: ymd("2026-11-19")},
		Content: offerscontracts.QuotationContent{
			CoupleNames: "Ayu & Bima", PICStaffID: staffRina, IsPKP: false,
			Events: []offerscontracts.SnapshotEvent{
				{EventKindID: kindAkad, KindName: "Akad", EventDate: strp("2027-06-12"), StartTime: strp("08:00"), VenueIndex: ip(0), Layout: "Seated",
					Sessions: []offerscontracts.SnapshotSession{{Label: "Sesi 1", Guests: 120}}},
				{EventKindID: kindResepsi, KindName: "Resepsi", VenueIndex: ip(1), Layout: "Standing",
					Sessions: []offerscontracts.SnapshotSession{{Label: "Sesi 1", StartTime: strp("11:00"), Guests: 300}, {Label: "Sesi 2", Guests: 250}}},
			},
			Suppliers: []offerscontracts.SnapshotSupplier{
				{Kind: offerscontracts.SupplierVenue, SupplierID: venueMasjid, Name: "Masjid Al-Ikhlas", PaidBy: offerscontracts.PayerWO, CapacitySeated: intp(150)},
				{Kind: offerscontracts.SupplierVenue, SupplierID: venueGedung, Name: "Gedung Melati", PaidBy: offerscontracts.PayerClient,
					CapacityStanding: intp(600), CapacitySeated: intp(400)},
				{Kind: offerscontracts.SupplierVendor, SupplierID: vendorKat, Name: "Katering Sari Rasa", CategoryID: i64p(categoryKt),
					PaidBy: offerscontracts.PayerWO, ShowName: true, AgreedCost: i64p(60_000_000), Notes: "Menu B"},
			},
			Lines: []offerscontracts.SnapshotLine{
				{Kind: "Package", PackageID: i64p(pkgGold), PackageRule: &rule, Description: "Paket Gold", EventIndex: ip(1), Amount: 150_000_000},
				{Kind: "Component", ParentIndex: ip(0), Description: "Dekorasi pelaminan", Details: "Bunga segar"},
				{Kind: "ExtraGuests", ParentIndex: ip(0), Description: "Tambahan tamu", Amount: 4_000_000},
				{Kind: "Item", Description: "Katering tambahan", Quantity: i64p(50), Unit: strp("porsi"), UnitPrice: i64p(200_000),
					Amount: 10_000_000, SupplierIndex: ip(2), CategoryID: i64p(categoryKt)},
				{Kind: "Discount", Description: "Diskon", Amount: 4_000_000},
			},
			Terms: []offerscontracts.SnapshotTerm{
				{Label: "DP", Percent: ip(30), DueBasis: offerscontracts.DueOnAcceptance, Computed: 48_000_000},
				{Label: "Termin 1", Percent: ip(40), DueBasis: offerscontracts.DueOnEvent, DueOffsetDays: -60, Computed: 64_000_000},
				{Label: "Pelunasan", Percent: ip(30), DueBasis: offerscontracts.DueOnEvent, DueOffsetDays: -7, DueEventIndex: ip(1), Computed: 48_000_000},
			},
			TermsText: "Pembatalan H-30 tidak dapat dikembalikan.",
		},
	}
}

func otherWorkspaceQuotation() offerscontracts.AcceptedQuotation {
	q := acceptedQuotation()
	q.ID, q.Number, q.PICStaffID = quotationB, "MO-LAIN-261119-001", 20
	q.Content.PICStaffID = 20
	return q
}

func convInput() ConversionInput {
	return ConversionInput{
		EventDates:    []time.Time{{}, ymd("2027-06-13")},
		PrepStartDate: ymd("2026-11-25"),
		PICStaffID:    10,
	}
}

var owner = ProjectActor{StaffID: 10, CanAssignPIC: true}

func (f *convFixture) convert(t *testing.T, in ConversionInput) *ConversionResult {
	t.Helper()
	res, err := f.svc.Convert(context.Background(), 1, owner, quotationA, in, ymd("2026-11-20"))
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	return res
}

// ---- tests ----

func TestConvertCopiesTheAcceptedRevision(t *testing.T) {
	f := newConvFixture(t)
	res := f.convert(t, convInput())
	if !res.Created {
		t.Fatal("the first conversion must create the project")
	}
	p := f.db.project
	if p.QuotationID == nil || *p.QuotationID != quotationA || p.QuotationNumber != "MO-PNW-261119-001" || p.QuotationRevisionN != 2 {
		t.Fatalf("quotation link %v %q %d", p.QuotationID, p.QuotationNumber, p.QuotationRevisionN)
	}
	if p.Name != "Ayu & Bima" || p.BrideName != "Ayu" || p.GroomName != "Bima" || p.Status != domain.StatusDraft {
		t.Errorf("project %q %q %q %s", p.Name, p.BrideName, p.GroomName, p.Status)
	}
	if p.PICStaffID != 10 || p.PICSalesStaffID != staffRina {
		t.Errorf("PIC %d, PIC Sales %d (want the quotation's PIC by default)", p.PICStaffID, p.PICSalesStaffID)
	}
	if !p.EventDate.Equal(ymd("2027-06-12")) || !p.PrepStartDate.Equal(ymd("2026-11-25")) {
		t.Errorf("dates %s / %s", p.EventDate, p.PrepStartDate)
	}

	// Vendors & venues: payer, show name, capacities, agreed cost; Planned.
	sups := f.db.suppliers.rows
	if len(sups) != 3 {
		t.Fatalf("%d suppliers, want 3", len(sups))
	}
	for _, s := range sups {
		if s.ProjectID != p.ID || s.EngagementStatus != domain.EngagementPlanned {
			t.Errorf("supplier %s: project %d status %s", s.Name, s.ProjectID, s.EngagementStatus)
		}
	}
	if s := sups[1]; s.Kind != domain.SupplierVenue || s.SupplierID != venueGedung || s.PaidBy != domain.PayerClient ||
		s.CapacityStanding == nil || *s.CapacityStanding != 600 || s.CapacitySeated == nil || *s.CapacitySeated != 400 {
		t.Errorf("hall %+v", s)
	}
	if s := sups[2]; s.Kind != domain.SupplierVendor || s.SupplierID != vendorKat || s.CategoryID != categoryKt || !s.ShowName ||
		s.AgreedCost == nil || *s.AgreedCost != 60_000_000 || s.PaidBy != domain.PayerWO || s.Notes != "Menu B" {
		t.Errorf("caterer %+v", s)
	}

	// Events and sessions, at their venues; the earliest is the main event.
	evs := f.db.events.rows
	if len(evs) != 2 {
		t.Fatalf("%d events, want 2", len(evs))
	}
	akad, resepsi := evs[0], evs[1]
	if akad.EventKindID != kindAkad || !akad.IsMain || !akad.EventDate.Equal(ymd("2027-06-12")) || akad.StartTime == nil || *akad.StartTime != "08:00" ||
		akad.Layout != domain.LayoutSeated || akad.VenueSupplierID == nil || *akad.VenueSupplierID != sups[0].ID {
		t.Errorf("akad %+v", akad)
	}
	if resepsi.EventKindID != kindResepsi || resepsi.IsMain || !resepsi.EventDate.Equal(ymd("2027-06-13")) ||
		resepsi.VenueSupplierID == nil || *resepsi.VenueSupplierID != sups[1].ID || len(resepsi.Sessions) != 2 ||
		resepsi.Sessions[0].Guests != 300 || resepsi.Sessions[1].Guests != 250 || resepsi.Sessions[0].StartTime == nil {
		t.Errorf("resepsi %+v", resepsi)
	}

	// Lines: same kinds and amounts in the same order, parents, events and
	// suppliers remapped, the rule copied.
	lines, _ := f.db.lines.List(context.Background(), 1, p.ID, false)
	frozen := acceptedQuotation().Content.Lines
	if len(lines) != len(frozen) {
		t.Fatalf("%d lines, want %d", len(lines), len(frozen))
	}
	for i, l := range lines {
		if string(l.Kind) != string(frozen[i].Kind) || l.Amount != frozen[i].Amount || l.Description != frozen[i].Description {
			t.Errorf("line %d: %s %q %d, want %s %q %d", i, l.Kind, l.Description, l.Amount, frozen[i].Kind, frozen[i].Description, frozen[i].Amount)
		}
	}
	pkg := lines[0]
	if pkg.PackageRule == nil || pkg.PackageRule.PackageID != pkgGold || pkg.PackageRule.Price != 150_000_000 || !pkg.PackageRule.ChargesExtraGuests() {
		t.Errorf("package rule %+v", pkg.PackageRule)
	}
	if pkg.EventID == nil || *pkg.EventID != resepsi.ID {
		t.Errorf("package event %v, want the Resepsi %d", pkg.EventID, resepsi.ID)
	}
	for _, i := range []int{1, 2} {
		if lines[i].ParentLineID == nil || *lines[i].ParentLineID != pkg.ID {
			t.Errorf("line %d parent %v, want %d", i, lines[i].ParentLineID, pkg.ID)
		}
	}
	if it := lines[3]; it.SupplierRef == nil || *it.SupplierRef != sups[2].ID || it.Unit != "porsi" || it.Quantity == nil || *it.Quantity != 50 {
		t.Errorf("item %+v", it)
	}

	// The contract cache equals the accepted revision.
	if p.ContractTotal != 160_000_000 || p.IsPKP {
		t.Errorf("contract %d pkp %v, want 160.000.000 without PPN", p.ContractTotal, p.IsPKP)
	}

	// Terms: amounts from the contract, due dates from the exact dates.
	want := []domain.PaymentTerm{
		{Label: "DP", Amount: 48_000_000, DueDate: ymd("2026-11-20")},
		{Label: "Termin 1", Amount: 64_000_000, DueDate: ymd("2027-04-13")},
		{Label: "Pelunasan", Amount: 48_000_000, DueDate: ymd("2027-06-06")},
	}
	if len(f.db.terms) != len(want) {
		t.Fatalf("%d terms, want %d", len(f.db.terms), len(want))
	}
	for i, w := range want {
		g := f.db.terms[i]
		if g.Label != w.Label || g.Amount != w.Amount || !g.DueDate.Equal(w.DueDate) {
			t.Errorf("term %d: %s %d %s, want %s %d %s", i, g.Label, g.Amount, g.DueDate.Format("2006-01-02"), w.Label, w.Amount, w.DueDate.Format("2006-01-02"))
		}
	}

	// The default Timeline, and offers told after the commit.
	if len(f.db.milestones) != 3 {
		t.Errorf("%d Timeline steps, want the 3 of the default template", len(f.db.milestones))
	}
	if len(f.quotes.marks) != 1 || f.quotes.marks[0] != p.ID {
		t.Errorf("MarkConverted calls %v", f.quotes.marks)
	}

	// Kesepakatan awal is the accepted revision.
	ag, err := f.svc.Agreement(context.Background(), 1, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if ag.Revision.N != 2 || ag.Number != "MO-PNW-261119-001" || len(ag.Content.Lines) != len(frozen) {
		t.Errorf("agreement %s rev %d", ag.Number, ag.Revision.N)
	}
}

func TestConvertIsIdempotentPerQuotation(t *testing.T) {
	f := newConvFixture(t)
	first := f.convert(t, convInput())
	again := f.convert(t, convInput())
	if again.Created || again.Project.ID != first.Project.ID {
		t.Fatalf("second call: created %v project %d, want the existing %d", again.Created, again.Project.ID, first.Project.ID)
	}
	if f.repo.creates != 1 || len(f.db.events.rows) != 2 {
		t.Errorf("%d projects, %d events: nothing may be created twice", f.repo.creates, len(f.db.events.rows))
	}
	if len(f.quotes.marks) != 2 || f.quotes.marks[1] != first.Project.ID {
		t.Errorf("MarkConverted calls %v: the second call repairs the mark", f.quotes.marks)
	}
}

// Two requests that both checked before either committed: the unique key
// refuses the second project, its writes roll back, and it answers the first.
func TestTwoConcurrentConversionsGiveOneProject(t *testing.T) {
	f := newConvFixture(t)
	first := f.convert(t, convInput())
	events, suppliers, lines := len(f.db.events.rows), len(f.db.suppliers.rows), len(f.db.lines.rows)

	f.lookup.stale = 1 // the second request did not see the first project yet
	second := f.convert(t, convInput())
	if second.Created || second.Project.ID != first.Project.ID {
		t.Fatalf("second: created %v project %d, want the first %d", second.Created, second.Project.ID, first.Project.ID)
	}
	if f.repo.creates != 1 {
		t.Errorf("%d projects created, want 1", f.repo.creates)
	}
	if len(f.db.events.rows) != events || len(f.db.suppliers.rows) != suppliers || len(f.db.lines.rows) != lines {
		t.Errorf("the losing request left rows behind: events %d→%d suppliers %d→%d lines %d→%d",
			events, len(f.db.events.rows), suppliers, len(f.db.suppliers.rows), lines, len(f.db.lines.rows))
	}
	if len(f.quotes.marks) != 2 || f.quotes.marks[0] != first.Project.ID || f.quotes.marks[1] != first.Project.ID {
		t.Errorf("MarkConverted calls %v", f.quotes.marks)
	}
}

func TestConvertRefusesAQuotationThatIsNotAccepted(t *testing.T) {
	for _, status := range []string{"Superseded", "Expired", "Draft", "Sent", "Rejected"} {
		t.Run(status, func(t *testing.T) {
			f := newConvFixture(t)
			f.quotes.status[quotationA] = status
			_, err := f.svc.Convert(context.Background(), 1, owner, quotationA, convInput(), ymd("2026-11-20"))
			ae, ok := apperror.As(err)
			if !ok || ae.Kind != apperror.KindValidation || ae.Code != "quotation_not_accepted" {
				t.Fatalf("err %v, want 422 quotation_not_accepted", err)
			}
			if f.db.project.ID != 0 || len(f.db.events.rows) != 0 || len(f.quotes.marks) != 0 {
				t.Error("nothing may be created for a quotation that is not Accepted")
			}
		})
	}
}

func TestConvertAnotherWorkspacesQuotationIsNotFound(t *testing.T) {
	f := newConvFixture(t)
	// Workspace 1 names workspace 2's quotation, and the other way round.
	if _, err := f.svc.Convert(context.Background(), 1, owner, quotationB, convInput(), ymd("2026-11-20")); !isNotFound(err) {
		t.Errorf("workspace 1 converting workspace 2's quotation: %v, want 404", err)
	}
	if _, err := f.svc.Convert(context.Background(), 2, ProjectActor{StaffID: 20, CanAssignPIC: true}, quotationA, convInput(), ymd("2026-11-20")); !isNotFound(err) {
		t.Errorf("workspace 2 converting workspace 1's quotation: %v, want 404", err)
	}
	// A person limited to their own work who is not the quotation's PIC.
	if _, err := f.svc.Convert(context.Background(), 1, ProjectActor{StaffID: 10, AssignedOnly: true}, quotationA, convInput(), ymd("2026-11-20")); !isNotFound(err) {
		t.Errorf("assigned-only non-PIC: %v, want 404", err)
	}
	if f.db.project.ID != 0 {
		t.Error("nothing may be created")
	}
	// Kesepakatan awal of workspace 1's project, asked by workspace 2.
	f.convert(t, convInput())
	if _, err := f.svc.Agreement(context.Background(), 2, f.db.project.ID); !isNotFound(err) {
		t.Errorf("workspace 2 reading workspace 1's agreement: %v, want 404", err)
	}
}

func TestConvertRecordsTheDownPaymentAsTheFirstClientPayment(t *testing.T) {
	f := newConvFixture(t)
	in := convInput()
	in.DownPayment = &DownPaymentInput{Amount: 10_000_000, PaymentDate: ymd("2026-11-20"), Method: "Transfer", ReferenceNumber: "TRX-1"}
	res := f.convert(t, in)
	if len(f.db.payments) != 1 {
		t.Fatalf("%d payments, want 1", len(f.db.payments))
	}
	pay := f.db.payments[0]
	if pay.ProjectID != res.Project.ID || pay.Type != domain.PaymentDP || pay.Amount != 10_000_000 || !pay.PaymentDate.Equal(ymd("2026-11-20")) ||
		pay.Method != "Transfer" || pay.ReferenceNumber != "TRX-1" || pay.RecordedByStaffID != 10 {
		t.Errorf("payment %+v", pay)
	}
	if res.DownPayment == nil || res.DownPayment.Payment == nil || res.DownPayment.Payment.ID != pay.ID {
		t.Error("the result names the down payment")
	}
	if res.DownPayment.Ledger.ContractValue != 160_000_000 {
		t.Errorf("ledger contract %d", res.DownPayment.Ledger.ContractValue)
	}

	// A down payment above the contract is refused before anything is written.
	g := newConvFixture(t)
	in.DownPayment.Amount = 200_000_000
	_, err := g.svc.Convert(context.Background(), 1, owner, quotationA, in, ymd("2026-11-20"))
	if ae, ok := apperror.As(err); !ok || len(ae.Fields["downPayment.amount"]) == 0 {
		t.Fatalf("err %v, want 422 on downPayment.amount", err)
	}
	if g.db.project.ID != 0 || len(g.db.payments) != 0 {
		t.Error("nothing may be created")
	}
}

func TestConvertRechecksCapacity(t *testing.T) {
	f := newConvFixture(t)
	q := acceptedQuotation()
	// The Resepsi grew to 700 in one session at a hall of 600 standing.
	q.Content.Events[1].Sessions = []offerscontracts.SnapshotSession{{Label: "Sesi 1", Guests: 700}}
	f.quotes.byTenant[1][quotationA] = q
	_, err := f.svc.Convert(context.Background(), 1, owner, quotationA, convInput(), ymd("2026-11-20"))
	ae, ok := apperror.As(err)
	if !ok || ae.Kind != apperror.KindValidation || len(ae.Fields["capacity"]) == 0 {
		t.Fatalf("err %v, want 422 on capacity", err)
	}
	if f.db.project.ID != 0 || len(f.db.events.rows) != 0 || len(f.db.suppliers.rows) != 0 || len(f.quotes.marks) != 0 {
		t.Error("an over-capacity conversion must write nothing")
	}
}

func TestConvertNeedsAnExactDateForEveryEvent(t *testing.T) {
	f := newConvFixture(t)
	in := convInput()
	in.EventDates = nil // the Resepsi has no date in the quotation
	in.PrepStartDate = time.Time{}
	_, err := f.svc.Convert(context.Background(), 1, owner, quotationA, in, ymd("2026-11-20"))
	ae, ok := apperror.As(err)
	if !ok || len(ae.Fields["events.1.eventDate"]) == 0 || len(ae.Fields["prepStartDate"]) == 0 {
		t.Fatalf("err %v, want 422 on events.1.eventDate and prepStartDate", err)
	}
	if len(ae.Fields["events.0.eventDate"]) != 0 {
		t.Error("the Akad keeps the quotation's date")
	}
}
