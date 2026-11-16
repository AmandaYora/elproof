package domain

import (
	"testing"
	"time"
)

// ADR-0047: a payment the couple sent from the Client Portal counts as paid only
// once staff verified it; a staff-recorded payment always counts.
func TestContractLinesWaitForVerificationOfClientSubmissions(t *testing.T) {
	verified := time.Date(2026, 11, 3, 9, 0, 0, 0, time.UTC)
	payments := []ClientPayment{
		{ID: 1, Type: PaymentDP, Amount: 3_000_000},                                                    // staff, unverified
		{ID: 2, Type: PaymentTermin, Amount: 2_000_000, SubmittedByClientID: 7},                        // couple, pending
		{ID: 3, Type: PaymentTermin, Amount: 1_000_000, SubmittedByClientID: 7, VerifiedAt: &verified}, // couple, verified
	}
	if payments[1].CountsTowardContract() {
		t.Error("a pending client submission does not count")
	}
	if !payments[0].CountsTowardContract() || !payments[2].CountsTowardContract() {
		t.Error("staff payments and verified submissions count")
	}
	lines := ContractLines(payments)
	if got := AppliedToContract(lines); got != 4_000_000 {
		t.Errorf("applied = %d, want 4.000.000", got)
	}
	if l := ComputeLedger(lines, 10_000_000); l.Remaining != 6_000_000 {
		t.Errorf("remaining = %d, want 6.000.000", l.Remaining)
	}
}
