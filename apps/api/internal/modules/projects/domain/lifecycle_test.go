package domain

import "testing"

func TestNextStatus(t *testing.T) {
	cases := []struct {
		from   ProjectStatus
		action StatusAction
		want   ProjectStatus
		ok     bool
	}{
		{StatusDraft, ActionStartPreparation, StatusPreparation, true},
		{StatusPreparation, ActionMarkReady, StatusReady, true},
		{StatusReady, ActionComplete, StatusCompleted, true},
		{StatusCompleted, ActionReopen, StatusPreparation, true},
		{StatusCancelled, ActionReopen, StatusPreparation, true},
		// Steps cannot be skipped or repeated.
		{StatusDraft, ActionMarkReady, "", false},
		{StatusDraft, ActionComplete, "", false},
		{StatusPreparation, ActionComplete, "", false},
		{StatusReady, ActionStartPreparation, "", false},
		{StatusCompleted, ActionComplete, "", false},
		// An open project is not "reopened".
		{StatusPreparation, ActionReopen, "", false},
		{StatusDraft, "nonsense", "", false},
	}
	for _, c := range cases {
		got, ok := NextStatus(c.from, c.action)
		if got != c.want || ok != c.ok {
			t.Errorf("%s + %s = (%q,%v), want (%q,%v)", c.from, c.action, got, ok, c.want, c.ok)
		}
	}
}

func TestClosingChecklist(t *testing.T) {
	paid := []PaymentLine{{PaymentDP, 50}, {PaymentPelunasan, 50}}
	all := BuildClosingChecklist(ClosingInput{
		ContractValue: 100, ClientPayments: paid,
		Vendors:   []VendorBalance{{ContractValue: 40, Payments: []PaymentLine{{PaymentPelunasan, 40}}}},
		VenueCost: 30, VenuePayments: []PaymentLine{{PaymentPelunasan, 30}},
	})
	for _, it := range all {
		if !it.Done {
			t.Errorf("%s should be done, detail %q", it.Key, it.Detail)
		}
	}

	open := BuildClosingChecklist(ClosingInput{
		ContractValue: 100, ClientPayments: []PaymentLine{{PaymentDP, 30}, {PaymentTambahan, 500}},
		Vendors:    []VendorBalance{{ContractValue: 40}, {ContractValue: 10, Payments: []PaymentLine{{PaymentPelunasan, 10}}}},
		VenueCost:  0,
		OpenIssues: 2,
	})
	byKey := map[string]ChecklistItem{}
	for _, it := range open {
		byKey[it.Key] = it
	}
	if byKey["client_paid"].Done || byKey["client_paid"].Detail != "Masih ada sisa Rp70" {
		t.Errorf("client: %+v (Tambahan must not count towards the contract)", byKey["client_paid"])
	}
	if byKey["vendors_paid"].Done || byKey["vendors_paid"].Detail != "1 vendor belum lunas, total sisa Rp40" {
		t.Errorf("vendors: %+v", byKey["vendors_paid"])
	}
	if _, has := byKey["venue_paid"]; has {
		t.Errorf("no venue cost: the venue line must not appear")
	}
	if byKey["issues_closed"].Done || byKey["issues_closed"].Detail != "2 kendala masih terbuka" {
		t.Errorf("issues: %+v", byKey["issues_closed"])
	}
}
