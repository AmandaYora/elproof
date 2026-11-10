package presentation

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"elproof/internal/modules/vendors/domain"
	"elproof/internal/shared/apperror"
	"elproof/internal/shared/pagination"
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

// categoryIDFilter reads ?categoryId= (nil when absent); a value that is not an
// integer is a 400, written here (ok=false).
func categoryIDFilter(w http.ResponseWriter, r *http.Request) (*int64, bool) {
	raw := r.URL.Query().Get("categoryId")
	if raw == "" {
		return nil, true
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "categoryId tidak valid", nil)
		return nil, false
	}
	return &id, true
}

// vendorListFilter reads the filters of GET /vendors: q (or its alias search),
// categoryId, city, isActive. GET /vendors and GET /vendors/export both call
// it, so a filtered list and its export always agree. On a bad value it writes
// the error and returns ok=false.
func vendorListFilter(w http.ResponseWriter, r *http.Request) (domain.VendorListFilter, bool) {
	categoryID, ok := categoryIDFilter(w, r)
	if !ok {
		return domain.VendorListFilter{}, false
	}
	isActive, err := isActiveFilter(r)
	if err != nil {
		writeAppError(w, err)
		return domain.VendorListFilter{}, false
	}
	return domain.VendorListFilter{
		CategoryID: categoryID, Search: pagination.SearchFromRequest(r), City: r.URL.Query().Get("city"), IsActive: isActive,
	}, true
}

// venueListFilter reads the filters of GET /venues: q (or its alias search),
// city, isActive. Shared by the list and GET /venues/export.
func venueListFilter(w http.ResponseWriter, r *http.Request) (domain.VenueListFilter, bool) {
	isActive, err := isActiveFilter(r)
	if err != nil {
		writeAppError(w, err)
		return domain.VenueListFilter{}, false
	}
	return domain.VenueListFilter{Search: pagination.SearchFromRequest(r), City: r.URL.Query().Get("city"), IsActive: isActive}, true
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
