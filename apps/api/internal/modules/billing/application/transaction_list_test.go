package application

import (
	"context"
	"reflect"
	"testing"
	"time"

	"elproof/internal/modules/billing/domain"
	"elproof/internal/shared/pagination"
)

type dirStub struct {
	names map[int64]string
	match []int64
}

func (d dirStub) TenantNames(_ context.Context, ids []int64) (map[int64]string, error) {
	out := map[int64]string{}
	for _, id := range ids {
		if n, ok := d.names[id]; ok {
			out[id] = n
		}
	}
	return out, nil
}
func (d dirStub) SearchTenantIDs(context.Context, string, int) ([]int64, error) { return d.match, nil }

// listRepo is a TransactionRepository whose list methods work on a fixed slice
// (the SQL ordering is covered by the integration scenarios).
type listRepo struct {
	TransactionRepository
	all       []domain.Transaction
	gotFilter domain.TransactionFilter
}

func (r *listRepo) ListAllFiltered(_ context.Context, _ *int64, f domain.TransactionFilter) ([]domain.Transaction, error) {
	r.gotFilter = f
	return append([]domain.Transaction(nil), r.all...), nil
}
func (r *listRepo) ListPaginated(_ context.Context, _ *int64, _ pagination.Params, f domain.TransactionFilter, _ pagination.Sort) ([]domain.Transaction, int64, error) {
	r.gotFilter = f
	return r.all, int64(len(r.all)), nil
}

func ids(rows []TransactionRow) []int64 {
	out := make([]int64, len(rows))
	for i, r := range rows {
		out[i] = r.ID
	}
	return out
}

func TestSortByTenantName(t *testing.T) {
	names := map[int64]string{1: "Bunga WO", 2: "anggrek WO", 3: "Cempaka WO"}
	list := []domain.Transaction{{ID: 10, TenantID: 1}, {ID: 11, TenantID: 99}, {ID: 12, TenantID: 2}, {ID: 13, TenantID: 1}, {ID: 14, TenantID: 3}}
	sortByTenantName(list, names, false)
	got := []int64{}
	for _, tx := range list {
		got = append(got, tx.ID)
	}
	// anggrek(12) < Bunga(10,13) < Cempaka(14); the unresolved tenant (11) last
	if want := []int64{12, 10, 13, 14, 11}; !reflect.DeepEqual(got, want) {
		t.Errorf("asc: %v, want %v", got, want)
	}
	sortByTenantName(list, names, true)
	got = got[:0]
	for _, tx := range list {
		got = append(got, tx.ID)
	}
	if want := []int64{14, 13, 10, 12, 11}; !reflect.DeepEqual(got, want) {
		t.Errorf("desc (unresolved still last): %v, want %v", got, want)
	}
}

func TestListPageNamesRowsAndPagesInMemoryByTenantName(t *testing.T) {
	repo := &listRepo{all: []domain.Transaction{
		{ID: 1, TenantID: 7, CreatedAt: time.Now()}, {ID: 2, TenantID: 8}, {ID: 3, TenantID: 9}, {ID: 4, TenantID: 7},
	}}
	svc := NewTransactionService(repo)
	svc.SetTenants(dirStub{names: map[int64]string{7: "Zeta", 8: "Alfa", 9: "Mega"}})

	rows, total, err := svc.ListPage(context.Background(), nil, pagination.Params{Page: 1, Limit: 2}, domain.TransactionFilter{}, pagination.By("tenantName", false))
	if err != nil {
		t.Fatal(err)
	}
	// SortFromRequest would have resolved the field; here the field name alone selects the in-memory path.
	if total != 4 || !reflect.DeepEqual(ids(rows), []int64{2, 3}) || rows[0].TenantName != "Alfa" {
		t.Errorf("page 1: total %d ids %v rows %+v", total, ids(rows), rows)
	}
	rows, _, _ = svc.ListPage(context.Background(), nil, pagination.Params{Page: 2, Limit: 2}, domain.TransactionFilter{}, pagination.By("tenantName", false))
	if !reflect.DeepEqual(ids(rows), []int64{1, 4}) {
		t.Errorf("page 2: %v", ids(rows))
	}
	rows, _, _ = svc.ListPage(context.Background(), nil, pagination.Params{Page: 9, Limit: 2}, domain.TransactionFilter{}, pagination.By("tenantName", false))
	if len(rows) != 0 {
		t.Errorf("a page past the end is empty, not an error: %v", ids(rows))
	}
}

func TestListPageSearchResolvesTenantNamesIntoTheFilter(t *testing.T) {
	repo := &listRepo{all: []domain.Transaction{{ID: 1, TenantID: 7}}}
	svc := NewTransactionService(repo)
	svc.SetTenants(dirStub{names: map[int64]string{7: "Melati"}, match: []int64{7, 8}})
	if _, _, err := svc.ListPage(context.Background(), nil, pagination.Params{Page: 1, Limit: 10}, domain.TransactionFilter{Search: "mel"}, pagination.By("createdAt", true)); err != nil {
		t.Fatal(err)
	}
	if repo.gotFilter.Search != "mel" || !reflect.DeepEqual(repo.gotFilter.TenantIDs, []int64{7, 8}) {
		t.Errorf("filter: %+v", repo.gotFilter)
	}
	// Without a search, no lookup is made and TenantIDs stays empty.
	_, _, _ = svc.ListPage(context.Background(), nil, pagination.Params{Page: 1, Limit: 10}, domain.TransactionFilter{}, pagination.By("createdAt", true))
	if repo.gotFilter.TenantIDs != nil {
		t.Errorf("no search, no tenant ids: %+v", repo.gotFilter)
	}
}

func TestListPageWithoutADirectoryStillWorks(t *testing.T) {
	repo := &listRepo{all: []domain.Transaction{{ID: 1, TenantID: 7}}}
	rows, _, err := NewTransactionService(repo).ListPage(context.Background(), nil, pagination.Params{Page: 1, Limit: 10}, domain.TransactionFilter{Search: "x"}, pagination.By("createdAt", true))
	if err != nil || len(rows) != 1 || rows[0].TenantName != "" {
		t.Errorf("%v %+v", err, rows)
	}
}
