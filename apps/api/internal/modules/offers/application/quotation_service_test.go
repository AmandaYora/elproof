package application

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"testing"
	"time"

	"elproof/internal/modules/offers/domain"
	"elproof/internal/shared/apperror"
	"elproof/internal/shared/pagination"
)

// --- quotation fakes -----------------------------------------------------------

type fakeQuotations struct {
	mu        sync.Mutex
	nextID    int64
	counters  map[string]int
	rows      map[int64]*domain.Quotation
	deleted   map[int64]bool
	content   map[int64]*domain.WorkingCopy
	revisions []domain.QuotationRevision
	revTenant map[int64]int64
	activity  []domain.Activity
	actTenant map[int64]int64
	// beforeInsertRevision runs before a revision is stored (another request
	// storing the same content first).
	beforeInsertRevision func()
}

func newFakeQuotations() *fakeQuotations {
	return &fakeQuotations{counters: map[string]int{}, rows: map[int64]*domain.Quotation{}, deleted: map[int64]bool{},
		content: map[int64]*domain.WorkingCopy{}, revTenant: map[int64]int64{}, actTenant: map[int64]int64{}}
}

func (f *fakeQuotations) id() int64 { f.nextID++; return f.nextID }

func (f *fakeQuotations) row(tenantID, id int64) *domain.Quotation {
	q, ok := f.rows[id]
	if !ok || q.TenantID != tenantID || f.deleted[id] {
		return nil
	}
	return q
}

func (f *fakeQuotations) NextSequence(_ context.Context, tenantID int64, docDate time.Time) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	key := fmt.Sprintf("%d|%s", tenantID, docDate.Format("2006-01-02"))
	f.counters[key]++
	return f.counters[key], nil
}

func (f *fakeQuotations) Create(_ context.Context, q *domain.Quotation) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	q.ID = f.id()
	now := time.Now().UTC()
	q.CreatedAt, q.UpdatedAt, q.LinesUpdatedAt, q.TermsUpdatedAt, q.ChangedAt = now, now, now, now, now
	c := *q
	f.rows[q.ID] = &c
	f.content[q.ID] = &domain.WorkingCopy{}
	return nil
}

func (f *fakeQuotations) FindByID(_ context.Context, tenantID, id int64) (*domain.Quotation, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if q := f.row(tenantID, id); q != nil {
		c := *q
		return &c, nil
	}
	return nil, nil
}

func (f *fakeQuotations) Lock(ctx context.Context, tenantID, id int64) (*domain.Quotation, error) {
	return f.FindByID(ctx, tenantID, id)
}

func (f *fakeQuotations) save(q *domain.Quotation) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.row(q.TenantID, q.ID) == nil {
		return apperror.NotFound(MsgQuotationNotFound)
	}
	c := *q
	c.UpdatedAt = time.Now().UTC()
	f.rows[q.ID] = &c
	return nil
}

func (f *fakeQuotations) SaveHeader(_ context.Context, q *domain.Quotation) error { return f.save(q) }
func (f *fakeQuotations) SaveState(_ context.Context, q *domain.Quotation) error  { return f.save(q) }

func (f *fakeQuotations) SaveTermsText(_ context.Context, tenantID, id int64, text string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if q := f.row(tenantID, id); q != nil {
		q.TermsText, q.TermsUpdatedAt = text, time.Now().UTC()
	}
	return nil
}

func (f *fakeQuotations) SetCaches(_ context.Context, tenantID, id int64, total int64, main *time.Time, bumpLines, bumpTerms bool) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if q := f.row(tenantID, id); q != nil {
		q.ContractTotal, q.MainEventDate = total, main
	}
	return nil
}

func (f *fakeQuotations) SoftDelete(_ context.Context, tenantID, id int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.row(tenantID, id) != nil {
		f.deleted[id] = true
	}
	return nil
}

func (f *fakeQuotations) ListPaginated(_ context.Context, tenantID int64, _ pagination.Params, _ QuotationListFilter, _ pagination.Sort) ([]domain.Quotation, int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []domain.Quotation
	for id := range f.rows {
		if q := f.row(tenantID, id); q != nil && q.AlternativeOfID == nil {
			out = append(out, *q)
		}
	}
	return out, int64(len(out)), nil
}

func (f *fakeQuotations) ListSortSpec() (map[string]pagination.SortColumn, pagination.Sort) {
	return map[string]pagination.SortColumn{}, pagination.Sort{}
}

func (f *fakeQuotations) StatusCounts(context.Context, int64, QuotationListFilter) (map[domain.QuotationStatus]int, error) {
	return map[domain.QuotationStatus]int{}, nil
}

func (f *fakeQuotations) Alternatives(_ context.Context, tenantID int64, rootIDs []int64) ([]domain.Quotation, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	roots := map[int64]bool{}
	for _, id := range rootIDs {
		roots[id] = true
	}
	var out []domain.Quotation
	for id := range f.rows {
		if q := f.row(tenantID, id); q != nil && q.AlternativeOfID != nil && roots[*q.AlternativeOfID] {
			out = append(out, *q)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (f *fakeQuotations) FindByProject(_ context.Context, tenantID, projectID int64) (*domain.Quotation, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for id := range f.rows {
		if q := f.row(tenantID, id); q != nil && q.ConvertedProjectID != nil && *q.ConvertedProjectID == projectID {
			c := *q
			return &c, nil
		}
	}
	return nil, nil
}

func (f *fakeQuotations) LoadContent(_ context.Context, tenantID, quotationID int64) (*domain.WorkingCopy, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	w, ok := f.content[quotationID]
	if !ok || f.row(tenantID, quotationID) == nil {
		return &domain.WorkingCopy{}, nil
	}
	out := &domain.WorkingCopy{
		Events: append([]domain.QuotationEvent(nil), w.Events...), Suppliers: append([]domain.QuotationSupplier(nil), w.Suppliers...),
		Lines: append([]domain.QuotationLine(nil), w.Lines...), Terms: append([]domain.QuotationTerm(nil), w.Terms...),
	}
	return out, nil
}

func (f *fakeQuotations) InsertEvent(_ context.Context, _ int64, e *domain.QuotationEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	e.ID, e.UpdatedAt = f.id(), time.Now().UTC()
	f.content[e.QuotationID].Events = append(f.content[e.QuotationID].Events, *e)
	return nil
}

func (f *fakeQuotations) UpdateEvent(_ context.Context, _ int64, e *domain.QuotationEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	w := f.content[e.QuotationID]
	for i := range w.Events {
		if w.Events[i].ID == e.ID {
			w.Events[i] = *e
		}
	}
	return nil
}

func (f *fakeQuotations) DeleteEvent(_ context.Context, _ int64, quotationID, eventID int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	w := f.content[quotationID]
	out := w.Events[:0]
	for _, e := range w.Events {
		if e.ID != eventID {
			out = append(out, e)
		}
	}
	w.Events = out
	return nil
}

func (f *fakeQuotations) SetEventOrder(_ context.Context, _ int64, quotationID int64, ids []int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	w := f.content[quotationID]
	for pos, id := range ids {
		for i := range w.Events {
			if w.Events[i].ID == id {
				w.Events[i].SortOrder = pos
			}
		}
	}
	return nil
}

func (f *fakeQuotations) InsertSupplier(_ context.Context, _ int64, s *domain.QuotationSupplier) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	s.ID, s.UpdatedAt = f.id(), time.Now().UTC()
	f.content[s.QuotationID].Suppliers = append(f.content[s.QuotationID].Suppliers, *s)
	return nil
}

func (f *fakeQuotations) UpdateSupplier(_ context.Context, _ int64, s *domain.QuotationSupplier) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	w := f.content[s.QuotationID]
	for i := range w.Suppliers {
		if w.Suppliers[i].ID == s.ID {
			w.Suppliers[i] = *s
		}
	}
	return nil
}

func (f *fakeQuotations) DeleteSupplier(_ context.Context, _ int64, quotationID, supplierID int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	w := f.content[quotationID]
	out := w.Suppliers[:0]
	for _, s := range w.Suppliers {
		if s.ID != supplierID {
			out = append(out, s)
		}
	}
	w.Suppliers = out
	return nil
}

func (f *fakeQuotations) ReplaceLines(_ context.Context, _ int64, quotationID int64, rows []LineRow) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	ids := make([]int64, len(rows))
	lines := make([]domain.QuotationLine, len(rows))
	for i, r := range rows {
		l := r.Line
		l.ID, l.QuotationID, l.ParentLineID = f.id(), quotationID, nil
		if r.Parent >= 0 {
			p := ids[r.Parent]
			l.ParentLineID = &p
		}
		ids[i], lines[i] = l.ID, l
	}
	f.content[quotationID].Lines = lines
	return nil
}

func (f *fakeQuotations) SetLineAmounts(_ context.Context, _ int64, quotationID int64, amounts map[int64]int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	w := f.content[quotationID]
	for i := range w.Lines {
		if a, ok := amounts[w.Lines[i].ID]; ok {
			w.Lines[i].Amount = a
		}
	}
	return nil
}

func (f *fakeQuotations) ReplaceTerms(_ context.Context, _ int64, quotationID int64, terms []domain.QuotationTerm) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]domain.QuotationTerm, len(terms))
	for i, t := range terms {
		t.ID, t.QuotationID = f.id(), quotationID
		out[i] = t
	}
	f.content[quotationID].Terms = out
	return nil
}

func (f *fakeQuotations) findRevision(tenantID int64, match func(domain.QuotationRevision) bool) *domain.QuotationRevision {
	for _, r := range f.revisions {
		if f.revTenant[r.ID] == tenantID && match(r) {
			c := r
			return &c
		}
	}
	return nil
}

func (f *fakeQuotations) FindRevisionByHash(_ context.Context, tenantID, quotationID int64, hash string) (*domain.QuotationRevision, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.findRevision(tenantID, func(r domain.QuotationRevision) bool { return r.QuotationID == quotationID && r.ContentHash == hash }), nil
}

func (f *fakeQuotations) FindRevision(_ context.Context, tenantID, quotationID int64, n int) (*domain.QuotationRevision, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.findRevision(tenantID, func(r domain.QuotationRevision) bool { return r.QuotationID == quotationID && r.N == n }), nil
}

func (f *fakeQuotations) FindRevisionByID(_ context.Context, tenantID, revisionID int64) (*domain.QuotationRevision, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.findRevision(tenantID, func(r domain.QuotationRevision) bool { return r.ID == revisionID }), nil
}

func (f *fakeQuotations) ListRevisions(_ context.Context, tenantID, quotationID int64) ([]domain.QuotationRevision, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []domain.QuotationRevision
	for _, r := range f.revisions {
		if f.revTenant[r.ID] == tenantID && r.QuotationID == quotationID {
			out = append(out, r)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].N > out[j].N })
	return out, nil
}

// storeRevision is the UNIQUE (tenant, quotation, hash) and (tenant,
// quotation, n) of the real table.
func (f *fakeQuotations) storeRevision(tenantID int64, r *domain.QuotationRevision) error {
	for _, e := range f.revisions {
		if f.revTenant[e.ID] == tenantID && e.QuotationID == r.QuotationID && (e.ContentHash == r.ContentHash || e.N == r.N) {
			return ErrDuplicateRevision
		}
	}
	r.ID = f.id()
	f.revisions = append(f.revisions, *r)
	f.revTenant[r.ID] = tenantID
	if q := f.row(tenantID, r.QuotationID); q != nil && q.RevisionCount < r.N {
		q.RevisionCount = r.N
	}
	return nil
}

func (f *fakeQuotations) InsertRevision(_ context.Context, tenantID int64, r *domain.QuotationRevision) error {
	if hook := f.beforeInsertRevision; hook != nil {
		f.beforeInsertRevision = nil
		hook()
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.storeRevision(tenantID, r)
}

func (f *fakeQuotations) AddActivity(_ context.Context, tenantID int64, a domain.Activity) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	a.ID, a.CreatedAt = f.id(), time.Now().UTC()
	f.activity = append(f.activity, a)
	f.actTenant[a.ID] = tenantID
	return nil
}

func (f *fakeQuotations) ListActivity(_ context.Context, tenantID, quotationID int64) ([]domain.Activity, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []domain.Activity
	for i := len(f.activity) - 1; i >= 0; i-- {
		if a := f.activity[i]; f.actTenant[a.ID] == tenantID && a.QuotationID == quotationID {
			out = append(out, a)
		}
	}
	return out, nil
}

func (f *fakeQuotations) PackageReferenced(context.Context, int64, int64) (bool, error) {
	return false, nil
}
func (f *fakeQuotations) VendorReferenced(context.Context, int64, int64) (bool, error) {
	return false, nil
}
func (f *fakeQuotations) VenueReferenced(context.Context, int64, int64) (bool, error) {
	return false, nil
}
func (f *fakeQuotations) CategoryReferenced(context.Context, int64, int64) (bool, error) {
	return false, nil
}
func (f *fakeQuotations) EventKindQuotationCounts(context.Context, int64) (map[int64]int, error) {
	return map[int64]int{}, nil
}

// fakeTx serializes transactions like the quotation row lock does; a nested
// InTx joins the outer one.
type fakeTx struct{ mu sync.Mutex }

type fakeTxKey struct{}

func (t *fakeTx) InTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if ctx.Value(fakeTxKey{}) != nil {
		return fn(ctx)
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	return fn(context.WithValue(ctx, fakeTxKey{}, true))
}

type fakeProfiles struct {
	mu sync.Mutex
	by map[int64]QuotationProfile
}

func (p *fakeProfiles) QuotationProfile(_ context.Context, tenantID int64) (QuotationProfile, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.by[tenantID], nil
}

func (p *fakeProfiles) set(tenantID int64, fn func(*QuotationProfile)) {
	p.mu.Lock()
	defer p.mu.Unlock()
	v := p.by[tenantID]
	fn(&v)
	p.by[tenantID] = v
}

type fakeStaffNames struct{ names map[int64]map[int64]string }

func (s fakeStaffNames) GetName(_ context.Context, tenantID, staffID int64) (string, bool, error) {
	n, ok := s.names[tenantID][staffID]
	return n, ok, nil
}

func (s fakeStaffNames) StaffNames(_ context.Context, tenantID int64, ids []int64) (map[int64]string, error) {
	out := map[int64]string{}
	for _, id := range ids {
		if n, ok := s.names[tenantID][id]; ok {
			out[id] = n
		}
	}
	return out, nil
}

// fakeRenderer "renders" a document as its JSON, so two PDFs are equal
// exactly when their documents are.
type fakeRenderer struct {
	mu   sync.Mutex
	docs []QuotationDocument
}

func (r *fakeRenderer) Render(doc QuotationDocument) ([]byte, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.docs = append(r.docs, doc)
	return json.Marshal(doc)
}

// --- fixture -----------------------------------------------------------------------

const (
	staffSari int64 = 5
	staffDodi int64 = 6
	staffB    int64 = 50
)

var owner = Viewer{StaffID: staffSari, AllQuotations: true, CanSeeCost: true}

type quotationFixture struct {
	svc      *QuotationService
	repo     *fakeQuotations
	profiles *fakeProfiles
	renderer *fakeRenderer
	now      time.Time
}

func newQuotationFixture() *quotationFixture {
	repo := newFakeQuotations()
	profiles := &fakeProfiles{by: map[int64]QuotationProfile{
		woA: {DocumentCode: "ABC", IsPKP: true, DefaultValidityDays: 14, DefaultTermsText: "DP tidak dapat dikembalikan.",
			Identity: domain.WorkspaceIdentity{Name: "Amora Wedding", City: "Bandung", WhatsApp: "081234567890"}},
		woB: {DocumentCode: "BWO", DefaultValidityDays: 14, Identity: domain.WorkspaceIdentity{Name: "Bunga WO"}},
	}}
	staff := fakeStaffNames{names: map[int64]map[int64]string{
		woA: {staffSari: "Sari", staffDodi: "Dodi"},
		woB: {staffB: "Bima"},
	}}
	renderer := &fakeRenderer{}
	svc := NewQuotationService(repo, &fakeTx{}, &fakePackages{}, &fakeKinds{}, fakeDirectory{})
	svc.SetPorts(profiles, staff, nil, renderer)
	fx := &quotationFixture{svc: svc, repo: repo, profiles: profiles, renderer: renderer,
		now: time.Date(2026, 11, 20, 3, 0, 0, 0, time.UTC)} // 10:00 WIB
	svc.SetClock(func() time.Time { return fx.now })
	return fx
}

func (fx *quotationFixture) create(t *testing.T, tenantID int64, v Viewer) *domain.Quotation {
	t.Helper()
	q, err := fx.svc.Create(context.Background(), tenantID, v, HeaderInput{CoupleNames: "Rina & Bayu", PICStaffID: v.StaffID})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	return q
}

// seedLines writes an item line straight into the working copy.
func (fx *quotationFixture) seedLines(t *testing.T, tenantID, quotationID int64, amounts ...int64) {
	t.Helper()
	rows := make([]LineRow, len(amounts))
	for i, a := range amounts {
		rows[i] = LineRow{Line: domain.QuotationLine{Kind: domain.LineItem, Description: fmt.Sprintf("Item %d", i+1),
			Quantity: i64p(1), UnitPrice: i64p(a), Amount: a, SortOrder: i}, Parent: -1}
	}
	if err := fx.repo.ReplaceLines(context.Background(), tenantID, quotationID, rows); err != nil {
		t.Fatal(err)
	}
}

func (fx *quotationFixture) get(t *testing.T, tenantID, id int64) *domain.Quotation {
	t.Helper()
	q, err := fx.repo.FindByID(context.Background(), tenantID, id)
	if err != nil || q == nil {
		t.Fatalf("quotation %d: %v", id, err)
	}
	return q
}

func codeOf(err error) string {
	if ae, ok := apperror.As(err); ok {
		return ae.Code
	}
	return ""
}

func (fx *quotationFixture) accept(t *testing.T, tenantID, id int64) {
	t.Helper()
	if _, _, err := fx.svc.CreateRevision(context.Background(), tenantID, id, owner); err != nil {
		t.Fatalf("revision: %v", err)
	}
	if _, err := fx.svc.SetStatus(context.Background(), tenantID, id, owner, StatusInput{Status: domain.QuotationAccepted, Channel: domain.ChannelWhatsApp}); err != nil {
		t.Fatalf("accept: %v", err)
	}
}

// --- numbering ------------------------------------------------------------------------

func TestQuotationNumbersCountPerWorkspaceAndDay(t *testing.T) {
	fx := newQuotationFixture()
	first := fx.create(t, woA, owner)
	second := fx.create(t, woA, owner)
	if first.Number != "ABC-PNW-261120-001" || second.Number != "ABC-PNW-261120-002" {
		t.Fatalf("WO A numbers = %s, %s", first.Number, second.Number)
	}
	// Another workspace on the same day starts at 001 too.
	b := fx.create(t, woB, Viewer{StaffID: staffB, AllQuotations: true})
	if b.Number != "BWO-PNW-261120-001" {
		t.Fatalf("WO B number = %s, want BWO-PNW-261120-001", b.Number)
	}
	// 17:30 UTC is 00:30 WIB on the next day: a new day, a new count.
	fx.now = time.Date(2026, 11, 20, 17, 30, 0, 0, time.UTC)
	next := fx.create(t, woA, owner)
	if next.Number != "ABC-PNW-261121-001" {
		t.Fatalf("next WIB day = %s, want ABC-PNW-261121-001", next.Number)
	}
}

func TestChangingTheDocumentCodeKeepsOldNumbers(t *testing.T) {
	fx := newQuotationFixture()
	old := fx.create(t, woA, owner)
	fx.profiles.set(woA, func(p *QuotationProfile) { p.DocumentCode = "AMR" })
	fresh := fx.create(t, woA, owner)
	if fresh.Number != "AMR-PNW-261120-002" {
		t.Fatalf("new number = %s, want AMR-PNW-261120-002 (the day's count goes on)", fresh.Number)
	}
	if got := fx.get(t, woA, old.ID).Number; got != "ABC-PNW-261120-001" {
		t.Fatalf("old number changed to %s", got)
	}
}

func TestNewQuotationDefaults(t *testing.T) {
	fx := newQuotationFixture()
	q := fx.create(t, woA, owner)
	if !q.ValidUntil.Equal(time.Date(2026, 12, 4, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("validUntil = %v, want 2026-12-04 (14 days)", q.ValidUntil)
	}
	if q.Status != domain.QuotationDraft || !q.IsPKP || q.TermsText != "DP tidak dapat dikembalikan." {
		t.Fatalf("defaults: %+v", q)
	}
	w, err := fx.svc.Content(context.Background(), woA, q.ID, owner)
	if err != nil {
		t.Fatal(err)
	}
	if len(w.Terms) != 3 || *w.Terms[0].Percent != 30 || *w.Terms[1].Percent != 40 || *w.Terms[2].Percent != 30 {
		t.Fatalf("terms = %+v, want 30/40/30", w.Terms)
	}
}

func TestQuotationOfAnotherWorkspaceIsNotFound(t *testing.T) {
	fx := newQuotationFixture()
	q := fx.create(t, woA, owner)
	bOwner := Viewer{StaffID: staffB, AllQuotations: true}
	ctx := context.Background()
	checks := map[string]error{}
	_, checks["get"] = fx.svc.Get(ctx, woB, q.ID, bOwner)
	_, checks["detail"] = fx.svc.Detail(ctx, woB, q.ID, bOwner)
	_, _, checks["revision"] = fx.svc.CreateRevision(ctx, woB, q.ID, bOwner)
	_, checks["status"] = fx.svc.SetStatus(ctx, woB, q.ID, bOwner, StatusInput{Status: domain.QuotationRejected})
	_, checks["copy"] = fx.svc.Copy(ctx, woB, q.ID, bOwner)
	_, _, checks["preview"] = fx.svc.PreviewPDF(ctx, woB, q.ID, bOwner)
	checks["delete"] = fx.svc.DeleteDraft(ctx, woB, q.ID, bOwner)
	for name, err := range checks {
		if apperror.HTTPStatus(err) != 404 {
			t.Errorf("%s: err = %v, want 404", name, err)
		}
	}
	// Someone who only sees their own quotations does not see Sari's.
	if _, err := fx.svc.Get(ctx, woA, q.ID, Viewer{StaffID: staffDodi}); apperror.HTTPStatus(err) != 404 {
		t.Fatalf("assigned scope: err = %v, want 404", err)
	}
}

// --- status machine ---------------------------------------------------------------------

func TestCopySupersedesTheOriginal(t *testing.T) {
	fx := newQuotationFixture()
	ctx := context.Background()
	orig := fx.create(t, woA, owner)
	fx.seedLines(t, woA, orig.ID, 10_000_000)
	cp, err := fx.svc.Copy(ctx, woA, orig.ID, owner)
	if err != nil {
		t.Fatal(err)
	}
	if cp.Number == orig.Number || cp.Status != domain.QuotationDraft || cp.AlternativeOfID != nil {
		t.Fatalf("copy = %+v", cp)
	}
	after := fx.get(t, woA, orig.ID)
	if after.Status != domain.QuotationSuperseded || after.SupersededByID == nil || *after.SupersededByID != cp.ID {
		t.Fatalf("original = %s by %v, want Superseded by %d", after.Status, after.SupersededByID, cp.ID)
	}
	if w, _ := fx.svc.Content(ctx, woA, cp.ID, owner); len(w.Lines) != 1 || w.Lines[0].Amount != 10_000_000 {
		t.Fatalf("the copy carries the content: %+v", w.Lines)
	}
	if _, err := fx.svc.Copy(ctx, woA, orig.ID, owner); codeOf(err) != domain.CodeSuperseded {
		t.Fatalf("copying a superseded quotation: err = %v", err)
	}

	// A Rejected original stays Rejected.
	rej := fx.create(t, woA, owner)
	if _, err := fx.svc.SetStatus(ctx, woA, rej.ID, owner, StatusInput{Status: domain.QuotationRejected, Reason: "Budget"}); err != nil {
		t.Fatal(err)
	}
	if _, err := fx.svc.Copy(ctx, woA, rej.ID, owner); err != nil {
		t.Fatal(err)
	}
	if got := fx.get(t, woA, rej.ID).Status; got != domain.QuotationRejected {
		t.Fatalf("rejected original = %s", got)
	}
}

func TestAcceptingAnAlternativeSupersedesTheOthers(t *testing.T) {
	fx := newQuotationFixture()
	ctx := context.Background()
	q1 := fx.create(t, woA, owner)
	q2, err := fx.svc.AddAlternative(ctx, woA, q1.ID, owner)
	if err != nil {
		t.Fatal(err)
	}
	// An alternative of an alternative joins the same group.
	q3, err := fx.svc.AddAlternative(ctx, woA, q2.ID, owner)
	if err != nil {
		t.Fatal(err)
	}
	if q2.AlternativeOfID == nil || *q2.AlternativeOfID != q1.ID || q3.AlternativeOfID == nil || *q3.AlternativeOfID != q1.ID {
		t.Fatalf("group roots: %v, %v; want %d", q2.AlternativeOfID, q3.AlternativeOfID, q1.ID)
	}
	if fx.get(t, woA, q1.ID).Status != domain.QuotationDraft {
		t.Fatal("making an alternative supersedes nothing")
	}
	fx.seedLines(t, woA, q2.ID, 50_000_000)
	fx.accept(t, woA, q2.ID)
	for _, id := range []int64{q1.ID, q3.ID} {
		got := fx.get(t, woA, id)
		if got.Status != domain.QuotationSuperseded || got.SupersededByID == nil || *got.SupersededByID != q2.ID {
			t.Fatalf("quotation %d = %s by %v, want Superseded by %d", id, got.Status, got.SupersededByID, q2.ID)
		}
	}
	// Unaccepting does not bring the others back.
	if _, err := fx.svc.SetStatus(ctx, woA, q2.ID, owner, StatusInput{Status: domain.QuotationDraft}); err != nil {
		t.Fatal(err)
	}
	if got := fx.get(t, woA, q2.ID); got.Status != domain.QuotationDraft || got.AcceptedRevisionID != nil {
		t.Fatalf("unaccepted = %+v", got)
	}
	for _, id := range []int64{q1.ID, q3.ID} {
		if got := fx.get(t, woA, id).Status; got != domain.QuotationSuperseded {
			t.Fatalf("quotation %d revived to %s", id, got)
		}
	}
}

func TestUnacceptOnlyBeforeConversion(t *testing.T) {
	fx := newQuotationFixture()
	ctx := context.Background()
	q := fx.create(t, woA, owner)
	fx.seedLines(t, woA, q.ID, 20_000_000)
	fx.accept(t, woA, q.ID)
	if err := fx.svc.MarkConverted(ctx, woA, q.ID, 77); err != nil {
		t.Fatal(err)
	}
	// Idempotent.
	if err := fx.svc.MarkConverted(ctx, woA, q.ID, 77); err != nil {
		t.Fatal(err)
	}
	if _, err := fx.svc.SetStatus(ctx, woA, q.ID, owner, StatusInput{Status: domain.QuotationDraft}); codeOf(err) != domain.CodeConverted {
		t.Fatalf("unaccept after conversion: err = %v, want %s", err, domain.CodeConverted)
	}
	got, rev, err := fx.svc.AcceptedQuotation(ctx, woA, q.ID)
	if err != nil || got.ConvertedProjectID == nil || *got.ConvertedProjectID != 77 || rev.N != 1 {
		t.Fatalf("accepted quotation after conversion: %+v %+v %v", got, rev, err)
	}
	if _, _, err := fx.svc.AcceptedQuotation(ctx, woB, q.ID); apperror.HTTPStatus(err) != 404 {
		t.Fatalf("another workspace: err = %v, want 404", err)
	}
}

func TestExpiredQuotationNeedsAnExtensionToBeAccepted(t *testing.T) {
	fx := newQuotationFixture()
	ctx := context.Background()
	q := fx.create(t, woA, owner)
	fx.seedLines(t, woA, q.ID, 30_000_000)
	if _, _, err := fx.svc.CreateRevision(ctx, woA, q.ID, owner); err != nil {
		t.Fatal(err)
	}
	fx.now = fx.now.AddDate(0, 0, 20)
	_, err := fx.svc.SetStatus(ctx, woA, q.ID, owner, StatusInput{Status: domain.QuotationAccepted, Channel: domain.ChannelWhatsApp})
	if codeOf(err) != domain.CodeNeedsExtension {
		t.Fatalf("err = %v, want %s", err, domain.CodeNeedsExtension)
	}
	until := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
	got, err := fx.svc.SetStatus(ctx, woA, q.ID, owner, StatusInput{Status: domain.QuotationAccepted, Channel: domain.ChannelWhatsApp, ExtendUntil: &until})
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != domain.QuotationAccepted || !got.ValidUntil.Equal(until) || got.AcceptedRevisionN != 1 {
		t.Fatalf("accepted = %+v", got)
	}
}

func TestAcceptNeedsTheWorkingCopyAsARevision(t *testing.T) {
	fx := newQuotationFixture()
	ctx := context.Background()
	q := fx.create(t, woA, owner)
	fx.seedLines(t, woA, q.ID, 30_000_000)
	if _, _, err := fx.svc.CreateRevision(ctx, woA, q.ID, owner); err != nil {
		t.Fatal(err)
	}
	fx.seedLines(t, woA, q.ID, 31_000_000)
	_, err := fx.svc.SetStatus(ctx, woA, q.ID, owner, StatusInput{Status: domain.QuotationAccepted, Channel: domain.ChannelWhatsApp})
	if codeOf(err) != domain.CodeNeedsRevision {
		t.Fatalf("err = %v, want %s", err, domain.CodeNeedsRevision)
	}
}

// --- revisions ----------------------------------------------------------------------------

func TestRevisionNeedsLines(t *testing.T) {
	fx := newQuotationFixture()
	q := fx.create(t, woA, owner)
	if _, _, err := fx.svc.CreateRevision(context.Background(), woA, q.ID, owner); codeOf(err) != CodeRevisionNeedsLines {
		t.Fatalf("err = %v, want %s", err, CodeRevisionNeedsLines)
	}
}

func TestSameContentGivesTheSameRevision(t *testing.T) {
	fx := newQuotationFixture()
	ctx := context.Background()
	q := fx.create(t, woA, owner)
	fx.seedLines(t, woA, q.ID, 40_000_000)
	r1, created, err := fx.svc.CreateRevision(ctx, woA, q.ID, owner)
	if err != nil || !created || r1.N != 1 {
		t.Fatalf("first: %+v created=%v err=%v", r1, created, err)
	}
	again, created, err := fx.svc.CreateRevision(ctx, woA, q.ID, owner)
	if err != nil || created || again.N != 1 {
		t.Fatalf("unchanged: %+v created=%v err=%v", again, created, err)
	}
	// Moving the validity is not a new offer.
	until := time.Date(2026, 12, 20, 0, 0, 0, 0, time.UTC)
	if _, err := fx.svc.Extend(ctx, woA, q.ID, owner, until); err != nil {
		t.Fatal(err)
	}
	if _, created, _ := fx.svc.CreateRevision(ctx, woA, q.ID, owner); created {
		t.Fatal("a new validity alone must not make a revision")
	}
	fx.seedLines(t, woA, q.ID, 45_000_000)
	r2, created, err := fx.svc.CreateRevision(ctx, woA, q.ID, owner)
	if err != nil || !created || r2.N != 2 || r2.ContractTotal == r1.ContractTotal {
		t.Fatalf("changed: %+v created=%v err=%v", r2, created, err)
	}
}

func TestConcurrentRevisionsMakeOne(t *testing.T) {
	fx := newQuotationFixture()
	q := fx.create(t, woA, owner)
	fx.seedLines(t, woA, q.ID, 60_000_000)
	type result struct {
		rev     *domain.QuotationRevision
		created bool
		err     error
	}
	results := make(chan result, 2)
	start := make(chan struct{})
	for i := 0; i < 2; i++ {
		go func() {
			<-start
			rev, created, err := fx.svc.CreateRevision(context.Background(), woA, q.ID, owner)
			results <- result{rev, created, err}
		}()
	}
	close(start)
	made := 0
	for i := 0; i < 2; i++ {
		r := <-results
		if r.err != nil {
			t.Fatal(r.err)
		}
		if r.rev.N != 1 {
			t.Fatalf("revision n = %d, want 1", r.rev.N)
		}
		if r.created {
			made++
		}
	}
	revs, _ := fx.repo.ListRevisions(context.Background(), woA, q.ID)
	if made != 1 || len(revs) != 1 {
		t.Fatalf("created %d, stored %d; want 1 and 1", made, len(revs))
	}
}

func TestDuplicateRevisionFromAnotherRequestIsReused(t *testing.T) {
	fx := newQuotationFixture()
	ctx := context.Background()
	q := fx.create(t, woA, owner)
	fx.seedLines(t, woA, q.ID, 60_000_000)
	w, _ := fx.svc.Content(ctx, woA, q.ID, owner)
	w.Quotation = *fx.get(t, woA, q.ID)
	c, err := domain.BuildContent(*w)
	if err != nil {
		t.Fatal(err)
	}
	// Another request stores the same content between our lookup and insert
	// (the unique hash answers ErrDuplicateRevision).
	var theirs domain.QuotationRevision
	fx.repo.beforeInsertRevision = func() {
		fx.repo.mu.Lock()
		defer fx.repo.mu.Unlock()
		theirs = domain.QuotationRevision{QuotationID: q.ID, N: 1, ContentHash: c.Hash(), ContractTotal: c.Totals.Total}
		if err := fx.repo.storeRevision(woA, &theirs); err != nil {
			t.Error(err)
		}
	}
	rev, created, err := fx.svc.CreateRevision(ctx, woA, q.ID, owner)
	if err != nil || created || rev.ID != theirs.ID {
		t.Fatalf("got %+v created=%v err=%v; want their revision %d", rev, created, err, theirs.ID)
	}
}

func TestPreviewMakesNoRevision(t *testing.T) {
	fx := newQuotationFixture()
	ctx := context.Background()
	q := fx.create(t, woA, owner)
	fx.seedLines(t, woA, q.ID, 25_000_000)
	before, _ := fx.repo.ListActivity(ctx, woA, q.ID)
	data, name, err := fx.svc.PreviewPDF(ctx, woA, q.ID, owner)
	if err != nil || len(data) == 0 {
		t.Fatalf("preview: %v", err)
	}
	if name != q.Number+"-Pratinjau.pdf" {
		t.Fatalf("file name = %s", name)
	}
	if doc := fx.renderer.docs[len(fx.renderer.docs)-1]; !doc.Draft {
		t.Fatal("a preview carries the DRAF watermark")
	}
	revs, _ := fx.repo.ListRevisions(ctx, woA, q.ID)
	after, _ := fx.repo.ListActivity(ctx, woA, q.ID)
	if len(revs) != 0 || fx.get(t, woA, q.ID).RevisionCount != 0 || len(after) != len(before) {
		t.Fatalf("preview left %d revisions and %d new activities", len(revs), len(after)-len(before))
	}
}

func TestRevisionOnePDFIsTheSameAfterRevisionTwo(t *testing.T) {
	fx := newQuotationFixture()
	ctx := context.Background()
	q := fx.create(t, woA, owner)
	fx.seedLines(t, woA, q.ID, 70_000_000)
	if _, _, err := fx.svc.CreateRevision(ctx, woA, q.ID, owner); err != nil {
		t.Fatal(err)
	}
	first, name, err := fx.svc.RevisionPDF(ctx, woA, q.ID, 1, owner)
	if err != nil {
		t.Fatal(err)
	}
	if name != q.Number+"-Rev1.pdf" {
		t.Fatalf("file name = %s", name)
	}
	// Everything moves: the lines, the PIC, the WO's profile.
	fx.seedLines(t, woA, q.ID, 75_000_000, 5_000_000)
	fx.profiles.set(woA, func(p *QuotationProfile) { p.Identity.Name = "Amora Wedding Organizer"; p.IsPKP = false })
	if _, err := fx.svc.UpdateHeader(ctx, woA, q.ID, owner, HeaderInput{CoupleNames: "Rina & Bayu Pratama", PICStaffID: staffDodi}, nil); err != nil {
		t.Fatal(err)
	}
	fx.now = fx.now.Add(48 * time.Hour)
	if _, created, err := fx.svc.CreateRevision(ctx, woA, q.ID, owner); err != nil || !created {
		t.Fatalf("revision 2: created=%v err=%v", created, err)
	}
	again, _, err := fx.svc.RevisionPDF(ctx, woA, q.ID, 1, owner)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, again) {
		t.Fatal("Revision 1's PDF changed after Revision 2")
	}
	second, _, err := fx.svc.RevisionPDF(ctx, woA, q.ID, 2, owner)
	if err != nil || bytes.Equal(first, second) {
		t.Fatalf("Revision 2 must differ from Revision 1 (err %v)", err)
	}
}

func TestDocumentNeverShowsAgreedCost(t *testing.T) {
	snap := domain.Snapshot{Number: "ABC-PNW-261120-001", RevisionN: 1, IssuedAt: time.Date(2026, 11, 20, 3, 0, 0, 0, time.UTC), ValidUntil: "2026-12-04",
		Workspace: domain.WorkspaceIdentity{Name: "Amora Wedding"},
		Content: domain.Content{CoupleNames: "Rina & Bayu", IsPKP: false,
			Suppliers: []domain.SnapshotSupplier{
				{Kind: domain.SupplierVendor, Name: "Katering Sedap", PaidBy: domain.PayerClient, ShowName: true, AgreedCost: i64p(98_765_432)},
				{Kind: domain.SupplierVendor, Name: "Dekor Rahasia", PaidBy: domain.PayerWO, AgreedCost: i64p(12_345_678)},
			},
			Lines:  []domain.SnapshotLine{{Kind: domain.LineItem, Description: "Photo booth", Amount: 5_000_000}},
			Totals: domain.Totals{Gross: 5_000_000, Subtotal: 5_000_000, Total: 5_000_000},
		}}
	raw, _ := json.Marshal(BuildDocument(snap, false, "Revisi 1", nil, ""))
	for _, secret := range []string{"98.765.432", "98765432", "12.345.678", "12345678"} {
		if bytes.Contains(raw, []byte(secret)) {
			t.Fatalf("the document carries an agreed cost (%s)", secret)
		}
	}
	if !bytes.Contains(raw, []byte("Katering Sedap")) {
		t.Fatal("a client-paid vendor shown by name is listed")
	}
}
