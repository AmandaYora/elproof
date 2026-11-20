package presentation

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"elproof/internal/modules/offers/application"
	"elproof/internal/modules/offers/domain"
	"elproof/internal/shared/apperror"
	"elproof/internal/shared/etag"
	"elproof/internal/shared/middleware"
	"elproof/internal/shared/pagination"
	"elproof/internal/shared/response"
)

// The quotation permissions (catalog: access/domain/catalog.go).
const (
	PermQuotationView   = "quotation.view"
	PermQuotationManage = "quotation.manage"
)

const quotationPrefix = "/api/v1/quotations/"

// QuotationHandler is the HTTP surface of quotations (PLAN §5).
type QuotationHandler struct {
	svc *application.QuotationService
}

func NewQuotationHandler(svc *application.QuotationService) *QuotationHandler {
	return &QuotationHandler{svc: svc}
}

// viewer is the person asking: whose quotations they reach (the `all` scope
// of their role, otherwise the ones they are PIC of) and whether they see
// agreed costs.
func viewer(r *http.Request) application.Viewer {
	v := application.Viewer{CanSeeCost: can(r, PermMarginView)}
	if claims, ok := middleware.FromContext(r.Context()); ok && claims.PrincipalType == "staff" {
		v.StaffID, _ = strconv.ParseInt(claims.PrincipalID, 10, 64)
	}
	if g := middleware.GrantsFrom(r.Context()); g != nil && (g.Owner || g.AllProjects) {
		v.AllQuotations = true
	}
	return v
}

func requireQuotationView(w http.ResponseWriter, r *http.Request) bool {
	if can(r, PermQuotationView) || can(r, PermQuotationManage) {
		return true
	}
	return requirePermission(w, r, PermQuotationView)
}

func decode(w http.ResponseWriter, r *http.Request, into any) bool {
	if err := json.NewDecoder(r.Body).Decode(into); err != nil {
		response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
		return false
	}
	return true
}

func notFound(w http.ResponseWriter) {
	response.Error(w, http.StatusNotFound, "Endpoint tidak ditemukan", nil)
}

// parseDay reads a "YYYY-MM-DD" field (nil when empty).
func parseDay(field string, raw *string) (*time.Time, error) {
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return nil, nil
	}
	t, err := time.Parse("2006-01-02", strings.TrimSpace(*raw))
	if err != nil {
		msg := "Tanggal tidak valid (format YYYY-MM-DD)"
		return nil, apperror.Validation(msg, map[string][]string{field: {msg}})
	}
	return &t, nil
}

// ---- names --------------------------------------------------------------------

func (h *QuotationHandler) names(ctx context.Context, tenantID int64, staffIDs, categoryIDs []int64) (names, error) {
	staff, err := h.svc.StaffNames(ctx, tenantID, staffIDs)
	if err != nil {
		return names{}, err
	}
	cats, err := h.svc.CategoryNames(ctx, tenantID, categoryIDs)
	if err != nil {
		return names{}, err
	}
	return names{staff: staff, categories: cats}, nil
}

func contentCategoryIDs(w *domain.WorkingCopy) []int64 {
	var ids []int64
	if w == nil {
		return ids
	}
	for _, s := range w.Suppliers {
		if s.CategoryID != nil {
			ids = append(ids, *s.CategoryID)
		}
	}
	for _, l := range w.Lines {
		if l.CategoryID != nil {
			ids = append(ids, *l.CategoryID)
		}
	}
	return ids
}

func (h *QuotationHandler) detail(r *http.Request, tenantID, id int64) (*quotationDetailResponse, error) {
	v := viewer(r)
	d, err := h.svc.Detail(r.Context(), tenantID, id, v)
	if err != nil {
		return nil, err
	}
	n, err := h.names(r.Context(), tenantID, []int64{d.Quotation.PICStaffID}, contentCategoryIDs(d.Content))
	if err != nil {
		return nil, err
	}
	out := toDetailResponse(d, n, v.CanSeeCost, h.svc.Now())
	return &out, nil
}

// respondDetail answers with the whole quotation page (every content write
// does, so the screen never assembles totals itself).
func (h *QuotationHandler) respondDetail(w http.ResponseWriter, r *http.Request, tenantID, id int64, status int, msg string) {
	d, err := h.detail(r, tenantID, id)
	if err != nil {
		response.FromError(w, err)
		return
	}
	if status == http.StatusCreated {
		response.Created(w, msg, *d)
		return
	}
	response.OK(w, msg, *d)
}

func (h *QuotationHandler) quotationShape(r *http.Request, tenantID int64, q domain.Quotation) (quotationResponse, error) {
	n, err := h.names(r.Context(), tenantID, []int64{q.PICStaffID}, nil)
	if err != nil {
		return quotationResponse{}, err
	}
	return toQuotationResponse(q, n, h.svc.Now()), nil
}

// stale swaps a Stale conflict's payload for its response shape: the header
// for header, terms text, and attachment edits; the item for events and
// suppliers; the whole page for lines and terms.
func (h *QuotationHandler) stale(r *http.Request, tenantID, id int64, err error) error {
	current, ok := apperror.IsStale(err)
	if !ok {
		return err
	}
	ae, _ := apperror.As(err)
	switch c := current.(type) {
	case *domain.Quotation:
		shape, serr := h.quotationShape(r, tenantID, *c)
		if serr != nil {
			return serr
		}
		return ae.WithData(shape)
	case domain.QuotationEvent:
		var suppliers []domain.QuotationSupplier
		if wc, cerr := h.svc.Content(r.Context(), tenantID, id, viewer(r)); cerr == nil {
			suppliers = wc.Suppliers
		}
		return ae.WithData(toEventResponse(c, suppliers))
	case domain.QuotationSupplier:
		var cats []int64
		if c.CategoryID != nil {
			cats = append(cats, *c.CategoryID)
		}
		n, nerr := h.names(r.Context(), tenantID, nil, cats)
		if nerr != nil {
			return nerr
		}
		return ae.WithData(toSupplierResponse(c, n, can(r, PermMarginView)))
	case domain.WorkingCopy:
		d, derr := h.detail(r, tenantID, id)
		if derr != nil {
			return derr
		}
		return ae.WithData(*d)
	}
	return err
}

// ---- collection -------------------------------------------------------------

type quotationHeaderBody struct {
	CoupleNames     string  `json:"coupleNames"`
	ContactWhatsApp string  `json:"contactWhatsapp"`
	PICStaffID      int64   `json:"picStaffId"`
	ValidUntil      *string `json:"validUntil"`
	Notes           string  `json:"notes"`
}

func (b quotationHeaderBody) input() (application.HeaderInput, error) {
	until, err := parseDay("validUntil", b.ValidUntil)
	if err != nil {
		return application.HeaderInput{}, err
	}
	return application.HeaderInput{CoupleNames: b.CoupleNames, ContactWhatsApp: b.ContactWhatsApp, PICStaffID: b.PICStaffID,
		ValidUntil: until, Notes: b.Notes}, nil
}

func listFilter(r *http.Request) (application.QuotationListFilter, error) {
	q := r.URL.Query()
	f := application.QuotationListFilter{Search: pagination.SearchFromRequest(r)}
	switch tab := domain.QuotationStatus(q.Get("tab")); tab {
	case "", domain.QuotationDraft, domain.QuotationExpired, domain.QuotationAccepted, domain.QuotationRejected, domain.QuotationSuperseded:
		f.Tab = tab
	default:
		return f, apperror.Validation("Filter tidak valid", map[string][]string{"tab": {"tab harus Draft, Expired, Accepted, Rejected, atau Superseded"}})
	}
	switch q.Get("expiringSoon") {
	case "", "false":
	case "true":
		f.ExpiringSoon = true
	default:
		return f, apperror.Validation("Filter tidak valid", map[string][]string{"expiringSoon": {"expiringSoon harus true atau false"}})
	}
	if raw := q.Get("picStaffId"); raw != "" {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || id <= 0 {
			return f, apperror.Validation("Filter tidak valid", map[string][]string{"picStaffId": {"picStaffId tidak valid"}})
		}
		f.PICStaffID = &id
	}
	return f, nil
}

// Collection is GET /quotations (groups of alternatives, sorted, filtered,
// and paged on the server) and POST (quotation.manage).
func (h *QuotationHandler) Collection(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := requireStaffTenant(w, r)
	if !ok {
		return
	}
	switch r.Method {
	case http.MethodGet:
		if !requireQuotationView(w, r) {
			return
		}
		filter, err := listFilter(r)
		if err != nil {
			response.FromError(w, err)
			return
		}
		params := pagination.FromRequest(r)
		columns, def := h.svc.ListSortSpec()
		sort, err := pagination.SortFromRequest(r, columns, def)
		if err != nil {
			response.FromError(w, err)
			return
		}
		rows, total, err := h.svc.List(r.Context(), tenantID, viewer(r), params, filter, sort)
		if err != nil {
			response.FromError(w, err)
			return
		}
		var staffIDs []int64
		for _, row := range rows {
			staffIDs = append(staffIDs, row.Quotation.PICStaffID)
			for _, a := range row.Alternatives {
				staffIDs = append(staffIDs, a.PICStaffID)
			}
		}
		n, err := h.names(r.Context(), tenantID, staffIDs, nil)
		if err != nil {
			response.FromError(w, err)
			return
		}
		now := h.svc.Now()
		out := make([]quotationListRow, 0, len(rows))
		for _, row := range rows {
			item := quotationListRow{quotationResponse: toQuotationResponse(row.Quotation, n, now), Alternatives: []quotationResponse{}}
			for _, a := range row.Alternatives {
				item.Alternatives = append(item.Alternatives, toQuotationResponse(a, n, now))
			}
			out = append(out, item)
		}
		response.OKPaginated(w, "ok", out, pagination.BuildMeta(params, total).WithSort(sort))
	case http.MethodPost:
		if !requirePermission(w, r, PermQuotationManage) {
			return
		}
		var body quotationHeaderBody
		if !decode(w, r, &body) {
			return
		}
		in, err := body.input()
		if err != nil {
			response.FromError(w, err)
			return
		}
		v := viewer(r)
		if in.PICStaffID == 0 {
			in.PICStaffID = v.StaffID
		}
		q, err := h.svc.Create(r.Context(), tenantID, v, in)
		if err != nil {
			response.FromError(w, err)
			return
		}
		h.respondDetail(w, r, tenantID, q.ID, http.StatusCreated, "Penawaran dibuat")
	default:
		methodNotAllowed(w)
	}
}

type quotationSummaryResponse struct {
	All          int `json:"all"`
	Draft        int `json:"draft"`
	Expired      int `json:"expired"`
	Accepted     int `json:"accepted"`
	Rejected     int `json:"rejected"`
	Superseded   int `json:"superseded"`
	ExpiringSoon int `json:"expiringSoon"`
}

// Summary is GET /quotations/summary: how many groups each list tab holds
// (same search and PIC filter as the list).
func (h *QuotationHandler) Summary(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := requireStaffTenant(w, r)
	if !ok || !requireQuotationView(w, r) {
		return
	}
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	filter, err := listFilter(r)
	if err != nil {
		response.FromError(w, err)
		return
	}
	counts, err := h.svc.Summary(r.Context(), tenantID, viewer(r), filter)
	if err != nil {
		response.FromError(w, err)
		return
	}
	response.OK(w, "ok", quotationSummaryResponse{All: counts[""], Draft: counts[domain.QuotationDraft], Expired: counts[domain.QuotationExpired],
		Accepted: counts[domain.QuotationAccepted], Rejected: counts[domain.QuotationRejected], Superseded: counts[domain.QuotationSuperseded],
		ExpiringSoon: counts["ExpiringSoon"]})
}

// ---- item -----------------------------------------------------------------------

// Item routes /quotations/{id}/…; reads need quotation.view (or manage),
// writes quotation.manage.
func (h *QuotationHandler) Item(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := requireStaffTenant(w, r)
	if !ok {
		return
	}
	id, segments, ok := itemID(w, r, quotationPrefix, application.MsgQuotationNotFound)
	if !ok {
		return
	}
	read := r.Method == http.MethodGet || r.Method == http.MethodHead
	if read && !requireQuotationView(w, r) {
		return
	}
	if !read && !requirePermission(w, r, PermQuotationManage) {
		return
	}
	rest := segments[1:]
	if len(rest) == 0 {
		h.root(w, r, tenantID, id)
		return
	}
	switch rest[0] {
	case "terms-text":
		h.only(w, r, len(rest) == 1, http.MethodPut, func() { h.termsText(w, r, tenantID, id) })
	case "events":
		h.events(w, r, tenantID, id, rest[1:])
	case "suppliers":
		h.suppliers(w, r, tenantID, id, rest[1:])
	case "lines":
		switch {
		case len(rest) == 1:
			h.only(w, r, true, http.MethodPut, func() { h.saveLines(w, r, tenantID, id) })
		case len(rest) == 2 && rest[1] == "preview":
			h.only(w, r, true, http.MethodPost, func() { h.previewLines(w, r, tenantID, id) })
		default:
			notFound(w)
		}
	case "terms":
		h.only(w, r, len(rest) == 1, http.MethodPut, func() { h.saveTerms(w, r, tenantID, id) })
	case "revisions":
		h.revisions(w, r, tenantID, id, rest[1:])
	case "compare":
		h.only(w, r, len(rest) == 1, http.MethodGet, func() { h.compare(w, r, tenantID, id) })
	case "status":
		h.only(w, r, len(rest) == 1, http.MethodPost, func() { h.setStatus(w, r, tenantID, id) })
	case "copy":
		h.only(w, r, len(rest) == 1, http.MethodPost, func() { h.duplicate(w, r, tenantID, id, false) })
	case "alternatives":
		h.only(w, r, len(rest) == 1, http.MethodPost, func() { h.duplicate(w, r, tenantID, id, true) })
	case "extend":
		h.only(w, r, len(rest) == 1, http.MethodPost, func() { h.extend(w, r, tenantID, id) })
	case "preview.pdf":
		h.only(w, r, len(rest) == 1, http.MethodGet, func() { h.previewPDF(w, r, tenantID, id) })
	case "acceptance-attachment":
		if len(rest) != 1 {
			notFound(w)
			return
		}
		switch r.Method {
		case http.MethodPut:
			h.putAttachment(w, r, tenantID, id)
		case http.MethodGet:
			h.getAttachment(w, r, tenantID, id)
		default:
			methodNotAllowed(w)
		}
	case "activity":
		h.only(w, r, len(rest) == 1, http.MethodGet, func() { h.activity(w, r, tenantID, id) })
	default:
		notFound(w)
	}
}

// only runs fn when the path matched and the method is the one allowed.
func (h *QuotationHandler) only(w http.ResponseWriter, r *http.Request, matched bool, method string, fn func()) {
	if !matched {
		notFound(w)
		return
	}
	if r.Method != method {
		methodNotAllowed(w)
		return
	}
	fn()
}

func (h *QuotationHandler) exists(r *http.Request, tenantID, id int64) func() error {
	return func() error { _, err := h.svc.Get(r.Context(), tenantID, id, viewer(r)); return err }
}

// root is GET (the quotation page), PUT (the header, If-Match updatedAt), and
// DELETE (a Draft that never had a revision; otherwise 422).
func (h *QuotationHandler) root(w http.ResponseWriter, r *http.Request, tenantID, id int64) {
	switch r.Method {
	case http.MethodGet:
		h.respondDetail(w, r, tenantID, id, http.StatusOK, "ok")
	case http.MethodPut:
		expected, ok := ifMatch(w, r, "penawaran", h.exists(r, tenantID, id))
		if !ok {
			return
		}
		var body quotationHeaderBody
		if !decode(w, r, &body) {
			return
		}
		in, err := body.input()
		if err != nil {
			response.FromError(w, err)
			return
		}
		if _, err := h.svc.UpdateHeader(r.Context(), tenantID, id, viewer(r), in, expected); err != nil {
			response.FromError(w, h.stale(r, tenantID, id, err))
			return
		}
		h.respondDetail(w, r, tenantID, id, http.StatusOK, "Penawaran disimpan")
	case http.MethodDelete:
		if err := h.svc.DeleteDraft(r.Context(), tenantID, id, viewer(r)); err != nil {
			response.FromError(w, err)
			return
		}
		response.OK(w, "Penawaran dihapus", nil)
	default:
		methodNotAllowed(w)
	}
}

func (h *QuotationHandler) termsText(w http.ResponseWriter, r *http.Request, tenantID, id int64) {
	expected, ok := ifMatch(w, r, "syarat & ketentuan (termsUpdatedAt)", h.exists(r, tenantID, id))
	if !ok {
		return
	}
	var body struct {
		TermsText string `json:"termsText"`
	}
	if !decode(w, r, &body) {
		return
	}
	if _, err := h.svc.UpdateTermsText(r.Context(), tenantID, id, viewer(r), body.TermsText, expected); err != nil {
		response.FromError(w, h.stale(r, tenantID, id, err))
		return
	}
	h.respondDetail(w, r, tenantID, id, http.StatusOK, "Syarat & ketentuan disimpan")
}

// ---- events -----------------------------------------------------------------------

type sessionBody struct {
	Label     string  `json:"label"`
	StartTime *string `json:"startTime"`
	EndTime   *string `json:"endTime"`
	Guests    int     `json:"guests"`
}

type eventBody struct {
	EventKindID int64         `json:"eventKindId"`
	EventDate   *string       `json:"eventDate"`
	StartTime   *string       `json:"startTime"`
	EndTime     *string       `json:"endTime"`
	VenueID     *int64        `json:"venueId"`
	Layout      domain.Layout `json:"layout"`
	Sessions    []sessionBody `json:"sessions"`
}

func (b eventBody) input() (application.QuotationEventInput, error) {
	day, err := parseDay("eventDate", b.EventDate)
	if err != nil {
		return application.QuotationEventInput{}, err
	}
	in := application.QuotationEventInput{EventKindID: b.EventKindID, EventDate: day, StartTime: b.StartTime, EndTime: b.EndTime,
		VenueID: b.VenueID, Layout: b.Layout}
	for _, s := range b.Sessions {
		in.Sessions = append(in.Sessions, application.QuotationSessionInput{Label: s.Label, StartTime: s.StartTime, EndTime: s.EndTime, Guests: s.Guests})
	}
	return in, nil
}

func childID(w http.ResponseWriter, raw, notFoundMsg string) (int64, bool) {
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		response.Error(w, http.StatusNotFound, notFoundMsg, nil)
		return 0, false
	}
	return id, true
}

// events is POST /events, POST /events/order {ids}, PUT and DELETE
// /events/{eventId}.
func (h *QuotationHandler) events(w http.ResponseWriter, r *http.Request, tenantID, id int64, rest []string) {
	ctx, v := r.Context(), viewer(r)
	switch {
	case len(rest) == 0:
		h.only(w, r, true, http.MethodPost, func() {
			var body eventBody
			if !decode(w, r, &body) {
				return
			}
			in, err := body.input()
			if err != nil {
				response.FromError(w, err)
				return
			}
			if _, err := h.svc.CreateEvent(ctx, tenantID, id, v, in); err != nil {
				response.FromError(w, err)
				return
			}
			h.respondDetail(w, r, tenantID, id, http.StatusCreated, "Acara ditambahkan")
		})
	case len(rest) == 1 && rest[0] == "order":
		h.only(w, r, true, http.MethodPost, func() {
			var body struct {
				IDs []int64 `json:"ids"`
			}
			if !decode(w, r, &body) {
				return
			}
			if err := h.svc.SetEventOrder(ctx, tenantID, id, v, body.IDs); err != nil {
				response.FromError(w, err)
				return
			}
			h.respondDetail(w, r, tenantID, id, http.StatusOK, "Urutan acara disimpan")
		})
	case len(rest) == 1:
		eventID, ok := childID(w, rest[0], application.MsgEventNotFound)
		if !ok {
			return
		}
		switch r.Method {
		case http.MethodPut:
			expected, ok := ifMatch(w, r, "acara", func() error {
				wc, err := h.svc.Content(ctx, tenantID, id, v)
				if err != nil {
					return err
				}
				for _, e := range wc.Events {
					if e.ID == eventID {
						return nil
					}
				}
				return apperror.NotFound(application.MsgEventNotFound)
			})
			if !ok {
				return
			}
			var body eventBody
			if !decode(w, r, &body) {
				return
			}
			in, err := body.input()
			if err != nil {
				response.FromError(w, err)
				return
			}
			if _, err := h.svc.UpdateEvent(ctx, tenantID, id, eventID, v, in, expected); err != nil {
				response.FromError(w, h.stale(r, tenantID, id, err))
				return
			}
			h.respondDetail(w, r, tenantID, id, http.StatusOK, "Acara disimpan")
		case http.MethodDelete:
			if err := h.svc.DeleteEvent(ctx, tenantID, id, eventID, v); err != nil {
				response.FromError(w, err)
				return
			}
			h.respondDetail(w, r, tenantID, id, http.StatusOK, "Acara dihapus")
		default:
			methodNotAllowed(w)
		}
	default:
		notFound(w)
	}
}

// ---- suppliers ----------------------------------------------------------------------

type supplierBody struct {
	VendorID   int64        `json:"vendorId"`
	PaidBy     domain.Payer `json:"paidBy"`
	ShowName   bool         `json:"showName"`
	AgreedCost *int64       `json:"agreedCost"`
	Notes      string       `json:"notes"`
}

func (b supplierBody) input() application.QuotationSupplierInput {
	return application.QuotationSupplierInput{VendorID: b.VendorID, PaidBy: b.PaidBy, ShowName: b.ShowName, AgreedCost: b.AgreedCost, Notes: b.Notes}
}

// suppliers is POST /suppliers (a vendor; venues come from events), PUT and
// DELETE /suppliers/{supplierId}, and POST /suppliers/{supplierId}/refresh
// (re-read name, category, and capacities from the directory).
func (h *QuotationHandler) suppliers(w http.ResponseWriter, r *http.Request, tenantID, id int64, rest []string) {
	ctx, v := r.Context(), viewer(r)
	if len(rest) == 0 {
		h.only(w, r, true, http.MethodPost, func() {
			var body supplierBody
			if !decode(w, r, &body) {
				return
			}
			if _, err := h.svc.CreateSupplier(ctx, tenantID, id, v, body.input()); err != nil {
				response.FromError(w, err)
				return
			}
			h.respondDetail(w, r, tenantID, id, http.StatusCreated, "Vendor ditambahkan")
		})
		return
	}
	supplierID, ok := childID(w, rest[0], application.MsgSupplierNotFound)
	if !ok {
		return
	}
	if len(rest) == 2 && rest[1] == "refresh" {
		h.only(w, r, true, http.MethodPost, func() {
			if _, err := h.svc.RefreshSupplier(ctx, tenantID, id, supplierID, v); err != nil {
				response.FromError(w, err)
				return
			}
			h.respondDetail(w, r, tenantID, id, http.StatusOK, "Data vendor diperbarui dari direktori")
		})
		return
	}
	if len(rest) != 1 {
		notFound(w)
		return
	}
	switch r.Method {
	case http.MethodPut:
		expected, ok := ifMatch(w, r, "vendor", func() error {
			wc, err := h.svc.Content(ctx, tenantID, id, v)
			if err != nil {
				return err
			}
			for _, s := range wc.Suppliers {
				if s.ID == supplierID {
					return nil
				}
			}
			return apperror.NotFound(application.MsgSupplierNotFound)
		})
		if !ok {
			return
		}
		var body supplierBody
		if !decode(w, r, &body) {
			return
		}
		if _, err := h.svc.UpdateSupplier(ctx, tenantID, id, supplierID, v, body.input(), expected); err != nil {
			response.FromError(w, h.stale(r, tenantID, id, err))
			return
		}
		h.respondDetail(w, r, tenantID, id, http.StatusOK, "Vendor disimpan")
	case http.MethodDelete:
		if err := h.svc.DeleteSupplier(ctx, tenantID, id, supplierID, v); err != nil {
			response.FromError(w, err)
			return
		}
		h.respondDetail(w, r, tenantID, id, http.StatusOK, "Vendor dihapus dari penawaran")
	default:
		methodNotAllowed(w)
	}
}

// ---- lines and terms ------------------------------------------------------------------

type lineComponentBody struct {
	Description string `json:"description"`
	Details     string `json:"details"`
	CategoryID  *int64 `json:"categoryId"`
}

type lineBody struct {
	ID                 int64               `json:"id"`
	Kind               domain.LineKind     `json:"kind"`
	PackageID          *int64              `json:"packageId"`
	RefreshFromCatalog bool                `json:"refreshFromCatalog"`
	Components         []lineComponentBody `json:"components"`
	Description        string              `json:"description"`
	Details            string              `json:"details"`
	CategoryID         *int64              `json:"categoryId"`
	EventID            *int64              `json:"eventId"`
	Quantity           *int64              `json:"quantity"`
	Unit               *string             `json:"unit"`
	UnitPrice          *int64              `json:"unitPrice"`
	Amount             *int64              `json:"amount"`
	IsBonus            bool                `json:"isBonus"`
	SupplierRef        *int64              `json:"supplierRef"`
}

type linesBody struct {
	Lines []lineBody `json:"lines"`
}

func (b linesBody) inputs() []application.QuotationLineInput {
	out := make([]application.QuotationLineInput, 0, len(b.Lines))
	for _, l := range b.Lines {
		in := application.QuotationLineInput{ID: l.ID, Kind: l.Kind, PackageID: l.PackageID, RefreshFromCatalog: l.RefreshFromCatalog,
			Description: l.Description, Details: l.Details, CategoryID: l.CategoryID, EventID: l.EventID, Quantity: l.Quantity,
			Unit: l.Unit, UnitPrice: l.UnitPrice, Amount: l.Amount, IsBonus: l.IsBonus, SupplierRef: l.SupplierRef}
		if l.Components != nil {
			// Absent = keep the package's components; [] = none.
			in.Components = make([]application.QuotationComponentInput, 0, len(l.Components))
			for _, c := range l.Components {
				in.Components = append(in.Components, application.QuotationComponentInput{Description: c.Description, Details: c.Details, CategoryID: c.CategoryID})
			}
		}
		out = append(out, in)
	}
	return out
}

// saveLines is PUT /lines ("Simpan rincian"): every top-level line in order,
// with If-Match linesUpdatedAt.
func (h *QuotationHandler) saveLines(w http.ResponseWriter, r *http.Request, tenantID, id int64) {
	expected, ok := ifMatch(w, r, "rincian (linesUpdatedAt)", h.exists(r, tenantID, id))
	if !ok {
		return
	}
	var body linesBody
	if !decode(w, r, &body) {
		return
	}
	if _, err := h.svc.SaveLines(r.Context(), tenantID, id, viewer(r), body.inputs(), expected); err != nil {
		response.FromError(w, h.stale(r, tenantID, id, err))
		return
	}
	h.respondDetail(w, r, tenantID, id, http.StatusOK, "Rincian disimpan")
}

// previewLines is POST /lines/preview: the lines priced as they would be
// saved (package extra guests, totals). Stores nothing.
func (h *QuotationHandler) previewLines(w http.ResponseWriter, r *http.Request, tenantID, id int64) {
	var body linesBody
	if !decode(w, r, &body) {
		return
	}
	res, err := h.svc.PreviewLines(r.Context(), tenantID, id, viewer(r), body.inputs())
	if err != nil {
		response.FromError(w, err)
		return
	}
	response.OK(w, "ok", toLinesPreview(res))
}

type termBody struct {
	Label         string          `json:"label"`
	Percent       *int            `json:"percent"`
	Amount        *int64          `json:"amount"`
	DueBasis      domain.DueBasis `json:"dueBasis"`
	DueOffsetDays int             `json:"dueOffsetDays"`
	DueEventID    *int64          `json:"dueEventId"`
}

// saveTerms is PUT /terms: the payment terms in order, with If-Match
// termsUpdatedAt.
func (h *QuotationHandler) saveTerms(w http.ResponseWriter, r *http.Request, tenantID, id int64) {
	expected, ok := ifMatch(w, r, "termin (termsUpdatedAt)", h.exists(r, tenantID, id))
	if !ok {
		return
	}
	var body struct {
		Terms []termBody `json:"terms"`
	}
	if !decode(w, r, &body) {
		return
	}
	in := make([]application.QuotationTermInput, 0, len(body.Terms))
	for _, t := range body.Terms {
		in = append(in, application.QuotationTermInput{Label: t.Label, Percent: t.Percent, Amount: t.Amount, DueBasis: t.DueBasis,
			DueOffsetDays: t.DueOffsetDays, DueEventID: t.DueEventID})
	}
	if _, err := h.svc.SaveTerms(r.Context(), tenantID, id, viewer(r), in, expected); err != nil {
		response.FromError(w, h.stale(r, tenantID, id, err))
		return
	}
	h.respondDetail(w, r, tenantID, id, http.StatusOK, "Termin disimpan")
}

// ---- revisions -------------------------------------------------------------------------

func revisionStaff(revs []domain.QuotationRevision) []int64 {
	var ids []int64
	for _, rev := range revs {
		if rev.CreatedByStaffID != nil {
			ids = append(ids, *rev.CreatedByStaffID)
		}
	}
	return ids
}

// revisions is GET /revisions (newest first), POST (freeze the working copy:
// 201 when a revision was made, 200 with the existing one when the content
// is unchanged), GET /revisions/{n}, GET /revisions/{n}/pdf, and POST
// /revisions/{n}/share.
func (h *QuotationHandler) revisions(w http.ResponseWriter, r *http.Request, tenantID, id int64, rest []string) {
	ctx, v := r.Context(), viewer(r)
	if len(rest) == 0 {
		switch r.Method {
		case http.MethodGet:
			q, err := h.svc.Get(ctx, tenantID, id, v)
			if err != nil {
				response.FromError(w, err)
				return
			}
			revs, err := h.svc.ListRevisions(ctx, tenantID, id, v)
			if err != nil {
				response.FromError(w, err)
				return
			}
			n, err := h.names(ctx, tenantID, revisionStaff(revs), nil)
			if err != nil {
				response.FromError(w, err)
				return
			}
			out := make([]revisionResponse, 0, len(revs))
			for _, rev := range revs {
				out = append(out, toRevisionResponse(rev, q, n, false, v.CanSeeCost))
			}
			response.OK(w, "ok", out)
		case http.MethodPost:
			rev, created, err := h.svc.CreateRevision(ctx, tenantID, id, v)
			if err != nil {
				response.FromError(w, err)
				return
			}
			q, err := h.svc.Get(ctx, tenantID, id, v)
			if err != nil {
				response.FromError(w, err)
				return
			}
			n, err := h.names(ctx, tenantID, revisionStaff([]domain.QuotationRevision{*rev}), nil)
			if err != nil {
				response.FromError(w, err)
				return
			}
			body := toRevisionResponse(*rev, q, n, false, v.CanSeeCost)
			if created {
				response.Created(w, fmt.Sprintf("Revisi %d dibuat", rev.N), body)
				return
			}
			response.OK(w, fmt.Sprintf("Tidak ada perubahan sejak Revisi %d", rev.N), body)
		default:
			methodNotAllowed(w)
		}
		return
	}
	revN, err := strconv.Atoi(rest[0])
	if err != nil || revN <= 0 {
		response.Error(w, http.StatusNotFound, application.MsgRevisionNotFound, nil)
		return
	}
	switch {
	case len(rest) == 1:
		h.only(w, r, true, http.MethodGet, func() {
			rev, err := h.svc.GetRevision(ctx, tenantID, id, revN, v)
			if err != nil {
				response.FromError(w, err)
				return
			}
			q, err := h.svc.Get(ctx, tenantID, id, v)
			if err != nil {
				response.FromError(w, err)
				return
			}
			n, err := h.names(ctx, tenantID, revisionStaff([]domain.QuotationRevision{*rev}), nil)
			if err != nil {
				response.FromError(w, err)
				return
			}
			response.OK(w, "ok", toRevisionResponse(*rev, q, n, true, v.CanSeeCost))
		})
	case len(rest) == 2 && rest[1] == "pdf":
		h.only(w, r, true, http.MethodGet, func() {
			data, name, err := h.svc.RevisionPDF(ctx, tenantID, id, revN, v)
			if err != nil {
				response.FromError(w, err)
				return
			}
			disposition := "attachment"
			if r.URL.Query().Get("inline") == "true" {
				disposition = "inline"
			}
			writePDF(w, data, name, disposition)
		})
	case len(rest) == 2 && rest[1] == "share":
		h.only(w, r, true, http.MethodPost, func() {
			if err := h.svc.ShareRevision(ctx, tenantID, id, revN, v); err != nil {
				response.FromError(w, err)
				return
			}
			response.OK(w, "Pengiriman dicatat", nil)
		})
	default:
		notFound(w)
	}
}

func writePDF(w http.ResponseWriter, data []byte, name, disposition string) {
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.Header().Set("Content-Disposition", fmt.Sprintf("%s; filename=%q", disposition, name))
	w.Header().Set("Cache-Control", "private, no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

// previewPDF is GET /preview.pdf: the working copy with a "DRAF" watermark.
// It never makes a revision.
func (h *QuotationHandler) previewPDF(w http.ResponseWriter, r *http.Request, tenantID, id int64) {
	data, name, err := h.svc.PreviewPDF(r.Context(), tenantID, id, viewer(r))
	if err != nil {
		response.FromError(w, err)
		return
	}
	writePDF(w, data, name, "inline")
}

// compare is GET /compare?from=n&to=m (to omitted or 0 = the working copy).
func (h *QuotationHandler) compare(w http.ResponseWriter, r *http.Request, tenantID, id int64) {
	q := r.URL.Query()
	from, err := strconv.Atoi(q.Get("from"))
	if err != nil || from <= 0 {
		response.FromError(w, apperror.Validation(application.MsgCompareInvalid, map[string][]string{"from": {"Pilih revisi pembanding"}}))
		return
	}
	to := 0
	if raw := q.Get("to"); raw != "" {
		if to, err = strconv.Atoi(raw); err != nil || to < 0 {
			response.FromError(w, apperror.Validation(application.MsgCompareInvalid, map[string][]string{"to": {"Revisi tidak valid"}}))
			return
		}
	}
	v := viewer(r)
	res, err := h.svc.Compare(r.Context(), tenantID, id, from, to, v)
	if err != nil {
		response.FromError(w, err)
		return
	}
	response.OK(w, "ok", toCompareResponse(res, v.CanSeeCost))
}

// ---- status, copies, validity ----------------------------------------------------------

type statusBody struct {
	Status      domain.QuotationStatus `json:"status"`
	RevisionN   *int                   `json:"revisionN"`
	AcceptedOn  *string                `json:"acceptedOn"`
	Channel     domain.AcceptChannel   `json:"channel"`
	ExtendUntil *string                `json:"extendUntil"`
	Reason      string                 `json:"reason"`
}

var statusMessages = map[domain.QuotationStatus]string{
	domain.QuotationAccepted:   "Penawaran ditandai diterima",
	domain.QuotationRejected:   "Penawaran ditandai ditolak",
	domain.QuotationDraft:      "Penawaran kembali menjadi Draf",
	domain.QuotationSuperseded: "Penawaran ditandai digantikan",
}

// setStatus is POST /status: one explicit status change (never a toggle).
func (h *QuotationHandler) setStatus(w http.ResponseWriter, r *http.Request, tenantID, id int64) {
	var body statusBody
	if !decode(w, r, &body) {
		return
	}
	if !domain.ValidStoredStatus(body.Status) {
		response.FromError(w, apperror.Validation("Status tidak valid", map[string][]string{"status": {"status harus Draft, Accepted, Rejected, atau Superseded"}}))
		return
	}
	acceptedOn, err := parseDay("acceptedOn", body.AcceptedOn)
	if err != nil {
		response.FromError(w, err)
		return
	}
	extendUntil, err := parseDay("extendUntil", body.ExtendUntil)
	if err != nil {
		response.FromError(w, err)
		return
	}
	in := application.StatusInput{Status: body.Status, RevisionN: body.RevisionN, AcceptedOn: acceptedOn, Channel: body.Channel,
		ExtendUntil: extendUntil, Reason: body.Reason}
	if _, err := h.svc.SetStatus(r.Context(), tenantID, id, viewer(r), in); err != nil {
		response.FromError(w, err)
		return
	}
	h.respondDetail(w, r, tenantID, id, http.StatusOK, statusMessages[body.Status])
}

// duplicate is POST /copy (a new Draft with a new number; the original, when
// still a Draft, becomes Superseded) and POST /alternatives (a new Draft in
// the same group; nothing is superseded until one is accepted).
func (h *QuotationHandler) duplicate(w http.ResponseWriter, r *http.Request, tenantID, id int64, alternative bool) {
	var (
		q   *domain.Quotation
		err error
		msg = "Penawaran disalin"
	)
	if alternative {
		q, err = h.svc.AddAlternative(r.Context(), tenantID, id, viewer(r))
		msg = "Alternatif penawaran dibuat"
	} else {
		q, err = h.svc.Copy(r.Context(), tenantID, id, viewer(r))
	}
	if err != nil {
		response.FromError(w, err)
		return
	}
	h.respondDetail(w, r, tenantID, q.ID, http.StatusCreated, msg)
}

// extend is POST /extend {validUntil}.
func (h *QuotationHandler) extend(w http.ResponseWriter, r *http.Request, tenantID, id int64) {
	var body struct {
		ValidUntil *string `json:"validUntil"`
	}
	if !decode(w, r, &body) {
		return
	}
	until, err := parseDay("validUntil", body.ValidUntil)
	if err != nil {
		response.FromError(w, err)
		return
	}
	if until == nil {
		response.FromError(w, apperror.Validation("Isi tanggal berlaku sampai", map[string][]string{"validUntil": {"Isi tanggal berlaku sampai"}}))
		return
	}
	if _, err := h.svc.Extend(r.Context(), tenantID, id, viewer(r), *until); err != nil {
		response.FromError(w, err)
		return
	}
	h.respondDetail(w, r, tenantID, id, http.StatusOK, "Masa berlaku diperpanjang")
}

// ---- acceptance evidence ----------------------------------------------------------------

type attachmentBody struct {
	FileName   string `json:"fileName"`
	MimeType   string `json:"mimeType"`
	Base64Data string `json:"base64Data"`
}

// putAttachment is PUT /acceptance-attachment (If-Match updatedAt): the
// photo or PDF of the signed quotation, replacing an earlier one.
func (h *QuotationHandler) putAttachment(w http.ResponseWriter, r *http.Request, tenantID, id int64) {
	expected, ok := ifMatch(w, r, "penawaran", h.exists(r, tenantID, id))
	if !ok {
		return
	}
	// base64 is 4/3 of the file, plus the JSON around it.
	r.Body = http.MaxBytesReader(w, r.Body, int64(application.MaxAcceptanceAttachmentMB)<<20*4/3+64<<10)
	var body attachmentBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			response.FromError(w, apperror.Validation(application.MsgAttachmentTooLarge, map[string][]string{"base64Data": {application.MsgAttachmentTooLarge}}))
			return
		}
		response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
		return
	}
	data, err := base64.StdEncoding.DecodeString(body.Base64Data)
	if err != nil {
		response.FromError(w, apperror.Validation(application.MsgAttachmentUnreadable, map[string][]string{"base64Data": {application.MsgAttachmentUnreadable}}))
		return
	}
	in := application.AttachmentInput{FileName: body.FileName, MimeType: body.MimeType, Data: data}
	if _, err := h.svc.SetAcceptanceAttachment(r.Context(), tenantID, id, viewer(r), in, expected); err != nil {
		response.FromError(w, h.stale(r, tenantID, id, err))
		return
	}
	h.respondDetail(w, r, tenantID, id, http.StatusOK, "Lampiran penerimaan disimpan")
}

// getAttachment is GET /acceptance-attachment.
func (h *QuotationHandler) getAttachment(w http.ResponseWriter, r *http.Request, tenantID, id int64) {
	rc, name, mime, err := h.svc.AcceptanceAttachment(r.Context(), tenantID, id, viewer(r))
	if err != nil {
		response.FromError(w, err)
		return
	}
	defer rc.Close()
	w.Header().Set("Content-Type", mime)
	w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=%q", name))
	w.Header().Set("Cache-Control", "private, no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, rc)
}

// activity is GET /activity: the quotation's history, newest first.
func (h *QuotationHandler) activity(w http.ResponseWriter, r *http.Request, tenantID, id int64) {
	list, err := h.svc.Activity(r.Context(), tenantID, id, viewer(r))
	if err != nil {
		response.FromError(w, err)
		return
	}
	var ids []int64
	for _, a := range list {
		if a.ActorStaffID != nil {
			ids = append(ids, *a.ActorStaffID)
		}
	}
	n, err := h.names(r.Context(), tenantID, ids, nil)
	if err != nil {
		response.FromError(w, err)
		return
	}
	out := make([]activityResponse, 0, len(list))
	for _, a := range list {
		out = append(out, activityResponse{ID: a.ID, Type: a.Type, ActorID: a.ActorStaffID, ActorName: n.staffName(a.ActorStaffID),
			Summary: a.Summary, CreatedAt: etag.Format(a.CreatedAt)})
	}
	response.OK(w, "ok", out)
}
