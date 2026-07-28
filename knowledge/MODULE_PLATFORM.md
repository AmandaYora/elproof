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
- **Tenant branding**: a logo + one of 20 fixed color presets, applied across WO Console and Client
  Portal — see §6 and ADR-0012.
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
  domain/          — Tenant, PlatformAdmin, PendingCharge, SubscriptionStatus, brand_preset.go
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
  timestamp, and (ADR-0012) `brand_color_preset`/`logo_storage_path`. Full columns:
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

## 6. Branding (logo + color presets) — ADR-0012

Every tenant gets a visual identity: an optional logo (object storage, ADR-0006) and one of 20
fixed color presets (`domain/brand_preset.go`'s `AllowedBrandColorPresets`, default `navy`) —
applied across WO Console and Client Portal by overriding 4 CSS variables at runtime
(`apps/web/src/theme/brandPresets.ts`; see `FRONTEND_GUIDE.md`'s "Theme / tenant branding" section
for the frontend half). No free-form hex — see ADR-0012 for the full rationale, including why
every preset's shades are chosen for WCAG contrast against white text, not just picked to look
vivid.

- **Configured by**: platform-admin only, via the same tenant edit form as every other tenant
  field (`PATCH /tenants/{id}` now also accepts `brandColorPreset`; `PUT /tenants/{id}/logo` for
  the logo). No tenant self-service branding settings page exists yet.
- **Read by**: `GET /tenants/me/branding` / `GET /tenants/me/logo` — deliberately **open to any
  tenant-scoped principal** (any staff role, or `client`), unlike `GET /tenants/me` which stays
  Owner-only (it also carries subscription/billing data). Both resolve the tenant strictly from the
  JWT's `tenant_id` claim.
- **Logo storage**: same `internal/shared/storage` utility as evidence (ADR-0006), a different key
  shape (`BuildKey(tenantId, "0", "branding", filename)` — no `projectId`), 2 MB decoded cap (vs.
  evidence's 15 MB), PNG/JPEG/WebP only (no SVG). Streamed through an authenticated Go handler,
  never a public URL — same byte-proxy shape as evidence download.
- **Where it never applies**: Platform Console (superadmin's own backoffice — not "inside" any one
  tenant) and the login page (no tenant is known before auth succeeds) — see ADR-0012 for why the
  login page was neutralized (not tenant-branded, but not ElProof-branded either) rather than left
  as the original default.

## 7. Platform Console's own admin accounts (`platform_admins`)

A separate, simpler CRUD (`/platform-admins`) with no cross-module orchestration — a
platform_admin's only side effect on creation is `identity.CreateCredential`. Self-lockout (an
admin deactivating their own account via `toggle-active`) is rejected with 403.
