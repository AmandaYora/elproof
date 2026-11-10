package application

import (
	"context"
	"encoding/base64"
	"io"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	accesscontracts "elproof/internal/modules/access/contracts"
	billingcontracts "elproof/internal/modules/billing/contracts"
	clientscontracts "elproof/internal/modules/clients/contracts"
	identitycontracts "elproof/internal/modules/identity/contracts"
	paymentcontracts "elproof/internal/modules/payment/contracts"
	"elproof/internal/modules/platform/domain"
	projectscontracts "elproof/internal/modules/projects/contracts"
	staffcontracts "elproof/internal/modules/staff/contracts"
	vendorscontracts "elproof/internal/modules/vendors/contracts"
	"elproof/internal/shared/apperror"
	"elproof/internal/shared/compress"
	"elproof/internal/shared/logger"
	"elproof/internal/shared/pagination"
	"elproof/internal/shared/utils"
)

type TenantRepository interface {
	List(ctx context.Context) ([]domain.Tenant, error)
	ListPaginated(ctx context.Context, params pagination.Params, filter domain.TenantFilter, sort pagination.Sort) ([]domain.Tenant, int64, error)
	// ListSortSpec is the ?sort= whitelist (public name -> fixed SQL) and the default order.
	ListSortSpec() (map[string]pagination.SortColumn, pagination.Sort)
	// ListFiltered returns every match, newest first (CSV export).
	ListFiltered(ctx context.Context, filter domain.TenantFilter) ([]domain.Tenant, error)
	FindByID(ctx context.Context, id int64) (*domain.Tenant, error)
	Create(ctx context.Context, tenant *domain.Tenant) error
	Update(ctx context.Context, tenant *domain.Tenant) error
	UpdateLogo(ctx context.Context, id int64, logoStoragePath *string) error
	SetSuspended(ctx context.Context, id int64, suspended bool) error
	SetCredentialResetAt(ctx context.Context, id int64, when time.Time) error
	UpdateSubscription(ctx context.Context, id int64, planID int64, status domain.SubscriptionStatus, expiresAt time.Time) error
	SetSubscriptionStatus(ctx context.Context, id int64, status domain.SubscriptionStatus) error
	UpdateProfile(ctx context.Context, id int64, businessName, phone, city string) error
	GetBusinessDetails(ctx context.Context, tenantID int64) (domain.BusinessDetails, error)
	SaveBusinessDetails(ctx context.Context, tenantID int64, d domain.BusinessDetails) error
	SetOnboardingDismissed(ctx context.Context, id int64, when time.Time) error
	CreateForSignup(ctx context.Context, tenant *domain.Tenant, signupID int64) (bool, error)
	UpdateUsername(ctx context.Context, id int64, username string) error
	// FreeGrantTenantIDs: tenants whose current paid period was granted for
	// free (compensation, trial extension, internal): active but not revenue.
	FreeGrantTenantIDs(ctx context.Context) (map[int64]bool, error)
	// JoinedSince returns the join time of every tenant that joined at or after since.
	JoinedSince(ctx context.Context, since time.Time) ([]time.Time, error)
	// NamesByIDs resolves business names for the given tenant ids.
	NamesByIDs(ctx context.Context, ids []int64) (map[int64]string, error)
	// SearchIDsByName returns the ids of up to limit tenants whose business name contains text.
	SearchIDsByName(ctx context.Context, text string, limit int) ([]int64, error)
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
// ManualActivationRepository stores the reason/proof of manual grants.
type ManualActivationRepository interface {
	// ApplyGrant updates the tenant and stores the record atomically.
	ApplyGrant(ctx context.Context, a *domain.ManualActivation, status domain.SubscriptionStatus) error
}

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
	projects       projectscontracts.Contracts
	storage        ObjectStorage
	buildKey       func(tenantID, projectID, category, filename string) string
	// access caches each tenant's lifecycle for the per-request access gate
	// (tenant_access.go).
	access      *accessCache
	activations ManualActivationRepository
	// Self-service signup (signup_service.go) and trial caps /
	// onboarding (onboarding_service.go).
	signups     SignupRepository
	signupLocks keyedLocks
	trialLimits TrialLimitsRepository
	clients     clientscontracts.Contracts
	roles       accesscontracts.Contracts
}

// SetAccess completes two-phase wiring with the access module: every new
// workspace gets its built-in roles (ADR-0026).
func (s *TenantService) SetAccess(a accesscontracts.Contracts) {
	s.roles = a
}

// ownerRole seeds a workspace's built-in roles and returns its Owner role.
func (s *TenantService) ownerRole(ctx context.Context, tenantID int64) (accesscontracts.RoleRef, error) {
	if s.roles == nil {
		return accesscontracts.RoleRef{}, apperror.Internal("Peran workspace belum dapat dibuat")
	}
	if err := s.roles.SeedDefaultRoles(ctx, tenantID); err != nil {
		return accesscontracts.RoleRef{}, err
	}
	return s.roles.TemplateRole(ctx, tenantID, accesscontracts.TemplateOwner)
}

// SetManualActivations plugs in the manual-activation record store.
func (s *TenantService) SetManualActivations(repo ManualActivationRepository) {
	s.activations = repo
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
		access: newAccessCache(accessCacheTTL),
	}
}

// SetVendors completes two-phase wiring with the vendors module — main.go
// builds `platform` before `vendors` (which itself depends on `projects`), so
// this can't be a NewTenantService constructor argument. Same idiom as
// projects' SetClientAccessResolver for the clients<->projects cycle.
func (s *TenantService) SetVendors(vendors vendorscontracts.Contracts) {
	s.vendors = vendors
}

// SetProjects completes the same two-phase wiring as SetVendors above, so
// Register() can seed a newly registered tenant's Timeline Default template
// (PLAN.md) via projects/contracts, alongside the existing default vendor
// categories seed.
func (s *TenantService) SetProjects(projects projectscontracts.Contracts) {
	s.projects = projects
}

func (s *TenantService) List(ctx context.Context) ([]domain.Tenant, error) {
	return s.repo.List(ctx)
}

func (s *TenantService) ListPaginated(ctx context.Context, params pagination.Params, filter domain.TenantFilter, sort pagination.Sort) ([]domain.Tenant, int64, error) {
	if sort.Field == "planName" {
		// The plan names belong to billing: rank the plan ids by name and let the
		// SQL order by position in that list. A catalog that cannot be read leaves
		// the plan-id order rather than failing the page.
		plans, err := s.billing.ListPlans(ctx)
		if err != nil {
			logger.Error("tenants list: reading plans to order by name: %v", err)
		}
		sort = sort.With(pagination.RankedCol("plan_id", planIDsByName(plans)))
	}
	return s.repo.ListPaginated(ctx, params, filter, sort)
}

// planIDsByName lists the plan ids in the order of the plan names (ignoring
// case, ties by id): the ranking the planName sort uses.
func planIDsByName(plans []billingcontracts.Plan) []int64 {
	sorted := append([]billingcontracts.Plan(nil), plans...)
	sort.Slice(sorted, func(i, j int) bool {
		a, b := strings.ToLower(sorted[i].Name), strings.ToLower(sorted[j].Name)
		if a == b {
			return sorted[i].ID < sorted[j].ID
		}
		return a < b
	})
	ids := make([]int64, len(sorted))
	for i, p := range sorted {
		ids[i] = p.ID
	}
	return ids
}

// ListSortSpec is the sort whitelist and default of the paginated tenant list.
func (s *TenantService) ListSortSpec() (map[string]pagination.SortColumn, pagination.Sort) {
	return s.repo.ListSortSpec()
}

// ListFiltered returns every workspace matching the filter, for export.
func (s *TenantService) ListFiltered(ctx context.Context, filter domain.TenantFilter) ([]domain.Tenant, error) {
	return s.repo.ListFiltered(ctx, filter)
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

type RegisterTenantInput struct {
	BusinessName string
	OwnerName    string
	// Email is the Google account the Owner must accept the invitation
	// with (Owners sign in with Google only, decision K5).
	Email string
	Phone string
	City  string
}

// OwnerInvitation is a one-time link (/undangan/{Token}) for a tenant's
// Owner, created by ElProof's team (sales-assisted path).
type OwnerInvitation struct {
	Token     string
	ExpiresAt time.Time
	// Email is the Google account that must accept it.
	Email string
}

type RegisterTenantResult struct {
	Tenant     domain.Tenant
	Invitation OwnerInvitation
}

// Register is the sales-assisted path (PLAN.md §3): ElProof's team creates
// the workspace and the Owner receives an invitation link instead of a
// password typed by an admin. It reuses the self-service saga's idempotent
// steps (seeds, Owner row) but has no signup row to resume from, so a
// failure leaves a tenant without Owner login that the team can complete
// with "Undang ulang Owner" (ReinviteOwner) — never a usable half-account.
func (s *TenantService) Register(ctx context.Context, input RegisterTenantInput) (*RegisterTenantResult, error) {
	input.Email = strings.TrimSpace(input.Email)
	if !utils.ValidEmail(input.Email) {
		return nil, apperror.Validation("Email Owner tidak valid", map[string][]string{
			"email": {"Masukkan email akun Google Owner"},
		})
	}
	name := normalizeName(input.BusinessName)
	if err := validateBusinessName(name); err != nil {
		return nil, err
	}
	if taken, err := s.identity.EmailTaken(ctx, input.Email); err != nil {
		return nil, err
	} else if taken {
		return nil, apperror.Validation("Email sudah dipakai", map[string][]string{
			"email": {"Email ini sudah dipakai akun ElProof lain"},
		})
	}
	username, err := s.newOwnerUsername(ctx, input.Email)
	if err != nil {
		return nil, err
	}
	trialEndsAt := time.Now().Truncate(time.Second).AddDate(0, 0, domain.CurrentSubscriptionPolicy().TrialDays)
	tenant := &domain.Tenant{
		BusinessName:       name,
		OwnerName:          normalizeName(input.OwnerName),
		Username:           username,
		Email:              input.Email,
		Phone:              NormalizeWhatsAppOrRaw(input.Phone),
		City:               normalizeName(input.City),
		JoinedAt:           time.Now(),
		SubscriptionStatus: domain.StatusTrialing,
		TrialEndsAt:        &trialEndsAt,
		SignupSource:       domain.SignupSourcePlatform,
	}
	if err := s.repo.Create(ctx, tenant); err != nil {
		return nil, err
	}
	inv, err := s.provisionInvitedOwner(ctx, tenant)
	if err != nil {
		return nil, err
	}
	return &RegisterTenantResult{Tenant: *tenant, Invitation: *inv}, nil
}

// provisionInvitedOwner seeds the workspace, ensures the Owner staff row
// and a pending login exist, and issues the Owner's invitation. Every step
// is idempotent, so ReinviteOwner can complete a failed registration.
func (s *TenantService) provisionInvitedOwner(ctx context.Context, tenant *domain.Tenant) (*OwnerInvitation, error) {
	owner, err := s.ownerRole(ctx, tenant.ID)
	if err != nil {
		return nil, err
	}
	if err := s.vendors.SeedDefaultCategories(ctx, tenant.ID); err != nil {
		return nil, err
	}
	if err := s.projects.SeedDefaultMilestoneTemplate(ctx, tenant.ID); err != nil {
		return nil, err
	}
	ownerID, err := s.staff.FindOwner(ctx, tenant.ID)
	if err != nil {
		return nil, err
	}
	if ownerID == "" {
		created, err := s.staff.CreateOwner(ctx, staffcontracts.CreateOwnerInput{
			TenantID: tenant.ID, Name: tenant.OwnerName, Email: tenant.Email, Phone: tenant.Phone, Username: tenant.Username, RoleID: owner.ID,
		})
		if err != nil {
			return nil, err
		}
		ownerID = created.StaffID
	}
	hasLogin, err := s.identity.CredentialExists(ctx, identitycontracts.PrincipalStaff, ownerID)
	if err != nil {
		return nil, err
	}
	inviteEmail := tenant.Email
	if !hasLogin {
		if err := s.identity.CreateCredential(ctx, identitycontracts.CreateCredentialInput{
			TenantID: &tenant.ID, PrincipalType: identitycontracts.PrincipalStaff, PrincipalID: ownerID,
			Username: tenant.Username, Email: tenant.Email, Pending: true, Role: "Owner", RoleID: owner.ID, DisplayName: tenant.OwnerName,
		}); err != nil {
			return nil, err
		}
	} else {
		// The invitation goes to the email already on the Owner's login (the
		// Owner keeps it current from Pengguna). tenants.email is a contact
		// field the Platform Console can edit; letting it redirect the
		// Owner's login would let any Support account take over a workspace.
		// It is used only when the login has no email at all.
		current, err := s.identity.LoginEmail(ctx, identitycontracts.PrincipalStaff, ownerID)
		if err != nil {
			return nil, err
		}
		if current == "" {
			if err := s.identity.UpdateEmail(ctx, identitycontracts.PrincipalStaff, ownerID, tenant.Email); err != nil {
				return nil, err
			}
		} else {
			inviteEmail = current
		}
	}
	link, err := s.identity.IssueAccessLink(ctx, identitycontracts.PrincipalStaff, ownerID)
	if err != nil {
		return nil, err
	}
	return &OwnerInvitation{Token: link.Token, ExpiresAt: link.ExpiresAt, Email: inviteEmail}, nil
}

// NormalizeWhatsAppOrRaw normalises an Indonesian number when it looks like
// one, otherwise keeps what was typed (Platform Console free-form field).
func NormalizeWhatsAppOrRaw(v string) string {
	if n := NormalizeWhatsApp(v); waNumber.MatchString(n) {
		return n
	}
	return strings.TrimSpace(v)
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
	s.access.invalidate(id)
	return s.Get(ctx, id)
}

// ReinviteOwner gives a tenant's Owner a new invitation link — the Owner
// lost access to their Google account, never accepted the first one, or a
// registration stopped half-way. Accepting it links the login to the
// Google account of the tenant's current Owner email. Earlier links stop
// working; the Owner's current sessions end when the new link is accepted.
func (s *TenantService) ReinviteOwner(ctx context.Context, id int64) (*OwnerInvitation, error) {
	tenant, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if !utils.ValidEmail(tenant.Email) {
		return nil, apperror.Validation("Email Owner tidak valid", map[string][]string{
			"email": {"Perbaiki email Owner tenant ini dulu"},
		})
	}
	inv, err := s.provisionInvitedOwner(ctx, tenant)
	if err != nil {
		return nil, err
	}
	if err := s.repo.SetCredentialResetAt(ctx, id, time.Now()); err != nil {
		return nil, err
	}
	return inv, nil
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
// ActivateSubscriptionInput is a manual grant by ElProof's team. Reason and
// proof are mandatory so every free or offline-paid period can be explained
// later (who granted it, why, on what evidence).
type ActivateSubscriptionInput struct {
	PlanID         int64
	Reason         domain.ManualActivationReason
	ProofReference string
	Note           string
	ActorAdminID   string
}

func validateActivation(input ActivateSubscriptionInput) error {
	fields := map[string][]string{}
	if input.PlanID <= 0 {
		fields["planId"] = []string{"Pilih paket yang diaktifkan"}
	}
	if _, ok := domain.ManualActivationReasonLabels[input.Reason]; !ok {
		fields["reason"] = []string{"Pilih alasan aktivasi"}
	}
	if n := len(strings.TrimSpace(input.ProofReference)); n < 3 || n > 255 {
		fields["proofReference"] = []string{"Isi bukti aktivasi (no. transaksi, no. invoice, atau tautan dokumen), 3–255 karakter"}
	}
	if input.Reason == domain.ReasonOther && len(strings.TrimSpace(input.Note)) < 5 {
		fields["note"] = []string{"Jelaskan alasan aktivasi (minimal 5 karakter)"}
	}
	if len(input.Note) > 500 {
		fields["note"] = []string{"Catatan maksimal 500 karakter"}
	}
	if len(fields) > 0 {
		return apperror.Validation("Data aktivasi belum lengkap", fields)
	}
	return nil
}

func (s *TenantService) ActivateSubscription(ctx context.Context, tenantID int64, input ActivateSubscriptionInput) (*domain.Tenant, error) {
	if err := validateActivation(input); err != nil {
		return nil, err
	}
	tenant, err := s.Get(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	plan, err := s.billing.GetPlan(ctx, input.PlanID)
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

	reference := generateGrantReference()
	newExpiry := computeNewExpiry(tenant, plan)
	if err := s.billing.RecordTransaction(ctx, billingcontracts.RecordTransactionInput{
		TenantID: tenantID, Type: txTypeFor(tenant), Amount: plan.Price,
		PaymentMethod:    "Aktivasi Manual — " + domain.ManualActivationReasonLabels[input.Reason],
		PaymentReference: reference,
		Status:           billingcontracts.StatusGranted,
	}); err != nil {
		return nil, err
	}
	// The ledger lives in `billing` (another module, so not the same DB
	// transaction): it is written first, and cancelled again if the
	// platform-side grant below cannot be applied.
	if err := s.activations.ApplyGrant(ctx, &domain.ManualActivation{
		TenantID: tenantID, PlanID: input.PlanID, ActorAdminID: input.ActorAdminID, Reason: input.Reason,
		ProofReference: strings.TrimSpace(input.ProofReference), Note: strings.TrimSpace(input.Note),
		TransactionReference: reference, ExpiresAt: newExpiry,
	}, domain.StatusActive); err != nil {
		if cerr := s.billing.UpdateTransactionStatus(ctx, reference, billingcontracts.StatusCancelled); cerr != nil {
			logger.Error("platform: manual activation of tenant %d failed and its ledger row %s could not be cancelled: %v", tenantID, reference, cerr)
		}
		return nil, err
	}
	s.access.invalidate(tenantID)
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
	s.access.invalidate(pending.TenantID)
	return s.pendingCharges.Delete(ctx, orderRef)
}

// txTypeFor: the first paid period ever is "new"; every later one is a
// renewal. Decided from the dates, not the cached status column.
func txTypeFor(tenant *domain.Tenant) billingcontracts.TransactionType {
	if tenant.SubscriptionExpiresAt != nil {
		return billingcontracts.TransactionRenewal
	}
	return billingcontracts.TransactionNew
}

// computeNewExpiry starts the new paid period where the tenant's current
// access ends, so nobody loses days they already have: a renewal stacks on
// top of the remaining paid time, and paying during the free trial starts
// the paid period when the trial ends. A lapsed tenant restarts from now.
func computeNewExpiry(tenant *domain.Tenant, plan *billingcontracts.Plan) time.Time {
	return computeNewExpiryAt(tenant, plan.DurationMonths, time.Now())
}

func computeNewExpiryAt(tenant *domain.Tenant, durationMonths int, now time.Time) time.Time {
	base := now
	switch {
	case tenant.SubscriptionExpiresAt != nil:
		if tenant.SubscriptionExpiresAt.After(base) {
			base = *tenant.SubscriptionExpiresAt
		}
	case tenant.TrialEndsAt != nil:
		if tenant.TrialEndsAt.After(base) {
			base = *tenant.TrialEndsAt
		}
	}
	return base.AddDate(0, durationMonths, 0)
}

// ParseTenantID converts a JWT tenant-id claim (string) to int64 — used by the
// presentation layer for the self-service "pay" endpoint.
func ParseTenantID(raw string) (int64, error) {
	return strconv.ParseInt(raw, 10, 64)
}

// PortalContact returns the workspace's business phone (the WhatsApp number)
// and bank account details, as the Client Portal home needs them.
func (s *TenantService) PortalContact(ctx context.Context, tenantID int64) (whatsapp, bankName, accountNumber, accountHolder string, err error) {
	t, err := s.Get(ctx, tenantID)
	if err != nil {
		return "", "", "", "", err
	}
	d, err := s.GetBusinessDetails(ctx, tenantID)
	if err != nil {
		return "", "", "", "", err
	}
	return t.Phone, d.BankName, d.BankAccountNumber, d.BankAccountHolder, nil
}
