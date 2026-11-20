package presentation

import (
	"context"
	"encoding/json"
	"net/http"

	"elproof/internal/modules/projects/application"
	"elproof/internal/modules/projects/domain"
	"elproof/internal/shared/clock"
	"elproof/internal/shared/response"
)

// Payments to a project's vendors and venues ("Pembayaran rekanan" when the WO
// pays, "dibayar klien langsung" otherwise):
//
//	GET    /projects/{id}/supplier-payments
//	POST   /projects/{id}/supplier-payments                 (Idempotency-Key)
//	PATCH  /projects/{id}/supplier-payments/{pid}           (If-Match)
//	DELETE /projects/{id}/supplier-payments/{pid}           soft delete
//	POST   /projects/{id}/supplier-payments/{pid}/restore
//	POST   /projects/{id}/supplier-payments/{pid}/verify    client's proof: "Tandai sesuai" (finance.record)
//	POST   /projects/{id}/supplier-payments/{pid}/reject    client's proof: "Tolak dengan alasan" (finance.record)
//
// Who may see and record a payment follows who pays the supplier: the WO's
// payments are for finance.margin.view, the client's for finance.record
// (PLAN §4.3). A Wedding Planner neither sees nor records the WO's.
func (h *Handler) routeSupplierPayments(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64, rest []string) {
	switch {
	case len(rest) == 0 && r.Method == http.MethodGet:
		h.listSupplierPayments(w, r, claims, projectID)
	case len(rest) == 0 && r.Method == http.MethodPost:
		h.idempotent(w, r, claims, func(w http.ResponseWriter, r *http.Request) {
			h.createSupplierPayment(w, r, claims, projectID)
		})
	case len(rest) == 1 && r.Method == http.MethodPatch:
		h.updateSupplierPayment(w, r, claims, projectID, rest[0])
	case len(rest) == 1 && r.Method == http.MethodDelete:
		h.deleteSupplierPayment(w, r, claims, projectID, rest[0])
	case len(rest) == 2 && rest[1] == "restore" && r.Method == http.MethodPost:
		h.restoreSupplierPayment(w, r, claims, projectID, rest[0])
	case len(rest) == 2 && rest[1] == "verify" && r.Method == http.MethodPost:
		h.verifySupplierSubmission(w, r, claims, projectID, rest[0])
	case len(rest) == 2 && rest[1] == "reject" && r.Method == http.MethodPost:
		h.rejectSupplierSubmission(w, r, claims, projectID, rest[0])
	default:
		response.Error(w, http.StatusNotFound, "Endpoint tidak ditemukan", nil)
	}
}

// canSeeSupplierPayment: staff see the client's payments to suppliers; the
// WO's need finance.margin.view.
func (c staffClaims) canSeeSupplierPayment(payer domain.Payer) bool {
	return payer != domain.PayerWO || c.canSeeMargin()
}

// canRecordSupplierPayment is the permission to record, change or bring back a
// payment by payer.
func (c staffClaims) canRecordSupplierPayment(payer domain.Payer) bool {
	if payer == domain.PayerWO {
		return c.canSeeMargin()
	}
	return c.can("finance.record")
}

func forbiddenSupplierPayment(w http.ResponseWriter) {
	forbiddenPermission(w, "Peran Anda tidak dapat mencatat pembayaran ke rekanan")
}

func (h *Handler) listSupplierPayments(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64) {
	list, err := h.supplierPayments.List(r.Context(), projectID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	view, err := h.portalViewFor(r.Context(), claims, projectID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	evidences, err := h.evidence.List(r.Context(), projectID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	hasInvoice, hasProof := domain.PaymentEvidenceStatus(evidences, domain.RelatedSupplierPayment)
	ix := indexEvidence(evidences, view)
	var names map[int64]string
	if !view.forClient() {
		var staffIDs []int64
		for _, p := range list {
			staffIDs = append(staffIDs, p.VerifiedByStaffID, p.RejectedByStaffID)
		}
		names = h.projects.StaffNames(r.Context(), claims.tenantID, staffIDs)
	}
	result := make([]supplierPaymentResponse, 0, len(list))
	for _, p := range list {
		if view.forClient() && !view.policy.SupplierPayment(p) {
			continue
		}
		if !view.forClient() && !claims.canSeeSupplierPayment(p.Payer) {
			continue
		}
		resp := toSupplierPaymentResponse(p, domain.IsPaymentEvidenceComplete(p.Type, p.ID, hasInvoice, hasProof))
		resp.Evidence = ix.refs(domain.RelatedSupplierPayment, p.ID)
		if view.forClient() {
			resp.Notes = ""
		} else {
			withReviewers(&resp, p, names)
		}
		result = append(result, resp)
	}
	response.OK(w, "ok", result)
}

// withReviewers names the staff member who checked a client's proof (staff
// view only; the portal never names the WO's staff).
func withReviewers(resp *supplierPaymentResponse, p domain.SupplierPayment, names map[int64]string) {
	if name, ok := names[p.VerifiedByStaffID]; ok && p.VerifiedAt != nil {
		resp.VerifiedBy = &personRef{Name: name}
	}
	if name, ok := names[p.RejectedByStaffID]; ok && p.RejectedAt != nil {
		resp.RejectedBy = &personRef{Name: name}
	}
}

type supplierPaymentInputBody struct {
	ProjectSupplierID int64  `json:"projectSupplierId"`
	Type              string `json:"type"`
	Amount            int64  `json:"amount"`
	PaymentDate       string `json:"paymentDate"`
	Method            string `json:"method"`
	ReferenceNumber   string `json:"referenceNumber"`
	Notes             string `json:"notes"`
	// Evidence names files already uploaded with POST /projects/{id}/uploads
	// ("Invoice" and/or "Transfer Proof", at most 4); they are recorded in the
	// same transaction as the payment.
	Evidence []proofBody `json:"evidence"`
}

// supplierPaymentCreatedResponse is the payment (as listed) plus where that
// supplier stands against its payer's payments.
type supplierPaymentCreatedResponse struct {
	supplierPaymentResponse
	LedgerSummary ledgerSummaryResponse `json:"ledgerSummary"`
}

// createSupplierPayment records a payment to a supplier, optionally with its
// proof, as one atomic action. The payer is the supplier's "dibayar oleh".
func (h *Handler) createSupplierPayment(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64) {
	var body supplierPaymentInputBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
		return
	}
	sup, err := h.supplierPayments.Supplier(r.Context(), projectID, body.ProjectSupplierID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	if !claims.canRecordSupplierPayment(sup.PaidBy) {
		forbiddenSupplierPayment(w)
		return
	}
	paymentDate, err := parseDate(body.PaymentDate)
	if err != nil {
		response.Error(w, http.StatusUnprocessableEntity, "Format tanggal tidak valid", map[string][]string{"paymentDate": {"Gunakan format YYYY-MM-DD"}})
		return
	}
	created, err := h.supplierPayments.CreateWithProof(r.Context(), claims.tenantID, projectID, claims.staffID, application.SupplierPaymentInput{
		ProjectSupplierID: body.ProjectSupplierID, Type: domain.PaymentType(body.Type), Amount: body.Amount,
		PaymentDate: paymentDate, Method: body.Method, ReferenceNumber: body.ReferenceNumber, Notes: body.Notes,
	}, toProofItems(body.Evidence))
	if err != nil {
		writeAppError(w, err)
		return
	}
	hasInvoice, hasProof := domain.PaymentEvidenceStatus(created.Evidence, domain.RelatedSupplierPayment)
	p := created.Payment
	resp := toSupplierPaymentResponse(*p, domain.IsPaymentEvidenceComplete(p.Type, p.ID, hasInvoice, hasProof))
	resp.Evidence = evidenceRefs(created.Evidence)
	response.Created(w, "Pembayaran berhasil dicatat", h.mutated(r.Context(), claims.tenantID, projectID,
		supplierPaymentCreatedResponse{supplierPaymentResponse: resp, LedgerSummary: toLedgerSummary(created.Ledger)}))
}

// readPaymentPatch reads the body of a PATCH on a client or supplier payment:
// type, amount, paymentDate, method, referenceNumber and notes (null clears the
// note). Who it was paid to is not patchable.
func readPaymentPatch(pr *patchReader) application.PaymentPatch {
	t := pr.text("type", false)
	patch := application.PaymentPatch{
		Amount: pr.integer("amount", false), PaymentDate: pr.date("paymentDate", false),
		Method: pr.text("method", false), ReferenceNumber: pr.text("referenceNumber", false), Notes: pr.text("notes", true),
	}
	if t.Present() {
		patch.Type = application.Some(domain.PaymentType(t.Value()))
	}
	return patch
}

// supplierPaymentItem is a payment as the list shows it (staff view).
func (h *Handler) supplierPaymentItem(ctx context.Context, claims staffClaims, projectID int64, p domain.SupplierPayment) (supplierPaymentResponse, error) {
	evidences, err := h.evidence.List(ctx, projectID)
	if err != nil {
		return supplierPaymentResponse{}, err
	}
	hasInvoice, hasProof := domain.PaymentEvidenceStatus(evidences, domain.RelatedSupplierPayment)
	resp := toSupplierPaymentResponse(p, domain.IsPaymentEvidenceComplete(p.Type, p.ID, hasInvoice, hasProof))
	resp.Evidence = indexEvidence(evidences, nil).refs(domain.RelatedSupplierPayment, p.ID)
	withReviewers(&resp, p, h.projects.StaffNames(ctx, claims.tenantID, []int64{p.VerifiedByStaffID, p.RejectedByStaffID}))
	return resp, nil
}

// allowedPayment loads a live payment and checks the caller may act on it by
// its payer (a payment the caller may not see answers 404, like a missing one).
func (h *Handler) allowedPayment(w http.ResponseWriter, ctx context.Context, claims staffClaims, projectID int64, raw string) (int64, bool) {
	id, err := parseInt64(raw)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "ID tidak valid", nil)
		return 0, false
	}
	p, err := h.supplierPayments.Get(ctx, projectID, id)
	if err != nil {
		writeAppError(w, err)
		return 0, false
	}
	if !claims.canSeeSupplierPayment(p.Payer) {
		response.Error(w, http.StatusNotFound, "Pembayaran tidak ditemukan", nil)
		return 0, false
	}
	if !claims.canRecordSupplierPayment(p.Payer) {
		forbiddenSupplierPayment(w)
		return 0, false
	}
	return id, true
}

// updateSupplierPayment is PATCH /projects/{id}/supplier-payments/{pid}: a
// partial update (type, amount, paymentDate, method, referenceNumber, notes).
// What the WO paid is checked against the agreed cost on the merged result.
// See updateProject for the body, If-Match (428 without) and 409 rules.
func (h *Handler) updateSupplierPayment(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64, raw string) {
	paymentID, ok := h.allowedPayment(w, r.Context(), claims, projectID, raw)
	if !ok {
		return
	}
	expected, ok := requireIfMatch(w, r, func(ctx context.Context) error {
		return h.supplierPayments.Exists(ctx, projectID, paymentID)
	})
	if !ok {
		return
	}
	pr, ok := newPatchReader(w, r)
	if !ok {
		return
	}
	patch := readPaymentPatch(pr)
	if err := pr.finish(); err != nil {
		writeAppError(w, err)
		return
	}
	p, err := h.supplierPayments.Patch(r.Context(), projectID, paymentID, claims.staffID, patch, expected)
	if err != nil {
		writePatchError(w, err, func(cur any) (any, error) {
			return h.supplierPaymentItem(r.Context(), claims, projectID, *cur.(*domain.SupplierPayment))
		})
		return
	}
	resp, err := h.supplierPaymentItem(r.Context(), claims, projectID, *p)
	if err != nil {
		writeAppError(w, err)
		return
	}
	ledger, err := h.supplierPayments.LedgerSummary(r.Context(), projectID, p.ProjectSupplierID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "Pembayaran berhasil diperbarui", h.mutated(r.Context(), claims.tenantID, projectID, resp, extra{"ledgerSummary", toLedgerSummary(ledger)}))
}

// deleteSupplierPayment soft-deletes the payment (finance.payment.delete, and
// for the WO's own payments finance.margin.view); see deleteClientPayment.
func (h *Handler) deleteSupplierPayment(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64, raw string) {
	paymentID, ok := h.allowedPayment(w, r.Context(), claims, projectID, raw)
	if !ok {
		return
	}
	p, until, err := h.supplierPayments.Delete(r.Context(), projectID, paymentID, claims.staffID, clock.Now())
	if err != nil {
		writeAppError(w, err)
		return
	}
	ledger, err := h.supplierPayments.LedgerSummary(r.Context(), projectID, p.ProjectSupplierID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "Pembayaran berhasil dihapus", h.mutated(r.Context(), claims.tenantID, projectID, toDeletedResponse(paymentID, until), extra{"ledgerSummary", toLedgerSummary(ledger)}))
}

// restoreSupplierPayment is POST .../supplier-payments/{pid}/restore
// (finance.payment.delete). 404 when it is not deleted or was purged.
func (h *Handler) restoreSupplierPayment(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64, raw string) {
	paymentID, err := parseInt64(raw)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "ID tidak valid", nil)
		return
	}
	deleted, err := h.supplierPayments.FindDeleted(r.Context(), projectID, paymentID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	if deleted != nil && !claims.canSeeSupplierPayment(deleted.Payer) {
		response.Error(w, http.StatusNotFound, "Pembayaran tidak ditemukan", nil)
		return
	}
	// Bringing a payment back is recording it again: the same payer rule as create, change and delete.
	if deleted != nil && !claims.canRecordSupplierPayment(deleted.Payer) {
		forbiddenSupplierPayment(w)
		return
	}
	p, err := h.supplierPayments.Restore(r.Context(), projectID, paymentID, claims.staffID, clock.Now())
	if err != nil {
		writeAppError(w, err)
		return
	}
	resp, err := h.supplierPaymentItem(r.Context(), claims, projectID, *p)
	if err != nil {
		writeAppError(w, err)
		return
	}
	ledger, err := h.supplierPayments.LedgerSummary(r.Context(), projectID, p.ProjectSupplierID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "Pembayaran dipulihkan", h.mutated(r.Context(), claims.tenantID, projectID, resp, extra{"ledgerSummary", toLedgerSummary(ledger)}))
}

// verifySupplierSubmission is POST .../supplier-payments/{pid}/verify: the WO
// marks the client's proof of a payment to a vendor they pay right ("Tandai
// sesuai", finance.record). Idempotent; 422 without a file (evidence), or with
// code submission_rejected for a refused proof; 404 for a payment the WO
// recorded itself.
func (h *Handler) verifySupplierSubmission(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64, raw string) {
	if !claims.can("finance.record") {
		forbiddenPermission(w, "Peran Anda tidak dapat memeriksa bukti pembayaran")
		return
	}
	paymentID, err := parseInt64(raw)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "ID tidak valid", nil)
		return
	}
	p, err := h.supplierBills.Verify(r.Context(), projectID, paymentID, claims.staffID, clock.Now())
	if err != nil {
		writeAppError(w, err)
		return
	}
	resp, err := h.supplierPaymentItem(r.Context(), claims, projectID, *p)
	if err != nil {
		writeAppError(w, err)
		return
	}
	ledger, err := h.supplierPayments.LedgerSummary(r.Context(), projectID, p.ProjectSupplierID)
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "Bukti pembayaran ditandai sesuai", h.mutated(r.Context(), claims.tenantID, projectID, resp, extra{"ledgerSummary", toLedgerSummary(ledger)}))
}

type rejectSubmissionBody struct {
	Reason string `json:"reason"`
}

// rejectSupplierSubmission is POST .../supplier-payments/{pid}/reject {reason}:
// the WO refuses the client's proof with a reason the couple reads ("Tolak
// dengan alasan", finance.record). Idempotent on a refused proof; 422 on
// reason, or with code already_verified for a proof marked right.
func (h *Handler) rejectSupplierSubmission(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64, raw string) {
	if !claims.can("finance.record") {
		forbiddenPermission(w, "Peran Anda tidak dapat memeriksa bukti pembayaran")
		return
	}
	paymentID, err := parseInt64(raw)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "ID tidak valid", nil)
		return
	}
	var body rejectSubmissionBody
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxSubmissionBody)).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
		return
	}
	p, err := h.supplierBills.Reject(r.Context(), projectID, paymentID, claims.staffID, body.Reason, clock.Now())
	if err != nil {
		writeAppError(w, err)
		return
	}
	resp, err := h.supplierPaymentItem(r.Context(), claims, projectID, *p)
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "Bukti pembayaran ditolak", h.mutated(r.Context(), claims.tenantID, projectID, resp))
}
