package application

import (
	"context"
	"strconv"

	identitycontracts "elproof/internal/modules/identity/contracts"
	"elproof/internal/modules/platform/domain"
	"elproof/internal/shared/apperror"
	"elproof/internal/shared/pagination"
)

type PlatformAdminRepository interface {
	List(ctx context.Context) ([]domain.PlatformAdmin, error)
	ListPaginated(ctx context.Context, params pagination.Params, search, role string) ([]domain.PlatformAdmin, int64, error)
	FindByID(ctx context.Context, id int64) (*domain.PlatformAdmin, error)
	Create(ctx context.Context, admin *domain.PlatformAdmin) error
	Update(ctx context.Context, admin *domain.PlatformAdmin) error
	SetActive(ctx context.Context, id int64, isActive bool) error
}

type PlatformAdminService struct {
	repo     PlatformAdminRepository
	identity identitycontracts.Contracts
}

func NewPlatformAdminService(repo PlatformAdminRepository, identity identitycontracts.Contracts) *PlatformAdminService {
	return &PlatformAdminService{repo: repo, identity: identity}
}

func (s *PlatformAdminService) List(ctx context.Context) ([]domain.PlatformAdmin, error) {
	return s.repo.List(ctx)
}

func (s *PlatformAdminService) ListPaginated(ctx context.Context, params pagination.Params, search, role string) ([]domain.PlatformAdmin, int64, error) {
	return s.repo.ListPaginated(ctx, params, search, role)
}

func (s *PlatformAdminService) Get(ctx context.Context, id int64) (*domain.PlatformAdmin, error) {
	admin, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if admin == nil {
		return nil, apperror.NotFound("Admin platform tidak ditemukan")
	}
	return admin, nil
}

type RegisterPlatformAdminInput struct {
	Name     string
	Title    string
	Role     domain.PlatformAdminRole
	Email    string
	Phone    string
	Password string
}

func (s *PlatformAdminService) Register(ctx context.Context, input RegisterPlatformAdminInput) (*domain.PlatformAdmin, error) {
	username := deriveUsername(input.Email)

	admin := &domain.PlatformAdmin{
		Name: input.Name, Title: input.Title, Role: input.Role,
		Username: username, Email: input.Email, Phone: input.Phone, IsActive: true,
	}
	if err := s.repo.Create(ctx, admin); err != nil {
		return nil, err
	}

	if err := s.identity.CreateCredential(ctx, identitycontracts.CreateCredentialInput{
		PrincipalType: identitycontracts.PrincipalPlatformAdmin,
		PrincipalID:   formatID(admin.ID),
		Username:      username,
		Password:      input.Password,
		Role:          string(input.Role),
		DisplayName:   input.Name,
	}); err != nil {
		return nil, err
	}

	return admin, nil
}

type UpdatePlatformAdminInput struct {
	Name  string
	Title string
	Role  domain.PlatformAdminRole
	Email string
	Phone string
}

func (s *PlatformAdminService) Update(ctx context.Context, id int64, input UpdatePlatformAdminInput) (*domain.PlatformAdmin, error) {
	admin, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	admin.Name = input.Name
	admin.Title = input.Title
	admin.Role = input.Role
	admin.Email = input.Email
	admin.Phone = input.Phone
	if err := s.repo.Update(ctx, admin); err != nil {
		return nil, err
	}
	return admin, nil
}

// SetActive deactivates/reactivates a platform admin — the presentation layer
// is responsible for preventing self-lockout (an admin deactivating their own
// account), since that's a request-context concern, not a domain rule.
func (s *PlatformAdminService) SetActive(ctx context.Context, id int64, isActive bool) (*domain.PlatformAdmin, error) {
	if _, err := s.Get(ctx, id); err != nil {
		return nil, err
	}
	if err := s.repo.SetActive(ctx, id, isActive); err != nil {
		return nil, err
	}
	if err := s.identity.SetActive(ctx, identitycontracts.PrincipalPlatformAdmin, formatID(id), isActive); err != nil {
		return nil, err
	}
	return s.Get(ctx, id)
}

func (s *PlatformAdminService) ResetPassword(ctx context.Context, id int64, newPassword string) (*domain.PlatformAdmin, error) {
	admin, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := s.identity.ResetPassword(ctx, identitycontracts.PrincipalPlatformAdmin, formatID(id), newPassword); err != nil {
		return nil, err
	}
	return admin, nil
}

func formatID(id int64) string {
	return strconv.FormatInt(id, 10)
}
