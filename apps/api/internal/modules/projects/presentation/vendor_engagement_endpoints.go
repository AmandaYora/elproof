package presentation

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"elproof/internal/modules/projects/application"
	"elproof/internal/modules/projects/domain"
	"elproof/internal/shared/apperror"
	"elproof/internal/shared/response"
)

// vendorPaidAmounts sums each project_vendor's own vendor_payments rows
// (Refund netted as a subtraction) into a map keyed by ProjectVendorID —
// the "Total Sudah Dibayar" figure is derived entirely from this ledger,
// never a stored/manually-typed column. See PLAN.md "Financial Calculation
// Correctness". A missing key correctly defaults to Go's int64 zero value
// (no payments recorded yet).
func (h *Handler) vendorPaidAmounts(ctx context.Context, projectID int64) (map[int64]int64, error) {
	payments, err := h.payments.List(ctx, projectID)
	if err != nil {
		return nil, err
	}
	paid := make(map[int64]int64, len(payments))
	for _, p := range payments {
		if p.Type == domain.PaymentRefund {
			paid[p.ProjectVendorID] -= p.Amount
		} else {
			paid[p.ProjectVendorID] += p.Amount
		}
	}
	return paid, nil
}

// listVendorEngagements embeds each engagement's own milestones in the same
// response (one batched query via ListMilestonesByProjectVendors) so the
// frontend's fetchVendorSection no longer needs its own per-engagement
// Promise.all loop — PLAN.md "Performance remediation" Phase E.
func (h *Handler) listVendorEngagements(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64) {
	isClient := claims.isClient
	list, err := h.vendors.List(r.Context(), projectID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	paid, err := h.vendorPaidAmounts(r.Context(), projectID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	pvIDs := make([]int64, len(list))
	for i, pv := range list {
		pvIDs[i] = pv.ID
	}
	milestonesByEngagement, err := h.vendors.ListMilestonesByProjectVendors(r.Context(), pvIDs)
	if err != nil {
		writeAppError(w, err)
		return
	}
	evidences, err := h.evidence.List(r.Context(), projectID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	ix := indexEvidence(evidences, isClient, claims.can)
	// Names come from this project's own engagements only, resolved inside the
	// caller's workspace in one lookup each: another vendor's name can never
	// reach a client, whatever ids exist in the directory.
	vendorIDs, categoryIDs := make([]int64, 0, len(list)), make([]int64, 0, len(list))
	for _, pv := range list {
		vendorIDs, categoryIDs = append(vendorIDs, pv.VendorID), append(categoryIDs, pv.CategoryID)
	}
	vendorNames := h.refs.VendorNames(r.Context(), claims.tenantID, vendorIDs)
	categoryNames := h.refs.CategoryNames(r.Context(), claims.tenantID, categoryIDs)
	result := make([]projectVendorResponse, 0, len(list))
	for _, pv := range list {
		resp := toProjectVendorResponse(pv, paid[pv.ID])
		resp.VendorName, resp.CategoryName = vendorNames[pv.VendorID], categoryNames[pv.CategoryID]
		milestones := milestonesByEngagement[pv.ID]
		resp.Milestones = make([]vendorMilestoneResponse, 0, len(milestones))
		for _, m := range milestones {
			mr := toVendorMilestoneResponse(m)
			mr.Evidence = ix.refs(domain.RelatedVendorMilestone, m.ID)
			resp.Milestones = append(resp.Milestones, mr)
		}
		if isClient {
			redactVendorForClient(&resp)
		}
		result = append(result, resp)
	}
	response.OK(w, "ok", result)
}

type vendorEngagementInputBody struct {
	VendorID         int64   `json:"vendorId"`
	CategoryID       int64   `json:"categoryId"`
	Scope            string  `json:"scope"`
	ContractValue    int64   `json:"contractValue"`
	PricingTier      string  `json:"pricingTier"`
	EngagementStatus string  `json:"engagementStatus"`
	BookingDate      string  `json:"bookingDate"`
	EventDate        string  `json:"eventDate"`
	EventStartTime   *string `json:"eventStartTime"`
	EventEndTime     *string `json:"eventEndTime"`
	DPAmount         int64   `json:"dpAmount"`
	DueDate          string  `json:"dueDate"`
	PICStaffID       int64   `json:"picStaffId"`
	Notes            string  `json:"notes"`
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
		PricingTier:      domain.VendorPricingTier(body.PricingTier),
		EngagementStatus: domain.EngagementStatus(body.EngagementStatus), BookingDate: bookingDate, EventDate: eventDate,
		EventStartTime: body.EventStartTime, EventEndTime: body.EventEndTime,
		DPAmount: body.DPAmount, DueDate: dueDate, PICStaffID: body.PICStaffID, Notes: body.Notes,
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

// createVendorEngagement attaches a vendor to the project. The category comes
// from the vendor and the date from the project (E7): both may be left out and
// are filled in here; if they are sent they must equal what the server knows
// (a 422 on that field otherwise). Every problem is reported together.
func (h *Handler) createVendorEngagement(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64) {
	var body vendorEngagementInputBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
		return
	}
	ctx := r.Context()
	fields := map[string][]string{}
	add := func(key, msg string) { fields[key] = append(fields[key], msg) }

	project, err := h.projects.Get(ctx, claims.tenantID, projectID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	// The event date: the project's. A date that is sent has to be the same day.
	if body.EventDate == "" {
		body.EventDate = project.EventDate.Format(dateLayout)
	} else if d, err := parseDate(body.EventDate); err != nil {
		add("eventDate", "Gunakan format YYYY-MM-DD")
	} else if d.Format(dateLayout) != project.EventDate.Format(dateLayout) {
		add("eventDate", "Tanggal vendor mengikuti tanggal acara project ("+project.EventDate.Format(dateLayout)+"). Kosongkan atau samakan.")
	}
	// The category: the vendor's own.
	if body.VendorID == 0 {
		add("vendorId", "Pilih vendor")
	} else {
		categoryID, ok, err := h.refs.VendorCategoryID(ctx, claims.tenantID, body.VendorID)
		if err != nil {
			writeAppError(w, err)
			return
		}
		switch {
		case !ok:
			add("vendorId", "Vendor tidak ditemukan")
		case body.CategoryID == 0:
			body.CategoryID = categoryID
		case body.CategoryID != categoryID:
			add("categoryId", "Kategori mengikuti vendor yang dipilih. Kosongkan atau samakan.")
		}
	}
	if err := h.refs.Category(ctx, claims.tenantID, "categoryId", body.CategoryID); err != nil && !collectValidation(fields, err) {
		writeAppError(w, err)
		return
	}
	if err := h.refs.Staff(ctx, claims.tenantID, "picStaffId", body.PICStaffID); err != nil && !collectValidation(fields, err) {
		writeAppError(w, err)
		return
	}
	if len(fields) > 0 {
		msg := "Data vendor belum lengkap atau tidak sesuai"
		for _, k := range []string{"vendorId", "categoryId", "eventDate", "picStaffId"} {
			if m := fields[k]; len(m) > 0 {
				msg = m[0]
				break
			}
		}
		response.Error(w, http.StatusUnprocessableEntity, msg, fields)
		return
	}
	input, err := toVendorEngagementInput(body)
	if err != nil {
		response.Error(w, http.StatusUnprocessableEntity, "Format tanggal tidak valid", map[string][]string{"eventDate": {"Gunakan format YYYY-MM-DD"}})
		return
	}
	pv, err := h.vendors.Create(ctx, projectID, claims.staffID, input)
	if err != nil {
		writeAppError(w, err)
		return
	}
	// A freshly created engagement has no vendor_payments rows yet — 0 paid,
	// no ledger lookup needed.
	response.Created(w, "Vendor berhasil ditambahkan ke project", toProjectVendorResponse(*pv, 0))
}

// collectValidation folds a validation error's fields into fields and reports
// true; for any other error it changes nothing and reports false.
func collectValidation(fields map[string][]string, err error) bool {
	ae, ok := apperror.As(err)
	if !ok || ae.Kind != apperror.KindValidation {
		return false
	}
	for k, v := range ae.Fields {
		fields[k] = append(fields[k], v...)
	}
	return true
}

// checkEngagementReferences: the vendor, its category and the PIC named in the
// body must all be this workspace's own.
func (h *Handler) checkEngagementReferences(ctx context.Context, tenantID int64, body vendorEngagementInputBody) error {
	if err := h.refs.Vendor(ctx, tenantID, "vendorId", body.VendorID); err != nil {
		return err
	}
	if err := h.refs.Category(ctx, tenantID, "categoryId", body.CategoryID); err != nil {
		return err
	}
	return h.refs.Staff(ctx, tenantID, "picStaffId", body.PICStaffID)
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
	if err := h.checkEngagementReferences(r.Context(), claims.tenantID, body); err != nil {
		writeAppError(w, err)
		return
	}
	pv, err := h.vendors.Update(r.Context(), projectID, pvID, claims.staffID, input)
	if err != nil {
		writeAppError(w, err)
		return
	}
	paid, err := h.vendorPaidAmounts(r.Context(), projectID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "Kerja sama vendor berhasil diperbarui", toProjectVendorResponse(*pv, paid[pv.ID]))
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
	paid, err := h.vendorPaidAmounts(r.Context(), projectID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "Kerja sama vendor dibatalkan", toProjectVendorResponse(*pv, paid[pv.ID]))
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
	if err := h.refs.Staff(r.Context(), claims.tenantID, "picStaffId", body.PICStaffID); err != nil {
		writeAppError(w, err)
		return
	}
	m, err := h.vendors.CreateMilestone(r.Context(), projectID, pvID, claims.staffID, application.VendorMilestoneInput{
		Name: body.Name, Description: body.Description, TargetDate: targetDate, PICStaffID: body.PICStaffID,
	})
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.Created(w, "Timeline vendor berhasil ditambahkan", toVendorMilestoneResponse(*m))
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
		response.Error(w, http.StatusBadRequest, "ID timeline tidak valid", nil)
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
	if err := h.refs.Staff(r.Context(), claims.tenantID, "picStaffId", body.PICStaffID); err != nil {
		writeAppError(w, err)
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
	response.OK(w, "Timeline vendor berhasil diperbarui", toVendorMilestoneResponse(*m))
}
