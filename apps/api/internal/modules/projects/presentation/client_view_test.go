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
		{"milestones"}, {"events"}, {"suppliers"}, {"supplier-payments"}, {"client-payments"}, {"issues"},
		{"evidence"}, {"documents"}, {"payment-terms"},
		{"evidence", "9", "file"},
	}
	for _, rest := range ok {
		if !clientCanRead(rest) {
			t.Errorf("client should read %v", rest)
		}
	}
	denied := [][]string{
		{"vendors"}, {"venue"}, {"payments"}, {"venue-payments"}, {"activity"}, {"closing-checklist"}, {"transition"},
		{"payment-terms", "1"}, {"events", "2"}, {"suppliers", "4"},
		{"suppliers", "4", "milestones"}, {"evidence", "9"}, {"milestones", "3"},
		{"client-payments", "3", "verify"}, {"evidence", "9", "visibility"},
	}
	for _, rest := range denied {
		if clientCanRead(rest) {
			t.Errorf("client must not read %v", rest)
		}
	}
}

// owners of the files below: supplier 1 is paid by the WO, supplier 2 by the
// client; milestone 10 belongs to supplier 1, 20 to supplier 2; payment 7 is
// the WO's (to supplier 1), payment 8 the client's (to supplier 2).
var testOwners = domain.EvidenceOwners{
	SupplierPaidBy:  map[int64]domain.Payer{1: domain.PayerWO, 2: domain.PayerClient},
	MilestonePaidBy: map[int64]domain.Payer{10: domain.PayerWO, 20: domain.PayerClient},
	PaymentPayer:    map[int64]domain.Payer{7: domain.PayerWO, 8: domain.PayerClient},
}

func clientView(can func(string) bool) *portalView {
	return &portalView{policy: domain.NewPortalPolicy(can), owners: testOwners}
}

func TestClientCanSeeEvidence(t *testing.T) {
	all := clientView(func(string) bool { return true })
	cases := []struct {
		kind    domain.EvidenceRelatedKind
		id      int64
		visible bool
		want    bool
	}{
		{domain.RelatedClientPayment, 3, false, true},
		{domain.RelatedIssue, 3, false, true},
		{domain.RelatedGeneral, 0, true, true},
		{domain.RelatedGeneral, 0, false, false},
		{domain.RelatedProjectMilestone, 1, false, false},
		// A timeline attachment is visible only once staff flag it (ADR-0032).
		{domain.RelatedProjectMilestone, 1, true, true},
		// What the WO pays its suppliers is never shown, flagged or not.
		{domain.RelatedSupplierPayment, 7, false, false},
		{domain.RelatedSupplierPayment, 7, true, false},
		// The client's own payment to a supplier they pay is.
		{domain.RelatedSupplierPayment, 8, false, true},
		// A payment whose owner is gone (deleted) is not shown.
		{domain.RelatedSupplierPayment, 99, false, false},
	}
	for _, c := range cases {
		got := all.canSee(domain.Evidence{RelatedKind: c.kind, RelatedID: c.id, IsClientVisible: c.visible})
		if got != c.want {
			t.Errorf("kind=%s id=%d visible=%v: got %v, want %v", c.kind, c.id, c.visible, got, c.want)
		}
	}
	// Without portal.finance.view (Keluarga) payment receipts stay hidden,
	// while documents the WO shared remain visible.
	noFinance := clientView(func(p string) bool { return p != "portal.finance.view" })
	if noFinance.canSee(domain.Evidence{RelatedKind: domain.RelatedClientPayment}) {
		t.Error("payment receipt visible without portal.finance.view")
	}
	if noFinance.canSee(domain.Evidence{RelatedKind: domain.RelatedSupplierPayment, RelatedID: 8}) {
		t.Error("supplier payment receipt visible without portal.finance.view")
	}
	if !noFinance.canSee(domain.Evidence{RelatedKind: domain.RelatedGeneral, IsClientVisible: true}) {
		t.Error("shared document hidden without portal.finance.view")
	}
	// A role without portal.documents.view sees no shared documents, but its
	// payment, step and issue proof still follow their own permissions.
	noDocs := clientView(func(p string) bool { return p != "portal.documents.view" })
	if noDocs.canSee(domain.Evidence{RelatedKind: domain.RelatedGeneral, IsClientVisible: true}) {
		t.Error("shared document visible without portal.documents.view")
	}
	if !noDocs.canSee(domain.Evidence{RelatedKind: domain.RelatedClientPayment}) {
		t.Error("payment proof must not depend on portal.documents.view")
	}
}

func TestStaffWithoutMarginDoNotSeeWOPaymentFiles(t *testing.T) {
	var full *portalView
	planner := &portalView{staff: true, owners: testOwners}
	wo := domain.Evidence{RelatedKind: domain.RelatedSupplierPayment, RelatedID: 7}
	client := domain.Evidence{RelatedKind: domain.RelatedSupplierPayment, RelatedID: 8}
	if !full.canSee(wo) || !full.canSee(client) {
		t.Error("staff with finance.margin.view see every file")
	}
	if planner.canSee(wo) {
		t.Error("a Wedding Planner must not see the proof of what the WO pays a supplier")
	}
	if !planner.canSee(client) || !planner.canSee(domain.Evidence{RelatedKind: domain.RelatedVendorMilestone, RelatedID: 10}) {
		t.Error("a Wedding Planner still sees the client's supplier payments and the vendor steps")
	}
}

func TestRedactionRemovesInternalFields(t *testing.T) {
	p := projectResponse{PrepStartDate: "2026-01-01", PICSalesStaffID: 4, ContractValue: money(150_000_000)}
	redactProjectForClient(&p)
	if p.PrepStartDate != "" || p.PICSalesStaffID != 0 {
		t.Errorf("project internal fields not redacted: %+v", p)
	}
	if p.ContractValue == nil || *p.ContractValue != 150_000_000 {
		t.Error("the client's own contract value must stay visible")
	}

	due, booked := "2026-05-01", "2026-01-10"
	cost, paid, standing := int64(10), int64(5), 600
	v := supplierResponse{
		SupplierID: 9, Name: "Rias Ayu", AgreedCost: &cost, PaidWO: &paid, DueDate: &due, BookingDate: &booked, Notes: "nego",
		PICStaffID: 3, PICName: "Dewi", CapacityStanding: &standing, PaidByLocked: "Sudah ada pembayaran", UsedByEvents: 1,
		EngagementStatus: "Booked", Milestones: []vendorMilestoneResponse{{Name: "Fitting", Notes: "internal", PICStaffID: 3}},
	}
	redactSupplierForClient(&v, "Rias & busana")
	if v.SupplierID != 0 || v.AgreedCost != nil || v.PaidWO != nil || v.DueDate != nil || v.BookingDate != nil || v.Notes != "" ||
		v.PICStaffID != 0 || v.PICName != "" || v.CapacityStanding != nil || v.PaidByLocked != "" || v.UsedByEvents != 0 {
		t.Errorf("supplier internal fields not redacted: %+v", v)
	}
	if v.Name != "Rias & busana" {
		t.Errorf("a client sees the name the portal policy chose: %q", v.Name)
	}
	if v.Milestones[0].Notes != "" || v.Milestones[0].PICStaffID != 0 || v.Milestones[0].Name != "Fitting" || v.EngagementStatus != "Booked" {
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
		{ID: 1, RelatedKind: domain.RelatedSupplierPayment, RelatedID: 7, Type: domain.EvidenceInvoice, FileName: "a.pdf", UploadedAt: up(1)},
	}
	all := clientView(func(string) bool { return true })
	noFinance := clientView(func(p string) bool { return p != "portal.finance.view" })

	staff := indexEvidence(list, nil)
	if got := staff.refs(domain.RelatedProjectMilestone, 1); len(got) != 2 || got[0].ID != 3 || got[1].ID != 4 {
		t.Errorf("staff see every file of the step, oldest first: %+v", got)
	}

	client := indexEvidence(list, all)
	if got := client.refs(domain.RelatedProjectMilestone, 1); len(got) != 1 || got[0].ID != 4 {
		t.Errorf("a client sees only the flagged file: %+v", got)
	}
	if got := client.refs(domain.RelatedClientPayment, 7); len(got) != 1 || got[0].ID != 2 || got[0].FileName != "b.pdf" || got[0].UploadedAt == "" {
		t.Errorf("a client sees their payment proof: %+v", got)
	}
	// supplier payment 7 (the WO's) shares the numeric id with client payment 7: kinds never mix.
	if got := client.refs(domain.RelatedSupplierPayment, 7); len(got) != 0 {
		t.Errorf("the WO's supplier payment files are never shown to a client: %+v", got)
	}

	keluarga := indexEvidence(list, noFinance)
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
