package presentation

import (
	"encoding/json"
	"net/http"
	"strconv"

	projectscontracts "elproof/internal/modules/projects/contracts"
	"elproof/internal/modules/vendors/application"
	"elproof/internal/modules/vendors/domain"
	"elproof/internal/shared/httpx"
	"elproof/internal/shared/pagination"
	"elproof/internal/shared/response"
)

type VendorHandler struct {
	vendors  *application.VendorService
	projects projectscontracts.Contracts
}

func NewVendorHandler(vendors *application.VendorService, projects projectscontracts.Contracts) *VendorHandler {
	return &VendorHandler{vendors: vendors, projects: projects}
}

type vendorResponse struct {
	ID         int64  `json:"id"`
	CategoryID int64  `json:"categoryId"`
	Name       string `json:"name"`
	PICName    string `json:"picName"`
	Phone      string `json:"phone"`
	Email      string `json:"email"`
	Address    string `json:"address"`
	Notes      string `json:"notes"`
	IsActive   bool   `json:"isActive"`
	CreatedAt  string `json:"createdAt"`
}

func toVendorResponse(v domain.Vendor) vendorResponse {
	return vendorResponse{
		ID: v.ID, CategoryID: v.CategoryID, Name: v.Name, PICName: v.PICName,
		Phone: v.Phone, Email: v.Email, Address: v.Address, Notes: v.Notes, IsActive: v.IsActive,
		CreatedAt: v.CreatedAt.Format("2006-01-02"),
	}
}

func (h *VendorHandler) Collection(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := requireTenant(w, r)
	if !ok {
		return
	}
	switch r.Method {
	case http.MethodGet:
		var categoryID *int64
		if raw := r.URL.Query().Get("categoryId"); raw != "" {
			id, err := strconv.ParseInt(raw, 10, 64)
			if err != nil {
				response.Error(w, http.StatusBadRequest, "categoryId tidak valid", nil)
				return
			}
			categoryID = &id
		}
		if r.URL.Query().Get("all") == "true" {
			vendors, err := h.vendors.List(r.Context(), tenantID, categoryID)
			if err != nil {
				writeAppError(w, err)
				return
			}
			result := make([]vendorResponse, 0, len(vendors))
			for _, v := range vendors {
				result = append(result, toVendorResponse(v))
			}
			response.OK(w, "ok", result)
			return
		}
		params := pagination.FromRequest(r)
		search := r.URL.Query().Get("search")
		vendors, total, err := h.vendors.ListPaginated(r.Context(), tenantID, categoryID, params, search)
		if err != nil {
			writeAppError(w, err)
			return
		}
		result := make([]vendorResponse, 0, len(vendors))
		for _, v := range vendors {
			result = append(result, toVendorResponse(v))
		}
		response.OKPaginated(w, "ok", result, pagination.BuildMeta(params, total))
	case http.MethodPost:
		h.create(w, r, tenantID)
	default:
		response.Error(w, http.StatusMethodNotAllowed, "Metode HTTP tidak diizinkan untuk endpoint ini", nil)
	}
}

type vendorInputBody struct {
	Name       string `json:"name"`
	CategoryID int64  `json:"categoryId"`
	PICName    string `json:"picName"`
	Phone      string `json:"phone"`
	Email      string `json:"email"`
	Address    string `json:"address"`
	Notes      string `json:"notes"`
}

func toVendorInput(body vendorInputBody) application.VendorInput {
	return application.VendorInput{
		Name: body.Name, CategoryID: body.CategoryID, PICName: body.PICName,
		Phone: body.Phone, Email: body.Email, Address: body.Address, Notes: body.Notes,
	}
}

func (h *VendorHandler) create(w http.ResponseWriter, r *http.Request, tenantID int64) {
	var body vendorInputBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
		return
	}
	vendor, err := h.vendors.Create(r.Context(), tenantID, toVendorInput(body))
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.Created(w, "Vendor berhasil ditambahkan", toVendorResponse(*vendor))
}

func (h *VendorHandler) Item(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := requireTenant(w, r)
	if !ok {
		return
	}
	segments := httpx.Segments(r.URL.Path, "/api/v1/vendors/")
	if len(segments) == 0 {
		response.Error(w, http.StatusNotFound, "Vendor tidak ditemukan", nil)
		return
	}
	id, err := strconv.ParseInt(segments[0], 10, 64)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "ID tidak valid", nil)
		return
	}

	switch {
	case len(segments) == 1 && r.Method == http.MethodPatch:
		var body vendorInputBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
			return
		}
		vendor, err := h.vendors.Update(r.Context(), tenantID, id, toVendorInput(body))
		if err != nil {
			writeAppError(w, err)
			return
		}
		response.OK(w, "Vendor berhasil diperbarui", toVendorResponse(*vendor))
	case len(segments) == 2 && segments[1] == "toggle-active" && r.Method == http.MethodPost:
		current, err := h.vendors.Get(r.Context(), tenantID, id)
		if err != nil {
			writeAppError(w, err)
			return
		}
		vendor, err := h.vendors.SetActive(r.Context(), tenantID, id, !current.IsActive)
		if err != nil {
			writeAppError(w, err)
			return
		}
		response.OK(w, "Status vendor diperbarui", toVendorResponse(*vendor))
	case len(segments) == 2 && segments[1] == "project-history" && r.Method == http.MethodGet:
		h.projectHistory(w, r, tenantID, id)
	default:
		response.Error(w, http.StatusNotFound, "Endpoint tidak ditemukan", nil)
	}
}

type vendorProjectHistoryResponse struct {
	ProjectID        int64  `json:"projectId"`
	ProjectName      string `json:"projectName"`
	EventDate        string `json:"eventDate"`
	Venue            string `json:"venue"`
	EngagementStatus string `json:"engagementStatus"`
}

// projectHistory backs "Lihat Project" — a vendor's engagement history
// across every project in the tenant. `project_vendors` is owned by
// `projects`, not this module, so the actual query is resolved through
// `projects.Contracts` (see vendors.module.go's doc comment).
func (h *VendorHandler) projectHistory(w http.ResponseWriter, r *http.Request, tenantID, id int64) {
	if _, err := h.vendors.Get(r.Context(), tenantID, id); err != nil {
		writeAppError(w, err)
		return
	}
	rows, err := h.projects.ListVendorEngagementHistory(r.Context(), tenantID, id)
	if err != nil {
		writeAppError(w, err)
		return
	}
	result := make([]vendorProjectHistoryResponse, 0, len(rows))
	for _, row := range rows {
		result = append(result, vendorProjectHistoryResponse{
			ProjectID: row.ProjectID, ProjectName: row.ProjectName,
			EventDate: row.EventDate.Format("2006-01-02"), Venue: row.Venue, EngagementStatus: row.EngagementStatus,
		})
	}
	response.OK(w, "ok", result)
}
