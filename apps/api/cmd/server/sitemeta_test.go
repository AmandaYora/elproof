package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"elproof/internal/modules/platform"
)

const testIndexHTML = `<!doctype html>
<html lang="en">
  <head>
    <title>ElProof — Client Transparency Portal</title>
    <meta property="og:type" content="website" />
    <meta property="og:site_name" content="ElProof" />
    <meta property="og:title" content="ElProof — Client Transparency Portal" />
    <meta property="og:description" content="Transparansi persiapan pernikahan Anda, dari WO hingga hari H." />
  </head>
  <body>
    <div id="root"></div>
  </body>
</html>
`

func TestInjectTenantSiteMeta_MatchedWithLogo(t *testing.T) {
	out := string(injectTenantSiteMeta([]byte(testIndexHTML), platform.SiteMeta{BusinessName: "JWS Wedding", HasLogo: true}, "journey.jwswedding.com:443"))

	if strings.Contains(out, "ElProof") {
		t.Errorf("expected every ElProof default to be replaced, got:\n%s", out)
	}
	if !strings.Contains(out, `<title>JWS Wedding</title>`) {
		t.Errorf("expected tenant title, got:\n%s", out)
	}
	if !strings.Contains(out, `<meta property="og:site_name" content="JWS Wedding" />`) {
		t.Errorf("expected tenant og:site_name, got:\n%s", out)
	}
	if !strings.Contains(out, `<meta property="og:title" content="JWS Wedding" />`) {
		t.Errorf("expected tenant og:title, got:\n%s", out)
	}
	if !strings.Contains(out, "Pantau progress pernikahan Anda secara real-time bersama JWS Wedding.") {
		t.Errorf("expected tenant og:description, got:\n%s", out)
	}
	// Port must be stripped and host lowercased before landing in the URL.
	if !strings.Contains(out, `<meta property="og:image" content="https://journey.jwswedding.com/api/v1/public/logo" />`) {
		t.Errorf("expected og:image with port stripped, got:\n%s", out)
	}
	if strings.Index(out, "og:image") > strings.Index(out, "</head>") {
		t.Errorf("expected og:image before </head>, got:\n%s", out)
	}
}

func TestInjectTenantSiteMeta_MatchedNoLogo(t *testing.T) {
	out := string(injectTenantSiteMeta([]byte(testIndexHTML), platform.SiteMeta{BusinessName: "Toko Kembang", HasLogo: false}, "example.com"))

	if strings.Contains(out, "og:image") {
		t.Errorf("expected no og:image when HasLogo is false, got:\n%s", out)
	}
	if !strings.Contains(out, `<title>Toko Kembang</title>`) {
		t.Errorf("expected tenant title, got:\n%s", out)
	}
}

// A business name is tenant-supplied and served raw to unauthenticated
// crawlers -- it must never be able to break out of the tag/attribute it's
// spliced into.
func TestInjectTenantSiteMeta_EscapesBusinessName(t *testing.T) {
	malicious := `Toko "Bunga" & <Kembang></title><script>alert(1)</script>`
	out := string(injectTenantSiteMeta([]byte(testIndexHTML), platform.SiteMeta{BusinessName: malicious, HasLogo: false}, "example.com"))

	if strings.Contains(out, "<script>alert(1)</script>") {
		t.Errorf("business name must be HTML-escaped, got:\n%s", out)
	}
	if !strings.Contains(out, "&lt;Kembang&gt;") {
		t.Errorf("expected escaped angle brackets, got:\n%s", out)
	}
	if !strings.Contains(out, "&#34;Bunga&#34;") && !strings.Contains(out, "&quot;Bunga&quot;") {
		t.Errorf("expected escaped quotes, got:\n%s", out)
	}
}

func TestServeIndexWithSiteMeta(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte(testIndexHTML), 0o644); err != nil {
		t.Fatal(err)
	}

	matched := func(ctx context.Context, host string) (platform.SiteMeta, bool) {
		if host == "journey.jwswedding.com" {
			return platform.SiteMeta{BusinessName: "JWS Wedding", HasLogo: true}, true
		}
		return platform.SiteMeta{}, false
	}

	t.Run("matched host rewrites the page", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Host = "journey.jwswedding.com"
		rec := httptest.NewRecorder()
		serveIndexWithSiteMeta(rec, req, dir, matched)
		if !strings.Contains(rec.Body.String(), "JWS Wedding") {
			t.Errorf("expected rewritten body, got:\n%s", rec.Body.String())
		}
	})

	t.Run("unmatched host serves the static file untouched", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Host = "elproof.elcodelabs.com"
		rec := httptest.NewRecorder()
		serveIndexWithSiteMeta(rec, req, dir, matched)
		if rec.Body.String() != testIndexHTML {
			t.Errorf("expected byte-identical default file, got:\n%s", rec.Body.String())
		}
	})
}
