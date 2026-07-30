package validator

import (
	"regexp"

	"elproof/internal/shared/apperror"
)

// customDomainPattern accepts a bare hostname only -- no scheme, path, or
// port -- lowercase letters/digits/hyphens in dot-separated labels, with at
// least one dot (rejects a single-label value like "localhost", since a
// tenant's own business domain is always a real registered domain).
var customDomainPattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?(\.[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?)+$`)

// CustomDomain enforces the format for a tenant's optional custom domain
// (ADR-0015) -- the value a platform_admin types into the tenant edit form,
// later matched verbatim against the incoming request's Host header.
func CustomDomain(v string) error {
	if !customDomainPattern.MatchString(v) {
		return apperror.Validation("Domain kustom tidak valid", map[string][]string{
			"customDomain": {"Masukkan domain valid tanpa http:// atau path, cth. app.namabisnis.com"},
		})
	}
	return nil
}
