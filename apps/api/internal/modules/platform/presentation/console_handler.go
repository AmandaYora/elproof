package presentation

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"elproof/internal/modules/platform/application"
	"elproof/internal/modules/platform/domain"
	"elproof/internal/shared/middleware"
	"elproof/internal/shared/response"
)

// ConsoleHandler serves the Platform Console's back-office endpoints
// (/api/v1/console/...) and the Owner-side support-access and announcement
// endpoints. The role policy (who may write what) is enforced once in
// cmd/server; handlers here only check the principal type.
type ConsoleHandler struct {
	console *application.ConsoleService
	tenants *application.TenantService
}

func NewConsoleHandler(console *application.ConsoleService, tenants *application.TenantService) *ConsoleHandler {
	return &ConsoleHandler{console: console, tenants: tenants}
}

const timeLayout = time.RFC3339

// wib is how every time leaves the API: the server may run in UTC, but the
// product speaks Western Indonesia Time (the web shows these strings as-is).
var wib = time.FixedZone("WIB", 7*60*60)

func tstr(t time.Time) string { return t.In(wib).Format(timeLayout) }
func tptr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := tstr(*t)
	return &s
}

func decode(w http.ResponseWriter, r *http.Request, v interface{}) bool {
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(v); err != nil {
		response.Error(w, http.StatusBadRequest, "Body permintaan tidak valid", nil)
		return false
	}
	return true
}

func adminOf(w http.ResponseWriter, r *http.Request) (*middleware.Claims, bool) {
	claims, ok := middleware.FromContext(r.Context())
	if !ok || claims.PrincipalType != "platform_admin" {
		response.Error(w, http.StatusForbidden, "Hanya admin platform yang dapat melakukan aksi ini", nil)
		return nil, false
	}
	return claims, true
}

func idAt(segments []string, i int) (int64, bool) {
	if len(segments) <= i {
		return 0, false
	}
	id, err := strconv.ParseInt(segments[i], 10, 64)
	return id, err == nil && id > 0
}

// Console routes everything under /api/v1/console/.
func (h *ConsoleHandler) Console(w http.ResponseWriter, r *http.Request) {
	if _, ok := adminOf(w, r); !ok {
		return
	}
	s := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/console"), "/"), "/")
	switch {
	case len(s) == 3 && s[0] == "tenants" && s[2] == "overview" && r.Method == http.MethodGet:
		h.overview(w, r, s)
	case len(s) == 3 && s[0] == "tenants" && s[2] == "notes" && r.Method == http.MethodGet:
		h.listNotes(w, r, s)
	case len(s) == 3 && s[0] == "tenants" && s[2] == "notes" && r.Method == http.MethodPost:
		h.addNote(w, r, s)
	case len(s) == 3 && s[0] == "tenants" && s[2] == "data-requests" && r.Method == http.MethodGet:
		h.listTenantDataRequests(w, r, s)
	case len(s) == 3 && s[0] == "tenants" && s[2] == "data-requests" && r.Method == http.MethodPost:
		h.addDataRequest(w, r, s)
	case len(s) == 1 && s[0] == "data-requests" && r.Method == http.MethodGet:
		h.listDataRequests(w, r)
	case len(s) == 3 && s[0] == "data-requests" && s[2] == "complete" && r.Method == http.MethodPost:
		h.completeDataRequest(w, r, s)
	case len(s) == 1 && s[0] == "announcements" && r.Method == http.MethodGet:
		h.listAnnouncements(w, r)
	case len(s) == 1 && s[0] == "announcements" && r.Method == http.MethodPost:
		h.saveAnnouncement(w, r, 0)
	case len(s) == 2 && s[0] == "announcements" && r.Method == http.MethodPatch:
		if id, ok := idAt(s, 1); ok {
			h.saveAnnouncement(w, r, id)
			return
		}
		response.Error(w, http.StatusBadRequest, "ID tidak valid", nil)
	case len(s) == 2 && s[0] == "announcements" && r.Method == http.MethodDelete:
		h.deleteAnnouncement(w, r, s)
	case len(s) == 1 && s[0] == "subscription-policy" && r.Method == http.MethodGet:
		h.getPolicy(w, r)
	case len(s) == 1 && s[0] == "subscription-policy" && r.Method == http.MethodPut:
		h.putPolicy(w, r)
	case len(s) == 3 && s[0] == "tenants" && s[2] == "support" && r.Method == http.MethodGet:
		h.adminSupportStatus(w, r, s)
	case len(s) == 3 && s[0] == "tenants" && s[2] == "impersonate" && r.Method == http.MethodPost:
		h.impersonate(w, r, s)
	case len(s) == 3 && s[0] == "support-sessions" && s[2] == "end" && r.Method == http.MethodPost:
		h.endSession(w, r, s)
	default:
		response.Error(w, http.StatusNotFound, "Endpoint tidak ditemukan", nil)
	}
}

// ---- overview ----

type memberResponse struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	IsActive bool   `json:"isActive"`
}

type activationResponse struct {
	ID             int64  `json:"id"`
	PlanID         int64  `json:"planId"`
	Admin          string `json:"admin"`
	Reason         string `json:"reason"`
	ReasonLabel    string `json:"reasonLabel"`
	ProofReference string `json:"proofReference"`
	Note           string `json:"note"`
	ExpiresAt      string `json:"expiresAt"`
	CreatedAt      string `json:"createdAt"`
}

type consentResponse struct {
	TermsVersion    string `json:"termsVersion"`
	TermsAcceptedAt string `json:"termsAcceptedAt"`
	MarketingOptIn  bool   `json:"marketingOptIn"`
	IPAddress       string `json:"ipAddress"`
}

type usageItem struct {
	Used int  `json:"used"`
	Max  *int `json:"max"` // nil: no limit (paid plan)
}

type overviewResponse struct {
	Tenant           tenantResponse       `json:"tenant"`
	PlanName         string               `json:"planName"`
	Consent          *consentResponse     `json:"consent"`
	Activations      []activationResponse `json:"activations"`
	Staff            usageItem            `json:"staff"`
	ActiveProjects   usageItem            `json:"activeProjects"`
	Clients          usageItem            `json:"clients"`
	Members          []memberResponse     `json:"members"`
	OpenDataRequests int                  `json:"openDataRequests"`
	SupportAccessTo  *string              `json:"supportAccessUntil"`
}

func (h *ConsoleHandler) overview(w http.ResponseWriter, r *http.Request, s []string) {
	id, ok := idAt(s, 1)
	if !ok {
		response.Error(w, http.StatusBadRequest, "ID tenant tidak valid", nil)
		return
	}
	o, err := h.console.Overview(r.Context(), id, time.Now())
	if err != nil {
		writeAppError(w, err)
		return
	}
	res := overviewResponse{
		Tenant: toTenantResponse(o.Tenant), PlanName: o.PlanName, OpenDataRequests: o.OpenDataRequests,
		Activations: make([]activationResponse, 0, len(o.Activations)), Members: make([]memberResponse, 0, len(o.Members)),
	}
	if o.Consent != nil {
		res.Consent = &consentResponse{TermsVersion: o.Consent.TermsVersion, TermsAcceptedAt: tstr(o.Consent.TermsAcceptedAt), MarketingOptIn: o.Consent.MarketingOptIn, IPAddress: o.Consent.IPAddress}
	}
	for _, a := range o.Activations {
		res.Activations = append(res.Activations, activationResponse{
			ID: a.ID, PlanID: a.PlanID, Admin: a.ActorAdminID, Reason: string(a.Reason), ReasonLabel: domain.ManualActivationReasonLabels[a.Reason],
			ProofReference: a.ProofReference, Note: a.Note, ExpiresAt: tstr(a.ExpiresAt), CreatedAt: tstr(a.CreatedAt),
		})
	}
	for _, m := range o.Members {
		res.Members = append(res.Members, memberResponse{ID: m.ID, Name: m.Name, Email: m.Email, Role: m.Role, IsActive: m.IsActive})
	}
	limited := o.Usage.Limits != nil
	capOf := func(max int) *int {
		if !limited {
			return nil
		}
		return &max
	}
	var lim domain.TrialLimits
	if limited {
		lim = *o.Usage.Limits
	}
	res.Staff = usageItem{Used: o.Usage.Staff, Max: capOf(lim.MaxStaff)}
	res.ActiveProjects = usageItem{Used: o.Usage.ActiveProjects, Max: capOf(lim.MaxActiveProjects)}
	res.Clients = usageItem{Used: o.Usage.Clients, Max: capOf(lim.MaxClients)}
	if o.SupportAccess != nil {
		res.SupportAccessTo = tptr(&o.SupportAccess.ExpiresAt)
	}
	response.OK(w, "ok", res)
}

// ---- notes ----

type noteResponse struct {
	ID        int64  `json:"id"`
	Admin     string `json:"admin"`
	Body      string `json:"body"`
	CreatedAt string `json:"createdAt"`
}

func (h *ConsoleHandler) listNotes(w http.ResponseWriter, r *http.Request, s []string) {
	id, ok := idAt(s, 1)
	if !ok {
		response.Error(w, http.StatusBadRequest, "ID tenant tidak valid", nil)
		return
	}
	notes, err := h.console.ListNotes(r.Context(), id)
	if err != nil {
		writeAppError(w, err)
		return
	}
	out := make([]noteResponse, 0, len(notes))
	for _, n := range notes {
		out = append(out, noteResponse{ID: n.ID, Admin: n.AdminName, Body: n.Body, CreatedAt: tstr(n.CreatedAt)})
	}
	response.OK(w, "ok", out)
}

func (h *ConsoleHandler) addNote(w http.ResponseWriter, r *http.Request, s []string) {
	claims, _ := adminOf(w, r)
	id, ok := idAt(s, 1)
	if !ok {
		response.Error(w, http.StatusBadRequest, "ID tenant tidak valid", nil)
		return
	}
	var body struct {
		Body string `json:"body"`
	}
	if !decode(w, r, &body) {
		return
	}
	n, err := h.console.AddNote(r.Context(), id, claims.PrincipalID, claims.Subject, body.Body)
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.Created(w, "Catatan disimpan", noteResponse{ID: n.ID, Admin: n.AdminName, Body: n.Body, CreatedAt: tstr(n.CreatedAt)})
}

// ---- data requests ----

type dataRequestResponse struct {
	ID             int64   `json:"id"`
	TenantID       int64   `json:"tenantId"`
	TenantName     string  `json:"tenantName"`
	Kind           string  `json:"kind"`
	Status         string  `json:"status"`
	RequestedBy    string  `json:"requestedBy"`
	Note           string  `json:"note"`
	DueAt          string  `json:"dueAt"`
	DaysUntilDue   int     `json:"daysUntilDue"`
	CreatedAt      string  `json:"createdAt"`
	CompletedAt    *string `json:"completedAt"`
	CompletedBy    string  `json:"completedBy"`
	CompletionNote string  `json:"completionNote"`
}

func toDataRequestResponses(list []domain.DataRequest) []dataRequestResponse {
	now := time.Now()
	out := make([]dataRequestResponse, 0, len(list))
	for _, d := range list {
		out = append(out, dataRequestResponse{
			ID: d.ID, TenantID: d.TenantID, TenantName: d.TenantName, Kind: string(d.Kind), Status: string(d.Status),
			RequestedBy: d.RequestedBy, Note: d.Note, DueAt: tstr(d.DueAt), DaysUntilDue: d.DaysUntilDue(now), CreatedAt: tstr(d.CreatedAt),
			CompletedAt: tptr(d.CompletedAt), CompletedBy: d.CompletedBy, CompletionNote: d.CompletionNote,
		})
	}
	return out
}

func (h *ConsoleHandler) listDataRequests(w http.ResponseWriter, r *http.Request) {
	list, err := h.console.ListDataRequests(r.Context(), nil, r.URL.Query().Get("open") == "true")
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "ok", toDataRequestResponses(list))
}

func (h *ConsoleHandler) listTenantDataRequests(w http.ResponseWriter, r *http.Request, s []string) {
	id, ok := idAt(s, 1)
	if !ok {
		response.Error(w, http.StatusBadRequest, "ID tenant tidak valid", nil)
		return
	}
	list, err := h.console.ListDataRequests(r.Context(), &id, false)
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "ok", toDataRequestResponses(list))
}

func (h *ConsoleHandler) addDataRequest(w http.ResponseWriter, r *http.Request, s []string) {
	claims, _ := adminOf(w, r)
	id, ok := idAt(s, 1)
	if !ok {
		response.Error(w, http.StatusBadRequest, "ID tenant tidak valid", nil)
		return
	}
	var body struct {
		Kind        string `json:"kind"`
		RequestedBy string `json:"requestedBy"`
		Note        string `json:"note"`
	}
	if !decode(w, r, &body) {
		return
	}
	d, err := h.console.AddDataRequest(r.Context(), id, domain.DataRequestKind(body.Kind), body.RequestedBy, body.Note, claims.PrincipalID, time.Now())
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.Created(w, "Permintaan data dicatat", toDataRequestResponses([]domain.DataRequest{*d})[0])
}

func (h *ConsoleHandler) completeDataRequest(w http.ResponseWriter, r *http.Request, s []string) {
	claims, _ := adminOf(w, r)
	id, ok := idAt(s, 1)
	if !ok {
		response.Error(w, http.StatusBadRequest, "ID tidak valid", nil)
		return
	}
	var body struct {
		Status string `json:"status"`
		Note   string `json:"note"`
	}
	if !decode(w, r, &body) {
		return
	}
	if err := h.console.CompleteDataRequest(r.Context(), id, domain.DataRequestStatus(body.Status), claims.PrincipalID, body.Note, time.Now()); err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "Permintaan data ditutup", nil)
}

// ---- announcements ----

type announcementResponse struct {
	ID            int64   `json:"id"`
	Title         string  `json:"title"`
	Body          string  `json:"body"`
	Kind          string  `json:"kind"`
	Audience      string  `json:"audience"`
	AudienceValue string  `json:"audienceValue"`
	StartsAt      string  `json:"startsAt"`
	EndsAt        *string `json:"endsAt"`
	Active        bool    `json:"active"`
}

func toAnnouncementResponses(list []domain.Announcement) []announcementResponse {
	now := time.Now()
	out := make([]announcementResponse, 0, len(list))
	for _, a := range list {
		out = append(out, announcementResponse{
			ID: a.ID, Title: a.Title, Body: a.Body, Kind: string(a.Kind), Audience: string(a.Audience), AudienceValue: a.AudienceValue,
			StartsAt: tstr(a.StartsAt), EndsAt: tptr(a.EndsAt), Active: a.ActiveAt(now),
		})
	}
	return out
}

func (h *ConsoleHandler) listAnnouncements(w http.ResponseWriter, r *http.Request) {
	list, err := h.console.ListAnnouncements(r.Context())
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "ok", toAnnouncementResponses(list))
}

type announcementBody struct {
	Title         string  `json:"title"`
	Body          string  `json:"body"`
	Kind          string  `json:"kind"`
	Audience      string  `json:"audience"`
	AudienceValue string  `json:"audienceValue"`
	StartsAt      string  `json:"startsAt"`
	EndsAt        *string `json:"endsAt"`
}

func (b announcementBody) input() (application.AnnouncementInput, bool) {
	in := application.AnnouncementInput{
		Title: b.Title, Body: b.Body, Kind: domain.AnnouncementKind(b.Kind), Audience: domain.AnnouncementAudience(b.Audience), AudienceValue: b.AudienceValue,
	}
	if b.StartsAt != "" {
		t, err := time.Parse(timeLayout, b.StartsAt)
		if err != nil {
			return in, false
		}
		in.StartsAt = t
	}
	if b.EndsAt != nil && *b.EndsAt != "" {
		t, err := time.Parse(timeLayout, *b.EndsAt)
		if err != nil {
			return in, false
		}
		in.EndsAt = &t
	}
	return in, true
}

// saveAnnouncement creates (id 0) or updates an announcement.
func (h *ConsoleHandler) saveAnnouncement(w http.ResponseWriter, r *http.Request, id int64) {
	claims, _ := adminOf(w, r)
	var body announcementBody
	if !decode(w, r, &body) {
		return
	}
	in, ok := body.input()
	if !ok {
		response.Error(w, http.StatusUnprocessableEntity, "Format waktu tidak valid", map[string][]string{"startsAt": {"Gunakan format tanggal dan waktu yang valid"}})
		return
	}
	if id == 0 {
		a, err := h.console.CreateAnnouncement(r.Context(), in, claims.PrincipalID)
		if err != nil {
			writeAppError(w, err)
			return
		}
		response.Created(w, "Pengumuman dibuat", toAnnouncementResponses([]domain.Announcement{*a})[0])
		return
	}
	a, err := h.console.UpdateAnnouncement(r.Context(), id, in)
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "Pengumuman diperbarui", toAnnouncementResponses([]domain.Announcement{*a})[0])
}

func (h *ConsoleHandler) deleteAnnouncement(w http.ResponseWriter, r *http.Request, s []string) {
	id, ok := idAt(s, 1)
	if !ok {
		response.Error(w, http.StatusBadRequest, "ID tidak valid", nil)
		return
	}
	if err := h.console.DeleteAnnouncement(r.Context(), id); err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "Pengumuman dihapus", nil)
}

// ActiveAnnouncements is GET /announcements/active for any workspace staff.
func (h *ConsoleHandler) ActiveAnnouncements(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.FromContext(r.Context())
	tenantID, hasTenant := selfTenantID(r)
	if !ok || claims.PrincipalType != "staff" || !hasTenant {
		response.Error(w, http.StatusForbidden, "Hanya staf workspace yang dapat melihat pengumuman", nil)
		return
	}
	list, err := h.console.ActiveAnnouncements(r.Context(), tenantID, time.Now())
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "ok", toAnnouncementResponses(list))
}

// ---- subscription policy ----

type policyBody struct {
	TrialDays        int `json:"trialDays"`
	ExpiringSoonDays int `json:"expiringSoonDays"`
	GraceDays        int `json:"graceDays"`
	ReadOnlyDays     int `json:"readOnlyDays"`
}

func policyResponse(p domain.SubscriptionPolicy) policyBody {
	return policyBody{TrialDays: p.TrialDays, ExpiringSoonDays: p.ExpiringSoonDays, GraceDays: p.GraceDays, ReadOnlyDays: p.ReadOnlyDays}
}

func (h *ConsoleHandler) getPolicy(w http.ResponseWriter, r *http.Request) {
	response.OK(w, "ok", policyResponse(h.console.Policy()))
}

func (h *ConsoleHandler) putPolicy(w http.ResponseWriter, r *http.Request) {
	claims, _ := adminOf(w, r)
	var body policyBody
	if !decode(w, r, &body) {
		return
	}
	p := domain.SubscriptionPolicy{TrialDays: body.TrialDays, ExpiringSoonDays: body.ExpiringSoonDays, GraceDays: body.GraceDays, ReadOnlyDays: body.ReadOnlyDays}
	if err := h.console.SavePolicy(r.Context(), p, claims.PrincipalID); err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "Kebijakan langganan disimpan", policyResponse(p))
}

// ---- support access ----

type grantResponse struct {
	GrantedAt string `json:"grantedAt"`
	ExpiresAt string `json:"expiresAt"`
}

type sessionResponse struct {
	ID        int64   `json:"id"`
	Admin     string  `json:"admin"`
	Reason    string  `json:"reason"`
	StartedAt string  `json:"startedAt"`
	ExpiresAt string  `json:"expiresAt"`
	EndedAt   *string `json:"endedAt"`
}

type supportStatusResponse struct {
	Grant    *grantResponse    `json:"grant"`
	Sessions []sessionResponse `json:"sessions"`
}

func toSupportStatus(st *application.SupportAccessStatus) supportStatusResponse {
	out := supportStatusResponse{Sessions: make([]sessionResponse, 0, len(st.Sessions))}
	if st.Grant != nil {
		out.Grant = &grantResponse{GrantedAt: tstr(st.Grant.GrantedAt), ExpiresAt: tstr(st.Grant.ExpiresAt)}
	}
	for _, s := range st.Sessions {
		out.Sessions = append(out.Sessions, sessionResponse{ID: s.ID, Admin: s.AdminName, Reason: s.Reason, StartedAt: tstr(s.StartedAt), ExpiresAt: tstr(s.ExpiresAt), EndedAt: tptr(s.EndedAt)})
	}
	return out
}

// MySupportAccess is the Owner's side: GET shows the consent and past
// sessions, PUT grants 24 hours, DELETE withdraws it
// (/api/v1/tenants/me/support-access, business_profile.manage).
func (h *ConsoleHandler) MySupportAccess(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := ownerTenantID(w, r)
	if !ok {
		return
	}
	now := time.Now()
	switch r.Method {
	case http.MethodPut:
		claims, _ := middleware.FromContext(r.Context())
		staffID, err := strconv.ParseInt(claims.PrincipalID, 10, 64)
		if err != nil {
			response.Error(w, http.StatusForbidden, "Identitas staf tidak valid", nil)
			return
		}
		if _, err := h.console.GrantSupportAccess(r.Context(), tenantID, staffID, now); err != nil {
			writeAppError(w, err)
			return
		}
	case http.MethodDelete:
		if err := h.console.RevokeSupportAccess(r.Context(), tenantID, now); err != nil {
			writeAppError(w, err)
			return
		}
	case http.MethodGet:
	default:
		response.Error(w, http.StatusMethodNotAllowed, "Metode HTTP tidak diizinkan untuk endpoint ini", nil)
		return
	}
	st, err := h.console.SupportStatus(r.Context(), tenantID, now)
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "ok", toSupportStatus(st))
}

func (h *ConsoleHandler) adminSupportStatus(w http.ResponseWriter, r *http.Request, s []string) {
	id, ok := idAt(s, 1)
	if !ok {
		response.Error(w, http.StatusBadRequest, "ID tenant tidak valid", nil)
		return
	}
	st, err := h.console.SupportStatus(r.Context(), id, time.Now())
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "ok", toSupportStatus(st))
}

type impersonateResponse struct {
	SessionID   int64  `json:"sessionId"`
	AccessToken string `json:"accessToken"`
	ExpiresAt   string `json:"expiresAt"`
	TenantID    int64  `json:"tenantId"`
	Role        string `json:"role"`
}

func (h *ConsoleHandler) impersonate(w http.ResponseWriter, r *http.Request, s []string) {
	claims, _ := adminOf(w, r)
	id, ok := idAt(s, 1)
	if !ok {
		response.Error(w, http.StatusBadRequest, "ID tenant tidak valid", nil)
		return
	}
	var body struct {
		Reason string `json:"reason"`
	}
	if !decode(w, r, &body) {
		return
	}
	res, err := h.console.StartSupportSession(r.Context(), id, claims.PrincipalID, claims.Subject, body.Reason, time.Now())
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.Created(w, "Sesi dukungan dimulai", impersonateResponse{
		SessionID: res.Session.ID, AccessToken: res.Token, ExpiresAt: tstr(res.Session.ExpiresAt), TenantID: id, Role: "Owner",
	})
}

func (h *ConsoleHandler) endSession(w http.ResponseWriter, r *http.Request, s []string) {
	claims, _ := adminOf(w, r)
	id, ok := idAt(s, 1)
	if !ok {
		response.Error(w, http.StatusBadRequest, "ID tidak valid", nil)
		return
	}
	if err := h.console.EndSupportSession(r.Context(), id, claims.PrincipalID, time.Now()); err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "Sesi dukungan berakhir", nil)
}
