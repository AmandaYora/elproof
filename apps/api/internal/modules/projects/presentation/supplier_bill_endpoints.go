package presentation

import (
	"context"
	"encoding/json"
	"net/http"

	"elproof/internal/modules/projects/application"
	"elproof/internal/modules/projects/domain"
	"elproof/internal/shared/clock"
	"elproof/internal/shared/etag"
	"elproof/internal/shared/response"
)

// Bills of the vendors and venues the client pays directly ("Tagihan vendor",
// Paket WO Tahap 4, PLAN §3.2, §4.3, §4.4):
//
//	GET    /projects/{id}/supplier-bills                        staff; client with portal.finance.view
//	POST   /projects/{id}/supplier-bills/terms                  finance.record (Idempotency-Key)
//	PATCH  /projects/{id}/supplier-bills/terms/{tid}            finance.record (If-Match)
//	DELETE /projects/{id}/supplier-bills/terms/{tid}            finance.record, soft delete
//	POST   /projects/{id}/supplier-bills/terms/{tid}/restore    finance.record
//	POST   /projects/{id}/supplier-bills/submissions            client with portal.finance.submit (Idempotency-Key)
//
// The WO checks a submission with POST /projects/{id}/supplier-payments/{pid}/verify
// or /reject. None of this money is the WO's: payer = Client everywhere.
func (h *Handler) routeSupplierBills(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64, rest []string) {
	if h.supplierBills == nil {
		response.Error(w, http.StatusNotFound, "Endpoint tidak ditemukan", nil)
		return
	}
	switch {
	case len(rest) == 0 && r.Method == http.MethodGet:
		h.listSupplierBills(w, r, claims, projectID)
	case len(rest) == 1 && rest[0] == "terms" && r.Method == http.MethodPost:
		h.idempotent(w, r, claims, func(w http.ResponseWriter, r *http.Request) {
			h.createSupplierBillTerm(w, r, claims, projectID)
		})
	case len(rest) == 2 && rest[0] == "terms" && r.Method == http.MethodPatch:
		h.updateSupplierBillTerm(w, r, claims, projectID, rest[1])
	case len(rest) == 2 && rest[0] == "terms" && r.Method == http.MethodDelete:
		h.deleteSupplierBillTerm(w, r, claims, projectID, rest[1])
	case len(rest) == 3 && rest[0] == "terms" && rest[2] == "restore" && r.Method == http.MethodPost:
		h.restoreSupplierBillTerm(w, r, claims, projectID, rest[1])
	case len(rest) == 1 && rest[0] == "submissions" && r.Method == http.MethodPost:
		h.submitSupplierBill(w, r, claims, projectID)
	default:
		response.Error(w, http.StatusNotFound, "Endpoint tidak ditemukan", nil)
	}
}

// supplierBillTermResponse is one bill with where it stands.
type supplierBillTermResponse struct {
	ID                int64  `json:"id"`
	ProjectSupplierID int64  `json:"projectSupplierId"`
	Label             string `json:"label"`
	Amount            int64  `json:"amount"`
	DueDate           string `json:"dueDate"`
	// Paid is what the counted payments cover of this bill (earliest due date
	// first); Remaining what is left; Pending the client's proofs sent for this
	// bill that the WO has not checked yet (not counted as paid).
	Paid      int64 `json:"paid"`
	Remaining int64 `json:"remaining"`
	Pending   int64 `json:"pending"`
	// Status: paid | overdue | partial | due_soon | upcoming (as the client's
	// own installments).
	Status    string `json:"status"`
	UpdatedAt string `json:"updatedAt"`
}

func toSupplierBillTermResponse(t domain.BillTermProgress) supplierBillTermResponse {
	return supplierBillTermResponse{
		ID: t.Term.ID, ProjectSupplierID: t.Term.ProjectSupplierID, Label: t.Term.Label, Amount: t.Term.Amount,
		DueDate: t.Term.DueDate.Format(dateLayout), Paid: t.Paid, Remaining: t.Remaining, Pending: t.Pending,
		Status: string(t.Status), UpdatedAt: etag.Format(t.Term.UpdatedAt),
	}
}

// supplierBillResponse is one vendor or venue the client pays directly: its
// real name (the client pays it, so it is always named), its bills, its
// payments and the totals.
type supplierBillResponse struct {
	ProjectSupplierID int64                      `json:"projectSupplierId"`
	Name              string                     `json:"name"`
	Kind              string                     `json:"kind"`
	Total             int64                      `json:"total"`
	Paid              int64                      `json:"paid"`
	Remaining         int64                      `json:"remaining"`
	Pending           int64                      `json:"pending"`
	Terms             []supplierBillTermResponse `json:"terms"`
	Payments          []supplierPaymentResponse  `json:"payments"`
}

// supplierBillResponses shapes the bills for the caller: a client sees them
// only with portal.finance.view (PortalPolicy.BillTerm), without the WO's
// notes or staff names; staff see who checked each proof.
func (h *Handler) supplierBillResponses(ctx context.Context, claims staffClaims, projectID int64, bills []domain.SupplierBill) ([]supplierBillResponse, error) {
	view, err := h.portalViewFor(ctx, claims, projectID)
	if err != nil {
		return nil, err
	}
	evidences, err := h.evidence.List(ctx, projectID)
	if err != nil {
		return nil, err
	}
	hasInvoice, hasProof := domain.PaymentEvidenceStatus(evidences, domain.RelatedSupplierPayment)
	ix := indexEvidence(evidences, view)
	var names map[int64]string
	if !claims.isClient {
		var staffIDs []int64
		for _, b := range bills {
			for _, p := range b.Payments {
				staffIDs = append(staffIDs, p.VerifiedByStaffID, p.RejectedByStaffID)
			}
		}
		names = h.projects.StaffNames(ctx, claims.tenantID, staffIDs)
	}
	out := make([]supplierBillResponse, 0, len(bills))
	for _, b := range bills {
		if claims.isClient && !domain.NewPortalPolicy(claims.can).BillTerm(b.Supplier) {
			continue
		}
		resp := supplierBillResponse{
			ProjectSupplierID: b.Supplier.ID, Name: b.Supplier.Name, Kind: string(b.Supplier.Kind),
			Total: b.Total, Paid: b.Paid, Remaining: b.Remaining, Pending: b.Pending,
			Terms: make([]supplierBillTermResponse, 0, len(b.Terms)), Payments: make([]supplierPaymentResponse, 0, len(b.Payments)),
		}
		for _, t := range b.Terms {
			resp.Terms = append(resp.Terms, toSupplierBillTermResponse(t))
		}
		for _, p := range b.Payments {
			pr := toSupplierPaymentResponse(p, domain.IsPaymentEvidenceComplete(p.Type, p.ID, hasInvoice, hasProof))
			pr.Evidence = ix.refs(domain.RelatedSupplierPayment, p.ID)
			if claims.isClient {
				pr.Notes = ""
			} else {
				withReviewers(&pr, p, names)
			}
			resp.Payments = append(resp.Payments, pr)
		}
		out = append(out, resp)
	}
	return out, nil
}

// listSupplierBills is GET /projects/{id}/supplier-bills: every vendor or venue
// the client pays directly, with its bills and payments (a supplier the WO
// pays is never listed). Empty for a client without portal.finance.view (the
// route itself already needs it).
func (h *Handler) listSupplierBills(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64) {
	bills, err := h.supplierBills.Bills(r.Context(), claims.tenantID, projectID, nowWIB())
	if err != nil {
		writeAppError(w, err)
		return
	}
	out, err := h.supplierBillResponses(r.Context(), claims, projectID, bills)
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "ok", out)
}

// supplierBillStanding is where the bill's supplier stands after a change (the
// `supplierBill` extra of a mutation response), nil when it cannot be built.
func (h *Handler) supplierBillStanding(ctx context.Context, claims staffClaims, projectID, projectSupplierID int64) *supplierBillResponse {
	bills, err := h.supplierBills.Bills(ctx, claims.tenantID, projectID, nowWIB())
	if err != nil {
		return nil
	}
	for _, b := range bills {
		if b.Supplier.ID != projectSupplierID {
			continue
		}
		out, err := h.supplierBillResponses(ctx, claims, projectID, []domain.SupplierBill{b})
		if err != nil || len(out) == 0 {
			return nil
		}
		return &out[0]
	}
	return nil
}

type supplierBillTermBody struct {
	ProjectSupplierID int64  `json:"projectSupplierId"`
	Label             string `json:"label"`
	Amount            int64  `json:"amount"`
	DueDate           string `json:"dueDate"`
}

// billTermItem is a bill as the list shows it, as of today.
func (h *Handler) billTermItem(ctx context.Context, claims staffClaims, projectID int64, t domain.SupplierBillTerm) supplierBillTermResponse {
	if b := h.supplierBillStanding(ctx, claims, projectID, t.ProjectSupplierID); b != nil {
		for _, tr := range b.Terms {
			if tr.ID == t.ID {
				return tr
			}
		}
	}
	return toSupplierBillTermResponse(domain.BillTermProgress{Term: t, Remaining: t.Amount, Status: domain.TermStatusOf(0, t.Amount, t.DueDate, nowWIB())})
}

// createSupplierBillTerm adds a bill to a vendor or venue the client pays (422
// with code supplier_paid_by_wo on one the WO pays).
func (h *Handler) createSupplierBillTerm(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64) {
	var body supplierBillTermBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
		return
	}
	dueDate, err := parseDate(body.DueDate)
	if err != nil {
		response.Error(w, http.StatusUnprocessableEntity, "Isi tanggal jatuh tempo.", map[string][]string{"dueDate": {"Isi tanggal jatuh tempo."}})
		return
	}
	t, err := h.supplierBills.CreateBill(r.Context(), claims.tenantID, projectID, claims.staffID, application.SupplierBillInput{
		ProjectSupplierID: body.ProjectSupplierID, Label: body.Label, Amount: body.Amount, DueDate: dueDate,
	})
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.Created(w, "Tagihan vendor ditambahkan", h.mutated(r.Context(), claims.tenantID, projectID,
		h.billTermItem(r.Context(), claims, projectID, *t), extra{"supplierBill", h.supplierBillStanding(r.Context(), claims, projectID, t.ProjectSupplierID)}))
}

// updateSupplierBillTerm is PATCH .../supplier-bills/terms/{tid}: label,
// amount, dueDate (If-Match; 428 without, 409 with data.current when stale).
func (h *Handler) updateSupplierBillTerm(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64, raw string) {
	id, err := parseInt64(raw)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "ID tidak valid", nil)
		return
	}
	expected, ok := requireIfMatch(w, r, func(ctx context.Context) error {
		_, err := h.supplierBills.GetBill(ctx, claims.tenantID, projectID, id)
		return err
	})
	if !ok {
		return
	}
	pr, ok := newPatchReader(w, r)
	if !ok {
		return
	}
	patch := application.SupplierBillPatch{
		Label: pr.text("label", false), Amount: pr.integer("amount", false), DueDate: pr.date("dueDate", false),
	}
	if err := pr.finish(); err != nil {
		writeAppError(w, err)
		return
	}
	t, err := h.supplierBills.PatchBill(r.Context(), claims.tenantID, projectID, id, claims.staffID, patch, expected)
	if err != nil {
		writePatchError(w, err, func(cur any) (any, error) {
			return h.billTermItem(r.Context(), claims, projectID, *cur.(*domain.SupplierBillTerm)), nil
		})
		return
	}
	response.OK(w, "Tagihan vendor diperbarui", h.mutated(r.Context(), claims.tenantID, projectID,
		h.billTermItem(r.Context(), claims, projectID, *t), extra{"supplierBill", h.supplierBillStanding(r.Context(), claims, projectID, t.ProjectSupplierID)}))
}

// deleteSupplierBillTerm soft-deletes a bill (restorable for 24 hours).
func (h *Handler) deleteSupplierBillTerm(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64, raw string) {
	id, err := parseInt64(raw)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "ID tidak valid", nil)
		return
	}
	t, until, err := h.supplierBills.DeleteBill(r.Context(), claims.tenantID, projectID, id, claims.staffID, clock.Now())
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "Tagihan vendor dihapus", h.mutated(r.Context(), claims.tenantID, projectID,
		toDeletedResponse(id, until), extra{"supplierBill", h.supplierBillStanding(r.Context(), claims, projectID, t.ProjectSupplierID)}))
}

// restoreSupplierBillTerm brings a deleted bill back (404 once purged; 422 on
// projectSupplierId while its supplier is deleted).
func (h *Handler) restoreSupplierBillTerm(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64, raw string) {
	id, err := parseInt64(raw)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "ID tidak valid", nil)
		return
	}
	t, err := h.supplierBills.RestoreBill(r.Context(), claims.tenantID, projectID, id, claims.staffID, clock.Now())
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "Tagihan vendor dipulihkan", h.mutated(r.Context(), claims.tenantID, projectID,
		h.billTermItem(r.Context(), claims, projectID, *t), extra{"supplierBill", h.supplierBillStanding(r.Context(), claims, projectID, t.ProjectSupplierID)}))
}

// supplierBillSubmissionBody is the couple's proof for one bill.
type supplierBillSubmissionBody struct {
	BillTermID      int64  `json:"billTermId"`
	Amount          int64  `json:"amount"`
	PaymentDate     string `json:"paymentDate"`
	Method          string `json:"method"`
	ReferenceNumber string `json:"referenceNumber"`
	Notes           string `json:"notes"`
	// Evidence names transfer proofs already uploaded with POST
	// /projects/{id}/uploads (type "Transfer Proof", at least one).
	Evidence []proofBody `json:"evidence"`
}

// supplierSubmittedResponse is the payment the couple sent plus where that
// vendor now stands for them.
type supplierSubmittedResponse struct {
	supplierPaymentResponse
	SupplierBill *supplierBillResponse `json:"supplierBill"`
}

// submitSupplierBill is POST /projects/{id}/supplier-bills/submissions: the
// couple (client principal of this project with portal.finance.submit) sends
// the proof of a payment they made to a vendor they pay directly. It is
// recorded unverified ("Menunggu diperiksa wedding organizer Anda") and counts
// as paid only once the WO marks it right. A staff member is refused (they
// record such a payment with POST .../supplier-payments).
func (h *Handler) submitSupplierBill(w http.ResponseWriter, r *http.Request, claims staffClaims, projectID int64) {
	if !claims.isClient {
		forbiddenPermission(w, "Bukti dari pasangan hanya dikirim lewat Portal Klien. Catat pembayaran ke vendor lewat menu Keuangan.")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxSubmissionBody)
	h.idempotent(w, r, claims, func(w http.ResponseWriter, r *http.Request) {
		var body supplierBillSubmissionBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
			return
		}
		paymentDate, err := parseDate(body.PaymentDate)
		if err != nil {
			response.Error(w, http.StatusUnprocessableEntity, "Tanggal transfer tidak terbaca", map[string][]string{"paymentDate": {"Pilih tanggal transfer."}})
			return
		}
		created, err := h.supplierBills.SubmitByClient(r.Context(), claims.tenantID, projectID, claims.clientID, application.SupplierSubmissionInput{
			BillTermID: body.BillTermID, Amount: body.Amount, PaymentDate: paymentDate, Method: body.Method,
			ReferenceNumber: body.ReferenceNumber, Notes: body.Notes,
		}, toProofItems(body.Evidence), nowWIB())
		if err != nil {
			writeAppError(w, err)
			return
		}
		hasInvoice, hasProof := domain.PaymentEvidenceStatus(created.Evidence, domain.RelatedSupplierPayment)
		p := created.Payment
		resp := toSupplierPaymentResponse(*p, domain.IsPaymentEvidenceComplete(p.Type, p.ID, hasInvoice, hasProof))
		resp.Evidence = evidenceRefs(created.Evidence)
		resp.Notes = ""
		var standing *supplierBillResponse
		if out, err := h.supplierBillResponses(r.Context(), claims, projectID, []domain.SupplierBill{created.Bill}); err == nil && len(out) == 1 {
			standing = &out[0]
		}
		response.Created(w, "Bukti pembayaran terkirim. Wedding organizer Anda akan memeriksanya.",
			supplierSubmittedResponse{supplierPaymentResponse: resp, SupplierBill: standing})
	})
}
