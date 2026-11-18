package clock

import (
	"testing"
	"time"
)

func TestRealByDefault(t *testing.T) {
	Reset()
	if Shifted() {
		t.Fatal("clock shifted without StartAt")
	}
	if d := time.Since(Now()); d < 0 || d > time.Second {
		t.Fatalf("Now() is %v away from time.Now", d)
	}
}

func TestStartAtTicksFromTheGivenInstant(t *testing.T) {
	t.Cleanup(Reset)
	start, err := Parse("2026-11-09T10:00:00+07:00")
	if err != nil {
		t.Fatal(err)
	}
	StartAt(start)
	if !Shifted() {
		t.Fatal("Shifted() = false after StartAt")
	}
	first := Now()
	if d := first.Sub(start); d < 0 || d > time.Second {
		t.Fatalf("Now() = %v, want about %v", first, start)
	}
	time.Sleep(5 * time.Millisecond)
	if !Now().After(first) {
		t.Fatal("the shifted clock does not advance")
	}
}

func TestParseRejectsAValueWithoutOffset(t *testing.T) {
	for _, v := range []string{"2026-11-09", "2026-11-09T10:00:00", "soon"} {
		if _, err := Parse(v); err == nil {
			t.Errorf("Parse(%q) accepted", v)
		}
	}
}
