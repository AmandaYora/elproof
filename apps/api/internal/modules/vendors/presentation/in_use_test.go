package presentation

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	projectscontracts "elproof/internal/modules/projects/contracts"
)

// FQ-02: deleting a vendor or venue a project still uses is refused with a 409
// that says where it is used and what to do instead.
func TestRefuseWhileInUse(t *testing.T) {
	projects := fakeProjects{pic: map[int64]int64{1: 7, 2: 9, 3: 7}}
	refs := []projectscontracts.ProjectRef{{ID: 1, Name: "Andi & Rina"}, {ID: 2, Name: "Proyek orang lain"}, {ID: 3, Name: "Budi & Sari"}}

	rec := httptest.NewRecorder()
	refused, err := refuseWhileInUse(staffCtx("7", true), rec, projects, 1, refs, "Vendor", codeVendorInUse)
	if err != nil || !refused {
		t.Fatalf("refused=%v err=%v", refused, err)
	}
	if rec.Code != http.StatusConflict {
		t.Fatalf("status %d", rec.Code)
	}
	var body struct {
		Success bool              `json:"success"`
		Message string            `json:"message"`
		Errors  map[string]string `json:"errors"`
		Data    deleteImpactResponse
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	want := "Vendor ini masih dipakai di 3 proyek: Andi & Rina, Proyek orang lain, Budi & Sari. Nonaktifkan saja bila tidak dipakai lagi."
	if body.Success || body.Message != want || body.Errors["code"] != codeVendorInUse || len(body.Data.AffectedProjects) != 3 {
		t.Errorf("body: %s", rec.Body.String())
	}

	// An "assigned" role is told how many projects, never the names it cannot open.
	rec = httptest.NewRecorder()
	if refused, err := refuseWhileInUse(staffCtx("7", false), rec, projects, 1, refs, "Venue", codeVenueInUse); err != nil || !refused {
		t.Fatalf("refused=%v err=%v", refused, err)
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	want = "Venue ini masih dipakai di 3 proyek: Andi & Rina, Budi & Sari, dan 1 proyek lain. Nonaktifkan saja bila tidak dipakai lagi."
	if body.Message != want || body.Data.HiddenProjectCount != 1 || body.Errors["code"] != codeVenueInUse {
		t.Errorf("assigned body: %s", rec.Body.String())
	}

	// Nothing uses it: nothing is written and the delete goes on.
	rec = httptest.NewRecorder()
	if refused, err := refuseWhileInUse(staffCtx("7", true), rec, projects, 1, nil, "Vendor", codeVendorInUse); err != nil || refused || rec.Body.Len() != 0 {
		t.Errorf("unused: refused=%v err=%v body=%q", refused, err, rec.Body.String())
	}
}

func TestInUseMessageNamesAtMostFive(t *testing.T) {
	impact := deleteImpactResponse{HiddenProjectCount: 2}
	for _, n := range []string{"A", "B", "C", "D", "E", "F", "G"} {
		impact.AffectedProjects = append(impact.AffectedProjects, deleteImpactProjectRef{Name: n})
	}
	want := "Vendor ini masih dipakai di 9 proyek: A, B, C, D, E, dan 4 proyek lain. Nonaktifkan saja bila tidak dipakai lagi."
	if got := inUseMessage("Vendor", impact); got != want {
		t.Errorf("got %q", got)
	}
	if got := inUseMessage("Venue", deleteImpactResponse{HiddenProjectCount: 2}); got != "Venue ini masih dipakai di 2 proyek. Nonaktifkan saja bila tidak dipakai lagi." {
		t.Errorf("only hidden: %q", got)
	}
}
