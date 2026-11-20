package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAcceptsEncoding(t *testing.T) {
	cases := []struct {
		header, coding string
		want           bool
	}{
		{"gzip, deflate, br", "br", true},
		{"gzip, deflate, br", "gzip", true},
		{"gzip;q=0.5, br;q=0", "br", false},
		{"GZIP", "gzip", true},
		{"*", "br", true},
		{"*;q=0", "gzip", false},
		{"br, *;q=0", "br", true},
		{"identity", "gzip", false},
		{"", "gzip", false},
	}
	for _, tc := range cases {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		if tc.header != "" {
			r.Header.Set("Accept-Encoding", tc.header)
		}
		if got := AcceptsEncoding(r, tc.coding); got != tc.want {
			t.Errorf("Accept-Encoding %q, %s: got %v want %v", tc.header, tc.coding, got, tc.want)
		}
	}
}

func serveGzip(h http.Handler, method, accept string, header map[string]string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, "/api/v1/x", nil)
	if accept != "" {
		r.Header.Set("Accept-Encoding", accept)
	}
	for k, v := range header {
		r.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	Gzip(h).ServeHTTP(rec, r)
	return rec
}

func handlerWriting(contentType string, status int, body string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if contentType != "" {
			w.Header().Set("Content-Type", contentType)
		}
		w.WriteHeader(status)
		// Written in pieces, as encoders do.
		for len(body) > 0 {
			n := min(300, len(body))
			_, _ = io.WriteString(w, body[:n])
			body = body[n:]
		}
	})
}

func gunzip(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	zr, err := gzip.NewReader(rec.Body)
	if err != nil {
		t.Fatal(err)
	}
	b, err := io.ReadAll(zr)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestGzipCompressesLargeJSON(t *testing.T) {
	body := `{"success":true,"data":[` + strings.Repeat(`{"name":"Gedung Serbaguna","status":"aktif"},`, 80) + `{}]}`
	rec := serveGzip(handlerWriting("application/json", http.StatusCreated, body), http.MethodGet, "gzip, br", nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status %d, want 201 kept", rec.Code)
	}
	if rec.Header().Get("Content-Encoding") != "gzip" || rec.Header().Get("Content-Length") != "" {
		t.Fatalf("headers %v", rec.Header())
	}
	if !strings.Contains(rec.Header().Get("Vary"), "Accept-Encoding") {
		t.Fatalf("Vary %q", rec.Header().Get("Vary"))
	}
	if rec.Body.Len() >= len(body)/3 {
		t.Fatalf("compressed %d of %d bytes", rec.Body.Len(), len(body))
	}
	if got := gunzip(t, rec); got != body {
		t.Fatalf("round trip differs")
	}
}

// The ETag is the record's version for If-Match (ADR-0036): compressing never rewrites it.
func TestGzipKeepsETag(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("ETag", `"2026-11-20T07:53:06.046340Z"`)
		_, _ = io.WriteString(w, `{"data":"`+strings.Repeat("x", 2000)+`"}`)
	})
	rec := serveGzip(h, http.MethodGet, "gzip", nil)
	if rec.Header().Get("Content-Encoding") != "gzip" {
		t.Fatalf("not compressed: %v", rec.Header())
	}
	if got := rec.Header().Get("ETag"); got != `"2026-11-20T07:53:06.046340Z"` {
		t.Fatalf("ETag %q, want it unchanged", got)
	}
}

func TestGzipLeavesOthersAlone(t *testing.T) {
	big := strings.Repeat("a", 4000)
	cases := []struct {
		name   string
		h      http.Handler
		method string
		accept string
		header map[string]string
	}{
		{"short JSON", handlerWriting("application/json", 200, `{"success":true}`), http.MethodGet, "gzip", nil},
		{"client without gzip", handlerWriting("application/json", 200, big), http.MethodGet, "br", nil},
		{"PDF", handlerWriting("application/pdf", 200, big), http.MethodGet, "gzip", nil},
		{"HEAD", handlerWriting("application/json", 200, big), http.MethodHead, "gzip", nil},
		{"range request", handlerWriting("text/plain", 200, big), http.MethodGet, "gzip", map[string]string{"Range": "bytes=0-10"}},
		{"already encoded", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/javascript")
			w.Header().Set("Content-Encoding", "br")
			_, _ = io.WriteString(w, big)
		}), http.MethodGet, "gzip, br", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := serveGzip(tc.h, tc.method, tc.accept, tc.header)
			if rec.Header().Get("Content-Encoding") == "gzip" {
				t.Fatalf("compressed: %v", rec.Header())
			}
			if tc.method != http.MethodHead && !strings.HasPrefix(big, rec.Body.String()) && rec.Body.String() != `{"success":true}` {
				t.Fatalf("body changed: %q", rec.Body.String()[:min(40, rec.Body.Len())])
			}
		})
	}
}

func TestGzipKeepsStatusWithoutBody(t *testing.T) {
	for _, status := range []int{http.StatusNoContent, http.StatusNotModified, http.StatusNotFound} {
		rec := serveGzip(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(status) }), http.MethodGet, "gzip", nil)
		if rec.Code != status || rec.Body.Len() != 0 || rec.Header().Get("Content-Encoding") != "" {
			t.Errorf("status %d: got %d, %d bytes, encoding %q", status, rec.Code, rec.Body.Len(), rec.Header().Get("Content-Encoding"))
		}
	}
}

func TestGzipSniffsUntypedBody(t *testing.T) {
	body := "<!doctype html><html><body>" + strings.Repeat("<p>halo</p>", 200) + "</body></html>"
	rec := serveGzip(handlerWriting("", 200, body), http.MethodGet, "gzip", nil)
	if rec.Header().Get("Content-Encoding") != "gzip" || !strings.HasPrefix(rec.Header().Get("Content-Type"), "text/html") {
		t.Fatalf("headers %v", rec.Header())
	}
	if gunzip(t, rec) != body {
		t.Fatal("round trip differs")
	}
}

func TestGzipFlushStreams(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/csv")
		_, _ = io.WriteString(w, "a,b\n")
		w.(http.Flusher).Flush()
		_, _ = io.WriteString(w, strings.Repeat("1,2\n", 500))
	})
	rec := serveGzip(h, http.MethodGet, "gzip", nil)
	if rec.Header().Get("Content-Encoding") != "gzip" {
		t.Fatalf("headers %v", rec.Header())
	}
	if got := gunzip(t, rec); got != "a,b\n"+strings.Repeat("1,2\n", 500) {
		t.Fatal("round trip differs")
	}
}
