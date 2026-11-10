package application

import (
	"context"
	"errors"
	"testing"

	"elproof/internal/modules/clients/domain"
	"elproof/internal/shared/apperror"
)

// An unknown client relation is refused as a validation error on `role`
// before anything is read or written (it used to reach the database ENUM and
// come back as a 500).
func TestCreateRefusesUnknownRelation(t *testing.T) {
	s := &ClientService{} // no dependencies: the check must run before any of them
	for _, role := range []domain.ClientRole{"Family", "", "bride", "Groom "} {
		_, _, err := s.Create(context.Background(), 1, CreateClientInput{ProjectID: 1, Role: role, Name: "Klien"})
		var appErr *apperror.AppError
		if !errors.As(err, &appErr) || appErr.Kind != apperror.KindValidation || len(appErr.Fields["role"]) == 0 {
			t.Errorf("role %q: want a 422 with errors.role, got %#v", role, err)
		}
	}
}
