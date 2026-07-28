# PLAN — Per-tenant branding (logo + preset brand color)

Status: **Implemented.** Built, backend build/vet clean, frontend `tsc -b`/`vite build` clean,
migration 000017 applied and interactively verified end-to-end (Playwright) for the platform-admin
edit flow and WO Console: color-preset save, logo upload/display, Sidebar re-theming, Platform
Console staying unbranded, and CSS-variable reset on logout with no bleed into the next session.
Client Portal was **not** independently exercised in the browser (no test client/project credential
was set up) — it reuses the exact same `useTenantBrandingStore`/`ClientPortalLayout` wiring already
covered by the frontend build, so risk is low, but this is a gap versus a full end-to-end check.
Kept here as the design record — see §9 for what changed vs. the original design.

**Permanent documentation now lives in the knowledge base** (this file remains the session's
narrative working record, not the source of truth going forward): `knowledge/decisions/
ADR-0012-tenant-branding.md` (the architectural decision + rationale), `knowledge/
MODULE_PLATFORM.md` §6 (the `platform` module deep-dive, branding section), `docs/DB_SCHEMA.md`/
`docs/API_CONTRACT.md` (columns/endpoints), `knowledge/FRONTEND_GUIDE.md` ("Theme / tenant
branding" section), and `knowledge/decisions/ADR-0006-file-storage-strategy.md`'s revision (logo
storage as a second consumer of the shared object-storage utility).

## 0. Goal

Today every tenant sees identical "ElProof" branding: a hardcoded `APP_NAME` text string
(`apps/web/src/shared/constants/brand.ts:1`) and one static navy color scheme
(`apps/web/src/theme/theme.css:1-43`). The ask: each tenant gets its **own logo** and its **own
brand color**, so the tenant's own staff (WO Console) and the tenant's own clients (Client Portal)
see that tenant's identity instead of ElProof's. Color choice is constrained to **15 fixed
presets** (no free-form hex/color-picker), and the logo image must still live in the existing
S3-compatible object storage — not inline base64 in the DB, not a public bucket URL.

### Scope decision: which layouts get tenant branding

- `apps/web/src/shared/layouts/Sidebar.tsx` (WO Console — the tenant's own staff working area) —
  **branded**.
- `apps/web/src/.../ClientPortalLayout.tsx` (the tenant's own clients, e.g. the bride/groom or
  B2B customer viewing proofs/milestones) — **branded**. This is the actual "tidak semua ElProof"
  case: a tenant's client should see the tenant's identity, not ElProof's.
- `apps/web/src/.../PlatformLayout.tsx` (ElProof's own superadmin backoffice managing *all*
  tenants) — **stays fixed ElProof branding**. Superadmin isn't "inside" any single tenant, so
  there's no one tenant color to apply here.
- `MarketingLayout.tsx` and `LoginPage.tsx` (public, pre-authentication, tenant unknown until
  credentials resolve) — **stays fixed ElProof branding**. There's no tenant-specific subdomain/URL
  in this app (login resolves the tenant from username/email, per `d148abf`), so there's no way to
  show tenant branding before login without a much bigger change (subdomain routing) that wasn't
  asked for — flagged as out of scope below.

## 1. Current state (confirmed by code inspection)

- `apps/api/internal/modules/platform/domain/tenant.go:24-40` — `Tenant` struct has no
  logo/color field at all (`BusinessName, OwnerName, Username, Email, Phone, City, JoinedAt,
  PlanID, SubscriptionStatus, SubscriptionExpiresAt, IsSuspended, LastCredentialResetAt,
  CreatedAt, UpdatedAt`).
- `apps/api/migrations/000005_create_platform_tables.up.sql` creates `tenants` with the same
  column set. Highest existing migration is `000016_add_unique_email_to_credentials` → next
  number is **000017**.
- `apps/web/src/theme/theme.css:1-43` — only **3 CSS variables** actually carry brand identity:
  `--brand-navy-950/900/800` (plus `--color-primary-soft: #e3ebf3` as a 4th, lighter tint used for
  soft/tinted backgrounds). Everything else (`--color-success/warning/danger/info`,
  background/text/border) is a fixed semantic/status color, not brand identity, and must **not**
  vary per tenant (status colors need to stay consistent and accessible platform-wide).
- These 3 vars are consumed by Tailwind 4's `@theme inline` block (`theme.css:45-48`) as the
  `navy-950/900/800` utility classes, used **194 times across 46 `.tsx` files**
  (`bg-navy-900`, `hover:bg-navy-800`, etc. — e.g. `Button.tsx:14`, `Sidebar.tsx`). This is the key
  finding that makes this tractable: **no component refactor is needed.** Overriding the 3 CSS
  variable values at runtime (e.g. on `document.documentElement`) re-colors every one of those 46
  files automatically, because Tailwind resolves the class to `var(--brand-navy-900)` at render
  time, not to a baked-in hex value.
- Object storage: `apps/api/internal/shared/storage/storage.go` — S3-compatible (MinIO client),
  `Save`/`Open`/`Delete`/`BuildKey(tenantID, projectID, category, filename)`. `Open` returns a raw
  `io.ReadCloser` (`storage.go:54-60`), **not** a signed URL — every consumer must proxy bytes
  through a Go handler.
- Existing precedent to mirror (project evidence uploads):
  - Upload: **base64-JSON body**, not multipart — `evidence_endpoints.go:26-54`
    (`evidenceUploadBody{base64Data, mimeType, fileName}`) → `evidence_service.go:65-105`
    (base64-decode, size cap `maxDecodedSize` = 15MB, `storage.Save(ctx, key, bytes, mimeType)`).
    Persisted DB column: `storage_path VARCHAR(500) NOT NULL`
    (`000008_create_project_tables.up.sql:115`), Go field `Evidence.StoragePath`
    (`domain/evidence.go:35`).
  - Download: **streamed through Go**, not a public URL — `GET .../evidence/{id}/file`
    (`handler.go:150-151` → `evidence_endpoints.go:56-72` → `evidence_service.go:110-118` →
    `storage.Open` → `io.Copy(w, reader)`). Frontend fetches it as a `blob` (axios
    `responseType: "blob"`) and renders an object URL — a bare `<img src=".../file">` can't work
    because the endpoint requires auth (`EvidenceViewerModal.tsx:24-27,41,118-123`).
  - The tenant logo must follow the **same two patterns**: base64-JSON upload, byte-proxy download.
- `apps/api/internal/modules/platform/contracts/` exists but is **currently empty** — no other
  module reads tenant data today, so no contract changes are needed for this feature. (Documented
  here so a future contributor doesn't wonder why it's untouched.)
- Frontend session store `apps/web/src/shared/stores/useAuthStore.ts:5-13` (`AuthSession`) has no
  branding fields; there is no `/me` endpoint — login response is the only hydration point today.
- Platform-admin tenant management UI: `apps/web/src/modules/platform-admin/components/
  TenantFormModal.tsx` (create/edit form) and `.../schemas/tenant.schema.ts` — this is where a
  superadmin sets up a tenant, so this is where logo + color-preset get configured (see open
  question in §6 about whether tenant self-service editing is wanted later).

## 2. Data model

New migration `apps/api/migrations/000017_add_tenant_branding.up.sql`:

```sql
ALTER TABLE tenants
  ADD COLUMN brand_color_preset VARCHAR(20) NOT NULL DEFAULT 'navy',
  ADD COLUMN logo_storage_path VARCHAR(500) NULL;
```

`.down.sql` drops both columns.

- `brand_color_preset` defaults to `'navy'` for every existing tenant — this reproduces the
  *exact current look* (same hex values as today), so shipping this migration alone changes
  nothing visually until a superadmin picks a different preset.
- `logo_storage_path` is nullable — no logo configured means "fall back to the `APP_NAME` text
  header", same as today.
- Validation of `brand_color_preset` against the allowed 15 values happens in the **application
  layer** (`platform` module), not a DB CHECK constraint — mirrors how this codebase already
  favors simple `VARCHAR` + app-level validation over DB-level enums.
- `apps/api/internal/modules/platform/domain/tenant.go`: add
  `BrandColorPreset string` and `LogoStoragePath *string` to the `Tenant` struct.
- New file `apps/api/internal/modules/platform/domain/brand_preset.go`:
  `var AllowedBrandColorPresets = []string{"navy","gold","orange","blue","emerald","red","purple","teal","indigo","rose","cyan","fuchsia","lime","slate","stone"}`
  + `func IsValidBrandColorPreset(s string) bool`. Single source of truth the service layer
  validates against.

## 3. The 15 presets

Proposal below — **open to adjustment**, this is a first pass, not a locked palette. Each preset
supplies the same 4 shades the app already uses for brand color today (`950`, `900`, `800`,
`soft`). To avoid inventing new colors from scratch (and to get accessible, professionally-tuned
shades for free), 14 of the 15 are sourced directly from Tailwind's own default palette families;
`navy` is kept as-is (today's exact custom values) so it can stay the default with zero visual
change.

| # | Key       | Label (id)      | 950       | 900       | 800       | soft (tint) |
|---|-----------|------------------|-----------|-----------|-----------|-------------|
| 1 | `navy`    | Navy (default)   | `#172741` | `#1e3a5f` | `#24476f` | `#e3ebf3`   |
| 2 | `gold`    | Emas             | `#451a03` | `#78350f` | `#92400e` | `#fef3c7`   |
| 3 | `orange`  | Oranye           | `#431407` | `#7c2d12` | `#9a3412` | `#ffedd5`   |
| 4 | `blue`    | Biru             | `#172554` | `#1e3a8a` | `#1e40af` | `#dbeafe`   |
| 5 | `emerald` | Hijau Emerald    | `#022c22` | `#064e3b` | `#065f46` | `#d1fae5`   |
| 6 | `red`     | Merah            | `#450a0a` | `#7f1d1d` | `#991b1b` | `#fee2e2`   |
| 7 | `purple`  | Ungu             | `#2e1065` | `#4c1d95` | `#5b21b6` | `#ede9fe`   |
| 8 | `teal`    | Tosca            | `#042f2c` | `#134e4a` | `#115e59` | `#ccfbf1`   |
| 9 | `indigo`  | Indigo           | `#1e1b4b` | `#312e81` | `#3730a3` | `#e0e7ff`   |
|10 | `rose`    | Merah Muda       | `#4c0519` | `#881337` | `#9f1239` | `#ffe4e6`   |
|11 | `cyan`    | Cyan             | `#083344` | `#164e63` | `#155e75` | `#cffafe`   |
|12 | `fuchsia` | Fuchsia          | `#4a044e` | `#701a75` | `#86198f` | `#fae8ff`   |
|13 | `lime`    | Hijau Lime       | `#1a2e05` | `#365314` | `#3f6212` | `#ecfccb`   |
|14 | `slate`   | Abu Grafit       | `#020617` | `#0f172a` | `#1e293b` | `#f1f5f9`   |
|15 | `stone`   | Cokelat Hangat   | `#0c0a09` | `#1c1917` | `#292524` | `#f5f5f4`   |

`key` is what's stored in `brand_color_preset` and sent over the API; `Label` is what's shown in
the picker UI.

## 4. Backend changes

- **Domain & migration**: as in §2.
- **`tenant_handler.go`**:
  - `tenantResponse` (currently lines 28-42): add `brandColorPreset string` and `hasLogo bool`
    (a derived boolean — never leak the raw `logo_storage_path` object key to the client, same
    reasoning evidence responses don't leak internal storage details either).
  - `updateTenantBody` (currently lines 226-232): add optional `BrandColorPreset *string`,
    validated against `domain.IsValidBrandColorPreset` in the service layer — reject with the
    standard `{success:false,...}` error shape (`api-standard.md`) if an unknown key is sent.
  - `registerTenantBody`: no branding fields at creation time — a new tenant starts on the
    `navy` default and a superadmin configures branding afterward via update, keeping the create
    form unchanged.
- **New logo endpoints** (mirroring evidence's upload/download pair exactly):
  - `PUT /api/v1/tenants/{id}/logo` — body `{base64Data, mimeType, fileName}`. Cap decoded size
    much lower than evidence's 15MB — propose **2MB**, since this is a small header/sidebar
    image, not a document. Restrict `mimeType` to `image/png`, `image/jpeg`, `image/webp` —
    deliberately **excluding `image/svg+xml`** (an SVG can embed `<script>`/event handlers;
    sanitizing it properly is extra scope not worth it for a logo upload). Stores via
    `storage.BuildKey(tenantID, 0, "branding", fileName)` (or similar — `0`/`"branding"` in place
    of the project-scoped fields, since a logo isn't tied to a project) and
    `storage.Save(...)`, then updates `logo_storage_path`.
  - `GET /api/v1/tenants/{id}/logo` — platform-admin viewing any tenant's logo by ID, streamed
    the same way as `downloadEvidence`.
  - `GET /api/v1/tenants/me/logo` and `GET /api/v1/tenants/me/branding` — for an authenticated
    WO Console / Client Portal session to fetch **its own** tenant's branding. Tenant ID comes
    from the same session/JWT context every other tenant-scoped query already uses (no new
    resolution mechanism). `.../me/branding` returns `{brandColorPreset, hasLogo}` (cheap JSON);
    `.../me/logo` streams bytes only when `hasLogo` is true.
- No `platform/contracts` changes — nothing outside the `platform` module needs this data today
  (confirmed empty `contracts/` dir, §1). If a future module needs it (e.g. PDF report
  generation stamping the tenant's logo), add it then.

## 5. Frontend changes

- New `apps/web/src/theme/brandPresets.ts` — `BRAND_COLOR_PRESETS` const object keyed by the 15
  preset keys from §3 (single source of truth for swatch colors in the picker UI and for the
  runtime CSS-variable override), plus `BrandColorPresetKey` type.
- New `apps/web/src/shared/stores/useTenantBrandingStore.ts` (Zustand, sibling to
  `useAuthStore.ts`): `{ colorPreset, logoUrl, hydrated, hydrate(), reset() }`.
  - `hydrate()`: calls `GET .../tenants/me/branding`, applies the preset by calling
    `document.documentElement.style.setProperty('--brand-navy-950', hex)` (and `-900`, `-800`,
    and `--color-primary-soft`) for the 4 values in §3 — **reusing the existing variable names**,
    so none of the 46 files using `navy-*` Tailwind classes need to change. If `hasLogo`, follow
    up with a blob-fetch of `.../tenants/me/logo` (same pattern as
    `EvidenceViewerModal.tsx:41`) and store the object URL.
  - `reset()`: clears the inline style overrides (reverting to `theme.css` defaults) and revokes
    the object URL — called on logout.
- `LoginPage.tsx`: call `useTenantBrandingStore.getState().hydrate()` right after
  `useAuthStore.getState().login(session)` — but only when the resolved session has a
  `tenantId` (superadmin/platform sessions have none, per `AuthSession.tenantId` being optional —
  `useAuthStore.ts:5-13` — and stay on default ElProof branding).
- `Sidebar.tsx` (WO Console) and `ClientPortalLayout.tsx`: replace the current
  `<span>{APP_NAME}</span>`-only header with `logoUrl ? <img src={logoUrl} alt={businessName}
  className="h-8 w-auto" /> : <span>{APP_NAME}</span>` reading from the branding store.
  `PlatformLayout.tsx` / `MarketingLayout.tsx` / `LoginPage.tsx` are **not** touched (§0 scope
  decision).
- `apps/web/src/modules/platform-admin/components/TenantFormModal.tsx`: add a color-preset
  picker (15-swatch grid, single-select) and a logo file input (client-side validate
  size/MIME before base64-encoding and `PUT`-ing to `.../tenants/{id}/logo`, same client-side
  flow the evidence upload UI already uses for its base64 encode step).
- `apps/web/src/modules/platform-admin/schemas/tenant.schema.ts`: add
  `brandColorPreset: z.enum([...15 keys])` to the update schema.

## 6. Open questions before implementation

1. **Who configures branding — superadmin only, or tenant self-service too?** This plan scopes
   it to the platform-admin `TenantFormModal` only (superadmin sets it up for the tenant), since
   that's the only tenant-editing UI that exists today. If tenants should be able to change their
   own logo/color from inside WO Console, that's a new settings page — straightforward to add
   later on top of the same backend endpoints, but flagged here since it changes the frontend
   scope in §5.
2. **Exact 15 colors/hex values** (§3) are a first proposal, not final — happy to adjust names,
   hues, or swap any of the 14 Tailwind-sourced families for different ones.
3. **Logo constraints** — proposed 2MB cap, PNG/JPEG/WebP only, no aspect-ratio/dimension
   enforcement (rendered at a fixed `h-8` in the sidebar regardless of source size). Confirm this
   is enough, or if a minimum/recommended dimension should be enforced server-side.

## 7. Explicitly out of scope

- Free-form/custom hex color picker — presets only, per the request.
- Per-tenant subdomains or pre-login branding on the marketing site/login screen.
- `platform/contracts` additions for other modules to consume branding (nothing needs it yet).
- SVG logo support (XSS-sanitization cost not worth it for this feature).
- Tenant self-service branding settings page (see open question 1 — can follow later on the same
  backend).

## 8. Testing plan (once implemented)

1. Migration up/down round-trip; confirm existing tenants read back `brand_color_preset='navy'`
   and unchanged visuals (no CSS var override applied when preset is `navy` and no logo set).
2. Backend: `PUT .../tenants/{id}` with an invalid `brandColorPreset` value → rejected with the
   standard error shape; with a valid one → persisted and reflected in the next `GET`.
3. Backend: `PUT .../tenants/{id}/logo` with an oversized file / disallowed MIME type → rejected;
   valid PNG/JPEG/WebP → stored, `hasLogo` flips to `true`, `GET .../logo` streams the same bytes
   back.
4. Frontend: log in as a tenant with a non-default preset + logo configured — confirm WO Console
   Sidebar and Client Portal both show the tenant's logo and every `bg-navy-*`/`hover:bg-navy-*`
   surface (buttons, sidebar background, etc.) re-colors to the chosen preset, while
   success/warning/danger/info colors stay unchanged.
5. Log in as a tenant with no branding configured — confirm identical appearance to today (navy +
   `APP_NAME` text).
6. Confirm PlatformLayout (superadmin) and the public marketing site/login page are unaffected
   regardless of any tenant's configured branding.
7. Log out — confirm CSS var overrides reset and the object URL is revoked (no stale branding
   leaking into the next login on a shared/kiosk browser).

## 9. Implementation notes (what changed vs. the design above)

- **Self-service branding read**: didn't need a brand-new endpoint pattern — `platform`'s
  `tenant_handler.go` already had a `/tenants/me` self-service GET (Owner-only, JWT-scoped via
  `middleware.FromContext` + `claims.TenantIDInt()`). Added `/tenants/me/branding` and
  `/tenants/me/logo` alongside it as siblings, but deliberately **open to any tenant-scoped
  principal** (any staff role, or client), not just Owner — branding has to render for everyone in
  WO Console/Client Portal, unlike `/tenants/me`'s subscription data which stays Owner-only.
- **Logo upload/download shape**: confirmed and reused evidence's exact pattern — base64-JSON
  body (no multipart), byte-proxied download (`storage.Open` → `io.Copy`), same `ObjectStorage`
  interface shape re-declared locally in `platform/application` (a module can't import another
  module's application layer, so this is intentional duplication, not an oversight).
- **`storageClient` wiring**: `platform.NewModule` now takes `*storage.Client` directly as a
  constructor argument (no two-phase `Set...` wiring needed) — `main.go` already constructs
  `storageClient` before `platformModule`, so this was a same-turn dependency, unlike the
  vendors/clients cross-module cases that need two-phase wiring.
- **Real bug caught by interactive verification, fixed**: `internal/shared/middleware/cors.go`
  hardcoded `Access-Control-Allow-Methods: GET, POST, PATCH, DELETE, OPTIONS` — no endpoint in this
  codebase had ever used `PUT` before the logo-upload endpoint, so the dev-mode CORS preflight
  silently blocked it (`405`-equivalent browser-side rejection, not caught by `go build`/`tsc`).
  Added `PUT` to the allow-list. Worth a mental note: any *next* new HTTP method introduced here
  will hit the same wall until this list is method-agnostic.
- **Color application mechanism**: exactly as designed — `applyBrandColorPreset` overrides
  `--brand-navy-950/900/800` + `--color-primary-soft` via `document.documentElement.style`,
  reusing the same variable names `theme.css` already defines, so none of the 46 files using
  `bg-navy-*`/`hover:bg-navy-*` Tailwind classes needed touching. Verified live: WO Console Sidebar
  correctly re-colored to the "gold" preset, Platform Console stayed on default navy, and logging
  out reset the variables before the next login (no cross-tenant bleed).

## 10. Post-implementation review pass

Ran two independent adversarial reviews (backend + frontend) over the full diff after the
interactive verification above already passed. Five real bugs found and fixed, re-verified
end-to-end afterward (still all green):

- `TenantService.Update` required `brandColorPreset` on *every* edit, rejecting the whole request
  (including unrelated fields) if it was missing — now optional; an empty value keeps the tenant's
  current preset instead of erroring, matching how no other field on this endpoint is required.
- `TenantService.Register` never set `BrandColorPreset` on the tenant it returns, so `POST
  /tenants`'s immediate response reported `""` instead of the `'navy'` the DB actually defaults
  to — now set explicitly via a new `domain.DefaultBrandColorPreset` constant.
- `TenantFormModal.tsx`'s `handleLogoChange` leaked an object URL every time a different logo file
  was re-selected in the same modal session (only the server-fetched preview's URL was tracked for
  cleanup) — now tracked in a ref and revoked before creating the next one.
- `useTenantBrandingStore.hydrate()` had no guard against a stale response winning a race against
  a newer call (React StrictMode's double-invoked mount effect, or a fast logout→login-as-a-
  different-tenant sequence) — added a generation-token guard so only the *latest* call's result
  is ever applied, and `reset()` (logout) invalidates any hydrate() still in flight.
- Closing the tenant edit modal while a logo upload was still in flight let its error/loading
  state update land on an unmounted component, silently swallowing an upload failure — guarded
  with a mounted ref.

Two more were flagged and deliberately left alone: `streamLogo`'s ignored `io.Copy` error / no
explicit `Content-Type` header, and no base64 size cap check ordering — both exact pre-existing
patterns already present in evidence upload/download, not regressions introduced by this feature.

## 11. Name fallback gap (caught by user follow-up, fixed)

§5's original design fell back to the literal `APP_NAME` ("ElProof") text whenever a tenant had no
logo configured yet — so an unbranded tenant's own staff/clients would still see "ElProof" in the
Sidebar/Client Portal header, defeating the point of the feature for any tenant that hasn't gotten
around to uploading a logo. Fixed: `brandingResponse`/`toBrandingResponse`
(`tenant_handler.go`) now also return `businessName`; `useTenantBrandingStore` exposes it; `Sidebar.tsx`
and `ClientPortalLayout.tsx` fall back to the tenant's own `businessName` instead of `APP_NAME`
when no logo is set — `APP_NAME` is now only ever shown for a session with no tenant at all
(platform_admin) or before hydration resolves. Verified interactively: a tenant with color/logo
untouched shows its own business name in the Sidebar, with zero "ElProof" text present.

## 12. Browser tab title + favicon (user follow-up: "closer to feeling like their own")

The last two ElProof touchpoints *inside* an authenticated session were the browser tab title
(`index.html`'s static `<title>ElProof — Client Transparency Portal</title>`, never overridden by
JS) and the favicon (there wasn't one at all — no `<link rel="icon">` existed anywhere). Fixed:

- New `apps/web/src/theme/tabIdentity.ts` — `applyTabIdentity(businessName, logoUrl, consoleLabel?)`
  sets `document.title` to `"{businessName} — {consoleLabel}"` (e.g. "Griya Pernikahan Nusantara —
  WO Console"), and lazily creates/updates a `<link rel="icon">` pointed at the *same* logo object
  URL already fetched for the Sidebar/header image — no separate favicon upload needed.
  `resetTabIdentity()` reverts the title to whatever `index.html` originally had (captured once at
  module load, so no duplicated literal to keep in sync) and removes the favicon link entirely.
- `useTenantBrandingStore.hydrate()` now takes an optional `consoleLabel` param — `AppLayout.tsx`
  passes `"WO Console"`, `ClientPortalLayout.tsx` passes `"Portal Klien"` — and calls
  `applyTabIdentity`/`resetTabIdentity` alongside the existing CSS-var/logo-object-URL lifecycle it
  already manages, so this needed no new hydration trigger or effect.

Verified interactively: logging in as a tenant with a logo shows tab title `"{businessName} — WO
Console"` and a `<link rel="icon">` pointing at a `blob:` URL of that tenant's logo; logging out
reverts both to the app's original default, and the full existing regression suite (color preset,
logo upload, Sidebar re-theming, Platform Console staying unbranded, no cross-session bleed) still
passes unchanged.

## 13. Comprehensive verification pass (backend edge cases + the two remaining untested gaps)

Ran a wider round covering everything not yet exercised: backend validation/authorization edge
cases directly via API, a full migration down→up round-trip, and — closing §0's two acknowledged
gaps — a real Client Portal login and a non-Owner staff login, both driven end-to-end through the
actual UI (create project → attach a client → create a "Staff"-role user), not just code review.

**Backend edge cases (curl, all correct):**
- Invalid `brandColorPreset` (`"neonpink"`) on `PATCH /tenants/{id}` → `422`, rejected.
- Empty `brandColorPreset` on the same → `200`, tenant's existing preset kept unchanged (the
  §10 fix working as intended).
- Disallowed logo `mimeType` (`image/gif`) → `422`, rejected.
- Oversized logo (3MB decoded, over the 2MB cap) → `422` "Maksimal 2 MB", rejected.
- `platform_admin` calling `GET /tenants/me/branding` or `/tenants/me/logo` (no tenant bound) →
  `403`.
- No auth token at all → `401`.
- A tenant Owner calling the platform-admin-only `PUT /tenants/{id}/logo` → `403`.
- A tenant Owner calling `GET /tenants/{id}` for a **different** tenant's id → `403` (no IDOR).
- Same Owner calling `GET /tenants/me/branding` (their own tenant) → `200`, correct data.

**Migration round-trip**: `migrate down` cleanly dropped both columns; `migrate up` restored them,
existing rows correctly defaulting back to `brand_color_preset='navy'`.

**Client Portal (previously untested)**: created a project as the tenant Owner, attached a Bride
client with login credentials via the project's Client tab, logged in as that client — landed on
`/portal/ringkasan` with the tenant's emerald color applied (`--brand-navy-900` = the emerald
900 hex), the tenant's logo rendered in the portal header, and tab title `"{businessName} — Portal
Klien"`. No console errors.

**Non-Owner staff (previously untested)**: created a new user with the default "Staff" role (not
Owner) via Pengguna → logged in as them — Sidebar correctly shows the same tenant's logo and
emerald color (branding isn't Owner-exclusive, as designed), and the Owner-only "Langganan" nav
item is correctly still hidden for this role — confirming the branding change didn't interfere
with the pre-existing role-based nav visibility logic.

**Regression**: zero console errors and zero backend panics/fatal log lines across the entire
pass (tenant/project/client/staff creation, all logins, all logouts) — no sign of the branding
change having broken anything in modules it didn't touch.

**Known side effect of this testing pass**: tenant id 1 ("WO Debug Test", a pre-existing local
dev-only fixture) had its `email`/`phone`/`city` overwritten with throwaway test values by an
early edge-case request before later tests switched to using dedicated fresh tenants instead —
harmless (disposable local dev data, not its `brand_color_preset`/name/credentials), but flagged
here rather than silently left unmentioned.

## 14. 15 → 20 presets, gold/orange fixed to read as vivid instead of brown

User feedback: comparing against a real vivid-orange CTA button, "Emas"/"Oranye" read as too
brown/muddy — both used the Tailwind family's `900`/`800` shades (e.g. orange `#7c2d12`/`#9a3412`),
which are dark and desaturated. Also requested reaching 20 presets total, including a specific
mustard/olive hex, `#b3a500`.

- **Fixed `gold`/`orange`**: shifted the whole shade ramp lighter — `family-600/500/400` instead of
  `family-900/800` — e.g. orange is now `950 #ea580c / 900 #f97316 / 800 #fb923c` (was
  `#431407/#7c2d12/#9a3412`). `900` (the button/active-nav role) is now visually near-identical to
  the reference screenshot's vivid orange. Verified: the "Oranye" swatch's computed background is
  `rgb(249, 115, 22)` = `#f97316`, exactly the intended hex.
- **5 new presets** (`apps/web/src/theme/brandPresets.ts` + `apps/api/.../domain/brand_preset.go`):
  `mustard` (Zaitun — `900` is exactly the requested `#b3a500`), `green` (Hijau — deliberately
  shifted off `green-600`/`#16a34a` since that hex is this app's fixed `--color-success`, to avoid
  a brand preset ever looking identical to the status color), `sky` (Biru Langit), `pink` (Merah
  Muda), `yellow` (Kuning).
- **Renamed one existing label**: `rose`'s Indonesian label changed from "Merah Muda" to "Mawar"
  (its actual dusty/mauve tone reads more like a rose than a bubblegum pink) so the new, more vivid
  `pink` preset could take the "Merah Muda" name instead — `rose`'s hex/key are unchanged, so no
  tenant that already picked it is affected, only the label text in the picker.
- No migration needed — presets are a validated string key in `VARCHAR(20)`, not stored hex; this
  is a pure code change on both sides.

Verified interactively: edit-tenant modal renders exactly 20 swatches with the expected labels in
order; selected "Zaitun" (mustard), saved (`PATCH` returned `brandColorPreset: "mustard"`), logged
in as that tenant's Owner — Sidebar background computed to exactly `#b3a500`, confirming the
user's exact requested hex renders correctly end-to-end.

## 15. LoginPage neutralized (no ElProof color, no ElProof wording)

Per §0's scope decision, `LoginPage.tsx` was deliberately left ElProof-branded since no tenant is
known pre-auth. User feedback: the navy color and literal "ElProof" wording there still work
against the "feels like my own app" goal, for the two principal types (staff, client) who
experience personalization everywhere else — even though a *literal* tenant-specific look here is
impossible without subdomain-based tenant resolution (out of scope), a **neutral, tenant-agnostic**
treatment (neither ElProof's nor any tenant's colors) is achievable and closes that gap.

- **Wording removed**: `"Selamat Datang di {APP_NAME}"` → `"Selamat Datang"`; the hardcoded (not
  even `APP_NAME`-templated) `"...admin platform ElProof."` → `"...admin platform."`. The
  `APP_NAME` import was dropped from the file entirely (no longer referenced). The hero-panel
  tagline ("Transparansi Persiapan Pernikahan...") needed no change — it never named ElProof to
  begin with, only needed the color fix below.
- **"EP" lettermark badge → generic `Lock` icon** (lucide-react) — the initials were themselves an
  ElProof identity mark independent of color.
- **Navy → slate**: the left hero panel's gradient/decorative blur circles and the icon badge
  changed from `bg-navy-*` (literal Tailwind classes on this page, unrelated to the tenant-branding
  CSS-variable system) to `bg-slate-*` — Tailwind's own neutral family, matching the neutral tone
  already used for this app's body text/borders elsewhere.
- **New `Button` variant, `"neutral"`** (`bg-slate-800 hover:bg-slate-700`, deliberately NOT tied to
  `--brand-navy-*`) — added rather than passing an overriding `className`, since the shared `cn()`
  helper is a plain class-joiner (no `tailwind-merge`), so two conflicting background classes on
  one element would race unpredictably. Used only by the "Masuk" submit button; every other
  `<Button>` call site in the app is untouched and still correctly resolves to `primary`
  (brand-tied navy/tenant color) — verified live that a platform-admin screen's button (e.g.
  "Tambah Tenant") is still exactly navy `#1e3a5f` after this change.
- **Deliberately left alone**: the shared `Input`'s focus ring (`focus:ring-navy-900/20`) and
  `Button`'s keyboard-focus outline (`focus-visible:outline-navy-900`) still reference navy on this
  page too — both are global, shared-component styles (touching them means touching every form in
  the app), and both are subtle, momentary, focus-only affordances, not a prominent brand
  statement — judged out of scope for this pass.
- **Deliberately left alone**: the pre-login browser tab title (still ElProof's default, since
  `tabIdentity.ts`'s tenant override only ever fires post-login). Neutralizing `index.html`'s
  static default would also strip ElProof's name from the marketing/homepage pages, which *should*
  keep it — that needs a proper per-route title system to do correctly, a separate, larger change.

Verified interactively: no "ElProof" substring anywhere in the rendered login page; heading reads
exactly "Selamat Datang"; hero panel and "Masuk" button both compute to Tailwind's `slate-800`
(confirmed via `getComputedStyle`, reported as an `oklch()` triple since Tailwind v4's default
palette is OKLCH-based — same color, just a different color-function representation than the
custom hex-based `--brand-navy-*` variables elsewhere); tagline unchanged; and, checked
specifically to rule out an accidental global regression, a platform-admin screen's ordinary
primary button is still the exact original navy after this change.

## 16. Contrast fix — gold/orange (and 5 more presets) were unreadable with white text

User feedback: darken gold/orange a bit, and — more importantly — **guarantee every preset's text
stays readable**, since Sidebar/Button text is always white. Investigated with the actual WCAG
relative-luminance contrast formula rather than eyeballing it, and the finding was worse than just
those two: §14's "shift lighter to look vivid" fix (`family-600/500/400`) broke white-text contrast
on the `"900"` role — used on **every primary button app-wide**, plus `"800"` (hover/active-nav) —
for 6 of the 20 presets:

| Preset | Old `900` | Contrast vs white | New `900` | New contrast |
|---|---|---|---|---|
| Emas (gold) | `#f59e0b` | **2.15:1** | `#b45309` | **~5.02:1** |
| Oranye | `#f97316` | **2.80:1** | `#c2410c` | **~5.18:1** |
| Zaitun (mustard) | `#b3a500` | **2.53:1** | `#7a7000` | **~5.07:1** |
| Kuning (yellow) | `#eab308` | **~2.1:1** | `#a16207` | **~4.92:1** |
| Biru Langit (sky) | `#0ea5e9` | **2.77:1** | `#0369a1` | **~5.93:1** |
| Merah Muda (pink) | `#ec4899` | **3.53:1** | `#db2777` | **~4.60:1** |

Plus `green`'s `"800"`/hover role (`#22c55e`, ~2.28:1 — its `"900"` role was already fine at
~5.01:1) → `#16a34a` (~3.30:1). WCAG AA requires 4.5:1 for normal text; all "900" (the
most-seen role — every primary button) targets are now comfortably above that. "800"/hover roles
were allowed a slightly lower bar (~3–4:1) since that state is momentary, not persistent.

**The one deviation from a literal user request**: the user's own specified mustard hex, `#b3a500`,
only measures ~2.53:1 — unusable on a button with white text. Flagged this explicitly and got
confirmation before darkening it to `#7a7000` (same olive/mustard character, ~5.07:1) rather than
keeping the exact hex at the cost of readability.

**Method, to avoid re-breaking this later**: don't pick "vivid" by eye — for any future preset
using a light/warm hue (yellow-through-green range), verify the WCAG relative-luminance contrast
of its `"900"` value against white before shipping it; family shades `700` and darker are safe for
most Tailwind hues, `600` is usually borderline/hover-only, `500` and lighter routinely fail badly
for hues in this range (they were fine for cooler hues like blue/red/purple/indigo even at `900`,
which is why those 14 original presets never needed this fix).

Verified interactively: all 7 changed presets' swatch colors in the picker match the new hex
exactly; full end-to-end recheck with "Oranye" (select → save → log in as that tenant's Owner) —
Sidebar renders the new vivid-but-readable orange, `--brand-navy-900` computes to exactly
`#c2410c`, and white Sidebar text (nav labels, avatar name, active-nav highlight) is clearly
legible.
