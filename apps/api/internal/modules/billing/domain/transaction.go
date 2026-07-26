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
	// StatusCancelled is a self-service charge the Owner explicitly gave up
	// on via "Batalkan" — distinct from StatusExpired (which means the
	// charge timed out on its own without anyone acting). Tripay itself has
	// no cancel/void API, so this is bookkeeping on our side only: the
	// underlying charge is still technically payable at the gateway until
	// its own expiry, but its pendingCharges tracking row is deleted when
	// cancelled, so a late webhook/reconciliation result for it is silently
	// ignored (idempotent no-op) rather than resurrecting the subscription.
	StatusCancelled TransactionStatus = "cancelled"
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
