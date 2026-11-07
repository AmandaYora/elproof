# Module Map

> Each module's responsibility and the public contract it exposes.

Backend modules live at `apps/api/internal/modules/<name>`. See ADR-0008 for how these map to the
frontend's own module folders under `apps/web/src/modules/<name>`.

| Module | Responsibility | Owns (tables) | Consumes (via other modules' contracts) |
|---|---|---|---|
| `identity` | Login/refresh/logout for all principal types; password hashing; token issuance; login rate limits; mandatory TOTP two-factor for platform admins (ADR-0023); Sign in with Google for WO Owners/staff with Google ID-token verification (ADR-0024); one-time invitation / password-reset links (`auth_tokens`, ADR-0025); `TenantGuard` port so a frozen tenant cannot log in (ADR-0022) | `credentials`, `refresh_tokens`, `auth_tokens` | none — deliberately profile-agnostic (see ADR-0005); the `TenantGuard` and `TenantNameResolver` ports are implemented by `platform` and wired in `cmd/server` |
| `audit` | Append-only audit trail of state-changing Platform Console requests (ADR-0023) and of workspace role changes, stored with `tenant_id` (ADR-0026); implements `middleware.AuditRecorder`; `GET /audit-log` (Super Admin) | `audit_log` | none |
| `access` | Per-workspace roles and permissions (ADR-0026): fixed permission catalog in code, role templates seeded per tenant, `/access/me`, `/access/catalog`, `/access/roles`; implements `middleware.GrantsResolver` for `LoadGrants` (30 s cache) | `tenant_roles`, `tenant_role_permissions` | `identity` (`CountLoginsWithRole`, through the `MemberCounter` port wired in `cmd/server`) |
| `payment` | One payment-gateway merchant wallet (Tripay), wrapped for many consumers ("Apps") — gateway config, App registry, webhook dispatch. Owns no business ledger of its own — see `MODULE_PAYMENT.md`. Fase 9: internal mode (`platform` as the one App internal). Fase 10 (implemented): external Apps over HTTP — `/auth/app/token` + `/external/payments/*`, Platform Console's "Manajemen Aplikasi" page. | `payment_gateway_config`, `payment_apps`, `payment_charge_dispatch`, `payment_webhook_events` | `identity` (mint bearer tokens for external Apps — one-way, same shape as `vendors -> projects` below) |
| `platform` | Tenant lifecycle (register/suspend/activate/pay; computed subscription lifecycle + per-request access level and hourly status sweeper — ADR-0022; accountable manual activation — ADR-0023; self-service signup saga, Owner onboarding and free-trial caps — ADR-0024), per-tenant branding (logo + 1-of-20 color presets), Platform Console's own admin accounts — see `MODULE_PLATFORM.md` | `tenants`, `platform_admins`, `pending_subscription_charges`, `manual_activations`, `signup_requests`, `trial_limits`, `tenant_business_details` | `staff` (create Owner on tenant registration), `identity` (create credentials), `billing` (read plan, record/update transaction), `payment` (create charge; also registers itself as `payment`'s webhook consumer for `platform-billing`, Fase 9), `vendors` (seed a new tenant's default vendor categories on registration), `projects` (seed a new tenant's default Timeline template on registration — `SeedDefaultMilestoneTemplate`, same moment as the `vendors` seed above); depends directly on `internal/shared/storage` (not another module's contract — a shared technical utility, same one `projects`/evidence uses, ADR-0006) for logo upload/download |
| `billing` | Subscription plan catalog + subscription transaction ledger — single source of truth shared by both consoles. Also owns per-App plan visibility (`subscription_plan_apps`, D3) and an external-App subscription ledger (`subscription_app_charges`, D7) — `GET /external/subscription-plans`, `POST /external/subscriptions/charges`, `GET`/`PUT /app-plans`, `GET /app-subscriptions` (`docs/plan/external-subscription-plans/PLAN.md`) | `subscription_plans`, `subscription_transactions`, `subscription_plan_apps`, `subscription_app_charges` | `payment` (`AppRegistry` — checks an App's active/exists status for `billing`'s own App-token endpoints; `Client` — creates a subscription charge, `CreateChannelCharge`) |
| `staff` | WO internal users, tenant-scoped; each has a workspace role (`role_id`, name kept in `role`, ADR-0026) | `staff_members` | `identity` (create credentials for Admin/Staff created via "Tambah Pengguna"; the Owner's own credential is created separately by `platform`'s tenant-registration orchestration, see below) |
| `clients` | Client contacts per project, tenant-scoped; each has a portal role (`access_role_id`, ADR-0026) | `clients` | `projects` (validate `project_id` exists), `identity` (create/reset credentials) |
| `vendors` | Vendor directory + categories, tenant-scoped, plus (ADR-0016) the Venue directory — its own table, but still this module, since Venue and Vendor are conceptually siblings | `vendors`, `vendor_categories`, `venues` | `projects` (resolve a vendor's cross-project engagement history for "Lihat Project" — `project_vendors` is owned by `projects`, not `vendors`) |
| `projects` | The core WO object: projects, milestones ("Timeline" in the UI, display-text-only rename), vendor engagements (each with a `pricingTier` label — "Akad"/"AkadResepsi" — for auto-filling their contract value from the vendor's own preset prices), vendor milestones ("Timeline Vendor"), vendor payments (money **out** to vendors), client payments (money **in** from the client against the project's own `contractValue`, no `project_vendor_id` — "Uang Masuk dari Client"), venue payments (money **out** to the project's venue, same two-slot evidence rule as vendor payments, no `project_vendor_id`-equivalent since a project has at most one venue), issues, evidence (incl. a `general`-kind, per-document client-visibility toggle for standalone documents with no other specific context — powers Client Portal's own "Dokumen" tab, `GET .../documents` — see ADR-0020), activity log. All three payment ledgers now also support full-field Edit (no role gate) and hard delete (**Owner-or-Admin**, cascading to the payment's own evidence — ADR-0019, the first hard-delete in this codebase that isn't Owner-only). Also owns project archive (reversible) and hard delete (guarded, ADR-0013), project duplicate (clones milestones + vendor lineup as a structural template, excludes payments/issues/evidence/activity/clients — ADR-0014), a per-tenant configurable Timeline Default Template (seeds new projects' own milestones instead of one hardcoded list shared by every tenant; each item's day-offset can now be negative — "Setelah Acara"/H+, an item scheduled after the event date), and permission checks (ADR-0026, superseding ADR-0017): a role with `assigned` scope sees only projects where the person is PIC or PIC Sales; writes are gated centrally by `staffWritePermission`, client portal reads by `portal.*` permissions | `projects`, `project_milestones`, `project_milestone_templates`, `project_vendors`, `vendor_milestones`, `vendor_payments`, `client_payments`, `client_payment_terms`, `venue_payments`, `vendor_issues`, `evidence`, `activity_log` | `clients` (Fase 6: resolve which single project a `client` principal may read, for `/projects/{id}/...`'s client-portal read scoping; ADR-0013: best-effort delete every client tied to a project being hard-deleted), `vendors` (ADR-0016: resolve a project's attached `venue_id` into display data — `projects → vendors`, the reciprocal direction of the row above). Vendor/category/staff *display names* are **not** resolved backend-side — the frontend resolves those from its own already-fetched `vendors`/`staff` stores. |

`dashboard` and `client-portal` (frontend-only concepts) are **not** separate backend modules —
they are read/composition endpoints living alongside the module(s) whose data they aggregate (WO
dashboard next to `projects`; Platform dashboard next to `billing`/`platform`), or a thin
cross-module app-service when they span more than one module's data.

**Breaking a circular module dependency (Fase 6):** `clients` already needed `projects.Contracts()`
(to validate `project_id` on create), and Fase 6 then needed `projects` to call back into `clients`
(to scope a client principal's reads to their own project) — a genuine two-way dependency that no
constructor argument order can satisfy (whichever module is built first can't yet hold a reference
to the other). Resolved with dependency inversion, not a cross-module import: `projects/presentation`
declares its own tiny local interface (`ClientAccessResolver`) shaped to match one method of
`clients.Contracts` — `projects` never imports `clients` at all. `main.go` (the composition root,
which already imports both) builds both modules, then bridges them with a setter
(`projectsModule.SetClientAccessResolver(clientsModule.Contracts())`). Reach for this pattern again
if another pair of modules ever needs a real two-way relationship.

**Same bridge, a second interface (ADR-0013):** hard-deleting a project needs to best-effort clean
up every client tied to it — another `projects`-needs-something-from-`clients` case, same
underlying cycle. Rather than overloading `ClientAccessResolver` (a read-access-control concern) with
an unrelated delete-cleanup method, `projects/application` declares its own second local interface,
`ClientCleaner`, bridged the same way (`projectsModule.SetClientCleaner(clientsModule.Contracts())`,
right after the existing `SetClientAccessResolver` call) — `clientsModule.Contracts()` is the same
concrete object satisfying both interfaces structurally, so no extra wiring object is needed.

**Same pattern reused for `payment`'s webhook dispatch (Fase 9):** `payment` needs to call back into
whichever App internal owns a confirmed charge (`platform`, so far) without ever importing that
App's package (`payment` must stay ignorant of every consumer's business domain — see
`MODULE_PAYMENT.md`). `payment/contracts` declares a `WebhookConsumer` interface shaped to one
method (`ApplyWebhookEvent`); `platform.Module` itself implements it by delegating to its own
`application.TenantService`. `main.go` builds `payment` first (it depends on nothing), builds
`platform` next (receiving `paymentModule.Client()` as a constructor argument), then bridges the
other direction with `paymentModule.Dispatcher().RegisterConsumer(paymentcontracts.InternalAppBilling,
platformModule)` — no cycle, no cross-module import either way.

**`vendors` → `projects`, a plain one-way dependency (not circular, no bridge needed):**
`vendors.NewModule` takes `projects.Contracts` as a constructor argument (to resolve "Lihat
Project" — see `ListVendorEngagementHistory` above). Since `projects` never needs anything back
from `vendors`, this is just an ordinary constructor dependency — `main.go` simply builds
`projectsModule` before `vendorsModule`. No dependency-inversion/bridge pattern required; that
machinery is only for genuine two-way relationships (see the `clients`↔`projects` and
`payment`↔`platform` cases above).

**The reciprocal direction, `projects` → `vendors` (ADR-0016), needed once Venue shipped:**
showing a project's attached venue (WO Console's own Venue tab, and the public-safe summary Client
Portal's Venue tab reads) means `projects` needs to resolve `venue_id` into venue details — the
opposite direction from the row above, and `vendorsModule` is built *after* `projectsModule`
(dependent on it), so `vendors.Contracts` can't be a `projects.NewModule` constructor argument
without inverting the whole build order. Same setter idiom as every other two-phase case in this
file: `projects/application` declares its own narrow consumer interface (`VenueResolver`, mirroring
`ClientCleaner`'s existing shape) returning a type from `vendors/contracts`; `main.go` builds both
modules, then bridges with `projectsModule.SetVenueResolver(vendorsModule.Contracts())`.

**`platform` → `vendors`, a one-way dependency blocked only by build order (not a real cycle):**
`platform` needs `vendors.Contracts` to seed a new tenant's default vendor categories on
registration, but `main.go` builds `platformModule` *before* `vendorsModule` (which itself must be
built after `projectsModule`, since `vendors.NewModule` takes `projects.Contracts`) — so
`vendors.Contracts` can't be a `platform.NewModule` constructor argument without reordering the
whole composition root. Since `vendors` never calls back into `platform`, this isn't a genuine
cycle and needs no dependency-inversion/interface-splitting — just the same setter idiom as the
two-phase cases above: `TenantService.SetVendors` / `platform.Module.SetVendors`, called from
`main.go` right after `vendorsModule` is built (`platformModule.SetVendors(vendorsModule.Contracts())`).

**`platform` → `projects`, the same build-order-blocked shape, for the Timeline Default Template
seed:** `platform` needs `projects.Contracts` (`SeedDefaultMilestoneTemplate`) at tenant
registration, but `platformModule` is built before `projectsModule` too. Same setter idiom again:
`TenantService.SetProjects` / `platform.Module.SetProjects`, called from `main.go` right after
`projectsModule` is built (`platformModule.SetProjects(projectsModule.Contracts())`) — this one
runs earlier than `SetVendors` above, since `projectsModule` itself is built before `vendorsModule`.

**`payment` → `identity`, the same plain one-way shape (Fase 10):** `payment.NewModule` takes
`identity.Contracts` as a constructor argument, to mint bearer tokens for external Apps
(`IssueServiceToken`, see `MODULE_PAYMENT.md` §7.1) — `identity` never needs anything back from
`payment`, so `main.go` simply builds `identityModule` before `paymentModule` (already the existing
order). `payment` still depends on nothing else, and no other App internal's webhook-dispatch
bridging (see above) changes because of this.

**`staff` → `identity`, same plain one-way shape:** originally `staff.NewModule` took only `db` —
regular Admin/Staff accounts created via "Tambah Pengguna" had no login credential at all, only the
tenant Owner did (provisioned separately by `platform`'s tenant-registration orchestration). Fixed
by giving `staff.NewModule` an `identity.Contracts` constructor argument too, so `Create` can
provision a real credential for Admin/Staff the same way `clients.Create` already does — including
the same compensating rollback (delete the just-inserted `staff_members` row) if
`identity.CreateCredential` fails after it. `identity` never needs anything back from `staff`, so
`main.go` just builds `identityModule` before `staffModule` (already the existing order).

**`billing` → `payment`, a plain one-way dependency (`docs/plan/external-subscription-plans/PLAN.md`,
A5):** `billing.NewModule` now takes `paymentcontracts.AppRegistry` and `paymentcontracts.Client` as
constructor arguments — the same direct-contracts-injection shape `platform` already uses for
`paymentModule.Client()` (`main.go:210`), not a duplicated local interface. `payment` needs nothing
back from `billing` (verified zero imports either direction before this change, T6 in the plan
above), so this is an ordinary one-way dependency, not a cycle: `main.go` simply builds
`paymentModule` before `billingModule` — the existing order already had `payment` first (A6), so no
reordering was needed either.

## Frontend module ↔ backend module

| Frontend (`apps/web/src/modules/*`) | Backend module(s) it talks to |
|---|---|
| `auth` | `identity` |
| `dashboard` | `projects` (aggregation endpoint) |
| `projects` | `projects` |
| `clients` | `clients` |
| `vendors` | `vendors` |
| `vendor-categories` | `vendors` |
| `venues` | `vendors` (ADR-0016) |
| `milestone-templates` | `projects` — "Pengaturan → Timeline Default," Owner-only |
| `users` | `staff` |
| `subscription` | `billing` |
| `client-portal` | `projects`, `clients` (read-scoped) |
| `platform-admin` | `platform`, `billing`, `payment` (Konfigurasi Gateway + Manajemen Aplikasi pages) |
| `homepage` | none — public marketing pages (`/homepage`, `/homepage/tentang-kami`, `/homepage/syarat-ketentuan`, `/homepage/kebijakan-privasi`, `/homepage/kebijakan-refund`, `/homepage/faq`, `/homepage/kontak`), static content only, no API calls |

Full column-level schema: [`docs/DB_SCHEMA.md`](../docs/DB_SCHEMA.md). Full endpoint list:
[`docs/API_CONTRACT.md`](../docs/API_CONTRACT.md).

**`staff`/`clients`/`platform` → `access` (ADR-0026):** `staff` and `clients` resolve and validate
role ids through small `RoleResolver` ports; `platform` seeds a new tenant's roles
(`SeedDefaultRoles`) and asks for the Owner role at signup/registration (`TenantService.SetAccess`).
All three are wired in `cmd/server/main.go` with adapters over `accessModule.Contracts()`; `access`
itself only reaches `identity` through its `MemberCounter` port, so there is no cycle. These are the
`access` roles; permission checks themselves read `middleware.GrantsFrom(ctx)`, not the module.
