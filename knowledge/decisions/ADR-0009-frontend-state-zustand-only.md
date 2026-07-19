# ADR-0009: Zustand-only state management, no TanStack Query

## Status
Accepted

## Context
The frontend audit (PLAN.md §2.1) found only 3 of 10 modules used a Zustand store
(`useAuthStore`, `useSubscriptionPlanStore`, `usePlatformAdminStore`); the rest used local
`useState` re-seeded from mock data on every mount, with no persistence across navigation. PLAN.md
§6 originally listed "React Query vs Zustand manual" as an open decision, since introducing
React Query would technically satisfy "server state management" while conflicting with
ADR-0003's "no TanStack libraries by default".

## Decision
The user explicitly confirmed: **all frontend state — including the 7 modules that had no store —
uses Zustand, consistently, with no exception.** No `@tanstack/react-query`, `swr`, or similar
library is introduced. Every domain module gets one store at
`modules/<module>/stores/use<Module>Store.ts` following the exact shape already established by
`usePlatformAdminStore.ts`:
- State holds the entity arrays (and any derived/loading/error flags needed).
- Actions are async functions that call `httpClient`, then update the store's state with the
  response — no separate cache layer, no automatic background refetching/stale-time semantics.
- Cross-store reads happen via `otherStore.getState()`, never deep imports of another module's
  internals — this was already the established pattern (e.g. `usePlatformAdminStore` reading
  `useSubscriptionPlanStore.getState().plans`) and continues unchanged.

## Consequences
- Manual cache invalidation: after a mutation, the action is responsible for updating/refetching
  the relevant slice of state itself — there is no automatic invalidation-by-key.
- This resolves PLAN.md §6 item 1 — it is no longer an open decision. Fase 3 and Fase 4's frontend
  work (building stores for `staff`, `clients`, `vendors`, `vendor-categories`, `projects`) must
  follow this exact pattern, not introduce a different one per module.
- **Update (Fase 9/10):** `platform-admin` ended up with three stores
  (`usePlatformAdminStore.ts`, `usePaymentGatewayStore.ts`, `useAppsStore.ts`) rather than one — this
  is not a violation of "one store per module," it's "one store per concern," since gateway config
  and the external App registry are genuinely unrelated resources that happen to live in the same
  frontend module (both are Platform Console admin surfaces over the `payment` backend module). The
  "no exception" language above still means no React Query/SWR and no per-module deviation from the
  fetch-then-set pattern — it was never meant to force unrelated resources into one store file.
