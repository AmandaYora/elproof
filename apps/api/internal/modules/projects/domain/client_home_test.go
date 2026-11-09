package domain

import (
	"testing"
	"time"
)

var today = time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC)

func off(offset int) time.Time { return today.AddDate(0, 0, offset) }

func term(id int64, label string, due int, amount, paid int64) TermProgress {
	return AllocateTerms([]PaymentTerm{{ID: id, Label: label, Amount: amount, DueDate: off(due)}}, paid, today)[0]
}

func TestRemainingAndNextTerm(t *testing.T) {
	if RemainingToPay(100, 30) != 70 || RemainingToPay(100, 100) != 0 || RemainingToPay(100, 130) != 0 || RemainingToPay(0, 0) != 0 {
		t.Error("remaining must be contract minus applied, never negative")
	}
	if PaidOnContract(-5) != 0 || PaidOnContract(40) != 40 {
		t.Error("paid is never negative")
	}

	terms := AllocateTerms([]PaymentTerm{
		{ID: 1, Label: "DP", Amount: 100, DueDate: off(-20)},
		{ID: 2, Label: "Termin 1", Amount: 100, DueDate: off(5)},
		{ID: 3, Label: "Pelunasan", Amount: 100, DueDate: off(40)},
	}, 150, today)
	next := NextOpenTerm(terms)
	if next == nil || next.Term.ID != 2 || next.Remaining != 50 || next.Status != TermPartial {
		t.Fatalf("next term should be the partly paid Termin 1, got %+v", next)
	}
	if OverdueTermCount(terms) != 0 {
		t.Errorf("nothing is overdue here")
	}
	all := AllocateTerms([]PaymentTerm{{ID: 1, Amount: 100, DueDate: off(-1)}, {ID: 2, Amount: 100, DueDate: off(-2)}}, 0, today)
	if OverdueTermCount(all) != 2 {
		t.Errorf("two overdue terms expected")
	}
	paid := AllocateTerms([]PaymentTerm{{ID: 1, Amount: 100, DueDate: off(-1)}}, 100, today)
	if NextOpenTerm(paid) != nil {
		t.Error("a fully paid schedule has no next term")
	}
	if NextOpenTerm(nil) != nil {
		t.Error("no schedule, no next term")
	}
}

func TestSelectHomeActions(t *testing.T) {
	overdueOld := term(1, "DP", -30, 1_000_000, 0)
	overdueNew := term(2, "Termin 1", -3, 2_000_000, 500_000)
	soon := term(3, "Termin 2", 4, 3_000_000, 0)
	soonPartial := term(4, "Termin 3", 6, 1_000_000, 200_000)
	later := term(5, "Pelunasan", 30, 1_000_000, 0)
	paid := term(6, "Lunas", -50, 100, 100)
	in := HomeActionInput{
		Terms:      []TermProgress{later, soon, overdueNew, overdueOld, paid, soonPartial},
		Today:      today,
		OpenIssues: 2,
		ShowTerms:  true,
		ShowIssues: true,
	}

	got := SelectHomeActions(in)
	if len(got) != MaxHomeActions {
		t.Fatalf("capped at %d, got %d", MaxHomeActions, len(got))
	}
	// Overdue first (oldest first), then due soon; the issue line is crowded out here.
	if got[0].Amount == nil || *got[0].Amount != 1_000_000 || got[1].Amount == nil || *got[1].Amount != 1_500_000 || got[2].Amount == nil || *got[2].Amount != 3_000_000 {
		t.Errorf("wrong order or amounts: %+v", got)
	}
	for _, a := range got {
		if a.Kind != HomeActionPayTerm || a.Target != "pembayaran" || a.DueDate == nil {
			t.Errorf("term action malformed: %+v", a)
		}
	}
	if got[0].Title != "Pembayaran DP sudah lewat jatuh tempo" {
		t.Errorf("overdue wording: %q", got[0].Title)
	}
	if got[2].Title != "Pembayaran Termin 2 jatuh tempo 14 Mei 2026" || got[2].Detail != "Rp3.000.000" {
		t.Errorf("due-soon wording: %q / %q", got[2].Title, got[2].Detail)
	}

	// With room left, the open-issues line follows the terms.
	in.Terms = []TermProgress{soon}
	got = SelectHomeActions(in)
	if len(got) != 2 || got[1].Kind != HomeActionOpenIssues || got[1].Title != "2 hal sedang kami tangani" || got[1].Target != "kendala" || got[1].DueDate != nil || got[1].Amount != nil {
		t.Errorf("issues action: %+v", got)
	}

	// Sections the role may not see produce nothing.
	in.ShowTerms = false
	if got = SelectHomeActions(in); len(got) != 1 || got[0].Kind != HomeActionOpenIssues {
		t.Errorf("without finance only issues remain: %+v", got)
	}
	in.ShowIssues = false
	if got = SelectHomeActions(in); len(got) != 0 {
		t.Errorf("nothing visible, nothing to do: %+v", got)
	}
	in.ShowIssues, in.OpenIssues = true, 0
	if got = SelectHomeActions(in); len(got) != 0 {
		t.Errorf("zero open issues is not an action: %+v", got)
	}
}

func TestOrderRecentUpdates(t *testing.T) {
	at := func(h int) time.Time { return today.Add(time.Duration(h) * time.Hour) }
	in := []RecentUpdate{
		{Kind: UpdateMilestoneDone, Title: "a", At: at(1), SourceID: 1},
		{Kind: UpdatePaymentVerified, Title: "b", At: at(5), SourceID: 2},
		{Kind: UpdateIssueResolved, Title: "c", At: at(3), SourceID: 3},
		{Kind: UpdateMilestoneDone, Title: "d", At: at(3), SourceID: 4},
		{Kind: UpdateMilestoneDone, Title: "e", At: at(3), SourceID: 9},
		{Kind: UpdatePaymentVerified, Title: "f", At: at(0), SourceID: 5},
		{Kind: UpdateMilestoneDone, Title: "g", At: at(-8), SourceID: 6},
	}
	got := OrderRecentUpdates(in)
	var titles string
	for _, u := range got {
		titles += u.Title
	}
	// Newest first; at the same time a milestone (higher id first) comes before an issue.
	if len(got) != MaxRecentUpdates || titles != "bedca" {
		t.Errorf("order = %q (len %d), want %q capped at %d", titles, len(got), "bedca", MaxRecentUpdates)
	}
	if len(in) != 7 || in[0].Title != "a" {
		t.Error("the input slice must not be reordered")
	}
}

func TestFirstVisibleProof(t *testing.T) {
	up := func(h int) time.Time { return today.Add(time.Duration(h) * time.Hour) }
	list := []Evidence{
		{ID: 3, RelatedKind: RelatedProjectMilestone, RelatedID: 5, Type: EvidencePhoto, FileName: "baru.jpg", UploadedAt: up(3), IsClientVisible: true},
		{ID: 1, RelatedKind: RelatedProjectMilestone, RelatedID: 5, Type: EvidencePhoto, FileName: "internal.jpg", UploadedAt: up(1)},
		{ID: 2, RelatedKind: RelatedProjectMilestone, RelatedID: 5, Type: EvidencePhoto, FileName: "asli.jpg", UploadedAt: up(2), IsClientVisible: true},
		{ID: 4, RelatedKind: RelatedProjectMilestone, RelatedID: 6, FileName: "lain.jpg", UploadedAt: up(0), IsClientVisible: true},
		{ID: 5, RelatedKind: RelatedClientPayment, RelatedID: 5, Type: EvidenceTransferProof, FileName: "bukti.pdf", UploadedAt: up(0)},
	}
	all := func(string) bool { return true }
	p := FirstVisibleProof(list, RelatedProjectMilestone, 5, all)
	if p == nil || p.EvidenceID != 2 || p.FileName != "asli.jpg" || p.Type != EvidencePhoto {
		t.Fatalf("want the earliest client-visible file of milestone 5, got %+v", p)
	}
	if FirstVisibleProof(list, RelatedProjectMilestone, 7, all) != nil {
		t.Error("no files, no proof")
	}
	noTimeline := func(perm string) bool { return perm != "portal.timeline.view" }
	if FirstVisibleProof(list, RelatedProjectMilestone, 5, noTimeline) != nil {
		t.Error("without portal.timeline.view no milestone proof")
	}
	noFinance := func(perm string) bool { return perm != "portal.finance.view" }
	if FirstVisibleProof(list, RelatedClientPayment, 5, noFinance) != nil {
		t.Error("without portal.finance.view no payment proof")
	}
	if p := FirstVisibleProof(list, RelatedClientPayment, 5, all); p == nil || p.EvidenceID != 5 {
		t.Errorf("payment proof expected, got %+v", p)
	}
}

func TestClientCanSeeEvidenceMilestoneNeedsFlagAndPermission(t *testing.T) {
	all := func(string) bool { return true }
	noTimeline := func(p string) bool { return p != "portal.timeline.view" }
	flagged := Evidence{RelatedKind: RelatedProjectMilestone, IsClientVisible: true}
	unflagged := Evidence{RelatedKind: RelatedProjectMilestone}
	if !ClientCanSeeEvidence(flagged, all) {
		t.Error("a flagged timeline attachment is visible with portal.timeline.view")
	}
	if ClientCanSeeEvidence(unflagged, all) {
		t.Error("an unflagged timeline attachment stays internal")
	}
	if ClientCanSeeEvidence(flagged, noTimeline) {
		t.Error("a flagged attachment is still hidden without portal.timeline.view")
	}
	for _, k := range []EvidenceRelatedKind{RelatedPayment, RelatedVenuePayment, RelatedProjectVendor} {
		if ClientCanSeeEvidence(Evidence{RelatedKind: k, IsClientVisible: true}, all) {
			t.Errorf("%s must never be visible to a client, flag or not", k)
		}
	}
	if !RelatedGeneral.SupportsClientVisibility() || !RelatedProjectMilestone.SupportsClientVisibility() || RelatedClientPayment.SupportsClientVisibility() {
		t.Error("only general documents and timeline attachments carry the flag")
	}
}
