# ADR-0004: Multi-tenant data scoping strategy

## Status
Accepted

## Context
The WO Console frontend was built against a single hardcoded tenant (see `mock/seed.ts`,
`TODAY`/`CURRENT_PLAN_ID` constants). A real backend must serve many tenants (WO businesses)
without letting one tenant's data leak into another's queries, while keeping the Platform Console
(which manages tenants themselves) outside any single tenant's scope.

## Decision
- Every tenant-owned table (`staff_members`, `clients`, `vendors`, `vendor_categories`, `projects`,
  and all `projects` sub-entities) carries a `tenant_id BIGINT UNSIGNED NOT NULL` column.
- `tenant_id` is never accepted from client input on writes — it is always taken from the
  authenticated principal's JWT claim (`tenant_id`), injected by the `identity` module's session
  issuance and read by the `shared` auth middleware.
- Every repository query for a tenant-owned table filters by `tenant_id` unconditionally — there is
  no "admin override" query path in tenant-scoped modules. Cross-tenant operations (e.g. Platform
  Console activating a tenant's subscription) happen only through the `platform`/`billing` modules,
  which are tenant-agnostic by design (they store `tenant_id` as a plain foreign-key-by-value on
  `tenants`/`subscription_transactions`, not as a scoping filter on themselves).
- Platform Console principals (`platform_admin`) carry no `tenant_id` (null) and cannot authenticate
  against tenant-scoped modules' endpoints.

## Consequences
- Every new tenant-scoped table must remember the `tenant_id` column and the filter — this is a
  manual discipline, not enforced by a database feature (no MySQL row-level security). Code review
  / a lint rule on repository query builders is the mitigation.
- No cross-module joins are affected, since `tenant_id` scoping happens within a single module's own
  tables, consistent with `.claude/rules/database.md`.
