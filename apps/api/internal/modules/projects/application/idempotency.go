package application

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"regexp"

	"elproof/internal/shared/apperror"
)

// IdempotencyKeyHeader is the request header that makes a create call safe to retry.
const IdempotencyKeyHeader = "Idempotency-Key"

var idempotencyKeyPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{8,128}$`)

// ValidIdempotencyKey: 8 to 128 characters, letters, digits, "_" and "-".
func ValidIdempotencyKey(key string) bool { return idempotencyKeyPattern.MatchString(key) }

// ErrBadIdempotencyKey is the 422 for a malformed key.
func ErrBadIdempotencyKey() error {
	msg := "Idempotency-Key harus 8 sampai 128 karakter: huruf, angka, garis bawah, atau tanda hubung."
	return apperror.Validation(msg, map[string][]string{IdempotencyKeyHeader: {msg}})
}

// HashRequest fingerprints a request body independent of key order and
// whitespace, so a retry that re-serialises the same JSON is still "the same
// request". A body that is not JSON is hashed as it is.
func HashRequest(body []byte) string {
	canonical := body
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err == nil {
		if out, err := json.Marshal(v); err == nil {
			canonical = out
		}
	}
	sum := sha256.Sum256(canonical)
	return hex.EncodeToString(sum[:])
}

// IdempotencyRecord is what is stored for one key.
type IdempotencyRecord struct {
	StaffID     int64
	RequestHash string
	// StatusCode is nil while the first request is still running.
	StatusCode *int
	Body       []byte
	// Stale: still running but older than any live request can be (it died).
	Stale bool
}

// IdempotencyRepository is the storage of keys (table idempotency_keys). Rows
// older than 24 hours do not exist as far as callers can tell.
type IdempotencyRepository interface {
	// Reserve claims the key for a new request. When the key is already there it
	// changes nothing and returns the existing record; otherwise nil.
	Reserve(ctx context.Context, tenantID, staffID int64, route, key, requestHash string) (*IdempotencyRecord, error)
	// TakeOver re-claims a key whose first request died, and says whether this
	// caller won (two retries cannot both win).
	TakeOver(ctx context.Context, tenantID, staffID int64, route, key, requestHash string) (bool, error)
	// Complete stores the response of the request that holds the key.
	Complete(ctx context.Context, tenantID int64, route, key string, statusCode int, body []byte) error
	// Release gives the key back (the request did not succeed, so a retry is a
	// fresh attempt).
	Release(ctx context.Context, tenantID int64, route, key string) error
}

// IdemOutcome is what to do with a request that carries a key.
type IdemOutcome int

const (
	// IdemProceed: first time (or the first attempt died): run the request.
	IdemProceed IdemOutcome = iota
	// IdemReplay: the same request already succeeded: answer with the stored response.
	IdemReplay
	// IdemInProgress: the same request is running right now: 409.
	IdemInProgress
	// IdemReuse: the key was used for a different request: 422.
	IdemReuse
	// IdemTakeOver: the first attempt died; claim the key and run the request.
	IdemTakeOver
)

// DecideIdempotency is the whole decision table:
//
//	no record                              -> Proceed
//	other person, or other body            -> Reuse   (even while running)
//	same request, finished                 -> Replay
//	same request, running                  -> InProgress (409)
//	same request, running but long dead    -> TakeOver
func DecideIdempotency(existing *IdempotencyRecord, staffID int64, requestHash string) IdemOutcome {
	switch {
	case existing == nil:
		return IdemProceed
	case existing.StaffID != staffID || existing.RequestHash != requestHash:
		return IdemReuse
	case existing.StatusCode != nil:
		return IdemReplay
	case existing.Stale:
		return IdemTakeOver
	default:
		return IdemInProgress
	}
}

// IdempotencyService decides, for a request that carries a key, whether to run it.
type IdempotencyService struct {
	repo IdempotencyRepository
}

func NewIdempotencyService(repo IdempotencyRepository) *IdempotencyService {
	return &IdempotencyService{repo: repo}
}

// Begin reserves the key or reports why it cannot. On IdemProceed the caller
// MUST finish with Complete (success) or Release (anything else). On IdemReplay
// the returned record holds the stored response.
func (s *IdempotencyService) Begin(ctx context.Context, tenantID, staffID int64, route, key, requestHash string) (IdemOutcome, *IdempotencyRecord, error) {
	existing, err := s.repo.Reserve(ctx, tenantID, staffID, route, key, requestHash)
	if err != nil {
		return 0, nil, err
	}
	outcome := DecideIdempotency(existing, staffID, requestHash)
	if outcome == IdemTakeOver {
		won, err := s.repo.TakeOver(ctx, tenantID, staffID, route, key, requestHash)
		if err != nil {
			return 0, nil, err
		}
		if won {
			return IdemProceed, nil, nil
		}
		return IdemInProgress, existing, nil
	}
	return outcome, existing, nil
}

func (s *IdempotencyService) Complete(ctx context.Context, tenantID int64, route, key string, statusCode int, body []byte) error {
	return s.repo.Complete(ctx, tenantID, route, key, statusCode, body)
}

func (s *IdempotencyService) Release(ctx context.Context, tenantID int64, route, key string) error {
	return s.repo.Release(ctx, tenantID, route, key)
}

// ErrIdempotencyReuse is the 422 for a key already used with a different request.
func ErrIdempotencyReuse() error {
	msg := "Idempotency-Key ini sudah dipakai untuk permintaan yang berbeda. Buat kunci baru untuk permintaan baru."
	return apperror.Validation(msg, map[string][]string{IdempotencyKeyHeader: {msg}}).WithCode("idempotency_key_reuse")
}

// ErrIdempotencyInProgress is the 409 for a retry that arrives while the first
// request is still running.
func ErrIdempotencyInProgress() error {
	return apperror.Conflict("Permintaan yang sama masih diproses. Tunggu sebentar lalu coba lagi.").WithCode("idempotency_in_progress")
}
