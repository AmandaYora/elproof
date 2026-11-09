package application

import (
	"strings"
	"testing"
	"time"
)

var tokenNow = time.Date(2026, 11, 9, 10, 0, 0, 0, time.UTC)

func testSigner(secret string) *UploadSigner {
	return NewUploadSigner(secret).WithClock(func() time.Time { return tokenNow })
}

func sampleClaims() UploadClaims {
	return UploadClaims{
		TenantID: 7, ProjectID: 42, Key: "elproof/upload/7/42/uploads/abc-bukti.jpg", MimeType: "image/jpeg",
		FileName: "bukti transfer.jpg", MaxSize: 2_000_000, ExpiresAt: tokenNow.Add(UploadTokenTTL), Nonce: NewNonce(),
	}
}

func TestUploadTokenRoundTrip(t *testing.T) {
	s := testSigner("secret-one")
	want := sampleClaims()
	got, err := s.Verify(s.Sign(want), 7, 42)
	if err != nil {
		t.Fatalf("a fresh token for its own project must verify: %v", err)
	}
	if got.Key != want.Key || got.MimeType != want.MimeType || got.FileName != want.FileName || got.MaxSize != want.MaxSize ||
		got.Nonce != want.Nonce || got.TenantID != 7 || got.ProjectID != 42 || !got.ExpiresAt.Equal(want.ExpiresAt) {
		t.Errorf("claims changed on the way: %+v vs %+v", got, want)
	}
}

func TestUploadTokenRefusals(t *testing.T) {
	s := testSigner("secret-one")
	good := s.Sign(sampleClaims())
	payload, sig, _ := strings.Cut(good, ".")

	// A token with another size cap but the old signature: tampered payload.
	other := sampleClaims()
	other.MaxSize = 99_999_999
	otherPayload, _, _ := strings.Cut(s.Sign(other), ".")

	expired := sampleClaims()
	expired.ExpiresAt = tokenNow.Add(-time.Second)
	atBoundary := sampleClaims()
	atBoundary.ExpiresAt = tokenNow // not before now: expired

	cases := []struct {
		name          string
		token         string
		tenant, proj  int64
		want          error
		signedByOther bool
	}{
		{"wrong project", good, 7, 43, ErrTokenScope, false},
		{"wrong tenant", good, 8, 42, ErrTokenScope, false},
		{"expired", s.Sign(expired), 7, 42, ErrTokenExpired, false},
		{"expires exactly now", s.Sign(atBoundary), 7, 42, ErrTokenExpired, false},
		{"payload swapped under the old signature", otherPayload + "." + sig, 7, 42, ErrTokenInvalid, false},
		{"signature flipped", payload + "." + sig[:len(sig)-2] + "AA", 7, 42, ErrTokenInvalid, false},
		{"signature missing", payload + ".", 7, 42, ErrTokenInvalid, false},
		{"no separator", payload, 7, 42, ErrTokenInvalid, false},
		{"empty", "", 7, 42, ErrTokenInvalid, false},
		{"garbage", "not.a token", 7, 42, ErrTokenInvalid, false},
		{"signed with another secret", testSigner("secret-two").Sign(sampleClaims()), 7, 42, ErrTokenInvalid, false},
	}
	for _, c := range cases {
		if _, err := s.Verify(c.token, c.tenant, c.proj); err != c.want {
			t.Errorf("%s: got %v, want %v", c.name, err, c.want)
		}
	}
}

// A valid signature on a token for another place is checked after the signature:
// a forged token never reaches the scope question.
func TestUploadTokenSignatureIsCheckedBeforeScope(t *testing.T) {
	forged := testSigner("attacker").Sign(sampleClaims())
	if _, err := testSigner("secret-one").Verify(forged, 8, 99); err != ErrTokenInvalid {
		t.Errorf("forged token must be invalid, not %v", err)
	}
}

func TestUploadTokensDifferPerGrant(t *testing.T) {
	s := testSigner("secret-one")
	a, b := sampleClaims(), sampleClaims()
	if s.Sign(a) == s.Sign(b) {
		t.Error("two grants for the same file must not produce the same token (nonce)")
	}
}
