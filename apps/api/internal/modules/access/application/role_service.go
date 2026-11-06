// Package application holds the access module's use cases: managing a
// workspace's roles and resolving what a signed-in person may do.
package application

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"elproof/internal/modules/access/domain"
	"elproof/internal/shared/apperror"
)

type RoleRepository interface {
	List(ctx context.Context, tenantID int64, kind domain.Kind) ([]*domain.Role, error)
	Find(ctx context.Context, tenantID, id int64) (*domain.Role, error)
	FindByTemplate(ctx context.Context, tenantID int64, templateKey string) (*domain.Role, error)
	Create(ctx context.Context, role *domain.Role) error
	Update(ctx context.Context, role *domain.Role) error
	Delete(ctx context.Context, tenantID, id int64) error
}

// MemberCounter tells how many logins hold a role (a role in use cannot be
// deleted). Implemented by `identity` (credentials.role_id), wired in
// cmd/server.
type MemberCounter interface {
	CountLoginsWithRole(ctx context.Context, tenantID, roleID int64) (int, error)
}

type RoleService struct {
	repo    RoleRepository
	members MemberCounter
	cache   *grantsCache
}

func NewRoleService(repo RoleRepository) *RoleService {
	return &RoleService{repo: repo, cache: newGrantsCache(30 * time.Second)}
}

// SetMemberCounter completes two-phase wiring (identity is built first, but
// keeps access free of a constructor dependency).
func (s *RoleService) SetMemberCounter(m MemberCounter) {
	s.members = m
}

func (s *RoleService) List(ctx context.Context, tenantID int64, kind domain.Kind) ([]*domain.Role, error) {
	roles, err := s.repo.List(ctx, tenantID, kind)
	if err != nil {
		return nil, err
	}
	for _, r := range roles {
		effective(r)
	}
	return roles, nil
}

func (s *RoleService) Get(ctx context.Context, tenantID, id int64) (*domain.Role, error) {
	role, err := s.repo.Find(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if role == nil {
		return nil, apperror.NotFound("Peran tidak ditemukan")
	}
	effective(role)
	return role, nil
}

// effective: the Owner role always holds every staff permission in the
// current catalog, whatever is stored (new permissions reach Owners at once).
func effective(r *domain.Role) {
	if r.IsSystem {
		r.Permissions = domain.AllKeys(domain.KindStaff)
		r.Scope = domain.ScopeAll
	}
}

// RoleInput is a create/update of a custom or built-in (non-Owner) role.
type RoleInput struct {
	Kind        domain.Kind
	Name        string
	Description string
	Scope       domain.Scope
	Permissions []string
}

func (s *RoleService) validate(in *RoleInput) error {
	fields := map[string][]string{}
	in.Name = strings.Join(strings.Fields(in.Name), " ")
	in.Description = strings.TrimSpace(in.Description)
	if n := utf8.RuneCountInString(in.Name); n < 2 || n > 60 {
		fields["name"] = []string{"Nama peran 2–60 karakter"}
	} else if strings.EqualFold(in.Name, "Owner") {
		fields["name"] = []string{"Nama 'Owner' dipakai peran pemilik workspace"}
	}
	if utf8.RuneCountInString(in.Description) > 255 {
		fields["description"] = []string{"Deskripsi maksimal 255 karakter"}
	}
	if in.Kind != domain.KindStaff && in.Kind != domain.KindClient {
		fields["kind"] = []string{"Jenis peran tidak valid"}
	}
	if in.Kind == domain.KindClient {
		in.Scope = domain.ScopeAll // a client only ever sees their own project
	} else if in.Scope != domain.ScopeAll && in.Scope != domain.ScopeAssigned {
		fields["projectScope"] = []string{"Pilih cakupan project"}
	}
	seen := map[string]bool{}
	clean := make([]string, 0, len(in.Permissions))
	for _, key := range in.Permissions {
		p, ok := domain.Lookup(key)
		if !ok || p.Kind != in.Kind {
			fields["permissions"] = []string{fmt.Sprintf("Izin %q tidak dikenal untuk peran ini", key)}
			break
		}
		if !seen[key] {
			seen[key] = true
			clean = append(clean, key)
		}
	}
	sort.Strings(clean)
	in.Permissions = clean
	if len(fields) > 0 {
		return apperror.Validation("Data peran belum valid", fields)
	}
	return nil
}

func duplicateName(err error) error {
	if errors.Is(err, domain.ErrDuplicateRoleName) {
		return apperror.Validation("Nama peran sudah dipakai", map[string][]string{"name": {"Nama peran ini sudah ada"}})
	}
	return err
}

// Grantor is the person changing roles; nil means the system itself.
// Implemented by *middleware.Grants.
type Grantor interface {
	Covers(permissions []string, allProjects bool) bool
}

// beyondGrantor refuses a staff role that allows more than the person
// changing it holds (ADR-0026): role.manage must not become a way to give
// oneself, or anyone, more access than one has. Portal roles grant nothing
// in the WO console and are not limited this way.
func beyondGrantor(by Grantor, kind domain.Kind, permissions []string, scope domain.Scope) error {
	if by == nil || kind != domain.KindStaff || by.Covers(permissions, scope == domain.ScopeAll) {
		return nil
	}
	return apperror.Forbidden("Peran ini memberi akses yang tidak Anda miliki. Minta Owner untuk mengaturnya.")
}

func (s *RoleService) Create(ctx context.Context, tenantID int64, in RoleInput, by Grantor) (*domain.Role, error) {
	if err := s.validate(&in); err != nil {
		return nil, err
	}
	if err := beyondGrantor(by, in.Kind, in.Permissions, in.Scope); err != nil {
		return nil, err
	}
	role := &domain.Role{TenantID: tenantID, Kind: in.Kind, Name: in.Name, Description: in.Description, Scope: in.Scope, Permissions: in.Permissions}
	if err := s.repo.Create(ctx, role); err != nil {
		return nil, duplicateName(err)
	}
	return s.Get(ctx, tenantID, role.ID)
}

func (s *RoleService) Update(ctx context.Context, tenantID, id int64, in RoleInput, by Grantor) (*domain.Role, error) {
	role, err := s.Get(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if role.IsSystem {
		return nil, apperror.Forbidden("Peran Owner tidak bisa diubah")
	}
	in.Kind = role.Kind // a role never changes kind
	if err := s.validate(&in); err != nil {
		return nil, err
	}
	if err := beyondGrantor(by, role.Kind, role.Permissions, role.Scope); err != nil {
		return nil, err
	}
	if err := beyondGrantor(by, in.Kind, in.Permissions, in.Scope); err != nil {
		return nil, err
	}
	role.Name, role.Description, role.Scope, role.Permissions = in.Name, in.Description, in.Scope, in.Permissions
	if err := s.repo.Update(ctx, role); err != nil {
		return nil, duplicateName(err)
	}
	s.cache.invalidate(id)
	return s.Get(ctx, tenantID, id)
}

// Delete removes a custom role nobody holds. Built-in roles stay (new staff
// and clients default to them); they can be edited instead.
func (s *RoleService) Delete(ctx context.Context, tenantID, id int64, by Grantor) error {
	role, err := s.Get(ctx, tenantID, id)
	if err != nil {
		return err
	}
	if role.IsSystem || role.TemplateKey != "" {
		return apperror.Forbidden("Peran bawaan tidak bisa dihapus. Ubah izinnya bila perlu.")
	}
	if err := beyondGrantor(by, role.Kind, role.Permissions, role.Scope); err != nil {
		return err
	}
	if s.members != nil {
		n, err := s.members.CountLoginsWithRole(ctx, tenantID, id)
		if err != nil {
			return err
		}
		if n > 0 {
			return apperror.Conflict(fmt.Sprintf("Peran ini masih dipakai %d orang. Pindahkan mereka ke peran lain dulu.", n))
		}
	}
	if err := s.repo.Delete(ctx, tenantID, id); err != nil {
		return err
	}
	s.cache.invalidate(id)
	return nil
}

// SeedDefaults gives a workspace its built-in roles; idempotent (a retried
// workspace setup only adds the missing ones).
func (s *RoleService) SeedDefaults(ctx context.Context, tenantID int64) (map[string]int64, error) {
	ids := make(map[string]int64, len(domain.Templates))
	for _, t := range domain.Templates {
		existing, err := s.repo.FindByTemplate(ctx, tenantID, t.Key)
		if err != nil {
			return nil, err
		}
		if existing != nil {
			ids[t.Key] = existing.ID
			continue
		}
		role := &domain.Role{TenantID: tenantID, Kind: t.Kind, Name: t.Name, Description: t.Description,
			TemplateKey: t.Key, IsSystem: t.System, Scope: t.Scope, Permissions: append([]string(nil), t.Permissions...)}
		if err := s.repo.Create(ctx, role); err != nil {
			return nil, err
		}
		ids[t.Key] = role.ID
	}
	return ids, nil
}

// TemplateRole returns a workspace's built-in role (seeding if missing).
func (s *RoleService) TemplateRole(ctx context.Context, tenantID int64, templateKey string) (*domain.Role, error) {
	role, err := s.repo.FindByTemplate(ctx, tenantID, templateKey)
	if err != nil {
		return nil, err
	}
	if role == nil {
		ids, err := s.SeedDefaults(ctx, tenantID)
		if err != nil {
			return nil, err
		}
		return s.Get(ctx, tenantID, ids[templateKey])
	}
	effective(role)
	return role, nil
}

// ForAssignment checks a role may be given to a person of this kind (never
// the Owner role: there is exactly one Owner, set at registration).
func (s *RoleService) ForAssignment(ctx context.Context, tenantID, roleID int64, kind domain.Kind) (*domain.Role, error) {
	role, err := s.repo.Find(ctx, tenantID, roleID)
	if err != nil {
		return nil, err
	}
	if role == nil || role.Kind != kind {
		return nil, apperror.Validation("Peran tidak valid", map[string][]string{"roleId": {"Pilih peran yang tersedia"}})
	}
	if role.IsSystem {
		return nil, apperror.Validation("Peran tidak valid", map[string][]string{"roleId": {"Peran Owner tidak bisa diberikan"}})
	}
	return role, nil
}

// Grants resolves what a role allows, cached briefly (edits invalidate).
func (s *RoleService) Grants(ctx context.Context, tenantID, roleID int64) (*domain.Role, error) {
	if r := s.cache.get(roleID); r != nil && r.TenantID == tenantID {
		return r, nil
	}
	role, err := s.repo.Find(ctx, tenantID, roleID)
	if err != nil {
		return nil, err
	}
	if role == nil {
		return nil, nil
	}
	effective(role)
	s.cache.put(role)
	return role, nil
}

type grantsCache struct {
	mu    sync.Mutex
	ttl   time.Duration
	items map[int64]cachedRole
}

type cachedRole struct {
	role    *domain.Role
	expires time.Time
}

func newGrantsCache(ttl time.Duration) *grantsCache {
	return &grantsCache{ttl: ttl, items: map[int64]cachedRole{}}
}

func (c *grantsCache) get(id int64) *domain.Role {
	c.mu.Lock()
	defer c.mu.Unlock()
	it, ok := c.items[id]
	if !ok || time.Now().After(it.expires) {
		return nil
	}
	return it.role
}

func (c *grantsCache) put(r *domain.Role) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.items) > 10000 {
		c.items = map[int64]cachedRole{}
	}
	c.items[r.ID] = cachedRole{role: r, expires: time.Now().Add(c.ttl)}
}

func (c *grantsCache) invalidate(id int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, id)
}

// MemberCount is how many logins hold a role (0 when unknown).
func (s *RoleService) MemberCount(ctx context.Context, tenantID, roleID int64) int {
	if s.members == nil {
		return 0
	}
	n, err := s.members.CountLoginsWithRole(ctx, tenantID, roleID)
	if err != nil {
		return 0
	}
	return n
}
