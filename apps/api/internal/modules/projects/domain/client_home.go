package domain

import (
	"sort"
	"strconv"
	"strings"
	"time"
)

// Pure rules behind GET /projects/me/home (Client Portal "Beranda", E5). Kept
// free of I/O so each rule is a plain unit test; the application service only
// fetches and assembles.

// RemainingToPay is what is still owed against the contract: the contract value
// minus what has been applied to it, never negative (an overpaid contract owes
// nothing; the overpayment itself is a staff matter).
func RemainingToPay(contractValue, applied int64) int64 {
	if r := contractValue - applied; r > 0 {
		return r
	}
	return 0
}

// PaidOnContract is the applied total shown to the client, never negative.
func PaidOnContract(applied int64) int64 {
	if applied < 0 {
		return 0
	}
	return applied
}

// DaysUntil counts calendar days from today to date (negative once past).
func DaysUntil(today, date time.Time) int { return daysBetween(today, date) }

// NextOpenTerm is the earliest installment that is not fully paid. `terms` must
// come from AllocateTerms, which orders by due date.
func NextOpenTerm(terms []TermProgress) *TermProgress {
	for i := range terms {
		if terms[i].Status != TermPaid {
			t := terms[i]
			return &t
		}
	}
	return nil
}

// OverdueTermCount counts installments past their due date and not fully paid.
func OverdueTermCount(terms []TermProgress) int {
	n := 0
	for _, t := range terms {
		if t.Status == TermOverdue {
			n++
		}
	}
	return n
}

type HomeActionKind string

const (
	HomeActionPayTerm    HomeActionKind = "pay_term"
	HomeActionOpenIssues HomeActionKind = "open_issues"
)

// Targets are the portal tabs an action points at.
const (
	HomeTargetPayments = "pembayaran"
	HomeTargetIssues   = "kendala"
)

// MaxHomeActions caps "Yang perlu Anda lakukan".
const MaxHomeActions = 3

// HomeAction is one thing the client should do or know about.
type HomeAction struct {
	Kind    HomeActionKind
	Title   string
	Detail  string
	DueDate *time.Time
	Amount  *int64
	Target  string
}

// HomeActionInput is everything action selection needs. ShowTerms is false
// when the role may not see finance or the project is cancelled; ShowIssues is
// false when the role may not see issues.
type HomeActionInput struct {
	Terms      []TermProgress
	Today      time.Time
	OpenIssues int
	ShowTerms  bool
	ShowIssues bool
}

// SelectHomeActions picks up to MaxHomeActions things, most urgent first:
// installments past their due date (oldest first), then installments due within
// the next 7 days (soonest first, also when partly paid), then the count of
// open issues. Same semantics the portal computed in the browser before E5: red
// only for what is overdue, amber for what is coming up.
func SelectHomeActions(in HomeActionInput) []HomeAction {
	var overdue, soon []TermProgress
	if in.ShowTerms {
		for _, t := range in.Terms {
			switch {
			case t.Status == TermOverdue:
				overdue = append(overdue, t)
			case (t.Status == TermDueSoon || t.Status == TermPartial) && t.Remaining > 0:
				if d := daysBetween(in.Today, t.Term.DueDate); d >= 0 && d <= dueSoonDays {
					soon = append(soon, t)
				}
			}
		}
	}
	byDue := func(l []TermProgress) {
		sort.SliceStable(l, func(i, j int) bool {
			if !l[i].Term.DueDate.Equal(l[j].Term.DueDate) {
				return l[i].Term.DueDate.Before(l[j].Term.DueDate)
			}
			return l[i].Term.ID < l[j].Term.ID
		})
	}
	byDue(overdue)
	byDue(soon)

	var out []HomeAction
	for _, t := range overdue {
		out = append(out, termAction(t, true))
	}
	for _, t := range soon {
		out = append(out, termAction(t, false))
	}
	if in.ShowIssues && in.OpenIssues > 0 {
		out = append(out, HomeAction{
			Kind:   HomeActionOpenIssues,
			Title:  strconv.Itoa(in.OpenIssues) + " hal sedang kami tangani",
			Detail: "Lihat apa yang terjadi dan apa yang sedang kami lakukan.",
			Target: HomeTargetIssues,
		})
	}
	if len(out) > MaxHomeActions {
		out = out[:MaxHomeActions]
	}
	return out
}

func termAction(t TermProgress, overdue bool) HomeAction {
	due := t.Term.DueDate
	amount := t.Remaining
	a := HomeAction{Kind: HomeActionPayTerm, DueDate: &due, Amount: &amount, Target: HomeTargetPayments}
	if overdue {
		a.Title = "Pembayaran " + t.Term.Label + " sudah lewat jatuh tempo"
		a.Detail = FormatRupiah(t.Remaining) + ", jatuh tempo " + FormatDateID(due) + ". Hubungi kami bila ada kendala."
	} else {
		a.Title = "Pembayaran " + t.Term.Label + " jatuh tempo " + FormatDateID(due)
		a.Detail = FormatRupiah(t.Remaining)
	}
	return a
}

var monthsID = [...]string{"Jan", "Feb", "Mar", "Apr", "Mei", "Jun", "Jul", "Agu", "Sep", "Okt", "Nov", "Des"}

// FormatDateID writes a date the way the portal does: "5 Mei 2026" (short month names, same as the web formatter).
func FormatDateID(t time.Time) string {
	return strconv.Itoa(t.Day()) + " " + monthsID[t.Month()-1] + " " + strconv.Itoa(t.Year())
}

// FormatRupiah writes an amount as "Rp1.500.000".
func FormatRupiah(n int64) string {
	digits := strconv.FormatInt(n, 10)
	var b strings.Builder
	for i, ch := range digits {
		if i > 0 && (len(digits)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(ch)
	}
	return "Rp" + b.String()
}

// ---- recent updates ("Kabar terbaru") ----

type RecentUpdateKind string

const (
	UpdateMilestoneDone   RecentUpdateKind = "milestone_done"
	UpdatePaymentVerified RecentUpdateKind = "payment_verified"
	UpdateIssueResolved   RecentUpdateKind = "issue_resolved"
)

// MaxRecentUpdates caps "Kabar terbaru".
const MaxRecentUpdates = 5

// UpdateProof is the proof file attached to an update (an evidence row the
// client may see).
type UpdateProof struct {
	EvidenceID int64
	Type       EvidenceType
	FileName   string
}

// RecentUpdate is one line of news for the client.
type RecentUpdate struct {
	Kind  RecentUpdateKind
	Title string
	At    time.Time
	Proof *UpdateProof
	// VerifiedByStaffID is resolved to a name by the caller; never sent as an id.
	VerifiedByStaffID int64
	// sourceID breaks ties between updates of the same time and kind.
	SourceID int64
}

// OrderRecentUpdates sorts newest first (ties: payments before milestones
// before issues, then the higher source id) and keeps MaxRecentUpdates.
func OrderRecentUpdates(updates []RecentUpdate) []RecentUpdate {
	rank := map[RecentUpdateKind]int{UpdatePaymentVerified: 0, UpdateMilestoneDone: 1, UpdateIssueResolved: 2}
	out := append([]RecentUpdate(nil), updates...)
	sort.SliceStable(out, func(i, j int) bool {
		if !out[i].At.Equal(out[j].At) {
			return out[i].At.After(out[j].At)
		}
		if out[i].Kind != out[j].Kind {
			return rank[out[i].Kind] < rank[out[j].Kind]
		}
		return out[i].SourceID > out[j].SourceID
	})
	if len(out) > MaxRecentUpdates {
		out = out[:MaxRecentUpdates]
	}
	return out
}

// PaymentVerifiedTitle is the plain-language headline for a verified payment.
func PaymentVerifiedTitle(t PaymentType) string {
	switch t {
	case PaymentDP:
		return "DP diterima"
	case PaymentTermin:
		return "Pembayaran termin diterima"
	case PaymentPelunasan:
		return "Pelunasan diterima"
	case PaymentTambahan:
		return "Pembayaran tambahan diterima"
	case PaymentRefund:
		return "Pengembalian dana tercatat"
	}
	return "Pembayaran diterima"
}

// FirstVisibleProof picks the proof for one related item: among the
// evidence rows of (kind, relatedID) that the client may see, the earliest
// uploaded (the original proof rather than a later extra). nil when none.
func FirstVisibleProof(evidences []Evidence, kind EvidenceRelatedKind, relatedID int64, can func(string) bool) *UpdateProof {
	var best *Evidence
	for i := range evidences {
		e := &evidences[i]
		if e.RelatedKind != kind || e.RelatedID != relatedID || !ClientCanSeeEvidence(*e, can) {
			continue
		}
		if best == nil || e.UploadedAt.Before(best.UploadedAt) || (e.UploadedAt.Equal(best.UploadedAt) && e.ID < best.ID) {
			best = e
		}
	}
	if best == nil {
		return nil
	}
	return &UpdateProof{EvidenceID: best.ID, Type: best.Type, FileName: best.FileName}
}
