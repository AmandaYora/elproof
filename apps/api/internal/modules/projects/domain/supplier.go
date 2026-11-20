package domain

import (
	"fmt"
	"time"
)

// EngagementStatus is where the work with one vendor or venue of a project stands.
type EngagementStatus string

const (
	EngagementPlanned     EngagementStatus = "Planned"
	EngagementNegotiation EngagementStatus = "Negotiation"
	EngagementBooked      EngagementStatus = "Booked"
	EngagementDPPaid      EngagementStatus = "DP Paid"
	EngagementInProgress  EngagementStatus = "In Progress"
	EngagementFullyPaid   EngagementStatus = "Fully Paid"
	EngagementReady       EngagementStatus = "Ready"
	EngagementCompleted   EngagementStatus = "Completed"
	EngagementCancelled   EngagementStatus = "Cancelled"
)

// IsValidEngagementStatus reports whether s is a known status (the column is an enum).
func IsValidEngagementStatus(s EngagementStatus) bool {
	switch s {
	case EngagementPlanned, EngagementNegotiation, EngagementBooked, EngagementDPPaid,
		EngagementInProgress, EngagementFullyPaid, EngagementReady, EngagementCompleted, EngagementCancelled:
		return true
	}
	return false
}

// SupplierKind is what a project supplier is in the directory.
type SupplierKind string

const (
	SupplierVendor SupplierKind = "vendor"
	SupplierVenue  SupplierKind = "venue"
)

// Payer is who pays a supplier ("Dibayar oleh"): the WO out of the contract, or
// the client directly, outside it (PLAN §3.1, #12).
type Payer string

const (
	PayerWO     Payer = "WO"
	PayerClient Payer = "Client"
)

// IsValidPayer reports whether p is one of the two payers.
func IsValidPayer(p Payer) bool { return p == PayerWO || p == PayerClient }

// ProjectSupplier is one vendor or venue working on a project ("Vendor &
// venue"). Its name, category and (for a venue) capacities are copies taken
// when it was chosen: the directory row may change or be deleted for good later
// (ADR-0018), the project keeps what was agreed.
type ProjectSupplier struct {
	ID         int64
	TenantID   int64
	ProjectID  int64
	Kind       SupplierKind
	SupplierID int64
	Name       string
	CategoryID int64 // 0 = none (a venue has no vendor category)
	PaidBy     Payer
	// ShowName only means something for a vendor the WO pays: the client then
	// sees the service ("Katering"), not who provides it, unless the WO opts in.
	// A venue and a supplier the client pays are always named (ClientSeesName).
	ShowName         bool
	CapacityStanding *int
	CapacitySeated   *int
	// AgreedCost is "Biaya rekanan": what the WO pays this supplier. Only for a
	// supplier the WO pays, optional, and only for finance.margin.view (§4.3).
	AgreedCost       *int64
	EngagementStatus EngagementStatus
	BookingDate      *time.Time
	DueDate          *time.Time
	PICStaffID       int64 // 0 = none
	Scope            string
	Notes            string
	// DeletedAt is set while the supplier waits in the "deleted" state (soft
	// delete, ADR-0034). Its steps, payments and kendala are hidden with it.
	DeletedAt *time.Time
	UpdatedAt time.Time
}

// ClientSeesName reports whether the client may see who this supplier is
// (PLAN §4.4): a venue and a supplier the client pays always; a vendor the WO
// pays only when the WO chose to show it.
func (s ProjectSupplier) ClientSeesName() bool {
	return s.Kind == SupplierVenue || s.PaidBy == PayerClient || s.ShowName
}

// IsUnconfirmed: not booked yet (still Planned or in Negotiation).
func (s ProjectSupplier) IsUnconfirmed() bool {
	return s.EngagementStatus == EngagementPlanned || s.EngagementStatus == EngagementNegotiation
}

// IsActive: not cancelled.
func (s ProjectSupplier) IsActive() bool { return s.EngagementStatus != EngagementCancelled }

// Capacity is the venue's capacity for the check (domain-neutral numbers).
func (s ProjectSupplier) Capacity(l Layout) *int {
	if l == LayoutSeated {
		return s.CapacitySeated
	}
	return s.CapacityStanding
}

// PaidByLockMessage explains why "Dibayar oleh" can no longer change: payments
// or proof files already hang off the supplier (PLAN §3.1). Empty when it may.
func PaidByLockMessage(paidBy Payer, payments, files int) string {
	who := "WO"
	if paidBy == PayerClient {
		who = "Klien langsung"
	}
	switch {
	case payments > 0:
		return fmt.Sprintf("%s. Terkunci karena sudah ada %d pembayaran. Hapus pembayaran itu untuk mengubah.", who, payments)
	case files > 0:
		return fmt.Sprintf("%s. Terkunci karena sudah ada %d bukti. Hapus bukti itu untuk mengubah.", who, files)
	}
	return ""
}

// SupplierHistoryRow is one project a directory vendor or venue works on — the
// directory's "Lihat proyek" (served to `vendors` through the contract).
type SupplierHistoryRow struct {
	ProjectID        int64
	ProjectName      string
	EventDate        time.Time
	Location         string
	EngagementStatus EngagementStatus
}

// VendorMilestone is one progress step of a supplier ("Tahapan vendor").
type VendorMilestone struct {
	ID                int64
	ProjectSupplierID int64
	SortOrder         int
	Name              string
	Description       string
	Status            MilestoneStatus
	TargetDate        time.Time
	CompletedDate     *time.Time
	PICStaffID        int64
	Notes             string
	UpdatedAt         time.Time
}

// SupplierAttachments is what already hangs off a project supplier.
type SupplierAttachments struct {
	Payments   int // live payments to it (either payer)
	Files      int // live proof files of it, its payments and its steps
	Milestones int
	Issues     int
	BillTerms  int // live bills of a supplier the client pays (Paket WO Tahap 4)
}

// Empty: nothing hangs off it, so it can be removed for good.
func (a SupplierAttachments) Empty() bool {
	return a.Payments == 0 && a.Files == 0 && a.Milestones == 0 && a.Issues == 0 && a.BillTerms == 0
}
