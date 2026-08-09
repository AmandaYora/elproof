// Package contracts is the ONLY package other modules may import from staff.
package contracts

import (
	"context"
	"strconv"

	"elproof/internal/modules/staff/application"
	"elproof/internal/shared/apperror"
)

type CreateOwnerInput struct {
	TenantID int64
	Name     string
	Email    string
	Phone    string
	Username string
}

type CreateOwnerResult struct {
	StaffID string // stringified — identity.principal_id is a primitive VARCHAR
}

// Summary is a staff member's public-safe {id, name} projection — used by
// `projects` to embed a project's PIC name directly in its own response
// (PLAN.md's Client Portal restructure), so a client principal never needs
// to reach the staff-only `/staff/summary` endpoint just to see who their
// WO contact is.
type Summary struct {
	ID   int64
	Name string
}

type Contracts interface {
	CreateOwner(ctx context.Context, input CreateOwnerInput) (CreateOwnerResult, error)
	// GetSummary resolves one staff member by ID. Returns (nil, nil) — not an
	// error — when the ID doesn't resolve (e.g. the staff row was
	// hard-deleted, PLAN.md's Staff hard delete), so a stale pic_staff_id
	// degrades to "no name available" rather than breaking the whole
	// response it's embedded in.
	GetSummary(ctx context.Context, tenantID, staffID int64) (*Summary, error)
}

type impl struct {
	service *application.StaffService
}

func New(service *application.StaffService) Contracts {
	return &impl{service: service}
}

func (c *impl) CreateOwner(ctx context.Context, input CreateOwnerInput) (CreateOwnerResult, error) {
	member, err := c.service.CreateOwner(ctx, input.TenantID, input.Name, input.Email, input.Phone, input.Username)
	if err != nil {
		return CreateOwnerResult{}, err
	}
	return CreateOwnerResult{StaffID: strconv.FormatInt(member.ID, 10)}, nil
}

func (c *impl) GetSummary(ctx context.Context, tenantID, staffID int64) (*Summary, error) {
	member, err := c.service.Get(ctx, tenantID, staffID)
	if err != nil {
		if appErr, ok := apperror.As(err); ok && appErr.Kind == apperror.KindNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &Summary{ID: member.ID, Name: member.Name}, nil
}
