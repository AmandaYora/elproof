# MODULE_PLATFORM.md — the `platform` module: tenant lifecycle, branding, and ElProof's own admins

## 1. Why this module exists

`platform` is the one module that spans both sides of the business: it owns the **tenant** entity
itself (a WO business subscribed to ElProof) and ElProof's **own** internal admin accounts
(`platform_admins`, who manage every tenant from the Platform Console). It orchestrates — never
owns — the pieces of tenant registration that belong to other modules (staff, credentials,
subscription billing, default vendor categories) via their `contracts` packages, per ADR-0008.

## 2. Scope

- **Tenant lifecycle**: register (creates the tenant row + its Owner staff row + Owner credential,
  in one flow), suspend/reactivate, reset the Owner's credential, and the subscription-activation
  paths (superadmin manual grant, or the Owner's own self-service Tripay payment — see §5).
- **Tenant branding**: the business name and an optional logo, shown in the WO Console and Client
  Portal — see §6, ADR-0012 and ADR-0039 (no per-tenant colour).
- **Platform Console's own admin accounts** (`platform_admins`) — a separate, simpler CRUD with no
  cross-module orchestration (a platform_admin has no Owner-equivalent staff row, no subscription).

### What this module is explicitly NOT
- **Not the subscription plan catalog or transaction ledger** — those belong to `billing`
  (`subscription_plans`, `subscription_transactions`). `platform` only calls `billing.Contracts` to
  read a plan or record/update a transaction.
- **Not the payment gateway** — `platform` is `payment`'s one internal consumer (`platform-billing`
  App, Fase 9), calling `payment.Client.CreateCharge`/`CheckStatusForApp` and implementing
  `payment.contracts.WebhookConsumer` to get notified when a charge resolves. See
  `MODULE_PAYMENT.md`.
- **Not aware of any other module's tables.** Cross-module relations (`plan_id`, a project's
  `tenant_id`, etc.) are primitive IDs resolved through contracts, never joined.

## 3. Structure and dependencies

```
apps/api/internal/modules/platform/
  domain/          — Tenant, PlatformAdmin, PendingCharge, SubscriptionStatus
  application/      — TenantService, PlatformAdminService
  infrastructure/    — MySQL repositories
  presentation/      — TenantHandler, PlatformAdminHandler
  platform.module.go
```

**Dependencies** (constructor arguments to `platform.NewModule`): `staff.Contracts` (create the
Owner on registration), `identity.Contracts` (create/reset the Owner's credential),
`billing.Contracts` (read plan, record/update transaction), `payment.Client` (create charge),
`*storage.Client` (tenant logo upload/download — a direct dependency on the shared technical
utility `internal/shared/storage`, same one `projects`/evidence uses, not another module's
contract — see ADR-0006's revision). `vendors.Contracts` is wired in two-phase (`SetVendors`) since
`vendors` itself depends on `projects`, built after `platform` — see `MODULE_MAP.md`.

`TenantService` declares its own local `ObjectStorage` interface (`Save`/`Open`) rather than
importing `projects/application`'s identical-shaped one — a module can never import another
module's `application` package, even for an interface type, so this small duplication across the
two modules is intentional (see `projects/application/evidence_service.go`'s `ObjectStorage` for
the sibling definition).

## 4. Data model

- **`tenants`** — the core entity. Business/owner contact fields, subscription state
  (`plan_id`/`subscription_status`/`subscription_expires_at`), suspension flag, credential-reset
  timestamp, and (ADR-0012) `logo_storage_path` (`brand_color_preset` was dropped by migration 000072, ADR-0039). Full columns:
  `docs/DB_SCHEMA.md`.
- **`platform_admins`** — ElProof's own staff. Role is `'Super Admin'` or `'Support'` — no tenant
  scoping (these accounts aren't tied to any `tenant_id`).
- **`pending_subscription_charges`** — tracks a tenant's still-open self-service Tripay charge
  (Fase 9's `/subscriptions/pay`), so a second charge can't be started while one is already pending,
  and so the Owner can re-view a still-pending charge's QR/pay-code after closing its modal. Cleared
  once `payment`'s webhook (or its reconciliation sweep) resolves the charge — see
  `MODULE_PAYMENT.md` §6.

## 5. Tenant lifecycle

- **Register** (`POST /tenants`, platform_admin only) orchestrates three modules in one flow:
  creates the tenant row (this module), the Owner's `staff_members` row (`staff.CreateOwner`), and
  the Owner's login credential (`identity.CreateCredential`) — see ADR-0008. No plan is bound at
  registration (`plan_id: null`, `subscription_status: pending_payment`) and no transaction is
  recorded — the first transaction row a tenant ever gets is whichever real subscription event
  happens first: the Owner's own `Pay`, or a superadmin's `ActivateSubscription`.
- **Suspend/reactivate** (`POST /tenants/{id}/toggle-suspension`) is a single toggle, not two
  separate actions.
- **Reset credential** (`POST /tenants/{id}/reset-credential`) resets the Owner's login via
  `identity.ResetPasswordByUsername` — looked up by username (the one value `tenants` and
  `credentials` share as plain, unenforced values, not a foreign key).
- **Subscription activation** has two independent paths that converge on the same
  `UpdateSubscription` call, so a granted and a paid subscription are indistinguishable afterward
  except in the transaction ledger:
  - **Manual grant** (`POST /tenants/{id}/activate-subscription`, platform_admin) — bypasses
    payment entirely, records a `granted` transaction. Also resolves any pending self-service
    charge for this tenant first, so a late webhook/reconciliation result for that old charge can't
    double-extend the expiry this call just set.
  - **Self-service pay** (`POST /subscriptions/pay`, the Owner) — creates a real Tripay charge,
    records a `pending` transaction, and only actually activates once `payment`'s webhook (or its
    reconciliation sweep) confirms it, via `ApplyWebhookEvent` (this module's implementation of
    `payment.contracts.WebhookConsumer`).

## 6. Branding (name + logo) — ADR-0012, ADR-0039

Every tenant shows its own business name and an optional logo (object storage, ADR-0006) in the WO
Console and Client Portal header and the browser tab (see `FRONTEND_GUIDE.md`'s "Theme / tenant
branding" section for the frontend half). There is **no per-tenant colour**: the 21 colour presets,
`PATCH /tenants/me/brand-color` and the `brand_color_preset` column were removed (migration 000072,
ADR-0039); every surface uses the one palette in `apps/web/src/theme/theme.css`.

- **Configured by**: platform-admin via the tenant edit form (`PUT /tenants/{id}/logo`), and the
  Owner on Profil Usaha (`PUT /tenants/me/logo`, name through `PATCH /tenants/me/profile`).
- **Read by**: `GET /tenants/me/branding` / `GET /tenants/me/logo` — deliberately **open to any
  tenant-scoped principal** (any staff role, or `client`), unlike `GET /tenants/me` which stays
  Owner-only (it also carries subscription/billing data). Both resolve the tenant strictly from the
  JWT's `tenant_id` claim.
- **Logo storage**: same `internal/shared/storage` utility as evidence (ADR-0006), a different key
  shape (`BuildKey(tenantId, "0", "branding", filename)` — no `projectId`), 2 MB decoded cap (vs.
  evidence's 15 MB), PNG/JPEG/WebP only (no SVG). Streamed through an authenticated Go handler,
  never a public URL — same byte-proxy shape as evidence download.
- **Where it never applies**: Platform Console (superadmin's own backoffice — not "inside" any one
  tenant). The login page was neutralized by ADR-0012 for the same reason (no tenant knowable
  before auth).

## 7. Platform Console's own admin accounts (`platform_admins`)

A separate, simpler CRUD (`/platform-admins`) with no cross-module orchestration — a
platform_admin's only side effect on creation is `identity.CreateCredential`. Self-lockout (an
admin deactivating their own account via `toggle-active`) is rejected with 403.

## 8. Custom domain — removed (ADR-0015, ADR-0030)

Tenants no longer have their own hostname. `tenants.custom_domain`, the Host-based
`GET /public/branding` / `/public/logo`, `Module.SiteMetaForHost` and the `index.html` meta injection in
`cmd/server` were deleted in Fase 6 (migration 62). Everyone uses the platform's own domain, so the login
page is always neutral, and any extra hostname would need its own certificate (DEPLOYMENT.md).

## 9. Login page marketing photos (global, not tenant-scoped)

A small set of fixed marketing photos (currently 3) plays as an auto-advancing slider on
`LoginPage`'s left panel, for every visitor regardless of tenant or domain — unlike everything
else in this document, these are **not** tenant data and have no `tenant_id`.

- **Upload**: `api upload-login-slides <source-dir>` (`internal/loginslides`, invoked from
  `cmd/server`'s subcommand dispatch, not a request handler) reads a small fixed table of
  `{sourceFile, key}` pairs, compresses each through the same pipeline evidence/logo uploads use
  (`internal/shared/compress`, ADR-0010), and saves them to object storage under
  `elproof/marketing/login-slides/slide-0N.jpg`. Re-running is safe — same keys get overwritten.
- **Served by**: `LoginSlidesHandler` (`platform/presentation`), registered on
  `RegisterPublicRoutes` alongside `PublicBranding`/`PublicLogo` — `GET /public/login-slides`
  (count) and `GET /public/login-slides/{index}` (raw JPEG, byte-proxied, `Cache-Control: public,
  max-age=86400` since these rarely change). The served key list
  (`login_slides_handler.go`'s `loginSlideKeys`) and the upload table
  (`loginslides.go`'s `slides`) must stay in sync by hand — no shared source, since one is a
  request-time read list and the other a one-off operator command.
- **Why `platform`, not `shared/`**: these are global marketing *content*, not a technical utility
  — `shared/` is documented as technical-utilities-only (errors, response, logger, validator,
  pagination). `platform` already owns every other pre-auth endpoint `LoginPage` depends on.

## 10. Platform Console back office (Fase 6, ADR-0030)

Files: `domain/console.go`, `domain/platform_metrics.go`, `application/console_service.go`,
`application/metrics_service.go`, `infrastructure/mysql_console_repository.go`,
`presentation/console_handler.go`, `metrics_handler.go`, `tenant_export.go`. Tables: `tenant_notes`,
`data_requests`, `announcements`, `subscription_policy`, `support_access_grants`, `support_sessions`
(migration 63). Like the rest of the module they hold tenant ids as plain numbers.

- **Metrics**: `GET /platform/metrics` (any platform role). `ComputePlatformMetrics` is pure: MRR = paid
  tenants' plan price / plan months, trial conversion = trials that ended in the last 30 days and became paid,
  churn = paid tenants that entered read-only in the window / paid tenants at its start. Free grants are
  excluded via `FreeGrantTenantIDs`. Revenue comes from `billing.PaidSince`.
- **Tenant 360°**: `GET /console/tenants/{id}/overview`, `notes`, `data-requests`, `support`; the member list
  comes from `staff.ListMembers`. Tenant list filters (`search`, `status`, `planId`, `source`) are applied in
  SQL by `tenantWhere`; `GET /tenants/export` reuses them.
- **Support sessions**: see ADR-0030. `ConsoleService.StartSupportSession` checks the active grant,
  validates the reason, finds the Owner through `staff.FindOwner`, records the session, then asks
  `identity.IssueSupportToken` for the token; if that fails the session row is deleted again. The Owner
  manages consent at `/tenants/me/support-access` (GET/PUT/DELETE, `business_profile.manage`).
- **Live check**: `ConsoleService.CheckSupportSession` backs `middleware.RequireLiveSupportSession` (session open, consent valid, admin active as Support/Super Admin) on every support-token request.
- **Policy**: `subscription_policy` has one row; `LoadSubscriptionPolicy` reads it at startup and
  `SavePolicy` writes, applies it to `domain.SetSubscriptionPolicy` and re-runs the status sweep so the cached `subscription_status` filter agrees. Callers read it only through
  `domain.CurrentSubscriptionPolicy()`.
- **Audit**: platform routes are wrapped in `AuditTrail`; exports write their own record through
  `SetAuditRecorder`.
- **Login photos**: `PUT /console/login-slides/{index}` (Super Admin) replaces one of the three fixed slots;
  the image is decoded, flattened to JPEG and compressed.
- **Trends and names (Fase 7)**: `GET /platform/trends` (`domain.ComputeTrends`, pure and tested, buckets in
  WIB) and `GET /tenants/names?ids=`; `GET /platform-admins/{id}`. `all=true` no longer exists on
  `/tenants` and `/platform-admins`.
