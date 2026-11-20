# Module Map

> Each module's responsibility and the public contract it exposes.

Backend modules live at `apps/api/internal/modules/<name>`. See ADR-0008 for how these map to the
frontend's own module folders under `apps/web/src/modules/<name>`.

| Module | Responsibility | Owns (tables) | Consumes (via other modules' contracts) |
|---|---|---|---|
| `identity` | Login/refresh/logout for all principal types; password hashing; token issuance; login rate limits; mandatory TOTP two-factor for platform admins (ADR-0023); Sign in with Google for WO Owners/staff with Google ID-token verification (ADR-0024); one-time invitation / password-reset links (`auth_tokens`, ADR-0025); `TenantGuard` port so a frozen tenant cannot log in (ADR-0022); `contracts.LoginStatuses` (ADR-0035) gives the sign-in state of many principals of one workspace in a few queries, for the paginated client and staff lists | `credentials`, `refresh_tokens`, `auth_tokens` | none — deliberately profile-agnostic (see ADR-0005); the `TenantGuard` and `TenantNameResolver` ports are implemented by `platform` and wired in `cmd/server` |
| `audit` | Append-only audit trail of state-changing Platform Console requests (ADR-0023) and of workspace role changes, stored with `tenant_id` (ADR-0026); implements `middleware.AuditRecorder`; `GET /audit-log` (Super Admin) | `audit_log` | none |
| `access` | Per-workspace roles and permissions (ADR-0026): fixed permission catalog in code, role templates seeded per tenant (client permissions `portal.finance.submit` and `portal.tasks.complete` added by ADR-0047; staff permission `package.manage`, group Katalog, added by paket WO Tahap 1 for Owner and Admin; `quotation.view` and `quotation.manage`, added by paket WO Tahap 5 for Owner, Admin and Sales), `/access/me`, `/access/catalog`, `/access/roles`; implements `middleware.GrantsResolver` for `LoadGrants` (30 s cache) | `tenant_roles`, `tenant_role_permissions` | `identity` (`CountLoginsWithRole`, through the `MemberCounter` port wired in `cmd/server`) |
| `payment` | One payment-gateway merchant wallet (Tripay), wrapped for many consumers ("Apps") — gateway config, App registry, webhook dispatch. Owns no business ledger of its own — see `MODULE_PAYMENT.md`. Fase 9: internal mode (`platform` as the one App internal). Fase 10 (implemented): external Apps over HTTP — `/auth/app/token` + `/external/payments/*`, Platform Console's "Manajemen Aplikasi" page. | `payment_gateway_config`, `payment_apps`, `payment_charge_dispatch`, `payment_webhook_events` | `identity` (mint bearer tokens for external Apps — one-way, same shape as `vendors -> projects` below) |
| `platform` | Tenant lifecycle (register/suspend/activate/pay; computed subscription lifecycle + per-request access level and hourly status sweeper — ADR-0022; accountable manual activation — ADR-0023; self-service signup saga, Owner onboarding and free-trial caps — ADR-0024), per-tenant branding (name + logo; no per-tenant colour, ADR-0039), Platform Console's own admin accounts, Tahap 6 (ADR-0035): sortable tenant list (plan names ranked through `billing.ListPlans`) and the `TenantNames`/`SearchTenantIDs` methods that satisfy `billing`'s `TenantDirectory` port; Fase 6 back office (SaaS metrics, tenant notes, personal-data requests, announcements, runtime subscription policy, consented read-only support sessions — ADR-0030) — see `MODULE_PLATFORM.md`; paket WO Tahap 1 (ADR-0048): the Owner's quote settings in `tenant_business_details` (`document_code`, `is_pkp`, default validity days, default terms — `GET/PUT /tenants/me/quote-settings`, Profil usaha › Penawaran), created from the business name at sign-up or on first read | `tenants`, `platform_admins`, `pending_subscription_charges`, `manual_activations`, `signup_requests`, `trial_limits`, `tenant_business_details`, `tenant_notes`, `data_requests`, `announcements`, `subscription_policy`, `support_access_grants`, `support_sessions` | `staff` (create Owner on tenant registration; `ListMembers` and `FindOwner` for Tenant 360° and support sessions), `identity` (create credentials; `IssueSupportToken` for support sessions), `billing` (read plan, record/update transaction; `PaidSince` for metrics), `payment` (create charge; also registers itself as `payment`'s webhook consumer for `platform-billing`, Fase 9), `vendors` (seed a new tenant's default vendor categories on registration), `projects` (seed a new tenant's default Timeline template on registration — `SeedDefaultMilestoneTemplate`, same moment as the `vendors` seed above), `offers` (seed a new tenant's six default event kinds on registration, through the local `EventKindSeeder` port set with `SetEventKinds` in `cmd/server` — platform never imports offers); depends directly on `internal/shared/storage` (not another module's contract — a shared technical utility, same one `projects`/evidence uses, ADR-0006) for logo upload/download |
| `billing` | Subscription plan catalog + subscription transaction ledger — single source of truth shared by both consoles. Every plan belongs to exactly one App (`subscription_plans.app_id`, ADR-0040 — ElProof's own WO catalog is the plans of the internal App `platform-billing`; `contracts.GetTenantPlan` refuses any other App's plan for a WO tenant), plus an external-App subscription ledger (`subscription_app_charges`, D7) — `GET /external/subscription-plans`, `POST /external/subscriptions/charges`, `GET /app-subscriptions` (`docs/plan/external-subscription-plans/PLAN.md`). Pre-auth `GET /public/plans` (`RegisterPublicRoutes`, T-20261110-9): the active plans of ElProof's own catalog for the public homepage — `id, name, durationMonths, price, features` only, ordered by duration then price, `Cache-Control: public, no-cache` with an ETag hashed from the body, so a plan change shows at the next request (EP-11; was `max-age=300`). Tahap 6 (ADR-0035): the transaction list carries `tenantName`, searches by tenant name and sorts by it, through a local `TenantDirectory` port that `cmd/server` adapts from `platform` (`billingModule.SetTenantDirectory(platformModule)`); `contracts.ListPlans` lets `platform` order its tenant list by plan name. **Vouchers (ADR-0041):** discount codes per App (optionally per plan) managed at `/vouchers`; a WO Owner previews at `POST /subscriptions/voucher-check` and `platform` reserves through `contracts.ReserveTenantVoucher` before charging (released by `ReleaseVoucher`, settled by `UpdateTransactionStatus`); an App uses `voucherCode` on `/external/subscriptions/charges` and `POST /external/subscriptions/voucher-check`; the subscription reconciler settles App uses and releases stale reservations | `subscription_plans`, `subscription_transactions`, `subscription_app_charges`, `subscription_vouchers`, `subscription_voucher_redemptions` | `payment` (`AppRegistry` — checks an App's active/exists status for `billing`'s own App-token endpoints and a plan's owning App; `Client` — creates a subscription charge, `CreateChannelCharge`); `platform` (names and name search of tenants, **one-way through the local `TenantDirectory` port**, wired in `cmd/server` — billing never imports platform) |
| `staff` | WO internal users, tenant-scoped; each has a workspace role (`role_id`, name kept in `role`, ADR-0026) | `staff_members` | `identity` (create credentials for Admin/Staff created via "Tambah Pengguna"; the Owner's own credential is created separately by `platform`'s tenant-registration orchestration, see below) |
| `clients` | Client contacts per project, tenant-scoped; each has a portal role (`access_role_id`, ADR-0026). Tahap 5 (ADR-0034): delete is a **soft delete** (`deleted_at`, restorable for 24 h; the login is deactivated, not removed) with `POST /clients/{id}/restore`, and a purge (`PurgeExpired`) removes expired rows with their logins; `contracts.ListProjectClients` names a project's live clients for the overview. Tahap 6 (ADR-0035): `GET /clients` is a flat paginated list with filters and sort; project name/date and the sign-in state are read in one batch from `projects` and `identity` for a page (ordering by them is done after reading them, never by a join) | `clients` | `projects` (validate `project_id` exists; `ProjectLabels` — names and event dates of many projects, Tahap 6), `identity` (create/reset credentials, deactivate/reactivate on delete/restore, delete the login on purge; `LoginStatuses` for the list) |
| `vendors` | Vendor directory + categories, tenant-scoped, plus (ADR-0016) the Venue directory — its own table, but still this module, since Venue and Vendor are conceptually siblings. Tahap 6 (ADR-0035): the directory lists take `?sort=` and `?isActive=`, vendors carry `categoryName` (a join inside this module), and `POST /vendors/bulk` / `POST /venues/bulk` activate or deactivate up to 100 rows in one transaction. Paket WO Tahap 1 (ADR-0048): a venue has two capacities (standing, seated — at least one) and three prices (Akad, Resepsi, Akad + Resepsi) plus the optional charge, in the form, import template and export alike; `contracts` adds `GetVenueRef`/`GetVendorRef`/`VenueNames` (`VenueRef`, `VendorRef`) for `offers`; a permanent delete of a vendor, venue or category a package refers to is refused (409 `vendor_in_offers`/`venue_in_offers`/`category_in_offers`, `usedInOffers` in `delete-impact`) through the module's own `OfferReferences` port | `vendors`, `vendor_categories`, `venues` | `projects` (resolve a vendor's cross-project history for "Lihat Project" from the project's suppliers — `project_suppliers` is owned by `projects`, not `vendors`; also the projects a vendor, venue or staff member is still used in, for `delete-impact`), `offers` (whether a package still refers to a row — the `OfferReferences` port, set with `SetOfferReferences` in `cmd/server`; vendors never imports offers) |
| `offers` | The WO's catalog for quoting (paket WO Tahap 1, ADR-0049) and its quotations (Penawaran, paket WO Tahap 5, ADR-0050): event kinds (Jenis acara — six built-in kinds seeded per workspace, renamable, deactivatable, deletable only when unused) and packages (Paket — price, covered kinds, included guests and an extra-guest step, an optional default venue, components with an optional category, vendor and estimated cost). Holds the pure pricing and capacity rules quotations and projects share (`domain/rules.go`: `ExtraGuests`, `LineTotals` with PPN, `CheckCapacity`). `/event-kinds`, `/packages`, `POST /packages/preview`; reads open to any staff, writes need `package.manage`; `estimatedCost` only with `finance.margin.view`. Tahap 5: `/quotations/**` — numbered `KODE-PNW-YYMMDD-NNN` per workspace and WIB day, the same events/Vendor & venue/lines shape as a project plus payment terms and T&C, revisions frozen as JSON snapshots and deduplicated by content hash, explicit status transitions (Draft, Accepted, Rejected, Superseded; Kedaluwarsa computed), copy and alternatives, and PDFs rendered with `go-pdf/fpdf` from a snapshot (`infrastructure/pdf`); `quotation.view`/`quotation.manage`, `assigned` scope = PIC. `contracts` serves `SeedDefaultEventKinds`, `VendorReferenced`/`VenueReferenced`/`CategoryReferenced` (packages, then quotations) and `Quotations` (`AcceptedQuotation`, `MarkConverted`, `MarkProjectDeleted` — for "Jadikan proyek" in `projects`) | `event_kinds`, `packages`, `package_event_kinds`, `package_components`, `quotations`, `quotation_counters`, `quotation_suppliers`, `quotation_events`, `quotation_event_sessions`, `quotation_lines`, `quotation_terms`, `quotation_revisions`, `quotation_activity` | `vendors` (**directly**, `vendors/contracts` as the `VendorDirectory` constructor argument: validate a venue, vendor or category of the workspace, a venue's capacities, and display names). `platform` and `vendors` reach `offers` only through setters in `cmd/server` (`platformModule.SetEventKinds`, `vendorsModule.SetOfferReferences`). Tahap 5 ports filled by `offersModule.SetQuotationPorts` in `cmd/server`: `WorkspaceProfile` (document code, PKP, validity, terms text, letterhead — from `platform.LetterheadProfile`), `StaffNameResolver` (from `staff` contracts) and `FileStore` (the shared storage client); offers never imports `staff`, `platform` or `projects` |
| `projects` | The core WO object: projects, milestones ("Timeline" in the UI, display-text-only rename), events (Acara — paket WO Tahap 2, ADR-0048: kind from `offers`, date, hours, an optional venue, layout Berdiri/Duduk and 1–10 sessions; one is the main event, whose date is `projects.event_date` and moves the H-n steps with it; a project has at least one event, and its location line is derived from the events' venues), project suppliers (Vendor & venue — one row per vendor or venue in the project, with name, category and venue capacities copied from the directory, who pays it — `paidBy` WO / Klien langsung, locked once a payment or file exists —, whether the client sees its name, and the biaya rekanan `agreed_cost` shown only with `finance.margin.view`; a venue enters only through an event), vendor milestones ("Tahapan vendor", per supplier), supplier payments (Pembayaran ke rekanan — money to a vendor or venue, each row recording its payer, WO or the client directly; the WO's own payments, like the biaya rekanan, need `finance.margin.view`), client payments (money **in** from the client against the project's own `contractValue` — "Uang Masuk dari Client"), the **Rincian** (paket WO Tahap 3: `project_lines` — packages with a copied rule, their components, a computed extra-guest charge, items, bonuses and discounts — at `/projects/{id}/lines` with preview, reorder, soft delete and restore under the lines' own If-Match `linesVersion`; the contract value is a cache `contract_subtotal|tax|total` rewritten in the same transaction as every line or guest change; PPN when `projects.is_pkp`, copied from the workspace at creation; `termsMismatch` and `POST …/payment-terms/recalculate`; margin from the subtotal with a "belum lengkap" count, `finance.margin.view` only; the portal sees the Rincian through `PortalPolicy.Line`), issues, evidence (incl. a `general`-kind, per-document client-visibility toggle for standalone documents with no other specific context — powers Client Portal's own "Dokumen" tab, `GET .../documents` — see ADR-0020), activity log. Both payment ledgers support full-field Edit (no role gate) and hard delete (**Owner-or-Admin**, cascading to the payment's own evidence — ADR-0019, the first hard-delete in this codebase that isn't Owner-only). Also owns project archive (reversible) and hard delete (guarded, ADR-0013), project duplicate (clones milestones, events and the non-cancelled suppliers reset to Planned as a structural template, excludes payments/issues/evidence/activity/clients — ADR-0014), a per-tenant configurable Timeline Default Template (seeds new projects' own milestones instead of one hardcoded list shared by every tenant; each item's day-offset can now be negative — "Setelah Acara"/H+, an item scheduled after the event date), and permission checks (ADR-0026, superseding ADR-0017): a role with `assigned` scope sees only projects where the person is PIC or PIC Sales; writes are gated centrally by `staffWritePermission`, client portal reads by `portal.*` permissions. Tahap 4 (ADR-0033): signed uploads, a payment recorded with its proof and a project created with its events, their venues and its schedule are each one module-local transaction, with an optional `Idempotency-Key`. Tahap 5 (ADR-0034): `GET /projects/{id}/overview` (header, side panel and Ringkasan in one call), names (`picName`, `actorName`, `uploadedByName`, `supplierName`) in responses, a fresh `projectProgress` on every mutation, **soft delete + restore** of client payments, supplier payments and project suppliers (with a purge — `api purge` and an in-process ticker — that also removes evidence rows and stored files), kendala without a supplier and with `isClientVisible`, and the dashboard's `cashFlow`, `supplierPaymentsDue` and `unconfirmedSuppliers`. Tahap 6 (ADR-0035): `GET /projects` filters (`picStaffId`, `eventFrom`, `eventTo`), `?sort=` and the derived columns `remaining`/`nextDueDate`; `contracts.ProjectLabels` names many projects in one query for the `clients` list. ADR-0047: the Client Portal cover photo (`PUT/DELETE/GET /projects/{id}/cover`, re-encoded by `shared/compress`) and the portal's only writes — a transfer proof sent by the couple (`POST /projects/{id}/client-payments/submissions` + the signed upload, an unverified client payment with `source: "client"`) and completing a step staff flagged `forClient` (`POST /projects/{id}/milestones/{mid}/complete`), each behind its own `portal.*` permission. Paket WO Tahap 4 (vendors the client pays directly): `supplier_bill_terms` (bills of a client-paid supplier, CRUD at `/projects/{id}/supplier-bills/terms` with `finance.record`; a supplier with a bill cannot switch to WO-paid), the couple's proof of paying one (`POST /projects/{id}/supplier-bills/submissions`, `portal.finance.submit`, the portal's fifth write) as a pending `supplier_payments` row (`submitted_by_client_id`) that counts only once the WO verifies it (`POST …/supplier-payments/{pid}/verify\|reject`), the verification queue's `kind: supplierPayment` rows, the dashboard's `clientBillsDue`/`supplierProofsPending` and the portal home's `nextSupplierBill`; the portal sees bills through `PortalPolicy.BillTerm` (client-paid and `portal.finance.view`, so never Keluarga). These payments never enter the WO's cash, revenue, cash flow or payables (`payer = 'WO'` only) Paket WO Tahap 5 (PLAN §5.4, migration `000091`): "Jadikan proyek" — `POST /projects/from-quotation/{id}` (+ `/preview`, `/uploads`) copies an Accepted quotation's revision (events, sessions, Vendor & venue, lines, terms, PKP, contract cache, default Timeline, an optional down payment with its proof) in one transaction, `UNIQUE (tenant_id, quotation_id)` keeps it one project; `GET /projects/{id}/agreement` ("Kesepakatan awal") reads the frozen revision back. | `projects`, `project_lines`, `project_milestones`, `project_milestone_templates`, `project_events`, `project_event_sessions`, `project_suppliers`, `vendor_milestones`, `supplier_payments`, `supplier_bill_terms`, `client_payments`, `client_payment_terms`, `vendor_issues`, `evidence`, `activity_log`, `idempotency_keys` | `clients` (Fase 6: resolve which single project a `client` principal may read, for `/projects/{id}/...`'s client-portal read scoping; ADR-0013: best-effort delete every client tied to a project being hard-deleted; ADR-0034: list a project's live clients for the overview, through a local `ClientDirectory` port that `cmd/server` adapts from `clients.Contracts`, so `projects` still never imports `clients`), `vendors` (`projects → vendors`, the reciprocal direction of the row above: check that a vendor, category or venue is the workspace's own and copy its name, category and venue capacities into the project supplier, and read a venue's public-safe summary for the portal — `VendorDirectory` and `EventVenues`), `offers` (an event's kind name and the capacity rule — `EventCatalog`; Tahap 3: `PackageCopy`, `ExtraGuests` and `LineTotals` — the `OffersPort`, set with `SetOffers` in `cmd/server` from `offers` contracts, so a project line copies a package and prices exactly as a quotation does), `platform` (whether the workspace is PKP — the `WorkspaceProfile` port, set with `SetWorkspaceProfile` in `cmd/server` from `platform.IsPKP`; projects never imports platform), `offers` again for Tahap 5 (`offers/contracts.Quotations` — `AcceptedQuotation`, `MarkConverted`, `MarkProjectDeleted` — the `QuotationSource` port set with `SetQuotations` in `cmd/server`). Vendor/category/staff *display names* are resolved in the response where a screen needs them (ADR-0032 for vendor/category names, ADR-0034 for `picName`, `actorName`, `uploadedByName`, dashboard `supplierName` comes from the copy kept on the supplier row), always through `vendors/contracts` and `staff/contracts`, batched and scoped to the caller's workspace. |

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

**The reciprocal direction, `projects` → `vendors` (ADR-0016, ADR-0048):**
adding a vendor to a project or a venue to an event means `projects` checks the row against the
directory and copies its name, category and venue capacities onto the project supplier, and the
Client Portal shows an event venue's public-safe summary — the
opposite direction from the row above, and `vendorsModule` is built *after* `projectsModule`
(dependent on it), so `vendors.Contracts` can't be a `projects.NewModule` constructor argument
without inverting the whole build order. Same setter idiom as every other two-phase case in this
file: `projects/application` declares its own narrow consumer interfaces (`VendorDirectory`,
`EventVenues`, mirroring `ClientCleaner`'s existing shape) returning types from `vendors/contracts`;
`main.go` builds both modules, then bridges with `projectsModule.SetVendorDirectory(...)` and
`projectsModule.SetEventVenues(vendorsModule.Contracts())`. `offers` is built after `vendors`, so the
event kinds come the same way: `projectsModule.SetEventCatalog(offersModule.Contracts())`.

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

**`offers` → `vendors` direct, `platform`/`vendors` → `offers` by setter (paket WO Tahap 1, ADR-0049):**
`offers.NewModule` takes `vendorsModule.Contracts()` as a constructor argument (an ordinary one-way
dependency, so `main.go` builds `offersModule` right after `vendorsModule`). The two modules that need
something back from `offers` declare their own port and get it by setter, so no import cycle can
form: `vendors/application.OfferReferences` (refuse a permanent delete of a row a package refers to —
`vendorsModule.SetOfferReferences(offersModule.Contracts())`) and `platform/application.EventKindSeeder`
(seed the default event kinds at sign-up — `platformModule.SetEventKinds(offersModule.Contracts())`).
Unset, both behave as "nothing refers to anything" / "no kinds to seed".

**`offers` quotation ports (paket WO Tahap 5, ADR-0050):** after `projectsModule.SetOffers(...)`,
`main.go` calls `offersModule.SetQuotationPorts(quotationProfile{platformModule}, staffNameResolver{…},
storageClient)`. `quotationProfile` (in `cmd/server/quotation_wiring.go`) adapts
`platform.Module.LetterheadProfile` to the `offers/application.WorkspaceProfile` port; the PDF renderer
is built inside `offers` (`infrastructure/pdf`). `offersModule.Quotations()` is the
`offers/contracts.Quotations` value `projects` will take for "Jadikan proyek".

## Frontend module ↔ backend module

| Frontend (`apps/web/src/modules/*`) | Backend module(s) it talks to |
|---|---|
| `auth` | `identity` |
| `dashboard` | `projects` (aggregation endpoint) |
| `projects` | `projects` — "Acara & rincian" wires `modules/rincian` to the project store (`fetchRincian`/`previewRincian`/`saveRincian`, `recalculatePaymentTerms`); Paket WO Tahap 4: `useSupplierBillStore` (`GET/POST/PATCH/DELETE …/supplier-bills*`, `…/supplier-payments/{pid}/verify\|reject`) behind Keuangan › Dibayar klien langsung (`ClientPaidBook`, `?periksa=<paymentId>` opens `SupplierProofReviewDrawer`) and the vendor drawer (`SupplierBillsPanel`) |
| `rincian` | none of its own (ADR-0049): the data-source-agnostic event, vendor/venue and line editors (`RincianEditor`, `CostSummaryPanel`, `PortalRincian`, `EventList`, `EventFields`, `SupplierFields`) fed through props and callbacks. `index.ts` is its only public entry; `projects`, `client-portal` and `offers` import that entry and nothing behind it |
| `clients` | `clients` |
| `vendors` | `vendors` |
| `vendor-categories` | `vendors` |
| `venues` | `vendors` (ADR-0016) |
| `packages` | `offers` (ADR-0049) — Katalog › Paket (`/paket`) |
| `event-kinds` | `offers` (ADR-0049) — Katalog › Jenis acara (`/jenis-acara`) |
| `business-profile` | `platform` — Profil usaha, including Penawaran (`/tenants/me/quote-settings`) |
| `offers` | `offers` (ADR-0050) — Penawaran (`/penawaran`, `?baru=1` opens "Penawaran baru"; `/penawaran/{id}`; `/penawaran/{id}/jadikan-proyek`) through `useQuotationStore`, the rincian editors fed by `lib/rincian-adapter.ts`; `projects` for "Jadikan proyek" (`POST /projects/from-quotation/{id}` + `/preview`, `/uploads`) and "Kesepakatan awal" (`GET /projects/{id}/agreement`, `AgreementDrawer`, lazy-loaded by Proyek › Acara & rincian with `?kesepakatan=1`) |
| `milestone-templates` | `projects` — "Pengaturan → Timeline Default," Owner-only |
| `users` | `staff` |
| `subscription` | `billing` |
| `client-portal` | `projects`, `clients` (read-scoped); Layanan › "Paket & rincian" reads `GET /projects/{id}/lines` (redacted for a client) and renders `PortalRincian`; Pembayaran › "Tagihan vendor" (`VendorBillsSection`, "Kirim bukti bayar" → `POST …/supplier-bills/submissions` through `useSupplierBillStore`) and Beranda's `NextVendorBillCard` (`finance.nextSupplierBill`) |
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
