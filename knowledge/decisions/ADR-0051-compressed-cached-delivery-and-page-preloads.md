# ADR-0051: Compressed, cached delivery and page preloads

## Status
Accepted — implemented 2026-11-20. Extends ADR-0043 (prerendered public pages) and ADR-0037 (entry chunk budget).

## Context
A Lighthouse run of the live Beranda (mobile, throttled) showed that the web build reached the browser as built: the
entry script went out as 472 KB and the style sheet as 129 KB, with no `Content-Encoding` and no `Cache-Control`.
Only HTML was compressed, by the host nginx's default `gzip` (which covers `text/html` alone), and the API's JSON was not
compressed at all. `npm run budget` measured gzip sizes that no browser ever received. Every visit downloaded every
file again, although each file under `assets/` is named after its content. The page also loaded its parts one after
another: HTML → entry script → the page's lazy chunks → the price list from `/api/v1/public/plans`.

These are properties of the application (the Go binary serves the build, ADR-0043), so they are fixed in the repository,
not in the server's configuration.

## Decision

### Compressed copies written at build time
`npm run prerender` (and so `npm run build`) ends with `scripts/precompress.mjs`. For each text file of `dist/` of 1 KB or
more (`.js`, `.css`, `.html`, `.svg`, `.json`, `.xml`, `.txt`, `.webmanifest`), it writes a Brotli copy (`.br`, quality
11) and a gzip copy (`.gz`, level 9) beside the file. A copy is kept only when it saves at least a tenth. The script
skips `routes.json` and dot-folders, and drops the copies of an earlier run first.

`cmd/server/spa.go` sends the copy the request's `Accept-Encoding` allows, preferring `.br` over `.gz`. It sends
`Content-Encoding`, the file's own `Content-Type` (from the extension, never sniffed from compressed bytes) and
`Vary: Accept-Encoding`. A copy older than its file is ignored, so a rebuild that skipped compressing never serves stale
bytes.

### Cache lifetimes
| What | `Cache-Control` |
|---|---|
| `assets/*` (named after their content) | `public, max-age=31536000, immutable` |
| HTML (shell, prerendered pages, 404), `sw.js`, `manifest.webmanifest` | `no-cache` |
| Anything else in `public/` (icons, robots.txt, sitemap) | `public, max-age=3600` |

A new build gives every changed file a new name. The HTML is always revalidated, so the browser learns the new names on
its next visit, and old files simply stop being asked for. A path under `assets/` that the build does not have gets the
404 shell (`no-cache`), never the year-long header. Dot-folders (`dist/.vite/`) are never served.

### Gzip for API answers
`middleware.Gzip` wraps the whole mux. It compresses a text answer of 1 KB or more (JSON, CSV, `text/*`, SVG,
JavaScript, XML) for a client that accepts gzip. It does nothing for:
* HEAD requests and `Range` requests;
* answers with status 204, 206 or 304;
* answers that already set `Content-Encoding` (the precompressed web files);
* images, PDFs and spreadsheets.

It removes `Content-Length`, turns a strong `ETag` weak, adds `Vary`, supports `Flush`, and reuses its gzip writers from a
pool.

### A page names what it needs next
Each prerendered page's `<head>` (`scripts/prerender.mjs`) gains:
* **`modulepreload` for every lazy chunk the page loads while it renders**, plus the chunks those import. The SSR build
  records each dynamic import's source file (`recordPrerenderImports` in `vite.config.ts`). Each page renders in its own
  worker (`scripts/prerender-imports.mjs`), because a lazy module loads only once per module graph. The source files
  are looked up in `dist/.vite/chunks.json`, written by the browser build (`writeChunkMap`; Vite's own manifest leaves
  out a lazy page that Rollup merged into a shared chunk). A lazy chunk's own style sheet is linked as a stylesheet.
* **`<link rel="preload" as="fetch" crossorigin="use-credentials">`** for each API path in `PublicPageMeta.prefetch`.
  Beranda prefetches `API.public.plans`. The credentials mode matches the API client (`withCredentials`), so the
  browser reuses the preloaded answer instead of asking twice.

Every page built from the shell, including the app shell `dist/index.html`, preloads the latin subset of Plus Jakarta Sans
(`preloadTextFont`).

### An open tab across a deploy
A tab opened before a deploy still names the previous build's chunks, which the new container does not have. The first
screen that tab opens and has not loaded yet then fails with `vite:preloadError`. `shared/lib/stale-build.ts` (installed
by `main.tsx`) reloads the page once, so the tab comes back on the current build. A second failure within a minute (no
connection, a real outage), or a browser without `sessionStorage`, is left to the error screen, so the page never
reloads in a loop.

## Consequences
* Measured on the build: the entry script goes from 472 KB to 133 KB (Brotli) and the style sheet from 129 KB to 19 KB.
  All text in the build goes from 2.9 MB to 0.8 MB. On a repeat visit, assets come from the browser's cache.
* Beranda's lazy chunks (≈32 KB Brotli), its font and the price list download alongside the entry script instead of
  after it.
* Host nginx needs no gzip settings for the app: it passes the encoded answers through. HTTP/2 at nginx remains a
  separate, infrastructure choice.
* `npm run budget` still measures gzip; what is served is the smaller Brotli copy where the browser accepts it.
