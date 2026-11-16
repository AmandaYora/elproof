package presentation

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"elproof/internal/modules/projects/domain"
	"elproof/internal/shared/apperror"
	"elproof/internal/shared/pagination"
	"elproof/internal/shared/response"
)

// FinanceReader is what the Keuangan endpoints need (application.FinanceService).
type FinanceReader interface {
	Summary(ctx context.Context, scope domain.FinanceScope, today time.Time) (*domain.FinanceSummary, error)
	Receivables(ctx context.Context, scope domain.FinanceScope, today time.Time, f domain.ReceivableFilter, params pagination.Params, sort pagination.Sort) ([]domain.ReceivableRow, int64, error)
	ReceivableSortSpec() (map[string]pagination.SortColumn, pagination.Sort)
	Payables(ctx context.Context, scope domain.FinanceScope, today time.Time, f domain.PayableFilter, params pagination.Params, sort pagination.Sort) ([]domain.PayableRow, int64, error)
	PayableSortSpec() (map[string]pagination.SortColumn, pagination.Sort)
	Verifications(ctx context.Context, scope domain.FinanceScope, params pagination.Params, sort pagination.Sort) ([]domain.VerificationRow, int64, error)
	VerificationSortSpec() (map[string]pagination.SortColumn, pagination.Sort)
}

// SetFinance gives the handler the Keuangan reader.
func (h *Handler) SetFinance(f FinanceReader) { h.finance = f }

// financeAccess is the gate of every /finance/* route: staff, GET, and the
// finance.record permission (the page is the cross-project view of what that
// permission records). The scope is the caller's: a role limited to its own
// projects sees only the projects where the person is PIC or PIC Sales — never
// a 403, the same narrowing GET /projects applies.
func (h *Handler) financeAccess(w http.ResponseWriter, r *http.Request) (domain.FinanceScope, bool) {
	claims, ok := requireStaff(w, r)
	if !ok {
		return domain.FinanceScope{}, false
	}
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, "Metode HTTP tidak diizinkan untuk endpoint ini", nil)
		return domain.FinanceScope{}, false
	}
	if !claims.can("finance.record") {
		forbiddenPermission(w, "Peran Anda tidak dapat membuka halaman keuangan")
		return domain.FinanceScope{}, false
	}
	return financeScopeOf(claims), true
}

func financeScopeOf(claims staffClaims) domain.FinanceScope {
	scope := domain.FinanceScope{TenantID: claims.tenantID}
	if claims.assignedOnly() {
		id := claims.staffID
		scope.HolderStaffID = &id
	}
	return scope
}

type moneyCountResponse struct {
	Amount   int64 `json:"amount"`
	Count    int   `json:"count"`
	Projects int   `json:"projects"`
}

type payablesSummaryResponse struct {
	Amount        int64 `json:"amount"`
	Count         int   `json:"count"`
	OverdueAmount int64 `json:"overdueAmount"`
	OverdueCount  int   `json:"overdueCount"`
}

type financeSummaryResponse struct {
	Overdue             moneyCountResponse      `json:"overdue"`
	DueSoon             moneyCountResponse      `json:"dueSoon"`
	Received            moneyCountResponse      `json:"received"`
	Payables            payablesSummaryResponse `json:"payables"`
	PendingVerification moneyCountResponse      `json:"pendingVerification"`
	CashFlow            []cashFlowWeekResponse  `json:"cashFlow"`
}

func toMoneyCount(m domain.MoneyCount) moneyCountResponse {
	return moneyCountResponse{Amount: m.Amount, Count: m.Count, Projects: m.Projects}
}

// FinanceSummary is GET /finance/summary.
func (h *Handler) FinanceSummary(w http.ResponseWriter, r *http.Request) {
	scope, ok := h.financeAccess(w, r)
	if !ok {
		return
	}
	s, err := h.finance.Summary(r.Context(), scope, nowWIB())
	if err != nil {
		writeAppError(w, err)
		return
	}
	resp := financeSummaryResponse{
		Overdue: toMoneyCount(s.Overdue), DueSoon: toMoneyCount(s.DueSoon), Received: toMoneyCount(s.Received),
		Payables:            payablesSummaryResponse{Amount: s.Payables.Amount, Count: s.Payables.Count, OverdueAmount: s.Payables.OverdueAmount, OverdueCount: s.Payables.OverdueCount},
		PendingVerification: toMoneyCount(s.PendingVerification),
		CashFlow:            make([]cashFlowWeekResponse, 0, len(s.CashFlow)),
	}
	for _, wk := range s.CashFlow {
		resp.CashFlow = append(resp.CashFlow, cashFlowWeekResponse{WeekStart: wk.WeekStart.Format(dateLayout), ExpectedIn: wk.ExpectedIn, ExpectedOut: wk.ExpectedOut})
	}
	response.OK(w, "ok", resp)
}

type receivableResponse struct {
	ID          int64  `json:"id"`
	ProjectID   int64  `json:"projectId"`
	ProjectName string `json:"projectName"`
	BrideName   string `json:"brideName"`
	GroomName   string `json:"groomName"`
	Label       string `json:"label"`
	DueDate     string `json:"dueDate"`
	Amount      int64  `json:"amount"`
	Paid        int64  `json:"paid"`
	Remaining   int64  `json:"remaining"`
	Status      string `json:"status"`
	DaysLate    int    `json:"daysLate"`
	PICStaffID  int64  `json:"picStaffId"`
	PICName     string `json:"picName"`
}

// FinanceReceivables is GET /finance/receivables.
func (h *Handler) FinanceReceivables(w http.ResponseWriter, r *http.Request) {
	scope, ok := h.financeAccess(w, r)
	if !ok {
		return
	}
	filter, err := receivableFilterOf(r)
	if err != nil {
		writeAppError(w, err)
		return
	}
	columns, def := h.finance.ReceivableSortSpec()
	sort, err := pagination.SortFromRequest(r, columns, def)
	if err != nil {
		writeAppError(w, err)
		return
	}
	params := pagination.FromRequest(r)
	today := nowWIB()
	rows, total, err := h.finance.Receivables(r.Context(), scope, today, filter, params, sort)
	if err != nil {
		writeAppError(w, err)
		return
	}
	picIDs := make([]int64, 0, len(rows))
	for _, row := range rows {
		picIDs = append(picIDs, row.PICStaffID)
	}
	names := h.staffNames(r.Context(), scope.TenantID, picIDs)
	items := make([]receivableResponse, 0, len(rows))
	for _, row := range rows {
		items = append(items, receivableResponse{
			ID: row.Term.ID, ProjectID: row.Term.ProjectID, ProjectName: row.ProjectName, BrideName: row.BrideName, GroomName: row.GroomName,
			Label: row.Term.Label, DueDate: row.Term.DueDate.Format(dateLayout), Amount: row.Term.Amount, Paid: row.Paid, Remaining: row.Remaining,
			Status: string(row.Status(today)), DaysLate: domain.DaysLate(row.Term.DueDate, today, row.Remaining),
			PICStaffID: row.PICStaffID, PICName: names[row.PICStaffID],
		})
	}
	response.OKPaginated(w, "ok", items, pagination.BuildMeta(params, total).WithSort(sort))
}

// receivableFilterOf reads status, q/search, picStaffId, dueFrom, dueTo. Bad
// values are one 422 naming every offending key.
func receivableFilterOf(r *http.Request) (domain.ReceivableFilter, error) {
	q := r.URL.Query()
	f := domain.ReceivableFilter{Search: pagination.SearchFromRequest(r)}
	bad := map[string][]string{}
	if raw := q.Get("status"); raw != "" {
		f.View = domain.ReceivableView(raw)
		valid := false
		for _, v := range domain.ReceivableViews {
			valid = valid || v == f.View
		}
		if !valid {
			bad["status"] = []string{"Status harus salah satu dari: overdue, due_30d, unpaid, paid, partial, due_soon, upcoming"}
		}
	}
	f.PICStaffID = picFilterOf(q.Get("picStaffId"), bad)
	f.DueFrom, f.DueTo = dateFilterOf(q.Get("dueFrom"), "dueFrom", bad), dateFilterOf(q.Get("dueTo"), "dueTo", bad)
	if f.DueFrom != nil && f.DueTo != nil && f.DueTo.Before(*f.DueFrom) {
		bad["dueTo"] = []string{"Tanggal akhir tidak boleh sebelum tanggal awal"}
	}
	if len(bad) > 0 {
		return f, apperror.Validation("Filter tagihan tidak valid", bad)
	}
	return f, nil
}

func picFilterOf(raw string, bad map[string][]string) *int64 {
	if raw == "" {
		return nil
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id < 1 {
		bad["picStaffId"] = []string{"PIC tidak valid"}
		return nil
	}
	return &id
}

func dateFilterOf(raw, key string, bad map[string][]string) *time.Time {
	if raw == "" {
		return nil
	}
	t, err := parseDate(raw)
	if err != nil {
		bad[key] = []string{"Tanggal harus berformat YYYY-MM-DD"}
		return nil
	}
	return &t
}

type payableResponse struct {
	Kind          string  `json:"kind"`
	ID            int64   `json:"id"`
	ProjectID     int64   `json:"projectId"`
	ProjectName   string  `json:"projectName"`
	BrideName     string  `json:"brideName"`
	GroomName     string  `json:"groomName"`
	PayeeName     string  `json:"payeeName"`
	CategoryName  string  `json:"categoryName"`
	ContractValue int64   `json:"contractValue"`
	Paid          int64   `json:"paid"`
	Remaining     int64   `json:"remaining"`
	DueDate       *string `json:"dueDate"`
	Status        string  `json:"status"`
	DaysLate      int     `json:"daysLate"`
}

// FinancePayables is GET /finance/payables.
func (h *Handler) FinancePayables(w http.ResponseWriter, r *http.Request) {
	scope, ok := h.financeAccess(w, r)
	if !ok {
		return
	}
	q := r.URL.Query()
	f := domain.PayableFilter{Search: pagination.SearchFromRequest(r), Kind: domain.PayableKind(q.Get("kind")), Status: q.Get("status")}
	bad := map[string][]string{}
	if f.Kind != "" && f.Kind != domain.PayableVendor && f.Kind != domain.PayableVenue {
		bad["kind"] = []string{"Jenis harus vendor atau venue"}
	}
	if f.Status != "" {
		valid := false
		for _, s := range domain.PayableStatusFilters {
			valid = valid || s == f.Status
		}
		if !valid {
			bad["status"] = []string{"Status harus salah satu dari: open, unpaid, overdue, paid"}
		}
	}
	f.PICStaffID = picFilterOf(q.Get("picStaffId"), bad)
	if len(bad) > 0 {
		writeAppError(w, apperror.Validation("Filter pembayaran keluar tidak valid", bad))
		return
	}
	columns, def := h.finance.PayableSortSpec()
	sort, err := pagination.SortFromRequest(r, columns, def)
	if err != nil {
		writeAppError(w, err)
		return
	}
	params := pagination.FromRequest(r)
	today := nowWIB()
	rows, total, err := h.finance.Payables(r.Context(), scope, today, f, params, sort)
	if err != nil {
		writeAppError(w, err)
		return
	}
	var vendorIDs, categoryIDs []int64
	for _, row := range rows {
		if row.Kind == domain.PayableVendor {
			vendorIDs = append(vendorIDs, row.VendorID)
			categoryIDs = append(categoryIDs, row.CategoryID)
		}
	}
	vendorNames, categoryNames := map[int64]string{}, map[int64]string{}
	if h.refs != nil {
		vendorNames = h.refs.VendorNames(r.Context(), scope.TenantID, vendorIDs)
		categoryNames = h.refs.CategoryNames(r.Context(), scope.TenantID, categoryIDs)
	}
	items := make([]payableResponse, 0, len(rows))
	for _, row := range rows {
		item := payableResponse{
			Kind: string(row.Kind), ID: row.ID, ProjectID: row.ProjectID, ProjectName: row.ProjectName, BrideName: row.BrideName, GroomName: row.GroomName,
			ContractValue: row.ContractValue, Paid: row.Paid, Remaining: row.Remaining(), Status: string(row.Status(today)),
		}
		if row.Kind == domain.PayableVendor {
			item.PayeeName, item.CategoryName = vendorNames[row.VendorID], categoryNames[row.CategoryID]
		} else {
			item.PayeeName = row.VenueName
		}
		if row.DueDate != nil {
			d := row.DueDate.Format(dateLayout)
			item.DueDate = &d
			item.DaysLate = domain.DaysLate(*row.DueDate, today, item.Remaining)
		}
		items = append(items, item)
	}
	response.OKPaginated(w, "ok", items, pagination.BuildMeta(params, total).WithSort(sort))
}

type verificationResponse struct {
	ID              int64  `json:"id"`
	ProjectID       int64  `json:"projectId"`
	ProjectName     string `json:"projectName"`
	BrideName       string `json:"brideName"`
	GroomName       string `json:"groomName"`
	Type            string `json:"type"`
	Amount          int64  `json:"amount"`
	PaymentDate     string `json:"paymentDate"`
	Method          string `json:"method"`
	ReferenceNumber string `json:"referenceNumber"`
	RecordedByName  string `json:"recordedByName"`
}

// FinanceVerifications is GET /finance/verifications: client payments with
// proof that nobody has verified yet.
func (h *Handler) FinanceVerifications(w http.ResponseWriter, r *http.Request) {
	scope, ok := h.financeAccess(w, r)
	if !ok {
		return
	}
	columns, def := h.finance.VerificationSortSpec()
	sort, err := pagination.SortFromRequest(r, columns, def)
	if err != nil {
		writeAppError(w, err)
		return
	}
	params := pagination.FromRequest(r)
	rows, total, err := h.finance.Verifications(r.Context(), scope, params, sort)
	if err != nil {
		writeAppError(w, err)
		return
	}
	ids := make([]int64, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.RecordedByID)
	}
	names := h.staffNames(r.Context(), scope.TenantID, ids)
	items := make([]verificationResponse, 0, len(rows))
	for _, row := range rows {
		items = append(items, verificationResponse{
			ID: row.PaymentID, ProjectID: row.ProjectID, ProjectName: row.ProjectName, BrideName: row.BrideName, GroomName: row.GroomName,
			Type: string(row.Type), Amount: row.Amount, PaymentDate: row.PaymentDate.Format(dateLayout), Method: row.Method,
			ReferenceNumber: row.ReferenceNumber, RecordedByName: names[row.RecordedByID],
		})
	}
	response.OKPaginated(w, "ok", items, pagination.BuildMeta(params, total).WithSort(sort))
}

// staffNames names staff ids (best effort; "" for what does not resolve).
func (h *Handler) staffNames(ctx context.Context, tenantID int64, ids []int64) map[int64]string {
	if h.projects == nil || len(ids) == 0 {
		return map[int64]string{}
	}
	return h.projects.StaffNames(ctx, tenantID, ids)
}
