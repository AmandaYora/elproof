package presentation

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"elproof/internal/shared/logger"
)

// Every mutation answers with what changed AND the project's progress computed
// after the change (E8), so a screen can update its header and counters from the
// answer instead of reloading. withExtras is the one helper that does it: the
// item is serialised exactly as before and the extra keys are added next to its
// own, so existing readers of the item keep working.

type extra struct {
	key   string
	value any
}

// withExtras marshals item (a JSON object, or nil for "nothing but the extras")
// with extra top-level keys appended.
type withExtras struct {
	item   any
	extras []extra
}

func (w withExtras) MarshalJSON() ([]byte, error) {
	var base []byte
	if w.item != nil {
		b, err := json.Marshal(w.item)
		if err != nil {
			return nil, err
		}
		base = b
	}
	if len(base) == 0 || string(base) == "null" {
		base = []byte("{}")
	}
	if base[0] != '{' || base[len(base)-1] != '}' {
		return nil, errors.New("withExtras: the item must serialise to a JSON object")
	}
	out := append([]byte(nil), base[:len(base)-1]...)
	needComma := len(base) > 2
	for _, e := range w.extras {
		v, err := json.Marshal(e.value)
		if err != nil {
			return nil, err
		}
		k, _ := json.Marshal(e.key)
		if needComma {
			out = append(out, ',')
		}
		out = append(out, k...)
		out = append(out, ':')
		out = append(out, v...)
		needComma = true
	}
	return append(out, '}'), nil
}

// projectProgress computes the project's progress now (staff view). It is what
// every mutation answer carries as `projectProgress`. The mutation has already
// been committed when this runs, so a failure here must not turn the answer into
// an error (a retry of a create would duplicate): it is logged and the key is
// left out.
func (h *Handler) projectProgress(ctx context.Context, tenantID, projectID int64) *progressResponse {
	p, err := h.projects.ComputeProgress(ctx, tenantID, projectID, time.Now())
	if err != nil {
		logger.Error("projectProgress for project %d after a mutation: %v", projectID, err)
		return nil
	}
	r := toProgressResponse(*p)
	return &r
}

// mutated is item plus `projectProgress` (and any other extras, such as
// `ledgerSummary`). Pass a nil item for a DELETE that used to answer null: the
// answer is then just {"projectProgress": ...}.
func (h *Handler) mutated(ctx context.Context, tenantID, projectID int64, item any, more ...extra) withExtras {
	ex := append([]extra(nil), more...)
	if pr := h.projectProgress(ctx, tenantID, projectID); pr != nil {
		ex = append(ex, extra{key: "projectProgress", value: pr})
	}
	return withExtras{item: item, extras: ex}
}

// deletedResponse is the body of a soft DELETE: what was deleted and until when
// it can be restored.
type deletedResponse struct {
	ID              int64  `json:"id"`
	RestorableUntil string `json:"restorableUntil"`
}

func toDeletedResponse(id int64, until time.Time) deletedResponse {
	return deletedResponse{ID: id, RestorableUntil: until.In(wib).Format(time.RFC3339)}
}
