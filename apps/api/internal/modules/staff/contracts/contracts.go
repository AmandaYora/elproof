// Package contracts is the ONLY package other modules may import from staff.
package contracts

import (
	"context"
	"strconv"

	"elproof/internal/modules/staff/application"
)

type CreateOwnerInput struct {
	TenantID int64
	Name     string
	Email    string
	Phone    string
}

type CreateOwnerResult struct {
	StaffID string // stringified — identity.principal_id is a primitive VARCHAR
}

type Contracts interface {
	CreateOwner(ctx context.Context, input CreateOwnerInput) (CreateOwnerResult, error)
}

type impl struct {
	service *application.StaffService
}

func New(service *application.StaffService) Contracts {
	return &impl{service: service}
}

func (c *impl) CreateOwner(ctx context.Context, input CreateOwnerInput) (CreateOwnerResult, error) {
	member, err := c.service.CreateOwner(ctx, input.TenantID, input.Name, input.Email, input.Phone)
	if err != nil {
		return CreateOwnerResult{}, err
	}
	return CreateOwnerResult{StaffID: strconv.FormatInt(member.ID, 10)}, nil
}
