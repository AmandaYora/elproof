package main

import (
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"log"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"elproof/internal/shared/middleware"
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

	// The build's own bookkeeping is never a page: a prerendered file is reached by its public path only, and a dot-folder
	// (Vite's .vite/manifest.json) is never served.
	if clean == "/routes.json" || clean == "/prerendered" || strings.HasPrefix(clean, "/prerendered/") || hasDotSegment(clean) {
		w.Header().Set("X-Robots-Tag", "noindex, nofollow")
		h.serveHTML(w, r, h.notFound, http.StatusNotFound)
		return
	}
	if file, ok := h.pages[clean]; ok {
		h.serveHTML(w, r, file, http.StatusOK)
		return
	}
	if info, err := os.Stat(filepath.Join(h.root, filepath.FromSlash(clean))); err == nil && !info.IsDir() {
		if strings.HasSuffix(clean, "/index.html") {
			// http.FileServer sends /index.html to "/" (and so to the page that path answers).
			h.files.ServeHTTP(w, r)
			return
		}
		h.serveFile(w, r, strings.TrimPrefix(clean, "/"))
		return
	}
	// The signed-in app is private. robots.txt does not list its paths (that would advertise them, the platform team's
	// pages included), so every app page and the 404 shell tell crawlers themselves.
	w.Header().Set("X-Robots-Tag", "noindex, nofollow")
	if !h.strict || h.isAppRoute(clean) {
		h.serveHTML(w, r, "index.html", http.StatusOK)
		return
	}
	h.serveHTML(w, r, h.notFound, http.StatusNotFound)
}

// serveHTML writes one HTML file of the build with status. A 200 goes through http.ServeContent (HEAD, If-Modified-Since);
// a 404 is written as is. HTML is always revalidated: it names the hashed asset files of the build it came from.
func (h *spaHandler) serveHTML(w http.ResponseWriter, r *http.Request, rel string, status int) {
	f, info, ok := h.openBuildFile(w, r, rel, "text/html; charset=utf-8")
	if !ok {
		http.NotFound(w, r)
		return
	}
	defer f.Close()
	w.Header().Set("Cache-Control", "no-cache")
	if status == http.StatusOK {
		http.ServeContent(w, r, rel, info.ModTime(), f)
		return
	}
	w.Header().Set("Content-Length", strconvSize(info.Size()))
	w.WriteHeader(status)
	if r.Method != http.MethodHead {
		_, _ = io.Copy(w, f)
	}
}

// serveFile writes one static file of the build (a script, a style sheet, an icon) with how long a browser may keep it.
func (h *spaHandler) serveFile(w http.ResponseWriter, r *http.Request, rel string) {
	f, info, ok := h.openBuildFile(w, r, rel, contentTypeOf(rel))
	if !ok {
		http.NotFound(w, r)
		return
	}
	defer f.Close()
	w.Header().Set("Cache-Control", cacheControlOf(rel))
	http.ServeContent(w, r, rel, info.ModTime(), f)
}

// openBuildFile opens rel, or the compressed copy of it the browser accepts (.br before .gz, written beside it by
// apps/web/scripts/precompress.mjs), and sets Content-Type, Content-Encoding and Vary to match. A copy older than the
// file (a rebuild that skipped compressing) is left alone, and so is any copy of a file of unknown type.
func (h *spaHandler) openBuildFile(w http.ResponseWriter, r *http.Request, rel, contentType string) (*os.File, fs.FileInfo, bool) {
	full := filepath.Join(h.root, filepath.FromSlash(rel))
	plain, err := os.Stat(full)
	if err != nil || plain.IsDir() {
		return nil, nil, false
	}
	if contentType != "" {
		w.Header().Set("Content-Type", contentType)
		hasCopy := false
		for _, c := range [...]struct{ coding, ext string }{{"br", ".br"}, {"gzip", ".gz"}} {
			info, err := os.Stat(full + c.ext)
			if err != nil || info.IsDir() || info.ModTime().Before(plain.ModTime()) {
				continue
			}
			hasCopy = true
			if !middleware.AcceptsEncoding(r, c.coding) {
				continue
			}
			f, err := os.Open(full + c.ext)
			if err != nil {
				continue
			}
			w.Header().Set("Content-Encoding", c.coding)
			w.Header().Add("Vary", "Accept-Encoding")
			return f, info, true
		}
		if hasCopy {
			w.Header().Add("Vary", "Accept-Encoding")
		}
	}
	f, err := os.Open(full)
	if err != nil {
		return nil, nil, false
	}
	return f, plain, true
}

// cacheControlOf: a file under assets/ is named after its content (index-Q3N4mQAD.js), so a browser keeps it for a year
// and a new build simply names new files. The service worker, the web manifest and HTML are asked for again every time;
// the rest (icons, robots.txt, the sitemap) for an hour.
func cacheControlOf(rel string) string {
	switch {
	case strings.HasPrefix(rel, "assets/"):
		return "public, max-age=31536000, immutable"
	case rel == "sw.js", strings.HasSuffix(rel, ".html"), strings.HasSuffix(rel, ".webmanifest"):
		return "no-cache"
	default:
		return "public, max-age=3600"
	}
}

// contentTypeOf is the type of rel by its extension, set before any compressed copy is chosen (http.ServeContent would
// otherwise read it from the compressed bytes). Empty when unknown: the file is then sent plain and sniffed.
func contentTypeOf(rel string) string {
	if strings.HasSuffix(rel, ".webmanifest") {
		return "application/manifest+json"
	}
	return mime.TypeByExtension(path.Ext(rel))
}

func strconvSize(n int64) string {
	return strconv.FormatInt(n, 10)
}

// hasDotSegment reports whether any segment of p starts with a dot.
func hasDotSegment(p string) bool {
	for _, s := range strings.Split(p, "/") {
		if strings.HasPrefix(s, ".") {
			return true
		}
	}
	return false
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
