package presentation

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"elproof/internal/modules/projects/domain"
)

// Paket WO Tahap 4: the portal's bills of the vendors the client pays directly.

func TestSupplierBillPortalWrite(t *testing.T) {
	if perm, ok := clientWritePermission(http.MethodPost, []string{"supplier-bills", "submissions"}); !ok || perm != "portal.finance.submit" {
		t.Errorf("POST supplier-bills/submissions: got (%q,%v), want portal.finance.submit", perm, ok)
	}
	for _, c := range []struct {
		method string
		rest   []string
	}{
		{http.MethodPost, []string{"supplier-bills", "terms"}},                 // the WO's bill CRUD
		{http.MethodPatch, []string{"supplier-bills", "terms", "3"}},           //
		{http.MethodDelete, []string{"supplier-bills", "terms", "3"}},          //
		{http.MethodPost, []string{"supplier-bills", "terms", "3", "restore"}}, //
		{http.MethodPut, []string{"supplier-bills", "submissions"}},
		{http.MethodPost, []string{"supplier-bills"}},
		{http.MethodPost, []string{"supplier-payments"}},                // the staff record
		{http.MethodPost, []string{"supplier-payments", "3", "verify"}}, // checking is the WO's
		{http.MethodPost, []string{"supplier-payments", "3", "reject"}},
	} {
		if _, ok := clientWritePermission(c.method, c.rest); ok {
			t.Errorf("%s %v must not be a client write", c.method, c.rest)
		}
	}
}

// Pengantin (portal.finance.view + submit) reads and sends; Keluarga (no
// finance) reaches neither; the WO's bill writes stay closed to every client.
func TestSupplierBillRoutesFollowThePortalPermissions(t *testing.T) {
	h := &Handler{clientAccess: ownProject{projectID: 12}}
	pengantin := []string{"portal.finance.view", "portal.finance.submit", "portal.vendor.view", "portal.timeline.view"}
	keluarga := []string{"portal.vendor.view", "portal.timeline.view", "portal.documents.view"}
	cases := []struct {
		name, method, path string
		perms              []string
		want               int // 0 = let through
	}{
		{"Pengantin lists the bills", http.MethodGet, "/api/v1/projects/12/supplier-bills", pengantin, 0},
		{"Pengantin sends a proof", http.MethodPost, "/api/v1/projects/12/supplier-bills/submissions", pengantin, 0},
		{"Keluarga cannot list the bills", http.MethodGet, "/api/v1/projects/12/supplier-bills", keluarga, http.StatusForbidden},
		{"Keluarga cannot send a proof", http.MethodPost, "/api/v1/projects/12/supplier-bills/submissions", keluarga, http.StatusForbidden},
		{"finance view alone cannot send a proof", http.MethodPost, "/api/v1/projects/12/supplier-bills/submissions", []string{"portal.finance.view"}, http.StatusForbidden},
		{"another couple's bills", http.MethodGet, "/api/v1/projects/13/supplier-bills", pengantin, http.StatusForbidden},
		{"another couple's submission", http.MethodPost, "/api/v1/projects/13/supplier-bills/submissions", pengantin, http.StatusForbidden},
		{"a client creates a bill", http.MethodPost, "/api/v1/projects/12/supplier-bills/terms", pengantin, http.StatusForbidden},
		{"a client edits a bill", http.MethodPatch, "/api/v1/projects/12/supplier-bills/terms/3", pengantin, http.StatusForbidden},
		{"a client deletes a bill", http.MethodDelete, "/api/v1/projects/12/supplier-bills/terms/3", pengantin, http.StatusForbidden},
		{"a client verifies its own proof", http.MethodPost, "/api/v1/projects/12/supplier-payments/3/verify", pengantin, http.StatusForbidden},
		{"a client rejects a proof", http.MethodPost, "/api/v1/projects/12/supplier-payments/3/reject", pengantin, http.StatusForbidden},
	}
	for _, c := range cases {
		rest := strings.Split(strings.TrimPrefix(c.path, "/api/v1/projects/"), "/")
		var projectID int64 = 12
		if rest[0] == "13" {
			projectID = 13
		}
		rec := httptest.NewRecorder()
		claims, ok := h.resolveProjectAccess(rec, clientRequest(c.method, c.path, c.perms...), projectID, rest[1:])
		switch {
		case c.want == 0 && (!ok || !claims.isClient || claims.clientID != 5):
			t.Errorf("%s: want let through, got ok=%v (%d %s)", c.name, ok, rec.Code, rec.Body.String())
		case c.want != 0 && (ok || rec.Code != c.want):
			t.Errorf("%s: want %d, got ok=%v %d", c.name, c.want, ok, rec.Code)
		}
	}
}

func TestStaffWritePermissionForSupplierBills(t *testing.T) {
	for _, c := range []struct {
		method string
		rest   []string
	}{
		{http.MethodPost, []string{"supplier-bills", "terms"}},
		{http.MethodPatch, []string{"supplier-bills", "terms", "3"}},
		{http.MethodDelete, []string{"supplier-bills", "terms", "3"}},
		{http.MethodPost, []string{"supplier-bills", "terms", "3", "restore"}},
		{http.MethodPost, []string{"supplier-payments", "3", "verify"}},
		{http.MethodPost, []string{"supplier-payments", "3", "reject"}},
	} {
		if perm, _ := staffWritePermission(c.method, c.rest); perm != "finance.record" {
			t.Errorf("%s %v: got %q, want finance.record", c.method, c.rest, perm)
		}
	}
}

// Staff never send a proof as if they were the couple: they record the
// payment in Keuangan.
func TestStaffCannotSubmitASupplierProof(t *testing.T) {
	h := &Handler{}
	staff := staffClaims{tenantID: 1, staffID: 9, grants: grantsOf("finance.record")}
	rec := httptest.NewRecorder()
	h.submitSupplierBill(rec, httptest.NewRequest(http.MethodPost, "/", strings.NewReader("{}")), staff, 12)
	if rec.Code != http.StatusForbidden || !strings.Contains(rec.Body.String(), "forbidden_permission") {
		t.Errorf("staff submission: got %d %s", rec.Code, rec.Body.String())
	}
}

// The dashboard's client-paid vendor items are there only with finance.record,
// and hiding the WO's supplier costs (no finance.margin.view) keeps them: they
// are the client's money (PLAN §9.15).
func TestDashboardClientBillSections(t *testing.T) {
	due := time.Date(2026, 11, 25, 0, 0, 0, 0, time.UTC)
	stats := domain.DashboardStats{
		ClientBillsDue: []domain.ClientBillDue{{
			BillTermID: 4, ProjectSupplierID: 7, ProjectID: 12, ProjectName: "Proyek", SupplierKind: domain.SupplierKind("vendor"),
			SupplierName: "Foto Klien", Label: "DP", DueDate: due, Remaining: 2_000_000, Pending: 500_000, Status: domain.TermDueSoon,
		}},
		SupplierProofsPending: []domain.SupplierProofPending{{
			SupplierPaymentID: 9, ProjectID: 12, ProjectName: "Proyek", ProjectSupplierID: 7, SupplierName: "Foto Klien",
			BillTermID: 4, BillTermLabel: "DP", Type: domain.PaymentType("DP"), Amount: 500_000, PaymentDate: due, SubmittedAt: due,
		}},
		SupplierProofsPendingCount: 1,
	}
	without, _ := json.Marshal(toDashboardResponse(stats, nil))
	for _, key := range []string{"clientBillsDue", "supplierProofsPending", "supplierProofsPendingCount"} {
		if strings.Contains(string(without), `"`+key+`"`) {
			t.Errorf("without finance.record %q must be absent: %s", key, without)
		}
	}

	stats.ClientBillSections = true
	resp := toDashboardResponse(stats, nil)
	hideSupplierCosts(&resp)
	body, _ := json.Marshal(resp)
	var got struct {
		ClientBillsDue []map[string]any `json:"clientBillsDue"`
		Proofs         []map[string]any `json:"supplierProofsPending"`
		Count          int              `json:"supplierProofsPendingCount"`
	}
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatal(err)
	}
	if len(got.ClientBillsDue) != 1 || got.ClientBillsDue[0]["supplierName"] != "Foto Klien" || got.ClientBillsDue[0]["remaining"] != float64(2_000_000) ||
		got.ClientBillsDue[0]["pending"] != float64(500_000) || got.ClientBillsDue[0]["dueDate"] != "2026-11-25" {
		t.Errorf("clientBillsDue: %s", body)
	}
	if len(got.Proofs) != 1 || got.Proofs[0]["supplierPaymentId"] != float64(9) || got.Proofs[0]["billTermId"] != float64(4) || got.Count != 1 {
		t.Errorf("supplierProofsPending: %s", body)
	}
}
