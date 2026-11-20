package domain

import (
	"errors"
	"testing"
	"time"
)

func day(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return t
}

// at is a WIB wall-clock instant.
func at(s string) time.Time {
	t, err := time.ParseInLocation("2006-01-02 15:04", s, WIB)
	if err != nil {
		panic(err)
	}
	return t
}

func TestQuotationNumberFormat(t *testing.T) {
	cases := []struct {
		code string
		seq  int
		want string
	}{
		{"ABC", 1, "ABC-PNW-261120-001"},
		{"ABC", 42, "ABC-PNW-261120-042"},
		{"WOMAN", 999, "WOMAN-PNW-261120-999"},
		// A code outside 2–5 capital letters or digits falls back to WO.
		{"abc", 1, "WO-PNW-261120-001"},
		{"", 7, "WO-PNW-261120-007"},
		{"TOOLONG", 3, "WO-PNW-261120-003"},
	}
	for _, c := range cases {
		got, err := QuotationNumber(c.code, day("2026-11-20"), c.seq)
		if err != nil || got != c.want {
			t.Errorf("QuotationNumber(%q, %d) = %q, %v; want %q", c.code, c.seq, got, err, c.want)
		}
	}
	for _, seq := range []int{0, 1000} {
		if _, err := QuotationNumber("ABC", day("2026-11-20"), seq); !errors.Is(err, ErrDailyQuotaReached) {
			t.Errorf("seq %d: err = %v, want ErrDailyQuotaReached", seq, err)
		}
	}
}

func TestTodayIsAWIBDay(t *testing.T) {
	// 18:30 UTC on the 19th is 01:30 WIB on the 20th: the number date and
	// "today" follow WIB.
	now := time.Date(2026, 11, 19, 18, 30, 0, 0, time.UTC)
	if got := TodayWIB(now); !got.Equal(day("2026-11-20")) {
		t.Fatalf("TodayWIB = %v, want 2026-11-20", got)
	}
	n, _ := QuotationNumber("ABC", TodayWIB(now), 1)
	if n != "ABC-PNW-261120-001" {
		t.Fatalf("number = %q", n)
	}
}

func TestDefaultValidityIsFourteenDays(t *testing.T) {
	now := at("2026-11-20 09:00")
	if got := DefaultValidUntil(now, 14); !got.Equal(day("2026-12-04")) {
		t.Fatalf("DefaultValidUntil(14) = %v, want 2026-12-04", got)
	}
	// An unset workspace default still gives 14 days.
	if got := DefaultValidUntil(now, 0); !got.Equal(day("2026-12-04")) {
		t.Fatalf("DefaultValidUntil(0) = %v, want 2026-12-04", got)
	}
	if got := DefaultValidUntil(now, 30); !got.Equal(day("2026-12-20")) {
		t.Fatalf("DefaultValidUntil(30) = %v, want 2026-12-20", got)
	}
}

func TestExpiredIsADraftPastItsLastDay(t *testing.T) {
	q := Quotation{Status: QuotationDraft, ValidUntil: day("2026-11-20")}
	if q.Expired(at("2026-11-20 23:59")) {
		t.Fatal("valid through the end of its last WIB day")
	}
	if got := q.DaysLeft(at("2026-11-20 08:00")); got != 0 {
		t.Fatalf("DaysLeft on the last day = %d, want 0", got)
	}
	next := at("2026-11-21 00:01")
	if !q.Expired(next) || q.DisplayStatus(next) != QuotationExpired || q.DaysLeft(next) != -1 {
		t.Fatalf("next day: expired=%v display=%s daysLeft=%d", q.Expired(next), q.DisplayStatus(next), q.DaysLeft(next))
	}
	q.Status = QuotationAccepted
	if q.Expired(next) || q.DisplayStatus(next) != QuotationAccepted {
		t.Fatal("only a Draft expires")
	}
}

func TestPlanTransition(t *testing.T) {
	now := at("2026-11-20 10:00")
	valid := day("2026-11-30")
	past := day("2026-11-10")
	pid := int64(9)
	draft := Quotation{Status: QuotationDraft, ValidUntil: valid}
	expired := Quotation{Status: QuotationDraft, ValidUntil: past}
	accepted := Quotation{Status: QuotationAccepted, ValidUntil: valid}
	converted := Quotation{Status: QuotationAccepted, ValidUntil: valid, ConvertedProjectID: &pid}
	rejected := Quotation{Status: QuotationRejected, ValidUntil: valid}
	superseded := Quotation{Status: QuotationSuperseded, ValidUntil: valid}
	matches := AcceptCheck{RevisionMatchesWorkingCopy: true}
	extendTo := day("2026-12-05")
	yesterday := day("2026-11-19")

	cases := []struct {
		name   string
		q      Quotation
		target QuotationStatus
		check  AcceptCheck
		want   Transition
		code   string
	}{
		{"accept a draft", draft, QuotationAccepted, matches, TransitionAccept, ""},
		{"accept needs a revision equal to the working copy", draft, QuotationAccepted, AcceptCheck{}, TransitionNone, CodeNeedsRevision},
		{"accept an expired draft needs an extension", expired, QuotationAccepted, matches, TransitionNone, CodeNeedsExtension},
		{"accept an expired draft with an extension", expired, QuotationAccepted, AcceptCheck{RevisionMatchesWorkingCopy: true, ExtendUntil: &extendTo}, TransitionAccept, ""},
		{"an extension in the past is refused", expired, QuotationAccepted, AcceptCheck{RevisionMatchesWorkingCopy: true, ExtendUntil: &yesterday}, TransitionNone, CodeNeedsExtension},
		{"reject a draft", draft, QuotationRejected, AcceptCheck{}, TransitionReject, ""},
		{"reject an expired draft", expired, QuotationRejected, AcceptCheck{}, TransitionReject, ""},
		{"reopen a rejected one", rejected, QuotationDraft, AcceptCheck{}, TransitionReopen, ""},
		{"unaccept before conversion", accepted, QuotationDraft, AcceptCheck{}, TransitionUnaccept, ""},
		{"no unaccept after conversion", converted, QuotationDraft, AcceptCheck{}, TransitionNone, CodeConverted},
		{"nothing changes after conversion", converted, QuotationRejected, AcceptCheck{}, TransitionNone, CodeConverted},
		{"accepted cannot be rejected", accepted, QuotationRejected, AcceptCheck{}, TransitionNone, CodeNotDraft},
		{"rejected cannot be accepted", rejected, QuotationAccepted, matches, TransitionNone, CodeNotDraft},
		{"superseded is final", superseded, QuotationDraft, AcceptCheck{}, TransitionNone, CodeSuperseded},
		{"superseded is never asked for", draft, QuotationSuperseded, AcceptCheck{}, TransitionNone, CodeAutomaticOnly},
		{"unknown status", draft, QuotationStatus("Lost"), AcceptCheck{}, TransitionNone, CodeInvalidTransition},
		{"expired is not a stored status", draft, QuotationExpired, AcceptCheck{}, TransitionNone, CodeInvalidTransition},
		{"same status is a no-op", accepted, QuotationAccepted, AcceptCheck{}, TransitionNone, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := PlanTransition(c.q, c.target, now, c.check)
			if got != c.want {
				t.Fatalf("transition = %q, want %q (err %v)", got, c.want, err)
			}
			var te *TransitionError
			switch {
			case c.code == "" && err != nil:
				t.Fatalf("unexpected error %v", err)
			case c.code != "" && (!errors.As(err, &te) || te.Code != c.code):
				t.Fatalf("err = %v, want code %s", err, c.code)
			}
		})
	}
}

func TestCopyAndAlternatives(t *testing.T) {
	pid := int64(3)
	if !CopySupersedesOriginal(Quotation{Status: QuotationDraft}) {
		t.Error("a copied Draft is superseded by its copy")
	}
	if !CopySupersedesOriginal(Quotation{Status: QuotationAccepted}) {
		t.Error("a copied Accepted quotation is superseded by its copy")
	}
	if CopySupersedesOriginal(Quotation{Status: QuotationRejected}) {
		t.Error("a Rejected original stays Rejected")
	}
	if err := CanCopy(Quotation{Status: QuotationSuperseded}); err == nil {
		t.Error("a superseded quotation cannot be copied")
	}
	if err := CanCopy(Quotation{Status: QuotationAccepted, ConvertedProjectID: &pid}); err == nil {
		t.Error("a converted quotation cannot be copied")
	}
	if err := CanAddAlternative(Quotation{Status: QuotationDraft}); err != nil {
		t.Errorf("a Draft takes alternatives: %v", err)
	}
	if err := CanAddAlternative(Quotation{Status: QuotationRejected}); err == nil {
		t.Error("only a Draft takes alternatives")
	}

	root := int64(1)
	group := []Quotation{
		{ID: 1, Status: QuotationDraft},
		{ID: 2, Status: QuotationDraft, AlternativeOfID: &root, ValidUntil: day("2026-01-01")}, // expired
		{ID: 3, Status: QuotationRejected, AlternativeOfID: &root},
		{ID: 4, Status: QuotationDraft, AlternativeOfID: &root},
	}
	got := SupersededByAcceptance(group[3], group)
	if len(got) != 2 || got[0] != 1 || got[1] != 2 {
		t.Fatalf("superseded = %v, want [1 2] (Drafts, expired included; Rejected stays)", got)
	}
}

func TestExtendAndDelete(t *testing.T) {
	now := at("2026-11-20 10:00")
	if err := CanExtend(Quotation{Status: QuotationDraft}, day("2026-11-20"), now); err != nil {
		t.Errorf("extend to today: %v", err)
	}
	if err := CanExtend(Quotation{Status: QuotationDraft}, day("2026-11-19"), now); err == nil {
		t.Error("extend into the past is refused")
	}
	if err := CanExtend(Quotation{Status: QuotationAccepted}, day("2026-12-01"), now); err == nil {
		t.Error("only a Draft is extended")
	}
	if err := CanDeleteDraft(Quotation{Status: QuotationDraft}, 0); err != nil {
		t.Errorf("a fresh draft can be deleted: %v", err)
	}
	if err := CanDeleteDraft(Quotation{Status: QuotationDraft, RevisionCount: 1}, 0); err == nil {
		t.Error("a draft with a revision cannot be deleted")
	}
	if err := CanDeleteDraft(Quotation{Status: QuotationDraft}, 1); err == nil {
		t.Error("a draft with alternatives cannot be deleted")
	}
}

func TestTermAmounts(t *testing.T) {
	terms := DefaultTerms()
	amounts, mismatch := TermAmounts(terms, 100_000_001)
	if mismatch {
		t.Fatal("30/40/30 always adds up")
	}
	var sum int64
	for _, a := range amounts {
		sum += a
	}
	if sum != 100_000_001 || amounts[0] != 30_000_000 || amounts[1] != 40_000_000 || amounts[2] != 30_000_001 {
		t.Fatalf("amounts = %v (sum %d)", amounts, sum)
	}
	fixed := []QuotationTerm{{Label: "DP", Amount: i64p(10_000_000)}, {Label: "Pelunasan", Percent: ip(50)}}
	if _, mismatch := TermAmounts(fixed, 50_000_000); !mismatch {
		t.Fatal("10 jt + 50% of 50 jt does not add up to 50 jt")
	}
	exact := []QuotationTerm{{Label: "DP", Amount: i64p(10_000_000)}, {Label: "Pelunasan", Amount: i64p(40_000_000)}}
	if _, mismatch := TermAmounts(exact, 50_000_000); mismatch {
		t.Fatal("fixed amounts that add up are fine")
	}
}

func sampleWorkingCopy() WorkingCopy {
	venue := int64(20)
	eventDate := day("2027-03-14")
	ev := int64(30)
	pkg := int64(40)
	return WorkingCopy{
		Quotation: Quotation{ID: 1, CoupleNames: "Rina & Bayu", ContactWhatsApp: "6281234567890", PICStaffID: 5, IsPKP: true,
			TermsText: "Harga berlaku untuk tanggal acara di atas.", ValidUntil: day("2026-12-04")},
		Suppliers: []QuotationSupplier{
			{ID: venue, Kind: SupplierVenue, SupplierID: 7, Name: "Gedung Serbaguna", PaidBy: PayerWO, SortOrder: 0},
			{ID: 21, Kind: SupplierVendor, SupplierID: 8, Name: "Katering Sedap", PaidBy: PayerClient, ShowName: true, AgreedCost: i64p(25_000_000), SortOrder: 1},
		},
		Events: []QuotationEvent{{ID: ev, EventKindID: 2, KindName: "Resepsi", EventDate: &eventDate, VenueSupplierID: &venue, Layout: LayoutSeated,
			Sessions: []QuotationSession{{ID: 1, Label: "Sesi 1", Guests: 400}}}},
		Lines: []QuotationLine{
			{ID: pkg, Kind: LinePackage, PackageID: i64p(3), Description: "Paket Gold", Amount: 80_000_000, EventID: &ev, SortOrder: 0},
			{ID: 41, Kind: LineComponent, ParentLineID: &pkg, Description: "Dekorasi pelaminan", SortOrder: 0},
			{ID: 42, Kind: LineItem, Description: "Photo booth", Quantity: i64p(1), UnitPrice: i64p(5_000_000), Amount: 5_000_000, SortOrder: 1},
			{ID: 43, Kind: LineDiscount, Description: "Potongan", Amount: 2_000_000, SortOrder: 2},
		},
		Terms: DefaultTerms(),
	}
}

// renumbered is the same content stored under other database ids (a copy).
func renumbered() WorkingCopy {
	venue := int64(120)
	ev := int64(130)
	pkg := int64(140)
	out := sampleWorkingCopy()
	out.Quotation.ID = 99
	out.Suppliers[0].ID, out.Suppliers[1].ID = venue, 121
	out.Events[0].ID, out.Events[0].VenueSupplierID = ev, &venue
	out.Lines[0].ID, out.Lines[0].EventID = pkg, &ev
	out.Lines[1].ID, out.Lines[1].ParentLineID = 141, &pkg
	out.Lines[2].ID, out.Lines[3].ID = 142, 143
	// Stored in another order: the content orders itself.
	out.Lines[0], out.Lines[3] = out.Lines[3], out.Lines[0]
	return out
}

func TestRevisionHashIsTheContent(t *testing.T) {
	base, err := BuildContent(sampleWorkingCopy())
	if err != nil {
		t.Fatal(err)
	}
	same, err := BuildContent(renumbered())
	if err != nil {
		t.Fatal(err)
	}
	if base.Hash() != same.Hash() {
		t.Fatal("the same content under other ids must hash the same")
	}
	// Validity is not part of the offer.
	w := sampleWorkingCopy()
	w.Quotation.ValidUntil = day("2027-01-31")
	if c, _ := BuildContent(w); c.Hash() != base.Hash() {
		t.Fatal("moving the validity must not make a new revision")
	}
	// A price is.
	w = sampleWorkingCopy()
	w.Lines[2].Amount = 6_000_000
	if c, _ := BuildContent(w); c.Hash() == base.Hash() {
		t.Fatal("a changed amount must change the hash")
	}
	// So is the terms text.
	w = sampleWorkingCopy()
	w.Quotation.TermsText += " DP tidak dapat dikembalikan."
	if c, _ := BuildContent(w); c.Hash() == base.Hash() {
		t.Fatal("a changed terms text must change the hash")
	}
	// Positions, never database ids.
	if base.Events[0].VenueIndex == nil || *base.Events[0].VenueIndex != 0 || base.Lines[1].ParentIndex == nil || *base.Lines[1].ParentIndex != 0 {
		t.Fatalf("references must be positions: %+v / %+v", base.Events[0], base.Lines[1])
	}
	// PPN (PKP) on the subtotal: 85 jt − 2 jt = 83 jt, PPN 11%.
	if base.Totals.Subtotal != 83_000_000 || base.Totals.Tax == 0 || base.Totals.Total != base.Totals.Subtotal+base.Totals.Tax {
		t.Fatalf("totals = %+v", base.Totals)
	}
}

func TestCompareRevisions(t *testing.T) {
	before, _ := BuildContent(sampleWorkingCopy())
	w := sampleWorkingCopy()
	w.Lines[2].Amount = 6_000_000
	w.Lines[2].UnitPrice = i64p(6_000_000)
	w.Lines = append(w.Lines, QuotationLine{ID: 44, Kind: LineItem, Description: "Live music", Amount: 7_500_000, SortOrder: 3})
	after, _ := BuildContent(w)
	c := Compare(before, after)
	var added, changed int
	for _, l := range c.Lines {
		switch l.Change {
		case ChangeAdded:
			added++
		case ChangeChanged:
			changed++
		}
	}
	if added != 1 || changed != 1 {
		t.Fatalf("lines: %d added, %d changed; want 1 and 1 (%+v)", added, changed, c.Lines)
	}
	if c.TotalsBefore == c.TotalsAfter {
		t.Fatal("totals moved")
	}
	if same := Compare(before, before); len(same.Lines) != 0 || len(same.Events) != 0 || len(same.Suppliers) != 0 || same.TermsChanged {
		t.Fatalf("a revision equals itself: %+v", same)
	}
}
