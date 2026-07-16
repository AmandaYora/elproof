package application

import (
	"context"
	"strconv"
	"strings"
	"time"

	billingcontracts "elproof/internal/modules/billing/contracts"
	identitycontracts "elproof/internal/modules/identity/contracts"
	paymentcontracts "elproof/internal/modules/payment/contracts"
	"elproof/internal/modules/platform/domain"
	staffcontracts "elproof/internal/modules/staff/contracts"
	"elproof/internal/shared/apperror"
	"elproof/internal/shared/pagination"
)

type TenantRepository interface {
	List(ctx context.Context) ([]domain.Tenant, error)
	ListPaginated(ctx context.Context, params pagination.Params, search, status string) ([]domain.Tenant, int64, error)
	FindByID(ctx context.Context, id int64) (*domain.Tenant, error)
	Create(ctx context.Context, tenant *domain.Tenant) error
	Update(ctx context.Context, tenant *domain.Tenant) error
	SetSuspended(ctx context.Context, id int64, suspended bool) error
	SetCredentialResetAt(ctx context.Context, id int64, when time.Time) error
	UpdateSubscription(ctx context.Context, id int64, planID int64, status domain.SubscriptionStatus, expiresAt time.Time) error
}

// PendingChargeRepository backs the Fase 9 self-service payment flow — see
// domain.PendingCharge.
type PendingChargeRepository interface {
	Create(ctx context.Context, orderRef string, tenantID, planID int64) error
	FindByOrderRef(ctx context.Context, orderRef string) (*domain.PendingCharge, error)
	Delete(ctx context.Context, orderRef string) error
}

type TenantService struct {
	repo           TenantRepository
	pendingCharges PendingChargeRepository
	staff          staffcontracts.Contracts
	identity       identitycontracts.Contracts
	billing        billingcontracts.Contracts
	payment        paymentcontracts.Client
}

func NewTenantService(
	repo TenantRepository,
	pendingCharges PendingChargeRepository,
	staff staffcontracts.Contracts,
	identity identitycontracts.Contracts,
	billing billingcontracts.Contracts,
	payment paymentcontracts.Client,
) *TenantService {
	return &TenantService{
		repo: repo, pendingCharges: pendingCharges,
		staff: staff, identity: identity, billing: billing, payment: payment,
	}
}

func (s *TenantService) List(ctx context.Context) ([]domain.Tenant, error) {
	return s.repo.List(ctx)
}

func (s *TenantService) ListPaginated(ctx context.Context, params pagination.Params, search, status string) ([]domain.Tenant, int64, error) {
	return s.repo.ListPaginated(ctx, params, search, status)
}

func (s *TenantService) Get(ctx context.Context, id int64) (*domain.Tenant, error) {
	tenant, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if tenant == nil {
		return nil, apperror.NotFound("Tenant tidak ditemukan")
	}
	return tenant, nil
}

func deriveUsername(email string) string {
	local, _, _ := strings.Cut(email, "@")
	return local
}

type RegisterTenantInput struct {
	BusinessName string
	OwnerName    string
	Email        string
	Phone        string
	City         string
	PlanID       int64
	Password     string
}

type RegisterTenantResult struct {
	Tenant   domain.Tenant
	Username string
}

// Register orchestrates three modules in one flow: creates the tenant row
// (platform), creates the Owner staff row (staff), creates the Owner's login
// credential (identity), and records an initial "unpaid" transaction
// (billing) — see ADR-0008. Each module still only writes its own tables.
func (s *TenantService) Register(ctx context.Context, input RegisterTenantInput) (*RegisterTenantResult, error) {
	plan, err := s.billing.GetPlan(ctx, input.PlanID)
	if err != nil {
		return nil, err
	}

	username := deriveUsername(input.Email)
	planID := input.PlanID

	tenant := &domain.Tenant{
		BusinessName:       input.BusinessName,
		OwnerName:          input.OwnerName,
		Username:           username,
		Email:              input.Email,
		Phone:              input.Phone,
		City:               input.City,
		JoinedAt:           time.Now(),
		PlanID:             &planID,
		SubscriptionStatus: domain.StatusPendingPayment,
	}
	if err := s.repo.Create(ctx, tenant); err != nil {
		return nil, err
	}

	ownerResult, err := s.staff.CreateOwner(ctx, staffcontracts.CreateOwnerInput{
		TenantID: tenant.ID, Name: input.OwnerName, Email: input.Email, Phone: input.Phone,
	})
	if err != nil {
		return nil, err
	}

	if err := s.identity.CreateCredential(ctx, identitycontracts.CreateCredentialInput{
		TenantID:      &tenant.ID,
		PrincipalType: identitycontracts.PrincipalStaff,
		PrincipalID:   ownerResult.StaffID,
		Username:      username,
		Password:      input.Password,
		Role:          "Owner",
		DisplayName:   input.OwnerName,
	}); err != nil {
		return nil, err
	}

	if err := s.billing.RecordTransaction(ctx, billingcontracts.RecordTransactionInput{
		TenantID: tenant.ID, Type: billingcontracts.TransactionNew, Amount: plan.Price,
		PaymentMethod: "-", PaymentReference: generatePaymentReference(), Status: billingcontracts.StatusUnpaid,
	}); err != nil {
		return nil, err
	}

	return &RegisterTenantResult{Tenant: *tenant, Username: username}, nil
}

type UpdateTenantInput struct {
	BusinessName string
	OwnerName    string
	Email        string
	Phone        string
	City         string
}

func (s *TenantService) Update(ctx context.Context, id int64, input UpdateTenantInput) (*domain.Tenant, error) {
	tenant, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	tenant.BusinessName = input.BusinessName
	tenant.OwnerName = input.OwnerName
	tenant.Email = input.Email
	tenant.Phone = input.Phone
	tenant.City = input.City
	if err := s.repo.Update(ctx, tenant); err != nil {
		return nil, err
	}
	return tenant, nil
}

func (s *TenantService) SetSuspended(ctx context.Context, id int64, suspended bool) (*domain.Tenant, error) {
	if _, err := s.Get(ctx, id); err != nil {
		return nil, err
	}
	if err := s.repo.SetSuspended(ctx, id, suspended); err != nil {
		return nil, err
	}
	return s.Get(ctx, id)
}

type ResetCredentialResult struct {
	Username string
}

func (s *TenantService) ResetCredential(ctx context.Context, id int64, newPassword string) (*ResetCredentialResult, error) {
	tenant, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	// The Owner's principal_id is the staff row created at registration —
	// identity resolves it by (principal_type, principal_id); platform never
	// stores or looks up the staff_members.id itself (no cross-module FK).
	// ResetPassword is looked up by username since that is the value both
	// `tenants` and `credentials` share as a plain, unenforced value.
	if err := s.identity.ResetPasswordByUsername(ctx, tenant.Username, newPassword); err != nil {
		return nil, err
	}
	if err := s.repo.SetCredentialResetAt(ctx, id, time.Now()); err != nil {
		return nil, err
	}
	return &ResetCredentialResult{Username: tenant.Username}, nil
}

// ActivateSubscription is the Platform Console's manual-activation flow —
// bypasses payment entirely; recorded as a "granted" transaction so it never
// counts as real revenue. See ADR (subscription bypass) and docs/DB_SCHEMA.md.
// Unlike Pay, this never touches the payment gateway at all — it activates
// synchronously, in this same call, exactly as before Fase 9.
func (s *TenantService) ActivateSubscription(ctx context.Context, tenantID int64, planID int64) (*domain.Tenant, error) {
	tenant, err := s.Get(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	plan, err := s.billing.GetPlan(ctx, planID)
	if err != nil {
		return nil, err
	}
	if err := s.billing.RecordTransaction(ctx, billingcontracts.RecordTransactionInput{
		TenantID: tenantID, Type: txTypeFor(tenant), Amount: plan.Price,
		PaymentMethod: "Aktivasi Manual (Super Admin)", PaymentReference: generateGrantReference(),
		Status: billingcontracts.StatusGranted,
	}); err != nil {
		return nil, err
	}
	if err := s.repo.UpdateSubscription(ctx, tenantID, planID, domain.StatusActive, computeNewExpiry(tenant, plan)); err != nil {
		return nil, err
	}
	return s.Get(ctx, tenantID)
}

// Pay is the tenant Owner's own self-service "Bayar Sekarang" action (Fase
// 9) — creates a real charge at the configured gateway (QRIS by default) and
// returns it for the frontend to render; the subscription is NOT activated
// here. Activation only happens once the gateway's webhook confirms payment
// (see ApplyWebhookEvent below) — this function only ever gets the charge
// started.
func (s *TenantService) Pay(ctx context.Context, tenantID int64, planID int64) (*paymentcontracts.ChargeResult, error) {
	tenant, err := s.Get(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	plan, err := s.billing.GetPlan(ctx, planID)
	if err != nil {
		return nil, err
	}

	orderRef := generatePaymentReference()
	charge, err := s.payment.CreateCharge(ctx, paymentcontracts.InternalAppBilling, strconv.FormatInt(tenantID, 10), orderRef, plan.Price)
	if err != nil {
		return nil, err
	}

	if err := s.billing.RecordTransaction(ctx, billingcontracts.RecordTransactionInput{
		TenantID: tenantID, Type: txTypeFor(tenant), Amount: plan.Price,
		PaymentMethod: charge.Channel, PaymentReference: orderRef, Status: billingcontracts.StatusPending,
	}); err != nil {
		return nil, err
	}
	if err := s.pendingCharges.Create(ctx, orderRef, tenantID, planID); err != nil {
		return nil, err
	}

	return charge, nil
}

// ApplyWebhookEvent is `platform`'s implementation of
// `paymentcontracts.WebhookConsumer` — registered with the payment module's
// Dispatcher as the consumer for `paymentcontracts.InternalAppBilling` (see
// main.go). Called in-process, same request, when the gateway's webhook
// confirms (or fails) a charge created by Pay above.
func (s *TenantService) ApplyWebhookEvent(ctx context.Context, orderRef string, event paymentcontracts.WebhookEvent) error {
	pending, err := s.pendingCharges.FindByOrderRef(ctx, orderRef)
	if err != nil {
		return err
	}
	if pending == nil {
		return nil // unknown or already-consumed order_ref — idempotent no-op
	}

	if !event.Paid {
		if err := s.billing.UpdateTransactionStatus(ctx, orderRef, billingcontracts.StatusExpired); err != nil {
			return err
		}
		return s.pendingCharges.Delete(ctx, orderRef)
	}

	tenant, err := s.Get(ctx, pending.TenantID)
	if err != nil {
		return err
	}
	plan, err := s.billing.GetPlan(ctx, pending.PlanID)
	if err != nil {
		return err
	}

	if err := s.billing.UpdateTransactionStatus(ctx, orderRef, billingcontracts.StatusPaid); err != nil {
		return err
	}
	if err := s.repo.UpdateSubscription(ctx, pending.TenantID, pending.PlanID, domain.StatusActive, computeNewExpiry(tenant, plan)); err != nil {
		return err
	}
	return s.pendingCharges.Delete(ctx, orderRef)
}

func txTypeFor(tenant *domain.Tenant) billingcontracts.TransactionType {
	if tenant.SubscriptionStatus != domain.StatusPendingPayment {
		return billingcontracts.TransactionRenewal
	}
	return billingcontracts.TransactionNew
}

// computeNewExpiry extends from the tenant's existing expiry if their plan
// is still active (renewal stacks on top of remaining time), otherwise from
// now (new subscription, or reactivating a lapsed one).
func computeNewExpiry(tenant *domain.Tenant, plan *billingcontracts.Plan) time.Time {
	wasActive := tenant.SubscriptionStatus == domain.StatusActive || tenant.SubscriptionStatus == domain.StatusExpiringSoon
	base := time.Now()
	if wasActive && tenant.SubscriptionExpiresAt != nil && tenant.SubscriptionExpiresAt.After(base) {
		base = *tenant.SubscriptionExpiresAt
	}
	return base.AddDate(0, plan.DurationMonths, 0)
}

// ParseTenantID converts a JWT tenant-id claim (string) to int64 — used by the
// presentation layer for the self-service "pay" endpoint.
func ParseTenantID(raw string) (int64, error) {
	return strconv.ParseInt(raw, 10, 64)
}
