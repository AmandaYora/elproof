package infrastructure

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base32"
	"encoding/binary"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"
)

// TOTP implements RFC 6238 (SHA-1, 30-second steps, 6 digits — what every
// authenticator app supports) plus at-rest encryption of the shared secret.
type TOTP struct {
	issuer string
	aead   cipher.AEAD
	now    func() time.Time
}

const (
	totpStep   = 30
	totpDigits = 6
	// totpSkew accepts one step before/after the current one (±30 s) to
	// tolerate phone clock drift.
	totpSkew = 1
)

var b32 = base32.StdEncoding.WithPadding(base32.NoPadding)

// NewTOTP derives the AES-256 key for secrets from the server's JWT secret
// (domain-separated), so no extra key has to be provisioned. Rotating the JWT
// secret therefore also invalidates enrolled authenticators; admins re-enroll.
func NewTOTP(issuer, masterSecret string) (*TOTP, error) {
	key := sha256.Sum256([]byte("elproof/identity/totp-secret/v1\x00" + masterSecret))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &TOTP{issuer: issuer, aead: aead, now: time.Now}, nil
}

// NewSecret returns a fresh 160-bit secret and its base32 form (what the
// admin types into an authenticator app).
func (t *TOTP) NewSecret() (raw []byte, encoded string, err error) {
	raw = make([]byte, 20)
	if _, err := rand.Read(raw); err != nil {
		return nil, "", err
	}
	return raw, b32.EncodeToString(raw), nil
}

// URL is the otpauth:// URI authenticator apps import.
func (t *TOTP) URL(account, encodedSecret string) string {
	label := url.PathEscape(t.issuer + ":" + account)
	q := url.Values{}
	q.Set("secret", encodedSecret)
	q.Set("issuer", t.issuer)
	q.Set("algorithm", "SHA1")
	q.Set("digits", fmt.Sprint(totpDigits))
	q.Set("period", fmt.Sprint(totpStep))
	return "otpauth://totp/" + label + "?" + q.Encode()
}

// Verify checks code against raw at the current time. It returns the matched
// time step so the caller can store it and reject any step <= lastStep
// (a code can be used only once).
func (t *TOTP) Verify(raw []byte, code string, lastStep int64) (int64, bool) {
	code = strings.ReplaceAll(strings.TrimSpace(code), " ", "")
	if len(code) != totpDigits {
		return 0, false
	}
	current := t.now().Unix() / totpStep
	for d := -int64(totpSkew); d <= totpSkew; d++ {
		step := current + d
		if step <= lastStep {
			continue
		}
		if subtle.ConstantTimeCompare([]byte(hotp(raw, step)), []byte(code)) == 1 {
			return step, true
		}
	}
	return 0, false
}

func hotp(secret []byte, counter int64) string {
	var msg [8]byte
	binary.BigEndian.PutUint64(msg[:], uint64(counter))
	mac := hmac.New(sha1.New, secret)
	mac.Write(msg[:])
	sum := mac.Sum(nil)
	off := sum[len(sum)-1] & 0x0f
	v := (uint32(sum[off])&0x7f)<<24 | uint32(sum[off+1])<<16 | uint32(sum[off+2])<<8 | uint32(sum[off+3])
	return fmt.Sprintf("%0*d", totpDigits, v%1_000_000)
}

// Seal encrypts a secret for storage (nonce || ciphertext).
func (t *TOTP) Seal(raw []byte) ([]byte, error) {
	nonce := make([]byte, t.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	return t.aead.Seal(nonce, nonce, raw, nil), nil
}

// Open decrypts a secret produced by Seal.
func (t *TOTP) Open(enc []byte) ([]byte, error) {
	n := t.aead.NonceSize()
	if len(enc) < n {
		return nil, errors.New("totp: stored secret is malformed")
	}
	return t.aead.Open(nil, enc[:n], enc[n:], nil)
}
