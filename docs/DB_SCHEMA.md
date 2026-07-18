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
| amount | BIGINT UNSIGNED | |
| payment_method | VARCHAR(100) | |
| payment_reference | VARCHAR(100) | |
| status | ENUM('unpaid','pending','paid','expired','granted') | `pending` (Fase 9) = a real gateway charge exists, awaiting webhook confirmation — distinct from `unpaid` (no charge attempt yet, e.g. right after tenant registration). `granted` = manual activation, excluded from paid-revenue reports |
| created_at | TIMESTAMP | |
| paid_at | TIMESTAMP NULL | |

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
| email | VARCHAR(150) | |
| phone | VARCHAR(30) | |
| is_active | BOOLEAN DEFAULT TRUE | |
| created_at, updated_at | TIMESTAMP | |

---

## Module `clients` — **implemented, Fase 4** (bundled with `projects`, see PLAN.md Fase 3 scope note —
`clients.project_id` needs a real `projects` row to reference)

### `clients`
| Column | Type | Notes |
|---|---|---|
| id | BIGINT UNSIGNED PK | |
| tenant_id | BIGINT UNSIGNED NOT NULL | |
| project_id | BIGINT UNSIGNED | `FK*` → `projects.projects.id` |
| role | ENUM('Bride','Groom','Family Representative') | |
| relation_note | VARCHAR(255) NULL | |
| name | VARCHAR(150) | |
| phone | VARCHAR(30) | |
| email | VARCHAR(150) | |
| is_active | BOOLEAN DEFAULT TRUE | |
| last_credential_reset_at | DATE NULL | |
| created_at, updated_at | TIMESTAMP | |

*(§6.3 resolved: "replace representative" (`POST /clients/{id}/replace-representative`) overwrites
this row with no history table — a deliberate choice carried over from the frontend mock's existing
behavior, not an oversight. Revisit only if a real need for an audit trail shows up in practice.)*

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

### `vendors`
| Column | Type | Notes |
|---|---|---|
| id | BIGINT UNSIGNED PK | |
| tenant_id | BIGINT UNSIGNED NOT NULL | |
| category_id | BIGINT UNSIGNED `FK` → vendor_categories.id | same-module FK |
| name | VARCHAR(150) | |
| pic_name | VARCHAR(150) | |
| phone | VARCHAR(30) | |
| email | VARCHAR(150) | |
| address | VARCHAR(255) | |
| notes | TEXT NULL | |
| is_active | BOOLEAN DEFAULT TRUE | |
| created_at, updated_at | TIMESTAMP | |

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
| venue | VARCHAR(255) | |
| package_name | VARCHAR(150) | |
| contract_value | BIGINT UNSIGNED | |
| status | ENUM('Draft','Preparation','Ready','Completed','Cancelled') | |
| pic_staff_id | BIGINT UNSIGNED | `FK*` → `staff.staff_members.id` |
| description | TEXT NULL | |
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

### `project_vendors`
| Column | Type | Notes |
|---|---|---|
| id | BIGINT UNSIGNED PK | |
| project_id | BIGINT UNSIGNED `FK` → projects.id | |
| vendor_id | BIGINT UNSIGNED | `FK*` → `vendors.vendors.id` |
| category_id | BIGINT UNSIGNED | `FK*` → `vendors.vendor_categories.id` (denormalized at engagement time) |
| scope | VARCHAR(255) | |
| contract_value | BIGINT UNSIGNED | |
| engagement_status | ENUM(...) | see `EngagementStatus` |
| booking_date, event_date, due_date | DATE NULL | |
| dp_amount, paid_amount | BIGINT UNSIGNED | |
| pic_staff_id | BIGINT UNSIGNED | `FK*` → `staff.staff_members.id` |
| notes | TEXT NULL | |

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

### `vendor_payments`
| Column | Type | Notes |
|---|---|---|
| id | BIGINT UNSIGNED PK | |
| project_id | BIGINT UNSIGNED `FK` → projects.id | |
| project_vendor_id | BIGINT UNSIGNED `FK` → project_vendors.id | |
| type | ENUM(...) | `PaymentType`, includes `Refund` (special evidence-completeness rule) |
| amount | BIGINT UNSIGNED | |
| payment_date | DATE | |
| method, reference_number | VARCHAR(100) | |
| invoice_evidence_id | BIGINT UNSIGNED NULL | `FK` → evidence.id (same module) |
| proof_evidence_id | BIGINT UNSIGNED NULL | `FK` → evidence.id (same module) |
| notes | TEXT NULL | |

### `vendor_issues`
| Column | Type | Notes |
|---|---|---|
| id | BIGINT UNSIGNED PK | |
| project_id | BIGINT UNSIGNED `FK` → projects.id | |
| project_vendor_id | BIGINT UNSIGNED `FK` → project_vendors.id | |
| title, description | VARCHAR(255), TEXT | |
| impact | ENUM(...) | `IssueImpact` |
| found_date | DATE | |
| status | ENUM(...) | `IssueStatus` |
| resolution_plan, resolution_notes | TEXT NULL | |
| pic_staff_id | BIGINT UNSIGNED | `FK*` → `staff.staff_members.id` |
| target_resolution_date, resolved_date | DATE NULL | |

### `evidence`
| Column | Type | Notes |
|---|---|---|
| id | BIGINT UNSIGNED PK | |
| project_id | BIGINT UNSIGNED `FK` → projects.id | denormalized for direct project-scoped queries |
| name | VARCHAR(255) | |
| type | ENUM(...) | `EvidenceType` |
| storage_path | VARCHAR(500) | see ADR-0006 — path returned by `shared/storage`, never a public URL |
| file_name | VARCHAR(255) | original filename |
| document_date | DATE NULL | |
| uploaded_at | TIMESTAMP | |
| description | VARCHAR(255) NULL | |
| uploaded_by_staff_id | BIGINT UNSIGNED | `FK*` → `staff.staff_members.id` |
| related_kind | ENUM('vendorMilestone','payment','projectVendor','issue') | |
| related_id | BIGINT UNSIGNED | polymorphic — same-module FK pointing at one of the tables above depending on `related_kind` |

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
| created_at | TIMESTAMP | |

Thin dispatch index only — PK uniqueness on `order_ref` doubles as idempotency (a repeat charge
attempt with the same ref fails with a DB duplicate-key error, surfaced as `409`-equivalent).

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

## Cross-module reference summary (no SQL FKs across this boundary)

```
staff_members.tenant_id        --*--> tenants.id            (platform)
clients.tenant_id              --*--> tenants.id            (platform)
clients.project_id             --*--> projects.id           (projects)
vendors.tenant_id              --*--> tenants.id            (platform)
projects.tenant_id             --*--> tenants.id            (platform)
projects.pic_staff_id          --*--> staff_members.id      (staff)
project_vendors.vendor_id      --*--> vendors.id            (vendors)
project_vendors.category_id    --*--> vendor_categories.id  (vendors)
project_vendors.pic_staff_id   --*--> staff_members.id      (staff)
vendor_milestones.pic_staff_id --*--> staff_members.id      (staff)
vendor_issues.pic_staff_id     --*--> staff_members.id      (staff)
evidence.uploaded_by_staff_id  --*--> staff_members.id      (staff)
activity_log.actor_staff_id    --*--> staff_members.id      (staff)
tenants.plan_id                --*--> subscription_plans.id (billing)
subscription_transactions.tenant_id --*--> tenants.id       (platform)
credentials.principal_id       --*--> (varies by principal_type)
pending_subscription_charges.tenant_id --*--> tenants.id             (platform, same module)
pending_subscription_charges.plan_id   --*--> subscription_plans.id  (billing)
payment_charge_dispatch.app_id --*--> payment_apps.app_id  (payment, same module)
```
