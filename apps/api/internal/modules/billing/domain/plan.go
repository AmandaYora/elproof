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
}
