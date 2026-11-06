package application

import (
	"context"
	"strconv"
	"strings"
	"time"

	identitycontracts "elproof/internal/modules/identity/contracts"
	projectscontracts "elproof/internal/modules/projects/contracts"

	"elproof/internal/modules/staff/domain"
	"elproof/internal/shared/apperror"
	"elproof/internal/shared/logger"
	"elproof/internal/shared/pagination"
	"elproof/internal/shared/utils"
)

type StaffRepository interface {
	List(ctx context.Context, tenantID int64) ([]domain.StaffMember, error)
	ListPaginated(ctx context.Context, tenantID int64, params pagination.Params, search, role string) ([]domain.StaffMember, int64, error)
	FindByID(ctx context.Context, tenantID, id int64) (*domain.StaffMember, error)
	Create(ctx context.Context, member *domain.StaffMember) error
	Update(ctx context.Context, member *domain.StaffMember) error
	SetActive(ctx context.Context, tenantID, id int64, isActive bool) error
	Delete(ctx context.Context, tenantID, id int64) error
	CountActiveNonOwner(ctx context.Context, tenantID int64) (int, error)
	FindOwnerID(ctx context.Context, tenantID int64) (*int64, error)
}

// ProjectReferenceLookup is the narrow shape StaffService needs from
// `projects` to check whether a staff member is still a live PIC assignment
// anywhere -- deliberately a narrow local interface (not the whole
// projects.Contracts), same idiom as ProjectService's own ClientCleaner/
// VenueResolver. Bridged from main.go via SetProjectReferenceLookup --
// needed because `staff` is built before `projects` in main.go, so `staff`
// can't take this as a constructor argument.
type ProjectReferenceLookup interface {
	ListAffectedProjectsForStaff(ctx context.Context, tenantID, staffID int64) ([]projectscontracts.ProjectRef, error)
}

// RoleResolver is the narrow slice of the access module staff needs:
// validating an assignable workspace role and naming roles. Declared here,
// wired in cmd/server.
type RoleResolver interface {
	// AssignableStaffRole describes roleID if it may be given to a staff
	// member (never the Owner role).
	AssignableStaffRole(ctx context.Context, tenantID, roleID int64) (RoleAccess, error)
	RoleNames(ctx context.Context, tenantID int64) (map[int64]string, error)
}

// RoleAccess is what staff needs to know about a workspace role.
type RoleAccess struct {
	Name        string
	Permissions []string
	AllProjects bool
}

// Grantor is the person managing the team (implemented by
// *middleware.Grants); nil means the system itself. Someone may only hand
// out, or manage people holding, roles their own role covers — so
// team.manage never becomes a way to gain more access (ADR-0026).
type Grantor interface {
	Covers(permissions []string, allProjects bool) bool
}

var errBeyondGrantor = apperror.Forbidden("Peran ini memiliki akses yang tidak Anda miliki. Minta Owner untuk mengaturnya.")

type StaffService struct {
	roles    RoleResolver
	repo     StaffRepository
	identity identitycontracts.Contracts
	projects ProjectReferenceLookup
	quota    TrialQuota
}

func NewStaffService(repo StaffRepository, identity identitycontracts.Contracts) *StaffService {
	return &StaffService{repo: repo, identity: identity}
}

// SetRoles completes two-phase wiring with the access module.
func (s *StaffService) SetRoles(r RoleResolver) {
	s.roles = r
}

// RoleNames maps role ids to their current names (nil when unavailable).
func (s *StaffService) RoleNames(ctx context.Context, tenantID int64) map[int64]string {
	if s.roles == nil {
		return nil
	}
	names, err := s.roles.RoleNames(ctx, tenantID)
	if err != nil {
		logger.Error("role names for tenant %d: %v", tenantID, err)
		return nil
	}
	return names
}

// assignableRole validates roleID and that `by` may hand it out.
func (s *StaffService) assignableRole(ctx context.Context, tenantID, roleID int64, by Grantor) (string, error) {
	if s.roles == nil {
		return "", apperror.Internal("Peran belum dapat diperiksa")
	}
	role, err := s.roles.AssignableStaffRole(ctx, tenantID, roleID)
	if err != nil {
		return "", err
	}
	if by != nil && !by.Covers(role.Permissions, role.AllProjects) {
		return "", errBeyondGrantor
	}
	return role.Name, nil
}

// CheckManageable refuses `by` changing, re-inviting, deactivating or
// deleting a member whose role allows more than `by` holds (otherwise one
// could, say, point an Admin's invitation at one's own Google account).
// Owner rows have their own, stricter rules in each operation.
func (s *StaffService) CheckManageable(ctx context.Context, tenantID, id int64, by Grantor) error {
	if by == nil {
		return nil
	}
	member, err := s.Get(ctx, tenantID, id)
	if err != nil {
		return err
	}
	if member.Role == domain.RoleOwner {
		return nil
	}
	if _, err := s.assignableRole(ctx, tenantID, member.RoleID, by); err != nil {
		return errBeyondGrantor
	}
	return nil
}

// SetProjectReferenceLookup completes the two-phase wiring described on
// ProjectReferenceLookup above. main.go calls this right after projectsModule
// is built.
func (s *StaffService) SetProjectReferenceLookup(lookup ProjectReferenceLookup) {
	s.projects = lookup
}

// CreateOwner is called by `platform`'s tenant-registration orchestration —
// the only way an Owner row is ever created (Fase 3's own create-staff
// endpoint rejects role=Owner). It only writes the staff_members row; the
// caller (tenant_service.go) still separately calls identity.CreateCredential
// with the same username right after — this just keeps a denormalized copy
// here too (same pattern as `platform`'s Tenant/PlatformAdmin) so the Owner's
// username can be displayed without a cross-module lookup on every read.
func (s *StaffService) CreateOwner(ctx context.Context, tenantID int64, name, email, phone, username string, ownerRoleID int64) (*domain.StaffMember, error) {
	member := &domain.StaffMember{
		TenantID: tenantID,
		Name:     name,
		Title:    "Owner",
		Initials: computeInitials(name),
		Role:     domain.RoleOwner,
		RoleID:   ownerRoleID,
		Username: username,
		Email:    email,
		Phone:    phone,
		IsActive: true,
	}
	if err := s.repo.Create(ctx, member); err != nil {
		return nil, err
	}
	return member, nil
}

func (s *StaffService) List(ctx context.Context, tenantID int64) ([]domain.StaffMember, error) {
	return s.repo.List(ctx, tenantID)
}

func (s *StaffService) ListPaginated(ctx context.Context, tenantID int64, params pagination.Params, search, role string) ([]domain.StaffMember, int64, error) {
	return s.repo.ListPaginated(ctx, tenantID, params, search, role)
}

func (s *StaffService) Get(ctx context.Context, tenantID, id int64) (*domain.StaffMember, error) {
	member, err := s.repo.FindByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if member == nil {
		return nil, apperror.NotFound("Pengguna tidak ditemukan")
	}
	return member, nil
}

type StaffInput struct {
	Name  string
	Title string
	// RoleID is a workspace role (Owner excluded).
	RoleID int64
	// Email is the Google account the person must sign in with to accept
	// the invitation (staff are Google-only, decision K5).
	Email string
	Phone string
}

// Invitation is a staff invitation link (token for /undangan/{token}).
type Invitation struct {
	Token     string
	ExpiresAt time.Time
}

// Create invites a new Admin/Wedding Planner/Sales member (PLAN.md §3
// "Undangan staf"): the staff row and a login with no way to sign in yet
// are created, plus a 7-day one-time link the Owner sends by WhatsApp or
// copy. The person accepts by signing in with the Google account of the
// invited email. Owner rows are only created by tenant registration.
// identity is a separate module (no shared transaction), so a failed login
// or link step deletes the staff row again.
func (s *StaffService) Create(ctx context.Context, tenantID int64, input StaffInput, by Grantor) (*domain.StaffMember, *Invitation, error) {
	roleName, err := s.assignableRole(ctx, tenantID, input.RoleID, by)
	if err != nil {
		return nil, nil, err
	}
	input.Email = strings.TrimSpace(input.Email)
	if !utils.ValidEmail(input.Email) {
		return nil, nil, apperror.Validation("Email tidak valid", map[string][]string{
			"email": {"Masukkan email akun Google orang ini"},
		})
	}
	if err := s.checkStaffQuota(ctx, tenantID); err != nil {
		return nil, nil, err
	}
	username, err := s.freeUsername(ctx, input.Email)
	if err != nil {
		return nil, nil, err
	}
	member := &domain.StaffMember{
		TenantID: tenantID, Name: input.Name, Title: input.Title, Initials: computeInitials(input.Name),
		Role: domain.StaffRole(roleName), RoleID: input.RoleID, Username: username, Email: input.Email, Phone: input.Phone, IsActive: true,
	}
	if err := s.repo.Create(ctx, member); err != nil {
		return nil, nil, err
	}
	principalID := strconv.FormatInt(member.ID, 10)
	rollback := func() {
		if delErr := s.repo.Delete(ctx, tenantID, member.ID); delErr != nil {
			logger.Error("failed to roll back staff member %d after invitation failed: %v", member.ID, delErr)
		}
	}
	if err := s.identity.CreateCredential(ctx, identitycontracts.CreateCredentialInput{
		TenantID: &tenantID, PrincipalType: identitycontracts.PrincipalStaff, PrincipalID: principalID,
		Username: username, Email: input.Email, Pending: true, Role: roleName, RoleID: input.RoleID, DisplayName: input.Name,
	}); err != nil {
		rollback()
		return nil, nil, err
	}
	link, err := s.identity.IssueAccessLink(ctx, identitycontracts.PrincipalStaff, principalID)
	if err != nil {
		if delErr := s.identity.DeleteCredential(ctx, identitycontracts.PrincipalStaff, principalID); delErr != nil {
			logger.Error("failed to remove login of staff %d after invitation failed: %v", member.ID, delErr)
		}
		rollback()
		return nil, nil, err
	}
	return member, &Invitation{Token: link.Token, ExpiresAt: link.ExpiresAt}, nil
}

// Reinvite issues a new invitation link for a member — not yet accepted,
// or who lost access to their Google account (accepting re-links the login
// to the Google account of the member's current email). Earlier links stop
// working.
func (s *StaffService) Reinvite(ctx context.Context, tenantID, id int64) (*Invitation, error) {
	member, err := s.Get(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if member.Role == domain.RoleOwner {
		return nil, apperror.Forbidden("Undangan ulang Owner dibuat oleh tim ElProof")
	}
	if !member.IsActive {
		return nil, apperror.Conflict("Aktifkan pengguna ini dulu sebelum mengirim undangan")
	}
	link, err := s.identity.IssueAccessLink(ctx, identitycontracts.PrincipalStaff, strconv.FormatInt(id, 10))
	if err != nil {
		return nil, err
	}
	return &Invitation{Token: link.Token, ExpiresAt: link.ExpiresAt}, nil
}

// LoginStatus is "invited" until the member accepted their invitation.
func (s *StaffService) LoginStatus(ctx context.Context, id int64) string {
	st, err := s.identity.LoginStatus(ctx, identitycontracts.PrincipalStaff, strconv.FormatInt(id, 10))
	if err != nil {
		logger.Error("login status for staff %d: %v", id, err)
		return "unknown"
	}
	return st
}

// freeUsername derives an internal username (staff sign in with Google and
// never type it) that no login uses yet.
func (s *StaffService) freeUsername(ctx context.Context, email string) (string, error) {
	base := utils.UsernameBase(email, "staf.wo")
	candidate := base
	for attempt := 0; attempt < 8; attempt++ {
		if attempt > 0 {
			candidate = utils.UsernameWithSuffix(base)
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

// Update rejects assigning role=Owner to a non-Owner row — the Owner seat is
// fixed at tenant registration and never reassigned through this endpoint.
// It also rejects editing an Owner row unless the caller IS that Owner:
// Platform Console only ever seeds the Owner's initial data, so from then on
// only the Owner themselves may adjust it (business/privacy reasons) — no
// other staff member may, even Admin. isSelf is true when the authenticated
// caller's own staff ID equals id (see staff_handler.go's isSelf check).
func (s *StaffService) Update(ctx context.Context, tenantID, id int64, isSelf bool, input StaffInput, by Grantor) (*domain.StaffMember, error) {
	member, err := s.Get(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	// The Owner keeps the Owner role and edits only their own row; nobody
	// else may be given the Owner role (exactly one per workspace).
	newRoleName, newRoleID := string(member.Role), member.RoleID
	if member.Role == domain.RoleOwner {
		if !isSelf {
			return nil, apperror.Forbidden("Data akun Owner hanya dapat diubah oleh pemiliknya sendiri")
		}
	} else if input.RoleID != member.RoleID {
		if isSelf {
			return nil, apperror.Forbidden("Anda tidak dapat mengubah peran Anda sendiri. Minta Owner untuk mengubahnya.")
		}
		name, err := s.assignableRole(ctx, tenantID, input.RoleID, by)
		if err != nil {
			return nil, err
		}
		newRoleName, newRoleID = name, input.RoleID
	}
	previousRole, previousRoleID, previousEmail := member.Role, member.RoleID, member.Email
	input.Email = strings.TrimSpace(input.Email)
	if input.Email != previousEmail && !utils.ValidEmail(input.Email) {
		return nil, apperror.Validation("Email tidak valid", map[string][]string{"email": {"Masukkan email akun Google orang ini"}})
	}
	member.Name = input.Name
	member.Title = input.Title
	member.Role, member.RoleID = domain.StaffRole(newRoleName), newRoleID
	member.Email = input.Email
	member.Phone = input.Phone
	member.Initials = computeInitials(input.Name)
	if err := s.repo.Update(ctx, member); err != nil {
		return nil, err
	}
	principalID := strconv.FormatInt(id, 10)
	rollback := func(what string) {
		member.Role, member.RoleID, member.Email = previousRole, previousRoleID, previousEmail
		if rollbackErr := s.repo.Update(ctx, member); rollbackErr != nil {
			logger.Error("failed to roll back staff member %d after %s sync failed: %v", id, what, rollbackErr)
		}
	}
	// The login's email is the Google account an invitation must be
	// accepted with; keep it in step (and roll back on a clash). It goes
	// first: it is the update that can fail on user input (email taken),
	// and the role sync below is then the last step.
	if member.Email != previousEmail {
		if err := s.identity.UpdateEmail(ctx, identitycontracts.PrincipalStaff, principalID, member.Email); err != nil {
			rollback("email")
			return nil, err
		}
	}
	// The login carries the role into every session; a changed role ends
	// the member's current sessions (they continue with the new one).
	if newRoleID != previousRoleID || newRoleName != string(previousRole) {
		if err := s.identity.UpdateRole(ctx, identitycontracts.PrincipalStaff, principalID, newRoleName, newRoleID); err != nil {
			if member.Email != previousEmail {
				if undoErr := s.identity.UpdateEmail(ctx, identitycontracts.PrincipalStaff, principalID, previousEmail); undoErr != nil {
					logger.Error("failed to roll back staff member %d's login email after role sync failed: %v", id, undoErr)
				}
			}
			rollback("role")
			return nil, err
		}
	}
	return member, nil
}

func (s *StaffService) SetActive(ctx context.Context, tenantID, id int64, isActive bool) (*domain.StaffMember, error) {
	member, err := s.Get(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if member.Role == domain.RoleOwner {
		return nil, apperror.Forbidden("Akun Owner tidak dapat dinonaktifkan dari sini")
	}
	if isActive && !member.IsActive {
		if err := s.checkStaffQuota(ctx, tenantID); err != nil {
			return nil, err
		}
	}
	if err := s.repo.SetActive(ctx, tenantID, id, isActive); err != nil {
		return nil, err
	}
	// Keep the login identity in step: a deactivated staff member must not
	// be able to log in or refresh (identity also revokes their sessions).
	if err := s.identity.SetActive(ctx, identitycontracts.PrincipalStaff, strconv.FormatInt(id, 10), isActive); err != nil {
		_ = s.repo.SetActive(ctx, tenantID, id, !isActive)
		return nil, err
	}
	member.IsActive = isActive
	return member, nil
}

// ListAffectedProjects backs the hard-delete confirmation dialog (PLAN.md) --
// every project this staff member is a live PIC assignment on. Returns
// (nil, nil) if the two-phase SetProjectReferenceLookup bridge hasn't been
// wired yet (should never happen once main.go finishes constructing every
// module, but avoids a nil-pointer panic if this is ever called before that).
func (s *StaffService) ListAffectedProjects(ctx context.Context, tenantID, id int64) ([]projectscontracts.ProjectRef, error) {
	if s.projects == nil {
		return nil, nil
	}
	return s.projects.ListAffectedProjectsForStaff(ctx, tenantID, id)
}

// Delete permanently removes a staff member -- an explicit, guarded exception
// to this codebase's soft-state convention (PLAN.md's hard-delete plan).
// Unlike Vendor/Venue, this carries one absolute, non-skippable rule that is
// NOT about orphaning and is NOT part of the confirm-and-proceed dialog: the
// tenant's Owner account can never be hard-deleted (a tenant losing its only
// Owner is a structural break, not a "some data goes stale" tradeoff a user
// can knowingly accept). Every other role is deletable directly, informed by
// ListAffectedProjects, mirroring Vendor/Venue's Owner-only, no-precondition
// design. The identity credential is deactivated best-effort before the row
// itself is removed, same pattern as clients.ClientService.Delete.
func (s *StaffService) Delete(ctx context.Context, tenantID, id int64) error {
	member, err := s.Get(ctx, tenantID, id)
	if err != nil {
		return err
	}
	if member.Role == domain.RoleOwner {
		return apperror.Forbidden("Akun Owner tidak dapat dihapus")
	}
	// Remove the login entirely (not just deactivate it) so the email and
	// username are free again: the same person can be invited later, and
	// their own Google sign-in is not stuck on an inactive login.
	if err := s.identity.DeleteCredential(ctx, identitycontracts.PrincipalStaff, strconv.FormatInt(member.ID, 10)); err != nil {
		return err
	}
	return s.repo.Delete(ctx, tenantID, id)
}

// computeInitials mirrors the frontend's own initials-from-name convention
// (UserFormModal's toFormValues) — up to the first two words' first letters.
func computeInitials(name string) string {
	words := strings.Fields(name)
	var b strings.Builder
	for i, w := range words {
		if i >= 2 {
			break
		}
		b.WriteString(strings.ToUpper(w[:1]))
	}
	if b.Len() == 0 {
		return "?"
	}
	return b.String()
}
