package application

import (
	"context"
	"time"

	identitycontracts "elproof/internal/modules/identity/contracts"
	projectscontracts "elproof/internal/modules/projects/contracts"

	"elproof/internal/modules/clients/domain"
	"elproof/internal/shared/apperror"
	"elproof/internal/shared/pagination"
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
	Email        string
	Password     string
}

// Create validates project_id belongs to the caller's tenant via the
// `projects` module's contract (no cross-module FK — see ADR-0004) and
// provisions a real login credential in the same flow, mirroring how
// `platform`'s tenant registration works (ADR-0008).
func (s *ClientService) Create(ctx context.Context, tenantID int64, input CreateClientInput) (*domain.Client, error) {
	exists, err := s.projects.ProjectExists(ctx, tenantID, input.ProjectID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, apperror.Validation("Project tidak valid", map[string][]string{"projectId": {"Project tidak ditemukan"}})
	}

	c := &domain.Client{
		TenantID: tenantID, ProjectID: input.ProjectID, Role: input.Role, RelationNote: input.RelationNote,
		Name: input.Name, Phone: input.Phone, Email: input.Email, IsActive: true,
	}
	if err := s.repo.Create(ctx, c); err != nil {
		return nil, err
	}

	username := deriveUsername(input.Email)
	if err := s.identity.CreateCredential(ctx, identitycontracts.CreateCredentialInput{
		TenantID: &tenantID, PrincipalType: identitycontracts.PrincipalClient, PrincipalID: formatID(c.ID),
		Username: username, Password: input.Password, Role: string(input.Role), DisplayName: input.Name,
	}); err != nil {
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
