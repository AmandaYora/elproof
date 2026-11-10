package etag

import (
	"testing"
	"time"
)

func TestFormatIsUTCWithSixFractionDigits(t *testing.T) {
	wib := time.FixedZone("WIB", 7*3600)
	in := time.Date(2026, 11, 10, 15, 15, 30, 123456000, wib)
	if got, want := Format(in), "2026-11-10T08:15:30.123456Z"; got != want {
		t.Errorf("Format = %q, want %q", got, want)
	}
	// Trailing zeros stay: the same instant always renders the same bytes.
	if got, want := Format(time.Date(2026, 1, 2, 3, 4, 5, 120000000, time.UTC)), "2026-01-02T03:04:05.120000Z"; got != want {
		t.Errorf("Format = %q, want %q", got, want)
	}
	if got, want := Header(in), `"2026-11-10T08:15:30.123456Z"`; got != want {
		t.Errorf("Header = %q, want %q", got, want)
	}
}

func TestParseIfMatch(t *testing.T) {
	want := time.Date(2026, 11, 10, 8, 15, 30, 123456000, time.UTC)
	for _, ok := range []string{`"2026-11-10T08:15:30.123456Z"`, `2026-11-10T08:15:30.123456Z`, ` "2026-11-10T08:15:30.123456Z" `, `"2026-11-10T15:15:30.123456+07:00"`} {
		got, err := ParseIfMatch(ok)
		if err != nil || got == nil || !got.Equal(want) {
			t.Errorf("ParseIfMatch(%q) = %v, %v; want %v", ok, got, err, want)
		}
	}
	// Nanoseconds below a microsecond are what a TIMESTAMP(6) cannot hold: dropped.
	if got, _ := ParseIfMatch(`"2026-11-10T08:15:30.123456789Z"`); got == nil || !got.Equal(want) {
		t.Errorf("sub-microsecond digits must be truncated, got %v", got)
	}
	// Absent is not an error: the caller writes unconditionally.
	if got, err := ParseIfMatch(""); got != nil || err != nil {
		t.Errorf("absent header: %v, %v", got, err)
	}
	if got, err := ParseIfMatch("   "); got != nil || err != nil {
		t.Errorf("blank header: %v, %v", got, err)
	}
	for _, bad := range []string{"banana", `"banana"`, "*", `W/"2026-11-10T08:15:30.123456Z"`, `"a","b"`, `"2026-11-10T08:15:30.123456Z`, `""`, `"2026-13-45T00:00:00Z"`, "2026-11-10"} {
		if got, err := ParseIfMatch(bad); err != ErrMalformed || got != nil {
			t.Errorf("ParseIfMatch(%q) = %v, %v; want ErrMalformed", bad, got, err)
		}
	}
}

func TestSameIgnoresWhatTheColumnCannotHold(t *testing.T) {
	a := time.Date(2026, 11, 10, 8, 15, 30, 123456000, time.UTC)
	if !Same(a, a.Add(789*time.Nanosecond)) {
		t.Error("below a microsecond is the same version")
	}
	if Same(a, a.Add(time.Microsecond)) {
		t.Error("one microsecond apart is another version")
	}
}
