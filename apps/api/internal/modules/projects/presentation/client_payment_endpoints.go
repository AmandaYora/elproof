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

// clientPaymentResponses builds the response rows for the given payments from
// ONE evidence query and ONE staff-name lookup. client_payments has no direct
// evidence-ID column, so evidenceComplete is an existence check against the
// polymorphic evidence table. For a client principal the staff-only fields
// (note, recordedBy) are dropped and only the proof files its role may see are
// embedded.
func (h *Handler) clientPaymentResponses(ctx context.Context, claims staffClaims, projectID int64, list []domain.ClientPayment) ([]clientPaymentResponse, error) {
	evidences, err := h.evidence.List(ctx, projectID)
	if err != nil {
		return nil, err
	}
	ix := indexEvidence(evidences, claims.isClient, claims.can)
	var staffIDs []int64
	for _, p := range list {
		if !claims.isClient {
			staffIDs = append(staffIDs, p.RecordedByStaffID)
		}
		staffIDs = append(staffIDs, p.VerifiedByStaffID)
	}
	names := h.projects.StaffNames(ctx, claims.tenantID, staffIDs)
	result := make([]clientPaymentResponse, 0, len(list))
	for _, p := range list {
		resp := toClientPaymentResponse(p, ix.has(domain.RelatedClientPayment, p.ID))
		resp.Evidence = ix.refs(domain.RelatedClientPayment, p.ID)
		if name, ok := names[p.RecordedByStaffID]; ok && !claims.isClient {
			resp.RecordedBy = &personRef{Name: name}
		}
		if name, ok := names[p.VerifiedByStaffID]; ok && p.IsVerified() {
			resp.VerifiedBy = &personRef{Name: name}
		}
		if claims.isClient {
			redactClientPaymentForClient(&resp)
		}
		result = append(result, resp)
	}
	return result, nil
}

func (h *Handler) listClientPayments(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64) {
	list, err := h.clientPayments.List(r.Context(), projectID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	result, err := h.clientPaymentResponses(r.Context(), claims, projectID, list)
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "ok", result)
}

// oneClientPaymentResponse is the full response row for a single payment.
func (h *Handler) oneClientPaymentResponse(ctx context.Context, claims staffClaims, projectID int64, p domain.ClientPayment) (clientPaymentResponse, error) {
	rows, err := h.clientPaymentResponses(ctx, claims, projectID, []domain.ClientPayment{p})
	if err != nil {
		return clientPaymentResponse{}, err
	}
	return rows[0], nil
}

type clientPaymentInputBody struct {
	Type            string `json:"type"`
	Amount          int64  `json:"amount"`
	PaymentDate     string `json:"paymentDate"`
	Method          string `json:"method"`
	ReferenceNumber string `json:"referenceNumber"`
	Notes           string `json:"notes"`
	// Evidence names files already uploaded with POST /projects/{id}/uploads
	// (type "Transfer Proof", at most 4); they are recorded in the same
	// transaction as the payment.
	Evidence []proofBody `json:"evidence"`
}

// clientPaymentCreatedResponse is the payment (as listed) plus where the
// contract and the payment schedule stand now.
type clientPaymentCreatedResponse struct {
	clientPaymentResponse
	LedgerSummary clientLedgerSummaryResponse `json:"ledgerSummary"`
}

// createClientPayment records a client payment, optionally with its proof, as
// one atomic action, under the optional Idempotency-Key header.
func (h *Handler) createClientPayment(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64) {
	h.idempotent(w, r, claims, func(w http.ResponseWriter, r *http.Request) {
		h.createClientPaymentNow(w, r, claims, projectID)
	})
}

func (h *Handler) createClientPaymentNow(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64) {
	var body clientPaymentInputBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
		return
	}
	paymentDate, err := parseDate(body.PaymentDate)
	if err != nil {
		response.Error(w, http.StatusUnprocessableEntity, "Format tanggal tidak valid", map[string][]string{"paymentDate": {"Gunakan format YYYY-MM-DD"}})
		return
	}
	created, err := h.clientPayments.CreateWithProof(r.Context(), claims.tenantID, projectID, claims.staffID, application.ClientPaymentInput{
		Type: domain.PaymentType(body.Type), Amount: body.Amount,
		PaymentDate: paymentDate, Method: body.Method, ReferenceNumber: body.ReferenceNumber, Notes: body.Notes,
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
	response.Created(w, "Pembayaran client berhasil dicatat", h.mutated(r.Context(), claims.tenantID, projectID, clientPaymentCreatedResponse{
		clientPaymentResponse: resp,
		LedgerSummary: clientLedgerSummaryResponse{
			ledgerSummaryResponse: toLedgerSummary(created.Ledger), Terms: toPaymentTermResponses(created.Terms),
		},
	}))
}

// updateClientPayment is a full-field overwrite, no role-gate — see
// updatePayment's doc comment.
func (h *Handler) updateClientPayment(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64, paymentIDRaw string) {
	paymentID, err := parseInt64(paymentIDRaw)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "ID tidak valid", nil)
		return
	}
	var body clientPaymentInputBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
		return
	}
	paymentDate, err := parseDate(body.PaymentDate)
	if err != nil {
		response.Error(w, http.StatusUnprocessableEntity, "Format tanggal tidak valid", map[string][]string{"paymentDate": {"Gunakan format YYYY-MM-DD"}})
		return
	}
	p, err := h.clientPayments.Update(r.Context(), claims.tenantID, projectID, paymentID, claims.staffID, application.ClientPaymentInput{
		Type: domain.PaymentType(body.Type), Amount: body.Amount,
		PaymentDate: paymentDate, Method: body.Method, ReferenceNumber: body.ReferenceNumber, Notes: body.Notes,
	})
	if err != nil {
		writeAppError(w, err)
		return
	}
	resp, err := h.oneClientPaymentResponse(r.Context(), claims, projectID, *p)
	if err != nil {
		writeAppError(w, err)
		return
	}
	ledger, err := h.clientLedgerSummary(r.Context(), claims.tenantID, projectID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "Pembayaran client berhasil diperbarui", h.mutated(r.Context(), claims.tenantID, projectID, resp, extra{"ledgerSummary", ledger}))
}

// clientLedgerSummary is where the contract and the schedule stand now.
func (h *Handler) clientLedgerSummary(ctx context.Context, tenantID, projectID int64) (clientLedgerSummaryResponse, error) {
	ledger, terms, err := h.clientPayments.LedgerSummary(ctx, tenantID, projectID, nowWIB())
	if err != nil {
		return clientLedgerSummaryResponse{}, err
	}
	return clientLedgerSummaryResponse{ledgerSummaryResponse: toLedgerSummary(ledger), Terms: toPaymentTermResponses(terms)}, nil
}

// verifyClientPayment is POST /projects/{id}/client-payments/{pid}/verify
// (finance.record, enforced for every write under client-payments by
// staffWritePermission; a client principal never reaches it, being GET-only).
// Idempotent; 422 on field `evidence` while no proof file is attached.
func (h *Handler) verifyClientPayment(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64, paymentIDRaw string) {
	paymentID, err := parseInt64(paymentIDRaw)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "ID tidak valid", nil)
		return
	}
	p, err := h.clientPayments.Verify(r.Context(), projectID, paymentID, claims.staffID, time.Now())
	if err != nil {
		writeAppError(w, err)
		return
	}
	resp, err := h.oneClientPaymentResponse(r.Context(), claims, projectID, *p)
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "Pembayaran client diverifikasi", h.mutated(r.Context(), claims.tenantID, projectID, resp))
}

// deleteClientPayment soft-deletes the payment (finance.payment.delete): it
// answers with when it can still be restored, the fresh ledger and schedule, and
// the project's progress. POST .../restore brings it back.
func (h *Handler) deleteClientPayment(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64, paymentIDRaw string) {
	if !claims.can("finance.payment.delete") {
		forbiddenPermission(w, "Peran Anda tidak dapat menghapus pembayaran")
		return
	}
	paymentID, err := parseInt64(paymentIDRaw)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "ID tidak valid", nil)
		return
	}
	until, err := h.clientPayments.Delete(r.Context(), claims.tenantID, projectID, paymentID, claims.staffID, time.Now())
	if err != nil {
		writeAppError(w, err)
		return
	}
	ledger, err := h.clientLedgerSummary(r.Context(), claims.tenantID, projectID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "Pembayaran client berhasil dihapus", h.mutated(r.Context(), claims.tenantID, projectID, toDeletedResponse(paymentID, until), extra{"ledgerSummary", ledger}))
}

// restoreClientPayment is POST /projects/{id}/client-payments/{pid}/restore
// (finance.payment.delete): the payment as listed, plus the ledger and progress.
// 404 when it is not deleted or was purged; 422 `amount` when putting it back
// would overpay the contract.
func (h *Handler) restoreClientPayment(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64, paymentIDRaw string) {
	paymentID, err := parseInt64(paymentIDRaw)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "ID tidak valid", nil)
		return
	}
	p, err := h.clientPayments.Restore(r.Context(), claims.tenantID, projectID, paymentID, claims.staffID, time.Now())
	if err != nil {
		writeAppError(w, err)
		return
	}
	resp, err := h.oneClientPaymentResponse(r.Context(), claims, projectID, *p)
	if err != nil {
		writeAppError(w, err)
		return
	}
	ledger, err := h.clientLedgerSummary(r.Context(), claims.tenantID, projectID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "Pembayaran client dipulihkan", h.mutated(r.Context(), claims.tenantID, projectID, resp, extra{"ledgerSummary", ledger}))
}
