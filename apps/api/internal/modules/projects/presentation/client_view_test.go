package presentation

import (
	"testing"

	"elproof/internal/modules/projects/domain"
)

func TestClientCanRead(t *testing.T) {
	ok := [][]string{
		{},
		{"milestones"}, {"vendors"}, {"client-payments"}, {"issues"},
		{"evidence"}, {"documents"}, {"venue"},
		{"evidence", "9", "file"},
	}
	for _, rest := range ok {
		if !clientCanRead(rest) {
			t.Errorf("client should read %v", rest)
		}
	}
	denied := [][]string{
		{"payments"}, {"venue-payments"}, {"activity"},
		{"vendors", "4", "milestones"}, {"evidence", "9"}, {"milestones", "3"},
	}
	for _, rest := range denied {
		if clientCanRead(rest) {
			t.Errorf("client must not read %v", rest)
		}
	}
}

func TestClientCanSeeEvidence(t *testing.T) {
	cases := []struct {
		kind    domain.EvidenceRelatedKind
		visible bool
		want    bool
	}{
		{domain.RelatedClientPayment, false, true},
		{domain.RelatedIssue, false, true},
		{domain.RelatedVendorMilestone, false, true},
		{domain.RelatedGeneral, true, true},
		{domain.RelatedGeneral, false, false},
		{domain.RelatedPayment, false, false},
		{domain.RelatedVenuePayment, false, false},
		{domain.RelatedProjectVendor, false, false},
		{domain.RelatedProjectMilestone, false, false},
	}
	for _, c := range cases {
		got := clientCanSeeEvidence(domain.Evidence{RelatedKind: c.kind, IsClientVisible: c.visible}, func(string) bool { return true })
		if got != c.want {
			t.Errorf("kind=%s visible=%v: got %v, want %v", c.kind, c.visible, got, c.want)
		}
	}
	// Without portal.finance.view (Keluarga) payment receipts stay hidden,
	// while documents the WO shared remain visible.
	noFinance := func(p string) bool { return p != "portal.finance.view" }
	if clientCanSeeEvidence(domain.Evidence{RelatedKind: domain.RelatedClientPayment}, noFinance) {
		t.Error("payment receipt visible without portal.finance.view")
	}
	if !clientCanSeeEvidence(domain.Evidence{RelatedKind: domain.RelatedGeneral, IsClientVisible: true}, noFinance) {
		t.Error("shared document hidden without portal.finance.view")
	}
}

func TestRedactionRemovesInternalFields(t *testing.T) {
	rental, charge := int64(50_000_000), int64(2_000_000)
	p := projectResponse{VenueRentalPrice: &rental, VenueCharge: &charge, PrepStartDate: "2026-01-01", PICSalesStaffID: 4, ContractValue: 150_000_000}
	redactProjectForClient(&p)
	if p.VenueRentalPrice != nil || p.VenueCharge != nil || p.PrepStartDate != "" || p.PICSalesStaffID != 0 {
		t.Errorf("project internal fields not redacted: %+v", p)
	}
	if p.ContractValue != 150_000_000 {
		t.Error("the client's own contract value must stay visible")
	}

	due, booked := "2026-05-01", "2026-01-10"
	v := projectVendorResponse{
		ContractValue: 10, DPAmount: 5, PaidAmount: 5, DueDate: &due, BookingDate: &booked, PricingTier: "Akad", Notes: "nego",
		EngagementStatus: "Booked", Milestones: []vendorMilestoneResponse{{Name: "Fitting", Notes: "internal"}},
	}
	redactVendorForClient(&v)
	if v.ContractValue != 0 || v.DPAmount != 0 || v.PaidAmount != 0 || v.DueDate != nil || v.BookingDate != nil || v.PricingTier != "" || v.Notes != "" {
		t.Errorf("vendor commercial fields not redacted: %+v", v)
	}
	if v.Milestones[0].Notes != "" || v.Milestones[0].Name != "Fitting" || v.EngagementStatus != "Booked" {
		t.Errorf("vendor milestone redaction wrong: %+v", v.Milestones[0])
	}

	cp := clientPaymentResponse{Amount: 1000, Notes: "internal"}
	redactClientPaymentForClient(&cp)
	if cp.Notes != "" || cp.Amount != 1000 {
		t.Errorf("client payment redaction wrong: %+v", cp)
	}
}
