package application

import (
	"context"
	"fmt"
	"time"

	"elproof/internal/modules/projects/domain"
	"elproof/internal/shared/apperror"
)

// SupplierRepository stores the vendors and venues of projects
// (project_suppliers). Every list and lookup skips deleted rows unless its name
// says otherwise.
type SupplierRepository interface {
	ListByProject(ctx context.Context, projectID int64) ([]domain.ProjectSupplier, error)
	// ListByProjects is every live supplier of the given projects in one query.
	ListByProjects(ctx context.Context, projectIDs []int64) ([]domain.ProjectSupplier, error)
	FindByID(ctx context.Context, projectID, id int64) (*domain.ProjectSupplier, error)
	// FindDeletedByID is the restore lookup (nil unless it waits in the "deleted" state).
	FindDeletedByID(ctx context.Context, projectID, id int64) (*domain.ProjectSupplier, error)
	// FindLiveVenue is the project's live venue supplier for a directory venue
	// (nil when the project has none).
	FindLiveVenue(ctx context.Context, projectID, venueID int64) (*domain.ProjectSupplier, error)
	// Create inserts the supplier; its tenant is the project's.
	Create(ctx context.Context, s *domain.ProjectSupplier) error
	// Patch writes the changed fields (compare-and-set on updated_at when
	// expected is given; see infrastructure.patchRow).
	Patch(ctx context.Context, s *domain.ProjectSupplier, changed []string, expected *time.Time) (bool, error)
	SoftDelete(ctx context.Context, projectID, id int64, at time.Time) (bool, error)
	Restore(ctx context.Context, projectID, id int64, notBefore time.Time) (bool, error)
	// HardDelete removes a supplier nothing hangs off (a venue no event uses).
	HardDelete(ctx context.Context, projectID, id int64) error
	// Attachments counts the live payments, files, steps and kendala of a supplier.
	Attachments(ctx context.Context, projectID, id int64) (domain.SupplierAttachments, error)
	// AttachmentsByProject is Attachments for every supplier of a project, by id.
	AttachmentsByProject(ctx context.Context, projectID int64) (map[int64]domain.SupplierAttachments, error)
	// ListBySupplier is every project of the workspace a directory vendor or
	// venue works on (live suppliers only), newest event first; Location is
	// left for the service to fill.
	ListBySupplier(ctx context.Context, tenantID int64, kind domain.SupplierKind, supplierID int64) ([]domain.SupplierHistoryRow, error)
}

// VendorMilestoneRepository stores the steps of a project supplier.
type VendorMilestoneRepository interface {
	ListBySupplier(ctx context.Context, projectSupplierID int64) ([]domain.VendorMilestone, error)
	// ListBySuppliers is every step of the given suppliers in one query.
	ListBySuppliers(ctx context.Context, projectSupplierIDs []int64) ([]domain.VendorMilestone, error)
	FindByID(ctx context.Context, projectSupplierID, id int64) (*domain.VendorMilestone, error)
	Create(ctx context.Context, m *domain.VendorMilestone) error
	Patch(ctx context.Context, m *domain.VendorMilestone, changed []string, expected *time.Time) (bool, error)
	NextSortOrder(ctx context.Context, projectSupplierID int64) (int, error)
}

// SupplierService runs "Vendor & venue" of a project: the vendors the WO adds,
// the venues its events bring in (EventService), who pays each one, what the WO
// agreed to pay, and their steps.
type SupplierService struct {
	repo       SupplierRepository
	milestones VendorMilestoneRepository
	events     EventRepository
	payments   SupplierPaymentRepository
	activity   *ActivityService
	// capacity is the event side of the capacity rule (SetCapacity), so new
	// numbers for a venue are checked against the events held there.
	capacity *EventService
}

// SetCapacity completes the wiring once the EventService exists (it is built
// after this one).
func (s *SupplierService) SetCapacity(e *EventService) { s.capacity = e }

func NewSupplierService(repo SupplierRepository, milestones VendorMilestoneRepository, events EventRepository, payments SupplierPaymentRepository, activity *ActivityService) *SupplierService {
	return &SupplierService{repo: repo, milestones: milestones, events: events, payments: payments, activity: activity}
}

var errSupplierNotFound = apperror.NotFound("Vendor atau venue tidak ditemukan")

func (s *SupplierService) List(ctx context.Context, projectID int64) ([]domain.ProjectSupplier, error) {
	return s.repo.ListByProject(ctx, projectID)
}

// Attachments is what hangs off each supplier of the project, by id.
func (s *SupplierService) Attachments(ctx context.Context, projectID int64) (map[int64]domain.SupplierAttachments, error) {
	return s.repo.AttachmentsByProject(ctx, projectID)
}

func (s *SupplierService) Get(ctx context.Context, projectID, id int64) (*domain.ProjectSupplier, error) {
	sup, err := s.repo.FindByID(ctx, projectID, id)
	if err != nil {
		return nil, err
	}
	if sup == nil {
		return nil, errSupplierNotFound
	}
	return sup, nil
}

// SupplierInput is a vendor added to a project. Name and CategoryID are what
// the directory says now (the handler resolved them); they are kept as copies.
type SupplierInput struct {
	VendorID         int64
	Name             string
	CategoryID       int64
	PaidBy           domain.Payer
	ShowName         bool
	AgreedCost       *int64
	EngagementStatus domain.EngagementStatus
	BookingDate      *time.Time
	DueDate          *time.Time
	PICStaffID       int64
	Scope            string
	Notes            string
}

func checkAgreedCost(fe *fieldErrors, paidBy domain.Payer, cost *int64) {
	if cost == nil {
		return
	}
	if paidBy != domain.PayerWO {
		fe.add("Biaya rekanan hanya diisi untuk vendor atau venue yang dibayar WO", "agreedCost")
		return
	}
	if *cost < 0 || *cost > domain.MaxMoney {
		fe.add("Biaya rekanan harus 0 atau lebih dan wajar", "agreedCost")
	}
}

// Create adds a directory vendor to the project. Venues come in through their
// events (EventService), never here.
func (s *SupplierService) Create(ctx context.Context, projectID, actorStaffID int64, in SupplierInput) (*domain.ProjectSupplier, error) {
	fe := newFieldErrors()
	if in.PaidBy == "" {
		in.PaidBy = domain.PayerWO
	}
	if !domain.IsValidPayer(in.PaidBy) {
		fe.add("Pilih siapa yang membayar", "paidBy")
	}
	if in.EngagementStatus == "" {
		in.EngagementStatus = domain.EngagementPlanned
	}
	if !domain.IsValidEngagementStatus(in.EngagementStatus) {
		fe.add("Status kerja sama tidak valid", "engagementStatus")
	}
	checkAgreedCost(fe, in.PaidBy, in.AgreedCost)
	if in.ShowName && in.PaidBy != domain.PayerWO {
		in.ShowName = false // a supplier the client pays is always named
	}
	if err := fe.err(); err != nil {
		return nil, err
	}
	sup := &domain.ProjectSupplier{
		ProjectID: projectID, Kind: domain.SupplierVendor, SupplierID: in.VendorID, Name: in.Name, CategoryID: in.CategoryID,
		PaidBy: in.PaidBy, ShowName: in.ShowName, AgreedCost: in.AgreedCost, EngagementStatus: in.EngagementStatus,
		BookingDate: in.BookingDate, DueDate: in.DueDate, PICStaffID: in.PICStaffID, Scope: in.Scope, Notes: in.Notes,
	}
	if err := s.repo.Create(ctx, sup); err != nil {
		return nil, err
	}
	s.activity.Record(ctx, &projectID, domain.ActivityVendorAdded, actorStaffID, "project_supplier", formatID(sup.ID), sup.Name,
		"Vendor ditambahkan: "+sup.Name)
	return s.Get(ctx, projectID, sup.ID)
}

// addVenue is the venue supplier an event brings in: the project's live one for
// that directory venue, or a new one paid by the WO with the venue's name and
// capacities copied. It runs inside the caller's transaction.
func (s *SupplierService) addVenue(ctx context.Context, projectID, actorStaffID int64, v venueChoice) (*domain.ProjectSupplier, error) {
	if v.existing != nil {
		return v.existing, nil
	}
	sup := &domain.ProjectSupplier{
		ProjectID: projectID, Kind: domain.SupplierVenue, SupplierID: v.ref.ID, Name: v.ref.Name,
		PaidBy: domain.PayerWO, CapacityStanding: v.ref.CapacityStanding, CapacitySeated: v.ref.CapacitySeated,
		EngagementStatus: domain.EngagementPlanned,
	}
	if err := s.repo.Create(ctx, sup); err != nil {
		return nil, err
	}
	s.activity.Record(ctx, &projectID, domain.ActivityVendorAdded, actorStaffID, "project_supplier", formatID(sup.ID), sup.Name,
		"Venue ditambahkan: "+sup.Name)
	return sup, nil
}

// releaseVenue runs when an event stops using a venue supplier (another venue
// chosen, or the event deleted). A venue no other event uses and nothing hangs
// off is removed; one with payments, files, steps or kendala stays, shown as
// "Tidak dipakai acara mana pun" (DESAIN §5.1). Inside the caller's transaction.
func (s *SupplierService) releaseVenue(ctx context.Context, projectID, supplierID int64) error {
	used, err := s.events.CountByVenueSupplier(ctx, projectID, supplierID)
	if err != nil || used > 0 {
		return err
	}
	a, err := s.repo.Attachments(ctx, projectID, supplierID)
	if err != nil || !a.Empty() {
		return err
	}
	return s.repo.HardDelete(ctx, projectID, supplierID)
}

// SupplierPatch is what PATCH .../suppliers/{sid} sent (ADR-0036). AgreedCost,
// BookingDate and DueDate may be null (clear); Notes null clears to "".
type SupplierPatch struct {
	PaidBy           Opt[domain.Payer]
	ShowName         Opt[bool]
	AgreedCost       Opt[int64]
	EngagementStatus Opt[domain.EngagementStatus]
	BookingDate      Opt[time.Time]
	DueDate          Opt[time.Time]
	PICStaffID       Opt[int64]
	Scope, Notes     Opt[string]
}

// Patch changes the fields that were sent, and only those. "Dibayar oleh" is
// locked once a payment or a proof file hangs off the supplier (the money and
// the file would change meaning); switching to the client clears the agreed
// cost and "tampilkan nama". The agreed cost is never set below what the WO has
// already paid. The handler checks finance.margin.view before it lets
// AgreedCost through.
func (s *SupplierService) Patch(ctx context.Context, projectID, id, actorStaffID int64, patch SupplierPatch, expected *time.Time) (*domain.ProjectSupplier, error) {
	sup, err := s.Get(ctx, projectID, id)
	if err != nil {
		return nil, err
	}
	loaded := *sup
	fe := newFieldErrors()
	if patch.PaidBy.Present() && !domain.IsValidPayer(patch.PaidBy.Value()) {
		fe.add("Pilih siapa yang membayar", "paidBy")
	}
	if patch.EngagementStatus.Present() && !domain.IsValidEngagementStatus(patch.EngagementStatus.Value()) {
		fe.add("Status kerja sama tidak valid", "engagementStatus")
	}
	if err := fe.err(); err != nil {
		return nil, err
	}

	var ch changeSet
	setVal(&ch, "paidBy", "dibayar oleh", &sup.PaidBy, patch.PaidBy)
	if ch.has("paidBy") {
		a, err := s.repo.Attachments(ctx, projectID, id)
		if err != nil {
			return nil, err
		}
		if msg := a.PaidByLock(loaded.PaidBy); msg != "" {
			return nil, apperror.Validation(msg, map[string][]string{"paidBy": {msg}}).WithCode("paid_by_locked")
		}
		if sup.PaidBy == domain.PayerClient {
			if sup.AgreedCost != nil {
				sup.AgreedCost = nil
				ch.note("agreedCost", "biaya rekanan")
			}
			if sup.ShowName {
				sup.ShowName = false
				ch.note("showName", "tampilkan nama")
			}
		}
	}
	if patch.ShowName.Present() {
		if patch.ShowName.Value() && !(sup.Kind == domain.SupplierVendor && sup.PaidBy == domain.PayerWO) {
			fe.add("Nama venue dan vendor yang dibayar klien selalu tampil ke klien", "showName")
		} else {
			setVal(&ch, "showName", "tampilkan nama", &sup.ShowName, patch.ShowName)
		}
	}
	if patch.AgreedCost.Present() && !patch.AgreedCost.IsNull() {
		v := patch.AgreedCost.Value()
		checkAgreedCost(fe, sup.PaidBy, &v)
	}
	if err := fe.err(); err != nil {
		return nil, err
	}
	before := sup.AgreedCost
	setNullable(&ch, "agreedCost", "biaya rekanan", &sup.AgreedCost, patch.AgreedCost)
	setVal(&ch, "engagementStatus", "status", &sup.EngagementStatus, patch.EngagementStatus)
	setNullableDate(&ch, "bookingDate", "tanggal booking", &sup.BookingDate, patch.BookingDate)
	setNullableDate(&ch, "dueDate", "jatuh tempo", &sup.DueDate, patch.DueDate)
	setVal(&ch, "picStaffId", "PIC", &sup.PICStaffID, patch.PICStaffID)
	setVal(&ch, "scope", "lingkup", &sup.Scope, patch.Scope)
	setVal(&ch, "notes", "catatan", &sup.Notes, patch.Notes)
	if !ch.any() {
		return sup, nil
	}
	if err := refuseIfBehind(&ch, expected, loaded.UpdatedAt, &loaded); err != nil {
		return nil, err
	}
	if ch.has("agreedCost") && sup.AgreedCost != nil {
		paid, err := s.paidByWO(ctx, projectID, id)
		if err != nil {
			return nil, err
		}
		var old int64
		if before != nil {
			old = *before
		}
		if msg := costBelowPaid(old, *sup.AgreedCost, paid); msg != "" {
			return nil, apperror.Validation(msg, map[string][]string{"agreedCost": {msg}})
		}
	}
	wrote, err := s.repo.Patch(ctx, sup, ch.keys, expected)
	if err != nil {
		return nil, err
	}
	fresh, err := finishPatch(ctx, expected, wrote, func(ctx context.Context) (*domain.ProjectSupplier, error) {
		return s.repo.FindByID(ctx, projectID, id)
	}, func(p *domain.ProjectSupplier) time.Time { return p.UpdatedAt }, errSupplierNotFound)
	if err != nil {
		return nil, err
	}
	typ := domain.ActivityVendorUpdated
	if ch.has("engagementStatus") {
		typ = domain.ActivityVendorStatusChanged
	}
	s.activity.Record(ctx, &projectID, typ, actorStaffID, "project_supplier", formatID(fresh.ID), fresh.Name,
		fresh.Name+" diperbarui: "+ch.summary())
	return fresh, nil
}

// costBelowPaid is the problem of setting an agreed cost under what the WO has
// already paid towards it (DP + Termin + Pelunasan - Refund), or "" when the
// change is fine. Like the payment ledger, only a change that makes things
// worse is refused: raising a cost still below what was paid is allowed, and 0
// limits nothing.
func costBelowPaid(oldCost, newCost, paid int64) string {
	if newCost <= 0 || newCost >= paid {
		return ""
	}
	if oldCost > 0 && newCost >= oldCost {
		return ""
	}
	return fmt.Sprintf("Biaya rekanan tidak bisa di bawah yang sudah dibayar WO (%s). Naikkan nilainya, atau koreksi pembayarannya lebih dulu.",
		formatRupiah(paid))
}

// paidByWO is what the WO has paid towards one supplier's agreed cost.
func (s *SupplierService) paidByWO(ctx context.Context, projectID, supplierID int64) (int64, error) {
	if s.payments == nil {
		return 0, nil
	}
	all, err := s.payments.ListByProject(ctx, projectID)
	if err != nil {
		return 0, err
	}
	var lines []domain.PaymentLine
	for _, p := range all {
		if p.ProjectSupplierID == supplierID && p.Payer == domain.PayerWO {
			lines = append(lines, lineOf(p.Type, p.Amount))
		}
	}
	return domain.AppliedToContract(lines), nil
}

// RefreshCapacity copies the directory venue's current capacities into the
// project's venue (they are copies taken when it was chosen). ref is what the
// directory says now. The events held there are checked against the new numbers
// first (PLAN §4.1): a session that no longer fits refuses the update, with the
// numbers, and nothing changes.
func (s *SupplierService) RefreshCapacity(ctx context.Context, projectID, id, actorStaffID int64, standing, seated *int) (*domain.ProjectSupplier, error) {
	sup, err := s.Get(ctx, projectID, id)
	if err != nil {
		return nil, err
	}
	if sup.Kind != domain.SupplierVenue {
		return nil, apperror.Validation("Hanya venue yang punya kapasitas", map[string][]string{"kind": {"Hanya venue yang punya kapasitas"}})
	}
	var ch changeSet
	setNullable(&ch, "capacityStanding", "kapasitas berdiri", &sup.CapacityStanding, optOf(standing))
	setNullable(&ch, "capacitySeated", "kapasitas duduk", &sup.CapacitySeated, optOf(seated))
	if !ch.any() {
		return sup, nil
	}
	if err := s.checkHeldEvents(ctx, projectID, *sup); err != nil {
		return nil, err
	}
	if _, err := s.repo.Patch(ctx, sup, ch.keys, nil); err != nil {
		return nil, err
	}
	s.activity.Record(ctx, &projectID, domain.ActivityVendorUpdated, actorStaffID, "project_supplier", formatID(sup.ID), sup.Name,
		"Kapasitas "+sup.Name+" diperbarui dari direktori")
	return s.Get(ctx, projectID, id)
}

// checkHeldEvents applies the capacity rule to the project's events held at
// venue (with the capacities it is about to have).
func (s *SupplierService) checkHeldEvents(ctx context.Context, projectID int64, venue domain.ProjectSupplier) error {
	if s.capacity == nil {
		return nil
	}
	all, err := s.events.ListByProject(ctx, projectID)
	if err != nil {
		return err
	}
	var held []domain.ProjectEvent
	for _, e := range all {
		if e.VenueSupplierID != nil && *e.VenueSupplierID == venue.ID {
			held = append(held, e)
		}
	}
	if len(held) == 0 {
		return nil
	}
	domain.SortEvents(held)
	return s.capacity.capacityRefusal("Kapasitas belum diperbarui. ", held, map[int64]domain.ProjectSupplier{venue.ID: venue})
}

func optOf[T any](v *T) Opt[T] {
	if v == nil {
		return Null[T]()
	}
	return Some(*v)
}

// Delete soft-deletes the supplier (ADR-0034): it leaves the list, the money,
// the progress and the dashboard, and so do its steps, payments, kendala and
// their files, until it is restored (24 hours) or purged. A venue an event
// still uses cannot be deleted: change the event's venue first.
func (s *SupplierService) Delete(ctx context.Context, projectID, id, actorStaffID int64, now time.Time) (*domain.ProjectSupplier, time.Time, error) {
	sup, err := s.Get(ctx, projectID, id)
	if err != nil {
		return nil, time.Time{}, err
	}
	if sup.Kind == domain.SupplierVenue {
		used, err := s.events.CountByVenueSupplier(ctx, projectID, id)
		if err != nil {
			return nil, time.Time{}, err
		}
		if used > 0 {
			msg := "Venue ini masih dipakai acara. Ganti venue di acaranya lebih dulu."
			return nil, time.Time{}, apperror.Validation(msg, map[string][]string{"id": {msg}}).WithCode("venue_in_use")
		}
	}
	ok, err := s.repo.SoftDelete(ctx, projectID, id, now)
	if err != nil {
		return nil, time.Time{}, err
	}
	if !ok {
		return nil, time.Time{}, errSupplierNotFound
	}
	s.activity.Record(ctx, &projectID, domain.ActivityVendorDeleted, actorStaffID, "project_supplier", formatID(sup.ID), sup.Name,
		sup.Name+" dihapus dari proyek")
	return sup, domain.RestorableUntil(now), nil
}

// Restore brings a deleted supplier back with everything that hung off it. Not
// deleted, or past the retention: 404.
func (s *SupplierService) Restore(ctx context.Context, projectID, id, actorStaffID int64, now time.Time) (*domain.ProjectSupplier, error) {
	sup, err := s.repo.FindDeletedByID(ctx, projectID, id)
	if err != nil {
		return nil, err
	}
	if sup == nil || sup.DeletedAt == nil || !domain.CanRestore(*sup.DeletedAt, now) {
		return nil, errNotRestorable()
	}
	ok, err := s.repo.Restore(ctx, projectID, id, domain.PurgeCutoff(now))
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errNotRestorable()
	}
	s.activity.Record(ctx, &projectID, domain.ActivityRestored, actorStaffID, "project_supplier", formatID(sup.ID), sup.Name,
		sup.Name+" dipulihkan")
	return s.Get(ctx, projectID, id)
}

// History is every project a directory vendor or venue works on, with where
// each one happens ("Lihat proyek" in the directory).
func (s *SupplierService) History(ctx context.Context, tenantID int64, kind domain.SupplierKind, supplierID int64) ([]domain.SupplierHistoryRow, error) {
	rows, err := s.repo.ListBySupplier(ctx, tenantID, kind, supplierID)
	if err != nil || len(rows) == 0 {
		return rows, err
	}
	ids := make([]int64, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r.ProjectID)
	}
	locations, err := s.Locations(ctx, ids)
	if err != nil {
		return nil, err
	}
	for i := range rows {
		rows[i].Location = locations[rows[i].ProjectID]
	}
	return rows, nil
}

// Locations is where each project happens (domain.LocationLine), by project id.
func (s *SupplierService) Locations(ctx context.Context, projectIDs []int64) (map[int64]string, error) {
	out := make(map[int64]string, len(projectIDs))
	if len(projectIDs) == 0 {
		return out, nil
	}
	events, err := s.events.ListByProjects(ctx, projectIDs)
	if err != nil {
		return nil, err
	}
	suppliers, err := s.repo.ListByProjects(ctx, projectIDs)
	if err != nil {
		return nil, err
	}
	names := make(map[int64]string, len(suppliers))
	for _, sup := range suppliers {
		if sup.Kind == domain.SupplierVenue {
			names[sup.ID] = sup.Name
		}
	}
	byProject := map[int64][]domain.ProjectEvent{}
	for _, e := range events {
		byProject[e.ProjectID] = append(byProject[e.ProjectID], e)
	}
	for id, list := range byProject {
		out[id] = domain.LocationLine(list, names)
	}
	return out, nil
}

// --- Steps of a supplier ("Tahapan") ---

type VendorMilestoneInput struct {
	Name        string
	Description string
	TargetDate  time.Time
	PICStaffID  int64
}

// MilestonesBySupplier is every step of the given suppliers, by supplier id.
func (s *SupplierService) MilestonesBySupplier(ctx context.Context, supplierIDs []int64) (map[int64][]domain.VendorMilestone, error) {
	all, err := s.milestones.ListBySuppliers(ctx, supplierIDs)
	if err != nil {
		return nil, err
	}
	out := make(map[int64][]domain.VendorMilestone, len(supplierIDs))
	for _, m := range all {
		out[m.ProjectSupplierID] = append(out[m.ProjectSupplierID], m)
	}
	return out, nil
}

func (s *SupplierService) CreateMilestone(ctx context.Context, projectID, supplierID, actorStaffID int64, input VendorMilestoneInput) (*domain.VendorMilestone, error) {
	if _, err := s.Get(ctx, projectID, supplierID); err != nil {
		return nil, err
	}
	order, err := s.milestones.NextSortOrder(ctx, supplierID)
	if err != nil {
		return nil, err
	}
	m := &domain.VendorMilestone{
		ProjectSupplierID: supplierID, SortOrder: order, Name: input.Name, Description: input.Description,
		Status: domain.MilestoneNotStarted, TargetDate: input.TargetDate, PICStaffID: input.PICStaffID,
	}
	if err := s.milestones.Create(ctx, m); err != nil {
		return nil, err
	}
	s.activity.Record(ctx, &projectID, domain.ActivityMilestoneUpdated, actorStaffID, "vendor_milestone", formatID(m.ID), m.Name,
		"Tahapan vendor ditambahkan: "+m.Name)
	return m, nil
}

// MilestoneExists is nil when the supplier is on this project and has this
// step, a not found otherwise.
func (s *SupplierService) MilestoneExists(ctx context.Context, projectID, supplierID, milestoneID int64) error {
	if _, err := s.Get(ctx, projectID, supplierID); err != nil {
		return err
	}
	m, err := s.milestones.FindByID(ctx, supplierID, milestoneID)
	if err != nil {
		return err
	}
	if m == nil {
		return apperror.NotFound("Tahapan vendor tidak ditemukan")
	}
	return nil
}

// FindMilestone is one step of a supplier on this project (nil when none).
func (s *SupplierService) FindMilestone(ctx context.Context, projectID, supplierID, milestoneID int64) (*domain.VendorMilestone, error) {
	if _, err := s.Get(ctx, projectID, supplierID); err != nil {
		return nil, err
	}
	return s.milestones.FindByID(ctx, supplierID, milestoneID)
}

// VendorMilestonePatch is what PATCH .../milestones/{mid} sent (ADR-0036).
// CompletedDate may be null (clear); Description and Notes null clear to "".
type VendorMilestonePatch struct {
	Status                    Opt[domain.MilestoneStatus]
	TargetDate, CompletedDate Opt[time.Time]
	PICStaffID                Opt[int64]
	Description, Notes        Opt[string]
}

// PatchMilestone changes the fields of a step that were sent, and only those. A
// status change without a completedDate stamps or clears the completed date as
// for a project step (domain.StampCompletion).
func (s *SupplierService) PatchMilestone(ctx context.Context, projectID, supplierID, milestoneID, actorStaffID int64, patch VendorMilestonePatch, expected *time.Time, today time.Time) (*domain.VendorMilestone, error) {
	if _, err := s.Get(ctx, projectID, supplierID); err != nil {
		return nil, err
	}
	missing := apperror.NotFound("Tahapan vendor tidak ditemukan")
	m, err := s.milestones.FindByID(ctx, supplierID, milestoneID)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, missing
	}
	if patch.Status.Present() && !domain.IsValidMilestoneStatus(patch.Status.Value()) {
		return nil, apperror.Validation("Status tahapan tidak valid", map[string][]string{"status": {"Status tahapan tidak valid"}})
	}
	loaded := *m
	var ch changeSet
	setVal(&ch, "status", "status", &m.Status, patch.Status)
	setDate(&ch, "targetDate", "tanggal target", &m.TargetDate, patch.TargetDate)
	setNullableDate(&ch, "completedDate", "tanggal selesai", &m.CompletedDate, patch.CompletedDate)
	setVal(&ch, "picStaffId", "PIC", &m.PICStaffID, patch.PICStaffID)
	if !patch.CompletedDate.Present() && domain.StampCompletion(loaded.Status, m.Status, &m.CompletedDate, today) {
		ch.note("completedDate", "tanggal selesai")
	}
	setVal(&ch, "description", "deskripsi", &m.Description, patch.Description)
	setVal(&ch, "notes", "catatan", &m.Notes, patch.Notes)
	if !ch.any() {
		return m, nil
	}
	if err := refuseIfBehind(&ch, expected, loaded.UpdatedAt, &loaded); err != nil {
		return nil, err
	}
	wrote, err := s.milestones.Patch(ctx, m, ch.keys, expected)
	if err != nil {
		return nil, err
	}
	fresh, err := finishPatch(ctx, expected, wrote, func(ctx context.Context) (*domain.VendorMilestone, error) {
		return s.milestones.FindByID(ctx, supplierID, milestoneID)
	}, func(m *domain.VendorMilestone) time.Time { return m.UpdatedAt }, missing)
	if err != nil {
		return nil, err
	}
	s.activity.Record(ctx, &projectID, domain.ActivityMilestoneUpdated, actorStaffID, "vendor_milestone", formatID(fresh.ID), fresh.Name,
		"Tahapan vendor diperbarui: "+ch.summary())
	return fresh, nil
}

// EvidenceOwners is what the portal's evidence rule needs about the project's
// suppliers, their steps and their payments (domain.EvidenceOwners).
func (s *SupplierService) EvidenceOwners(ctx context.Context, projectID int64) (domain.EvidenceOwners, error) {
	out := domain.EvidenceOwners{SupplierPaidBy: map[int64]domain.Payer{}, MilestonePaidBy: map[int64]domain.Payer{}, PaymentPayer: map[int64]domain.Payer{}}
	suppliers, err := s.repo.ListByProject(ctx, projectID)
	if err != nil {
		return out, err
	}
	ids := make([]int64, 0, len(suppliers))
	for _, sup := range suppliers {
		out.SupplierPaidBy[sup.ID] = sup.PaidBy
		ids = append(ids, sup.ID)
	}
	if len(ids) > 0 {
		steps, err := s.milestones.ListBySuppliers(ctx, ids)
		if err != nil {
			return out, err
		}
		for _, m := range steps {
			out.MilestonePaidBy[m.ID] = out.SupplierPaidBy[m.ProjectSupplierID]
		}
	}
	if s.payments != nil {
		payments, err := s.payments.ListByProject(ctx, projectID)
		if err != nil {
			return out, err
		}
		for _, p := range payments {
			out.PaymentPayer[p.ID] = p.Payer
		}
	}
	return out, nil
}
