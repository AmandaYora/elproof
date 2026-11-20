package domain

import (
	"testing"
	"time"
)

func allBut(denied ...string) PortalPolicy {
	return NewPortalPolicy(func(p string) bool {
		for _, d := range denied {
			if p == d {
				return false
			}
		}
		return true
	})
}

func TestSupplierNameFollowsWhoPays(t *testing.T) {
	p := allBut()
	cases := []struct {
		name string
		s    ProjectSupplier
		cat  string
		want string
	}{
		{"wo-paid vendor, hidden", ProjectSupplier{Kind: SupplierVendor, Name: "Dapur Ibu", PaidBy: PayerWO}, "Katering", "Katering"},
		{"wo-paid vendor, no category", ProjectSupplier{Kind: SupplierVendor, Name: "Dapur Ibu", PaidBy: PayerWO}, "", "Layanan"},
		{"wo-paid vendor, shown", ProjectSupplier{Kind: SupplierVendor, Name: "Dapur Ibu", PaidBy: PayerWO, ShowName: true}, "Katering", "Dapur Ibu"},
		{"client-paid vendor", ProjectSupplier{Kind: SupplierVendor, Name: "Dapur Ibu", PaidBy: PayerClient}, "Katering", "Dapur Ibu"},
		{"venue paid by WO", ProjectSupplier{Kind: SupplierVenue, Name: "Gedung Melati", PaidBy: PayerWO}, "", "Gedung Melati"},
	}
	for _, c := range cases {
		if got := p.SupplierName(c.s, c.cat); got != c.want {
			t.Errorf("%s: %q, want %q", c.name, got, c.want)
		}
	}
}

func TestSupplierPaymentOnlyClientPaidWithFinance(t *testing.T) {
	if allBut().SupplierPayment(SupplierPayment{Payer: PayerWO}) {
		t.Error("what the WO pays a supplier is never shown")
	}
	if !allBut().SupplierPayment(SupplierPayment{Payer: PayerClient}) {
		t.Error("the client's own payment to a supplier is shown with portal.finance.view")
	}
	if allBut(PermPortalFinance).SupplierPayment(SupplierPayment{Payer: PayerClient}) {
		t.Error("without portal.finance.view no payment at all")
	}
}

func TestEvidenceRules(t *testing.T) {
	all := allBut()
	wo := EvidenceOwner{Known: true, PaidBy: PayerWO, Payer: PayerWO}
	client := EvidenceOwner{Known: true, PaidBy: PayerClient, Payer: PayerClient}
	step := Evidence{RelatedKind: RelatedVendorMilestone}
	flagged := Evidence{RelatedKind: RelatedVendorMilestone, IsClientVisible: true}
	cases := []struct {
		name  string
		p     PortalPolicy
		e     Evidence
		owner EvidenceOwner
		want  bool
	}{
		{"step of a wo-paid supplier, not flagged", all, step, wo, false},
		{"step of a wo-paid supplier, flagged", all, flagged, wo, true},
		{"step of a client-paid supplier", all, step, client, true},
		{"step without portal.vendor.view", allBut(PermPortalVendor), flagged, client, false},
		{"step of a deleted supplier", all, flagged, EvidenceOwner{}, false},
		{"wo payment proof", all, Evidence{RelatedKind: RelatedSupplierPayment}, wo, false},
		{"client payment to supplier", all, Evidence{RelatedKind: RelatedSupplierPayment}, client, true},
		{"client payment to supplier, no finance", allBut(PermPortalFinance), Evidence{RelatedKind: RelatedSupplierPayment}, client, false},
		{"supplier papers", all, Evidence{RelatedKind: RelatedProjectSupplier, IsClientVisible: true}, client, false},
		{"client payment proof", all, Evidence{RelatedKind: RelatedClientPayment}, EvidenceOwner{}, true},
		{"client payment proof, no finance", allBut(PermPortalFinance), Evidence{RelatedKind: RelatedClientPayment}, EvidenceOwner{}, false},
		{"document opted in", all, Evidence{RelatedKind: RelatedGeneral, IsClientVisible: true}, EvidenceOwner{}, true},
		{"document not opted in", all, Evidence{RelatedKind: RelatedGeneral}, EvidenceOwner{}, false},
		{"file of a hidden kendala", all, Evidence{RelatedKind: RelatedIssue, ClientHidden: true}, EvidenceOwner{}, false},
	}
	for _, c := range cases {
		if got := c.p.Evidence(c.e, c.owner); got != c.want {
			t.Errorf("%s: %v, want %v", c.name, got, c.want)
		}
	}
}

func TestIssueNeedsShareAndPermission(t *testing.T) {
	if !allBut().Issue(true) || allBut().Issue(false) || allBut(PermPortalIssues).Issue(true) {
		t.Error("a kendala is shown only when shared and with portal.issues.view")
	}
}

func TestNilPermissionCheckShowsNothing(t *testing.T) {
	p := NewPortalPolicy(nil)
	if p.Events() || p.Suppliers() || p.Finance() {
		t.Error("a policy without permissions shows nothing")
	}
}

func TestLocationAndEventsLine(t *testing.T) {
	d := func(s string) time.Time { t, _ := time.Parse("2006-01-02", s); return t }
	v := func(n int64) *int64 { return &n }
	events := []ProjectEvent{
		{ID: 2, KindName: "Resepsi", EventDate: d("2027-03-14"), VenueSupplierID: v(8)},
		{ID: 1, KindName: "Akad", EventDate: d("2027-03-13"), VenueSupplierID: v(7)},
	}
	names := map[int64]string{7: "Masjid Al-Ikhlas", 8: "Gedung Melati"}
	if got := LocationLine(events, names); got != "Akad: Masjid Al-Ikhlas · Resepsi: Gedung Melati" {
		t.Errorf("two venues: %q", got)
	}
	events[1].VenueSupplierID = v(8)
	if got := LocationLine(events, names); got != "Gedung Melati" {
		t.Errorf("one venue for every event: %q", got)
	}
	if got := LocationLine([]ProjectEvent{{KindName: "Akad", EventDate: d("2027-03-13")}}, names); got != "" {
		t.Errorf("no venue: %q", got)
	}
	if got := EventsLine(events); got != "2 acara: Akad 13 Mar · Resepsi 14 Mar" {
		t.Errorf("events line: %q", got)
	}
	if got := EventsLine(events[:1]); got != "" {
		t.Errorf("one event has no second line: %q", got)
	}
}

func TestMainEventAndGuests(t *testing.T) {
	events := []ProjectEvent{{ID: 1}, {ID: 2, IsMain: true, Sessions: []EventSession{{Guests: 300}, {Guests: 250}}}}
	m := MainEvent(events)
	if m == nil || m.ID != 2 || m.TotalGuests() != 550 {
		t.Errorf("main event: %+v", m)
	}
	if MainEvent(nil) != nil {
		t.Error("no events, no main event")
	}
}

func TestPaidByLockMessage(t *testing.T) {
	if got := PaidByLockMessage(PayerWO, 0, 0); got != "" {
		t.Errorf("nothing attached: %q", got)
	}
	if got := PaidByLockMessage(PayerClient, 2, 3); got != "Klien langsung. Terkunci karena sudah ada 2 pembayaran. Hapus pembayaran itu untuk mengubah." {
		t.Errorf("payments: %q", got)
	}
	if got := PaidByLockMessage(PayerWO, 0, 1); got != "WO. Terkunci karena sudah ada 1 bukti. Hapus bukti itu untuk mengubah." {
		t.Errorf("files: %q", got)
	}
}
