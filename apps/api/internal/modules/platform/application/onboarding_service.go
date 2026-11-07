package application

import (
	"context"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	clientscontracts "elproof/internal/modules/clients/contracts"
	"elproof/internal/modules/platform/domain"
	"elproof/internal/shared/apperror"
)

// TrialLimitsRepository reads/writes the single trial_limits row.
type TrialLimitsRepository interface {
	Get(ctx context.Context) (domain.TrialLimits, error)
	Update(ctx context.Context, l domain.TrialLimits) error
}

// SetTrialLimits plugs in the trial-cap store (platform.module.go).
func (s *TenantService) SetTrialLimits(repo TrialLimitsRepository) {
	s.trialLimits = repo
}

// SetClients completes two-phase wiring with the clients module (client
// usage on the onboarding checklist); clients is built after platform.
func (s *TenantService) SetClients(clients clientscontracts.Contracts) {
	s.clients = clients
}

// TrialLimit implements the TrialQuota port of staff/projects/clients: the
// cap for one resource while the tenant is on its free trial; limited=false
// once it has ever paid (paid plans are unlimited, decision K6).
func (s *TenantService) TrialLimit(ctx context.Context, tenantID int64, resource string) (int, bool, error) {
	if s.trialLimits == nil {
		return 0, false, nil
	}
	tenant, err := s.Get(ctx, tenantID)
	if err != nil {
		return 0, false, err
	}
	if !s.Lifecycle(*tenant).IsTrial {
		return 0, false, nil
	}
	limits, err := s.trialLimits.Get(ctx)
	if err != nil {
		return 0, false, err
	}
	limit, ok := limits.Limit(resource)
	return limit, ok, nil
}

func (s *TenantService) GetTrialLimits(ctx context.Context) (domain.TrialLimits, error) {
	return s.trialLimits.Get(ctx)
}

const maxTrialLimit = 1000

// UpdateTrialLimits is the Platform Console edit of the trial caps.
func (s *TenantService) UpdateTrialLimits(ctx context.Context, l domain.TrialLimits) (domain.TrialLimits, error) {
	fields := map[string][]string{}
	check := func(key string, v int) {
		if v < 1 || v > maxTrialLimit {
			fields[key] = []string{"Isi angka 1–1000"}
		}
	}
	check("maxStaff", l.MaxStaff)
	check("maxActiveProjects", l.MaxActiveProjects)
	check("maxClients", l.MaxClients)
	if len(fields) > 0 {
		return domain.TrialLimits{}, apperror.Validation("Batas masa coba tidak valid", fields)
	}
	if err := s.trialLimits.Update(ctx, l); err != nil {
		return domain.TrialLimits{}, err
	}
	return l, nil
}

// OnboardingStatus backs the Owner's "Mulai di sini" checklist (PLAN.md §3
// step 4) and the trial usage meter.
type OnboardingStatus struct {
	Dismissed       bool
	ProfileComplete bool
	HasLogo         bool
	HasPaidPlan     bool
	Staff           int
	ActiveProjects  int
	Clients         int
	// Limits is set while the tenant is on its free trial.
	Limits *domain.TrialLimits
}

func (s *TenantService) Onboarding(ctx context.Context, tenantID int64) (*OnboardingStatus, error) {
	tenant, err := s.Get(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	st := &OnboardingStatus{
		Dismissed:       tenant.OnboardingDismissedAt != nil,
		ProfileComplete: tenant.Phone != "" && tenant.City != "",
		HasLogo:         tenant.LogoStoragePath != nil,
		HasPaidPlan:     tenant.SubscriptionExpiresAt != nil,
	}
	if st.Staff, err = s.staff.CountActiveStaff(ctx, tenantID); err != nil {
		return nil, err
	}
	if st.ActiveProjects, err = s.projects.CountActiveProjects(ctx, tenantID); err != nil {
		return nil, err
	}
	if s.clients != nil {
		if st.Clients, err = s.clients.CountActiveClients(ctx, tenantID); err != nil {
			return nil, err
		}
	}
	if s.trialLimits != nil && s.Lifecycle(*tenant).IsTrial {
		limits, err := s.trialLimits.Get(ctx)
		if err != nil {
			return nil, err
		}
		st.Limits = &limits
	}
	return st, nil
}

func (s *TenantService) DismissOnboarding(ctx context.Context, tenantID int64) error {
	return s.repo.SetOnboardingDismissed(ctx, tenantID, time.Now())
}

// UpdateProfileInput is the Owner's own "Lengkapi Profil" form.
type UpdateProfileInput struct {
	BusinessName string
	Phone        string
	City         string
}

var waNumber = regexp.MustCompile(`^62[1-9][0-9]{7,12}$`)

// NormalizeWhatsApp turns the common Indonesian spellings (0812…, +62 812…,
// 62-812…) into 62812… — the form wa.me links need.
func NormalizeWhatsApp(v string) string {
	var b strings.Builder
	for _, r := range v {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	d := b.String()
	switch {
	case strings.HasPrefix(d, "62"):
	case strings.HasPrefix(d, "0"):
		d = "62" + d[1:]
	case strings.HasPrefix(d, "8"):
		d = "62" + d
	}
	return d
}

// UpdateOwnProfile is the WO Owner editing their workspace profile
// (business name, WhatsApp, city). Other fields stay with the Platform
// Console.
func (s *TenantService) UpdateOwnProfile(ctx context.Context, tenantID int64, in UpdateProfileInput) (*domain.Tenant, error) {
	name := normalizeName(in.BusinessName)
	phone := NormalizeWhatsApp(in.Phone)
	city := normalizeName(in.City)
	fields := map[string][]string{}
	if err := validateBusinessName(name); err != nil {
		fields["businessName"] = []string{"Nama WO 3–100 karakter"}
	}
	if !waNumber.MatchString(phone) {
		fields["phone"] = []string{"Masukkan nomor WhatsApp yang valid, contoh 0812 3456 7890"}
	}
	if n := utf8.RuneCountInString(city); n < 2 || n > 100 {
		fields["city"] = []string{"Kota 2–100 karakter"}
	}
	if len(fields) > 0 {
		return nil, apperror.Validation("Profil belum lengkap", fields)
	}
	if err := s.repo.UpdateProfile(ctx, tenantID, name, phone, city); err != nil {
		return nil, err
	}
	return s.Get(ctx, tenantID)
}

// UpdateOwnBrandColor lets the Owner choose the workspace's brand color from
// the fixed presets (it recolors the WO Console and the Client Portal).
func (s *TenantService) UpdateOwnBrandColor(ctx context.Context, tenantID int64, preset string) (*domain.Tenant, error) {
	if !domain.IsValidBrandColorPreset(preset) {
		return nil, apperror.Validation("Warna tidak dikenal", map[string][]string{"preset": {"Pilih salah satu warna yang tersedia"}})
	}
	if err := s.repo.UpdateBrandColor(ctx, tenantID, preset); err != nil {
		return nil, err
	}
	return s.Get(ctx, tenantID)
}

var bankAccountNumber = regexp.MustCompile(`^[0-9][0-9 .-]{3,38}[0-9]$`)

// BusinessDetailsInput is the Owner's form for address and bank account.
type BusinessDetailsInput struct {
	Address           string
	BankName          string
	BankAccountNumber string
	BankAccountHolder string
}

// GetBusinessDetails returns what the Owner has shared with clients so far.
func (s *TenantService) GetBusinessDetails(ctx context.Context, tenantID int64) (domain.BusinessDetails, error) {
	return s.repo.GetBusinessDetails(ctx, tenantID)
}

// SaveBusinessDetails stores the address and the bank account clients pay
// into. Everything is optional, but a bank account is all-or-nothing: a
// number without its bank and holder would send clients to an account they
// cannot recognize.
func (s *TenantService) SaveBusinessDetails(ctx context.Context, tenantID int64, in BusinessDetailsInput) (domain.BusinessDetails, error) {
	d := domain.BusinessDetails{
		Address:           normalizeName(in.Address),
		BankName:          normalizeName(in.BankName),
		BankAccountNumber: strings.TrimSpace(in.BankAccountNumber),
		BankAccountHolder: normalizeName(in.BankAccountHolder),
	}
	fields := map[string][]string{}
	if utf8.RuneCountInString(d.Address) > 255 {
		fields["address"] = []string{"Alamat maksimal 255 karakter"}
	}
	anyBank := d.BankName != "" || d.BankAccountNumber != "" || d.BankAccountHolder != ""
	if anyBank {
		if n := utf8.RuneCountInString(d.BankName); n < 2 || n > 60 {
			fields["bankName"] = []string{"Isi nama bank (2–60 karakter)"}
		}
		if !bankAccountNumber.MatchString(d.BankAccountNumber) {
			fields["bankAccountNumber"] = []string{"Isi nomor rekening (5–40 angka)"}
		}
		if n := utf8.RuneCountInString(d.BankAccountHolder); n < 2 || n > 100 {
			fields["bankAccountHolder"] = []string{"Isi nama pemilik rekening (2–100 karakter)"}
		}
	}
	if len(fields) > 0 {
		return domain.BusinessDetails{}, apperror.Validation("Data usaha belum lengkap", fields)
	}
	if err := s.repo.SaveBusinessDetails(ctx, tenantID, d); err != nil {
		return domain.BusinessDetails{}, err
	}
	return d, nil
}
