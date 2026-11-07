package domain

import "testing"

func TestIsValidEvidenceType(t *testing.T) {
	for _, ok := range []EvidenceType{EvidenceContract, EvidenceTransferProof, EvidenceBookingProof, EvidenceOther} {
		if !IsValidEvidenceType(ok) {
			t.Errorf("%q must be valid", ok)
		}
	}
	for _, bad := range []EvidenceType{"", "Kontrak", "contract", "Contract "} {
		if IsValidEvidenceType(bad) {
			t.Errorf("%q must be refused", bad)
		}
	}
}
