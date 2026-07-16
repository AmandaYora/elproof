package infrastructure

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"elproof/internal/modules/identity/domain"
	"elproof/internal/shared/middleware"
)

type JWTIssuer struct {
	secret []byte
}

func NewJWTIssuer(secret string) *JWTIssuer {
	return &JWTIssuer{secret: []byte(secret)}
}

// IssueAccessToken signs a JWT carrying the claims shape shared/middleware
// verifies on every subsequent request (principal_type, principal_id, tenant_id,
// role) — see ADR-0005.
func (i *JWTIssuer) IssueAccessToken(cred *domain.Credential, ttl time.Duration) (string, error) {
	var tenantID string
	if cred.TenantID != nil {
		tenantID = strconv.FormatInt(*cred.TenantID, 10)
	}

	claims := middleware.Claims{
		PrincipalType: string(cred.PrincipalType),
		PrincipalID:   cred.PrincipalID,
		TenantID:      tenantID,
		Role:          cred.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   cred.Username,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(i.secret)
}

// NewRefreshTokenValue mints a random opaque refresh token and returns both the
// plain value (sent to the client) and its sha256 hash (stored server-side) —
// the server never stores the plain refresh token.
func (i *JWTIssuer) NewRefreshTokenValue() (plain string, hash string, err error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", "", err
	}
	plain = base64.RawURLEncoding.EncodeToString(buf)
	return plain, i.HashToken(plain), nil
}

func (i *JWTIssuer) HashToken(plain string) string {
	sum := sha256.Sum256([]byte(plain))
	return hex.EncodeToString(sum[:])
}

