package presentation

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestVendorListFilterReadsEveryListParameter(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/vendors/export?q=%20Mawar%20&search=Melati&categoryId=7&city=Kota+Bandung&isActive=false", nil)
	rec := httptest.NewRecorder()
	f, ok := vendorListFilter(rec, req)
	if !ok {
		t.Fatalf("filter rejected: %d %s", rec.Code, rec.Body)
	}
	if f.Search != "Mawar" {
		t.Errorf("q wins over search and is trimmed: %q", f.Search)
	}
	if f.CategoryID == nil || *f.CategoryID != 7 || f.City != "Kota Bandung" || f.IsActive == nil || *f.IsActive {
		t.Errorf("filter = %+v", f)
	}

	alias, _ := vendorListFilter(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/x?search=Melati", nil))
	if alias.Search != "Melati" {
		t.Errorf("search stays an alias: %q", alias.Search)
	}
	none, ok := vendorListFilter(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/x", nil))
	if !ok || none.Search != "" || none.CategoryID != nil || none.City != "" || none.IsActive != nil {
		t.Errorf("no parameters = no filter: %+v", none)
	}
}

func TestVendorListFilterRejectsBadValues(t *testing.T) {
	for target, want := range map[string]int{
		"/x?isActive=maybe": http.StatusUnprocessableEntity,
		"/x?categoryId=abc": http.StatusBadRequest,
	} {
		rec := httptest.NewRecorder()
		if _, ok := vendorListFilter(rec, httptest.NewRequest(http.MethodGet, target, nil)); ok || rec.Code != want {
			t.Errorf("%s: ok=%v code=%d, want %d", target, ok, rec.Code, want)
		}
	}
}

func TestVenueListFilterReadsEveryListParameter(t *testing.T) {
	f, ok := venueListFilter(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/x?q=Aula&search=Taman&city=Kota+Bandung&isActive=true", nil))
	if !ok || f.Search != "Aula" || f.City != "Kota Bandung" || f.IsActive == nil || !*f.IsActive {
		t.Errorf("filter = %+v ok=%v", f, ok)
	}
	rec := httptest.NewRecorder()
	if _, ok := venueListFilter(rec, httptest.NewRequest(http.MethodGet, "/x?isActive=2", nil)); ok || rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("bad isActive: ok=%v code=%d", ok, rec.Code)
	}
}
