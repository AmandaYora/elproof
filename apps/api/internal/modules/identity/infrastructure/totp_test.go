package infrastructure

import (
	"encoding/hex"
	"testing"
	"time"
)

// RFC 6238 Appendix B test vectors (SHA-1, secret "12345678901234567890"),
// truncated to the 6 digits authenticator apps show.
func TestHOTPMatchesRFC6238Vectors(t *testing.T) {
	secret := []byte("12345678901234567890")
	vectors := map[int64]string{59: "287082", 1111111109: "081804", 1111111111: "050471", 1234567890: "005924", 2000000000: "279037"}
	for unix, want := range vectors {
		if got := hotp(secret, unix/30); got != want {
			t.Errorf("t=%d: got %s, want %s", unix, got, want)
		}
	}
}

func TestVerifyWindowAndReplay(t *testing.T) {
	tp, err := NewTOTP("ElProof", "jwt-secret")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Unix(1_800_000_000, 0)
	tp.now = func() time.Time { return now }
	raw, _ := hex.DecodeString("3132333435363738393031323334353637383930")
	step := now.Unix() / 30

	if _, ok := tp.Verify(raw, hotp(raw, step), 0); !ok {
		t.Error("current code must verify")
	}
	if _, ok := tp.Verify(raw, hotp(raw, step-1), 0); !ok {
		t.Error("previous step must verify (clock drift)")
	}
	if _, ok := tp.Verify(raw, hotp(raw, step-2), 0); ok {
		t.Error("code two steps old must be rejected")
	}
	if _, ok := tp.Verify(raw, hotp(raw, step), step); ok {
		t.Error("a code already used (step <= lastStep) must be rejected")
	}
	if _, ok := tp.Verify(raw, "12345", 0); ok {
		t.Error("wrong length must be rejected")
	}
}

func TestSealOpenRoundTrip(t *testing.T) {
	tp, _ := NewTOTP("ElProof", "jwt-secret")
	raw, encoded, err := tp.NewSecret()
	if err != nil || len(raw) != 20 || len(encoded) != 32 {
		t.Fatalf("secret: %v %d %q", err, len(raw), encoded)
	}
	enc, err := tp.Seal(raw)
	if err != nil {
		t.Fatal(err)
	}
	got, err := tp.Open(enc)
	if err != nil || string(got) != string(raw) {
		t.Fatalf("round trip failed: %v", err)
	}
	other, _ := NewTOTP("ElProof", "different-jwt-secret")
	if _, err := other.Open(enc); err == nil {
		t.Error("a different master secret must not decrypt")
	}
	if u := tp.URL("superadmin", encoded); u[:15] != "otpauth://totp/" {
		t.Errorf("bad url %s", u)
	}
}
