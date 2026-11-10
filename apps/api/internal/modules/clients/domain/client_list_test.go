package domain

import (
	"reflect"
	"testing"
	"time"
)

func ptrDay(s string) *time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return &t
}

func ids(rows []ClientRow) []int64 {
	out := make([]int64, len(rows))
	for i, r := range rows {
		out[i] = r.Client.ID
	}
	return out
}

func row(id int64, project string, event *time.Time, status string) ClientRow {
	return ClientRow{Client: Client{ID: id}, ProjectName: project, ProjectEventDate: event, LoginStatus: status}
}

func TestSortRowsByProjectNameIsCaseInsensitiveWithIDTieBreaker(t *testing.T) {
	rows := []ClientRow{row(4, "budi & ani", nil, ""), row(2, "Adi & Sari", nil, ""), row(3, "adi & sari", nil, ""), row(1, "Zed", nil, "")}
	SortRows(rows, SortProjectName, false)
	if got := ids(rows); !reflect.DeepEqual(got, []int64{2, 3, 4, 1}) {
		t.Errorf("asc: %v", got)
	}
	SortRows(rows, SortProjectName, true)
	if got := ids(rows); !reflect.DeepEqual(got, []int64{1, 4, 3, 2}) {
		t.Errorf("desc (ties by id descending too): %v", got)
	}
}

func TestSortRowsByEventDateKeepsUnresolvedProjectsLast(t *testing.T) {
	rows := []ClientRow{row(1, "", nil, ""), row(2, "", ptrDay("2027-05-01"), ""), row(3, "", ptrDay("2027-01-01"), ""), row(4, "", nil, "")}
	SortRows(rows, SortProjectEventDate, false)
	if got := ids(rows); !reflect.DeepEqual(got, []int64{3, 2, 1, 4}) {
		t.Errorf("asc: %v", got)
	}
	SortRows(rows, SortProjectEventDate, true)
	if got := ids(rows); !reflect.DeepEqual(got, []int64{2, 3, 4, 1}) {
		t.Errorf("desc: %v", got)
	}
}

func TestSortRowsByLoginStatus(t *testing.T) {
	rows := []ClientRow{row(1, "", nil, "none"), row(2, "", nil, "active"), row(3, "", nil, "invited"), row(4, "", nil, "active")}
	SortRows(rows, SortLoginStatus, false)
	if got := ids(rows); !reflect.DeepEqual(got, []int64{2, 4, 3, 1}) {
		t.Errorf("%v", got)
	}
}

func TestFilterByLoginStatusAndNeedsMemoryPaging(t *testing.T) {
	rows := []ClientRow{row(1, "", nil, "invited"), row(2, "", nil, "active"), row(3, "", nil, "invited")}
	if got := ids(FilterByLoginStatus(rows, "invited")); !reflect.DeepEqual(got, []int64{1, 3}) {
		t.Errorf("%v", got)
	}
	if len(FilterByLoginStatus(rows, "")) != 3 {
		t.Error("an empty status keeps every row")
	}
	if len(FilterByLoginStatus(rows, "none")) != 0 {
		t.Error("no row is in that state")
	}
	for _, c := range []struct {
		sort, status string
		want         bool
	}{{"name", "", false}, {"role", "", false}, {"createdAt", "", false}, {"projectName", "", true}, {"projectEventDate", "", true}, {"loginStatus", "", true}, {"name", "active", true}} {
		if got := NeedsMemoryPaging(c.sort, c.status); got != c.want {
			t.Errorf("%+v: %v", c, got)
		}
	}
	if !IsLoginStatus("invited") || IsLoginStatus("unknown") || !IsClientRole("Bride") || IsClientRole("Cousin") {
		t.Error("vocabulary checks")
	}
}

func TestSortRowsByOwnColumnsMatchesTheDatabaseOrder(t *testing.T) {
	mk := func(id int64, name string, role ClientRole, at string) ClientRow {
		return ClientRow{Client: Client{ID: id, Name: name, Role: role, CreatedAt: *ptrDay(at)}}
	}
	rows := []ClientRow{mk(1, "bima", RoleFamilyRepresentative, "2027-01-03"), mk(2, "Ayu", RoleGroom, "2027-01-02"), mk(3, "ayu", RoleBride, "2027-01-01")}
	SortRows(rows, "name", false)
	if got := ids(rows); !reflect.DeepEqual(got, []int64{2, 3, 1}) {
		t.Errorf("name: %v", got)
	}
	SortRows(rows, "role", false)
	if got := ids(rows); !reflect.DeepEqual(got, []int64{3, 2, 1}) {
		t.Errorf("role (Bride, Groom, Family): %v", got)
	}
	SortRows(rows, "createdAt", true)
	if got := ids(rows); !reflect.DeepEqual(got, []int64{1, 2, 3}) {
		t.Errorf("createdAt desc: %v", got)
	}
}
