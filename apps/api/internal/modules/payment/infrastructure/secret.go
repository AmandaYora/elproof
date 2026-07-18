package infrastructure

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"

	"golang.org/x/crypto/bcrypt"
)

// generateAppID mints a new external App's client_id — not a secret itself
// (fine to display/log), just needs to be unique and URL-safe.
func generateAppID() (string, error) {
	buf := make([]byte, 6)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return "app_" + hex.EncodeToString(buf), nil
}

// generateSecret mints a new plaintext App secret — shown to the caller
// exactly once (see knowledge/MODULE_PAYMENT.md §7.5); only its bcrypt hash
// and an AES-GCM-encrypted copy (via cryptor) are ever persisted.
func generateSecret() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func hashSecret(secret string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(secret), bcrypt.DefaultCost)
	return string(hash), err
}

func compareSecret(hash, secret string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(secret))
}
