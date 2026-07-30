package application

import (
	"context"
	"encoding/base64"
	"errors"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	billingcontracts "elproof/internal/modules/billing/contracts"
	identitycontracts "elproof/internal/modules/identity/contracts"
	paymentcontracts "elproof/internal/modules/payment/contracts"
	"elproof/internal/modules/platform/domain"
	staffcontracts "elproof/internal/modules/staff/contracts"
	vendorscontracts "elproof/internal/modules/vendors/contracts"
	"elproof/internal/shared/apperror"
	"elproof/internal/shared/compress"
	"elproof/internal/shared/logger"
	"elproof/internal/shared/pagination"
	"elproof/internal/shared/validator"
)

type TenantRepository interface {
	List(ctx context.Context) ([]domain.Tenant, error)
	ListPaginated(ctx context.Context, params pagination.Params, search, status string) ([]domain.Tenant, int64, error)
	FindByID(ctx context.Context, id int64) (*domain.Tenant, error)
	FindByDomain(ctx context.Context, host string) (*domain.Tenant, error)
	Create(ctx context.Context, tenant *domain.Tenant) error
	Update(ctx context.Context, tenant *domain.Tenant) error
	UpdateLogo(ctx context.Context, id int64, logoStoragePath *string) error
	SetSuspended(ctx context.Context, id int64, suspended bool) error
	SetCredentialResetAt(ctx context.Context, id int64, when time.Time) error
	UpdateSubscription(ctx context.Context, id int64, planID int64, status domain.SubscriptionStatus, expiresAt time.Time) error
}

// ObjectStorage is the narrow slice of internal/shared/storage.Client this
// service depends on for tenant logo uploads — package-local (like
// projects/application.ObjectStorage) so this module never imports another
// module's application layer, per the modular-monolith boundary.
type ObjectStorage interface {
	Save(ctx context.Context, key string, data []byte, contentType string) (string, error)
	Open(ctx context.Context, key string) (io.ReadCloser, error)
}

// PendingChargeRepository backs the Fase 9 self-service payment flow — see
// domain.PendingCharge.
type PendingChargeRepository interface {
	Create(ctx context.Context, orderRef string, tenantID, planID int64) error
	FindByOrderRef(ctx context.Context, orderRef string) (*domain.PendingCharge, error)
	FindByTenant(ctx context.Context, tenantID int64) ([]domain.PendingCharge, error)
	Delete(ctx context.Context, orderRef string) error
}

type TenantService struct {
	repo           TenantRepository
	pendingCharges PendingChargeRepository
	staff          staffcontracts.Contracts
	identity       identitycontracts.Contracts
	billing        billingcontracts.Contracts
	payment        paymentcontracts.Client
	vendors        vendorscontracts.Contracts
	storage        ObjectStorage
	buildKey       func(tenantID, projectID, category, filename string) string
}

func NewTenantService(
	repo TenantRepository,
	pendingCharges PendingChargeRepository,
	staff staffcontracts.Contracts,
	identity identitycontracts.Contracts,
	billing billingcontracts.Contracts,
	payment paymentcontracts.Client,
	storage ObjectStorage,
	buildKey func(string, string, string, string) string,
) *TenantService {
	return &TenantService{
		repo: repo, pendingCharges: pendingCharges,
		staff: staff, identity: identity, billing: billing, payment: payment,
		storage: storage, buildKey: buildKey,
	}
}

// SetVendors completes two-phase wiring with the vendors module — main.go
// builds `platform` before `vendors` (which itself depends on `projects`), so
// this can't be a NewTenantService constructor argument. Same idiom as
// projects' SetClientAccessResolver for the clients<->projects cycle.
func (s *TenantService) SetVendors(vendors vendorscontracts.Contracts) {
	s.vendors = vendors
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

// GetBrandingByDomain resolves a tenant from an incoming request's Host
// header (ADR-0015) — the pre-auth counterpart to Get, used only by the
// public branding/logo endpoints, never by anything JWT-scoped.
func (s *TenantService) GetBrandingByDomain(ctx context.Context, host string) (*domain.Tenant, error) {
	tenant, err := s.repo.FindByDomain(ctx, host)
	if err != nil {
		return nil, err
	}
	if tenant == nil {
		return nil, apperror.NotFound("Domain tidak terdaftar")
	}
	return tenant, nil
}

type RegisterTenantInput struct {
	BusinessName string
	OwnerName    string
	Username     string
	Email        string
	Phone        string
	City         string
	Password     string
}

type RegisterTenantResult struct {
	Tenant   domain.Tenant
	Username string
}

// Register orchestrates three modules in one flow: creates the tenant row
// (platform), creates the Owner staff row (staff), and creates the Owner's
// login credential (identity) — see ADR-0008. Each module still only writes
// its own tables. The tenant starts unbound to any plan (PlanID nil,
// StatusPendingPayment) — no transaction is recorded here, since none exists
// yet: the Owner hasn't chosen a plan or paid, and the Platform Console
// hasn't manually activated one either. The first transaction row this
// tenant ever gets is whichever real subscription event happens first (Pay
// or ActivateSubscription), so there is never a placeholder row left
// dangling forever if the tenant is instead activated manually.
func (s *TenantService) Register(ctx context.Context, input RegisterTenantInput) (*RegisterTenantResult, error) {
	if err := validator.Username(input.Username); err != nil {
		return nil, err
	}

	username := input.Username

	tenant := &domain.Tenant{
		BusinessName:       input.BusinessName,
		OwnerName:          input.OwnerName,
		Username:           username,
		Email:              input.Email,
		Phone:              input.Phone,
		City:               input.City,
		JoinedAt:           time.Now(),
		PlanID:             nil,
		SubscriptionStatus: domain.StatusPendingPayment,
		BrandColorPreset:   domain.DefaultBrandColorPreset,
	}
	if err := s.repo.Create(ctx, tenant); err != nil {
		return nil, err
	}

	if err := s.vendors.SeedDefaultCategories(ctx, tenant.ID); err != nil {
		return nil, err
	}

	ownerResult, err := s.staff.CreateOwner(ctx, staffcontracts.CreateOwnerInput{
		TenantID: tenant.ID, Name: input.OwnerName, Email: input.Email, Phone: input.Phone, Username: username,
	})
	if err != nil {
		return nil, err
	}

	if err := s.identity.CreateCredential(ctx, identitycontracts.CreateCredentialInput{
		TenantID:      &tenant.ID,
		PrincipalType: identitycontracts.PrincipalStaff,
		PrincipalID:   ownerResult.StaffID,
		Username:      username,
		Email:         input.Email,
		Password:      input.Password,
		Role:          "Owner",
		DisplayName:   input.OwnerName,
	}); err != nil {
		return nil, err
	}

	return &RegisterTenantResult{Tenant: *tenant, Username: username}, nil
}

type UpdateTenantInput struct {
	BusinessName     string
	OwnerName        string
	Email            string
	Phone            string
	City             string
	BrandColorPreset string
	// CustomDomain is a pointer, unlike every other field on this input: nil
	// means the caller's JSON body omitted the key entirely (e.g. a stale
	// cached frontend bundle that predates this field, same "don't touch it"
	// concern BrandColorPreset's own comment below addresses) and the
	// tenant's existing domain is left untouched. A non-nil pointer means the
	// key was present — "" explicitly clears it to NULL, anything else is
	// normalized/validated and set.
	CustomDomain *string
}

func (s *TenantService) Update(ctx context.Context, id int64, input UpdateTenantInput) (*domain.Tenant, error) {
	tenant, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	// BrandColorPreset is optional on this otherwise-full-replace update: an
	// empty value (an older cached frontend bundle mid-deploy, calling this
	// endpoint without knowing about branding yet) keeps the tenant's current
	// preset rather than rejecting the whole edit; a non-empty, invalid value
	// is still rejected, since presets are a fixed enum, not free text.
	if input.BrandColorPreset != "" {
		if !domain.IsValidBrandColorPreset(input.BrandColorPreset) {
			return nil, apperror.Validation("Warna brand tidak valid", map[string][]string{
				"brandColorPreset": {"Pilih salah satu preset warna yang tersedia"},
			})
		}
		tenant.BrandColorPreset = input.BrandColorPreset
	}
	// CustomDomain (ADR-0015): nil means the key was absent from the request
	// — leave tenant.CustomDomain exactly as Get() returned it. Only a
	// present key can change it: "" clears to NULL, anything else is
	// normalized + validated then set.
	if input.CustomDomain != nil {
		if *input.CustomDomain != "" {
			normalized := strings.ToLower(strings.TrimSpace(*input.CustomDomain))
			if err := validator.CustomDomain(normalized); err != nil {
				return nil, err
			}
			tenant.CustomDomain = &normalized
		} else {
			tenant.CustomDomain = nil
		}
	}
	tenant.BusinessName = input.BusinessName
	tenant.OwnerName = input.OwnerName
	tenant.Email = input.Email
	tenant.Phone = input.Phone
	tenant.City = input.City
	if err := s.repo.Update(ctx, tenant); err != nil {
		if errors.Is(err, domain.ErrDuplicateCustomDomain) {
			return nil, apperror.Validation("Domain kustom sudah digunakan tenant lain", map[string][]string{
				"customDomain": {"Domain ini sudah dipakai tenant lain"},
			})
		}
		return nil, err
	}
	return tenant, nil
}

// maxLogoDecodedSize caps a tenant logo well below evidence's 15MB — this is
// a small header/sidebar image, not a document.
const maxLogoDecodedSize = 2 * 1024 * 1024

var allowedLogoMimeTypes = map[string]bool{
	"image/png":  true,
	"image/jpeg": true,
	"image/webp": true,
}

type UploadLogoInput struct {
	FileName   string
	MimeType   string
	Base64Data string
}

// UploadLogo stores a tenant's logo in object storage (never inline in the
// DB) and records only its key. SVG is deliberately not in
// allowedLogoMimeTypes — sanitizing embedded <script>/event-handler risk
// properly is out of scope for this feature (see PLAN.md §7).
func (s *TenantService) UploadLogo(ctx context.Context, tenantID int64, input UploadLogoInput) (*domain.Tenant, error) {
	if !allowedLogoMimeTypes[input.MimeType] {
		return nil, apperror.Validation("Format logo tidak didukung", map[string][]string{
			"mimeType": {"Gunakan PNG, JPEG, atau WebP"},
		})
	}
	tenant, err := s.Get(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	decoded, err := base64.StdEncoding.DecodeString(input.Base64Data)
	if err != nil {
		return nil, apperror.Validation("Data logo tidak valid", map[string][]string{"base64Data": {"Gagal membaca data file"}})
	}
	if len(decoded) == 0 {
		return nil, apperror.Validation("File logo kosong", map[string][]string{"base64Data": {"File tidak boleh kosong"}})
	}
	if len(decoded) > maxLogoDecodedSize {
		return nil, apperror.Validation("Ukuran logo terlalu besar", map[string][]string{"base64Data": {"Maksimal 2 MB"}})
	}

	// Same backend re-compression pass evidence uploads get (ADR-0010) —
	// authoritative regardless of what the frontend already did.
	compressed, err := compress.Image(decoded, input.MimeType)
	if err != nil {
		return nil, apperror.Internal("Gagal memproses logo")
	}

	key := s.buildKey(
		strconv.FormatInt(tenantID, 10), "0", "branding",
		uuid.NewString()+"-"+sanitizeLogoFileName(input.FileName),
	)
	if _, err := s.storage.Save(ctx, key, compressed, input.MimeType); err != nil {
		return nil, apperror.Internal("Gagal mengunggah logo ke object storage")
	}

	if err := s.repo.UpdateLogo(ctx, tenantID, &key); err != nil {
		return nil, err
	}
	tenant.LogoStoragePath = &key
	return tenant, nil
}

// DownloadLogo streams a tenant's stored logo back — same byte-proxy shape
// as evidence.Download, since object storage requires backend auth and can't
// be linked to directly from the browser.
func (s *TenantService) DownloadLogo(ctx context.Context, tenantID int64) (io.ReadCloser, error) {
	tenant, err := s.Get(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if tenant.LogoStoragePath == nil {
		return nil, apperror.NotFound("Tenant belum memiliki logo")
	}
	reader, err := s.storage.Open(ctx, *tenant.LogoStoragePath)
	if err != nil {
		return nil, apperror.Internal("Gagal mengambil logo dari object storage")
	}
	return reader, nil
}

var unsafeLogoFileNameChars = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

func sanitizeLogoFileName(name string) string {
	cleaned := unsafeLogoFileNameChars.ReplaceAllString(name, "-")
	if cleaned == "" {
		return "logo"
	}
	return cleaned
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
//
// If the Owner's own self-service "Bayar Sekarang" charge is still pending
// for this tenant, it's resolved here first: expired in the transaction
// ledger and cleared from pendingCharges tracking. Without this, that row
// stayed stuck at "Menunggu Pembayaran" until the payment module's
// reconciliation sweep eventually caught up with the gateway (its own
// natural charge expiry, potentially hours later) — and if the Owner
// actually completed that old payment in the meantime, the late webhook
// would still fire ApplyWebhookEvent and call UpdateSubscription a second
// time, silently double-extending the expiry this call is about to set
// (computeNewExpiry stacks on top of whatever expiry is already there).
func (s *TenantService) ActivateSubscription(ctx context.Context, tenantID int64, planID int64) (*domain.Tenant, error) {
	tenant, err := s.Get(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	plan, err := s.billing.GetPlan(ctx, planID)
	if err != nil {
		return nil, err
	}

	pending, err := s.pendingCharges.FindByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	for _, p := range pending {
		if err := s.billing.UpdateTransactionStatus(ctx, p.OrderRef, billingcontracts.StatusExpired); err != nil {
			return nil, err
		}
		if err := s.pendingCharges.Delete(ctx, p.OrderRef); err != nil {
			return nil, err
		}
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
//
// Guard: rejects a second charge while one is already pending for this
// tenant (409), instead of silently starting a second, independent charge —
// this is real money moved at an external gateway (unlike ActivateSubscription,
// an internal admin action), so the guard lives here in the backend rather
// than relying only on the frontend disabling its button while a request is
// in flight. If both charges were ever paid, the tenant would be billed
// twice and the subscription would be double-extended (computeNewExpiry
// stacks on top of whatever expiry is already there).
func (s *TenantService) Pay(ctx context.Context, tenantID int64, planID int64) (*paymentcontracts.ChargeResult, error) {
	tenant, err := s.Get(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	plan, err := s.billing.GetPlan(ctx, planID)
	if err != nil {
		return nil, err
	}

	existing, err := s.pendingCharges.FindByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if len(existing) > 0 {
		return nil, apperror.Conflict("Anda masih memiliki tagihan yang belum diselesaikan. Selesaikan atau tunggu tagihan tersebut kedaluwarsa sebelum membuat tagihan baru.")
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

// GetPendingCharge lets the Owner re-view a still-pending self-service
// charge after closing its QR modal (accidentally or otherwise) — instead of
// the charge's display fields (QR image, pay code, checkout URL) being lost
// forever, since they were never persisted anywhere beyond Pay's one-time
// response. Re-fetches them live from the gateway via CheckStatusForApp
// rather than caching a copy that could go stale. Returns nil (not an error)
// when there's nothing pending — this is a normal, common state.
func (s *TenantService) GetPendingCharge(ctx context.Context, tenantID int64) (*paymentcontracts.ChargeResult, error) {
	pending, err := s.pendingCharges.FindByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if len(pending) == 0 {
		return nil, nil
	}
	p := pending[0]
	charge, err := s.payment.CheckStatusForApp(ctx, paymentcontracts.InternalAppBilling, p.OrderRef)
	if err != nil {
		// The live gateway call can fail for reasons that have nothing to do
		// with whether the charge is still genuinely pending (a rotated API
		// key, a Tripay outage, a transient network error) — letting that
		// failure bubble up as an opaque error here would silently hide the
		// pending-charge banner (the frontend fetch has no visible failure
		// state) and strand the Owner exactly like the bug this endpoint
		// exists to fix: unable to see or cancel their pending charge. Log
		// for diagnosis and fall back to what we already know locally — no
		// QR/pay code/checkout URL (only the gateway has those), but enough
		// for the Owner to see it's pending and use "Batalkan".
		logger.Error("gagal memuat status live charge %s dari gateway: %v", p.OrderRef, err)
		plan, planErr := s.billing.GetPlan(ctx, p.PlanID)
		amount := int64(0)
		if planErr == nil && plan != nil {
			amount = plan.Price
		}
		return &paymentcontracts.ChargeResult{OrderRef: p.OrderRef, Amount: amount, Status: "pending"}, nil
	}
	return charge, nil
}

// CancelPendingCharge is the Owner's "Batalkan" action — gives up on a
// pending self-service charge without waiting for it to expire on its own,
// freeing Pay() to start a fresh one immediately. Tripay has no cancel/void
// API, so this is bookkeeping on our side only: the transaction is marked
// StatusCancelled (distinct from StatusExpired, which means it timed out
// unattended) and its pendingCharges tracking row is deleted. If the Owner
// somehow still completes payment on the cancelled QR afterward, the late
// webhook/reconciliation result finds no tracking row for that order_ref and
// is silently ignored (see ApplyWebhookEvent's nil-pending branch) rather
// than resurrecting a subscription the Owner explicitly walked away from.
func (s *TenantService) CancelPendingCharge(ctx context.Context, tenantID int64) error {
	pending, err := s.pendingCharges.FindByTenant(ctx, tenantID)
	if err != nil {
		return err
	}
	if len(pending) == 0 {
		return apperror.NotFound("Tidak ada tagihan yang menggantung untuk dibatalkan")
	}
	for _, p := range pending {
		if err := s.billing.UpdateTransactionStatus(ctx, p.OrderRef, billingcontracts.StatusCancelled); err != nil {
			return err
		}
		if err := s.pendingCharges.Delete(ctx, p.OrderRef); err != nil {
			return err
		}
	}
	return nil
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
