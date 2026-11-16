package presentation

import (
	"net/http"
	"strconv"

	"elproof/internal/modules/projects/domain"
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
	"milestones":      true,
	"vendors":         true,
	"client-payments": true,
	"payment-terms":   true,
	"issues":          true,
	"evidence":        true,
	"documents":       true,
	"venue":           true,
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
		// downloadEvidence and linkEvidence via clientCanSeeEvidence.
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
// proof submission itself, and marking a task done. ok is false for anything
// else. Keep in step with cmd/server/client_scope.go.
func clientWritePermission(method string, rest []string) (perm string, ok bool) {
	switch {
	case method == http.MethodPost && len(rest) == 1 && rest[0] == "uploads":
		return permPortalFinanceSubmit, true
	case method == http.MethodPut && len(rest) == 2 && rest[0] == "uploads" && rest[1] != "":
		return permPortalFinanceSubmit, true
	case method == http.MethodPost && len(rest) == 2 && rest[0] == "client-payments" && rest[1] == "submissions":
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

// clientCanSeeEvidence is the per-document visibility rule for a client
// principal; the rule itself lives in domain.ClientCanSeeEvidence so the
// Client Portal home (application layer) applies the very same one.
func clientCanSeeEvidence(e domain.Evidence, can func(string) bool) bool {
	return domain.ClientCanSeeEvidence(e, can)
}

// redactProjectForClient removes fields that are internal to the WO: the
// venue's commercial terms, the booking date and the Sales PIC.
func redactProjectForClient(p *projectResponse) {
	p.VenueRentalPrice = nil
	p.VenueCharge = nil
	p.PrepStartDate = ""
	p.PICSalesStaffID = 0
}

// redactVendorForClient keeps what the portal's Vendor tab renders (which
// vendor, category, scope, status, schedule and deliverables) and drops the
// WO's commercial relationship with the vendor and staff-only notes.
func redactVendorForClient(v *projectVendorResponse) {
	v.ContractValue = 0
	v.DPAmount = 0
	v.PaidAmount = 0
	v.DueDate = nil
	v.BookingDate = nil
	v.PricingTier = ""
	v.Notes = ""
	for i := range v.Milestones {
		v.Milestones[i].Notes = ""
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
