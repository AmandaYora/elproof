package main

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func spaGetWith(h http.Handler, target, acceptEncoding string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, target, nil)
	if acceptEncoding != "" {
		req.Header.Set("Accept-Encoding", acceptEncoding)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func writeFile(t *testing.T, root, rel string, body []byte) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, body, 0o644); err != nil {
		t.Fatal(err)
	}
}

func gzipped(t *testing.T, s string) []byte {
	t.Helper()
	var b bytes.Buffer
	zw := gzip.NewWriter(&b)
	_, _ = zw.Write([]byte(s))
	_ = zw.Close()
	return b.Bytes()
}

// The precompressed copies precompress.mjs writes go to the browser that accepts them, with the file's own type.
func TestSPAFileServerPrecompressedCopies(t *testing.T) {
	root := writeWebBuild(t, true)
	writeFile(t, root, "assets/index-abc.js.br", []byte("BROTLI-BYTES"))
	writeFile(t, root, "assets/index-abc.js.gz", gzipped(t, "console.log(1)"))
	writeFile(t, root, "prerendered/faq.html.br", []byte("BROTLI-FAQ"))
	h := spaFileServer(root)

	cases := []struct {
		name, target, accept, encoding, body, typeHas string
	}{
		{"brotli first", "/assets/index-abc.js", "gzip, deflate, br", "br", "BROTLI-BYTES", "javascript"},
		{"gzip when brotli is refused", "/assets/index-abc.js", "gzip, br;q=0", "gzip", "", "javascript"},
		{"plain without Accept-Encoding", "/assets/index-abc.js", "", "", "console.log(1)", "javascript"},
		{"prerendered page", "/faq", "br", "br", "BROTLI-FAQ", "text/html"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := spaGetWith(h, tc.target, tc.accept)
			if rec.Code != http.StatusOK {
				t.Fatalf("status %d", rec.Code)
			}
			if got := rec.Header().Get("Content-Encoding"); got != tc.encoding {
				t.Fatalf("Content-Encoding %q, want %q", got, tc.encoding)
			}
			if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, tc.typeHas) {
				t.Fatalf("Content-Type %q, want it to contain %q", ct, tc.typeHas)
			}
			if !strings.Contains(rec.Header().Get("Vary"), "Accept-Encoding") {
				t.Fatalf("Vary %q, want Accept-Encoding", rec.Header().Get("Vary"))
			}
			if tc.encoding == "gzip" {
				zr, err := gzip.NewReader(rec.Body)
				if err != nil {
					t.Fatal(err)
				}
				plain, _ := io.ReadAll(zr)
				if string(plain) != "console.log(1)" {
					t.Fatalf("gzip body decodes to %q", plain)
				}
				return
			}
			if rec.Body.String() != tc.body {
				t.Fatalf("body %q, want %q", rec.Body.String(), tc.body)
			}
		})
	}
}

// A copy older than its file (a rebuild that skipped compressing) is stale: the file itself goes out.
func TestSPAFileServerSkipsStaleCopy(t *testing.T) {
	root := writeWebBuild(t, true)
	writeFile(t, root, "assets/index-abc.js.br", []byte("OLD"))
	old := time.Now().Add(-time.Hour)
	if err := os.Chtimes(filepath.Join(root, "assets", "index-abc.js.br"), old, old); err != nil {
		t.Fatal(err)
	}
	rec := spaGetWith(spaFileServer(root), "/assets/index-abc.js", "br")
	if rec.Header().Get("Content-Encoding") != "" || rec.Body.String() != "console.log(1)" {
		t.Fatalf("stale copy served: encoding %q body %q", rec.Header().Get("Content-Encoding"), rec.Body.String())
	}
}

// Hashed build files are kept for a year; what names them (HTML), the service worker and the manifest are revalidated.
func TestSPAFileServerCacheControl(t *testing.T) {
	root := writeWebBuild(t, true)
	writeFile(t, root, "sw.js", []byte("self.x=1"))
	writeFile(t, root, "icons/icon-192.png", []byte("png"))
	h := spaFileServer(root)
	cases := map[string]string{
		"/assets/index-abc.js":  "public, max-age=31536000, immutable",
		"/sw.js":                "no-cache",
		"/manifest.webmanifest": "no-cache",
		"/robots.txt":           "public, max-age=3600",
		"/icons/icon-192.png":   "public, max-age=3600",
		"/":                     "no-cache",
		"/faq":                  "no-cache",
	}
	for target, want := range cases {
		if got := spaGet(h, http.MethodGet, target).Header().Get("Cache-Control"); got != want {
			t.Errorf("GET %s: Cache-Control %q, want %q", target, got, want)
		}
	}
	if ct := spaGet(h, http.MethodGet, "/manifest.webmanifest").Header().Get("Content-Type"); ct != "application/manifest+json" {
		t.Errorf("manifest Content-Type %q", ct)
	}
}

// A file of a build that is gone answers 404, never the year-long cache header, so the browser asks again next time.
func TestSPAFileServerMissingAssetIsNotCached(t *testing.T) {
	rec := spaGet(spaFileServer(writeWebBuild(t, true)), http.MethodGet, "/assets/index-old.js")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status %d, want 404", rec.Code)
	}
	if cc := rec.Header().Get("Cache-Control"); strings.Contains(cc, "immutable") {
		t.Fatalf("Cache-Control %q on a missing file", cc)
	}
}

// Vite's .vite/manifest.json (and any other dot-folder) is the build's bookkeeping, never served.
func TestSPAFileServerHidesDotFolders(t *testing.T) {
	root := writeWebBuild(t, true)
	writeFile(t, root, ".vite/manifest.json", []byte(`{"src/main.tsx":{}}`))
	rec := spaGet(spaFileServer(root), http.MethodGet, "/.vite/manifest.json")
	if rec.Code != http.StatusNotFound || strings.Contains(rec.Body.String(), "src/main.tsx") {
		t.Fatalf("status %d body %q, want the 404 shell", rec.Code, rec.Body.String())
	}
}
