package domain

import "time"

type TransactionType string

const (
	TransactionNew     TransactionType = "new"
	TransactionRenewal TransactionType = "renewal"
)

type TransactionStatus string

const (
	StatusUnpaid TransactionStatus = "unpaid"
	// StatusPending means a real charge exists at the gateway (Fase 9) and is
	// awaiting webhook confirmation — distinct from StatusUnpaid, which is an
	// invoice that has never had a charge attempt at all (e.g. right after
	// tenant registration).
	StatusPending TransactionStatus = "pending"
	StatusPaid    TransactionStatus = "paid"
	StatusExpired TransactionStatus = "expired"
	// StatusGranted is a manually-activated transaction (Platform Console
	// bypassing payment) — deliberately excluded from paid-revenue reporting.
	StatusGranted TransactionStatus = "granted"
)

type Transaction struct {
	ID               int64
	TenantID         int64
	Type             TransactionType
	Amount           int64
	PaymentMethod    string
	PaymentReference string
	Status           TransactionStatus
	CreatedAt        time.Time
	PaidAt           *time.Time
}
