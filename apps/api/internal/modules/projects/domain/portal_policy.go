package domain

import "time"

// PortalPolicy is the one rule book for what a client principal (Client
// Portal) may see of their own project (PLAN §4.4). Every portal read goes
// through it, so a rule is written once and the route test can rely on it.
// can is the client's portal permission check (ADR-0026).
type PortalPolicy struct {
	can func(string) bool
}

// NewPortalPolicy builds the policy for one client's permissions.
func NewPortalPolicy(can func(string) bool) PortalPolicy {
	if can == nil {
		can = func(string) bool { return false }
	}
	return PortalPolicy{can: can}
}

// Portal permissions the policy reads.
const (
	PermPortalTimeline  = "portal.timeline.view"
	PermPortalVendor    = "portal.vendor.view"
	PermPortalFinance   = "portal.finance.view"
	PermPortalIssues    = "portal.issues.view"
	PermPortalDocuments = "portal.documents.view"
)

// Events: the events, their sessions, guests and venues (name, address, photo).
func (p PortalPolicy) Events() bool { return p.can(PermPortalTimeline) }

// Suppliers: the vendors and venues working on the project and their steps.
func (p PortalPolicy) Suppliers() bool { return p.can(PermPortalVendor) }

// Finance: amounts, the contract value, terms and money in.
func (p PortalPolicy) Finance() bool { return p.can(PermPortalFinance) }

// SupplierName is what the client is shown for a supplier: its name when they
// may know who it is (ProjectSupplier.ClientSeesName), otherwise the service it
// provides (the category name, or "Layanan" when it has none). Used for the
// supplier itself, its steps, its kendala and the names of its proof files.
func (p PortalPolicy) SupplierName(s ProjectSupplier, categoryName string) string {
	if s.ClientSeesName() {
		return s.Name
	}
	if categoryName != "" {
		return categoryName
	}
	return "Layanan"
}

// SupplierPayment: only a payment the client made to a supplier they pay
// directly, and only with portal.finance.view. What the WO pays its suppliers
// (and its proof) is never shown.
func (p PortalPolicy) SupplierPayment(pay SupplierPayment) bool {
	return pay.Payer == PayerClient && p.Finance()
}

// BillTerm: the bills of a supplier the client pays directly (Paket WO Tahap
// 4), with portal.finance.view, only for a supplier whose payer is the client.
// A Keluarga (no finance) sees none; the WO's own suppliers never have bills.
func (p PortalPolicy) BillTerm(sup ProjectSupplier) bool {
	return sup.PaidBy == PayerClient && p.Finance()
}

// Issue: a kendala staff shared (is_client_visible), with portal.issues.view.
func (p PortalPolicy) Issue(isClientVisible bool) bool {
	return isClientVisible && p.can(PermPortalIssues)
}

// EvidenceOwner is what the evidence rule needs to know about the thing a file
// hangs off, beyond the file itself: for a supplier's step, who pays the
// supplier; for a supplier payment, who made it. Known is false when the owner
// is not live (deleted, or not found): such a file is never shown.
type EvidenceOwner struct {
	Known bool
	// PaidBy is the supplier's payer (vendorMilestone, projectSupplier).
	PaidBy Payer
	// Payer is the payment's payer (supplierPayment).
	Payer Payer
}

// Evidence is the per-file rule. Only files that back something the portal
// shows are visible:
//   - proof of the client's own payments (portal.finance.view);
//   - files of a shared kendala (portal.issues.view);
//   - files of a supplier step (portal.vendor.view); for a supplier the WO pays
//     only when staff marked the file "tampilkan ke klien" (a receipt photo can
//     reveal the name or the partner price);
//   - files of a supplier payment the client made (portal.finance.view);
//   - timeline attachments and project documents staff opted in.
//
// What the WO pays its suppliers and the supplier's own papers (booking proof,
// contract) stay internal.
func (p PortalPolicy) Evidence(e Evidence, owner EvidenceOwner) bool {
	if e.ClientHidden {
		return false
	}
	switch e.RelatedKind {
	case RelatedClientPayment:
		return p.Finance()
	case RelatedIssue:
		return p.can(PermPortalIssues)
	case RelatedVendorMilestone:
		if !owner.Known || !p.Suppliers() {
			return false
		}
		return owner.PaidBy == PayerClient || e.IsClientVisible
	case RelatedSupplierPayment:
		return owner.Known && owner.Payer == PayerClient && p.Finance()
	case RelatedProjectMilestone:
		return p.can(PermPortalTimeline) && e.IsClientVisible
	case RelatedGeneral:
		return e.IsClientVisible && p.can(PermPortalDocuments)
	default:
		return false
	}
}

// EvidenceOwners answers EvidenceOwner for every file of one project: who pays
// each live supplier (by supplier and by step) and who made each live supplier
// payment. Built once per request (SupplierService.EvidenceOwners).
type EvidenceOwners struct {
	SupplierPaidBy  map[int64]Payer // project supplier id → its payer
	MilestonePaidBy map[int64]Payer // vendor milestone id → its supplier's payer
	PaymentPayer    map[int64]Payer // supplier payment id → its payer
}

// Of is the owner of one file (Known false when the owner is not live).
func (o EvidenceOwners) Of(e Evidence) EvidenceOwner {
	var payer Payer
	var ok bool
	switch e.RelatedKind {
	case RelatedVendorMilestone:
		payer, ok = o.MilestonePaidBy[e.RelatedID]
		return EvidenceOwner{Known: ok, PaidBy: payer}
	case RelatedProjectSupplier:
		payer, ok = o.SupplierPaidBy[e.RelatedID]
		return EvidenceOwner{Known: ok, PaidBy: payer}
	case RelatedSupplierPayment:
		payer, ok = o.PaymentPayer[e.RelatedID]
		return EvidenceOwner{Known: ok, Payer: payer}
	}
	return EvidenceOwner{Known: true}
}

// CanSee is the per-file rule with the file's owner looked up.
func (p PortalPolicy) CanSee(e Evidence, owners EvidenceOwners) bool {
	return p.Evidence(e, owners.Of(e))
}

// PortalProject is a project as the portal shows it: the internal fields
// cleared, and its money (contract value, PPN) only when ShowMoney.
type PortalProject struct {
	Project   Project
	ShowMoney bool
}

// Project: the project itself is always the client's to see; the preparation
// start, the Sales PIC and the lines' version are internal, and the contract
// value and PPN need portal.finance.view (Keluarga sees no numbers at all).
func (p PortalPolicy) Project(pr Project) PortalProject {
	pr.PrepStartDate = time.Time{}
	pr.PICSalesStaffID = 0
	pr.LinesVersion = time.Time{}
	show := p.Finance()
	if !show {
		pr.ContractSubtotal, pr.ContractTax, pr.ContractTotal = 0, 0, 0
	}
	return PortalProject{Project: pr, ShowMoney: show}
}

// PortalHome says which sections of the portal home a client sees.
type PortalHome struct {
	// Money: the contract value, PPN, what was paid, the next installment.
	Money bool
	// Services: the package names and the Layanan tab (lines and suppliers).
	Services bool
	// Events: the events, where they happen and the countdown details.
	Events bool
}

// Home is the portal home's sections for this client.
func (p PortalPolicy) Home() PortalHome {
	return PortalHome{Money: p.Finance(), Services: p.Suppliers(), Events: p.Events()}
}

// PortalLine is a line as the portal shows it. ShowAmount says whether its
// amount (and an item's unit price, a rule's prices) may be shown.
type PortalLine struct {
	Line       ProjectLine
	ShowAmount bool
}

// Line is the rule for one line of the Rincian (PLAN §4.4): the lines' names
// and a package's components need portal.vendor.view; amounts, unit prices
// and the rule's prices need portal.finance.view; a component's price is never
// shown (it has none of its own) and neither is the supplier that provides a
// line (the WO's partner). ok is false when the line is not shown at all.
func (p PortalPolicy) Line(l ProjectLine) (PortalLine, bool) {
	if !p.Suppliers() || l.DeletedAt != nil {
		return PortalLine{}, false
	}
	l.SupplierRef = nil
	show := p.Finance() && l.Kind != LineComponent
	if !show {
		l.Amount = 0
		l.UnitPrice = nil
	}
	if l.PackageRule != nil {
		rule := *l.PackageRule
		rule.EventKindIDs = append([]int64(nil), rule.EventKindIDs...)
		if !p.Finance() {
			rule.Price = 0
			rule.ExtraGuestStepPrice = nil
		}
		l.PackageRule = &rule
	}
	return PortalLine{Line: l, ShowAmount: show}, true
}

// Term: the client's payment schedule, with portal.finance.view.
func (p PortalPolicy) Term() bool { return p.Finance() }
