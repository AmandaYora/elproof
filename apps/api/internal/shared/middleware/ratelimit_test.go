package middleware

import (
	"net/http/httptest"
	"testing"
	"time"
)

func TestClientIPTrustsHeadersOnlyFromProxy(t *testing.T) {
	cases := []struct {
		remote, realIP, want string
	}{
		{"127.0.0.1:5000", "203.0.113.9", "203.0.113.9"},     // host nginx over loopback
		{"172.17.0.1:5000", "203.0.113.9", "203.0.113.9"},    // nginx via Docker bridge
		{"198.51.100.7:5000", "203.0.113.9", "198.51.100.7"}, // direct public caller: header ignored
		{"127.0.0.1:5000", "", "127.0.0.1"},
	}
	for _, c := range cases {
		r := httptest.NewRequest("POST", "/api/v1/auth/login", nil)
		r.RemoteAddr = c.remote
		if c.realIP != "" {
			r.Header.Set("X-Real-IP", c.realIP)
		}
		if got := ClientIP(r); got != c.want {
			t.Errorf("remote=%s real=%q: got %s, want %s", c.remote, c.realIP, got, c.want)
		}
	}
}

func TestRateLimiterSweepsExpiredKeys(t *testing.T) {
	rl := NewRateLimiter(3, time.Millisecond)
	for i := 0; i < 50; i++ {
		rl.Record(string(rune('a'+i%26)) + string(rune('A'+i/26)))
	}
	time.Sleep(5 * time.Millisecond)
	for i := 0; i < sweepEvery; i++ {
		rl.Record("hot")
	}
	rl.mu.Lock()
	n := len(rl.hits)
	rl.mu.Unlock()
	if n > 2 {
		t.Fatalf("expired keys should be swept, %d remain", n)
	}
}

func TestRateLimiterExceededAndRecord(t *testing.T) {
	rl := NewRateLimiter(2, time.Minute)
	if rl.Exceeded("k") {
		t.Fatal("fresh key must not be exceeded")
	}
	rl.Record("k")
	rl.Record("k")
	if !rl.Exceeded("k") {
		t.Fatal("two failures with max 2 must be exceeded")
	}
	rl.Reset("k")
	if rl.Exceeded("k") {
		t.Fatal("reset must clear the key")
	}
}
