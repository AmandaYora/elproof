package application

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"

	"elproof/internal/modules/identity/domain"
	"elproof/internal/shared/apperror"
)

// MFAEngine is the TOTP implementation (infrastructure.TOTP).
type MFAEngine interface {
	NewSecret() (raw []byte, encoded string, err error)
	URL(account, encodedSecret string) string
	Verify(raw []byte, code string, lastStep int64) (step int64, ok bool)
	Seal(raw []byte) ([]byte, error)
	Open(enc []byte) ([]byte, error)
}

// requiresMFA: two-factor is mandatory for Platform Console admins (they can
// see every tenant and change billing/gateway settings). WO staff and
// clients are out of scope for Fase 1.
func requiresMFA(cred *domain.Credential) bool {
	return cred.PrincipalType == domain.PrincipalPlatformAdmin
}

// MFASetup is what an admin needs to add ElProof to an authenticator app.
type MFASetup struct {
	Secret     string
	OTPAuthURL string
}

// VerifyMFA answers a login challenge with a TOTP code.
func (s *AuthService) VerifyMFA(ctx context.Context, mfaToken, code string) (*Session, error) {
	if s.mfa == nil {
		return nil, apperror.Internal("Verifikasi dua langkah belum dikonfigurasi")
	}
	credentialID, ok := s.challenges.use(mfaToken)
	if !ok {
		return nil, apperror.Unauthorized("Sesi verifikasi sudah berakhir, silakan masuk kembali")
	}
	cred, err := s.credentials.FindByID(ctx, credentialID)
	if err != nil {
		return nil, err
	}
	if cred == nil || !cred.IsActive || !cred.TOTPEnabled {
		return nil, apperror.Unauthorized("Sesi verifikasi sudah berakhir, silakan masuk kembali")
	}
	if err := s.checkCode(ctx, cred, code); err != nil {
		s.challenges.fail(mfaToken)
		return nil, err
	}
	s.challenges.drop(mfaToken)
	return s.issueSession(ctx, cred, true)
}

// SetupMFA starts enrolment for the signed-in admin: a new secret is stored
// (unconfirmed) and returned once so it can be added to an authenticator.
// Re-running it before confirming replaces the secret.
func (s *AuthService) SetupMFA(ctx context.Context, principalType, principalID string) (*MFASetup, error) {
	cred, err := s.mfaCandidate(ctx, principalType, principalID)
	if err != nil {
		return nil, err
	}
	if cred.TOTPEnabled {
		return nil, apperror.Conflict("Verifikasi dua langkah sudah aktif untuk akun ini")
	}
	raw, encoded, err := s.mfa.NewSecret()
	if err != nil {
		return nil, apperror.Internal("Gagal membuat kunci verifikasi")
	}
	enc, err := s.mfa.Seal(raw)
	if err != nil {
		return nil, apperror.Internal("Gagal menyimpan kunci verifikasi")
	}
	if err := s.credentials.SetTOTPSecret(ctx, cred.ID, enc); err != nil {
		return nil, err
	}
	return &MFASetup{Secret: encoded, OTPAuthURL: s.mfa.URL(cred.Username, encoded)}, nil
}

// EnableMFA confirms enrolment with a first code, ends every other session
// of the account, and returns a fresh two-factor session.
func (s *AuthService) EnableMFA(ctx context.Context, principalType, principalID, code string) (*Session, error) {
	cred, err := s.mfaCandidate(ctx, principalType, principalID)
	if err != nil {
		return nil, err
	}
	if cred.TOTPEnabled {
		return nil, apperror.Conflict("Verifikasi dua langkah sudah aktif untuk akun ini")
	}
	if len(cred.TOTPSecretEnc) == 0 {
		return nil, apperror.Validation("Mulai penyiapan verifikasi dua langkah terlebih dahulu", nil)
	}
	raw, err := s.mfa.Open(cred.TOTPSecretEnc)
	if err != nil {
		return nil, apperror.Internal("Kunci verifikasi tidak dapat dibaca, ulangi penyiapan")
	}
	step, ok := s.mfa.Verify(raw, code, cred.TOTPLastStep)
	if !ok {
		return nil, invalidCode()
	}
	if err := s.credentials.EnableTOTP(ctx, cred.ID, step); err != nil {
		return nil, err
	}
	if err := s.refreshTokens.RevokeAllForCredential(ctx, cred.ID); err != nil {
		return nil, err
	}
	cred.TOTPEnabled = true
	return s.issueSession(ctx, cred, true)
}

func (s *AuthService) mfaCandidate(ctx context.Context, principalType, principalID string) (*domain.Credential, error) {
	if s.mfa == nil {
		return nil, apperror.Internal("Verifikasi dua langkah belum dikonfigurasi")
	}
	cred, err := s.credentials.FindByPrincipal(ctx, domain.PrincipalType(principalType), principalID)
	if err != nil {
		return nil, err
	}
	if cred == nil || !cred.IsActive {
		return nil, apperror.Unauthorized("Akun tidak ditemukan")
	}
	if !requiresMFA(cred) {
		return nil, apperror.Forbidden("Verifikasi dua langkah hanya tersedia untuk admin platform")
	}
	return cred, nil
}

func (s *AuthService) checkCode(ctx context.Context, cred *domain.Credential, code string) error {
	raw, err := s.mfa.Open(cred.TOTPSecretEnc)
	if err != nil {
		return apperror.Internal("Kunci verifikasi tidak dapat dibaca; minta Super Admin mereset verifikasi dua langkah")
	}
	step, ok := s.mfa.Verify(raw, code, cred.TOTPLastStep)
	if !ok {
		return invalidCode()
	}
	fresh, err := s.credentials.SetTOTPLastStep(ctx, cred.ID, step)
	if err != nil {
		return err
	}
	if !fresh {
		return invalidCode() // the same code was just used by another login
	}
	return nil
}

func invalidCode() error {
	return apperror.Unauthorized("Kode verifikasi salah atau sudah kedaluwarsa")
}

// --- login challenges -----------------------------------------------------

const (
	mfaChallengeTTL      = 5 * time.Minute
	mfaChallengeAttempts = 5
)

type mfaChallenge struct {
	credentialID int64
	expiresAt    time.Time
	attempts     int
}

// mfaChallenges holds pending "password OK, waiting for code" logins. They
// are opaque random tokens kept in memory (single instance, ADR-0011) — not
// JWTs, so a challenge can never be mistaken for an access token.
type mfaChallenges struct {
	mu    sync.Mutex
	items map[string]*mfaChallenge
}

func newMFAChallenges() *mfaChallenges {
	return &mfaChallenges{items: make(map[string]*mfaChallenge)}
}

func (c *mfaChallenges) issue(credentialID int64) string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	token := hex.EncodeToString(b)
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now()
	for k, v := range c.items { // opportunistic cleanup
		if now.After(v.expiresAt) {
			delete(c.items, k)
		}
	}
	c.items[token] = &mfaChallenge{credentialID: credentialID, expiresAt: now.Add(mfaChallengeTTL)}
	return token
}

func (c *mfaChallenges) use(token string) (int64, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	ch, ok := c.items[token]
	if !ok || time.Now().After(ch.expiresAt) || ch.attempts >= mfaChallengeAttempts {
		delete(c.items, token)
		return 0, false
	}
	return ch.credentialID, true
}

func (c *mfaChallenges) fail(token string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if ch, ok := c.items[token]; ok {
		ch.attempts++
		if ch.attempts >= mfaChallengeAttempts {
			delete(c.items, token)
		}
	}
}

func (c *mfaChallenges) drop(token string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, token)
}
