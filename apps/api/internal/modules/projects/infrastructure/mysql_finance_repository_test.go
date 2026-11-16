package infrastructure

import (
	"strings"
	"testing"
	"time"

	"elproof/internal/modules/projects/domain"
)

// The scope is part of every finance query, in both halves of the payables
// union, before any filter the caller chose.
func TestFinanceQueriesCarryTheScope(t *testing.T) {
	holder := int64(42)
	today := time.Date(2026, 11, 9, 10, 0, 0, 0, time.FixedZone("WIB", 7*3600))
	for _, scope := range []domain.FinanceScope{{TenantID: 7}, {TenantID: 7, HolderStaffID: &holder}} {
		where, args := financeProjectWhere(scope)
		if !strings.Contains(where, "p.tenant_id = ?") || !strings.Contains(where, "p.is_archived = 0") || !strings.Contains(where, "p.status <> 'Cancelled'") || args[0] != int64(7) {
			t.Fatalf("project where %q %v", where, args)
		}
		holderClause := strings.Contains(where, "(p.pic_staff_id = ? OR p.pic_sales_staff_id = ?)")
		if holderClause != (scope.HolderStaffID != nil) {
			t.Errorf("holder clause %v for scope %+v", holderClause, scope)
		}
		if scope.HolderStaffID != nil && (len(args) != 3 || args[1] != holder || args[2] != holder) {
			t.Errorf("holder args %v", args)
		}

		q, qargs := receivableRowsSQL(scope, domain.ReceivableFilter{Search: "50%_off"}, today)
		if strings.Count(q, "?") != len(qargs) {
			t.Errorf("receivables: %d placeholders, %d args", strings.Count(q, "?"), len(qargs))
		}
		if qargs[0] != "2026-11-09" || qargs[1] != "2026-11-16" {
			t.Errorf("receivables: today/soon args %v (the WIB calendar day)", qargs[:2])
		}
		if !strings.Contains(q, where) {
			t.Errorf("receivables query lacks the scope")
		}
		// LIKE wildcards in the search are taken literally.
		var sawEscaped bool
		for _, a := range qargs {
			sawEscaped = sawEscaped || a == `%50\%\_off%`
		}
		if !sawEscaped {
			t.Errorf("search not escaped: %v", qargs)
		}

		p, pargs := payableRowsSQL(scope, today)
		if strings.Count(p, where) != 2 {
			t.Errorf("payables: scope appears %d times, want 2 (vendor and venue)", strings.Count(p, where))
		}
		if strings.Count(p, "?") != len(pargs) {
			t.Errorf("payables: %d placeholders, %d args", strings.Count(p, "?"), len(pargs))
		}

		v, vargs := pendingVerificationWhere(scope)
		if !strings.Contains(v, where) || strings.Count(v, "?") != len(vargs) {
			t.Errorf("verifications: %q %v", v, vargs)
		}
	}
}

func TestReceivableViews(t *testing.T) {
	today := time.Date(2026, 11, 9, 23, 0, 0, 0, time.FixedZone("WIB", 7*3600))
	for _, v := range domain.ReceivableViews {
		w, args := receivableViewWhere(v, today)
		if w == "" || strings.Count(w, "?") != len(args) {
			t.Errorf("view %q: %q %v", v, w, args)
		}
	}
	if w, _ := receivableViewWhere(domain.ReceivableAll, today); w != "" {
		t.Errorf("all: %q", w)
	}
	_, args := receivableViewWhere(domain.ReceivableDue30, today)
	if args[0] != "2026-11-09" || args[1] != "2026-12-09" {
		t.Errorf("due_30d window %v", args)
	}
}
