package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	testShell = `<!doctype html><html lang="id"><head><title>shell</title></head><body><div id="root"></div></body></html>`
	testHome  = `<!doctype html><html lang="id" data-surface="marketing"><head><title>Beranda</title></head><body><div id="root" data-prerendered="/">beranda</div></body></html>`
	testFaq   = `<!doctype html><html lang="id" data-surface="marketing"><head><title>FAQ</title></head><body><div id="root" data-prerendered="/faq">faq</div></body></html>`
	// The shape apps/web/scripts/prerender.mjs writes; patterns as the router declares them.
	testManifest = `{
  "pages": { "/": "prerendered/index.html", "/faq": "prerendered/faq.html" },
  "routes": ["/", "/faq", "/dashboard", "/daftar", "/daftar/workspace", "/projects/:projectId", "/projects/:projectId/pembayaran/client",
             "/platform/tenant/:id/:tab?", "/undangan/:token", "/portal/beranda"]
}`
)

// writeWebBuild lays out a small copy of apps/web/dist in a temp dir.
func writeWebBuild(t *testing.T, withManifest bool) string {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"index.html":                testShell,
		"prerendered/index.html":    testHome,
		"prerendered/faq.html":      testFaq,
		"assets/index-abc.js":       "console.log(1)",
		"sitemap.xml":               `<?xml version="1.0" encoding="UTF-8"?><urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"></urlset>`,
		"robots.txt":                "User-agent: *\nAllow: /\n",
		"manifest.webmanifest":      "{}",
		"demo-files/contoh-bukti.p": "x",
	}
	if withManifest {
		files["routes.json"] = testManifest
	}
	for name, body := range files {
		full := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func spaGet(h http.Handler, method, target string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(method, target, nil))
	return rec
}

func TestSPAFileServerPages(t *testing.T) {
	h := spaFileServer(writeWebBuild(t, true))
	cases := []struct {
		name, target string
		status       int
		bodyHas      string
		typeHas      string
	}{
		{"prerendered Beranda", "/", 200, `data-prerendered="/"`, "text/html"},
		{"prerendered page", "/faq", 200, `data-prerendered="/faq"`, "text/html"},
		{"prerendered page, trailing slash", "/faq/", 200, `data-prerendered="/faq"`, "text/html"},
		{"prerendered page, query string", "/faq?utm_source=wa", 200, `data-prerendered="/faq"`, "text/html"},
		{"app route", "/dashboard", 200, "<title>shell</title>", "text/html"},
		{"app route with a parameter", "/projects/42", 200, "<title>shell</title>", "text/html"},
		{"nested app route", "/projects/42/pembayaran/client", 200, "<title>shell</title>", "text/html"},
		{"optional segment left out", "/platform/tenant/7", 200, "<title>shell</title>", "text/html"},
		{"optional segment given", "/platform/tenant/7/langganan", 200, "<title>shell</title>", "text/html"},
		{"token route", "/undangan/abc123", 200, "<title>shell</title>", "text/html"},
		{"public route without prerender", "/daftar/workspace", 200, "<title>shell</title>", "text/html"},
		{"literal segments ignore case, like the router", "/Dashboard", 200, "<title>shell</title>", "text/html"},
		{"unknown path", "/tidak-ada", 404, "<title>shell</title>", "text/html"},
		{"unknown deeper path", "/projects/42/tidak-ada", 404, "<title>shell</title>", "text/html"},
		{"extra segment after a page", "/faq/lainnya", 404, "<title>shell</title>", "text/html"},
		{"missing file", "/favicon.ico", 404, "<title>shell</title>", "text/html"},
		{"directory", "/assets/", 404, "<title>shell</title>", "text/html"},
		{"build bookkeeping is not served", "/routes.json", 404, "<title>shell</title>", "text/html"},
		{"prerendered file only by its public path", "/prerendered/faq.html", 404, "<title>shell</title>", "text/html"},
		{"asset", "/assets/index-abc.js", 200, "console.log(1)", "javascript"},
		{"sitemap", "/sitemap.xml", 200, "<urlset", "xml"},
		{"robots", "/robots.txt", 200, "User-agent", "text/plain"},
		{"unknown API path", "/api/v1/tidak-ada", 404, `"success":false`, "application/json"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := spaGet(h, http.MethodGet, tc.target)
			if rec.Code != tc.status {
				t.Fatalf("GET %s: status %d, want %d", tc.target, rec.Code, tc.status)
			}
			if !strings.Contains(rec.Body.String(), tc.bodyHas) {
				t.Fatalf("GET %s: body %q lacks %q", tc.target, rec.Body.String(), tc.bodyHas)
			}
			if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, tc.typeHas) {
				t.Fatalf("GET %s: Content-Type %q, want it to contain %q", tc.target, ct, tc.typeHas)
			}
		})
	}
}

func TestSPAFileServerHTMLHeaders(t *testing.T) {
	h := spaFileServer(writeWebBuild(t, true))
	for _, target := range []string{"/", "/dashboard", "/tidak-ada"} {
		if cc := spaGet(h, http.MethodGet, target).Header().Get("Cache-Control"); cc != "no-cache" {
			t.Errorf("GET %s: Cache-Control %q, want no-cache", target, cc)
		}
	}
	head := spaGet(h, http.MethodHead, "/tidak-ada")
	if head.Code != http.StatusNotFound || head.Body.Len() != 0 {
		t.Errorf("HEAD /tidak-ada: status %d with %d body bytes, want 404 and none", head.Code, head.Body.Len())
	}
	headPage := spaGet(h, http.MethodHead, "/faq")
	if headPage.Code != http.StatusOK || headPage.Body.Len() != 0 {
		t.Errorf("HEAD /faq: status %d with %d body bytes, want 200 and none", headPage.Code, headPage.Body.Len())
	}
}

// Without routes.json (the Vite dev build, an older image) every unknown path keeps getting the shell with 200, as before.
func TestSPAFileServerWithoutManifest(t *testing.T) {
	h := spaFileServer(writeWebBuild(t, false))
	for _, target := range []string{"/", "/faq", "/tidak-ada", "/dashboard"} {
		rec := spaGet(h, http.MethodGet, target)
		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "<title>shell</title>") {
			t.Errorf("GET %s: status %d body %q, want the shell with 200", target, rec.Code, rec.Body.String())
		}
	}
}

// A broken manifest is ignored (lenient mode), never a reason to answer 404 for the whole app.
func TestSPAFileServerBrokenManifest(t *testing.T) {
	root := writeWebBuild(t, false)
	if err := os.WriteFile(filepath.Join(root, "routes.json"), []byte(`{"pages":`), 0o644); err != nil {
		t.Fatal(err)
	}
	h := spaFileServer(root)
	if rec := spaGet(h, http.MethodGet, "/dashboard"); rec.Code != http.StatusOK {
		t.Errorf("GET /dashboard with a broken routes.json: status %d, want 200", rec.Code)
	}
}

func TestSPAFileServerNoBuild(t *testing.T) {
	h := spaFileServer(t.TempDir())
	if rec := spaGet(h, http.MethodGet, "/dashboard"); rec.Code != http.StatusNotFound {
		t.Errorf("GET /dashboard without a web build: status %d, want 404", rec.Code)
	}
}

func TestMatchSPARoute(t *testing.T) {
	cases := []struct {
		pattern, path string
		want          bool
	}{
		{"/", "/", true},
		{"/", "/x", false},
		{"/faq", "/faq", true},
		{"/faq", "/FAQ", true},
		{"/faq", "/faqs", false},
		{"/projects/:projectId", "/projects/12", true},
		{"/projects/:projectId", "/projects", false},
		{"/projects/:projectId", "/projects/12/ringkasan", false},
		{"/platform/tenant/:id/:tab?", "/platform/tenant/3", true},
		{"/platform/tenant/:id/:tab?", "/platform/tenant/3/profil", true},
		{"/platform/tenant/:id/:tab?", "/platform/tenant/3/profil/x", false},
		{"/platform/tenant/:id/:tab?", "/platform/tenant", false},
		{"/docs/*", "/docs/a/b/c", true},
		{"/docs/*", "/docs", true},
		{"/a/b?/c", "/a/c", true},
		{"/a/b?/c", "/a/b/c", true},
		{"/a/b?/c", "/a/x/c", false},
	}
	for _, tc := range cases {
		if got := matchSPARoute(splitSPAPath(tc.pattern), splitSPAPath(tc.path)); got != tc.want {
			t.Errorf("match(%q, %q) = %v, want %v", tc.pattern, tc.path, got, tc.want)
		}
	}
}

// A build with a 404 shell (prerender.mjs writes dist/prerendered/404.html and routes.json "notFound") answers every
// not-found path with it, so the tab already reads "Halaman tidak ditemukan" before the app loads (QA RT-06).
func TestSPAFileServerNotFoundShell(t *testing.T) {
	root := writeWebBuild(t, true)
	manifest := strings.Replace(testManifest, `"routes":`, `"notFound": "prerendered/404.html", "routes":`, 1)
	notFound := `<!doctype html><html lang="id"><head><title>Halaman tidak ditemukan — ElProof</title><meta name="robots" content="noindex"></head><body><div id="root"></div></body></html>`
	for name, body := range map[string]string{"routes.json": manifest, "prerendered/404.html": notFound} {
		if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(name)), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	h := spaFileServer(root)
	for _, target := range []string{"/halaman-tidak-ada", "/projects/42/tidak-ada", "/routes.json", "/prerendered/faq.html"} {
		rec := spaGet(h, http.MethodGet, target)
		if rec.Code != http.StatusNotFound || !strings.Contains(rec.Body.String(), "<title>Halaman tidak ditemukan — ElProof</title>") {
			t.Errorf("GET %s: %d %q, want 404 with the not-found shell", target, rec.Code, rec.Body.String())
		}
	}
	if rec := spaGet(h, http.MethodGet, "/dashboard"); rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "<title>shell</title>") {
		t.Errorf("an app route still gets the app shell with 200: %d", rec.Code)
	}

	// A manifest naming a file the build lacks falls back to index.html.
	if err := os.Remove(filepath.Join(root, "prerendered", "404.html")); err != nil {
		t.Fatal(err)
	}
	if rec := spaGet(spaFileServer(root), http.MethodGet, "/tidak-ada"); rec.Code != http.StatusNotFound || !strings.Contains(rec.Body.String(), "<title>shell</title>") {
		t.Errorf("missing 404 shell: %d %q, want 404 with index.html", rec.Code, rec.Body.String())
	}
}

// robots.txt names no private path (listing /platform/ would advertise the team's console); the server marks every app
// page and the 404 shell noindex instead, while a public marketing page stays indexable.
func TestSPAFileServerNoindexOutsidePublicPages(t *testing.T) {
	h := spaFileServer(writeWebBuild(t, true))
	for _, target := range []string{"/dashboard", "/halaman-tidak-ada", "/routes.json"} {
		if got := spaGet(h, http.MethodGet, target).Header().Get("X-Robots-Tag"); got != "noindex, nofollow" {
			t.Errorf("GET %s: X-Robots-Tag %q, want noindex, nofollow", target, got)
		}
	}
	for _, target := range []string{"/", "/faq"} {
		if got := spaGet(h, http.MethodGet, target).Header().Get("X-Robots-Tag"); got != "" {
			t.Errorf("GET %s: public page carries X-Robots-Tag %q", target, got)
		}
	}
}
