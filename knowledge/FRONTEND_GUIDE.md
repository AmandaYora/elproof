# Frontend Guide

> Frontend conventions specific to this project.

Base rules: `.claude/rules/frontend-react.md`. This file adds ElProof-specific state-management
convention, since it's the single biggest structural gap found in the pre-integration frontend
audit (see ADR-0009).

## UI kit and conventions (ADR-0037)

Read this before touching a screen. Layers, from the bottom:

| Layer | Where | Rule |
|---|---|---|
| Tokens | `src/theme/theme.css` | Semantic names only (`bg-card`, `text-muted-foreground`, `text-danger-text`, `border-input`). No hex or palette colours in a component; text in a status colour uses the `-text` token (4.5:1). One palette for every surface, no per-tenant colour (ADR-0039); the public site only swaps its surface tokens in `marketing.css`. A control that hides the browser outline adds `focus-visible:outline-solid` with `outline-2` (Tailwind 4 `outline-none` sets `--tw-outline-style: none`). |
| Primitives | `src/shared/components/kit/` | shadcn copies on **Base UI**. Add one with the shadcn CLI only when a screen needs it; delete what nothing imports. |
| App components | `src/shared/components/ui/` | `Button`, `Input`, `Field`, `Modal`, `Drawer`, `ConfirmDialog`, `Badge`, `Table`, `IconActionButton`, `TrendBarChart`, `ProofStamp`… The API screens use. Domain-agnostic. |
| Forms | `src/shared/forms/` | `useZodForm(schema, …)` + `<Field name="…">` + `FormErrorSummary`; types with `z.infer`; a 422's `errors` keys land on the same fields; the first error takes focus. |
| Lists | `src/shared/data-table/` | `useListState` (page, sort, search, filters in the URL) + `DataTable`. The server sorts, filters and pages (ADR-0035); never load `all=true` to filter in the browser. |
| Layout | `src/shared/layouts/` | `AppSidebar`/`Sidebar`/`Topbar`, `Breadcrumbs`, ⌘K palette. Sheet, palette dialog, tooltips on the rail, conflict dialog and toaster load on demand. |

**Rules that a test enforces:** wording (`npm run check:terms`), accessibility (axe strict in CI), the entry bundle and per-screen API-call ceilings (`perf-budget.json`),
and the behaviour in `e2e/tests/*.spec.ts`. Run from `apps/web`:

```bash
npx tsc -b && npm run check:terms && npm run build && npm run budget   # the fast gates, no browser
npm run e2e                                                             # snapshots (macOS), axe, behaviour, writes
npm run e2e:perf && npm run budget:routes                               # per-screen calls and first-load JS
```

**Words.** One term per concept, Indonesian (`DESIGN.md` §7): Proyek, Klien, Tagihan, Dokumen/bukti, Kendala, Jadwal, Tahapan, Tim. API enum values stay English; show them through
`shared/lib/labels.ts` (`MILESTONE_STATUS_LABELS.staff[...]`, `evidenceTypeLabel`). An English word that really must stay in a string gets `// ui-terms-ok` and a reason.

**Overlays and focus.** Use `Modal`/`Drawer`/`ConfirmDialog`, never a hand-made fixed div: focus goes in and returns, Escape closes only the top one, background does not scroll.
Row actions that are only an icon are `IconActionButton` (named, with a tooltip); never a control inside a control (a `<button>` inside a row `<button>`).

**Motion.** Only CSS transitions/animations through Tailwind (`motion-safe:` for the decorative); `theme.css` removes durations under `prefers-reduced-motion`, and an e2e test checks
it. Do not add JS-driven animation.

**Deleting.** A routine delete is a soft delete with "Urungkan" (`shared/lib/undo-delete.ts`); only a verified payment asks first.

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

### Saving over someone else's change (ADR-0036)

Items of the editable project resources carry `updatedAt`. A form keeps the `updatedAt` it was **opened** with (not the one the store holds when the person presses save, which
a background refresh may have replaced) and passes it as the last argument of the store's update method; the store sends it as `If-Match` (`shared/lib/concurrency.ts: ifMatch`).
Inline edits (a status dropdown) send only the changed field. A 409 is handled in one place: the HTTP client opens the global "Data diubah orang lain" dialog
(`useConflictStore`, `ConflictHost`, loaded on demand) and the page that owns the data registers how to reload it (`registerRevalidator`, e.g. `ProjectDetailLayout`), so "Muat ulang"
rebuilds the page and any form still holding old values. State changes are explicit (`setProjectArchived(id, true)`), never a flip of whatever is current.

## Theme / tenant branding (ADR-0012, ADR-0039)

`apps/web/src/theme/`:
- `theme.css` — the one palette for every surface (plum `--primary`/`--ring`, seal gold `--seal*`,
  plum-tinted neutrals, status colours; values and contrast in `DESIGN.md` §4.1). Nothing overrides
  it at runtime: there is no per-tenant colour.
- `marketing.css` — only the public surface's differences (white page, blush bands) under
  `:root[data-surface="marketing"]`, plus its scoped motion (ADR-0038).
- `tabIdentity.ts` — `applyTabIdentity(businessName, logoUrl, consoleLabel?)` sets the browser tab
  title and a lazily-created `<link rel="icon">` (reusing the same logo object URL already fetched
  for the header, no separate favicon upload); `resetTabIdentity()` reverts both.

`shared/stores/useTenantBrandingStore.ts` follows the module-store convention above but isn't tied
to one domain module — `hydrate(consoleLabel?)` fetches `GET /tenants/me/branding` (+ the logo
blob, `GET /tenants/me/logo`, as an object URL) and calls `applyTabIdentity`; `reset()` calls
`resetTabIdentity` and revokes the logo object URL. A module-level generation
token guards `hydrate()` against a stale response winning a race against a newer call (React
StrictMode's double-invoked mount effect, or a fast logout→login-as-a-different-tenant sequence) —
only the *latest* call's result is ever applied, and `reset()` invalidates any `hydrate()` still in
flight.

- **Called from**: `AppLayout.tsx` (`hydrate("WO Console")`) and `ClientPortalLayout.tsx`
  (`hydrate("Portal Klien")`) on mount — never from `PlatformLayout.tsx` (superadmin's own
  backoffice stays unbranded). `reset()` is called from both places a session actually ends:
  `shared/lib/auth-actions.ts`'s `logoutAndRedirect` and `http-client.ts`'s silent
  logout-on-refresh-failure.
- **`LoginPage`** uses the public surface (`AuthSplitLayout`, ADR-0038) and the same palette; no tenant is knowable before auth (per-tenant custom domains were removed, ADR-0030).

### Anti-flash render gating (ADR-0012's Revision)

The header and sidebar show the tenant's logo and name, which arrive from an async lookup; painting
the chrome before it resolves would show the text fallback first, then visibly swap in the logo. The places this app renders brand-tied chrome hold off
painting it until the real answer is known, rather than accepting the flash:

- `useTenantBrandingStore` exposes `hydrated: boolean` (false until `hydrate()` settles, reset to
  false on `reset()`); `AppLayout`/`ClientPortalLayout` render a plain white loading screen while
  it's false, the real Sidebar/header/chrome only once it's true.

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

**Forms in dialogs (Fase 4a):** `Modal` and `Drawer` share `useDialogBehavior` (focus moves in, Tab is
trapped, Escape closes only the top-most dialog, scroll lock, focus returns to the opener). A form
inside a dialog follows one pattern: its state lives in an inner component that exists only while the
dialog is open (so every opening starts from current data), `onSubmit` returns a Promise, the dialog
shows `busy` (close disabled, button "Menyimpan...") and `error` (message inside the dialog, input
kept), and only a successful save closes it and shows a `toast.success`. Field errors use
`<Field error=...>` with `aria-invalid` on the input (red border, first invalid field is focused).
`Select`/`Combobox` handle Escape themselves so the dialog behind stays open. `ActionMenu` is the
"⋯" list for secondary actions.

**Project page and dashboard (Fase 4b/4c):** `ProjectDetailLayout` is header + full-width tabs +
(content | `ProjectSidePanel`). The dashboard is action-first (`OverdueHero`, `DueList`). Creating a
project is the two-step page `ProjectCreatePage` (`/projects/baru`); the installment list is the
shared `PaymentTermsEditor` (also used by the schedule drawer on the Keuangan tab). Pages that load
data separate three states: skeleton while loading, `LoadError` with "Coba lagi" when the request
failed (a network failure never looks like "empty" and never redirects away), and an `EmptyState`
that says what to do. Touch devices (`pointer-coarse`) get 44px targets on buttons, inputs and icon
actions; the closed mobile sidebar is `inert`. `IconActionButton` disappears in a read-only workspace
unless it is marked `readOnly` (viewing actions).

**Client Portal (Fase 5):** a phone-first app. `ClientPortalLayout` loads project, grants, branding and
the WO contact (`useContactStore`, `GET /tenants/me/contact`), then fetches what the role may see; a
bottom tab bar on phones (Beranda, Jadwal, Pembayaran, Vendor, Lainnya) becomes a top bar from `md` up.
Beranda (`HomeTabPage`) is hero (countdown in the display serif), "Yang perlu Anda lakukan" (the only
red in the portal, shown only for a payment past due), next payment with the account to pay into and a
WhatsApp confirmation button, four-step progress, latest news, contact. Client-audience badges turn
`danger` into amber (`toneFor`). Errors never log the client out: not signed in is the HTTP client's job,
"no project" and "failed to load" are their own screens with a retry. Roles drive everything
(Keluarga has no Pembayaran tab, no amounts, no bank details; the API refuses them too). It is a
PWA (`manifest.webmanifest`, `sw.js` keeps only an offline page, never API data; install help on Lainnya).
Staff roles limited to their own projects land on `MyProjectsPage` ("Proyek saya") at Beranda.

**Read-only workspace (Fase 4a):** while the subscription gives only read access (`access !== "full"`),
`useCan(permission)` answers false for every permission that changes data (anything not ending in
`.view`, except `subscription.manage`), so write buttons disappear on every page. Routes and the
sidebar use `hasPermission` directly and stay reachable. `useWriteLocked()` is for write controls that
have no permission of their own. The API still refuses writes with `subscription_read_only`.

## Platform Console (Fase 6, ADR-0030)

- **Navigation** is grouped (Pelanggan, Produk, Sistem) in `PlatformLayout`; entries a role cannot use
  are hidden through `lib/capabilities.ts` (`usePlatformCan`), which mirrors the API's role policy in
  `cmd/server/platform_scope.go`. The API stays the real gate.
- **Ringkasan** reads `GET /platform/metrics` and `GET /platform/trends?period=` (`useConsoleStore`); the
  console never loads every tenant or transaction. List pages name their rows' tenants with
  `fetchTenantNames(ids)`, and the layout reads only the signed-in admin (`GET /platform-admins/{id}`).
- **Global search** (`shared/layouts/GlobalSearch`) debounces, aborts stale requests and asks the paginated
  project, vendor and client endpoints for four hits each; one of them failing (a Wedding Planner gets
  403 for tenant-wide clients) never hides the others.
- **Evidence files** open through `GET .../evidence/{id}/link` (direct storage URL, no bytes through the
  API) and fall back to the authenticated `/file` blob on 501; the download link is requested on click
  because it only lives two minutes.
- **Tenant 360°** is `/platform/tenant/:id/:tab?` (`TenantDetailPage` + `components/tenant-detail/*`).
  The edit/activate/freeze/re-invite actions and their dialogs are the shared `useTenantActions` hook, used by
  the list and the detail page alike. Per-tenant data is returned by `useConsoleStore` calls, not cached.
- **Forms** keep the module pattern: Zod schemas in `schemas/console.schema.ts`, field errors from
  `safeParse`, server messages through `getApiErrorMessage`.
- **Support session** (`shared/lib/impersonation.ts`): `enterSupportSession` stores the admin's own session in
  `session.impersonation.returnTo`, signs the browser in as the Owner (no refresh token) and reloads;
  `endSupportSession` restores it (without its refresh token, so a script in the tenant's page cannot read one) and tells the API; the banner also ends the session when the hour is up. `useCan`/`useWriteLocked` treat a support session as
  read-only, the http client turns `impersonation_read_only` into a toast and, when the token lapses, returns
  the admin to their own session instead of refreshing. `SupportSessionBanner` is always visible in the WO
  Console during a session.
- **WO side**: `AnnouncementBanner` (dismissible, remembered per browser) and `SupportAccessCard` in
  Profil Usaha (grant/revoke consent, session history).
- **Per-page meta**: `usePageMeta({title, description, noindex})` sets the tab title and search/share tags and
  restores them on unmount; marketing and auth pages use it.
- **QR code**: `shared/components/ui/QrCode` draws in the browser with the `qrcode` package; the text never
  leaves the page. **Receipt**: `/langganan/kuitansi/:id` is a plain page; `print:` utilities hide the console
  chrome and `window.print()` is the "Simpan sebagai PDF".
