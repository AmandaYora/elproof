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
| token_hash | VARCHAR(255) | sha256 of the opaque refresh token |
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
| name | VARCHAR(100) | |
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
| price_akad | BIGINT UNSIGNED NULL | vendor's own preset price for an "Akad Saja" engagement — migration `000023`. Reference/catalog price only: a specific project engagement's own `project_vendors.contract_value` is always the source of truth and may be freely negotiated away from this number (see `pricing_tier` below) |
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
siblings), not a vendor category. Exactly one venue per project, referenced by `projects.venue_id`
below (never a `project_venues` engagement table — no per-project negotiated price/DP/paid-amount
tracking for venues, unlike vendor engagements).

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
| rental_price | BIGINT UNSIGNED NULL | mandatory on the create form; nullable at the DB for the same migration-safety reason as `city` — the UI flags "harga sewa belum diisi" rather than blocking attachment to a project |
| charge | BIGINT UNSIGNED NULL | separate fixed charge, distinct from `rental_price` |
| capacity | INT NULL | |
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
old `project_vendors` history (fuzzy-matching risk — see ADR-0016).

Composite index `(tenant_id, city)` — migration `000033` — same shape as `vendors` above, backs the
Venue list page's Kota filter.

Both `vendors` and `venues` (and `staff`, and `vendor_categories`) can now also be permanently hard-
deleted (Owner-only, informed-consent — ADR-0018) — no schema change for this, since every
cross-module reference to these rows was already a primitive, un-enforced ID.

---

## Module `projects` (largest module) — **implemented, Fase 4**

### `projects`
| Column | Type | Notes |
|---|---|---|
| id | BIGINT UNSIGNED PK | |
| tenant_id | BIGINT UNSIGNED NOT NULL | |
| name | VARCHAR(150) | |
| bride_name, groom_name | VARCHAR(150) | |
| event_date, prep_start_date | DATE | |
| venue | VARCHAR(255) | free-text location — kept as a fallback for projects that never get a structured `venue_id` below (ADR-0016); not touched by attaching one |
| venue_id | BIGINT UNSIGNED NULL | `FK*` → `vendors.venues.id` (ADR-0016) — at most one venue per project; resolved via `vendors.Contracts` (`VenueResolver`), never a SQL join. `NULL` means no structured venue attached yet |
| venue_rental_price, venue_charge | BIGINT UNSIGNED NULL | per-project cost **snapshot**, captured (and freely editable) the moment `venue_id` is attached/changed — never a live join against `venues.rental_price`/`charge`. Both `NULL` whenever `venue_id` is `NULL`; force-cleared to `NULL` on detach regardless of what a request sends. Exists so a completed project's recorded Margin can never drift just because venue master data changed later (or the venue is hard-deleted — ADR-0018) |
| package_name | VARCHAR(150) | |
| contract_value | BIGINT UNSIGNED | the WO's own price quoted to the client for the whole event — a distinct figure from any vendor's or the venue's own cost (see `project_vendors.contract_value` and this table's own `venue_rental_price`/`venue_charge`); the frontend derives a "Margin/Keuntungan" figure as this value minus those costs, computed client-side, never stored |
| status | ENUM('Draft','Preparation','Ready','Completed','Cancelled') | |
| pic_staff_id | BIGINT UNSIGNED | `FK*` → `staff.staff_members.id` — reassignable only by Owner/Admin, never by the "Staff" (Wedding Planner) role itself, see ADR-0017 |
| description | TEXT NULL | |
| is_archived | BOOLEAN NOT NULL DEFAULT FALSE | orthogonal to `status` — see ADR-0013 |
| created_at, updated_at | TIMESTAMP | |

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
| days_before_event | INT (signed) | target date = project's `event_date` minus this many days. **Can be negative** ("Setelah Acara"/H+ — target = `event_date` **plus** `abs(days_before_event)` days, e.g. "Pengembalian Kotak Seserahan" 3 days after the event); positive values (H-) are clamped to `prep_start_date` for short-notice projects — a negative value's target is always after `event_date`, which is always ≥ `prep_start_date`, so the clamp never applies to H+ rows. Frontend represents this as a Sebelum Acara/Setelah Acara toggle + a non-negative day count, converting to/from the signed column value only at the form boundary — the API body itself is unchanged, still one signed integer |
| created_at, updated_at | TIMESTAMP | |

No `is_active` column (unlike `vendor_categories`/`vendors`/`venues`) — items are hard-deleted, one
of five deliberate hard-delete exceptions in this codebase (see
[`DATABASE_GUIDE.md`](../knowledge/DATABASE_GUIDE.md) and ADR-0018): once copied into a project's own
`project_milestones` row, a template row is never referenced again, so there's nothing to orphan.
Migration `000024` also backfilled every already-registered tenant with the previous hardcoded
6-item list (guarded by `WHERE NOT EXISTS` against a re-run); a brand-new tenant gets the same
6-item seed via `platform`'s tenant-registration flow instead (`SeedDefaultMilestoneTemplate`,
mirroring `vendors.SeedDefaultCategories`).

### `project_vendors`
| Column | Type | Notes |
|---|---|---|
| id | BIGINT UNSIGNED PK | |
| project_id | BIGINT UNSIGNED `FK` → projects.id | |
| vendor_id | BIGINT UNSIGNED | `FK*` → `vendors.vendors.id` |
| category_id | BIGINT UNSIGNED | `FK*` → `vendors.vendor_categories.id` (denormalized at engagement time) |
| scope | VARCHAR(255) | |
| contract_value | BIGINT UNSIGNED | the actual, possibly negotiated, amount for this specific engagement — never re-derived from `pricing_tier`/the vendor's own preset prices after the fact |
| pricing_tier | VARCHAR(20) NOT NULL DEFAULT 'Akad' | `Akad` \| `AkadResepsi` — which of the vendor's own two preset prices (`vendors.price_akad`/`price_akad_resepsi`) this engagement's `contract_value` was auto-filled from when created/last edited; purely an informational label, not backend-validated against the two-value set (same lax convention as `engagement_status`) |
| engagement_status | ENUM(...) | see `EngagementStatus` |
| booking_date, event_date, due_date | DATE NULL | |
| dp_amount | BIGINT UNSIGNED | plain reference figure ("what DP was agreed at booking") — never summed into any total |
| pic_staff_id | BIGINT UNSIGNED | `FK*` → `staff.staff_members.id` |
| notes | TEXT NULL | |
| deleted_at | DATETIME NULL, `KEY idx_project_vendors_deleted` | migration `000068`, ADR-0034 — soft delete. A stamped engagement hides its `vendor_milestones`, `vendor_payments`, `vendor_issues` and every `evidence` row hanging off any of them (the children carry no stamp of their own: visibility is decided by the engagement, so restoring it brings everything back). The purge removes the engagement and all its children together |

No `paid_amount` column (removed, migration `000028`) — "Total Sudah Dibayar" is **computed**, summing
this engagement's own `vendor_payments` rows (`Refund` netted as a subtraction) in the presentation
layer, never a stored/manually-typed value. This was a real bug: the old manual field and the actual
`vendor_payments` ledger were two independently-maintained numbers that could silently contradict each
other on the same tab.

### `vendor_milestones`
| Column | Type | Notes |
|---|---|---|
| id | BIGINT UNSIGNED PK | |
| project_vendor_id | BIGINT UNSIGNED `FK` → project_vendors.id | |
| sort_order | SMALLINT UNSIGNED | |
| name, description | VARCHAR(255), TEXT NULL | |
| status | ENUM(...) | `MilestoneStatus` |
| target_date, completed_date | DATE NULL | |
| pic_staff_id | BIGINT UNSIGNED | `FK*` → `staff.staff_members.id` |
| notes | TEXT NULL | |

Displayed as "Timeline Vendor" in the UI — same display-only rename as `project_milestones` above.

### `vendor_payments`
| Column | Type | Notes |
|---|---|---|
| id | BIGINT UNSIGNED PK | |
| project_id | BIGINT UNSIGNED `FK` → projects.id | |
| project_vendor_id | BIGINT UNSIGNED `FK` → project_vendors.id | |
| type | ENUM(...) | `PaymentType`, includes `Refund` (special evidence-completeness rule) |
| amount | BIGINT UNSIGNED | |
| payment_date | DATE | |
| method | VARCHAR(100) | frontend constrains this to a fixed `Tunai`/`Transfer Bank`/`QRIS` dropdown (`PAYMENT_METHOD_OPTIONS`, shared with `client_payments`); not enforced at this column, same lax convention as `pricing_tier`/`engagement_status` |
| reference_number | VARCHAR(100) | optional in the frontend form (empty string allowed) — a `Tunai` payment has no transfer reference number |
| notes | TEXT NULL | |
| deleted_at | DATETIME NULL, `KEY idx_vendor_payments_deleted` | migration `000068`, ADR-0034 — soft delete. A payment is also hidden while its *engagement* is stamped (no stamp of its own then; restoring the engagement brings it back) |

No `invoice_evidence_id`/`proof_evidence_id` columns (removed, migration `000029`) — confirmed via a
full-repo search that no code path had ever written a non-null value to either since this table's
first migration. "Lengkap"/`evidenceComplete` is **computed**, cross-referencing the polymorphic
`evidence` table (`related_kind = 'payment'`, distinguished by `evidence.type` — `Invoice` vs
`Transfer Proof`) via `domain.PaymentEvidenceStatus`/`IsPaymentEvidenceComplete` — the same
presentation-layer derivation pattern already used for `client_payments`' `evidenceComplete` and the
vendor paid-amount fix. A `Refund` only needs a `Transfer Proof`; every other type needs both an
`Invoice` and a `Transfer Proof` attached — the same rule the old (dead) columns encoded, now finally
collectible since "Tambah Pembayaran" attaches both file slots in the same submit that records the
payment.

Rows can now be edited (full-field overwrite, no role gate) and deleted (`finance.payment.delete`) — see
ADR-0019. **Since ADR-0034 a delete is soft** (`deleted_at`, restorable for 24 h; the evidence is hidden, not removed, until the purge hard-deletes row, evidence rows and stored objects together). The description that follows is the earlier hard delete. Deleting a row cascades to its own `evidence` rows (deleted first, then the payment row —
deliberately reversed order, see ADR-0019) and their S3 objects, best-effort; no schema change was
needed for this.

### `tenant_business_details`
| Column | Type | Notes |
|---|---|---|
| tenant_id | BIGINT UNSIGNED PK `FK` → tenants.id | one row per tenant; no row = nothing set |
| address | VARCHAR(255) | '' when unset |
| bank_name | VARCHAR(60) | '' when unset |
| bank_account_number | VARCHAR(40) | '' when unset |
| bank_account_holder | VARCHAR(100) | '' when unset |
| updated_at | TIMESTAMP | |

What the Owner shares with clients beyond name and WhatsApp (migration 000061, ADR-0029). A separate
table keeps the hot `tenants` row narrow. The bank account is all-or-nothing, enforced in the service.

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
| project_id | BIGINT UNSIGNED `FK` → projects.id | no `project_vendor_id` — belongs to the project itself, not any vendor |
| type | ENUM(...) | `PaymentType`, reused verbatim from `vendor_payments` |
| amount | BIGINT UNSIGNED | |
| payment_date | DATE | |
| method | VARCHAR(100) | same fixed `Tunai`/`Transfer Bank`/`QRIS` frontend dropdown as `vendor_payments`, shared constant |
| reference_number | VARCHAR(100) | optional in the frontend form, same reasoning as `vendor_payments` |
| notes | TEXT NULL | |
| created_at | TIMESTAMP | |
| recorded_by_staff_id | BIGINT UNSIGNED NULL | migration 65 — staff who entered the payment; primitive id, **no FK** (cross-module); `NULL` on older rows, never sent to a client |
| verified_at | DATETIME NULL | migration 65 — set by `POST .../verify` (whole seconds); cleared when the payment's amount/type/date/method/reference change or its last evidence file is removed (ADR-0032) |
| verified_by_staff_id | BIGINT UNSIGNED NULL | migration 65 — staff who verified; primitive id, no FK |
| deleted_at | DATETIME NULL, `KEY idx_client_payments_deleted` | migration `000068`, ADR-0034 — soft delete: a stamped row is invisible to every read (list, ledger, schedule allocation, portal home, dashboard) and so are its proof files; verification stays on the row and returns on restore. Hard-deleted by the purge after 24 h |

Money coming **in** from the client against the project's own `contract_value` ("Uang Masuk dari
Client") — the opposite accounting direction from `vendor_payments`, and structurally simpler: no
invoice/proof evidence-ID columns either, since it only ever has one evidence slot (a
transfer proof) resolved via the polymorphic `evidence` table (`related_kind = 'clientPayment'`),
not a direct FK column.

Edit — same as `vendor_payments` above, see ADR-0019; delete is a soft delete since ADR-0034 (the evidence cascade of ADR-0019 now happens in the purge).

### `venue_payments`
| Column | Type | Notes |
|---|---|---|
| id | BIGINT UNSIGNED PK | independent `AUTO_INCREMENT` sequence from `vendor_payments.id` — the two can numerically collide, which is exactly why `evidence.related_kind` distinguishes them (`'venuePayment'` vs `'payment'`) |
| project_id | BIGINT UNSIGNED `FK` → projects.id | no `project_vendor_id`-equivalent — a project has at most one venue (`projects.venue_id`), so this ties directly to the project like `client_payments` does |
| type | ENUM(...) | `PaymentType`, reused verbatim from `vendor_payments` |
| amount | BIGINT UNSIGNED | |
| payment_date | DATE | |
| method | VARCHAR(100) | same fixed `Tunai`/`Transfer Bank`/`QRIS` frontend dropdown, shared constant |
| reference_number | VARCHAR(100) | optional in the frontend form, same reasoning as `vendor_payments` |
| notes | TEXT NULL | |
| created_at | TIMESTAMP | |
| deleted_at | DATETIME NULL, `KEY idx_venue_payments_deleted` | migration `000068`, ADR-0034 — soft delete, as `client_payments.deleted_at` |

Money going **out** to the project's venue — same accounting direction and same two-slot evidence
rule (Invoice + Transfer Proof, `Refund` needs only proof) as `vendor_payments`, resolved via the
polymorphic `evidence` table (`related_kind = 'venuePayment'`), never a direct FK column — same
lesson already learned from `vendor_payments`' now-removed dead columns.

Edit — same as `vendor_payments` above, see ADR-0019; delete is a soft delete since ADR-0034.

### `vendor_issues`
| Column | Type | Notes |
|---|---|---|
| id | BIGINT UNSIGNED PK | |
| project_id | BIGINT UNSIGNED `FK` → projects.id | |
| project_vendor_id | BIGINT UNSIGNED NULL `FK` → project_vendors.id | **NULL since migration `000069`** (ADR-0034): a kendala about the event in general has no vendor. The FK stays (same module) |
| title, description | VARCHAR(255), TEXT | |
| impact | ENUM(...) | `IssueImpact` |
| found_date | DATE | |
| status | ENUM(...) | `IssueStatus` |
| resolution_plan, resolution_notes | TEXT NULL | |
| pic_staff_id | BIGINT UNSIGNED | `FK*` → `staff.staff_members.id` |
| target_resolution_date, resolved_date | DATE NULL | |
| is_client_visible | BOOLEAN NOT NULL DEFAULT TRUE | migration `000069` — staff decide whether the couple sees the kendala; **every Client Portal read filters it** (list, home counts/actions/news, progress count and condition, proof files). Existing rows are `TRUE` |

A kendala of a soft-deleted engagement is hidden with it (`project_vendor_id` points at a stamped engagement); a kendala with no vendor is never hidden that way.

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
| uploaded_by_staff_id | BIGINT UNSIGNED | `FK*` → `staff.staff_members.id` |
| related_kind | ENUM('vendorMilestone','payment','projectVendor','issue','clientPayment','venuePayment','general') | `'general'` (migration `000035`, ADR-0020) is a project-level document with no specific vendor/payment/vendor-engagement/issue to attach to (rundown, buku acara, teks juru bicara, banquet order, rekap dekor, etc.) — powers Client Portal's own "Dokumen" tab |
| related_id | BIGINT UNSIGNED | polymorphic — same-module FK pointing at one of the tables above depending on `related_kind`. For `related_kind = 'general'`, this is always the sentinel `0` (`AUTO_INCREMENT` never starts at 0), mirroring `projects.venue_id`'s own "0 = none" convention — there's nothing to point at |
| is_client_visible | BOOLEAN NOT NULL DEFAULT FALSE | migration `000035`, ADR-0020. Only ever meaningful for `related_kind = 'general'` — every other kind stays unconditionally visible to a `client` principal the moment it's attached to something inside their own project (unchanged); toggled via `POST .../evidence/{id}/toggle-client-visible` (rejected 422 for any other `related_kind`). Defaults `FALSE` (safe-by-default): a staff member must explicitly opt a general document in before `GET /projects/{id}/documents` (Client Portal's "Dokumen" tab) will ever return it |

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

* `deleted_at DATETIME NULL` (+ index) on `client_payments`, `vendor_payments`, `venue_payments`, `project_vendors` (module `projects`) and `clients` (module `clients`). `NULL` = live. The application writes the stamp (not `CURRENT_TIMESTAMP`), so `restorableUntil` is exactly `deleted_at + 24 h`.
* **Every read of a live row filters it**: repositories, the batched progress sources, the dashboard queries, the portal home, the overview, `CountActive`. A deleted engagement hides its milestones, payments, kendala and their evidence by joining the stamp of the engagement (the children are not stamped). `evidence` rows hang off these items by `(related_kind, related_id)` with no foreign key; every evidence read adds `NOT (related_kind = … AND related_id IN (SELECT id FROM <table> WHERE deleted_at IS NOT NULL …))` per kind, so a deleted item's files are not listed, opened or linked until it is restored. The unique `evidence.storage_path` keeps a deleted payment's upload claimed (a token cannot be reused by a new payment before the purge); a deleted client keeps its username and email reserved (its login is deactivated, not removed).
* **Purge** (`api purge`, ticker `PURGE_INTERVAL`): per module, under a MySQL advisory lock (`GET_LOCK`, `elproof_projects_purge` / `elproof_clients_purge`), `DELETE … WHERE deleted_at IS NOT NULL AND deleted_at <= now − 24 h` in batches of 200 per kind (evidence rows first, then the row; an engagement takes its vendor_issues → vendor_payments → vendor_milestones → itself), then the stored objects (best effort, after the commit). Restore accepts `deleted_at > now − 24 h`: exactly the complement, so a row is always restorable or due for purge, never both.
* `vendor_issues.project_vendor_id` is NULLable and `vendor_issues.is_client_visible BOOLEAN NOT NULL DEFAULT TRUE` was added (migration `000069`).
* Down migrations delete what the old shape cannot hold (soft-deleted rows with their evidence, kendala without a vendor); stored objects stay in the bucket.

## Module `audit` — migrations `000049`, `000059`, `000070` (ADR-0035 for the last)

### `audit_log`
| Column | Type | Notes |
|---|---|---|
| id | BIGINT UNSIGNED PK | |
| occurred_at | DATETIME(3) | `KEY idx_audit_log_occurred_at` |
| actor_type, actor_id, actor_role | VARCHAR(30), VARCHAR(64), VARCHAR(50) | `KEY idx_audit_log_actor (actor_type, actor_id, occurred_at)` |
| tenant_id | BIGINT UNSIGNED NULL | migration `000059`; `KEY idx_audit_log_tenant (tenant_id, occurred_at)` |
| action | VARCHAR(120) | readable label, e.g. `POST /tenants/{id}/toggle-suspension`. **Migration `000070`:** `KEY idx_audit_log_action (action, id)` — serves the exact `?action=` filter of `GET /audit-log` and `ORDER BY action, id` (Tahap 6 sorting) without sorting the whole append-only table; down drops it |
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
projects.venue_id              --*--> venues.id             (vendors)
project_milestone_templates.tenant_id --*--> tenants.id     (platform)
project_vendors.vendor_id      --*--> vendors.id            (vendors)
project_vendors.category_id    --*--> vendor_categories.id  (vendors)
project_vendors.pic_staff_id   --*--> staff_members.id      (staff)
vendor_milestones.pic_staff_id --*--> staff_members.id      (staff)
vendor_issues.pic_staff_id     --*--> staff_members.id      (staff)
evidence.uploaded_by_staff_id  --*--> staff_members.id      (staff)
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
