package middleware

import (
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"elproof/internal/shared/response"
)

// RateLimiter is a simple in-memory, per-key sliding-window limiter — no
// Redis/Memcache per .claude/rules/monorepo.md, which is sufficient given
// this project's single-instance deployment (see ADR-0011). Not meant for
// general-purpose reuse beyond guarding a handful of sensitive endpoints
// (e.g. App token exchange, see knowledge/MODULE_PAYMENT.md §7.1/§7.6).
type RateLimiter struct {
	mu     sync.Mutex
	hits   map[string][]time.Time
	max    int
	window time.Duration
}

func NewRateLimiter(max int, window time.Duration) *RateLimiter {
	return &RateLimiter{hits: make(map[string][]time.Time), max: max, window: window}
}

// Allow records one attempt for key and reports whether it's within budget.
// Old entries are filtered in place (standard Go slice-filter idiom — the
// write index never overtakes the read index, so aliasing the same backing
// array is safe).
func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	cutoff := time.Now().Add(-rl.window)
	kept := rl.hits[key][:0]
	for _, t := range rl.hits[key] {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	if len(kept) >= rl.max {
		rl.hits[key] = kept
		return false
	}
	rl.hits[key] = append(kept, time.Now())
	return true
}

// RateLimit rejects requests once the calling IP exceeds limiter's budget.
// Sets Retry-After (seconds, the limiter's own window) on a 429 so a caller
// doesn't have to guess a backoff — external integrators in particular
// (knowledge/MODULE_PAYMENT.md §7.1) depend on this being present.
func RateLimit(limiter *RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !limiter.Allow(clientIP(r)) {
				w.Header().Set("Retry-After", strconv.Itoa(int(limiter.window.Seconds())))
				response.Error(w, http.StatusTooManyRequests, "Terlalu banyak percobaan, coba lagi nanti", map[string]string{"code": "rate_limited"})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func clientIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		if first, _, ok := strings.Cut(fwd, ","); ok {
			return strings.TrimSpace(first)
		}
		return strings.TrimSpace(fwd)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
