# ADR-0015: Tenant custom domain (pre-login branding on a second hostname)

## Status
Accepted — **Deprecated 2026-11-06, removed 2026-11-07** (see "Deprecation" at the end and ADR-0030).
Tenant custom domains are no longer offered and the code described here no longer exists.

## Context
ADR-0012 shipped per-tenant branding (logo + one of 20 color presets) applied at runtime once a
staff member or client is authenticated — but explicitly left the login page neutral, since no
tenant is knowable before auth succeeds (login resolves by username/email, not by subdomain). It
flagged, as "related, not yet built": a premium tenant reaching the app from their own business
domain (e.g. `app.namabisnis.com`) with branding applied *before* login, confirmed feasible on the
existing single-VPS/single-container deployment (ADR-0011) without cloning a second deployment.

This ADR builds exactly that, closing the gap ADR-0012 left open.

## Decision

### Mechanism: Host-header resolution, not JWT
`tenants` gains a nullable, unique `custom_domain` column. Two new pre-auth, unauthenticated
endpoints — `GET /api/v1/public/branding` and `GET /api/v1/public/logo` — resolve the tenant from
the *incoming request's Host header* (`r.Host`, stripped of any `:port`) instead of a JWT claim,
the only pair of endpoints in this module that do so. This is a deliberate, narrow exception to
"every other lookup in this repo is JWT/ID-scoped" — no other module or endpoint needs it, so it
stays entirely inside `platform`'s own layers (domain → infrastructure → application →
presentation), with no `platform/contracts` addition.

A Host that matches no tenant (the platform's own `elproof.elcodelabs.com`, `localhost`, or any
domain not yet configured) 404s. The frontend treats that 404 as "no custom branding" and renders
today's neutral login page unchanged — visiting the platform's own domain is not a regression.

### Uniqueness: enforced at the database, translated at the service
`custom_domain` is `UNIQUE` (MySQL permits multiple `NULL`s in a unique index, so tenants without a
domain never collide with each other). `TenantService.Update` catches the resulting MySQL 1062
error (same pattern `mysql_webhook_log_repository.go` already uses) and turns it into a field-level
validation error (`customDomain: "Domain ini sudah dipakai tenant lain"`) rather than a raw 500 —
same translation point as every other validation rule in that method.

### Configuration: same tenant edit form, same permission model as branding
No new tenant self-service page — a platform-admin sets `customDomain` on the existing tenant edit
form (`PATCH /tenants/{id}`), right next to the brand-color picker. Unlike every other field on
this update, `customDomain` is carried as a `*string` (`UpdateTenantInput.CustomDomain`/
`updateTenantBody.CustomDomain`) so the service can tell "key omitted" apart from "key present but
empty": an omitted key leaves the tenant's existing domain untouched — the same defensive concern
`BrandColorPreset`'s own comment already flags for a stale/cached frontend bundle mid-deploy, or
any future API caller that doesn't know about this field yet — while a present, empty value
explicitly clears it back to `NULL`. A present, non-empty value is lowercased/trimmed and validated
as a bare hostname (`validator.CustomDomain` / `apps/web`'s matching Zod regex — no scheme, no
path, no port). This app's own tenant edit form always sends the key (pre-filled from the tenant's
current value via `toFormState()`), so the "omitted" case only matters for callers other than this
frontend.

### LoginPage becomes domain-aware, but only additively
On mount, `LoginPage` fetches `/public/branding` (and `/public/logo` if one exists) for the current
Host. On a match, it applies the *same* `applyBrandColorPreset`/`applyTabIdentity` helpers
ADR-0012 already built for the authenticated session, and swaps the generic lock-icon/"Selamat
Datang" header for the tenant's own logo/name. On no match (or any failure), it swallows the error
and renders exactly what it rendered before this ADR — the neutral, ElProof-agnostic look ADR-0012
deliberately chose. Nothing is a new Zustand store: this is page-local, one-shot state, unlike the
session-scoped `useTenantBrandingStore`.

### What this does NOT change
- **CORS**: none needed. Production already serves the frontend with same-origin relative API
  paths regardless of which domain it's reached through (no `VITE_API_BASE_URL` baked into the
  build) — a second domain is just another `Host` value hitting the same origin.
- **The Tripay webhook**: unaffected. Trust there is HMAC-signature-based, never IP/domain-scoped
  (`MODULE_PAYMENT.md` §6), so it doesn't matter which domain a webhook happens to arrive on.
- **Login itself**: still resolves by username/email. A custom domain is cosmetic pre-login
  branding only — it does not restrict, scope, or change which tenant's staff/clients can
  authenticate from it.

## Consequences

- **Migration** `000019_add_tenant_custom_domain`: adds `tenants.custom_domain VARCHAR(255) NULL
  UNIQUE`. No backfill — every existing tenant is unaffected until a platform-admin sets one.
- **New endpoints**: `GET /api/v1/public/branding`, `GET /api/v1/public/logo` — both unauthenticated,
  registered via a new `platform.Module.RegisterPublicRoutes(mux)` (mirroring `identity`'s existing
  unauthenticated route-registration pattern), never wrapped in `authed(...)`.
- **Still a manual, out-of-repo ops step per domain** — this ADR does not add a provisioning script.
  Neither `deploy.sh` nor `docker-compose.prod.yml` exist in this repo today (ADR-0011 — both live
  only on the VPS); nginx/certbot provisioning for `elproof.elcodelabs.com` itself was always a
  manual, one-off step, not something this repo automates. Adding one column and two endpoints
  doesn't yet justify becoming the first thing that does. Per new tenant custom domain, whoever
  operates the VPS must:
  1. Have the tenant point their domain's DNS (A or CNAME) at the VPS's public IP.
  2. Add a second `server_name <domain>;` to the *existing* nginx site file, pointed at the same
     upstream (`127.0.0.1:8082`) — no new site, no second container, no new deployment.
  3. Run `certbot --nginx -d <domain>` to extend/issue a certificate.
  4. Update `/opt/elcodelabs/SERVER_PLAYBOOK.md`'s elproof section, per that file's own "update this
     when you add something" convention (ADR-0011's precedent for the app registry itself).

  If/when tenant custom domains become common enough to justify automating steps 2–3, that's its
  own future ADR — not assumed here.

## Revision (2026-08-11): link-preview metadata, pre-login flash elimination, tenant-color scrim

Three follow-on gaps found once a real tenant (JWS Wedding, `journey.jwswedding.com`) started
using this feature, none of which change the Decision above — all additive.

### Gap 1: a shared link's preview card still said "ElProof"
`LoginPage`'s tab-title/branding swap (§ above) is client-side JS — invisible to link-preview
crawlers (WhatsApp, Telegram, Messenger, ...), which only ever read the raw HTML `index.html`
ships, never execute a page's JS. That raw HTML had "ElProof — Client Transparency Portal" and no
Open Graph tags at all, baked in statically, identical for every Host.

Fixed at the same layer that already serves that file: `spaFileServer` (`cmd/server/main.go`,
serves the built frontend — see `infra/docker/Dockerfile`) now resolves the request's Host to a
tenant the same way `PublicBranding` already does, via a new narrow method,
`platform.Module.SiteMetaForHost(ctx, host) (SiteMeta{BusinessName, HasLogo}, bool)` — same
primitive-typed-bridge shape as `projects/application`'s `StaffNameResolver`, so `main.go`
(composition-root code, not another module) never needs `platform`'s domain types. On a match,
`injectTenantSiteMeta` rewrites `index.html`'s `<title>`/`og:title`/`og:site_name`/
`og:description` (exact-string replace against the static defaults — `apps/web/index.html` carries
a comment flagging the coupling) and appends an `og:image` pointing at the tenant's logo when one
exists, HTML-escaping the business name first (this document is served to unauthenticated
crawlers, not sanitized by React). Unmatched Hosts (the platform's own domain, local dev — Vite
serves `index.html` directly there, untouched) fall through to the exact static file, byte-for-
byte. Covered by `cmd/server/sitemeta_test.go` — matched/unmatched, logo on/off, and that a
malicious business name can't break out of the document.

### Gap 2: the tenant's own branding flashed in *after* a wrong default painted first
Both the pre-auth `LoginPage` (this ADR) and the post-auth `AppLayout`/`ClientPortalLayout`
(ADR-0012) painted a default look (ElProof neutral / navy) on first render, then visibly swapped to
the tenant's real branding once the async lookup resolved — jarring on every single page load,
since neither lookup is cached. Root-caused and fixed as a render-order problem, not a data
problem — see ADR-0012's own Revision section below for the mechanism (a `hydrated`/`loading` gate
that holds off painting *either* look until the real one is known, rendering a plain white
loading screen in between). On `LoginPage` specifically, the gate only activates for a
hostname that could plausibly be a configured custom domain (checked against a small
`PLATFORM_HOSTNAMES` list) — the platform's own domain still paints instantly, since it can never
match a tenant and there's nothing worth waiting for.

### Gap 3: the login panel's text scrim never reflected the tenant's own color
`LoginPage`'s dark gradient behind the headline/subtext (over the photo slider) was a fixed
`slate-950`, regardless of a matched tenant's brand preset — so a bronze-branded tenant like JWS
Wedding still got a flat grey-black gradient instead of their own warm tone. Fixed by giving the
scrim the same `neutral`/`branded` class-pair split the panel background already used: on a
matched custom domain the scrim now uses `navy-950`/`navy-900`, which — like the panel — resolves
through the `--brand-navy-*` CSS variables `applyBrandColorPreset` overrides per tenant (ADR-0012),
so it automatically reads as the tenant's actual color with no per-preset special-casing. The
platform's own domain keeps the original neutral slate scrim, preserving this ADR's own §6
reasoning (no tenant is knowable there). A thin translucent-white divider between headline and
subtext (matching a reference design) rides on top of whichever scrim is active without needing
its own tenant-color logic — it just reads as tinted by whatever gradient sits behind it.

## Deprecation (2026-11-06): tenant custom domains discontinued

**Decision (product owner):** ElProof no longer offers per-tenant custom domains. Every tenant and
client uses `elproof.elcodelabs.com` only. Pre-login branding by Host header is therefore no longer
a supported feature.

**Why:** the operational model in "Consequences" step 3 (`certbot --nginx -d <domain>`) expanded
the platform's own certificate with the tenant domain. When `journey.jwswedding.com` later moved to
another server, its HTTP-01 challenge failed and — because certbot renews a certificate
all-or-nothing — `elproof.elcodelabs.com` itself expired on 2026-10-28 (production served
`ERR_CERT_DATE_INVALID` for 9 days). A per-tenant manual DNS/nginx/certbot step also does not scale
to self-service registration (docs/plan/registrasi-wo-self-service/PLAN.md).

**Done (Fase 0, 2026-11-06):** `journey.jwswedding.com` removed from the elproof nginx site and from
the certificate; certificate reissued for `elproof.elcodelabs.com` only (valid to 2027-02-03);
nginx reload deploy hook added; `certbot renew --dry-run` passes for every certificate on the VPS.
See `/opt/elcodelabs/SERVER_PLAYBOOK.md` "Incident — elproof TLS expired".

**Done (Fase 6, 2026-11-07):** `tenants.custom_domain` is dropped (migration 62), together with the field in
the tenant form and API, `GET /public/branding` and `/public/logo`, the Host-based site-meta injection and
`useDomainBranding()` in `LoginPage`. See ADR-0030.

**Rule kept for any future extra hostname:** one certificate per domain (`--cert-name <domain>`),
never `--expand` the platform certificate with a domain ElProof does not control.
