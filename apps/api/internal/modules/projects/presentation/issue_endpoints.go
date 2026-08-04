package presentation

import (
	"encoding/json"
	"net/http"
	"time"

	"elproof/internal/modules/projects/application"
	"elproof/internal/modules/projects/domain"
	"elproof/internal/shared/response"
)

func (h *Handler) listIssues(w http.ResponseWriter, r *http.Request, projectID int64) {
	list, err := h.issues.List(r.Context(), projectID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	result := make([]issueResponse, 0, len(list))
	for _, i := range list {
		result = append(result, toIssueResponse(i))
	}
	response.OK(w, "ok", result)
}

type issueInputBody struct {
	ProjectVendorID int64 `json:"projectVendorId"`
	// VendorMilestoneID is nil/omitted when the kendala is general to the
	// vendor engagement, not tied to one specific deliverable.
	VendorMilestoneID    *int64 `json:"vendorMilestoneId"`
	Title                string `json:"title"`
	Description          string `json:"description"`
	Impact               string `json:"impact"`
	FoundDate            string `json:"foundDate"`
	ResolutionPlan       string `json:"resolutionPlan"`
	PICStaffID           int64  `json:"picStaffId"`
	TargetResolutionDate string `json:"targetResolutionDate"`
}

func (h *Handler) createIssue(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64) {
	var body issueInputBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
		return
	}
	foundDate, err := parseDate(body.FoundDate)
	if err != nil {
		response.Error(w, http.StatusUnprocessableEntity, "Format tanggal tidak valid", map[string][]string{"foundDate": {"Gunakan format YYYY-MM-DD"}})
		return
	}
	issue, err := h.issues.Create(r.Context(), projectID, claims.staffID, application.IssueInput{
		ProjectVendorID: body.ProjectVendorID, VendorMilestoneID: body.VendorMilestoneID, Title: body.Title, Description: body.Description,
		Impact: domain.IssueImpact(body.Impact), FoundDate: foundDate, ResolutionPlan: body.ResolutionPlan,
		PICStaffID: body.PICStaffID, TargetResolutionDate: parseOptionalDate(body.TargetResolutionDate),
	})
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.Created(w, "Kendala berhasil dicatat", toIssueResponse(*issue))
}

// issueUpdateBody is issueInputBody's shape plus Status -- a full overwrite,
// not a partial patch (PLAN.md "Retire the standalone Kendala tab"). The
// frontend's quick status-change dropdown sends this same shape with every
// other field spread from the existing issue and only Status changed.
type issueUpdateBody struct {
	ProjectVendorID      int64  `json:"projectVendorId"`
	VendorMilestoneID    *int64 `json:"vendorMilestoneId"`
	Title                string `json:"title"`
	Description          string `json:"description"`
	Impact               string `json:"impact"`
	FoundDate            string `json:"foundDate"`
	Status               string `json:"status"`
	ResolutionPlan       string `json:"resolutionPlan"`
	PICStaffID           int64  `json:"picStaffId"`
	TargetResolutionDate string `json:"targetResolutionDate"`
}

func (h *Handler) updateIssue(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64, issueIDRaw string) {
	issueID, err := parseInt64(issueIDRaw)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "ID tidak valid", nil)
		return
	}
	var body issueUpdateBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
		return
	}
	foundDate, err := parseDate(body.FoundDate)
	if err != nil {
		response.Error(w, http.StatusUnprocessableEntity, "Format tanggal tidak valid", map[string][]string{"foundDate": {"Gunakan format YYYY-MM-DD"}})
		return
	}
	issue, err := h.issues.Update(r.Context(), projectID, issueID, claims.staffID, application.IssueUpdateInput{
		ProjectVendorID: body.ProjectVendorID, VendorMilestoneID: body.VendorMilestoneID, Title: body.Title, Description: body.Description,
		Impact: domain.IssueImpact(body.Impact), FoundDate: foundDate, Status: domain.IssueStatus(body.Status), ResolutionPlan: body.ResolutionPlan,
		PICStaffID: body.PICStaffID, TargetResolutionDate: parseOptionalDate(body.TargetResolutionDate),
	}, time.Now())
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "Kendala berhasil diperbarui", toIssueResponse(*issue))
}
