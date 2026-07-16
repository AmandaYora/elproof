package presentation

import (
	"encoding/json"
	"net/http"
	"strconv"

	"elproof/internal/modules/vendors/application"
	"elproof/internal/modules/vendors/domain"
	"elproof/internal/shared/apperror"
	"elproof/internal/shared/httpx"
	"elproof/internal/shared/middleware"
	"elproof/internal/shared/pagination"
	"elproof/internal/shared/response"
)

type VendorCategoryHandler struct {
	categories *application.VendorCategoryService
}

func NewVendorCategoryHandler(categories *application.VendorCategoryService) *VendorCategoryHandler {
	return &VendorCategoryHandler{categories: categories}
}

type vendorCategoryResponse struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	IsActive    bool   `json:"isActive"`
	CreatedAt   string `json:"createdAt"`
}

func toVendorCategoryResponse(c domain.VendorCategory) vendorCategoryResponse {
	return vendorCategoryResponse{
		ID: c.ID, Name: c.Name, Description: c.Description, IsActive: c.IsActive,
		CreatedAt: c.CreatedAt.Format("2006-01-02"),
	}
}

func requireTenant(w http.ResponseWriter, r *http.Request) (int64, bool) {
	claims, ok := middleware.FromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "Tidak terautentikasi", nil)
		return 0, false
	}
	tenantID, ok := claims.TenantIDInt()
	if !ok {
		response.Error(w, http.StatusForbidden, "Akun ini tidak terikat ke tenant manapun", nil)
		return 0, false
	}
	return tenantID, true
}

func (h *VendorCategoryHandler) Collection(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := requireTenant(w, r)
	if !ok {
		return
	}
	switch r.Method {
	case http.MethodGet:
		if r.URL.Query().Get("all") == "true" {
			categories, err := h.categories.List(r.Context(), tenantID)
			if err != nil {
				writeAppError(w, err)
				return
			}
			result := make([]vendorCategoryResponse, 0, len(categories))
			for _, c := range categories {
				result = append(result, toVendorCategoryResponse(c))
			}
			response.OK(w, "ok", result)
			return
		}
		params := pagination.FromRequest(r)
		search := r.URL.Query().Get("search")
		categories, total, err := h.categories.ListPaginated(r.Context(), tenantID, params, search)
		if err != nil {
			writeAppError(w, err)
			return
		}
		result := make([]vendorCategoryResponse, 0, len(categories))
		for _, c := range categories {
			result = append(result, toVendorCategoryResponse(c))
		}
		response.OKPaginated(w, "ok", result, pagination.BuildMeta(params, total))
	case http.MethodPost:
		h.create(w, r, tenantID)
	default:
		response.Error(w, http.StatusMethodNotAllowed, "Metode HTTP tidak diizinkan untuk endpoint ini", nil)
	}
}

type vendorCategoryInputBody struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (h *VendorCategoryHandler) create(w http.ResponseWriter, r *http.Request, tenantID int64) {
	var body vendorCategoryInputBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
		return
	}
	category, err := h.categories.Create(r.Context(), tenantID, application.VendorCategoryInput{Name: body.Name, Description: body.Description})
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.Created(w, "Kategori vendor berhasil dibuat", toVendorCategoryResponse(*category))
}

func (h *VendorCategoryHandler) Item(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := requireTenant(w, r)
	if !ok {
		return
	}
	segments := httpx.Segments(r.URL.Path, "/api/v1/vendor-categories/")
	if len(segments) == 0 {
		response.Error(w, http.StatusNotFound, "Kategori vendor tidak ditemukan", nil)
		return
	}
	id, err := strconv.ParseInt(segments[0], 10, 64)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "ID tidak valid", nil)
		return
	}

	switch {
	case len(segments) == 1 && r.Method == http.MethodPatch:
		var body vendorCategoryInputBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
			return
		}
		category, err := h.categories.Update(r.Context(), tenantID, id, application.VendorCategoryInput{Name: body.Name, Description: body.Description})
		if err != nil {
			writeAppError(w, err)
			return
		}
		response.OK(w, "Kategori vendor berhasil diperbarui", toVendorCategoryResponse(*category))
	case len(segments) == 2 && segments[1] == "toggle-active" && r.Method == http.MethodPost:
		current, err := h.categories.Get(r.Context(), tenantID, id)
		if err != nil {
			writeAppError(w, err)
			return
		}
		category, err := h.categories.SetActive(r.Context(), tenantID, id, !current.IsActive)
		if err != nil {
			writeAppError(w, err)
			return
		}
		response.OK(w, "Status kategori vendor diperbarui", toVendorCategoryResponse(*category))
	default:
		response.Error(w, http.StatusNotFound, "Endpoint tidak ditemukan", nil)
	}
}

func writeAppError(w http.ResponseWriter, err error) {
	status := apperror.HTTPStatus(err)
	if appErr, ok := apperror.As(err); ok {
		if appErr.Kind == apperror.KindValidation {
			response.Error(w, status, appErr.Message, appErr.Fields)
			return
		}
		response.Error(w, status, appErr.Message, nil)
		return
	}
	response.Error(w, status, "Terjadi kesalahan pada server", nil)
}
