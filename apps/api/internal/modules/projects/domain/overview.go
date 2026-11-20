package domain

import (
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// Pure rules behind GET /projects/{id}/overview (E1): what needs attention on a
// project, the next timeline items, which vendors are still unconfirmed and the
// money summary. They are the rules the web used to compute in the browser
// (ProjectSummaryTabPage, projects/lib/payments.ts), moved here so the server is
// the one place that decides.

// ActionKind says what a "perlu tindakan" item is about.
type ActionKind string

const (
	ActionTermOverdue      ActionKind = "term_overdue"
	ActionIssueOpen        ActionKind = "issue_open"
	ActionVendorUnconfirm  ActionKind = "vendor_unconfirmed"
	ActionMilestoneLate    ActionKind = "milestone_late"
	ActionToneDanger                  = "danger"
	ActionToneWarning                 = "warning"
	vendorConfirmWindowDay            = 30
	// MaxNextMilestones caps the "Timeline berikutnya" list.
	MaxNextMilestones = 5
)

// Tab names are the project detail tabs an action links to.
const (
	TabClientPayments = "keuangan/klien"
	TabIssues         = "kendala"
	TabVendors        = "vendor"
	TabMilestones     = "jadwal"
)

// TermLabel names an installment in a sentence: "Termin DP", but a label that
// already says "Termin 2" stays as it is (never "Termin Termin 2").
func TermLabel(label string) string {
	trimmed := strings.TrimSpace(label)
	const word = "termin"
	if len(trimmed) >= len(word) && strings.EqualFold(trimmed[:len(word)], word) {
		// Only the whole word: "Termin 2", "termin-3", not "Terminal".
		if rest := trimmed[len(word):]; rest == "" || !unicode.IsLetter([]rune(rest)[0]) {
			return trimmed
		}
	}
	return "Termin " + trimmed
}

// ProjectAction is one item of "Perlu tindakan".
type ProjectAction struct {
	Kind   ActionKind
	ID     string
	Tone   string
	Title  string
	Detail string
	Tab    string
}

// ActionInput is everything action selection needs.
type ActionInput struct {
	Today      time.Time
	EventDate  time.Time
	Terms      []TermProgress
	Issues     []VendorIssue
	Suppliers  []ProjectSupplier
	Milestones []ProjectMilestone
}

// BuildProjectActions returns the project's open actions: an overdue payment
// term (danger), an open kendala (danger for High/Critical, else warning), a
// vendor still Planned/Negotiation within 30 days of the event (warning) and a
// timeline item past its target date (warning). Danger items come first; inside
// each tone the order is terms, kendala, vendors, timeline.
func BuildProjectActions(in ActionInput) []ProjectAction {
	var items []ProjectAction
	for _, t := range in.Terms {
		if t.Status != TermOverdue {
			continue
		}
		items = append(items, ProjectAction{
			Kind: ActionTermOverdue, ID: "term-" + strconv.FormatInt(t.Term.ID, 10), Tone: ActionToneDanger,
			Title:  TermLabel(t.Term.Label) + " terlambat",
			Detail: "Sisa " + FormatRupiah(t.Remaining) + ", jatuh tempo " + FormatDateID(t.Term.DueDate),
			Tab:    TabClientPayments,
		})
	}
	for _, i := range in.Issues {
		if !i.Status.IsOpen() {
			continue
		}
		tone := ActionToneWarning
		if i.Impact == ImpactHigh || i.Impact == ImpactCritical {
			tone = ActionToneDanger
		}
		items = append(items, ProjectAction{
			Kind: ActionIssueOpen, ID: "issue-" + strconv.FormatInt(i.ID, 10), Tone: tone,
			Title: i.Title, Detail: "Kendala masih terbuka", Tab: TabIssues,
		})
	}
	if d := daysBetween(in.Today, in.EventDate); d >= 0 && d <= vendorConfirmWindowDay {
		when := "Acara H-" + strconv.Itoa(d)
		if d == 0 {
			when = "Acara hari ini" // not "H-0" (DESIGN.md §7: H-30, Hari H, H+3)
		}
		for _, e := range in.Suppliers {
			if !e.IsUnconfirmed() {
				continue
			}
			name := e.Name
			items = append(items, ProjectAction{
				Kind: ActionVendorUnconfirm, ID: "vendor-" + strconv.FormatInt(e.ID, 10), Tone: ActionToneWarning,
				Title:  name + " belum terkonfirmasi",
				Detail: when + ", status masih belum dipesan",
				Tab:    TabVendors,
			})
		}
	}
	for _, m := range in.Milestones {
		if m.Status == MilestoneCompleted || m.Status == MilestoneCancelled || daysBetween(in.Today, m.TargetDate) >= 0 {
			continue
		}
		items = append(items, ProjectAction{
			Kind: ActionMilestoneLate, ID: "ms-" + strconv.FormatInt(m.ID, 10), Tone: ActionToneWarning,
			Title: "Tahapan \"" + m.Name + "\" lewat tenggat", Detail: "Target " + FormatDateID(m.TargetDate), Tab: TabMilestones,
		})
	}
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].Tone == ActionToneDanger && items[j].Tone != ActionToneDanger
	})
	return items
}

// NextMilestones are the (up to five) timeline items not yet completed or
// cancelled, earliest target date first (items with the same date keep their
// order on the timeline).
func NextMilestones(list []ProjectMilestone) []ProjectMilestone {
	var open []ProjectMilestone
	for _, m := range list {
		if m.Status != MilestoneCompleted && m.Status != MilestoneCancelled {
			open = append(open, m)
		}
	}
	sort.SliceStable(open, func(i, j int) bool { return open[i].TargetDate.Before(open[j].TargetDate) })
	if len(open) > MaxNextMilestones {
		open = open[:MaxNextMilestones]
	}
	return open
}

// UnconfirmedSuppliers are the vendors and venues not booked yet (Planned or
// Negotiation).
func UnconfirmedSuppliers(list []ProjectSupplier) []ProjectSupplier {
	var out []ProjectSupplier
	for _, e := range list {
		if e.IsUnconfirmed() {
			out = append(out, e)
		}
	}
	return out
}

// OpenIssueCount counts kendala that are not Resolved or Closed.
func OpenIssueCount(list []VendorIssue) int {
	n := 0
	for _, i := range list {
		if i.Status.IsOpen() {
			n++
		}
	}
	return n
}

// OverdueRemaining is the unpaid part of every overdue installment.
func OverdueRemaining(terms []TermProgress) int64 {
	var total int64
	for _, t := range terms {
		if t.Status == TermOverdue {
			total += t.Remaining
		}
	}
	return total
}

// FinanceOverview is the money summary of one project.
type FinanceOverview struct {
	// ContractValue is the contract total (subtotal + PPN); ContractSubtotal and
	// ContractTax are its parts (the project's cache of its lines).
	ContractValue    int64
	ContractSubtotal int64
	ContractTax      int64
	Paid             int64
	Remaining        int64
	OverdueTotal     int64
	// TermsTotal is what the client's schedule adds up to; TermsMismatch is set
	// when a schedule exists and no longer equals the contract value.
	TermsTotal    int64
	TermsMismatch bool
	// SupplierCost is the agreed cost of the active vendors and venues the WO
	// pays; MissingCost counts those without one yet. Margin = subtotal -
	// SupplierCost (may be negative; PPN is not the WO's) and is incomplete while
	// MissingCost > 0 ("Margin belum lengkap"). A supplier the client pays is
	// outside both.
	SupplierCost int64
	MissingCost  int
	Margin       int64
}

// ComputeFinanceOverview derives the project's money summary from its raw
// numbers. Paid and Remaining are the client's payments applied to the contract
// (DP + Termin + Pelunasan - Refund, Tambahan excluded; Remaining never
// negative: see ComputeLedger).
func ComputeFinanceOverview(p Project, clientPayments []PaymentLine, terms []TermProgress, suppliers []ProjectSupplier) FinanceOverview {
	ledger := ComputeLedger(clientPayments, p.ContractTotal)
	margin := ComputeMargin(p.ContractSubtotal, suppliers)
	raw := make([]PaymentTerm, len(terms))
	for i, t := range terms {
		raw[i] = t.Term
	}
	return FinanceOverview{
		ContractValue: p.ContractTotal, ContractSubtotal: p.ContractSubtotal, ContractTax: p.ContractTax,
		Paid: ledger.Paid, Remaining: ledger.Remaining, OverdueTotal: OverdueRemaining(terms),
		TermsTotal: TermsTotal(raw), TermsMismatch: TermsMismatch(raw, p.ContractTotal),
		SupplierCost: margin.SupplierCost, MissingCost: margin.MissingCost, Margin: margin.Margin,
	}
}

// SupplierCost sums the agreed cost of the active (not cancelled, not deleted)
// suppliers the WO pays, and counts those that have none yet.
func SupplierCost(suppliers []ProjectSupplier) (cost int64, missing int) {
	for _, s := range suppliers {
		if !s.IsActive() || s.DeletedAt != nil || s.PaidBy != PayerWO {
			continue
		}
		if s.AgreedCost == nil {
			missing++
			continue
		}
		cost += *s.AgreedCost
	}
	return cost, missing
}
