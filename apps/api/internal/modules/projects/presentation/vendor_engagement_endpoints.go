package presentation

import (
	"encoding/json"
	"net/http"
	"time"

	"elproof/internal/modules/projects/application"
	"elproof/internal/modules/projects/domain"
	"elproof/internal/shared/response"
)

func (h *Handler) listVendorEngagements(w http.ResponseWriter, r *http.Request, projectID int64) {
	list, err := h.vendors.List(r.Context(), projectID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	result := make([]projectVendorResponse, 0, len(list))
	for _, pv := range list {
		result = append(result, toProjectVendorResponse(pv))
	}
	response.OK(w, "ok", result)
}

type vendorEngagementInputBody struct {
	VendorID         int64  `json:"vendorId"`
	CategoryID       int64  `json:"categoryId"`
	Scope            string `json:"scope"`
	ContractValue    int64  `json:"contractValue"`
	EngagementStatus string `json:"engagementStatus"`
	BookingDate      string `json:"bookingDate"`
	EventDate        string `json:"eventDate"`
	DPAmount         int64  `json:"dpAmount"`
	PaidAmount       int64  `json:"paidAmount"`
	DueDate          string `json:"dueDate"`
	PICStaffID       int64  `json:"picStaffId"`
	Notes            string `json:"notes"`
}

func toVendorEngagementInput(body vendorEngagementInputBody) (application.VendorEngagementInput, error) {
	eventDate, err := parseDate(body.EventDate)
	if err != nil {
		return application.VendorEngagementInput{}, err
	}
	bookingDate := parseOptionalDate(body.BookingDate)
	dueDate := parseOptionalDate(body.DueDate)
	return application.VendorEngagementInput{
		VendorID: body.VendorID, CategoryID: body.CategoryID, Scope: body.Scope, ContractValue: body.ContractValue,
		EngagementStatus: domain.EngagementStatus(body.EngagementStatus), BookingDate: bookingDate, EventDate: eventDate,
		DPAmount: body.DPAmount, PaidAmount: body.PaidAmount, DueDate: dueDate, PICStaffID: body.PICStaffID, Notes: body.Notes,
	}, nil
}

func parseOptionalDate(s string) *time.Time {
	if s == "" {
		return nil
	}
	t, err := parseDate(s)
	if err != nil {
		return nil
	}
	return &t
}

func (h *Handler) createVendorEngagement(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64) {
	var body vendorEngagementInputBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
		return
	}
	input, err := toVendorEngagementInput(body)
	if err != nil {
		response.Error(w, http.StatusUnprocessableEntity, "Format tanggal tidak valid", map[string][]string{"eventDate": {"Gunakan format YYYY-MM-DD"}})
		return
	}
	pv, err := h.vendors.Create(r.Context(), projectID, claims.staffID, input)
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.Created(w, "Vendor berhasil ditambahkan ke project", toProjectVendorResponse(*pv))
}

func (h *Handler) updateVendorEngagement(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64, pvIDRaw string) {
	pvID, err := parseInt64(pvIDRaw)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "ID tidak valid", nil)
		return
	}
	var body vendorEngagementInputBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
		return
	}
	input, err := toVendorEngagementInput(body)
	if err != nil {
		response.Error(w, http.StatusUnprocessableEntity, "Format tanggal tidak valid", map[string][]string{"eventDate": {"Gunakan format YYYY-MM-DD"}})
		return
	}
	pv, err := h.vendors.Update(r.Context(), projectID, pvID, claims.staffID, input)
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "Kerja sama vendor berhasil diperbarui", toProjectVendorResponse(*pv))
}

func (h *Handler) cancelVendorEngagement(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64, pvIDRaw string) {
	pvID, err := parseInt64(pvIDRaw)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "ID tidak valid", nil)
		return
	}
	pv, err := h.vendors.Cancel(r.Context(), projectID, pvID, claims.staffID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "Kerja sama vendor dibatalkan", toProjectVendorResponse(*pv))
}

// --- Vendor milestones ---

func (h *Handler) listVendorMilestones(w http.ResponseWriter, r *http.Request, projectID int64, pvIDRaw string) {
	pvID, err := parseInt64(pvIDRaw)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "ID tidak valid", nil)
		return
	}
	list, err := h.vendors.ListMilestones(r.Context(), projectID, pvID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	result := make([]vendorMilestoneResponse, 0, len(list))
	for _, m := range list {
		result = append(result, toVendorMilestoneResponse(m))
	}
	response.OK(w, "ok", result)
}

type vendorMilestoneInputBody struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	TargetDate  string `json:"targetDate"`
	PICStaffID  int64  `json:"picStaffId"`
}

func (h *Handler) createVendorMilestone(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64, pvIDRaw string) {
	pvID, err := parseInt64(pvIDRaw)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "ID tidak valid", nil)
		return
	}
	var body vendorMilestoneInputBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
		return
	}
	targetDate, err := parseDate(body.TargetDate)
	if err != nil {
		response.Error(w, http.StatusUnprocessableEntity, "Format tanggal tidak valid", map[string][]string{"targetDate": {"Gunakan format YYYY-MM-DD"}})
		return
	}
	m, err := h.vendors.CreateMilestone(r.Context(), projectID, pvID, claims.staffID, application.VendorMilestoneInput{
		Name: body.Name, Description: body.Description, TargetDate: targetDate, PICStaffID: body.PICStaffID,
	})
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.Created(w, "Milestone vendor berhasil ditambahkan", toVendorMilestoneResponse(*m))
}

type vendorMilestoneUpdateBody struct {
	Status        string `json:"status"`
	TargetDate    string `json:"targetDate"`
	CompletedDate string `json:"completedDate"`
	PICStaffID    int64  `json:"picStaffId"`
	Description   string `json:"description"`
	Notes         string `json:"notes"`
}

func (h *Handler) updateVendorMilestone(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64, pvIDRaw, milestoneIDRaw string) {
	pvID, err := parseInt64(pvIDRaw)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "ID tidak valid", nil)
		return
	}
	milestoneID, err := parseInt64(milestoneIDRaw)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "ID milestone tidak valid", nil)
		return
	}
	var body vendorMilestoneUpdateBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
		return
	}
	targetDate, err := parseDate(body.TargetDate)
	if err != nil {
		response.Error(w, http.StatusUnprocessableEntity, "Format tanggal tidak valid", map[string][]string{"targetDate": {"Gunakan format YYYY-MM-DD"}})
		return
	}
	m, err := h.vendors.UpdateMilestone(r.Context(), projectID, pvID, milestoneID, claims.staffID, application.VendorMilestoneUpdateInput{
		Status: domain.MilestoneStatus(body.Status), TargetDate: targetDate, CompletedDate: parseOptionalDate(body.CompletedDate),
		PICStaffID: body.PICStaffID, Description: body.Description, Notes: body.Notes,
	})
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "Milestone vendor berhasil diperbarui", toVendorMilestoneResponse(*m))
}
