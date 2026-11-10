package pagination

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"elproof/internal/shared/apperror"
)

var testWhitelist = map[string]SortColumn{
	"name":      Col("v.name"),
	"price":     NullableCol("v.price"),
	"createdAt": Col("v.created_at"),
}

func req(query string) *http.Request {
	return httptest.NewRequest(http.MethodGet, "/x?"+query, nil)
}

func sortErr(t *testing.T, err error) map[string][]string {
	t.Helper()
	appErr, ok := apperror.As(err)
	if !ok || appErr.Kind != apperror.KindValidation {
		t.Fatalf("want a validation error, got %v", err)
	}
	return appErr.Fields
}

func TestSortDefaultWhenAbsentOrBlank(t *testing.T) {
	for _, q := range []string{"", "sort=", "sort=%20", "page=2"} {
		s, err := SortFromRequest(req(q), testWhitelist, By("createdAt", true))
		if err != nil {
			t.Fatalf("%q: %v", q, err)
		}
		if got := s.OrderBy("v.id"); got != " ORDER BY v.created_at DESC, v.id DESC" {
			t.Errorf("%q: %s", q, got)
		}
		if s.String() != "createdAt:desc" {
			t.Errorf("%q: echo %s", q, s.String())
		}
	}
}

func TestSortDirectionDefaultsToAsc(t *testing.T) {
	s, err := SortFromRequest(req("sort=name"), testWhitelist, By("createdAt", false))
	if err != nil {
		t.Fatal(err)
	}
	if s.Desc || s.String() != "name:asc" {
		t.Errorf("got %s", s.String())
	}
	if got := s.OrderBy("v.id"); got != " ORDER BY v.name ASC, v.id ASC" {
		t.Errorf("got %s", got)
	}
}

func TestSortDirectionIsCaseInsensitiveFieldIsNot(t *testing.T) {
	s, err := SortFromRequest(req("sort=name:DESC"), testWhitelist, By("createdAt", false))
	if err != nil || !s.Desc {
		t.Fatalf("DESC should be accepted: %v %v", s, err)
	}
	if _, err := SortFromRequest(req("sort=Name:asc"), testWhitelist, By("createdAt", false)); err == nil {
		t.Error("field names are exact: Name is not name")
	}
}

func TestSortAlwaysEndsWithThePrimaryKeyInTheSameDirection(t *testing.T) {
	for _, dir := range []string{"asc", "desc"} {
		s, _ := SortFromRequest(req("sort=name:"+dir), testWhitelist, By("createdAt", false))
		got := s.OrderBy("v.id")
		if !strings.HasSuffix(got, ", v.id "+strings.ToUpper(dir)) {
			t.Errorf("%s: no tie-breaker in %q", dir, got)
		}
	}
}

func TestSortNullableColumnPutsNullsLast(t *testing.T) {
	for dir, want := range map[string]string{
		"asc":  " ORDER BY (v.price IS NULL), v.price ASC, v.id ASC",
		"desc": " ORDER BY (v.price IS NULL), v.price DESC, v.id DESC",
	} {
		s, _ := SortFromRequest(req("sort=price:"+dir), testWhitelist, By("createdAt", false))
		if got := s.OrderBy("v.id"); got != want {
			t.Errorf("%s: %q", dir, got)
		}
	}
}

func TestSortRejectsUnknownFieldAndListsTheAllowedOnes(t *testing.T) {
	_, err := SortFromRequest(req("sort=password:asc"), testWhitelist, By("createdAt", false))
	msgs := sortErr(t, err)["sort"]
	if len(msgs) != 1 || msgs[0] != "Urutkan menurut salah satu dari: createdAt, name, price" {
		t.Errorf("message: %v", msgs)
	}
}

func TestSortInjectionAttemptsAreRejectedNotInterpolated(t *testing.T) {
	attempts := []string{
		"name; DROP TABLE vendors",
		"name:asc; DROP TABLE vendors--",
		"name,(SELECT 1)",
		"name:asc,price:desc",
		"1",
		"(SELECT SLEEP(5))",
		"v.name",
		"name:asc UNION SELECT 1",
		"name:asc:desc",
		"name:",
		"name:ascending",
		"`name`",
		"name'--",
		"name%00",
		":asc",
		"name:asc/**/",
		"-name",
	}
	for _, a := range attempts {
		q := url.Values{"sort": {a}}.Encode()
		s, err := SortFromRequest(req(q), testWhitelist, By("createdAt", false))
		if err == nil {
			t.Errorf("%q must be rejected, got %q", a, s.OrderBy("v.id"))
			continue
		}
		sortErr(t, err)
	}
}

func TestSortMoreThanOneParameterIsRejected(t *testing.T) {
	_, err := SortFromRequest(req("sort=name&sort=price"), testWhitelist, By("createdAt", false))
	if msgs := sortErr(t, err)["sort"]; len(msgs) != 1 {
		t.Errorf("got %v", msgs)
	}
}

func TestSortDefaultMustBeWhitelisted(t *testing.T) {
	if _, err := SortFromRequest(req(""), testWhitelist, By("nope", false)); err == nil {
		t.Error("a default outside the whitelist is a programming error and must not silently order by nothing")
	}
}

func TestSortEchoedInMeta(t *testing.T) {
	s, _ := SortFromRequest(req("sort=name:desc"), testWhitelist, By("createdAt", false))
	meta := BuildMeta(Params{Page: 2, Limit: 10}, 35).WithSort(s)
	if meta.Sort != "name:desc" || meta.TotalPages != 4 {
		t.Errorf("%+v", meta)
	}
	if plain := BuildMeta(Params{Page: 1, Limit: 10}, 0); plain.Sort != "" {
		t.Errorf("an endpoint without sorting must not echo one: %+v", plain)
	}
}

func TestRankedColOrdersByPositionInTheNameOrderedList(t *testing.T) {
	allowed := map[string]SortColumn{"planName": Col("")}
	s, err := SortFromRequest(req("sort=planName:desc"), allowed, By("planName", false))
	if err != nil {
		t.Fatal(err)
	}
	s = s.With(RankedCol("t.plan_id", []int64{7, 3, 12}))
	want := " ORDER BY (t.plan_id IS NULL), FIELD(t.plan_id, 7, 3, 12) DESC, t.id DESC"
	if got := s.OrderBy("t.id"); got != want {
		t.Errorf("got %q", got)
	}
	if s.String() != "planName:desc" {
		t.Errorf("echo %s", s.String())
	}
	none := Sort{Field: "planName"}.With(RankedCol("t.plan_id", nil))
	if got := none.OrderBy("t.id"); got != " ORDER BY (t.plan_id IS NULL), t.plan_id ASC, t.id ASC" {
		t.Errorf("no ids degrades to the column: %q", got)
	}
}
