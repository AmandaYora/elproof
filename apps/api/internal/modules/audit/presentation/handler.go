// Package presentation exposes the audit log to the Platform Console.
package presentation

import (
	"context"
	"encoding/csv"
	"net/http"
	"strconv"
	"strings"
	"time"

	"elproof/internal/modules/audit/domain"

	"elproof/internal/modules/audit/application"
	"elproof/internal/shared/middleware"
	"elproof/internal/shared/pagination"
	"elproof/internal/shared/response"
)

type Handler struct {
	service *application.Service
}

func NewHandler(service *application.Service) *Handler {
	return &Handler{service: service}
}

type entryResponse struct {
	ID         int64  `json:"id"`
	OccurredAt string `json:"occurredAt"`
	ActorType  string `json:"actorType"`
	ActorID    string `json:"actorId"`
	ActorRole  string `json:"actorRole"`
	TenantID   *int64 `json:"tenantId"`
	Action     string `json:"action"`
	Method     string `json:"method"`
	Path       string `json:"path"`
	StatusCode int    `json:"statusCode"`
	Succeeded  bool   `json:"succeeded"`
	IP         string `json:"ip"`
}

func isSuperAdmin(w http.ResponseWriter, r *http.Request) (*middleware.Claims, bool) {
	claims, ok := middleware.FromContext(r.Context())
	if !ok || claims.PrincipalType != "platform_admin" || claims.Role != "Super Admin" {
		response.Error(w, http.StatusForbidden, "Hanya Super Admin yang dapat melihat audit log", nil)
		return nil, false
	}
	return claims, true
}

// filterFrom reads ?actorId=, ?actorType=, ?tenantId=, ?outcome=ok|failed,
// ?action= (exact action label), ?q= or ?search= (substring of action or path),
// ?from= and ?to= (dates, YYYY-MM-DD; "to" is inclusive of that day).
func filterFrom(r *http.Request) domain.Filter {
	q := r.URL.Query()
	f := domain.Filter{
		ActorID: q.Get("actorId"), ActorType: q.Get("actorType"), Outcome: q.Get("outcome"), Action: strings.TrimSpace(q.Get("action")),
		Query: pagination.SearchFromRequest(r),
	}
	if id, err := strconv.ParseInt(q.Get("tenantId"), 10, 64); err == nil && id > 0 {
		f.TenantID = &id
	}
	if t, err := time.ParseInLocation("2006-01-02", q.Get("from"), wib); err == nil {
		f.From = t
	}
	if t, err := time.ParseInLocation("2006-01-02", q.Get("to"), wib); err == nil {
		f.To = t.AddDate(0, 0, 1)
	}
	return f
}

// wib is Asia/Jakarta (UTC+7): the day an admin picks is an Indonesian day.
var wib = time.FixedZone("WIB", 7*60*60)

// List is GET /api/v1/audit-log — Super Admin only (who did what is itself
// sensitive). Filters: see filterFrom.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	if _, ok := isSuperAdmin(w, r); !ok {
		return
	}
	params := pagination.FromRequest(r)
	columns, defaultSort := h.service.ListSortSpec()
	sort, err := pagination.SortFromRequest(r, columns, defaultSort)
	if err != nil {
		response.FromError(w, err)
		return
	}
	entries, total, err := h.service.List(r.Context(), params, filterFrom(r), sort)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Gagal memuat audit log", nil)
		return
	}
	out := make([]entryResponse, 0, len(entries))
	for _, e := range entries {
		out = append(out, toEntryResponse(e))
	}
	response.OKPaginated(w, "ok", out, pagination.BuildMeta(params, total).WithSort(sort))
}

func toEntryResponse(e domain.Entry) entryResponse {
	return entryResponse{
		ID: e.ID, OccurredAt: e.OccurredAt.In(wib).Format("2006-01-02T15:04:05Z07:00"), ActorType: e.ActorType,
		ActorID: e.ActorID, ActorRole: e.ActorRole, TenantID: e.TenantID, Action: e.Action, Method: e.Method, Path: e.Path,
		StatusCode: e.StatusCode, Succeeded: e.Succeeded(), IP: e.IP,
	}
}

// csvCell neutralizes spreadsheet formulas (see the tenant export).
func csvCell(v string) string {
	if v != "" && strings.ContainsRune("=+-@\t\r", rune(v[0])) {
		return "'" + v
	}
	return v
}

// Export is GET /api/v1/audit-log/export: the filtered entries as CSV (at most
// application.ExportLimit rows). The export itself is written to the audit log.
func (h *Handler) Export(w http.ResponseWriter, r *http.Request) {
	claims, ok := isSuperAdmin(w, r)
	if !ok {
		return
	}
	entries, err := h.service.Export(r.Context(), filterFrom(r))
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Gagal mengekspor audit log", nil)
		return
	}
	// No record, no export: the audit entry is written before any data leaves.
	if err := h.service.Record(context.WithoutCancel(r.Context()), middleware.AuditRecord{
		ActorType: claims.PrincipalType, ActorID: claims.PrincipalID, ActorRole: claims.Role,
		Method: http.MethodGet, Path: "/api/v1/audit-log/export", StatusCode: http.StatusOK,
		IP: middleware.ClientIP(r), UserAgent: r.UserAgent(),
	}); err != nil {
		response.Error(w, http.StatusInternalServerError, "Ekspor dibatalkan karena jejak audit gagal dicatat", nil)
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="audit-log-`+time.Now().In(wib).Format("2006-01-02")+`.csv"`)
	cw := csv.NewWriter(w)
	_ = cw.Write([]string{"ID", "Waktu", "Jenis pelaku", "ID pelaku", "Peran", "Tenant", "Aksi", "Metode", "Path", "Status", "Berhasil", "IP"})
	for _, e := range entries {
		tenant := ""
		if e.TenantID != nil {
			tenant = strconv.FormatInt(*e.TenantID, 10)
		}
		ok := "tidak"
		if e.Succeeded() {
			ok = "ya"
		}
		_ = cw.Write([]string{
			strconv.FormatInt(e.ID, 10), e.OccurredAt.In(wib).Format("2006-01-02 15:04:05"), e.ActorType, csvCell(e.ActorID), csvCell(e.ActorRole),
			tenant, csvCell(e.Action), e.Method, csvCell(e.Path), strconv.Itoa(e.StatusCode), ok, e.IP,
		})
	}
	cw.Flush()
}
