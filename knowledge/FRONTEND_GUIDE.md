# Frontend Guide

> Frontend conventions specific to this project.

Base rules: `.claude/rules/frontend-react.md`. This file adds ElProof-specific state-management
convention, since it's the single biggest structural gap found in the pre-integration audit
(`PLAN.md` §2.1).

## State management: Zustand only, one store per module (ADR-0009)

Every domain module gets exactly one store at `modules/<module>/stores/use<Module>Store.ts`,
shaped like the existing `usePlatformAdminStore.ts`:

- State: the entity array(s) for that module, plus any `isLoading`/`error` flags the UI needs.
- Actions: async functions that call `httpClient` against the module's endpoints
  (`shared/services/api-endpoints.ts`), then update the store's own state from the response.
  No React Query/SWR — no automatic caching, no stale-time, no background refetch. If a page needs
  fresh data, its action re-fetches explicitly.
- Cross-module reads: `otherStore.getState()` (e.g. `usePlatformAdminStore` reads
  `useSubscriptionPlanStore.getState().plans`) — never import another module's store's internal
  types/files directly from a component.
- Shared/reference data (e.g. the subscription plan catalog) lives in `shared/stores/`, not
  duplicated per module.

Components consume state via the store's selector hook (`useXStore((s) => s.items)`), never by
reading a module-level singleton array — this is what makes data persist across navigation instead
of resetting to seed data on every mount (the bug found throughout the pre-integration audit).

## Auth

`shared/stores/useAuthStore.ts` holds the current session: access token, principal type, tenant ID,
role, display name — populated from `identity` module's login response, not from a hardcoded
default. A logged-out state is a real, distinct state (no default-valid staff ID) — see PLAN.md
Fase 1/7.

## HTTP

All requests go through `shared/services/http-client.ts` — one Axios instance, with a request
interceptor attaching `Authorization: Bearer <token>` from `useAuthStore`, and a response
interceptor normalizing the `{success, message, data}` / `{success:false, message, errors}` envelope
so module stores can just read `response.data.data` (or handle the error object) without
re-implementing envelope parsing per store.

## Routing

Route guards (`ProtectedRoute`/`RequireAuth` wrapper components, one per principal type/role
combination) wrap the three route trees (`protected.routes.tsx`, `client-portal.routes.tsx`,
`platform.routes.tsx`) — tracked as PLAN.md Fase 7, since none exist yet.
