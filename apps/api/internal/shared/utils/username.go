package utils

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
	"unicode"
)

// UsernameBase turns a name or email into a username stem matching
// validator.Username ([a-z0-9_.]{4,32}): "Rina Kirana" -> "rina.kirana",
// "Rina.K@gmail.com" -> "rina.k". Falls back to fallback when too little is
// left (fallback must itself be valid).
func UsernameBase(text, fallback string) string {
	if at := strings.IndexByte(text, '@'); at >= 0 {
		text = text[:at]
	}
	var b strings.Builder
	lastDot := false
	for _, r := range strings.ToLower(text) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '_':
			b.WriteRune(r)
			lastDot = false
		case r == '.' || unicode.IsSpace(r) || r == '-':
			if !lastDot && b.Len() > 0 {
				b.WriteByte('.')
				lastDot = true
			}
		}
	}
	base := strings.Trim(b.String(), "._")
	if len(base) > 24 {
		base = strings.TrimRight(base[:24], "._")
	}
	if len(base) < 4 {
		return fallback
	}
	return base
}

// UsernameWithSuffix appends a random 4-digit suffix ("rina.kirana.4821")
// for when the stem is already taken.
func UsernameWithSuffix(base string) string {
	n, err := rand.Int(rand.Reader, big.NewInt(9000))
	if err != nil {
		return base + ".0000"
	}
	return fmt.Sprintf("%s.%d", base, 1000+n.Int64())
}

// ValidEmail is a light syntax check (one @, a dot in the domain, no
// spaces); real ownership is proven by Google or by the person using it.
func ValidEmail(email string) bool {
	email = strings.TrimSpace(email)
	local, domain, ok := strings.Cut(email, "@")
	return ok && local != "" && strings.Contains(domain, ".") && !strings.ContainsAny(email, " \t\r\n") &&
		!strings.Contains(domain, "@") && len(email) <= 150 && !strings.HasPrefix(domain, ".") && !strings.HasSuffix(domain, ".")
}
