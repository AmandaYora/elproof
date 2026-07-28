# ADR-0012: Per-tenant branding — logo, fixed color presets, and where it does/doesn't apply

## Status
Accepted

## Context
Every tenant saw identical ElProof branding inside an authenticated session: a hardcoded `APP_NAME`
text string and one static navy color scheme. The user wants a tenant's own staff (WO Console) and
a tenant's own clients (Client Portal) to feel like they're using the WO's own product, not a
visibly third-party one — while keeping this a config choice a superadmin makes per tenant, not a
free-form white-label system.

## Decision

### Scope: a logo + one of 20 fixed color presets, not free-form
No hex color picker, no arbitrary logo placement/sizing rules — a tenant picks one of a fixed set
of named presets (`navy` default, plus 19 others; see below) and optionally uploads a logo. This
bounds the design-quality/contrast problem to something verifiable once per preset (see below)
instead of once per tenant's arbitrary choice.

### Mechanism: runtime CSS-variable override, not a component refactor
Exactly 4 values carry brand identity: `--brand-navy-950/900/800` (Sidebar background, primary
buttons, hover/active-nav state) + `--color-primary-soft` (tinted backgrounds) —
`apps/web/src/theme/theme.css`. Every other color (success/warning/danger/info, background, text,
border) is a fixed semantic color and never varies per tenant. Because 46 files' `bg-navy-*`/
`hover:bg-navy-*` Tailwind classes already resolve through these 4 variables (not a baked-in hex),
overriding them on `document.documentElement` after login re-themes the entire app with zero
component changes. `apps/web/src/theme/brandPresets.ts` is the single source of truth for all 20
presets' 4 values each; `apps/api/internal/modules/platform/domain/brand_preset.go`'s
`AllowedBrandColorPresets` only needs to agree on the valid *keys* — hex values live only on the
frontend, so adding/adjusting a preset is never a migration.

### Contrast is a hard constraint on every preset, not an aesthetic afterthought
Sidebar/button text is always white. An early version of 6 presets (gold, orange, mustard, yellow,
sky, pink) was tuned to look "vivid" by eye and shipped with white-text contrast as low as ~2.1:1
against WCAG AA's 4.5:1 minimum for normal text — found only when checked with the actual
relative-luminance formula, not visually. **Every preset's `900` role (used on every primary button
app-wide) must clear ~4.5:1+ against white before shipping**, `800`/hover role at least ~3:1 (a
momentary state gets a lower bar). For Tailwind-sourced hues, family shades `700` and darker are
usually safe; `500` and lighter routinely fail for warm/light hues (yellow-through-green) even
though the exact same shade number is fine for cooler hues (blue/red/purple/indigo) — verify per
hue, don't assume a shade number is safe just because it worked for a different color.

### Self-service read is broader than the existing Owner-only tenant read
`GET /tenants/me` (subscription/billing data) stayed Owner-only by design. Branding needed a
**separate**, more permissive self-service pair — `GET /tenants/me/branding` /
`GET /tenants/me/logo` — open to any tenant-scoped principal (any staff role, or `client`), since
every staff member and every client needs to see their own tenant's branding, not just the Owner.
Both resolve the tenant strictly from the JWT's `tenant_id` claim, never a request parameter.

### Who configures it: platform-admin only, via the existing tenant edit form
No tenant self-service branding settings page (yet) — logo/color are set by a superadmin editing
the tenant, the same place/flow as every other tenant field. Straightforward to add a tenant-side
settings page later on top of the same backend endpoints if wanted.

### Where it deliberately does NOT apply
- **Platform Console** (superadmin's own backoffice) — never branded; a superadmin isn't "inside"
  any one tenant, so there's no single tenant color to apply.
- **Login page** — no tenant is known before authentication succeeds (login resolves by username/
  email, not by subdomain — see ADR-0004), so a *literal* tenant-specific look there is impossible
  without domain-based tenant resolution (a separate, larger feature — see "Related, not yet built"
  below). Initially left ElProof-branded (navy + "ElProof" wording) as the simplest choice; later
  neutralized (slate gray, no "ElProof" text, generic lock icon, a new `Button` `"neutral"` variant
  deliberately not tied to `--brand-navy-*`) since showing *ElProof's own* brand color/name there
  undercuts the "feels like the tenant's own app" goal just as much as showing the wrong tenant's
  branding would.
- **Marketing/homepage pages** and the pre-login browser tab title — both stay ElProof's own
  identity on purpose (that's their actual purpose); not touched by this feature.

### Fallback behavior (no surprises for an unconfigured tenant)
`brand_color_preset` defaults to `'navy'` (identical hex to the pre-existing hardcoded look) and
`logo_storage_path` is nullable — shipping the migration alone changes nothing visually for any
existing tenant. With no logo, the header falls back to the tenant's own `businessName` (not
`APP_NAME`) — an unbranded tenant still never sees "ElProof" in its own Sidebar/Client Portal
header, only its own name in plain text.

### Also covered by this ADR: tab title + favicon
The browser tab title and favicon are the last "ElProof" touchpoints *inside* an authenticated
session. `apps/web/src/theme/tabIdentity.ts` sets `document.title` to `"{businessName} —
{consoleLabel}"` and lazily creates a `<link rel="icon">` pointed at the same logo object URL
already fetched for the header image — reusing it rather than requiring a separate favicon upload.

## Consequences
- Logo lives in the existing S3-compatible object storage (ADR-0006 revision), never inline
  base64/DB, never a public URL — streamed through an authenticated Go handler like evidence is.
- Adding a 21st preset (or adjusting an existing one) is a frontend-only code change plus a
  1-line addition to the backend's allowed-keys list — no migration, since only the key string is
  persisted, not any hex value.
- `internal/shared/middleware/cors.go`'s dev-mode `Access-Control-Allow-Methods` list had to gain
  `PUT` (the logo-upload endpoint) — see `BACKEND_GUIDE.md`.
- **Related, not yet built:** a premium tenant accessing the app from their own business domain
  (e.g. `app.namabisnis.com`) with branding applied *before* login. Confirmed feasible on the
  existing single-VPS/single-container deployment (ADR-0011) — nginx gains a second `server_name`
  pointed at the same upstream, no cloning/second deployment — but needs a new `custom_domain`
  tenant column, a new pre-auth domain-based branding lookup endpoint (Host-header resolution, not
  JWT-based like the self-service pair above), and the frontend's `LoginPage` becoming
  domain-aware. Confirmed this doesn't interfere with the Tripay payment integration either way:
  the webhook route doesn't inspect `Host` at all (trust is HMAC-signature-based, not IP/domain
  scoped — see `MODULE_PAYMENT.md` §6), and production already serves the frontend with
  same-origin relative API paths (no `VITE_API_BASE_URL` baked into the prod build), so a second
  domain needs no CORS changes either. Out of scope for this ADR; would be its own ADR if built.
