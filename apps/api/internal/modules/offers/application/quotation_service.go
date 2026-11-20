package application

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"elproof/internal/modules/offers/domain"
	"elproof/internal/shared/apperror"
	"elproof/internal/shared/clock"
	"elproof/internal/shared/etag"
	"elproof/internal/shared/pagination"
)

// QuotationService runs quotations ("Penawaran", PLAN §5): the header, the
// content (events, suppliers, lines, terms), revisions, the status machine,
// copies and alternatives, and the PDF.
type QuotationService struct {
	repo      QuotationRepository
	tx        Transactor
	packages  PackageRepository
	kinds     EventKindRepository
	directory VendorDirectory
	profile   WorkspaceProfile
	staff     StaffNameResolver
	files     FileStore
	renderer  DocumentRenderer
	now       func() time.Time
}

func NewQuotationService(repo QuotationRepository, tx Transactor, packages PackageRepository, kinds EventKindRepository, directory VendorDirectory) *QuotationService {
	return &QuotationService{repo: repo, tx: tx, packages: packages, kinds: kinds, directory: directory, now: clock.Now}
}

// SetPorts completes the wiring in main.go: the workspace profile (platform),
// staff names (staff), the object storage, and the PDF renderer.
func (s *QuotationService) SetPorts(profile WorkspaceProfile, staff StaffNameResolver, files FileStore, renderer DocumentRenderer) {
	s.profile, s.staff, s.files, s.renderer = profile, staff, files, renderer
}

// SetClock replaces the business clock (tests).
func (s *QuotationService) SetClock(now func() time.Time) { s.now = now }

// Now is the business clock the screens count days against (DaysLeft).
func (s *QuotationService) Now() time.Time { return s.now() }

// Messages answered by the quotation service.
const (
	MsgQuotationNotFound   = "Penawaran tidak ditemukan"
	MsgQuotationInvalid    = "Data penawaran belum sesuai"
	MsgQuotationReadOnly   = "Penawaran ini tidak bisa diubah lagi. Hanya penawaran berstatus Draf yang bisa diubah."
	MsgCoupleNamesRequired = "Isi nama pasangan"
	MsgPICRequired         = "Pilih PIC penawaran"
	MsgPICNotFound         = "PIC tidak ditemukan"
	MsgPICSelfOnly         = "Anda hanya bisa memegang penawaran sendiri. Pilih diri Anda sebagai PIC."
	MsgWhatsAppInvalid     = "Nomor WhatsApp tidak valid"
	MsgRevisionNotFound    = "Revisi tidak ditemukan"
	MsgProfileUnavailable  = "Profil usaha belum tersedia"
	CodeQuotationReadOnly  = "quotation_read_only"
)

// Viewer is who asks: their staff id, whether their role reaches every
// quotation (otherwise only the ones they are PIC of: the `assigned` scope),
// and whether they may see agreed costs (finance.margin.view).
type Viewer struct {
	StaffID       int64
	AllQuotations bool
	CanSeeCost    bool
}

func (v Viewer) actor() *int64 {
	if v.StaffID <= 0 {
		return nil
	}
	id := v.StaffID
	return &id
}

// fieldErrs collects field problems in the order they were found, so the
// message is always the first one.
type fieldErrs struct {
	fields map[string][]string
	first  string
}

func newFieldErrs() *fieldErrs { return &fieldErrs{fields: map[string][]string{}} }

func (f *fieldErrs) add(field, msg string) {
	if f.first == "" {
		f.first = msg
	}
	f.fields[field] = append(f.fields[field], msg)
}

func (f *fieldErrs) err() error {
	if len(f.fields) == 0 {
		return nil
	}
	return apperror.Validation(f.first, f.fields)
}

func transitionErr(err error) error {
	var te *domain.TransitionError
	if errors.As(err, &te) {
		return apperror.Validation(te.Message, map[string][]string{"status": {te.Message}}).WithCode(te.Code)
	}
	return err
}

var whatsAppPattern = regexp.MustCompile(`^\+?[0-9][0-9 \-]{6,22}$`)

func (s *QuotationService) get(ctx context.Context, tenantID, id int64, v Viewer) (*domain.Quotation, error) {
	q, err := s.repo.FindByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	return visible(q, v)
}

// lock is get with the row locked for the rest of the transaction.
func (s *QuotationService) lock(ctx context.Context, tenantID, id int64, v Viewer) (*domain.Quotation, error) {
	q, err := s.repo.Lock(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	return visible(q, v)
}

// visible answers 404 for a missing quotation and for one outside the
// viewer's scope (another tenant's never reaches here: the repository scopes).
func visible(q *domain.Quotation, v Viewer) (*domain.Quotation, error) {
	if q == nil || (!v.AllQuotations && q.PICStaffID != v.StaffID) {
		return nil, apperror.NotFound(MsgQuotationNotFound)
	}
	return q, nil
}

func editable(q *domain.Quotation) error {
	if !q.Editable() {
		return apperror.Validation(MsgQuotationReadOnly, map[string][]string{"status": {MsgQuotationReadOnly}}).WithCode(CodeQuotationReadOnly)
	}
	return nil
}

// Get is one quotation of the tenant the viewer may see.
func (s *QuotationService) Get(ctx context.Context, tenantID, id int64, v Viewer) (*domain.Quotation, error) {
	return s.get(ctx, tenantID, id, v)
}

func (s *QuotationService) activity(ctx context.Context, tenantID, quotationID int64, v Viewer, t domain.ActivityType, summary string) error {
	if utf8.RuneCountInString(summary) > 500 {
		summary = string([]rune(summary)[:497]) + "..."
	}
	return s.repo.AddActivity(ctx, tenantID, domain.Activity{QuotationID: quotationID, Type: t, ActorStaffID: v.actor(), Summary: summary})
}

// ---- header ------------------------------------------------------------

// HeaderInput is the header form (create and "Info › Ubah").
type HeaderInput struct {
	CoupleNames     string
	ContactWhatsApp string
	PICStaffID      int64
	// ValidUntil and Notes are only read on update.
	ValidUntil *time.Time
	Notes      string
}

func (s *QuotationService) checkHeader(ctx context.Context, tenantID int64, v Viewer, in *HeaderInput, f *fieldErrs) error {
	in.CoupleNames = strings.TrimSpace(in.CoupleNames)
	switch {
	case in.CoupleNames == "":
		f.add("coupleNames", MsgCoupleNamesRequired)
	case utf8.RuneCountInString(in.CoupleNames) > domain.MaxCoupleNamesLen:
		f.add("coupleNames", fmt.Sprintf("Nama pasangan maksimal %d karakter", domain.MaxCoupleNamesLen))
	}
	in.ContactWhatsApp = strings.TrimSpace(in.ContactWhatsApp)
	if in.ContactWhatsApp != "" && !whatsAppPattern.MatchString(in.ContactWhatsApp) {
		f.add("contactWhatsapp", MsgWhatsAppInvalid)
	}
	in.Notes = strings.TrimSpace(in.Notes)
	if utf8.RuneCountInString(in.Notes) > domain.MaxNotesLen {
		f.add("notes", fmt.Sprintf("Catatan maksimal %d karakter", domain.MaxNotesLen))
	}
	if in.PICStaffID <= 0 {
		f.add("picStaffId", MsgPICRequired)
		return nil
	}
	if !v.AllQuotations && in.PICStaffID != v.StaffID {
		f.add("picStaffId", MsgPICSelfOnly)
		return nil
	}
	if s.staff == nil {
		return apperror.Internal("Daftar staf belum tersedia")
	}
	_, found, err := s.staff.GetName(ctx, tenantID, in.PICStaffID)
	if err != nil {
		return err
	}
	if !found {
		f.add("picStaffId", MsgPICNotFound)
	}
	return nil
}

func (s *QuotationService) quotationProfile(ctx context.Context, tenantID int64) (QuotationProfile, error) {
	if s.profile == nil {
		return QuotationProfile{}, apperror.Internal(MsgProfileUnavailable)
	}
	return s.profile.QuotationProfile(ctx, tenantID)
}

// insertNew numbers and inserts a new quotation inside the caller's
// transaction: the counter of today's WIB date moves under its row lock, so
// two quotations made at once never share a number, and two workspaces each
// start at 001.
func (s *QuotationService) insertNew(ctx context.Context, tenantID int64, q *domain.Quotation, p QuotationProfile) error {
	now := s.now()
	q.TenantID = tenantID
	q.DocDate = domain.TodayWIB(now)
	seq, err := s.repo.NextSequence(ctx, tenantID, q.DocDate)
	if err != nil {
		return err
	}
	number, err := domain.QuotationNumber(p.DocumentCode, q.DocDate, seq)
	if errors.Is(err, domain.ErrDailyQuotaReached) {
		return apperror.Validation(domain.MsgDailyQuotaReached, map[string][]string{"number": {domain.MsgDailyQuotaReached}}).WithCode("quotation_daily_limit")
	}
	if err != nil {
		return err
	}
	q.Number = number
	q.ValidUntil = domain.DefaultValidUntil(now, p.DefaultValidityDays)
	q.Status = domain.QuotationDraft
	q.IsPKP = p.IsPKP
	return s.repo.Create(ctx, q)
}

// Create makes a quotation from couple names, an optional WhatsApp number,
// and a PIC (DESAIN A1). The number is made at once; the terms start at
// 30/40/30 and the terms text from the workspace.
func (s *QuotationService) Create(ctx context.Context, tenantID int64, v Viewer, in HeaderInput) (*domain.Quotation, error) {
	f := newFieldErrs()
	if err := s.checkHeader(ctx, tenantID, v, &in, f); err != nil {
		return nil, err
	}
	if err := f.err(); err != nil {
		return nil, err
	}
	p, err := s.quotationProfile(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	q := &domain.Quotation{CoupleNames: in.CoupleNames, ContactWhatsApp: in.ContactWhatsApp, PICStaffID: in.PICStaffID,
		TermsText: p.DefaultTermsText, CreatedByStaffID: v.actor()}
	err = s.tx.InTx(ctx, func(ctx context.Context) error {
		if err := s.insertNew(ctx, tenantID, q, p); err != nil {
			return err
		}
		if err := s.repo.ReplaceTerms(ctx, tenantID, q.ID, domain.DefaultTerms()); err != nil {
			return err
		}
		return s.activity(ctx, tenantID, q.ID, v, domain.ActivityCreated, "Penawaran dibuat: "+q.Number)
	})
	if err != nil {
		return nil, err
	}
	return s.get(ctx, tenantID, q.ID, Viewer{AllQuotations: true})
}

// UpdateHeader saves "Info": couple names, contact, PIC, validity, notes.
func (s *QuotationService) UpdateHeader(ctx context.Context, tenantID, id int64, v Viewer, in HeaderInput, expected *time.Time) (*domain.Quotation, error) {
	err := s.tx.InTx(ctx, func(ctx context.Context) error {
		q, err := s.lock(ctx, tenantID, id, v)
		if err != nil {
			return err
		}
		if expected != nil && !etag.Same(*expected, q.UpdatedAt) {
			return apperror.Stale(q)
		}
		if err := editable(q); err != nil {
			return err
		}
		f := newFieldErrs()
		if err := s.checkHeader(ctx, tenantID, v, &in, f); err != nil {
			return err
		}
		extended := false
		if in.ValidUntil != nil {
			until := domain.CivilDate(*in.ValidUntil)
			if !until.Equal(domain.CivilDate(q.ValidUntil)) {
				if until.Before(domain.TodayWIB(s.now())) {
					f.add("validUntil", "Tanggal berlaku paling cepat hari ini")
				}
				extended = true
				q.ValidUntil = until
			}
		}
		if err := f.err(); err != nil {
			return err
		}
		q.CoupleNames, q.ContactWhatsApp, q.PICStaffID, q.Notes = in.CoupleNames, in.ContactWhatsApp, in.PICStaffID, in.Notes
		if err := s.repo.SaveHeader(ctx, q); err != nil {
			return err
		}
		if extended {
			if err := s.activity(ctx, tenantID, id, v, domain.ActivityExtended, "Masa berlaku diubah sampai "+longDate(q.ValidUntil)); err != nil {
				return err
			}
		}
		return s.activity(ctx, tenantID, id, v, domain.ActivityHeaderChanged, "Info penawaran diubah")
	})
	if err != nil {
		return nil, err
	}
	return s.get(ctx, tenantID, id, Viewer{AllQuotations: true})
}

// UpdateTermsText saves the terms & conditions text of one quotation.
func (s *QuotationService) UpdateTermsText(ctx context.Context, tenantID, id int64, v Viewer, text string, expected *time.Time) (*domain.Quotation, error) {
	text = strings.TrimSpace(text)
	if utf8.RuneCountInString(text) > domain.MaxTermsTextLen {
		msg := fmt.Sprintf("Syarat & ketentuan maksimal %d karakter", domain.MaxTermsTextLen)
		return nil, apperror.Validation(msg, map[string][]string{"termsText": {msg}})
	}
	err := s.tx.InTx(ctx, func(ctx context.Context) error {
		q, err := s.lock(ctx, tenantID, id, v)
		if err != nil {
			return err
		}
		if expected != nil && !etag.Same(*expected, q.TermsUpdatedAt) {
			return apperror.Stale(q)
		}
		if err := editable(q); err != nil {
			return err
		}
		if err := s.repo.SaveTermsText(ctx, tenantID, id, text); err != nil {
			return err
		}
		return s.activity(ctx, tenantID, id, v, domain.ActivityTermsChanged, "Syarat & ketentuan diubah")
	})
	if err != nil {
		return nil, err
	}
	return s.get(ctx, tenantID, id, Viewer{AllQuotations: true})
}

// ---- list --------------------------------------------------------------

// QuotationListRow is one group of the list: the first quotation and its
// alternatives.
type QuotationListRow struct {
	Quotation    domain.Quotation
	Alternatives []domain.Quotation
}

func (s *QuotationService) ListSortSpec() (map[string]pagination.SortColumn, pagination.Sort) {
	return s.repo.ListSortSpec()
}

func (s *QuotationService) scoped(filter QuotationListFilter, v Viewer) QuotationListFilter {
	filter.Today = domain.TodayWIB(s.now())
	if !v.AllQuotations {
		id := v.StaffID
		filter.OnlyPICStaffID = &id
	}
	return filter
}

// List is a page of groups (sorted, filtered, and paged on the server).
func (s *QuotationService) List(ctx context.Context, tenantID int64, v Viewer, params pagination.Params, filter QuotationListFilter, sort pagination.Sort) ([]QuotationListRow, int64, error) {
	filter = s.scoped(filter, v)
	roots, total, err := s.repo.ListPaginated(ctx, tenantID, params, filter, sort)
	if err != nil {
		return nil, 0, err
	}
	ids := make([]int64, len(roots))
	for i, q := range roots {
		ids[i] = q.ID
	}
	alts, err := s.repo.Alternatives(ctx, tenantID, ids)
	if err != nil {
		return nil, 0, err
	}
	byRoot := map[int64][]domain.Quotation{}
	for _, a := range alts {
		if !v.AllQuotations && a.PICStaffID != v.StaffID {
			continue
		}
		byRoot[*a.AlternativeOfID] = append(byRoot[*a.AlternativeOfID], a)
	}
	out := make([]QuotationListRow, len(roots))
	for i, q := range roots {
		out[i] = QuotationListRow{Quotation: q, Alternatives: byRoot[q.ID]}
	}
	return out, total, nil
}

// Summary is how many groups each tab holds (badges of the list tabs).
func (s *QuotationService) Summary(ctx context.Context, tenantID int64, v Viewer, filter QuotationListFilter) (map[domain.QuotationStatus]int, error) {
	return s.repo.StatusCounts(ctx, tenantID, s.scoped(filter, v))
}

// StaffNames resolves PIC and activity names.
func (s *QuotationService) StaffNames(ctx context.Context, tenantID int64, ids []int64) (map[int64]string, error) {
	if s.staff == nil || len(ids) == 0 {
		return map[int64]string{}, nil
	}
	return s.staff.StaffNames(ctx, tenantID, uniqueIDs(ids))
}

// ---- status machine ----------------------------------------------------

// StatusInput is a status change (setQuotationStatus). RevisionN, AcceptedOn,
// Channel, and ExtendUntil are read for Accepted; Reason for Rejected.
type StatusInput struct {
	Status      domain.QuotationStatus
	RevisionN   *int
	AcceptedOn  *time.Time
	Channel     domain.AcceptChannel
	ExtendUntil *time.Time
	Reason      string
}

// SetStatus applies one explicit status change (PLAN §5.3), with everything
// it implies in the same transaction: an expired quotation extended and then
// accepted (two activities), the other Drafts of its group superseded.
func (s *QuotationService) SetStatus(ctx context.Context, tenantID, id int64, v Viewer, in StatusInput) (*domain.Quotation, error) {
	err := s.tx.InTx(ctx, func(ctx context.Context) error {
		q, err := s.lock(ctx, tenantID, id, v)
		if err != nil {
			return err
		}
		now := s.now()
		check := domain.AcceptCheck{ExtendUntil: in.ExtendUntil}
		var rev *domain.QuotationRevision
		if in.Status == domain.QuotationAccepted && q.Status == domain.QuotationDraft {
			if rev, check.RevisionMatchesWorkingCopy, err = s.acceptedRevision(ctx, tenantID, q, in.RevisionN); err != nil {
				return err
			}
			f := newFieldErrs()
			if in.Channel == "" || !domain.ValidChannel(in.Channel) {
				f.add("channel", "Pilih lewat apa penawaran diterima")
			}
			if in.AcceptedOn != nil && domain.CivilDate(*in.AcceptedOn).After(domain.TodayWIB(now)) {
				f.add("acceptedOn", "Tanggal diterima tidak boleh setelah hari ini")
			}
			if err := f.err(); err != nil {
				return err
			}
		}
		if in.Status == domain.QuotationRejected && utf8.RuneCountInString(strings.TrimSpace(in.Reason)) > 500 {
			return apperror.Validation("Alasan maksimal 500 karakter", map[string][]string{"reason": {"Alasan maksimal 500 karakter"}})
		}
		t, err := domain.PlanTransition(*q, in.Status, now, check)
		if err != nil {
			return transitionErr(err)
		}
		switch t {
		case domain.TransitionNone:
			return nil
		case domain.TransitionAccept:
			return s.accept(ctx, tenantID, q, rev, in, v)
		case domain.TransitionReject:
			q.Status = domain.QuotationRejected
			reason := strings.TrimSpace(in.Reason)
			q.RejectedReason = nil
			if reason != "" {
				q.RejectedReason = &reason
			}
			at := now
			q.RejectedAt = &at
			if err := s.repo.SaveState(ctx, q); err != nil {
				return err
			}
			summary := "Penawaran ditolak"
			if reason != "" {
				summary += ": " + reason
			}
			return s.activity(ctx, tenantID, id, v, domain.ActivityRejected, summary)
		case domain.TransitionReopen:
			q.Status, q.RejectedReason, q.RejectedAt = domain.QuotationDraft, nil, nil
			if err := s.repo.SaveState(ctx, q); err != nil {
				return err
			}
			return s.activity(ctx, tenantID, id, v, domain.ActivityReopened, "Penawaran dibuka lagi")
		case domain.TransitionUnaccept:
			n := q.AcceptedRevisionN
			q.Status = domain.QuotationDraft
			q.AcceptedRevisionID, q.AcceptedRevisionN, q.AcceptedOn, q.AcceptedChannel = nil, 0, nil, nil
			q.AcceptedAttachmentKey, q.AcceptedAttachmentName, q.AcceptedAttachmentType = nil, nil, nil
			if err := s.repo.SaveState(ctx, q); err != nil {
				return err
			}
			return s.activity(ctx, tenantID, id, v, domain.ActivityUnaccepted, fmt.Sprintf("Penerimaan Revisi %d dibatalkan", n))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.get(ctx, tenantID, id, Viewer{AllQuotations: true})
}

// acceptedRevision is the revision an acceptance names (the one equal to the
// working copy when none is named) and whether it equals the working copy.
func (s *QuotationService) acceptedRevision(ctx context.Context, tenantID int64, q *domain.Quotation, n *int) (*domain.QuotationRevision, bool, error) {
	w, err := s.workingCopy(ctx, tenantID, q)
	if err != nil {
		return nil, false, err
	}
	content, err := domain.BuildContent(*w)
	if err != nil {
		return nil, false, linesErr(err, 0)
	}
	hash := content.Hash()
	if n == nil {
		rev, err := s.repo.FindRevisionByHash(ctx, tenantID, q.ID, hash)
		return rev, rev != nil, err
	}
	rev, err := s.repo.FindRevision(ctx, tenantID, q.ID, *n)
	if err != nil {
		return nil, false, err
	}
	if rev == nil {
		return nil, false, apperror.Validation(MsgRevisionNotFound, map[string][]string{"revisionN": {MsgRevisionNotFound}})
	}
	return rev, rev.ContentHash == hash, nil
}

func (s *QuotationService) accept(ctx context.Context, tenantID int64, q *domain.Quotation, rev *domain.QuotationRevision, in StatusInput, v Viewer) error {
	now := s.now()
	if q.Expired(now) && in.ExtendUntil != nil {
		q.ValidUntil = domain.CivilDate(*in.ExtendUntil)
		if err := s.activity(ctx, tenantID, q.ID, v, domain.ActivityExtended, "Masa berlaku diperpanjang sampai "+longDate(q.ValidUntil)); err != nil {
			return err
		}
	}
	on := domain.TodayWIB(now)
	if in.AcceptedOn != nil {
		on = domain.CivilDate(*in.AcceptedOn)
	}
	ch := in.Channel
	q.Status = domain.QuotationAccepted
	q.AcceptedRevisionID, q.AcceptedRevisionN, q.AcceptedOn, q.AcceptedChannel = &rev.ID, rev.N, &on, &ch
	if err := s.repo.SaveState(ctx, q); err != nil {
		return err
	}
	if err := s.activity(ctx, tenantID, q.ID, v, domain.ActivityAccepted,
		fmt.Sprintf("Revisi %d ditandai diterima lewat %s", rev.N, channelWord(ch))); err != nil {
		return err
	}
	group, err := s.group(ctx, tenantID, q)
	if err != nil {
		return err
	}
	for _, otherID := range domain.SupersededByAcceptance(*q, group) {
		other, err := s.repo.Lock(ctx, tenantID, otherID)
		if err != nil {
			return err
		}
		if other == nil || other.Status != domain.QuotationDraft {
			continue
		}
		other.Status = domain.QuotationSuperseded
		by := q.ID
		other.SupersededByID = &by
		if err := s.repo.SaveState(ctx, other); err != nil {
			return err
		}
		if err := s.activity(ctx, tenantID, other.ID, v, domain.ActivitySuperseded, "Digantikan oleh alternatif "+q.Number+" yang diterima"); err != nil {
			return err
		}
	}
	return nil
}

// group is every live member of q's group of alternatives (q included).
func (s *QuotationService) group(ctx context.Context, tenantID int64, q *domain.Quotation) ([]domain.Quotation, error) {
	rootID := q.GroupRootID()
	alts, err := s.repo.Alternatives(ctx, tenantID, []int64{rootID})
	if err != nil {
		return nil, err
	}
	out := alts
	if rootID == q.ID {
		out = append(out, *q)
	} else if root, err := s.repo.FindByID(ctx, tenantID, rootID); err != nil {
		return nil, err
	} else if root != nil {
		out = append(out, *root)
	}
	return out, nil
}

// GroupMembers is the other members of q's group, for the "Tandai diterima"
// dialog ("…002 akan menjadi Digantikan") and the alternatives links.
func (s *QuotationService) GroupMembers(ctx context.Context, tenantID int64, q *domain.Quotation) ([]domain.Quotation, error) {
	all, err := s.group(ctx, tenantID, q)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Quotation, 0, len(all))
	for _, g := range all {
		if g.ID != q.ID {
			out = append(out, g)
		}
	}
	return out, nil
}

// Extend moves the validity of a Draft ("Perpanjang masa berlaku").
func (s *QuotationService) Extend(ctx context.Context, tenantID, id int64, v Viewer, until time.Time) (*domain.Quotation, error) {
	err := s.tx.InTx(ctx, func(ctx context.Context) error {
		q, err := s.lock(ctx, tenantID, id, v)
		if err != nil {
			return err
		}
		if err := domain.CanExtend(*q, until, s.now()); err != nil {
			var te *domain.TransitionError
			if errors.As(err, &te) {
				return apperror.Validation(te.Message, map[string][]string{"validUntil": {te.Message}}).WithCode(te.Code)
			}
			return err
		}
		q.ValidUntil = domain.CivilDate(until)
		if err := s.repo.SaveState(ctx, q); err != nil {
			return err
		}
		return s.activity(ctx, tenantID, id, v, domain.ActivityExtended, "Masa berlaku diperpanjang sampai "+longDate(q.ValidUntil))
	})
	if err != nil {
		return nil, err
	}
	return s.get(ctx, tenantID, id, Viewer{AllQuotations: true})
}

// DeleteDraft removes a Draft without revisions ("Hapus draf"). Its number
// stays used: numbers are never given twice.
func (s *QuotationService) DeleteDraft(ctx context.Context, tenantID, id int64, v Viewer) error {
	return s.tx.InTx(ctx, func(ctx context.Context) error {
		q, err := s.lock(ctx, tenantID, id, v)
		if err != nil {
			return err
		}
		alts, err := s.repo.Alternatives(ctx, tenantID, []int64{q.ID})
		if err != nil {
			return err
		}
		if err := domain.CanDeleteDraft(*q, len(alts)); err != nil {
			return transitionErr(err)
		}
		return s.repo.SoftDelete(ctx, tenantID, id)
	})
}

// ---- copy and alternatives ---------------------------------------------

// Copy is "Salin jadi penawaran baru": a new quotation with a new number and
// the same content. A Draft or Accepted original becomes Superseded in the
// same transaction; a Rejected one stays Rejected.
func (s *QuotationService) Copy(ctx context.Context, tenantID, id int64, v Viewer) (*domain.Quotation, error) {
	return s.duplicate(ctx, tenantID, id, v, false)
}

// AddAlternative is "Buat alternatif": a new quotation with its own number,
// the same content, in the group of the original. Every member stays a Draft.
func (s *QuotationService) AddAlternative(ctx context.Context, tenantID, id int64, v Viewer) (*domain.Quotation, error) {
	return s.duplicate(ctx, tenantID, id, v, true)
}

func (s *QuotationService) duplicate(ctx context.Context, tenantID, id int64, v Viewer, alternative bool) (*domain.Quotation, error) {
	p, err := s.quotationProfile(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	var newID int64
	err = s.tx.InTx(ctx, func(ctx context.Context) error {
		orig, err := s.lock(ctx, tenantID, id, v)
		if err != nil {
			return err
		}
		if alternative {
			err = domain.CanAddAlternative(*orig)
		} else {
			err = domain.CanCopy(*orig)
		}
		if err != nil {
			return transitionErr(err)
		}
		w, err := s.workingCopy(ctx, tenantID, orig)
		if err != nil {
			return err
		}
		q := &domain.Quotation{CoupleNames: orig.CoupleNames, ContactWhatsApp: orig.ContactWhatsApp, PICStaffID: orig.PICStaffID,
			Notes: orig.Notes, TermsText: orig.TermsText, CreatedByStaffID: v.actor()}
		if alternative {
			root := orig.GroupRootID()
			q.AlternativeOfID = &root
		}
		if err := s.insertNew(ctx, tenantID, q, p); err != nil {
			return err
		}
		if err := s.copyContent(ctx, tenantID, q, *w); err != nil {
			return err
		}
		newID = q.ID
		if alternative {
			if err := s.activity(ctx, tenantID, q.ID, v, domain.ActivityCreated, "Penawaran dibuat sebagai alternatif dari "+orig.Number); err != nil {
				return err
			}
			return s.activity(ctx, tenantID, orig.ID, v, domain.ActivityAlternativeAdded, "Alternatif dibuat: "+q.Number)
		}
		if err := s.activity(ctx, tenantID, q.ID, v, domain.ActivityCreated, "Penawaran dibuat sebagai salinan dari "+orig.Number); err != nil {
			return err
		}
		if err := s.activity(ctx, tenantID, orig.ID, v, domain.ActivityCopied, "Disalin jadi penawaran baru "+q.Number); err != nil {
			return err
		}
		if !domain.CopySupersedesOriginal(*orig) {
			return nil
		}
		orig.Status = domain.QuotationSuperseded
		by := q.ID
		orig.SupersededByID = &by
		if err := s.repo.SaveState(ctx, orig); err != nil {
			return err
		}
		return s.activity(ctx, tenantID, orig.ID, v, domain.ActivitySuperseded, "Digantikan oleh "+q.Number)
	})
	if err != nil {
		return nil, err
	}
	return s.get(ctx, tenantID, newID, Viewer{AllQuotations: true})
}

// copyContent writes w's events, suppliers, lines, and terms into q (a new
// quotation), mapping every reference to the new rows.
func (s *QuotationService) copyContent(ctx context.Context, tenantID int64, q *domain.Quotation, w domain.WorkingCopy) error {
	supMap := map[int64]int64{}
	for _, sup := range w.Suppliers {
		c := sup
		c.ID, c.QuotationID = 0, q.ID
		if err := s.repo.InsertSupplier(ctx, tenantID, &c); err != nil {
			return err
		}
		supMap[sup.ID] = c.ID
	}
	evMap := map[int64]int64{}
	for _, e := range w.Events {
		c := e
		c.ID, c.QuotationID = 0, q.ID
		c.Sessions = append([]domain.QuotationSession(nil), e.Sessions...)
		if e.VenueSupplierID != nil {
			id := supMap[*e.VenueSupplierID]
			c.VenueSupplierID = &id
		}
		if err := s.repo.InsertEvent(ctx, tenantID, &c); err != nil {
			return err
		}
		evMap[e.ID] = c.ID
	}
	ordered := domain.OrderedLines(w.Lines)
	pos := map[int64]int{}
	rows := make([]LineRow, len(ordered))
	for i, l := range ordered {
		pos[l.ID] = i
		c := l
		c.ID, c.QuotationID, c.ParentLineID = 0, q.ID, nil
		c.EventID = mapped(l.EventID, evMap)
		c.SupplierRef = mapped(l.SupplierRef, supMap)
		parent := -1
		if l.ParentLineID != nil {
			if p, ok := pos[*l.ParentLineID]; ok {
				parent = p
			}
		}
		rows[i] = LineRow{Line: c, Parent: parent}
	}
	if err := s.repo.ReplaceLines(ctx, tenantID, q.ID, rows); err != nil {
		return err
	}
	terms := make([]domain.QuotationTerm, len(w.Terms))
	for i, t := range w.Terms {
		c := t
		c.ID, c.QuotationID = 0, q.ID
		c.DueEventID = mapped(t.DueEventID, evMap)
		terms[i] = c
	}
	if err := s.repo.ReplaceTerms(ctx, tenantID, q.ID, terms); err != nil {
		return err
	}
	totals, err := domain.TotalsOf(w.Lines, q.IsPKP)
	if err != nil {
		totals = domain.Totals{}
	}
	return s.repo.SetCaches(ctx, tenantID, q.ID, totals.Total, domain.MainEventDate(w.Events), true, true)
}

func mapped(id *int64, m map[int64]int64) *int64 {
	if id == nil {
		return nil
	}
	if v, ok := m[*id]; ok {
		return &v
	}
	return nil
}

// workingCopy loads q's content.
func (s *QuotationService) workingCopy(ctx context.Context, tenantID int64, q *domain.Quotation) (*domain.WorkingCopy, error) {
	w, err := s.repo.LoadContent(ctx, tenantID, q.ID)
	if err != nil {
		return nil, err
	}
	w.Quotation = *q
	return w, nil
}

// refreshCaches recomputes the extra guests lines and the list's caches after
// the content changed, inside the caller's transaction.
func (s *QuotationService) refreshCaches(ctx context.Context, tenantID int64, q *domain.Quotation, bumpLines, bumpTerms bool) error {
	w, err := s.workingCopy(ctx, tenantID, q)
	if err != nil {
		return err
	}
	before := map[int64]int64{}
	for _, l := range w.Lines {
		before[l.ID] = l.Amount
	}
	if domain.RecomputeExtraGuests(w.Lines, w.Events) {
		amounts := map[int64]int64{}
		for _, l := range w.Lines {
			if before[l.ID] != l.Amount {
				amounts[l.ID] = l.Amount
			}
		}
		if err := s.repo.SetLineAmounts(ctx, tenantID, q.ID, amounts); err != nil {
			return err
		}
	}
	totals, err := domain.TotalsOf(w.Lines, q.IsPKP)
	if err != nil {
		// More extra guests can never make discounts exceed the subtotal, and
		// fewer is refused before it is written; keep the old total if it does.
		totals.Total = q.ContractTotal
	}
	return s.repo.SetCaches(ctx, tenantID, q.ID, totals.Total, domain.MainEventDate(w.Events), bumpLines, bumpTerms)
}

// ---- activity ------------------------------------------------------------

// Activity is the quotation's log, newest first.
func (s *QuotationService) Activity(ctx context.Context, tenantID, id int64, v Viewer) ([]domain.Activity, error) {
	if _, err := s.get(ctx, tenantID, id, v); err != nil {
		return nil, err
	}
	return s.repo.ListActivity(ctx, tenantID, id)
}

func channelWord(c domain.AcceptChannel) string {
	switch c {
	case domain.ChannelWhatsApp:
		return "WhatsApp"
	case domain.ChannelInPerson:
		return "tatap muka"
	case domain.ChannelEmail:
		return "email"
	}
	return "saluran lain"
}
