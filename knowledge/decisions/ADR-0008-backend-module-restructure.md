# ADR-0008: Backend module boundaries (apps/api restructure)

## Status
Accepted

## Context
`apps/api/internal/modules` originally scaffolded three generic placeholder modules (`auth`,
`order`, `users`) from the `monorepo-standard` skill's bootstrap script. `order` does not correspond
to any ElProof domain concept and was never adapted.

## Decision
Backend modules are renamed/created to match the frontend-to-backend module map in PLAN.md §3:

| Module | Owns | Replaces/derived from |
|---|---|---|
| `identity` | `credentials`, `refresh_tokens` | renamed from `auth` |
| `platform` | `tenants`, `platform_admins` | new (from `platform-admin` frontend module) |
| `billing` | `subscription_plans`, `subscription_transactions` | new (from `subscription` + `platform-admin` frontend modules) |
| `staff` | `staff_members` | renamed from `users` |
| `clients` | `clients` | new |
| `vendors` | `vendors`, `vendor_categories` | new |
| `projects` | `projects` + 7 sub-entity tables | new |

`order` is deleted outright — it held no ElProof-specific code (only `.gitkeep` files and a
2-line stub). Each module keeps the standard `application/contracts/domain/events/infrastructure/
queries/presentation` layout already established by the scaffold, so the shape is consistent across
modules even though they were created at different times.

## Consequences
- `dashboard` and `client-portal` (frontend modules) do **not** get their own backend module — they
  are read-composition layers over `projects`, `vendors`, `clients`, `billing`, `platform` via those
  modules' own contracts (or a thin app-service where composition spans modules), consistent with
  "orchestrate cross-module flows via app services" in `.claude/rules/backend-modular-monolith.md`.
- Future modules follow the same seven-folder layout; deviations should be raised as a new ADR
  rather than silently diverging per module.
