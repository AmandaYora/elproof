package application

import (
	"context"
	"time"

	offerscontracts "elproof/internal/modules/offers/contracts"
	"elproof/internal/modules/projects/domain"
	vendorscontracts "elproof/internal/modules/vendors/contracts"
	"elproof/internal/shared/apperror"
)

// ---- in-memory suppliers, supplier payments and events ----

// memSuppliers is an in-memory SupplierRepository. Attachments counts the live
// payments of pays (when wired) plus whatever extra says.
type memSuppliers struct {
	rows        []domain.ProjectSupplier
	next        int64
	pays        *memSupplierPayments
	extra       map[int64]domain.SupplierAttachments
	writes      int
	hardDeleted []int64
}

func (m *memSuppliers) snapshot() func() {
	rows, next, writes := append([]domain.ProjectSupplier(nil), m.rows...), m.next, m.writes
	hd := append([]int64(nil), m.hardDeleted...)
	return func() { m.rows, m.next, m.writes, m.hardDeleted = rows, next, writes, hd }
}

func (m *memSuppliers) live(projectID int64) []domain.ProjectSupplier {
	var out []domain.ProjectSupplier
	for _, r := range m.rows {
		if r.ProjectID == projectID && r.DeletedAt == nil {
			out = append(out, r)
		}
	}
	return out
}

func (m *memSuppliers) ListByProject(_ context.Context, projectID int64) ([]domain.ProjectSupplier, error) {
	return m.live(projectID), nil
}
func (m *memSuppliers) ListByProjects(_ context.Context, ids []int64) ([]domain.ProjectSupplier, error) {
	var out []domain.ProjectSupplier
	for _, id := range ids {
		out = append(out, m.live(id)...)
	}
	return out, nil
}
func (m *memSuppliers) FindByID(_ context.Context, projectID, id int64) (*domain.ProjectSupplier, error) {
	for _, r := range m.rows {
		if r.ProjectID == projectID && r.ID == id && r.DeletedAt == nil {
			s := r
			return &s, nil
		}
	}
	return nil, nil
}
func (m *memSuppliers) FindDeletedByID(_ context.Context, projectID, id int64) (*domain.ProjectSupplier, error) {
	for _, r := range m.rows {
		if r.ProjectID == projectID && r.ID == id && r.DeletedAt != nil {
			s := r
			return &s, nil
		}
	}
	return nil, nil
}
func (m *memSuppliers) FindLiveVenue(_ context.Context, projectID, venueID int64) (*domain.ProjectSupplier, error) {
	for _, r := range m.rows {
		if r.ProjectID == projectID && r.Kind == domain.SupplierVenue && r.SupplierID == venueID && r.DeletedAt == nil {
			s := r
			return &s, nil
		}
	}
	return nil, nil
}
func (m *memSuppliers) Create(_ context.Context, s *domain.ProjectSupplier) error {
	m.next++
	s.ID = m.next
	m.rows = append(m.rows, *s)
	return nil
}
func (m *memSuppliers) Patch(_ context.Context, s *domain.ProjectSupplier, _ []string, _ *time.Time) (bool, error) {
	for i := range m.rows {
		if m.rows[i].ID == s.ID && m.rows[i].DeletedAt == nil {
			m.rows[i] = *s
			m.writes++
			return true, nil
		}
	}
	return false, nil
}
func (m *memSuppliers) SoftDelete(_ context.Context, _, id int64, at time.Time) (bool, error) {
	for i := range m.rows {
		if m.rows[i].ID == id && m.rows[i].DeletedAt == nil {
			m.rows[i].DeletedAt = &at
			return true, nil
		}
	}
	return false, nil
}
func (m *memSuppliers) Restore(_ context.Context, _, id int64, notBefore time.Time) (bool, error) {
	for i := range m.rows {
		if m.rows[i].ID == id && m.rows[i].DeletedAt != nil && m.rows[i].DeletedAt.After(notBefore) {
			m.rows[i].DeletedAt = nil
			return true, nil
		}
	}
	return false, nil
}
func (m *memSuppliers) HardDelete(_ context.Context, _, id int64) error {
	for i := range m.rows {
		if m.rows[i].ID == id {
			m.rows = append(m.rows[:i], m.rows[i+1:]...)
			m.hardDeleted = append(m.hardDeleted, id)
			return nil
		}
	}
	return nil
}
func (m *memSuppliers) Attachments(_ context.Context, _, id int64) (domain.SupplierAttachments, error) {
	a := m.extra[id]
	if m.pays != nil {
		for _, p := range m.pays.rows {
			if p.ProjectSupplierID == id && p.DeletedAt == nil {
				a.Payments++
			}
		}
	}
	return a, nil
}
func (m *memSuppliers) AttachmentsByProject(ctx context.Context, projectID int64) (map[int64]domain.SupplierAttachments, error) {
	out := map[int64]domain.SupplierAttachments{}
	for _, r := range m.live(projectID) {
		out[r.ID], _ = m.Attachments(ctx, projectID, r.ID)
	}
	return out, nil
}
func (m *memSuppliers) ListBySupplier(context.Context, int64, domain.SupplierKind, int64) ([]domain.SupplierHistoryRow, error) {
	return nil, nil
}

// memSupplierPayments is an in-memory SupplierPaymentRepository.
type memSupplierPayments struct {
	rows    []domain.SupplierPayment
	next    int64
	patches int
}

func (m *memSupplierPayments) ListByProject(_ context.Context, projectID int64) ([]domain.SupplierPayment, error) {
	var out []domain.SupplierPayment
	for _, r := range m.rows {
		if r.ProjectID == projectID && r.DeletedAt == nil {
			out = append(out, r)
		}
	}
	return out, nil
}
func (m *memSupplierPayments) ListByProjects(ctx context.Context, ids []int64) ([]domain.SupplierPayment, error) {
	var out []domain.SupplierPayment
	for _, id := range ids {
		list, _ := m.ListByProject(ctx, id)
		out = append(out, list...)
	}
	return out, nil
}
func (m *memSupplierPayments) FindByID(_ context.Context, _, id int64) (*domain.SupplierPayment, error) {
	for _, r := range m.rows {
		if r.ID == id && r.DeletedAt == nil {
			p := r
			return &p, nil
		}
	}
	return nil, nil
}
func (m *memSupplierPayments) FindDeletedByID(_ context.Context, _, id int64) (*domain.SupplierPayment, error) {
	for _, r := range m.rows {
		if r.ID == id && r.DeletedAt != nil {
			p := r
			return &p, nil
		}
	}
	return nil, nil
}
func (m *memSupplierPayments) Create(_ context.Context, p *domain.SupplierPayment) error {
	m.next++
	p.ID = m.next
	m.rows = append(m.rows, *p)
	return nil
}
func (m *memSupplierPayments) Patch(_ context.Context, p *domain.SupplierPayment, _ []string, _ *time.Time) (bool, error) {
	m.patches++
	for i := range m.rows {
		if m.rows[i].ID == p.ID {
			m.rows[i] = *p
			return true, nil
		}
	}
	return false, nil
}
func (m *memSupplierPayments) SoftDelete(_ context.Context, _, id int64, at time.Time) (bool, error) {
	for i := range m.rows {
		if m.rows[i].ID == id && m.rows[i].DeletedAt == nil {
			m.rows[i].DeletedAt = &at
			return true, nil
		}
	}
	return false, nil
}
func (m *memSupplierPayments) Restore(_ context.Context, _, id int64, notBefore time.Time) (bool, error) {
	for i := range m.rows {
		if m.rows[i].ID == id && m.rows[i].DeletedAt != nil && m.rows[i].DeletedAt.After(notBefore) {
			m.rows[i].DeletedAt = nil
			return true, nil
		}
	}
	return false, nil
}

// memEvents is an in-memory EventRepository (sessions travel with the event).
// failCreate makes Create fail like a database error.
type memEvents struct {
	rows       []domain.ProjectEvent
	next       int64
	failCreate bool
}

func (m *memEvents) snapshot() func() {
	rows, next := append([]domain.ProjectEvent(nil), m.rows...), m.next
	return func() { m.rows, m.next = rows, next }
}

func copyEvent(e domain.ProjectEvent) domain.ProjectEvent {
	e.Sessions = append([]domain.EventSession(nil), e.Sessions...)
	return e
}

func (m *memEvents) ListByProject(_ context.Context, projectID int64) ([]domain.ProjectEvent, error) {
	var out []domain.ProjectEvent
	for _, e := range m.rows {
		if e.ProjectID == projectID {
			out = append(out, copyEvent(e))
		}
	}
	return out, nil
}
func (m *memEvents) ListByProjects(ctx context.Context, ids []int64) ([]domain.ProjectEvent, error) {
	var out []domain.ProjectEvent
	for _, id := range ids {
		list, _ := m.ListByProject(ctx, id)
		out = append(out, list...)
	}
	return out, nil
}
func (m *memEvents) FindByID(_ context.Context, projectID, id int64) (*domain.ProjectEvent, error) {
	for _, e := range m.rows {
		if e.ProjectID == projectID && e.ID == id {
			c := copyEvent(e)
			return &c, nil
		}
	}
	return nil, nil
}
func (m *memEvents) Create(_ context.Context, e *domain.ProjectEvent) error {
	if m.failCreate {
		return apperror.Internal("disk full")
	}
	m.next++
	e.ID = m.next
	m.rows = append(m.rows, copyEvent(*e))
	return nil
}
func (m *memEvents) Replace(_ context.Context, e *domain.ProjectEvent, _ *time.Time) (bool, error) {
	for i := range m.rows {
		if m.rows[i].ID == e.ID && m.rows[i].ProjectID == e.ProjectID {
			m.rows[i] = copyEvent(*e)
			return true, nil
		}
	}
	return false, nil
}
func (m *memEvents) Delete(_ context.Context, projectID, id int64) error {
	for i := range m.rows {
		if m.rows[i].ProjectID == projectID && m.rows[i].ID == id {
			m.rows = append(m.rows[:i], m.rows[i+1:]...)
			return nil
		}
	}
	return nil
}
func (m *memEvents) SetMain(_ context.Context, projectID, id int64) error {
	for i := range m.rows {
		if m.rows[i].ProjectID == projectID {
			m.rows[i].IsMain = m.rows[i].ID == id
		}
	}
	return nil
}
func (m *memEvents) SetSortOrder(_ context.Context, projectID, id int64, order int) error {
	for i := range m.rows {
		if m.rows[i].ProjectID == projectID && m.rows[i].ID == id {
			m.rows[i].SortOrder = order
		}
	}
	return nil
}
func (m *memEvents) CountByVenueSupplier(_ context.Context, projectID, supplierID int64) (int, error) {
	n := 0
	for _, e := range m.rows {
		if e.ProjectID == projectID && e.VenueSupplierID != nil && *e.VenueSupplierID == supplierID {
			n++
		}
	}
	return n, nil
}
func (m *memEvents) NextSortOrder(_ context.Context, projectID int64) (int, error) {
	n := 0
	for _, e := range m.rows {
		if e.ProjectID == projectID && e.SortOrder >= n {
			n = e.SortOrder + 1
		}
	}
	return n, nil
}

// ---- the other modules, as their contracts answer ----

// fakeCatalog knows the event kinds of each workspace and applies the real
// capacity rule of `offers` (through its contract).
type fakeCatalog struct {
	kinds map[int64]map[int64]offerscontracts.EventKindRef
}

func (c fakeCatalog) GetEventKind(_ context.Context, tenantID, id int64) (offerscontracts.EventKindRef, error) {
	if k, ok := c.kinds[tenantID][id]; ok {
		return k, nil
	}
	return offerscontracts.EventKindRef{}, apperror.NotFound("Jenis acara tidak ditemukan")
}
func (fakeCatalog) CheckCapacity(events []offerscontracts.CapacityEvent, venues map[int64]offerscontracts.CapacityVenue) []offerscontracts.CapacityProblem {
	return offerscontracts.New(nil, nil).CheckCapacity(events, venues)
}

// fakeVenueDir knows the directory venues of each workspace.
type fakeVenueDir map[int64]map[int64]vendorscontracts.VenueRef

func (d fakeVenueDir) GetVenueRef(_ context.Context, tenantID, id int64) (vendorscontracts.VenueRef, error) {
	if v, ok := d[tenantID][id]; ok {
		return v, nil
	}
	return vendorscontracts.VenueRef{}, apperror.NotFound("Venue tidak ditemukan")
}

func intp(n int) *int { return &n }

// Event kinds and venues of workspace 1 (workspace 2 has one of each too).
const (
	kindAkad    int64 = 1
	kindResepsi int64 = 2

	venueMasjid    int64 = 11 // 150 seated
	venueGedung    int64 = 12 // 600 standing
	venueKecil     int64 = 13 // 500 standing
	venueNonaktif  int64 = 14
	venueOtherTeam int64 = 21 // workspace 2
)

func testCatalog() fakeCatalog {
	return fakeCatalog{kinds: map[int64]map[int64]offerscontracts.EventKindRef{
		1: {
			kindAkad:    {ID: kindAkad, Name: "Akad", DefaultKey: "akad", IsActive: true},
			kindResepsi: {ID: kindResepsi, Name: "Resepsi", DefaultKey: "resepsi", IsActive: true},
		},
		2: {9: {ID: 9, Name: "Akad", IsActive: true}},
	}}
}

func testVenues() fakeVenueDir {
	return fakeVenueDir{
		1: {
			venueMasjid:   {ID: venueMasjid, Name: "Masjid Al-Ikhlas", CapacitySeated: intp(150), IsActive: true},
			venueGedung:   {ID: venueGedung, Name: "Gedung Melati", CapacityStanding: intp(600), CapacitySeated: intp(400), IsActive: true},
			venueKecil:    {ID: venueKecil, Name: "Aula Kenanga", CapacityStanding: intp(500), IsActive: true},
			venueNonaktif: {ID: venueNonaktif, Name: "Gedung Lama", CapacityStanding: intp(900), IsActive: false},
		},
		2: {venueOtherTeam: {ID: venueOtherTeam, Name: "Venue lawan", CapacityStanding: intp(1000), IsActive: true}},
	}
}

func i64p(n int64) *int64 { return &n }

func session(label string, guests int) SessionInput {
	return SessionInput{Label: label, Guests: guests}
}

func akadAtMasjid(date time.Time) EventInput {
	return EventInput{EventKindID: kindAkad, EventDate: date, VenueID: i64p(venueMasjid), Layout: domain.LayoutSeated,
		Sessions: []SessionInput{session("Sesi 1", 120)}}
}

func resepsiAtGedung(date time.Time, guests ...int) EventInput {
	in := EventInput{EventKindID: kindResepsi, EventDate: date, VenueID: i64p(venueGedung), Layout: domain.LayoutStanding}
	for i, g := range guests {
		in.Sessions = append(in.Sessions, session("Sesi "+formatID(int64(i+1)), g))
	}
	return in
}
