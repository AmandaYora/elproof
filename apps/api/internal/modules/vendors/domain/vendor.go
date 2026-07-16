package domain

import "time"

type Vendor struct {
	ID         int64
	TenantID   int64
	CategoryID int64
	Name       string
	PICName    string
	Phone      string
	Email      string
	Address    string
	Notes      string
	IsActive   bool
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
