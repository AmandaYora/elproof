package application

import (
	"context"
	"strings"
	"testing"
	"time"

	offerscontracts "elproof/internal/modules/offers/contracts"
	"elproof/internal/modules/projects/domain"
	"elproof/internal/shared/apperror"
)

// ---- fakes: the Rincian of the follow project, and the offers catalog ----

// memLines is an in-memory LineRepository over followDB: RefreshContract
// rewrites db.project's cache the way the SQL does (subtotal from the live
// lines, PPN 11% floored when PKP and positive).
type memLines struct {
	db        *followDB
	rows      []domain.ProjectLine
	next      int64
	refreshes int
	bumps     int
}

func (m *memLines) snapshot() func() {
	rows, next, refreshes, bumps := append([]domain.ProjectLine(nil), m.rows...), m.next, m.refreshes, m.bumps
	return func() { m.rows, m.next, m.refreshes, m.bumps = rows, next, refreshes, bumps }
}

func (m *memLines) List(_ context.Context, tenantID, projectID int64, withDeleted bool) ([]domain.ProjectLine, error) {
	out := []domain.ProjectLine{}
	for _, l := range m.rows {
		if l.TenantID == tenantID && l.ProjectID == projectID && (withDeleted || l.DeletedAt == nil) {
			out = append(out, l)
		}
	}
	domain.SortLines(out)
	return out, nil
}
func (m *memLines) Get(_ context.Context, tenantID, projectID, id int64) (*domain.ProjectLine, error) {
	for _, l := range m.rows {
		if l.TenantID == tenantID && l.ProjectID == projectID && l.ID == id {
			c := l
			return &c, nil
		}
	}
	return nil, nil
}
func (m *memLines) Insert(_ context.Context, l *domain.ProjectLine) error {
	m.next++
	l.ID = m.next
	l.CreatedAt, l.UpdatedAt = time.Now(), time.Now()
	m.rows = append(m.rows, *l)
	return nil
}
func (m *memLines) Update(_ context.Context, l *domain.ProjectLine) error {
	for i := range m.rows {
		if m.rows[i].ID == l.ID && m.rows[i].DeletedAt == nil {
			c := *l
			c.ParentLineID, c.PackageRule, c.Kind = m.rows[i].ParentLineID, m.rows[i].PackageRule, m.rows[i].Kind
			m.rows[i] = c
		}
	}
	return nil
}
func (m *memLines) SetAmount(_ context.Context, _, _, id, amount int64) error {
	for i := range m.rows {
		if m.rows[i].ID == id {
			m.rows[i].Amount = amount
		}
	}
	return nil
}
func (m *memLines) SetSortOrder(_ context.Context, _, _, id int64, order int) error {
	for i := range m.rows {
		if m.rows[i].ID == id {
			m.rows[i].SortOrder = order
		}
	}
	return nil
}
func (m *memLines) SoftDelete(_ context.Context, _, _, id int64, at time.Time) (bool, error) {
	found := false
	for i := range m.rows {
		l := &m.rows[i]
		if l.DeletedAt != nil {
			continue
		}
		if l.ID == id || (l.ParentLineID != nil && *l.ParentLineID == id) {
			stamp := at
			l.DeletedAt = &stamp
			found = found || l.ID == id
		}
	}
	return found, nil
}
func (m *memLines) Restore(_ context.Context, _, _, id int64, notBefore time.Time) (bool, error) {
	var at *time.Time
	for _, l := range m.rows {
		if l.ID == id && l.DeletedAt != nil && l.DeletedAt.After(notBefore) {
			at = l.DeletedAt
		}
	}
	if at == nil {
		return false, nil
	}
	stamp := *at
	for i := range m.rows {
		l := &m.rows[i]
		if l.DeletedAt != nil && l.DeletedAt.Equal(stamp) && (l.ID == id || (l.ParentLineID != nil && *l.ParentLineID == id)) {
			l.DeletedAt = nil
		}
	}
	return true, nil
}
func (m *memLines) RefreshContract(_ context.Context, _, _ int64, bumpVersion bool) error {
	m.refreshes++
	var sub int64
	for _, l := range m.rows {
		if l.DeletedAt != nil {
			continue
		}
		switch l.Kind {
		case domain.LinePackage, domain.LineExtraGuests, domain.LineItem:
			sub += l.Amount
		case domain.LineDiscount:
			sub -= l.Amount
		}
	}
	p := &m.db.project
	p.ContractSubtotal, p.ContractTax = sub, 0
	if p.IsPKP && sub > 0 {
		p.ContractTax = sub * 11 / 100
	}
	p.ContractTotal = p.ContractSubtotal + p.ContractTax
	if bumpVersion {
		m.bumps++
		p.LinesVersion = p.LinesVersion.Add(time.Millisecond)
	}
	return nil
}
func (m *memLines) Contract(context.Context, int64, int64) (domain.ContractCache, error) {
	p := m.db.project
	return domain.ContractCache{IsPKP: p.IsPKP, Subtotal: p.ContractSubtotal, Tax: p.ContractTax, Total: p.ContractTotal, LinesVersion: p.LinesVersion}, nil
}
func (m *memLines) PackageNames(_ context.Context, tenantID int64, ids []int64) (map[int64][]string, error) {
	out := map[int64][]string{}
	for _, id := range ids {
		list, _ := m.List(context.Background(), tenantID, id, false)
		for _, l := range list {
			if l.Kind == domain.LinePackage {
				out[id] = append(out[id], l.Description)
			}
		}
	}
	return out, nil
}

// fakeOffers is the offers catalog per workspace, priced with the real pure
// functions of offers (through its contract).
type fakeOffers struct {
	packages map[int64]map[int64]*offerscontracts.PackageCopy
}

func (f *fakeOffers) PackageCopy(_ context.Context, tenantID, id int64) (offerscontracts.PackageCopy, error) {
	if p, ok := f.packages[tenantID][id]; ok {
		cp := *p
		cp.Rule.EventKindIDs = append([]int64(nil), p.Rule.EventKindIDs...)
		cp.Components = append([]offerscontracts.PackageComponentCopy(nil), p.Components...)
		return cp, nil
	}
	return offerscontracts.PackageCopy{}, apperror.NotFound("Paket tidak ditemukan")
}
func (*fakeOffers) ExtraGuests(rule offerscontracts.PackageRule, events []offerscontracts.EventGuests) offerscontracts.ExtraGuestsResult {
	return offerscontracts.ExtraGuestsOf(rule, events)
}
func (*fakeOffers) LineTotals(lines []offerscontracts.Line, isPKP bool) (offerscontracts.Totals, error) {
	return offerscontracts.LineTotalsOf(lines, isPKP)
}

// Packages of workspace 1 (and one of workspace 2).
const (
	pkgGold     int64 = 7  // Resepsi, Rp150.000.000, 500 included, +Rp4.000.000 per 50
	pkgAkad     int64 = 8  // Akad, Rp20.000.000, no extra-guest charge
	pkgInactive int64 = 9  // nonaktif
	pkgSiraman  int64 = 10 // covers an event kind the project has not
	pkgOther    int64 = 31 // workspace 2
	kindSiraman int64 = 3
)

func testOffers() *fakeOffers {
	gold := offerscontracts.PackageCopy{IsActive: true, Rule: offerscontracts.PackageRule{
		PackageID: pkgGold, Name: "Paket Gold", Price: 150_000_000, EventKindIDs: []int64{kindResepsi},
		IncludedGuests: intp(500), ExtraGuestStep: intp(50), ExtraGuestStepPrice: i64p(4_000_000),
	}, Components: []offerscontracts.PackageComponentCopy{{Description: "Dekorasi pelaminan"}, {Description: "Rias pengantin"}}}
	akad := offerscontracts.PackageCopy{IsActive: true, Rule: offerscontracts.PackageRule{
		PackageID: pkgAkad, Name: "Paket Akad", Price: 20_000_000, EventKindIDs: []int64{kindAkad},
	}}
	inactive := offerscontracts.PackageCopy{IsActive: false, Rule: offerscontracts.PackageRule{PackageID: pkgInactive, Name: "Lama", Price: 1}}
	siraman := offerscontracts.PackageCopy{IsActive: true, Rule: offerscontracts.PackageRule{
		PackageID: pkgSiraman, Name: "Paket Siraman", Price: 5_000_000, EventKindIDs: []int64{kindSiraman},
	}}
	other := offerscontracts.PackageCopy{IsActive: true, Rule: offerscontracts.PackageRule{PackageID: pkgOther, Name: "Paket B", Price: 9}}
	return &fakeOffers{packages: map[int64]map[int64]*offerscontracts.PackageCopy{
		1: {pkgGold: &gold, pkgAkad: &akad, pkgInactive: &inactive, pkgSiraman: &siraman},
		2: {pkgOther: &other},
	}}
}

type lineFixture struct {
	svc      *LineService
	projects *ProjectService
	db       *followDB
	offers   *fakeOffers
	activity *memActivity
	project  *domain.Project
}

// newLineFixture is the follow project (workspace 1) with an Akad at the
// mosque (120 guests) and a Resepsi at the hall (550), and an empty Rincian.
func newLineFixture(t *testing.T, pkp bool) *lineFixture {
	t.Helper()
	projects, db, mem := newFollow()
	projects.SetWorkspaceProfile(fixedProfile{pkp: pkp})
	db.lines = &memLines{db: db}
	p, err := projects.CreateAtomic(context.Background(), 1, ProjectActor{StaffID: 10, CanAssignPIC: true}, ProjectInput{
		Name: "Ayu & Bima", PrepStartDate: ymd("2026-11-15"),
	}, ProjectSetup{Events: []EventInput{akadAtMasjid(ymd("2027-06-12")), resepsiAtGedung(ymd("2027-06-13"), 550)}})
	if err != nil {
		t.Fatal(err)
	}
	offers := testOffers()
	svc := NewLineService(db.lines, projects, db.eventSvc, db.events, db.suppliers, followTerms{db: db}, nil, db.tx,
		NewActivityService(activityRepoAdapter{mem}))
	svc.SetOffers(offers)
	db.eventSvc.SetLines(svc)
	projects.SetLines(svc)
	return &lineFixture{svc: svc, projects: projects, db: db, offers: offers, activity: mem, project: p}
}

func (f *lineFixture) version() *time.Time {
	v := f.db.project.LinesVersion
	return &v
}

func (f *lineFixture) replace(t *testing.T, drafts ...LineDraft) *LinesView {
	t.Helper()
	v, err := f.svc.Replace(context.Background(), 1, f.project.ID, 10, drafts, f.version())
	if err != nil {
		t.Fatalf("replace: %v", err)
	}
	return v
}

func (f *lineFixture) resepsi(t *testing.T) domain.ProjectEvent {
	t.Helper()
	for _, e := range f.db.events.rows {
		if e.EventKindID == kindResepsi {
			return e
		}
	}
	t.Fatal("no Resepsi")
	return domain.ProjectEvent{}
}

// setResepsiGuests edits the Resepsi (one session) through the event service.
func (f *lineFixture) setResepsiGuests(t *testing.T, guests int) error {
	t.Helper()
	e := f.resepsi(t)
	_, err := f.db.eventSvc.Update(context.Background(), 1, f.project.ID, e.ID, 10, resepsiAtGedung(e.EventDate, guests), nil)
	return err
}

// current is the stored Rincian as an editor would send it back unchanged.
func (f *lineFixture) current(t *testing.T) []LineDraft {
	t.Helper()
	list, err := f.db.lines.List(context.Background(), 1, f.project.ID, false)
	if err != nil {
		t.Fatal(err)
	}
	return draftsOf(list)
}

func (f *lineFixture) live(kind domain.LineKind) []domain.ProjectLine {
	var out []domain.ProjectLine
	for _, l := range f.db.lines.rows {
		if l.Kind == kind && l.DeletedAt == nil {
			out = append(out, l)
		}
	}
	return out
}

func gold() LineDraft {
	return LineDraft{Key: "gold", Kind: domain.LinePackage, PackageID: pkgGold}
}

func item(desc string, qty, price int64) LineDraft {
	return LineDraft{Kind: domain.LineItem, Description: desc, Quantity: i64p(qty), UnitPrice: i64p(price), Unit: "unit"}
}

func discount(amount int64) LineDraft {
	return LineDraft{Kind: domain.LineDiscount, Description: "Potongan", Amount: amount}
}

func fieldKeys(t *testing.T, err error) map[string][]string {
	t.Helper()
	ae, ok := apperror.As(err)
	if !ok || ae.Kind != apperror.KindValidation {
		t.Fatalf("want a 422, got %v", err)
	}
	return ae.Fields
}

// ---- tests ----

// A package brings its components (from the catalog) and an extra-guest
// charge; contract = subtotal + PPN, the cache rewritten with the save.
func TestLinesContractIsSubtotalPlusPPN(t *testing.T) {
	f := newLineFixture(t, true)
	v := f.replace(t, gold(), item("Photobooth", 2, 3_000_000), discount(5_000_000))

	if got := len(f.live(domain.LineComponent)); got != 2 {
		t.Errorf("the catalog's two components are copied: %d", got)
	}
	extra := f.live(domain.LineExtraGuests)
	if len(extra) != 1 || extra[0].Amount != 4_000_000 {
		t.Fatalf("550 guests over 500 included = 1 step of Rp4.000.000: %+v", extra)
	}
	// 150.000.000 + 4.000.000 + 6.000.000 - 5.000.000 = 155.000.000; PPN 11%.
	want := offerscontracts.Totals{Gross: 160_000_000, Discount: 5_000_000, Subtotal: 155_000_000, Tax: 17_050_000, Total: 172_050_000}
	if v.Totals != want {
		t.Errorf("totals %+v, want %+v", v.Totals, want)
	}
	p := f.db.project
	if p.ContractSubtotal != want.Subtotal || p.ContractTax != want.Tax || p.ContractTotal != want.Total {
		t.Errorf("cache %d + %d = %d, want the totals", p.ContractSubtotal, p.ContractTax, p.ContractTotal)
	}
	if f.db.lines.bumps != 1 || !p.LinesVersion.Equal(v.LinesVersion) {
		t.Errorf("a line write moves lines_version once: %d bumps", f.db.lines.bumps)
	}

	// Not PKP: no PPN.
	g := newLineFixture(t, false)
	v = g.replace(t, gold(), item("Photobooth", 2, 3_000_000), discount(5_000_000))
	if v.Totals.Tax != 0 || v.Totals.Total != 155_000_000 || g.db.project.ContractTotal != 155_000_000 {
		t.Errorf("not PKP: %+v / %d", v.Totals, g.db.project.ContractTotal)
	}
}

// The contract is never below 0: a discount above the subtotal is refused and
// nothing is written.
func TestLinesTotalIsNeverNegative(t *testing.T) {
	f := newLineFixture(t, true)
	_, err := f.svc.Replace(context.Background(), 1, f.project.ID, 10, []LineDraft{item("Photobooth", 1, 6_000_000), discount(200_000_000)}, f.version())
	ae, ok := apperror.As(err)
	if !ok || ae.Code != "discount_exceeds_subtotal" || len(ae.Fields["lines.1.amount"]) == 0 {
		t.Fatalf("want 422 discount_exceeds_subtotal on lines.1.amount, got %v", err)
	}
	if len(f.db.lines.rows) != 0 || f.db.project.ContractTotal != 0 || f.db.lines.refreshes != 0 {
		t.Errorf("nothing written: %d lines, contract %d", len(f.db.lines.rows), f.db.project.ContractTotal)
	}
	// A bonus is an item at 0 whatever price is sent.
	v := f.replace(t, LineDraft{Kind: domain.LineItem, Description: "Bonus foto", Quantity: i64p(1), UnitPrice: i64p(9_000_000), IsBonus: true})
	if v.Totals.Total != 0 || f.live(domain.LineItem)[0].Amount != 0 {
		t.Errorf("a bonus adds nothing: %+v", v.Totals)
	}
}

// Raising the Resepsi's guests recomputes the extra-guest charge and the
// contract in the event write's own transaction; a guest change that would
// leave the discount above the subtotal is refused with nothing changed.
func TestGuestChangeUpdatesExtraGuestsInTheSameTransaction(t *testing.T) {
	f := newLineFixture(t, false)
	f.replace(t, gold())
	before := f.db.project.LinesVersion
	began := f.db.tx.began

	if err := f.setResepsiGuests(t, 600); err != nil {
		t.Fatal(err)
	}
	if got := f.live(domain.LineExtraGuests)[0].Amount; got != 8_000_000 {
		t.Errorf("600 guests = 2 steps = Rp8.000.000, got %d", got)
	}
	if f.db.project.ContractTotal != 158_000_000 {
		t.Errorf("contract follows: %d", f.db.project.ContractTotal)
	}
	if f.db.tx.began != began+1 {
		t.Errorf("one transaction for the event and the Rincian: %d", f.db.tx.began-began)
	}
	if !f.db.project.LinesVersion.Equal(before) {
		t.Error("a guest change does not move lines_version (the lines were not edited)")
	}
	logged := false
	for _, e := range f.activity.entries {
		if e.Type == domain.ActivityLineUpdated && strings.Contains(e.Description, "Rp4.000.000") && strings.Contains(e.Description, "Rp8.000.000") {
			logged = true
		}
	}
	if !logged {
		t.Errorf("the change is logged with the old and new value: %+v", f.activity.entries)
	}

	// Discount 156M needs at least Rp6.000.000 of extra guests: 500 guests
	// (no charge) would make the total negative.
	f.replace(t, append(f.current(t), discount(156_000_000))...)
	total := f.db.project.ContractTotal
	if total != 2_000_000 {
		t.Fatalf("150M + 8M - 156M = 2M, got %d", total)
	}
	err := f.setResepsiGuests(t, 500)
	if ae, ok := apperror.As(err); !ok || ae.Code != "discount_exceeds_subtotal" {
		t.Fatalf("want the guest change refused, got %v", err)
	}
	if g := domain.EventGuestTotal(f.resepsi(t)); g != 600 {
		t.Errorf("the event keeps its guests: %d", g)
	}
	if f.live(domain.LineExtraGuests)[0].Amount != 8_000_000 || f.db.project.ContractTotal != total {
		t.Errorf("the Rincian and the contract are untouched: %d / %d", f.live(domain.LineExtraGuests)[0].Amount, f.db.project.ContractTotal)
	}
}

// The project keeps its copy of the package: changing the catalog afterwards
// changes neither the price nor the extra-guest rule of the project.
func TestCatalogChangeDoesNotChangeTheProject(t *testing.T) {
	f := newLineFixture(t, false)
	f.replace(t, gold())
	cat := f.offers.packages[1][pkgGold]
	cat.Rule.Price = 999_000_000
	cat.Rule.IncludedGuests = intp(100)
	cat.Rule.ExtraGuestStepPrice = i64p(50_000_000)

	if err := f.setResepsiGuests(t, 600); err != nil {
		t.Fatal(err)
	}
	v, err := f.svc.View(context.Background(), 1, f.project.ID)
	if err != nil {
		t.Fatal(err)
	}
	if pkg := f.live(domain.LinePackage)[0]; pkg.Amount != 150_000_000 || pkg.PackageRule.Price != 150_000_000 {
		t.Errorf("the package keeps its copied price: %+v", pkg)
	}
	if v.Totals.Total != 158_000_000 {
		t.Errorf("the extra-guest charge uses the copied rule (500 included, Rp4.000.000): %+v", v.Totals)
	}
}

// The preview computes with the save's own functions and writes nothing; its
// total is what the save then stores.
func TestPreviewEqualsSaveAndWritesNothing(t *testing.T) {
	f := newLineFixture(t, true)
	drafts := []LineDraft{gold(), item("Photobooth", 2, 3_000_000), discount(5_000_000)}
	version, logged := f.db.project.LinesVersion, len(f.activity.entries)
	pv, err := f.svc.Preview(context.Background(), 1, f.project.ID, drafts)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.db.lines.rows) != 0 || f.db.lines.refreshes != 0 || f.db.project.ContractTotal != 0 || !f.db.project.LinesVersion.Equal(version) {
		t.Fatalf("a preview writes nothing: %d lines, %d refreshes", len(f.db.lines.rows), f.db.lines.refreshes)
	}
	if len(f.activity.entries) != logged {
		t.Errorf("a preview logs nothing: %+v", f.activity.entries[logged:])
	}
	if len(pv.Lines) != 6 {
		t.Errorf("the preview shows the generated components and charge: %d lines", len(pv.Lines))
	}
	for _, l := range pv.Lines {
		if l.Kind == domain.LineComponent && (l.ParentIndex() < 0 || pv.Lines[l.ParentIndex()].Key != "gold") {
			t.Errorf("a new component points at its package in the view: %+v", l)
		}
	}
	saved := f.replace(t, drafts...)
	if saved.Totals != pv.Totals || f.db.project.ContractTotal != pv.Totals.Total {
		t.Errorf("preview %+v, save %+v, stored %d", pv.Totals, saved.Totals, f.db.project.ContractTotal)
	}
}

// "Margin belum lengkap (n vendor)": the margin counts the WO-paid suppliers
// without an agreed cost; the subtotal (not the PPN) is its base.
func TestMarginIncompleteCount(t *testing.T) {
	f := newLineFixture(t, true)
	for i := range f.db.suppliers.rows {
		f.db.suppliers.rows[i].AgreedCost = i64p(10_000_000) // the two venues
	}
	f.db.suppliers.rows = append(f.db.suppliers.rows,
		domain.ProjectSupplier{ID: 90, ProjectID: f.project.ID, Kind: domain.SupplierVendor, Name: "Katering", PaidBy: domain.PayerWO, EngagementStatus: domain.EngagementBooked},
		domain.ProjectSupplier{ID: 91, ProjectID: f.project.ID, Kind: domain.SupplierVendor, Name: "MUA", PaidBy: domain.PayerClient, EngagementStatus: domain.EngagementBooked},
	)
	v := f.replace(t, gold())
	m := v.Margin
	if m.Subtotal != 154_000_000 || m.SupplierCost != 20_000_000 || m.MissingCost != 1 || m.Margin != 134_000_000 {
		t.Errorf("margin %+v, want subtotal 154M, cost 20M, 1 missing (the client-paid one is not the WO's), margin 134M", m)
	}
}

// A schedule that no longer adds up to the contract is flagged, never
// rewritten by itself; "Hitung ulang termin" fits it.
func TestTermsMismatchAndRecalculate(t *testing.T) {
	f := newLineFixture(t, false)
	f.db.terms = []domain.PaymentTerm{
		{ID: 1, ProjectID: f.project.ID, Label: "DP", Amount: 50_000_000, DueDate: ymd("2026-12-01")},
		{ID: 2, ProjectID: f.project.ID, Label: "Pelunasan", Amount: 50_000_000, DueDate: ymd("2027-06-01")},
	}
	v := f.replace(t, gold())
	if !v.HasTerms || v.TermsTotal != 100_000_000 || !v.TermsMismatch {
		t.Fatalf("schedule 100M vs contract 154M: %+v", v)
	}
	if f.db.terms[0].Amount != 50_000_000 {
		t.Error("a line save never rewrites the schedule")
	}
	terms := NewPaymentTermService(followTerms{db: f.db}, followPayments{db: f.db}, f.projects, nil)
	if _, err := terms.Recalculate(context.Background(), 1, f.project.ID, 10, time.Date(2026, 11, 20, 9, 0, 0, 0, domain.WIB)); err != nil {
		t.Fatal(err)
	}
	v, err := f.svc.View(context.Background(), 1, f.project.ID)
	if err != nil {
		t.Fatal(err)
	}
	if v.TermsMismatch || v.TermsTotal != 154_000_000 {
		t.Errorf("after Hitung ulang termin: %d, mismatch=%v", v.TermsTotal, v.TermsMismatch)
	}
}

// A schedule needs a contract: terms before the Rincian are refused.
func TestTermsNeedTheRincianFirst(t *testing.T) {
	f := newLineFixture(t, false)
	terms := NewPaymentTermService(followTerms{db: f.db}, followPayments{db: f.db}, f.projects, nil)
	_, err := terms.Replace(context.Background(), 1, f.project.ID, 10, []PaymentTermInput{{Label: "DP", Amount: 1, DueDate: ymd("2026-12-01")}}, ymd("2026-11-20"))
	if ae, ok := apperror.As(err); !ok || ae.Code != "no_contract" {
		t.Fatalf("want 422 no_contract, got %v", err)
	}
}

// What a line may point at: a package of this workspace that is active and
// whose event kinds the project has; an event of this project; a vendor or
// venue of this project the WO pays.
func TestLineReferencesAreChecked(t *testing.T) {
	f := newLineFixture(t, false)
	f.db.suppliers.rows = append(f.db.suppliers.rows,
		domain.ProjectSupplier{ID: 91, ProjectID: f.project.ID, Kind: domain.SupplierVendor, Name: "MUA", PaidBy: domain.PayerClient, EngagementStatus: domain.EngagementBooked})
	withSupplier := item("Rias", 1, 1_000_000)
	withSupplier.SupplierRef = i64p(91)
	withEvent := item("Tenda", 1, 1_000_000)
	withEvent.EventID = i64p(999)
	_, err := f.svc.Replace(context.Background(), 1, f.project.ID, 10, []LineDraft{
		{Kind: domain.LinePackage, PackageID: pkgOther},    // another workspace's
		{Kind: domain.LinePackage, PackageID: pkgInactive}, // nonaktif
		{Kind: domain.LinePackage, PackageID: pkgSiraman},  // Siraman is not among the events
		withSupplier, withEvent,
	}, f.version())
	fields := fieldKeys(t, err)
	for key, want := range map[string]string{
		"lines.0.packageId":   "Paket tidak ditemukan",
		"lines.1.packageId":   "nonaktif",
		"lines.2.packageId":   "belum ada di proyek",
		"lines.3.supplierRef": "dibayar WO",
		"lines.4.eventId":     "tidak ditemukan",
	} {
		if len(fields[key]) == 0 || !strings.Contains(fields[key][0], want) {
			t.Errorf("%s: %v, want %q", key, fields[key], want)
		}
	}
	if len(f.db.lines.rows) != 0 {
		t.Error("nothing written")
	}
}

// Deleting a package takes its components and extra-guest charge, never a
// vendor, a venue or a payment; restore brings exactly them back.
func TestDeletePackageTakesItsChildrenOnly(t *testing.T) {
	f := newLineFixture(t, false)
	f.replace(t, gold(), item("Photobooth", 1, 6_000_000))
	suppliers := len(f.db.suppliers.live(f.project.ID))
	pkg := f.live(domain.LinePackage)[0]
	v, until, err := f.svc.Delete(context.Background(), 1, f.project.ID, pkg.ID, 10, f.version(), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if until.IsZero() || len(f.live(domain.LineComponent)) != 0 || len(f.live(domain.LineExtraGuests)) != 0 || len(v.Lines) != 1 {
		t.Errorf("the package goes with its children: %d lines left", len(v.Lines))
	}
	if len(f.db.suppliers.live(f.project.ID)) != suppliers {
		t.Error("no vendor or venue is touched")
	}
	if f.db.project.ContractTotal != 6_000_000 {
		t.Errorf("the contract follows: %d", f.db.project.ContractTotal)
	}
	if _, err := f.svc.Restore(context.Background(), 1, f.project.ID, pkg.ID, 10, time.Now()); err != nil {
		t.Fatal(err)
	}
	if len(f.live(domain.LineComponent)) != 2 || len(f.live(domain.LineExtraGuests)) != 1 || f.db.project.ContractTotal != 160_000_000 {
		t.Errorf("restore brings them back: %d components, contract %d", len(f.live(domain.LineComponent)), f.db.project.ContractTotal)
	}
	// The extra-guest charge is never typed nor deleted on its own.
	extra := f.live(domain.LineExtraGuests)[0]
	if _, _, err := f.svc.Delete(context.Background(), 1, f.project.ID, extra.ID, 10, f.version(), time.Now()); err == nil {
		t.Error("deleting the extra-guest charge alone is refused")
	}
	if _, err := f.svc.Patch(context.Background(), 1, f.project.ID, extra.ID, 10, LinePatch{Amount: Some(int64(1))}, f.version()); err == nil {
		t.Error("typing the extra-guest charge is refused")
	}
}

// lines_version is the Rincian's own If-Match: a stale one is a 409 carrying
// the Rincian as it is now.
func TestStaleLinesVersionIsAConflict(t *testing.T) {
	f := newLineFixture(t, false)
	old := f.db.project.LinesVersion
	f.replace(t, gold())
	_, err := f.svc.Add(context.Background(), 1, f.project.ID, 10, item("Photobooth", 1, 1), &old)
	current, ok := apperror.IsStale(err)
	if !ok {
		t.Fatalf("want a stale conflict, got %v", err)
	}
	if v, _ := current.(*LinesView); v == nil || len(v.Lines) != 4 {
		t.Errorf("the conflict carries the current Rincian: %+v", current)
	}
	if len(f.live(domain.LineItem)) != 0 {
		t.Error("nothing written")
	}
}

// Another workspace never reaches the project's Rincian (its id answers 404).
func TestLinesOfAnotherWorkspaceAreNotFound(t *testing.T) {
	f := newLineFixture(t, false)
	f.replace(t, gold())
	if _, err := f.svc.View(context.Background(), 2, f.project.ID); !isNotFound(err) {
		t.Errorf("view: %v", err)
	}
	if _, err := f.svc.Preview(context.Background(), 2, f.project.ID, []LineDraft{gold()}); !isNotFound(err) {
		t.Errorf("preview: %v", err)
	}
	if _, err := f.svc.Replace(context.Background(), 2, f.project.ID, 20, nil, nil); !isNotFound(err) {
		t.Errorf("replace: %v", err)
	}
	if len(f.live(domain.LinePackage)) != 1 {
		t.Error("the Rincian is untouched")
	}
}
