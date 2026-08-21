// Package contracts is the ONLY package other modules may import from
// platform. Its first (and so far only) consumer is `projects`' Kwitansi &
// Tagihan PDF feature (PLAN.md), which needs a tenant's real branding
// (business name, brand color, logo bytes) to render on a generated
// document — resolved authentically from the server, never trusted from a
// frontend request body (which could otherwise let a client forge another
// business's name/logo onto its own official documents).
package contracts

import (
	"context"
	"io"

	"elproof/internal/modules/platform/application"
	"elproof/internal/shared/apperror"
)

// BrandingInfo is the cross-module-safe projection of a tenant's branding --
// Logo is raw file bytes (nil when the tenant has no logo), never an object
// storage path/key: `projects` has its own storage client (for evidence) but
// has no business reading a key that belongs to platform's own storage
// namespace directly, which would be a cross-module boundary violation
// equivalent to a cross-module DB join. Resolving the bytes here, inside
// platform itself (which already owns TenantService.DownloadLogo), is the
// only correct place to do it.
type BrandingInfo struct {
	BusinessName     string
	BrandColorPreset string
	Logo             []byte
}

type Contracts interface {
	ResolveBranding(ctx context.Context, tenantID int64) (BrandingInfo, error)
}

type impl struct {
	tenants *application.TenantService
}

func New(tenants *application.TenantService) Contracts {
	return &impl{tenants: tenants}
}

// ResolveBranding treats "tenant has no logo yet" as a normal, non-error
// case (Logo stays nil) rather than propagating TenantService.DownloadLogo's
// NotFound -- a tenant without a logo must still be able to generate a
// Kwitansi/Tagihan (PLAN.md's checklist), just without an embedded image.
func (c *impl) ResolveBranding(ctx context.Context, tenantID int64) (BrandingInfo, error) {
	tenant, err := c.tenants.Get(ctx, tenantID)
	if err != nil {
		return BrandingInfo{}, err
	}
	info := BrandingInfo{BusinessName: tenant.BusinessName, BrandColorPreset: tenant.BrandColorPreset}

	reader, err := c.tenants.DownloadLogo(ctx, tenantID)
	if err != nil {
		if appErr, ok := apperror.As(err); ok && appErr.Kind == apperror.KindNotFound {
			return info, nil
		}
		return BrandingInfo{}, err
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		return BrandingInfo{}, err
	}
	info.Logo = data
	return info, nil
}
