package infrastructure

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const testClientID = "test-client.apps.googleusercontent.com"

type fakeGoogle struct {
	key     *rsa.PrivateKey
	kid     string
	fetches atomic.Int32
	server  *httptest.Server
}

func newFakeGoogle(t *testing.T) *fakeGoogle {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	g := &fakeGoogle{key: key, kid: "k1"}
	g.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		g.fetches.Add(1)
		w.Header().Set("Cache-Control", "public, max-age=3600")
		_ = json.NewEncoder(w).Encode(map[string]any{"keys": []map[string]string{{
			"kid": g.kid, "kty": "RSA", "alg": "RS256", "use": "sig",
			"n": base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
			"e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes()),
		}}})
	}))
	t.Cleanup(g.server.Close)
	return g
}

func (g *fakeGoogle) sign(t *testing.T, key *rsa.PrivateKey, kid string, claims jwt.MapClaims) string {
	t.Helper()
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tok.Header["kid"] = kid
	s, err := tok.SignedString(key)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func validClaims() jwt.MapClaims {
	now := time.Now()
	return jwt.MapClaims{
		"iss": "https://accounts.google.com", "aud": testClientID, "sub": "1234567890",
		"email": "owner@example.com", "email_verified": true, "name": "Owner WO",
		"iat": now.Unix(), "exp": now.Add(time.Hour).Unix(),
	}
}

func TestGoogleVerifierAcceptsValidToken(t *testing.T) {
	g := newFakeGoogle(t)
	v := NewGoogleVerifier(testClientID, g.server.URL)
	id, err := v.Verify(context.Background(), g.sign(t, g.key, g.kid, validClaims()))
	if err != nil {
		t.Fatalf("valid token rejected: %v", err)
	}
	if id.Sub != "1234567890" || id.Email != "owner@example.com" || id.Name != "Owner WO" {
		t.Fatalf("unexpected identity: %+v", id)
	}
	// Second verification uses the cached keys.
	if _, err := v.Verify(context.Background(), g.sign(t, g.key, g.kid, validClaims())); err != nil {
		t.Fatal(err)
	}
	if g.fetches.Load() != 1 {
		t.Fatalf("keys fetched %d times, want 1", g.fetches.Load())
	}
}

func TestGoogleVerifierRejects(t *testing.T) {
	g := newFakeGoogle(t)
	other, _ := rsa.GenerateKey(rand.Reader, 2048)
	cases := map[string]func() string{
		"forged signature": func() string { return g.sign(t, other, g.kid, validClaims()) },
		"wrong audience": func() string {
			c := validClaims()
			c["aud"] = "someone-else.apps.googleusercontent.com"
			return g.sign(t, g.key, g.kid, c)
		},
		"wrong issuer": func() string {
			c := validClaims()
			c["iss"] = "https://evil.example.com"
			return g.sign(t, g.key, g.kid, c)
		},
		"expired": func() string {
			c := validClaims()
			c["exp"] = time.Now().Add(-2 * time.Hour).Unix()
			c["iat"] = time.Now().Add(-3 * time.Hour).Unix()
			return g.sign(t, g.key, g.kid, c)
		},
		"no expiry": func() string {
			c := validClaims()
			delete(c, "exp")
			return g.sign(t, g.key, g.kid, c)
		},
		"unverified email": func() string {
			c := validClaims()
			c["email_verified"] = false
			return g.sign(t, g.key, g.kid, c)
		},
		"no email": func() string {
			c := validClaims()
			delete(c, "email")
			return g.sign(t, g.key, g.kid, c)
		},
		"no subject": func() string {
			c := validClaims()
			delete(c, "sub")
			return g.sign(t, g.key, g.kid, c)
		},
		"unknown key id": func() string { return g.sign(t, g.key, "nope", validClaims()) },
		"HS256 confusion": func() string {
			tok := jwt.NewWithClaims(jwt.SigningMethodHS256, validClaims())
			tok.Header["kid"] = g.kid
			s, _ := tok.SignedString([]byte("secret"))
			return s
		},
		"garbage": func() string { return "not.a.token" },
	}
	for name, mk := range cases {
		t.Run(name, func(t *testing.T) {
			v := NewGoogleVerifier(testClientID, g.server.URL)
			if _, err := v.Verify(context.Background(), mk()); err == nil {
				t.Fatal("accepted, want rejection")
			}
		})
	}
}

func TestGoogleVerifierLegacyStringEmailVerified(t *testing.T) {
	g := newFakeGoogle(t)
	c := validClaims()
	c["email_verified"] = "true"
	c["iss"] = "accounts.google.com"
	if _, err := NewGoogleVerifier(testClientID, g.server.URL).Verify(context.Background(), g.sign(t, g.key, g.kid, c)); err != nil {
		t.Fatalf("string email_verified/bare issuer rejected: %v", err)
	}
}

func TestGoogleVerifierUnknownKidRefetchIsThrottled(t *testing.T) {
	g := newFakeGoogle(t)
	v := NewGoogleVerifier(testClientID, g.server.URL)
	for range 5 {
		_, _ = v.Verify(context.Background(), g.sign(t, g.key, "rotated-away", validClaims()))
	}
	if n := g.fetches.Load(); n != 1 {
		t.Fatalf("unknown kid caused %d fetches, want 1 (throttled)", n)
	}
}

func TestGoogleVerifierNotConfigured(t *testing.T) {
	if _, err := NewGoogleVerifier("", "http://unused").Verify(context.Background(), "x"); err == nil {
		t.Fatal("verifier without client ID must refuse")
	}
}

func TestMaxAge(t *testing.T) {
	if got := maxAge("public, max-age=19800, must-revalidate"); got != 19800*time.Second {
		t.Fatalf("got %v", got)
	}
	if got := maxAge(""); got != defaultKeysLifetime {
		t.Fatalf("default: got %v", got)
	}
	if got := maxAge("max-age=999999"); got != 24*time.Hour {
		t.Fatalf("cap: got %v", got)
	}
}

func TestGoogleVerifierBacksOffWhileGoogleIsDown(t *testing.T) {
	var fetches atomic.Int32
	down := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fetches.Add(1)
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	defer down.Close()
	g := newFakeGoogle(t)
	v := NewGoogleVerifier(testClientID, down.URL)
	for range 5 {
		if _, err := v.Verify(context.Background(), g.sign(t, g.key, g.kid, validClaims())); err == nil {
			t.Fatal("verified without keys")
		}
	}
	if n := fetches.Load(); n != 1 {
		t.Fatalf("Google down: %d fetches for 5 logins, want 1", n)
	}
}
