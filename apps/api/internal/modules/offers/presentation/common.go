// Package presentation is the HTTP surface of the offers module.
package presentation

import (
	"net/http"
	"strconv"
	"time"

	"elproof/internal/shared/apperror"
	"elproof/internal/shared/etag"
	"elproof/internal/shared/httpx"
	"elproof/internal/shared/middleware"
	"elproof/internal/shared/response"
)

// PermManage is the permission every catalog write needs.
const PermManage = "package.manage"

// requireStaffTenant answers 401/403 unless the caller is staff of a
// workspace, and returns that workspace (always from the session, ADR-0004).
func requireStaffTenant(w http.ResponseWriter, r *http.Request) (int64, bool) {
	claims, ok := middleware.FromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "Tidak terautentikasi", nil)
		return 0, false
	}
	if claims.PrincipalType == "client" {
		response.Error(w, http.StatusForbidden, "Peran Anda tidak memiliki izin untuk aksi ini", map[string]string{"code": "forbidden_permission"})
		return 0, false
	}
	tenantID, ok := claims.TenantIDInt()
	if !ok {
		response.Error(w, http.StatusForbidden, "Akun ini tidak terikat ke tenant manapun", nil)
		return 0, false
	}
	return tenantID, true
}

func can(r *http.Request, permission string) bool {
	return middleware.GrantsFrom(r.Context()).Can(permission)
}

func requirePermission(w http.ResponseWriter, r *http.Request, permission string) bool {
	if !can(r, permission) {
		response.Error(w, http.StatusForbidden, "Peran Anda tidak memiliki izin untuk aksi ini", map[string]string{"code": "forbidden_permission"})
		return false
	}
	return true
}

// itemID reads the id segment after prefix; ok=false means the answer was
// written (404 for no segment, 400 for a bad id).
func itemID(w http.ResponseWriter, r *http.Request, prefix, notFound string) (int64, []string, bool) {
	segments := httpx.Segments(r.URL.Path, prefix)
	if len(segments) == 0 {
		response.Error(w, http.StatusNotFound, notFound, nil)
		return 0, nil, false
	}
	id, err := strconv.ParseInt(segments[0], 10, 64)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "ID tidak valid", nil)
		return 0, nil, false
	}
	return id, segments, true
}

// ifMatch reads the If-Match header a full edit must carry (ADR-0036). A
// missing header answers 428 once exists() confirms the row is there (a
// missing row stays the 404); ok=false means the answer was written.
func ifMatch(w http.ResponseWriter, r *http.Request, what string, exists func() error) (*time.Time, bool) {
	expected, err := etag.ParseIfMatch(r.Header.Get("If-Match"))
	if err != nil {
		msg := "Header If-Match tidak valid. Kirim nilai updatedAt dari data yang Anda lihat."
		response.Error(w, http.StatusBadRequest, msg, map[string][]string{"If-Match": {msg}})
		return nil, false
	}
	if expected == nil {
		if err := exists(); err != nil {
			response.FromError(w, err)
			return nil, false
		}
		response.Error(w, http.StatusPreconditionRequired, "Kirim header If-Match berisi updatedAt "+what+" yang Anda ubah.", map[string][]string{"If-Match": {"wajib"}})
		return nil, false
	}
	return expected, true
}

func quoted(updatedAt string) string {
	if updatedAt == "" {
		return ""
	}
	return `"` + updatedAt + `"`
}

// staleAs swaps the payload of a Stale conflict for its response shape.
func staleAs[T any](err error, shape func(T) any) error {
	if current, ok := apperror.IsStale(err); ok {
		if v, ok := current.(T); ok {
			ae, _ := apperror.As(err)
			return ae.WithData(shape(v))
		}
	}
	return err
}

func methodNotAllowed(w http.ResponseWriter) {
	response.Error(w, http.StatusMethodNotAllowed, "Metode HTTP tidak diizinkan untuk endpoint ini", nil)
}
