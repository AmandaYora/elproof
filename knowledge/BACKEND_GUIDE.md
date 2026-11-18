# Backend Guide

> Backend conventions specific to this project.

Base rules: `.claude/rules/backend-modular-monolith.md`, `.claude/rules/database.md`,
`.claude/rules/api-standard.md`. This file is the ElProof-specific instantiation of those rules —
see [MODULE_MAP.md](MODULE_MAP.md) for the module list.

## Per-module layout

```
internal/modules/<name>/
  domain/            # entities, value objects, domain errors — no framework/DB imports
    events/           # domain events, if any
  application/        # use cases (services) — orchestrate domain + call repository interfaces
  infrastructure/      # MySQL repository implementations, sqlc-generated code consumers
    queries/           # .sql files sqlc reads (sqlc.yaml: queries scanned from internal/modules)
  presentation/        # HTTP handlers — parse request, call application use case, map to response envelope
  contracts/           # the ONLY package other modules may import
  <name>.module.go     # wiring: constructs repository → service → handler, registers routes
```

**Documented exception: `payment` (Fase 9/10) has no `application/` and no `domain/events/`.** Its
`infrastructure.PaymentService` implements the module's contracts directly — see `MODULE_PAYMENT.md`
§3 and ADR-0008's update for why (it owns no business process complex enough to need a use-case
layer separated from its infrastructure). Don't treat this template as an unconditional per-module
invariant; a module with a similarly thin domain may reasonably skip `application/` too, as long as
the deviation gets the same explicit callout `payment` did rather than silently diverging.

## Cross-module calls

A module needing data from another module calls that module's `contracts` package directly (in-process
function call — this is a monolith, not microservices) — never its `domain`/`infrastructure`. Example:
`platform`'s tenant-registration use case calls `staff.Contracts.CreateOwner(...)` and
`identity.Contracts.CreateCredential(...)`; it never imports `staff/domain` or touches `staff`'s
tables.

## Auth middleware

Lives in `internal/shared/middleware` (technical utility: JWT signature/expiry verification), not
inside `identity`. `identity` owns *issuing* tokens (business rules); `shared` owns *verifying* them
on every other request. See ADR-0005.

## Time: the business clock

"Now" for a domain rule — what day it is for H-n, overdue installments, lagging preparation, trial
and subscription days, undo windows, the purge of deleted rows, a record's own `createdAt` — comes from
`clock.Now()` (`internal/shared/clock`), never `time.Now()`. The calendar day is the WIB day
(ADR-0007): `nowWIB()` in a handler is `clock.Now().In(wib)`. A service that needs "now" takes it as an
argument (`Get(ctx, tenantID, asOf, …)`) or as a `now func() time.Time` field defaulting to
`clock.Now`, so unit tests pass a fixed instant.

In every real environment `clock.Now()` is `time.Now()`. Only the end-to-end stack sets
`APP_CLOCK_START` (RFC 3339, e.g. `2026-11-09T10:00:00+07:00`): the clock starts at that instant when
the process starts and keeps ticking, so the server agrees with the test browser, which is frozen at
the same instant, whatever day the suite runs. `cmd/server` refuses to start with it when
`APP_ENV=production`.

Security and wall-clock timing stay on `time.Now()` on purpose: JWT and refresh-token lifetimes, TOTP,
sign-in links, Google ID tokens, rate limits, signed storage URLs, payment gateway calls and the
subscription reconciler are checked against parties that only know real time. SQL keeps `NOW()` only
where both sides of the comparison are database time (idempotency keys). A new `time.Now()` in a
module's application/presentation code is a bug unless it is one of these.

## Local dev

```bash
npm run dev:api     # from repo root — runs apps/api via Air (hot reload), not `go run` directly
```
`migrate` CLI applies migrations against the host MySQL (`elproof_db`); `sqlc generate` regenerates
typed query code after editing any `infrastructure/queries/*.sql` file (currently blocked on this
dev machine — see `knowledge/DATABASE_GUIDE.md`'s known-limitation note; hand-written
`database/sql` repositories are the interim fallback).

`internal/shared/middleware/cors.go` is only active when `APP_ENV=development` — it reflects
whatever `Origin` sent the request, since `npm run dev:web` (Vite) and `npm run dev:api` run on
different ports locally. It is a no-op in production, where one Docker app container serves both
under the same origin.

Its `Access-Control-Allow-Methods` list is a manually-maintained allow-list, not derived from the
routes actually registered — `PUT /tenants/{id}/logo` (tenant branding, `MODULE_PLATFORM.md` §6)
was the first endpoint in this codebase to use `PUT`, and briefly broke in dev (silently
CORS-blocked, not caught by `go build`/`tsc`) until `PUT` was added to the list. Any *next* new HTTP
method introduced here will hit the same wall until this list is made method-agnostic instead.

## What NOT to do (anti-overengineering, project-specific)

No microservices, no message queue, no Redis/Memcache, no GORM/auto-migrate, no cross-module SQL
joins even when it would be "one query" — resolve via contracts instead, even at the cost of an
extra in-process call.
