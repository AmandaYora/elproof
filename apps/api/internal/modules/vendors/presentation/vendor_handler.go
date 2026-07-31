package presentation

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"

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
	ID                int64   `json:"id"`
	CategoryID        int64   `json:"categoryId"`
	Name              string  `json:"name"`
	PICName           string  `json:"picName"`
	Phone             string  `json:"phone"`
	Email             *string `json:"email"`
	SocialMedia       *string `json:"socialMedia"`
	City              *string `json:"city"`
	Address           *string `json:"address"`
	PriceAkad         *int64  `json:"priceAkad"`
	PriceAkadResepsi  *int64  `json:"priceAkadResepsi"`
	Notes             string  `json:"notes"`
	HasAttachment     bool    `json:"hasAttachment"`
	AttachmentIsImage bool    `json:"attachmentIsImage"`
	IsActive          bool    `json:"isActive"`
	CreatedAt         string  `json:"createdAt"`
}

func toVendorResponse(v domain.Vendor) vendorResponse {
	attachmentIsImage := v.AttachmentMimeType != nil && strings.HasPrefix(*v.AttachmentMimeType, "image/")
	return vendorResponse{
		ID: v.ID, CategoryID: v.CategoryID, Name: v.Name, PICName: v.PICName, Phone: v.Phone,
		Email: v.Email, SocialMedia: v.SocialMedia, City: v.City, Address: v.Address,
		PriceAkad: v.PriceAkad, PriceAkadResepsi: v.PriceAkadResepsi, Notes: v.Notes,
		HasAttachment: v.AttachmentPath != nil, AttachmentIsImage: attachmentIsImage, IsActive: v.IsActive,
		CreatedAt: v.CreatedAt.Format("2006-01-02"),
	}
}

func (h *VendorHandler) Collection(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := requireStaffTenant(w, r)
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
		if !requireManagerRole(w, r) {
			return
		}
		h.create(w, r, tenantID)
	default:
		response.Error(w, http.StatusMethodNotAllowed, "Metode HTTP tidak diizinkan untuk endpoint ini", nil)
	}
}

type vendorInputBody struct {
	Name             string `json:"name"`
	CategoryID       int64  `json:"categoryId"`
	PICName          string `json:"picName"`
	Phone            string `json:"phone"`
	Email            string `json:"email"`
	SocialMedia      string `json:"socialMedia"`
	City             string `json:"city"`
	Address          string `json:"address"`
	PriceAkad        *int64 `json:"priceAkad"`
	PriceAkadResepsi *int64 `json:"priceAkadResepsi"`
	Notes            string `json:"notes"`
}

func toVendorInput(body vendorInputBody) application.VendorInput {
	return application.VendorInput{
		Name: body.Name, CategoryID: body.CategoryID, PICName: body.PICName, Phone: body.Phone,
		Email: body.Email, SocialMedia: body.SocialMedia, City: body.City, Address: body.Address,
		PriceAkad: body.PriceAkad, PriceAkadResepsi: body.PriceAkadResepsi, Notes: body.Notes,
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
	tenantID, ok := requireStaffTenant(w, r)
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
	rest := segments[1:]

	switch {
	case len(rest) == 0 && r.Method == http.MethodPatch:
		if !requireManagerRole(w, r) {
			return
		}
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
	case len(rest) == 1 && rest[0] == "toggle-active" && r.Method == http.MethodPost:
		if !requireManagerRole(w, r) {
			return
		}
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
	case len(rest) == 1 && rest[0] == "project-history" && r.Method == http.MethodGet:
		if !requireManagerRole(w, r) {
			return
		}
		h.projectHistory(w, r, tenantID, id)
	case len(rest) == 1 && rest[0] == "attachment" && r.Method == http.MethodPut:
		if !requireManagerRole(w, r) {
			return
		}
		h.uploadAttachment(w, r, tenantID, id)
	case len(rest) == 1 && rest[0] == "attachment" && r.Method == http.MethodGet:
		h.downloadAttachment(w, r, tenantID, id)
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
// `projects.Contracts` (see vendors.module.go's doc comment). Owner/Admin
// only (requireManagerRole in Item's routing above) — unlike the plain
// vendor list, a Wedding Planner has no legitimate need to see everywhere
// else in the tenant a vendor has ever been engaged, only their own
// project's own engagement (which lives in the `projects` module itself).
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

type vendorAttachmentUploadBody struct {
	FileName   string `json:"fileName"`
	MimeType   string `json:"mimeType"`
	Base64Data string `json:"base64Data"`
}

func (h *VendorHandler) uploadAttachment(w http.ResponseWriter, r *http.Request, tenantID, vendorID int64) {
	var body vendorAttachmentUploadBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
		return
	}
	vendor, err := h.vendors.UploadAttachment(r.Context(), tenantID, vendorID, application.UploadVendorAttachmentInput{
		FileName: body.FileName, MimeType: body.MimeType, Base64Data: body.Base64Data,
	})
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "Lampiran vendor berhasil diunggah", toVendorResponse(*vendor))
}

// downloadAttachment is staff-only end to end (unlike Venue's, which carves
// out a client-visible exception for image attachments) — nothing today
// requires Client Portal to see a vendor's attachment, so this stays gated
// entirely by requireStaffTenant in Item's routing above.
func (h *VendorHandler) downloadAttachment(w http.ResponseWriter, r *http.Request, tenantID, vendorID int64) {
	_, reader, err := h.vendors.DownloadAttachment(r.Context(), tenantID, vendorID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	defer reader.Close()
	w.Header().Set("Content-Disposition", `inline; filename="attachment"`)
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, reader)
}

// --- Public-safe summary (reachable by `client` principals too) ---

type vendorSummaryResponse struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// Summary backs Client Portal's Vendor Progress tab, which only ever needs
// a vendor's name (confirmed by code inspection: `VendorProgressTabPage.tsx`
// resolves `vendors.find(v => v.id === pv.vendorId)?.name` and nothing else
// from the vendor record — category name comes from a separate store).
// Deliberately includes inactive vendors too, unlike the rest of this
// module's `all=true` listing convention: a vendor deactivated after being
// engaged on a project must still resolve to its real name here, not
// silently disappear from a client's already-committed engagement history.
func (h *VendorHandler) Summary(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := requireTenant(w, r)
	if !ok {
		return
	}
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, "Metode HTTP tidak diizinkan untuk endpoint ini", nil)
		return
	}
	vendors, err := h.vendors.List(r.Context(), tenantID, nil)
	if err != nil {
		writeAppError(w, err)
		return
	}
	result := make([]vendorSummaryResponse, 0, len(vendors))
	for _, v := range vendors {
		result = append(result, vendorSummaryResponse{ID: v.ID, Name: v.Name})
	}
	response.OK(w, "ok", result)
}

// --- Bulk import (Excel) ---

// vendorTemplateHeaders is both the template's header row (Template) and the
// authoritative column order the import parser (Import) reads by index.
var vendorTemplateHeaders = []string{
	"Nama Vendor", "Kategori", "Nama PIC", "No Tlp Vendor", "Email",
	"Sosial Media", "Kota", "Alamat", "Harga Akad", "Harga Akad+Resepsi", "Catatan",
}

func (h *VendorHandler) Template(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireStaffTenant(w, r); !ok {
		return
	}
	if !requireManagerRole(w, r) {
		return
	}
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, "Metode HTTP tidak diizinkan untuk endpoint ini", nil)
		return
	}

	f := excelize.NewFile()
	defer f.Close()
	sheet := f.GetSheetName(0)
	for i, header := range vendorTemplateHeaders {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, header)
	}

	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", `attachment; filename="template-vendor.xlsx"`)
	if _, err := f.WriteTo(w); err != nil {
		response.Error(w, http.StatusInternalServerError, "Gagal membuat berkas template", nil)
	}
}

type vendorImportRequestBody struct {
	Base64Data string `json:"base64Data"`
}

type vendorImportRowErrorBody struct {
	Row     int    `json:"row"`
	Message string `json:"message"`
}

type vendorImportResultBody struct {
	InsertedCount int                        `json:"insertedCount"`
	UpdatedCount  int                        `json:"updatedCount"`
	Errors        []vendorImportRowErrorBody `json:"errors"`
}

func toVendorImportResultBody(result application.VendorImportResult) vendorImportResultBody {
	errs := make([]vendorImportRowErrorBody, 0, len(result.Errors))
	for _, e := range result.Errors {
		errs = append(errs, vendorImportRowErrorBody{Row: e.Row, Message: e.Message})
	}
	return vendorImportResultBody{InsertedCount: result.InsertedCount, UpdatedCount: result.UpdatedCount, Errors: errs}
}

// parseVendorImportRow reads one spreadsheet row's cells in
// vendorTemplateHeaders' exact column order. Optional numeric cells that
// don't parse as an integer are treated as blank rather than a hard row
// failure — the service layer's own required-field check is the real gate.
func parseVendorImportRow(cells []string) application.VendorImportRow {
	get := func(i int) string {
		if i < len(cells) {
			return strings.TrimSpace(cells[i])
		}
		return ""
	}
	parseIntPtr := func(s string) *int64 {
		if s == "" {
			return nil
		}
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return nil
		}
		return &n
	}
	return application.VendorImportRow{
		Name: get(0), CategoryName: get(1), PICName: get(2), Phone: get(3), Email: get(4),
		SocialMedia: get(5), City: get(6), Address: get(7), PriceAkad: parseIntPtr(get(8)),
		PriceAkadResepsi: parseIntPtr(get(9)), Notes: get(10),
	}
}

func (h *VendorHandler) Import(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := requireStaffTenant(w, r)
	if !ok {
		return
	}
	if !requireManagerRole(w, r) {
		return
	}
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, "Metode HTTP tidak diizinkan untuk endpoint ini", nil)
		return
	}

	var body vendorImportRequestBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
		return
	}
	decoded, err := base64.StdEncoding.DecodeString(body.Base64Data)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "Data berkas tidak valid", nil)
		return
	}

	f, err := excelize.OpenReader(strings.NewReader(string(decoded)))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "Berkas Excel tidak valid", nil)
		return
	}
	defer f.Close()

	sheet := f.GetSheetName(0)
	rowIter, err := f.Rows(sheet)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "Berkas Excel tidak valid", nil)
		return
	}
	defer rowIter.Close()

	var rows []application.VendorImportRow
	rowIter.Next() // header row -- skip, position and meaning are fixed (vendorTemplateHeaders)
	for rowIter.Next() {
		cells, err := rowIter.Columns()
		if err != nil {
			response.Error(w, http.StatusBadRequest, "Gagal membaca salah satu baris pada berkas Excel", nil)
			return
		}
		if len(cells) == 0 || strings.TrimSpace(strings.Join(cells, "")) == "" {
			continue // fully blank row (e.g. trailing rows) -- not a data row
		}
		if len(rows) >= 1000 {
			// The service layer re-checks and rejects at exactly this cap --
			// stop reading further rows here too so a pathological file with
			// far more than 1000 rows doesn't keep streaming needlessly.
			rows = append(rows, parseVendorImportRow(cells))
			break
		}
		rows = append(rows, parseVendorImportRow(cells))
	}

	result, err := h.vendors.ImportVendors(r.Context(), tenantID, rows)
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "Impor vendor selesai diproses", toVendorImportResultBody(result))
}
