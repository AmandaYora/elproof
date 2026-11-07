package presentation

import (
	"net/http"
	"time"

	"elproof/internal/modules/platform/domain"
	"elproof/internal/shared/response"
)

type attentionTenantResponse struct {
	TenantID int64  `json:"tenantId"`
	Name     string `json:"name"`
	Status   string `json:"status"`
	DaysLeft int    `json:"daysLeft"`
}

type platformMetricsResponse struct {
	TotalTenants int `json:"totalTenants"`
	Trialing     int `json:"trialing"`
	Active       int `json:"active"`
	PastDue      int `json:"pastDue"`
	ReadOnly     int `json:"readOnly"`
	Suspended    int `json:"suspended"`
	Frozen       int `json:"frozen"`
	PaidTenants  int `json:"paidTenants"`

	MRR  int64 `json:"mrr"`
	ARPA int64 `json:"arpa"`

	TrialsEnded30d     int      `json:"trialsEnded30d"`
	TrialsConverted30d int      `json:"trialsConverted30d"`
	TrialConversionPct *float64 `json:"trialConversionPct"`

	Churned30d int      `json:"churned30d"`
	ChurnedMRR int64    `json:"churnedMrr"`
	ChurnPct   *float64 `json:"churnPct"`
	NewPaid30d int      `json:"newPaid30d"`
	NewMRR30d  int64    `json:"newMrr30d"`

	Revenue30d     int64 `json:"revenue30d"`
	RevenuePrev30d int64 `json:"revenuePrev30d"`
	OverdueValue   int64 `json:"overdueValue"`

	TrialsEndingSoon []attentionTenantResponse `json:"trialsEndingSoon"`
	OverdueTenants   []attentionTenantResponse `json:"overdueTenants"`
}

func toAttention(list []domain.AttentionTenant) []attentionTenantResponse {
	out := make([]attentionTenantResponse, 0, len(list))
	for _, a := range list {
		out = append(out, attentionTenantResponse{TenantID: a.TenantID, Name: a.Name, Status: string(a.Status), DaysLeft: a.DaysLeft})
	}
	return out
}

// Metrics is GET /platform/metrics: the SaaS numbers behind the Platform
// Console's Ringkasan. Any platform admin role may read it.
func (h *TenantHandler) Metrics(w http.ResponseWriter, r *http.Request) {
	if !requirePlatformAdmin(w, r) {
		return
	}
	m, err := h.tenants.PlatformMetrics(r.Context(), time.Now())
	if err != nil {
		writeAppError(w, err)
		return
	}
	response.OK(w, "ok", platformMetricsResponse{
		TotalTenants: m.TotalTenants, Trialing: m.Trialing, Active: m.Active, PastDue: m.PastDue, ReadOnly: m.ReadOnly,
		Suspended: m.Suspended, Frozen: m.Frozen, PaidTenants: m.PaidTenants,
		MRR: m.MRR, ARPA: m.ARPA,
		TrialsEnded30d: m.TrialsEnded30d, TrialsConverted30d: m.TrialsConverted30d, TrialConversionPct: m.TrialConversionPct(),
		Churned30d: m.Churned30d, ChurnedMRR: m.ChurnedMRR, ChurnPct: m.ChurnPct(), NewPaid30d: m.NewPaid30d, NewMRR30d: m.NewMRR30d,
		Revenue30d: m.Revenue30d, RevenuePrev30d: m.RevenuePrev30d, OverdueValue: m.OverdueValue,
		TrialsEndingSoon: toAttention(m.TrialsEndingSoon), OverdueTenants: toAttention(m.OverdueTenants),
	})
}
