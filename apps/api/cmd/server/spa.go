package main

import (
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"elproof/internal/shared/response"
)

// spaManifest is dist/routes.json, written by apps/web/scripts/prerender.mjs at build time (ADR-0043).
type spaManifest struct {
	// Pages maps a public path ("/", "/faq") to its prerendered HTML file, relative to the web root.
	Pages map[string]string `json:"pages"`
	// Routes lists every path pattern of the React router ("/projects/:projectId/ringkasan", "/platform/tenant/:id/:tab?").
	Routes []string `json:"routes"`
	// NotFound is the app shell titled "Halaman tidak ditemukan" (dist/prerendered/404.html), relative to the web root.
	NotFound string `json:"notFound"`
}

// spaHandler serves the built web app: prerendered public pages, static files, the app shell for the router's own paths,
// and the same shell with 404 for any other path, so the not-found page shows and crawlers see a real 404.
type spaHandler struct {
	root   string
	files  http.Handler
	pages  map[string]string
	routes [][]string
	// strict is set when routes.json was read: only then is a path outside the route list a 404. A build without it (the
	// Vite dev build, an older image) keeps the old behaviour, the shell with 200 for every path.
	strict bool
	// notFound is the HTML answered with 404 (index.html when the build has no 404 shell).
	notFound string
}

// spaFileServer serves the web build in root (see spaHandler). Registered on "/", it would otherwise also swallow any
// unmatched /api/v1/... request into an HTML page, so those get the JSON 404 before anything else.
func spaFileServer(root string) http.Handler {
	h := &spaHandler{root: root, files: http.FileServer(http.Dir(root)), pages: map[string]string{}, notFound: "index.html"}
	m, err := loadSPAManifest(root)
	switch {
	case err == nil:
		for p, file := range m.Pages {
			h.pages[normalizeSPAPath(p)] = file
		}
		for _, pattern := range m.Routes {
			h.routes = append(h.routes, splitSPAPath(pattern))
		}
		h.strict = true
		if m.NotFound != "" {
			if info, err := os.Stat(filepath.Join(root, filepath.FromSlash(m.NotFound))); err == nil && !info.IsDir() {
				h.notFound = m.NotFound
			}
		}
	case errors.Is(err, fs.ErrNotExist):
		// No manifest: lenient mode.
	default:
		log.Printf("web: %s ignored, every unknown path answers 200: %v", filepath.Join(root, "routes.json"), err)
	}
	return h
}

func loadSPAManifest(root string) (spaManifest, error) {
	var m spaManifest
	raw, err := os.ReadFile(filepath.Join(root, "routes.json"))
	if err != nil {
		return m, err
	}
	if err := json.Unmarshal(raw, &m); err != nil {
		return m, err
	}
	if len(m.Routes) == 0 {
		return m, errors.New("no routes")
	}
	return m, nil
}

func (h *spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/api/") {
		response.Error(w, http.StatusNotFound, "Endpoint tidak ditemukan", nil)
		return
	}
	clean := normalizeSPAPath(r.URL.Path)

	// The build's own bookkeeping is never a page: a prerendered file is reached by its public path only.
	if clean == "/routes.json" || clean == "/prerendered" || strings.HasPrefix(clean, "/prerendered/") {
		h.serveHTML(w, r, h.notFound, http.StatusNotFound)
		return
	}
	if file, ok := h.pages[clean]; ok {
		h.serveHTML(w, r, file, http.StatusOK)
		return
	}
	if info, err := os.Stat(filepath.Join(h.root, filepath.FromSlash(clean))); err == nil && !info.IsDir() {
		h.files.ServeHTTP(w, r)
		return
	}
	if !h.strict || h.isAppRoute(clean) {
		h.serveHTML(w, r, "index.html", http.StatusOK)
		return
	}
	h.serveHTML(w, r, h.notFound, http.StatusNotFound)
}

// serveHTML writes one HTML file of the build with status. A 200 goes through http.ServeContent (HEAD, If-Modified-Since);
// a 404 is written as is. HTML is always revalidated: it names the hashed asset files of the build it came from.
func (h *spaHandler) serveHTML(w http.ResponseWriter, r *http.Request, rel string, status int) {
	f, err := os.Open(filepath.Join(h.root, filepath.FromSlash(rel)))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || info.IsDir() {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	if status == http.StatusOK {
		http.ServeContent(w, r, rel, info.ModTime(), f)
		return
	}
	w.WriteHeader(status)
	if r.Method != http.MethodHead {
		_, _ = io.Copy(w, f)
	}
}

// isAppRoute reports whether the React router has a route for p (its own matching: literal segments ignore case, ":x"
// is one segment, a trailing "?" makes a segment optional, "*" takes the rest).
func (h *spaHandler) isAppRoute(p string) bool {
	segs := splitSPAPath(p)
	for _, pattern := range h.routes {
		if matchSPARoute(pattern, segs) {
			return true
		}
	}
	return false
}

func matchSPARoute(pattern, segs []string) bool {
	if len(pattern) == 0 {
		return len(segs) == 0
	}
	p := pattern[0]
	if p == "*" {
		return true
	}
	optional := strings.HasSuffix(p, "?")
	name := strings.TrimSuffix(p, "?")
	if optional && matchSPARoute(pattern[1:], segs) {
		return true
	}
	if len(segs) == 0 {
		return false
	}
	if !strings.HasPrefix(name, ":") && !strings.EqualFold(name, segs[0]) {
		return false
	}
	return matchSPARoute(pattern[1:], segs[1:])
}

// normalizeSPAPath cleans a request path the way the router reads it: "/faq/" and "/faq" are one page.
func normalizeSPAPath(p string) string {
	if p == "" {
		return "/"
	}
	return path.Clean("/" + p)
}

func splitSPAPath(p string) []string {
	var out []string
	for _, s := range strings.Split(p, "/") {
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}
