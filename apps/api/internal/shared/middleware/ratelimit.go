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

// Exceeded reports whether key has already used its budget in the current
// window, without recording a new attempt. Paired with Record it lets a
// caller count only *failed* attempts (e.g. wrong passwords) instead of
// every request.
func (rl *RateLimiter) Exceeded(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	cutoff := time.Now().Add(-rl.window)
	n := 0
	for _, t := range rl.hits[key] {
		if t.After(cutoff) {
			n++
		}
	}
	return n >= rl.max
}

// Record counts one attempt for key (see Exceeded).
func (rl *RateLimiter) Record(key string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.hits[key] = append(rl.hits[key], time.Now())
}

// Reset forgets every attempt recorded for key (e.g. after a successful
// login, so earlier typos don't count toward a later lockout).
func (rl *RateLimiter) Reset(key string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	delete(rl.hits, key)
}

// RetryAfterSeconds is the window length, for a 429's Retry-After header.
func (rl *RateLimiter) RetryAfterSeconds() int {
	return int(rl.window.Seconds())
}

// RateLimit rejects requests once the calling IP exceeds limiter's budget.
// Sets Retry-After (seconds, the limiter's own window) on a 429 so a caller
// doesn't have to guess a backoff — external integrators in particular
// (knowledge/MODULE_PAYMENT.md §7.1) depend on this being present.
func RateLimit(limiter *RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !limiter.Allow(ClientIP(r)) {
				w.Header().Set("Retry-After", strconv.Itoa(int(limiter.window.Seconds())))
				response.Error(w, http.StatusTooManyRequests, "Terlalu banyak percobaan, coba lagi nanti", map[string]string{"code": "rate_limited"})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// ClientIP returns the caller's IP for rate limiting. In production the app
// only listens on 127.0.0.1 behind host nginx, which sets X-Real-IP to
// $remote_addr — overwriting anything the caller sent — so it is preferred.
// X-Forwarded-For is only a fallback: nginx *appends* to it
// ($proxy_add_x_forwarded_for), so its first entry is whatever the caller
// claimed and could be rotated to dodge a per-IP limit.
func ClientIP(r *http.Request) string {
	if real := strings.TrimSpace(r.Header.Get("X-Real-IP")); real != "" {
		return real
	}
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
