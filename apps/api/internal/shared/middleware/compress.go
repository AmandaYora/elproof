package middleware

import (
	"compress/gzip"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

// gzipMinBytes: below this a response goes out as it is; gzip's own header and the CPU would cost more than they save.
const gzipMinBytes = 1024

// AcceptsEncoding reports whether the request's Accept-Encoding allows coding ("br", "gzip"): named, or covered by "*",
// with a q-value above zero.
func AcceptsEncoding(r *http.Request, coding string) bool {
	star := false
	for _, header := range r.Header.Values("Accept-Encoding") {
		for _, part := range strings.Split(header, ",") {
			name, params, _ := strings.Cut(strings.TrimSpace(part), ";")
			name = strings.ToLower(strings.TrimSpace(name))
			allowed := qAboveZero(params)
			switch name {
			case coding:
				return allowed
			case "*":
				star = allowed
			}
		}
	}
	return star
}

func qAboveZero(params string) bool {
	for _, p := range strings.Split(params, ";") {
		key, value, ok := strings.Cut(strings.TrimSpace(p), "=")
		if !ok || !strings.EqualFold(strings.TrimSpace(key), "q") {
			continue
		}
		q, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
		return err == nil && q > 0
	}
	return true
}

// compressibleType: text the API and the web build answer with. Images, PDFs, spreadsheets and archives are compressed
// already and go out as they are.
func compressibleType(contentType string) bool {
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return false
	}
	if strings.HasPrefix(mediaType, "text/") {
		return true
	}
	switch mediaType {
	case "application/json", "application/problem+json", "application/javascript", "application/xml",
		"application/manifest+json", "image/svg+xml":
		return true
	}
	return false
}

var gzipWriters = sync.Pool{New: func() any {
	w, _ := gzip.NewWriterLevel(io.Discard, gzip.DefaultCompression)
	return w
}}

// Gzip compresses a text response of at least 1 KB for a client that accepts gzip: the API's JSON and CSV, and any web
// file the build has no compressed copy of. A response that already names its Content-Encoding (a precompressed web
// file, see cmd/server/spa.go), a partial or bodiless one, and a HEAD answer pass through untouched.
func Gzip(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead || r.Header.Get("Range") != "" || !AcceptsEncoding(r, "gzip") {
			next.ServeHTTP(w, r)
			return
		}
		gw := &gzipResponseWriter{ResponseWriter: w, status: http.StatusOK}
		defer gw.finish()
		next.ServeHTTP(gw, r)
	})
}

// gzipResponseWriter holds the status and the first bytes until it knows whether the body is worth compressing.
type gzipResponseWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
	decided     bool
	compressing bool
	buf         []byte
	gz          *gzip.Writer
}

func (g *gzipResponseWriter) WriteHeader(status int) {
	if g.wroteHeader {
		return
	}
	// 1xx answers (103 Early Hints) are not the final response.
	if status >= 100 && status < 200 {
		g.ResponseWriter.WriteHeader(status)
		return
	}
	g.wroteHeader = true
	g.status = status
}

func (g *gzipResponseWriter) Write(p []byte) (int, error) {
	if !g.wroteHeader {
		g.WriteHeader(http.StatusOK)
	}
	if g.decided {
		if g.compressing {
			return g.gz.Write(p)
		}
		return g.ResponseWriter.Write(p)
	}
	if !g.eligible() {
		g.decide(false)
		return g.ResponseWriter.Write(p)
	}
	g.buf = append(g.buf, p...)
	if len(g.buf) >= gzipMinBytes {
		if err := g.start(); err != nil {
			return 0, err
		}
	}
	return len(p), nil
}

// eligible: what the handler said so far allows compressing.
func (g *gzipResponseWriter) eligible() bool {
	h := g.Header()
	if h.Get("Content-Encoding") != "" {
		return false
	}
	switch {
	case g.status < 200, g.status == http.StatusNoContent, g.status == http.StatusPartialContent, g.status == http.StatusNotModified:
		return false
	}
	ct := h.Get("Content-Type")
	return ct == "" || compressibleType(ct)
}

// start begins the compressed body with what is buffered.
func (g *gzipResponseWriter) start() error {
	h := g.Header()
	if h.Get("Content-Type") == "" {
		// What net/http would have sniffed from the plain bytes; it cannot sniff compressed ones.
		h.Set("Content-Type", http.DetectContentType(g.buf))
	}
	if !compressibleType(h.Get("Content-Type")) {
		g.decide(false)
		_, err := g.ResponseWriter.Write(g.flushBuf())
		return err
	}
	g.decide(true)
	_, err := g.gz.Write(g.flushBuf())
	return err
}

func (g *gzipResponseWriter) decide(compress bool) {
	g.decided = true
	g.compressing = compress
	h := g.Header()
	if compress {
		h.Set("Content-Encoding", "gzip")
		h.Del("Content-Length")
		h.Del("Accept-Ranges")
		// A strong validator names the plain bytes; the compressed ones differ.
		if etag := h.Get("ETag"); strings.HasPrefix(etag, `"`) {
			h.Set("ETag", "W/"+etag)
		}
		g.gz = gzipWriters.Get().(*gzip.Writer)
		g.gz.Reset(g.ResponseWriter)
	}
	if compress || g.eligibleType() {
		addVary(h, "Accept-Encoding")
	}
	g.ResponseWriter.WriteHeader(g.status)
}

// eligibleType: the answer could have been compressed for another client, so caches must keep them apart.
func (g *gzipResponseWriter) eligibleType() bool {
	ct := g.Header().Get("Content-Type")
	return ct != "" && compressibleType(ct) && g.Header().Get("Content-Encoding") == ""
}

func (g *gzipResponseWriter) flushBuf() []byte {
	b := g.buf
	g.buf = nil
	return b
}

// finish sends what is still held: a short body as it is, or the end of the compressed stream.
func (g *gzipResponseWriter) finish() {
	if !g.decided {
		if !g.wroteHeader && len(g.buf) == 0 {
			// The handler wrote nothing at all: net/http answers 200 with an empty body, as without this wrapper.
			return
		}
		g.decide(false)
		if len(g.buf) > 0 {
			_, _ = g.ResponseWriter.Write(g.flushBuf())
		}
		return
	}
	if g.compressing {
		_ = g.gz.Close()
		g.gz.Reset(io.Discard)
		gzipWriters.Put(g.gz)
		g.gz = nil
	}
}

// Flush sends what is buffered now, compressed when the response qualifies.
func (g *gzipResponseWriter) Flush() {
	if !g.decided {
		if !g.wroteHeader {
			g.WriteHeader(http.StatusOK)
		}
		if g.eligible() && len(g.buf) > 0 {
			_ = g.start()
		} else {
			g.decide(false)
			if len(g.buf) > 0 {
				_, _ = g.ResponseWriter.Write(g.flushBuf())
			}
		}
	}
	if g.compressing {
		_ = g.gz.Flush()
	}
	if f, ok := g.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Unwrap lets http.ResponseController reach the connection underneath.
func (g *gzipResponseWriter) Unwrap() http.ResponseWriter { return g.ResponseWriter }

func addVary(h http.Header, value string) {
	for _, v := range h.Values("Vary") {
		for _, part := range strings.Split(v, ",") {
			if p := strings.TrimSpace(part); p == "*" || strings.EqualFold(p, value) {
				return
			}
		}
	}
	h.Add("Vary", value)
}
