package application

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	identitycontracts "elproof/internal/modules/identity/contracts"
	"elproof/internal/modules/platform/domain"
	staffcontracts "elproof/internal/modules/staff/contracts"
	"elproof/internal/shared/apperror"
	"elproof/internal/shared/logger"
	"elproof/internal/shared/validator"
)

// SignupRepository stores self-service registrations (signup_requests).
type SignupRepository interface {
	FindByGoogleSub(ctx context.Context, sub string) (*domain.SignupRequest, error)
	// Create returns domain.ErrSignupExists when the Google account already
	// has a row.
	Create(ctx context.Context, s *domain.SignupRequest) error
	UpdateBusinessName(ctx context.Context, id int64, name string) error
	MarkCompleted(ctx context.Context, id int64, when time.Time) error
}

// SetSignups plugs in the signup store (platform.module.go).
func (s *TenantService) SetSignups(repo SignupRepository) {
	s.signups = repo
}

// CreateWorkspaceInput is step 2 of the self-service flow (PLAN.md §3): the
// Google credential from step 1 plus the one-screen workspace form.
type CreateWorkspaceInput struct {
	IDToken        string
	BusinessName   string
	AcceptTerms    bool
	MarketingOptIn bool
	IPAddress      string
	UserAgent      string
}

type CreateWorkspaceResult struct {
	TenantID int64
	Email    string
}

const (
	minBusinessNameLen = 3
	maxBusinessNameLen = 100
	// maxEmailLen matches the email columns (VARCHAR(150)).
	maxEmailLen = 150
	// provisionTimeout bounds one saga run; it is detached from the HTTP
	// request so a closed browser tab cannot stop it half-way.
	provisionTimeout = 30 * time.Second
)

// normalizeName trims and collapses inner whitespace.
func normalizeName(v string) string {
	return strings.Join(strings.Fields(v), " ")
}

func validateBusinessName(name string) error {
	n := utf8.RuneCountInString(name)
	if n < minBusinessNameLen || n > maxBusinessNameLen {
		return apperror.Validation("Nama WO tidak valid", map[string][]string{
			"businessName": {fmt.Sprintf("Nama WO %d–%d karakter", minBusinessNameLen, maxBusinessNameLen)},
		})
	}
	return nil
}

// CreateWorkspace registers a new WO from a Google account and returns once
// its workspace is ready; the caller then signs in with the same Google
// credential (POST /auth/google).
//
// Provisioning spans several modules (tenant here, vendor categories,
// Timeline Default, Owner staff row, login), and a transaction cannot cross
// modules, so it runs as a resumable saga: the signup row is written first
// and remembers the tenant, every later step is idempotent, and any retry —
// pressing the button again, or signing in with Google again — continues
// where the last attempt stopped instead of creating a second workspace.
// Nothing is reachable before the last step (the Owner's login) exists.
func (s *TenantService) CreateWorkspace(ctx context.Context, in CreateWorkspaceInput) (*CreateWorkspaceResult, error) {
	if s.signups == nil {
		return nil, apperror.Internal("Pendaftaran mandiri belum dikonfigurasi")
	}
	businessName := normalizeName(in.BusinessName)
	if err := validateBusinessName(businessName); err != nil {
		return nil, err
	}
	if !in.AcceptTerms {
		return nil, apperror.Validation("Persetujuan wajib", map[string][]string{
			"acceptTerms": {"Setujui Syarat & Ketentuan dan Perjanjian Pemrosesan Data untuk melanjutkan"},
		})
	}

	google, err := s.identity.VerifyGoogleIDToken(ctx, in.IDToken)
	if err != nil {
		return nil, err
	}
	if len(google.Email) > maxEmailLen {
		return nil, apperror.Validation("Alamat email terlalu panjang", map[string][]string{
			"email": {"Gunakan akun Google dengan alamat email maksimal 150 karakter"},
		})
	}
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), provisionTimeout)
	defer cancel()

	// One provisioning run per Google account at a time (double-clicks,
	// two tabs). Single-instance deployment, so an in-process lock is
	// enough; the guarded AttachTenant update backs it up in SQL.
	unlock := s.signupLocks.lock(google.Sub)
	defer unlock()

	req, err := s.signups.FindByGoogleSub(ctx, google.Sub)
	if err != nil {
		return nil, err
	}
	if req != nil && req.Status == domain.SignupCompleted {
		return &CreateWorkspaceResult{TenantID: derefID(req.TenantID), Email: req.Email}, nil
	}
	if req != nil && req.TenantID == nil && req.BusinessName != businessName {
		// No tenant yet: the latest name typed wins.
		if err := s.signups.UpdateBusinessName(ctx, req.ID, businessName); err != nil {
			return nil, err
		}
		req.BusinessName = businessName
	}
	if req == nil {
		if err := s.identity.CheckGoogleSignup(ctx, google.Sub, google.Email); err != nil {
			return nil, err
		}
		ownerName := normalizeName(google.Name)
		if ownerName == "" {
			ownerName, _, _ = strings.Cut(google.Email, "@")
		}
		req = &domain.SignupRequest{
			GoogleSub:       google.Sub,
			Email:           google.Email,
			OwnerName:       truncateRunes(ownerName, 150),
			BusinessName:    businessName,
			TermsVersion:    domain.TermsVersion,
			TermsAcceptedAt: time.Now(),
			MarketingOptIn:  in.MarketingOptIn,
			IPAddress:       truncateRunes(in.IPAddress, 45),
			UserAgent:       truncateRunes(in.UserAgent, 255),
		}
		if err := s.signups.Create(ctx, req); err != nil {
			if errors.Is(err, domain.ErrSignupExists) {
				return nil, apperror.Conflict("Pendaftaran untuk akun Google ini sedang diproses. Coba lagi sebentar.")
			}
			return nil, err
		}
	}

	if err := s.provisionWorkspace(ctx, req); err != nil {
		logger.Error("signup %d (tenant %v) provisioning stopped: %v", req.ID, req.TenantID, err)
		return nil, err
	}
	return &CreateWorkspaceResult{TenantID: derefID(req.TenantID), Email: req.Email}, nil
}

// provisionWorkspace runs (or resumes) the saga steps for one signup.
func (s *TenantService) provisionWorkspace(ctx context.Context, req *domain.SignupRequest) error {
	if req.TenantID == nil {
		username, err := s.newOwnerUsername(ctx, req.Email)
		if err != nil {
			return err
		}
		now := time.Now()
		trialEndsAt := now.Truncate(time.Second).AddDate(0, 0, domain.CurrentSubscriptionPolicy().TrialDays)
		tenant := &domain.Tenant{
			BusinessName:       req.BusinessName,
			OwnerName:          req.OwnerName,
			Username:           username,
			Email:              req.Email,
			JoinedAt:           now,
			SubscriptionStatus: domain.StatusTrialing,
			TrialEndsAt:        &trialEndsAt,
			SignupSource:       domain.SignupSourceSelfService,
		}
		// Tenant row and signup link in one transaction: never a tenant
		// that no signup owns.
		created, err := s.repo.CreateForSignup(ctx, tenant, req.ID)
		if err != nil {
			return err
		}
		if !created {
			// Cannot happen while signupLocks serialises runs.
			return apperror.Conflict("Pendaftaran untuk akun Google ini sedang diproses. Coba lagi sebentar.")
		}
		req.TenantID = &tenant.ID
	}
	tenantID := *req.TenantID
	tenant, err := s.Get(ctx, tenantID)
	if err != nil {
		return err
	}
	owner, err := s.ownerRole(ctx, tenantID)
	if err != nil {
		return err
	}

	if err := s.vendors.SeedDefaultCategories(ctx, tenantID); err != nil {
		return err
	}
	if err := s.projects.SeedDefaultMilestoneTemplate(ctx, tenantID); err != nil {
		return err
	}

	ownerID, err := s.staff.FindOwner(ctx, tenantID)
	if err != nil {
		return err
	}
	if ownerID == "" {
		created, err := s.staff.CreateOwner(ctx, staffcontracts.CreateOwnerInput{
			TenantID: tenantID, Name: req.OwnerName, Email: req.Email, Username: tenant.Username, RoleID: owner.ID,
		})
		if err != nil {
			return err
		}
		ownerID = created.StaffID
	}

	hasLogin, err := s.identity.CredentialExists(ctx, identitycontracts.PrincipalStaff, ownerID)
	if err != nil {
		return err
	}
	if !hasLogin {
		// Between the first attempt and this one, another login may have
		// taken the chosen username or this email. The username is only
		// internal (Owners sign in with Google): pick a new one. The email is
		// left off the login (Owners never sign in by email; it stays on the
		// tenant and staff rows) instead of blocking the workspace forever.
		username := tenant.Username
		if taken, err := s.identity.UsernameTaken(ctx, username); err != nil {
			return err
		} else if taken {
			if username, err = s.newOwnerUsername(ctx, req.Email); err != nil {
				return err
			}
			if err := s.repo.UpdateUsername(ctx, tenantID, username); err != nil {
				return err
			}
		}
		email := req.Email
		if taken, err := s.identity.EmailTaken(ctx, email); err != nil {
			return err
		} else if taken {
			email = ""
		}
		if err := s.identity.CreateCredential(ctx, identitycontracts.CreateCredentialInput{
			TenantID:      &tenantID,
			PrincipalType: identitycontracts.PrincipalStaff,
			PrincipalID:   ownerID,
			Username:      username,
			Email:         email,
			GoogleSub:     req.GoogleSub,
			Role:          "Owner",
			RoleID:        owner.ID,
			DisplayName:   req.OwnerName,
		}); err != nil {
			return err
		}
	}

	now := time.Now()
	if err := s.signups.MarkCompleted(ctx, req.ID, now); err != nil {
		return err
	}
	req.Status, req.CompletedAt = domain.SignupCompleted, &now
	logger.Info("self-service signup %d completed: tenant %d", req.ID, tenantID)
	return nil
}

// newOwnerUsername derives an internal username from the Google email (the
// Owner never types it — they sign in with Google) and makes it unique.
func (s *TenantService) newOwnerUsername(ctx context.Context, email string) (string, error) {
	local, _, _ := strings.Cut(strings.ToLower(email), "@")
	var b strings.Builder
	for _, r := range local {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '.' {
			b.WriteRune(r)
		}
	}
	base := strings.Trim(b.String(), "._")
	if len(base) > 24 {
		base = base[:24]
	}
	if len(base) < 4 {
		base = "wo." + base
		base = strings.TrimRight(base, ".")
		if len(base) < 4 {
			base = "wo.owner"
		}
	}
	candidate := base
	for attempt := 0; attempt < 8; attempt++ {
		if attempt > 0 {
			n, err := rand.Int(rand.Reader, big.NewInt(9000))
			if err != nil {
				return "", err
			}
			candidate = fmt.Sprintf("%s.%d", base, 1000+n.Int64())
		}
		if validator.Username(candidate) != nil {
			continue
		}
		taken, err := s.identity.UsernameTaken(ctx, candidate)
		if err != nil {
			return "", err
		}
		if !taken {
			return candidate, nil
		}
	}
	return "", apperror.Internal("Gagal membuat nama pengguna. Silakan coba lagi.")
}

func derefID(id *int64) int64 {
	if id == nil {
		return 0
	}
	return *id
}

func truncateRunes(v string, max int) string {
	if utf8.RuneCountInString(v) <= max {
		return v
	}
	return string([]rune(v)[:max])
}

// keyedLocks serialises work per key (one signup per Google account).
type keyedLocks struct {
	mu    sync.Mutex
	locks map[string]*keyedLock
}

type keyedLock struct {
	mu   sync.Mutex
	refs int
}

func (k *keyedLocks) lock(key string) (unlock func()) {
	k.mu.Lock()
	if k.locks == nil {
		k.locks = make(map[string]*keyedLock)
	}
	l := k.locks[key]
	if l == nil {
		l = &keyedLock{}
		k.locks[key] = l
	}
	l.refs++
	k.mu.Unlock()

	l.mu.Lock()
	return func() {
		l.mu.Unlock()
		k.mu.Lock()
		l.refs--
		if l.refs == 0 {
			delete(k.locks, key)
		}
		k.mu.Unlock()
	}
}
