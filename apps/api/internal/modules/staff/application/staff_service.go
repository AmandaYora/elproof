package application

import (
	"context"
	"strconv"
	"strings"

	identitycontracts "elproof/internal/modules/identity/contracts"

	"elproof/internal/modules/staff/domain"
	"elproof/internal/shared/apperror"
	"elproof/internal/shared/logger"
	"elproof/internal/shared/pagination"
	"elproof/internal/shared/validator"
)

type StaffRepository interface {
	List(ctx context.Context, tenantID int64) ([]domain.StaffMember, error)
	ListPaginated(ctx context.Context, tenantID int64, params pagination.Params, search, role string) ([]domain.StaffMember, int64, error)
	FindByID(ctx context.Context, tenantID, id int64) (*domain.StaffMember, error)
	Create(ctx context.Context, member *domain.StaffMember) error
	Update(ctx context.Context, member *domain.StaffMember) error
	SetActive(ctx context.Context, tenantID, id int64, isActive bool) error
	Delete(ctx context.Context, tenantID, id int64) error
}

type StaffService struct {
	repo     StaffRepository
	identity identitycontracts.Contracts
}

func NewStaffService(repo StaffRepository, identity identitycontracts.Contracts) *StaffService {
	return &StaffService{repo: repo, identity: identity}
}

// CreateOwner is called by `platform`'s tenant-registration orchestration —
// the only way an Owner row is ever created (Fase 3's own create-staff
// endpoint rejects role=Owner). It only writes the staff_members row; the
// caller (tenant_service.go) still separately calls identity.CreateCredential
// with the same username right after — this just keeps a denormalized copy
// here too (same pattern as `platform`'s Tenant/PlatformAdmin) so the Owner's
// username can be displayed without a cross-module lookup on every read.
func (s *StaffService) CreateOwner(ctx context.Context, tenantID int64, name, email, phone, username string) (*domain.StaffMember, error) {
	member := &domain.StaffMember{
		TenantID: tenantID,
		Name:     name,
		Title:    "Owner",
		Initials: computeInitials(name),
		Role:     domain.RoleOwner,
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
	Name     string
	Title    string
	Role     domain.StaffRole
	Username string
	Password string
	Email    string
	Phone    string
}

// Create rejects role=Owner — the Owner row is only ever created via
// `platform`'s tenant-registration orchestration (CreateOwner above). It
// provisions a real login credential (identity module) for the new
// Admin/Staff account, mirroring clients' ClientService.Create — including
// the same compensating rollback if identity.CreateCredential fails after
// the staff_members row already committed (e.g. username taken by ANY
// principal on the platform — usernames aren't tenant-scoped).
func (s *StaffService) Create(ctx context.Context, tenantID int64, input StaffInput) (*domain.StaffMember, error) {
	if input.Role == domain.RoleOwner {
		return nil, apperror.Validation("Role Owner tidak dapat dibuat dari sini", map[string][]string{
			"role": {"Akun Owner hanya dibuat otomatis saat registrasi tenant"},
		})
	}
	if err := validator.Username(input.Username); err != nil {
		return nil, err
	}
	member := &domain.StaffMember{
		TenantID: tenantID, Name: input.Name, Title: input.Title, Initials: computeInitials(input.Name),
		Role: input.Role, Username: input.Username, Email: input.Email, Phone: input.Phone, IsActive: true,
	}
	if err := s.repo.Create(ctx, member); err != nil {
		return nil, err
	}
	if err := s.identity.CreateCredential(ctx, identitycontracts.CreateCredentialInput{
		TenantID: &tenantID, PrincipalType: identitycontracts.PrincipalStaff, PrincipalID: strconv.FormatInt(member.ID, 10),
		Username: input.Username, Password: input.Password, Role: string(input.Role), DisplayName: input.Name,
	}); err != nil {
		if delErr := s.repo.Delete(ctx, tenantID, member.ID); delErr != nil {
			logger.Error("failed to roll back orphaned staff member %d after credential creation failed: %v", member.ID, delErr)
		}
		return nil, err
	}
	return member, nil
}

// Update rejects assigning role=Owner to a non-Owner row — the Owner seat is
// fixed at tenant registration and never reassigned through this endpoint.
// It also rejects editing an Owner row unless the caller IS that Owner:
// Platform Console only ever seeds the Owner's initial data, so from then on
// only the Owner themselves may adjust it (business/privacy reasons) — no
// other staff member may, even Admin. isSelf is true when the authenticated
// caller's own staff ID equals id (see staff_handler.go's isSelf check).
func (s *StaffService) Update(ctx context.Context, tenantID, id int64, isSelf bool, input StaffInput) (*domain.StaffMember, error) {
	member, err := s.Get(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if input.Role == domain.RoleOwner && member.Role != domain.RoleOwner {
		return nil, apperror.Forbidden("Role Owner tidak dapat diberikan dari sini")
	}
	if member.Role == domain.RoleOwner {
		if !isSelf {
			return nil, apperror.Forbidden("Data akun Owner hanya dapat diubah oleh pemiliknya sendiri")
		}
		if input.Role != domain.RoleOwner {
			return nil, apperror.Forbidden("Role Owner tidak dapat diubah dari sini")
		}
	}
	member.Name = input.Name
	member.Title = input.Title
	member.Role = input.Role
	member.Email = input.Email
	member.Phone = input.Phone
	member.Initials = computeInitials(input.Name)
	if err := s.repo.Update(ctx, member); err != nil {
		return nil, err
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
	if err := s.repo.SetActive(ctx, tenantID, id, isActive); err != nil {
		return nil, err
	}
	member.IsActive = isActive
	return member, nil
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
