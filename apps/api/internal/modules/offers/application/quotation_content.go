package application

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"elproof/internal/modules/offers/domain"
	"elproof/internal/shared/apperror"
	"elproof/internal/shared/etag"
)

// Messages of a quotation's events and suppliers.
const (
	MsgEventNotFound        = "Acara tidak ditemukan"
	MsgSupplierNotFound     = "Vendor atau venue tidak ditemukan"
	MsgVendorAlreadyAdded   = "Vendor ini sudah ada di penawaran"
	MsgVenueViaEvent        = "Venue ditambahkan lewat acaranya. Pilih venue di acara."
	MsgVenueInUse           = "Venue ini dipakai acara. Ganti venue di acaranya dulu."
	MsgSupplierUsedByLines  = "Baris rincian memakai vendor ini sebagai pemasok yang dibayar WO. Ubah barisnya dulu."
	MsgAgreedCostOnlyWO     = "Biaya rekanan hanya diisi untuk vendor atau venue yang dibayar WO"
	MsgAgreedCostRange      = "Biaya rekanan harus 0 atau lebih dan wajar"
	MsgTooManyEvents        = "Penawaran paling banyak punya 10 acara"
	MsgTooManySuppliers     = "Penawaran paling banyak punya 100 vendor dan venue"
	MsgEventOrderMismatch   = "Urutan acara harus memuat setiap acara penawaran tepat satu kali"
	MsgPackageNeedsEventFmt = "Paket %s mencakup %s. Sisakan minimal satu acara itu, atau hapus paketnya dulu dari rincian."
)

// QuotationSessionInput is one session of an event as sent.
type QuotationSessionInput struct {
	Label              string
	StartTime, EndTime *string
	Guests             int
}

// QuotationEventInput is an event as sent. VenueID is a directory venue (nil:
// none yet); EventDate may stay empty in a quotation.
type QuotationEventInput struct {
	EventKindID        int64
	EventDate          *time.Time
	StartTime, EndTime *string
	VenueID            *int64
	Layout             domain.Layout
	Sessions           []QuotationSessionInput
}

func validClock(v *string) bool {
	if v == nil {
		return true
	}
	_, err := time.Parse("15:04", *v)
	return err == nil && len(*v) == 5
}

func trimmedOrNil(v *string) *string {
	if v == nil || strings.TrimSpace(*v) == "" {
		return nil
	}
	t := strings.TrimSpace(*v)
	return &t
}

func clockRange(f *fieldErrs, prefix string, start, end *string) {
	if !validClock(start) {
		f.add(prefix+"startTime", "Jam mulai harus berformat JJ:MM")
	}
	if !validClock(end) {
		f.add(prefix+"endTime", "Jam selesai harus berformat JJ:MM")
	}
	if start != nil && end != nil && validClock(start) && validClock(end) && *end <= *start {
		f.add(prefix+"endTime", "Jam selesai harus setelah jam mulai")
	}
}

// venuePick is the venue an event takes place at: the quotation's venue
// supplier for that directory venue when there is one, otherwise the
// directory venue to add.
type venuePick struct {
	existing *domain.QuotationSupplier
	venueID  int64
	name     string
	standing *int
	seated   *int
}

func (p *venuePick) capacity() domain.VenueCapacity {
	return domain.VenueCapacity{VenueID: p.venueID, Name: p.name, Standing: p.standing, Seated: p.seated}
}

// prepareEvent validates an event and fills e (current is the event being
// edited, nil for a new one).
func (s *QuotationService) prepareEvent(ctx context.Context, tenantID int64, w *domain.WorkingCopy, in QuotationEventInput, current *domain.QuotationEvent, f *fieldErrs) (*domain.QuotationEvent, *venuePick, error) {
	e := &domain.QuotationEvent{QuotationID: w.Quotation.ID}
	if current != nil {
		c := *current
		e = &c
	}
	if in.EventKindID <= 0 {
		f.add("eventKindId", "Pilih jenis acara")
	} else if current == nil || current.EventKindID != in.EventKindID {
		kind, err := s.kinds.FindByID(ctx, tenantID, in.EventKindID)
		switch {
		case err != nil:
			return nil, nil, err
		case kind == nil:
			f.add("eventKindId", MsgEventKindUnknown)
		case !kind.IsActive:
			f.add("eventKindId", "Jenis acara ini sudah nonaktif. Pilih jenis lain.")
		default:
			e.EventKindID, e.KindName = kind.ID, kind.Name
		}
	}
	e.EventDate = nil
	if in.EventDate != nil && !in.EventDate.IsZero() {
		d := domain.CivilDate(*in.EventDate)
		e.EventDate = &d
	}
	e.StartTime, e.EndTime = trimmedOrNil(in.StartTime), trimmedOrNil(in.EndTime)
	clockRange(f, "", e.StartTime, e.EndTime)
	e.Layout = in.Layout
	if e.Layout == "" {
		e.Layout = domain.LayoutStanding
	}
	if !domain.ValidLayout(e.Layout) {
		f.add("layout", "Pilih tata letak berdiri atau duduk")
	}
	switch {
	case len(in.Sessions) == 0:
		f.add("sessions", "Tambahkan minimal satu sesi")
	case len(in.Sessions) > domain.MaxEventSessions:
		f.add("sessions", fmt.Sprintf("Maksimal %d sesi per acara", domain.MaxEventSessions))
	}
	e.Sessions = make([]domain.QuotationSession, 0, len(in.Sessions))
	for i, si := range in.Sessions {
		p := fmt.Sprintf("sessions.%d.", i)
		label := strings.TrimSpace(si.Label)
		if label == "" {
			label = fmt.Sprintf("Sesi %d", i+1)
		}
		if utf8.RuneCountInString(label) > 60 {
			f.add(p+"label", "Nama sesi paling banyak 60 karakter")
		}
		if si.Guests < 1 || si.Guests > domain.MaxSessionGuests {
			f.add(p+"guests", "Jumlah tamu minimal 1")
		}
		start, end := trimmedOrNil(si.StartTime), trimmedOrNil(si.EndTime)
		clockRange(f, p, start, end)
		e.Sessions = append(e.Sessions, domain.QuotationSession{Label: label, StartTime: start, EndTime: end, Guests: si.Guests, SortOrder: i})
	}

	if in.VenueID == nil || *in.VenueID == 0 {
		return e, nil, nil
	}
	venueID := *in.VenueID
	if venueID < 0 {
		f.add("venueId", MsgVenueNotFound)
		return e, nil, nil
	}
	for i := range w.Suppliers {
		sup := w.Suppliers[i]
		if sup.Kind == domain.SupplierVenue && sup.SupplierID == venueID {
			return e, &venuePick{existing: &sup, venueID: venueID, name: sup.Name, standing: sup.CapacityStanding, seated: sup.CapacitySeated}, nil
		}
	}
	ref, err := s.directory.GetVenueRef(ctx, tenantID, venueID)
	switch {
	case isNotFound(err):
		f.add("venueId", MsgVenueNotFound)
		return e, nil, nil
	case err != nil:
		return nil, nil, err
	case !ref.IsActive:
		f.add("venueId", "Venue ini sudah nonaktif. Pilih venue lain.")
		return e, nil, nil
	}
	return e, &venuePick{venueID: ref.ID, name: ref.Name, standing: ref.CapacityStanding, seated: ref.CapacitySeated}, nil
}

// checkEventCapacity applies the capacity rule to one event at its venue.
func checkEventCapacity(e *domain.QuotationEvent, venue *venuePick, f *fieldErrs) {
	if venue == nil {
		return
	}
	id := venue.venueID
	ec := domain.EventCapacity{KindName: e.KindName, Layout: e.Layout, VenueID: &id}
	if ec.KindName == "" {
		ec.KindName = "Acara"
	}
	for _, ss := range e.Sessions {
		ec.Sessions = append(ec.Sessions, domain.SessionGuests{Label: ss.Label, Guests: ss.Guests})
	}
	for _, p := range domain.CheckCapacity([]domain.EventCapacity{ec}, map[int64]domain.VenueCapacity{id: venue.capacity()}) {
		if p.Session < 0 {
			f.add("venueId", p.Message)
		} else {
			f.add(fmt.Sprintf("sessions.%d.guests", p.Session), p.Message)
		}
	}
}

// packageCoverage refuses events that would leave a package line without any
// event of a kind it covers (PLAN §3.1).
func (s *QuotationService) packageCoverage(ctx context.Context, tenantID int64, lines []domain.QuotationLine, events []domain.QuotationEvent, f *fieldErrs, field string) error {
	have := map[int64]bool{}
	for _, e := range events {
		have[e.EventKindID] = true
	}
	var names map[int64]string
	for _, l := range lines {
		if l.Kind != domain.LinePackage || l.PackageRule == nil {
			continue
		}
		covered := false
		for _, k := range l.PackageRule.EventKindIDs {
			if have[k] {
				covered = true
				break
			}
		}
		if covered {
			continue
		}
		if names == nil {
			kinds, err := s.kinds.List(ctx, tenantID)
			if err != nil {
				return err
			}
			names = map[int64]string{}
			for _, k := range kinds {
				names[k.ID] = k.Name
			}
		}
		var kindNames []string
		for _, k := range l.PackageRule.EventKindIDs {
			if n, ok := names[k]; ok {
				kindNames = append(kindNames, n)
			}
		}
		f.add(field, fmt.Sprintf(MsgPackageNeedsEventFmt, l.Description, strings.Join(kindNames, ", ")))
	}
	return nil
}

// addVenueSupplier is the venue supplier an event brings in: the existing one,
// or a new one paid by the WO with the venue's name and capacities copied.
func (s *QuotationService) addVenueSupplier(ctx context.Context, tenantID int64, w *domain.WorkingCopy, pick *venuePick) (int64, error) {
	if pick.existing != nil {
		return pick.existing.ID, nil
	}
	sup := &domain.QuotationSupplier{QuotationID: w.Quotation.ID, Kind: domain.SupplierVenue, SupplierID: pick.venueID, Name: pick.name,
		PaidBy: domain.PayerWO, CapacityStanding: pick.standing, CapacitySeated: pick.seated, SortOrder: len(w.Suppliers)}
	if err := s.repo.InsertSupplier(ctx, tenantID, sup); err != nil {
		return 0, err
	}
	w.Suppliers = append(w.Suppliers, *sup)
	return sup.ID, nil
}

// releaseVenue removes a venue supplier no event and no line uses any more
// (a quotation has no payments that would keep it).
func (s *QuotationService) releaseVenue(ctx context.Context, tenantID int64, w *domain.WorkingCopy, supplierID int64, exceptEvent int64) error {
	for _, e := range w.Events {
		if e.ID != exceptEvent && e.VenueSupplierID != nil && *e.VenueSupplierID == supplierID {
			return nil
		}
	}
	for _, l := range w.Lines {
		if l.SupplierRef != nil && *l.SupplierRef == supplierID {
			return nil
		}
	}
	return s.repo.DeleteSupplier(ctx, tenantID, w.Quotation.ID, supplierID)
}

func findEvent(w *domain.WorkingCopy, id int64) *domain.QuotationEvent {
	for i := range w.Events {
		if w.Events[i].ID == id {
			return &w.Events[i]
		}
	}
	return nil
}

func findSupplier(w *domain.WorkingCopy, id int64) *domain.QuotationSupplier {
	for i := range w.Suppliers {
		if w.Suppliers[i].ID == id {
			return &w.Suppliers[i]
		}
	}
	return nil
}

// editContent runs fn on the locked, editable quotation's working copy inside
// one transaction, then refreshes the computed lines and the list's caches.
func (s *QuotationService) editContent(ctx context.Context, tenantID, quotationID int64, v Viewer, fn func(ctx context.Context, w *domain.WorkingCopy) error) error {
	return s.tx.InTx(ctx, func(ctx context.Context) error {
		q, err := s.lock(ctx, tenantID, quotationID, v)
		if err != nil {
			return err
		}
		if err := editable(q); err != nil {
			return err
		}
		w, err := s.workingCopy(ctx, tenantID, q)
		if err != nil {
			return err
		}
		if err := fn(ctx, w); err != nil {
			return err
		}
		return s.refreshCaches(ctx, tenantID, q, false, false)
	})
}

// Content is the working copy of a quotation the viewer may see.
func (s *QuotationService) Content(ctx context.Context, tenantID, id int64, v Viewer) (*domain.WorkingCopy, error) {
	q, err := s.get(ctx, tenantID, id, v)
	if err != nil {
		return nil, err
	}
	return s.workingCopy(ctx, tenantID, q)
}

// CreateEvent adds an event; a venue picked from the directory comes in as a
// venue supplier paid by the WO in the same transaction.
func (s *QuotationService) CreateEvent(ctx context.Context, tenantID, quotationID int64, v Viewer, in QuotationEventInput) (*domain.QuotationEvent, error) {
	var id int64
	err := s.editContent(ctx, tenantID, quotationID, v, func(ctx context.Context, w *domain.WorkingCopy) error {
		f := newFieldErrs()
		if len(w.Events) >= domain.MaxQuotationEvents {
			f.add("eventKindId", MsgTooManyEvents)
		}
		e, venue, err := s.prepareEvent(ctx, tenantID, w, in, nil, f)
		if err != nil {
			return err
		}
		checkEventCapacity(e, venue, f)
		if err := f.err(); err != nil {
			return err
		}
		if venue != nil {
			supID, err := s.addVenueSupplier(ctx, tenantID, w, venue)
			if err != nil {
				return err
			}
			e.VenueSupplierID = &supID
		}
		e.SortOrder = len(w.Events)
		if err := s.repo.InsertEvent(ctx, tenantID, e); err != nil {
			return err
		}
		id = e.ID
		return s.activity(ctx, tenantID, quotationID, v, domain.ActivityEventsChanged, "Acara ditambahkan: "+e.KindName)
	})
	if err != nil {
		return nil, err
	}
	return s.event(ctx, tenantID, quotationID, id)
}

func (s *QuotationService) event(ctx context.Context, tenantID, quotationID, id int64) (*domain.QuotationEvent, error) {
	w, err := s.repo.LoadContent(ctx, tenantID, quotationID)
	if err != nil {
		return nil, err
	}
	if e := findEvent(w, id); e != nil {
		return e, nil
	}
	return nil, apperror.NotFound(MsgEventNotFound)
}

// UpdateEvent saves an event and its sessions (If-Match: the event's updatedAt).
func (s *QuotationService) UpdateEvent(ctx context.Context, tenantID, quotationID, eventID int64, v Viewer, in QuotationEventInput, expected *time.Time) (*domain.QuotationEvent, error) {
	err := s.editContent(ctx, tenantID, quotationID, v, func(ctx context.Context, w *domain.WorkingCopy) error {
		current := findEvent(w, eventID)
		if current == nil {
			return apperror.NotFound(MsgEventNotFound)
		}
		if expected != nil && !etag.Same(*expected, current.UpdatedAt) {
			return apperror.Stale(*current)
		}
		f := newFieldErrs()
		e, venue, err := s.prepareEvent(ctx, tenantID, w, in, current, f)
		if err != nil {
			return err
		}
		checkEventCapacity(e, venue, f)
		if e.EventKindID != current.EventKindID {
			after := make([]domain.QuotationEvent, 0, len(w.Events))
			for _, other := range w.Events {
				if other.ID == eventID {
					other.EventKindID = e.EventKindID
				}
				after = append(after, other)
			}
			if err := s.packageCoverage(ctx, tenantID, w.Lines, after, f, "eventKindId"); err != nil {
				return err
			}
		}
		if err := f.err(); err != nil {
			return err
		}
		old := current.VenueSupplierID
		e.VenueSupplierID = nil
		if venue != nil {
			supID, err := s.addVenueSupplier(ctx, tenantID, w, venue)
			if err != nil {
				return err
			}
			e.VenueSupplierID = &supID
		}
		if err := s.repo.UpdateEvent(ctx, tenantID, e); err != nil {
			return err
		}
		if old != nil && (e.VenueSupplierID == nil || *old != *e.VenueSupplierID) {
			if err := s.releaseVenue(ctx, tenantID, w, *old, eventID); err != nil {
				return err
			}
		}
		return s.activity(ctx, tenantID, quotationID, v, domain.ActivityEventsChanged, "Acara diubah: "+e.KindName)
	})
	if err != nil {
		return nil, err
	}
	return s.event(ctx, tenantID, quotationID, eventID)
}

// DeleteEvent removes an event; its venue supplier goes with it when nothing
// else uses it.
func (s *QuotationService) DeleteEvent(ctx context.Context, tenantID, quotationID, eventID int64, v Viewer) error {
	return s.editContent(ctx, tenantID, quotationID, v, func(ctx context.Context, w *domain.WorkingCopy) error {
		current := findEvent(w, eventID)
		if current == nil {
			return apperror.NotFound(MsgEventNotFound)
		}
		after := make([]domain.QuotationEvent, 0, len(w.Events))
		for _, other := range w.Events {
			if other.ID != eventID {
				after = append(after, other)
			}
		}
		f := newFieldErrs()
		if err := s.packageCoverage(ctx, tenantID, w.Lines, after, f, "event"); err != nil {
			return err
		}
		if err := f.err(); err != nil {
			return err
		}
		if err := s.repo.DeleteEvent(ctx, tenantID, quotationID, eventID); err != nil {
			return err
		}
		if current.VenueSupplierID != nil {
			if err := s.releaseVenue(ctx, tenantID, w, *current.VenueSupplierID, eventID); err != nil {
				return err
			}
		}
		return s.activity(ctx, tenantID, quotationID, v, domain.ActivityEventsChanged, "Acara dihapus: "+current.KindName)
	})
}

// SetEventOrder puts the events in the order given (every event exactly once).
func (s *QuotationService) SetEventOrder(ctx context.Context, tenantID, quotationID int64, v Viewer, ids []int64) error {
	return s.editContent(ctx, tenantID, quotationID, v, func(ctx context.Context, w *domain.WorkingCopy) error {
		seen := map[int64]bool{}
		for _, id := range ids {
			if findEvent(w, id) == nil || seen[id] {
				return apperror.Validation(MsgEventOrderMismatch, map[string][]string{"eventIds": {MsgEventOrderMismatch}})
			}
			seen[id] = true
		}
		if len(ids) != len(w.Events) {
			return apperror.Validation(MsgEventOrderMismatch, map[string][]string{"eventIds": {MsgEventOrderMismatch}})
		}
		return s.repo.SetEventOrder(ctx, tenantID, quotationID, ids)
	})
}

// ---- suppliers -----------------------------------------------------------

// QuotationSupplierInput is a vendor added to a quotation, or the editable
// part of a supplier (VendorID is ignored on update).
type QuotationSupplierInput struct {
	VendorID   int64
	PaidBy     domain.Payer
	ShowName   bool
	AgreedCost *int64
	Notes      string
}

func checkAgreedCost(f *fieldErrs, paidBy domain.Payer, cost *int64) {
	if cost == nil {
		return
	}
	if paidBy != domain.PayerWO {
		f.add("agreedCost", MsgAgreedCostOnlyWO)
		return
	}
	if *cost < 0 || *cost > domain.MaxMoney {
		f.add("agreedCost", MsgAgreedCostRange)
	}
}

func usedByLines(w *domain.WorkingCopy, supplierID int64) bool {
	for _, l := range w.Lines {
		if l.SupplierRef != nil && *l.SupplierRef == supplierID {
			return true
		}
	}
	return false
}

// CreateSupplier adds a directory vendor (venues come in through their events).
func (s *QuotationService) CreateSupplier(ctx context.Context, tenantID, quotationID int64, v Viewer, in QuotationSupplierInput) (*domain.QuotationSupplier, error) {
	var id int64
	err := s.editContent(ctx, tenantID, quotationID, v, func(ctx context.Context, w *domain.WorkingCopy) error {
		f := newFieldErrs()
		if len(w.Suppliers) >= domain.MaxQuotationSuppliers {
			f.add("vendorId", MsgTooManySuppliers)
		}
		if in.PaidBy == "" {
			in.PaidBy = domain.PayerWO
		}
		if !domain.ValidPayer(in.PaidBy) {
			f.add("paidBy", "Pilih siapa yang membayar")
		}
		if !v.CanSeeCost {
			in.AgreedCost = nil
		}
		checkAgreedCost(f, in.PaidBy, in.AgreedCost)
		in.Notes = strings.TrimSpace(in.Notes)
		if utf8.RuneCountInString(in.Notes) > domain.MaxNotesLen {
			f.add("notes", fmt.Sprintf("Catatan maksimal %d karakter", domain.MaxNotesLen))
		}
		var ref struct {
			name     string
			category int64
		}
		if in.VendorID <= 0 {
			f.add("vendorId", "Pilih vendor")
		} else {
			vr, err := s.directory.GetVendorRef(ctx, tenantID, in.VendorID)
			switch {
			case isNotFound(err):
				f.add("vendorId", MsgVendorNotFound)
			case err != nil:
				return err
			case !vr.IsActive:
				f.add("vendorId", MsgVendorInactive)
			default:
				ref.name, ref.category = vr.Name, vr.CategoryID
				for _, sup := range w.Suppliers {
					if sup.Kind == domain.SupplierVendor && sup.SupplierID == vr.ID {
						f.add("vendorId", MsgVendorAlreadyAdded)
					}
				}
			}
		}
		if err := f.err(); err != nil {
			return err
		}
		sup := &domain.QuotationSupplier{QuotationID: quotationID, Kind: domain.SupplierVendor, SupplierID: in.VendorID, Name: ref.name,
			PaidBy: in.PaidBy, ShowName: in.ShowName && in.PaidBy == domain.PayerWO, AgreedCost: in.AgreedCost, Notes: in.Notes,
			SortOrder: len(w.Suppliers)}
		if ref.category > 0 {
			c := ref.category
			sup.CategoryID = &c
		}
		if err := s.repo.InsertSupplier(ctx, tenantID, sup); err != nil {
			return err
		}
		id = sup.ID
		return s.activity(ctx, tenantID, quotationID, v, domain.ActivitySuppliersChanged, "Vendor ditambahkan: "+sup.Name)
	})
	if err != nil {
		return nil, err
	}
	return s.supplier(ctx, tenantID, quotationID, id)
}

func (s *QuotationService) supplier(ctx context.Context, tenantID, quotationID, id int64) (*domain.QuotationSupplier, error) {
	w, err := s.repo.LoadContent(ctx, tenantID, quotationID)
	if err != nil {
		return nil, err
	}
	if sup := findSupplier(w, id); sup != nil {
		return sup, nil
	}
	return nil, apperror.NotFound(MsgSupplierNotFound)
}

// UpdateSupplier saves who pays a supplier, whether its name shows, the
// agreed cost (finance.margin.view only), and notes.
func (s *QuotationService) UpdateSupplier(ctx context.Context, tenantID, quotationID, supplierID int64, v Viewer, in QuotationSupplierInput, expected *time.Time) (*domain.QuotationSupplier, error) {
	err := s.editContent(ctx, tenantID, quotationID, v, func(ctx context.Context, w *domain.WorkingCopy) error {
		current := findSupplier(w, supplierID)
		if current == nil {
			return apperror.NotFound(MsgSupplierNotFound)
		}
		if expected != nil && !etag.Same(*expected, current.UpdatedAt) {
			return apperror.Stale(*current)
		}
		sup := *current
		f := newFieldErrs()
		if in.PaidBy == "" {
			in.PaidBy = sup.PaidBy
		}
		if !domain.ValidPayer(in.PaidBy) {
			f.add("paidBy", "Pilih siapa yang membayar")
		}
		if in.PaidBy == domain.PayerClient && usedByLines(w, supplierID) {
			f.add("paidBy", MsgSupplierUsedByLines)
		}
		if v.CanSeeCost {
			sup.AgreedCost = in.AgreedCost
		} else if in.PaidBy != domain.PayerWO {
			sup.AgreedCost = nil
		}
		checkAgreedCost(f, in.PaidBy, sup.AgreedCost)
		in.Notes = strings.TrimSpace(in.Notes)
		if utf8.RuneCountInString(in.Notes) > domain.MaxNotesLen {
			f.add("notes", fmt.Sprintf("Catatan maksimal %d karakter", domain.MaxNotesLen))
		}
		if err := f.err(); err != nil {
			return err
		}
		sup.PaidBy, sup.Notes = in.PaidBy, in.Notes
		sup.ShowName = in.ShowName && sup.Kind == domain.SupplierVendor && sup.PaidBy == domain.PayerWO
		if err := s.repo.UpdateSupplier(ctx, tenantID, &sup); err != nil {
			return err
		}
		return s.activity(ctx, tenantID, quotationID, v, domain.ActivitySuppliersChanged, "Vendor & venue diubah: "+sup.Name)
	})
	if err != nil {
		return nil, err
	}
	return s.supplier(ctx, tenantID, quotationID, supplierID)
}

// RefreshSupplier is "Perbarui dari direktori": the name (and a venue's
// capacities, checked against its events; a vendor's category) copied again.
func (s *QuotationService) RefreshSupplier(ctx context.Context, tenantID, quotationID, supplierID int64, v Viewer) (*domain.QuotationSupplier, error) {
	err := s.editContent(ctx, tenantID, quotationID, v, func(ctx context.Context, w *domain.WorkingCopy) error {
		current := findSupplier(w, supplierID)
		if current == nil {
			return apperror.NotFound(MsgSupplierNotFound)
		}
		sup := *current
		if sup.Kind == domain.SupplierVenue {
			ref, err := s.directory.GetVenueRef(ctx, tenantID, sup.SupplierID)
			if isNotFound(err) {
				return apperror.Validation(MsgVenueNotFound, map[string][]string{"supplier": {MsgVenueNotFound}})
			}
			if err != nil {
				return err
			}
			sup.Name, sup.CapacityStanding, sup.CapacitySeated = ref.Name, ref.CapacityStanding, ref.CapacitySeated
			f := newFieldErrs()
			pick := &venuePick{existing: &sup, venueID: sup.SupplierID, name: sup.Name, standing: sup.CapacityStanding, seated: sup.CapacitySeated}
			for i := range w.Events {
				e := w.Events[i]
				if e.VenueSupplierID != nil && *e.VenueSupplierID == supplierID {
					checkEventCapacity(&e, pick, f)
				}
			}
			if err := f.err(); err != nil {
				return err
			}
		} else {
			ref, err := s.directory.GetVendorRef(ctx, tenantID, sup.SupplierID)
			if isNotFound(err) {
				return apperror.Validation(MsgVendorNotFound, map[string][]string{"supplier": {MsgVendorNotFound}})
			}
			if err != nil {
				return err
			}
			sup.Name = ref.Name
			if ref.CategoryID > 0 {
				c := ref.CategoryID
				sup.CategoryID = &c
			}
		}
		if err := s.repo.UpdateSupplier(ctx, tenantID, &sup); err != nil {
			return err
		}
		return s.activity(ctx, tenantID, quotationID, v, domain.ActivitySuppliersChanged, "Diperbarui dari direktori: "+sup.Name)
	})
	if err != nil {
		return nil, err
	}
	return s.supplier(ctx, tenantID, quotationID, supplierID)
}

// DeleteSupplier removes a vendor. A venue leaves with its events.
func (s *QuotationService) DeleteSupplier(ctx context.Context, tenantID, quotationID, supplierID int64, v Viewer) error {
	return s.editContent(ctx, tenantID, quotationID, v, func(ctx context.Context, w *domain.WorkingCopy) error {
		current := findSupplier(w, supplierID)
		if current == nil {
			return apperror.NotFound(MsgSupplierNotFound)
		}
		if current.Kind == domain.SupplierVenue {
			for _, e := range w.Events {
				if e.VenueSupplierID != nil && *e.VenueSupplierID == supplierID {
					return apperror.Validation(MsgVenueInUse, map[string][]string{"supplier": {MsgVenueInUse}})
				}
			}
		}
		if err := s.repo.DeleteSupplier(ctx, tenantID, quotationID, supplierID); err != nil {
			return err
		}
		return s.activity(ctx, tenantID, quotationID, v, domain.ActivitySuppliersChanged, "Vendor & venue dihapus: "+current.Name)
	})
}
