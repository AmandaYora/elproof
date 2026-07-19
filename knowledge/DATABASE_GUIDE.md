# Database Guide

> Which module owns which tables; cross-module relations stored as primitive IDs.

See [`docs/DB_SCHEMA.md`](../docs/DB_SCHEMA.md) for full column-level schema. This file covers
conventions that apply across all tables.

## Ownership

Each table belongs to exactly one module (see [MODULE_MAP.md](MODULE_MAP.md)). A module's
repository code is the only code allowed to `SELECT`/`INSERT`/`UPDATE`/`DELETE` against its own
tables. No other module's repository touches it directly, and no SQL `JOIN` ever spans two
modules' tables — a cross-module reference is always just a stored ID, resolved by calling the
owning module's contract function when the display layer needs the related data.

## Conventions

- Primary keys: `BIGINT UNSIGNED AUTO_INCREMENT`.
- Cross-module references: plain `BIGINT UNSIGNED` / `VARCHAR` columns named `<entity>_id`, with
  **no** SQL `FOREIGN KEY` constraint when the referenced table lives in another module (per
  `.claude/rules/database.md`). Foreign keys are only allowed within the same module (e.g.
  `project_milestones.project_id → projects.id` is fine; `projects.tenant_id → tenants.id` is not,
  since `tenants` belongs to `platform`).
- Tenant-scoped tables: `tenant_id BIGINT UNSIGNED NOT NULL` (see ADR-0004), always the first
  filter in any query.
- Timestamps: `created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP`, `updated_at TIMESTAMP NOT
  NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP` where mutation history matters.
- Soft state over hard delete: the frontend consistently uses `is_active`/status-toggle rather than
  deleting rows (vendors, clients, staff, plans all "deactivate" instead of delete) — mirror this
  server-side; don't introduce hard deletes the frontend never asks for.

## Tooling

- Migrations: `golang-migrate` (`migrate create -ext sql -dir migrations -seq <name>`), MySQL
  engine. Never hand-edit an already-applied migration file — add a new one.
- Query access: `sqlc` (`sqlc.yaml` at `apps/api/` root) generates typed Go from `.sql` files under
  each module's `infrastructure/queries/`. Not GORM, no ORM auto-migrate.
- Local DB: MySQL running on the host (Laragon on this dev machine), database name `elproof_db`,
  matching `.env` / `.env.example` (`DB_HOST=localhost`, `DB_USER=root`, `DB_PASSWORD=`).

### Known local-environment limitation: `sqlc` CLI install

`go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest` currently fails on this dev machine with
`cc1.exe: sorry, unimplemented: 64-bit mode not compiled in` — the local MinGW/gcc toolchain used
for CGO is 32-bit-only, but one of sqlc's transitive dependencies needs to compile a CGO package in
64-bit mode. Fixing this requires installing a 64-bit MinGW-w64 toolchain (out of scope for a
quick fix). Until then, **every module's** repositories are hand-written with `database/sql`
directly, behind the exact same repository interfaces their `application`/`infrastructure` layers
expect — not just `identity` (the first module built): `billing`, `staff`, `clients`, `vendors`,
`platform`, `projects`, and `payment` (Fase 9/10) all followed the identical fallback. Every
module's `infrastructure/queries/` directory (where sqlc-generated code would read `.sql` from)
exists but is empty — dead scaffolding, not active input — and `internal/database/` (sqlc's `out:`
target) has never had anything generated into it. Swapping in sqlc-generated code later (once the
toolchain is fixed) only touches each module's own `infrastructure/`, nothing else, since the
repository interfaces were always designed against this eventual swap.
