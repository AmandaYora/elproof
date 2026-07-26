# Architecture Overview

> Apps, modules, data flow, and boundaries.

## Apps

- `apps/web` — React 19 + Tailwind 4 SPA. Route trees sharing one router (`app/routes/index.tsx`):
  protected WO Console (`AppLayout`), Client Portal (`ClientPortalLayout`), Platform Console
  (`PlatformLayout`), public auth (`/login`), and the public marketing site (`MarketingLayout`,
  `/homepage*`) — the latter is frontend-only, no backend module or API calls (see `homepage` row
  in [MODULE_MAP.md](MODULE_MAP.md)).
- `apps/api` — Go modular monolith. See [MODULE_MAP.md](MODULE_MAP.md) for the module list and
  [BACKEND_GUIDE.md](BACKEND_GUIDE.md) for conventions.

## Data flow (current)

```
Browser (apps/web)
  → shared/services/http-client.ts (Axios, Authorization header from useAuthStore)
  → apps/api (Go, net/http mux)
      → shared/middleware: verifies JWT, injects {principal_type, principal_id, tenant_id, role}
      → module presentation handler → application use case → domain → infrastructure (MySQL,
        hand-written database/sql — see DATABASE_GUIDE.md)
  ← {success, message, data} / {success:false, message, errors} envelope, paginated lists include
    meta: {page, limit, total, total_pages}
```

External consumers of `payment`'s Fase 10 API follow the same shape, just with a 4th JWT principal
type (`app`, minted via `POST /auth/app/token` rather than `/auth/login`) — see
`MODULE_PAYMENT.md` §7 and ADR-0005's update.

Alongside this request/response pipeline, one background process runs for the lifetime of the
server: `payment`'s reconciliation sweep (`ReconcilePending`, a `time.Ticker` goroutine started
from `main.go`), which re-checks charges whose webhook was never received directly against the
gateway — see `MODULE_PAYMENT.md` §6 step 6. It doesn't fit the diagram above (no browser/HTTP
request triggers it), but it's a real, always-running part of the current architecture.

## Module boundaries

Same rule on both sides of the stack: **a module only exposes its public contract to others.**
- Backend: `internal/modules/<module>/contracts/` is the only importable package from outside the
  module. No cross-module service/repository/domain imports, no cross-module DB joins/FKs — relations
  are primitive IDs, resolved via another module's contract client at the application-service layer.
- Frontend: cross-module reads happen via `otherStore.getState()` (Zustand), never by importing
  another module's internal types/data files directly. Shared reference data (e.g. the subscription
  plan catalog) lives in `shared/`, not duplicated per module — see ADR-0009.

## Multi-tenancy

Tenant-scoped modules (`staff`, `clients`, `vendors`, `projects`) filter every query by `tenant_id`
taken from the JWT, never from client input. `platform` and `billing` are tenant-agnostic — they
manage tenants and the plan/transaction catalog *across* tenants. See ADR-0004. `payment` is also
tenant-agnostic, for a different reason: it's deliberately ignorant of tenants entirely (no
`payment_*` table has a `tenant_id` column) — its `contextID` parameter is opaque, threaded through
for a caller's own use but never inspected, since the same gateway wallet serves both ElProof's own
tenant-scoped billing and external Apps that have no concept of an ElProof tenant at all. See
`MODULE_PAYMENT.md` §2.2.

## Current status

`apps/api` is fully built and live in production (see [PROJECT_BRIEF.md](PROJECT_BRIEF.md)'s
"Current state") — all eight modules listed in [MODULE_MAP.md](MODULE_MAP.md) are implemented, not
in progress. The phase-by-phase build log that tracked this work while it was underway has since
been removed now that it's done; `MODULE_MAP.md`, `docs/API_CONTRACT.md`, `docs/DB_SCHEMA.md`, and
`knowledge/decisions/` are the current source of truth for what exists and why.
