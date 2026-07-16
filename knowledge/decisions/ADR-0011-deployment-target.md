# ADR-0011: Source repository and production deployment target

## Status
Accepted — updated 2026-07-17 once the VPS's own deployment convention was discovered and
followed (see "Update" section below). Infrastructure is now set up; production deployment
(first real `docker compose up`) has still not been executed — pending a domain assignment.

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
- **nginx/TLS deferred.** elkasir currently holds nginx's one allowed `default_server` for port
  80 (catches the bare IP). ElProof needs its own domain and `server_name` before an nginx site
  can be added — this is the one remaining blocker before the first real deploy, and is a decision
  only the project owner can make (which domain to point at this VPS).
- The VPS's own `/opt/elcodelabs/SERVER_PLAYBOOK.md` (root-owned, backed up before editing) now
  has an elproof row in its App registry table and a "Status — elproof" section, kept in sync
  with reality per that document's own instruction ("UPDATE THIS when you add an app").
