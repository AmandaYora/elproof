# ADR-0011: Source repository and production deployment target

## Status
Accepted and deployed. ElProof is **live** at <https://elproof.elcodelabs.com> as of 2026-07-17
(see "Update" section for the pipeline alignment, and "Deployment" below for the first real
deploy and the bugs it surfaced).

## Context
The project needed a canonical Git remote and a concrete production server before any real deploy
could happen. Both are now fixed:

- **Source repository:** `https://github.com/AmandaYora/elproof.git` — `main` branch, pushed
  2026-07-16 as the project's first commit (382 files; `node_modules/`, `.env`, `tmp/`, and the
  personal `.claude/settings.local.json` excluded via `.gitignore`).
- **Target VPS:** an existing IDCloudHost virtual machine, dashboard-labeled **"Elcodelabs"**
  (matches this project's `.env` — `S3_BUCKET=elcodelabs`, see ADR-0006), specs:
  - OS: Ubuntu 24.04.4 LTS, hostname `Septi`
  - 2 vCPU / 2 GB RAM / 20 GB disk, region Singapore (`sgp01`)
  - Public IP `103.189.235.79`, private IP `10.0.131.3`
  - Login user: `dimasprasetio`

This VM already existed before ElProof (originally provisioned for a different project, "elkasir")
and is being repurposed rather than a fresh VPS being ordered. SSH access was already configured on
the development machine and was verified working (key-based, no password prompt) on 2026-07-16 via
the pre-existing `~/.ssh/config` entry:

```
Host elkasir vps-elkasir
    HostName 103.189.235.79
    User dimasprasetio
    IdentityFile ~/.ssh/elkasir_vps_ed25519
```

## Decision
- Use the GitHub repo above as the single source of truth for deploys (clone/pull on the VPS rather
  than copying files manually).
- Use the `elkasir` / `vps-elkasir` VPS (IP `103.189.235.79`) as ElProof's production target,
  reusing the existing SSH key rather than provisioning new infrastructure or new credentials.
- The local SSH alias keeps its old name (`elkasir`) for now — it is not renamed to `elproof` in
  this pass, to avoid touching shared machine config as a side effect of a documentation-only task.
  Rename it (and update this ADR) whenever the actual deploy work happens, if desired.
- Deployment on this VPS follows the one-app-container model, but **not** via `docker compose up
  --build` on the server itself (that was this ADR's original, incorrect assumption — see
  "Update" below). The VPS enforces its own stricter, pre-existing multi-app convention.

## Consequences
- `docs/DEPLOYMENT.md` §6 documents this VPS as the concrete production target and the real
  CI→GHCR→pull pipeline (superseding its original "run `docker compose up --build` on the VPS"
  draft).
- Because this VM is a reused, previously-named-for-another-project host ("elkasir", still live
  as a sibling app on the same box), ElProof must never assume it owns the whole machine: its own
  database/user, its own `127.0.0.1` port, its own nginx `server_name` — never `elkasir`'s.

## Update — 2026-07-17: the VPS already has its own deployment convention

This ADR's original "Decision" assumed a generic single-VPS Docker deploy (`docker compose up
--build`, reasoned through in `docs/DEPLOYMENT.md` §5 for *local* Docker testing). Once actually
inspecting the VPS (`~/AGENTS.md` → `/opt/elcodelabs/SERVER_PLAYBOOK.md`, the canonical,
already-established multi-app convention this box's owner uses for every app on it, `elkasir`
included), that assumption turned out to be wrong for this specific host: the VPS has only 2 GB
RAM and its golden rule is **the box never builds** — images are built in CI and pushed to GHCR;
the VPS only `docker pull`s and runs. Aligning with that existing, working convention (rather than
introducing a second, inconsistent deploy style on the same machine) means:

- **CI builds, VPS pulls.** `.github/workflows/deploy.yml` (new) builds
  `infra/docker/Dockerfile` and pushes `ghcr.io/amandayora/elproof:<git-sha>` + `:latest` on push
  to `main`. The VPS's `~/elproof/docker-compose.prod.yml` is image-only (no `build:` key).
- **One binary, no separate ops tooling in the image.** `cmd/server`'s compiled binary now
  dispatches on an optional subcommand (`migrate up`/`down`, `seed`, `healthcheck`, default
  serve) — mirroring `elkasir`'s own binary, which already works this way on this VPS. This
  avoids needing the `golang-migrate` CLI, a separate seed binary, or curl/wget inside the
  minimal Alpine image; those local-dev conveniences (`npm run migrate:*`, `go run ./cmd/seed`)
  are unaffected and still work exactly as before. See `apps/api/internal/migrator` (embeds
  `apps/api/migrations/*.sql` via `go:embed`) and `apps/api/internal/adminseed` (the seed logic,
  now shared between `cmd/seed` and `cmd/server`'s `seed` subcommand).
- **Per-app database isolation.** `elproof_db` / `elproof_user@'172.%'` (own `mysql_native_password`
  credential, own grants) — created alongside, never sharing, `elkasir`'s existing database/user
  on the same host-level MySQL instance.
- **Own port, own secrets.** Container publishes to `127.0.0.1:8082` (elkasir already holds
  `8081`); `~/elproof/.env` holds freshly generated `JWT_SECRET`/`PAYMENT_ENCRYPTION_KEY`/DB
  password — none copied from local dev's `.env` or from elkasir's.
- **Own domain, own nginx site.** `elproof.elcodelabs.com`, its own `server_name` — never touching
  elkasir's `default_server` on port 80.
- The VPS's own `/opt/elcodelabs/SERVER_PLAYBOOK.md` (root-owned, backed up before each edit) is
  kept in sync with reality per that document's own instruction ("UPDATE THIS when you add an
  app") — App registry row + a "Status — elproof" section.

## Deployment — 2026-07-17: first real deploy, live

Domain chosen: `elproof.elcodelabs.com` (same subdomain pattern as `elkasir.elcodelabs.com`). Its
DNS had to be corrected first — it was an ALIAS record pointing at an unrelated Hostinger-hosting
CDN target (leftover from something else on the same registrar account), not this VPS; fixed to a
plain `A` record → `103.189.235.79` at the registrar, then verified propagated before touching
nginx.

nginx site + `certbot --nginx -d elproof.elcodelabs.com` succeeded cleanly. The first
`deploy.sh` run then surfaced two real bugs (both fixed, both now covered by this repo's own
regression-proofing via manual verification, not automated tests — see `docs/DEPLOYMENT.md` §6
for the full technical detail):

1. `infra/docker/Dockerfile` used `CMD` instead of `ENTRYPOINT`, so passing `migrate up` to
   `docker run` replaced the whole command instead of appending an argument.
2. `apps/api/internal/migrator` wrapped the app's shared `*sql.DB` instead of opening its own
   multi-statement-enabled connection, so migration files with more than one `CREATE TABLE`
   (`000004_create_billing_tables`) failed partway through.

Bug #2 left `schema_migrations` dirty at version 4 (with migration 4's DDL never actually
applied — verified by hand before touching anything). Recovering it by hand would have meant a
raw SQL `UPDATE` against the production database over SSH; instead, a `migrate force <version>`
subcommand (mirroring upstream golang-migrate's own recovery command) was added to `cmd/server`
so the fix goes through the same image/tooling as every other deploy operation, not an
out-of-band database edit. Redeployed clean afterward: all 25 tables present, `schema_migrations`
at version 10 / not dirty, minimal seed data in place, login verified end-to-end over HTTPS.

**Update — since 2026-07-17:** migrations `000011` (Fase 10 reconciliation columns on
`payment_charge_dispatch`) have landed and deployed the same way (`deploy.sh` → `migrate up`), each
time without incident. Table count is unaffected by 000011 (it only adds columns to an existing
table), so "all 25 tables" is still accurate; `schema_migrations`'s version number itself is not —
treat "version 10" above as a point-in-time snapshot from the first deploy, not a claim about the
current version. Check the deploy log or `SELECT version FROM schema_migrations` for the live
number rather than trusting a number written here, since this note will itself go stale again the
next time a migration lands.
