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
	"elproof/internal/shared/clock"
	"elproof/internal/shared/etag"
)

// OffersPort is what the Rincian needs from `offers` (ADR-0049): a package of
// the workspace as a line copies it, and the pricing rules of PLAN §2.1 (the
// same pure functions a quotation uses). Wired from main.go.
type OffersPort interface {
	PackageCopy(ctx context.Context, tenantID, packageID int64) (offerscontracts.PackageCopy, error)
	ExtraGuests(rule offerscontracts.PackageRule, events []offerscontracts.EventGuests) offerscontracts.ExtraGuestsResult
	LineTotals(lines []offerscontracts.Line, isPKP bool) (offerscontracts.Totals, error)
}

// LineRepository stores project_lines and keeps the contract cache on projects
// (rewritten with one UPDATE … SET … = (SELECT SUM …) in the caller's
// transaction).
type LineRepository interface {
	List(ctx context.Context, tenantID, projectID int64, withDeleted bool) ([]domain.ProjectLine, error)
	Get(ctx context.Context, tenantID, projectID, id int64) (*domain.ProjectLine, error)
	Insert(ctx context.Context, l *domain.ProjectLine) error
	Update(ctx context.Context, l *domain.ProjectLine) error
	SetAmount(ctx context.Context, tenantID, projectID, id, amount int64) error
	SetSortOrder(ctx context.Context, tenantID, projectID, id int64, order int) error
	SoftDelete(ctx context.Context, tenantID, projectID, id int64, at time.Time) (bool, error)
	Restore(ctx context.Context, tenantID, projectID, id int64, notBefore time.Time) (bool, error)
	RefreshContract(ctx context.Context, tenantID, projectID int64, bumpVersion bool) error
	Contract(ctx context.Context, tenantID, projectID int64) (domain.ContractCache, error)
	PackageNames(ctx context.Context, tenantID int64, projectIDs []int64) (map[int64][]string, error)
}

// LineDraft is one line as an editor sends it. An existing line has its ID; a
// new one may carry a Key its children point at (ParentKey) before it has an
// ID. ExtraGuests lines are never taken from a draft: the server keeps one per
// package whose rule charges extra guests.
type LineDraft struct {
	ID        int64
	Key       string
	ParentID  int64
	ParentKey string
	Kind      domain.LineKind
	// PackageID is the catalog package a NEW Package line copies.
	PackageID   int64
	Description string
	CategoryID  *int64
	EventID     *int64
	Quantity    *int64
	Unit        string
	UnitPrice   *int64
	// Amount is typed for a Discount only (every other amount is derived).
	Amount      int64
	IsBonus     bool
	SupplierRef *int64
}

// LinePatch is PATCH /lines/{id}: only the fields sent change.
type LinePatch struct {
	Description, Unit                                Opt[string]
	CategoryID, EventID, Quantity, UnitPrice, Amount Opt[int64]
	SupplierRef                                      Opt[int64]
	IsBonus                                          Opt[bool]
}

// LinesView is the Rincian as a screen shows it: the lines in order (each
// ExtraGuests line explained), the totals, the lines' version, how the payment
// schedule compares with the contract, and the margin (the presentation shows
// it only with finance.margin.view).
type LinesView struct {
	ProjectID     int64
	Lines         []LineView
	Totals        offerscontracts.Totals
	IsPKP         bool
	LinesVersion  time.Time
	TermsTotal    int64
	HasTerms      bool
	TermsMismatch bool
	Margin        domain.Margin
}

// LineView is one line with what explains it.
type LineView struct {
	domain.ProjectLine
	// Key echoes the draft's key (new lines in a preview or a save).
	Key string
	// ExtraGuests explains an ExtraGuests line ("650 tamu − 500 termasuk = 150
	// → 3 × 50 tamu × Rp4.000.000").
	ExtraGuests *ExtraGuestsView
	// parent is the index of the line's package in the view (-1: none); draft
	// is the index of the draft the line came from (-1: made by the server).
	parent, draft int
	// components is the catalog's components of a new package, generated when
	// the draft sent none.
	components []offerscontracts.PackageComponentCopy
}

// ParentIndex is the index in the same view of the package this line hangs
// off (-1 when it is none, or when it is a stored line: then ParentLineID
// says). A preview's new lines have no ID yet, so this is how a screen links
// them.
func (v LineView) ParentIndex() int { return v.parent }

// ExtraGuestsView is the arithmetic behind an extra-guest charge.
type ExtraGuestsView struct {
	EventID        *int64
	EventKindName  string
	Guests         int
	IncludedGuests int
	Over           int
	Step           int
	StepPrice      int64
	Steps          int
	// NoGuests: the covered events have no guests yet, so the charge is 0
	// (PLAN §4.2 "dengan tanda").
	NoGuests bool
}

// LineService runs a project's Rincian (PLAN §3.1, §3.2): packages copied
// from the catalog with their components and extra-guest charge, items,
// bonuses and discounts. Every write recomputes the extra-guest charges, checks
// the totals with LineTotals and rewrites the contract cache in the same
// transaction; a line write also moves lines_version (its own If-Match).
type LineService struct {
	repo      LineRepository
	projects  *ProjectService
	eventSvc  *EventService
	events    EventRepository
	suppliers SupplierRepository
	terms     PaymentTermRepository
	refs      *References
	tx        Transactor
	activity  *ActivityService
	offers    OffersPort
}

func NewLineService(repo LineRepository, projects *ProjectService, eventSvc *EventService, events EventRepository,
	suppliers SupplierRepository, terms PaymentTermRepository, refs *References, tx Transactor, activity *ActivityService) *LineService {
	if tx == nil {
		tx = noTransactor{}
	}
	return &LineService{repo: repo, projects: projects, eventSvc: eventSvc, events: events, suppliers: suppliers,
		terms: terms, refs: refs, tx: tx, activity: activity}
}

// SetOffers completes the wiring with `offers` (built after `projects`).
func (s *LineService) SetOffers(o OffersPort) { s.offers = o }

var errLineNotFound = apperror.NotFound("Baris rincian tidak ditemukan")

func (s *LineService) ready() error {
	if s == nil || s.offers == nil {
		return apperror.Internal("Rincian belum tersedia")
	}
	return nil
}

func (s *LineService) inProjectTx(ctx context.Context, projectID int64, fn func(ctx context.Context) error) error {
	return s.tx.InTx(ctx, func(ctx context.Context) error {
		if err := s.tx.LockProject(ctx, projectID); err != nil {
			return err
		}
		return fn(ctx)
	})
}

// lineState is what every computation of one project's lines starts from.
type lineState struct {
	project *domain.Project
	events  []domain.ProjectEvent
	current []domain.ProjectLine
}

func (s *LineService) load(ctx context.Context, tenantID, projectID int64) (*lineState, error) {
	p, err := s.projects.Get(ctx, tenantID, projectID)
	if err != nil {
		return nil, err
	}
	events, err := s.events.ListByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	domain.SortEvents(events)
	current, err := s.repo.List(ctx, tenantID, projectID, false)
	if err != nil {
		return nil, err
	}
	domain.SortLines(current)
	return &lineState{project: p, events: events, current: current}, nil
}

// --- reads ---

// View is the project's Rincian now.
func (s *LineService) View(ctx context.Context, tenantID, projectID int64) (*LinesView, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	st, err := s.load(ctx, tenantID, projectID)
	if err != nil {
		return nil, err
	}
	views := make([]LineView, len(st.current))
	for i, l := range st.current {
		views[i] = LineView{ProjectLine: l, parent: -1, draft: -1}
	}
	return s.viewOf(ctx, st, views, st.project.LinesVersion)
}

// Preview computes unsaved drafts with the same functions a save uses and
// writes nothing (PLAN §2.2): its total is the total the save would store.
func (s *LineService) Preview(ctx context.Context, tenantID, projectID int64, drafts []LineDraft) (*LinesView, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	st, err := s.load(ctx, tenantID, projectID)
	if err != nil {
		return nil, err
	}
	views, err := s.resolve(ctx, tenantID, st, drafts, batchPrefix)
	if err != nil {
		return nil, err
	}
	return s.viewOf(ctx, st, views, st.project.LinesVersion)
}

func (s *LineService) viewOf(ctx context.Context, st *lineState, views []LineView, version time.Time) (*LinesView, error) {
	s.explain(st, views)
	totals, err := s.offers.LineTotals(offerLines(views), st.project.IsPKP)
	if err != nil {
		// Stored lines that no longer add up (cannot be written that way):
		// show the stored cache rather than fail the read.
		totals = offerscontracts.Totals{Subtotal: st.project.ContractSubtotal, Tax: st.project.ContractTax, Total: st.project.ContractTotal}
	}
	out := &LinesView{ProjectID: st.project.ID, Lines: views, Totals: totals, IsPKP: st.project.IsPKP, LinesVersion: version}
	if out.TermsTotal, out.HasTerms, err = s.termsTotal(ctx, st.project.ID); err != nil {
		return nil, err
	}
	out.TermsMismatch = out.HasTerms && out.TermsTotal != totals.Total
	suppliers, err := s.suppliers.ListByProject(ctx, st.project.ID)
	if err != nil {
		return nil, err
	}
	out.Margin = domain.ComputeMargin(totals.Subtotal, suppliers)
	return out, nil
}

func (s *LineService) termsTotal(ctx context.Context, projectID int64) (int64, bool, error) {
	if s.terms == nil {
		return 0, false, nil
	}
	terms, err := s.terms.ListByProject(ctx, projectID)
	if err != nil {
		return 0, false, err
	}
	return domain.TermsTotal(terms), len(terms) > 0, nil
}

// explain fills the ExtraGuests arithmetic of every ExtraGuests line.
func (s *LineService) explain(st *lineState, views []LineView) {
	rules := map[int64]*domain.LinePackageRule{}
	for i := range views {
		if views[i].Kind == domain.LinePackage && views[i].PackageRule != nil {
			rules[views[i].ID] = views[i].PackageRule
		}
	}
	for i := range views {
		v := &views[i]
		if v.Kind != domain.LineExtraGuests {
			continue
		}
		var rule *domain.LinePackageRule
		if v.parent >= 0 {
			rule = views[v.parent].PackageRule
		} else if v.ParentLineID != nil {
			rule = rules[*v.ParentLineID]
		}
		if rule == nil {
			continue
		}
		res := s.offers.ExtraGuests(toOfferRule(*rule), eventGuests(st.events))
		ex := &ExtraGuestsView{Guests: res.Guests, Over: res.Over, Steps: res.Steps, NoGuests: res.Guests == 0}
		if rule.IncludedGuests != nil {
			ex.IncludedGuests = *rule.IncludedGuests
		}
		if rule.ExtraGuestStep != nil {
			ex.Step = *rule.ExtraGuestStep
		}
		if rule.ExtraGuestStepPrice != nil {
			ex.StepPrice = *rule.ExtraGuestStepPrice
		}
		if e := domain.DecisiveEvent(*rule, st.events); e != nil {
			id := e.ID
			ex.EventID, ex.EventKindName = &id, e.KindName
		}
		v.ExtraGuests = ex
	}
}

// --- the one write path ---

// prefixFunc names the field-error prefix of draft i ("lines.3." in a batch,
// "" for the one line a single-line request is about).
type prefixFunc func(i int) string

func batchPrefix(i int) string { return "lines." + strconv.Itoa(i) + "." }

func singlePrefix(target int) prefixFunc {
	return func(i int) string {
		if i == target {
			return ""
		}
		return batchPrefix(i)
	}
}

// save runs one line write: under the project's lock it checks the lines'
// version, builds the drafts from the current lines, resolves and validates
// them, writes the difference, rewrites the contract cache and moves
// lines_version, all in one transaction.
func (s *LineService) save(ctx context.Context, tenantID, projectID, actorStaffID int64, expected *time.Time,
	build func(current []domain.ProjectLine) ([]LineDraft, prefixFunc, error)) (*LinesView, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	if _, err := s.projects.Get(ctx, tenantID, projectID); err != nil {
		return nil, err
	}
	err := s.inProjectTx(ctx, projectID, func(ctx context.Context) error {
		st, err := s.load(ctx, tenantID, projectID)
		if err != nil {
			return err
		}
		if err := s.checkVersion(ctx, tenantID, st, expected); err != nil {
			return err
		}
		drafts, prefix, err := build(st.current)
		if err != nil {
			return err
		}
		views, err := s.resolve(ctx, tenantID, st, drafts, prefix)
		if err != nil {
			return err
		}
		return s.persist(ctx, tenantID, st, views, actorStaffID)
	})
	if err != nil {
		return nil, err
	}
	return s.View(ctx, tenantID, projectID)
}

// checkVersion is the lines' own If-Match (ADR-0036): a stale version is a 409
// carrying the Rincian as it is now.
func (s *LineService) checkVersion(ctx context.Context, tenantID int64, st *lineState, expected *time.Time) error {
	if expected == nil || etag.Same(st.project.LinesVersion, *expected) {
		return nil
	}
	views := make([]LineView, len(st.current))
	for i, l := range st.current {
		views[i] = LineView{ProjectLine: l, parent: -1, draft: -1}
	}
	current, err := s.viewOf(ctx, st, views, st.project.LinesVersion)
	if err != nil {
		return err
	}
	return apperror.Stale(current)
}

// Replace saves the whole Rincian as the editor holds it ("Simpan rincian"):
// lines left out are deleted (a package with its children), new ones added,
// the rest updated, in the order sent.
func (s *LineService) Replace(ctx context.Context, tenantID, projectID, actorStaffID int64, drafts []LineDraft, expected *time.Time) (*LinesView, error) {
	return s.save(ctx, tenantID, projectID, actorStaffID, expected, func([]domain.ProjectLine) ([]LineDraft, prefixFunc, error) {
		return drafts, batchPrefix, nil
	})
}

// Add adds one line (a Component goes after its package's other components;
// anything else at the end).
func (s *LineService) Add(ctx context.Context, tenantID, projectID, actorStaffID int64, draft LineDraft, expected *time.Time) (*LinesView, error) {
	draft.ID = 0
	return s.save(ctx, tenantID, projectID, actorStaffID, expected, func(current []domain.ProjectLine) ([]LineDraft, prefixFunc, error) {
		drafts := draftsOf(current)
		at := len(drafts)
		if draft.Kind == domain.LineComponent && draft.ParentID != 0 {
			for i, d := range drafts {
				if d.ID == draft.ParentID || d.ParentID == draft.ParentID {
					at = i + 1
				}
			}
		}
		drafts = append(drafts[:at], append([]LineDraft{draft}, drafts[at:]...)...)
		return drafts, singlePrefix(at), nil
	})
}

// Patch changes the fields sent of one line.
func (s *LineService) Patch(ctx context.Context, tenantID, projectID, lineID, actorStaffID int64, patch LinePatch, expected *time.Time) (*LinesView, error) {
	return s.save(ctx, tenantID, projectID, actorStaffID, expected, func(current []domain.ProjectLine) ([]LineDraft, prefixFunc, error) {
		drafts := draftsOf(current)
		at := -1
		for i := range drafts {
			if drafts[i].ID == lineID {
				at = i
			}
		}
		if at < 0 {
			return nil, nil, errLineNotFound
		}
		if drafts[at].Kind == domain.LineExtraGuests {
			msg := "Tambahan tamu dihitung dari tamu acara dan tidak bisa diubah. Untuk harga lain, tambahkan Potongan."
			return nil, nil, apperror.Validation(msg, map[string][]string{"amount": {msg}}).WithCode("extra_guests_computed")
		}
		applyLinePatch(&drafts[at], patch)
		return drafts, singlePrefix(at), nil
	})
}

// Reorder puts the lines in the order of ids (every live line except the
// extra-guest charges, which always close their package). A component stays
// inside its package.
func (s *LineService) Reorder(ctx context.Context, tenantID, projectID, actorStaffID int64, ids []int64, expected *time.Time) (*LinesView, error) {
	return s.save(ctx, tenantID, projectID, actorStaffID, expected, func(current []domain.ProjectLine) ([]LineDraft, prefixFunc, error) {
		byID := map[int64]domain.ProjectLine{}
		want := 0
		for _, l := range current {
			if l.Kind != domain.LineExtraGuests {
				byID[l.ID] = l
				want++
			}
		}
		msg := "Kirim urutan semua baris rincian, masing-masing satu kali."
		bad := apperror.Validation(msg, map[string][]string{"ids": {msg}})
		if len(ids) != want {
			return nil, nil, bad
		}
		seen := map[int64]bool{}
		drafts := make([]LineDraft, 0, len(ids))
		for _, id := range ids {
			l, ok := byID[id]
			if !ok || seen[id] {
				return nil, nil, bad
			}
			seen[id] = true
			drafts = append(drafts, draftOf(l))
		}
		return drafts, batchPrefix, nil
	})
}

// Delete puts a line in the "deleted" state (restorable for the retention,
// ADR-0034); a package takes its components and extra-guest charge with it,
// never a vendor, a venue or a payment. The extra-guest charge itself cannot
// be deleted on its own.
func (s *LineService) Delete(ctx context.Context, tenantID, projectID, lineID, actorStaffID int64, expected *time.Time, now time.Time) (*LinesView, time.Time, error) {
	if err := s.ready(); err != nil {
		return nil, time.Time{}, err
	}
	if _, err := s.projects.Get(ctx, tenantID, projectID); err != nil {
		return nil, time.Time{}, err
	}
	err := s.inProjectTx(ctx, projectID, func(ctx context.Context) error {
		st, err := s.load(ctx, tenantID, projectID)
		if err != nil {
			return err
		}
		if err := s.checkVersion(ctx, tenantID, st, expected); err != nil {
			return err
		}
		var target *domain.ProjectLine
		for i := range st.current {
			if st.current[i].ID == lineID {
				target = &st.current[i]
			}
		}
		if target == nil {
			return errLineNotFound
		}
		if target.Kind == domain.LineExtraGuests {
			msg := "Tambahan tamu ikut paketnya. Hapus paketnya, atau ubah tamu acara."
			return apperror.Validation(msg, map[string][]string{"id": {msg}}).WithCode("extra_guests_computed")
		}
		if _, err := s.repo.SoftDelete(ctx, tenantID, projectID, lineID, now); err != nil {
			return err
		}
		if err := s.recompute(ctx, tenantID, projectID, actorStaffID, true); err != nil {
			return err
		}
		s.activity.Record(ctx, &projectID, domain.ActivityLineDeleted, actorStaffID, "project_line", formatID(lineID), target.Description,
			lineKindLabel(*target)+" dihapus: "+target.Description+amountNote(*target))
		return nil
	})
	if err != nil {
		return nil, time.Time{}, err
	}
	view, err := s.View(ctx, tenantID, projectID)
	return view, domain.RestorableUntil(now), err
}

// Restore brings a deleted line back (a package with the children deleted
// together with it) while the retention lasts. A component whose package is
// still deleted cannot come back alone.
func (s *LineService) Restore(ctx context.Context, tenantID, projectID, lineID, actorStaffID int64, now time.Time) (*LinesView, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	if _, err := s.projects.Get(ctx, tenantID, projectID); err != nil {
		return nil, err
	}
	err := s.inProjectTx(ctx, projectID, func(ctx context.Context) error {
		l, err := s.repo.Get(ctx, tenantID, projectID, lineID)
		if err != nil {
			return err
		}
		if l == nil || l.DeletedAt == nil || !domain.CanRestore(*l.DeletedAt, now) {
			return apperror.NotFound("Baris rincian tidak ditemukan atau sudah tidak bisa dipulihkan")
		}
		if l.ParentLineID != nil {
			parent, err := s.repo.Get(ctx, tenantID, projectID, *l.ParentLineID)
			if err != nil {
				return err
			}
			if parent == nil || parent.DeletedAt != nil {
				msg := "Pulihkan paketnya dulu."
				return apperror.Validation(msg, map[string][]string{"id": {msg}})
			}
		}
		ok, err := s.repo.Restore(ctx, tenantID, projectID, lineID, domain.PurgeCutoff(now))
		if err != nil {
			return err
		}
		if !ok {
			return apperror.NotFound("Baris rincian tidak ditemukan atau sudah tidak bisa dipulihkan")
		}
		if err := s.recompute(ctx, tenantID, projectID, actorStaffID, true); err != nil {
			return err
		}
		s.activity.Record(ctx, &projectID, domain.ActivityRestored, actorStaffID, "project_line", formatID(lineID), l.Description,
			lineKindLabel(*l)+" dipulihkan: "+l.Description+amountNote(*l))
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.View(ctx, tenantID, projectID)
}

// AfterEventsChanged is the hook the event service runs inside its own
// transaction whenever guests, sessions or events change: every extra-guest
// charge is recomputed from its package's rule copy (never the catalog), the
// totals are checked and the contract cache rewritten, in the same
// transaction. It does not move lines_version (the lines were not edited). A
// change that would leave the discounts above the subtotal is refused.
func (s *LineService) AfterEventsChanged(ctx context.Context, tenantID, projectID, actorStaffID int64) error {
	if s == nil || s.offers == nil {
		return nil
	}
	return s.recompute(ctx, tenantID, projectID, actorStaffID, false)
}

// recompute brings every extra-guest charge in line with the events, checks
// the totals and rewrites the contract cache. Runs inside the caller's
// transaction.
func (s *LineService) recompute(ctx context.Context, tenantID, projectID, actorStaffID int64, bump bool) error {
	st, err := s.load(ctx, tenantID, projectID)
	if err != nil {
		return err
	}
	guests := eventGuests(st.events)
	lines := append([]domain.ProjectLine(nil), st.current...)
	for i := range lines {
		pkg := lines[i]
		if pkg.Kind != domain.LinePackage || pkg.PackageRule == nil || !pkg.PackageRule.ChargesExtraGuests() {
			continue
		}
		amount := s.offers.ExtraGuests(toOfferRule(*pkg.PackageRule), guests).Amount
		child, lastOrder := -1, pkg.SortOrder
		for j := range lines {
			if lines[j].ParentLineID != nil && *lines[j].ParentLineID == pkg.ID {
				if lines[j].SortOrder > lastOrder {
					lastOrder = lines[j].SortOrder
				}
				if lines[j].Kind == domain.LineExtraGuests {
					child = j
				}
			}
		}
		if child < 0 {
			pid := pkg.ID
			l := domain.ProjectLine{TenantID: tenantID, ProjectID: projectID, Kind: domain.LineExtraGuests, ParentLineID: &pid,
				Description: extraGuestsDescription, Amount: amount, SortOrder: lastOrder}
			if err := s.repo.Insert(ctx, &l); err != nil {
				return err
			}
			lines = append(lines, l)
			continue
		}
		if old := lines[child].Amount; old != amount {
			if err := s.repo.SetAmount(ctx, tenantID, projectID, lines[child].ID, amount); err != nil {
				return err
			}
			lines[child].Amount = amount
			s.activity.Record(ctx, &projectID, domain.ActivityLineUpdated, actorStaffID, "project_line", formatID(lines[child].ID), pkg.Description,
				"Tambahan tamu "+pkg.Description+" berubah dari "+formatRupiah(old)+" menjadi "+formatRupiah(amount))
		}
	}
	views := make([]LineView, len(lines))
	for i, l := range lines {
		views[i] = LineView{ProjectLine: l, parent: -1, draft: -1}
	}
	if _, err := s.offers.LineTotals(offerLines(views), st.project.IsPKP); err != nil {
		return totalsRefusal(err, views, func(int) string { return "" })
	}
	return s.repo.RefreshContract(ctx, tenantID, projectID, bump)
}

const extraGuestsDescription = "Tambahan tamu"

// resolve turns drafts into the lines as they would be stored, in screen
// order (each package followed by its components and its extra-guest charge),
// with every problem of every line reported together as one 422.
func (s *LineService) resolve(ctx context.Context, tenantID int64, st *lineState, drafts []LineDraft, prefix prefixFunc) ([]LineView, error) {
	fe := newFieldErrors()
	if len(drafts) > domain.MaxProjectLines {
		msg := "Rincian terlalu panjang: maksimal " + strconv.Itoa(domain.MaxProjectLines) + " baris."
		return nil, apperror.Validation(msg, map[string][]string{"lines": {msg}})
	}
	current := map[int64]domain.ProjectLine{}
	for _, l := range st.current {
		current[l.ID] = l
	}
	eventIDs := map[int64]bool{}
	kindsPresent := map[int64]bool{}
	for _, e := range st.events {
		eventIDs[e.ID] = true
		kindsPresent[e.EventKindID] = true
	}

	items := make([]LineView, 0, len(drafts))
	seenIDs := map[int64]bool{}
	for i, d := range drafts {
		p := prefix(i)
		var l domain.ProjectLine
		var cur *domain.ProjectLine
		if d.ID != 0 {
			c, ok := current[d.ID]
			if !ok || seenIDs[d.ID] {
				fe.add("Baris rincian tidak ditemukan", p+"id")
				continue
			}
			seenIDs[d.ID] = true
			if c.Kind == domain.LineExtraGuests {
				continue // always the server's
			}
			if d.Kind != "" && d.Kind != c.Kind {
				fe.add("Jenis baris tidak bisa diubah", p+"kind")
				continue
			}
			cur, l = &c, c
		} else {
			if d.Kind == domain.LineExtraGuests {
				continue // always the server's
			}
			if !domain.IsValidLineKind(d.Kind) {
				fe.add("Jenis baris tidak dikenal", p+"kind")
				continue
			}
			l = domain.ProjectLine{TenantID: tenantID, ProjectID: st.project.ID, Kind: d.Kind}
		}
		v := LineView{Key: d.Key, parent: -1, draft: i}
		l.Description = strings.TrimSpace(d.Description)
		l.CategoryID, l.EventID, l.SupplierRef = positive(d.CategoryID), positive(d.EventID), positive(d.SupplierRef)
		l.Quantity, l.UnitPrice, l.Unit, l.IsBonus = nil, nil, "", false

		switch l.Kind {
		case domain.LinePackage:
			if cur == nil {
				cp, ok, err := s.packageCopy(ctx, tenantID, d.PackageID, p, fe)
				if err != nil {
					return nil, err
				}
				if !ok {
					continue
				}
				rule := fromOfferRule(cp.Rule)
				l.PackageRule = &rule
				if l.Description == "" {
					l.Description = rule.Name
				}
				if missing := missingKinds(rule, kindsPresent); len(missing) > 0 {
					msg := "Paket ini mencakup acara " + s.kindNames(ctx, tenantID, missing) + " yang belum ada di proyek. Tambahkan acaranya dulu."
					fe.add(msg, p+"packageId")
				}
				v.components = cp.Components
			}
			if l.PackageRule != nil {
				l.Amount = l.PackageRule.Price
			}
		case domain.LineComponent:
			l.Amount = 0
		case domain.LineItem:
			l.Unit = strings.TrimSpace(d.Unit)
			if len([]rune(l.Unit)) > domain.MaxLineUnitLen {
				fe.add("Satuan maksimal "+strconv.Itoa(domain.MaxLineUnitLen)+" karakter", p+"unit")
			}
			l.IsBonus = d.IsBonus
			q, price := d.Quantity, d.UnitPrice
			if l.IsBonus {
				zero := int64(0)
				price = &zero
			}
			switch {
			case q == nil:
				fe.add("Isi jumlah", p+"quantity")
			case *q < 1 || *q > domain.MaxLineQuantity:
				fe.add("Jumlah harus 1 sampai "+strconv.Itoa(domain.MaxLineQuantity), p+"quantity")
			}
			switch {
			case price == nil:
				fe.add("Isi harga satuan", p+"unitPrice")
			case *price < 0 || *price > domain.MaxMoney:
				fe.add("Harga satuan tidak boleh negatif dan harus wajar", p+"unitPrice")
			}
			if q != nil && price != nil && *q >= 1 && *q <= domain.MaxLineQuantity && *price >= 0 && *price <= domain.MaxMoney {
				if *price > 0 && *q > domain.MaxMoney / *price {
					fe.add("Nominal baris terlalu besar", p+"unitPrice")
				} else {
					qq, pp := *q, *price
					l.Quantity, l.UnitPrice, l.Amount = &qq, &pp, qq*pp
				}
			}
		case domain.LineDiscount:
			if d.Amount <= 0 || d.Amount > domain.MaxMoney {
				fe.add("Nominal potongan harus lebih dari 0 dan wajar", p+"amount")
			}
			l.Amount = d.Amount
		}
		if l.Description == "" {
			fe.add("Isi uraian", p+"description")
		} else if len([]rune(l.Description)) > domain.MaxLineDescriptionLen {
			fe.add("Uraian maksimal "+strconv.Itoa(domain.MaxLineDescriptionLen)+" karakter", p+"description")
		}
		if l.EventID != nil && !eventIDs[*l.EventID] {
			fe.add("Acara tidak ditemukan di proyek ini", p+"eventId")
		}
		if l.CategoryID != nil && (cur == nil || !sameRef(cur.CategoryID, l.CategoryID)) && s.refs != nil {
			if err := s.refs.Category(ctx, tenantID, p+"categoryId", *l.CategoryID); err != nil && !fe.mergeErr(err) {
				return nil, err
			}
		}
		if l.SupplierRef != nil && (cur == nil || !sameRef(cur.SupplierRef, l.SupplierRef)) {
			sup, err := s.suppliers.FindByID(ctx, st.project.ID, *l.SupplierRef)
			if err != nil {
				return nil, err
			}
			if sup == nil || sup.PaidBy != domain.PayerWO || !sup.IsActive() {
				fe.add("Pilih vendor atau venue proyek ini yang dibayar WO", p+"supplierRef")
			}
		}
		if cur != nil {
			l.ParentLineID = cur.ParentLineID
		}
		v.ProjectLine = l
		items = append(items, v)
	}

	// Structure: every component hangs off a package of this same set.
	pkgByID, pkgByKey := map[int64]int{}, map[string]int{}
	for i, v := range items {
		if v.Kind != domain.LinePackage {
			continue
		}
		if v.ID != 0 {
			pkgByID[v.ID] = i
		} else if v.Key != "" {
			pkgByKey[v.Key] = i
		}
	}
	children := map[int][]int{}
	hasComponents := map[int]bool{}
	var top []int
	for i := range items {
		v := &items[i]
		if v.Kind != domain.LineComponent {
			top = append(top, i)
			continue
		}
		d := drafts[v.draft]
		parent, ok := -1, false
		switch {
		case v.ID != 0 && v.ParentLineID != nil:
			parent, ok = pkgByID[*v.ParentLineID]
			if d.ParentID != 0 && d.ParentID != *v.ParentLineID {
				fe.add("Isi paket hanya bisa berpindah di dalam paketnya", prefix(v.draft)+"parentId")
			}
		case d.ParentID != 0:
			parent, ok = pkgByID[d.ParentID]
		case d.ParentKey != "":
			parent, ok = pkgByKey[d.ParentKey]
		}
		if !ok {
			fe.add("Isi paket harus berada di bawah paketnya", prefix(v.draft)+"parentId")
			continue
		}
		v.parent = parent
		children[parent] = append(children[parent], i)
		hasComponents[parent] = true
	}
	if err := fe.err(); err != nil {
		return nil, err
	}

	// Screen order, generated components and the extra-guest charges.
	guests := eventGuests(st.events)
	out := make([]LineView, 0, len(items)+8)
	for _, i := range top {
		pkg := items[i]
		pkg.parent = -1
		out = append(out, pkg)
		if pkg.Kind != domain.LinePackage {
			continue
		}
		at := len(out) - 1
		for _, c := range children[i] {
			child := items[c]
			child.parent = at
			out = append(out, child)
		}
		if !hasComponents[i] && pkg.ID == 0 {
			for _, comp := range pkg.components {
				var cat *int64
				if comp.CategoryID != nil {
					v := *comp.CategoryID
					cat = &v
				}
				out = append(out, LineView{ProjectLine: domain.ProjectLine{TenantID: tenantID, ProjectID: st.project.ID,
					Kind: domain.LineComponent, Description: comp.Description, CategoryID: cat}, parent: at, draft: -1})
			}
		}
		if pkg.PackageRule == nil || !pkg.PackageRule.ChargesExtraGuests() {
			continue
		}
		extra := LineView{ProjectLine: domain.ProjectLine{TenantID: tenantID, ProjectID: st.project.ID,
			Kind: domain.LineExtraGuests, Description: extraGuestsDescription}, parent: at, draft: -1}
		if pkg.ID != 0 {
			for _, l := range st.current {
				if l.Kind == domain.LineExtraGuests && l.ParentLineID != nil && *l.ParentLineID == pkg.ID {
					extra.ProjectLine = l
				}
			}
		}
		extra.Amount = s.offers.ExtraGuests(toOfferRule(*pkg.PackageRule), guests).Amount
		out = append(out, extra)
	}
	for i := range out {
		out[i].SortOrder = i
	}
	if _, err := s.offers.LineTotals(offerLines(out), st.project.IsPKP); err != nil {
		return nil, totalsRefusal(err, out, prefix)
	}
	return out, nil
}

// packageCopy reads the catalog package a new Package line copies; ok is false
// (with the problem added to fe) when there is none usable.
func (s *LineService) packageCopy(ctx context.Context, tenantID, packageID int64, p string, fe *fieldErrors) (offerscontracts.PackageCopy, bool, error) {
	if packageID <= 0 {
		fe.add("Pilih paket dari katalog", p+"packageId")
		return offerscontracts.PackageCopy{}, false, nil
	}
	cp, err := s.offers.PackageCopy(ctx, tenantID, packageID)
	if err != nil {
		if isNotFound(err) {
			fe.add("Paket tidak ditemukan", p+"packageId")
			return cp, false, nil
		}
		return cp, false, err
	}
	if !cp.IsActive {
		fe.add("Paket ini nonaktif. Pilih paket lain.", p+"packageId")
		return cp, false, nil
	}
	return cp, true, nil
}

func (s *LineService) kindNames(ctx context.Context, tenantID int64, ids []int64) string {
	names := make([]string, 0, len(ids))
	for _, id := range ids {
		name := ""
		if s.eventSvc != nil && s.eventSvc.catalog != nil {
			if k, err := s.eventSvc.catalog.GetEventKind(ctx, tenantID, id); err == nil {
				name = k.Name
			}
		}
		if name == "" {
			name = "lain"
		}
		names = append(names, name)
	}
	return strings.Join(names, ", ")
}

// totalsRefusal turns a LineTotals error into the 422 the editor shows, keyed
// on every discount ("Potongan melebihi subtotal (Rp95.000.000). Kurangi
// potongan.").
func totalsRefusal(err error, views []LineView, prefix prefixFunc) error {
	if errors.Is(err, offerscontracts.ErrDiscountExceedsSubtotal) {
		var gross int64
		for _, v := range views {
			switch v.Kind {
			case domain.LinePackage, domain.LineExtraGuests, domain.LineItem:
				gross += v.Amount
			}
		}
		msg := "Potongan melebihi subtotal (" + formatRupiah(gross) + "). Kurangi potongan."
		fields := map[string][]string{}
		for _, v := range views {
			if v.Kind == domain.LineDiscount {
				key := "amount"
				if v.draft >= 0 {
					key = prefix(v.draft) + "amount"
				}
				fields[key] = append(fields[key], msg)
			}
		}
		if len(fields) == 0 {
			fields["lines"] = []string{msg}
		}
		return apperror.Validation(msg, fields).WithCode("discount_exceeds_subtotal")
	}
	return apperror.Validation("Nominal baris tidak valid", map[string][]string{"lines": {err.Error()}})
}

// persist writes the difference between the current lines and views, then
// rewrites the contract cache and moves lines_version. Activity records the
// old and new value of every change.
func (s *LineService) persist(ctx context.Context, tenantID int64, st *lineState, views []LineView, actorStaffID int64) error {
	projectID := st.project.ID
	keep := map[int64]bool{}
	for _, v := range views {
		if v.ID != 0 {
			keep[v.ID] = true
		}
	}
	current := map[int64]domain.ProjectLine{}
	stamp := clock.Now()
	for _, l := range st.current {
		current[l.ID] = l
		if keep[l.ID] || (l.ParentLineID != nil && !keep[*l.ParentLineID]) {
			continue // kept, or goes with its package
		}
		if _, err := s.repo.SoftDelete(ctx, tenantID, projectID, l.ID, stamp); err != nil {
			return err
		}
		if l.Kind != domain.LineExtraGuests {
			s.activity.Record(ctx, &projectID, domain.ActivityLineDeleted, actorStaffID, "project_line", formatID(l.ID), l.Description,
				lineKindLabel(l)+" dihapus: "+l.Description+amountNote(l))
		}
	}
	newPackages := map[int]bool{}
	for i := range views {
		v := &views[i]
		if v.parent >= 0 {
			pid := views[v.parent].ID
			v.ParentLineID = &pid
		}
		if v.ID == 0 {
			if err := s.repo.Insert(ctx, &v.ProjectLine); err != nil {
				return err
			}
			switch {
			case v.Kind == domain.LinePackage:
				newPackages[i] = true
				n := 0
				for _, c := range views[i+1:] {
					if c.parent == i && c.Kind == domain.LineComponent {
						n++
					}
				}
				s.activity.Record(ctx, &projectID, domain.ActivityLineAdded, actorStaffID, "project_line", formatID(v.ID), v.Description,
					"Paket ditambahkan: "+v.Description+amountNote(v.ProjectLine)+", "+strconv.Itoa(n)+" isi paket")
			case v.parent >= 0 && newPackages[v.parent]:
				// Logged with its package.
			case v.Kind == domain.LineExtraGuests:
				if v.Amount > 0 {
					s.activity.Record(ctx, &projectID, domain.ActivityLineAdded, actorStaffID, "project_line", formatID(v.ID), v.Description,
						"Tambahan tamu "+views[v.parent].Description+": "+formatRupiah(v.Amount))
				}
			default:
				s.activity.Record(ctx, &projectID, domain.ActivityLineAdded, actorStaffID, "project_line", formatID(v.ID), v.Description,
					lineKindLabel(v.ProjectLine)+" ditambahkan: "+v.Description+amountNote(v.ProjectLine))
			}
			continue
		}
		old := current[v.ID]
		if !lineChanged(old, v.ProjectLine) {
			continue
		}
		if err := s.repo.Update(ctx, &v.ProjectLine); err != nil {
			return err
		}
		if diff := s.lineDiff(old, v.ProjectLine, st.events); diff != "" {
			label := lineKindLabel(v.ProjectLine) + " " + old.Description
			if v.Kind == domain.LineExtraGuests && v.parent >= 0 {
				label = "Tambahan tamu " + views[v.parent].Description
			}
			s.activity.Record(ctx, &projectID, domain.ActivityLineUpdated, actorStaffID, "project_line", formatID(v.ID), v.Description,
				label+": "+diff)
		}
	}
	return s.repo.RefreshContract(ctx, tenantID, projectID, true)
}

func lineChanged(a, b domain.ProjectLine) bool {
	return a.Description != b.Description || !sameRef(a.CategoryID, b.CategoryID) || !sameRef(a.EventID, b.EventID) ||
		!sameRef(a.Quantity, b.Quantity) || a.Unit != b.Unit || !sameRef(a.UnitPrice, b.UnitPrice) || a.Amount != b.Amount ||
		a.IsBonus != b.IsBonus || !sameRef(a.SupplierRef, b.SupplierRef) || a.SortOrder != b.SortOrder
}

// lineDiff is "uraian A → B; jumlah 300 → 350; nominal Rp3.600.000 →
// Rp4.200.000" (empty when only the position moved).
func (s *LineService) lineDiff(a, b domain.ProjectLine, events []domain.ProjectEvent) string {
	var parts []string
	add := func(label, from, to string) {
		if from != to {
			parts = append(parts, label+" "+from+" → "+to)
		}
	}
	add("uraian", quoteOrDash(a.Description), quoteOrDash(b.Description))
	add("jumlah", intOrDash(a.Quantity), intOrDash(b.Quantity))
	add("satuan", quoteOrDash(a.Unit), quoteOrDash(b.Unit))
	add("harga satuan", rupiahOrDash(a.UnitPrice), rupiahOrDash(b.UnitPrice))
	add("nominal", formatRupiah(a.Amount), formatRupiah(b.Amount))
	add("bonus", yesNo(a.IsBonus), yesNo(b.IsBonus))
	add("acara", eventName(a.EventID, events), eventName(b.EventID, events))
	if !sameRef(a.SupplierRef, b.SupplierRef) {
		parts = append(parts, "pemasok diganti")
	}
	if !sameRef(a.CategoryID, b.CategoryID) {
		parts = append(parts, "kategori diganti")
	}
	return strings.Join(parts, "; ")
}

func quoteOrDash(s string) string {
	if s == "" {
		return "—"
	}
	return "\"" + s + "\""
}

func intOrDash(v *int64) string {
	if v == nil {
		return "—"
	}
	return strconv.FormatInt(*v, 10)
}

func rupiahOrDash(v *int64) string {
	if v == nil {
		return "—"
	}
	return formatRupiah(*v)
}

func yesNo(b bool) string {
	if b {
		return "ya"
	}
	return "tidak"
}

func eventName(id *int64, events []domain.ProjectEvent) string {
	if id == nil {
		return "—"
	}
	for _, e := range events {
		if e.ID == *id {
			return e.KindName
		}
	}
	return "—"
}

// lineKindLabel is the glossary word of a line's kind (DESIGN §7).
func lineKindLabel(l domain.ProjectLine) string {
	switch l.Kind {
	case domain.LinePackage:
		return "Paket"
	case domain.LineComponent:
		return "Isi paket"
	case domain.LineExtraGuests:
		return "Tambahan tamu"
	case domain.LineDiscount:
		return "Potongan"
	}
	if l.IsBonus {
		return "Bonus"
	}
	return "Item"
}

func amountNote(l domain.ProjectLine) string {
	if l.Kind == domain.LineComponent || l.IsBonus {
		return ""
	}
	if l.Kind == domain.LineDiscount {
		return " (−" + formatRupiah(l.Amount) + ")"
	}
	return " (" + formatRupiah(l.Amount) + ")"
}

func sameRef(a, b *int64) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

// positive treats 0 or below as "none" (a select that was cleared).
func positive(v *int64) *int64 {
	if v == nil || *v <= 0 {
		return nil
	}
	n := *v
	return &n
}

func missingKinds(rule domain.LinePackageRule, present map[int64]bool) []int64 {
	var out []int64
	for _, id := range rule.EventKindIDs {
		if !present[id] {
			out = append(out, id)
		}
	}
	return out
}

// draftsOf turns the stored lines into drafts (the single-line writes edit one
// of them and save the whole set through the same path).
func draftsOf(lines []domain.ProjectLine) []LineDraft {
	out := make([]LineDraft, 0, len(lines))
	for _, l := range lines {
		if l.Kind == domain.LineExtraGuests {
			continue
		}
		out = append(out, draftOf(l))
	}
	return out
}

func draftOf(l domain.ProjectLine) LineDraft {
	d := LineDraft{ID: l.ID, Kind: l.Kind, Description: l.Description, CategoryID: l.CategoryID, EventID: l.EventID,
		Quantity: l.Quantity, Unit: l.Unit, UnitPrice: l.UnitPrice, IsBonus: l.IsBonus, SupplierRef: l.SupplierRef}
	if l.ParentLineID != nil {
		d.ParentID = *l.ParentLineID
	}
	if l.Kind == domain.LineDiscount {
		d.Amount = l.Amount
	}
	return d
}

func applyLinePatch(d *LineDraft, p LinePatch) {
	if p.Description.Present() {
		d.Description = p.Description.Value()
	}
	if p.Unit.Present() {
		d.Unit = p.Unit.Value()
	}
	optRef := func(dst **int64, o Opt[int64]) {
		if !o.Present() {
			return
		}
		if o.IsNull() {
			*dst = nil
			return
		}
		v := o.Value()
		*dst = &v
	}
	optRef(&d.CategoryID, p.CategoryID)
	optRef(&d.EventID, p.EventID)
	optRef(&d.Quantity, p.Quantity)
	optRef(&d.UnitPrice, p.UnitPrice)
	optRef(&d.SupplierRef, p.SupplierRef)
	if p.Amount.Present() {
		d.Amount = p.Amount.Value()
	}
	if p.IsBonus.Present() {
		d.IsBonus = p.IsBonus.Value()
	}
}

func offerLines(views []LineView) []offerscontracts.Line {
	out := make([]offerscontracts.Line, len(views))
	for i, v := range views {
		out[i] = offerscontracts.Line{Kind: offerscontracts.LineKind(v.Kind), Amount: v.Amount}
	}
	return out
}

func eventGuests(events []domain.ProjectEvent) []offerscontracts.EventGuests {
	out := make([]offerscontracts.EventGuests, len(events))
	for i, e := range events {
		out[i] = offerscontracts.EventGuests{EventKindID: e.EventKindID, Guests: domain.EventGuestTotal(e)}
	}
	return out
}

func toOfferRule(r domain.LinePackageRule) offerscontracts.PackageRule {
	return offerscontracts.PackageRule{PackageID: r.PackageID, Name: r.Name, Price: r.Price,
		EventKindIDs: append([]int64(nil), r.EventKindIDs...), IncludedGuests: r.IncludedGuests,
		ExtraGuestStep: r.ExtraGuestStep, ExtraGuestStepPrice: r.ExtraGuestStepPrice}
}

func fromOfferRule(r offerscontracts.PackageRule) domain.LinePackageRule {
	return domain.LinePackageRule{PackageID: r.PackageID, Name: r.Name, Price: r.Price,
		EventKindIDs: append([]int64(nil), r.EventKindIDs...), IncludedGuests: r.IncludedGuests,
		ExtraGuestStep: r.ExtraGuestStep, ExtraGuestStepPrice: r.ExtraGuestStepPrice}
}

// --- what other services read ---

// PackageNames is, per project, the names of its packages in screen order.
func (s *LineService) PackageNames(ctx context.Context, tenantID int64, projectIDs []int64) map[int64][]string {
	if s == nil || len(projectIDs) == 0 {
		return map[int64][]string{}
	}
	names, err := s.repo.PackageNames(ctx, tenantID, projectIDs)
	if err != nil || names == nil {
		return map[int64][]string{}
	}
	return names
}

// CloneLines copies a project's live lines onto a new project (Duplicate,
// ADR-0014): the amounts and rule copies come along, events and suppliers are
// remapped to their copies (a supplier left behind leaves the line without
// one), the extra-guest charges are recomputed for the copied events and the
// contract cache is written. Runs inside the caller's transaction.
func (s *LineService) CloneLines(ctx context.Context, tenantID, sourceID, targetID, actorStaffID int64, events, suppliers map[int64]int64) error {
	if s == nil {
		return nil
	}
	lines, err := s.repo.List(ctx, tenantID, sourceID, false)
	if err != nil {
		return err
	}
	domain.SortLines(lines)
	ids := map[int64]int64{}
	remap := func(id *int64, m map[int64]int64) *int64 {
		if id == nil {
			return nil
		}
		if n, ok := m[*id]; ok {
			return &n
		}
		return nil
	}
	// Packages first, so every child finds its parent's copy.
	for pass := 0; pass < 2; pass++ {
		for _, l := range lines {
			if (pass == 0) != (l.ParentLineID == nil) {
				continue
			}
			clone := l
			clone.ID, clone.ProjectID, clone.DeletedAt = 0, targetID, nil
			clone.EventID = remap(l.EventID, events)
			clone.SupplierRef = remap(l.SupplierRef, suppliers)
			if l.ParentLineID != nil {
				clone.ParentLineID = remap(l.ParentLineID, ids)
				if clone.ParentLineID == nil {
					continue
				}
			}
			if l.PackageRule != nil {
				r := *l.PackageRule
				r.EventKindIDs = append([]int64(nil), l.PackageRule.EventKindIDs...)
				clone.PackageRule = &r
			}
			if err := s.repo.Insert(ctx, &clone); err != nil {
				return err
			}
			ids[l.ID] = clone.ID
		}
	}
	if s.offers == nil {
		return s.repo.RefreshContract(ctx, tenantID, targetID, true)
	}
	return s.recompute(ctx, tenantID, targetID, actorStaffID, true)
}
