package presentation

import (
	"encoding/json"
	"net/http"

	"elproof/internal/modules/offers/application"
	"elproof/internal/modules/offers/domain"
	"elproof/internal/shared/apperror"
	"elproof/internal/shared/etag"
	"elproof/internal/shared/httpx"
	"elproof/internal/shared/pagination"
	"elproof/internal/shared/response"
)

// PermMarginView is the permission that shows (and lets one type) the
// estimated cost of a package component (PLAN §4.3).
const PermMarginView = "finance.margin.view"

type PackageHandler struct {
	packages *application.PackageService
}

func NewPackageHandler(packages *application.PackageService) *PackageHandler {
	return &PackageHandler{packages: packages}
}

type namedRef struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type componentResponse struct {
	ID           int64   `json:"id"`
	Description  string  `json:"description"`
	CategoryID   *int64  `json:"categoryId"`
	CategoryName *string `json:"categoryName"`
	VendorID     *int64  `json:"vendorId"`
	VendorName   *string `json:"vendorName"`
	// EstimatedCost is left out entirely without finance.margin.view.
	EstimatedCost *int64 `json:"estimatedCost,omitempty"`
}

type packageResponse struct {
	ID                  int64               `json:"id"`
	Name                string              `json:"name"`
	Price               int64               `json:"price"`
	IncludedGuests      *int                `json:"includedGuests"`
	ExtraGuestStep      *int                `json:"extraGuestStep"`
	ExtraGuestStepPrice *int64              `json:"extraGuestStepPrice"`
	DefaultVenue        *namedRef           `json:"defaultVenue"`
	EventKinds          []namedRef          `json:"eventKinds"`
	Components          []componentResponse `json:"components"`
	ComponentCount      int                 `json:"componentCount"`
	IsActive            bool                `json:"isActive"`
	CreatedAt           string              `json:"createdAt"`
	UpdatedAt           string              `json:"updatedAt"`
}

func (r packageResponse) ETag() string { return quoted(r.UpdatedAt) }

func nameOf(m map[int64]string, id *int64) *string {
	if id == nil {
		return nil
	}
	if n, ok := m[*id]; ok {
		return &n
	}
	return nil
}

func toPackageResponse(p domain.Package, n application.Names, showCost bool) packageResponse {
	out := packageResponse{
		ID: p.ID, Name: p.Name, Price: p.Price, IncludedGuests: p.IncludedGuests,
		ExtraGuestStep: p.ExtraGuestStep, ExtraGuestStepPrice: p.ExtraGuestStepPrice,
		EventKinds: []namedRef{}, Components: []componentResponse{}, ComponentCount: len(p.Components),
		IsActive: p.IsActive, CreatedAt: p.CreatedAt.Format("2006-01-02"), UpdatedAt: etag.Format(p.UpdatedAt),
	}
	if p.DefaultVenueID != nil {
		ref := namedRef{ID: *p.DefaultVenueID}
		if name := nameOf(n.Venues, p.DefaultVenueID); name != nil {
			ref.Name = *name
		}
		out.DefaultVenue = &ref
	}
	for _, id := range p.EventKindIDs {
		out.EventKinds = append(out.EventKinds, namedRef{ID: id, Name: n.Kinds[id]})
	}
	for _, c := range p.Components {
		cr := componentResponse{ID: c.ID, Description: c.Description, CategoryID: c.CategoryID, CategoryName: nameOf(n.Categories, c.CategoryID),
			VendorID: c.VendorID, VendorName: nameOf(n.Vendors, c.VendorID)}
		if showCost {
			cr.EstimatedCost = c.EstimatedCost
		}
		out.Components = append(out.Components, cr)
	}
	return out
}

type componentBody struct {
	ID            int64  `json:"id"`
	Description   string `json:"description"`
	CategoryID    *int64 `json:"categoryId"`
	VendorID      *int64 `json:"vendorId"`
	EstimatedCost *int64 `json:"estimatedCost"`
}

type packageBody struct {
	Name                string          `json:"name"`
	Price               int64           `json:"price"`
	EventKindIDs        []int64         `json:"eventKindIds"`
	IncludedGuests      *int            `json:"includedGuests"`
	ExtraGuestStep      *int            `json:"extraGuestStep"`
	ExtraGuestStepPrice *int64          `json:"extraGuestStepPrice"`
	DefaultVenueID      *int64          `json:"defaultVenueId"`
	Components          []componentBody `json:"components"`
}

func (b packageBody) input() application.PackageInput {
	in := application.PackageInput{Name: b.Name, Price: b.Price, EventKindIDs: b.EventKindIDs, IncludedGuests: b.IncludedGuests,
		ExtraGuestStep: b.ExtraGuestStep, ExtraGuestStepPrice: b.ExtraGuestStepPrice, DefaultVenueID: b.DefaultVenueID}
	for _, c := range b.Components {
		in.Components = append(in.Components, application.ComponentInput{ID: c.ID, Description: c.Description,
			CategoryID: c.CategoryID, VendorID: c.VendorID, EstimatedCost: c.EstimatedCost})
	}
	return in
}

// respond writes one package with its names resolved.
func (h *PackageHandler) respond(w http.ResponseWriter, r *http.Request, tenantID int64, status int, msg string, p *domain.Package) {
	names, err := h.packages.Names(r.Context(), tenantID, []domain.Package{*p})
	if err != nil {
		response.FromError(w, err)
		return
	}
	body := toPackageResponse(*p, names, can(r, PermMarginView))
	if status == http.StatusCreated {
		response.Created(w, msg, body)
		return
	}
	response.OK(w, msg, body)
}

// stale swaps a Stale conflict's payload for the response shape.
func (h *PackageHandler) stale(r *http.Request, tenantID int64, err error) error {
	current, ok := apperror.IsStale(err)
	if !ok {
		return err
	}
	p, ok := current.(*domain.Package)
	if !ok {
		return err
	}
	names, nerr := h.packages.Names(r.Context(), tenantID, []domain.Package{*p})
	if nerr != nil {
		return nerr
	}
	ae, _ := apperror.As(err)
	return ae.WithData(toPackageResponse(*p, names, can(r, PermMarginView)))
}

func isActiveFilter(r *http.Request) (*bool, error) {
	switch raw := r.URL.Query().Get("isActive"); raw {
	case "":
		return nil, nil
	case "true", "false":
		v := raw == "true"
		return &v, nil
	default:
		return nil, apperror.Validation("Filter tidak valid", map[string][]string{"isActive": {"isActive harus true atau false"}})
	}
}

// Collection is GET /packages (any staff: packages are picked for quotations;
// sorted, filtered, and paged on the server) and POST (package.manage).
func (h *PackageHandler) Collection(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := requireStaffTenant(w, r)
	if !ok {
		return
	}
	switch r.Method {
	case http.MethodGet:
		active, err := isActiveFilter(r)
		if err != nil {
			response.FromError(w, err)
			return
		}
		params := pagination.FromRequest(r)
		columns, def := h.packages.ListSortSpec()
		sort, err := pagination.SortFromRequest(r, columns, def)
		if err != nil {
			response.FromError(w, err)
			return
		}
		list, total, err := h.packages.ListPaginated(r.Context(), tenantID, params, application.PackageListFilter{Search: pagination.SearchFromRequest(r), IsActive: active}, sort)
		if err != nil {
			response.FromError(w, err)
			return
		}
		names, err := h.packages.Names(r.Context(), tenantID, list)
		if err != nil {
			response.FromError(w, err)
			return
		}
		showCost := can(r, PermMarginView)
		out := make([]packageResponse, 0, len(list))
		for _, p := range list {
			out = append(out, toPackageResponse(p, names, showCost))
		}
		response.OKPaginated(w, "ok", out, pagination.BuildMeta(params, total).WithSort(sort))
	case http.MethodPost:
		if !requirePermission(w, r, PermManage) {
			return
		}
		var body packageBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
			return
		}
		p, err := h.packages.Create(r.Context(), tenantID, body.input(), can(r, PermMarginView))
		if err != nil {
			response.FromError(w, err)
			return
		}
		h.respond(w, r, tenantID, http.StatusCreated, "Paket disimpan", p)
	default:
		methodNotAllowed(w)
	}
}

type previewBody struct {
	Price               int64  `json:"price"`
	IncludedGuests      *int   `json:"includedGuests"`
	ExtraGuestStep      *int   `json:"extraGuestStep"`
	ExtraGuestStepPrice *int64 `json:"extraGuestStepPrice"`
	Guests              int    `json:"guests"`
}

type previewResponse struct {
	Price int64                    `json:"price"`
	Extra domain.ExtraGuestsResult `json:"extra"`
	Total int64                    `json:"total"`
}

// Preview is POST /packages/preview: the "Contoh hitung" box of the package
// form, priced by the same rule quotations use. Stores nothing.
func (h *PackageHandler) Preview(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireStaffTenant(w, r); !ok || !requirePermission(w, r, PermManage) {
		return
	}
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	var body previewBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
		return
	}
	p, err := h.packages.Preview(application.PreviewInput{Price: body.Price, IncludedGuests: body.IncludedGuests,
		ExtraGuestStep: body.ExtraGuestStep, ExtraGuestStepPrice: body.ExtraGuestStepPrice, Guests: body.Guests})
	if err != nil {
		response.FromError(w, err)
		return
	}
	response.OK(w, "ok", previewResponse{Price: body.Price, Extra: p.Extra, Total: p.Total})
}

// Item is GET /packages/{id} (any staff), PATCH ({"isActive"} alone sets the
// state; a full body edits with If-Match), and DELETE (package.manage).
func (h *PackageHandler) Item(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := requireStaffTenant(w, r)
	if !ok {
		return
	}
	id, segments, ok := itemID(w, r, "/api/v1/packages/", application.MsgPackageNotFound)
	if !ok {
		return
	}
	if len(segments) != 1 {
		response.Error(w, http.StatusNotFound, "Endpoint tidak ditemukan", nil)
		return
	}
	if r.Method == http.MethodGet {
		p, err := h.packages.Get(r.Context(), tenantID, id)
		if err != nil {
			response.FromError(w, err)
			return
		}
		h.respond(w, r, tenantID, http.StatusOK, "ok", p)
		return
	}
	if !requirePermission(w, r, PermManage) {
		return
	}
	switch r.Method {
	case http.MethodPatch:
		sp, err := httpx.ReadStatePatch(r, "isActive")
		if err != nil {
			httpx.WriteStatePatchError(w, err)
			return
		}
		if sp.IsState {
			p, err := h.packages.SetActive(r.Context(), tenantID, id, sp.Value)
			if err != nil {
				response.FromError(w, err)
				return
			}
			msg := "Paket dinonaktifkan"
			if sp.Value {
				msg = "Paket diaktifkan"
			}
			h.respond(w, r, tenantID, http.StatusOK, msg, p)
			return
		}
		expected, ok := ifMatch(w, r, "paket", func() error { _, err := h.packages.Get(r.Context(), tenantID, id); return err })
		if !ok {
			return
		}
		var body packageBody
		if err := json.Unmarshal(sp.Body, &body); err != nil {
			response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
			return
		}
		p, err := h.packages.Update(r.Context(), tenantID, id, body.input(), expected, can(r, PermMarginView))
		if err != nil {
			response.FromError(w, h.stale(r, tenantID, err))
			return
		}
		h.respond(w, r, tenantID, http.StatusOK, "Paket disimpan", p)
	case http.MethodDelete:
		if err := h.packages.Delete(r.Context(), tenantID, id); err != nil {
			response.FromError(w, err)
			return
		}
		response.OK(w, "Paket dihapus", nil)
	default:
		methodNotAllowed(w)
	}
}
