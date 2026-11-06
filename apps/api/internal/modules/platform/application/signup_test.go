package application

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"elproof/internal/modules/platform/domain"
)

func TestNormalizeWhatsApp(t *testing.T) {
	cases := map[string]string{
		"0812 3456 7890":  "6281234567890",
		"+62 812-3456-78": "62812345678",
		"62812345678":     "62812345678",
		"812345678":       "62812345678",
		"(0274) 512 345":  "62274512345",
	}
	for in, want := range cases {
		if got := NormalizeWhatsApp(in); got != want {
			t.Errorf("NormalizeWhatsApp(%q) = %q, want %q", in, got, want)
		}
	}
	for _, bad := range []string{"", "12345", "0800", "620812345678"} {
		if waNumber.MatchString(NormalizeWhatsApp(bad)) {
			t.Errorf("%q must not pass as a WhatsApp number", bad)
		}
	}
}

func TestKeyedLocksSerialisePerKey(t *testing.T) {
	var locks keyedLocks
	var inside, maxInside atomic.Int32
	var wg sync.WaitGroup
	for range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			unlock := locks.lock("same-google-account")
			n := inside.Add(1)
			for {
				m := maxInside.Load()
				if n <= m || maxInside.CompareAndSwap(m, n) {
					break
				}
			}
			time.Sleep(time.Millisecond)
			inside.Add(-1)
			unlock()
		}()
	}
	wg.Wait()
	if maxInside.Load() != 1 {
		t.Fatalf("%d runs overlapped for one key", maxInside.Load())
	}
	if len(locks.locks) != 0 {
		t.Fatalf("lock entries leaked: %d", len(locks.locks))
	}
}

func TestTrialLimitsLookup(t *testing.T) {
	l := domain.TrialLimits{MaxStaff: 3, MaxActiveProjects: 5, MaxClients: 10}
	for res, want := range map[string]int{domain.ResourceStaff: 3, domain.ResourceActiveProjects: 5, domain.ResourceClients: 10} {
		if got, ok := l.Limit(res); !ok || got != want {
			t.Errorf("%s: got %d,%v", res, got, ok)
		}
	}
	if _, ok := l.Limit("storage"); ok {
		t.Error("unknown resource must not be limited")
	}
}

func TestValidateBusinessName(t *testing.T) {
	if validateBusinessName(normalizeName("  JWS   Wedding  ")) != nil {
		t.Error("normal name rejected")
	}
	if validateBusinessName("ab") == nil {
		t.Error("too short accepted")
	}
}
