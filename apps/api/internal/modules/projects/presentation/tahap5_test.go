package presentation

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"elproof/internal/modules/projects/application"
	"elproof/internal/modules/projects/domain"
)

func asMap(t *testing.T, v any) map[string]any {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("not an object: %s (%v)", b, err)
	}
	return m
}

// E8: the item serialises exactly as before and the extras sit next to its keys.
func TestWithExtrasAddsKeysNextToTheItem(t *testing.T) {
	type item struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
	}
	got := asMap(t, withExtras{item: item{ID: 3, Name: "x"}, extras: []extra{
		{"projectProgress", map[string]int{"overallPercent": 40}}, {"ledgerSummary", map[string]int64{"paid": 5}},
	}})
	if got["id"] != float64(3) || got["name"] != "x" {
		t.Errorf("the item's own keys must survive: %v", got)
	}
	if got["projectProgress"].(map[string]any)["overallPercent"] != float64(40) || got["ledgerSummary"].(map[string]any)["paid"] != float64(5) {
		t.Errorf("extras: %v", got)
	}
}

func TestWithExtrasWithoutAnItemIsJustTheExtras(t *testing.T) {
	// A DELETE that used to answer null now answers {"projectProgress": ...}.
	b, err := json.Marshal(withExtras{extras: []extra{{"projectProgress", map[string]int{"overallPercent": 1}}}})
	if err != nil || string(b) != `{"projectProgress":{"overallPercent":1}}` {
		t.Errorf("got %s %v", b, err)
	}
	// An item with no keys at all (an empty object) must not produce a stray comma.
	b, err = json.Marshal(withExtras{item: struct{}{}, extras: []extra{{"a", 1}, {"b", 2}}})
	if err != nil || string(b) != `{"a":1,"b":2}` {
		t.Errorf("got %s %v", b, err)
	}
	if b, _ := json.Marshal(withExtras{item: nil, extras: nil}); string(b) != `{}` {
		t.Errorf("nothing at all is {}: %s", b)
	}
	if _, err := json.Marshal(withExtras{item: []int{1}, extras: []extra{{"a", 1}}}); err == nil {
		t.Error("an item that is not an object cannot take extra keys")
	}
	if _, err := json.Marshal(withExtras{item: func() {}, extras: nil}); err == nil {
		t.Error("an unserialisable item is an error")
	}
}

func TestWithExtrasKeepsRealResponseShape(t *testing.T) {
	// A real response type (embedded struct with its own extra key) keeps every key.
	resp := paymentCreatedResponse{
		paymentResponse: toPaymentResponse(domain.VendorPayment{ID: 4, Type: domain.PaymentDP, Amount: 10, PaymentDate: time.Now()}, false),
		LedgerSummary:   ledgerSummaryResponse{ContractValue: 100, Paid: 10, Remaining: 90},
	}
	got := asMap(t, withExtras{item: resp, extras: []extra{{"projectProgress", progressResponse{OverallPercent: 7}}}})
	for _, k := range []string{"id", "type", "amount", "evidence", "ledgerSummary", "projectProgress"} {
		if _, ok := got[k]; !ok {
			t.Errorf("missing %q in %v", k, got)
		}
	}
	if got["projectProgress"].(map[string]any)["overallPercent"] != float64(7) {
		t.Errorf("progress: %v", got["projectProgress"])
	}
}

func TestDeletedResponseCarriesRFC3339RestorableUntil(t *testing.T) {
	now := time.Date(2026, 11, 10, 2, 0, 0, 0, time.UTC)
	got := asMap(t, toDeletedResponse(12, domain.RestorableUntil(now)))
	if got["id"] != float64(12) {
		t.Errorf("id: %v", got)
	}
	until, err := time.Parse(time.RFC3339, got["restorableUntil"].(string))
	if err != nil || !until.Equal(now.Add(24*time.Hour)) {
		t.Errorf("restorableUntil = %v (%v)", got["restorableUntil"], err)
	}
}

func overviewFixture() *application.ProjectOverview {
	next := domain.TermProgress{Term: domain.PaymentTerm{ID: 1, Label: "DP", Amount: 10, DueDate: time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)}, Remaining: 10, Status: domain.TermUpcoming}
	return &application.ProjectOverview{
		Project:  domain.Project{ID: 5, Name: "Akad", EventDate: time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC), PrepStartDate: time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC), ContractValue: 100},
		PICName:  "Sari",
		Finance:  domain.FinanceOverview{ContractValue: 100, Paid: 20, Remaining: 80, VendorCost: 30, VenueCost: 10, Margin: 60},
		NextTerm: &next, Terms: []domain.TermProgress{next},
		Team:    application.OverviewTeam{PICStaffID: 1, PICName: "Sari", PICSalesStaffID: 0, PICSalesName: ""},
		Clients: []application.ProjectClientInfo{{ID: 3, Name: "Rina", Role: "Bride", HasLogin: true}},
	}
}

func TestOverviewMarginOnlyWithThePermission(t *testing.T) {
	with := asMap(t, toOverviewResponse(overviewFixture(), true))["finance"].(map[string]any)
	if with["margin"] != float64(60) {
		t.Errorf("finance.margin.view sees the margin: %v", with)
	}
	without := asMap(t, toOverviewResponse(overviewFixture(), false))["finance"].(map[string]any)
	if _, present := without["margin"]; present {
		t.Errorf("without finance.margin.view the key must be absent, not zero or null: %v", without)
	}
	for _, k := range []string{"contractValue", "paid", "remaining", "overdueTotal", "nextTerm", "terms", "vendorCost", "venueCost"} {
		if _, ok := without[k]; !ok {
			t.Errorf("finance.%s is missing: %v", k, without)
		}
	}
}

func TestOverviewShape(t *testing.T) {
	got := asMap(t, toOverviewResponse(overviewFixture(), false))
	for _, k := range []string{"project", "finance", "actions", "nextMilestones", "unconfirmedVendors", "openIssueCount", "team", "clients"} {
		if _, ok := got[k]; !ok {
			t.Errorf("missing %q", k)
		}
	}
	project := got["project"].(map[string]any)
	if project["picName"] != "Sari" || project["progress"] == nil {
		t.Errorf("project is what GET /projects/{id} answers (picName, progress): %v", project)
	}
	for _, k := range []string{"actions", "nextMilestones", "unconfirmedVendors"} {
		if arr, ok := got[k].([]any); !ok || len(arr) != 0 {
			t.Errorf("%s must be an empty array, not null: %v", k, got[k])
		}
	}
	team := got["team"].(map[string]any)
	if team["picName"] != "Sari" || team["picSalesName"] != "" || team["picSalesStaffId"] != float64(0) {
		t.Errorf("team: %v", team)
	}
	client := got["clients"].([]any)[0].(map[string]any)
	if client["name"] != "Rina" || client["role"] != "Bride" || client["hasLogin"] != true {
		t.Errorf("clients: %v", client)
	}
	next := got["finance"].(map[string]any)["nextTerm"].(map[string]any)
	if next["id"] != float64(1) || next["label"] != "DP" || next["dueDate"] != "2026-12-01" || next["status"] != "upcoming" || next["remaining"] != float64(10) || next["amount"] != float64(10) {
		t.Errorf("nextTerm: %v", next)
	}
	// No clients/terms: arrays, and no nextTerm is null.
	o := overviewFixture()
	o.NextTerm, o.Terms, o.Clients = nil, nil, nil
	got = asMap(t, toOverviewResponse(o, false))
	fin := got["finance"].(map[string]any)
	if fin["nextTerm"] != nil || len(fin["terms"].([]any)) != 0 || len(got["clients"].([]any)) != 0 {
		t.Errorf("empty overview: %v", got)
	}
}

func TestOverviewIsNotReadableByAClient(t *testing.T) {
	if clientCanRead([]string{"overview"}) {
		t.Error("the staff overview carries margin and vendor costs: a client principal must be refused")
	}
}

func TestIssueResponseWithoutAVendorIsNullAndCarriesVisibility(t *testing.T) {
	none := asMap(t, toIssueResponse(domain.VendorIssue{ID: 1, IsClientVisible: false, FoundDate: time.Now()}))
	if v, present := none["projectVendorId"]; !present || v != nil {
		t.Errorf("projectVendorId must be null when there is no vendor: %v", none)
	}
	if none["isClientVisible"] != false || none["vendorName"] != "" || none["vendorMilestoneName"] != "" || none["picName"] != "" {
		t.Errorf("shape: %v", none)
	}
	some := asMap(t, toIssueResponse(domain.VendorIssue{ID: 2, ProjectVendorID: 7, IsClientVisible: true, FoundDate: time.Now()}))
	if some["projectVendorId"] != float64(7) || some["isClientVisible"] != true {
		t.Errorf("with a vendor: %v", some)
	}
}

func TestDashboardMoneySectionsAreAbsentWithoutRevenuePermission(t *testing.T) {
	stats := domain.DashboardStats{
		MoneySections:      true,
		CashFlow:           []domain.CashFlowWeek{{WeekStart: time.Date(2026, 11, 9, 0, 0, 0, 0, time.UTC), ExpectedIn: 5, ExpectedOut: 3}},
		VendorPaymentsDue:  []domain.VendorDue{{ProjectID: 1, ProjectName: "P", VendorID: 4, DueDate: time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC), Remaining: 9}},
		UnconfirmedVendors: []domain.UnconfirmedVendorRow{{ProjectID: 1, ProjectName: "P", VendorID: 4, EngagementStatus: domain.EngagementPlanned, EventDate: time.Date(2026, 11, 30, 0, 0, 0, 0, time.UTC)}},
	}
	names := map[int64]string{4: "Bunga"}
	with := asMap(t, toDashboardResponse(stats, names, nil))
	week := with["cashFlow"].([]any)[0].(map[string]any)
	if week["weekStart"] != "2026-11-09" || week["expectedIn"] != float64(5) || week["expectedOut"] != float64(3) {
		t.Errorf("cashFlow: %v", week)
	}
	due := with["vendorPaymentsDue"].([]any)[0].(map[string]any)
	if due["vendorName"] != "Bunga" || due["dueDate"] != "2026-11-20" || due["remaining"] != float64(9) || due["projectName"] != "P" {
		t.Errorf("vendorPaymentsDue: %v", due)
	}
	un := with["unconfirmedVendors"].([]any)[0].(map[string]any)
	if un["vendorName"] != "Bunga" || un["engagementStatus"] != "Planned" || un["eventDate"] != "2026-11-30" {
		t.Errorf("unconfirmedVendors: %v", un)
	}
	// Money allowed but nothing to show: the keys exist with [].
	empty := asMap(t, toDashboardResponse(domain.DashboardStats{MoneySections: true}, nil, nil))
	if a, ok := empty["cashFlow"].([]any); !ok || len(a) != 0 {
		t.Errorf("cashFlow with permission but no data is []: %v", empty["cashFlow"])
	}
	// No revenue permission: the service never computed them; the keys are not there.
	denied := asMap(t, toDashboardResponse(domain.DashboardStats{UnconfirmedVendors: stats.UnconfirmedVendors}, names, nil))
	for _, k := range []string{"cashFlow", "vendorPaymentsDue"} {
		if _, present := denied[k]; present {
			t.Errorf("%s must be absent without dashboard.revenue.view", k)
		}
	}
	if len(denied["unconfirmedVendors"].([]any)) != 1 {
		t.Error("unconfirmedVendors is not money: always present")
	}
}

func TestDashboardRowsCarryNamesNotJustIds(t *testing.T) {
	stats := domain.DashboardStats{
		OpenIssues:     []domain.DashboardIssueRow{{Issue: domain.VendorIssue{ID: 1}, ProjectID: 2, VendorID: 4}, {Issue: domain.VendorIssue{ID: 2}, ProjectID: 2}},
		RecentActivity: []domain.ActivityLogEntry{{ID: 1, ActorStaffID: 9}, {ID: 2, ActorStaffID: 0}, {ID: 3, ActorStaffID: 77}},
	}
	got := asMap(t, toDashboardResponse(stats, map[int64]string{4: "Bunga"}, map[int64]string{9: "Sari"}))
	issues := got["openIssues"].([]any)
	if issues[0].(map[string]any)["vendorName"] != "Bunga" || issues[1].(map[string]any)["vendorName"] != "" {
		t.Errorf("issues: %v", issues)
	}
	acts := got["recentActivity"].([]any)
	if acts[0].(map[string]any)["actorName"] != "Sari" || acts[1].(map[string]any)["actorName"] != "" || acts[2].(map[string]any)["actorName"] != "" {
		t.Errorf("an unresolved actor has an empty name, never a made-up one: %v", acts)
	}
	if strings.Contains(string(mustJSON(got)), "Sistem") {
		t.Error("the string \"Sistem\" is never invented")
	}
}

func mustJSON(v any) []byte { b, _ := json.Marshal(v); return b }

func TestActivityResponseKeepsExistingFieldsAndAddsActorName(t *testing.T) {
	pid := int64(5)
	got := asMap(t, toActivityResponse(domain.ActivityLogEntry{ID: 1, ProjectID: &pid, Type: domain.ActivityPaymentDeleted, ActorStaffID: 9, EntityType: "client_payment", EntityID: "3", EntityLabel: "DP", Description: "d", CreatedAt: time.Now()}))
	for _, k := range []string{"id", "type", "actorStaffId", "projectId", "entityType", "entityId", "entityLabel", "description", "timestamp", "actorName"} {
		if _, ok := got[k]; !ok {
			t.Errorf("missing %q: %v", k, got)
		}
	}
}

func TestActivityDefaultsToTwentyPerPage(t *testing.T) {
	if activityDefaultLimit != 20 {
		t.Errorf("default page size is 20, got %d", activityDefaultLimit)
	}
}
