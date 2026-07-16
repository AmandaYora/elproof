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

## What NOT to do (anti-overengineering, project-specific)

No microservices, no message queue, no Redis/Memcache, no GORM/auto-migrate, no cross-module SQL
joins even when it would be "one query" — resolve via contracts instead, even at the cost of an
extra in-process call.
