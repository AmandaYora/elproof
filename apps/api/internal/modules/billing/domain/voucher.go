package domain

import "time"

// DiscountType mirrors subscription_vouchers.discount_type (migration 000074).
type DiscountType string

const (
	DiscountPercent DiscountType = "percent"
	DiscountAmount  DiscountType = "amount"
)

// Voucher is a discount code for a subscription payment (ADR-0041). It
// belongs to exactly one App — the App whose plans it discounts — and may be
// narrowed to one plan of that App (PlanID nil = every plan of the App).
type Voucher struct {
	ID    int64
	Code  string
	AppID string
	// PlanID nil: every plan of AppID. PlanName is read with it (same module).
	PlanID   *int64
	PlanName string
	Type     DiscountType
	// Value is a percentage (1..99) for DiscountPercent, rupiah for DiscountAmount.
	Value int64
	// MaxDiscount caps a percent discount in rupiah (nil = no cap).
	MaxDiscount *int64
	// ValidFrom/ValidUntil are inclusive WIB calendar days "YYYY-MM-DD" ("" = open).
	ValidFrom  string
	ValidUntil string
	// Quota is the total number of uses (nil = unlimited).
	Quota           *int64
	OncePerCustomer bool
	IsActive        bool
	// UsedCount and ReservedCount are read from the redemptions: paid uses, and
	// uses whose charge is still pending (both count against Quota).
	UsedCount     int64
	ReservedCount int64
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// DiscountFor is the discount this voucher gives on a price, in whole rupiah:
// a percentage is rounded down, then capped by MaxDiscount. It does not
// enforce the "never Rp0" rule; the service refuses a discount that reaches
// the whole price.
func (v Voucher) DiscountFor(price int64) int64 {
	if price <= 0 {
		return 0
	}
	var d int64
	switch v.Type {
	case DiscountPercent:
		d = price * v.Value / 100
		if v.MaxDiscount != nil && d > *v.MaxDiscount {
			d = *v.MaxDiscount
		}
	case DiscountAmount:
		d = v.Value
	}
	if d < 0 {
		return 0
	}
	return d
}

// VoucherState is a voucher's standing on a given day, for lists.
type VoucherState string

const (
	VoucherStateActive    VoucherState = "active"
	VoucherStateInactive  VoucherState = "inactive"
	VoucherStateScheduled VoucherState = "scheduled"
	VoucherStateExpired   VoucherState = "expired"
	VoucherStateExhausted VoucherState = "exhausted"
)

// StateOn reports the voucher's state on day (a "YYYY-MM-DD" WIB date).
func (v Voucher) StateOn(day string) VoucherState {
	switch {
	case !v.IsActive:
		return VoucherStateInactive
	case v.ValidUntil != "" && day > v.ValidUntil:
		return VoucherStateExpired
	case v.ValidFrom != "" && day < v.ValidFrom:
		return VoucherStateScheduled
	case v.Quota != nil && v.UsedCount+v.ReservedCount >= *v.Quota:
		return VoucherStateExhausted
	}
	return VoucherStateActive
}

// CustomerType mirrors subscription_voucher_redemptions.customer_type.
type CustomerType string

const (
	CustomerTenant CustomerType = "tenant"
	CustomerApp    CustomerType = "app"
)

// Customer is who pays a subscription: a WO tenant (Ref = tenant id) or an
// external App (Ref = app_id) — the subscriber the ledgers already key on.
type Customer struct {
	Type CustomerType
	Ref  string
}

// RedemptionStatus mirrors subscription_voucher_redemptions.status.
type RedemptionStatus string

const (
	RedemptionReserved RedemptionStatus = "reserved"
	RedemptionUsed     RedemptionStatus = "used"
	RedemptionReleased RedemptionStatus = "released"
)

// VoucherRedemption is one use of a voucher on one subscription charge.
type VoucherRedemption struct {
	ID             int64
	VoucherID      int64
	VoucherCode    string
	OrderRef       string
	Customer       Customer
	PlanID         int64
	OriginalAmount int64
	DiscountAmount int64
	FinalAmount    int64
	Status         RedemptionStatus
	CreatedAt      time.Time
	ResolvedAt     *time.Time
}

// VoucherFilter narrows the voucher list.
type VoucherFilter struct {
	Search string // matches the code
	AppID  string
}
