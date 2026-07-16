// Package contracts is the ONLY package other modules may import from
// clients. It exists solely for Fase 6's Client Portal scoping: the
// `projects` module needs to know which single project a `client` principal
// is allowed to read, without importing clients' domain/application/
// infrastructure internals.
package contracts

import (
	"context"

	"elproof/internal/modules/clients/application"
	"elproof/internal/shared/apperror"
)

type Contracts interface {
	// ProjectIDForClient resolves the one project a client principal may
	// read. Returns a NotFound apperror if the client row doesn't exist
	// (or belongs to a different tenant) — callers should treat any error
	// here as "deny access", never as "allow".
	ProjectIDForClient(ctx context.Context, tenantID, clientID int64) (int64, error)
}

type impl struct {
	clients *application.ClientService
}

func New(clients *application.ClientService) Contracts {
	return &impl{clients: clients}
}

func (c *impl) ProjectIDForClient(ctx context.Context, tenantID, clientID int64) (int64, error) {
	client, err := c.clients.Get(ctx, tenantID, clientID)
	if err != nil {
		return 0, err
	}
	if !client.IsActive {
		return 0, apperror.Forbidden("Akun client ini sudah dinonaktifkan")
	}
	return client.ProjectID, nil
}
