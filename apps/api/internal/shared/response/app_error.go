package response

import (
	"net/http"

	"elproof/internal/shared/apperror"
	"elproof/internal/shared/logger"
)

// FromError writes the error envelope for any error returned by an
// application service: typed errors keep their status, message, per-field
// validation messages and code; anything else is logged and answered with a
// generic 500 so internals never leak. Every module's writeAppError delegates
// here so the error shape cannot drift between modules.
func FromError(w http.ResponseWriter, err error) {
	status := apperror.HTTPStatus(err)
	if appErr, ok := apperror.As(err); ok {
		if appErr.Data != nil {
			ErrorWithData(w, status, appErr.Message, appErr.Payload(), appErr.Data)
			return
		}
		Error(w, status, appErr.Message, appErr.Payload())
		return
	}
	logger.Error("unhandled error: %v", err)
	Error(w, status, "Terjadi kesalahan pada server", nil)
}
