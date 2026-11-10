package presentation

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"elproof/internal/modules/vendors/domain"
	"elproof/internal/shared/apperror"
	"elproof/internal/shared/response"
)

// isActiveFilter reads ?isActive=true|false (nil when absent); anything else is
// a 422 under "isActive".
func isActiveFilter(r *http.Request) (*bool, error) {
	switch raw := r.URL.Query().Get("isActive"); raw {
	case "":
		return nil, nil
	case "true":
		v := true
		return &v, nil
	case "false":
		v := false
		return &v, nil
	default:
		return nil, apperror.Validation("Filter tidak valid", map[string][]string{"isActive": {"isActive harus true atau false"}})
	}
}

// bulkBody is the body of POST /vendors/bulk and POST /venues/bulk.
type bulkBody struct {
	IDs    []int64 `json:"ids"`
	Action string  `json:"action"`
}

// readBulk decodes and validates a bulk request: 1 to 100 distinct positive ids
// and an action of activate|deactivate; every problem is reported under its own
// key (ids, action) in one 422. A body that is not JSON is a 400.
func readBulk(w http.ResponseWriter, r *http.Request) ([]int64, domain.BulkAction, bool) {
	var body bulkBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		var typeErr *json.UnmarshalTypeError
		if errors.As(err, &typeErr) {
			key := "ids"
			if strings.HasPrefix(typeErr.Field, "action") {
				key = "action"
			}
			writeAppError(w, apperror.Validation("Permintaan massal tidak valid", map[string][]string{key: {"Format " + key + " tidak valid"}}))
			return nil, "", false
		}
		response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
		return nil, "", false
	}
	fields := map[string][]string{}
	ids, err := domain.NormalizeBulkIDs(body.IDs)
	if err != nil {
		if appErr, ok := apperror.As(err); ok {
			for k, v := range appErr.Fields {
				fields[k] = v
			}
		}
	}
	action, ok := domain.ParseBulkAction(body.Action)
	if !ok {
		fields["action"] = []string{"Aksi harus activate atau deactivate"}
	}
	if len(fields) > 0 {
		writeAppError(w, apperror.Validation("Permintaan massal tidak valid", fields))
		return nil, "", false
	}
	return ids, action, true
}
