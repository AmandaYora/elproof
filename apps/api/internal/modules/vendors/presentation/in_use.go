package presentation

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	projectscontracts "elproof/internal/modules/projects/contracts"
	"elproof/internal/shared/response"
)

// Codes of the 409 a directory delete answers while a project still uses the
// row (FQ-02). The body's data is the same shape as delete-impact.
const (
	codeVendorInUse = "vendor_in_use"
	codeVenueInUse  = "venue_in_use"
)

// inUseNamed is how many project names the refusal spells out; the rest are
// counted ("dan 3 proyek lain").
const inUseNamed = 5

// refuseWhileInUse answers 409 and returns true when refs (the projects that
// still use a vendor or venue, from projects' contract) is not empty. A project
// that exists counts whatever its state (archived or cancelled included): it
// still shows the vendor or venue; only a hard-deleted project no longer counts.
// Projects the caller may not open are counted, never named (ADR-0026).
func refuseWhileInUse(ctx context.Context, w http.ResponseWriter, projects picLookup, tenantID int64, refs []projectscontracts.ProjectRef, subject, code string) (bool, error) {
	if len(refs) == 0 {
		return false, nil
	}
	impact, err := scopedImpact(ctx, projects, tenantID, refs)
	if err != nil {
		return false, err
	}
	response.ErrorWithData(w, http.StatusConflict, inUseMessage(subject, impact), map[string]string{"code": code}, impact)
	return true, nil
}

// inUseMessage is the sentence a person reads, e.g. "Vendor ini masih dipakai di
// 2 proyek: A, B. Nonaktifkan saja bila tidak dipakai lagi."
func inUseMessage(subject string, impact deleteImpactResponse) string {
	total := len(impact.AffectedProjects) + impact.HiddenProjectCount
	names := make([]string, 0, inUseNamed)
	for i, p := range impact.AffectedProjects {
		if i == inUseNamed {
			break
		}
		names = append(names, p.Name)
	}
	others := total - len(names)
	list := strings.Join(names, ", ")
	switch {
	case len(names) == 0:
		list = ""
	case others > 0:
		list += fmt.Sprintf(", dan %d proyek lain", others)
	}
	if list != "" {
		list = ": " + list
	}
	return fmt.Sprintf("%s ini masih dipakai di %d proyek%s. Nonaktifkan saja bila tidak dipakai lagi.", subject, total, list)
}
