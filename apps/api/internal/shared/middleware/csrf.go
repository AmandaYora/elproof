package middleware

import (
	"net"
	"net/http"
	"net/url"
	"strings"

	"elproof/internal/shared/response"
)

// CSRFHeader must be present on requests the browser authenticates with the
// session cookie (refresh, logout). A cross-site page can only add a custom
// header through CORS, which this API never grants it, so the header alone
// proves the request came from ElProof's own pages (ADR-0044).
const CSRFHeader = "X-Requested-With"

// SameOrigin refuses state-changing requests that a browser marks as coming
// from another site. It is a technical check (request metadata only), so it
// lives in shared/ next to CORS.
//
//   - Sec-Fetch-Site: cross-site is refused (every current browser sends it).
//   - An Origin header must name this server (its Host, or the Host the
//     reverse proxy received in X-Forwarded-Host); a cross-site form cannot
//     set either header, so trusting them here is safe.
//   - In development any loopback origin is accepted, because the Vite dev
//     server and the API listen on different ports.
//
// Requests without Origin and Sec-Fetch-Site (curl, Postman, server-to-server)
// are not browsers and pass; requireHeader additionally demands CSRFHeader.
type SameOrigin struct {
	DevMode bool
}

// Guard wraps next. requireHeader: also demand CSRFHeader (cookie readers).
func (p SameOrigin) Guard(requireHeader bool, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !p.Allowed(r, requireHeader) {
			response.Error(w, http.StatusForbidden, "Permintaan ditolak karena tidak berasal dari halaman ElProof. Muat ulang halaman lalu coba lagi.",
				map[string]string{"code": "csrf_refused"})
			return
		}
		next(w, r)
	}
}

// Allowed reports whether r passes the policy described on SameOrigin.
func (p SameOrigin) Allowed(r *http.Request, requireHeader bool) bool {
	if requireHeader && strings.TrimSpace(r.Header.Get(CSRFHeader)) == "" {
		return false
	}
	origin := r.Header.Get("Origin")
	if origin != "" {
		if p.DevMode && IsLoopbackOrigin(origin) {
			return true
		}
		if !p.originIsThisServer(r, origin) {
			return false
		}
	}
	return r.Header.Get("Sec-Fetch-Site") != "cross-site"
}

func (p SameOrigin) originIsThisServer(r *http.Request, origin string) bool {
	u, err := url.Parse(origin)
	if err != nil || u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") {
		return false // includes the opaque origin "null"
	}
	if strings.EqualFold(u.Host, r.Host) {
		return true
	}
	if fwd := r.Header.Get("X-Forwarded-Host"); fwd != "" {
		first, _, _ := strings.Cut(fwd, ",")
		if strings.EqualFold(u.Host, strings.TrimSpace(first)) {
			return true
		}
	}
	return false
}

// IsLoopbackOrigin reports whether origin is an http(s) origin on this
// machine: localhost, *.localhost, 127.0.0.0/8 or [::1], any port.
func IsLoopbackOrigin(origin string) bool {
	u, err := url.Parse(origin)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return false
	}
	host := strings.ToLower(u.Hostname())
	if host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
