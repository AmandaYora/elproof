package domain

import (
	"sort"
	"strconv"
	"time"
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
	TabClientPayments = "pembayaran/client"
	TabIssues         = "kendala"
	TabVendors        = "vendor"
	TabMilestones     = "milestone"
)

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
	Engagement []ProjectVendor
	// VendorName names an engagement (by engagement id); "" falls back to "Vendor".
	VendorName map[int64]string
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
			Title:  "Termin " + t.Term.Label + " terlambat",
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
		for _, e := range in.Engagement {
			if !e.IsUnconfirmed() {
				continue
			}
			name := in.VendorName[e.ID]
			if name == "" {
				name = "Vendor"
			}
			items = append(items, ProjectAction{
				Kind: ActionVendorUnconfirm, ID: "vendor-" + strconv.FormatInt(e.ID, 10), Tone: ActionToneWarning,
				Title:  name + " belum terkonfirmasi",
				Detail: "Acara H-" + strconv.Itoa(d) + ", status masih belum dipesan",
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
			Title: "Timeline \"" + m.Name + "\" lewat tenggat", Detail: "Target " + FormatDateID(m.TargetDate), Tab: TabMilestones,
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

// UnconfirmedEngagements are the vendors not booked yet (Planned or Negotiation).
func UnconfirmedEngagements(list []ProjectVendor) []ProjectVendor {
	var out []ProjectVendor
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
	ContractValue int64
	Paid          int64
	Remaining     int64
	OverdueTotal  int64
	VendorCost    int64
	VenueCost     int64
	Margin        int64
}

// ComputeFinanceOverview derives the project's money summary from its raw
// numbers. Paid and Remaining are the client's payments applied to the contract
// (DP + Termin + Pelunasan - Refund, Tambahan excluded; Remaining never
// negative: see ComputeLedger), VendorCost is the sum of the contract values of
// the engagements that are not cancelled, VenueCost the venue's rental price
// plus charge and Margin = contract - vendor cost - venue cost (may be negative).
func ComputeFinanceOverview(contractValue int64, clientPayments []PaymentLine, terms []TermProgress, engagements []ProjectVendor, venueRental, venueCharge *int64) FinanceOverview {
	ledger := ComputeLedger(clientPayments, contractValue)
	var vendorCost int64
	for _, e := range engagements {
		if e.EngagementStatus != EngagementCancelled {
			vendorCost += e.ContractValue
		}
	}
	var venueCost int64
	if venueRental != nil {
		venueCost += *venueRental
	}
	if venueCharge != nil {
		venueCost += *venueCharge
	}
	return FinanceOverview{
		ContractValue: contractValue, Paid: ledger.Paid, Remaining: ledger.Remaining,
		OverdueTotal: OverdueRemaining(terms), VendorCost: vendorCost, VenueCost: venueCost,
		Margin: contractValue - vendorCost - venueCost,
	}
}
