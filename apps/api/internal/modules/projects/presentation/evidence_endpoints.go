package presentation

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"elproof/internal/modules/projects/application"
	"elproof/internal/modules/projects/domain"
	"elproof/internal/shared/middleware"
	"elproof/internal/shared/response"
)

func (h *Handler) listEvidence(w http.ResponseWriter, r *http.Request, tenantID, projectID int64, isClient bool) {
	list, err := h.evidence.List(r.Context(), projectID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	var names map[int64]string
	if !isClient {
		ids := make([]int64, 0, len(list))
		for _, e := range list {
			ids = append(ids, e.UploadedByStaffID)
		}
		names = h.projects.StaffNames(r.Context(), tenantID, ids)
	}
	result := make([]evidenceResponse, 0, len(list))
	for _, e := range list {
		if isClient && !clientCanSeeEvidence(e, middleware.GrantsFrom(r.Context()).Can) {
			continue
		}
		resp := toEvidenceResponse(e)
		resp.UploadedByName = names[e.UploadedByStaffID]
		if isClient {
			redactEvidenceForClient(&resp)
		}
		result = append(result, resp)
	}
	response.OK(w, "ok", result)
}

// listDocuments backs GET /projects/{id}/documents — Client Portal's own
// "Dokumen" tab. Reachable by staff too; the safety property is that this
// endpoint ALWAYS only returns general-kind, client-visible documents,
// unconditionally, regardless of who's calling — see
// EvidenceService.ListClientDocuments.
func (h *Handler) listDocuments(w http.ResponseWriter, r *http.Request, projectID int64) {
	list, err := h.evidence.ListClientDocuments(r.Context(), projectID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	result := make([]evidenceResponse, 0, len(list))
	for _, e := range list {
		result = append(result, toEvidenceResponse(e))
	}
	response.OK(w, "ok", result)
}

type evidenceUploadBody struct {
	Name            string `json:"name"`
	Type            string `json:"type"`
	FileName        string `json:"fileName"`
	MimeType        string `json:"mimeType"`
	Base64Data      string `json:"base64Data"`
	DocumentDate    string `json:"documentDate"`
	Description     string `json:"description"`
	RelatedKind     string `json:"relatedKind"`
	RelatedID       int64  `json:"relatedId"`
	IsClientVisible bool   `json:"isClientVisible"`
}

func (h *Handler) uploadEvidence(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64) {
	var body evidenceUploadBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
		return
	}
	e, err := h.evidence.Upload(r.Context(), claims.tenantID, projectID, claims.staffID, application.UploadEvidenceInput{
		Name: body.Name, Type: domain.EvidenceType(body.Type), FileName: body.FileName, MimeType: body.MimeType,
		Base64Data: body.Base64Data, DocumentDate: parseOptionalDate(body.DocumentDate), Description: body.Description,
		RelatedKind: domain.EvidenceRelatedKind(body.RelatedKind), RelatedID: body.RelatedID,
		IsClientVisible: body.IsClientVisible,
	})
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.Created(w, "Evidence berhasil diunggah", h.mutated(r.Context(), claims.tenantID, projectID, h.evidenceResponseFor(r.Context(), claims.tenantID, *e)))
}

// toggleEvidenceClientVisible flips a general-kind document's client
// visibility without re-uploading the file — see
// EvidenceService.ToggleClientVisible for why this is rejected for every
// other RelatedKind.
func (h *Handler) toggleEvidenceClientVisible(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64, evidenceIDRaw string) {
	evidenceID, err := parseInt64(evidenceIDRaw)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "ID tidak valid", nil)
		return
	}
	e, err := h.evidence.ToggleClientVisible(r.Context(), projectID, evidenceID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "Visibilitas dokumen diperbarui", h.mutated(r.Context(), claims.tenantID, projectID, h.evidenceResponseFor(r.Context(), claims.tenantID, *e)))
}

// evidenceResponseFor is a file as a staff mutation answers it: the row plus the
// uploader's name.
func (h *Handler) evidenceResponseFor(ctx context.Context, tenantID int64, e domain.Evidence) evidenceResponse {
	resp := toEvidenceResponse(e)
	resp.UploadedByName = h.projects.ResolvePICName(ctx, tenantID, e.UploadedByStaffID)
	return resp
}

// deleteEvidence is DELETE /projects/{id}/evidence/{eid}: removes one proof file
// for good (row and stored object). It needs project.edit (or finance.record),
// like every evidence write, and finance.payment.delete when the file is the
// proof of a payment, since removing it changes what the payment is worth. A
// verified client payment loses its verification with its last file (ADR-0032).
// The answer is {"id", "projectProgress"}.
func (h *Handler) deleteEvidence(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64, evidenceIDRaw string) {
	evidenceID, err := parseInt64(evidenceIDRaw)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "ID tidak valid", nil)
		return
	}
	e, err := h.evidence.Find(r.Context(), projectID, evidenceID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	switch e.RelatedKind {
	case domain.RelatedClientPayment, domain.RelatedPayment, domain.RelatedVenuePayment:
		if !claims.can("finance.payment.delete") {
			forbiddenPermission(w, "Peran Anda tidak dapat menghapus bukti pembayaran")
			return
		}
	}
	if err := h.evidence.Delete(r.Context(), projectID, evidenceID, claims.staffID); err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "Evidence berhasil dihapus", h.mutated(r.Context(), claims.tenantID, projectID, map[string]int64{"id": evidenceID}))
}

func (h *Handler) downloadEvidence(w http.ResponseWriter, r *http.Request, projectID int64, evidenceIDRaw string, isClient bool) {
	evidenceID, err := parseInt64(evidenceIDRaw)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "ID tidak valid", nil)
		return
	}
	e, err := h.evidence.Find(r.Context(), projectID, evidenceID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	// Same rule as listEvidence: a client who learns an internal document's
	// id still cannot fetch its file. 404 (not 403) so the response doesn't
	// confirm the document exists. Judged before storage is touched, so the
	// answer never depends on the file being there.
	if isClient && !clientCanSeeEvidence(*e, middleware.GrantsFrom(r.Context()).Can) {
		response.Error(w, http.StatusNotFound, "Dokumen tidak ditemukan", nil)
		return
	}
	reader, err := h.evidence.Open(r.Context(), e)
	if err != nil {
		writeAppError(w, err)
		return
	}
	defer reader.Close()

	w.Header().Set("Content-Disposition", `inline; filename="`+e.FileName+`"`)
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, reader)
}

// linkEvidence is GET /projects/{id}/evidence/{evidenceId}/link[?disposition=attachment]:
// a direct, two-minute link to the stored file, so the bytes go from object
// storage to the browser instead of through this server. It applies exactly
// the rules of downloadEvidence (project scope, client visibility); when the
// deployment has no direct links the answer is 501 presign_unavailable and the
// web falls back to /file.
func (h *Handler) linkEvidence(w http.ResponseWriter, r *http.Request, projectID int64, evidenceIDRaw string, isClient bool) {
	evidenceID, err := parseInt64(evidenceIDRaw)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "ID tidak valid", nil)
		return
	}
	attachment := r.URL.Query().Get("disposition") == "attachment"
	e, url, expiresAt, err := h.evidence.Link(r.Context(), projectID, evidenceID, attachment)
	// A client may not learn that an internal document exists, even by the
	// shape of an error, so visibility is judged before anything else is said.
	if e != nil && isClient && !clientCanSeeEvidence(*e, middleware.GrantsFrom(r.Context()).Can) {
		response.Error(w, http.StatusNotFound, "Dokumen tidak ditemukan", nil)
		return
	}
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "ok", map[string]string{"url": url, "expiresAt": expiresAt.In(wib).Format(time.RFC3339)})
}
