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
	ID       int64
	TenantID int64
	Type     TransactionType
	// Amount is what was charged; OriginalAmount the plan price before the
	// voucher discount DiscountAmount (equal to Amount without a voucher).
	Amount           int64
	OriginalAmount   int64
	DiscountAmount   int64
	VoucherCode      string
	PaymentMethod    string
	PaymentReference string
	Status           TransactionStatus
	CreatedAt        time.Time
	PaidAt           *time.Time
}

// TransactionFilter narrows the subscription-transaction list.
type TransactionFilter struct {
	Status string
	// Search matches the payment reference or the tenant's name; the name lives
	// in `platform`, so the application layer turns it into TenantIDs.
	Search string
	// TenantIDs are the tenants whose name matched Search (nil when Search is empty).
	TenantIDs []int64
	// From is inclusive, To exclusive (the instant after the last picked day); zero = unbounded.
	From, To time.Time
}
