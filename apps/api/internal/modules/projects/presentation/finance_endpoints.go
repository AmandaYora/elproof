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
func (h *Handler) financeAccess(w http.ResponseWriter, r *http.Request) (domain.FinanceScope, staffClaims, bool) {
	claims, ok := requireStaff(w, r)
	if !ok {
		return domain.FinanceScope{}, claims, false
	}
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, "Metode HTTP tidak diizinkan untuk endpoint ini", nil)
		return domain.FinanceScope{}, claims, false
	}
	if !claims.can("finance.record") {
		forbiddenPermission(w, "Peran Anda tidak dapat membuka halaman keuangan")
		return domain.FinanceScope{}, claims, false
	}
	return financeScopeOf(claims), claims, true
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
	// IncompleteCount: WO-paid suppliers without an agreed cost, not in amount.
	IncompleteCount int `json:"incompleteCount"`
}

type financeSummaryResponse struct {
	Overdue  moneyCountResponse `json:"overdue"`
	DueSoon  moneyCountResponse `json:"dueSoon"`
	Received moneyCountResponse `json:"received"`
	// Payables is what the WO still owes its vendors and venues: absent
	// without finance.margin.view, and the cash flow then has no money out.
	Payables            *payablesSummaryResponse `json:"payables,omitempty"`
	PendingVerification moneyCountResponse       `json:"pendingVerification"`
	CashFlow            []cashFlowWeekResponse   `json:"cashFlow"`
}

func toMoneyCount(m domain.MoneyCount) moneyCountResponse {
	return moneyCountResponse{Amount: m.Amount, Count: m.Count, Projects: m.Projects}
}

// FinanceSummary is GET /finance/summary.
func (h *Handler) FinanceSummary(w http.ResponseWriter, r *http.Request) {
	scope, claims, ok := h.financeAccess(w, r)
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
		PendingVerification: toMoneyCount(s.PendingVerification),
		CashFlow:            make([]cashFlowWeekResponse, 0, len(s.CashFlow)),
	}
	margin := claims.can("finance.margin.view")
	if margin {
		resp.Payables = &payablesSummaryResponse{Amount: s.Payables.Amount, Count: s.Payables.Count, OverdueAmount: s.Payables.OverdueAmount, OverdueCount: s.Payables.OverdueCount,
			IncompleteCount: s.Payables.IncompleteCount}
	}
	for _, wk := range s.CashFlow {
		week := cashFlowWeekResponse{WeekStart: wk.WeekStart.Format(dateLayout), ExpectedIn: wk.ExpectedIn}
		if margin {
			week.ExpectedOut = wk.ExpectedOut
		}
		resp.CashFlow = append(resp.CashFlow, week)
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
	scope, _, ok := h.financeAccess(w, r)
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

// payableResponse is one vendor or venue the WO pays, with what is still owed.
type payableResponse struct {
	Kind         string  `json:"kind"`
	ID           int64   `json:"id"`
	ProjectID    int64   `json:"projectId"`
	ProjectName  string  `json:"projectName"`
	BrideName    string  `json:"brideName"`
	GroomName    string  `json:"groomName"`
	PayeeName    string  `json:"payeeName"`
	CategoryName string  `json:"categoryName"`
	AgreedCost   int64   `json:"agreedCost"`
	Paid         int64   `json:"paid"`
	Remaining    int64   `json:"remaining"`
	DueDate      *string `json:"dueDate"`
	Status       string  `json:"status"`
	DaysLate     int     `json:"daysLate"`
	Incomplete   bool    `json:"incomplete"` // no agreed cost yet: agreedCost and remaining are 0, status "incomplete"
}

// FinancePayables is GET /finance/payables: what the WO still owes the vendors
// and venues it pays. It needs finance.margin.view on top of finance.record.
func (h *Handler) FinancePayables(w http.ResponseWriter, r *http.Request) {
	scope, claims, ok := h.financeAccess(w, r)
	if !ok {
		return
	}
	if !claims.can("finance.margin.view") {
		forbiddenPermission(w, "Peran Anda tidak dapat melihat pembayaran WO ke rekanan")
		return
	}
	q := r.URL.Query()
	f := domain.PayableFilter{Search: pagination.SearchFromRequest(r), Kind: domain.SupplierKind(q.Get("kind")), Status: q.Get("status")}
	bad := map[string][]string{}
	if f.Kind != "" && f.Kind != domain.SupplierVendor && f.Kind != domain.SupplierVenue {
		bad["kind"] = []string{"Jenis harus vendor atau venue"}
	}
	if f.Status != "" {
		valid := false
		for _, s := range domain.PayableStatusFilters {
			valid = valid || s == f.Status
		}
		if !valid {
			bad["status"] = []string{"Status harus salah satu dari: open, unpaid, overdue, paid, incomplete"}
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
	var categoryIDs []int64
	for _, row := range rows {
		if row.CategoryID != 0 {
			categoryIDs = append(categoryIDs, row.CategoryID)
		}
	}
	categoryNames := map[int64]string{}
	if h.refs != nil {
		categoryNames = h.refs.CategoryNames(r.Context(), scope.TenantID, categoryIDs)
	}
	items := make([]payableResponse, 0, len(rows))
	for _, row := range rows {
		item := payableResponse{
			Kind: string(row.Kind), ID: row.ID, ProjectID: row.ProjectID, ProjectName: row.ProjectName, BrideName: row.BrideName, GroomName: row.GroomName,
			PayeeName: row.SupplierName, CategoryName: categoryNames[row.CategoryID],
			AgreedCost: row.AgreedCost, Paid: row.Paid, Remaining: row.Remaining(), Status: string(row.Status(today)),
			Incomplete: row.Incomplete,
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
	// Source is "client" when the couple sent the proof from the portal (ADR-0047).
	Source string `json:"source"`
	// Kind is "clientPayment" (money to the WO: verify with POST
	// /projects/{projectId}/client-payments/{id}/verify) or "supplierPayment"
	// (the client's proof of a payment to a vendor they pay directly, Paket WO
	// Tahap 4: POST /projects/{projectId}/supplier-payments/{id}/verify or
	// /reject). The ids of the two kinds may collide: key rows by kind + id.
	Kind string `json:"kind"`
	// For a supplierPayment: the vendor or venue (always named: the client pays
	// it) and the bill it pays; null for a clientPayment.
	ProjectSupplierID *int64  `json:"projectSupplierId"`
	SupplierName      *string `json:"supplierName"`
	BillTermID        *int64  `json:"billTermId"`
	BillTermLabel     *string `json:"billTermLabel"`
}

// FinanceVerifications is GET /finance/verifications: client payments with
// proof that nobody has verified yet, and the client's proofs to vendors they
// pay directly that the WO has not checked yet (kind, Paket WO Tahap 4).
func (h *Handler) FinanceVerifications(w http.ResponseWriter, r *http.Request) {
	scope, _, ok := h.financeAccess(w, r)
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
		item := verificationResponse{
			ID: row.PaymentID, ProjectID: row.ProjectID, ProjectName: row.ProjectName, BrideName: row.BrideName, GroomName: row.GroomName,
			Type: string(row.Type), Amount: row.Amount, PaymentDate: row.PaymentDate.Format(dateLayout), Method: row.Method,
			ReferenceNumber: row.ReferenceNumber, RecordedByName: names[row.RecordedByID],
			Source: string(domain.ClientPayment{SubmittedByClientID: row.SubmittedByClientID}.Source()),
			Kind:   row.Kind,
		}
		if item.Kind == "" {
			item.Kind = domain.VerificationClientPayment
		}
		if row.Kind == domain.VerificationSupplierPayment {
			supID, supName, label := row.ProjectSupplierID, row.SupplierName, row.BillTermLabel
			item.ProjectSupplierID, item.SupplierName = &supID, &supName
			if row.BillTermID != 0 {
				billID := row.BillTermID
				item.BillTermID, item.BillTermLabel = &billID, &label
			}
		}
		items = append(items, item)
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
