# Architecture Overview

> Apps, modules, data flow, and boundaries.

## Apps

- `apps/web` — React 19 + Tailwind 4 SPA. Three route trees sharing one router
  (`app/routes/index.tsx`): protected WO Console (`AppLayout`), Client Portal (`ClientPortalLayout`),
  Platform Console (`PlatformLayout`), plus public routes (`/login`).
- `apps/api` — Go modular monolith. See [MODULE_MAP.md](MODULE_MAP.md) for the module list and
  [BACKEND_GUIDE.md](BACKEND_GUIDE.md) for conventions.

## Data flow (target, post-integration)

```
Browser (apps/web)
  → shared/services/http-client.ts (Axios, Authorization header from useAuthStore)
  → apps/api (Go, net/http mux)
      → shared/middleware: verifies JWT, injects {principal_type, principal_id, tenant_id, role}
      → module presentation handler → application use case → domain → infrastructure (MySQL via sqlc)
  ← {success, message, data} / {success:false, message, errors} envelope, paginated lists include
    meta: {page, limit, total, total_pages}
```

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
manage tenants and the plan/transaction catalog *across* tenants. See ADR-0004.

## Full readiness audit & phased build plan

The as-was state of the frontend (before backend integration began) and the phase-by-phase plan to
build `apps/api` and wire it up is tracked in `PLAN.md` at the repo root — treat it as a living
checklist, not a one-time document.
