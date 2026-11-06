package application

import (
	"context"
	"strings"
	"time"

	identitycontracts "elproof/internal/modules/identity/contracts"
	projectscontracts "elproof/internal/modules/projects/contracts"

	"elproof/internal/modules/clients/domain"
	"elproof/internal/shared/apperror"
	"elproof/internal/shared/logger"
	"elproof/internal/shared/pagination"
	"elproof/internal/shared/utils"
	"elproof/internal/shared/validator"
)

type ClientRepository interface {
	ListByProject(ctx context.Context, tenantID, projectID int64) ([]domain.Client, error)
	ListByTenant(ctx context.Context, tenantID int64) ([]domain.Client, error)
	ListByTenantPaginated(ctx context.Context, tenantID int64, params pagination.Params, search string) ([]domain.Client, int64, error)
	FindByID(ctx context.Context, tenantID, id int64) (*domain.Client, error)
	Create(ctx context.Context, c *domain.Client) error
	Update(ctx context.Context, c *domain.Client) error
	SetActive(ctx context.Context, tenantID, id int64, isActive bool) error
	SetCredentialResetAt(ctx context.Context, tenantID, id int64, when time.Time) error
	Delete(ctx context.Context, tenantID, id int64) error
	CountActive(ctx context.Context, tenantID int64) (int, error)
	UpdateUsername(ctx context.Context, tenantID, id int64, username string) error
}

type ClientService struct {
	repo     ClientRepository
	projects projectscontracts.Contracts
	identity identitycontracts.Contracts
	quota    TrialQuota
}

func NewClientService(repo ClientRepository, projects projectscontracts.Contracts, identity identitycontracts.Contracts) *ClientService {
	return &ClientService{repo: repo, projects: projects, identity: identity}
}

func (s *ClientService) ListByProject(ctx context.Context, tenantID, projectID int64) ([]domain.Client, error) {
	return s.repo.ListByProject(ctx, tenantID, projectID)
}

// VerifyProjectReadAccess scopes a Wedding Planner ("Staff" role) or Sales
// staff member to reading client data only for a project they're
// PIC/PIC Sales of — the same restriction `projects` itself already
// enforces on every one of its own sub-resources (see handler.go's
// resolveProjectAccess there); Owner/Admin pass through unconditionally,
// matching their full-tenant access everywhere else.
func (s *ClientService) VerifyProjectReadAccess(ctx context.Context, tenantID, projectID, staffID int64, role string) error {
	switch role {
	case "Staff":
		picStaffID, err := s.projects.ProjectPICStaffID(ctx, tenantID, projectID)
		if err != nil {
			return err
		}
		if picStaffID != staffID {
			return apperror.Forbidden("Anda tidak memiliki akses ke project ini")
		}
	case "Sales":
		if err := s.VerifyProjectSalesOwner(ctx, tenantID, projectID, staffID); err != nil {
			return err
		}
	}
	return nil
}

// VerifyProjectSalesOwner confirms a Sales-role caller is the PIC Sales of
// projectID — used both by VerifyProjectReadAccess above and by the
// handler's write-side gate (requireManagerOrProjectSales), since a Sales
// staff member's write access to Client (PLAN.md
// revisi-timeline-vendor-role-sales, confirmed: full write, unlike a
// Wedding Planner's read-only) is scoped to their own project the same way
// their read access is.
func (s *ClientService) VerifyProjectSalesOwner(ctx context.Context, tenantID, projectID, staffID int64) error {
	picSalesStaffID, err := s.projects.ProjectPICSalesStaffID(ctx, tenantID, projectID)
	if err != nil {
		return err
	}
	if picSalesStaffID != staffID {
		return apperror.Forbidden("Anda tidak memiliki akses ke project ini")
	}
	return nil
}

// ListByTenant powers the WO Console's global search, which needs to match
// client names across every project at once rather than one project at a time.
func (s *ClientService) ListByTenant(ctx context.Context, tenantID int64) ([]domain.Client, error) {
	return s.repo.ListByTenant(ctx, tenantID)
}

func (s *ClientService) ListByTenantPaginated(ctx context.Context, tenantID int64, params pagination.Params, search string) ([]domain.Client, int64, error) {
	return s.repo.ListByTenantPaginated(ctx, tenantID, params, search)
}

func (s *ClientService) Get(ctx context.Context, tenantID, id int64) (*domain.Client, error) {
	c, err := s.repo.FindByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, apperror.NotFound("Client tidak ditemukan")
	}
	return c, nil
}

type CreateClientInput struct {
	ProjectID    int64
	Role         domain.ClientRole
	RelationNote string
	Name         string
	Phone        string
	// Username is optional: empty derives one from Name. The client sets
	// their own password through the invitation link.
	Username string
	Email    string
}

// AccessLinkResult is a client invitation or reset link (/undangan/{Token}).
type AccessLinkResult struct {
	Token     string
	Purpose   string // client_invite or client_reset
	ExpiresAt time.Time
}

// Create validates project_id belongs to the caller's tenant via the
// `projects` module's contract (no cross-module FK — see ADR-0004) and
// provisions a real login credential in the same flow, mirroring how
// `platform`'s tenant registration works (ADR-0008).
func (s *ClientService) Create(ctx context.Context, tenantID int64, input CreateClientInput) (*domain.Client, *AccessLinkResult, error) {
	exists, err := s.projects.ProjectExists(ctx, tenantID, input.ProjectID)
	if err != nil {
		return nil, nil, err
	}
	if !exists {
		return nil, nil, apperror.Validation("Project tidak valid", map[string][]string{"projectId": {"Project tidak ditemukan"}})
	}
	if err := s.checkClientQuota(ctx, tenantID); err != nil {
		return nil, nil, err
	}
	input.Username, err = s.freeUsername(ctx, input.Username, input.Name)
	if err != nil {
		return nil, nil, err
	}

	c := &domain.Client{
		TenantID: tenantID, ProjectID: input.ProjectID, Role: input.Role, Username: input.Username, RelationNote: input.RelationNote,
		Name: input.Name, Phone: input.Phone, Email: input.Email, IsActive: true,
	}
	if err := s.repo.Create(ctx, c); err != nil {
		return nil, nil, err
	}

	// identity.CreateCredential can fail on its own terms (e.g. username
	// already taken by ANY principal on the platform — usernames aren't
	// tenant-scoped, see identity's credentials.username column) after the
	// client row above already committed. clients and identity own separate
	// tables in separate modules, so there's no single DB transaction to
	// wrap both in (see backend-modular-monolith rule); compensate manually
	// instead, or the client row is left orphaned with no login credential
	// — permanently unable to log in or have "Reset Credential" succeed,
	// since that also resolves through identity by principal.
	if err := s.identity.CreateCredential(ctx, identitycontracts.CreateCredentialInput{
		TenantID: &tenantID, PrincipalType: identitycontracts.PrincipalClient, PrincipalID: formatID(c.ID),
		Username: input.Username, Email: input.Email, Pending: true, Role: string(input.Role), DisplayName: input.Name,
	}); err != nil {
		if delErr := s.repo.Delete(ctx, tenantID, c.ID); delErr != nil {
			logger.Error("failed to roll back orphaned client %d after credential creation failed: %v", c.ID, delErr)
		}
		return nil, nil, err
	}
	link, err := s.identity.IssueAccessLink(ctx, identitycontracts.PrincipalClient, formatID(c.ID))
	if err != nil {
		if delErr := s.identity.DeleteCredential(ctx, identitycontracts.PrincipalClient, formatID(c.ID)); delErr != nil {
			logger.Error("failed to remove login of client %d after invitation failed: %v", c.ID, delErr)
		}
		if delErr := s.repo.Delete(ctx, tenantID, c.ID); delErr != nil {
			logger.Error("failed to roll back client %d after invitation failed: %v", c.ID, delErr)
		}
		return nil, nil, err
	}
	return c, &AccessLinkResult{Token: link.Token, Purpose: link.Purpose, ExpiresAt: link.ExpiresAt}, nil
}

type UpdateContactInput struct {
	Name  string
	Phone string
	Email string
}

func (s *ClientService) UpdateContact(ctx context.Context, tenantID, id int64, input UpdateContactInput) (*domain.Client, error) {
	c, err := s.Get(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	previousEmail := c.Email
	c.Name = input.Name
	c.Phone = input.Phone
	c.Email = input.Email
	if err := s.repo.Update(ctx, c); err != nil {
		return nil, err
	}
	if err := s.syncEmail(ctx, c, previousEmail); err != nil {
		return nil, err
	}
	return c, nil
}

// syncEmail keeps the login's email (clients may sign in with it) in step,
// rolling the row back if another login already uses it.
func (s *ClientService) syncEmail(ctx context.Context, c *domain.Client, previousEmail string) error {
	if c.Email == previousEmail {
		return nil
	}
	if err := s.identity.UpdateEmail(ctx, identitycontracts.PrincipalClient, formatID(c.ID), c.Email); err != nil {
		c.Email = previousEmail
		if rollbackErr := s.repo.Update(ctx, c); rollbackErr != nil {
			logger.Error("failed to roll back client %d email after credential sync failed: %v", c.ID, rollbackErr)
		}
		return err
	}
	return nil
}

// Delete permanently removes a client row — the one hard-delete in this
// module (every other toggle elsewhere in the codebase is soft, via
// SetActive/is_active). It exists specifically to let an operator clear out
// a client stuck with no working login credential (e.g. left behind by the
// compensating rollback in Create, or from before that rollback existed) —
// SetActive alone wouldn't free up the client's role slot for the project,
// since ProjectClientsSection looks up "the client for this role" without
// filtering on is_active. Best-effort deactivates the identity credential
// first (if one even exists) so its username isn't left silently usable;
// failure there doesn't block the delete itself, since the whole point of
// this method is to clear rows that may have no working credential at all.
func (s *ClientService) Delete(ctx context.Context, tenantID, id int64) error {
	c, err := s.Get(ctx, tenantID, id)
	if err != nil {
		return err
	}
	// Remove the login entirely so its username and email are free again.
	if err := s.identity.DeleteCredential(ctx, identitycontracts.PrincipalClient, formatID(c.ID)); err != nil {
		logger.Error("failed to remove login of client %d before delete: %v", c.ID, err)
	}
	return s.repo.Delete(ctx, tenantID, id)
}

// DeleteAllForProject best-effort deletes every client tied to a project —
// called only by `projects`' hard-delete flow (ADR-0013) via the
// ClientCleaner bridge (see projects.module.go's two-phase wiring). Continues
// past an individual client's failure (logged) rather than aborting, same
// best-effort spirit as Delete's own credential-deactivation step — the
// whole point is clearing out a project being permanently removed, not
// preserving partial state if one row is awkward.
func (s *ClientService) DeleteAllForProject(ctx context.Context, tenantID, projectID int64) error {
	list, err := s.repo.ListByProject(ctx, tenantID, projectID)
	if err != nil {
		return err
	}
	var firstErr error
	for _, c := range list {
		if err := s.Delete(ctx, tenantID, c.ID); err != nil {
			logger.Error("failed to delete client %d while cleaning up deleted project %d: %v", c.ID, projectID, err)
			if firstErr == nil {
				firstErr = err
			}
		}
	}
	return firstErr
}

func (s *ClientService) SetActive(ctx context.Context, tenantID, id int64, isActive bool) (*domain.Client, error) {
	c, err := s.Get(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if isActive && !c.IsActive {
		if err := s.checkClientQuota(ctx, tenantID); err != nil {
			return nil, err
		}
	}
	if err := s.repo.SetActive(ctx, tenantID, id, isActive); err != nil {
		return nil, err
	}
	// Keep the login identity in step: before Fase 1 a deactivated client
	// could still log in (only project reads were refused).
	if err := s.identity.SetActive(ctx, identitycontracts.PrincipalClient, formatID(c.ID), isActive); err != nil {
		_ = s.repo.SetActive(ctx, tenantID, id, !isActive)
		return nil, err
	}
	c.IsActive = isActive
	return c, nil
}

// AccessLink issues a one-time link for the client: an invitation while
// they never set a password (7 days), otherwise a password reset (24 hours).
// The WO sends it by WhatsApp; the WO never types or sees the password
// (PLAN.md §3). Earlier links stop working.
func (s *ClientService) AccessLink(ctx context.Context, tenantID, id int64) (*AccessLinkResult, error) {
	c, err := s.Get(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if !c.IsActive {
		return nil, apperror.Conflict("Aktifkan client ini dulu sebelum membuat link")
	}
	link, err := s.identity.IssueAccessLink(ctx, identitycontracts.PrincipalClient, formatID(c.ID))
	if err != nil {
		return nil, err
	}
	if link.Purpose == "client_reset" {
		now := time.Now()
		if err := s.repo.SetCredentialResetAt(ctx, tenantID, id, now); err != nil {
			logger.Error("record reset link for client %d: %v", id, err)
		}
	}
	return &AccessLinkResult{Token: link.Token, Purpose: link.Purpose, ExpiresAt: link.ExpiresAt}, nil
}

// LoginStatus is "invited" until the client set their password.
func (s *ClientService) LoginStatus(ctx context.Context, id int64) string {
	st, err := s.identity.LoginStatus(ctx, identitycontracts.PrincipalClient, formatID(id))
	if err != nil {
		logger.Error("login status for client %d: %v", id, err)
		return "unknown"
	}
	return st
}

// freeUsername returns the requested username (validated, must be free) or,
// when none was given, one derived from the client's name.
func (s *ClientService) freeUsername(ctx context.Context, requested, name string) (string, error) {
	if requested != "" {
		if err := validator.Username(requested); err != nil {
			return "", err
		}
		taken, err := s.identity.UsernameTaken(ctx, requested)
		if err != nil {
			return "", err
		}
		if taken {
			return "", apperror.Validation("Nama pengguna sudah dipakai", map[string][]string{"username": {"Nama pengguna ini sudah dipakai, pilih yang lain"}})
		}
		return requested, nil
	}
	base := utils.UsernameBase(name, "client.wo")
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

// ReplaceRepresentative overwrites the same Family Representative row with
// new contact details — no history is kept (matches the frontend mock's
// existing behavior; see PLAN.md §6.3, an explicitly open decision, not an
// oversight).
//
// The login now belongs to a different person: the previous
// representative's password, sessions and links stop working at once, and
// an invitation link for the new person is returned.
func (s *ClientService) ReplaceRepresentative(ctx context.Context, tenantID, id int64, input UpdateContactInput, relationNote string) (*domain.Client, *AccessLinkResult, error) {
	c, err := s.Get(ctx, tenantID, id)
	if err != nil {
		return nil, nil, err
	}
	if c.Role != domain.RoleFamilyRepresentative {
		return nil, nil, apperror.Forbidden("Hanya Family Representative yang dapat diganti")
	}
	if !c.IsActive {
		return nil, nil, apperror.Conflict("Aktifkan client ini dulu sebelum mengganti perwakilannya")
	}
	// Check what can fail before changing anything, then cut off the
	// previous person first: a later failure must never leave them with
	// access while the card already shows the new person.
	input.Email = strings.TrimSpace(input.Email)
	if input.Email != "" && !strings.EqualFold(input.Email, c.Email) {
		taken, err := s.identity.EmailTaken(ctx, input.Email)
		if err != nil {
			return nil, nil, err
		}
		if taken {
			return nil, nil, apperror.Validation("Email sudah dipakai", map[string][]string{"email": {"Email ini sudah dipakai akun lain"}})
		}
	}
	username, err := s.freeUsername(ctx, "", input.Name)
	if err != nil {
		return nil, nil, err
	}
	if err := s.identity.ResetSignIn(ctx, identitycontracts.PrincipalClient, formatID(c.ID), username); err != nil {
		return nil, nil, err
	}
	previousEmail := c.Email
	c.Name = input.Name
	c.Phone = input.Phone
	c.Email = input.Email
	c.RelationNote = relationNote
	c.Username = username
	if err := s.repo.Update(ctx, c); err != nil {
		return nil, nil, err
	}
	if err := s.repo.UpdateUsername(ctx, tenantID, c.ID, username); err != nil {
		return nil, nil, err
	}
	if err := s.syncEmail(ctx, c, previousEmail); err != nil {
		return nil, nil, err
	}
	link, err := s.identity.IssueAccessLink(ctx, identitycontracts.PrincipalClient, formatID(c.ID))
	if err != nil {
		return nil, nil, err
	}
	return c, &AccessLinkResult{Token: link.Token, Purpose: link.Purpose, ExpiresAt: link.ExpiresAt}, nil
}
