package presentation

import (
	"time"

	"elproof/internal/modules/offers/application"
	"elproof/internal/modules/offers/domain"
	"elproof/internal/shared/etag"
)

func dateOf(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := domain.CivilDate(*t).Format("2006-01-02")
	return &s
}

func stampOf(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := etag.Format(*t)
	return &s
}

// names are the display names a quotation response needs.
type names struct {
	staff      map[int64]string
	categories map[int64]string
}

func (n names) staffName(id *int64) *string {
	if id == nil {
		return nil
	}
	if s, ok := n.staff[*id]; ok {
		return &s
	}
	return nil
}

func (n names) category(id *int64) *string {
	if id == nil {
		return nil
	}
	if s, ok := n.categories[*id]; ok {
		return &s
	}
	return nil
}

// ---- header -----------------------------------------------------------------

type quotationResponse struct {
	ID              int64   `json:"id"`
	Number          string  `json:"number"`
	DocDate         string  `json:"docDate"`
	CoupleNames     string  `json:"coupleNames"`
	ContactWhatsApp string  `json:"contactWhatsapp"`
	PICStaffID      int64   `json:"picStaffId"`
	PICName         *string `json:"picName"`
	ValidUntil      string  `json:"validUntil"`
	DaysLeft        int     `json:"daysLeft"`
	Notes           string  `json:"notes"`
	// Status is the stored status; DisplayStatus adds Expired (a Draft past
	// its validity).
	Status                  domain.QuotationStatus `json:"status"`
	DisplayStatus           domain.QuotationStatus `json:"displayStatus"`
	AlternativeOfID         *int64                 `json:"alternativeOfId"`
	SupersededByID          *int64                 `json:"supersededById"`
	AcceptedRevisionN       *int                   `json:"acceptedRevisionN"`
	AcceptedOn              *string                `json:"acceptedOn"`
	AcceptedChannel         *domain.AcceptChannel  `json:"acceptedChannel"`
	AcceptanceAttachment    *string                `json:"acceptanceAttachmentName"`
	HasAcceptanceAttachment bool                   `json:"hasAcceptanceAttachment"`
	RejectedReason          *string                `json:"rejectedReason"`
	RejectedAt              *string                `json:"rejectedAt"`
	ConvertedProjectID      *int64                 `json:"convertedProjectId"`
	ConvertedAt             *string                `json:"convertedAt"`
	ProjectDeleted          bool                   `json:"projectDeleted"`
	Editable                bool                   `json:"editable"`
	IsPKP                   bool                   `json:"isPkp"`
	RevisionCount           int                    `json:"revisionCount"`
	ContractTotal           int64                  `json:"contractTotal"`
	MainEventDate           *string                `json:"mainEventDate"`
	CreatedAt               string                 `json:"createdAt"`
	// UpdatedAt is the header's version (If-Match of PUT /quotations/{id},
	// POST …/acceptance-attachment); LinesUpdatedAt of PUT …/lines;
	// TermsUpdatedAt of PUT …/terms and PUT …/terms-text.
	UpdatedAt      string `json:"updatedAt"`
	LinesUpdatedAt string `json:"linesUpdatedAt"`
	TermsUpdatedAt string `json:"termsUpdatedAt"`
	ChangedAt      string `json:"changedAt"`
}

func (r quotationResponse) ETag() string { return quoted(r.UpdatedAt) }

func toQuotationResponse(q domain.Quotation, n names, now time.Time) quotationResponse {
	out := quotationResponse{
		ID: q.ID, Number: q.Number, DocDate: q.DocDate.Format("2006-01-02"), CoupleNames: q.CoupleNames, ContactWhatsApp: q.ContactWhatsApp,
		PICStaffID: q.PICStaffID, PICName: n.staffName(&q.PICStaffID), ValidUntil: q.ValidUntil.Format("2006-01-02"),
		DaysLeft: q.DaysLeft(now), Notes: q.Notes, Status: q.Status, DisplayStatus: q.DisplayStatus(now),
		AlternativeOfID: q.AlternativeOfID, SupersededByID: q.SupersededByID, AcceptedOn: dateOf(q.AcceptedOn),
		AcceptedChannel: q.AcceptedChannel, AcceptanceAttachment: q.AcceptedAttachmentName, HasAcceptanceAttachment: q.AcceptedAttachmentKey != nil,
		RejectedReason: q.RejectedReason, RejectedAt: stampOf(q.RejectedAt), ConvertedProjectID: q.ConvertedProjectID,
		ConvertedAt: stampOf(q.ConvertedAt), ProjectDeleted: q.ProjectDeletedAt != nil, Editable: q.Editable(), IsPKP: q.IsPKP,
		RevisionCount: q.RevisionCount, ContractTotal: q.ContractTotal, MainEventDate: dateOf(q.MainEventDate),
		CreatedAt: etag.Format(q.CreatedAt), UpdatedAt: etag.Format(q.UpdatedAt), LinesUpdatedAt: etag.Format(q.LinesUpdatedAt),
		TermsUpdatedAt: etag.Format(q.TermsUpdatedAt), ChangedAt: etag.Format(q.ChangedAt),
	}
	if q.AcceptedRevisionN > 0 {
		n := q.AcceptedRevisionN
		out.AcceptedRevisionN = &n
	}
	return out
}

// quotationListRow is a group of the list: the first quotation and its
// alternatives as children.
type quotationListRow struct {
	quotationResponse
	Alternatives []quotationResponse `json:"alternatives"`
}

// ---- content ------------------------------------------------------------------

type sessionResponse struct {
	ID        int64   `json:"id"`
	Label     string  `json:"label"`
	StartTime *string `json:"startTime"`
	EndTime   *string `json:"endTime"`
	Guests    int     `json:"guests"`
}

type eventResponse struct {
	ID          int64   `json:"id"`
	EventKindID int64   `json:"eventKindId"`
	KindName    string  `json:"kindName"`
	EventDate   *string `json:"eventDate"`
	StartTime   *string `json:"startTime"`
	EndTime     *string `json:"endTime"`
	// VenueSupplierID is the venue's row in "Vendor & venue"; VenueID and
	// VenueName are the directory venue it stands for.
	VenueSupplierID *int64            `json:"venueSupplierId"`
	VenueID         *int64            `json:"venueId"`
	VenueName       *string           `json:"venueName"`
	Layout          domain.Layout     `json:"layout"`
	SortOrder       int               `json:"sortOrder"`
	Sessions        []sessionResponse `json:"sessions"`
	TotalGuests     int               `json:"totalGuests"`
	UpdatedAt       string            `json:"updatedAt"`
}

func (r eventResponse) ETag() string { return quoted(r.UpdatedAt) }

func toEventResponse(e domain.QuotationEvent, suppliers []domain.QuotationSupplier) eventResponse {
	out := eventResponse{ID: e.ID, EventKindID: e.EventKindID, KindName: e.KindName, EventDate: dateOf(e.EventDate),
		StartTime: e.StartTime, EndTime: e.EndTime, VenueSupplierID: e.VenueSupplierID, Layout: e.Layout, SortOrder: e.SortOrder,
		Sessions: []sessionResponse{}, TotalGuests: e.TotalGuests(), UpdatedAt: etag.Format(e.UpdatedAt)}
	if e.VenueSupplierID != nil {
		for _, s := range suppliers {
			if s.ID == *e.VenueSupplierID {
				id, name := s.SupplierID, s.Name
				out.VenueID, out.VenueName = &id, &name
			}
		}
	}
	for _, s := range e.Sessions {
		out.Sessions = append(out.Sessions, sessionResponse{ID: s.ID, Label: s.Label, StartTime: s.StartTime, EndTime: s.EndTime, Guests: s.Guests})
	}
	return out
}

type supplierResponse struct {
	ID               int64               `json:"id"`
	Kind             domain.SupplierKind `json:"kind"`
	SupplierID       int64               `json:"supplierId"`
	Name             string              `json:"name"`
	CategoryID       *int64              `json:"categoryId"`
	CategoryName     *string             `json:"categoryName"`
	PaidBy           domain.Payer        `json:"paidBy"`
	ShowName         bool                `json:"showName"`
	CapacityStanding *int                `json:"capacityStanding"`
	CapacitySeated   *int                `json:"capacitySeated"`
	// AgreedCost is left out entirely without finance.margin.view.
	AgreedCost *int64 `json:"agreedCost,omitempty"`
	Notes      string `json:"notes"`
	SortOrder  int    `json:"sortOrder"`
	UpdatedAt  string `json:"updatedAt"`
}

func (r supplierResponse) ETag() string { return quoted(r.UpdatedAt) }

func toSupplierResponse(s domain.QuotationSupplier, n names, showCost bool) supplierResponse {
	out := supplierResponse{ID: s.ID, Kind: s.Kind, SupplierID: s.SupplierID, Name: s.Name, CategoryID: s.CategoryID,
		CategoryName: n.category(s.CategoryID), PaidBy: s.PaidBy, ShowName: s.ShowName, CapacityStanding: s.CapacityStanding,
		CapacitySeated: s.CapacitySeated, Notes: s.Notes, SortOrder: s.SortOrder, UpdatedAt: etag.Format(s.UpdatedAt)}
	if showCost {
		out.AgreedCost = s.AgreedCost
	}
	return out
}

type lineResponse struct {
	ID           int64               `json:"id"`
	Kind         domain.LineKind     `json:"kind"`
	ParentLineID *int64              `json:"parentLineId"`
	PackageID    *int64              `json:"packageId"`
	PackageRule  *domain.PackageRule `json:"packageRule"`
	CopiedAt     *string             `json:"copiedAt"`
	// CatalogChanged: the package changed in the catalog after this line
	// copied it ("Paket di katalog sudah berubah").
	CatalogChanged bool    `json:"catalogChanged"`
	Description    string  `json:"description"`
	Details        string  `json:"details"`
	CategoryID     *int64  `json:"categoryId"`
	CategoryName   *string `json:"categoryName"`
	EventID        *int64  `json:"eventId"`
	Quantity       *int64  `json:"quantity"`
	Unit           *string `json:"unit"`
	UnitPrice      *int64  `json:"unitPrice"`
	Amount         int64   `json:"amount"`
	IsBonus        bool    `json:"isBonus"`
	SupplierRef    *int64  `json:"supplierRef"`
	SortOrder      int     `json:"sortOrder"`
}

func toLineResponse(l domain.QuotationLine, n names, changed bool) lineResponse {
	return lineResponse{ID: l.ID, Kind: l.Kind, ParentLineID: l.ParentLineID, PackageID: l.PackageID, PackageRule: l.PackageRule,
		CopiedAt: stampOf(l.CopiedAt), CatalogChanged: changed, Description: l.Description, Details: l.Details, CategoryID: l.CategoryID,
		CategoryName: n.category(l.CategoryID), EventID: l.EventID, Quantity: l.Quantity, Unit: l.Unit, UnitPrice: l.UnitPrice,
		Amount: l.Amount, IsBonus: l.IsBonus, SupplierRef: l.SupplierRef, SortOrder: l.SortOrder}
}

type termResponse struct {
	ID            int64           `json:"id"`
	Label         string          `json:"label"`
	Percent       *int            `json:"percent"`
	Amount        *int64          `json:"amount"`
	DueBasis      domain.DueBasis `json:"dueBasis"`
	DueOffsetDays int             `json:"dueOffsetDays"`
	DueEventID    *int64          `json:"dueEventId"`
	// Computed is what the term asks for of the current contract value.
	Computed  int64 `json:"computed"`
	SortOrder int   `json:"sortOrder"`
}

type groupMember struct {
	ID            int64                  `json:"id"`
	Number        string                 `json:"number"`
	Status        domain.QuotationStatus `json:"status"`
	DisplayStatus domain.QuotationStatus `json:"displayStatus"`
	ContractTotal int64                  `json:"contractTotal"`
	IsCurrent     bool                   `json:"isCurrent"`
}

// quotationDetailResponse is GET /quotations/{id}: the header, the working
// copy, its totals, and how it stands against its revisions and its group.
type quotationDetailResponse struct {
	Quotation quotationResponse  `json:"quotation"`
	Events    []eventResponse    `json:"events"`
	Suppliers []supplierResponse `json:"suppliers"`
	Lines     []lineResponse     `json:"lines"`
	Terms     []termResponse     `json:"terms"`
	TermsText string             `json:"termsText"`
	Totals    domain.Totals      `json:"totals"`
	// TotalsProblem is the message when discounts exceed the subtotal (the
	// totals are then zero).
	TotalsProblem *string `json:"totalsProblem"`
	TermsMismatch bool    `json:"termsMismatch"`
	// MatchingRevisionN is the revision equal to the working copy (null:
	// HasUnsentChanges — "Ada perubahan yang belum dikirim").
	MatchingRevisionN *int          `json:"matchingRevisionN"`
	LatestRevisionN   int           `json:"latestRevisionN"`
	HasUnsentChanges  bool          `json:"hasUnsentChanges"`
	Group             []groupMember `json:"group"`
}

func (r quotationDetailResponse) ETag() string { return r.Quotation.ETag() }

func toDetailResponse(d *application.QuotationDetail, n names, showCost bool, now time.Time) quotationDetailResponse {
	w := d.Content
	out := quotationDetailResponse{
		Quotation: toQuotationResponse(d.Quotation, n, now), Events: []eventResponse{}, Suppliers: []supplierResponse{},
		Lines: []lineResponse{}, Terms: []termResponse{}, TermsText: d.Quotation.TermsText, Totals: d.Totals,
		TermsMismatch: d.TermsMismatch, MatchingRevisionN: d.MatchingRevisionN, LatestRevisionN: d.LatestRevisionN,
		HasUnsentChanges: d.MatchingRevisionN == nil, Group: []groupMember{},
	}
	if d.TotalsError != nil {
		msg := "Potongan melebihi subtotal. Kurangi potongan."
		out.TotalsProblem = &msg
	}
	events := append([]domain.QuotationEvent(nil), w.Events...)
	domain.SortQuotationEvents(events)
	for _, e := range events {
		out.Events = append(out.Events, toEventResponse(e, w.Suppliers))
	}
	for _, s := range w.Suppliers {
		out.Suppliers = append(out.Suppliers, toSupplierResponse(s, n, showCost))
	}
	for _, l := range domain.OrderedLines(w.Lines) {
		out.Lines = append(out.Lines, toLineResponse(l, n, d.CatalogChanged[l.ID]))
	}
	for i, t := range w.Terms {
		tr := termResponse{ID: t.ID, Label: t.Label, Percent: t.Percent, Amount: t.Amount, DueBasis: t.DueBasis,
			DueOffsetDays: t.DueOffsetDays, DueEventID: t.DueEventID, SortOrder: t.SortOrder}
		if i < len(d.TermAmounts) {
			tr.Computed = d.TermAmounts[i]
		}
		out.Terms = append(out.Terms, tr)
	}
	if len(d.Group) > 1 {
		for _, g := range d.Group {
			out.Group = append(out.Group, groupMember{ID: g.ID, Number: g.Number, Status: g.Status, DisplayStatus: g.DisplayStatus(now),
				ContractTotal: g.ContractTotal, IsCurrent: g.ID == d.Quotation.ID})
		}
	}
	return out
}

// ---- lines preview --------------------------------------------------------------

type previewLineResponse struct {
	Index       int                       `json:"index"`
	ParentIndex *int                      `json:"parentIndex"`
	Kind        domain.LineKind           `json:"kind"`
	PackageID   *int64                    `json:"packageId"`
	PackageRule *domain.PackageRule       `json:"packageRule"`
	Description string                    `json:"description"`
	Quantity    *int64                    `json:"quantity"`
	UnitPrice   *int64                    `json:"unitPrice"`
	Amount      int64                     `json:"amount"`
	IsBonus     bool                      `json:"isBonus"`
	ExtraGuests *domain.ExtraGuestsResult `json:"extraGuests"`
}

type linesPreviewResponse struct {
	Lines  []previewLineResponse `json:"lines"`
	Totals domain.Totals         `json:"totals"`
}

func toLinesPreview(res *application.LinesResult) linesPreviewResponse {
	out := linesPreviewResponse{Lines: []previewLineResponse{}, Totals: res.Totals}
	for i, row := range res.Rows {
		l := row.Line
		pl := previewLineResponse{Index: i, Kind: l.Kind, PackageID: l.PackageID, PackageRule: l.PackageRule, Description: l.Description,
			Quantity: l.Quantity, UnitPrice: l.UnitPrice, Amount: l.Amount, IsBonus: l.IsBonus}
		if row.Parent >= 0 {
			p := row.Parent
			pl.ParentIndex = &p
		}
		if eg, ok := res.ExtraGuests[i]; ok {
			e := eg
			pl.ExtraGuests = &e
		}
		out.Lines = append(out.Lines, pl)
	}
	return out
}

// ---- revisions --------------------------------------------------------------------

type revisionResponse struct {
	N              int     `json:"n"`
	ContractTotal  int64   `json:"contractTotal"`
	ContentHash    string  `json:"contentHash"`
	CreatedAt      string  `json:"createdAt"`
	CreatedByID    *int64  `json:"createdByStaffId"`
	CreatedByName  *string `json:"createdByName"`
	IsAccepted     bool    `json:"isAccepted"`
	MatchesCurrent bool    `json:"matchesWorkingCopy,omitempty"`
	Snapshot       any     `json:"snapshot,omitempty"`
}

// withoutCosts is a copy of the content with every agreed cost removed (a
// person without finance.margin.view never receives one).
func withoutCosts(c domain.Content) domain.Content {
	sup := make([]domain.SnapshotSupplier, len(c.Suppliers))
	for i, s := range c.Suppliers {
		s.AgreedCost = nil
		sup[i] = s
	}
	c.Suppliers = sup
	return c
}

func toRevisionResponse(rev domain.QuotationRevision, q *domain.Quotation, n names, withSnapshot, showCost bool) revisionResponse {
	out := revisionResponse{N: rev.N, ContractTotal: rev.ContractTotal, ContentHash: rev.ContentHash, CreatedAt: etag.Format(rev.CreatedAt),
		CreatedByID: rev.CreatedByStaffID, CreatedByName: n.staffName(rev.CreatedByStaffID),
		IsAccepted: q != nil && q.AcceptedRevisionID != nil && *q.AcceptedRevisionID == rev.ID}
	if withSnapshot {
		snap := rev.Snapshot
		if !showCost {
			snap.Content = withoutCosts(snap.Content)
		}
		out.Snapshot = snap
	}
	return out
}

// ---- compare ----------------------------------------------------------------------

type lineChangeResponse struct {
	Change domain.ChangeKind    `json:"change"`
	Before *domain.SnapshotLine `json:"before"`
	After  *domain.SnapshotLine `json:"after"`
	Parent string               `json:"parent"`
}

type eventChangeResponse struct {
	Change domain.ChangeKind     `json:"change"`
	Before *domain.SnapshotEvent `json:"before"`
	After  *domain.SnapshotEvent `json:"after"`
}

type supplierChangeResponse struct {
	Change domain.ChangeKind        `json:"change"`
	Before *domain.SnapshotSupplier `json:"before"`
	After  *domain.SnapshotSupplier `json:"after"`
}

type compareResponse struct {
	From               int                      `json:"from"`
	To                 int                      `json:"to"`
	Lines              []lineChangeResponse     `json:"lines"`
	UnchangedLines     int                      `json:"unchangedLines"`
	Events             []eventChangeResponse    `json:"events"`
	UnchangedEvents    int                      `json:"unchangedEvents"`
	Suppliers          []supplierChangeResponse `json:"suppliers"`
	UnchangedSuppliers int                      `json:"unchangedSuppliers"`
	TermsChanged       bool                     `json:"termsChanged"`
	TermsTextChanged   bool                     `json:"termsTextChanged"`
	HeaderChanged      bool                     `json:"headerChanged"`
	TotalsBefore       domain.Totals            `json:"totalsBefore"`
	TotalsAfter        domain.Totals            `json:"totalsAfter"`
}

func costless(s *domain.SnapshotSupplier, showCost bool) *domain.SnapshotSupplier {
	if s == nil || showCost {
		return s
	}
	c := *s
	c.AgreedCost = nil
	return &c
}

func toCompareResponse(res *application.ComparisonResult, showCost bool) compareResponse {
	c := res.Comparison
	out := compareResponse{From: res.From, To: res.To, Lines: []lineChangeResponse{}, UnchangedLines: c.UnchangedLines,
		Events: []eventChangeResponse{}, UnchangedEvents: c.UnchangedEvents, Suppliers: []supplierChangeResponse{},
		UnchangedSuppliers: c.UnchangedSuppliers, TermsChanged: c.TermsChanged, TermsTextChanged: c.TermsTextChanged,
		HeaderChanged: c.HeaderChanged, TotalsBefore: c.TotalsBefore, TotalsAfter: c.TotalsAfter}
	for _, l := range c.Lines {
		out.Lines = append(out.Lines, lineChangeResponse{Change: l.Change, Before: l.Before, After: l.After, Parent: l.Parent})
	}
	for _, e := range c.Events {
		out.Events = append(out.Events, eventChangeResponse{Change: e.Change, Before: e.Before, After: e.After})
	}
	for _, s := range c.Suppliers {
		before, after := costless(s.Before, showCost), costless(s.After, showCost)
		if !showCost && s.Change == domain.ChangeChanged && before != nil && after != nil && *before == *after {
			// Only the agreed cost changed: nothing this person may see.
			out.UnchangedSuppliers++
			continue
		}
		out.Suppliers = append(out.Suppliers, supplierChangeResponse{Change: s.Change, Before: before, After: after})
	}
	return out
}

// ---- activity -----------------------------------------------------------------------

type activityResponse struct {
	ID        int64               `json:"id"`
	Type      domain.ActivityType `json:"type"`
	ActorID   *int64              `json:"actorStaffId"`
	ActorName *string             `json:"actorName"`
	Summary   string              `json:"summary"`
	CreatedAt string              `json:"createdAt"`
}
