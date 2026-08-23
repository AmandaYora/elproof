package domain

import "time"

type Plan struct {
	ID             int64
	Name           string
	DurationMonths int
	Price          int64
	Features       []string
	IsActive       bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
	// AppIDs is populated only by List/ListPaginated (the platform_admin
	// path) — nil everywhere else. A non-empty value means this plan is
	// private to those apps (D3); an empty/nil value means it's part of
	// ElProof's own public catalog. See PLAN.md A23/D12.
	AppIDs []string
}
