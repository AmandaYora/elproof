package domain

import (
	"errors"
	"fmt"
	"regexp"
	"time"
)

// WIB is the business time zone: a quotation's number date, its validity, and
// "today" are all WIB days (PLAN §5.1). A fixed zone keeps it working without
// tzdata.
var WIB = time.FixedZone("WIB", 7*60*60)

// CivilDate keeps only the calendar day of t (as written in t's own zone), at
// midnight UTC, so two dates compare as days whatever zone the driver used.
func CivilDate(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// TodayWIB is the WIB calendar day of now.
func TodayWIB(now time.Time) time.Time { return CivilDate(now.In(WIB)) }

// QuotationStatus is the stored status of a quotation (PLAN §5.3).
// "Kedaluwarsa" is not stored: it is a Draft past its validity, computed on
// read (see Quotation.Expired).
type QuotationStatus string

const (
	QuotationDraft      QuotationStatus = "Draft"
	QuotationAccepted   QuotationStatus = "Accepted"
	QuotationRejected   QuotationStatus = "Rejected"
	QuotationSuperseded QuotationStatus = "Superseded"
	// QuotationExpired is only ever a display status (list tab, badge).
	QuotationExpired QuotationStatus = "Expired"
)

// ValidStoredStatus reports whether s is one of the four stored statuses.
func ValidStoredStatus(s QuotationStatus) bool {
	switch s {
	case QuotationDraft, QuotationAccepted, QuotationRejected, QuotationSuperseded:
		return true
	}
	return false
}

// AcceptChannel is how the client said yes (DESAIN §6.5).
type AcceptChannel string

const (
	ChannelWhatsApp AcceptChannel = "WhatsApp"
	ChannelInPerson AcceptChannel = "InPerson"
	ChannelEmail    AcceptChannel = "Email"
	ChannelOther    AcceptChannel = "Other"
)

// ValidChannel reports whether c is one of the four channels.
func ValidChannel(c AcceptChannel) bool {
	switch c {
	case ChannelWhatsApp, ChannelInPerson, ChannelEmail, ChannelOther:
		return true
	}
	return false
}

// Quotation is the header of a quotation ("Penawaran"): who it is for, who
// handles it, how long it is valid, its status, and the caches the list shows.
type Quotation struct {
	ID       int64
	TenantID int64
	// Number is KODE-PNW-YYMMDD-NNN, frozen when the quotation is created.
	Number string
	// DocDate is the WIB day of the number.
	DocDate         time.Time
	CoupleNames     string
	ContactWhatsApp string
	PICStaffID      int64
	// ValidUntil is a day: the quotation is valid through its end (WIB).
	ValidUntil time.Time
	Notes      string
	Status     QuotationStatus
	// AlternativeOfID is the first quotation of the group of alternatives this
	// one belongs to (nil for that first one, or a quotation without any).
	AlternativeOfID *int64
	// SupersededByID is the quotation that replaced this one.
	SupersededByID     *int64
	AcceptedRevisionID *int64
	// AcceptedRevisionN is the number of the accepted revision (0 = none).
	AcceptedRevisionN      int
	AcceptedOn             *time.Time
	AcceptedChannel        *AcceptChannel
	AcceptedAttachmentKey  *string
	AcceptedAttachmentName *string
	AcceptedAttachmentType *string
	RejectedReason         *string
	RejectedAt             *time.Time
	// ConvertedProjectID is a cache written by projects through the offers
	// contract (MarkConverted); the project is the truth.
	ConvertedProjectID *int64
	ConvertedAt        *time.Time
	// ProjectDeletedAt is set when that project was deleted for good.
	ProjectDeletedAt *time.Time
	IsPKP            bool
	TermsText        string
	// RevisionCount is the highest revision number so far (0 = none yet).
	RevisionCount    int
	ContractTotal    int64
	MainEventDate    *time.Time
	CreatedByStaffID *int64
	CreatedAt        time.Time
	// UpdatedAt is the header's version (If-Match), LinesUpdatedAt the lines'
	// and TermsUpdatedAt the terms' (and terms text's).
	UpdatedAt      time.Time
	LinesUpdatedAt time.Time
	TermsUpdatedAt time.Time
	// ChangedAt is the last change of anything ("terakhir diubah").
	ChangedAt time.Time
}

// Expired reports whether the quotation reads as Kedaluwarsa: a Draft whose
// last valid WIB day is before today.
func (q Quotation) Expired(now time.Time) bool {
	return q.Status == QuotationDraft && TodayWIB(now).After(CivilDate(q.ValidUntil))
}

// DisplayStatus is the status the screens show: Expired for a Draft past its
// validity, otherwise the stored one.
func (q Quotation) DisplayStatus(now time.Time) QuotationStatus {
	if q.Expired(now) {
		return QuotationExpired
	}
	return q.Status
}

// DaysLeft is how many WIB days remain until ValidUntil (0 on the last day,
// negative once it has passed): the "H-n" of the list.
func (q Quotation) DaysLeft(now time.Time) int {
	return int(CivilDate(q.ValidUntil).Sub(TodayWIB(now)).Hours() / 24)
}

// Converted reports whether a project was made from the quotation.
func (q Quotation) Converted() bool { return q.ConvertedProjectID != nil }

// Editable reports whether the working copy may change: only a Draft (an
// expired one too: it may be revised before it is extended).
func (q Quotation) Editable() bool { return q.Status == QuotationDraft && !q.Converted() }

// GroupRootID is the first quotation of q's group of alternatives (q itself
// when it is not an alternative).
func (q Quotation) GroupRootID() int64 {
	if q.AlternativeOfID != nil {
		return *q.AlternativeOfID
	}
	return q.ID
}

// ---- number -----------------------------------------------------------

// DocumentTypeQuotation is the document type in a quotation number.
const DocumentTypeQuotation = "PNW"

// MaxQuotationsPerDay is the last sequence a day can give (NNN).
const MaxQuotationsPerDay = 999

// ErrDailyQuotaReached is returned for the 1,000th quotation of a WIB day.
var ErrDailyQuotaReached = errors.New("quotation number: daily sequence exhausted")

// MsgDailyQuotaReached is what the WO reads when that happens.
const MsgDailyQuotaReached = "Penawaran hari ini sudah mencapai 999 nomor. Buat penawaran berikutnya besok."

var documentCodePattern = regexp.MustCompile(`^[A-Z0-9]{2,5}$`)

// QuotationNumber formats KODE-PNW-YYMMDD-NNN (PLAN §5.1) for the WIB day of
// docDate and the day's sequence (1..999). A code outside 2–5 letters or
// digits falls back to "WO" so a number can always be made.
func QuotationNumber(documentCode string, docDate time.Time, seq int) (string, error) {
	if seq < 1 || seq > MaxQuotationsPerDay {
		return "", ErrDailyQuotaReached
	}
	code := documentCode
	if !documentCodePattern.MatchString(code) {
		code = "WO"
	}
	return fmt.Sprintf("%s-%s-%s-%03d", code, DocumentTypeQuotation, CivilDate(docDate).Format("060102"), seq), nil
}

// DefaultValidUntil is the day a new quotation stays valid through: the day it
// is made plus the workspace's default validity.
func DefaultValidUntil(now time.Time, validityDays int) time.Time {
	if validityDays < 1 {
		validityDays = 14
	}
	return TodayWIB(now).AddDate(0, 0, validityDays)
}

// ---- status machine -----------------------------------------------------

// TransitionError is a refused status change: a message for the WO and a
// stable code the web can react to.
type TransitionError struct {
	Code    string
	Message string
}

func (e *TransitionError) Error() string { return e.Message }

// Codes of refused transitions.
const (
	CodeNotDraft          = "quotation_not_draft"
	CodeConverted         = "quotation_converted"
	CodeSuperseded        = "quotation_superseded"
	CodeNeedsExtension    = "quotation_expired_needs_extension"
	CodeNeedsRevision     = "quotation_needs_revision"
	CodeAutomaticOnly     = "quotation_status_automatic"
	CodeInvalidTransition = "quotation_invalid_transition"
)

// Transition is the change a status request makes.
type Transition string

const (
	TransitionNone     Transition = ""         // already in that status
	TransitionAccept   Transition = "accept"   // Draft → Accepted
	TransitionReject   Transition = "reject"   // Draft → Rejected
	TransitionReopen   Transition = "reopen"   // Rejected → Draft
	TransitionUnaccept Transition = "unaccept" // Accepted → Draft, only before conversion
)

// AcceptCheck is what Draft → Accepted needs besides the status: the
// revision chosen equals the working copy, and an expired quotation comes
// with a new validity in the same request.
type AcceptCheck struct {
	RevisionMatchesWorkingCopy bool
	ExtendUntil                *time.Time
}

// PlanTransition decides the change from q's status to target (PLAN §5.3).
// Superseded is never asked for: it follows a copy or an accepted alternative.
// accept is only read for target Accepted.
func PlanTransition(q Quotation, target QuotationStatus, now time.Time, accept AcceptCheck) (Transition, error) {
	if !ValidStoredStatus(target) {
		return TransitionNone, &TransitionError{CodeInvalidTransition, "Status penawaran tidak dikenal"}
	}
	if target == QuotationSuperseded {
		return TransitionNone, &TransitionError{CodeAutomaticOnly,
			"Status Digantikan terjadi otomatis saat penawaran disalin jadi penawaran baru atau alternatifnya diterima"}
	}
	if q.Converted() {
		return TransitionNone, &TransitionError{CodeConverted, "Penawaran ini sudah dijadikan proyek dan tidak bisa diubah lagi"}
	}
	if q.Status == QuotationSuperseded {
		return TransitionNone, &TransitionError{CodeSuperseded, "Penawaran ini sudah digantikan dan tidak bisa diubah lagi"}
	}
	if q.Status == target {
		return TransitionNone, nil
	}
	switch target {
	case QuotationAccepted:
		if q.Status != QuotationDraft {
			return TransitionNone, &TransitionError{CodeNotDraft, "Hanya penawaran berstatus Draf yang bisa ditandai diterima"}
		}
		if !accept.RevisionMatchesWorkingCopy {
			return TransitionNone, &TransitionError{CodeNeedsRevision,
				"Ada perubahan yang belum dikirim ke klien. Buat revisi untuk klien dulu, lalu tandai diterima."}
		}
		if q.Expired(now) {
			if accept.ExtendUntil == nil {
				return TransitionNone, &TransitionError{CodeNeedsExtension,
					"Masa berlaku penawaran ini sudah lewat. Isi tanggal perpanjangan untuk menandainya diterima."}
			}
			if CivilDate(*accept.ExtendUntil).Before(TodayWIB(now)) {
				return TransitionNone, &TransitionError{CodeNeedsExtension, "Tanggal perpanjangan paling cepat hari ini"}
			}
		}
		return TransitionAccept, nil
	case QuotationRejected:
		if q.Status != QuotationDraft {
			return TransitionNone, &TransitionError{CodeNotDraft, "Hanya penawaran berstatus Draf yang bisa ditolak"}
		}
		return TransitionReject, nil
	case QuotationDraft:
		switch q.Status {
		case QuotationRejected:
			return TransitionReopen, nil
		case QuotationAccepted:
			return TransitionUnaccept, nil
		}
	}
	return TransitionNone, &TransitionError{CodeInvalidTransition, "Perubahan status ini tidak bisa dilakukan"}
}

// CanCopy reports whether "Salin jadi penawaran baru" may start from q: not
// from a superseded quotation or one already made into a project.
func CanCopy(q Quotation) error {
	if q.Converted() {
		return &TransitionError{CodeConverted, "Penawaran ini sudah dijadikan proyek. Perubahan berikutnya dilakukan di proyek."}
	}
	if q.Status == QuotationSuperseded {
		return &TransitionError{CodeSuperseded, "Penawaran ini sudah digantikan. Salin dari penawaran penggantinya."}
	}
	return nil
}

// CopySupersedesOriginal reports whether copying q marks q Superseded: a
// Draft or an Accepted quotation is replaced by its copy; a Rejected one stays
// Rejected (the client said no to it, which is worth keeping).
func CopySupersedesOriginal(q Quotation) bool {
	return q.Status == QuotationDraft || q.Status == QuotationAccepted
}

// CanAddAlternative reports whether "Buat alternatif" may start from q: only
// a Draft (every member of a group stays a Draft until one is accepted).
func CanAddAlternative(q Quotation) error {
	if q.Status != QuotationDraft || q.Converted() {
		return &TransitionError{CodeNotDraft, "Alternatif hanya bisa dibuat dari penawaran berstatus Draf"}
	}
	return nil
}

// SupersededByAcceptance is which members of an accepted quotation's group
// become Superseded: every other member that is still a Draft (expired ones
// included). Rejected members stay Rejected.
func SupersededByAcceptance(accepted Quotation, group []Quotation) []int64 {
	var out []int64
	for _, g := range group {
		if g.ID != accepted.ID && g.Status == QuotationDraft {
			out = append(out, g.ID)
		}
	}
	return out
}

// CanExtend reports whether the validity of q may move: only a Draft.
func CanExtend(q Quotation, until time.Time, now time.Time) error {
	if q.Status != QuotationDraft || q.Converted() {
		return &TransitionError{CodeNotDraft, "Masa berlaku hanya bisa diubah pada penawaran berstatus Draf"}
	}
	if CivilDate(until).Before(TodayWIB(now)) {
		return &TransitionError{CodeInvalidTransition, "Tanggal berlaku paling cepat hari ini"}
	}
	return nil
}

// CanDeleteDraft reports whether "Hapus draf" may remove q: a Draft without
// any revision and without alternatives of its own.
func CanDeleteDraft(q Quotation, alternatives int) error {
	if q.Status != QuotationDraft || q.RevisionCount > 0 {
		return &TransitionError{CodeNotDraft, "Hanya draf yang belum punya revisi yang bisa dihapus"}
	}
	if alternatives > 0 {
		return &TransitionError{CodeInvalidTransition, "Penawaran ini punya alternatif. Hapus atau tolak alternatifnya dulu."}
	}
	return nil
}

// ---- activity -----------------------------------------------------------

// ActivityType names one entry of a quotation's activity log.
type ActivityType string

const (
	ActivityCreated          ActivityType = "created"
	ActivityHeaderChanged    ActivityType = "header_changed"
	ActivityEventsChanged    ActivityType = "events_changed"
	ActivitySuppliersChanged ActivityType = "suppliers_changed"
	ActivityLinesChanged     ActivityType = "lines_changed"
	ActivityTermsChanged     ActivityType = "terms_changed"
	ActivityRevisionCreated  ActivityType = "revision_created"
	ActivityShared           ActivityType = "shared"
	ActivityExtended         ActivityType = "extended"
	ActivityAccepted         ActivityType = "accepted"
	ActivityUnaccepted       ActivityType = "unaccepted"
	ActivityRejected         ActivityType = "rejected"
	ActivityReopened         ActivityType = "reopened"
	ActivitySuperseded       ActivityType = "superseded"
	ActivityCopied           ActivityType = "copied"
	ActivityAlternativeAdded ActivityType = "alternative_added"
	ActivityConverted        ActivityType = "converted"
	ActivityProjectDeleted   ActivityType = "project_deleted"
	ActivityAttachmentAdded  ActivityType = "attachment_added"
)

// Activity is one entry of the log.
type Activity struct {
	ID           int64
	QuotationID  int64
	Type         ActivityType
	ActorStaffID *int64
	Summary      string
	CreatedAt    time.Time
}
