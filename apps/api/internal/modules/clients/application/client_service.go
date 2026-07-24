package application

import (
	"context"
	"time"

	identitycontracts "elproof/internal/modules/identity/contracts"
	projectscontracts "elproof/internal/modules/projects/contracts"

	"elproof/internal/modules/clients/domain"
	"elproof/internal/shared/apperror"
	"elproof/internal/shared/logger"
	"elproof/internal/shared/pagination"
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
}

type ClientService struct {
	repo     ClientRepository
	projects projectscontracts.Contracts
	identity identitycontracts.Contracts
}

func NewClientService(repo ClientRepository, projects projectscontracts.Contracts, identity identitycontracts.Contracts) *ClientService {
	return &ClientService{repo: repo, projects: projects, identity: identity}
}

func (s *ClientService) ListByProject(ctx context.Context, tenantID, projectID int64) ([]domain.Client, error) {
	return s.repo.ListByProject(ctx, tenantID, projectID)
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
	Username     string
	Email        string
	Password     string
}

// Create validates project_id belongs to the caller's tenant via the
// `projects` module's contract (no cross-module FK — see ADR-0004) and
// provisions a real login credential in the same flow, mirroring how
// `platform`'s tenant registration works (ADR-0008).
func (s *ClientService) Create(ctx context.Context, tenantID int64, input CreateClientInput) (*domain.Client, error) {
	if err := validator.Username(input.Username); err != nil {
		return nil, err
	}

	exists, err := s.projects.ProjectExists(ctx, tenantID, input.ProjectID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, apperror.Validation("Project tidak valid", map[string][]string{"projectId": {"Project tidak ditemukan"}})
	}

	c := &domain.Client{
		TenantID: tenantID, ProjectID: input.ProjectID, Role: input.Role, Username: input.Username, RelationNote: input.RelationNote,
		Name: input.Name, Phone: input.Phone, Email: input.Email, IsActive: true,
	}
	if err := s.repo.Create(ctx, c); err != nil {
		return nil, err
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
		Username: input.Username, Password: input.Password, Role: string(input.Role), DisplayName: input.Name,
	}); err != nil {
		if delErr := s.repo.Delete(ctx, tenantID, c.ID); delErr != nil {
			logger.Error("failed to roll back orphaned client %d after credential creation failed: %v", c.ID, delErr)
		}
		return nil, err
	}
	return c, nil
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
	c.Name = input.Name
	c.Phone = input.Phone
	c.Email = input.Email
	if err := s.repo.Update(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
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
	if err := s.identity.SetActive(ctx, identitycontracts.PrincipalClient, formatID(c.ID), false); err != nil {
		logger.Error("failed to deactivate credential for client %d before delete: %v", c.ID, err)
	}
	return s.repo.Delete(ctx, tenantID, id)
}

func (s *ClientService) SetActive(ctx context.Context, tenantID, id int64, isActive bool) (*domain.Client, error) {
	c, err := s.Get(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if err := s.repo.SetActive(ctx, tenantID, id, isActive); err != nil {
		return nil, err
	}
	c.IsActive = isActive
	return c, nil
}

func (s *ClientService) ResetCredential(ctx context.Context, tenantID, id int64, newPassword string) (*domain.Client, error) {
	c, err := s.Get(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if err := s.identity.ResetPassword(ctx, identitycontracts.PrincipalClient, formatID(c.ID), newPassword); err != nil {
		return nil, err
	}
	now := time.Now()
	if err := s.repo.SetCredentialResetAt(ctx, tenantID, id, now); err != nil {
		return nil, err
	}
	c.LastCredentialResetAt = &now
	return c, nil
}

// ReplaceRepresentative overwrites the same Family Representative row with
// new contact details — no history is kept (matches the frontend mock's
// existing behavior; see PLAN.md §6.3, an explicitly open decision, not an
// oversight).
func (s *ClientService) ReplaceRepresentative(ctx context.Context, tenantID, id int64, input UpdateContactInput, relationNote string) (*domain.Client, error) {
	c, err := s.Get(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if c.Role != domain.RoleFamilyRepresentative {
		return nil, apperror.Forbidden("Hanya Family Representative yang dapat diganti")
	}
	c.Name = input.Name
	c.Phone = input.Phone
	c.Email = input.Email
	c.RelationNote = relationNote
	if err := s.repo.Update(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}
