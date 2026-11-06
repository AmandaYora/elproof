package application

import (
	"context"
	"time"

	"elproof/internal/modules/identity/domain"
	"elproof/internal/shared/apperror"
)

// GoogleIdentity is a Google account proven by a verified ID token.
type GoogleIdentity struct {
	Sub     string // stable Google account id, never reassigned
	Email   string // verified by Google
	Name    string
	Picture string
	// ExpiresAt is when the ID token stops being accepted.
	ExpiresAt time.Time
}

// GoogleVerifier validates a Google ID token (infrastructure/google_verifier.go).
type GoogleVerifier interface {
	Verify(ctx context.Context, idToken string) (*GoogleIdentity, error)
}

// SetGoogleVerifier plugs in Sign in with Google (identity.module.go).
func (s *AuthService) SetGoogleVerifier(v GoogleVerifier) {
	s.google = v
}

// VerifyGoogle checks an ID token without signing anyone in (used by the
// platform module's workspace signup).
func (s *AuthService) VerifyGoogle(ctx context.Context, idToken string) (*GoogleIdentity, error) {
	if s.google == nil {
		return nil, apperror.Internal("Login dengan Google belum dikonfigurasi")
	}
	return s.google.Verify(ctx, idToken)
}

// GoogleLoginResult is one of: a session; LinkRequired — a staff login
// already uses this verified email and must confirm its ElProof password once
// before Google is linked to it; or SignupRequired — no usable login, the
// caller may offer to create a new workspace (PLAN.md §3 step 2).
type GoogleLoginResult struct {
	Session        *Session
	LinkRequired   bool
	SignupRequired bool
	Identity       *GoogleIdentity
}

// LoginWithGoogle signs a WO Owner/staff member in with their Google
// account. Only a login already linked to this Google account (google_sub)
// signs in directly. A staff login that merely has the same email is never
// linked silently: that email was typed by a WO Owner (anyone can be an
// Owner since self-service signup), so a stranger could otherwise plant a
// staff account with someone else's address and capture their Google
// sign-in. The person proves ownership with the account's ElProof password
// once (LinkGoogleWithPassword); invitations replace this in Fase 3.
//
// Client and Platform Console logins never sign in with Google; a Google
// account whose email only matches one of those simply gets SignupRequired.
func (s *AuthService) LoginWithGoogle(ctx context.Context, idToken string) (*GoogleLoginResult, error) {
	identity, err := s.VerifyGoogle(ctx, idToken)
	if err != nil {
		return nil, err
	}

	cred, err := s.credentials.FindByGoogleSub(ctx, identity.Sub)
	if err != nil {
		return nil, err
	}
	if cred != nil {
		if cred.PrincipalType != domain.PrincipalStaff {
			return nil, apperror.Forbidden("Akun ini tidak dapat masuk dengan Google. Masuk dengan nama pengguna dan kata sandi.")
		}
		if !cred.IsActive {
			return nil, inactiveAccount()
		}
		session, err := s.issueSession(ctx, cred, false)
		if err != nil {
			return nil, err
		}
		return &GoogleLoginResult{Session: session}, nil
	}

	staff, err := s.staffByEmail(ctx, identity.Email)
	if err != nil {
		return nil, err
	}
	switch {
	case staff == nil:
		return &GoogleLoginResult{SignupRequired: true, Identity: identity}, nil
	case staff.GoogleSub != "":
		// Linked to another Google account with the same address.
		return nil, apperror.Conflict("Email ini sudah terhubung dengan akun Google lain. Hubungi Owner WO Anda.")
	case !staff.IsActive:
		return nil, inactiveAccount()
	case staff.PasswordHash == "":
		return &GoogleLoginResult{SignupRequired: true, Identity: identity}, nil
	}
	return &GoogleLoginResult{LinkRequired: true, Identity: identity}, nil
}

// LinkGoogleWithPassword links a Google account to the staff login that has
// the same email, after that login's ElProof password is confirmed, then
// signs in. Wrong passwords count against the account's failure budget.
func (s *AuthService) LinkGoogleWithPassword(ctx context.Context, idToken, password string) (*Session, error) {
	identity, err := s.VerifyGoogle(ctx, idToken)
	if err != nil {
		return nil, err
	}
	if linked, err := s.credentials.FindByGoogleSub(ctx, identity.Sub); err != nil {
		return nil, err
	} else if linked != nil {
		return nil, apperror.Conflict("Akun Google ini sudah terhubung. Silakan masuk dengan Google.")
	}
	staff, err := s.staffByEmail(ctx, identity.Email)
	if err != nil {
		return nil, err
	}
	wrong := apperror.Unauthorized("Kata sandi salah")
	if staff == nil || staff.GoogleSub != "" || !staff.IsActive {
		return nil, wrong
	}
	ok, err := s.checkPassword(staff, password)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, wrong
	}
	linked, err := s.credentials.LinkGoogle(ctx, staff.ID, identity.Sub)
	if err != nil {
		return nil, err
	}
	if !linked {
		return nil, apperror.Conflict("Akun ini baru saja dihubungkan ke akun Google lain. Hubungi Owner WO Anda.")
	}
	return s.issueSession(ctx, staff, false)
}

func inactiveAccount() error {
	return apperror.Unauthorized("Akun Anda sudah dinonaktifkan. Hubungi Owner WO Anda.").WithCode("account_inactive")
}

// staffByEmail returns the staff login using this email, if any (emails are
// unique across logins, migration 000016).
func (s *AuthService) staffByEmail(ctx context.Context, email string) (*domain.Credential, error) {
	candidates, err := s.credentials.FindAllByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	for _, c := range candidates {
		if c.PrincipalType == domain.PrincipalStaff {
			return c, nil
		}
	}
	return nil, nil
}
