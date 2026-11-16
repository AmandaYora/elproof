package presentation

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"elproof/internal/modules/projects/domain"
	"elproof/internal/shared/middleware"
	"elproof/internal/shared/pagination"
)

// fakeFinance records the scope and filters each call received.
type fakeFinance struct {
	scopes      []domain.FinanceScope
	recvFilter  domain.ReceivableFilter
	payFilter   domain.PayableFilter
	sort        pagination.Sort
	receivables []domain.ReceivableRow
}

func (f *fakeFinance) Summary(_ context.Context, s domain.FinanceScope, _ time.Time) (*domain.FinanceSummary, error) {
	f.scopes = append(f.scopes, s)
	return &domain.FinanceSummary{CashFlow: domain.BuildCashFlow(time.Now(), nil, nil)}, nil
}
func (f *fakeFinance) Receivables(_ context.Context, s domain.FinanceScope, _ time.Time, fl domain.ReceivableFilter, _ pagination.Params, so pagination.Sort) ([]domain.ReceivableRow, int64, error) {
	f.scopes, f.recvFilter, f.sort = append(f.scopes, s), fl, so
	return f.receivables, int64(len(f.receivables)), nil
}
func (f *fakeFinance) ReceivableSortSpec() (map[string]pagination.SortColumn, pagination.Sort) {
	return map[string]pagination.SortColumn{"dueDate": pagination.Col("z.due_date"), "remaining": pagination.Col("z.remaining")}, pagination.By("dueDate", false)
}
func (f *fakeFinance) Payables(_ context.Context, s domain.FinanceScope, _ time.Time, fl domain.PayableFilter, _ pagination.Params, so pagination.Sort) ([]domain.PayableRow, int64, error) {
	f.scopes, f.payFilter, f.sort = append(f.scopes, s), fl, so
	return nil, 0, nil
}
func (f *fakeFinance) PayableSortSpec() (map[string]pagination.SortColumn, pagination.Sort) {
	return map[string]pagination.SortColumn{"remaining": pagination.Col("z.remaining")}, pagination.By("remaining", true)
}
func (f *fakeFinance) Verifications(_ context.Context, s domain.FinanceScope, _ pagination.Params, so pagination.Sort) ([]domain.VerificationRow, int64, error) {
	f.scopes, f.sort = append(f.scopes, s), so
	return nil, 0, nil
}
func (f *fakeFinance) VerificationSortSpec() (map[string]pagination.SortColumn, pagination.Sort) {
	return map[string]pagination.SortColumn{"paymentDate": pagination.Col("cp.payment_date")}, pagination.By("paymentDate", false)
}

func financeRequest(method, target string, principal string, grants *middleware.Grants) *http.Request {
	r := httptest.NewRequest(method, target, nil)
	ctx := middleware.NewContextWithClaims(r.Context(), &middleware.Claims{PrincipalType: principal, PrincipalID: "42", TenantID: "7"})
	if grants != nil {
		ctx = middleware.WithGrants(ctx, grants)
	}
	return r.WithContext(ctx)
}

func financeGrants(allProjects bool, perms ...string) *middleware.Grants {
	g := &middleware.Grants{AllProjects: allProjects, Permissions: map[string]bool{}}
	for _, p := range perms {
		g.Permissions[p] = true
	}
	return g
}

// Every /finance/* route: finance.record is required; the scope is the caller's
// workspace, narrowed to their own projects when their role is.
func TestFinanceAccessAndScope(t *testing.T) {
	routes := map[string]func(*Handler, http.ResponseWriter, *http.Request){
		"/api/v1/finance/summary":       (*Handler).FinanceSummary,
		"/api/v1/finance/receivables":   (*Handler).FinanceReceivables,
		"/api/v1/finance/payables":      (*Handler).FinancePayables,
		"/api/v1/finance/verifications": (*Handler).FinanceVerifications,
	}
	for path, route := range routes {
		t.Run(path, func(t *testing.T) {
			call := func(r *http.Request) (*httptest.ResponseRecorder, *fakeFinance) {
				f := &fakeFinance{}
				h := &Handler{finance: f}
				w := httptest.NewRecorder()
				route(h, w, r)
				return w, f
			}

			// Without finance.record: 403 forbidden_permission, nothing read.
			w, f := call(financeRequest(http.MethodGet, path, "staff", financeGrants(true, "dashboard.view")))
			if w.Code != http.StatusForbidden || len(f.scopes) != 0 {
				t.Fatalf("no finance.record: status %d, reads %d", w.Code, len(f.scopes))
			}
			var body struct {
				Errors map[string]any `json:"errors"`
			}
			_ = json.Unmarshal(w.Body.Bytes(), &body)
			if body.Errors["code"] != "forbidden_permission" {
				t.Errorf("no finance.record: errors %v", body.Errors)
			}

			// A client principal (the portal) never reaches it.
			if w, f := call(financeRequest(http.MethodGet, path, "client", financeGrants(true, "finance.record"))); w.Code != http.StatusForbidden || len(f.scopes) != 0 {
				t.Errorf("client: status %d, reads %d", w.Code, len(f.scopes))
			}

			// Read-only.
			if w, f := call(financeRequest(http.MethodPost, path, "staff", financeGrants(true, "finance.record"))); w.Code != http.StatusMethodNotAllowed || len(f.scopes) != 0 {
				t.Errorf("POST: status %d, reads %d", w.Code, len(f.scopes))
			}

			// Every project: the whole workspace, no holder.
			w, f = call(financeRequest(http.MethodGet, path, "staff", financeGrants(true, "finance.record")))
			if w.Code != http.StatusOK || len(f.scopes) != 1 {
				t.Fatalf("all projects: status %d body %s", w.Code, w.Body.String())
			}
			if s := f.scopes[0]; s.TenantID != 7 || s.HolderStaffID != nil {
				t.Errorf("all projects: scope %+v", s)
			}

			// Own projects only: the holder is the caller, whatever the query says.
			w, f = call(financeRequest(http.MethodGet, path+"?picStaffId=99", "staff", financeGrants(false, "finance.record")))
			if w.Code != http.StatusOK || len(f.scopes) != 1 {
				t.Fatalf("assigned: status %d body %s", w.Code, w.Body.String())
			}
			if s := f.scopes[0]; s.TenantID != 7 || s.HolderStaffID == nil || *s.HolderStaffID != 42 {
				t.Errorf("assigned: scope %+v", s)
			}
		})
	}
}

func TestFinanceReceivablesFilters(t *testing.T) {
	f := &fakeFinance{receivables: []domain.ReceivableRow{{
		Term: domain.PaymentTerm{ID: 3, ProjectID: 9, Label: "Termin 2", Amount: 100, DueDate: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)},
		Paid: 40, Remaining: 60, ProjectName: "Citra & Bima",
	}}}
	h := &Handler{finance: f}
	w := httptest.NewRecorder()
	h.FinanceReceivables(w, financeRequest(http.MethodGet,
		"/api/v1/finance/receivables?status=overdue&q=citra&picStaffId=5&dueFrom=2026-11-01&dueTo=2026-11-30&sort=remaining:desc",
		"staff", financeGrants(true, "finance.record")))
	if w.Code != http.StatusOK {
		t.Fatalf("status %d: %s", w.Code, w.Body.String())
	}
	fl := f.recvFilter
	if fl.View != domain.ReceivableOverdue || fl.Search != "citra" || fl.PICStaffID == nil || *fl.PICStaffID != 5 ||
		fl.DueFrom == nil || fl.DueFrom.Format("2006-01-02") != "2026-11-01" || fl.DueTo == nil || f.sort.String() != "remaining:desc" {
		t.Errorf("filter %+v sort %s", fl, f.sort.String())
	}
	var body struct {
		Data []receivableResponse `json:"data"`
		Meta pagination.Meta      `json:"meta"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Data) != 1 || body.Data[0].Status != "overdue" || body.Data[0].DaysLate <= 0 || body.Data[0].Remaining != 60 || body.Meta.Sort != "remaining:desc" {
		t.Errorf("response %+v meta %+v", body.Data, body.Meta)
	}

	// Bad values: one 422 naming every key, nothing read.
	f2 := &fakeFinance{}
	w = httptest.NewRecorder()
	(&Handler{finance: f2}).FinanceReceivables(w, financeRequest(http.MethodGet,
		"/api/v1/finance/receivables?status=lunas&picStaffId=x&dueFrom=2026-13-01&sort=label",
		"staff", financeGrants(true, "finance.record")))
	var bad struct {
		Errors map[string][]string `json:"errors"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &bad)
	if w.Code != http.StatusUnprocessableEntity || len(f2.scopes) != 0 || bad.Errors["status"] == nil || bad.Errors["picStaffId"] == nil || bad.Errors["dueFrom"] == nil {
		t.Errorf("bad filters: status %d errors %v", w.Code, bad.Errors)
	}
}

func TestFinancePayablesFilters(t *testing.T) {
	f := &fakeFinance{}
	w := httptest.NewRecorder()
	(&Handler{finance: f}).FinancePayables(w, financeRequest(http.MethodGet, "/api/v1/finance/payables?kind=venue&status=open&q=graha",
		"staff", financeGrants(true, "finance.record")))
	if w.Code != http.StatusOK || f.payFilter.Kind != domain.PayableVenue || f.payFilter.Status != "open" || f.payFilter.Search != "graha" {
		t.Errorf("status %d filter %+v", w.Code, f.payFilter)
	}
	w = httptest.NewRecorder()
	(&Handler{finance: &fakeFinance{}}).FinancePayables(w, financeRequest(http.MethodGet, "/api/v1/finance/payables?kind=dekor&status=x",
		"staff", financeGrants(true, "finance.record")))
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("bad kind/status: status %d", w.Code)
	}
}
