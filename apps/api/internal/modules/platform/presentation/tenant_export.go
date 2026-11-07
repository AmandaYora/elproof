package presentation

import (
	"context"
	"elproof/internal/shared/response"
	"encoding/csv"
	"net/http"
	"strconv"
	"strings"
	"time"

	"elproof/internal/modules/platform/domain"
	"elproof/internal/shared/middleware"
)

// SetAuditRecorder lets the export write its own audit record: a download is a
// read, which the request-level audit trail does not log, yet the team's list
// of every customer leaving the system is exactly what should be traceable.
func (h *TenantHandler) SetAuditRecorder(rec middleware.AuditRecorder) { h.auditor = rec }

// csvCell neutralizes spreadsheet formulas: a value that starts with =, +, -
// or @ would run when the file is opened in Excel or Sheets.
func csvCell(v string) string {
	if v != "" && strings.ContainsRune("=+-@\t\r", rune(v[0])) {
		return "'" + v
	}
	return v
}

// Export is GET /tenants/export: the filtered tenant list as CSV (Platform
// Console admins; recorded in the audit log).
func (h *TenantHandler) Export(w http.ResponseWriter, r *http.Request) {
	if !requirePlatformAdmin(w, r) {
		return
	}
	claims, _ := middleware.FromContext(r.Context())
	tenants, err := h.tenants.ListFiltered(r.Context(), tenantFilterFrom(r))
	if err != nil {
		writeAppError(w, err)
		return
	}
	plans := h.tenants.PlanNames(r.Context(), tenants)
	now := time.Now()

	// No record, no export: the audit entry is written before any data leaves.
	if h.auditor != nil {
		if err := h.auditor.Record(context.WithoutCancel(r.Context()), middleware.AuditRecord{
			ActorType: claims.PrincipalType, ActorID: claims.PrincipalID, ActorRole: claims.Role,
			Method: http.MethodGet, Path: "/api/v1/tenants/export", StatusCode: http.StatusOK,
			IP: middleware.ClientIP(r), UserAgent: r.UserAgent(),
		}); err != nil {
			response.Error(w, http.StatusInternalServerError, "Ekspor dibatalkan karena jejak audit gagal dicatat", nil)
			return
		}
	}

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="tenant-`+now.Format("2006-01-02")+`.csv"`)
	cw := csv.NewWriter(w)
	_ = cw.Write([]string{"ID", "Nama WO", "Owner", "Email", "Telepon", "Kota", "Paket", "Status langganan", "Berakhir", "Sumber daftar", "Tanggal daftar"})
	for _, t := range tenants {
		lc := domain.ComputeLifecycle(t, now, domain.CurrentSubscriptionPolicy())
		plan := ""
		if t.PlanID != nil {
			plan = plans[*t.PlanID]
		}
		ends := ""
		if lc.PeriodEndsAt != nil {
			ends = lc.PeriodEndsAt.Format("2006-01-02")
		}
		_ = cw.Write([]string{
			strconv.FormatInt(t.ID, 10), csvCell(t.BusinessName), csvCell(t.OwnerName), csvCell(t.Email), csvCell(t.Phone), csvCell(t.City),
			csvCell(plan), string(lc.Status), ends, string(t.SignupSource), t.JoinedAt.Format("2006-01-02"),
		})
	}
	cw.Flush()

}
