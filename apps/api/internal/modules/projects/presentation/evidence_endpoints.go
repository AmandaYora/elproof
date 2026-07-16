package presentation

import (
	"encoding/json"
	"io"
	"net/http"

	"elproof/internal/modules/projects/application"
	"elproof/internal/modules/projects/domain"
	"elproof/internal/shared/response"
)

func (h *Handler) listEvidence(w http.ResponseWriter, r *http.Request, projectID int64) {
	list, err := h.evidence.List(r.Context(), projectID)
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
	Name         string `json:"name"`
	Type         string `json:"type"`
	FileName     string `json:"fileName"`
	MimeType     string `json:"mimeType"`
	Base64Data   string `json:"base64Data"`
	DocumentDate string `json:"documentDate"`
	Description  string `json:"description"`
	RelatedKind  string `json:"relatedKind"`
	RelatedID    int64  `json:"relatedId"`
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
	})
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.Created(w, "Evidence berhasil diunggah", toEvidenceResponse(*e))
}

func (h *Handler) downloadEvidence(w http.ResponseWriter, r *http.Request, projectID int64, evidenceIDRaw string) {
	evidenceID, err := parseInt64(evidenceIDRaw)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "ID tidak valid", nil)
		return
	}
	e, reader, err := h.evidence.Download(r.Context(), projectID, evidenceID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	defer reader.Close()

	w.Header().Set("Content-Disposition", `inline; filename="`+e.FileName+`"`)
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, reader)
}
