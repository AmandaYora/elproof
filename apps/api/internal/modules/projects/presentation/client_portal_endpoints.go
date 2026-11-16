package presentation

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"elproof/internal/modules/projects/application"
	"elproof/internal/shared/response"
)

// Client Portal additions (ADR-0047): the project's cover photo (staff upload,
// both read), the couple's transfer-proof submission and their tasks.

// maxCoverBody bounds PUT /projects/{id}/cover: 6 MB of photo as base64 plus
// the JSON around it.
const maxCoverBody = 6*1024*1024*4/3 + 4096

// maxSubmissionBody bounds the proof submission (a few short fields and tokens).
const maxSubmissionBody = 64 << 10

type coverBody struct {
	FileName   string `json:"fileName"`
	MimeType   string `json:"mimeType"`
	Base64Data string `json:"base64Data"`
}

type coverResponse struct {
	CoverURL *string `json:"coverUrl"`
}

// SetCovers gives the handler the cover photo service.
func (h *Handler) SetCovers(s *application.CoverService) { h.covers = s }

// putCover is PUT /projects/{id}/cover (staff, project.edit — enforced by
// staffWritePermission like every project write, and the same project scoping):
// {fileName, mimeType, base64Data}; JPEG, PNG or WebP of at most 6 MB whose bytes
// are that type. The server re-encodes it (longest side 2000 px, no metadata)
// and replaces any previous cover. 200 {coverUrl}.
func (h *Handler) putCover(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64) {
	if claims.isClient {
		forbiddenPermission(w, "Hanya anggota tim WO yang dapat mengubah foto sampul")
		return
	}
	var body coverBody
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxCoverBody)).Decode(&body); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			response.Error(w, http.StatusRequestEntityTooLarge, "Ukuran foto terlalu besar",
				map[string][]string{"base64Data": {"Ukuran foto maksimal 6 MB."}})
			return
		}
		response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
		return
	}
	p, err := h.covers.Set(r.Context(), claims.tenantID, projectID, claims.staffID, application.CoverInput{
		FileName: body.FileName, MimeType: body.MimeType, Base64Data: body.Base64Data,
	})
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "Foto sampul berhasil disimpan", coverResponse{CoverURL: coverURL(*p)})
}

// deleteCover is DELETE /projects/{id}/cover (same permission as putCover):
// 200 {coverUrl: null}, also when there was no cover.
func (h *Handler) deleteCover(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64) {
	if claims.isClient {
		forbiddenPermission(w, "Hanya anggota tim WO yang dapat mengubah foto sampul")
		return
	}
	if err := h.covers.Remove(r.Context(), claims.tenantID, projectID, claims.staffID); err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "Foto sampul dihapus", coverResponse{CoverURL: nil})
}

// getCover is GET /projects/{id}/cover: the photo itself, for staff who can
// reach the project and for the project's own client principal (no portal
// permission: it is the hero of every portal page). 404 when there is none.
// The URL carries ?v=<last change>, so a short private cache is safe.
func (h *Handler) getCover(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64) {
	reader, contentType, err := h.covers.Open(r.Context(), claims.tenantID, projectID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	defer reader.Close()
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "private, max-age=300")
	w.Header().Set("Content-Disposition", "inline")
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, reader)
}

// clientSubmissionBody is POST /projects/{id}/client-payments/submissions.
type clientSubmissionBody struct {
	Amount          int64       `json:"amount"`
	PaymentDate     string      `json:"paymentDate"`
	Method          string      `json:"method"`
	ReferenceNumber string      `json:"referenceNumber"`
	Notes           string      `json:"notes"`
	Evidence        []proofBody `json:"evidence"`
}

// submitClientPayment is POST /projects/{id}/client-payments/submissions: the
// couple sends a transfer proof from the portal (client principal of this
// project with portal.finance.submit — checked in resolveProjectAccess). The
// proof files were uploaded first with POST/PUT /projects/{id}/uploads. Under
// the optional Idempotency-Key header. 201 with the payment as the portal lists
// it (unverified, source "client").
func (h *Handler) submitClientPayment(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64) {
	if !claims.isClient {
		forbiddenPermission(w, "Bukti dari pasangan hanya dikirim lewat Portal Klien. Catat pembayaran lewat menu Keuangan.")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxSubmissionBody)
	h.idempotent(w, r, claims, func(w http.ResponseWriter, r *http.Request) {
		var body clientSubmissionBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
			return
		}
		paymentDate, err := parseDate(body.PaymentDate)
		if err != nil {
			response.Error(w, http.StatusUnprocessableEntity, "Tanggal transfer tidak terbaca", map[string][]string{"paymentDate": {"Pilih tanggal transfer."}})
			return
		}
		created, err := h.clientPayments.SubmitByClient(r.Context(), claims.tenantID, projectID, claims.clientID, application.ClientSubmissionInput{
			Amount: body.Amount, PaymentDate: paymentDate, Method: body.Method, ReferenceNumber: body.ReferenceNumber, Notes: body.Notes,
		}, toProofItems(body.Evidence), nowWIB())
		if err != nil {
			writeAppError(w, err)
			return
		}
		resp, err := h.oneClientPaymentResponse(r.Context(), claims, projectID, *created.Payment)
		if err != nil {
			writeAppError(w, err)
			return
		}
		response.Created(w, "Bukti pembayaran terkirim. WO akan memeriksanya.", resp)
	})
}

// completeMilestoneByClient is POST /projects/{id}/milestones/{mid}/complete:
// the couple marks one of their tasks done (client principal of this project
// with portal.tasks.complete). Only a step staff flagged forClient (else 403
// not_client_task), never a cancelled one (422 task_cancelled). Idempotent:
// an already completed step is answered 200 unchanged. 200 with the milestone.
func (h *Handler) completeMilestoneByClient(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64, milestoneIDRaw string) {
	if !claims.isClient {
		forbiddenPermission(w, "Tandai selesai dari Jadwal proyek. Aksi ini khusus untuk pasangan di Portal Klien.")
		return
	}
	milestoneID, err := parseInt64(milestoneIDRaw)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "ID tugas tidak valid", nil)
		return
	}
	m, err := h.projects.CompleteMilestoneByClient(r.Context(), claims.tenantID, projectID, milestoneID, claims.clientID, nowWIB())
	if err != nil {
		writeAppError(w, err)
		return
	}
	item, err := h.milestoneItem(r.Context(), claims, projectID, *m)
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "Tugas ditandai selesai", item)
}
