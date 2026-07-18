# Module Map

> Each module's responsibility and the public contract it exposes.

Backend modules live at `apps/api/internal/modules/<name>`. See ADR-0008 for how these map to the
frontend's own module folders under `apps/web/src/modules/<name>`.

| Module | Responsibility | Owns (tables) | Consumes (via other modules' contracts) |
|---|---|---|---|
| `identity` | Login/refresh/logout for all principal types; password hashing; token issuance | `credentials`, `refresh_tokens` | none — deliberately profile-agnostic (see ADR-0005) |
| `payment` | One payment-gateway merchant wallet (Tripay), wrapped for many consumers ("Apps") — gateway config, App registry, webhook dispatch. Owns no business ledger of its own — see `MODULE_PAYMENT.md`. Fase 9: internal mode (`platform` as the one App internal). Fase 10 (implemented): external Apps over HTTP — `/auth/app/token` + `/external/payments/*`, Platform Console's "Manajemen Aplikasi" page. | `payment_gateway_config`, `payment_apps`, `payment_charge_dispatch`, `payment_webhook_events` | `identity` (mint bearer tokens for external Apps — one-way, same shape as `vendors -> projects` below) |
| `platform` | Tenant lifecycle (register/suspend/activate/pay), Platform Console's own admin accounts | `tenants`, `platform_admins`, `pending_subscription_charges` | `staff` (create Owner on tenant registration), `identity` (create credentials), `billing` (read plan, record/update transaction), `payment` (create charge; also registers itself as `payment`'s webhook consumer for `platform-billing`, Fase 9) |
| `billing` | Subscription plan catalog + subscription transaction ledger — single source of truth shared by both consoles | `subscription_plans`, `subscription_transactions` | none |
| `staff` | WO internal users (Owner/Admin/Staff), tenant-scoped | `staff_members` | none |
| `clients` | Client contacts per project, tenant-scoped | `clients` | `projects` (validate `project_id` exists) |
| `vendors` | Vendor directory + categories, tenant-scoped | `vendors`, `vendor_categories` | `projects` (resolve a vendor's cross-project engagement history for "Lihat Project" — `project_vendors` is owned by `projects`, not `vendors`) |
| `projects` | The core WO object: projects, milestones, vendor engagements, vendor milestones, payments, issues, evidence, activity log | `projects`, `project_milestones`, `project_vendors`, `vendor_milestones`, `vendor_payments`, `vendor_issues`, `evidence`, `activity_log` | `clients` (Fase 6: resolve which single project a `client` principal may read, for `/projects/{id}/...`'s client-portal read scoping). Vendor/category/staff *display names* are **not** resolved backend-side — the frontend resolves those from its own already-fetched `vendors`/`staff` stores. |

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

**`payment` → `identity`, the same plain one-way shape (Fase 10):** `payment.NewModule` takes
`identity.Contracts` as a constructor argument, to mint bearer tokens for external Apps
(`IssueServiceToken`, see `MODULE_PAYMENT.md` §7.1) — `identity` never needs anything back from
`payment`, so `main.go` simply builds `identityModule` before `paymentModule` (already the existing
order). `payment` still depends on nothing else, and no other App internal's webhook-dispatch
bridging (see above) changes because of this.

## Frontend module ↔ backend module

| Frontend (`apps/web/src/modules/*`) | Backend module(s) it talks to |
|---|---|
| `auth` | `identity` |
| `dashboard` | `projects` (aggregation endpoint) |
| `projects` | `projects` |
| `clients` | `clients` |
| `vendors` | `vendors` |
| `vendor-categories` | `vendors` |
| `users` | `staff` |
| `subscription` | `billing` |
| `client-portal` | `projects`, `clients` (read-scoped) |
| `platform-admin` | `platform`, `billing`, `payment` (Konfigurasi Gateway + Manajemen Aplikasi pages) |
| `homepage` | none — public marketing pages (`/homepage`, `/homepage/tentang-kami`, `/homepage/syarat-ketentuan`, `/homepage/kebijakan-privasi`, `/homepage/kebijakan-refund`, `/homepage/faq`, `/homepage/kontak`), static content only, no API calls |

Full column-level schema: [`docs/DB_SCHEMA.md`](../docs/DB_SCHEMA.md). Full endpoint list:
[`docs/API_CONTRACT.md`](../docs/API_CONTRACT.md).
