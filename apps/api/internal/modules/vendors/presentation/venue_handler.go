package presentation

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"

	"elproof/internal/modules/vendors/application"
	"elproof/internal/modules/vendors/domain"
	"elproof/internal/shared/httpx"
	"elproof/internal/shared/middleware"
	"elproof/internal/shared/pagination"
	"elproof/internal/shared/response"
)

// requireStaffTenant is requireTenant's stricter sibling: `client` principals
// are otherwise tenant-scoped just like staff (requireTenant lets both
// through), but venue records carry commercial data (rental price, charge,
// PIC contact) never meant for a client to read directly — see ADR-0016's
// Client Portal field-visibility decision. The attachment GET route
// deliberately stays on requireTenant below (see downloadAttachment) since
// it's the one piece of content ADR-0016 says clients should conditionally see.
func requireStaffTenant(w http.ResponseWriter, r *http.Request) (int64, bool) {
	claims, ok := middleware.FromContext(r.Context())
	if !ok || claims.PrincipalType != "staff" {
		response.Error(w, http.StatusForbidden, "Hanya staff WO yang dapat mengakses endpoint ini", nil)
		return 0, false
	}
	tenantID, ok := claims.TenantIDInt()
	if !ok {
		response.Error(w, http.StatusForbidden, "Akun ini tidak terikat ke tenant manapun", nil)
		return 0, false
	}
	return tenantID, true
}

// requireManagerRole is the extra gate every Vendor/Venue **write** action
// needs on top of requireStaffTenant — confirmed role rule: a Wedding
// Planner keeps read access (list/get/attachment-download) so the vendor-
// and venue-pickers inside their own project keep working, but every
// mutation (create/update/toggle-active/upload/template/import) is
// Owner/Admin only. Assumes requireStaffTenant already ran and succeeded
// (so claims is guaranteed present) — called only from write branches below.
func requireManagerRole(w http.ResponseWriter, r *http.Request) bool {
	claims, _ := middleware.FromContext(r.Context())
	if !claims.HasRole("Owner", "Admin") {
		response.Error(w, http.StatusForbidden, "Hanya Owner atau Admin yang dapat melakukan aksi ini", nil)
		return false
	}
	return true
}

type VenueHandler struct {
	venues *application.VenueService
}

func NewVenueHandler(venues *application.VenueService) *VenueHandler {
	return &VenueHandler{venues: venues}
}

type venueResponse struct {
	ID                int64   `json:"id"`
	Name              string  `json:"name"`
	PICName           string  `json:"picName"`
	PhonePIC          string  `json:"phonePic"`
	PhoneVenue        *string `json:"phoneVenue"`
	Email             *string `json:"email"`
	Address           *string `json:"address"`
	City              *string `json:"city"`
	RentalPrice       *int64  `json:"rentalPrice"`
	Charge            *int64  `json:"charge"`
	Capacity          *int    `json:"capacity"`
	Facilities        *string `json:"facilities"`
	SocialMedia       *string `json:"socialMedia"`
	Notes             string  `json:"notes"`
	HasAttachment     bool    `json:"hasAttachment"`
	AttachmentIsImage bool    `json:"attachmentIsImage"`
	IsActive          bool    `json:"isActive"`
	CreatedAt         string  `json:"createdAt"`
}

func toVenueResponse(v domain.Venue) venueResponse {
	attachmentIsImage := v.AttachmentMimeType != nil && strings.HasPrefix(*v.AttachmentMimeType, "image/")
	return venueResponse{
		ID: v.ID, Name: v.Name, PICName: v.PICName, PhonePIC: v.PhonePIC, PhoneVenue: v.PhoneVenue,
		Email: v.Email, Address: v.Address, City: v.City, RentalPrice: v.RentalPrice, Charge: v.Charge,
		Capacity: v.Capacity, Facilities: v.Facilities, SocialMedia: v.SocialMedia, Notes: v.Notes,
		HasAttachment: v.AttachmentPath != nil, AttachmentIsImage: attachmentIsImage, IsActive: v.IsActive,
		CreatedAt: v.CreatedAt.Format("2006-01-02"),
	}
}

func (h *VenueHandler) Collection(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := requireStaffTenant(w, r)
	if !ok {
		return
	}
	switch r.Method {
	case http.MethodGet:
		if r.URL.Query().Get("all") == "true" {
			venues, err := h.venues.List(r.Context(), tenantID)
			if err != nil {
				writeAppError(w, err)
				return
			}
			result := make([]venueResponse, 0, len(venues))
			for _, v := range venues {
				result = append(result, toVenueResponse(v))
			}
			response.OK(w, "ok", result)
			return
		}
		params := pagination.FromRequest(r)
		search := r.URL.Query().Get("search")
		venues, total, err := h.venues.ListPaginated(r.Context(), tenantID, params, search)
		if err != nil {
			writeAppError(w, err)
			return
		}
		result := make([]venueResponse, 0, len(venues))
		for _, v := range venues {
			result = append(result, toVenueResponse(v))
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

type venueInputBody struct {
	Name        string `json:"name"`
	PICName     string `json:"picName"`
	PhonePIC    string `json:"phonePic"`
	PhoneVenue  string `json:"phoneVenue"`
	Email       string `json:"email"`
	Address     string `json:"address"`
	City        string `json:"city"`
	RentalPrice *int64 `json:"rentalPrice"`
	Charge      *int64 `json:"charge"`
	Capacity    *int   `json:"capacity"`
	Facilities  string `json:"facilities"`
	SocialMedia string `json:"socialMedia"`
	Notes       string `json:"notes"`
}

func toVenueInput(body venueInputBody) application.VenueInput {
	return application.VenueInput{
		Name: body.Name, PICName: body.PICName, PhonePIC: body.PhonePIC, PhoneVenue: body.PhoneVenue,
		Email: body.Email, Address: body.Address, City: body.City, RentalPrice: body.RentalPrice,
		Charge: body.Charge, Capacity: body.Capacity, Facilities: body.Facilities, SocialMedia: body.SocialMedia,
		Notes: body.Notes,
	}
}

func (h *VenueHandler) create(w http.ResponseWriter, r *http.Request, tenantID int64) {
	var body venueInputBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
		return
	}
	venue, err := h.venues.Create(r.Context(), tenantID, toVenueInput(body))
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.Created(w, "Venue berhasil ditambahkan", toVenueResponse(*venue))
}

func (h *VenueHandler) Item(w http.ResponseWriter, r *http.Request) {
	segments := httpx.Segments(r.URL.Path, "/api/v1/venues/")
	if len(segments) == 0 {
		response.Error(w, http.StatusNotFound, "Venue tidak ditemukan", nil)
		return
	}
	id, err := strconv.ParseInt(segments[0], 10, 64)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "ID tidak valid", nil)
		return
	}
	rest := segments[1:]

	// GET .../attachment is the one route reachable by a `client` principal
	// (requireTenant, not requireStaffTenant) -- downloadAttachment itself
	// enforces the mime-type gate below, so a client can't fetch a
	// document-type attachment even by calling this endpoint directly.
	var tenantID int64
	var ok bool
	if len(rest) == 1 && rest[0] == "attachment" && r.Method == http.MethodGet {
		tenantID, ok = requireTenant(w, r)
	} else {
		tenantID, ok = requireStaffTenant(w, r)
	}
	if !ok {
		return
	}

	switch {
	case len(rest) == 0 && r.Method == http.MethodGet:
		h.get(w, r, tenantID, id)
	case len(rest) == 0 && r.Method == http.MethodPatch:
		if !requireManagerRole(w, r) {
			return
		}
		h.update(w, r, tenantID, id)
	case len(rest) == 1 && rest[0] == "toggle-active" && r.Method == http.MethodPost:
		if !requireManagerRole(w, r) {
			return
		}
		h.toggleActive(w, r, tenantID, id)
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

func (h *VenueHandler) get(w http.ResponseWriter, r *http.Request, tenantID, id int64) {
	venue, err := h.venues.Get(r.Context(), tenantID, id)
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "ok", toVenueResponse(*venue))
}

func (h *VenueHandler) update(w http.ResponseWriter, r *http.Request, tenantID, id int64) {
	var body venueInputBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
		return
	}
	venue, err := h.venues.Update(r.Context(), tenantID, id, toVenueInput(body))
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "Venue berhasil diperbarui", toVenueResponse(*venue))
}

func (h *VenueHandler) toggleActive(w http.ResponseWriter, r *http.Request, tenantID, id int64) {
	current, err := h.venues.Get(r.Context(), tenantID, id)
	if err != nil {
		writeAppError(w, err)
		return
	}
	venue, err := h.venues.SetActive(r.Context(), tenantID, id, !current.IsActive)
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "Status venue diperbarui", toVenueResponse(*venue))
}

type venueAttachmentUploadBody struct {
	FileName   string `json:"fileName"`
	MimeType   string `json:"mimeType"`
	Base64Data string `json:"base64Data"`
}

func (h *VenueHandler) uploadAttachment(w http.ResponseWriter, r *http.Request, tenantID, venueID int64) {
	var body venueAttachmentUploadBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
		return
	}
	venue, err := h.venues.UploadAttachment(r.Context(), tenantID, venueID, application.UploadVenueAttachmentInput{
		FileName: body.FileName, MimeType: body.MimeType, Base64Data: body.Base64Data,
	})
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "Lampiran venue berhasil diunggah", toVenueResponse(*venue))
}

func (h *VenueHandler) downloadAttachment(w http.ResponseWriter, r *http.Request, tenantID, venueID int64) {
	venue, reader, err := h.venues.DownloadAttachment(r.Context(), tenantID, venueID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	defer reader.Close()

	// requireTenant let both `staff` and `client` principals reach this
	// handler -- the mime-type gate here is the real boundary for `client`
	// (ADR-0016): a document-type attachment is never served to a client,
	// even though they know a valid venue ID, only an image is.
	if claims, ok := middleware.FromContext(r.Context()); ok && claims.PrincipalType == "client" {
		isImage := venue.AttachmentMimeType != nil && strings.HasPrefix(*venue.AttachmentMimeType, "image/")
		if !isImage {
			response.Error(w, http.StatusForbidden, "Lampiran ini tidak dapat diakses", nil)
			return
		}
	}

	w.Header().Set("Content-Disposition", `inline; filename="attachment"`)
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, reader)
}

// --- Bulk import (Excel) ---

// venueTemplateHeaders is both the template's header row (Template) and the
// authoritative column order the import parser (Import) reads by index --
// keeping them as one slice means the two can never drift apart.
var venueTemplateHeaders = []string{
	"Nama Venue", "Nama PIC", "No Tlp PIC", "No Tlp Venue", "Email",
	"Alamat", "Kota", "Harga Sewa", "Charge", "Kapasitas", "Fasilitas", "Sosial Media", "Catatan",
}

func (h *VenueHandler) Template(w http.ResponseWriter, r *http.Request) {
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
	for i, header := range venueTemplateHeaders {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, header)
	}

	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", `attachment; filename="template-venue.xlsx"`)
	if _, err := f.WriteTo(w); err != nil {
		response.Error(w, http.StatusInternalServerError, "Gagal membuat berkas template", nil)
	}
}

type venueImportRequestBody struct {
	Base64Data string `json:"base64Data"`
}

type venueImportRowErrorBody struct {
	Row     int    `json:"row"`
	Message string `json:"message"`
}

type venueImportResultBody struct {
	InsertedCount int                       `json:"insertedCount"`
	UpdatedCount  int                       `json:"updatedCount"`
	Errors        []venueImportRowErrorBody `json:"errors"`
}

func toVenueImportResultBody(result application.VenueImportResult) venueImportResultBody {
	errs := make([]venueImportRowErrorBody, 0, len(result.Errors))
	for _, e := range result.Errors {
		errs = append(errs, venueImportRowErrorBody{Row: e.Row, Message: e.Message})
	}
	return venueImportResultBody{InsertedCount: result.InsertedCount, UpdatedCount: result.UpdatedCount, Errors: errs}
}

// parseVenueImportRow reads one spreadsheet row's cells in venueTemplateHeaders'
// exact column order. Optional numeric cells that don't parse as an integer
// are treated as blank rather than a hard row failure -- the service layer's
// own required-field check (name/picName/phonePic/city) is the real gate.
func parseVenueImportRow(cells []string) application.VenueImportRow {
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
	parseIntPtrSmall := func(s string) *int {
		if s == "" {
			return nil
		}
		n, err := strconv.Atoi(s)
		if err != nil {
			return nil
		}
		return &n
	}
	return application.VenueImportRow{
		Name: get(0), PICName: get(1), PhonePIC: get(2), PhoneVenue: get(3), Email: get(4),
		Address: get(5), City: get(6), RentalPrice: parseIntPtr(get(7)), Charge: parseIntPtr(get(8)),
		Capacity: parseIntPtrSmall(get(9)), Facilities: get(10), SocialMedia: get(11), Notes: get(12),
	}
}

func (h *VenueHandler) Import(w http.ResponseWriter, r *http.Request) {
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

	var body venueImportRequestBody
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

	var rows []application.VenueImportRow
	rowIter.Next() // header row -- skip, position and meaning are fixed (venueTemplateHeaders)
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
			rows = append(rows, parseVenueImportRow(cells))
			break
		}
		rows = append(rows, parseVenueImportRow(cells))
	}

	result, err := h.venues.ImportVenues(r.Context(), tenantID, rows)
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "Impor venue selesai diproses", toVenueImportResultBody(result))
}
