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

---

# PLAN — Venue Extraction (Revisi 2)

Status: **Not started — supersedes the original version of this section.** Nothing from the first
pass was ever committed or migrated against a real database, so this is a straight rewrite, not an
addendum. Architectural decision (Venue = own table but still the `vendors` module, 1:1 cardinality
via `projects.venue_id`, two-phase `projects ↔ vendors` wiring) is unchanged — see ADR-0016 and its
"Revisi" section for what changed and why. Next migration number is still **000020** (last shipped
migration is `000019_add_tenant_custom_domain`).

## 0. Goal

Pull "Venue" out of the generic `vendors` category system into its own directory: its own fields
(mandatory rental price, a separate fixed charge, capacity, a free-text facilities/social-media
block, a single document-or-photo attachment, a mandatory city, PIC vs. venue phone numbers kept
distinct, email/address optional), its own top-level menu, exactly one venue attachable per
project, a dedicated tab in both WO Console's Project Detail and Client Portal (public-safe fields
only there), and a bulk Excel import/template flow. See ADR-0016 for the full reasoning; this
section is the concrete "what to build, where."

## 1. Current state (confirmed by code inspection)

- `apps/api/internal/modules/vendors/domain/vendor.go` — `Vendor` struct: `ID, TenantID,
  CategoryID, Name, PICName, Phone, Email, Address, Notes, IsActive, CreatedAt, UpdatedAt`. No
  category-specific fields exist anywhere in this module.
- `apps/api/internal/modules/vendors/application/vendor_category_service.go:73-89` —
  `defaultCategoryTemplate`, the hardcoded 8-entry seed list; `{"Venue", "Gedung, hotel, atau
  lokasi acara pernikahan"}` is the first entry. `SeedDefaultCategories` (94-102) loops and calls
  `Create` per entry — no branching, no `IsSystem` flag.
- `apps/api/migrations/000006_create_vendor_tables.up.sql` — the only migration touching
  `vendors`/`vendor_categories`; `vendors.email VARCHAR(150) NOT NULL` (mandatory today),
  `category_id` is a real same-module FK to `vendor_categories.id`.
- `apps/api/internal/modules/vendors/vendors.module.go` — `Module{categoryHandler, vendorHandler,
  contracts}`; `NewModule(db, projects projectscontracts.Contracts)` — vendors already depends on
  `projects` one-way (for "Lihat Project" vendor-engagement history).
- `apps/api/internal/modules/vendors/contracts/contracts.go` — `Contracts` interface currently has
  exactly one method, `SeedDefaultCategories`, consumed by `platform`.
- `apps/api/internal/modules/projects/domain/project.go` — `Project.Venue string`: the wedding's
  free-text location, completely unrelated to vendor categories. **Not touched by this plan** — see
  ADR-0016 "What this does not change."
- `apps/api/internal/modules/projects/projects.module.go` — `SetClientAccessResolver`/
  `SetClientCleaner` are the two-phase-wiring template this plan's `SetVenueResolver` copies
  (interface defined by the consumer, satisfied structurally by the producer's concrete
  `Contracts()` value, wired in `main.go` after both modules exist).
- `apps/api/cmd/server/main.go` — `projectsModule` is built **before** `vendorsModule` — confirms
  the `projects → vendors` read needs the two-phase setter, not a constructor argument.
- `apps/web/src/shared/components/ui/Combobox.tsx` — **already exists**, already a search-as-you-
  type single-select (search input, filters options by label, keyboard nav) — this is the exact
  component the new "Kota" field needs. No new frontend component required for the search-select
  requirement, only a data source (the fixed city list, §2).
- Frontend: `apps/web/src/modules/vendors/` (`VendorFormModal.tsx`, `VendorListPage.tsx`,
  `vendor.schema.ts`, `useVendorStore.ts`, `types.ts`) and the sibling `apps/web/src/modules/
  vendor-categories/` — the template this plan's new `apps/web/src/modules/venues/` copies.
- `apps/web/src/shared/layouts/Sidebar.tsx`, `ProjectDetailLayout.tsx`, `ClientPortalLayout.tsx` —
  flat arrays / `<Outlet context>` patterns, unchanged from the original plan's description.
- **No Excel/spreadsheet dependency exists yet** in `apps/api/go.mod` — the bulk-import feature
  (§7) is this project's first use of one.

## 2. Data model

**City master list — the one genuinely new reference data this feature needs.** A tenant picks
exactly one city per venue, validated against a **fixed, closed list of 128 official
kota/kabupaten** across Java's six provinces plus Bali — not a curated "wedding destination" list
(so Ubud/Kuta/Nusa Dua/Seminyak — all *kabupaten* Gianyar/Badung, never Kota Denpasar — resolve to
their real regency, not an inaccurate stand-in). Same "fixed enum, frontend+backend must agree on
keys" discipline as `AllowedBrandColorPresets`, not free text and not a master-data CRUD table.

New file `apps/api/internal/modules/vendors/domain/venue_city.go`:
```go
package domain

// AllowedVenueCities is the fixed set of 128 official kota/kabupaten across Java's six provinces
// (DKI Jakarta, Banten, Jawa Barat, Jawa Tengah, DI Yogyakarta, Jawa Timur) plus Bali — the only
// two islands this feature covers (ADR-0016). Deliberately kota AND kabupaten, not kota alone:
// Bali has exactly one official kota (Denpasar), so a kota-only list would leave every venue in
// Ubud/Kuta/Nusa Dua/Seminyak (all kabupaten Gianyar/Badung) with no accurate city to pick.
var AllowedVenueCities = []string{
	// DKI Jakarta (5 kota administrasi + 1 kabupaten)
	"Jakarta Pusat", "Jakarta Utara", "Jakarta Barat", "Jakarta Selatan", "Jakarta Timur",
	"Kepulauan Seribu",
	// Banten (4 kota + 4 kabupaten)
	"Kota Tangerang", "Kota Tangerang Selatan", "Kota Serang", "Kota Cilegon",
	"Kabupaten Tangerang", "Kabupaten Serang", "Kabupaten Pandeglang", "Kabupaten Lebak",
	// Jawa Barat (9 kota + 18 kabupaten)
	"Kota Bandung", "Kota Bekasi", "Kota Bogor", "Kota Depok", "Kota Cimahi", "Kota Cirebon",
	"Kota Sukabumi", "Kota Tasikmalaya", "Kota Banjar",
	"Kabupaten Bandung", "Kabupaten Bandung Barat", "Kabupaten Bekasi", "Kabupaten Bogor",
	"Kabupaten Ciamis", "Kabupaten Cianjur", "Kabupaten Cirebon", "Kabupaten Garut",
	"Kabupaten Indramayu", "Kabupaten Karawang", "Kabupaten Kuningan", "Kabupaten Majalengka",
	"Kabupaten Pangandaran", "Kabupaten Purwakarta", "Kabupaten Subang", "Kabupaten Sukabumi",
	"Kabupaten Sumedang", "Kabupaten Tasikmalaya",
	// Jawa Tengah (6 kota + 29 kabupaten)
	"Kota Semarang", "Kota Surakarta", "Kota Salatiga", "Kota Magelang", "Kota Pekalongan",
	"Kota Tegal",
	"Kabupaten Banjarnegara", "Kabupaten Banyumas", "Kabupaten Batang", "Kabupaten Blora",
	"Kabupaten Boyolali", "Kabupaten Brebes", "Kabupaten Cilacap", "Kabupaten Demak",
	"Kabupaten Grobogan", "Kabupaten Jepara", "Kabupaten Karanganyar", "Kabupaten Kebumen",
	"Kabupaten Kendal", "Kabupaten Klaten", "Kabupaten Kudus", "Kabupaten Magelang",
	"Kabupaten Pati", "Kabupaten Pekalongan", "Kabupaten Pemalang", "Kabupaten Purbalingga",
	"Kabupaten Purworejo", "Kabupaten Rembang", "Kabupaten Semarang", "Kabupaten Sragen",
	"Kabupaten Sukoharjo", "Kabupaten Tegal", "Kabupaten Temanggung", "Kabupaten Wonogiri",
	"Kabupaten Wonosobo",
	// DI Yogyakarta (1 kota + 4 kabupaten)
	"Kota Yogyakarta", "Kabupaten Sleman", "Kabupaten Bantul", "Kabupaten Kulon Progo",
	"Kabupaten Gunung Kidul",
	// Jawa Timur (9 kota + 29 kabupaten)
	"Kota Surabaya", "Kota Malang", "Kota Batu", "Kota Kediri", "Kota Blitar", "Kota Madiun",
	"Kota Mojokerto", "Kota Pasuruan", "Kota Probolinggo",
	"Kabupaten Bangkalan", "Kabupaten Banyuwangi", "Kabupaten Blitar", "Kabupaten Bojonegoro",
	"Kabupaten Bondowoso", "Kabupaten Gresik", "Kabupaten Jember", "Kabupaten Jombang",
	"Kabupaten Kediri", "Kabupaten Lamongan", "Kabupaten Lumajang", "Kabupaten Madiun",
	"Kabupaten Magetan", "Kabupaten Malang", "Kabupaten Mojokerto", "Kabupaten Nganjuk",
	"Kabupaten Ngawi", "Kabupaten Pacitan", "Kabupaten Pamekasan", "Kabupaten Pasuruan",
	"Kabupaten Ponorogo", "Kabupaten Probolinggo", "Kabupaten Sampang", "Kabupaten Sidoarjo",
	"Kabupaten Situbondo", "Kabupaten Sumenep", "Kabupaten Trenggalek", "Kabupaten Tuban",
	"Kabupaten Tulungagung",
	// Bali (1 kota + 8 kabupaten)
	"Kota Denpasar", "Kabupaten Badung", "Kabupaten Bangli", "Kabupaten Buleleng",
	"Kabupaten Gianyar", "Kabupaten Jembrana", "Kabupaten Karangasem", "Kabupaten Klungkung",
	"Kabupaten Tabanan",
}

// allowedVenueCitySet mirrors AllowedVenueCities as a set, built exactly once at package load
// (not per call) so IsValidVenueCity is an O(1) map lookup instead of an O(128) linear scan.
// AllowedBrandColorPresets' own IsValidBrandColorPreset uses a linear scan and that's fine at 21
// entries — this list is 128 and will likely keep growing (more provinces later), so it earns the
// map instead of copying that precedent as-is. AllowedVenueCities itself stays a plain []string
// (needed in-order for the Kota dropdown/Excel template — a map has no stable iteration order).
var allowedVenueCitySet = buildVenueCitySet()

func buildVenueCitySet() map[string]struct{} {
	set := make(map[string]struct{}, len(AllowedVenueCities))
	for _, c := range AllowedVenueCities {
		set[c] = struct{}{}
	}
	return set
}

func IsValidVenueCity(city string) bool {
	_, ok := allowedVenueCitySet[city]
	return ok
}
```
`apps/web/src/modules/venues/constants/cities.ts` mirrors this list verbatim (same "both sides
agree on the exact strings" discipline as `brandPresets.ts` vs. `AllowedBrandColorPresets`) — the
frontend's `Combobox` renders it as `<option value={city}>{city}</option>` per entry, no fixed-enum
TypeScript union needed (unlike brand presets, nothing branches per-city in code, so a plain
`string[]` constant is enough).

**Migration `000020_create_venue_tables`** (rewritten in place — never applied anywhere, so this
edits the existing file rather than adding a new one):
```sql
CREATE TABLE venues (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  tenant_id BIGINT UNSIGNED NOT NULL,
  name VARCHAR(150) NOT NULL,
  pic_name VARCHAR(150) NOT NULL,
  phone_pic VARCHAR(30) NOT NULL,
  phone_venue VARCHAR(30) NULL,
  email VARCHAR(150) NULL,
  address VARCHAR(255) NULL,
  city VARCHAR(100) NULL,
  rental_price BIGINT UNSIGNED NULL,
  charge BIGINT UNSIGNED NULL,
  capacity INT UNSIGNED NULL,
  facilities TEXT NULL,
  social_media TEXT NULL,
  notes TEXT NULL,
  attachment_path VARCHAR(500) NULL,
  attachment_mime_type VARCHAR(100) NULL,
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_venues_tenant (tenant_id)
);
```
No more `venue_photos` table — one venue has at most one attachment now (see §3's "Lampiran"
decision), so a side table is no longer warranted. `phone_pic` is `NOT NULL` (the mandatory
personal contact number); `phone_venue`, `email`, `address` are nullable/optional. `city` is
`NULL`-able at the DB **despite being a mandatory field on the create form** — same "DB permits
it, the create-schema enforces it" split already used for `rental_price`, needed so the data
migration (§2 below) can insert legacy rows with no city information without failing. `facilities`/
`social_media` are plain `TEXT` now, not `JSON` — free text, no array/struct, no marshal/unmarshal
code needed in the repository at all. `attachment_mime_type` is what lets the cross-module contract
(§4) and the download endpoint (§3) decide "is this safe to show a client" without re-opening the
file.

**Migration `000022_migrate_venue_vendor_data`** (rewritten in place, same reasoning): only the
column list changes to match the renamed/removed columns —
```sql
INSERT INTO venues (tenant_id, name, pic_name, phone_pic, email, address, notes, is_active, created_at, updated_at)
SELECT v.tenant_id, v.name, v.pic_name, v.phone, v.email, v.address, v.notes, v.is_active, v.created_at, v.updated_at
FROM vendors v
JOIN vendor_categories vc ON vc.id = v.category_id
WHERE vc.name = 'Venue'
  AND NOT EXISTS (
    SELECT 1 FROM venues existing
    WHERE existing.tenant_id = v.tenant_id AND existing.name = v.name
  );

UPDATE vendors v
JOIN vendor_categories vc ON vc.id = v.category_id
SET v.is_active = FALSE
WHERE vc.name = 'Venue';

UPDATE vendor_categories SET is_active = FALSE WHERE name = 'Venue';
```
(`NOT EXISTS` guard kept from the prior revision — makes this migration safe to re-run, per the
real migration re-run incident ADR-0011 documents.) `city`, `facilities`, `social_media`,
`attachment_path` are simply absent from the INSERT column list — old `vendors` rows never had
this data, so migrated venues get `NULL`/empty for all of them, same as before. `phone` (the old
vendor's single phone column) maps to the new `phone_pic` — the closest semantic match, since it
was the vendor's actual contact number.

**Migration `000021_add_venue_id_to_projects`** — unaffected by this revision, stays exactly as
originally planned (`ALTER TABLE projects ADD COLUMN venue_id BIGINT UNSIGNED NULL`, no FK).

**Go struct** — `apps/api/internal/modules/vendors/domain/venue.go` (rewritten):
```go
type Venue struct {
	ID                  int64
	TenantID            int64
	Name                string
	PICName             string
	PhonePIC            string
	PhoneVenue          *string
	Email               *string
	Address             *string
	City                *string
	RentalPrice         *int64
	Charge              *int64
	Capacity            *int
	Facilities          *string
	SocialMedia         *string
	Notes               string
	AttachmentPath      *string
	AttachmentMimeType  *string
	IsActive            bool
	CreatedAt           time.Time
	UpdatedAt           time.Time
}
```
No more `VenueSocialMediaLink`/`VenuePhoto` types — deleted entirely. `Facilities`/`SocialMedia`
are now plain `*string` (nullable free text), scanned/written exactly like `Notes`, no JSON
marshal/unmarshal code anywhere in the repository.

## 3. Backend — files inside the existing `vendors` module

- `domain/venue.go` — struct above. `domain/venue_city.go` — the fixed list (§2). Delete
  `domain/venue_photo.go` (not created if following this revision from scratch).
- `application/venue_service.go` — `VenueInput` (name, picName, phonePIC, phoneVenue, email,
  address, city, rentalPrice, charge, capacity, facilities, socialMedia, notes — one shared struct
  for Create/Update, matching `VendorInput`'s style). `Create`/`Update` validate `City` with
  `domain.IsValidVenueCity` **whenever it's non-empty** (both create and edit — same "always valid
  if provided, only create requires it non-empty" split as `rental_price`, enforced at the frontend
  create-schema, not re-checked here since this codebase doesn't re-validate simple required-ness
  server-side beyond what Zod already does — see `VendorInput.Name`'s precedent). `Get/List/
  ListPaginated/SetActive` unchanged in shape from the original plan. `UploadAttachment`/
  `DownloadAttachment` replace `UploadContractDocument`/`AddPhoto`/etc. entirely — one base64-JSON
  upload (mirrors the tenant-logo single-file shape), storing both `attachment_path` and the
  caller-supplied MIME type. No more photo-gallery methods at all.
- `infrastructure/mysql_venue_repository.go` — plain `database/sql`; **no JSON marshal/unmarshal
  helpers needed** (facilities/social_media are `TEXT`, not `JSON`, so they scan/bind exactly like
  `notes`). Delete `mysql_venue_photo_repository.go` (not created). Gains one new method for §7's
  bulk import: `CreateBatch(ctx, venues []domain.Venue) error` — one multi-row `INSERT` per chunk
  (see §7's performance design). No per-row lookup method is added here — §7 deliberately resolves
  duplicates via one bulk `List` + an in-memory map, not a query-per-row helper, so there's no
  `FindByNameAndCity` (or similar) sitting unused outside the import path. The regular create/
  update endpoints stay exactly as unconstrained as every other module's — no DB-level uniqueness
  on name+city; a WO manually creating two similarly-named venues through the normal form is a
  deliberate action, not a batch-file artifact to guard against.
- `presentation/venue_handler.go` — `Collection`/`Item` mirroring `vendor_handler.go`. The
  attachment download handler enforces the privacy split itself, not just the frontend: it fetches
  the caller's principal type from `middleware.FromContext`, and if `PrincipalType == "client"` **and**
  `AttachmentMimeType` doesn't start with `"image/"`, returns 403 — so a client can't see a
  document-type attachment even by calling the endpoint directly with a venue ID they know about.

**New endpoints** (registered in `vendors.module.go`'s `RegisterRoutes`):
```
GET/POST      /api/v1/venues                      (authed, requireStaffTenant)
GET/PATCH     /api/v1/venues/{id}                  (authed, requireStaffTenant)
POST          /api/v1/venues/{id}/toggle-active    (authed, requireStaffTenant)
PUT/GET       /api/v1/venues/{id}/attachment       (authed; PUT requireStaffTenant, GET requireTenant + the mime-type/principal check above)
GET           /api/v1/venues/template              (authed, requireStaffTenant) -- downloads a blank Excel template, §7
POST          /api/v1/venues/import                (authed, requireStaffTenant) -- bulk insert from an uploaded Excel file, §7
```
(`requireStaffTenant` is the stricter helper already added in the first implementation pass — kept
as-is; still the right gate for anything commercial/management-facing. `requireTenant` on the GET
attachment route is what lets a `client` principal reach the handler at all, with the mime-type
check inside doing the real gating — mirrors the "photos are the one exception" reasoning from the
original plan, just narrowed to a single conditional file instead of a whole gallery route.)

**`vendors/contracts/contracts.go`** — `VenueSummary` simplified:
```go
type VenueSummary struct {
	ID                  int64
	Name                string
	Address             *string
	City                *string
	Capacity            *int
	Facilities          *string
	SocialMedia         *string
	HasVisibleAttachment bool // true only if an attachment exists AND its mime type is image/*
}
```
No more `PhotoIDs []int64` or a `VenueSocialMediaLink` type — `SocialMedia` is now the same plain
`*string` the venue itself stores. `GetVenueSummary`'s implementation computes
`HasVisibleAttachment` by checking `venue.AttachmentPath != nil && strings.HasPrefix(*venue.AttachmentMimeType, "image/")`.
Still deliberately excludes `RentalPrice`/`Charge`/PIC contact — those stay staff-only via
`GET /venues/{id}` directly, never through this contract.

**Performance note**: this also means `GetVenueSummary` is down to **one query** (`c.venues.Get`)
instead of the two the photo-gallery version needed (`Get` + `ListPhotos`) — a side effect of
dropping the gallery, not a separate optimization pass. `ProjectVenueTabPage` (WO Console) still
issues two requests total for one tab open — this summary contract call, plus a direct
`GET /venues/{id}` for the staff-only commercial fields — deliberately kept separate rather than
merged into one broader response: the split is a security boundary (§3's field-visibility rule),
and two lightweight indexed-PK reads on a single page load is not a real cost worth trading that
boundary away for.

**`vendor_category_service.go:73-89`** — unaffected by this revision; still drops the `{"Venue",
...}` seed entry, exactly as originally planned.

## 4. Cross-module wiring — `projects` reads from `vendors`

Unchanged from the original plan in every structural respect (the `VenueResolver` interface,
`SetVenueResolver` two-phase wiring, `GET /projects/{id}/venue`, `projects.venue_id`'s tri-state
PATCH semantics, `main.go`'s wiring line). Only the payload shape riding through it changed, per
§3's simplified `VenueSummary`. One addition worth calling out explicitly here since it was a real
gap found during the first pass's own verification: `ProjectService.GetVenue` must check
`s.venues == nil` (not just `p.VenueID == nil`) before calling `s.venues.GetVenueSummary`, matching
the existing nil-guard convention `ClientCleaner` already uses in this exact file — carry this
forward into the rewrite, don't reintroduce the gap.

## 5. Frontend — new `venues` module (mirrors `modules/vendors/`)

- `apps/web/src/modules/venues/constants/cities.ts` — the 128-entry list from §2, verbatim match
  with the backend's `AllowedVenueCities`.
- `apps/web/src/modules/venues/types.ts` — `Venue` type: `phonePic: string`, `phoneVenue: string |
  null`, `address: string | null`, `city: string | null`, `facilities: string | null`,
  `socialMedia: string | null`, `hasAttachment: boolean`, `attachmentIsImage: boolean` (drives
  whether the edit form shows an image preview vs. a generic "document" icon).
- `apps/web/src/modules/venues/schemas/venue.schema.ts` — `name, picName, phonePic` required;
  `phoneVenue, email, address, notes` optional; `city` required **only** on `venueCreateSchema`
  (same split pattern as `rentalPrice`, mirroring `tenant.schema.ts`'s `tenantSchema`/
  `tenantCreateSchema` two-schema convention) and validated with `.refine()` against the imported
  `VENUE_CITIES` constant; `facilities`, `socialMedia` are now plain `z.string().optional()` — no
  array schema, no chip-list state in the component at all.
- `apps/web/src/modules/venues/components/VenueFormModal.tsx` — `Facilities`/`Sosial Media` render
  as plain `<Textarea>` (delete the chip-list add/remove UI from the first pass entirely). New
  "Kota" field uses the existing `Combobox` component directly, options built from
  `VENUE_CITIES.map(c => <option value={c}>{c}</option>)` — no new component. "Lampiran" becomes a
  **single** file input (delete the photo-gallery grid UI and its multi-file state entirely) —
  accepts image or PDF, shows an image preview if the stored mime type is `image/*` else a generic
  file/PDF icon with a "Lihat Dokumen" link, same blob-fetch-then-`window.open` pattern already
  used for the old contract-document view action.
- `apps/web/src/modules/venues/pages/VenueListPage.tsx` — mirrors `VendorListPage.tsx`, plus (§7) a
  "Download Template" button and an "Import Excel" upload control with a results panel.
- `apps/web/src/shared/services/api-endpoints.ts` — `venues: {...}` group: `base, item,
  toggleActive, attachment(id), template, import` — no more `photos`/`photo`/`photoFile` entries.
- Sidebar/routes wiring (`route-paths.ts`, `protected.routes.tsx`, `Sidebar.tsx`) — unchanged from
  the original plan.

## 6. Frontend — Project Detail (WO Console) and Client Portal integration

- **WO Console** (`ProjectVenueTabPage.tsx`): summary card now shows `city`, and the attachment
  section is a single link/preview instead of a gallery grid — otherwise unchanged (still fetches
  `GET /venues/{id}` directly for the full staff-only record, still has the "Ubah Venue"/"Lepas
  Venue" actions against `PATCH /projects/{id}`).
- **Client Portal** (`VenueTabPage.tsx`): renders `city` alongside address/capacity; the photo
  gallery grid becomes **one** conditional `<img>` — only fetched/rendered when
  `summary.hasVisibleAttachment` is true (the field name itself documents the mime-type gate
  already enforced server-side, per §3). No document/PDF ever attempted here.

## 7. NEW — Bulk upload / import

Two staff-only actions, both living in `venue_handler.go`/`venue_service.go` alongside the rest of
this module:

**Dependency**: add `github.com/xuri/excelize/v2` to `apps/api/go.mod` — this project's first
Excel dependency (no prior precedent to mirror; this is a genuinely new capability, not a repeat
of an existing pattern).

**`GET /venues/template`** — generates and streams a blank `.xlsx` with one header row matching the
create form's fields, in this order: `Nama Venue, Nama PIC, No Tlp PIC, No Tlp Venue, Email,
Alamat, Kota, Harga Sewa, Charge, Kapasitas, Fasilitas, Sosial Media, Catatan`. No attachment
column — files can't round-trip through a spreadsheet cell, so a venue imported this way starts
with no attachment, added later through the normal edit form. Content-Type
`application/vnd.openxmlformats-officedocument.spreadsheetml.sheet`, `Content-Disposition:
attachment; filename="template-venue.xlsx"`.

**`POST /venues/import`** — accepts a base64-encoded `.xlsx` (same upload-body convention as every
other file upload in this codebase, not multipart). Performance matters here specifically because
this endpoint processes N rows in one synchronous HTTP request — the design below is deliberately
**O(1) database round-trips for the duplicate check and batched writes**, not the naive "one query
per row" shape:

1. **Row cap, checked before anything else parses**: reject the file outright (422, no rows
   processed) if it has more than **1000 data rows** — same spirit as this codebase's existing
   per-upload size caps (evidence 15 MB, logo 2 MB), just measured in rows instead of bytes, so one
   pathological file can't tie up a request indefinitely or blow up memory.
2. **Stream-parse with `excelize`'s row iterator** (`Rows()`), not `GetRows()` — reads row-by-row
   instead of materializing the whole sheet into memory at once. At this feature's realistic scale
   (tens to low hundreds of venues per import) the difference is invisible, but there's no reason
   to default to the less scalable API when the streaming one is equally easy to use.
3. **Prefetch once, not per row**: before touching any row's data, call
   `VenueRepository.List(ctx, tenantID)` (the existing full-roster method, already just
   `WHERE tenant_id = ?` against the `idx_venues_tenant` index) **exactly once**, and build an
   in-memory Go map `map[string]domain.Venue` keyed by `strings.ToLower(name) + "|" + strings.ToLower(city)`.
   Every row's duplicate check becomes an O(1) map lookup — **zero additional queries** for the
   whole file, versus a SELECT-per-row shape. No new single-row lookup repository method is needed
   at all (see §3) — the existing `List` method is enough.
4. For each row: validate required fields present (`Nama Venue`, `Nama PIC`, `No Tlp PIC`, `Kota`)
   and that `Kota` matches `domain.IsValidVenueCity` exactly. On failure, skip the row and collect
   its 1-based row number + reason — nothing is written for that row, and it costs no database
   work at all (rejected before the write phase).
5. **Batch the writes, separated by kind**:
   - Rows with **no match** in the prefetch map are collected into an `[]domain.Venue` slice and
     inserted via a single multi-row `INSERT INTO venues (...) VALUES (?,?,...), (?,?,...), ...`
     per **chunk of ~200 rows** (chunked so one INSERT statement doesn't grow unbounded for a
     1000-row file, and so MySQL's max packet size is never a concern) — not one `INSERT` per row.
   - Rows with a **match** in the prefetch map already carry that venue's `ID` (read straight from
     the map, no extra lookup) — each runs its own `UPDATE ... WHERE id = ?` (a single indexed
     primary-key write; not worth batching the way inserts are, since every row's `SET` values
     differ and MySQL has no clean multi-row `UPDATE ... VALUES` equivalent). `attachment_path`/
     `attachment_mime_type`/`is_active` are **never touched** by an import-triggered update — the
     Excel row has no attachment data and no active-status column at all, so leaving them alone
     (not clearing them) is the only sane behavior.
   - Neither path is wrapped in one all-file transaction — a row that fails at the database level
     (rare, but possible — e.g. a transient connection error) is caught and reported as a row error
     exactly like a validation failure, without undoing every other row already committed. This is
     what makes "partial success" a real guarantee rather than an accidental side effect of
     skipping validation-only failures.

Response:
```go
type VenueImportResult struct {
	InsertedCount int
	UpdatedCount  int
	Errors        []VenueImportRowError // {Row int, Message string}
}
```
Net query shape for a file of N rows, `u` of which match an existing venue: **1** (prefetch) +
**⌈(N-u)/200⌉** (batched inserts) + **u** (per-row updates) — versus the naive **2N** (a
`FindByNameAndCity` plus a `Create`/`Update` per row). For a 500-row file with 50 updates, that's
roughly 1 + 3 + 50 = 54 database round-trips instead of 1000.

Frontend renders "X venue baru ditambahkan, Y venue diperbarui (nama+kota sudah ada sebelumnya)"
plus a table of failed rows + reasons when `Errors` is non-empty, so the WO can see at a glance how
many rows were genuinely new vs. silently merged into an existing venue, and fix just the failed
rows in the original file and re-upload — not forced to
redo the whole batch.

## 8. Order of implementation

1. `domain/venue_city.go` (the fixed list) + rewrite migrations 000020/000022 in place (000021
   untouched) — apply and verify locally before writing dependent Go code.
2. Backend: `venues` domain/application/infrastructure/presentation, standalone (no project
   linkage, no bulk import yet) — buildable and curl-testable in isolation.
3. `vendors/contracts` extension (simplified `VenueSummary`/`GetVenueSummary`, including the
   `HasVisibleAttachment` mime-type check).
4. `projects` module wiring (`VenueID`, `VenueResolver` + nil-guard, `GET /projects/{id}/venue`,
   `main.go`'s line) — same shape as the original plan's §4, unaffected by the field revisions.
5. Bulk import: `excelize` dependency, template generation, upload parsing with partial-success
   reporting.
6. Remove "Venue" from `defaultCategoryTemplate`.
7. Frontend: `modules/venues/` (cities constant, schema, store, form with textarea fields + Kota
   combobox + single attachment, list page with import/template controls), sidebar, routes.
8. Frontend: WO Console's `ProjectVenueTabPage`, Client Portal's `VenueTabPage`.
9. Verification (§9).

## 9. Verification

- `go build ./...` / `go vet ./...` after each backend step, not just at the end.
- `npx tsc --noEmit -p apps/web/tsconfig.json` after the frontend steps.
- Apply migrations 000020/000021/000022 locally; confirm existing "Venue"-category vendors migrate
  with `phone` landing in `phone_pic`, and the source `vendors`/`vendor_categories` rows flip to
  `is_active = FALSE`.
- Manual pass: create a venue with an image attachment and one with a PDF attachment; confirm the
  image-attached venue's project shows it in Client Portal, and the PDF-attached one's does not
  (while WO Console shows both regardless of type). Confirm the Kota combobox search-filters
  correctly and rejects free text not in the list.
- Bulk import: upload a file with (a) one brand-new venue, (b) one row whose name+city exactly
  matches an existing venue (confirm it updates that row in place — including confirming its
  existing attachment and `is_active` value are untouched — rather than creating a duplicate), and
  (c) one row with an invalid Kota value; confirm the response reports `InsertedCount: 1,
  UpdatedCount: 1`, and exactly one row error with a useful message.
- Confirm a tenant's Vendor list/picker no longer offers "Venue" as a category for brand-new
  vendors, while a pre-existing tenant's now-deactivated "Venue" category simply disappears from
  active pickers (not deleted, just hidden).

# PLAN — Vendor Field Adjustment (sesuai slide "DATA VENDOR")

Status: **Planning only — no code touched yet.** Next migration number is **000023** (last shipped is
000019; 000020–000022 are the Venue Revisi 2 migrations, already written but also still uncommitted/
unapplied — see the section above).

## 0. Goal

Adjust the generic `Vendor` entity (still a member of the `vendors` module, sitting alongside
`VendorCategory` and `Venue`) to match the field list on the "DATA VENDOR" slide: a mandatory city
(same Java+Bali list Venue already uses), free-text social media, two mandatory package prices
("Harga Akad" and "Harga Akad+Resepsi"), a single document/photo attachment, Email/Alamat becoming
optional, plus a bulk Excel import/template flow — the same shape of feature Venue just got, applied
to Vendor. Unlike Venue, Vendor's `CategoryID` and `PICName` are kept (see §1's confirmed decisions).

## 1. Decisions already confirmed with the user

1. **Kategori dan Nama PIC dipertahankan.** The slide's field list only calls out what's new/changed,
   not the full form — `CategoryID` stays a required FK into `vendor_categories` (that module is
   untouched), `PICName` stays a required, separate contact-person field.
2. **Pemisahan staff-only diterapkan**, mengikuti pola Venue (ADR-0016). Today `GET /vendors` (and
   every other vendor endpoint) sits on `requireTenant`, which lets a `client` principal read the
   *entire* vendor record — confirmed via `apps/web/.../VendorProgressTabPage.tsx` (Client Portal),
   which calls `useVendorStore.fetchVendors()` directly, the same action the WO Console list page
   uses. Once commercial fields (harga akad, harga akad+resepsi) and a document attachment exist on
   this record, that gap becomes a real leak. This plan tightens every vendor endpoint to
   `requireStaffTenant` (already defined in this same `presentation` package from the Venue work — no
   new helper needed) and adds one new, deliberately minimal public endpoint for what Client Portal
   actually renders (see §5).
3. **Harga Akad dan Harga Akad+Resepsi wajib untuk semua kategori vendor**, tanpa pengecualian per
   kategori — one form shape for every vendor regardless of `CategoryID`.

Defaults assumed below (not yet separately confirmed — flagged here so they're easy to correct in
the next round, same iterative pattern as Venue's revisions):

- **Kota**: required only at *creation* (frontend `vendorCreateSchema`), nullable at the DB and
  editable-open on existing rows — same "DB permits it, create-schema enforces it" split already
  used for Venue's `RentalPrice`/`City` and this same table's own legacy-migration concern (existing
  vendor rows have no city value).
- **Harga Akad / Harga Akad+Resepsi**: same treatment — nullable at DB (existing rows have neither),
  required only on the create form via `vendorCreateSchema`.
- **Lampiran Dokumen**: fully staff-only (`requireStaffTenant` on both the upload and download route)
  — unlike Venue's attachment, nothing today suggests Client Portal's Vendor Progress tab should ever
  render it, so there's no image-only exception carved out (see §5).
- **Bulk import upsert key**: `(name, city, categoryId)`, case-insensitive on name/city — not just
  `(name, city)` like Venue. Reasoning: `CategoryID` stays a real identity dimension for Vendor (two
  different business relationships can share a name+city under different categories, e.g. a supplier
  who's both a caterer and, separately, a decorator), so collapsing across category would silently
  merge two distinct vendors.
- **`GET /vendors/{id}/project-history`** moves to `requireStaffTenant` too (tightened along with
  everything else) — confirmed unused by Client Portal (only `VendorListPage.tsx`'s "Lihat Project"
  modal calls it).

## 2. Current state (confirmed by code inspection)

- `apps/api/internal/modules/vendors/domain/vendor.go` — `Vendor{ID, TenantID, CategoryID, Name,
  PICName, Phone, Email, Address, Notes, IsActive, CreatedAt, UpdatedAt}`. `Email`/`Address` are plain
  `string` (mandatory) today.
- `apps/api/migrations/000006_create_vendor_tables.up.sql` — `vendors.email VARCHAR(150) NOT NULL`,
  `address VARCHAR(255) NOT NULL`. `category_id` is a real same-module FK to `vendor_categories.id`
  (`fk_vendors_category`) — this stays; it's a same-module FK (both tables owned by `vendors`), not a
  cross-module one, so it isn't subject to the no-cross-module-FK rule anyway.
- `apps/api/internal/modules/vendors/application/vendor_service.go` — `VendorInput{Name, CategoryID,
  PICName, Phone, Email, Address, Notes}`; `validateCategory` checks the category exists and belongs
  to the tenant (kept as-is).
- `apps/api/internal/modules/vendors/infrastructure/mysql_vendor_repository.go` — plain
  `database/sql`, no JSON columns, straightforward `vendorColumns` + `scanVendor`.
- `apps/api/internal/modules/vendors/presentation/vendor_handler.go` — `Collection`/`Item` both on
  `requireTenant` (defined in `vendor_category_handler.go`, shared across this package). `requireStaffTenant`
  already exists in `venue_handler.go` (same package) — directly reusable, not something new to write.
- `apps/api/internal/modules/vendors/domain/venue_city.go` — `AllowedVenueCities []string` (128
  entries) + `allowedVenueCitySet` + `IsValidVenueCity`, added for Venue. Since `Vendor` and `Venue`
  are both in this same module/package, Vendor's new City field can reference this directly with zero
  cross-module concern — but the name is Venue-specific today and needs a rename (§4) now that a
  second entity uses it.
- **Client Portal's actual vendor data need** — `apps/web/src/modules/client-portal/pages/tabs/
  VendorProgressTabPage.tsx:24-36,66-67` calls `useVendorStore.fetchVendors()` (→ `GET /vendors?
  all=true`, full record) purely to resolve `vendors.find(v => v.id === pv.vendorId)?.name` for each
  engagement card — category name is resolved separately via `useVendorCategoryStore`, not from the
  vendor record. **Nothing else from the vendor record is read here** — no PIC, phone, email, address,
  or (once added) price/attachment fields. This is the exact shape of the new public endpoint in §5.
- `apps/web/src/modules/vendors/{types.ts, schemas/vendor.schema.ts, stores/useVendorStore.ts,
  components/VendorFormModal.tsx, pages/VendorListPage.tsx}` — the template this plan's changes mirror
  onto, following the exact same shape as `modules/venues/`'s equivalent files post-Revisi 2.
- `apps/web/src/modules/venues/constants/cities.ts` — `VENUE_CITIES`/`isValidVenueCity`, needs to move
  to a shared, entity-agnostic location now that Vendor also needs it (§4).

## 3. Data model

**Migration `000023_alter_vendors_add_fields`** (new — the existing `vendors` table already has real
data via `000006`, so this is an `ALTER TABLE`, not a rewrite like Venue's still-unapplied migrations):

```sql
-- apps/api/migrations/000023_alter_vendors_add_fields.up.sql
ALTER TABLE vendors
  MODIFY COLUMN email VARCHAR(150) NULL,
  MODIFY COLUMN address VARCHAR(255) NULL,
  ADD COLUMN social_media TEXT NULL AFTER email,
  ADD COLUMN city VARCHAR(100) NULL AFTER address,
  ADD COLUMN price_akad BIGINT UNSIGNED NULL AFTER city,
  ADD COLUMN price_akad_resepsi BIGINT UNSIGNED NULL AFTER price_akad,
  ADD COLUMN attachment_path VARCHAR(500) NULL AFTER notes,
  ADD COLUMN attachment_mime_type VARCHAR(100) NULL AFTER attachment_path;
```
```sql
-- apps/api/migrations/000023_alter_vendors_add_fields.down.sql
-- Best-effort only (ADR-0011): reverting email/address to NOT NULL fails if
-- any row picked up a NULL value after the up-migration ran — same caveat
-- already accepted by 000022's down migration for this exact reason.
ALTER TABLE vendors
  DROP COLUMN attachment_mime_type,
  DROP COLUMN attachment_path,
  DROP COLUMN price_akad_resepsi,
  DROP COLUMN price_akad,
  DROP COLUMN city,
  DROP COLUMN social_media,
  MODIFY COLUMN address VARCHAR(255) NOT NULL,
  MODIFY COLUMN email VARCHAR(150) NOT NULL;
```

`email`/`address` become nullable to match the slide (no more asterisk on either). `social_media`/
`facilities`-style free text follows the exact `TEXT NULL` convention Venue's Revisi 2 already
established (no JSON, no array). `price_akad`/`price_akad_resepsi` are `BIGINT UNSIGNED NULL` (Rupiah,
same unit convention as Venue's `rental_price`/`charge`) — nullable at the DB despite being mandatory
on the create form, so this migration doesn't fail against existing vendor rows that have neither.

**Go struct** — `apps/api/internal/modules/vendors/domain/vendor.go` (rewritten):
```go
type Vendor struct {
	ID                 int64
	TenantID           int64
	CategoryID         int64
	Name               string
	PICName            string
	Phone              string
	Email              *string
	SocialMedia        *string
	City               *string
	Address            *string
	PriceAkad          *int64
	PriceAkadResepsi   *int64
	Notes              string
	AttachmentPath     *string
	AttachmentMimeType *string
	IsActive           bool
	CreatedAt          time.Time
	UpdatedAt          time.Time
}
```

## 4. Shared city list — rename, don't duplicate

Since `Vendor` and `Venue` are both inside the `vendors` module, reusing the exact same fixed
128-city list is a same-package reference, not a cross-module violation — but `AllowedVenueCities`/
`IsValidVenueCity` are named for an entity that no longer exclusively owns them:

- Backend: rename `apps/api/internal/modules/vendors/domain/venue_city.go` → `city.go`;
  `AllowedVenueCities` → `AllowedCities`; `IsValidVenueCity` → `IsValidCity`. Update
  `venue_service.go`'s one call site (`domain.IsValidVenueCity` → `domain.IsValidCity`). No behavior
  change, pure rename.
- Frontend: move `apps/web/src/modules/venues/constants/cities.ts` → `apps/web/src/shared/constants/
  cities.ts` (a cities list is domain-agnostic data, same tier as `shared/constants/brand.ts`); rename
  `VENUE_CITIES` → `CITIES`, `isValidVenueCity` → `isValidCity`. Update Venue's existing imports
  (`venue.schema.ts`, `VenueFormModal.tsx`) to the new path/names. New `vendor.schema.ts`/
  `VendorFormModal.tsx` import from this same shared location — no second copy of the 128 entries.

## 5. Backend — presentation layer changes

- `application/vendor_service.go` — `VendorInput` gains `SocialMedia, City string`,
  `PriceAkad, PriceAkadResepsi *int64`; `Create`/`Update` validate `City` via `domain.IsValidCity`
  whenever non-empty (identical rule to Venue's `validateVenueCity`). Add `UploadAttachment`/
  `DownloadAttachment` (single base64-JSON slot, same shape/size-cap/mime-whitelist as Venue's — PNG/
  JPEG/WebP/PDF, 15 MB) and `ImportVendors`/`VendorImportRow`/`VendorImportResult` mirroring Venue's
  `ImportVenues` almost exactly, with one addition: category-name resolution (§7).
- `infrastructure/mysql_vendor_repository.go` — extend `vendorColumns`/`scanVendor`/`Create`/`Update`
  for the new columns (plain nullable-string/int64 binding, no JSON). Add `CreateBatch` (multi-row
  INSERT, chunks of 200, identical shape to Venue's) and `UpdateAttachment`.
- `presentation/vendor_handler.go` — `Collection`/`Item` switch from `requireTenant` to
  `requireStaffTenant` (already defined in this package). `vendorResponse` gains `socialMedia, city
  *string`, `priceAkad, priceAkadResepsi *int64`, `hasAttachment, attachmentIsImage bool`. Add
  `attachment` PUT/GET routes (both `requireStaffTenant` — see §1's default), `Template`/`Import`
  handlers (mirrors Venue's `venue_handler.go` almost verbatim, swapping in category resolution). Add
  one new lightweight handler + type:
  ```go
  type vendorSummaryResponse struct {
  	ID   int64  `json:"id"`
  	Name string `json:"name"`
  }
  ```
  `GET /vendors/summary` (or `?summary=true` on the existing collection — a separate route reads
  cleaner given the very different auth gate) → `requireTenant` (not staff-only — this is the one
  route a `client` principal still needs), returns `{id, name}[]` for every active vendor, nothing
  else. This directly replaces what Client Portal's `fetchVendors()` call currently over-fetches.
- **Route table** (`vendors.module.go`):
  ```
  GET/POST      /api/v1/vendors                      (requireStaffTenant)
  GET/PATCH     /api/v1/vendors/{id}                  (requireStaffTenant)
  POST          /api/v1/vendors/{id}/toggle-active    (requireStaffTenant)
  GET           /api/v1/vendors/{id}/project-history  (requireStaffTenant) -- tightened, was requireTenant
  PUT/GET       /api/v1/vendors/{id}/attachment       (requireStaffTenant, both)
  GET           /api/v1/vendors/template              (requireStaffTenant)
  POST          /api/v1/vendors/import                (requireStaffTenant)
  GET           /api/v1/vendors/summary               (requireTenant -- staff AND client)
  ```
  `/vendors/summary` needs the same fixed-path-beats-subtree registration Venue's `/venues/template`/
  `/venues/import` already established — register it before the `/api/v1/vendors/` catch-all.

No `vendors/contracts` change — Client Portal calls this module's HTTP API directly (not through a
cross-module contract like Venue's `GetVenueSummary`), so this stays entirely inside `vendors`'
presentation layer.

## 6. Frontend

- `apps/web/src/modules/vendors/types.ts` — `Vendor` gains `email: string | null`, `socialMedia:
  string | null`, `city: string | null`, `address: string | null`, `priceAkad: number | null`,
  `priceAkadResepsi: number | null`, `hasAttachment: boolean`, `attachmentIsImage: boolean`. New
  `VendorSummary { id: string; name: string }`, `VendorImportResult`/`VendorImportRowError` (same
  shape as Venue's).
- `apps/web/src/modules/vendors/schemas/vendor.schema.ts` — split into `vendorSchema` (edit) /
  `vendorCreateSchema` (create), same two-schema pattern as `venue.schema.ts`: `city` required only
  on create (`.refine(isValidCity, ...)`, imported from the new shared `constants/cities.ts`),
  `priceAkad`/`priceAkadResepsi` required only on create (`min(1, ...)`), `email` becomes
  `z.union([z.literal(""), z.string().email(...)]).default("")`, `address` becomes
  `z.string().optional().default("")`, `socialMedia: z.string().optional().default("")`.
- `apps/web/src/modules/vendors/stores/useVendorStore.ts` — add `uploadVendorAttachment`,
  `downloadVendorTemplate`, `importVendors` (mirrors `useVenueStore.ts`'s equivalents exactly). Add
  `fetchVendorSummaries()` calling the new `GET /vendors/summary` — this is what Client Portal switches
  to; `fetchVendors()`/`fetchVendorPage()` keep hitting the now-staff-only `GET /vendors` (fine, only
  ever called from WO Console pages: `VendorListPage.tsx`, `ProjectVendorFormModal.tsx`).
- `apps/web/src/modules/vendors/components/VendorFormModal.tsx` — add "Kota" (`Combobox`, options from
  the shared `CITIES` constant), "Sosial Media" (`Textarea`), "Harga Akad"/"Harga Akad+Resepsi"
  (numeric Rp inputs, required-marked only on create — same `${isEditing ? "" : " *"}` label pattern
  Venue uses), a single "Lampiran" upload slot (mirrors Venue's exactly — one file input, image
  preview or generic document link, PNG/JPEG/WebP/PDF, 15 MB). "Alamat"/"Email" drop their `required`
  marker.
- `apps/web/src/modules/vendors/pages/VendorListPage.tsx` — add a "Kota" column, "Download Template"/
  "Import Excel" buttons + results panel (mirrors `VenueListPage.tsx`'s exactly).
- `apps/web/src/modules/client-portal/pages/tabs/VendorProgressTabPage.tsx` — swap `vendors =
  useVendorStore(s => s.vendors)` / `fetchVendors` for the new `vendorSummaries =
  useVendorStore(s => s.vendorSummaries)` / `fetchVendorSummaries` — the only line that actually reads
  from it (`vendors.find(v => v.id === pv.vendorId)?.name`) stays structurally identical, just against
  the smaller list.
- `apps/web/src/shared/services/api-endpoints.ts` — `vendors: {...}` gains `attachment(id), template,
  import, summary`.

## 7. Bulk import/export — same shape as Venue, plus category resolution

**`GET /vendors/template`** — blank `.xlsx`, header row: `Nama Vendor, Kategori, Nama PIC, No Tlp
Vendor, Email, Sosial Media, Kota, Alamat, Harga Akad, Harga Akad+Resepsi, Catatan`. No Lampiran
column (files don't round-trip through a spreadsheet cell, same reasoning as Venue's template).

**`POST /vendors/import`** — identical performance design to Venue's `ImportVenues` (row cap 1000,
streamed parse, one prefetch, batched inserts, per-row updates, no all-file transaction), with one
extra prefetch:

1. Row cap 1000, checked first (422 if exceeded), same as Venue.
2. **Two prefetches, not one**: `VendorRepository.List(ctx, tenantID, nil)` (existing vendors → dedupe
   map keyed by `lower(name)|lower(city)|categoryId`) **and** `VendorCategoryRepository.List(ctx,
   tenantID)` (tenant's categories → a `map[string]int64` keyed by `lower(category name)` →
   `categoryId`, built once). Both are already-existing, already-indexed, tenant-scoped queries — this
   stays O(1) additional round-trips, not O(1) *per row*.
3. Per row: validate `Nama Vendor`, `Nama PIC`, `No Tlp Vendor`, `Kota`, `Kategori` all present; `Kota`
   via `domain.IsValidCity`; `Kategori` resolved against the category-name map — no match is a row
   error ("Kategori tidak ditemukan: <nama>"), not a silent skip or auto-create (creating categories
   from a spreadsheet typo would be worse than rejecting the row).
4. Upsert key `lower(name)|lower(city)|categoryId` (the *resolved* numeric ID, not the raw text) —
   same in-batch dedupe fix already proven necessary for Venue (two brand-new rows sharing a key
   within one file collapse into a single insert, last-row-wins), applied here from the start.
5. Batched multi-row `INSERT` (chunks of 200) for new vendors; per-row `UPDATE ... WHERE id = ?` for
   matches — `attachment_path`/`attachment_mime_type`/`is_active` never touched by an import-triggered
   update, identical reasoning to Venue.

Response shape identical to Venue's `VenueImportResult` (`InsertedCount`, `UpdatedCount`, `Errors:
[]{Row, Message}`).

## 8. Order of implementation

1. Rename the shared city list (§4) — backend + frontend — and confirm Venue still builds/type-checks
   against the renamed symbols before touching anything Vendor-specific.
2. Migration `000023` (§3), apply and verify locally.
3. Backend: `domain/vendor.go`, `application/vendor_service.go` (incl. attachment methods), 
   `infrastructure/mysql_vendor_repository.go` (incl. `CreateBatch`), buildable/curl-testable in
   isolation before wiring routes.
4. `presentation/vendor_handler.go` route/gate changes (§5) + the new `/vendors/summary` endpoint.
5. Bulk import (§7) — reuses the `excelize` dependency Venue's work already added to `go.mod`.
6. Frontend: shared `constants/cities.ts` move, then `modules/vendors/` (types, schema, store, form,
   list page).
7. Frontend: `VendorProgressTabPage.tsx` switched to `fetchVendorSummaries`.
8. Verification (§9).

## 9. Verification

- `go build ./...` / `go vet ./...` after each backend step.
- `npx tsc --noEmit -p apps/web/tsconfig.json` after the frontend steps.
- Apply migration `000023` against a local DB seeded with existing vendor rows (from `000006`) to
  confirm the `MODIFY COLUMN ... NULL` changes don't fail against real data.
- Manually confirm: a `client` JWT gets 403 from `GET /vendors`, `GET /vendors/{id}`, and
  `GET /vendors/{id}/attachment`, but 200 from `GET /vendors/summary` with only `{id, name}`.
- Manually confirm Client Portal's Vendor Progress tab still renders vendor names correctly after the
  `fetchVendorSummaries` swap (no regression from the endpoint split).
- Bulk import: upload a file with (a) one brand-new vendor, (b) one row matching an existing
  vendor's exact name+city+category (confirms update-in-place, existing attachment/is_active
  untouched), (c) one row whose Kategori text doesn't match any existing category (confirms a row
  error, not a crash or silent skip), (d) two brand-new rows sharing name+city+category within the
  same file (confirms exactly one vendor created, not two).

# PLAN — Role-Based Access Control (Owner / Admin / Wedding Planner)

Status: **Planning only — no code touched yet.**

## 0. Goal

Restrict which menus/pages/actions each of the WO Console's 3 staff roles can reach — not just
hide sidebar items (already cosmetic-only today), but enforce it at the actual security boundary
(the backend API), plus a frontend route guard so a hidden menu can't be reached by typing its URL
directly. Confirmed target matrix:

| Area | Owner | Admin | Wedding Planner (stored as `Staff`) |
|---|---|---|---|
| Dashboard | full | full | **no access** |
| Project (list + detail + all tabs) | all projects | all projects | **only projects where `PICStaffID` = self** |
| Create / duplicate project | ✓ | ✓ | **no access** (never creates, not even a copy) |
| Assign / reassign a project's PIC | ✓ | ✓ | **no access** (can't change PIC even on their own project) |
| Modify their own assigned project (milestones, status, dates, etc.) | — | — | ✓ |
| Client (standalone `/clients` menu) | full | full | **no access** (menu hidden, route blocked) |
| Client data read (inside own project's Client tab) | — | — | **read allowed** |
| Vendor (standalone `/vendors` menu, create/edit/delete/import) | full | full | **no access** |
| Vendor data read (picker inside own project) | — | — | **read allowed** |
| Venue (standalone `/venues` menu, create/edit/delete/import) | full | full | **no access** |
| Venue data read (picker inside own project) | — | — | **read allowed** |
| Pengaturan → Pengguna | full | **no access** | **no access** |
| Pengaturan → Langganan | full | **no access** | **no access** |
| Pengaturan → Kategori Vendor | full | **no access** | **no access** |

## 1. Current state (confirmed by code inspection)

- **Role already flows end-to-end today** — this is the one piece of infrastructure already fully
  built: `apps/api/internal/modules/staff/domain/staff_member.go` defines `StaffRole` = `Owner` |
  `Admin` | `Staff`. `identity/infrastructure/jwt_issuer.go:38` writes `cred.Role` into the JWT's
  `role` claim at login. `shared/middleware/auth.go`'s `Claims.Role string` carries it on every
  authenticated request via `middleware.FromContext(ctx)`. **No new plumbing needed** — every
  handler that wants to gate by role can already read `claims.Role` today.
- **Only 2 places already check role**, both ad-hoc inline (not a shared helper):
  `platform/presentation/tenant_handler.go` (4 spots, `claims.Role != "Owner"`, gating
  subscription/activation actions) and `projects/presentation/project_endpoints.go:227`
  (`claims.role != "Owner"`, gating project hard-delete per ADR-0013).
- **Nothing else is role-gated** — every Vendor/Venue/Client/Staff-CRUD/Vendor-Category endpoint
  only checks `PrincipalType` (`staff` vs `client`), never `Role`. Any authenticated staff member,
  regardless of role, can call any of these today.
- **Frontend has zero route-level enforcement** — `protected.routes.tsx` has no role check at all;
  hiding a sidebar item (`Sidebar.tsx`'s `ownerOnly` boolean, added for Langganan only) is purely
  cosmetic. Typing the URL directly still renders the page and its API calls still succeed (except
  the 2 already-gated actions above, which would fail server-side with a 403).
- **`Project.PICStaffID`** (and the same field on `ProjectMilestone`, `Issue`, `ProjectVendor`)
  already exists and is already populated on create/update — but is **purely a display label**
  today ("who's responsible"), never used to filter a query. `ProjectService.List`/`ListPaginated`
  return every project in the tenant unconditionally — no staff-scoping parameter exists yet.
- **Vendor/Venue picker dependency inside Project Detail** — `ProjectVendorFormModal.tsx` calls
  `useVendorStore.fetchVendors()` (→ `GET /vendors?all=true`) to let staff pick a vendor to engage;
  `ProjectVenueTabPage.tsx` calls `useVenueStore.fetchVenues()` (→ `GET /venues?all=true`) for the
  "Pilih Venue" picker. Both hit the same **staff-only, not role-gated** endpoints the standalone
  Vendor/Venue list pages use. Confirmed with the user: Wedding Planner keeps **read** access to
  these two endpoints (and to Client data) specifically so these in-project pickers keep working —
  only the standalone menu pages and all write actions (create/edit/delete/toggle/import/attachment
  upload) become Owner/Admin-only.
- **Role naming** — confirmed: "Wedding Planner" is a **display-label-only** rename of the existing
  `Staff` role. The stored value (DB column, JWT claim, `StaffRole` Go type, TS `StaffRole` union)
  stays exactly `"Staff"` — no data migration, safe against the already-live "JWS Wedding" tenant.
  Only user-facing text (role badge, role dropdown option label) changes to read "Wedding Planner".
- **Login redirect is hardcoded today** — `LoginPage.tsx:108` sends every `staff` principal to
  `ROUTE_PATHS.dashboard` regardless of role. Since Wedding Planner loses Dashboard access, this
  needs to become role-aware (→ `ROUTE_PATHS.projects` for that role).

## 2. Backend — authorization layer

**New shared primitive** — `internal/shared/middleware/auth.go` gets one small addition:
```go
// HasRole reports whether claims.Role matches one of the given roles — a
// technical string-comparison utility (same tier as RequireAuth itself),
// not domain logic, so every module's own requireXxx helper can call this
// instead of re-implementing the comparison inline (as platform/projects
// already do today, ad-hoc, in 5 separate spots).
func (c *Claims) HasRole(roles ...string) bool {
	for _, r := range roles {
		if c.Role == r {
			return true
		}
	}
	return false
}
```
Each module's existing `requireStaffTenant`-style helper stays where it is (tenant-scoping needs
already differ per module — some allow `client` through, some don't) but gets a role-aware sibling
or an extra parameter, calling `claims.HasRole(...)` instead of a fresh inline comparison.

**`vendors` module** (`vendor_handler.go` + `venue_handler.go`):
- `Collection`/`Item` GET (list/get by ID), plus the two picker call sites — stay reachable by
  **any** staff role (Owner/Admin/Wedding Planner) — no change to the read path.
- Every write action — `create`, `update`, `toggle-active`, `uploadAttachment`, `Import` — gains a
  role check: `if !claims.HasRole("Owner", "Admin") { 403 }`. `Template`/`downloadAttachment`
  (GET) stay read-reachable by all staff, consistent with the read-access decision above.
- Same split for `vendor_category_handler.go`, except **read is Owner-only too** (Kategori Vendor
  has no read-for-picker requirement the way Vendor/Venue do — nothing inside Project Detail reads
  from vendor-categories directly for a Wedding Planner's own workflow).

**`staff` module** (Pengguna / `/api/v1/staff`) — every endpoint (list, create, update, toggle,
reset-password) becomes **Owner-only**. No read exception needed here (unlike Vendor/Venue) — no
in-project workflow reads the staff directory.

**`platform` module's subscription endpoints** — already partially Owner-gated (4 spots). Needs a
**full audit** during implementation: confirm every subscription-related read endpoint (not just
the 4 already-gated write actions) is Owner-only, since the sidebar's Langganan entry is now
Owner-only in full, not just certain actions.

**`projects` module** — the one genuinely new piece of logic, not just a role check. **Confirmed
correction from the user**: only Owner/Admin create projects and assign (or reassign) a project's
PIC; a Wedding Planner only ever *operates within* a project already assigned to them — they never
create one, never duplicate one (duplicate produces a new project, same restriction as create), and
never change who the PIC is (including reassigning it away from themselves).

- `ProjectRepository.List`/`ListPaginated` gain an optional `picStaffID *int64` filter parameter —
  same shape as `VendorRepository.List`'s existing optional `categoryID *int64` filter — added as
  `AND pic_staff_id = ?` when non-nil.
- `ProjectService.List`/`ListPaginated` (and the handler above them) pass `picStaffID = &staffID`
  when `claims.Role == "Staff"`, and `nil` for Owner/Admin (full tenant view).
- `presentation/handler.go`'s `Collection` (`POST /projects`, → `createProject`) — gains a role
  check: `if claims.role == "Staff" { 403 "Hanya Owner atau Admin yang dapat membuat project baru" }`.
- `resolveProjectAccess` (`handler.go:81`, the single choke point every `/projects/{id}/...`
  sub-route already passes through — `getProject`, `updateProject`, `deleteProject`, `cancelProject`,
  `toggleArchiveProject`, `duplicateProject`, and every milestone/issue/vendor-engagement/evidence/
  payment/venue sub-endpoint) — for a `staff` principal whose role is `Staff`, additionally fetches
  the project's current `PICStaffID` and 403s if it doesn't match `claims.staffID`. This is the one
  gate that makes a Wedding Planner's entire operational surface (not just the list) correctly
  scoped — a Wedding Planner who knows another project's numeric ID still can't reach any of its
  sub-resources by URL, since they all funnel through this same check.
- `duplicateProject` — gains its own extra role check (`Staff` → 403) **in addition to** passing
  `resolveProjectAccess`'s PIC-ownership gate — duplicating is a creation action, so being the PIC of
  the source project doesn't matter; a Wedding Planner is never allowed to create the resulting copy.
- `updateProject` — when `claims.role == "Staff"`, compares the incoming body's `PICStaffID` against
  the project's current value and rejects the request (`apperror.Validation`, not a silent
  no-op — a Wedding Planner should get a clear "only Owner/Admin can reassign PIC" message, not have
  their submitted value quietly dropped) if they differ. Every other field on their own project stays
  freely editable (milestones, status, dates, package, contract value, description, venue, etc.).
- `deleteProject` (hard delete, ADR-0013) — already Owner-only regardless of PIC; unaffected, no
  Wedding Planner exception even for their own project.
- `cancelProject`/`toggleArchiveProject` — no additional restriction beyond `resolveProjectAccess`'s
  PIC gate; these are ordinary operational actions on a project the Wedding Planner already owns.

## 3. Frontend

- `apps/web/src/modules/users/types.ts` — no change to the `StaffRole` union (stays `"Owner" |
  "Admin" | "Staff"`). New `ROLE_LABELS: Record<StaffRole, string>` (e.g. in `types.ts` or a new
  `constants.ts`) mapping `Staff` → `"Wedding Planner"` for display; `Owner`/`Admin` map to
  themselves. `UserRoleBadge.tsx` and `UserFormModal.tsx`'s `STAFF_ROLE_OPTIONS` dropdown render
  through this map instead of the raw role string.
- `Sidebar.tsx` — `NavLinkItem`/`NavGroupItem`'s `ownerOnly: boolean` becomes `allowedRoles?:
  StaffRole[]` (`undefined` = all roles, matching how `ownerOnly: false` meant "everyone" today):
  - Dashboard, Client, Vendor, Venue: `["Owner", "Admin"]`
  - Project: unset (all 3 roles)
  - Pengaturan (group): `["Owner"]` — the group vanishes entirely for Admin/Wedding Planner via the
    same "group hides if zero visible children" logic already built for `ownerOnly`.
- **New route guard** — `protected.routes.tsx` needs a `<RequireRole roles={[...]}>` wrapper (new
  small component, e.g. `shared/components/RequireRole.tsx`) around Dashboard/Client/Vendor/Venue/
  Pengguna/Langganan/Kategori-Vendor routes, redirecting to the caller's own first-accessible route
  (`/projects` for Wedding Planner, `/dashboard` otherwise) if `session.role` isn't allowed — this is
  what actually stops a hidden menu from being reached by typing its URL, which nothing does today.
- `LoginPage.tsx:108` — the hardcoded `navigate(ROUTE_PATHS.dashboard)` for `staff` principals
  becomes role-aware: `Staff` role → `ROUTE_PATHS.projects`, `Owner`/`Admin` → unchanged.
- `ProjectVendorFormModal.tsx`, `ProjectVenueTabPage.tsx`, `ProjectClientTabPage.tsx` — **no changes
  needed** — they keep calling the same read endpoints, which stay open to all staff roles per §2.
- **Project list page** — the "Tambah Project" (create) button and any "Duplikat" action are hidden
  for `session.role === "Staff"` (mirrors the backend's own create/duplicate rejection — this is the
  UX convenience half, the backend check from §2 is the actual boundary).
- **Project edit form** — the `PICStaffID` field renders as an editable staff picker for Owner/Admin
  (unchanged from today), but as a **disabled, read-only** "Ditugaskan ke: {nama}" display for a
  Wedding Planner editing their own project — visible so they know who's assigned, but not editable,
  matching the backend's rejection of a changed value from that role.

## 4. Order of implementation

1. `middleware.Claims.HasRole` helper.
2. Backend: `staff` module → Owner-only (smallest, self-contained).
3. Backend: `vendor-categories` → Owner-only; `vendors`/`venues` → split read (all staff) vs. write
   (Owner/Admin) per handler.
4. Backend: `platform` subscription read-endpoint audit → confirm Owner-only throughout.
5. Backend: `projects` → `picStaffID` filter on List/ListPaginated + PIC ownership check on
   Get/detail-tab endpoints for the `Staff` role.
6. Frontend: `ROLE_LABELS` relabeling (Badge/dropdown), `Sidebar.tsx`'s `allowedRoles` migration.
7. Frontend: `RequireRole` route guard wired into `protected.routes.tsx`.
8. Frontend: `LoginPage.tsx` role-aware redirect.
9. Verification (§5).

## 5. Verification

- `go build ./...` / `go vet ./...` after each backend step.
- `npx tsc --noEmit -p apps/web/tsconfig.json` after the frontend steps.
- Manually confirm with 3 seeded staff accounts (Owner/Admin/Staff) against a local DB:
  - Wedding Planner (`Staff`) JWT gets 403 from `GET /staff`, `GET /vendor-categories`, every
    Vendor/Venue **write** endpoint, and every `/subscriptions/*`/`platform` tenant-admin action —
    but 200 from `GET /vendors`, `GET /venues` (read), and `GET /projects` (filtered to their own
    `PICStaffID` only).
  - A Wedding Planner given another project's numeric ID directly cannot reach it via
    `GET /projects/{id}` or any of its tabs.
  - Admin gets 403 from `GET /staff` and `GET /vendor-categories`, 200 from everything else.
  - Sidebar renders the correct subset per role (Wedding Planner sees only Project; Admin doesn't
    see Pengaturan at all; Owner sees everything).
  - Typing a blocked URL directly (e.g. Wedding Planner navigating to `/vendors`) redirects instead
    of rendering the page.
  - Login redirects a Wedding Planner straight to `/projects`, not a blocked `/dashboard`.
  - Inside a Wedding Planner's own project, the vendor-engagement picker and venue picker still list
    the full tenant roster (read access preserved) and still work end to end.
  - A Wedding Planner gets 403 from `POST /projects` (create) and `POST /projects/{id}/duplicate`
    (even for their own project) — the "Tambah Project"/"Duplikat" buttons are also absent from the
    UI for this role.
  - A Wedding Planner's `PATCH /projects/{id}` with a changed `picStaffId` is rejected with a clear
    validation error, while every other field (milestones, status, dates, package, description,
    venue) on their own project still saves normally — and the edit form shows PIC as a disabled
    read-only display for this role, not an editable dropdown.
  - Owner/Admin can still freely create projects and reassign any project's PIC (including to/from a
    Wedding Planner) exactly as before — no regression to their existing workflow.

# PLAN — Terminology: "Milestone" → "Timeline" (display text only)

Status: **Planning only — no code touched yet.**

## 0. Goal

Replace every user-visible occurrence of the word "Milestone"/"milestone" with "Timeline"/"timeline"
across WO Console, Client Portal, and the public marketing/legal pages. This is a **display-text-only**
rename (confirmed scope, see §1) — no code identifiers, no API contract, no database schema change.

## 1. Decisions confirmed with the user

1. **Depth: display text only.** Variable/type/component/file names, API route paths
   (`/api/v1/projects/{id}/milestones`), and database schema (`project_milestones`, `vendor_milestones`
   tables, and the `evidence.related_kind` ENUM's `'vendorMilestone'` literal — which is real data
   already stored for the live "JWS Wedding" tenant) all stay exactly as they are today. Only strings a
   human actually reads change. This mirrors the same reasoning already used for the "Wedding Planner"
   role relabel — the risk of touching schema/data that's already live in production isn't justified by
   a pure terminology preference.
2. **Project Milestone and Vendor Milestone stay two distinct concepts** — both get the same word
   swap ("Timeline"), neither is merged or renamed to something more specific.

## 2. Current state (confirmed by code inspection)

- "Milestone" appears in **58 files** (21 backend Go, ~34 frontend, 5 marketing/legal pages) and in
  the hundreds across those files — but the overwhelming majority of those occurrences are **code
  identifiers** (`domain.ProjectMilestone`, `ProjectService.ListMilestones`, `MilestoneRail.tsx`,
  `useProjectStore`'s `milestones`/`fetchMilestones` fields, the `/milestones` route segment, the
  `project_milestones`/`vendor_milestones` tables) — all explicitly out of scope per §1.
- The **actual display-text inventory** is ~45 distinct strings across 15 files, detailed in §3.
- **"Timeline" is not used anywhere else in the codebase today** as a separate concept or entity —
  confirmed by search, so this rename creates no naming collision. Its one existing appearance
  (`HomePage.tsx`'s "Timeline & Milestone" feature-card title) already anticipates this rename.
- **Client Portal's own display text already says "tahapan", not "milestone"**
  (`RingkasanTabPage.tsx`, `VendorProgressTabPage.tsx`) — confirmed by code inspection. This rename is
  therefore purely a WO Console (staff-facing) + Dashboard + marketing-page concern; nothing in the
  Client Portal's rendered text needs to change.
- Some backend Go string literals are genuinely user-facing despite living in Go source — `response.OK`/
  `response.Error`/`response.Created` messages (shown as toasts) and `activity.Record`'s human-readable
  description argument (shown in the Activity Log / Recent Activity feed) — these count as "display
  text" under §1's scope and are included in §3, even though they sit inside `.go` files.

## 3. Full inventory — string changes by file

**`apps/web/src/modules/projects/pages/ProjectDetailLayout.tsx`**
- Tab list entry: `label: "Milestone"` → `"Timeline"` (the tab name itself, in Project Detail's tab bar)

**`apps/web/src/modules/projects/components/detail/ProjectMilestonesSection.tsx`**
- Card title: `"Milestone Persiapan Acara"` → `"Timeline Persiapan Acara"`
- Subtitle: `"Progress keseluruhan didasarkan pada milestone yang benar-benar telah diselesaikan, bukan
  angka manual."` → replace "milestone" → "timeline"
- `"{stats.overdue} milestone terlambat dari target"` → `"{stats.overdue} timeline terlambat dari
  target"`
- `IconActionButton` labels (mobile + desktop card, 2 occurrences each): `"Edit milestone"` →
  `"Edit timeline"`, `"Batalkan milestone"` → `"Batalkan timeline"`
- Table header: `<TH>Milestone</TH>` → `<TH>Timeline</TH>`
- Error messages (via `getApiErrorMessage` fallback text): `"Gagal memperbarui status milestone"`,
  `"Gagal menambahkan milestone"`, `"Gagal memperbarui milestone"`, `"Gagal mengubah urutan
  milestone"` → each "milestone" → "timeline"
- `<div id="milestone">` stays unchanged — an HTML anchor id, not rendered text (§1 scope: identifiers
  stay)

**`apps/web/src/modules/projects/components/ProjectMilestoneFormModal.tsx`**
- Modal title: `"Tambah Milestone"` → `"Tambah Timeline"`
- Description: `"Milestone baru akan ditambahkan di urutan paling akhir — gunakan tombol urutkan untuk
  memindahkannya."` → "Milestone" → "Timeline"
- Button: `"Simpan Milestone"` → `"Simpan Timeline"`
- Field label: `"Nama Milestone"` → `"Nama Timeline"`

**`apps/web/src/modules/projects/components/detail/ProjectMilestoneEditModal.tsx`**
- Description: `"Perbarui status dan jadwal milestone ini."` → "milestone" → "timeline"
- (`title={milestone.name}` is dynamic data, not literal text — no change)

**`apps/web/src/modules/projects/components/VendorMilestoneFormModal.tsx`**
- Modal title: `"Tambah Milestone Vendor"` → `"Tambah Timeline Vendor"`
- Button: `"Simpan Milestone"` → `"Simpan Timeline"`
- Field label: `"Nama Milestone"` → `"Nama Timeline"`

**`apps/web/src/modules/projects/components/detail/VendorMilestoneEditModal.tsx`**
- Placeholder: `"Tambahkan catatan perkembangan milestone ini..."` → "milestone" → "timeline"
- Empty-state description: `"Perubahan pada milestone ini akan tercatat di sini."` → "milestone" →
  "timeline"

**`apps/web/src/modules/projects/components/detail/ProjectVendorsSection.tsx`**
- Subtitle: `"Progress setiap vendor berdasarkan pencapaian milestone yang telah diselesaikan."` →
  "milestone" → "timeline"
- Label: `"Milestone Vendor"` → `"Timeline Vendor"`
- Table header: `<th>Milestone</th>` → `<th>Timeline</th>`
- `IconActionButton` labels: `"Edit Milestone"` → `"Edit Timeline"`, `"Batalkan milestone"` →
  `"Batalkan timeline"`
- Error messages: `"Gagal menambahkan milestone vendor"`, `"Gagal memperbarui milestone"`, `"Gagal
  memperbarui status milestone"`, `"Gagal menyimpan milestone"` → each "milestone" → "timeline"
- `relatedKind: "vendorMilestone"` / `entityType === "vendor_milestone"` stay unchanged — internal
  tag values matching the backend's own enum/DB literal (§1 scope: identifiers/data values stay)

**`apps/web/src/modules/projects/components/detail/ProjectHeaderCard.tsx`**
- `"Milestone project {x}/{y} · Milestone vendor {x}/{y}"` → `"Timeline project {x}/{y} · Timeline
  vendor {x}/{y}"`

**`apps/web/src/modules/projects/components/ProjectCard.tsx`**
- `"Belum ada milestone"` → `"Belum ada timeline"`
- `` `${completedMilestones}/${totalMilestones} milestone selesai...` `` → "milestone" → "timeline"
  (the `completedMilestones`/`totalMilestones` variable names stay — §1 scope)

**`apps/web/src/shared/components/ui/MilestoneRail.tsx`**
- `"Belum ada milestone"` → `"Belum ada timeline"`
- (Component name `MilestoneRail`/`MilestoneRailLegend`, type `RailMilestoneStatus`, prop `milestones`
  all stay — §1 scope)

**`apps/web/src/shared/components/ui/ProgressMeter.tsx`**
- Tooltip: `title="Milestone terhambat"` → `title="Timeline terhambat"`

**`apps/web/src/modules/dashboard/lib/attention.ts`**
- `category: "Milestone Terlambat"` → `"Timeline Terlambat"`
- `` title: `Milestone "${m.name}" terlambat` `` → `` `Timeline "${m.name}" terlambat` ``
- `` description: `...${row.overallPercent}% milestone selesai, H-...` `` → "milestone" → "timeline"

**Backend — `apps/api/internal/modules/projects/presentation/project_endpoints.go`**
- `"Milestone berhasil ditambahkan"` → `"Timeline berhasil ditambahkan"`
- `"ID milestone tidak valid"` → `"ID timeline tidak valid"`
- `"Milestone berhasil diperbarui"` → `"Timeline berhasil diperbarui"`
- `"Urutan milestone berhasil diperbarui"` → `"Urutan timeline berhasil diperbarui"`

**Backend — `apps/api/internal/modules/projects/presentation/vendor_engagement_endpoints.go`**
- `"Milestone vendor berhasil ditambahkan"` → `"Timeline vendor berhasil ditambahkan"`
- `"ID milestone tidak valid"` → `"ID timeline tidak valid"`
- `"Milestone vendor berhasil diperbarui"` → `"Timeline vendor berhasil diperbarui"`

**Backend — `apps/api/internal/modules/projects/application/project_service.go`**
- Activity description: `"Milestone project ditambahkan: "+m.Name` → `"Timeline project
  ditambahkan: "+m.Name`
- `apperror.NotFound("Milestone tidak ditemukan")` → `"Timeline tidak ditemukan"`
- Activity description: `"Milestone project diperbarui: "+m.Name` → `"Timeline project
  diperbarui: "+m.Name`
- `apperror.Validation("Urutan milestone tidak valid", nil)` (2 occurrences) → `"Urutan timeline
  tidak valid"`
- Activity description: `"Urutan milestone diubah"` → `"Urutan timeline diubah"`
- `domain.ActivityMilestoneUpdated` (the activity-type enum value) and `"project_milestone"` (the
  entity-type tag) stay unchanged — never rendered as literal text, confirmed by tracing the frontend's
  `ActivityType` union has no label map that would surface this raw value (§1 scope: identifiers stay)

**Backend — `apps/api/internal/modules/projects/application/vendor_engagement_service.go`**
- Activity description: `"Milestone vendor ditambahkan: "+m.Name` → `"Timeline vendor
  ditambahkan: "+m.Name`
- `apperror.NotFound("Milestone vendor tidak ditemukan")` → `"Timeline vendor tidak ditemukan"`
- Activity description: `"Milestone vendor diperbarui"` → `"Timeline vendor diperbarui"`
- `domain.ActivityMilestoneUpdated` / `"vendor_milestone"` stay unchanged (same reasoning as above)

**Marketing/legal pages**
- `apps/web/src/modules/homepage/pages/HomePage.tsx`:
  - Feature-card title `"Timeline & Milestone"` → `"Timeline"` (redundant once both words would say
    the same thing)
  - Body: `"Setiap tahap persiapan punya milestone dan tenggat yang dipantau bersama."` → "milestone"
    → "timeline"
  - Floating card label: `"Milestone Persiapan"` → `"Timeline Persiapan"`
- `apps/web/src/modules/homepage/pages/AboutPage.tsx`: `"...vendor, milestone, dan pembayaran..."` →
  "milestone" → "timeline"
- `apps/web/src/modules/homepage/pages/FaqPage.tsx`: `"...vendor, milestone, pembayaran, dan
  kendala..."` → "milestone" → "timeline"
- `apps/web/src/modules/homepage/pages/PrivacyPage.tsx`: two occurrences ("...vendor, milestone,
  status pembayaran..." and "...terbatas pada status milestone, ringkasan pembayaran...") → "milestone"
  → "timeline"
- `apps/web/src/modules/homepage/pages/TermsPage.tsx`: three occurrences (feature list, "pelacakan
  milestone dan pembayaran", "status milestone, ringkasan pembayaran") → "milestone" → "timeline"

**Client Portal — confirmed no change needed**: `RingkasanTabPage.tsx` and `VendorProgressTabPage.tsx`
already render "tahapan" in their client-facing text, never the literal word "milestone" — nothing to
do here.

## 4. Order of implementation

1. Backend response/error/activity-description strings (§3's 4 Go files) — smallest, self-contained,
   verify with `go build`/`go vet` (string-literal-only changes, should never fail either, but keep the
   habit).
2. Frontend — Project Detail's own Milestone section + modals (`ProjectDetailLayout.tsx`,
   `ProjectMilestonesSection.tsx`, `ProjectMilestoneFormModal.tsx`, `ProjectMilestoneEditModal.tsx`).
3. Frontend — Vendor Milestone (nested in the Vendor tab): `ProjectVendorsSection.tsx`,
   `VendorMilestoneFormModal.tsx`, `VendorMilestoneEditModal.tsx`.
4. Frontend — shared progress/summary surfaces: `ProjectHeaderCard.tsx`, `ProjectCard.tsx`,
   `MilestoneRail.tsx`, `ProgressMeter.tsx`, `dashboard/lib/attention.ts`.
5. Marketing/legal pages: `HomePage.tsx`, `AboutPage.tsx`, `FaqPage.tsx`, `PrivacyPage.tsx`,
   `TermsPage.tsx`.
6. Verification (§5).

## 5. Verification

- `go build ./...` / `go vet ./...` after step 1 (pure string literal edits — should be a no-op
  compile-wise, but confirms nothing was accidentally broken, e.g. a stray quote).
- `npx tsc --noEmit -p apps/web/tsconfig.json` after steps 2–5.
- `npm run build -w apps/web` to confirm the production build still succeeds.
- Manual/visual pass through WO Console: Project Detail's Milestone tab (list, add, edit, reorder,
  cancel — every button/label/error toast reads "Timeline"), the Vendor tab's nested vendor-timeline
  UI, the project card's mini-progress line, the Dashboard's "Perlu Perhatian" widget, and the
  Activity Log / Recent Activity feed (descriptions read "Timeline ditambahkan/diperbarui").
- Manual pass through the public marketing site (Beranda, Tentang Kami, FAQ, Syarat & Ketentuan,
  Kebijakan Privasi) to confirm no leftover "milestone" wording and no awkward phrasing from the
  word swap.
- Confirm Client Portal is visually unchanged (it already said "tahapan", nothing to regress there).
- Grep the whole repo for remaining `[Mm]ilestone` afterward, cross-check every remaining hit against
  §3's list of confirmed identifiers/out-of-scope items (route paths, DB tables, enum values, file/
  component names) — any hit NOT on that list is a missed string worth revisiting.

# PLAN — Timeline Default Template (configurable per tenant)

## 0. Goal

Let each tenant (WO business) configure its own default Timeline template — the checklist of items
auto-seeded into every newly created project — instead of today's single hardcoded list shared by
every tenant. Managed from a new **Pengaturan → Timeline Default** submenu.

## 1. Decisions confirmed

1. **Scope: per-tenant**, not a platform-wide setting. Each WO business can have a different starting
   checklist, mirroring how Kategori Vendor is already per-tenant rather than global.
2. **RBAC: Owner-only.** Consistent with every existing item inside the Pengaturan group — Admin does
   not see the Pengaturan group at all (confirmed in `Sidebar.tsx`'s `NAV_ITEMS` comment), so nothing
   about this feature changes that boundary.
3. **Delete semantics: hard delete**, not soft-deactivate. This deliberately diverges from the Kategori
   Vendor precedent: a vendor category is referenced by `project_vendors` rows (deleting one out from
   under existing data would orphan a foreign key), but a milestone template item is only ever *copied*
   into a new project's `project_milestones` at creation time — once copied, the two have zero
   ongoing relationship. There is nothing to orphan, so a plain `DELETE` is safe and simpler than a
   soft-state column nobody would ever need to query.
4. **Empty template is allowed.** A tenant may delete every item down to zero. The next project that
   tenant creates then starts with an empty Timeline tab (no server-side minimum-1-item validation) —
   consistent with "the default is just a starter, not a requirement" philosophy already implied by
   every timeline item being freely editable/cancellable after creation.
5. **Scope of effect: `Create()` only.** `Duplicate()` (Duplikat Project) is untouched — it continues
   cloning the *source project's actual milestones* via `cloneMilestonesFrom`, never the tenant
   template. Existing projects' timelines are never retroactively touched by editing the template
   afterward.
6. **Existing tenants must not regress.** The live production tenant ("JWS Wedding") and every other
   already-registered tenant have zero rows in the new table the moment it's created. If the migration
   only creates the table without backfilling data, every existing tenant's *next* new project would
   silently seed with **zero** timeline items until someone remembers to visit the new menu first —
   a real behavior regression, not just an empty-state edge case. The migration must backfill the
   current 6-item template for every row already in `tenants`, not just new tenants going forward (see
   §3.1).

## 2. Current state

- `apps/api/internal/modules/projects/application/default_milestones.go` — `defaultMilestoneTemplate`
  (6 hardcoded items: Name + `DaysBeforeEvent`) and `seedDefaultMilestones()`, called unconditionally
  from `ProjectService.Create()` (`project_service.go:195`). No DB table, no tenant scoping, no API,
  no UI — a tenant cannot see or change this today.
- **Nearest architectural precedent: Kategori Vendor** (`vendors` module) — a per-tenant list, seeded
  once at tenant registration via a cross-module contracts bridge
  (`platform/application/tenant_service.go:177` calls `vendorscontracts.SeedDefaultCategories`),
  managed from Pengaturan → Kategori Vendor, Owner-only for writes. This plan mirrors that shape almost
  exactly, with the one intentional divergence noted in §1.3 (hard delete instead of soft-deactivate).

## 3. Design

### 3.1 Database

New migration `apps/api/migrations/000024_create_project_milestone_templates.up.sql` /
`.down.sql`:

```sql
-- up
CREATE TABLE project_milestone_templates (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  tenant_id BIGINT UNSIGNED NOT NULL,
  sort_order INT NOT NULL,
  name VARCHAR(255) NOT NULL,
  days_before_event INT NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_project_milestone_templates_tenant (tenant_id)
);

-- Backfill (§1.6): every tenant that already exists gets today's hardcoded
-- template, so no existing tenant's next new project silently loses its
-- starter checklist the moment this migration ships.
INSERT INTO project_milestone_templates (tenant_id, sort_order, name, days_before_event)
SELECT id, 1, 'Survei Venue & Vendor', 90 FROM tenants
UNION ALL SELECT id, 2, 'DP / Tanda Jadi ke Vendor', 60 FROM tenants
UNION ALL SELECT id, 3, 'Technical Meeting', 14 FROM tenants
UNION ALL SELECT id, 4, 'Pelunasan Vendor', 7 FROM tenants
UNION ALL SELECT id, 5, 'Gladi Resik', 1 FROM tenants
UNION ALL SELECT id, 6, 'Hari-H Pernikahan', 0 FROM tenants;
```

No `is_active` column (unlike `vendor_categories`) — per §1.3, hard delete means there's nothing to
deactivate; a row's existence *is* the on/off switch. `days_before_event` must be `>= 0`, validated at
the service layer (mirroring how every other numeric field in this codebase is validated — no DB-level
CHECK constraint, per this project's MySQL/sqlc conventions).

### 3.2 Backend — domain

New file `apps/api/internal/modules/projects/domain/milestone_template.go`:

```go
package domain

import "time"

type ProjectMilestoneTemplate struct {
	ID              int64
	TenantID        int64
	SortOrder       int
	Name            string
	DaysBeforeEvent int
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
```

### 3.3 Backend — repository

New file `apps/api/internal/modules/projects/infrastructure/mysql_milestone_template_repository.go`,
`MySQLMilestoneTemplateRepository` — `List(ctx, tenantID)`, `FindByID(ctx, tenantID, id)`, `Create`,
`Update`, `Delete(ctx, tenantID, id)`, `NextSortOrder(ctx, tenantID)`, `Reorder(ctx, tenantID,
orderedIDs []int64)` — same column-scan and reorder-transaction shape as the existing
`MySQLMilestoneRepository` (project milestones), just keyed by `tenant_id` instead of `project_id`.

### 3.4 Backend — application service

New file `apps/api/internal/modules/projects/application/milestone_template_service.go`,
`MilestoneTemplateService` — `List`, `Create`, `Update`, `Delete`, `Reorder`, and `SeedDefaults(ctx,
tenantID)` (the 6-item literal moves here from `default_milestones.go`, same names/offsets as today —
no behavior change for any tenant that never opens the new menu).

`default_milestones.go`'s `seedDefaultMilestones()` (still called from `ProjectService.Create()`)
changes from reading the hardcoded var to calling `s.milestoneTemplates.List(ctx, tenantID)` and
looping over the result — same clamp-to-`prepStartDate` logic as today, applied per row's
`DaysBeforeEvent`. An empty list is not an error (§1.4): the loop simply does nothing and the new
project's Timeline tab starts empty.

`ProjectService` gains a `milestoneTemplates MilestoneTemplateRepository` field, constructor-injected
(same module, no cross-module wiring needed for this part — only the seeding trigger in §3.6 crosses a
module boundary).

### 3.5 Backend — presentation

New file `apps/api/internal/modules/projects/presentation/milestone_template_handler.go`, a dedicated
`MilestoneTemplateHandler` mirroring `VendorCategoryHandler`'s shape — registered directly in
`projects.module.go`'s `RegisterRoutes`, **not** funneled through the existing big
`Handler.Item`/`resolveProjectAccess` dispatcher used for `/projects/{id}/...` sub-resources, since
template data is tenant-level configuration, not scoped to one project.

Unlike Kategori Vendor's read-open-to-all-staff `GET`, every method here is Owner-only — a milestone
template has no dropdown/display use case elsewhere the way a vendor category name does, so there's no
reason to relax the read side.

- `GET /api/v1/milestone-templates` — list (un-paginated; expected to stay a short list, same as a
  single project's own Timeline tab).
- `POST /api/v1/milestone-templates` — create.
- `PATCH /api/v1/milestone-templates/{id}` — update name/days.
- `DELETE /api/v1/milestone-templates/{id}` — hard delete; `response.OK(w, "Template timeline berhasil
  dihapus", nil)`, mirroring `deleteProject`'s existing convention (`project_endpoints.go:249`) for
  the only other hard-delete action in this codebase.
- `PATCH /api/v1/milestone-templates` (collection endpoint, `{orderedIds: number[]}` body) — reorder,
  mirroring `reorderMilestones`'s existing shape for a project's own Timeline reordering.

### 3.6 Backend — seeding wiring (tenant registration)

- `apps/api/internal/modules/projects/contracts/contracts.go`'s `Contracts` interface gains
  `SeedDefaultMilestoneTemplate(ctx context.Context, tenantID int64) error`, implemented by calling
  `MilestoneTemplateService.SeedDefaults`.
- `apps/api/internal/modules/platform/application/tenant_service.go`'s `TenantService` gains a
  `projects projectscontracts.Contracts` field + `SetProjects(projects projectscontracts.Contracts)`
  method — same two-phase-wiring idiom as the existing `SetVendors` (needed because `platform` is
  built before `projects` in `main.go`, so this can't be a constructor argument).
- `Register()` gains `s.projects.SeedDefaultMilestoneTemplate(ctx, tenant.ID)`, right after the
  existing `s.vendors.SeedDefaultCategories(...)` call (`tenant_service.go:177`) — this only affects
  tenants registered *after* this change ships; §3.1's migration already backfilled everyone else.
- `apps/api/cmd/server/main.go`: `platformModule` (line ~157) is built before `projectsModule` (line
  ~164), so add `platformModule.SetProjects(projectsModule.Contracts())` right after `projectsModule :=
  projects.NewModule(...)`, mirroring the existing `platformModule.SetVendors(vendorsModule.Contracts())`
  at line 170.

### 3.7 Frontend

New module `apps/web/src/modules/milestone-templates/` (mirrors `modules/vendor-categories`'s shape):

- `types.ts` — `MilestoneTemplate { id: string; name: string; daysBeforeEvent: number; sortOrder:
  number }`.
- `schemas/milestone-template.schema.ts` — Zod: `name` (min 3 chars, same rule as
  `project-milestone.schema.ts`), `daysBeforeEvent` (int, `min(0)`).
- `stores/useMilestoneTemplateStore.ts` — `templates`, `fetchTemplates`, `createTemplate`,
  `updateTemplate`, `deleteTemplate`, `reorderTemplates` — Zustand, shaped like
  `useVendorCategoryStore` but without its pagination half (an un-paginated short list, like a
  project's own Timeline tab).
- `components/MilestoneTemplateFormModal.tsx` — mirrors `VendorCategoryFormModal.tsx`; fields: Nama,
  Hari Sebelum Acara (H-).
- `pages/MilestoneTemplateListPage.tsx` — mirrors `VendorCategoryListPage.tsx` structurally (header +
  add button + table/card-list + error banner), but: no search bar (short un-paginated list), Up/Down
  reorder buttons per row (mirroring `ProjectMilestonesSection.tsx`'s existing reorder UI) instead of a
  status column, and a straight delete action with an inline confirm row (mirroring
  `ProjectVendorsSection.tsx`'s `confirmingCancel` pattern) instead of an activate/deactivate toggle.

Routing:

- `route-paths.ts`: add `milestoneTemplates: "/timeline-default"`.
- `api-endpoints.ts`: add a `milestoneTemplates: { base: "/api/v1/milestone-templates", item: (id:
  string) => \`/api/v1/milestone-templates/${id}\`, reorder: "/api/v1/milestone-templates" }` group.
- `protected.routes.tsx`: add `MilestoneTemplateListPage` into the existing `RequireRole
  allow={["Owner"]}` block (line ~61), alongside `vendorCategories`/`users`/`subscription`.
- `Sidebar.tsx`: add `{ kind: "link", to: ROUTE_PATHS.milestoneTemplates, label: "Timeline Default",
  icon: CalendarClock, allowedRoles: ["Owner"] }` inside the existing Pengaturan group's `children`
  array, after `Kategori Vendor`.

## 4. Order of implementation

1. Migration (schema + tenant backfill) + domain + repository.
2. Application service (`MilestoneTemplateService` + `SeedDefaults`) + rewire
   `ProjectService.Create()`'s `seedDefaultMilestones` to read from the repository instead of the
   hardcoded var.
3. Contracts + two-phase wiring (`SetProjects` in `platform`, `main.go` call site) +
   `tenant_service.go`'s `Register()` call.
4. Presentation (new handler + routes registered in `projects.module.go`).
5. Backend verification: `go build ./...`, `go vet ./...`; apply the migration locally
   (`npm run migrate:up`) and confirm every existing tenant row got the 6 backfilled items.
6. Frontend module scaffold (types/schema/store/components/page).
7. Routing (`route-paths.ts`, `api-endpoints.ts`, `protected.routes.tsx`) + `Sidebar.tsx` entry.
8. Frontend verification: `npx tsc --noEmit -p apps/web/tsconfig.json`, `npm run build -w apps/web`.
9. Manual pass (as Owner): edit the template (add/edit/delete/reorder items), create a new project,
   confirm its Timeline tab matches the edited template exactly; confirm Duplikat Project still clones
   the source project's actual milestones, unaffected by template edits; confirm Admin and Wedding
   Planner never see "Timeline Default" in the sidebar and get 403 hitting the API directly.

## 5. Verification

- `go build ./...` / `go vet ./...` after backend changes.
- Apply migration locally, spot-check `SELECT * FROM project_milestone_templates WHERE tenant_id = X`
  for an existing tenant to confirm the 6-item backfill landed correctly.
- `npx tsc --noEmit -p apps/web/tsconfig.json` and `npm run build -w apps/web` after frontend changes.
- Manual pass through Pengaturan → Timeline Default as Owner: add, edit, reorder (up/down), delete
  down to zero and back up.
- Create a new project and confirm its Timeline tab exactly matches the current template (names, and
  target dates computed from each item's `daysBeforeEvent`, clamped to `prepStartDate` for short-notice
  projects same as before).
- Duplicate an existing project and confirm its Timeline is cloned from the *source project*, not from
  the tenant's current template (i.e., editing the template afterward must not retroactively change
  what a duplicate produces).
- Confirm Admin and Wedding Planner (Staff) roles: menu item absent from Sidebar, and direct API calls
  to `/api/v1/milestone-templates` return 403.
- Confirm a brand-new tenant registration still seeds the same 6 starter items via
  `SeedDefaultMilestoneTemplate`, exercised the same way `SeedDefaultCategories` already is.

# PLAN — Vendor Pricing Tier (Akad / Akad + Resepsi) & Project Margin

## 0. Goal

Close the disconnect between a Vendor's own preset prices (`priceAkad`/`priceAkadResepsi`, added per
the "DATA VENDOR" slide) and Project Vendor's "Nilai Kerja Sama," which today is a fully manual number
with zero link to those preset prices. When adding a vendor to a project, staff picks a package tier
("Akad Saja" / "Akad + Resepsi") that auto-fills the contract value from the vendor's own price —
staff can still override for negotiated deals. On top of that, surface a **Margin/Keuntungan** figure
on the project (Nilai Kontrak minus every cost sourced for that project), turning what's currently
disconnected numbers into an actual profitability view.

## 1. Decisions confirmed

1. **Package tier is chosen per vendor engagement**, not once for the whole project. A project's own
   `packageName` field stays free text, untouched by this plan — different vendors on the same project
   legitimately have different scopes (e.g., MC/entertainment booked resepsi-only while catering
   covers both), so a single project-wide tier would be wrong.
2. **The tier is stored, not just a one-time UI convenience.** `ProjectVendor` gains a `pricingTier`
   field recording which of the vendor's two prices this engagement started from — so re-opening the
   edit modal later still shows "Paket: Akad + Resepsi" as context, even after `contractValue` has been
   negotiated away from that starting number.
3. **`contractValue` stays freely editable** — auto-fill is a *starting point*, not a lock. Real
   engagements get negotiated (discounts, bundling, custom add-ons described in the free-text "Scope"
   field), and `dpAmount`/`paidAmount` (actual payment-progress tracking) can never be derived from
   vendor master data regardless. Once staff manually edits the Nilai Kerja Sama field directly, further
   vendor/tier changes stop overwriting it (last explicit edit wins).
4. **Margin = Nilai Kontrak (Project) − Σ Nilai Kerja Sama of every non-Cancelled vendor engagement −
   Venue cost** (`rentalPrice + charge`, `0` if no venue attached). Cancelled engagements are excluded
   — a called-off engagement isn't a real cost, mirroring how this codebase's own milestone stats
   already exclude `Cancelled` from every "relevant" computation.
5. **Venue's cost is included in the deduction.** Venue was extracted into its own directory (ADR-0016)
   but is still, economically, a cost the WO incurs for the project — omitting it would systematically
   overstate margin by exactly the venue's rental price, usually one of the largest line items.
6. **Margin is computed frontend-side, not added to any backend contract.** `vendorscontracts.VenueSummary`
   (the cross-module type shared with Client Portal's own Venue tab — see `getProjectVenue` in
   `project_endpoints.go`) deliberately carries no price fields today. Extending it to include
   `rentalPrice`/`charge` would leak the WO's internal cost data straight to the Client Portal, which
   must never see it. `ProjectHeaderCard.tsx` (WO Console, staff-only) already has a precedent for
   fetching the *full* Venue record directly — `ProjectVenueTabPage.tsx` does exactly this via
   `useVenueStore.fetchVenue(venueId)` (staff-only `GET /venues/{id}`), not through the public-safe
   project-venue endpoint. This plan reuses that same direct fetch, purely for the margin calculation,
   so no backend change is needed for the margin itself — only for persisting `pricingTier`.
7. **Displayed on `ProjectHeaderCard.tsx`**, directly beside the existing "Nilai Kontrak" field — the
   natural home since that's already where the project's financial summary lives.

## 2. Current state

- `apps/web/src/modules/projects/components/ProjectVendorFormModal.tsx` — "Nilai Kerja Sama (Rp)" is a
  bare `Input type="number"` (`project-vendor.schema.ts:18`), no reference to the selected vendor's
  `priceAkad`/`priceAkadResepsi` at all. The Vendor `<Select>` only shows names.
- `domain.ProjectVendor` (`apps/api/internal/modules/projects/domain/vendor_engagement.go`) has no
  pricing-tier concept — just a flat `ContractValue int64`.
- `domain.Vendor` (`apps/api/internal/modules/vendors/domain/vendor.go`) already has `PriceAkad
  *int64` / `PriceAkadResepsi *int64` (and the frontend `Vendor` type in
  `apps/web/src/modules/vendors/types.ts:16-17` already exposes both as `number | null`) — this data is
  simply never read anywhere in the Project Vendor flow today.
- No margin/profit/keuntungan concept exists anywhere in the codebase today (confirmed by a full-repo
  search) — `Project.ContractValue` ("Nilai Kontrak") is displayed as a flat, disconnected number in
  `ProjectHeaderCard.tsx:200`.
- `vendorscontracts.VenueSummary` (`apps/api/internal/modules/vendors/contracts/contracts.go:17-26`) —
  the type `ProjectService.GetVenue` returns for `GET /projects/{id}/venue` — has no price fields,
  confirmed deliberately absent (it's the same shape Client Portal's own Venue tab consumes).

## 3. Design

### 3.1 Database

New migration `apps/api/migrations/000025_add_pricing_tier_to_project_vendors.up.sql` / `.down.sql`:

```sql
-- up
ALTER TABLE project_vendors ADD COLUMN pricing_tier VARCHAR(20) NOT NULL DEFAULT 'Akad' AFTER contract_value;

-- down
ALTER TABLE project_vendors DROP COLUMN pricing_tier;
```

Existing rows default to `'Akad'` — purely an informational label going forward (it never changes
`contract_value`), so an arbitrary default on old rows is harmless.

### 3.2 Backend — domain

`apps/api/internal/modules/projects/domain/vendor_engagement.go`: add

```go
type VendorPricingTier string

const (
	PricingTierAkad        VendorPricingTier = "Akad"
	PricingTierAkadResepsi VendorPricingTier = "AkadResepsi"
)
```

and a `PricingTier VendorPricingTier` field on `ProjectVendor` (right after `ContractValue`, mirroring
the migration's column position).

### 3.3 Backend — repository

`apps/api/internal/modules/projects/infrastructure/mysql_vendor_engagement_repository.go`: add
`pricing_tier` to `projectVendorColumns`, `scanProjectVendor`, and both the `Create`/`Update` SQL
(same column-list-extension shape as every other field already there).

### 3.4 Backend — application

`apps/api/internal/modules/projects/application/vendor_engagement_service.go`'s
`VendorEngagementInput` gains `PricingTier domain.VendorPricingTier`, threaded through unchanged in
`Create`/`Update` (both already build `*domain.ProjectVendor` field-by-field from the input — just add
one more assignment each, no other logic changes).

`project_service.go`'s `cloneVendorEngagementsFrom` (used by Duplicate Project) explicitly lists
fields when building each clone — add `PricingTier: pv.PricingTier` there too, or a duplicated project
silently loses which tier every one of its cloned vendor engagements started from.

No new validation layer for the enum value itself — `EngagementStatus` already goes through the same
`domain.EngagementStatus(body.X)` bare cast with no backend-side allow-list check today, so
`PricingTier` follows the same existing (lax) convention rather than introducing inconsistent rigor
for one field only. The frontend Zod enum is the actual gate in practice, same as it already is for
every other enum-shaped field in this form.

### 3.5 Backend — presentation

`apps/api/internal/modules/projects/presentation/vendor_engagement_endpoints.go`'s
`vendorEngagementInputBody` gains `PricingTier string \`json:"pricingTier"\``, threaded into
`toVendorEngagementInput`. `apps/api/internal/modules/projects/presentation/dto.go`'s
`projectVendorResponse` gains `PricingTier string \`json:"pricingTier"\`` and `toProjectVendorResponse`
passes it through — same shape as every other string-backed enum field already on this DTO.

### 3.6 Frontend — pricing tier + auto-fill

- `apps/web/src/modules/projects/types.ts`: `ProjectVendor` gains `pricingTier: "Akad" | "AkadResepsi"`.
- `apps/web/src/modules/projects/schemas/project-vendor.schema.ts`: add
  `pricingTier: z.enum(["Akad", "AkadResepsi"])`.
- `apps/web/src/modules/projects/stores/useProjectStore.ts`: extend the existing raw→`ProjectVendor`
  mapping function to pass `pricingTier` through unchanged (same as every other passthrough field).
- `apps/web/src/modules/projects/components/ProjectVendorFormModal.tsx`:
  - New "Paket" `<Select>` (Akad Saja / Akad + Resepsi) next to the Vendor field, defaulting to
    `"Akad"` for a brand-new engagement.
  - Show the selected vendor's own `priceAkad`/`priceAkadResepsi` as small reference text near the
    Vendor `<Select>` (e.g. "Harga Akad: Rp X · Harga Akad+Resepsi: Rp Y", or "Harga belum diisi" if
    both are `null`) — closes the "invisible price" gap even before any auto-fill happens.
  - A `contractValueTouched` boolean (reset whenever the modal opens fresh) flips to `true` the moment
    the user edits the Nilai Kerja Sama input directly. An effect watching `vendorId`/`pricingTier`
    auto-fills `contractValue` from the matching vendor price *only while `!contractValueTouched`* and
    the matching price isn't `null` — so picking a vendor/tier sets a sensible starting number, but
    never clobbers a value the user already typed themselves.

### 3.7 Frontend — Margin display

`apps/web/src/modules/projects/components/detail/ProjectHeaderCard.tsx`:

- Add a venue fetch mirroring `ProjectVenueTabPage.tsx`'s own pattern exactly: local `venue: Venue |
  null` state, a `useEffect` keyed on `project.venueId` calling `useVenueStore`'s `fetchVenue` when set
  (and clearing to `null` when not) — this file doesn't currently fetch venue data at all.
- Compute, from data already in scope:
  ```ts
  const vendorCost = vendorEngagements
    .filter((v) => v.engagementStatus !== "Cancelled")
    .reduce((sum, v) => sum + v.contractValue, 0);
  const venueCost = venue ? (venue.rentalPrice ?? 0) + (venue.charge ?? 0) : 0;
  const margin = project.contractValue - vendorCost - venueCost;
  ```
- Add `<InfoField label="Margin/Keuntungan" value={formatCurrency(margin)} />` immediately after the
  existing `"Nilai Kontrak"` field (`ProjectHeaderCard.tsx:200`).

No backend change backs this — everything it needs (`project.contractValue`, `vendorEngagements`,
venue's `rentalPrice`/`charge`) is already reachable from data this page fetches or can fetch through
existing staff-only endpoints, per §1.6.

## 4. Order of implementation

1. Migration (add `pricing_tier` column, default `'Akad'` for existing rows) + domain field/enum.
2. Repository (columns/scan/Create/Update).
3. Application (`VendorEngagementInput.PricingTier` passthrough in Create/Update; `cloneVendorEngagementsFrom`
   carries it into duplicated projects).
4. Presentation (input body + response DTO).
5. Backend verification: `go build ./...`, `go vet ./...`; apply the migration locally and confirm
   existing `project_vendors` rows show `pricing_tier = 'Akad'`.
6. Frontend: `types.ts`, `project-vendor.schema.ts`, `useProjectStore.ts` passthrough.
7. `ProjectVendorFormModal.tsx`: Paket select, reference price text, auto-fill-until-touched logic.
8. `ProjectHeaderCard.tsx`: venue fetch + margin computation + `InfoField`.
9. Frontend verification: `npx tsc --noEmit -p apps/web/tsconfig.json`, `npm run build -w apps/web`.
10. Manual pass (see §5).

## 5. Verification

- `go build ./...` / `go vet ./...` after backend changes; apply migration locally and spot-check
  `SELECT pricing_tier FROM project_vendors LIMIT 5` shows `'Akad'` on pre-existing rows.
- `npx tsc --noEmit -p apps/web/tsconfig.json` and `npm run build -w apps/web` after frontend changes.
- Add a vendor with "Akad Saja" selected → Nilai Kerja Sama auto-fills to that vendor's `priceAkad`.
  Switch the Paket selector to "Akad + Resepsi" → refills to `priceAkadResepsi`. Manually edit Nilai
  Kerja Sama, then change Paket again → the manual value must **not** be overwritten.
- Add a vendor whose `priceAkad`/`priceAkadResepsi` are both unset → confirm no auto-fill happens and
  no crash (reference text shows "Harga belum diisi").
- On a project with a venue attached (with `rentalPrice`/`charge` set) and 2–3 vendor engagements
  (mix of active and one Cancelled) — hand-compute the expected margin and confirm
  `ProjectHeaderCard`'s "Margin/Keuntungan" matches exactly, and that the Cancelled engagement's
  `contractValue` is excluded.
- Detach the venue from the project → margin recalculates without the venue cost term.
- Duplicate a project that has vendor engagements with different pricing tiers → confirm the
  duplicate's engagements retain the same `pricingTier` each cloned from.
- Confirm Client Portal's own Venue tab is visually and functionally unchanged — `VenueSummary` was not
  touched by this plan, so no cost data is newly exposed there.

# PLAN — Client Payments ("Uang Masuk dari Client")

## 0. Goal

Today's "Pembayaran" tab on a project tracks only money going **out** to vendors
(`vendor_payments`). Add a second, distinct concept — money coming **in** from the client, against
the project's own Nilai Kontrak, paid in installments (DP/Termin/Pelunasan/Tambahan/Refund) — living
in the same tab, and switch the Client Portal's own Pembayaran tab from showing vendor-payment data
(not very relevant to a client) to the client's own payment history (directly relevant). Explicit
design constraint from the user: **this must land as an easy-to-use, easy-to-understand feature** —
every design choice below is made with that in mind, not just "correct."

## 1. Decisions confirmed

1. **One tab, two sections** — no new tab added to Project Detail's navigation. "Uang Masuk dari
   Client" (new) sits above "Pembayaran ke Vendor" (today's feature, relabeled from "Pembayaran
   Vendor" for clarity, otherwise untouched).
2. **A new, separate table** (`client_payments`), not a modification of `vendor_payments`. These are
   opposite accounting directions (receivable vs. payable) — conflating them into one table with a
   "direction" flag would make every future query/report need to filter by direction, for no benefit
   over just having two tables. `client_payments` has no `project_vendor_id` at all (it belongs to
   the project itself, not any vendor) — structurally simpler than `vendor_payments`, not a subset of it.
3. **Reuses `domain.PaymentType`** (`DP`/`Termin`/`Pelunasan`/`Tambahan`/`Refund`) — the same five
   tranche concepts apply directly to a client's own installment schedule, so no new enum is needed.
4. **Evidence: one slot, not two, and attachable in the same step as recording the payment.**
   Unlike a vendor payment (which asks for both an Invoice and a Transfer Proof — and, per §2 below,
   has never actually been attachable at creation time in this codebase, a known standing gap), a
   client payment only ever needs a transfer proof — there's no "invoice from the WO to itself."
   Uses the existing polymorphic evidence mechanism (`related_kind`/`related_id`, already used
   elsewhere) with a new `relatedKind` value, `clientPayment` — not `vendor_payments`' own two direct
   FK columns (`invoiceEvidenceId`/`proofEvidenceId`), which is an inconsistent, harder-to-follow
   pattern this plan doesn't repeat. Crucially, the "Tambah Pembayaran Client" form includes the file
   picker directly — one submit both creates the payment record and attaches its proof, closing the
   exact "always incomplete in practice" gap `vendor_payments` has today (§2) rather than reproducing it.
5. **Client Portal's Pembayaran tab is repurposed**, not left alongside a new one: it stops showing
   vendor-payment data (WO's own cost management, not something a client needs visibility into) and
   shows the client's own payment history instead — how much they've paid, how much is left, exactly
   the two questions a bride/groom actually has.
6. **Visible to Wedding Planner** — this is day-to-day operational status ("has the client paid this
   installment"), not a profitability figure like Margin/Keuntungan, so it follows the same access
   Wedding Planner already has to `project_vendors.dpAmount`/`paidAmount`, not Margin's Owner/Admin-only gate.
7. **Recommended, flagged as optional — a new "Sisa Tagihan Client" figure on `ProjectHeaderCard`**,
   alongside the existing Nilai Kontrak/Margin fields: Nilai Kontrak − Total Diterima Client. This
   wasn't explicitly asked for, but it's the natural next number once client payments are tracked at
   all, and completes the financial story already started by Margin. Drop it from scope if unwanted —
   nothing else in this plan depends on it.

## 2. Current state (what exists today, that this plan must not break)

- `vendor_payments` (`domain/payment.go`, `VendorPayment` struct) — tied to one `project_vendor_id`,
  types `DP/Termin/Pelunasan/Tambahan/Refund`. `IsEvidenceComplete()` (payment.go:32-37): a `Refund`
  needs only `ProofEvidenceID`; every other type needs both `InvoiceEvidenceID` and `ProofEvidenceID`.
- **Standing gap, confirmed, not touched by this plan**: `PaymentService.Create`
  (`application/payment_service.go:39`) never sets `InvoiceEvidenceID`/`ProofEvidenceID`, and no
  endpoint exists to set them afterward either (`API_CONTRACT.md` already documents this: "always
  incomplete in practice"). `ProjectPaymentsSection.tsx`'s displayed totals (`totalContractValue`/
  `totalPaid`/`totalRemaining`, lines 38-40) are summed from `project_vendors.contractValue`/
  `paidAmount` — **not** from actually summing `vendor_payments` rows, a second pre-existing quirk
  (the ledger and the running totals are two independently-maintained numbers today). Both are
  called out here so the new client-payment feature isn't built the same way by copy-paste habit.
- `evidence.related_kind` is currently `ENUM('vendorMilestone','payment','projectVendor','issue')`
  (`domain/evidence.go:24-28`) — a `'payment'` value already exists but, per the point above, is
  never actually populated by anything (dead in practice, since `vendor_payments` uses its own direct
  columns instead). This plan does not repurpose `'payment'` for client payments — a fresh
  `'clientPayment'` value keeps the two concepts unambiguous or, and doesn't inherit `'payment'`'s
  history of never actually being wired up to anything.
- `Project.ContractValue` has no "amount collected"/"outstanding" concept anywhere today — confirmed
  by a full-repo search (no `amountPaid`, `clientPayment`, `receivable`, `tagihan` hits before this
  plan).
- Client Portal's `PembayaranTabPage.tsx` today renders the **same** `payments`/`vendorEngagements`
  data as staff's own vendor-payment view (three summary cards computed the same way, a card list
  per `vendor_payments` row with Invoice/Bukti Transfer evidence buttons) — i.e., a client's own money
  is nowhere in this picture at all today; only the WO's spend on vendors is shown.

## 3. Design

### 3.1 Database

New migration `apps/api/migrations/000027_create_client_payments_table.up.sql` / `.down.sql`:

```sql
-- up
CREATE TABLE client_payments (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  project_id BIGINT UNSIGNED NOT NULL,
  type ENUM('DP','Termin','Pelunasan','Tambahan','Refund') NOT NULL,
  amount BIGINT UNSIGNED NOT NULL,
  payment_date DATE NOT NULL,
  method VARCHAR(100) NOT NULL,
  reference_number VARCHAR(100) NOT NULL,
  notes TEXT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (project_id) REFERENCES projects(id),
  INDEX idx_client_payments_project (project_id)
);

ALTER TABLE evidence MODIFY COLUMN related_kind
  ENUM('vendorMilestone','payment','projectVendor','issue','clientPayment') NOT NULL;
```

Same-module FK to `projects.id` (both tables live in `projects`, so a real constraint is allowed —
mirrors `project_milestones.project_id`). No `project_vendor_id` column at all, unlike
`vendor_payments` — this table doesn't belong to any vendor.

### 3.2 Backend — domain

`apps/api/internal/modules/projects/domain/payment.go`: add (reusing the existing `PaymentType`)

```go
type ClientPayment struct {
	ID              int64
	ProjectID       int64
	Type            PaymentType
	Amount          int64
	PaymentDate     time.Time
	Method          string
	ReferenceNumber string
	Notes           string
}
```

No `IsEvidenceComplete()` method needed on this struct — evidence-completeness for a client payment
is a simple existence check (§3.4), not the two-field dual condition `VendorPayment` has.

`apps/api/internal/modules/projects/domain/evidence.go`: add `RelatedClientPayment
EvidenceRelatedKind = "clientPayment"` alongside the existing four values.

### 3.3 Backend — repository

New file `apps/api/internal/modules/projects/infrastructure/mysql_client_payment_repository.go`,
`MySQLClientPaymentRepository` — `ListByProject`, `FindByID`, `Create`. Same shape as
`mysql_payment_repository.go`, minus every `project_vendor_id`/evidence-ID column.

### 3.4 Backend — application

New file `apps/api/internal/modules/projects/application/client_payment_service.go`:
`ClientPaymentRepository` interface, `ClientPaymentService` (`List`, `Create`), `ClientPaymentInput`
(`Type, Amount, PaymentDate, Method, ReferenceNumber, Notes`). `Create` logs activity reusing the
existing `domain.ActivityPaymentRecorded` type (same as vendor payments already do) with
`entity_type: "client_payment"` and description `"Pembayaran client dicatat"` — distinguishing the
two only by `entity_type`/description text, the same reuse pattern `ActivityMilestoneUpdated`
already has across Project Milestones and Vendor Milestones. No new `ActivityType` value needed.

Evidence-completeness (used by the response DTO, §3.5) is computed by the **presentation** layer by
checking whether any fetched `evidence` row has `relatedKind: "clientPayment"` and `relatedId` equal
to the payment's id — mirroring how the frontend already checks milestone evidence completeness
elsewhere, not a new backend concept.

### 3.5 Backend — presentation

New file `apps/api/internal/modules/projects/presentation/client_payment_endpoints.go`:
- `GET /projects/{id}/client-payments` — list, response `{id, type, amount, paymentDate, method,
  referenceNumber, notes}` (no vendor field — there is none).
- `POST /projects/{id}/client-payments` — body `{type, amount, paymentDate, method,
  referenceNumber, notes}`. Same `resolveProjectAccess` gate every other `/projects/{id}/...`
  sub-resource already goes through (Wedding Planner PIC-scoped, any staff role can create — matches
  `vendor_payments`' own unrestricted-by-role create today, §1.6).

Routed in `handler.go`'s existing big `Item` switch, alongside the `payments` cases (`rest[0] ==
"client-payments"`), not a new dispatcher — same file, same pattern, immediately next to the
existing `payments` routes for discoverability.

### 3.6 Frontend — data layer

- `apps/web/src/modules/projects/types.ts`: add `ClientPayment { id, type, amount, paymentDate,
  method, referenceNumber, notes }`.
- `apps/web/src/modules/projects/schemas/client-payment.schema.ts` (new, simpler than
  `payment.schema.ts` — no `projectVendorId` field): `{ type, amount, paymentDate, method,
  referenceNumber, notes, proofFile? }` — `proofFile` is a plain in-memory `File`/undefined, validated
  only client-side (optional), never sent as part of the JSON body itself (see below).
- `apps/web/src/modules/projects/stores/useProjectStore.ts`: add `clientPayments: ClientPayment[]`
  state, `fetchClientPayments(projectId)`, and `createClientPayment(projectId, values)` — the latter
  does the two-call orchestration behind one action (§1.4): `POST .../client-payments` first, then
  (only if `values.proofFile` is set) `compressFileForUpload` + `POST .../evidence` with
  `relatedKind: "clientPayment"`, `relatedId: <new payment id>` — mirroring the exact
  compress-then-upload two-step already used by `VendorMilestoneEditModal`'s own evidence flow, just
  triggered automatically from one store action instead of a second manual "Tambah Evidence" click.
- `apps/web/src/shared/services/api-endpoints.ts`: add `clientPayments: (id) =>
  \`/api/v1/projects/${id}/client-payments\`` under `API.projects`.

### 3.7 Frontend — "Uang Masuk dari Client" (new component)

New file `apps/web/src/modules/projects/components/detail/ClientPaymentsSection.tsx` — same visual
language as `ProjectPaymentsSection.tsx` (Card, summary stats row, Table/CardList + Pagination,
"Tambah" button + Modal), deliberately **simpler**, per the "mudah digunakan" goal:
- 3 summary stats: **Nilai Kontrak** (`project.contractValue`), **Total Diterima** (sum of
  non-Cancelled — n/a here, all rows count — client payments, `Refund` subtracted rather than
  added), **Sisa Tagihan** (Nilai Kontrak − Total Diterima).
- Table/CardList columns: Jenis, Nominal, Tanggal, Metode, No. Referensi, Bukti (single badge: Ada/
  Belum Ada — not the two-part "Lengkap/Belum Lengkap" vendor payments show, since there's only one
  evidence slot to begin with, nothing to be "partially" complete about).
- "Tambah Pembayaran Client" modal: **one fewer field than the vendor version** (no Vendor picker —
  nothing to pick), plus one new optional field, **Bukti Transfer** (a plain file input, same
  accept-list convention as every other attachment in this app — image/pdf), submitted together with
  the rest in one "Simpan" click.
- No "Kelengkapan Evidence" ambiguity to explain to a user: the badge is binary (uploaded or not),
  matching whether the optional field was filled in, not a derived multi-field rule.

`apps/web/src/modules/projects/pages/tabs/ProjectPaymentsTabPage.tsx` renders **both** sections
stacked, `ClientPaymentsSection` first: this is the order the tab's own subtitle/mental model should
follow ("money in, then money out"), and matches where the equivalent info now sits on
`ProjectHeaderCard` (Nilai Kontrak → Sisa Tagihan → Margin, left to right, §3.9).

`ProjectPaymentsSection.tsx` itself: only its `CardHeader`'s `title` changes, `"Pembayaran Vendor"` →
`"Pembayaran ke Vendor"` — everything else (fields, totals, evidence-complete logic) stays exactly
as it is today; this plan does not touch the vendor-payment gap noted in §2, since fixing that
wasn't asked for and would be unrelated scope creep on top of an already-large plan.

### 3.8 Frontend — Client Portal (repurposed)

`apps/web/src/modules/client-portal/pages/tabs/PembayaranTabPage.tsx` is rewritten, not extended:
fetches `clientPayments` instead of `payments`/`vendorEngagements`, keeps the exact same three-card
visual layout (Total Nilai Kerja Sama → **Nilai Kontrak**, Total Sudah Dibayar → **Total Diterima**,
Sisa Pembayaran → **Sisa Tagihan** — same colors/emphasis, new source numbers) and the same
per-payment card list styling, but each card now shows a single "Bukti Transfer" button (if
attached) instead of the Invoice/Bukti Transfer pair, and drops the vendor-name line entirely (a
client payment has no vendor). The intro copy ("Halaman ini menampilkan setiap pembayaran... kepada
vendor...") is rewritten to describe the client's own payment history instead.

### 3.9 Frontend — optional "Sisa Tagihan Client" on `ProjectHeaderCard` (§1.7)

`apps/web/src/modules/projects/components/detail/ProjectHeaderCard.tsx`: fetch `clientPayments` the
same way `vendorEngagements` already is (existing mount effect), compute `totalReceived = sum(type
!= 'Refund') - sum(type == 'Refund')`, `outstanding = project.contractValue - totalReceived`, and
add `<InfoField label="Sisa Tagihan Client" value={formatCurrency(outstanding)} />` immediately after
the existing "Nilai Kontrak" field and before "Margin/Keuntungan" — reading left to right as: what
was agreed → what's still owed → what's left after costs. Visible to every role that already sees
Nilai Kontrak (including Wedding Planner, §1.6) — not gated behind `canSeeMargin`.

## 4. Order of implementation

1. Migration (`client_payments` table + `evidence.related_kind` ENUM extension) + domain additions.
2. Repository + application service (`ClientPaymentService`).
3. Presentation (new endpoints, wired into the existing `Item` routing switch).
4. Backend verification: `go build ./...`, `go vet ./...`; apply migration locally.
5. Frontend data layer (`types.ts`, schema, store actions, `api-endpoints.ts`).
6. `ClientPaymentsSection.tsx` (new) + `ProjectPaymentsTabPage.tsx` (stack both sections) +
   `ProjectPaymentsSection.tsx`'s title-only rename.
7. Client Portal's `PembayaranTabPage.tsx` rewrite.
8. `ProjectHeaderCard.tsx`'s "Sisa Tagihan Client" addition (§3.9) — confirm with the user first if
   they'd rather skip it, per §1.7's "optional" framing.
9. Frontend verification: `npx tsc --noEmit -p apps/web/tsconfig.json`, `npm run build -w apps/web`.
10. Manual pass (§5).

## 5. Verification

- `go build ./...` / `go vet ./...`; apply migration locally, confirm `client_payments` exists and
  `evidence.related_kind` accepts `'clientPayment'`.
- `npx tsc --noEmit -p apps/web/tsconfig.json` and `npm run build -w apps/web`.
- Record a client payment with a proof file attached in one submit — confirm both the payment row
  and its evidence row exist afterward, linked correctly (`relatedKind: "clientPayment"`), and the
  Bukti badge shows "Ada" immediately without a page refresh.
- Record one without a file — confirm it saves fine and shows "Belum Ada," recoverable later only by
  the general `ProjectEvidenceSection` (no dedicated "attach evidence after the fact" flow for client
  payments either, matching the same scope boundary vendor payments already have, §2).
- Confirm the two Payments-tab sections are visually distinct and independently paginated/scoped —
  adding a client payment must not affect the vendor-payment table below it, and vice versa.
- Confirm a `Refund`-type client payment reduces (not increases) "Total Diterima"/"Sisa Tagihan" —
  hand-compute against 3-4 mixed-type rows.
- Log in as Wedding Planner on their own PIC'd project: confirm "Uang Masuk dari Client" is visible
  (unlike Margin/Keuntungan, which stays hidden).
- Client Portal: confirm the Pembayaran tab now shows the client's own payment history (no vendor
  names, no vendor-payment data at all), same visual polish as before.
- Confirm `ProjectPaymentsSection.tsx`'s (vendor) behavior is byte-for-byte unchanged apart from its
  title string — same fields, same totals, same known evidence-completeness gap from §2 (deliberately
  not fixed here).

