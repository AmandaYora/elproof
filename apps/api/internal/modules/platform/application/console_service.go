package application

import (
	"context"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"elproof/internal/modules/platform/domain"
	staffcontracts "elproof/internal/modules/staff/contracts"
	"elproof/internal/shared/apperror"
)

// ConsoleRepository stores the Platform Console's own data.
type ConsoleRepository interface {
	ListNotes(ctx context.Context, tenantID int64) ([]domain.TenantNote, error)
	AddNote(ctx context.Context, n *domain.TenantNote) error
	ListDataRequests(ctx context.Context, tenantID *int64, openOnly bool) ([]domain.DataRequest, error)
	AddDataRequest(ctx context.Context, d *domain.DataRequest, adminID string) error
	CompleteDataRequest(ctx context.Context, id int64, status domain.DataRequestStatus, adminID, note string, at time.Time) (bool, error)
	ListAnnouncements(ctx context.Context) ([]domain.Announcement, error)
	CreateAnnouncement(ctx context.Context, a *domain.Announcement) error
	UpdateAnnouncement(ctx context.Context, a *domain.Announcement) (bool, error)
	DeleteAnnouncement(ctx context.Context, id int64) (bool, error)
	GetPolicy(ctx context.Context) (domain.SubscriptionPolicy, bool, error)
	SavePolicy(ctx context.Context, p domain.SubscriptionPolicy, adminID string) error
	LatestGrant(ctx context.Context, tenantID int64) (*domain.SupportAccessGrant, error)
	CreateGrant(ctx context.Context, g *domain.SupportAccessGrant) error
	RevokeGrants(ctx context.Context, tenantID int64, at time.Time) error
	CreateSession(ctx context.Context, s *domain.SupportSession) error
	EndSession(ctx context.Context, id int64, adminID string, at time.Time) (bool, error)
	FindSession(ctx context.Context, id int64) (*domain.SupportSession, error)
	DeleteSession(ctx context.Context, id int64) error
	ListSessions(ctx context.Context, tenantID int64, limit int) ([]domain.SupportSession, error)
	ListManualActivations(ctx context.Context, tenantID int64) ([]domain.ManualActivation, error)
	GetSignupConsent(ctx context.Context, tenantID int64) (*domain.SignupConsent, error)
}

// SupportTokenIssuer mints the read-only token of a support session
// (identity's contract, narrowed).
type SupportTokenIssuer interface {
	IssueSupportToken(ctx context.Context, staffPrincipalID, adminID string, sessionID int64, ttl time.Duration) (string, error)
}

// ConsoleService is the Platform Console's back office: the 360° view of a
// workspace, internal notes, data requests, announcements, the subscription
// policy and support access.
type ConsoleService struct {
	repo    ConsoleRepository
	tenants *TenantService
	support SupportTokenIssuer
	admins  SupportAdminLookup

	onPolicyChanged func(ctx context.Context)
}

// SupportAdminLookup finds the platform admin behind a support session, so a
// session ends when its admin is deactivated or loses the support role.
type SupportAdminLookup interface {
	FindByID(ctx context.Context, id int64) (*domain.PlatformAdmin, error)
}

// SetAdmins completes wiring (the admin repository lives in the same module).
func (s *ConsoleService) SetAdmins(a SupportAdminLookup) { s.admins = a }

func NewConsoleService(repo ConsoleRepository, tenants *TenantService, support SupportTokenIssuer) *ConsoleService {
	return &ConsoleService{repo: repo, tenants: tenants, support: support}
}

func validationErr(msg, field string) error {
	return apperror.Validation(msg, map[string][]string{field: {msg}})
}

// ---- 360° ----

// TenantOverview is everything the Detail Tenant page shows besides the
// ledger (which the page reads from the billing endpoints).
type TenantOverview struct {
	Tenant           domain.Tenant
	Lifecycle        domain.Lifecycle
	PlanName         string
	Consent          *domain.SignupConsent
	Activations      []domain.ManualActivation
	Usage            OnboardingStatus
	Members          []staffcontracts.MemberSummary
	OpenDataRequests int
	SupportAccess    *domain.SupportAccessGrant
}

func (s *ConsoleService) Overview(ctx context.Context, tenantID int64, now time.Time) (*TenantOverview, error) {
	tenant, err := s.tenants.Get(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	o := &TenantOverview{Tenant: *tenant, Lifecycle: domain.ComputeLifecycle(*tenant, now, domain.CurrentSubscriptionPolicy())}
	if tenant.PlanID != nil {
		if plan, err := s.tenants.billing.GetPlan(ctx, *tenant.PlanID); err == nil && plan != nil {
			o.PlanName = plan.Name
		}
	}
	if o.Consent, err = s.repo.GetSignupConsent(ctx, tenantID); err != nil {
		return nil, err
	}
	if o.Activations, err = s.repo.ListManualActivations(ctx, tenantID); err != nil {
		return nil, err
	}
	usage, err := s.tenants.Onboarding(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	o.Usage = *usage
	if o.Members, err = s.tenants.staff.ListMembers(ctx, tenantID); err != nil {
		return nil, err
	}
	reqs, err := s.repo.ListDataRequests(ctx, &tenantID, true)
	if err != nil {
		return nil, err
	}
	o.OpenDataRequests = len(reqs)
	if g, err := s.repo.LatestGrant(ctx, tenantID); err != nil {
		return nil, err
	} else if g != nil && g.ActiveAt(now) {
		o.SupportAccess = g
	}
	return o, nil
}

// ---- notes ----

func (s *ConsoleService) ListNotes(ctx context.Context, tenantID int64) ([]domain.TenantNote, error) {
	if _, err := s.tenants.Get(ctx, tenantID); err != nil {
		return nil, err
	}
	return s.repo.ListNotes(ctx, tenantID)
}

func (s *ConsoleService) AddNote(ctx context.Context, tenantID int64, adminID, adminName, body string) (*domain.TenantNote, error) {
	body = strings.TrimSpace(body)
	if n := utf8.RuneCountInString(body); n < 2 || n > 2000 {
		return nil, validationErr("Catatan 2–2000 karakter", "body")
	}
	if _, err := s.tenants.Get(ctx, tenantID); err != nil {
		return nil, err
	}
	note := &domain.TenantNote{TenantID: tenantID, AdminID: adminID, AdminName: adminName, Body: body}
	if err := s.repo.AddNote(ctx, note); err != nil {
		return nil, err
	}
	note.CreatedAt = time.Now()
	return note, nil
}

// ---- data requests (UU PDP) ----

func (s *ConsoleService) ListDataRequests(ctx context.Context, tenantID *int64, openOnly bool) ([]domain.DataRequest, error) {
	return s.repo.ListDataRequests(ctx, tenantID, openOnly)
}

// AddDataRequest logs a workspace's request to export or delete its data; the
// platform has 30 days to answer.
func (s *ConsoleService) AddDataRequest(ctx context.Context, tenantID int64, kind domain.DataRequestKind, requestedBy, note, adminID string, now time.Time) (*domain.DataRequest, error) {
	if kind != domain.DataRequestExport && kind != domain.DataRequestDelete {
		return nil, validationErr("Pilih ekspor atau penghapusan data", "kind")
	}
	requestedBy = strings.TrimSpace(requestedBy)
	if n := utf8.RuneCountInString(requestedBy); n < 2 || n > 200 {
		return nil, validationErr("Isi siapa yang meminta (nama atau email, 2–200 karakter)", "requestedBy")
	}
	if utf8.RuneCountInString(note) > 1000 {
		return nil, validationErr("Catatan maksimal 1000 karakter", "note")
	}
	if _, err := s.tenants.Get(ctx, tenantID); err != nil {
		return nil, err
	}
	d := &domain.DataRequest{
		TenantID: tenantID, Kind: kind, Status: domain.DataRequestOpen, RequestedBy: requestedBy, Note: strings.TrimSpace(note),
		DueAt: now.AddDate(0, 0, domain.DataRequestDeadlineDays),
	}
	if err := s.repo.AddDataRequest(ctx, d, adminID); err != nil {
		return nil, err
	}
	d.CreatedAt = now
	return d, nil
}

// CompleteDataRequest closes an open request as done or rejected, recording
// who and the proof or reason (required: it is the evidence if asked).
func (s *ConsoleService) CompleteDataRequest(ctx context.Context, id int64, status domain.DataRequestStatus, adminID, note string, now time.Time) error {
	if status != domain.DataRequestDone && status != domain.DataRequestRejected {
		return validationErr("Pilih selesai atau ditolak", "status")
	}
	note = strings.TrimSpace(note)
	if n := utf8.RuneCountInString(note); n < 5 || n > 1000 {
		return validationErr("Isi bukti penyelesaian atau alasan penolakan (5–1000 karakter)", "note")
	}
	ok, err := s.repo.CompleteDataRequest(ctx, id, status, adminID, note, now)
	if err != nil {
		return err
	}
	if !ok {
		return apperror.Conflict("Permintaan ini tidak ada atau sudah ditutup")
	}
	return nil
}

// ---- announcements ----

// AnnouncementInput is the editor's form.
type AnnouncementInput struct {
	Title         string
	Body          string
	Kind          domain.AnnouncementKind
	Audience      domain.AnnouncementAudience
	AudienceValue string
	StartsAt      time.Time
	EndsAt        *time.Time
}

func (in AnnouncementInput) validate() error {
	fields := map[string][]string{}
	if n := utf8.RuneCountInString(strings.TrimSpace(in.Title)); n < 3 || n > 150 {
		fields["title"] = []string{"Judul 3–150 karakter"}
	}
	if n := utf8.RuneCountInString(strings.TrimSpace(in.Body)); n < 3 || n > 2000 {
		fields["body"] = []string{"Isi pengumuman 3–2000 karakter"}
	}
	switch in.Kind {
	case domain.AnnouncementInfo, domain.AnnouncementWarning, domain.AnnouncementMaintenance:
	default:
		fields["kind"] = []string{"Pilih jenis pengumuman"}
	}
	switch in.Audience {
	case domain.AudienceAll:
	case domain.AudiencePlan, domain.AudienceStatus:
		if strings.TrimSpace(in.AudienceValue) == "" {
			fields["audienceValue"] = []string{"Pilih paket atau status yang dituju"}
		}
	default:
		fields["audience"] = []string{"Pilih siapa yang dituju"}
	}
	if in.StartsAt.IsZero() {
		fields["startsAt"] = []string{"Isi waktu tayang"}
	}
	if in.EndsAt != nil && !in.StartsAt.IsZero() && !in.EndsAt.After(in.StartsAt) {
		fields["endsAt"] = []string{"Waktu berakhir harus setelah waktu tayang"}
	}
	if len(fields) > 0 {
		return apperror.Validation("Pengumuman belum lengkap", fields)
	}
	return nil
}

func (in AnnouncementInput) toDomain() domain.Announcement {
	a := domain.Announcement{
		Title: strings.TrimSpace(in.Title), Body: strings.TrimSpace(in.Body), Kind: in.Kind, Audience: in.Audience,
		StartsAt: in.StartsAt, EndsAt: in.EndsAt,
	}
	if in.Audience != domain.AudienceAll {
		a.AudienceValue = strings.TrimSpace(in.AudienceValue)
	}
	return a
}

func (s *ConsoleService) ListAnnouncements(ctx context.Context) ([]domain.Announcement, error) {
	return s.repo.ListAnnouncements(ctx)
}

func (s *ConsoleService) CreateAnnouncement(ctx context.Context, in AnnouncementInput, adminID string) (*domain.Announcement, error) {
	if err := in.validate(); err != nil {
		return nil, err
	}
	a := in.toDomain()
	a.CreatedBy = adminID
	if err := s.repo.CreateAnnouncement(ctx, &a); err != nil {
		return nil, err
	}
	return &a, nil
}

func (s *ConsoleService) UpdateAnnouncement(ctx context.Context, id int64, in AnnouncementInput) (*domain.Announcement, error) {
	if err := in.validate(); err != nil {
		return nil, err
	}
	a := in.toDomain()
	a.ID = id
	ok, err := s.repo.UpdateAnnouncement(ctx, &a)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, apperror.NotFound("Pengumuman tidak ditemukan")
	}
	return &a, nil
}

func (s *ConsoleService) DeleteAnnouncement(ctx context.Context, id int64) error {
	ok, err := s.repo.DeleteAnnouncement(ctx, id)
	if err != nil {
		return err
	}
	if !ok {
		return apperror.NotFound("Pengumuman tidak ditemukan")
	}
	return nil
}

// ActiveAnnouncements are what a workspace's staff see right now: on air and
// aimed at them (all, their plan, or their subscription status).
func (s *ConsoleService) ActiveAnnouncements(ctx context.Context, tenantID int64, now time.Time) ([]domain.Announcement, error) {
	tenant, err := s.tenants.Get(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	all, err := s.repo.ListAnnouncements(ctx)
	if err != nil {
		return nil, err
	}
	lc := domain.ComputeLifecycle(*tenant, now, domain.CurrentSubscriptionPolicy())
	var out []domain.Announcement
	for _, a := range all {
		if a.ActiveAt(now) && a.Targets(*tenant, lc) {
			out = append(out, a)
		}
	}
	return out, nil
}

// ---- subscription policy ----

// LoadPolicy applies the stored policy (called at startup).
func (s *ConsoleService) LoadPolicy(ctx context.Context) error {
	p, found, err := s.repo.GetPolicy(ctx)
	if err != nil || !found {
		return err
	}
	if len(domain.ValidatePolicy(p)) == 0 {
		domain.SetSubscriptionPolicy(p)
	}
	return nil
}

func (s *ConsoleService) Policy() domain.SubscriptionPolicy {
	return domain.CurrentSubscriptionPolicy()
}

// SavePolicy stores and applies new durations. Existing tenants follow the new
// policy from now on (their state is always derived from dates + policy).
func (s *ConsoleService) SavePolicy(ctx context.Context, p domain.SubscriptionPolicy, adminID string) error {
	if fields := domain.ValidatePolicy(p); len(fields) > 0 {
		return apperror.Validation("Kebijakan belum valid", fields)
	}
	if err := s.repo.SavePolicy(ctx, p, adminID); err != nil {
		return err
	}
	domain.SetSubscriptionPolicy(p)
	// The cached status column the tenant list filters on follows the policy
	// only when the hourly sweep runs; refresh it now so the list agrees.
	if s.onPolicyChanged != nil {
		s.onPolicyChanged(ctx)
	}
	return nil
}

// OnPolicyChanged registers what runs after a policy is saved (module wiring).
func (s *ConsoleService) OnPolicyChanged(fn func(ctx context.Context)) { s.onPolicyChanged = fn }

// ---- support access ----

// SupportAccessStatus is what the Owner sees: the current consent and past sessions.
type SupportAccessStatus struct {
	Grant    *domain.SupportAccessGrant
	Sessions []domain.SupportSession
}

func (s *ConsoleService) SupportStatus(ctx context.Context, tenantID int64, now time.Time) (*SupportAccessStatus, error) {
	g, err := s.repo.LatestGrant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if g != nil && !g.ActiveAt(now) {
		g = nil // expired or revoked: nothing is granted
	}
	sessions, err := s.repo.ListSessions(ctx, tenantID, 20)
	if err != nil {
		return nil, err
	}
	return &SupportAccessStatus{Grant: g, Sessions: sessions}, nil
}

// GrantSupportAccess records the Owner's consent for 24 hours (replacing any open one).
func (s *ConsoleService) GrantSupportAccess(ctx context.Context, tenantID, staffID int64, now time.Time) (*domain.SupportAccessGrant, error) {
	if err := s.repo.RevokeGrants(ctx, tenantID, now); err != nil {
		return nil, err
	}
	g := &domain.SupportAccessGrant{TenantID: tenantID, GrantedByStaffID: staffID, GrantedAt: now, ExpiresAt: now.Add(domain.SupportAccessHours * time.Hour)}
	if err := s.repo.CreateGrant(ctx, g); err != nil {
		return nil, err
	}
	return g, nil
}

func (s *ConsoleService) RevokeSupportAccess(ctx context.Context, tenantID int64, now time.Time) error {
	return s.repo.RevokeGrants(ctx, tenantID, now)
}

// SupportSessionResult is a started session and its token.
type SupportSessionResult struct {
	Session domain.SupportSession
	Token   string
}

// StartSupportSession lets a platform admin view a workspace read-only for up
// to 60 minutes, but only while its Owner's consent is active, and only with a
// reason (FLOW-IDEAL 1.5). The session is recorded and visible to the Owner.
func (s *ConsoleService) StartSupportSession(ctx context.Context, tenantID int64, adminID, adminName, reason string, now time.Time) (*SupportSessionResult, error) {
	reason = strings.TrimSpace(reason)
	if n := utf8.RuneCountInString(reason); n < 10 || n > 255 {
		return nil, validationErr("Isi alasan melihat workspace ini (10–255 karakter)", "reason")
	}
	if _, err := s.tenants.Get(ctx, tenantID); err != nil {
		return nil, err
	}
	g, err := s.repo.LatestGrant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if g == nil || !g.ActiveAt(now) {
		return nil, apperror.Forbidden("Owner workspace ini belum memberi izin akses dukungan, atau izinnya sudah berakhir").WithCode("support_access_not_granted")
	}
	ownerID, err := s.tenants.staff.FindOwner(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if ownerID == "" {
		return nil, apperror.NotFound("Workspace ini belum punya Owner")
	}
	session := &domain.SupportSession{
		TenantID: tenantID, AdminID: adminID, AdminName: adminName, Reason: reason,
		StartedAt: now, ExpiresAt: now.Add(domain.SupportSessionMinutes * time.Minute),
	}
	if err := s.repo.CreateSession(ctx, session); err != nil {
		return nil, err
	}
	token, err := s.support.IssueSupportToken(ctx, ownerID, adminID, session.ID, domain.SupportSessionMinutes*time.Minute)
	if err != nil {
		// No token, no session: do not leave a record of a view that never happened.
		_ = s.repo.DeleteSession(ctx, session.ID)
		return nil, err
	}
	return &SupportSessionResult{Session: *session, Token: token}, nil
}

// EndSupportSession closes a session early (only by the admin who started it).
func (s *ConsoleService) EndSupportSession(ctx context.Context, id int64, adminID string, now time.Time) error {
	ok, err := s.repo.EndSession(ctx, id, adminID, now)
	if err != nil {
		return err
	}
	if !ok {
		return apperror.NotFound("Sesi dukungan tidak ditemukan atau sudah berakhir")
	}
	return nil
}

// CheckSupportSession is asked on every request made with a support token. A
// token alone is not enough: the session must still be open, the Owner's
// consent still valid, and the admin still active with a role that may look.
// Ending a session, withdrawing consent or deactivating the admin therefore
// takes effect at once, not when the token's hour is up.
func (s *ConsoleService) CheckSupportSession(ctx context.Context, tenantID, sessionID int64, adminID string, now time.Time) error {
	ended := apperror.Unauthorized("Sesi dukungan sudah berakhir").WithCode("support_session_ended")
	session, err := s.repo.FindSession(ctx, sessionID)
	if err != nil {
		return err
	}
	if session == nil || session.TenantID != tenantID || session.AdminID != adminID || session.EndedAt != nil || !now.Before(session.ExpiresAt) {
		return ended
	}
	grant, err := s.repo.LatestGrant(ctx, tenantID)
	if err != nil {
		return err
	}
	if grant == nil || !grant.ActiveAt(now) {
		return ended
	}
	if s.admins != nil {
		id, convErr := strconv.ParseInt(adminID, 10, 64)
		if convErr != nil {
			return ended
		}
		admin, err := s.admins.FindByID(ctx, id)
		if err != nil {
			return err
		}
		if admin == nil || !admin.IsActive || (admin.Role != domain.RoleSuperAdmin && admin.Role != domain.RoleSupport) {
			return ended
		}
	}
	return nil
}
