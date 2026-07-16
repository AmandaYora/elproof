package domain

import "time"

type VendorCategory struct {
	ID          int64
	TenantID    int64
	Name        string
	Description string
	IsActive    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
