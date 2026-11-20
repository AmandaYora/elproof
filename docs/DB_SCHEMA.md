# Database Schema — ElProof

> Tables, fields, ownership, relations as primitive IDs. Conventions: see
> [`knowledge/DATABASE_GUIDE.md`](../knowledge/DATABASE_GUIDE.md). Ownership map:
> [`knowledge/MODULE_MAP.md`](../knowledge/MODULE_MAP.md).

Legend: `PK` primary key, `FK*` cross-module reference stored as a plain value (no SQL foreign key
constraint — resolved via that module's contract, never joined), `FK` same-module foreign key
(constraint allowed).

---

## Module `identity`

### `credentials`
| Column | Type | Notes |
|---|---|---|
| id | BIGINT UNSIGNED PK | |
| tenant_id | BIGINT UNSIGNED NULL | `FK*` → `platform.tenants.id`. Null for `platform_admin`. |
| principal_type | ENUM('staff','client','platform_admin') | |
| principal_id | VARCHAR(64) | `FK*` → `staff.staff_members.id` / `clients.clients.id` / `platform.platform_admins.id` depending on `principal_type` |
| username | VARCHAR(100) UNIQUE | |
| email | VARCHAR(150) NULL UNIQUE | denormalized copy of the owning module's email (`staff_members`/`clients`/`platform_admins`), written at `CreateCredential` time — lets `Login` resolve either identifier without a cross-module join (migration `000015`). Unique since migration `000016` (rejected in `CreateCredential` before it ever reaches the DB constraint) — login-by-email would otherwise be ambiguous between two accounts sharing one email. MySQL's `UNIQUE` allows any number of `NULL` rows (accounts never backfilled), just never two equal non-null values. |
| password_hash | VARCHAR(255) | bcrypt |
| role | VARCHAR(50) | e.g. `Owner`/`Admin`/`Staff`, `Super Admin`/`Support` — denormalized for JWT claim convenience |
| display_name | VARCHAR(150) | denormalized for immediate post-login UI display (see ADR-0005) |
| is_active | BOOLEAN DEFAULT TRUE | |
| created_at, updated_at | TIMESTAMP | |

### `refresh_tokens`
| Column | Type | Notes |
|---|---|---|
| id | BIGINT UNSIGNED PK | |
| credential_id | BIGINT UNSIGNED `FK` → credentials.id | same-module FK, allowed |
| family_id | BIGINT UNSIGNED NULL | id of the first token of the sign-in this one was rotated from; NULL = this token is the family root (migration `000076`, ADR-0044). Indexed; logout and reuse detection revoke `id = f OR family_id = f` |
| token_hash | VARCHAR(255) | sha256 of the opaque refresh token |
| mfa | BOOLEAN | the sign-in used a second factor (migration `000052`) |
| expires_at | TIMESTAMP | |
| revoked_at | TIMESTAMP NULL | |
| created_at | TIMESTAMP | |

---

## Module `platform`

### `tenants`
| Column | Type | Notes |
|---|---|---|
| id | BIGINT UNSIGNED PK | |
| business_name | VARCHAR(150) | |
| owner_name | VARCHAR(150) | |
| username | VARCHAR(100) | the Owner's login username (mirrors `credentials.username`) |
| email | VARCHAR(150) | |
| phone | VARCHAR(30) | |
| city | VARCHAR(100) | |
| joined_at | DATE | |
| plan_id | BIGINT UNSIGNED NULL | `FK*` → `billing.subscription_plans.id` |
| subscription_status | ENUM('active','expiring_soon','expired','pending_payment') | |
| subscription_expires_at | DATE NULL | |
| is_suspended | BOOLEAN DEFAULT FALSE | |
| last_credential_reset_at | DATE NULL | |
| logo_storage_path | VARCHAR(500) NULL | object storage key (same `shared/storage` utility as `evidence.storage_path`, ADR-0006) — `NULL` means no logo configured yet |
| created_at, updated_at | TIMESTAMP | |

### `platform_admins`
| Column | Type | Notes |
|---|---|---|
| id | BIGINT UNSIGNED PK | |
| name | VARCHAR(150) | |
| title | VARCHAR(100) | |
| role | ENUM('Super Admin','Support') | |
| username | VARCHAR(100) | mirrors `credentials.username` |
| email | VARCHAR(150) | |
| phone | VARCHAR(30) | |
| is_active | BOOLEAN DEFAULT TRUE | |
| created_at, updated_at | TIMESTAMP | |

---

## Module `billing`

### `subscription_plans`
| Column | Type | Notes |
|---|---|---|
| id | BIGINT UNSIGNED PK | |
| app_id | VARCHAR(100) NOT NULL | `FK*` → `payment.payment_apps.app_id` — the one App that sells this plan (migration `000073`, ADR-0040); cross-module primitive ID, no SQL constraint, checked by `POST`/`PATCH /plans`. `platform-billing` = ElProof's own catalog for WO tenants. Indexed |
| name | VARCHAR(100) | |
| duration_months | SMALLINT UNSIGNED | |
| price | BIGINT UNSIGNED | Rupiah, integer |
| is_active | BOOLEAN DEFAULT TRUE | |
| created_at, updated_at | TIMESTAMP | |

### `plan_features`
| Column | Type | Notes |
|---|---|---|
| id | BIGINT UNSIGNED PK | |
| plan_id | BIGINT UNSIGNED `FK` → subscription_plans.id | same-module FK |
| label | VARCHAR(255) | |
| sort_order | SMALLINT UNSIGNED | preserves CRUD order from `PlanFormModal` |

**Resolved (Fase 2):** implemented as this normalized table (not a JSON column) — `Update` deletes
and re-inserts the full feature list per save, matching `PlanFormModal`'s "whole list replace"
submit semantics.

**Migration `000083` (EP-14, data only):** a `label` still equal to the English text typed on production
(`Full Features`/`Full Feature` → `Semua fitur`, `Full Seat(s)` → `Anggota tim tanpa batas`, `Unlimited Project(s)`
→ `Proyek tanpa batas`; exact, case-sensitive match after trimming spaces) is rewritten; any other label is left
alone. Down puts the English text back only where the row still holds exactly what up wrote.

### `subscription_transactions`
| Column | Type | Notes |
|---|---|---|
| id | BIGINT UNSIGNED PK | |
| tenant_id | BIGINT UNSIGNED | `FK*` → `platform.tenants.id` |
| type | ENUM('new','renewal') | |
| amount | BIGINT UNSIGNED | what was charged (after a voucher discount) — revenue reports read this |
| original_amount, discount_amount | BIGINT UNSIGNED | migration `000074` (ADR-0041): price before the voucher and the discount; without a voucher `original_amount = amount`, `discount_amount = 0` |
| voucher_code | VARCHAR(32) NULL | the voucher used, if any |
| payment_method | VARCHAR(100) | |
| payment_reference | VARCHAR(100) | |
| status | ENUM('unpaid','pending','paid','expired','granted') | `pending` (Fase 9) = a real gateway charge exists, awaiting webhook confirmation — distinct from `unpaid` (no charge attempt yet, e.g. right after tenant registration). `granted` = manual activation, excluded from paid-revenue reports |
| created_at | TIMESTAMP | |
| paid_at | TIMESTAMP NULL | |

`subscription_plan_apps` (migration `000043`, the many-to-many visibility mapping of ADR-0021) was
dropped by migration `000073` (ADR-0040): every plan now names its one owning App in
`subscription_plans.app_id`.

### `subscription_app_charges` — migration `000044`, `docs/plan/external-subscription-plans/PLAN.md`
| Column | Type | Notes |
|---|---|---|
| order_ref | VARCHAR(100) PK | `FK*` → `payment.payment_charge_dispatch.order_ref` — cross-module primitive ID, ties this ledger row back to the underlying gateway charge |
| app_id | VARCHAR(100) | `FK*` → `payment.payment_apps.app_id`, no SQL constraint |
| plan_id | BIGINT UNSIGNED `FK` → subscription_plans.id | same-module FK |
| plan_name, plan_price, plan_duration_months | VARCHAR(150), BIGINT UNSIGNED, INT | **snapshot** of the plan at the moment the intent row is written (before `payment.Client` is ever called) — a later catalog price edit never rewrites what a customer already agreed to pay |
| discount_amount, amount, voucher_code | BIGINT UNSIGNED, BIGINT UNSIGNED, VARCHAR(32) NULL | migration `000074` (ADR-0041): the voucher discount, the amount actually charged (`plan_price − discount_amount`) and the code |
| status | ENUM('pending','paid','expired','failed','cancelled') | |
| subscription_until | DATE NULL | filled only when `status='paid'` — the answer to "paid up until when" |
| created_at | TIMESTAMP | |
| paid_at | TIMESTAMP NULL | |
| resolved_at | TIMESTAMP NULL | always written in the **same** `UPDATE` as `status` — never a column set on its own. `WHERE resolved_at IS NULL` is the single-winner guard `MarkPaidIfUnresolved`/`MarkResolvedIfUnresolved` use; there is no separate "claim" step, because a claim-then-write split would leave a window where a row is claimed but still `status='pending'` if the process died in between — losing an already-paid charge permanently |

Not a reuse of `subscription_transactions` above — that table's `tenant_id` is `NOT NULL` and is
actively used by the live WO-tenant subscription flow, so an App-keyed row (which has no tenant)
would require changing that column's nullability on a table already carrying real traffic. This
ledger is intentionally empty until an App switches its payment call from `payment`'s
`/external/payments/charges` to `billing`'s own `/external/subscriptions/charges` (see
`docs/API_CONTRACT.md`) — an empty table here is expected, not a bug.

`KEY(app_id, status)` and `KEY(resolved_at, created_at)` back the reconciler's own polling query
(`billing`'s `SubscriptionReconciler`, independent from `payment`'s own reconciler — see
`knowledge/MODULE_PAYMENT.md`).

### `subscription_vouchers` — migration `000074`, ADR-0041
| Column | Type | Notes |
|---|---|---|
| id | BIGINT UNSIGNED PK | |
| code | VARCHAR(32) utf8mb4_bin UNIQUE | stored upper case; the service upper-cases every input |
| app_id | VARCHAR(100) utf8mb4_bin | `FK*` → `payment.payment_apps.app_id` (the App whose plans it discounts; `platform-billing` = ElProof), no SQL constraint |
| plan_id | BIGINT UNSIGNED NULL `FK` → subscription_plans.id | NULL = every plan of the App |
| discount_type, discount_value | ENUM('percent','amount'), BIGINT UNSIGNED | percent 1–99, or rupiah |
| max_discount | BIGINT UNSIGNED NULL | cap of a percent discount |
| valid_from, valid_until | DATE NULL | inclusive WIB days |
| quota | INT UNSIGNED NULL | total uses (reserved + used); NULL = unlimited |
| once_per_customer, is_active | BOOLEAN | |
| created_at, updated_at | TIMESTAMP(6) | `updated_at` is the If-Match version |

### `subscription_voucher_redemptions` — migration `000074`, ADR-0041
One row per use of a voucher on one subscription charge: `voucher_id` (FK), `voucher_code` (snapshot), `order_ref`
(UNIQUE; `subscription_transactions.payment_reference` or `subscription_app_charges.order_ref`), `customer_type`
(`tenant`\|`app`) + `customer_ref` (tenant id or app_id, primitive), `plan_id`, `original_amount`, `discount_amount`,
`final_amount`, `status` (`reserved` while the charge is pending, `used` once paid, `released` when it expired / failed /
 was cancelled — released rows no longer count), `created_at`, `resolved_at`. A reservation is written under
`SELECT … FOR UPDATE` on the voucher row (READ COMMITTED), so the quota and once-per-customer rule hold under concurrent
charges.

---

## Module `staff` — **implemented, Fase 3**

### `staff_members`
| Column | Type | Notes |
|---|---|---|
| id | BIGINT UNSIGNED PK | |
| tenant_id | BIGINT UNSIGNED NOT NULL | scoping column, see ADR-0004 |
| name | VARCHAR(150) | |
| title | VARCHAR(100) | |
| initials | VARCHAR(4) | computed at create time, stored for display |
| role | ENUM('Owner','Admin','Staff') | Owner rows only insertable via `platform` module's tenant-registration orchestration |
| username | VARCHAR(100) NOT NULL DEFAULT '' | mirrors `credentials.username` (added migration `000013`) — every row now gets a real login credential via `identity.CreateCredential` at create time (Owner rows too, via `platform`'s registration flow) |
| email | VARCHAR(150) | |
| phone | VARCHAR(30) | |
| is_active | BOOLEAN DEFAULT TRUE | |
| created_at, updated_at | TIMESTAMP | |

---

## Module `clients` — **implemented, Fase 4** (bundled with `projects` since
`clients.project_id` needs a real `projects` row to reference)

### `clients`
| Column | Type | Notes |
|---|---|---|
| id | BIGINT UNSIGNED PK | |
| tenant_id | BIGINT UNSIGNED NOT NULL | |
| project_id | BIGINT UNSIGNED | `FK*` → `projects.projects.id` |
| role | ENUM('Bride','Groom','Family Representative') | |
| username | VARCHAR(100) NOT NULL DEFAULT '' | mirrors `credentials.username` (added migration `000012`) |
| relation_note | VARCHAR(255) NULL | |
| name | VARCHAR(150) | |
| phone | VARCHAR(30) | |
| email | VARCHAR(150) | |
| is_active | BOOLEAN DEFAULT TRUE | |
| last_credential_reset_at | DATE NULL | |
| created_at, updated_at | TIMESTAMP | |
| deleted_at | DATETIME NULL, `KEY idx_clients_deleted` | migration `000068`, ADR-0034 — soft delete: `NULL` = live. A stamped row is invisible to every read (lists, counts, `CountActive`, the project's client slots, the portal's project lookup); the client's `identity` login is **kept but deactivated** (so restore is always possible and the username/email stay reserved until the purge removes the login). Hard-deleted by the purge once older than 24 h |

*(§6.3 resolved: "replace representative" (`POST /clients/{id}/replace-representative`) overwrites
this row with no history table — a deliberate choice carried over from the frontend mock's existing
behavior, not an oversight. Revisit only if a real need for an audit trail shows up in practice.)*

**Create is not (and cannot be) wrapped in a single DB transaction with `identity.CreateCredential`**
(separate modules own separate tables — no cross-module transaction, per the modular-monolith
rule). If credential creation fails after the `clients` row already committed (e.g. the username —
unique *platform-wide*, not per-tenant — collides with any other principal), `ClientService.Create`
compensates by deleting the row it just inserted, rather than leaving a client with no working
login. `DELETE /clients/{id}` (self-service, see API_CONTRACT.md) exists specifically to clear out
any client that predates this fix and is still stuck that way — deactivating alone doesn't free up
its role slot on the project, since the role lookup doesn't filter on `is_active`.

---

## Module `vendors` — **implemented, Fase 3**

### `vendor_categories`
| Column | Type | Notes |
|---|---|---|
| id | BIGINT UNSIGNED PK | |
| tenant_id | BIGINT UNSIGNED NOT NULL | |
| name | VARCHAR(100) | unique per tenant **in the service** (FQ-03): a create or rename onto another category's name (case-insensitive, trimmed) is a 422. No unique index: production already holds duplicates, which stay as they are |
| description | VARCHAR(255) | |
| is_active | BOOLEAN DEFAULT TRUE | |
| created_at, updated_at | TIMESTAMP | |

"Venue" was removed from this table's default seed template (ADR-0016) — a tenant that already had
it gets that row deactivated by data migration, never deleted (this module's usual convention).
Venue is now its own directory (`venues`, below), not a category.

### `vendors`
| Column | Type | Notes |
|---|---|---|
| id | BIGINT UNSIGNED PK | |
| tenant_id | BIGINT UNSIGNED NOT NULL | |
| category_id | BIGINT UNSIGNED `FK` → vendor_categories.id | same-module FK |
| name | VARCHAR(150) | |
| pic_name | VARCHAR(150) | |
| phone | VARCHAR(30) | |
| email | VARCHAR(150) NULL | no longer mandatory — migration `000023` |
| social_media | TEXT NULL | free text, migration `000023` |
| city | VARCHAR(100) NULL | migration `000023` |
| price_akad | BIGINT UNSIGNED NULL | vendor's own preset price for an Akad only — migration `000023`. Reference/catalog price only: what the WO agreed with the vendor on a project is that project's `project_suppliers.agreed_cost` (Biaya rekanan), negotiated freely |
| price_resepsi | BIGINT UNSIGNED NULL | same as above, for "Resepsi" — migration `000042` |
| price_akad_resepsi | BIGINT UNSIGNED NULL | same as above, for "Akad + Resepsi" |
| address | VARCHAR(255) NULL | no longer mandatory — migration `000023` |
| notes | TEXT NULL | |
| attachment_path | VARCHAR(500) NULL | object storage key (single slot, document or photo — same convention as `venues.attachment_path` below), migration `000023` |
| attachment_mime_type | VARCHAR(100) NULL | migration `000023` |
| is_active | BOOLEAN DEFAULT TRUE | |
| created_at, updated_at | TIMESTAMP | |

Composite index `(tenant_id, city)` — migration `000033` — backs the Vendor list page's Kota
filter.

### `venues` — **ADR-0016**
Its own directory inside `vendors` (not a new top-level module — Venue and Vendor are conceptually
siblings), not a vendor category. A venue enters a project only through an event (`project_events.venue_supplier_id`
→ a `project_suppliers` row of kind `venue`, below), so a project may use several venues, one per event or
shared between events (ADR-0048).

| Column | Type | Notes |
|---|---|---|
| id | BIGINT UNSIGNED PK | |
| tenant_id | BIGINT UNSIGNED NOT NULL | |
| name | VARCHAR(150) | |
| pic_name | VARCHAR(150) | mandatory at creation |
| phone_pic | VARCHAR(30) | mandatory — the PIC's own personal contact number |
| phone_venue | VARCHAR(30) NULL | optional official/office number, distinct from `phone_pic` |
| email | VARCHAR(150) NULL | |
| address | VARCHAR(255) NULL | optional (Revisi — was mandatory in the original ADR-0016 plan) |
| city | VARCHAR(100) NULL | mandatory on the create form, validated against a fixed 128 kota/kabupaten list; nullable at the DB only so the one-time vendor→venue data migration doesn't fail for rows with no city info |
| capacity_standing | INT UNSIGNED NULL | capacity for a standing layout (Kapasitas berdiri) — migration `000084` (ADR-0048, paket WO Tahap 1). At least one of `capacity_standing`/`capacity_seated` is required (≥ 1) by the service, the form and the import; the DB keeps both nullable. Packages and quotation events check guests against the number for their layout |
| capacity_seated | INT UNSIGNED NULL | capacity for a seated layout (Kapasitas duduk) — migration `000084` |
| price_akad | BIGINT UNSIGNED NULL | rental price for an Akad — migration `000084`; optional |
| price_resepsi | BIGINT UNSIGNED NULL | rental price for a Resepsi — migration `000084`; optional |
| price_akad_resepsi | BIGINT UNSIGNED NULL | rental price for Akad + Resepsi — migration `000084`; optional |
| charge | BIGINT UNSIGNED NULL | separate optional extra charge (Charge tambahan), distinct from the three prices |
| facilities | TEXT NULL | free text (Revisi — the original ADR-0016 plan used a JSON array; simplified for UI consistency + easier bulk-import) |
| social_media | TEXT NULL | free text, same Revisi reasoning as `facilities` |
| notes | TEXT NULL | |
| attachment_path | VARCHAR(500) NULL | single slot — document OR photo (Revisi — the original plan had a separate `venue_photos` gallery table, dropped entirely) |
| attachment_mime_type | VARCHAR(100) NULL | determines Client Portal visibility: an image is treated as a venue photo (client-visible); a non-image (e.g. PDF contract) is treated as an internal document (hidden from `client` principals) — enforced at the file-stream endpoint itself, not just the frontend |
| is_active | BOOLEAN DEFAULT TRUE | |
| created_at, updated_at | TIMESTAMP | |

Existing tenants' real venue data (previously `vendors` rows under a "Venue" category) was
one-time-migrated into this table by `000022_migrate_venue_vendor_data` — new fields land empty for
the WO to fill in; the source `vendors` rows are deactivated, not deleted, and never auto-linked to
a project's history (fuzzy-matching risk — see ADR-0016).

Composite index `(tenant_id, city)` — migration `000033` — same shape as `vendors` above, backs the
Venue list page's Kota filter.

Migration `000084` replaced the single `capacity` and `rental_price` columns: the old capacity
became `capacity_standing`, the old rental price became `price_akad_resepsi`, then both old columns
were dropped (pre-launch, no compatibility columns). The import template and export columns follow
the same shape (`Kapasitas Berdiri`, `Kapasitas Duduk`, `Harga Akad`, `Harga Resepsi`,
`Harga Akad+Resepsi`, `Charge`).

Both `vendors` and `venues` (and `staff`, and `vendor_categories`) can now also be permanently hard-
deleted (Owner-only, informed-consent — ADR-0018) — no schema change for this, since every
cross-module reference to these rows was already a primitive, un-enforced ID.
Since paket WO Tahap 1 the delete is refused (409 `vendor_in_offers` / `venue_in_offers` /
`category_in_offers`) while a row of the `offers` module (a package's default venue or a
component's vendor or category) still points at it — checked through vendors' `OfferReferences`
port, still with no SQL foreign key.

---

## Module `offers` — migration `000086`, ADR-0049 (paket WO Tahap 1)

The catalog a WO quotes from: the kinds of event it organises and its packages, and (Tahap 5)
the quotations themselves. Every table carries `tenant_id` (ADR-0004), child tables included, and every unique key
leads with it. Venue, vendor and vendor category are primitive ids into `vendors` (`FK*`, resolved
through `vendors.Contracts`); the only SQL foreign keys stay inside this module.

### `event_kinds`
| Column | Type | Notes |
|---|---|---|
| id | BIGINT UNSIGNED PK | |
| tenant_id | BIGINT UNSIGNED NOT NULL | `FK*` → `platform.tenants.id` |
| name | VARCHAR(60) NOT NULL | Jenis acara. `UNIQUE uq_event_kinds_tenant_name (tenant_id, name)`; the service also refuses a case-insensitive duplicate (422). Quotation and project events copy the name, so a rename never rewrites an earlier document |
| default_key | VARCHAR(20) NULL | which built-in kind the row was seeded as (`lamaran`, `pengajian`, `siraman`, `akad`, `resepsi`, `ngunduh_mantu`); `NULL` for a kind the WO added. Survives a rename, so seeding stays idempotent. `UNIQUE uq_event_kinds_tenant_default (tenant_id, default_key)` |
| sort_order | INT NOT NULL DEFAULT 0 | display order; a new kind goes last |
| is_active | BOOLEAN NOT NULL DEFAULT TRUE | an inactive kind cannot be picked for a new package or event |
| created_at | TIMESTAMP | |
| updated_at | TIMESTAMP(6) | the `If-Match` version of a rename (ADR-0036) |

The migration seeds the six built-in kinds for every existing tenant (`INSERT IGNORE`); a new
workspace gets them at sign-up (`platform` → `offers.SeedDefaultEventKinds`). A kind a package covers
cannot be deleted (409 `event_kind_in_use`); it is deactivated instead.

### `packages`
| Column | Type | Notes |
|---|---|---|
| id | BIGINT UNSIGNED PK | |
| tenant_id | BIGINT UNSIGNED NOT NULL | `FK*` → `platform.tenants.id` |
| name | VARCHAR(150) NOT NULL | `UNIQUE uq_packages_tenant_name (tenant_id, name)`; the service also refuses a case-insensitive duplicate |
| price | BIGINT UNSIGNED NOT NULL | package price in rupiah |
| included_guests | INT UNSIGNED NULL | Tamu termasuk: how many guests `price` covers; `NULL` = the price does not depend on guests |
| extra_guest_step | INT UNSIGNED NULL | Tambahan per … tamu: every started step of this many guests above `included_guests` costs `extra_guest_step_price`. Both step columns are `NULL` (no charge) or both set, and only with `included_guests` (service) |
| extra_guest_step_price | BIGINT UNSIGNED NULL | price per step |
| default_venue_id | BIGINT UNSIGNED NULL | `FK*` → `vendors.venues.id`; `KEY idx_packages_tenant_venue (tenant_id, default_venue_id)` backs the venue delete check. `included_guests` may not exceed the venue's largest capacity (service) |
| is_active | BOOLEAN NOT NULL DEFAULT TRUE | |
| created_at | TIMESTAMP | |
| updated_at | TIMESTAMP(6) | the `If-Match` version of an edit (ADR-0036) |

### `package_event_kinds`
| Column | Type | Notes |
|---|---|---|
| tenant_id | BIGINT UNSIGNED NOT NULL | |
| package_id | BIGINT UNSIGNED | `FK` → `packages.id` `ON DELETE CASCADE` |
| event_kind_id | BIGINT UNSIGNED | `FK` → `event_kinds.id` (no cascade: a kind in use cannot be deleted) |

`PK (package_id, event_kind_id)`, `KEY idx_package_event_kinds_kind (tenant_id, event_kind_id)`. The
kinds a package covers; at least one (service).

### `package_components`
| Column | Type | Notes |
|---|---|---|
| id | BIGINT UNSIGNED PK | |
| tenant_id | BIGINT UNSIGNED NOT NULL | |
| package_id | BIGINT UNSIGNED | `FK` → `packages.id` `ON DELETE CASCADE` |
| description | VARCHAR(255) NOT NULL | Isi paket ("Katering buffet 450 porsi") |
| category_id | BIGINT UNSIGNED NULL | `FK*` → `vendors.vendor_categories.id`, optional; filled from the vendor's category when only `vendor_id` is given |
| vendor_id | BIGINT UNSIGNED NULL | `FK*` → `vendors.vendors.id`, optional |
| estimated_cost | BIGINT UNSIGNED NULL | the WO's own cost estimate; read and written only with `finance.margin.view` (an edit without it keeps the stored value) |
| sort_order | INT NOT NULL DEFAULT 0 | row order in the form |

Keys `(tenant_id, package_id)`, `(tenant_id, vendor_id)`, `(tenant_id, category_id)` — the last two
back the vendor and category delete checks. At most 50 components per package (service). A package
edit rewrites its kinds and components in one transaction.

Migration `000086` also grants the new permission `package.manage` to every workspace's built-in
Owner and Admin roles (`tenant_role_permissions`, module `access`).

### Quotations — migration `000090`, ADR-0050 (paket WO Tahap 5)

A **Penawaran** has the same shape as a project (ADR-0048): events with sessions, Vendor & venue,
lines; plus payment terms, terms & conditions, frozen revisions and an activity log. Every table
carries `tenant_id` and every key leads with it. Kind, venue, vendor, category, package, staff and
project ids are primitive (`FK*`); SQL foreign keys stay among the `quotation_*` tables.

#### `quotations`
| Column | Type | Notes |
|---|---|---|
| id | BIGINT UNSIGNED PK | |
| tenant_id | BIGINT UNSIGNED NOT NULL | `FK*` → `platform.tenants.id` |
| number | VARCHAR(24) NOT NULL | `KODE-PNW-YYMMDD-NNN`, frozen at creation. `UNIQUE uq_quotations_tenant_number (tenant_id, number)` |
| doc_date | DATE NOT NULL | the WIB day in the number |
| couple_names | VARCHAR(150) NOT NULL | |
| contact_whatsapp | VARCHAR(30) NOT NULL DEFAULT '' | optional |
| pic_staff_id | BIGINT UNSIGNED NOT NULL | `FK*` → `staff.staff_members.id`; the `assigned` scope matches it |
| valid_until | DATE NOT NULL | valid through the end of this WIB day; default today + the workspace's validity days (14) |
| notes | TEXT NULL | internal |
| status | ENUM('Draft','Accepted','Rejected','Superseded') | `Kedaluwarsa` is computed (a Draft past `valid_until`), never stored |
| alternative_of_id | BIGINT UNSIGNED NULL | the group's first quotation (NULL for that first one) |
| superseded_by_id | BIGINT UNSIGNED NULL | the copy or accepted alternative that replaced it |
| accepted_revision_id | BIGINT UNSIGNED NULL | the revision the couple accepted |
| accepted_on | DATE NULL | |
| accepted_channel | ENUM('WhatsApp','InPerson','Email','Other') NULL | |
| accepted_attachment_key / _name / _type | VARCHAR NULL | optional evidence of acceptance in the workspace's storage path |
| rejected_reason | VARCHAR(500) NULL | |
| rejected_at | DATETIME NULL | |
| converted_project_id | BIGINT UNSIGNED NULL | `FK*` → `projects.projects.id`; a cache written through `offers/contracts.MarkConverted` |
| converted_at | DATETIME NULL | |
| project_deleted_at | DATETIME NULL | set by `MarkProjectDeleted` ("Proyek sudah dihapus") |
| is_pkp | BOOLEAN NOT NULL | copied from the workspace at creation |
| terms_text | TEXT NOT NULL | Syarat & ketentuan, seeded from the workspace default |
| revision_count | INT UNSIGNED NOT NULL DEFAULT 0 | highest revision n (0 = none) |
| contract_total | BIGINT UNSIGNED NOT NULL | working-copy total cache for the list |
| main_event_date | DATE NULL | earliest event date cache for the list |
| created_by_staff_id | BIGINT UNSIGNED NULL | |
| deleted_at | DATETIME NULL | |
| created_at | TIMESTAMP(6) | |
| updated_at | TIMESTAMP(6) | header `If-Match` version (written explicitly, no `ON UPDATE`) |
| lines_updated_at | TIMESTAMP(6) | lines `If-Match` version |
| terms_updated_at | TIMESTAMP(6) | terms and terms-text `If-Match` version |
| changed_at | TIMESTAMP(6) | last change of anything; the list's default order |

Keys `(tenant_id, status, valid_until)`, `(tenant_id, alternative_of_id)`, `(tenant_id, pic_staff_id)`,
`(tenant_id, changed_at)`, `(tenant_id, converted_project_id)`.

#### `quotation_counters`
`PK (tenant_id, doc_date)`, `last_seq INT UNSIGNED` — the per-workspace, per-WIB-day sequence of the
number, taken under a row lock in the creating transaction (at most 999 a day).

#### `quotation_suppliers`, `quotation_events`, `quotation_event_sessions`, `quotation_lines`
Same columns and meaning as `project_suppliers`, `project_events`, `project_event_sessions` and
`project_lines` (with `quotation_id` instead of `project_id`): `supplier_kind ENUM('vendor','venue')`
+ `supplier_id` (`FK*` → `vendors.vendors` / `vendors.venues`), copied name, category and capacities,
`paid_by ENUM('WO','Client')`, `show_name`, `agreed_cost` (only with `finance.margin.view`);
events with `event_kind_id` (`FK*` → `event_kinds.id`), copied `kind_name`, an optional `event_date`
(a quotation may be an estimate), `venue_supplier_id` (`FK` → `quotation_suppliers`, `SET NULL`),
`layout ENUM('Standing','Seated')`; sessions with `guests`; lines `kind ENUM('Package','Component',
'ExtraGuests','Item','Discount')` with `parent_line_id`, `package_id` (`FK*`, `KEY (tenant_id,
package_id)` backs "a package in a quotation can only be deactivated"), the copied `package_rule`
JSON and `copied_at`, `event_id` and `supplier_ref` (`FK` within the quotation, `SET NULL`). All
cascade from `quotations`.

#### `quotation_terms`
| Column | Type | Notes |
|---|---|---|
| label | VARCHAR(60) NOT NULL | "DP", "Termin 1", "Pelunasan" |
| percent / amount | INT / BIGINT UNSIGNED NULL | exactly one is set |
| due_basis | ENUM('Accepted','Event') | due relative to the acceptance day or an event's date |
| due_offset_days | INT NOT NULL | days after acceptance (0–365) or before/after the event (−730…365) |
| due_event_id | BIGINT UNSIGNED NULL | `FK` → `quotation_events` `SET NULL`; NULL = the earliest event |
| sort_order | INT | |

Defaults for a new quotation: DP 30% on acceptance, Termin 1 40% at H-60, Pelunasan 30% at H-7.

#### `quotation_revisions`
| Column | Type | Notes |
|---|---|---|
| n | INT UNSIGNED NOT NULL | Revisi n; `UNIQUE (tenant_id, quotation_id, n)` |
| snapshot | JSON NOT NULL | everything the PDF needs, frozen (`schemaVersion`, number, issued at, validity, PIC name, letterhead identity incl. logo key, content with positions instead of ids) |
| content_hash | CHAR(64) NOT NULL | sha256 of the content only; `UNIQUE (tenant_id, quotation_id, content_hash)` — identical content never adds a revision |
| contract_total | BIGINT UNSIGNED | |
| created_by_staff_id | BIGINT UNSIGNED NULL | |
| created_at | TIMESTAMP(6) | |

#### `quotation_activity`
`type VARCHAR(40)` (`created`, `header_changed`, `events_changed`, `suppliers_changed`, `lines_changed`,
`terms_changed`, `revision_created`, `shared`, `extended`, `accepted`, `unaccepted`, `rejected`,
`reopened`, `superseded`, `copied`, `alternative_added`, `converted`, `project_deleted`,
`attachment_added`), `actor_staff_id`, `summary VARCHAR(500)`, `created_at`;
`KEY (tenant_id, quotation_id, created_at)`.

Migration `000090` also grants `quotation.view` and `quotation.manage` to every workspace's built-in
Owner, Admin and Sales roles.

---

## Module `projects` (largest module) — **implemented, Fase 4**

### `projects`
| Column | Type | Notes |
|---|---|---|
| id | BIGINT UNSIGNED PK | |
| tenant_id | BIGINT UNSIGNED NOT NULL | |
| name | VARCHAR(150) | |
| bride_name, groom_name | VARCHAR(150) | |
| event_date | DATE | the date of the project's **main event** (`project_events.is_main`), Hari H of the schedule. Written only by the event service, in the same transaction as the event that moves it (create, edit, set-main, delete of the main event), together with the steps that follow H-n |
| prep_start_date | DATE | |
| is_pkp | BOOLEAN NOT NULL DEFAULT FALSE | migration `000088` — **copied** from the workspace (`tenant_business_details.is_pkp`, through the `WorkspaceProfile` port) when the project is created (a project made from a quotation copies the accepted revision's `is_pkp` instead); later changes to the workspace never touch an existing project. PPN 11% applies to this project's lines when set |
| contract_subtotal | BIGINT NOT NULL DEFAULT 0 | migration `000088` — **cache** of the live `project_lines`: Σ Package + ExtraGuests + Item − Σ Discount (never negative; a discount above the gross is refused, 422 `discount_exceeds_subtotal`). The WO's own price to the client — distinct from what it agreed to pay its vendors and venues (`project_suppliers.agreed_cost`); margin = this subtotal minus those costs, computed on read for `finance.margin.view`, never stored |
| contract_tax | BIGINT NOT NULL DEFAULT 0 | migration `000088` — cache: `contract_subtotal × 11 / 100` when `is_pkp`, else 0 (offers `LineTotals`) |
| contract_total | BIGINT NOT NULL DEFAULT 0 | migration `000088` — cache: subtotal + PPN, the **contract value** every reader uses (ledger, schedule, overpayment, dashboard, finance, portal, exports). The three columns are rewritten by one `UPDATE … SET … = (SELECT SUM …)` in the same transaction as every change of a line, an event's guests, a line or project soft delete/restore, and a duplicate; nothing else writes them |
| lines_version | TIMESTAMP(6) NOT NULL | migration `000088` — the Rincian's own version (If-Match of every line write, ADR-0036), separate from `updated_at` so editing lines never makes an open project form stale. Bumped by every line write; a guest-driven recompute of the extra-guest charge keeps it |
| status | ENUM('Draft','Preparation','Ready','Completed','Cancelled') | |
| pic_staff_id | BIGINT UNSIGNED | `FK*` → `staff.staff_members.id` — reassignable only by Owner/Admin, never by the "Staff" (Wedding Planner) role itself, see ADR-0017 |
| description | TEXT NULL | |
| quotation_id | BIGINT UNSIGNED NULL | migration `000091` (paket WO Tahap 5, PLAN §5.4) — `FK*` → `offers.quotations.id` (primitive id, read through `offers/contracts.Quotations`): the accepted quotation this project was made from ("Jadikan proyek"); `NULL` for a project made by hand. `UNIQUE uq_projects_tenant_quotation (tenant_id, quotation_id)` makes a quotation one project at most — two concurrent conversions meet this key, the loser rolls back and answers the winner's project. A duplicate never copies it; a purge tells `offers` (`MarkProjectDeleted`), after which the quotation may be converted again |
| quotation_number, quotation_revision_n | VARCHAR(24) NULL, INT UNSIGNED NULL | migration `000091` — **cache** of the quotation's number and the accepted revision's n at conversion, so the project page shows "Dari penawaran MO-PNW-… Revisi n" without a call to `offers`. Written only at conversion |
| is_archived | BOOLEAN NOT NULL DEFAULT FALSE | orthogonal to `status` — see ADR-0013 |
| cover_photo_path | VARCHAR(500) NULL | migration `000077` (ADR-0047) — object-storage key of the Client Portal hero photo (`…/{tenant}/{project}/cover/<unix>-<uuid>.jpg|png`, always re-encoded by the server). `NULL` = no cover. Written only by `PUT/DELETE /projects/{id}/cover`, which keep `updated_at` as it is; the old object is deleted after a replace, and on project delete |
| cover_updated_at | TIMESTAMP NULL | migration `000077` — when the cover last changed: the `?v=` of `coverUrl`. `NULL` with the cover |
| created_at, updated_at | TIMESTAMP | |

### `project_lines` — migration `000088` (Rincian, paket WO Tahap 3)
| Column | Type | Notes |
|---|---|---|
| id | BIGINT UNSIGNED PK | |
| tenant_id | BIGINT UNSIGNED NOT NULL | every read and write filters by it |
| project_id | BIGINT UNSIGNED `FK` → projects.id | |
| kind | ENUM('Package','Component','ExtraGuests','Item','Discount') | a Package brings its Components (amount 0) and, when its rule charges extra guests and the decisive event has more than the included guests, one ExtraGuests line, always computed by the server |
| parent_line_id | BIGINT UNSIGNED NULL `FK` → project_lines.id `ON DELETE CASCADE` | Component and ExtraGuests point at their Package; deleting a Package takes its children with it |
| package_rule | JSON NULL | Package only: the **copy** of the catalog package's rule when it was added (`offers` `PackageCopy`: id, name, price, event kind ids, included guests, step, step price). A later catalog change never changes the project |
| description | VARCHAR(255) | Uraian, required |
| category_id | BIGINT UNSIGNED NULL | `FK*` → `vendors.vendor_categories.id` (primitive id, checked through the references port) |
| event_id | BIGINT UNSIGNED NULL `FK` → project_events.id `ON DELETE SET NULL` | the event a line belongs to; must be this project's |
| quantity, unit, unit_price | INT UNSIGNED NULL, VARCHAR(30), BIGINT UNSIGNED NULL | Item only: `amount = quantity × unit_price`; a bonus (`is_bonus`) has unit price 0 |
| amount | BIGINT UNSIGNED NOT NULL DEFAULT 0 | always ≥ 0; the kind says whether it adds or (Discount) subtracts. Component = 0 |
| is_bonus | BOOLEAN | Item only |
| supplier_ref | BIGINT UNSIGNED NULL `FK` → project_suppliers.id `ON DELETE SET NULL` | the project's own WO-paid, live supplier that provides the line; never shown in the portal |
| sort_order | INT | screen order; a package's children follow it |
| deleted_at | DATETIME NULL | soft delete (ADR-0034): restorable for 24 h, then purged |
| created_at, updated_at | TIMESTAMP, TIMESTAMP(6) | |

Keys `idx_project_lines_project (tenant_id, project_id, deleted_at)`, `idx_project_lines_parent (tenant_id, parent_line_id)`,
`idx_project_lines_deleted (deleted_at)`. Every change (add, edit, delete, restore, reorder, replace, guest change)
writes an `activity_log` entry with the old and new value (`line_added`, `line_updated`, `line_deleted`).

### `project_milestones`
| Column | Type | Notes |
|---|---|---|
| id | BIGINT UNSIGNED PK | |
| project_id | BIGINT UNSIGNED `FK` → projects.id | |
| sort_order | SMALLINT UNSIGNED | |
| name | VARCHAR(150) | |
| status | ENUM('Pending','In Progress','Completed','Cancelled') | see `MilestoneStatus` |
| target_date | DATE | |
| completed_date | DATE NULL | |
| days_before_event | INT NULL (signed) | migration `000075` (ADR-0042, EP-06): the Jadwal bawaan offset the step's `target_date` follows (`event_date` minus this many days; pulled to `prep_start_date` only when earlier than it and the booking is on or before the event). `NULL` = fixed date (added by hand, or target set by hand). Re-dated in the same transaction when the project's `event_date`/`prep_start_date` changes, unless the step is Completed/Cancelled. `completed_date` is stamped/cleared by the server when only the status changes |
| for_client | BOOLEAN NOT NULL DEFAULT FALSE | migration `000079` (ADR-0047) — a task for the couple: listed in the portal home `clientTasks` and completable by a client with `portal.tasks.complete`. Copied by project duplicate |
| completed_by_client_id | BIGINT UNSIGNED NULL | migration `000079` — the client who marked it done from the portal; primitive id into `clients`, **no FK**. Cleared when staff change the step's status |

Displayed in the UI as "Timeline" (Project Timeline), not "Milestone" — a display-text-only rename;
this table, its Go domain type (`ProjectMilestone`), and every internal identifier keep the
`milestone` name. A live tenant's `evidence.related_kind = 'vendorMilestone'` data made a
schema-level rename too risky for what was only ever meant to be a display-only ask — same
reasoning behind every other display-only rename in this codebase (e.g. "Tanggal Booking," see
`DOMAIN_GLOSSARY.md`).

### `project_milestone_templates`
Per-tenant configurable checklist auto-seeded into a new project's own `project_milestones` at
creation time ("Timeline Default", Pengaturan menu, Owner-only) — replaces what used to be a single
hardcoded 6-item list shared by every tenant.

| Column | Type | Notes |
|---|---|---|
| id | BIGINT UNSIGNED PK | |
| tenant_id | BIGINT UNSIGNED NOT NULL | |
| sort_order | INT | |
| name | VARCHAR(255) | |
| days_before_event | INT (signed) | target date = project's `event_date` minus this many days. **Can be negative** ("Setelah Acara"/H+ — target = `event_date` **plus** `abs(days_before_event)` days, e.g. "Pengembalian Kotak Seserahan" 3 days after the event); positive values (H-) are pulled to `prep_start_date` for short-notice projects, only when the booking is on or before the event (ADR-0042) — Hari H (0) is always the event day, and a project recorded after its event keeps the real offsets. The offset is copied into `project_milestones.days_before_event`. Frontend represents this as a Sebelum Acara/Setelah Acara toggle + a non-negative day count, converting to/from the signed column value only at the form boundary — the API body itself is unchanged, still one signed integer |
| created_at, updated_at | TIMESTAMP | |

No `is_active` column (unlike `vendor_categories`/`vendors`/`venues`) — items are hard-deleted, one
of five deliberate hard-delete exceptions in this codebase (see
[`DATABASE_GUIDE.md`](../knowledge/DATABASE_GUIDE.md) and ADR-0018): once copied into a project's own
`project_milestones` row, a template row is never referenced again, so there's nothing to orphan.
Migration `000024` also backfilled every already-registered tenant with the previous hardcoded
6-item list (guarded by `WHERE NOT EXISTS` against a re-run); a brand-new tenant gets the same
6-item seed via `platform`'s tenant-registration flow instead (`SeedDefaultMilestoneTemplate`,
mirroring `vendors.SeedDefaultCategories`).

### `project_events` — migration `000087` (ADR-0048, paket WO Tahap 2)
| Column | Type | Notes |
|---|---|---|
| id | BIGINT UNSIGNED PK | |
| tenant_id | BIGINT UNSIGNED NOT NULL | |
| project_id | BIGINT UNSIGNED `FK` → projects.id | |
| event_kind_id | BIGINT UNSIGNED | `FK*` → `offers.event_kinds.id` (primitive id, checked through the `offers` contract: the kind must exist and be active when picked) |
| kind_name | VARCHAR(60) | the kind's name, **copied** when picked: renaming the kind in Katalog does not change a project |
| event_date | DATE NOT NULL | |
| start_time, end_time | CHAR(5) NULL | `HH:MM`; the end after the start when both are set |
| venue_supplier_id | BIGINT UNSIGNED NULL `FK` → project_suppliers.id | the event's venue, a `project_suppliers` row of kind `venue` (same module). `NULL` = no venue yet. Events at the same directory venue share one supplier row |
| layout | ENUM('Standing','Seated') DEFAULT 'Standing' | Tata letak Berdiri/Duduk: which capacity of the venue the guests are checked against |
| is_main | BOOLEAN NOT NULL DEFAULT FALSE | Acara utama. Exactly one per project, kept by the service (not by a constraint); its `event_date` is `projects.event_date` |
| sort_order | INT NOT NULL DEFAULT 0 | order among events of the **same date**; the list itself is ordered by `event_date`, then `sort_order`, then `id` |
| created_at | TIMESTAMP | |
| updated_at | TIMESTAMP(6) | the row's version (If-Match, ADR-0036) |

Keys `idx_project_events_project (tenant_id, project_id)`, `idx_project_events_date (tenant_id, event_date)`,
`idx_project_events_kind (tenant_id, event_kind_id)`. A project always has at least one event (the service refuses
deleting the last one). An event is deleted for good, with its sessions; there is no soft delete.

### `project_event_sessions` — migration `000087`
| Column | Type | Notes |
|---|---|---|
| id | BIGINT UNSIGNED PK | |
| tenant_id | BIGINT UNSIGNED NOT NULL | |
| event_id | BIGINT UNSIGNED `FK` → project_events.id `ON DELETE CASCADE` | |
| label | VARCHAR(60) | "Sesi 1", "Sesi 2" … when left empty |
| start_time, end_time | CHAR(5) NULL | `HH:MM` |
| guests | INT UNSIGNED NOT NULL | 1 … 100,000; checked against the venue capacity for the event's layout |
| sort_order | INT NOT NULL DEFAULT 0 | |

Key `idx_project_event_sessions_event (tenant_id, event_id)`. An event has 1–10 sessions; the whole set is
rewritten when the event is saved.

### `project_suppliers` — migration `000087` ("Vendor & venue" of a project)
| Column | Type | Notes |
|---|---|---|
| id | BIGINT UNSIGNED PK | |
| tenant_id | BIGINT UNSIGNED NOT NULL | |
| project_id | BIGINT UNSIGNED `FK` → projects.id | |
| supplier_kind | ENUM('vendor','venue') | a vendor is added on the Vendor & venue tab; a venue only ever through an event |
| supplier_id | BIGINT UNSIGNED | `FK*` → `vendors.vendors.id` or `vendors.venues.id` by kind (primitive id, resolved through `vendors.Contracts`) |
| supplier_name | VARCHAR(150) | **copied** when chosen: the directory row may be renamed or deleted later |
| category_id | BIGINT UNSIGNED NULL | `FK*` → `vendors.vendor_categories.id`, copied from the vendor; `NULL` for a venue |
| paid_by | ENUM('WO','Client') DEFAULT 'WO' | Dibayar oleh. Locked once the supplier has a payment or a file (service, 422 `paid_by_locked`). Switching to `Client` clears `agreed_cost` and `show_name` |
| show_name | BOOLEAN DEFAULT FALSE | Tampilkan nama ke klien — only meaningful for a vendor paid by the WO; a venue and a supplier the client pays are always named in the portal (the service keeps it `FALSE` for them) |
| capacity_standing, capacity_seated | INT UNSIGNED NULL | venue only: **copied** from the directory when chosen, refreshed only by "Perbarui kapasitas" (`POST …/refresh-capacity`). The event capacity check reads these copies (the directory's numbers only for a venue not yet in the project) |
| agreed_cost | BIGINT UNSIGNED NULL | Biaya rekanan: what the WO agreed to pay. Only for `paid_by = 'WO'`, optional; written and read only with `finance.margin.view`. Not below what the WO already paid it (service). `NULL` or `0` = not set: the WO's payments are then not limited |
| engagement_status | ENUM('Planned','Negotiation','Booked','DP Paid','In Progress','Fully Paid','Ready','Completed','Cancelled') DEFAULT 'Planned' | `EngagementStatus` |
| booking_date, due_date | DATE NULL | `due_date` = when the WO's balance is due (dashboard, Keuangan) |
| pic_staff_id | BIGINT UNSIGNED NULL | `FK*` → `staff.staff_members.id` |
| scope | VARCHAR(255) DEFAULT '' | |
| notes | TEXT NULL | |
| deleted_at | DATETIME NULL | soft delete (ADR-0034): a stamped supplier hides its `vendor_milestones`, `supplier_payments`, `vendor_issues` and every `evidence` row hanging off any of them (they carry no stamp of their own; restoring the supplier brings them back). A venue an event still uses cannot be deleted (422 `venue_in_use`) |
| created_at | TIMESTAMP | |
| updated_at | TIMESTAMP(6) | version (If-Match) |

Keys `idx_project_suppliers_project (tenant_id, project_id)`, `idx_project_suppliers_supplier (tenant_id, supplier_kind, supplier_id)`
(the directory's "Lihat Project" and delete checks), `idx_project_suppliers_deleted (deleted_at)`.

A venue supplier is created by the event service, paid by the WO, in the same transaction as the event that picks it; when
no event uses it any more and nothing hangs off it (payment, step, kendala, file) it is deleted for good, otherwise it stays,
shown as "Tidak dipakai acara mana pun". What a supplier has been paid is **computed** from `supplier_payments`
(DP + Termin + Pelunasan − Refund, per payer), never stored.

### `vendor_milestones` — recreated by migration `000087`
| Column | Type | Notes |
|---|---|---|
| id | BIGINT UNSIGNED PK | |
| tenant_id | BIGINT UNSIGNED NOT NULL | |
| project_supplier_id | BIGINT UNSIGNED `FK` → project_suppliers.id | a step of one vendor or venue of the project |
| sort_order | SMALLINT UNSIGNED | |
| name | VARCHAR(150) | |
| description | TEXT NULL | |
| status | ENUM('Not Started','In Progress','Completed','Blocked','Cancelled') DEFAULT 'Not Started' | `MilestoneStatus` |
| target_date | DATE NOT NULL | |
| completed_date | DATE NULL | stamped/cleared by the server when only the status changes (EP-06) |
| pic_staff_id | BIGINT UNSIGNED NOT NULL | `FK*` → `staff.staff_members.id` |
| notes | TEXT NULL | |
| updated_at | TIMESTAMP(6) | version (If-Match) |

Key `idx_vendor_milestones_supplier (tenant_id, project_supplier_id)`. Displayed as "Tahapan vendor" in the UI.

### `supplier_payments` — migration `000087`
| Column | Type | Notes |
|---|---|---|
| id | BIGINT UNSIGNED PK | |
| tenant_id | BIGINT UNSIGNED NOT NULL | |
| project_id | BIGINT UNSIGNED `FK` → projects.id | |
| project_supplier_id | BIGINT UNSIGNED `FK` → project_suppliers.id | not changeable after recording (delete and record again) |
| payer | ENUM('WO','Client') NOT NULL | who paid: the supplier's `paid_by` at the moment the payment is recorded, **kept on the row**. Every query filters on this column, never on the supplier's `paid_by`, so an old payment never changes meaning. `WO` = money out of the WO (Pembayaran ke rekanan, margin permission); `Client` = the client paid the supplier directly (Dibayar klien, `finance.record`) |
| type | ENUM('DP','Termin','Pelunasan','Tambahan','Refund') | `PaymentType`; `Refund` needs only a transfer proof to be complete |
| amount | BIGINT UNSIGNED | 1 … 10^15 |
| payment_date | DATE | |
| method | VARCHAR(100) | frontend dropdown `Tunai`/`Transfer Bank`/`QRIS`, not enforced at the column |
| reference_number | VARCHAR(100) DEFAULT '' | optional |
| notes | TEXT NULL | never sent to a client |
| recorded_by_staff_id | BIGINT UNSIGNED NULL | `FK*` → `staff.staff_members.id`, no FK |
| bill_term_id | BIGINT UNSIGNED NULL `FK` → supplier_bill_terms.id ON DELETE SET NULL | migration `000089`: the bill a client's proof pays (only on a portal submission; NULL on what staff record) |
| submitted_by_client_id | BIGINT UNSIGNED NULL | migration `000089`: `FK*` → `clients.clients.id`, no FK. Set = the couple sent this proof from the portal (payer `Client`); it counts as paid only once `verified_at` is set |
| verified_at / verified_by_staff_id | DATETIME NULL / BIGINT UNSIGNED NULL | migration `000089`: the WO marked the client's proof right (`FK*` → staff). Cleared when the payment's type, amount, date, method or reference changes, or its last proof file is removed (ADR-0032) |
| rejected_at / rejected_by_staff_id / rejection_reason | DATETIME NULL / BIGINT UNSIGNED NULL / VARCHAR(500) NULL | migration `000089`: the WO refused the client's proof, with the reason the couple sees. A rejected proof never counts and cannot be verified afterwards |
| deleted_at | DATETIME NULL | soft delete (ADR-0034), restorable 24 h; also hidden while its supplier is stamped |
| created_at | TIMESTAMP | |
| updated_at | TIMESTAMP(6) | version (If-Match) |

Keys `idx_supplier_payments_project (tenant_id, project_id, payer)`, `idx_supplier_payments_supplier (tenant_id, project_supplier_id)`,
`idx_supplier_payments_date (tenant_id, payment_date)`, `idx_supplier_payments_deleted (deleted_at)`,
`idx_supplier_payments_bill_term (tenant_id, bill_term_id)`, `idx_supplier_payments_pending (tenant_id, payer, verified_at, rejected_at)`.

**Counts as paid** (`domain.SupplierPayment.CountsAsPaid`, SQL `domain.SupplierPaymentCountsSQL`): a row staff recorded, or a
client's proof with `verified_at` set. A pending proof (`SupplierPaymentPendingSQL`: submitted, neither verified nor rejected)
waits in the WO's verification queue; a rejected one never counts. Payer `Client` rows are the client's own money: the WO's
cash, revenue, cash flow and payables read `payer = 'WO'` only.

### `supplier_bill_terms` — migration `000089` (Paket WO Tahap 4)
The bills of a vendor or venue the client pays directly (`project_suppliers.paid_by = 'Client'`): what it asks for and by when.
| Column | Type | Notes |
|---|---|---|
| id | BIGINT UNSIGNED PK | |
| tenant_id | BIGINT UNSIGNED NOT NULL | every query filters on it (ADR-0004) |
| project_id | BIGINT UNSIGNED `FK` → projects.id | |
| project_supplier_id | BIGINT UNSIGNED `FK` → project_suppliers.id | must be paid by the client (422 `supplier_paid_by_wo` otherwise); a supplier with a live bill cannot switch to WO-paid |
| label | VARCHAR(100) | "DP fotografer" |
| amount | BIGINT UNSIGNED | 1 … 10^15 |
| due_date | DATE | |
| sort_order | INT DEFAULT 0 | the order of one supplier's bills |
| deleted_at | DATETIME NULL | soft delete (ADR-0034), restorable 24 h; also hidden while its supplier is stamped; purged with it |
| created_at | TIMESTAMP | |
| updated_at | TIMESTAMP(6) | version (If-Match) |

Keys `idx_supplier_bill_terms_project (tenant_id, project_id)`, `idx_supplier_bill_terms_supplier (tenant_id, project_supplier_id)`,
`idx_supplier_bill_terms_due (tenant_id, due_date)`, `idx_supplier_bill_terms_deleted (deleted_at)`. What is paid of each bill is
**computed**: the supplier's counted client payments (DP + Termin + Pelunasan − Refund; Tambahan is left out) are spread over its bills,
earliest due date first (`domain.BuildSupplierBill`, the same rule as `client_payment_terms`); `pending` is the sum of the
unchecked proofs sent for that bill.

"Lengkap"/`evidenceComplete` is **computed** from `evidence` (`related_kind = 'supplierPayment'`, by `evidence.type`):
an `Invoice` and a `Transfer Proof`, or only a `Transfer Proof` for a `Refund` (`domain.IsPaymentEvidenceComplete`). What the
WO pays (DP + Termin + Pelunasan − Refund, payer `WO`) may not exceed the supplier's `agreed_cost` when one is set
(422 `payment_exceeds_contract`); payments by the client are not limited. Edit and delete follow ADR-0019; a delete is soft
(ADR-0034) and the purge removes the row with its evidence and stored objects.

### `tenant_business_details`
| Column | Type | Notes |
|---|---|---|
| tenant_id | BIGINT UNSIGNED PK `FK` → tenants.id | one row per tenant; no row = nothing set |
| address | VARCHAR(255) | '' when unset |
| bank_name | VARCHAR(60) | '' when unset |
| bank_account_number | VARCHAR(40) | '' when unset |
| bank_account_holder | VARCHAR(100) | '' when unset |
| document_code | VARCHAR(5) NOT NULL DEFAULT '' | Kode usaha: starts every quotation number (`KODE-PNW-YYMMDD-NNN`) — migration `000085`. 2–5 capital letters or digits (service). `''` = not set yet: the first read (or sign-up) fills it from the business name (initials, or the first three letters of a one-word name; fallback `WO`). Not unique across tenants |
| is_pkp | TINYINT(1) NOT NULL DEFAULT 0 | the WO is a PKP, so PPN is added to its quotations — migration `000085` |
| default_quote_validity_days | SMALLINT UNSIGNED NOT NULL DEFAULT 14 | how many days a new quotation stays valid, 1–365 (service) — migration `000085` |
| default_terms_text | TEXT NULL | default terms copied into each new quotation, max 5,000 characters (service); seeded with example terms on first read — migration `000085` |
| updated_at | TIMESTAMP | |

What the Owner shares with clients beyond name and WhatsApp (migration 000061, ADR-0029). A separate
table keeps the hot `tenants` row narrow. The bank account is all-or-nothing, enforced in the service.
Migration `000085` added the Penawaran settings (Profil usaha › Penawaran, `GET/PUT
/tenants/me/quote-settings`); they are upserted on their own, leaving the address and bank account
of the same row untouched.

### `client_payment_terms`
| Column | Type | Notes |
|---|---|---|
| id | BIGINT UNSIGNED PK | |
| project_id | BIGINT UNSIGNED `FK` → projects.id | same module, so the foreign key is allowed |
| label | VARCHAR(100) | "DP", "Termin 1", "Pelunasan"... |
| amount | BIGINT UNSIGNED | |
| due_date | DATE | |
| created_at | TIMESTAMP | |

The client's agreed payment schedule (migration 000060, ADR-0028). Whether an installment is paid,
partial, due soon or overdue is **never stored**: the API spreads what the client really paid
(`client_payments`, DP + Termin + Pelunasan − Refund) over the installments, earliest due date
first, so the status can never disagree with the payments. The whole schedule is replaced in one
transaction (`PUT`), and deleted with the project.

### `client_payments`
| Column | Type | Notes |
|---|---|---|
| id | BIGINT UNSIGNED PK | |
| project_id | BIGINT UNSIGNED `FK` → projects.id | belongs to the project itself, not any supplier (what the client pays a vendor or venue directly is a `supplier_payments` row with payer `Client`) |
| type | ENUM(...) | `PaymentType`, the same set as `supplier_payments` |
| amount | BIGINT UNSIGNED | |
| payment_date | DATE | |
| method | VARCHAR(100) | same fixed `Tunai`/`Transfer Bank`/`QRIS` frontend dropdown as `supplier_payments`, shared constant |
| reference_number | VARCHAR(100) | optional in the frontend form, same reasoning as `supplier_payments` |
| notes | TEXT NULL | |
| created_at | TIMESTAMP | |
| recorded_by_staff_id | BIGINT UNSIGNED NULL | migration 65 — staff who entered the payment; primitive id, **no FK** (cross-module); `NULL` on older rows, never sent to a client |
| submitted_by_client_id | BIGINT UNSIGNED NULL | migration `000078` (ADR-0047) — the client who sent this payment's transfer proof from the portal (then `recorded_by_staff_id` is `NULL`); primitive id into `clients`, **no FK**. Drives `source: "client"` in every payload |
| verified_at | DATETIME NULL | migration 65 — set by `POST .../verify` (whole seconds); cleared when the payment's amount/type/date/method/reference change or its last evidence file is removed (ADR-0032) |
| verified_by_staff_id | BIGINT UNSIGNED NULL | migration 65 — staff who verified; primitive id, no FK |
| deleted_at | DATETIME NULL, `KEY idx_client_payments_deleted` | migration `000068`, ADR-0034 — soft delete: a stamped row is invisible to every read (list, ledger, schedule allocation, portal home, dashboard) and so are its proof files; verification stays on the row and returns on restore. Hard-deleted by the purge after 24 h |

Money coming **in** from the client against the project's contract value (`projects.contract_total`) ("Uang Masuk dari
Client") — the opposite accounting direction from the WO's `supplier_payments`, and structurally simpler: no
invoice/proof evidence-ID columns either, since it only ever has one evidence slot (a
transfer proof) resolved via the polymorphic `evidence` table (`related_kind = 'clientPayment'`),
not a direct FK column.

Edit — see ADR-0019; delete is a soft delete since ADR-0034 (the evidence cascade of ADR-0019 now happens in the purge).

### `vendor_issues`
| Column | Type | Notes |
|---|---|---|
| id | BIGINT UNSIGNED PK | |
| project_id | BIGINT UNSIGNED `FK` → projects.id | |
| project_supplier_id | BIGINT UNSIGNED NULL `FK` → project_suppliers.id | the vendor or venue the kendala is about; `NULL` = a kendala about the event in general (migration `000087`). Key `idx_vendor_issues_supplier` |
| vendor_milestone_id | BIGINT UNSIGNED NULL `FK` → vendor_milestones.id | one step of that supplier; only with `project_supplier_id`, and the step must belong to it (service) |
| title, description | VARCHAR(255), TEXT | |
| impact | ENUM(...) | `IssueImpact` |
| found_date | DATE | |
| status | ENUM(...) | `IssueStatus` |
| resolution_plan, resolution_notes | TEXT NULL | |
| pic_staff_id | BIGINT UNSIGNED | `FK*` → `staff.staff_members.id` |
| target_resolution_date, resolved_date | DATE NULL | |
| is_client_visible | BOOLEAN NOT NULL DEFAULT TRUE | migration `000069` — staff decide whether the couple sees the kendala; **every Client Portal read filters it** (list, home counts/actions/news, progress count and condition, proof files). Existing rows are `TRUE` |

A kendala of a soft-deleted supplier is hidden with it (`project_supplier_id` points at a stamped supplier); a kendala about the event in general is never hidden that way.

### `evidence`
| Column | Type | Notes |
|---|---|---|
| id | BIGINT UNSIGNED PK | |
| project_id | BIGINT UNSIGNED `FK` → projects.id | denormalized for direct project-scoped queries |
| name | VARCHAR(255) | |
| type | ENUM(...) | `EvidenceType` |
| storage_path | VARCHAR(500) **UNIQUE** (`uq_evidence_storage_path`, migration `000067`) | see ADR-0006 — path returned by `shared/storage`, never a public URL. Unique so one stored object belongs to at most one evidence row: a signed upload (ADR-0033) names its object before the payment exists, and a second payment trying to use the same upload token fails on this index (its transaction is rolled back) |
| file_name | VARCHAR(255) | original filename |
| document_date | DATE NULL | |
| uploaded_at | TIMESTAMP | |
| description | VARCHAR(255) NULL | |
| uploaded_by_staff_id | BIGINT UNSIGNED | `FK*` → `staff.staff_members.id`; `0` for a file a client sent from the Client Portal |
| uploaded_by_client_id | BIGINT UNSIGNED NULL | migration `000080` (RT-07) — the client who sent the file from the Client Portal (a proof of a client payment they submitted); primitive id into `clients`, **no FK**; `NULL` for a staff upload. Backfilled for existing portal proofs from `client_payments.submitted_by_client_id`. Drives `uploadedBySource: "client"` and the client's name as `uploadedByName` |
| related_kind | ENUM('vendorMilestone','supplierPayment','projectSupplier','issue','clientPayment','general','projectMilestone') | what the file hangs off (migration `000087`): a supplier's step, a supplier payment, the supplier itself, a kendala, a client payment, a project step, or nothing (`'general'`, migration `000035`, ADR-0020: a project-level document such as a rundown or buku acara — powers the Client Portal's "Dokumen" tab) |
| related_id | BIGINT UNSIGNED | polymorphic — same-module id of a row of the table `related_kind` names. For `related_kind = 'general'` always the sentinel `0` (`AUTO_INCREMENT` never starts at 0) |
| is_client_visible | BOOLEAN NOT NULL DEFAULT FALSE | migration `000035`, ADR-0020. Set by staff (`PATCH …/evidence/{id} {isClientVisible}`) for `general`, `projectMilestone` and `vendorMilestone` files only; forced `FALSE` for every other kind. Safe by default. What a client may see is decided per file by `domain.PortalPolicy.Evidence`: a general document needs the flag; a step attachment needs the flag; a supplier step's file is shown when the supplier is paid by the client or the file is flagged; a supplier payment's file only for a payment by the client; a `projectSupplier` file never |

### `idempotency_keys` — migration `000066`, ADR-0033
| Column | Type | Notes |
|---|---|---|
| id | BIGINT UNSIGNED PK | |
| tenant_id | BIGINT UNSIGNED | `FK*` → `platform.tenants.id` (primitive reference, no SQL foreign key) |
| staff_id | BIGINT UNSIGNED | `FK*` → `staff.staff_members.id`; a key replays only for the person who made it |
| route | VARCHAR(255) `utf8mb4_bin` | `METHOD /api/v1/path` (so the project and which of the four calls are part of the key's scope) |
| idempotency_key | VARCHAR(128) `utf8mb4_bin` | the `Idempotency-Key` header, 8–128 of `[A-Za-z0-9_-]`, compared byte for byte |
| request_hash | CHAR(64) | SHA-256 of the canonicalised JSON body (key order and whitespace do not matter) |
| status_code | SMALLINT UNSIGNED NULL | `NULL` = the first request is still running; set when it succeeded (only 2xx answers are stored — a failed request releases its key) |
| response_body | MEDIUMTEXT NULL | the exact body replayed with `Idempotent-Replay: true` |
| created_at | DATETIME DEFAULT CURRENT_TIMESTAMP | retention is 24 hours: expired rows are deleted opportunistically by each reservation (this key's own expired row first, then a bounded batch); a row still "running" after 5 minutes is considered dead and can be re-claimed |

Keys / indexes: `UNIQUE uq_idempotency_tenant_key_route (tenant_id, idempotency_key, route)` — the arbiter between
two simultaneous requests with one key; `idx_idempotency_created (created_at)` for the cleanup. Owned by `projects`;
written only by the idempotency repository.

### `activity_log`
| Column | Type | Notes |
|---|---|---|
| id | BIGINT UNSIGNED PK | |
| project_id | BIGINT UNSIGNED NULL `FK` → projects.id | |
| type | VARCHAR(50) | `ActivityType` |
| actor_staff_id | BIGINT UNSIGNED | `FK*` → `staff.staff_members.id` |
| entity_type, entity_id, entity_label | VARCHAR | what changed |
| description | VARCHAR(500) | |
| created_at | TIMESTAMP | append-only — see ADR-0007, populated by every mutating use case in this module, never client-supplied |

---

## Module `platform` (additional, Fase 9)

### `pending_subscription_charges`
| Column | Type | Notes |
|---|---|---|
| order_ref | VARCHAR(100) PK | matches the `payment_charge_dispatch.order_ref` this charge was created under |
| tenant_id | BIGINT UNSIGNED | `FK*` → `tenants.id` |
| plan_id | BIGINT UNSIGNED | `FK*` → `billing.subscription_plans.id` |
| created_at | TIMESTAMP | |

Not a ledger — a thin "which tenant+plan was this still-unconfirmed charge for" index. Consumed
(read then deleted) by `TenantService.ApplyWebhookEvent` once the `payment` module's webhook
confirms or fails the charge. `payment` itself is never told about tenants or plans (see
MODULE_PAYMENT.md's non-goals), so this mapping has to live here, not in `payment`.

### Platform Console back office (Fase 6, migration 63, ADR-0030)

These tables belong to `platform`, so they may reference `tenants` with real foreign keys; admin and staff
ids are plain values. Migration 62 drops `tenants.custom_domain`.

#### `tenant_notes`
| Column | Type | Notes |
|---|---|---|
| id | BIGINT UNSIGNED PK | |
| tenant_id | BIGINT UNSIGNED | `FK` → `tenants.id`; indexed with `created_at` |
| admin_id, admin_name | VARCHAR(64), VARCHAR(100) | author (name kept so it survives renames) |
| body | TEXT | internal, never shown to the tenant |
| created_at | TIMESTAMP | |

#### `data_requests`
| Column | Type | Notes |
|---|---|---|
| id | BIGINT UNSIGNED PK | |
| tenant_id | BIGINT UNSIGNED | `FK` → `tenants.id` |
| kind | ENUM('export','delete') | UU PDP: copy of data / erasure |
| status | ENUM('open','done','rejected') | default `open`; indexed with `due_at` |
| requested_by | VARCHAR(200) | who asked, e.g. the Owner's email |
| note | TEXT NULL | |
| due_at | DATETIME | created + 30 days |
| created_by_admin | VARCHAR(64) | who logged it |
| completed_at, completed_by, completion_note | DATETIME NULL, VARCHAR(64) NULL, TEXT NULL | proof is mandatory when closing |
| created_at | TIMESTAMP | |

#### `announcements`
| Column | Type | Notes |
|---|---|---|
| id | BIGINT UNSIGNED PK | |
| title, body | VARCHAR(150), TEXT | |
| kind | ENUM('info','warning','maintenance') | |
| audience, audience_value | ENUM('all','plan','status'), VARCHAR(50) NULL | plan id or lifecycle status |
| starts_at, ends_at | DATETIME, DATETIME NULL | `NULL` end = until deleted; indexed together |
| created_by | VARCHAR(64) | |
| created_at, updated_at | TIMESTAMP | |

#### `subscription_policy`
One row (`id = 1`): `trial_days` 14, `expiring_soon_days` 7, `grace_days` 7, `read_only_days` 23, `updated_by`,
`updated_at`. Seeded by the migration; loaded at startup into `domain.CurrentSubscriptionPolicy()`.

#### `support_access_grants`
| Column | Type | Notes |
|---|---|---|
| id | BIGINT UNSIGNED PK | |
| tenant_id | BIGINT UNSIGNED | `FK` → `tenants.id`; indexed with `expires_at` |
| granted_by_staff_id | BIGINT UNSIGNED | the Owner who consented |
| granted_at | TIMESTAMP | |
| expires_at | DATETIME | granted + 24 hours |
| revoked_at | DATETIME NULL | |

#### `support_sessions`
| Column | Type | Notes |
|---|---|---|
| id | BIGINT UNSIGNED PK | the `sid` claim of the support token |
| tenant_id | BIGINT UNSIGNED | `FK` → `tenants.id`; indexed with `started_at` |
| admin_id, admin_name | VARCHAR(64), VARCHAR(100) | the `imp` claim, and the name shown to the Owner |
| reason | VARCHAR(255) | 10–255 characters, shown to the Owner |
| started_at | TIMESTAMP | |
| expires_at | DATETIME | started + 60 minutes |
| ended_at | DATETIME NULL | |

### Composite indexes (Fase 7, migration 64)

| Table | Added | Replaces | Serves |
|---|---|---|---|
| `projects` | `idx_projects_tenant_archived_event (tenant_id, is_archived, event_date, id)` | `idx_projects_tenant` | the paginated project list and its count |
| `subscription_transactions` | `idx_subscription_transactions_created (created_at, id)`, `_status_created (status, created_at, id)`, `_tenant_created (tenant_id, created_at, id)`, `_status_paid (status, paid_at)` | `idx_subscription_transactions_tenant` | the platform list (with or without a status), one tenant's list, the paid-revenue window of the metrics |
| `tenants` | `idx_tenants_joined (joined_at, id)` | | the tenant list order and the join counts of the dashboard |

---

## Module `payment` — **implemented, internal + external mode (Fase 9 + Fase 10)** — see `knowledge/MODULE_PAYMENT.md`

None of these four tables is a business ledger — see `MODULE_PAYMENT.md` §4. The schema needed zero
new migrations between Fase 9 and Fase 10: `payment_apps`' `kind='external'`-only columns
(`secret_hash`, `secret_encrypted`, `callback_url`) were already present from Fase 9's own
migration, just unpopulated until Fase 10's "Manajemen Aplikasi" page started writing external App
rows through them.

### `payment_gateway_config`
| Column | Type | Notes |
|---|---|---|
| id | BIGINT UNSIGNED PK | single row by application convention (id=1), not a DB constraint |
| active_provider | VARCHAR(50) NULL | `tripay` \| NULL (simulation mode — no charge can be created) |
| is_sandbox | BOOLEAN DEFAULT TRUE | |
| tripay_merchant_code | VARCHAR(100) NULL | not secret, stored plaintext |
| tripay_api_key_encrypted | TEXT NULL | AES-256-GCM, key from `PAYMENT_ENCRYPTION_KEY` |
| tripay_private_key_encrypted | TEXT NULL | AES-256-GCM, same key |
| created_at, updated_at | TIMESTAMP | |

### `payment_apps`
| Column | Type | Notes |
|---|---|---|
| id | BIGINT UNSIGNED PK | |
| app_id | VARCHAR(100) UNIQUE | `platform-billing` (`kind=internal`, auto-bootstrapped) plus one row per registered `kind=external` App, e.g. `app_a1b2c3d4e5f6` |
| name | VARCHAR(150) | |
| kind | ENUM('internal','external') | |
| secret_hash | VARCHAR(255) NULL | bcrypt — `kind='external'` only (Fase 10) |
| secret_encrypted | TEXT NULL | reversible copy, used only to sign outbound webhook relays — `kind='external'` only (Fase 10), see §7.5 |
| callback_url | VARCHAR(500) NULL | `kind='external'` only (Fase 10) |
| is_active | BOOLEAN DEFAULT TRUE | checked live on every request, never cached |
| created_at, updated_at | TIMESTAMP | |

### `payment_charge_dispatch`
| Column | Type | Notes |
|---|---|---|
| order_ref | VARCHAR(100) PK | the ref the calling App supplied when creating the charge |
| app_id | VARCHAR(100) | `FK*` → `payment_apps.app_id` — who owns this charge, for webhook routing |
| provider_ref | VARCHAR(150) NULL | the gateway's own transaction reference, for pull-based status checks |
| expires_at | TIMESTAMP NULL | the charge's own deadline, recorded at creation — NULL for rows created before this column existed |
| resolved_at | TIMESTAMP NULL | NULL until a terminal outcome (paid/expired/failed/refund) has been dispatched to the owning App — via webhook or the reconciliation sweep, see `knowledge/MODULE_PAYMENT.md` §6 step 6. Indexed with `created_at` for that sweep's query. |
| created_at | TIMESTAMP | |

Thin dispatch index only — PK uniqueness on `order_ref` doubles as idempotency (a repeat charge
attempt with the same ref fails with a DB duplicate-key error, surfaced as `409`-equivalent).
`expires_at`/`resolved_at` are the one exception to "not a ledger": completion markers, not amounts.

### `payment_webhook_events`
| Column | Type | Notes |
|---|---|---|
| id | BIGINT UNSIGNED PK | |
| provider | VARCHAR(50) | |
| event_id | VARCHAR(150) | provider-scoped id (Tripay: `{reference}:{status}`, since Tripay sends no dedicated event id) |
| received_at | TIMESTAMP | |

Unique on `(provider, event_id)` — the same gateway callback delivered twice is a no-op the second
time, never double-applied.

---

### Soft delete, restore and purge — migrations `000068`, `000069`, ADR-0034

* `deleted_at DATETIME NULL` (+ index) on `client_payments`, `supplier_payments`, `project_suppliers` (module `projects`) and `clients` (module `clients`). `NULL` = live. The application writes the stamp (not `CURRENT_TIMESTAMP`), so `restorableUntil` is exactly `deleted_at + 24 h`.
* **Every read of a live row filters it**: repositories, the batched progress sources, the dashboard queries, the portal home, the overview, `CountActive`. A deleted supplier hides its steps, payments, kendala and their evidence by joining the stamp of the supplier (the children are not stamped). `evidence` rows hang off these items by `(related_kind, related_id)` with no foreign key; every evidence read adds `NOT (related_kind = … AND related_id IN (SELECT id FROM <table> WHERE deleted_at IS NOT NULL …))` per kind, so a deleted item's files are not listed, opened or linked until it is restored. The unique `evidence.storage_path` keeps a deleted payment's upload claimed (a token cannot be reused by a new payment before the purge); a deleted client keeps its username and email reserved (its login is deactivated, not removed).
* **Purge** (`api purge`, ticker `PURGE_INTERVAL`): per module, under a MySQL advisory lock (`GET_LOCK`, `elproof_projects_purge` / `elproof_clients_purge`), `DELETE … WHERE deleted_at IS NOT NULL AND deleted_at <= now − 24 h` in batches of 200 per kind (evidence rows first, then the row; a supplier takes its vendor_issues → supplier_payments → vendor_milestones → supplier_bill_terms → itself; an expired `supplier_bill_terms` row is purged on its own too), then the stored objects (best effort, after the commit). Restore accepts `deleted_at > now − 24 h`: exactly the complement, so a row is always restorable or due for purge, never both.
* `vendor_issues.project_supplier_id` is NULLable (a kendala about the event in general) and `vendor_issues.is_client_visible BOOLEAN NOT NULL DEFAULT TRUE` (migration `000069`).
* Down migrations delete what the old shape cannot hold (soft-deleted rows with their evidence, kendala without a vendor); stored objects stay in the bucket.

### `tenant_roles` built-in descriptions — migration `000082` (FQ-08, data only)

The `access` module's `tenant_roles` (migration `000058`, ADR-0026) hold each workspace's copy of the built-in role
descriptions. Those of Admin, Wedding Planner, Sales and Keuangan used English words ("project", "client"); 000082
rewrites a row only when its `description` still equals the seeded text exactly (BINARY, matched with its
`template_key`), keeping `updated_at`. A description a WO edited is left alone. New workspaces get the Indonesian text from
`access/domain/catalog.go` Templates. Down restores the English text only where up's text is still there.

## Module `audit` — migrations `000049`, `000059`, `000070` (ADR-0035 for the last), `000081`

### `audit_log`
| Column | Type | Notes |
|---|---|---|
| id | BIGINT UNSIGNED PK | |
| occurred_at | DATETIME(3) | `KEY idx_audit_log_occurred_at` |
| actor_type, actor_id, actor_role | VARCHAR(30), VARCHAR(64), VARCHAR(50) | `KEY idx_audit_log_actor (actor_type, actor_id, occurred_at)` |
| tenant_id | BIGINT UNSIGNED NULL | migration `000059`; `KEY idx_audit_log_tenant (tenant_id, occurred_at)` |
| action | VARCHAR(120) | readable label, e.g. `POST /tenants/{id}/toggle-suspension`. **Migration `000070`:** `KEY idx_audit_log_action (action, id)` — serves the exact `?action=` filter of `GET /audit-log` and `ORDER BY action, id` (Tahap 6 sorting) without sorting the whole append-only table; down drops it |
| description | VARCHAR(255) NOT NULL DEFAULT '' | migration `000081` (RT-08) — the sentence the handler stated when the request ran (`middleware.SetAuditDescription`): a toggle's resulting state and object name ("Menonaktifkan paket Pro", "Membekukan tenant WO Contoh"). `''` for every other route and every older row; the read side then builds the sentence from method + path |
| method, path, status_code, ip, user_agent | VARCHAR(10), VARCHAR(255), SMALLINT UNSIGNED, VARCHAR(64), VARCHAR(255) | |

Append-only from the application. No other index was added for Tahap 6: the other sortable columns of the paginated lists (`projects`, `clients`, `vendors`, `venues`,
`staff_members`, `vendor_categories`) are ordered inside one workspace's rows (`tenant_id` is the leading key of their existing indexes), `remaining` and `nextDueDate` of
`GET /projects` are correlated sub-selects on the existing `client_payments (project_id)` and `client_payment_terms (project_id, due_date)` keys, and the transaction list
sorts on `created_at`/`status` (existing composite indexes) or `amount` (a scan, on the Platform Console only).

## Cross-module reference summary (no SQL FKs across this boundary)

```
staff_members.tenant_id        --*--> tenants.id            (platform)
clients.tenant_id              --*--> tenants.id            (platform)
clients.project_id             --*--> projects.id           (projects)
vendors.tenant_id              --*--> tenants.id            (platform)
venues.tenant_id               --*--> tenants.id            (platform)
projects.tenant_id             --*--> tenants.id            (platform)
projects.pic_staff_id          --*--> staff_members.id      (staff)
event_kinds.tenant_id          --*--> tenants.id            (platform)
packages.tenant_id             --*--> tenants.id            (platform)
packages.default_venue_id      --*--> venues.id             (vendors)
package_components.vendor_id   --*--> vendors.id            (vendors)
package_components.category_id --*--> vendor_categories.id  (vendors)
project_milestone_templates.tenant_id --*--> tenants.id     (platform)
project_events.event_kind_id   --*--> event_kinds.id        (offers)
project_suppliers.supplier_id  --*--> vendors.id | venues.id (vendors, by supplier_kind)
project_suppliers.category_id  --*--> vendor_categories.id  (vendors)
project_suppliers.pic_staff_id --*--> staff_members.id      (staff)
supplier_payments.recorded_by_staff_id --*--> staff_members.id (staff)
supplier_payments.submitted_by_client_id --*--> clients.id  (clients)
supplier_payments.verified_by_staff_id / rejected_by_staff_id --*--> staff_members.id (staff)
vendor_milestones.pic_staff_id --*--> staff_members.id      (staff)
vendor_issues.pic_staff_id     --*--> staff_members.id      (staff)
evidence.uploaded_by_staff_id  --*--> staff_members.id      (staff)
evidence.uploaded_by_client_id --*--> clients.id            (clients)
activity_log.actor_staff_id    --*--> staff_members.id      (staff)
idempotency_keys.tenant_id     --*--> tenants.id            (platform)
idempotency_keys.staff_id      --*--> staff_members.id      (staff)
tenants.plan_id                --*--> subscription_plans.id (billing)
subscription_transactions.tenant_id --*--> tenants.id       (platform)
credentials.principal_id       --*--> (varies by principal_type)
pending_subscription_charges.tenant_id --*--> tenants.id             (platform, same module)
pending_subscription_charges.plan_id   --*--> subscription_plans.id  (billing)
payment_charge_dispatch.app_id --*--> payment_apps.app_id  (payment, same module)
```
