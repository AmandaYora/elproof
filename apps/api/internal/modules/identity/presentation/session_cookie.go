package presentation

import (
	"net/http"
	"time"
)

// SessionCookiePath scopes the refresh-token cookie to the auth endpoints:
// the browser never sends it with ordinary API calls or page loads.
const SessionCookiePath = "/api/v1/auth"

// SessionCookie describes the refresh-token cookie (ADR-0042). It is
// HttpOnly (page scripts cannot read it), SameSite=Strict (never sent on a
// request started by another site) and Secure everywhere except local
// development over plain http.
type SessionCookie struct {
	Secure bool
	MaxAge time.Duration
}

// Name is the cookie's name. Over https it carries the __Secure- prefix, so
// a browser refuses it from an insecure page (cookie injection).
func (c SessionCookie) Name() string {
	if c.Secure {
		return "__Secure-elproof_refresh"
	}
	return "elproof_refresh"
}

func (c SessionCookie) base(value string) *http.Cookie {
	return &http.Cookie{
		Name:     c.Name(),
		Value:    value,
		Path:     SessionCookiePath,
		HttpOnly: true,
		Secure:   c.Secure,
		SameSite: http.SameSiteStrictMode,
	}
}

// Set stores token in the browser for MaxAge (the refresh token's lifetime).
func (c SessionCookie) Set(w http.ResponseWriter, token string) {
	cookie := c.base(token)
	cookie.MaxAge = int(c.MaxAge / time.Second)
	cookie.Expires = time.Now().Add(c.MaxAge)
	http.SetCookie(w, cookie)
	w.Header().Set("Cache-Control", "no-store")
}

// Clear removes the cookie from the browser.
func (c SessionCookie) Clear(w http.ResponseWriter) {
	cookie := c.base("")
	cookie.MaxAge = -1
	cookie.Expires = time.Unix(0, 0)
	http.SetCookie(w, cookie)
	w.Header().Set("Cache-Control", "no-store")
}

// Read returns the refresh token the browser sent, or "".
func (c SessionCookie) Read(r *http.Request) string {
	cookie, err := r.Cookie(c.Name())
	if err != nil {
		return ""
	}
	return cookie.Value
}
