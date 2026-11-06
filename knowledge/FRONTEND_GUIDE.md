# Frontend Guide

> Frontend conventions specific to this project.

Base rules: `.claude/rules/frontend-react.md`. This file adds ElProof-specific state-management
convention, since it's the single biggest structural gap found in the pre-integration frontend
audit (see ADR-0009).

## State management: Zustand only, one store per module (ADR-0009)

Every domain module gets exactly one store at `modules/<module>/stores/use<Module>Store.ts`,
shaped like the existing `usePlatformAdminStore.ts` — **except** `platform-admin`, which owns three
separate admin concerns that don't share state (`usePlatformAdminStore.ts` for tenants/plans/admins,
`usePaymentGatewayStore.ts` for Fase 9's gateway config, `useAppsStore.ts` for Fase 10's external App
registry): one store per *concern* within that module, not literally one file, when a module's admin
surface covers genuinely unrelated resources.

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
default. A logged-out state is a real, distinct state (no default-valid staff ID).

## HTTP

All requests go through `shared/services/http-client.ts` — one Axios instance, with a request
interceptor attaching `Authorization: Bearer <token>` from `useAuthStore`, and a response
interceptor normalizing the `{success, message, data}` / `{success:false, message, errors}` envelope
so module stores can just read `response.data.data` (or handle the error object) without
re-implementing envelope parsing per store.

## Theme / tenant branding (ADR-0012)

`apps/web/src/theme/`:
- `theme.css` — the app's default look (`--brand-navy-950/900/800` + `--color-primary-soft`, plus
  fixed semantic colors that never vary per tenant: success/warning/danger/info, background, text,
  border).
- `brandPresets.ts` — the 20 fixed presets (keys MUST match `apps/api`'s
  `domain.AllowedBrandColorPresets` exactly; hex values live only here, never on the backend).
  `applyBrandColorPreset(key)` overrides the 4 variables above on `document.documentElement`;
  `resetBrandColorPreset()` removes the overrides (reverting to `theme.css`'s own defaults, which
  are identical to the `navy` preset by design).
- `tabIdentity.ts` — `applyTabIdentity(businessName, logoUrl, consoleLabel?)` sets the browser tab
  title and a lazily-created `<link rel="icon">` (reusing the same logo object URL already fetched
  for the header, no separate favicon upload); `resetTabIdentity()` reverts both.

`shared/stores/useTenantBrandingStore.ts` follows the module-store convention above but isn't tied
to one domain module — `hydrate(consoleLabel?)` fetches `GET /tenants/me/branding` (+ the logo
blob, `GET /tenants/me/logo`, as an object URL) and calls the `apply*` functions above;
`reset()` calls the `reset*` functions and revokes the logo object URL. A module-level generation
token guards `hydrate()` against a stale response winning a race against a newer call (React
StrictMode's double-invoked mount effect, or a fast logout→login-as-a-different-tenant sequence) —
only the *latest* call's result is ever applied, and `reset()` invalidates any `hydrate()` still in
flight.

- **Called from**: `AppLayout.tsx` (`hydrate("WO Console")`) and `ClientPortalLayout.tsx`
  (`hydrate("Portal Klien")`) on mount — never from `PlatformLayout.tsx` (superadmin's own
  backoffice stays unbranded). `reset()` is called from both places a session actually ends:
  `shared/lib/auth-actions.ts`'s `logoutAndRedirect` and `http-client.ts`'s silent
  logout-on-refresh-failure.
- **`Button`'s `"neutral"` variant** (`bg-slate-800`, deliberately not tied to `--brand-navy-*`)
  exists specifically so `LoginPage` can render a primary-looking button without showing *anyone's*
  brand color, not even ElProof's own — every other `<Button>` call site still uses the default
  `primary` variant (brand-tied).
- **`LoginPage` itself is pre-auth but not always unbranded** (ADR-0015): a local
  `useDomainBranding()` hook (not `useTenantBrandingStore` — this is page-local, one-shot state, no
  session to key off yet) fetches `GET /public/branding` (Host-header-resolved, not JWT) and calls
  the *same* `applyBrandColorPreset`/`applyTabIdentity` functions above when the current domain
  matches a tenant's `customDomain`. A 404 (any other domain, including the platform's own) is
  swallowed and the page keeps its original neutral look — the `"neutral"` `Button` variant above
  is untouched either way, since it was never wired to `--brand-navy-*` in the first place.

### Anti-flash render gating (ADR-0012's Revision, ADR-0015's Revision)

Every consumer of the CSS-variable-override mechanism above shares the same race: a `bg-navy-*`-
classed element renders on mount with whatever the CSS variable currently holds, which is the
`theme.css` default until an async branding lookup resolves and overrides it — painting the wrong
identity first, then visibly swapping. Both places this app renders brand-tied chrome hold off
painting it until the real answer is known, rather than accepting the flash:

- `useTenantBrandingStore` exposes `hydrated: boolean` (false until `hydrate()` settles, reset to
  false on `reset()`); `AppLayout`/`ClientPortalLayout` render a plain white loading screen while
  it's false, the real Sidebar/header/chrome only once it's true.
- `LoginPage`'s own `useDomainBranding()` exposes an equivalent `loading` flag, but only actually
  waits for hostnames that could plausibly resolve to a tenant (a small `PLATFORM_HOSTNAMES` list
  is checked synchronously first) — the platform's own domain paints instantly, since a request
  there can never match a tenant and there's nothing worth waiting on.

Reach for this same pattern (a boolean gate on the *first* real value, not just an initial `null`)
any time a component renders something whose "empty"/default appearance is visually distinct
enough from its real one that swapping between them would read as a flash, not a normal loading
state.

## Routing

Route guards (`RequireAuth` wrapper components, one per principal type/role combination) wrap the
three *authenticated* route trees (`protected.routes.tsx`, `client-portal.routes.tsx`,
`platform.routes.tsx`) — built in Fase 7. Two more route files carry no guard, by design:
`public.routes.tsx` (`/login`) and `homepage.routes.tsx` (the public marketing site, `/homepage/*`
— frontend-only, no API calls, see the `homepage` row in `MODULE_MAP.md`).

**Permission guards within `protected.routes.tsx` (ADR-0026):** `AppLayout` loads the signed-in
staff member's grants (`GET /access/me`) into `shared/stores/usePermissionStore.ts` before rendering.
`RequirePermission` (`shared/components/RequirePermission.tsx`, props `all` = permission keys,
`allProjects`) wraps a route sub-tree and redirects to `/projects` when a grant is missing — kept 1:1
with `Sidebar.tsx`'s `requires`/`allProjects` per nav item, so a hidden menu entry is never reachable
by URL. Buttons inside shared pages use `useCan("project.delete")` / `useAllProjects()`; never branch
on the role's name. A `403 forbidden_permission` from the API makes `http-client.ts` re-fetch grants
(the role was edited); `403 role_missing` signs the user out.

**UI foundation (Fase 3b):** `Drawer`, `Toaster` + `toast.success/error/info`
(`shared/stores/useToastStore.ts`), `Skeleton`/`SkeletonRows`, `ConfirmDialog` (type-to-confirm), and
density tokens (`--density-*` in `theme.css`; layouts set `data-density="compact"` for the consoles and
`"comfortable"` for the client portal). Status labels live in `shared/lib/labels.ts` with a staff and a
client wording per status (FLOW-IDEAL §4); `StatusBadges` take `audience="client"` in the portal.
