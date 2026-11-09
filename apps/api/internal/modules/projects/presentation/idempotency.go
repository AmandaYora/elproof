package presentation

import (
	"bytes"
	"context"
	"io"
	"net/http"

	"elproof/internal/modules/projects/application"
	"elproof/internal/shared/logger"
	"elproof/internal/shared/response"
)

// capture records a response instead of sending it, so the handler's answer can
// be stored under an Idempotency-Key before it goes out.
type capture struct {
	header http.Header
	status int
	body   bytes.Buffer
}

func newCapture() *capture { return &capture{header: http.Header{}, status: http.StatusOK} }

func (c *capture) Header() http.Header         { return c.header }
func (c *capture) WriteHeader(code int)        { c.status = code }
func (c *capture) Write(b []byte) (int, error) { return c.body.Write(b) }

func (c *capture) sendTo(w http.ResponseWriter) {
	for k, v := range c.header {
		w.Header()[k] = v
	}
	w.WriteHeader(c.status)
	_, _ = w.Write(c.body.Bytes())
}

// maxIdempotentBody bounds the body read for hashing (these bodies are tiny JSON).
const maxIdempotentBody = 1 << 20

// idempotent runs a create handler under the optional Idempotency-Key header
// (E3/E7). Without the header nothing changes. With it:
//
//   - first time: the handler runs; a 2xx answer is stored with the key and sent;
//     any other answer releases the key, so fixing the request and retrying with
//     the same key is a fresh attempt, not a "reuse";
//   - same key, same body, already succeeded: the stored status and body are sent
//     again with `Idempotent-Replay: true`, and nothing is created;
//   - same key, different body (or another person): 422 idempotency_key_reuse;
//   - same key while the first request is still running: 409.
//
// A key is scoped to the workspace and to the method+path (so project, and
// which of the four calls). A malformed key is a 422 on "Idempotency-Key".
func (h *Handler) idempotent(w http.ResponseWriter, r *http.Request, claims staffClaims, next func(http.ResponseWriter, *http.Request)) {
	key := r.Header.Get(application.IdempotencyKeyHeader)
	if key == "" || h.idem == nil {
		next(w, r)
		return
	}
	if !application.ValidIdempotencyKey(key) {
		writeAppError(w, application.ErrBadIdempotencyKey())
		return
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxIdempotentBody))
	if err != nil {
		response.Error(w, http.StatusRequestEntityTooLarge, "Body permintaan terlalu besar", nil)
		return
	}
	r.Body = io.NopCloser(bytes.NewReader(body))

	route := r.Method + " " + r.URL.Path
	outcome, rec, err := h.idem.Begin(r.Context(), claims.tenantID, claims.staffID, route, key, application.HashRequest(body))
	if err != nil {
		writeAppError(w, err)
		return
	}
	switch outcome {
	case application.IdemReplay:
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Idempotent-Replay", "true")
		w.WriteHeader(*rec.StatusCode)
		_, _ = w.Write(rec.Body)
		return
	case application.IdemReuse:
		writeAppError(w, application.ErrIdempotencyReuse())
		return
	case application.IdemInProgress:
		writeAppError(w, application.ErrIdempotencyInProgress())
		return
	}

	// This request holds the key. It must be finished or released, whatever happens.
	settle := context.WithoutCancel(r.Context())
	finished := false
	defer func() {
		if !finished {
			if err := h.idem.Release(settle, claims.tenantID, route, key); err != nil {
				logger.Error("failed to release idempotency key: %v", err)
			}
		}
	}()
	out := newCapture()
	next(out, r)
	if out.status >= 200 && out.status < 300 {
		if err := h.idem.Complete(settle, claims.tenantID, route, key, out.status, out.body.Bytes()); err != nil {
			// The action is done; only the replay record is missing. The key
			// stays "running" and is released for good after its stale window.
			logger.Error("failed to store idempotent response: %v", err)
		}
		finished = true
	}
	out.sendTo(w)
}
