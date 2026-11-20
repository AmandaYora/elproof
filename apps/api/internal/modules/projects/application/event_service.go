package application

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	offerscontracts "elproof/internal/modules/offers/contracts"
	"elproof/internal/modules/projects/domain"
	vendorscontracts "elproof/internal/modules/vendors/contracts"
	"elproof/internal/shared/apperror"
	"elproof/internal/shared/etag"
)

// EventRepository stores a project's events and their sessions. Every read
// returns the sessions with the event.
type EventRepository interface {
	ListByProject(ctx context.Context, projectID int64) ([]domain.ProjectEvent, error)
	ListByProjects(ctx context.Context, projectIDs []int64) ([]domain.ProjectEvent, error)
	FindByID(ctx context.Context, projectID, id int64) (*domain.ProjectEvent, error)
	// Create inserts the event and its sessions (tenant from the project).
	Create(ctx context.Context, e *domain.ProjectEvent) error
	// Replace writes every field of the event and replaces its sessions; with
	// expected it is a compare-and-set on updated_at (false: nothing written).
	Replace(ctx context.Context, e *domain.ProjectEvent, expected *time.Time) (bool, error)
	Delete(ctx context.Context, projectID, id int64) error
	// SetMain makes id the project's one main event.
	SetMain(ctx context.Context, projectID, id int64) error
	SetSortOrder(ctx context.Context, projectID, id int64, order int) error
	// CountByVenueSupplier is how many events of the project take place at a
	// venue supplier.
	CountByVenueSupplier(ctx context.Context, projectID, supplierID int64) (int, error)
	NextSortOrder(ctx context.Context, projectID int64) (int, error)
}

// EventCatalog is what events need from `offers`: the workspace's event kinds
// and the one capacity rule.
type EventCatalog interface {
	GetEventKind(ctx context.Context, tenantID, id int64) (offerscontracts.EventKindRef, error)
	CheckCapacity(events []offerscontracts.CapacityEvent, venues map[int64]offerscontracts.CapacityVenue) []offerscontracts.CapacityProblem
}

// EventVenues is what events need from `vendors`: a directory venue with its
// capacities.
type EventVenues interface {
	GetVenueRef(ctx context.Context, tenantID, venueID int64) (vendorscontracts.VenueRef, error)
}

// EventService runs the events of a project ("Acara & rincian"): their kind,
// date, hours, venue, layout and sessions, which one is the main event, and
// their order. The main event's date is the project's event date: every write
// that moves it moves the project's date and the steps that follow it, in the
// same transaction.
type EventService struct {
	repo      EventRepository
	suppliers *SupplierService
	projects  *ProjectService
	catalog   EventCatalog
	venues    EventVenues
	tx        Transactor
	activity  *ActivityService
	// lines recomputes the extra-guest charges and the contract cache in the
	// same transaction as every change of guests or events (SetLines).
	lines *LineService
}

// SetLines wires the Rincian (built after the events).
func (s *EventService) SetLines(l *LineService) { s.lines = l }

func NewEventService(repo EventRepository, suppliers *SupplierService, projects *ProjectService, tx Transactor, activity *ActivityService) *EventService {
	if tx == nil {
		tx = noTransactor{}
	}
	return &EventService{repo: repo, suppliers: suppliers, projects: projects, tx: tx, activity: activity}
}

// SetCatalog and SetVenues complete the wiring once `offers` and `vendors`
// exist (they are built after `projects`).
func (s *EventService) SetCatalog(c EventCatalog) { s.catalog = c }
func (s *EventService) SetVenues(v EventVenues)   { s.venues = v }

var errEventNotFound = apperror.NotFound("Acara tidak ditemukan")

const (
	maxEventSessions = 10
	maxSessionGuests = 100_000
)

// SessionInput is one session of an event as sent.
type SessionInput struct {
	Label              string
	StartTime, EndTime *string
	Guests             int
}

// EventInput is an event as sent. VenueID is a directory venue (nil: none yet).
type EventInput struct {
	EventKindID        int64
	EventDate          time.Time
	StartTime, EndTime *string
	VenueID            *int64
	Layout             domain.Layout
	Sessions           []SessionInput
}

// venueChoice is the venue an event takes place at: the project's live venue
// supplier for it when there is one, otherwise the directory venue to add.
type venueChoice struct {
	ref      vendorscontracts.VenueRef
	existing *domain.ProjectSupplier
}

func (v *venueChoice) capacity() offerscontracts.CapacityVenue {
	if v.existing != nil {
		return offerscontracts.CapacityVenue{Name: v.existing.Name, Standing: v.existing.CapacityStanding, Seated: v.existing.CapacitySeated}
	}
	return offerscontracts.CapacityVenue{Name: v.ref.Name, Standing: v.ref.CapacityStanding, Seated: v.ref.CapacitySeated}
}

// preparedEvent is a validated event, ready to write once its venue supplier
// is known.
type preparedEvent struct {
	event domain.ProjectEvent
	venue *venueChoice
}

func validClock(v *string) bool {
	if v == nil {
		return true
	}
	_, err := time.Parse("15:04", *v)
	return err == nil && len(*v) == 5
}

func clockRange(fe *fieldErrors, prefix string, start, end *string) {
	if !validClock(start) {
		fe.add("Jam mulai harus berformat JJ:MM", prefix+"startTime")
	}
	if !validClock(end) {
		fe.add("Jam selesai harus berformat JJ:MM", prefix+"endTime")
	}
	if start != nil && end != nil && validClock(start) && validClock(end) && *end <= *start {
		fe.add("Jam selesai harus setelah jam mulai", prefix+"endTime")
	}
}

func emptyToNil(v *string) *string {
	if v == nil || strings.TrimSpace(*v) == "" {
		return nil
	}
	t := strings.TrimSpace(*v)
	return &t
}

// prepare validates one event (current is the event being edited, nil for a
// new one; projectID is 0 while the project itself is being created). Problems
// are added to fe under prefix ("" or "events.2.").
func (s *EventService) prepare(ctx context.Context, tenantID, projectID int64, in EventInput, current *domain.ProjectEvent, prefix string, fe *fieldErrors) (*preparedEvent, error) {
	if s.catalog == nil || s.venues == nil {
		return nil, apperror.Internal("Katalog acara belum tersedia")
	}
	e := domain.ProjectEvent{ProjectID: projectID, EventDate: in.EventDate, Layout: in.Layout}
	if current != nil {
		e = *current
		e.EventDate, e.Layout = in.EventDate, in.Layout
	}
	if in.EventKindID <= 0 {
		fe.add("Pilih jenis acara", prefix+"eventKindId")
	} else if current == nil || current.EventKindID != in.EventKindID {
		kind, err := s.catalog.GetEventKind(ctx, tenantID, in.EventKindID)
		switch {
		case isNotFound(err):
			fe.add("Jenis acara tidak ditemukan", prefix+"eventKindId")
		case err != nil:
			return nil, err
		case !kind.IsActive:
			fe.add("Jenis acara ini sudah nonaktif. Pilih jenis lain.", prefix+"eventKindId")
		default:
			e.EventKindID, e.KindName = kind.ID, kind.Name
		}
	}
	if in.EventDate.IsZero() {
		fe.add("Tanggal acara wajib diisi", prefix+"eventDate")
	}
	e.StartTime, e.EndTime = emptyToNil(in.StartTime), emptyToNil(in.EndTime)
	clockRange(fe, prefix, e.StartTime, e.EndTime)
	if e.Layout == "" {
		e.Layout = domain.LayoutStanding
	}
	if !domain.IsValidLayout(e.Layout) {
		fe.add("Pilih tata letak berdiri atau duduk", prefix+"layout")
	}

	switch {
	case len(in.Sessions) == 0:
		fe.add("Tambahkan minimal satu sesi", prefix+"sessions")
	case len(in.Sessions) > maxEventSessions:
		fe.add(fmt.Sprintf("Maksimal %d sesi per acara", maxEventSessions), prefix+"sessions")
	}
	e.Sessions = make([]domain.EventSession, 0, len(in.Sessions))
	for i, si := range in.Sessions {
		sp := fmt.Sprintf("%ssessions.%d.", prefix, i)
		label := strings.TrimSpace(si.Label)
		if label == "" {
			label = fmt.Sprintf("Sesi %d", i+1)
		}
		if utf8.RuneCountInString(label) > 60 {
			fe.add("Nama sesi paling banyak 60 karakter", sp+"label")
		}
		if si.Guests < 1 || si.Guests > maxSessionGuests {
			fe.add("Jumlah tamu minimal 1", sp+"guests")
		}
		start, end := emptyToNil(si.StartTime), emptyToNil(si.EndTime)
		clockRange(fe, sp, start, end)
		e.Sessions = append(e.Sessions, domain.EventSession{Label: label, StartTime: start, EndTime: end, Guests: si.Guests, SortOrder: i})
	}

	out := &preparedEvent{event: e}
	if in.VenueID == nil || *in.VenueID == 0 {
		return out, nil
	}
	venueID := *in.VenueID
	if venueID < 0 {
		fe.add("Venue tidak ditemukan", prefix+"venueId")
		return out, nil
	}
	if projectID > 0 {
		existing, err := s.suppliers.repo.FindLiveVenue(ctx, projectID, venueID)
		if err != nil {
			return nil, err
		}
		if existing != nil {
			out.venue = &venueChoice{existing: existing, ref: vendorscontracts.VenueRef{ID: venueID, Name: existing.Name}}
			return out, nil
		}
	}
	ref, err := s.venues.GetVenueRef(ctx, tenantID, venueID)
	switch {
	case isNotFound(err):
		fe.add("Venue tidak ditemukan", prefix+"venueId")
		return out, nil
	case err != nil:
		return nil, err
	case !ref.IsActive:
		fe.add("Venue ini sudah nonaktif. Pilih venue lain.", prefix+"venueId")
		return out, nil
	}
	out.venue = &venueChoice{ref: ref}
	return out, nil
}

func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	ae, ok := apperror.As(err)
	return ok && ae.Kind == apperror.KindNotFound
}

// checkCapacity applies the capacity rule to prepared events; prefixOf gives
// each event's error key prefix.
func (s *EventService) checkCapacity(events []*preparedEvent, prefixOf func(i int) string, fe *fieldErrors) {
	in := make([]offerscontracts.CapacityEvent, len(events))
	venues := map[int64]offerscontracts.CapacityVenue{}
	for i, pe := range events {
		ce := offerscontracts.CapacityEvent{KindName: pe.event.KindName, Layout: string(pe.event.Layout)}
		if ce.KindName == "" {
			ce.KindName = "Acara"
		}
		if pe.venue != nil {
			id := pe.venue.ref.ID
			ce.VenueID = &id
			venues[id] = pe.venue.capacity()
		}
		for _, ss := range pe.event.Sessions {
			ce.Sessions = append(ce.Sessions, offerscontracts.CapacitySession{Label: ss.Label, Guests: ss.Guests})
		}
		in[i] = ce
	}
	for _, p := range s.catalog.CheckCapacity(in, venues) {
		if p.Session < 0 {
			fe.add(p.Message, prefixOf(p.Event)+"venueId")
		} else {
			fe.add(p.Message, fmt.Sprintf("%ssessions.%d.guests", prefixOf(p.Event), p.Session))
		}
	}
}

// PrepareForProject validates the events sent with a new project (keys
// "events.%d.*"). At least one is required; the first is the main event.
func (s *EventService) PrepareForProject(ctx context.Context, tenantID int64, inputs []EventInput, fe *fieldErrors) ([]*preparedEvent, error) {
	if len(inputs) == 0 {
		fe.add("Tambahkan minimal satu acara", "events")
		return nil, nil
	}
	out := make([]*preparedEvent, len(inputs))
	prefixOf := func(i int) string { return fmt.Sprintf("events.%d.", i) }
	for i, in := range inputs {
		pe, err := s.prepare(ctx, tenantID, 0, in, nil, prefixOf(i), fe)
		if err != nil {
			return nil, err
		}
		out[i] = pe
	}
	s.checkCapacity(out, prefixOf, fe)
	return out, nil
}

// createForProject writes the prepared events of a new project, inside the
// caller's transaction. The first is the main event; events at the same
// directory venue share one venue supplier.
func (s *EventService) createForProject(ctx context.Context, projectID, actorStaffID int64, events []*preparedEvent) error {
	byVenue := map[int64]*domain.ProjectSupplier{}
	for i, pe := range events {
		e := pe.event
		e.ProjectID, e.IsMain, e.SortOrder = projectID, i == 0, i
		if pe.venue != nil {
			sup := byVenue[pe.venue.ref.ID]
			if sup == nil {
				var err error
				if sup, err = s.suppliers.addVenue(ctx, projectID, actorStaffID, *pe.venue); err != nil {
					return err
				}
				byVenue[pe.venue.ref.ID] = sup
			}
			id := sup.ID
			e.VenueSupplierID = &id
		}
		if err := s.repo.Create(ctx, &e); err != nil {
			return err
		}
	}
	return nil
}

// List is the project's events in screen order.
func (s *EventService) List(ctx context.Context, tenantID, projectID int64) ([]domain.ProjectEvent, error) {
	if _, err := s.projects.Get(ctx, tenantID, projectID); err != nil {
		return nil, err
	}
	events, err := s.repo.ListByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	domain.SortEvents(events)
	return events, nil
}

// ListByProjects is the events of several projects, by project id, in screen order.
func (s *EventService) ListByProjects(ctx context.Context, projectIDs []int64) (map[int64][]domain.ProjectEvent, error) {
	out := make(map[int64][]domain.ProjectEvent, len(projectIDs))
	if len(projectIDs) == 0 {
		return out, nil
	}
	all, err := s.repo.ListByProjects(ctx, projectIDs)
	if err != nil {
		return nil, err
	}
	for _, e := range all {
		out[e.ProjectID] = append(out[e.ProjectID], e)
	}
	for id := range out {
		domain.SortEvents(out[id])
	}
	return out, nil
}

func (s *EventService) get(ctx context.Context, projectID, id int64) (*domain.ProjectEvent, error) {
	e, err := s.repo.FindByID(ctx, projectID, id)
	if err != nil {
		return nil, err
	}
	if e == nil {
		return nil, errEventNotFound
	}
	return e, nil
}

// Get is one event of the project.
func (s *EventService) Get(ctx context.Context, tenantID, projectID, id int64) (*domain.ProjectEvent, error) {
	if _, err := s.projects.Get(ctx, tenantID, projectID); err != nil {
		return nil, err
	}
	return s.get(ctx, projectID, id)
}

// resolveVenue turns the prepared venue into the event's venue supplier,
// inside the caller's transaction.
func (s *EventService) resolveVenue(ctx context.Context, projectID, actorStaffID int64, pe *preparedEvent) error {
	pe.event.VenueSupplierID = nil
	if pe.venue == nil {
		return nil
	}
	sup, err := s.suppliers.addVenue(ctx, projectID, actorStaffID, *pe.venue)
	if err != nil {
		return err
	}
	id := sup.ID
	pe.event.VenueSupplierID = &id
	return nil
}

func (s *EventService) inProjectTx(ctx context.Context, projectID int64, fn func(ctx context.Context) error) error {
	return s.tx.InTx(ctx, func(ctx context.Context) error {
		if err := s.tx.LockProject(ctx, projectID); err != nil {
			return err
		}
		return fn(ctx)
	})
}

// Create adds an event to a project. A project's first event is its main event.
func (s *EventService) Create(ctx context.Context, tenantID, projectID, actorStaffID int64, in EventInput) (*domain.ProjectEvent, error) {
	if _, err := s.projects.Get(ctx, tenantID, projectID); err != nil {
		return nil, err
	}
	fe := newFieldErrors()
	pe, err := s.prepare(ctx, tenantID, projectID, in, nil, "", fe)
	if err != nil {
		return nil, err
	}
	s.checkCapacity([]*preparedEvent{pe}, func(int) string { return "" }, fe)
	if err := fe.err(); err != nil {
		return nil, err
	}
	var id int64
	err = s.inProjectTx(ctx, projectID, func(ctx context.Context) error {
		existing, err := s.repo.ListByProject(ctx, projectID)
		if err != nil {
			return err
		}
		if err := s.resolveVenue(ctx, projectID, actorStaffID, pe); err != nil {
			return err
		}
		e := pe.event
		e.ProjectID, e.IsMain = projectID, len(existing) == 0
		if e.SortOrder, err = s.repo.NextSortOrder(ctx, projectID); err != nil {
			return err
		}
		if err := s.repo.Create(ctx, &e); err != nil {
			return err
		}
		id = e.ID
		s.activity.Record(ctx, &projectID, domain.ActivityEventAdded, actorStaffID, "project_event", formatID(e.ID), e.KindName,
			"Acara ditambahkan: "+e.KindName+" "+e.EventDate.Format("02-01-2006")+guestNote(e))
		if err := s.lines.AfterEventsChanged(ctx, tenantID, projectID, actorStaffID); err != nil {
			return err
		}
		if e.IsMain {
			return s.projects.syncEventDate(ctx, tenantID, projectID, actorStaffID, e.EventDate)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.get(ctx, projectID, id)
}

// Update replaces an event with what was sent (PUT: every field; ADR-0036
// If-Match). A venue the event stops using is released (removed when nothing
// else uses it or hangs off it). Moving the main event moves the project's date.
func (s *EventService) Update(ctx context.Context, tenantID, projectID, id, actorStaffID int64, in EventInput, expected *time.Time) (*domain.ProjectEvent, error) {
	if _, err := s.projects.Get(ctx, tenantID, projectID); err != nil {
		return nil, err
	}
	current, err := s.get(ctx, projectID, id)
	if err != nil {
		return nil, err
	}
	if expected != nil && !etag.Same(current.UpdatedAt, *expected) {
		return nil, apperror.Stale(current)
	}
	fe := newFieldErrors()
	pe, err := s.prepare(ctx, tenantID, projectID, in, current, "", fe)
	if err != nil {
		return nil, err
	}
	s.checkCapacity([]*preparedEvent{pe}, func(int) string { return "" }, fe)
	if err := fe.err(); err != nil {
		return nil, err
	}
	err = s.inProjectTx(ctx, projectID, func(ctx context.Context) error {
		if err := s.resolveVenue(ctx, projectID, actorStaffID, pe); err != nil {
			return err
		}
		e := pe.event
		wrote, err := s.repo.Replace(ctx, &e, expected)
		if err != nil {
			return err
		}
		if !wrote {
			fresh, err := s.get(ctx, projectID, id)
			if err != nil {
				return err
			}
			return apperror.Stale(fresh)
		}
		if old := current.VenueSupplierID; old != nil && (e.VenueSupplierID == nil || *e.VenueSupplierID != *old) {
			if err := s.suppliers.releaseVenue(ctx, projectID, *old); err != nil {
				return err
			}
		}
		s.activity.Record(ctx, &projectID, domain.ActivityEventUpdated, actorStaffID, "project_event", formatID(e.ID), e.KindName,
			"Acara diperbarui: "+e.KindName+" "+e.EventDate.Format("02-01-2006")+guestChanges(*current, e))
		if err := s.lines.AfterEventsChanged(ctx, tenantID, projectID, actorStaffID); err != nil {
			return err
		}
		if e.IsMain && !e.EventDate.Equal(current.EventDate) {
			return s.projects.syncEventDate(ctx, tenantID, projectID, actorStaffID, e.EventDate)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.get(ctx, projectID, id)
}

// Delete removes an event (there is no restore: it can be added again). A
// project keeps at least one event; deleting the main event makes the earliest
// remaining one main.
func (s *EventService) Delete(ctx context.Context, tenantID, projectID, id, actorStaffID int64) error {
	if _, err := s.projects.Get(ctx, tenantID, projectID); err != nil {
		return err
	}
	return s.inProjectTx(ctx, projectID, func(ctx context.Context) error {
		events, err := s.repo.ListByProject(ctx, projectID)
		if err != nil {
			return err
		}
		var target *domain.ProjectEvent
		var rest []domain.ProjectEvent
		for i := range events {
			if events[i].ID == id {
				target = &events[i]
			} else {
				rest = append(rest, events[i])
			}
		}
		if target == nil {
			return errEventNotFound
		}
		if len(rest) == 0 {
			msg := "Proyek perlu minimal satu acara. Tambahkan acara lain sebelum menghapus yang ini."
			return apperror.Validation(msg, map[string][]string{"id": {msg}}).WithCode("last_event")
		}
		if err := s.repo.Delete(ctx, projectID, id); err != nil {
			return err
		}
		if target.VenueSupplierID != nil {
			if err := s.suppliers.releaseVenue(ctx, projectID, *target.VenueSupplierID); err != nil {
				return err
			}
		}
		s.activity.Record(ctx, &projectID, domain.ActivityEventDeleted, actorStaffID, "project_event", formatID(id), target.KindName,
			"Acara dihapus: "+target.KindName+" "+target.EventDate.Format("02-01-2006")+guestNote(*target))
		if err := s.lines.AfterEventsChanged(ctx, tenantID, projectID, actorStaffID); err != nil {
			return err
		}
		if !target.IsMain {
			return nil
		}
		domain.SortEvents(rest)
		next := rest[0]
		if err := s.repo.SetMain(ctx, projectID, next.ID); err != nil {
			return err
		}
		return s.projects.syncEventDate(ctx, tenantID, projectID, actorStaffID, next.EventDate)
	})
}

// SetMain makes an event the project's main event; the project's date (and the
// steps that follow it) moves to its date.
func (s *EventService) SetMain(ctx context.Context, tenantID, projectID, id, actorStaffID int64) (*domain.ProjectEvent, error) {
	if _, err := s.projects.Get(ctx, tenantID, projectID); err != nil {
		return nil, err
	}
	err := s.inProjectTx(ctx, projectID, func(ctx context.Context) error {
		e, err := s.get(ctx, projectID, id)
		if err != nil {
			return err
		}
		if e.IsMain {
			return nil
		}
		if err := s.repo.SetMain(ctx, projectID, id); err != nil {
			return err
		}
		s.activity.Record(ctx, &projectID, domain.ActivityEventUpdated, actorStaffID, "project_event", formatID(e.ID), e.KindName,
			e.KindName+" dijadikan acara utama")
		return s.projects.syncEventDate(ctx, tenantID, projectID, actorStaffID, e.EventDate)
	})
	if err != nil {
		return nil, err
	}
	return s.get(ctx, projectID, id)
}

// Move swaps an event with its neighbour above ("up") or below ("down"). Events
// are ordered by date first, so only a neighbour on the same date can swap.
func (s *EventService) Move(ctx context.Context, tenantID, projectID, id, actorStaffID int64, direction string) ([]domain.ProjectEvent, error) {
	if direction != "up" && direction != "down" {
		return nil, apperror.Validation("Arah tidak dikenal", map[string][]string{"direction": {"Gunakan up atau down"}})
	}
	if _, err := s.projects.Get(ctx, tenantID, projectID); err != nil {
		return nil, err
	}
	err := s.inProjectTx(ctx, projectID, func(ctx context.Context) error {
		events, err := s.repo.ListByProject(ctx, projectID)
		if err != nil {
			return err
		}
		domain.SortEvents(events)
		at := -1
		for i := range events {
			if events[i].ID == id {
				at = i
			}
		}
		if at < 0 {
			return errEventNotFound
		}
		other := at - 1
		if direction == "down" {
			other = at + 1
		}
		msg := "Urutan acara mengikuti tanggal. Ubah tanggalnya untuk memindahkan acara ini."
		if other < 0 || other >= len(events) || !events[other].EventDate.Equal(events[at].EventDate) {
			return apperror.Validation(msg, map[string][]string{"direction": {msg}}).WithCode("event_order_by_date")
		}
		events[at], events[other] = events[other], events[at]
		for i, e := range events {
			if e.SortOrder != i {
				if err := s.repo.SetSortOrder(ctx, projectID, e.ID, i); err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.List(ctx, tenantID, projectID)
}

// Duplicate copies an event (same kind, date, venue and sessions) as a new,
// not-main event placed right after the others.
func (s *EventService) Duplicate(ctx context.Context, tenantID, projectID, id, actorStaffID int64) (*domain.ProjectEvent, error) {
	if _, err := s.projects.Get(ctx, tenantID, projectID); err != nil {
		return nil, err
	}
	var newID int64
	err := s.inProjectTx(ctx, projectID, func(ctx context.Context) error {
		src, err := s.get(ctx, projectID, id)
		if err != nil {
			return err
		}
		e := *src
		e.ID, e.IsMain = 0, false
		e.Sessions = append([]domain.EventSession(nil), src.Sessions...)
		for i := range e.Sessions {
			e.Sessions[i].ID, e.Sessions[i].EventID = 0, 0
		}
		if e.SortOrder, err = s.repo.NextSortOrder(ctx, projectID); err != nil {
			return err
		}
		if err := s.repo.Create(ctx, &e); err != nil {
			return err
		}
		newID = e.ID
		s.activity.Record(ctx, &projectID, domain.ActivityEventAdded, actorStaffID, "project_event", formatID(e.ID), e.KindName,
			"Acara diduplikat: "+e.KindName+" "+e.EventDate.Format("02-01-2006")+guestNote(e))
		return s.lines.AfterEventsChanged(ctx, tenantID, projectID, actorStaffID)
	})
	if err != nil {
		return nil, err
	}
	return s.get(ctx, projectID, newID)
}

// capacityRefusal applies the capacity rule (PLAN §4.1) to events already
// stored, against the venues given (a project venue's copied capacities, by
// supplier id). An event whose venue is not in venues is not checked. It is
// nil when everything fits, or when `offers` is not wired yet; otherwise a 422
// whose message is lead followed by the first problem, which names the numbers
// ("Gedung Melati muat 500 orang berdiri, ...").
func (s *EventService) capacityRefusal(lead string, events []domain.ProjectEvent, venues map[int64]domain.ProjectSupplier) error {
	if s == nil || s.catalog == nil {
		return nil
	}
	in := make([]offerscontracts.CapacityEvent, 0, len(events))
	caps := map[int64]offerscontracts.CapacityVenue{}
	for _, e := range events {
		ce := offerscontracts.CapacityEvent{KindName: e.KindName, Layout: string(e.Layout)}
		if ce.KindName == "" {
			ce.KindName = "Acara"
		}
		if e.VenueSupplierID != nil {
			if v, ok := venues[*e.VenueSupplierID]; ok {
				id := v.ID
				ce.VenueID = &id
				caps[id] = offerscontracts.CapacityVenue{Name: v.Name, Standing: v.CapacityStanding, Seated: v.CapacitySeated}
			}
		}
		for _, ss := range e.Sessions {
			ce.Sessions = append(ce.Sessions, offerscontracts.CapacitySession{Label: ss.Label, Guests: ss.Guests})
		}
		in = append(in, ce)
	}
	problems := s.catalog.CheckCapacity(in, caps)
	if len(problems) == 0 {
		return nil
	}
	msgs := make([]string, len(problems))
	for i, p := range problems {
		msgs[i] = p.Message
	}
	return apperror.Validation(lead+msgs[0], map[string][]string{"capacity": msgs})
}

// guestNote is ", 650 tamu" for the activity log (empty without guests).
func guestNote(e domain.ProjectEvent) string {
	if n := domain.EventGuestTotal(e); n > 0 {
		return ", " + strconv.Itoa(n) + " tamu"
	}
	return ""
}

// guestChanges lists, for the activity log, every session whose guests
// changed, was added or was removed, with the old and new number ("; tamu
// Sesi 1 550 → 650").
func guestChanges(before, after domain.ProjectEvent) string {
	old := map[string]int{}
	var order []string
	for _, s := range before.Sessions {
		old[s.Label] = s.Guests
		order = append(order, s.Label)
	}
	var parts []string
	seen := map[string]bool{}
	for _, s := range after.Sessions {
		seen[s.Label] = true
		prev, had := old[s.Label]
		switch {
		case !had:
			parts = append(parts, "tamu "+s.Label+" 0 → "+strconv.Itoa(s.Guests))
		case prev != s.Guests:
			parts = append(parts, "tamu "+s.Label+" "+strconv.Itoa(prev)+" → "+strconv.Itoa(s.Guests))
		}
	}
	for _, label := range order {
		if !seen[label] {
			parts = append(parts, "tamu "+label+" "+strconv.Itoa(old[label])+" → 0 (sesi dihapus)")
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return "; " + strings.Join(parts, "; ")
}
