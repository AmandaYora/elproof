# Deployment — ElProof

## 1. Prerequisites

- Node.js 20+, Go 1.25+
- A MySQL 8 server reachable from your machine (this project never runs the
  database inside Docker — see `.claude/rules/monorepo.md`)
- [`golang-migrate`](https://github.com/golang-migrate/migrate) CLI on your `PATH`
  (only needed for `npm run migrate:*`)

## 2. Clone and configure

```bash
git clone https://github.com/AmandaYora/elproof.git
cd elproof
cp .env.example .env
cp apps/web/.env.example apps/web/.env
npm install
```

Edit `.env` with your real MySQL credentials (`DB_HOST`, `DB_USER`,
`DB_PASSWORD`, `DB_NAME`, and the matching `DATABASE_URL`) and a real
`JWT_SECRET`. `apps/web/.env` only needs to change if the API doesn't run on
`http://localhost:8080`.

## 3. Database

Create the database, then run migrations and (optionally) seed demo data:

```bash
mysql -u root -e "CREATE DATABASE IF NOT EXISTS elproof_db"
npm run migrate:up
cd apps/api && go run ./cmd/seed && cd ../..
```

`cmd/seed` is safe to re-run — it truncates and reseeds every table it owns,
leaving a deliberately minimal clean slate: one Platform Console super admin
(`superadmin` / `superadmin`) and one subscription plan ("Paket 1 Tahun",
12 bulan, Rp 2.000.000). No tenant/staff/client exists until one is created
through the app itself (Platform Console → Tenant → Tambah Tenant, which in
turn provisions that tenant's own Owner login).

## 4. Run locally (two processes, from the repo root)

```bash
npm run dev:api   # Go API on :8080 (Air, hot reload)
npm run dev:web   # Vite dev server on :5173, proxies API calls to :8080
```

Open `http://localhost:5173`.

## 5. Docker (one app container; database stays on the host)

The image is a multi-stage build: it compiles the frontend to static assets
and the API to a single Go binary, then serves both from one Alpine
container — the Go server hosts `/api/v1/*` and falls back to the built
frontend (with an SPA fallback to `index.html`) for everything else, so
there's no separate frontend server or reverse proxy to configure.

```bash
docker compose up --build
```

This reads `.env` for JWT/S3 config, but **overrides `DATABASE_URL`** in
`docker-compose.yml` to point at `host.docker.internal` instead of
`localhost` — inside the container, `localhost` would resolve to the
container itself, not your host machine, so a plain copy of `.env`'s
`DATABASE_URL` would never reach a host-run MySQL. The override reuses
`.env`'s `DB_USER`/`DB_PASSWORD`/`DB_PORT`/`DB_NAME` values, so as long as
those are correct for your host MySQL, nothing else needs to change.

Before the first `docker compose up`, make sure the database already exists
and migrations have been applied (step 3) — the container does not run
migrations itself.

Once it's up, the app is served at `http://localhost:8080` (frontend and API
on the same origin, no CORS involved).

### Known limitation of this document

This Docker path (`docker compose up --build` reaching a host MySQL,
end-to-end) has been reasoned through and the compose/Dockerfile config
fixed accordingly, but **has not been executed in this environment** — Docker
is not installed here. If `docker compose up` surfaces an issue not covered
above, treat this section as a strong starting point rather than a
guarantee, and update it once verified.

## 6. Production deployment (VPS, image-only — different from §5)

See ADR-0011 for the full decision record. The target VPS ("Elcodelabs", also hosting the
`elkasir` app) follows a fixed, shared server convention — canonical copy on the VPS itself at
`/opt/elcodelabs/SERVER_PLAYBOOK.md` — that is **stricter than §5 above**: the VPS has only
2 GB RAM and **never builds anything**. Do not run `docker compose up --build` there.

**The pipeline:**
1. **CI builds.** `.github/workflows/deploy.yml` builds `infra/docker/Dockerfile` and pushes
   `ghcr.io/amandayora/elproof:<git-sha>` (+ `:latest`) on every push to `main`.
2. **The VPS only pulls and runs**, from `~/elproof/` (image-only
   `docker-compose.prod.yml`, no `build:` key), via `~/elproof/deploy.sh <git-sha>`:
   pull → run migrations → `compose up -d` → poll `/api/v1/health` until ready → prune old images.
3. **One binary does everything the image needs** — no separate migrate CLI, seed binary, or
   curl/wget baked into the container. `cmd/server`'s compiled binary (`./api`) dispatches on an
   optional first argument:
   - `./api` (no args) — serve, exactly like local dev.
   - `./api migrate up` / `./api migrate down` — apply/roll back one step, using the same
     `apps/api/migrations/*.sql` files (now embedded in the binary via `go:embed`, see
     `apps/api/internal/migrator`) that `npm run migrate:up`/`migrate:down` drive locally through
     the external `golang-migrate` CLI. Both paths share one `schema_migrations` table format, so
     they're interchangeable — the CLI (§1 prerequisite) stays a local-dev convenience only, never
     required in production.
   - `./api seed` — the same minimal reseed as `go run ./cmd/seed` (§3), shared via
     `apps/api/internal/adminseed`.
   - `./api healthcheck` — self-GETs `/api/v1/health`, exit 0/1; this is what
     `docker-compose.prod.yml`'s `HEALTHCHECK` runs, so the Alpine image needs no extra HTTP client
     installed.
4. **Per-app isolation, shared MySQL:** the VPS's single host-level MySQL instance hosts one
   database + one dedicated user per app — `elproof_db` / `elproof_user@'172.%'` — never a
   cross-app user or a Dockerized database. Secrets (`DB_PASSWORD`, `JWT_SECRET`,
   `PAYMENT_ENCRYPTION_KEY`, S3 keys) live only in `~/elproof/.env` on the VPS (`chmod 600`),
   generated fresh for this environment — never copied from local dev's `.env`.
5. **nginx + TLS** front the container's `127.0.0.1:8082` once a domain is assigned (not done yet
   — see status below).

**Status as of 2026-07-17 — infra set up, nothing deployed yet:**
- Done: `elproof_db` + `elproof_user` created; `~/elproof/.env`, `docker-compose.prod.yml`,
  `deploy.sh` written on the VPS; `.github/workflows/deploy.yml` added to this repo; VPS's own
  `SERVER_PLAYBOOK.md` App registry updated with elproof's row (port `8082`).
- Not done: no CI run yet (no image in GHCR yet), so no container has ever been started on the
  VPS; no domain assigned; no nginx site/TLS. `elkasir` currently holds nginx's one allowed
  `default_server` on port 80 — elproof's future nginx config must set its own `server_name` and
  must **not** add a second `default_server` (see the playbook's caveat).
- Next steps once a domain is chosen: push to `main` (or run the workflow manually) to get the
  first image into GHCR, add the nginx site + `certbot --nginx -d <domain>`, then run
  `~/elproof/deploy.sh <git-sha>` for the first real deploy.
