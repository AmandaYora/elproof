package infrastructure

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
)

// cryptor encrypts/decrypts secrets at rest (gateway credentials, and the
// reversible copy of an external App's secret used to sign outbound webhook
// relays — see MODULE_PAYMENT.md §7.5/§8). The key is derived from a single
// env var (PAYMENT_ENCRYPTION_KEY) — the one secret this module accepts
// living outside the database, because something has to hold the lock to
// its own encryption.
type cryptor struct {
	gcm cipher.AEAD
}

func newCryptor(rawKey string) (*cryptor, error) {
	if rawKey == "" {
		return nil, errors.New("PAYMENT_ENCRYPTION_KEY tidak boleh kosong")
	}
	// SHA-256 of the raw env value gives a stable 32-byte AES-256 key
	// regardless of the operator-supplied string's own length.
	key := sha256.Sum256([]byte(rawKey))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &cryptor{gcm: gcm}, nil
}

// encrypt returns base64(nonce || ciphertext), safe to store in a TEXT column.
func (c *cryptor) encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	nonce := make([]byte, c.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	sealed := c.gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(sealed), nil
}

func (c *cryptor) decrypt(encoded string) (string, error) {
	if encoded == "" {
		return "", nil
	}
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}
	nonceSize := c.gcm.NonceSize()
	if len(raw) < nonceSize {
		return "", errors.New("ciphertext tidak valid")
	}
	nonce, sealed := raw[:nonceSize], raw[nonceSize:]
	plain, err := c.gcm.Open(nil, nonce, sealed, nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}
