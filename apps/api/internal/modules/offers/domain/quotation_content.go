package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"time"
)

// SupplierKind is a directory vendor or venue working on a quotation.
type SupplierKind string

const (
	SupplierVendor SupplierKind = "vendor"
	SupplierVenue  SupplierKind = "venue"
)

// Payer is who pays a supplier (PLAN §3.1, #12).
type Payer string

const (
	PayerWO     Payer = "WO"
	PayerClient Payer = "Client"
)

// ValidPayer reports whether p is WO or Client.
func ValidPayer(p Payer) bool { return p == PayerWO || p == PayerClient }

// DueBasis is what a term's due day counts from.
type DueBasis string

const (
	DueOnAcceptance DueBasis = "Accepted"
	DueOnEvent      DueBasis = "Event"
)

// ValidDueBasis reports whether b is Accepted or Event.
func ValidDueBasis(b DueBasis) bool { return b == DueOnAcceptance || b == DueOnEvent }

// Limits of a quotation's content.
const (
	MaxQuotationEvents          = 10
	MaxEventSessions            = 10
	MaxSessionGuests            = 100_000
	MaxQuotationSuppliers       = 100
	MaxQuotationLines           = 300
	MaxQuotationTerms           = 12
	MaxMoney              int64 = 1_000_000_000_000 // Rp1 triliun: far above any wedding
	MaxTermsTextLen             = 10_000
	MaxCoupleNamesLen           = 150
	MaxNotesLen                 = 5_000
	MaxLineDescriptionLen       = 255
	MaxLineDetailsLen           = 2_000
	MaxTermLabelLen             = 60
	MaxQuantity           int64 = 1_000_000
)

// QuotationSession is one session of an event.
type QuotationSession struct {
	ID        int64
	Label     string
	StartTime *string
	EndTime   *string
	Guests    int
	SortOrder int
}

// QuotationEvent is one event of a quotation. Its date may stay empty in a
// quotation (an estimate); a project needs it.
type QuotationEvent struct {
	ID              int64
	QuotationID     int64
	EventKindID     int64
	KindName        string
	EventDate       *time.Time
	StartTime       *string
	EndTime         *string
	VenueSupplierID *int64
	Layout          Layout
	SortOrder       int
	Sessions        []QuotationSession
	UpdatedAt       time.Time
}

// TotalGuests is Σ guests of the event's sessions.
func (e QuotationEvent) TotalGuests() int {
	n := 0
	for _, s := range e.Sessions {
		n += s.Guests
	}
	return n
}

// QuotationSupplier is a directory vendor or venue working on a quotation.
type QuotationSupplier struct {
	ID               int64
	QuotationID      int64
	Kind             SupplierKind
	SupplierID       int64
	Name             string
	CategoryID       *int64
	PaidBy           Payer
	ShowName         bool
	CapacityStanding *int
	CapacitySeated   *int
	AgreedCost       *int64
	Notes            string
	SortOrder        int
	UpdatedAt        time.Time
}

// QuotationLine is one line of "Rincian". Component and ExtraGuests lines
// point at their Package line (ParentLineID).
type QuotationLine struct {
	ID           int64
	QuotationID  int64
	Kind         LineKind
	ParentLineID *int64
	PackageID    *int64
	PackageRule  *PackageRule
	CopiedAt     *time.Time
	Description  string
	Details      string
	CategoryID   *int64
	EventID      *int64
	Quantity     *int64
	Unit         *string
	UnitPrice    *int64
	Amount       int64
	IsBonus      bool
	SupplierRef  *int64
	SortOrder    int
}

// QuotationTerm is one installment of the payment terms: a percent of the
// contract value or a fixed amount, due some days after acceptance or
// before/after an event.
type QuotationTerm struct {
	ID            int64
	QuotationID   int64
	Label         string
	Percent       *int
	Amount        *int64
	DueBasis      DueBasis
	DueOffsetDays int
	// DueEventID is the event an Event-based term counts from (nil = the
	// earliest event).
	DueEventID *int64
	SortOrder  int
}

// WorkingCopy is a quotation with everything it contains, as it is now.
type WorkingCopy struct {
	Quotation Quotation
	Events    []QuotationEvent
	Suppliers []QuotationSupplier
	Lines     []QuotationLine
	Terms     []QuotationTerm
}

// DefaultTerms are the terms a new quotation starts with: 30/40/30, like a
// new project (DP on acceptance, Termin 1 sixty days and Pelunasan seven days
// before the event).
func DefaultTerms() []QuotationTerm {
	p := func(v int) *int { return &v }
	return []QuotationTerm{
		{Label: "DP", Percent: p(30), DueBasis: DueOnAcceptance, DueOffsetDays: 0, SortOrder: 0},
		{Label: "Termin 1", Percent: p(40), DueBasis: DueOnEvent, DueOffsetDays: -60, SortOrder: 1},
		{Label: "Pelunasan", Percent: p(30), DueBasis: DueOnEvent, DueOffsetDays: -7, SortOrder: 2},
	}
}

// SortEvents puts events in screen order: by date (undated last), then by
// their own order.
func SortQuotationEvents(events []QuotationEvent) {
	sort.SliceStable(events, func(i, j int) bool {
		a, b := events[i], events[j]
		switch {
		case a.EventDate != nil && b.EventDate != nil && !a.EventDate.Equal(*b.EventDate):
			return a.EventDate.Before(*b.EventDate)
		case (a.EventDate == nil) != (b.EventDate == nil):
			return a.EventDate != nil
		}
		if a.SortOrder != b.SortOrder {
			return a.SortOrder < b.SortOrder
		}
		return a.ID < b.ID
	})
}

// MainEventDate is the earliest event date (nil when no event has one).
func MainEventDate(events []QuotationEvent) *time.Time {
	var out *time.Time
	for _, e := range events {
		if e.EventDate == nil {
			continue
		}
		d := CivilDate(*e.EventDate)
		if out == nil || d.Before(*out) {
			out = &d
		}
	}
	return out
}

// EventGuestsOf is what ExtraGuests needs of the events.
func EventGuestsOf(events []QuotationEvent) []EventGuests {
	out := make([]EventGuests, len(events))
	for i, e := range events {
		out[i] = EventGuests{EventKindID: e.EventKindID, Guests: e.TotalGuests()}
	}
	return out
}

// RecomputeExtraGuests sets the amount of every ExtraGuests line from its
// package line's copied rule and the events' guests now (PLAN §4.2). It
// reports whether an amount changed.
func RecomputeExtraGuests(lines []QuotationLine, events []QuotationEvent) bool {
	rules := map[int64]*PackageRule{}
	for _, l := range lines {
		if l.Kind == LinePackage && l.PackageRule != nil {
			rules[l.ID] = l.PackageRule
		}
	}
	guests := EventGuestsOf(events)
	changed := false
	for i := range lines {
		l := &lines[i]
		if l.Kind != LineExtraGuests || l.ParentLineID == nil {
			continue
		}
		rule := rules[*l.ParentLineID]
		if rule == nil {
			continue
		}
		amount := ExtraGuests(*rule, guests).Amount
		if amount != l.Amount {
			l.Amount = amount
			changed = true
		}
	}
	return changed
}

// HasExtraGuestPricing reports whether a rule can charge for extra guests.
func HasExtraGuestPricing(r PackageRule) bool {
	return r.IncludedGuests != nil && r.ExtraGuestStep != nil && *r.ExtraGuestStep > 0 && r.ExtraGuestStepPrice != nil
}

// TotalsOf adds the lines up with LineTotals.
func TotalsOf(lines []QuotationLine, isPKP bool) (Totals, error) {
	in := make([]Line, len(lines))
	for i, l := range lines {
		in[i] = Line{Kind: l.Kind, Amount: l.Amount}
	}
	return LineTotals(in, isPKP)
}

// TermAmounts is what each term asks for, given the contract value: a fixed
// amount as written, a percent of the value rounded down, and when the
// percents add up to 100 with no fixed amount the last percent term takes
// what rounding left, so the terms add up to the value exactly. Mismatch is
// set when they do not add up to the value.
func TermAmounts(terms []QuotationTerm, total int64) (amounts []int64, mismatch bool) {
	amounts = make([]int64, len(terms))
	sumPercent, fixed, last := 0, false, -1
	var sum int64
	for i, t := range terms {
		switch {
		case t.Amount != nil:
			amounts[i] = *t.Amount
			fixed = true
		case t.Percent != nil:
			amounts[i] = total * int64(*t.Percent) / 100
			sumPercent += *t.Percent
			last = i
		}
		sum += amounts[i]
	}
	if !fixed && sumPercent == 100 && last >= 0 {
		amounts[last] += total - sum
		sum = total
	}
	return amounts, len(terms) > 0 && sum != total
}

// ---- revision content and snapshot ----------------------------------------

// SnapshotSession is a session as frozen in a revision.
type SnapshotSession struct {
	Label     string  `json:"label"`
	StartTime *string `json:"startTime"`
	EndTime   *string `json:"endTime"`
	Guests    int     `json:"guests"`
}

// SnapshotEvent is an event as frozen in a revision. VenueIndex points into
// the revision's suppliers (positions, never database ids).
type SnapshotEvent struct {
	EventKindID int64             `json:"eventKindId"`
	KindName    string            `json:"kindName"`
	EventDate   *string           `json:"eventDate"`
	StartTime   *string           `json:"startTime"`
	EndTime     *string           `json:"endTime"`
	VenueIndex  *int              `json:"venueIndex"`
	Layout      Layout            `json:"layout"`
	Sessions    []SnapshotSession `json:"sessions"`
}

// SnapshotSupplier is a supplier as frozen in a revision.
type SnapshotSupplier struct {
	Kind             SupplierKind `json:"kind"`
	SupplierID       int64        `json:"supplierId"`
	Name             string       `json:"name"`
	CategoryID       *int64       `json:"categoryId"`
	PaidBy           Payer        `json:"paidBy"`
	ShowName         bool         `json:"showName"`
	CapacityStanding *int         `json:"capacityStanding"`
	CapacitySeated   *int         `json:"capacitySeated"`
	AgreedCost       *int64       `json:"agreedCost"`
	Notes            string       `json:"notes"`
}

// SnapshotLine is a line as frozen in a revision; ParentIndex, EventIndex
// and SupplierIndex are positions in the revision.
type SnapshotLine struct {
	Kind          LineKind     `json:"kind"`
	ParentIndex   *int         `json:"parentIndex"`
	PackageID     *int64       `json:"packageId"`
	PackageRule   *PackageRule `json:"packageRule"`
	Description   string       `json:"description"`
	Details       string       `json:"details"`
	CategoryID    *int64       `json:"categoryId"`
	EventIndex    *int         `json:"eventIndex"`
	Quantity      *int64       `json:"quantity"`
	Unit          *string      `json:"unit"`
	UnitPrice     *int64       `json:"unitPrice"`
	Amount        int64        `json:"amount"`
	IsBonus       bool         `json:"isBonus"`
	SupplierIndex *int         `json:"supplierIndex"`
}

// SnapshotTerm is a term as frozen in a revision, with what it asks for.
type SnapshotTerm struct {
	Label         string   `json:"label"`
	Percent       *int     `json:"percent"`
	Amount        *int64   `json:"amount"`
	DueBasis      DueBasis `json:"dueBasis"`
	DueOffsetDays int      `json:"dueOffsetDays"`
	DueEventIndex *int     `json:"dueEventIndex"`
	Computed      int64    `json:"computed"`
}

// Content is everything a revision promises the client; its hash decides
// whether two revisions are the same. It leaves out who the WO is, the
// validity, the revision number, and the time it was made: none of these make
// the offer different.
type Content struct {
	CoupleNames     string             `json:"coupleNames"`
	ContactWhatsApp string             `json:"contactWhatsapp"`
	PICStaffID      int64              `json:"picStaffId"`
	IsPKP           bool               `json:"isPkp"`
	Events          []SnapshotEvent    `json:"events"`
	Suppliers       []SnapshotSupplier `json:"suppliers"`
	Lines           []SnapshotLine     `json:"lines"`
	Terms           []SnapshotTerm     `json:"terms"`
	TermsText       string             `json:"termsText"`
	Totals          Totals             `json:"totals"`
}

// WorkspaceIdentity is the WO as the PDF letterhead shows it, frozen in each
// revision (the logo by its storage key, which never changes for a file).
type WorkspaceIdentity struct {
	Name              string  `json:"name"`
	LogoKey           *string `json:"logoKey"`
	Address           string  `json:"address"`
	City              string  `json:"city"`
	WhatsApp          string  `json:"whatsapp"`
	BankName          string  `json:"bankName"`
	BankAccountNumber string  `json:"bankAccountNumber"`
	BankAccountHolder string  `json:"bankAccountHolder"`
}

// Snapshot is a revision: the content plus what the PDF prints around it.
type Snapshot struct {
	SchemaVersion int               `json:"schemaVersion"`
	Number        string            `json:"number"`
	RevisionN     int               `json:"revisionN"`
	IssuedAt      time.Time         `json:"issuedAt"`
	ValidUntil    string            `json:"validUntil"`
	PICName       string            `json:"picName"`
	Workspace     WorkspaceIdentity `json:"workspace"`
	// CategoryNames are the directory's names when the revision was made.
	CategoryNames map[int64]string `json:"categoryNames"`
	Content       Content          `json:"content"`
}

// SnapshotSchemaVersion is the shape of Snapshot written today.
const SnapshotSchemaVersion = 1

func dateString(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := CivilDate(*t).Format("2006-01-02")
	return &s
}

// BuildContent freezes a working copy into revision content. Events are in
// screen order, suppliers and lines in their own order, and every reference
// becomes a position.
func BuildContent(w WorkingCopy) (Content, error) {
	events := append([]QuotationEvent(nil), w.Events...)
	SortQuotationEvents(events)
	suppliers := append([]QuotationSupplier(nil), w.Suppliers...)
	sort.SliceStable(suppliers, func(i, j int) bool {
		if suppliers[i].SortOrder != suppliers[j].SortOrder {
			return suppliers[i].SortOrder < suppliers[j].SortOrder
		}
		return suppliers[i].ID < suppliers[j].ID
	})
	lines := OrderedLines(w.Lines)
	terms := append([]QuotationTerm(nil), w.Terms...)
	sort.SliceStable(terms, func(i, j int) bool { return terms[i].SortOrder < terms[j].SortOrder })

	supIdx := map[int64]int{}
	c := Content{
		CoupleNames: w.Quotation.CoupleNames, ContactWhatsApp: w.Quotation.ContactWhatsApp,
		PICStaffID: w.Quotation.PICStaffID, IsPKP: w.Quotation.IsPKP, TermsText: w.Quotation.TermsText,
		Events: []SnapshotEvent{}, Suppliers: []SnapshotSupplier{}, Lines: []SnapshotLine{}, Terms: []SnapshotTerm{},
	}
	for i, s := range suppliers {
		supIdx[s.ID] = i
		c.Suppliers = append(c.Suppliers, SnapshotSupplier{Kind: s.Kind, SupplierID: s.SupplierID, Name: s.Name, CategoryID: s.CategoryID,
			PaidBy: s.PaidBy, ShowName: s.ShowName, CapacityStanding: s.CapacityStanding, CapacitySeated: s.CapacitySeated,
			AgreedCost: s.AgreedCost, Notes: s.Notes})
	}
	evIdx := map[int64]int{}
	for i, e := range events {
		evIdx[e.ID] = i
		se := SnapshotEvent{EventKindID: e.EventKindID, KindName: e.KindName, EventDate: dateString(e.EventDate),
			StartTime: e.StartTime, EndTime: e.EndTime, Layout: e.Layout, Sessions: []SnapshotSession{}}
		if e.VenueSupplierID != nil {
			if v, ok := supIdx[*e.VenueSupplierID]; ok {
				se.VenueIndex = &v
			}
		}
		for _, s := range e.Sessions {
			se.Sessions = append(se.Sessions, SnapshotSession{Label: s.Label, StartTime: s.StartTime, EndTime: s.EndTime, Guests: s.Guests})
		}
		c.Events = append(c.Events, se)
	}
	lineIdx := map[int64]int{}
	for i, l := range lines {
		lineIdx[l.ID] = i
	}
	for _, l := range lines {
		sl := SnapshotLine{Kind: l.Kind, PackageID: l.PackageID, PackageRule: l.PackageRule, Description: l.Description, Details: l.Details,
			CategoryID: l.CategoryID, Quantity: l.Quantity, Unit: l.Unit, UnitPrice: l.UnitPrice, Amount: l.Amount, IsBonus: l.IsBonus}
		if l.ParentLineID != nil {
			if v, ok := lineIdx[*l.ParentLineID]; ok {
				sl.ParentIndex = &v
			}
		}
		if l.EventID != nil {
			if v, ok := evIdx[*l.EventID]; ok {
				sl.EventIndex = &v
			}
		}
		if l.SupplierRef != nil {
			if v, ok := supIdx[*l.SupplierRef]; ok {
				sl.SupplierIndex = &v
			}
		}
		c.Lines = append(c.Lines, sl)
	}
	totals, err := TotalsOf(lines, w.Quotation.IsPKP)
	if err != nil {
		return Content{}, err
	}
	c.Totals = totals
	amounts, _ := TermAmounts(terms, totals.Total)
	for i, t := range terms {
		st := SnapshotTerm{Label: t.Label, Percent: t.Percent, Amount: t.Amount, DueBasis: t.DueBasis, DueOffsetDays: t.DueOffsetDays, Computed: amounts[i]}
		if t.DueEventID != nil {
			if v, ok := evIdx[*t.DueEventID]; ok {
				st.DueEventIndex = &v
			}
		}
		c.Terms = append(c.Terms, st)
	}
	return c, nil
}

// OrderedLines is the lines in screen order: top-level lines by their order,
// each package followed by its components and then its extra guests line.
func OrderedLines(lines []QuotationLine) []QuotationLine {
	var top []QuotationLine
	children := map[int64][]QuotationLine{}
	for _, l := range lines {
		if l.ParentLineID != nil {
			children[*l.ParentLineID] = append(children[*l.ParentLineID], l)
		} else {
			top = append(top, l)
		}
	}
	byOrder := func(s []QuotationLine) {
		sort.SliceStable(s, func(i, j int) bool {
			if (s[i].Kind == LineExtraGuests) != (s[j].Kind == LineExtraGuests) {
				return s[j].Kind == LineExtraGuests
			}
			if s[i].SortOrder != s[j].SortOrder {
				return s[i].SortOrder < s[j].SortOrder
			}
			return s[i].ID < s[j].ID
		})
	}
	byOrder(top)
	out := make([]QuotationLine, 0, len(lines))
	for _, l := range top {
		out = append(out, l)
		kids := children[l.ID]
		byOrder(kids)
		out = append(out, kids...)
	}
	return out
}

// Hash is the sha256 of the content's JSON: two revisions with the same hash
// promise the same thing.
func (c Content) Hash() string {
	raw, _ := json.Marshal(c)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

// QuotationRevision is a revision as stored.
type QuotationRevision struct {
	ID               int64
	QuotationID      int64
	N                int
	Snapshot         Snapshot
	ContentHash      string
	ContractTotal    int64
	CreatedByStaffID *int64
	CreatedAt        time.Time
}
