package presentation

import (
	"net/http"
	"strconv"
)

// Client Portal data boundary (docs/plan/registrasi-wo-self-service/PLAN.md
// Fase 1, gap G1). A `client` principal used to be limited only by HTTP
// method (GET), which let a bride/groom read every sub-resource of their own
// project — vendor payments, venue payments, the internal activity log, and
// every uploaded document including ones never shared with them. Everything
// a client may read is now an explicit allowlist, and each readable response
// is redacted down to the client-facing fields, so the boundary holds at the
// API rather than relying on what the portal UI happens to render.

// clientReadableSubresources is every /projects/{id}/<rest> path a client
// principal may GET. Anything not listed here — notably "payments" (vendor
// payments), "venue-payments" and "activity" — is internal to the WO.
var clientReadableSubresources = map[string]bool{
	"milestones": true,
	"events":     true,
	"suppliers":  true,
	// The Rincian ("Layanan › Paket & rincian", PortalPolicy.Line).
	"lines":           true,
	"client-payments": true,
	"payment-terms":   true,
	"issues":          true,
	"evidence":        true,
	"documents":       true,
	// Payments the client made straight to a supplier they pay (Tahap 4 adds
	// the submission); the WO's own supplier payments are filtered out by
	// PortalPolicy.SupplierPayment.
	"supplier-payments": true,
	// The bills of the vendors the client pays directly, what is paid and left
	// and the proofs sent (Paket WO Tahap 4, PortalPolicy.BillTerm).
	"supplier-bills": true,
	// The hero photo of the portal (ADR-0047).
	"cover": true,
}

// clientCanRead reports whether a client principal may GET
// /projects/{id}/<rest...>. rest is the path after the project id.
func clientCanRead(rest []string) bool {
	switch len(rest) {
	case 0:
		return true
	case 1:
		return clientReadableSubresources[rest[0]]
	case 3:
		// /projects/{id}/evidence/{evidenceId}/file (streamed) and /link (a
		// direct two-minute link) — per-file visibility is checked again in
		// downloadEvidence and linkEvidence via canSeeEvidence.
		return rest[0] == "evidence" && (rest[2] == "file" || rest[2] == "link")
	default:
		return false
	}
}

// Portal permissions of the client portal's writes (ADR-0047).
const (
	permPortalFinanceSubmit = "portal.finance.submit"
	permPortalTasksComplete = "portal.tasks.complete"
)

// clientWritePermission is the complete list of non-GET requests a client
// principal may make under /projects/{id}/ and the portal permission each
// needs (ADR-0047): the signed upload of a transfer proof (both halves), the
// proof submission itself (to the WO, or to a vendor the client pays), and
// marking a task done. ok is false for anything
// else. Keep in step with cmd/server/client_scope.go.
func clientWritePermission(method string, rest []string) (perm string, ok bool) {
	switch {
	case method == http.MethodPost && len(rest) == 1 && rest[0] == "uploads":
		return permPortalFinanceSubmit, true
	case method == http.MethodPut && len(rest) == 2 && rest[0] == "uploads" && rest[1] != "":
		return permPortalFinanceSubmit, true
	case method == http.MethodPost && len(rest) == 2 && rest[0] == "client-payments" && rest[1] == "submissions":
		return permPortalFinanceSubmit, true
	// The proof of a payment to a vendor the client pays directly (Paket WO Tahap 4).
	case method == http.MethodPost && len(rest) == 2 && rest[0] == "supplier-bills" && rest[1] == "submissions":
		return permPortalFinanceSubmit, true
	case method == http.MethodPost && len(rest) == 3 && rest[0] == "milestones" && isPositiveID(rest[1]) && rest[2] == "complete":
		return permPortalTasksComplete, true
	}
	return "", false
}

func isPositiveID(s string) bool {
	n, err := strconv.ParseInt(s, 10, 64)
	return err == nil && n > 0
}

// redactProjectForClient removes fields that are internal to the WO: the
// preparation start and the Sales PIC.
func redactProjectForClient(p *projectResponse) {
	p.PrepStartDate = ""
	p.PICSalesStaffID = 0
	p.Quotation = nil
}

// redactSupplierForClient keeps what the portal's Layanan tab renders (the
// service, who provides it when the client may know, its status and steps)
// and drops the WO's commercial relationship and staff-only notes. name is
// what PortalPolicy.SupplierName allows.
func redactSupplierForClient(v *supplierResponse, name string) {
	v.Name = name
	v.SupplierID = 0
	v.AgreedCost = nil
	v.PaidWO = nil
	v.DueDate = nil
	v.BookingDate = nil
	v.Notes = ""
	v.PICStaffID = 0
	v.PICName = ""
	v.ShowName = false
	v.CapacityStanding, v.CapacitySeated = nil, nil
	v.PaidByLocked = ""
	v.UsedByEvents = 0
	for i := range v.Milestones {
		v.Milestones[i].Notes = ""
		v.Milestones[i].PICStaffID = 0
	}
}

// redactClientPaymentForClient drops the staff-only note and who recorded a
// payment the client made; amount, date, method, reference, the verification
// (when and by whom) and the visible proof files stay visible to them.
func redactClientPaymentForClient(p *clientPaymentResponse) {
	p.Notes = ""
	p.RecordedBy = nil
}

// redactEvidenceForClient hides which staff member uploaded a document.
func redactEvidenceForClient(e *evidenceResponse) {
	e.UploadedBy = 0
}
