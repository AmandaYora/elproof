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
	ProjectVendorID      int64  `json:"projectVendorId"`
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
		ProjectVendorID: body.ProjectVendorID, Title: body.Title, Description: body.Description,
		Impact: domain.IssueImpact(body.Impact), FoundDate: foundDate, ResolutionPlan: body.ResolutionPlan,
		PICStaffID: body.PICStaffID, TargetResolutionDate: parseOptionalDate(body.TargetResolutionDate),
	})
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.Created(w, "Kendala berhasil dicatat", toIssueResponse(*issue))
}

type issueStatusBody struct {
	Status string `json:"status"`
}

func (h *Handler) updateIssueStatus(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64, issueIDRaw string) {
	issueID, err := parseInt64(issueIDRaw)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "ID tidak valid", nil)
		return
	}
	var body issueStatusBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
		return
	}
	issue, err := h.issues.UpdateStatus(r.Context(), projectID, issueID, claims.staffID, domain.IssueStatus(body.Status), time.Now())
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "Status kendala diperbarui", toIssueResponse(*issue))
}
