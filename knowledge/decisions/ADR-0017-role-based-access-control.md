# ADR-0017: Role-Based Access Control (Owner / Admin / Wedding Planner)

## Status
Accepted

## Context
`staff_members.role` (`Owner`/`Admin`/`Staff`) has flowed through `credentials`/JWT claims since
Fase 3 (see `DOMAIN_GLOSSARY.md`), but almost nothing in the app actually branched on it — the only
pre-existing role checks anywhere were `Langganan` being Owner-only (frontend convention only) and
ADR-0013's hard-delete guard (`claims.role != "Owner"`, the module's first real server-side role
check). Every other page and endpoint was reachable by any authenticated staff member regardless of
role.

The user asked for a real three-tier access model across the whole WO Console:
- **Owner**: bisa akses semua (access to everything).
- **Admin**: akses ke semua kecuali user & setting (everything except Users & Settings).
- **Wedding Planner**: bisa akses project saja (Project access only).

Refined once during analysis (the user's own correction, not an assumption): *"yang membuat project
itu adalah owner atau admin saja, kemudian menugaskan ke wedding planner... yang melakukan assigned
project terhadap wedding planner itu hanya admin atau owner saja"* — only Owner/Admin ever create a
project or decide who its PIC is; a Wedding Planner only ever works within a project already handed
to them.

## Decision

### The three roles, exactly
1. **Owner** — full access to everything.
2. **Admin** — full access to everything **except** the "Pengaturan" group entirely (Pengguna,
   Langganan, Kategori Vendor, Timeline Default) — not just restricted, invisible in the Sidebar too.
3. **Wedding Planner** — the backend role/enum value stays `"Staff"` (unchanged identifier; this is
   a display-label-only distinction, same precedent as the Milestone→Timeline rename). Scoped to
   only **Project**, and within Project, only to the ones where `picStaffId` equals their own staff
   id. Cannot see Dashboard, Client, Vendor, Venue, or any Pengaturan page at all. Cannot create a
   project, cannot duplicate one, and cannot reassign a project's PIC (Owner/Admin only, even on a
   project this Wedding Planner already manages day to day).

### Enforcement is server-side on every request, never just a frontend convention
This codebase has no shared `middleware.RequireRole()` — role checks are small, duplicated guard
functions per package (`requireOwnerRole`, `requireOwnerTenant`, `requireManagerRole`, `requireOwnerForTemplates`,
etc.), matching the existing convention `platform`'s `tenant_handler.go` already established for
ADR-0013's own Owner-only check. The one new shared primitive is `Claims.HasRole(roles ...string)
bool` on `internal/shared/middleware/auth.go`, used by every one of those guard functions instead of
each hand-rolling its own `==`/loop comparison.

### The single choke point for Wedding Planner's project scoping: `resolveProjectAccess`
Every `/projects/{id}/...` sub-resource (milestones, vendor engagements, payments, client-payments,
issues, evidence, activity, venue) already funneled through one function in
`projects/presentation/handler.go`. The Wedding Planner's PIC check was added exactly once, there —
`sc.role == "Staff"` → `p.PICStaffID != sc.staffID` → 403 — rather than repeated per sub-resource, so
a Wedding Planner can't reach another project's data by guessing its numeric id even within their own
tenant. `GET /projects` (list) and `GET /projects?all=true` apply the equivalent scoping at the query
level (`picStaffID *int64` threaded through `ProjectRepository.List`/`ListPaginated`) instead — a
narrower result set, never a 403, since listing is expected to return *something* for every role.

### Addendum (found + fixed alongside the Client Payments feature): cross-tenant gap in `resolveProjectAccess`
This ADR's own text above already claimed "Owner/Admin get the existing full access (any project in
their tenant)" — but the code never actually enforced the "in their tenant" part for those two roles.
`h.projects.Get(ctx, sc.tenantID, projectID)` (the tenant-scoped lookup that returns `NotFound` when
`projectID` belongs to a different tenant) was only ever called inside the `if sc.role == "Staff"`
branch — Owner/Admin skipped it entirely and fell straight through to `return sc, true`. Since none
of the sub-resource repositories filter by `tenant_id` either (only `project_id`), and project ids
are a single global `BIGINT AUTO_INCREMENT` sequence (not per-tenant), any authenticated Owner/Admin
in *any* tenant could read — and, for evidence, download the actual file bytes of — another tenant's
`vendors`, `milestones`, `payments`, `client-payments`, `issues`, `evidence`, and `activity` by
guessing/enumerating a numeric project id. This predates this ADR (this ADR only added the
*additional* Wedding Planner narrowing on top of an assumed-already-tenant-scoped baseline) and was
never an accepted tradeoff — ADR-0004 states unconditionally that every tenant-owned table's queries
filter by `tenant_id`. **Fixed**: `h.projects.Get(ctx, sc.tenantID, projectID)` now runs for every
staff principal regardless of role; the PIC check applies only on top of that, for `role == "Staff"`.
Verified with a throwaway JWT-minting smoke test (not committed) hitting a locally running server:
cross-tenant access now 404s (both for `client-payments` and, confirming the fix isn't
client-payments-specific, the sibling `payments` endpoint), same-tenant Owner/Admin access and the
Wedding Planner PIC-scoping 403 both still behave exactly as before.

### Read-open, write-restricted split for shared reference data
Vendor, Venue, and Vendor Category reads (list/get, including the picker-friendly `?all=true`
shape) stay open to every staff role, Wedding Planner included — a project's own Vendor/Venue tabs
need to resolve names and reference prices regardless of who's viewing. Every write
(create/update/toggle-active/attachment-upload/template/import) on those three is Owner/Admin only
(`requireManagerRole` in `vendors`, `requireOwnerRole` in Vendor Category). This is a deliberate,
narrower reading than "Wedding Planner cannot access Vendor/Venue at all" — the literal stated rule
— accepted as a known, pre-existing gap (not touched by this ADR) rather than tightened, since the
picker use case is real and load-bearing. The Vendor/Venue Sidebar entries themselves are
`allowedRoles: ["Owner", "Admin"]` (Wedding Planner never sees the menu item), but a Wedding
Planner's own project tabs still call these same read endpoints under the hood.

### Public-safe "summary" endpoints, where an Owner-only surface broke a shared display need
Making `GET /staff` Owner-only (per the Pengaturan rule) broke PIC-name resolution across every
PIC picker/label in the `projects` module (project PIC, vendor-engagement PIC, milestone PIC, issue
PIC, activity actor) — all roles need to resolve a staff id into a name, not just Owner. Fixed with
a second, narrower endpoint, `GET /staff/summary` (`requireTenant`, any staff role) —
`{id, name, title}` only, none of `Pengguna`'s management fields (`username`/`email`/`phone`/
`isActive`). Same shape reused for `GET /vendors/summary` (any staff, includes inactive vendors so
an already-engaged-but-deactivated vendor still resolves) once the Vendor list was similarly locked
down for its own paginated management view.

### Sidebar restructuring: a collapsible "Pengaturan" group
Pengguna, Langganan, and Kategori Vendor (later joined by Timeline Default) were flat top-level
Sidebar entries before this ADR — grouped into one collapsible "Pengaturan" entry (collapsed by
default, auto-opens only when the current route lands on one of its children) specifically because
all four ended up sharing the same Owner-only visibility rule; the grouping is a direct consequence
of the access-control decision, not an independent redesign.

## Consequences
- New shared primitive: `Claims.HasRole` (`internal/shared/middleware/auth.go`).
- Every module gained or tightened role checks: `projects` (create/duplicate/PIC-reassignment
  Owner/Admin-only, `resolveProjectAccess`'s PIC scoping, dashboard endpoint Owner/Admin-only —
  previously had *zero* role check), `clients` (was entirely unrestricted before this ADR; now
  PIC-scoped read for Wedding Planner via a new `projects.Contracts.ProjectPICStaffID` method, and
  Owner/Admin-only write), `vendors`/`venues` (`requireManagerRole` write gate, added alongside
  Venue's own build in ADR-0016), `vendors` (Vendor Category corrected mid-verification from an
  over-broad fully-Owner-only lock to the same read-open/write-Owner-only split — see
  `requireOwnerRole`'s own doc comment), `staff` (`requireOwnerTenant` + the new `/staff/summary`
  carve-out), and the new `milestone-templates` resource (Owner-only end to end — no read-open
  exception needed, unlike Vendor/Venue/Category, since nothing else in the app displays a template
  name anywhere).
- Frontend: `Sidebar.tsx`'s `allowedRoles` per nav item, a `RequireRole` route wrapper
  (`protected.routes.tsx`) kept 1:1 with the Sidebar's own gating (a menu item hidden but its route
  still reachable by direct URL is exactly the class of bug a verification pass caught and fixed —
  see below), and per-page role checks for narrower in-page restrictions that don't warrant hiding
  a whole route (`canDuplicate`/`picLocked` in `ProjectHeaderCard.tsx`/`ProjectFormModal.tsx`,
  `canSeeMargin` gating the Margin/Keuntungan figure introduced later alongside Vendor Pricing
  Tier).
- Verified in two dedicated passes (the second specifically because the first missed real gaps):
  found and fixed a Dashboard endpoint with no role check at all, the `clients` module being
  entirely unrestricted, Vendor Categories over-corrected to fully Owner-only (breaking the
  Vendor form's category picker and every project's Vendor tab for non-Owner roles),
  `GlobalSearch.tsx`'s unhandled promise rejection once a fetch it made started 403ing for
  Wedding Planner, and `ProjectClientsSection.tsx`'s write-action buttons not hidden for that role.
  A later, unrelated audit (triggered by an unrelated Margin/Keuntungan feature becoming a new
  caller of the pre-existing broad Vendor/Venue read access) reconfirmed the read-open/write-
  restricted split above is a deliberate, known characteristic — not something to silently tighten
  without the user's explicit sign-off, per the "Read-open, write-restricted split" section above.
