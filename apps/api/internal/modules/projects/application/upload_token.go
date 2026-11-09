package application

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

// UploadTokenTTL is how long a signed upload (the URL and its token) works.
const UploadTokenTTL = 15 * time.Minute

// Why a token was refused. The three are told apart so the message can say what
// to do (upload again), never so a caller can probe the signature.
var (
	ErrTokenInvalid = errors.New("upload token is malformed or not signed by this server")
	ErrTokenExpired = errors.New("upload token has expired")
	ErrTokenScope   = errors.New("upload token belongs to another workspace or project")
)

// UploadClaims is everything a signed upload carries. The server keeps no
// record of a grant: the token IS the record, so nothing has to be stored,
// cleaned up or looked up before the payment that uses it arrives.
type UploadClaims struct {
	TenantID  int64
	ProjectID int64
	// Key is the object key the file must be stored under.
	Key      string
	MimeType string
	FileName string
	// MaxSize is the size cap (bytes) the object must respect.
	MaxSize   int64
	ExpiresAt time.Time
	// Nonce makes two grants for the same file differ.
	Nonce string
}

// wireClaims is the compact, stable JSON form signed inside the token.
type wireClaims struct {
	T int64  `json:"t"`
	P int64  `json:"p"`
	K string `json:"k"`
	M string `json:"m"`
	F string `json:"f"`
	S int64  `json:"s"`
	E int64  `json:"e"`
	N string `json:"n"`
}

// UploadSigner signs and verifies upload tokens (HMAC-SHA256).
type UploadSigner struct {
	key []byte
	now func() time.Time
}

// NewUploadSigner derives the signing key from the application's one secret
// (the JWT secret): HMAC(secret, purpose label). The purpose label means a
// token can never be mistaken for a login token, nor the reverse, although
// both start from the same secret; there is no second secret to configure.
func NewUploadSigner(secret string) *UploadSigner {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte("elproof/projects/upload-token/v1"))
	return &UploadSigner{key: mac.Sum(nil), now: time.Now}
}

// WithClock replaces the clock (tests).
func (s *UploadSigner) WithClock(now func() time.Time) *UploadSigner {
	c := *s
	c.now = now
	return &c
}

// NewNonce returns 16 random bytes, hex encoded.
func NewNonce() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand unavailable: " + err.Error())
	}
	return hex.EncodeToString(b)
}

func (s *UploadSigner) sign(payload []byte) []byte {
	mac := hmac.New(sha256.New, s.key)
	mac.Write(payload)
	return mac.Sum(nil)
}

// Sign returns "<payload>.<signature>", both base64url without padding.
func (s *UploadSigner) Sign(c UploadClaims) string {
	raw, _ := json.Marshal(wireClaims{
		T: c.TenantID, P: c.ProjectID, K: c.Key, M: c.MimeType, F: c.FileName, S: c.MaxSize, E: c.ExpiresAt.Unix(), N: c.Nonce,
	})
	enc := base64.RawURLEncoding
	return enc.EncodeToString(raw) + "." + enc.EncodeToString(s.sign(raw))
}

// Verify checks the signature, then the expiry, then that the token was granted
// for exactly this workspace and project. The signature is compared in constant
// time and is checked before anything in the payload is trusted.
func (s *UploadSigner) Verify(token string, tenantID, projectID int64) (UploadClaims, error) {
	payload, sig, ok := strings.Cut(strings.TrimSpace(token), ".")
	if !ok || payload == "" || sig == "" {
		return UploadClaims{}, ErrTokenInvalid
	}
	enc := base64.RawURLEncoding
	raw, err := enc.DecodeString(payload)
	if err != nil {
		return UploadClaims{}, ErrTokenInvalid
	}
	got, err := enc.DecodeString(sig)
	if err != nil || !hmac.Equal(got, s.sign(raw)) {
		return UploadClaims{}, ErrTokenInvalid
	}
	var w wireClaims
	if err := json.Unmarshal(raw, &w); err != nil {
		return UploadClaims{}, ErrTokenInvalid
	}
	c := UploadClaims{
		TenantID: w.T, ProjectID: w.P, Key: w.K, MimeType: w.M, FileName: w.F, MaxSize: w.S,
		ExpiresAt: time.Unix(w.E, 0), Nonce: w.N,
	}
	if !s.now().Before(c.ExpiresAt) {
		return UploadClaims{}, ErrTokenExpired
	}
	if c.TenantID != tenantID || c.ProjectID != projectID {
		return UploadClaims{}, ErrTokenScope
	}
	return c, nil
}
