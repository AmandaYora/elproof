# Deployment — ElProof

## 1. Prerequisites

- Node.js 20+, Go 1.25+
- A MySQL 8 server reachable from your machine (this project never runs the
  database inside Docker — see `.claude/rules/monorepo.md`)
- [`golang-migrate`](https://github.com/golang-migrate/migrate) CLI on your `PATH`
  (only needed for `npm run migrate:*`)

## 2. Clone and configure

```bash
git clone <repo-url> elproof
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

### Demo login in Docker

The login page's demo-login buttons call a dev-only backend endpoint
(`POST /api/v1/auth/demo-login`) that is only registered when `APP_ENV=development`
(see `identity.module.go`). Set `APP_ENV=production` in `.env` for a real
deployment to disable it — the frontend then hides the "Akun Demo" panel
automatically (`import.meta.env.PROD`), and the endpoint won't exist to hit
even if someone tried.

### Known limitation of this document

This Docker path (`docker compose up --build` reaching a host MySQL,
end-to-end) has been reasoned through and the compose/Dockerfile config
fixed accordingly, but **has not been executed in this environment** — Docker
is not installed here. If `docker compose up` surfaces an issue not covered
above, treat this section as a strong starting point rather than a
guarantee, and update it once verified.
