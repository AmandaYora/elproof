# System Design — ElProof

Architecture, boundaries, apps, modules, data flow.

Full detail lives in [`knowledge/ARCHITECTURE_OVERVIEW.md`](../knowledge/ARCHITECTURE_OVERVIEW.md)
(apps/data-flow/multi-tenancy) and [`knowledge/MODULE_MAP.md`](../knowledge/MODULE_MAP.md) (module
responsibility + ownership) — not duplicated here.

Quick summary: `apps/web` (React 19 SPA) talks to `apps/api` (Go modular monolith, seven modules:
`identity`, `platform`, `billing`, `staff`, `clients`, `vendors`, `projects`) over
`/api/v1`, one JWT scheme for three principal types (staff/client/platform_admin), one MySQL
database on the host, one Docker app container. See `PLAN.md` for the phased build/integration plan
and `knowledge/decisions/` (ADR-0004 through ADR-0009) for the specific decisions behind this shape.
