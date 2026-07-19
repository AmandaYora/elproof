package validator

import (
	"regexp"

	"elproof/internal/shared/apperror"
)

var usernamePattern = regexp.MustCompile(`^[a-z0-9_.]{4,32}$`)

// Username enforces the format shared by every registration flow that
// provisions a login credential (tenant Owner, platform admin, client) —
// the value is always operator-supplied, never derived.
func Username(v string) error {
	if !usernamePattern.MatchString(v) {
		return apperror.Validation("Username tidak valid", map[string][]string{
			"username": {"Username 4-32 karakter, huruf kecil, angka, underscore, atau titik"},
		})
	}
	return nil
}
