package presentation

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"elproof/internal/modules/projects/application"
	"elproof/internal/modules/projects/domain"
)

func TestClientCanRead(t *testing.T) {
	ok := [][]string{
		{},
		{"milestones"}, {"vendors"}, {"client-payments"}, {"issues"},
		{"evidence"}, {"documents"}, {"venue"}, {"payment-terms"},
		{"evidence", "9", "file"},
	}
	for _, rest := range ok {
		if !clientCanRead(rest) {
			t.Errorf("client should read %v", rest)
		}
	}
	denied := [][]string{
		{"payments"}, {"venue-payments"}, {"activity"}, {"closing-checklist"}, {"transition"},
		{"payment-terms", "1"},
		{"vendors", "4", "milestones"}, {"evidence", "9"}, {"milestones", "3"},
		{"client-payments", "3", "verify"}, {"evidence", "9", "toggle-client-visible"},
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
		// A timeline attachment is visible only once staff flag it (ADR-0032).
		{domain.RelatedProjectMilestone, true, true},
		{domain.RelatedPayment, true, false},
		{domain.RelatedVenuePayment, true, false},
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
	// A role without portal.documents.view sees no shared documents, but its
	// payment, step and issue proof still follow their own permissions.
	noDocs := func(p string) bool { return p != "portal.documents.view" }
	if clientCanSeeEvidence(domain.Evidence{RelatedKind: domain.RelatedGeneral, IsClientVisible: true}, noDocs) {
		t.Error("shared document visible without portal.documents.view")
	}
	if !clientCanSeeEvidence(domain.Evidence{RelatedKind: domain.RelatedClientPayment}, noDocs) {
		t.Error("payment proof must not depend on portal.documents.view")
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

	verifiedAt := "2026-03-02T10:00:00+07:00"
	cp := clientPaymentResponse{
		Amount: 1000, Notes: "internal", RecordedBy: &personRef{Name: "Staf"},
		VerifiedAt: &verifiedAt, VerifiedBy: &personRef{Name: "Pemverifikasi"},
		Evidence: []evidenceRefResponse{{ID: 1, Type: "Transfer Proof", FileName: "bukti.jpg"}},
	}
	redactClientPaymentForClient(&cp)
	if cp.Notes != "" || cp.Amount != 1000 {
		t.Errorf("client payment redaction wrong: %+v", cp)
	}
	if cp.RecordedBy != nil {
		t.Error("who recorded a payment is internal: recordedBy must be null for a client")
	}
	if cp.VerifiedAt == nil || cp.VerifiedBy == nil || len(cp.Evidence) != 1 {
		t.Errorf("the verification and the proof stay visible to the client: %+v", cp)
	}
}

func TestIndexEvidenceFiltersForClientsAndKeepsStaffView(t *testing.T) {
	up := func(h int) time.Time { return time.Date(2026, 1, 1, h, 0, 0, 0, time.UTC) }
	// Newest first, as the repository lists them.
	list := []domain.Evidence{
		{ID: 4, RelatedKind: domain.RelatedProjectMilestone, RelatedID: 1, Type: domain.EvidencePhoto, FileName: "d.jpg", UploadedAt: up(4), IsClientVisible: true},
		{ID: 3, RelatedKind: domain.RelatedProjectMilestone, RelatedID: 1, Type: domain.EvidencePhoto, FileName: "c.jpg", UploadedAt: up(3)},
		{ID: 2, RelatedKind: domain.RelatedClientPayment, RelatedID: 7, Type: domain.EvidenceTransferProof, FileName: "b.pdf", UploadedAt: up(2)},
		{ID: 1, RelatedKind: domain.RelatedPayment, RelatedID: 7, Type: domain.EvidenceInvoice, FileName: "a.pdf", UploadedAt: up(1)},
	}
	all := func(string) bool { return true }
	noFinance := func(p string) bool { return p != "portal.finance.view" }

	staff := indexEvidence(list, false, nil)
	if got := staff.refs(domain.RelatedProjectMilestone, 1); len(got) != 2 || got[0].ID != 3 || got[1].ID != 4 {
		t.Errorf("staff see every file of the step, oldest first: %+v", got)
	}

	client := indexEvidence(list, true, all)
	if got := client.refs(domain.RelatedProjectMilestone, 1); len(got) != 1 || got[0].ID != 4 {
		t.Errorf("a client sees only the flagged file: %+v", got)
	}
	if got := client.refs(domain.RelatedClientPayment, 7); len(got) != 1 || got[0].ID != 2 || got[0].FileName != "b.pdf" || got[0].UploadedAt == "" {
		t.Errorf("a client sees their payment proof: %+v", got)
	}
	// payments (vendor) 7 shares the numeric id with client payment 7: kinds never mix.
	if got := client.refs(domain.RelatedPayment, 7); len(got) != 0 {
		t.Errorf("vendor payment files are never shown to a client: %+v", got)
	}

	keluarga := indexEvidence(list, true, noFinance)
	if got := keluarga.refs(domain.RelatedClientPayment, 7); got == nil || len(got) != 0 {
		t.Errorf("without portal.finance.view there is no payment proof, and the field is [] not null: %#v", got)
	}
	// evidenceComplete is a fact about the payment, not about the viewer.
	if !keluarga.has(domain.RelatedClientPayment, 7) {
		t.Error("the payment still has a file, whoever looks")
	}
}

func TestHomeResponseOmitsSectionsTheRoleCannotSee(t *testing.T) {
	// A Keluarga-style client: no finance, no timeline, no issues.
	home := &application.ClientHome{
		Project:     application.HomeProject{ID: 1, Name: "Pernikahan", EventDate: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)},
		DaysToEvent: 12,
		PICName:     "Dewi", PICPhone: "0812",
	}
	raw, err := json.Marshal(toClientHomeResponse(home))
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]json.RawMessage
	_ = json.Unmarshal(raw, &m)
	for _, forbidden := range []string{"progress", "finance", "payTo", "contact", "openIssueCount"} {
		if _, present := m[forbidden]; present {
			t.Errorf("%q must be absent (not null) when the role may not see it: %s", forbidden, raw)
		}
	}
	for _, required := range []string{"project", "countdown", "actions", "recentUpdates", "pic"} {
		if _, present := m[required]; !present {
			t.Errorf("%q is always present: %s", required, raw)
		}
	}
	if string(m["actions"]) != "[]" || string(m["recentUpdates"]) != "[]" {
		t.Errorf("empty lists serialize as [], got %s / %s", m["actions"], m["recentUpdates"])
	}
	for _, leak := range []string{"contractValue", "picSalesStaffId", "prepStartDate", "venueRentalPrice", "venueCharge", "recordedBy", "notes"} {
		if strings.Contains(string(raw), leak) {
			t.Errorf("the home must never carry %q: %s", leak, raw)
		}
	}
}

func TestHomeResponseShapeWithEverythingVisible(t *testing.T) {
	zero := 0
	due := time.Date(2026, 5, 20, 0, 0, 0, 0, time.UTC)
	amount := int64(500_000)
	home := &application.ClientHome{
		Project:  application.HomeProject{ID: 1, Name: "Pernikahan", EventDate: due},
		Progress: &application.HomeProgress{OverallPercent: 40, Condition: "On Track"},
		Finance: &application.HomeFinance{ContractValue: 10, Paid: 4, Remaining: 6, OverdueCount: 0,
			NextTerm: &application.HomeNextTerm{ID: 3, Label: "Termin 1", Amount: 600, DueDate: due, Status: "upcoming", Remaining: 600}},
		PayTo:          &application.HomePayTo{BankName: "BCA", AccountNumber: "123", AccountHolder: "WO"},
		WhatsApp:       "62812",
		Actions:        []domain.HomeAction{{Kind: domain.HomeActionPayTerm, Title: "t", Detail: "d", DueDate: &due, Amount: &amount, Target: "pembayaran"}, {Kind: domain.HomeActionOpenIssues, Title: "1 hal", Target: "kendala"}},
		OpenIssueCount: &zero,
		RecentUpdates: []application.HomeUpdate{
			{Kind: domain.UpdatePaymentVerified, Title: "DP diterima", At: due, Proof: &domain.UpdateProof{EvidenceID: 9, Type: domain.EvidenceTransferProof, FileName: "b.pdf"}, VerifiedBy: "Rina"},
			{Kind: domain.UpdateMilestoneDone, Title: "Dekorasi selesai", At: due},
		},
	}
	raw, _ := json.Marshal(toClientHomeResponse(home))
	var m map[string]any
	_ = json.Unmarshal(raw, &m)
	if _, ok := m["openIssueCount"]; !ok || m["openIssueCount"].(float64) != 0 {
		t.Errorf("a visible issue count of 0 is sent as 0: %s", raw)
	}
	fin := m["finance"].(map[string]any)
	if fin["nextTerm"].(map[string]any)["dueDate"] != "2026-05-20" || fin["remaining"].(float64) != 6 {
		t.Errorf("finance: %v", fin)
	}
	acts := m["actions"].([]any)
	if _, has := acts[1].(map[string]any)["dueDate"]; has {
		t.Errorf("an issues action has no dueDate/amount keys: %v", acts[1])
	}
	ups := m["recentUpdates"].([]any)
	first, second := ups[0].(map[string]any), ups[1].(map[string]any)
	if first["verifiedBy"] != "Rina" || first["proof"].(map[string]any)["evidenceId"].(float64) != 9 {
		t.Errorf("payment update: %v", first)
	}
	if _, has := second["verifiedBy"]; has || second["proof"] != nil {
		t.Errorf("a milestone update has proof null and no verifiedBy: %v", second)
	}
}

func TestFinanceNextTermNullWhenNone(t *testing.T) {
	home := &application.ClientHome{Finance: &application.HomeFinance{ContractValue: 5}}
	raw, _ := json.Marshal(toClientHomeResponse(home))
	if !strings.Contains(string(raw), `"nextTerm":null`) {
		t.Errorf("nextTerm is null when no installment is open: %s", raw)
	}
}
