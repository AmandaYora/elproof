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

The one deliberate exception: a **one-time migration backfill** may cross module tables directly in
raw SQL (`000022`, `000026`, `000028`) — this is a DBA-level, run-once operation, never a live
application-code query path, so it doesn't reintroduce the coupling the rule above exists to
prevent. Going forward, the same data is always maintained by ordinary in-module repository code,
never a live join.

## Conventions

- Primary keys: `BIGINT UNSIGNED AUTO_INCREMENT`.
- Cross-module references: plain `BIGINT UNSIGNED` / `VARCHAR` columns named `<entity>_id`, with
  **no** SQL `FOREIGN KEY` constraint when the referenced table lives in another module (per
  `.claude/rules/database.md`). Foreign keys are only allowed within the same module (e.g.
  `project_milestones.project_id → projects.id` is fine; `projects.tenant_id → tenants.id` is not,
  since `tenants` belongs to `platform`).
- Tenant-scoped tables: `tenant_id BIGINT UNSIGNED NOT NULL` (see ADR-0004), always the first
  filter in any query.
- Indexes follow the queries (Fase 7, migration 64): a list that filters by tenant and sorts needs one
  composite index in that order, e.g. `projects (tenant_id, is_archived, event_date, id)` for
  `WHERE tenant_id = ? AND is_archived = ? ORDER BY event_date DESC, id DESC`. Measured on 80 000
  projects and 60 000 transactions: the project page went from a 46 ms filesort of the whole tenant to
  0.2 ms, the platform transaction list from a 23 ms table scan to 0.1 ms. Drop the single-column index a
  composite one covers (its leftmost prefix does the same job); InnoDB appends the primary key to every
  secondary index, so `ORDER BY id` after an equality filter needs no extra column. Check with
  `EXPLAIN FORMAT=TREE`; `LIKE '%x%'` searches cannot use an index and are bounded by the tenant filter.
- Timestamps: `created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP`, `updated_at TIMESTAMP NOT
  NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP` where mutation history matters.
- Soft state over hard delete: the frontend consistently uses `is_active`/status-toggle rather than
  deleting rows (plans still only "deactivate") — mirror this server-side; don't introduce hard
  deletes the frontend never asks for. Five deliberate, explicit exceptions exist, each narrowly
  scoped and guarded rather than general-purpose: `clients.Delete` (one bug-workaround scenario,
  see its own doc comment), `projects.Delete` (Owner-only, requires the project already
  archived/cancelled first — block-then-cascade, see ADR-0013), `MilestoneTemplateService.Delete`
  (Timeline Default Template items, Owner-only — safe here specifically because a template row is
  only ever *copied* into a project's own `project_milestones` at creation time, never referenced
  again afterward, so there's nothing to orphan), `vendors.Delete`/`venues.Delete`/
  `staff.Delete` (Owner-only, **no** precondition — an *informed-consent* confirmation dialog names
  exactly which projects would be affected, rather than blocking the delete; the tenant's own Owner
  account is the one absolute non-negotiable exception; Vendor Category stays hard-blocked instead,
  since its FK is same-module and non-nullable — see ADR-0018 for the full guardrail design and why
  it deliberately differs from ADR-0013's block-then-cascade model), and
  `PaymentService`/`ClientPaymentService`/`VenuePaymentService.Delete` (vendor/client/venue
  payments — **Owner-or-Admin**, the one exception that isn't Owner-only, an explicit user request;
  no precondition, but cascades to the payment's own `evidence` rows first, since evidence has no
  delete capability of its own — see ADR-0019). Don't treat any of these five as license to add
  more hard deletes without the same explicit reasoning and guardrails.

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
