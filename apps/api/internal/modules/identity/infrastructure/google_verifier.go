package infrastructure

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"elproof/internal/modules/identity/application"
	"elproof/internal/shared/apperror"
)

// Google's published signing keys and issuers for ID tokens
// (developers.google.com/identity/gsi/web/guides/verify-google-id-token).
const (
	GoogleJWKSURL       = "https://www.googleapis.com/oauth2/v3/certs"
	defaultKeysLifetime = time.Hour
	// A token signed with a key we have not seen triggers at most one
	// refetch per minute (key rotation), so garbage tokens cannot make us
	// hammer Google.
	minRefetchInterval = time.Minute
	maxJWKSBytes       = 64 << 10
)

var googleIssuers = map[string]bool{"accounts.google.com": true, "https://accounts.google.com": true}

// GoogleVerifier checks a Google ID token ("credential" from Google
// Identity Services) entirely on our side: RS256 signature against Google's
// public keys, audience = our OAuth client ID, issuer, expiry, and a
// verified email. No call to Google's tokeninfo endpoint is needed.
type GoogleVerifier struct {
	clientID string
	jwksURL  string
	client   *http.Client

	mu        sync.Mutex
	keys      map[string]*rsa.PublicKey
	expiresAt time.Time
	fetchedAt time.Time
}

// NewGoogleVerifier builds a verifier for tokens issued to clientID.
// jwksURL is GoogleJWKSURL in production (tests point it at a local server).
func NewGoogleVerifier(clientID, jwksURL string) *GoogleVerifier {
	return &GoogleVerifier{
		clientID: clientID,
		jwksURL:  jwksURL,
		client:   &http.Client{Timeout: 5 * time.Second},
	}
}

type googleClaims struct {
	Email         string `json:"email"`
	EmailVerified any    `json:"email_verified"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
	jwt.RegisteredClaims
}

func invalidGoogleToken() error {
	return apperror.Unauthorized("Login Google tidak berhasil atau sudah kedaluwarsa. Silakan coba lagi.")
}

// Verify returns the Google account behind a valid ID token.
func (v *GoogleVerifier) Verify(ctx context.Context, idToken string) (*application.GoogleIdentity, error) {
	if v.clientID == "" {
		return nil, apperror.Internal("Login dengan Google belum dikonfigurasi")
	}
	var claims googleClaims
	_, err := jwt.ParseWithClaims(idToken, &claims,
		func(t *jwt.Token) (any, error) {
			kid, _ := t.Header["kid"].(string)
			if kid == "" {
				return nil, errors.New("missing kid")
			}
			return v.key(ctx, kid)
		},
		jwt.WithValidMethods([]string{"RS256"}),
		jwt.WithAudience(v.clientID),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
		jwt.WithLeeway(time.Minute),
	)
	if err != nil {
		return nil, invalidGoogleToken()
	}
	if !googleIssuers[claims.Issuer] || claims.Subject == "" {
		return nil, invalidGoogleToken()
	}
	email := strings.TrimSpace(claims.Email)
	if email == "" || !isTrue(claims.EmailVerified) {
		return nil, apperror.Unauthorized("Email akun Google ini belum terverifikasi. Gunakan akun Google lain.")
	}
	identity := &application.GoogleIdentity{
		Sub:     claims.Subject,
		Email:   email,
		Name:    strings.TrimSpace(claims.Name),
		Picture: claims.Picture,
	}
	if claims.ExpiresAt != nil {
		identity.ExpiresAt = claims.ExpiresAt.Time
	}
	return identity, nil
}

// email_verified is a JSON boolean in current tokens; older ones carried
// the string "true".
func isTrue(v any) bool {
	switch b := v.(type) {
	case bool:
		return b
	case string:
		return b == "true"
	}
	return false
}

func (v *GoogleVerifier) key(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	now := time.Now()
	if k, ok := v.keys[kid]; ok && now.Before(v.expiresAt) {
		return k, nil
	}
	// Unknown kid, stale cache or no keys yet: refresh, but never more than
	// once a minute — also while Google is unreachable, so requests do not
	// queue behind one outbound call each. Stale keys keep working meanwhile.
	if v.fetchedAt.IsZero() || now.Sub(v.fetchedAt) >= minRefetchInterval {
		if err := v.fetch(ctx, now); err != nil {
			// Keep serving known keys if Google is briefly unreachable.
			if k, ok := v.keys[kid]; ok {
				return k, nil
			}
			return nil, err
		}
	}
	if k, ok := v.keys[kid]; ok {
		return k, nil
	}
	return nil, fmt.Errorf("unknown signing key %q", kid)
}

type jwkSet struct {
	Keys []struct {
		Kid string `json:"kid"`
		Kty string `json:"kty"`
		Alg string `json:"alg"`
		Use string `json:"use"`
		N   string `json:"n"`
		E   string `json:"e"`
	} `json:"keys"`
}

// fetch loads Google's current keys; caller holds v.mu.
func (v *GoogleVerifier) fetch(ctx context.Context, now time.Time) error {
	v.fetchedAt = now
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.jwksURL, nil)
	if err != nil {
		return err
	}
	resp, err := v.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("jwks: status %d", resp.StatusCode)
	}
	var set jwkSet
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxJWKSBytes)).Decode(&set); err != nil {
		return err
	}
	keys := make(map[string]*rsa.PublicKey, len(set.Keys))
	for _, k := range set.Keys {
		if k.Kty != "RSA" || k.Kid == "" || (k.Use != "" && k.Use != "sig") {
			continue
		}
		pub, err := rsaKey(k.N, k.E)
		if err != nil {
			continue
		}
		keys[k.Kid] = pub
	}
	if len(keys) == 0 {
		return errors.New("jwks: no usable keys")
	}
	v.keys = keys
	v.expiresAt = now.Add(maxAge(resp.Header.Get("Cache-Control")))
	return nil
}

func rsaKey(n, e string) (*rsa.PublicKey, error) {
	nb, err := base64.RawURLEncoding.DecodeString(n)
	if err != nil {
		return nil, err
	}
	eb, err := base64.RawURLEncoding.DecodeString(e)
	if err != nil {
		return nil, err
	}
	exp := new(big.Int).SetBytes(eb)
	if !exp.IsInt64() || exp.Int64() < 3 || exp.Int64() > 1<<31-1 {
		return nil, errors.New("bad exponent")
	}
	mod := new(big.Int).SetBytes(nb)
	if mod.BitLen() < 2048 {
		return nil, errors.New("key too small")
	}
	return &rsa.PublicKey{N: mod, E: int(exp.Int64())}, nil
}

// maxAge reads Cache-Control's max-age (Google rotates keys and says how
// long the current set is valid); falls back to one hour, capped at a day.
func maxAge(cacheControl string) time.Duration {
	for _, part := range strings.Split(cacheControl, ",") {
		part = strings.TrimSpace(part)
		if v, ok := strings.CutPrefix(part, "max-age="); ok {
			if secs, err := strconv.Atoi(v); err == nil && secs > 0 {
				d := time.Duration(secs) * time.Second
				if d > 24*time.Hour {
					d = 24 * time.Hour
				}
				return d
			}
		}
	}
	return defaultKeysLifetime
}
