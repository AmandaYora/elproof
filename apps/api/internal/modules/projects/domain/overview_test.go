package domain

import "testing"

func ptr(n int64) *int64 { return &n }

func TestProjectActionsFollowTheRulesTheWebHadInTheBrowser(t *testing.T) {
	today := d("2026-11-10")
	in := ActionInput{
		Today: today, EventDate: d("2026-11-25"), // H-15: inside the 30-day vendor window
		Terms: []TermProgress{
			{Term: PaymentTerm{ID: 7, Label: "DP", DueDate: d("2026-11-01")}, Remaining: 2_500_000, Status: TermOverdue},
			{Term: PaymentTerm{ID: 8, Label: "Pelunasan", DueDate: d("2026-12-01")}, Remaining: 1, Status: TermUpcoming},
		},
		Issues: []VendorIssue{
			{ID: 1, Title: "Tenda bocor", Impact: ImpactMedium, Status: IssueOpen},
			{ID: 2, Title: "Genset mati", Impact: ImpactCritical, Status: IssueInReview},
			{ID: 3, Title: "Sudah beres", Impact: ImpactCritical, Status: IssueResolved},
		},
		Engagement: []ProjectVendor{
			{ID: 10, EngagementStatus: EngagementPlanned},
			{ID: 11, EngagementStatus: EngagementNegotiation},
			{ID: 12, EngagementStatus: EngagementBooked},
		},
		VendorName: map[int64]string{10: "Bunga Indah"},
		Milestones: []ProjectMilestone{
			{ID: 20, Name: "Fitting", Status: MilestoneNotStarted, TargetDate: d("2026-11-09")},
			{ID: 21, Name: "Undangan", Status: MilestoneCompleted, TargetDate: d("2026-11-01")},
			{ID: 22, Name: "Batal", Status: MilestoneCancelled, TargetDate: d("2026-11-01")},
			{ID: 23, Name: "Hari ini", Status: MilestoneInProgress, TargetDate: today},
		},
	}
	got := BuildProjectActions(in)
	var ids []string
	for _, a := range got {
		ids = append(ids, a.ID)
	}
	want := []string{"term-7", "issue-2", "issue-1", "vendor-10", "vendor-11", "ms-20"}
	// danger first (the overdue term, the critical kendala), then warnings in
	// terms/kendala/vendor/timeline order.
	wantOrder := []string{"term-7", "issue-2", "issue-1", "vendor-10", "vendor-11", "ms-20"}
	if len(ids) != len(wantOrder) {
		t.Fatalf("got %v, want %v", ids, want)
	}
	for i := range wantOrder {
		if ids[i] != wantOrder[i] {
			t.Fatalf("got %v, want %v", ids, wantOrder)
		}
	}
	byID := map[string]ProjectAction{}
	for _, a := range got {
		byID[a.ID] = a
	}
	if a := byID["term-7"]; a.Tone != "danger" || a.Kind != ActionTermOverdue || a.Tab != "pembayaran/client" ||
		a.Title != "Termin DP terlambat" || a.Detail != "Sisa Rp2.500.000, jatuh tempo 1 Nov 2026" {
		t.Errorf("term action: %+v", a)
	}
	if a := byID["issue-1"]; a.Tone != "warning" || a.Kind != ActionIssueOpen || a.Tab != "kendala" || a.Detail != "Kendala masih terbuka" {
		t.Errorf("a Medium kendala is a warning: %+v", a)
	}
	if byID["issue-2"].Tone != "danger" {
		t.Error("a Critical kendala is danger")
	}
	if a := byID["vendor-10"]; a.Title != "Bunga Indah belum terkonfirmasi" || a.Detail != "Acara H-15, status masih belum dipesan" || a.Tab != "vendor" || a.Kind != ActionVendorUnconfirm {
		t.Errorf("vendor action: %+v", a)
	}
	if byID["vendor-11"].Title != "Vendor belum terkonfirmasi" {
		t.Errorf("an unresolved vendor name falls back to \"Vendor\": %q", byID["vendor-11"].Title)
	}
	if a := byID["ms-20"]; a.Title != `Timeline "Fitting" lewat tenggat` || a.Detail != "Target 9 Nov 2026" || a.Tab != "milestone" || a.Kind != ActionMilestoneLate {
		t.Errorf("milestone action: %+v", a)
	}
}

func TestVendorActionOnlyWithinThirtyDaysOfTheEvent(t *testing.T) {
	engagement := []ProjectVendor{{ID: 1, EngagementStatus: EngagementPlanned}}
	for _, c := range []struct {
		event string
		want  int
	}{{"2026-12-10", 1}, {"2026-12-11", 0}, {"2026-11-10", 1}, {"2026-11-09", 0}} {
		got := BuildProjectActions(ActionInput{Today: d("2026-11-10"), EventDate: d(c.event), Engagement: engagement})
		if len(got) != c.want {
			t.Errorf("event %s: %d actions, want %d", c.event, len(got), c.want)
		}
	}
}

func TestNoActionsWhenAllIsWell(t *testing.T) {
	if got := BuildProjectActions(ActionInput{Today: d("2026-11-10"), EventDate: d("2027-06-01")}); len(got) != 0 {
		t.Errorf("got %+v", got)
	}
}

func TestNextMilestonesAreTheFiveSoonestOpenOnes(t *testing.T) {
	list := []ProjectMilestone{
		{ID: 1, TargetDate: d("2026-12-05"), Status: MilestoneNotStarted},
		{ID: 2, TargetDate: d("2026-11-01"), Status: MilestoneCompleted},
		{ID: 3, TargetDate: d("2026-11-02"), Status: MilestoneCancelled},
		{ID: 4, TargetDate: d("2026-11-20"), Status: MilestoneInProgress},
		{ID: 5, TargetDate: d("2026-11-20"), Status: MilestoneBlocked},
		{ID: 6, TargetDate: d("2026-12-30"), Status: MilestoneNotStarted},
		{ID: 7, TargetDate: d("2026-11-15"), Status: MilestoneNotStarted},
		{ID: 8, TargetDate: d("2027-01-30"), Status: MilestoneNotStarted},
		{ID: 9, TargetDate: d("2027-02-28"), Status: MilestoneNotStarted},
	}
	got := NextMilestones(list)
	var ids []int64
	for _, m := range got {
		ids = append(ids, m.ID)
	}
	want := []int64{7, 4, 5, 1, 6} // ties keep timeline order (4 before 5); 8 and 9 are the sixth and seventh open
	if len(ids) != len(want) {
		t.Fatalf("got %v, want %v", ids, want)
	}
	for i := range want {
		if ids[i] != want[i] {
			t.Fatalf("got %v, want %v", ids, want)
		}
	}
}

func TestFinanceOverview(t *testing.T) {
	lines := []PaymentLine{{PaymentDP, 30}, {PaymentTermin, 20}, {PaymentTambahan, 999}, {PaymentRefund, 10}}
	terms := []TermProgress{
		{Status: TermOverdue, Remaining: 15}, {Status: TermOverdue, Remaining: 5}, {Status: TermUpcoming, Remaining: 40}, {Status: TermPaid},
	}
	engagements := []ProjectVendor{
		{ContractValue: 25, EngagementStatus: EngagementBooked},
		{ContractValue: 1000, EngagementStatus: EngagementCancelled},
		{ContractValue: 15, EngagementStatus: EngagementPlanned},
	}
	f := ComputeFinanceOverview(100, lines, terms, engagements, ptr(12), ptr(3))
	if f.Paid != 40 || f.Remaining != 60 {
		t.Errorf("paid/remaining = %d/%d, want 40/60 (Tambahan outside the contract, Refund subtracts)", f.Paid, f.Remaining)
	}
	if f.OverdueTotal != 20 {
		t.Errorf("overdue total = %d", f.OverdueTotal)
	}
	if f.VendorCost != 40 || f.VenueCost != 15 {
		t.Errorf("costs = %d/%d, want 40/15 (a cancelled engagement costs nothing)", f.VendorCost, f.VenueCost)
	}
	if f.Margin != 100-40-15 {
		t.Errorf("margin = %d", f.Margin)
	}
	// Overpaid: remaining never negative. No venue: cost 0. Costs above the contract: negative margin.
	f = ComputeFinanceOverview(10, []PaymentLine{{PaymentDP, 50}}, nil, []ProjectVendor{{ContractValue: 30, EngagementStatus: EngagementBooked}}, nil, nil)
	if f.Remaining != 0 || f.Paid != 50 || f.VenueCost != 0 || f.Margin != -20 {
		t.Errorf("edge: %+v", f)
	}
}

func TestOpenIssueCountAndUnconfirmed(t *testing.T) {
	if n := OpenIssueCount([]VendorIssue{{Status: IssueOpen}, {Status: IssueResolved}, {Status: IssueInResolution}, {Status: IssueClosed}}); n != 2 {
		t.Errorf("open = %d", n)
	}
	got := UnconfirmedEngagements([]ProjectVendor{{ID: 1, EngagementStatus: EngagementBooked}, {ID: 2, EngagementStatus: EngagementNegotiation}, {ID: 3, EngagementStatus: EngagementPlanned}})
	if len(got) != 2 || got[0].ID != 2 || got[1].ID != 3 {
		t.Errorf("unconfirmed: %+v", got)
	}
}
