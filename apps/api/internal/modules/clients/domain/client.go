package domain

import "time"

type ClientRole string

const (
	RoleBride               ClientRole = "Bride"
	RoleGroom               ClientRole = "Groom"
	RoleFamilyRepresentative ClientRole = "Family Representative"
)

type Client struct {
	ID                    int64
	TenantID              int64
	ProjectID             int64
	Role                  ClientRole
	Username              string
	RelationNote          string
	Name                  string
	Phone                 string
	Email                 string
	IsActive              bool
	LastCredentialResetAt *time.Time
	CreatedAt             time.Time
	UpdatedAt             time.Time
}
