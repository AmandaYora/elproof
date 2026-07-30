# ADR-0015: Tenant custom domain (pre-login branding on a second hostname)

## Status
Accepted

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
