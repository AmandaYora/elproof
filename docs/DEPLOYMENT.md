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
   - `./api migrate up` — applies **every pending** migration in one call (`golang-migrate`'s
     `m.Up()`, `apps/api/internal/migrator/migrator.go:41`), not "one step" — a deploy after
     several unreleased migrations have accumulated applies all of them in a single run. `./api
     migrate down` **is** one step (`m.Steps(-1)`, `migrator.go:53`) — `up` and `down` are NOT
     symmetric, despite an earlier version of this doc claiming otherwise. This distinction matters
     in practice: it's what let migrations 37–45 (`docs/plan/external-subscription-plans/PLAN.md`)
     silently queue up behind three unreleased commits without anyone noticing until deploy time —
     see §7 below.
   - `./api` (no args) — serve, exactly like local dev.
   - Both `migrate up`/`migrate down` and the local `npm run migrate:up`/`migrate:down` drive the
     same `apps/api/migrations/*.sql` files (now embedded in the binary via `go:embed`, see
     `apps/api/internal/migrator`) — one through the external `golang-migrate` CLI, the other
     through the compiled binary. Both paths share one `schema_migrations` table format, so
     they're interchangeable — the CLI (§1 prerequisite) stays a local-dev convenience only, never
     required in production.
   - `./api migrate force <version>` — clear a "dirty" `schema_migrations` state without running
     any SQL (mirrors the upstream golang-migrate CLI's `migrate force`); the standard recovery
     tool if a migration is ever interrupted mid-way.
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
5. **nginx + TLS** front the container's `127.0.0.1:8082`.

**TLS (updated 2026-11-06):** the certificate is `--cert-name elproof.elcodelabs.com` covering
**only** `elproof.elcodelabs.com` (authenticator + installer `nginx`), renewed by the system
`certbot.timer` (twice daily), with `/etc/letsencrypt/renewal-hooks/deploy/reload-nginx.sh`
reloading nginx after every renewal. Never add another domain to this certificate — on
2026-10-28 it expired because a tenant custom domain (`journey.jwswedding.com`) that had been
added to it moved to another server and failed its challenge (see ADR-0015 "Deprecation").
Quick checks:

```bash
ssh elcodelabs 'sudo certbot certificates && sudo certbot renew --dry-run'
echo | openssl s_client -servername elproof.elcodelabs.com -connect elproof.elcodelabs.com:443 2>/dev/null | openssl x509 -noout -dates
```

**Status as of 2026-07-17 — LIVE at <https://elproof.elcodelabs.com>:**
- `elproof-app` container running and healthy on `127.0.0.1:8082`; `elproof_db` fully migrated
  (all 25 tables) and seeded (`superadmin`/`superadmin`, "Paket 1 Tahun"); nginx site + certbot
  TLS in place (its own `server_name`, does not touch elkasir's `default_server`); end-to-end
  verified (health check, login, and the built frontend all reachable over HTTPS). This is a
  point-in-time snapshot from the first deploy — later migrations (e.g. `000011` for Fase 10's
  reconciliation sweep) have landed since through the same `deploy.sh` pipeline without incident;
  see ADR-0011's own "Update" note rather than treating the migration count/version here as current.
- The domain's DNS record needed a fix before this worked: `elproof.elcodelabs.com` initially
  pointed at an unrelated Hostinger-hosting ALIAS record (leftover from something else on the
  same registrar account), not this VPS — corrected to a plain `A` record → `103.189.235.79`.
- The first deploy attempt hit two real bugs, both fixed and worth knowing about if this ever
  needs debugging again:
  1. `infra/docker/Dockerfile` used `CMD ["./api"]` instead of `ENTRYPOINT ["./api"]` — `CMD` is
     fully replaced by arguments passed to `docker run`, so `docker run <image> migrate up` tried
     to exec a binary literally named `migrate` instead of running `./api migrate up`.
  2. `apps/api/internal/migrator` originally wrapped the app's shared `*sql.DB` via
     `mysql.WithInstance`, which doesn't enable multi-statement queries — several migration files
     (e.g. `000004_create_billing_tables`) have more than one `CREATE TABLE` per file, and the
     driver rejected the second statement onward. Fixed by using
     `migrate.NewWithSourceInstance(...databaseURL)`, which opens its own connection and enables
     multi-statement support the same way the golang-migrate CLI already does for local dev.
  3. Bug #2 left `schema_migrations` "dirty" at version 4 with migration 4's DDL never actually
     applied. Recovered with the new `./api migrate force <version>` subcommand (mirrors upstream
     golang-migrate's `migrate force`) rather than a manual SQL `UPDATE` against the production
     database — see `apps/api/internal/migrator`'s `Force`.

## 7. Two independent reconcilers (`docs/plan/external-subscription-plans/PLAN.md`)

Since that plan landed, the server starts **two** background reconciliation loops, not one — both
should be visible ticking in the container logs after a deploy:

- `payment: reconciliation sweep` — `payment`'s own sweep over `payment_charge_dispatch`
  (`PAYMENT_RECONCILE_INTERVAL`, default `3m`), unchanged from before.
- `billing: subscription reconciliation sweep` — a **separate** reconciler over
  `subscription_app_charges` (`SUBSCRIPTION_RECONCILE_INTERVAL`, default `3m`; also reads
  `SUBSCRIPTION_CHARGE_MAX_AGE`, default `24h`, before force-expiring a still-unpaid charge).

The two call the payment gateway **independently** — for a subscription charge, both reconcilers
may end up checking the same `order_ref` against the gateway around the same time. This is
accepted and by design (`billing` deliberately doesn't hook into `payment`'s own dispatch switch,
to avoid touching the code path that processes real WO-tenant money) — two nearby log lines for one
`order_ref` is expected behavior, not a duplicate-firing bug. If only one of the two reconcilers is
ticking after a deploy, that's the actual problem to chase.
