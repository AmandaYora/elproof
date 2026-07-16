package application

import (
	"context"
	"strings"

	"elproof/internal/modules/staff/domain"
	"elproof/internal/shared/apperror"
	"elproof/internal/shared/pagination"
)

type StaffRepository interface {
	List(ctx context.Context, tenantID int64) ([]domain.StaffMember, error)
	ListPaginated(ctx context.Context, tenantID int64, params pagination.Params, search, role string) ([]domain.StaffMember, int64, error)
	FindByID(ctx context.Context, tenantID, id int64) (*domain.StaffMember, error)
	Create(ctx context.Context, member *domain.StaffMember) error
	Update(ctx context.Context, member *domain.StaffMember) error
	SetActive(ctx context.Context, tenantID, id int64, isActive bool) error
}

type StaffService struct {
	repo StaffRepository
}

func NewStaffService(repo StaffRepository) *StaffService {
	return &StaffService{repo: repo}
}

// CreateOwner is called by `platform`'s tenant-registration orchestration —
// the only way an Owner row is ever created (Fase 3's own create-staff
// endpoint rejects role=Owner).
func (s *StaffService) CreateOwner(ctx context.Context, tenantID int64, name, email, phone string) (*domain.StaffMember, error) {
	member := &domain.StaffMember{
		TenantID: tenantID,
		Name:     name,
		Title:    "Owner",
		Initials: computeInitials(name),
		Role:     domain.RoleOwner,
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
	Role  domain.StaffRole
	Email string
	Phone string
}

// Create rejects role=Owner — the Owner row is only ever created via
// `platform`'s tenant-registration orchestration (CreateOwner above).
func (s *StaffService) Create(ctx context.Context, tenantID int64, input StaffInput) (*domain.StaffMember, error) {
	if input.Role == domain.RoleOwner {
		return nil, apperror.Validation("Role Owner tidak dapat dibuat dari sini", map[string][]string{
			"role": {"Akun Owner hanya dibuat otomatis saat registrasi tenant"},
		})
	}
	member := &domain.StaffMember{
		TenantID: tenantID, Name: input.Name, Title: input.Title, Initials: computeInitials(input.Name),
		Role: input.Role, Email: input.Email, Phone: input.Phone, IsActive: true,
	}
	if err := s.repo.Create(ctx, member); err != nil {
		return nil, err
	}
	return member, nil
}

// Update rejects changing role to/from Owner — the Owner seat is fixed at
// tenant registration and never reassigned through this endpoint.
func (s *StaffService) Update(ctx context.Context, tenantID, id int64, input StaffInput) (*domain.StaffMember, error) {
	member, err := s.Get(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if member.Role == domain.RoleOwner || input.Role == domain.RoleOwner {
		return nil, apperror.Forbidden("Role Owner tidak dapat diubah dari sini")
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
