// Package presentation exposes the projects module's large surface as a
// single dispatcher (net/http.ServeMux can't register overlapping prefix
// patterns, so /api/v1/projects/ is one subtree handled entirely here,
// split across files by concern for readability — see each file's own
// endpoints).
package presentation

import (
	"context"
	"net/http"
	"strconv"

	"elproof/internal/modules/projects/application"
	"elproof/internal/shared/apperror"
	"elproof/internal/shared/httpx"
	"elproof/internal/shared/middleware"
	"elproof/internal/shared/response"
)

// ClientAccessResolver is the narrow shape this module needs from `clients`
// to scope a `client` principal's read access to their own project (Fase 6)
// — deliberately a local interface (not an import of clients/contracts) so
// `projects` never has to import `clients`. main.go bridges the two via
// SetClientAccessResolver once both modules exist (see projects.module.go's
// two-phase wiring note).
type ClientAccessResolver interface {
	ProjectIDForClient(ctx context.Context, tenantID, clientID int64) (int64, error)
}

type Handler struct {
	projects     *application.ProjectService
	vendors      *application.VendorEngagementService
	payments     *application.PaymentService
	issues       *application.IssueService
	evidence     *application.EvidenceService
	activity     *application.ActivityService
	dashboard    *application.DashboardService
	clientAccess ClientAccessResolver
}

func NewHandler(
	projects *application.ProjectService,
	vendors *application.VendorEngagementService,
	payments *application.PaymentService,
	issues *application.IssueService,
	evidence *application.EvidenceService,
	activity *application.ActivityService,
	dashboard *application.DashboardService,
) *Handler {
	return &Handler{projects: projects, vendors: vendors, payments: payments, issues: issues, evidence: evidence, activity: activity, dashboard: dashboard}
}

// SetClientAccessResolver completes the two-phase wiring needed to break the
// projects<->clients construction cycle (clients.NewModule needs
// projects.Contracts() already built, so projects can't take a clients
// dependency at construction time) — see projects.module.go.
func (h *Handler) SetClientAccessResolver(resolver ClientAccessResolver) {
	h.clientAccess = resolver
}

func (h *Handler) Collection(w http.ResponseWriter, r *http.Request) {
	claims, ok := requireStaff(w, r)
	if !ok {
		return
	}
	switch r.Method {
	case http.MethodGet:
		h.listProjects(w, r, claims.tenantID)
	case http.MethodPost:
		h.createProject(w, r, claims)
	default:
		response.Error(w, http.StatusMethodNotAllowed, "Metode HTTP tidak diizinkan untuk endpoint ini", nil)
	}
}

// Item dispatches the entire /api/v1/projects/{id}/... subtree. Staff get
// full read/write access to any project in their own tenant; a `client`
// principal gets read-only access to exactly one project — their own (Fase
// 6) — enforced here, not just as a frontend convention.
func (h *Handler) Item(w http.ResponseWriter, r *http.Request) {
	segments := httpx.Segments(r.URL.Path, "/api/v1/projects/")
	if len(segments) == 0 {
		response.Error(w, http.StatusNotFound, "Project tidak ditemukan", nil)
		return
	}

	// "me" is the client principal's own self-service read — see me()'s doc
	// comment. Everything else under /projects/ is the numeric-id subtree.
	if segments[0] == "me" && len(segments) == 1 && r.Method == http.MethodGet {
		h.me(w, r)
		return
	}

	projectID, err := strconv.ParseInt(segments[0], 10, 64)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "ID project tidak valid", nil)
		return
	}
	rest := segments[1:]

	claims, ok := h.resolveProjectAccess(w, r, projectID)
	if !ok {
		return
	}

	switch {
	case len(rest) == 0 && r.Method == http.MethodGet:
		h.getProject(w, r, claims.tenantID, projectID)
	case len(rest) == 0 && r.Method == http.MethodPatch:
		h.updateProject(w, r, claims, projectID)
	case len(rest) == 1 && rest[0] == "cancel" && r.Method == http.MethodPost:
		h.cancelProject(w, r, claims, projectID)
	case len(rest) == 1 && rest[0] == "milestones" && r.Method == http.MethodGet:
		h.listMilestones(w, r, claims.tenantID, projectID)
	case len(rest) == 1 && rest[0] == "milestones" && r.Method == http.MethodPost:
		h.createMilestone(w, r, claims, projectID)
	case len(rest) == 2 && rest[0] == "milestones" && r.Method == http.MethodPatch:
		h.updateMilestoneStatus(w, r, claims, projectID, rest[1])
	case len(rest) == 1 && rest[0] == "vendors" && r.Method == http.MethodGet:
		h.listVendorEngagements(w, r, projectID)
	case len(rest) == 1 && rest[0] == "vendors" && r.Method == http.MethodPost:
		h.createVendorEngagement(w, r, claims, projectID)
	case len(rest) == 2 && rest[0] == "vendors" && r.Method == http.MethodPatch:
		h.updateVendorEngagement(w, r, claims, projectID, rest[1])
	case len(rest) == 3 && rest[0] == "vendors" && rest[2] == "cancel" && r.Method == http.MethodPost:
		h.cancelVendorEngagement(w, r, claims, projectID, rest[1])
	case len(rest) == 3 && rest[0] == "vendors" && rest[2] == "milestones" && r.Method == http.MethodGet:
		h.listVendorMilestones(w, r, projectID, rest[1])
	case len(rest) == 3 && rest[0] == "vendors" && rest[2] == "milestones" && r.Method == http.MethodPost:
		h.createVendorMilestone(w, r, claims, projectID, rest[1])
	case len(rest) == 4 && rest[0] == "vendors" && rest[2] == "milestones" && r.Method == http.MethodPatch:
		h.updateVendorMilestone(w, r, claims, projectID, rest[1], rest[3])
	case len(rest) == 1 && rest[0] == "payments" && r.Method == http.MethodGet:
		h.listPayments(w, r, projectID)
	case len(rest) == 1 && rest[0] == "payments" && r.Method == http.MethodPost:
		h.createPayment(w, r, claims, projectID)
	case len(rest) == 1 && rest[0] == "issues" && r.Method == http.MethodGet:
		h.listIssues(w, r, projectID)
	case len(rest) == 1 && rest[0] == "issues" && r.Method == http.MethodPost:
		h.createIssue(w, r, claims, projectID)
	case len(rest) == 2 && rest[0] == "issues" && r.Method == http.MethodPatch:
		h.updateIssueStatus(w, r, claims, projectID, rest[1])
	case len(rest) == 1 && rest[0] == "evidence" && r.Method == http.MethodGet:
		h.listEvidence(w, r, projectID)
	case len(rest) == 1 && rest[0] == "evidence" && r.Method == http.MethodPost:
		h.uploadEvidence(w, r, claims, projectID)
	case len(rest) == 3 && rest[0] == "evidence" && rest[2] == "file" && r.Method == http.MethodGet:
		h.downloadEvidence(w, r, projectID, rest[1])
	case len(rest) == 1 && rest[0] == "activity" && r.Method == http.MethodGet:
		h.listActivity(w, r, projectID)
	default:
		response.Error(w, http.StatusNotFound, "Endpoint tidak ditemukan", nil)
	}
}

// staffClaims is the narrow, parsed set of claims every handler in this
// module needs — parsing tenantID/staffID once here keeps every endpoint
// method below from repeating the same boilerplate.
type staffClaims struct {
	tenantID int64
	staffID  int64
	role     string
}

func requireStaff(w http.ResponseWriter, r *http.Request) (staffClaims, bool) {
	claims, ok := middleware.FromContext(r.Context())
	if !ok || claims.PrincipalType != "staff" {
		response.Error(w, http.StatusForbidden, "Hanya staff WO yang dapat mengakses endpoint ini", nil)
		return staffClaims{}, false
	}
	tenantID, ok := claims.TenantIDInt()
	if !ok {
		response.Error(w, http.StatusForbidden, "Akun ini tidak terikat ke tenant manapun", nil)
		return staffClaims{}, false
	}
	staffID, err := strconv.ParseInt(claims.PrincipalID, 10, 64)
	if err != nil {
		response.Error(w, http.StatusForbidden, "Identitas staff tidak valid", nil)
		return staffClaims{}, false
	}
	return staffClaims{tenantID: tenantID, staffID: staffID, role: claims.Role}, true
}

// resolveProjectAccess authorizes the /projects/{id}/... subtree for either
// principal type. Staff get the existing full access (any project in their
// tenant). A client principal only ever gets a synthetic staffClaims with
// staffID/role left zero-valued — safe because every write branch in Item
// requires POST/PATCH, which this function already rejects for clients
// before the big switch is ever reached, so those zero values are never read.
func (h *Handler) resolveProjectAccess(w http.ResponseWriter, r *http.Request, projectID int64) (staffClaims, bool) {
	claims, ok := middleware.FromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "Tidak terautentikasi", nil)
		return staffClaims{}, false
	}

	switch claims.PrincipalType {
	case "staff":
		return requireStaff(w, r)
	case "client":
		if r.Method != http.MethodGet {
			response.Error(w, http.StatusForbidden, "Client hanya memiliki akses baca", nil)
			return staffClaims{}, false
		}
		tenantID, ok := claims.TenantIDInt()
		if !ok {
			response.Error(w, http.StatusForbidden, "Akun ini tidak terikat ke tenant manapun", nil)
			return staffClaims{}, false
		}
		clientID, err := strconv.ParseInt(claims.PrincipalID, 10, 64)
		if err != nil {
			response.Error(w, http.StatusForbidden, "Identitas client tidak valid", nil)
			return staffClaims{}, false
		}
		if h.clientAccess == nil {
			response.Error(w, http.StatusForbidden, "Akses client belum dikonfigurasi", nil)
			return staffClaims{}, false
		}
		allowedProjectID, err := h.clientAccess.ProjectIDForClient(r.Context(), tenantID, clientID)
		if err != nil || allowedProjectID != projectID {
			response.Error(w, http.StatusForbidden, "Anda tidak memiliki akses ke project ini", nil)
			return staffClaims{}, false
		}
		return staffClaims{tenantID: tenantID}, true
	default:
		response.Error(w, http.StatusForbidden, "Prinsipal ini tidak dapat mengakses project", nil)
		return staffClaims{}, false
	}
}

func writeAppError(w http.ResponseWriter, err error) {
	status := apperror.HTTPStatus(err)
	if appErr, ok := apperror.As(err); ok {
		if appErr.Kind == apperror.KindValidation {
			response.Error(w, status, appErr.Message, appErr.Fields)
			return
		}
		response.Error(w, status, appErr.Message, nil)
		return
	}
	response.Error(w, status, "Terjadi kesalahan pada server", nil)
}
