package domain

import "time"

// Plan is one subscription plan. Every plan belongs to exactly one App of the
// `payment` registry (AppID, a primitive cross-module ID — no FK): the App
// that sells it. ElProof's own catalog for WO tenants is simply the plans of
// its internal App (paymentcontracts.InternalAppBilling). See ADR-0040.
type Plan struct {
	ID             int64
	AppID          string
	Name           string
	DurationMonths int
	Price          int64
	Features       []string
	IsActive       bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
