// Package contracts is the only package other modules may import from
// access: workspace role provisioning and lookups (ADR-0026).
package contracts

import (
	"context"

	"elproof/internal/modules/access/application"
	"elproof/internal/modules/access/domain"
)

// Built-in role templates.
const (
	TemplateOwner          = domain.TemplateOwner
	TemplateAdmin          = domain.TemplateAdmin
	TemplateWeddingPlanner = domain.TemplateWeddingPlanner
	TemplateSales          = domain.TemplateSales
	TemplateFinance        = domain.TemplateFinance
	TemplateBride          = domain.TemplateBride
	TemplateFamily         = domain.TemplateFamily
)

// Kinds of roles.
const (
	KindStaff  = string(domain.KindStaff)
	KindClient = string(domain.KindClient)
)

// RoleRef is what other modules keep about a role: its id (stored by value)
// and name (shown in lists).
type RoleRef struct {
	ID   int64
	Name string
	// Permissions and AllProjects describe what the role allows; filled by
	// RoleForAssignment so callers can refuse handing out more than the
	// person assigning it holds.
	Permissions []string
	AllProjects bool
}

type Contracts interface {
	// SeedDefaultRoles gives a new workspace its built-in roles (idempotent).
	SeedDefaultRoles(ctx context.Context, tenantID int64) error
	// TemplateRole returns a workspace's built-in role (e.g. owner, bride).
	TemplateRole(ctx context.Context, tenantID int64, templateKey string) (RoleRef, error)
	// RoleForAssignment validates that roleID may be given to a person of
	// this kind ("staff"/"client"); never the Owner role.
	RoleForAssignment(ctx context.Context, tenantID, roleID int64, kind string) (RoleRef, error)
	// RoleNames maps a workspace's role ids to their current names.
	RoleNames(ctx context.Context, tenantID int64) (map[int64]string, error)
}

type impl struct {
	roles *application.RoleService
}

func New(roles *application.RoleService) Contracts {
	return &impl{roles: roles}
}

func (c *impl) SeedDefaultRoles(ctx context.Context, tenantID int64) error {
	_, err := c.roles.SeedDefaults(ctx, tenantID)
	return err
}

func (c *impl) TemplateRole(ctx context.Context, tenantID int64, templateKey string) (RoleRef, error) {
	r, err := c.roles.TemplateRole(ctx, tenantID, templateKey)
	if err != nil {
		return RoleRef{}, err
	}
	return RoleRef{ID: r.ID, Name: r.Name}, nil
}

func (c *impl) RoleForAssignment(ctx context.Context, tenantID, roleID int64, kind string) (RoleRef, error) {
	r, err := c.roles.ForAssignment(ctx, tenantID, roleID, domain.Kind(kind))
	if err != nil {
		return RoleRef{}, err
	}
	return RoleRef{ID: r.ID, Name: r.Name, Permissions: r.Permissions, AllProjects: r.Scope == domain.ScopeAll}, nil
}

func (c *impl) RoleNames(ctx context.Context, tenantID int64) (map[int64]string, error) {
	roles, err := c.roles.List(ctx, tenantID, "")
	if err != nil {
		return nil, err
	}
	m := make(map[int64]string, len(roles))
	for _, r := range roles {
		m[r.ID] = r.Name
	}
	return m, nil
}
