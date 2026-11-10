package presentation

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"elproof/internal/modules/projects/application"
	"elproof/internal/modules/projects/domain"
	"elproof/internal/shared/response"
)

func (h *Handler) listIssues(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64) {
	list, err := h.issues.List(r.Context(), projectID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	// A client sees only the kendala staff shared with them (isClientVisible).
	if claims.isClient {
		list = domain.ClientVisibleIssues(list)
	}
	// Names: this project's engagements (project-scoped) -> their vendors
	// (workspace-scoped, one lookup) and their milestones (one query). Proof
	// files: one project-wide evidence query, filtered for a client.
	engagements, err := h.vendors.List(r.Context(), projectID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	vendorOf := make(map[int64]int64, len(engagements))
	pvIDs := make([]int64, 0, len(engagements))
	vendorIDs := make([]int64, 0, len(engagements))
	for _, pv := range engagements {
		vendorOf[pv.ID] = pv.VendorID
		pvIDs = append(pvIDs, pv.ID)
		vendorIDs = append(vendorIDs, pv.VendorID)
	}
	milestonesByEngagement, err := h.vendors.ListMilestonesByProjectVendors(r.Context(), pvIDs)
	if err != nil {
		writeAppError(w, err)
		return
	}
	milestoneName := make(map[int64]string)
	for _, ms := range milestonesByEngagement {
		for _, m := range ms {
			milestoneName[m.ID] = m.Name
		}
	}
	vendorNames := h.refs.VendorNames(r.Context(), claims.tenantID, vendorIDs)
	evidences, err := h.evidence.List(r.Context(), projectID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	ix := indexEvidence(evidences, claims.isClient, claims.can)
	var picNames map[int64]string
	if !claims.isClient {
		picIDs := make([]int64, 0, len(list))
		for _, i := range list {
			picIDs = append(picIDs, i.PICStaffID)
		}
		picNames = h.projects.StaffNames(r.Context(), claims.tenantID, picIDs)
	}
	result := make([]issueResponse, 0, len(list))
	for _, i := range list {
		resp := toIssueResponse(i)
		resp.PICName = picNames[i.PICStaffID]
		resp.VendorName = vendorNames[vendorOf[i.ProjectVendorID]]
		if i.VendorMilestoneID != nil {
			resp.VendorMilestoneName = milestoneName[*i.VendorMilestoneID]
		}
		resp.Evidence = ix.refs(domain.RelatedIssue, i.ID)
		result = append(result, resp)
	}
	response.OK(w, "ok", result)
}

type issueInputBody struct {
	// ProjectVendorID is optional: omitted, null or 0 is a kendala about the event
	// in general.
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
	// IsClientVisible: omitted means shown to the client (the default).
	IsClientVisible *bool `json:"isClientVisible"`
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
	if err := h.refs.Staff(r.Context(), claims.tenantID, "picStaffId", body.PICStaffID); err != nil {
		writeAppError(w, err)
		return
	}
	issue, err := h.issues.Create(r.Context(), projectID, claims.staffID, application.IssueInput{
		ProjectVendorID: body.ProjectVendorID, VendorMilestoneID: body.VendorMilestoneID, Title: body.Title, Description: body.Description,
		Impact: domain.IssueImpact(body.Impact), FoundDate: foundDate, ResolutionPlan: body.ResolutionPlan,
		PICStaffID: body.PICStaffID, TargetResolutionDate: parseOptionalDate(body.TargetResolutionDate),
		IsClientVisible: body.IsClientVisible,
	})
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.Created(w, "Kendala berhasil dicatat", h.mutated(r.Context(), claims.tenantID, projectID, h.issueResponseFor(r.Context(), claims.tenantID, projectID, *issue)))
}

// updateIssue is PATCH /projects/{id}/issues/{iid}: a partial update (ADR-0036) of
// projectVendorId (null or 0: a kendala about the event in general),
// vendorMilestoneId (nullable), title, description, impact, foundDate, status,
// resolutionPlan (null clears), picStaffId, targetResolutionDate (nullable) and
// isClientVisible. Only the fields present change; a status change to
// Resolved/Closed stamps resolvedDate once. See updateProject for the body,
// If-Match and 409 rules.
func (h *Handler) updateIssue(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64, issueIDRaw string) {
	issueID, err := parseInt64(issueIDRaw)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "ID tidak valid", nil)
		return
	}
	expected, ok := ifMatch(w, r)
	if !ok {
		return
	}
	pr, ok := newPatchReader(w, r)
	if !ok {
		return
	}
	impact, status := pr.text("impact", false), pr.text("status", false)
	patch := application.IssuePatch{
		ProjectVendorID: pr.integer("projectVendorId", true), VendorMilestoneID: pr.integer("vendorMilestoneId", true),
		PICStaffID: pr.integer("picStaffId", false),
		Title:      pr.text("title", false), Description: pr.text("description", false), ResolutionPlan: pr.text("resolutionPlan", true),
		FoundDate: pr.date("foundDate", false), TargetResolutionDate: pr.date("targetResolutionDate", true),
		IsClientVisible: pr.boolean("isClientVisible"),
	}
	if impact.Present() {
		patch.Impact = application.Some(domain.IssueImpact(impact.Value()))
	}
	if status.Present() {
		patch.Status = application.Some(domain.IssueStatus(status.Value()))
	}
	if err := pr.finish(); err != nil {
		writeAppError(w, err)
		return
	}
	if patch.PICStaffID.Present() {
		if err := h.refs.Staff(r.Context(), claims.tenantID, "picStaffId", patch.PICStaffID.Value()); err != nil {
			writeAppError(w, err)
			return
		}
	}
	issue, err := h.issues.Patch(r.Context(), projectID, issueID, claims.staffID, patch, time.Now(), expected)
	if err != nil {
		writePatchError(w, err, func(cur any) (any, error) {
			return h.issueResponseFor(r.Context(), claims.tenantID, projectID, *cur.(*domain.VendorIssue)), nil
		})
		return
	}
	response.OK(w, "Kendala berhasil diperbarui", h.mutated(r.Context(), claims.tenantID, projectID, h.issueResponseFor(r.Context(), claims.tenantID, projectID, *issue)))
}

// issueResponseFor is a kendala as a mutation answers it: the row plus the
// names the list also carries (the PIC, the vendor and the vendor milestone,
// "" when there is none), without another round of listing the whole project.
func (h *Handler) issueResponseFor(ctx context.Context, tenantID, projectID int64, i domain.VendorIssue) issueResponse {
	resp := toIssueResponse(i)
	resp.PICName = h.projects.ResolvePICName(ctx, tenantID, i.PICStaffID)
	if i.HasVendor() {
		if pv, err := h.vendors.Get(ctx, projectID, i.ProjectVendorID); err == nil {
			resp.VendorName = h.refs.VendorNames(ctx, tenantID, []int64{pv.VendorID})[pv.VendorID]
		}
		if i.VendorMilestoneID != nil {
			if ms, err := h.vendors.ListMilestones(ctx, projectID, i.ProjectVendorID); err == nil {
				for _, m := range ms {
					if m.ID == *i.VendorMilestoneID {
						resp.VendorMilestoneName = m.Name
					}
				}
			}
		}
	}
	return resp
}
