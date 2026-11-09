# API Contract — ElProof

> Endpoint list and request/response formats. Conventions: `.claude/rules/api-standard.md` +
> [`knowledge/API_GUIDE.md`](../knowledge/API_GUIDE.md). All paths are under `/api/v1`. Auth header
> required on every endpoint except login/refresh (see API_GUIDE.md).

Envelope reminder — success: `{ "success": true, "message": "...", "data": ... }`; error:
`{ "success": false, "message": "...", "errors": {...} }`; paginated lists additionally return
`meta: { page, limit, total, total_pages }`.

**Pagination — Fase 7, applies to every `GET` list endpoint marked "paginated" below:** default
response is paginated (`?page=`/`?limit=`, default `page=1`/`limit=10`, `limit` capped at 100).
Some **tenant-scoped rosters** of the WO Console (`/vendors`, `/venues`, `/vendor-categories`,
`/clients`, `/staff`, `/projects`, `/plans`) still accept `?all=true` for the unpaginated shape
(`response.OK`, no `meta`) — only the pickers and dropdowns use it, and each is bounded by one
tenant. **The Platform Console endpoints do not:** `/tenants`, `/platform-admins` and
`/subscription-transactions` are always paginated (Fase 7), and global search asks the paginated
endpoints with `?search=&limit=4` instead of loading every list. `?search=`
does a `LIKE` match (columns vary per endpoint, noted below); other filters (`role`, `categoryId`,
`status`) apply at the SQL level too, so `search` + a filter + a page number combine correctly
instead of a filter being applied only to one already-paginated page.

---

## `identity`

| Method | Path | Notes |
|---|---|---|
| POST | `/auth/login` | body `{username, password}` — `username` accepts either a real username or an email; tried as username first, then as email (against every credential sharing that email, since email isn't unique the way username is — see ADR-0005) → `{accessToken, refreshToken, principalType, principalId, tenantId, role, displayName}` |
| POST | `/auth/refresh` | body `{refreshToken}` → new `{accessToken, refreshToken}`, rotates old refresh token |
| POST | `/auth/logout` | revokes the refresh token; requires auth header |

## `payment` — "one merchant wallet, many consumers" — **implemented, internal + external mode (Fase 9 + Fase 10)** — see [`knowledge/MODULE_PAYMENT.md`](../knowledge/MODULE_PAYMENT.md)

| Method | Path | Notes |
|---|---|---|
| POST | `/webhooks/payment` | **Unauthenticated** (no JWT — the gateway can't carry a bearer token). Trust comes entirely from `X-Callback-Signature` (HMAC-SHA256 of the raw body, keyed with the merchant private key), verified inside the handler. Idempotent — a duplicate delivery of the same event is a no-op. The single, permanent callback URL registered with Tripay; never changes as new Apps are added. Dispatches to `kind=internal` Apps in-process, or relays (HMAC-signed, fire-and-forget) to `kind=external` Apps' `callback_url`. A charge whose webhook never arrives at all is still eventually resolved by a periodic reconciliation sweep (`PAYMENT_RECONCILE_INTERVAL`, default 3m) that re-checks the gateway directly — see `knowledge/MODULE_PAYMENT.md` §6 step 6. |
| GET | `/payment/gateway-config` | `platform_admin` only. Returns `{activeProvider, isSandbox, tripayMerchantCode, hasTripayApiKey, hasTripayPrivateKey}` — booleans only for the two secret fields, never the plaintext/encrypted value itself. |
| PATCH | `/payment/gateway-config` | `platform_admin` only. Body `{activeProvider, isSandbox, tripayMerchantCode, tripayApiKey, tripayPrivateKey}` — `tripayApiKey`/`tripayPrivateKey` are write-only: an empty string means "leave the currently-stored value unchanged," never "clear it." |
| GET | `/payment/apps` | `platform_admin` only. Lists every App (internal + external) — `{appId, name, kind, callbackUrl, isActive, createdAt}`, never the secret hash/encrypted copy. |
| POST | `/payment/apps` | `platform_admin` only. Body `{name, callbackUrl}` → registers a new `kind=external` App, returns `{appId, name, secret}` — the plaintext secret is shown exactly once. |
| POST | `/payment/apps/{appId}/reset-secret` | `platform_admin` only. Rotates an external App's secret, returns `{appId, secret}` — again, shown once. |
| POST | `/payment/apps/{appId}/toggle-active` | `platform_admin` only. Single toggle (flips current state); rejected for `kind=internal` Apps — disabling `platform-billing` would sever ElProof's own subscription billing. |
| POST | `/auth/app/token` | **Unauthenticated** (this IS the login step for external Apps) but rate-limited per IP. Body `{appId, secret}` → `{accessToken, tokenType: "Bearer", expiresIn}`. No refresh token — the App re-exchanges appId+secret once expired. |
| POST | `/external/payments/charges` | `app` principal only (see below). Body `{orderRef, amount, channel?, customerName?, customerEmail?, customerPhone?}` — `appId` always resolved from the token, never the body. Duplicate `orderRef` → `409` with `errors.code = "conflict"`. |
| GET | `/external/payments/charges/{orderRef}/status` | `app` principal only. `404` if `orderRef` doesn't belong to the calling App. |
| GET | `/external/payments/channels` | `app` principal only. |

**The `app` principal (Fase 10):** minted only via `/auth/app/token`, never through `/auth/login`.
Every `/external/payments/*` request re-checks the App's `is_active` status live against the
registry (not just JWT validity) — deactivating an App from Platform Console's "Manajemen Aplikasi"
page takes effect on the very next request, not at token expiry. The three external routes return
an additional machine-readable `errors.code` (`bad_request`/`unauthorized`/`forbidden`/`not_found`/
`conflict`/`rate_limited`/`internal`) alongside the usual `message` — every other route in the app
is unaffected, this is additive.

## `platform` (Platform Console — `platform_admin` only unless noted) — **implemented, Fase 2**

| Method | Path | Notes |
|---|---|---|
| GET | `/tenants` | **paginated** (Fase 7); `?search=` matches business/owner name + email; `?status=` filters `subscriptionStatus`; `?planId=` and `?source=self_service|platform` (Fase 6) |
| POST | `/tenants` | orchestrates `staff.CreateOwner` + `identity.CreateCredential` — see ADR-0008. No `planId` in the body and no transaction recorded: the tenant starts unbound to any plan (`planId: null`, `subscriptionStatus: pending_payment`); the first transaction row it ever gets comes from whichever real subscription event happens first — the Owner's own `/subscriptions/pay` or the Platform Console's `/activate-subscription` below — never a placeholder created here |
| GET | `/tenants/{id}` | |
| PATCH | `/tenants/{id}` | body now also accepts `brandColorPreset` (see `MODULE_PLATFORM.md` §6) — empty string leaves the tenant's current preset unchanged rather than rejecting the whole update; a non-empty value not in the fixed 20-key set is rejected (422). |
| PUT | `/tenants/{id}/logo` | body `{fileName, mimeType, base64Data}` — same base64-JSON shape as evidence upload (ADR-0010), not multipart. `mimeType` restricted to `image/png`/`image/jpeg`/`image/webp` (no SVG — script-injection risk not worth it for a logo); decoded size capped at 2 MB (evidence's cap is 15 MB — a logo is a small header image, not a document). First `PUT` in this API — `internal/shared/middleware/cors.go`'s dev-mode `Access-Control-Allow-Methods` already includes it |
| GET | `/tenants/{id}/logo` | streams the stored logo bytes back (same byte-proxy-through-Go shape as evidence download) — `404` if the tenant has no logo configured |
| POST | `/tenants/{id}/toggle-suspension` | single toggle (flips current state), not separate suspend/reactivate |
| POST | `/tenants/{id}/reset-credential` | body `{password}`, resets the Owner's login via `identity.ResetPasswordByUsername` |
| POST | `/tenants/{id}/activate-subscription` | body `{planId}` → `billing.RecordTransaction` status `granted`, bypasses payment. End state (`subscriptionStatus`/`subscriptionExpiresAt`) is computed by the exact same `UpdateSubscription` call as a real payment confirmation — granted and paid subscriptions are indistinguishable afterward except in the transaction ledger. Also resolves (expires + untracks) any self-service charge still pending for this tenant first, so a late webhook/reconciliation-sweep result for that old charge can't double-extend the expiry this call just set |
| GET | `/tenants/me` | **staff Owner self-service** (not platform_admin) — own tenant's plan/status/expiry, powers WO Console's `Langganan` |
| GET/PUT | `/tenants/me/business-details` | **self-service, needs `business_profile.manage`**. `{address, bankName, bankAccountNumber, bankAccountHolder}`, all optional, but a bank account is all-or-nothing (422 with per-field `errors` otherwise; number 5–40 digits/space/dot/dash). Shown to clients as the account to pay into — ADR-0029 |
| GET | `/tenants/me/contact` | **any tenant-scoped principal** (staff or client). `{businessName, whatsapp (62…), city, address}` plus `bankName`/`bankAccountNumber`/`bankAccountHolder` **only** for staff or a client whose portal role has `portal.finance.view` (a Keluarga account gets none). Powers the Client Portal's contact and payment cards |
| PATCH | `/tenants/me/brand-color` | **self-service, needs `business_profile.manage`** (Owner by default). Body `{preset}`, one of the 21 fixed presets (422 `errors.preset` otherwise). Recolors the WO Console and Client Portal for everyone in the workspace |
| GET | `/tenants/me/branding` | **self-service, any tenant-scoped principal** (any staff role, or `client` — not Owner-only, unlike `/tenants/me` above, since branding must render for every staff role and every client) — `{businessName, brandColorPreset, hasLogo}`. Tenant resolved strictly from the JWT's `tenant_id` claim, never a request param |
| GET | `/tenants/me/logo` | same self-service scoping as `/tenants/me/branding` — streams the caller's own tenant's logo |
| GET | `/public/login-slides` | **unauthenticated**, global (not tenant-scoped) — `{count}`, the number of login-page marketing photos currently uploaded (see `MODULE_PLATFORM.md` §9) |
| GET | `/public/login-slides/{index}` | **unauthenticated** — streams one slide's JPEG bytes by 0-based index, `Cache-Control: public, max-age=86400`; `404` if out of range |
| POST | `/subscriptions/pay` | **staff Owner self-service** — body `{planId}`. **Changed Fase 9:** no longer activates synchronously — creates a real charge via `payment.Client.CreateCharge` (QRIS), records a `pending` transaction, and returns the charge (`{orderRef, providerRef, channel, qrImageUrl, payCode, checkoutUrl, amount, feeAmount, expiresAt, status}`). The subscription only activates once `/webhooks/payment` confirms it — see `payment` module below. Rejects (409) if this tenant already has a pending charge — see `GET`/`cancel` below for how the Owner resolves that instead of getting stuck |
| GET | `/subscriptions/pending-charge` | **staff Owner self-service** — returns `data: null` if nothing is pending, otherwise the same `{orderRef, providerRef, channel, qrImageUrl, payCode, checkoutUrl, amount, feeAmount, expiresAt, status}` shape as `Pay`, re-fetched live from the gateway via `payment.Client.CheckStatusForApp` (never cached — these fields were never persisted after `Pay`'s one-time response). Lets the Owner re-view a still-pending charge (e.g. after accidentally closing its QR modal) instead of losing access to it until it expires |
| POST | `/subscriptions/pending-charge/cancel` | **staff Owner self-service** — 404 if nothing is pending. Marks the pending transaction `cancelled` (distinct from `expired` — this was a deliberate give-up, not a timeout) and stops tracking it, freeing `Pay` to start a fresh charge immediately. Bookkeeping only: Tripay has no cancel/void API, so the old charge is still technically payable at the gateway until its own expiry — if the Owner somehow still completes it, the late webhook/reconciliation result finds no tracking row and is silently ignored rather than reactivating the subscription |
| GET | `/platform-admins` | **paginated** (Fase 7); `?search=` matches name + email; `?role=` filters exact role |
| POST | `/platform-admins` | orchestrates `identity.CreateCredential` |
| PATCH | `/platform-admins/{id}` | |
| POST | `/platform-admins/{id}/toggle-active` | single toggle; self-lockout (an admin deactivating their own account) rejected with 403 |
| POST | `/platform-admins/{id}/reset-password` | |

### Platform Console back office (Fase 6, ADR-0030)

All timestamps in these responses carry the WIB offset (+07:00). Changing a tenant's `email` through `PATCH /tenants/{id}` is Super Admin only (403 `forbidden_role` for Support). Role policy (`cmd/server/platform_scope.go`): every platform role may read except **Read-only**, which cannot read
a workspace's internals (`/console/tenants/**`, `/console/data-requests`, `/console/support-sessions`,
`/tenants/export`). Writes: notes and data requests need Support or Super Admin; starting/ending a support session
needs Support or Super Admin; announcements, policy, audit log and login photos need Super Admin.

| Method | Path | Notes |
|---|---|---|
| GET | `/platform/metrics` | any platform role. `{totalTenants, trialing, active, pastDue, readOnly, suspended, frozen, paidTenants, mrr, arpa, trialsEnded30d, trialsConverted30d, trialConversionPct, churned30d, churnedMrr, churnPct, newPaid30d, newMrr30d, revenue30d, revenuePrev30d, overdueValue, trialsEndingSoon[], overdueTenants[]}`. Percentages are `null` with no base; free grants never count as revenue |
| GET | `/tenants/export` | CSV with the same `search/status/planId/source` filters; formula-injection safe; writes its own audit record |
| GET | `/console/tenants/{id}/overview` | `{tenant, planName, consent, activations[], staff, activeProjects, clients, members[], openDataRequests, supportAccessUntil}`; `staff/activeProjects/clients` are `{used, max}` with `max: null` on a paid plan |
| GET/POST | `/console/tenants/{id}/notes` | internal notes `{id, admin, body, createdAt}`; POST body `{body}` |
| GET/POST | `/console/tenants/{id}/data-requests` | POST body `{kind: export|delete, requestedBy, note}`; `dueAt` is created + 30 days |
| GET | `/console/data-requests` | `?open=true` for the queue; each item carries `tenantName`, `daysUntilDue` |
| POST | `/console/data-requests/{id}/complete` | body `{status: done|rejected, note}`; the note (proof) is mandatory |
| GET/POST | `/console/announcements` | `{title, body, kind: info|warning|maintenance, audience: all|plan|status, audienceValue, startsAt, endsAt?}` (RFC 3339) |
| PATCH/DELETE | `/console/announcements/{id}` | |
| GET | `/announcements/active` | any staff role: announcements that apply to the caller's workspace now |
| GET/PUT | `/console/subscription-policy` | `{trialDays 1–90, expiringSoonDays 0–30, graceDays 0–60, readOnlyDays 0–180}`; PUT is Super Admin and takes effect for every tenant |
| GET | `/console/tenants/{id}/support` | consent and session history `{grant, sessions[]}` |
| POST | `/console/tenants/{id}/impersonate` | body `{reason}` (10–255 chars). Needs the Owner's active consent (403 `support_access_not_granted` otherwise). Returns `{sessionId, accessToken, expiresAt, tenantId, role}`: a 60-minute **read-only** token for the Owner's credential (claims `imp`, `sid`; no refresh token). Every non-GET with it → 403 `impersonation_read_only`; any request after the session ended, consent was withdrawn or the admin was deactivated → 401 `support_session_ended` |
| POST | `/console/support-sessions/{id}/end` | only the admin who started it |
| GET/PUT/DELETE | `/tenants/me/support-access` | **Owner side** (`business_profile.manage`): GET status + history, PUT grants 24 hours, DELETE withdraws consent |
| PUT | `/console/login-slides/{index}` | Super Admin. Body `{mimeType: image/jpeg|image/png, base64Data}`, ≤ 6 MB decoded; replaces one of the three login photos |
| GET | `/audit-log` | **Super Admin**, paginated; filters `actorType`, `actorId`, `tenantId`, `outcome=ok|failed`, `q`, `from`, `to` (WIB days) |
| GET | `/audit-log/export` | **Super Admin**, same filters, CSV (max 10 000 rows); writes its own audit record |

Fase 7 additions for the console (any platform role may read them):

| Method | Path | Notes |
|---|---|---|
| GET | `/platform/trends` | `?period=month|year` (default month; 400 otherwise). `{period, revenue[], newTenants[]}`, each point `{key, label, value}`; bars run from the 1st (or January) through today, bucketed in WIB. Replaces the client-side calculation over every tenant and transaction |
| GET | `/tenants/names` | `?ids=1,2,3` (at most 100) → `{"1": "WO name"}`; unknown ids are absent. Lets a list page name the tenants of its own rows without loading the roster |
| GET | `/platform-admins/{id}` | one admin; the console reads its own profile this way instead of listing the whole team |
| GET | `/subscription-transactions/{id}` | one transaction. A staff Owner reaches only their tenant's (anyone else's id → 404, never 403); used by the receipt page |

## `billing` (read by both consoles; writes are `platform_admin`-only) — **implemented, Fase 2**; isolated per-App catalog + subscription ledger added `docs/plan/external-subscription-plans/PLAN.md`

| Method | Path | Notes |
|---|---|---|
| GET | `/plans` | **principal-aware since the isolated-catalog change** (was: public to any authenticated principal reading the same list). `platform_admin` sees every plan (public + private); any other principal — including no claim at all — sees only plans with no App mapping (ElProof's own public catalog). `planResponse` gained an additive field `appIds: string[]`, populated only for `platform_admin` ([] otherwise) — a non-empty value marks a plan private to those App IDs. WO Console's `Langganan` reads this too; **paginated** (Fase 7), but every current frontend consumer calls `?all=true` since the plan catalog is a small, bounded reference list, not a growing per-tenant one |
| POST | `/plans` | platform_admin only |
| PATCH | `/plans/{id}` | platform_admin only |
| POST | `/plans/{id}/toggle-active` | platform_admin only |
| GET | `/subscription-transactions` | platform_admin: all tenants (optional `?tenantId=`); staff: forced to own tenant from JWT regardless of query; **paginated** (Fase 7), `?status=` filters transaction status (now includes `pending`, added Fase 9) |
| GET | `/external/subscription-plans` | `app` principal only. Every plan mapped to the calling App (isolated per App — App A never sees App B's plans), including non-active ones with an `active` flag. Response: `{id, name, durationMonths, price, active, features}` — note `active`, not `isActive` (deliberately different from `/plans` above), and `features` always present, never `null`. No mapping → `200` with `data: []`, not `404`. |
| GET, PUT | `/app-plans` | `platform_admin` only. `?appId=` query on `GET` → `{appId, planIds}`; `PUT` body `{appId, planIds}` **replaces the entire mapping set** (not a delta) — Platform Console's "Kelola Paket" tab inside `AppDetailModal` uses this, never SQL by hand. `appId` not found in the App registry → `422` (no FK exists between this table and `payment_apps`, since that would be a cross-module FK). |
| POST | `/external/subscriptions/charges` | `app` principal only. The correct way for an external App to buy/renew an ElProof subscription — as opposed to `payment`'s `/external/payments/charges` (general-purpose, amount supplied by caller). Body `{orderRef, planId, customerName?, customerEmail?, customerPhone?}` — **no `amount`**, the price comes from the catalog server-side. `planId` not mapped to the calling App → `422` (the real guard against buying another App's plan — `planId` is guessable, the catalog endpoint alone isn't sufficient). Response shape identical to `payment`'s own charge response. Status polling reuses `payment`'s existing `GET /external/payments/charges/{orderRef}/status` unchanged — not duplicated here. |
| GET | `/app-subscriptions` | `platform_admin` only. `[{appId, planName, subscriptionUntil, status}]` — "who's paying for what, until when" per external App, backing `AppDetailModal`'s tab Langganan. An App with no paid history simply doesn't appear (ledger starts empty until a consumer switches to the subscription-charge endpoint above) — the frontend renders that as "—", not an error. |

No `POST /subscription-transactions` on this module — both "pay" and "activate" mutations live on
`platform`'s routes (`/subscriptions/pay`, `/tenants/{id}/activate-subscription`) since they must
also update the `tenants` row `billing` doesn't own; they call `billing.Contracts.RecordTransaction`
internally. See ADR-0008.

`subscription_plan_apps` (visibility mapping) and `subscription_app_charges` (ledger) are owned by
`billing`, not `payment`, even though the two `/external/subscriptions/*` and `/app-plans`/
`/app-subscriptions` routes above are token-App-authenticated the same way `payment`'s own external
routes are — `billing` receives `payment.contracts.AppRegistry`/`Client` at construction time
(`main.go`) rather than duplicating App-registry logic. See `docs/DB_SCHEMA.md` and
`knowledge/MODULE_PAYMENT.md` for the module-boundary reasoning.

## `staff` (WO Console, tenant-scoped) — Owner-only (`requireOwnerTenant`) except `/staff/summary`

| Method | Path | Notes |
|---|---|---|
| GET | `/staff` | Owner-only; **paginated** (Fase 7); `?search=` matches name + email; `?role=` filters exact role; response includes `username` |
| GET | `/staff/summary` | any staff role (`requireTenant`, not `requireOwnerTenant`) — `[{id, name, title}]` only, deliberately excludes `username`/`email`/`phone`/`isActive`. Backs every PIC picker/"assigned to" label across the `projects` module (project PIC, vendor-engagement PIC, milestone PIC, issue PIC, activity actor) — needed by every role, not just Owner |
| POST | `/staff` | body `{name, title, role, username, password, email, phone}` — rejects `role=Owner` (422); provisions a real login credential via `identity.CreateCredential` in the same call (mirrors `clients`' `POST /clients`), rolling back (deleting) the just-inserted row if that fails, e.g. username or email already taken by *any* principal on the platform (neither is tenant-scoped) |
| PATCH | `/staff/{id}` | body `{name, title, role, email, phone}` — no `username` (immutable after creation). Rejects assigning `role=Owner` to a non-Owner row (403). For a row that's already `role=Owner`: rejected (403) unless the caller's own JWT principal id equals `{id}` — i.e. the Owner may edit their own name/title/contact details, but no other staff member (even Admin) may touch the Owner's row; even the Owner can't change their own role away from `Owner` this way |
| POST | `/staff/{id}/toggle-active` | rejects deactivating the Owner row (403), regardless of caller |
| GET | `/staff/{id}/delete-impact` | Owner-only — `{affectedProjects: [{id, name}, ...]}`, deduplicated across the four **live PIC-assignment** columns only (`projects.pic_staff_id`, `project_vendors.pic_staff_id`, `vendor_milestones.pic_staff_id`, `vendor_issues.pic_staff_id`) — the two audit-trail columns (`activity_log.actor_staff_id`, `evidence.uploaded_by_staff_id`) are deliberately excluded, see ADR-0018 |
| DELETE | `/staff/{id}` | Owner-only — hard delete, no precondition (ADR-0018); **unconditionally rejected (403), no dialog possible**, if `{id}` is the tenant's own Owner row — a tenant can never end up with zero Owner accounts, regardless of what `delete-impact` would show |

## `clients` — **implemented, Fase 4** (bundled with `projects`, since `clients.project_id` needs
a real `projects` row to reference)

| Method | Path | Notes |
|---|---|---|
| GET | `/clients` | tenant-scoped; `?projectId=` filters to one project (unpaginated — small, bounded per-project list). Omit `projectId` to list every client for the tenant instead (added Fase 5, powers `GlobalSearch` via `?all=true`) — this tenant-wide mode is **paginated** (Fase 7), `?search=` matches name + email; response includes `username` |
| POST | `/clients` | body `{projectId, role, relationNote, name, phone, email, username, password}` — validates `projectId` via `projects.Contracts.ProjectExists`, provisions a real login credential in the same call; if `identity.CreateCredential` fails (e.g. username or email already taken by *any* principal on the platform — neither is tenant-scoped), the just-inserted `clients` row is rolled back (deleted) rather than left orphaned with no working credential |
| PATCH | `/clients/{id}` | contact edit (`{name, phone, email}`) — no `username` (immutable after creation) |
| POST | `/clients/{id}/toggle-active` | |
| POST | `/clients/{id}/reset-credential` | body `{password}` — sets a new password and stamps `lastCredentialResetAt`. Fails with `404`/"Kredensial tidak ditemukan" for a client left behind by a pre-rollback-fix `Create` failure (no `identity` row ever existed) — use `DELETE` below to clear those out, not this endpoint |
| POST | `/clients/{id}/replace-representative` | body `{name, phone, email, relationNote}`; only valid when `role=Family Representative`; overwrites the same row, no history kept (§6.3) |
| DELETE | `/clients/{id}` | hard delete — the one permanent-delete endpoint in this module (every other mutation elsewhere is a soft toggle). Best-effort deactivates the client's `identity` credential first (harmless no-op if none exists); exists specifically so an operator can self-service clear a client stuck with no working credential, since deactivating alone doesn't free up that client's role slot on the project (the role lookup doesn't filter on `is_active`) |

## `vendors` (tenant-scoped) — **implemented, Fase 3** (Venue directory added ADR-0016; RBAC below added later — see ADR-0017)

**Role gate, applies uniformly across `vendors`/`venues` below (`requireManagerRole`):** reading
(list/get/attachment-download) stays open to any staff role, including Wedding Planner (`"Staff"`),
so the vendor/venue pickers inside their own project keep working — but every write
(create/update/toggle-active/attachment-upload/template/import) is Owner/Admin only.

| Method | Path | Notes |
|---|---|---|
| GET | `/vendor-categories` | tenant-scoped, open to any staff role; **paginated** (Fase 7) view is Owner-only, `?all=true` (picker/display use) stays open to any role; `?search=` matches category name |
| POST | `/vendor-categories` | Owner-only |
| PATCH | `/vendor-categories/{id}` | Owner-only |
| POST | `/vendor-categories/{id}/toggle-active` | Owner-only |
| DELETE | `/vendor-categories/{id}` | Owner-only — **hard delete, no `delete-impact` counterpart** (blocked, not informed-consent): `409` if any vendor still references it (`vendors.category_id` is a real, same-module, `NOT NULL` FK) with a count in the message; see ADR-0018 |
| GET | `/vendors` | `?categoryId=`, `?city=` (exact match against the fixed 128-entry city enum) filters; **paginated** (Fase 7); `?search=` matches name + PIC + email |
| POST | `/vendors` | Owner/Admin only. Body now also accepts `socialMedia`, `city`, `priceAkad`, `priceAkadResepsi` (migration `000023`) — `email`/`address` are no longer mandatory; validates `categoryId` belongs to the same tenant (422 if not) |
| PATCH | `/vendors/{id}` | Owner/Admin only; same body/`categoryId` validation as create |
| POST | `/vendors/{id}/toggle-active` | Owner/Admin only |
| GET | `/vendors/{id}/delete-impact` | Owner-only — `{affectedProjects: [{id, name}, ...]}`, every project whose `project_vendors.vendor_id` points at this vendor; called by the frontend the instant "Hapus" is clicked, before `DELETE` — see ADR-0018 |
| DELETE | `/vendors/{id}` | Owner-only — hard delete, no precondition (informed-consent model, not block-then-cascade — ADR-0018); best-effort deletes the object-storage attachment too, log-and-continue on failure |
| PUT | `/vendors/{id}/attachment` | Owner/Admin only. Body `{fileName, mimeType, base64Data}` — same base64-JSON shape as evidence/logo upload; single slot (document or photo), replaces any existing attachment |
| GET | `/vendors/{id}/attachment` | any staff role — streams the stored bytes back, `Content-Disposition: inline` |
| GET | `/vendors/{id}/project-history` | Owner/Admin only — backs "Lihat Project," the vendor's engagement history across every project in the tenant, `[{projectId, projectName, eventDate, venue, engagementStatus}]` ordered by `eventDate DESC`. `project_vendors` is owned by `projects`, not `vendors` — resolved through `projects.Contracts.ListVendorEngagementHistory` (`vendors.NewModule` now takes `projects.Contracts` as a constructor arg; `main.go` builds `projects` before `vendors` accordingly). A Wedding Planner has no legitimate need to see a vendor's history across the whole tenant, only their own project's own engagement (lives in `projects` itself) |
| GET | `/vendors/summary` | any staff role — `[{id, name}]`, **includes inactive vendors** (unlike every other `?all=true` listing in this module) so a project already engaging a since-deactivated vendor can still resolve its name. Backs Client Portal's Vendor tab (which also pins the project's venue as its first card, client-facing presentation only — ADR-0016's Revision) and any picker that only ever needs a name |
| GET | `/vendors/template` | Owner/Admin only — downloads an `.xlsx` with the fixed header row `Nama Vendor, Kategori, Nama PIC, No Tlp Vendor, Email, Sosial Media, Kota, Alamat, Harga Akad, Harga Akad+Resepsi, Catatan`, styled (frozen header row, sized columns, thousand-separator number format on the two price columns) and with an in-file dropdown on Kategori (the tenant's own registered categories) and Kota (the fixed 128-entry enum), sourced from a hidden helper sheet since both lists exceed Excel's inline dropdown-formula character cap |
| POST | `/vendors/import` | Owner/Admin only. Body `{base64Data}` (an `.xlsx` matching the template's column order) → partial-success import, `{insertedCount, updatedCount, errors: [{row, message}]}` — invalid rows are reported back by row number, valid rows still commit; capped at 1000 data rows; a price cell formatted with thousand separators (e.g. from the styled template above) is parsed correctly (separators stripped before `ParseInt`) |
| GET | `/vendors/export` | Owner/Admin only — same `?categoryId=`/`?city=`/`?search=` filters as `GET /vendors` (unpaginated, exports the matching set); column order matches `/vendors/template` exactly plus an appended `Status` column, so the file round-trips back through `/vendors/import` unmodified |

## `venues` (tenant-scoped) — **implemented, ADR-0016**

Its own directory inside the `vendors` module (not a new top-level Go module) — see ADR-0016. Same
`requireManagerRole` read/write split as `vendors` above, with one exception: the attachment
download route is reachable by a `client` principal too (see below).

| Method | Path | Notes |
|---|---|---|
| GET | `/venues` | any staff role; `?city=` (exact match against the fixed 128-entry city enum) filter; **paginated** (Fase 7); `?search=` |
| POST | `/venues` | Owner/Admin only. Body `{name, picName, phonePic, phoneVenue, email, address, city, rentalPrice, charge, capacity, facilities, socialMedia, notes}` |
| GET | `/venues/{id}` | any staff role — full record (rental price, charge, PIC contact included); the Client Portal never calls this directly, only the public-safe `GET /projects/{id}/venue` below |
| PATCH | `/venues/{id}` | Owner/Admin only |
| POST | `/venues/{id}/toggle-active` | Owner/Admin only |
| GET | `/venues/{id}/delete-impact` | Owner-only — `{affectedProjects: [{id, name}, ...]}`, every project whose `projects.venue_id` points at this venue — see ADR-0018 |
| DELETE | `/venues/{id}` | Owner-only — hard delete, no precondition (ADR-0018); best-effort deletes the object-storage attachment too |
| PUT | `/venues/{id}/attachment` | Owner/Admin only. Body `{fileName, mimeType, base64Data}` — single slot (document or photo) |
| GET | `/venues/{id}/attachment` | reachable by **both** `staff` and `client` principals (`requireTenant`, not `requireStaffTenant`) — the mime-type gate is enforced inside the handler itself, not just at the route level: an image-type attachment is served to a `client` principal (treated as a venue photo), a non-image (e.g. a PDF contract) is rejected 403 for `client` even with a valid venue id |
| GET | `/venues/template` | Owner/Admin only — `.xlsx` header row `Nama Venue, Nama PIC, No Tlp PIC, No Tlp Venue, Email, Alamat, Kota, Harga Sewa, Charge, Kapasitas, Fasilitas, Sosial Media, Catatan`, styled the same way as `/vendors/template` (frozen header, sized columns, thousand-separator format on Harga Sewa/Charge) with an in-file Kota dropdown |
| POST | `/venues/import` | Owner/Admin only. Same partial-success shape as `/vendors/import`; required fields per row: name/picName/phonePic/city; a thousand-separator-formatted price cell parses correctly, same fix as `/vendors/import` |
| GET | `/venues/export` | Owner/Admin only — same `?city=`/`?search=` filters as `GET /venues` (unpaginated); column order matches `/venues/template` plus an appended `Status` column |

Migration `000022_migrate_venue_vendor_data` one-time-copied existing tenants' "Venue"-category
`vendors` rows into this table (new fields land empty) and deactivated both the source rows and the
"Venue" `vendor_categories` seed entry — see ADR-0016.

## `projects` (tenant-scoped — the largest surface) — **implemented, Fase 4**; RBAC added later (ADR-0017)

Both "Project Milestone" and "Vendor Milestone" below are displayed in the UI as "Timeline" and
"Timeline Vendor" respectively — a display-text-only rename; every path segment (`milestones`),
field name (`orderedIds`, activity type `milestone_updated`), and response shape stayed as-is.

| Method | Path | Notes |
|---|---|---|
| GET | `/projects` | staff-only; **paginated** (Fase 7); `?search=` matches name + bride/groom name + venue, `?status=` filters exact status; includes `progress` for every row (one batched computation per page, `toProjectResponsesWithProgress`). `?archived=true` switches to the archived-only view — the two are disjoint, never merged (ADR-0013); omitted/`false` shows only non-archived projects. Note `GET /projects?all=true` (the full-roster/dashboard consumer) is **not** archive-filtered — it always includes archived projects, a deliberate scope boundary (see ADR-0013). **Wedding Planner** (`role="Staff"`): both paginated and `all=true` results are silently scoped to only projects where `picStaffId` equals the caller — never a 403, just a narrower result set (see ADR-0017). |
| POST | `/projects` | **Owner/Admin only** (403 for `role="Staff"`) — a Wedding Planner only ever operates within a project already assigned to them, never creates one (ADR-0017) |
| GET | `/projects/me` | **client-only** (Fase 6) — resolves and returns the calling client's own single project (same shape as `GET /projects/{id}`, `progress` included); the Client Portal's entry point, since a client never learns its own project id any other way (mirrors `GET /tenants/me` from Fase 2). Handled as a special `"me"` segment inside the same dispatcher as `/projects/{id}/...`, not a separate route — see implementation note. |
| GET | `/projects/{id}` | includes `progress` (computed on read, not stored). Staff: any project in their tenant, **except** Wedding Planner, who gets 403 on any project they aren't PIC of (ADR-0017, enforced by `resolveProjectAccess` — the single choke point every sub-resource below funnels through). **Client** (Fase 6): only their own project — any other `{id}` is 403, checked server-side via `clients.Contracts.ProjectIDForClient`, not just a frontend convention. |
| PATCH | `/projects/{id}` | body now also accepts `venueId` (tri-state: omitted key = leave current attachment unchanged, `0` = detach, positive id = attach — see `GET .../venue` below), plus `venueRentalPrice`/`venueCharge` (migration `000028`) — a per-project cost **snapshot**, meaningful only alongside a positive `venueId`; force-cleared to `null` server-side on detach regardless of what the body sends, and left untouched whenever `venueId` itself is omitted. Never a live join against `venues.rentalPrice`/`charge` — a project's own cost history must stay
stable even if the venue's master price changes (or the venue is later hard-deleted, ADR-0018)
afterward. **Wedding Planner**: rejected 403 if `picStaffId` in the body differs from the project's current PIC — only Owner/Admin may reassign a project's PIC, even on a project the Wedding Planner already manages day to day (ADR-0017) |
| POST | `/projects/{id}/cancel` | soft status change; only an open project (Completed/Cancelled → **409** `invalid_status_transition`); conditional on the status the request read, so a concurrent change also answers 409 |
| POST | `/projects/{id}/transition` | body `{action}`, one of `start_preparation` (Draft→Preparation), `mark_ready` (Preparation→Ready), `complete` (Ready→Completed), `reopen` (Completed/Cancelled→Preparation; counts toward the trial's active-project cap). Needs `project.edit`. An action that does not fit the current status is refused **409** with code `invalid_status_transition`. Returns the project. **`PATCH /projects/{id}` no longer changes `status`** (ignored), and `POST /projects` and `.../duplicate` always create a Draft — ADR-0027. An unknown `action` is 422 |
| GET | `/projects/{id}/payment-terms` | staff, **and a client with `portal.finance.view`** (403 `forbidden_permission` for a role without it, e.g. Keluarga; `PUT` stays staff-only). The client's payment schedule with each installment's standing, derived from real payments: `[{id, label, amount, dueDate, paid, remaining, status}]`, `status` one of `paid`, `overdue` (due date passed, not fully paid), `partial`, `due_soon` (within 7 days), `upcoming`. Earliest due date first — ADR-0028 |
| PUT | `/projects/{id}/payment-terms` | needs `finance.record`. Body `{terms:[{label, amount, dueDate}]}` **replaces the whole schedule** in one transaction (max 20; label required, amount > 0, valid date). 422 when the installments together exceed the contract value (when one is set). Returns the new schedule with status |
| GET | `/projects/{id}/closing-checklist` | staff only (403 for a client). `{ready, items:[{key,label,done,detail}]}`: `client_paid`, `vendors_paid`, `venue_paid` (only when the project has a venue cost), `issues_closed`. Informs the person completing a project; it never blocks `complete` |
| POST | `/projects/{id}/toggle-archive` | single toggle (flips `isArchived`), no body, no role restriction — see ADR-0013 |
| POST | `/projects/{id}/duplicate` | body: same shape as create (`{name, brideName, ..., description}`) — the new project's own fields, since the frontend pre-fills its normal create/edit form from the source project. Clones the source's Project Milestones and Vendor Engagements (with their Vendor Milestones, including each one's `pricingTier`) as a structural template; does **not** copy payments, issues, evidence, activity log, or clients — see ADR-0014. **Owner/Admin only** (403 for `role="Staff"`, same reasoning as `POST /projects` — duplicating still produces a brand-new project, ADR-0017) |
| DELETE | `/projects/{id}` | **hard delete, Owner-only** (403 for any other staff role) and requires the project already be archived or cancelled (422 otherwise, enforced server-side) — see ADR-0013 for the full cascade scope (every same-module sub-entity table, evidence's object-storage files, and a best-effort cleanup of every client tied to the project) |
| GET | `/projects/{id}/venue` | resolves the project's attached `venueId` into public-safe display data (`{id, name, address, city, capacity, facilities, socialMedia, hasVisibleAttachment}` — no `rentalPrice`/`charge`/PIC contact, ADR-0016). `data: null` (not an error) when no venue is attached. Reachable by both staff and `client` principals — the WO Console's own Project Detail Venue tab instead fetches the *full* record directly from `GET /venues/{id}` (staff-only), since it needs the commercial fields this endpoint deliberately omits |
| GET/POST | `/projects/{id}/milestones` | — |
| PATCH | `/projects/{id}/milestones` | body `{orderedIds}` — full new order of all of the project's milestone IDs; rejected (422) unless it's an exact permutation of the existing set |
| PATCH | `/projects/{id}/milestones/{milestoneId}` | body `{status, targetDate, completedDate}` — full update, mirrors vendor milestones below; `completedDate` is client-supplied, not server-stamped |
| GET/POST | `/projects/{id}/vendors` | project-vendor engagements; create/update body requires `eventDate` (frontend fills it from the parent project's own `eventDate`) and now also `pricingTier` (`"Akad"` \| `"AkadResepsi"`, migration `000025`) — which of the vendor's own two preset prices (`vendors.priceAkad`/`priceAkadResepsi`) the frontend auto-filled `contractValue` from; purely a stored label, not backend-validated against the two-value set (same lax convention as `engagementStatus`) — `contractValue` itself stays whatever was actually submitted, negotiated or not. The response's `paidAmount` (migration `000028`) is **computed**, not a request body field — the handler sums this engagement's own `vendor_payments` rows (`Refund` netted as a subtraction) fresh on every read; sending `paidAmount` in a create/update body has no effect |
| PATCH | `/projects/{id}/vendors/{projectVendorId}` | full-body update, same shape as create (including `pricingTier`) |
| POST | `/projects/{id}/vendors/{projectVendorId}/cancel` | one-way; "un-cancelling" is just editing `engagementStatus` back via PATCH |
| GET/POST | `/projects/{id}/vendors/{projectVendorId}/milestones` | vendor milestones |
| PATCH | `/projects/{id}/vendors/{projectVendorId}/milestones/{id}` | full-body update (`status`, `targetDate`, `completedDate`, `picStaffId`, `description`, `notes`) — no partial-field PATCH |
| GET/POST | `/projects/{id}/payments` | vendor payments (money **out**). `method` is one of `Tunai`/`Transfer Bank`/`QRIS` (frontend-constrained, not backend-validated); `referenceNumber` is optional. Response's `evidenceComplete` (migration `000029`) is **computed**, cross-referencing `evidence` rows with `relatedKind: "payment"` for this payment's id, split by `type` (`Invoice` vs `Transfer Proof`) — a `Refund` needs only a Transfer Proof, every other type needs both. "Tambah Pembayaran" attaches both file slots (via two separate `POST .../evidence` calls, `relatedKind: "payment"`) in the same submit that records the payment — finally closing what was previously an unreachable "always incomplete in practice" state (the columns this used to read from were never written by any code path) |
| PATCH | `/projects/{id}/payments/{paymentId}` | full-field overwrite (`type`, `amount`, `paymentDate`, `method`, `referenceNumber`, `notes`) — **no role gate**, same convention as every other Update in this module; `projectVendorId` is not editable (see ADR-0019) |
| DELETE | `/projects/{id}/payments/{paymentId}` | hard delete — **Owner-or-Admin only** (403 for `role="Staff"`, checked inline, not via `vendors`/`venues`' `requireOwnerRole` helper). Cascades to this payment's own `evidence` rows (deleted first, then the payment row) and their S3 objects, best-effort — see ADR-0019 |
| GET/POST | `/projects/{id}/client-payments` | client payments (money **in** — "Uang Masuk dari Client"), no `projectVendorId`. `method`/`referenceNumber` follow the same fixed-dropdown/optional rules as `/payments` above. Response includes `evidenceComplete`, computed by the handler as a plain existence check against `evidence` (`relatedKind = "clientPayment"`) — a single slot, since there's no "invoice" concept for money coming in (unlike `vendor_payments`, which needs both an Invoice and a Transfer Proof) |
| PATCH/DELETE | `/projects/{id}/client-payments/{paymentId}` | same shape/role-gate as `/payments/{paymentId}` above — see ADR-0019. **Overpayment (ADR-0027), all three payment ledgers:** a `DP`/`Termin`/`Pelunasan` that would take what is paid past what is owed (client: project `contractValue`; vendor: that engagement's `contractValue`; venue: rental price + charge) is refused **422** with the explanation in `message` and `errors.amount`. The rule judges the change: refused only when it exceeds what is owed **and** raises the paid total, so lowering/deleting a payment and editing a reference or date stay allowed, while deleting or shrinking a `Refund` is checked like adding a payment. Amounts must be 1…10^15 (422 otherwise). `DELETE` on a payment can therefore also answer 422. `Tambahan` and `Refund` are never limited; `Tambahan` does not count against the contract; editing a payment leaves itself out of the sum; a contract of 0 (not set) is not limited |
| GET/POST | `/projects/{id}/venue-payments` | venue payments (money **out** to the project's venue), no `projectVendorId`-equivalent (a project has at most one venue). `method`/`referenceNumber` follow the same fixed-dropdown/optional rules as `/payments` above. Response's `evidenceComplete` is computed the same two-slot way as `/payments` (`relatedKind = "venuePayment"`, distinct from `"payment"` so `vendor_payments.id`/`venue_payments.id` — independent `AUTO_INCREMENT` sequences that can collide in value — are never ambiguous) |
| PATCH/DELETE | `/projects/{id}/venue-payments/{paymentId}` | same shape/role-gate as `/payments/{paymentId}` above — see ADR-0019 |
| GET/POST | `/projects/{id}/issues` | |
| PATCH | `/projects/{id}/issues/{issueId}` | body **penuh** (`projectVendorId, vendorMilestoneId?, title, description, impact, foundDate, status, resolutionPlan, picStaffId, targetResolutionDate`), bukan hanya `{status}` (PATCH parsial direncanakan di `docs/plan/revamp-ui-shadcn/PLAN.md` E9); auto-stamps `resolvedDate` exactly once when status becomes `Resolved`/`Closed` |
| GET | `/projects/{id}/evidence` | returns every evidence row for the project directly (not derived client-side from milestones/payments/issues) |
| POST | `/projects/{id}/evidence` | JSON body `{name, type, fileName, mimeType, base64Data, documentDate, description, relatedKind, relatedId, isClientVisible}` — base64, not multipart (see ADR-0006, ADR-0010); 15 MB decoded-size cap; `relatedKind` is one of `vendorMilestone`/`payment`/`projectVendor`/`issue`/`clientPayment`/`venuePayment`/`general` (migration `000035`, ADR-0020); `isClientVisible` is server-side re-normalized to `false` unless `relatedKind = "general"` (never trusted blindly from the request body, even for a `general` document it still defaults `false`) |
| POST | `/projects/{id}/evidence/{evidenceId}/toggle-client-visible` | flips `is_client_visible` for one evidence row — rejected `422` unless `relatedKind = "general"` (visibility is meaningless for every other kind, see `DB_SCHEMA.md` and ADR-0020) |
| GET | `/projects/{id}/documents` | Client Portal's own "Dokumen" tab feed (ADR-0020) — **always** only `relatedKind = "general"` **and** `isClientVisible = true`, unconditionally regardless of caller (reachable by staff too, harmlessly — the safety property is what the endpoint returns, not who can call it). Deliberately its own query/endpoint rather than a filter bolted onto `GET .../evidence`, so "only ever returns documents safe to show a client" stays trivially auditable |
| GET | `/projects/{id}/evidence/{evidenceId}/link` | **Fase 7.** `{url, expiresAt}`: a direct, two-minute link to the stored file in object storage, so the bytes do not pass through the API. Same project scope and client-visibility rule as `/file` (an internal document is 404 for a client). `?disposition=attachment` makes the link force a download. `501` with `errors.code = presign_unavailable` when the deployment has `S3_PRESIGN=false`; the web then falls back to `/file`. Allowed for client principals |
| GET | `/projects/{id}/evidence/{evidenceId}/file` | authenticated download (Bearer token required — not a public URL); streams the object exactly as stored (already compressed at upload time); frontend fetches it as a blob via `httpClient`, not a bare `<img src>` |
| GET | `/projects/{id}/activity` | append-only, populated server-side on every mutation above; activity types actually emitted: `project_created`, `project_updated`, `project_status_changed`, `vendor_added`, `vendor_status_changed`, `milestone_updated`, `payment_recorded`/`payment_updated`/`payment_deleted` (each shared by vendor, client, and venue payments, distinguished only by `entityType`: `"vendor_payment"` vs. `"client_payment"` vs. `"venue_payment"`), `evidence_uploaded`, `issue_created`, `issue_updated`; response includes `projectId` (needed by the dashboard's cross-project activity feed, not by the per-project tab which already knows it from the URL) |

## `milestone-templates` (tenant-scoped, "Pengaturan → Timeline Default") — every route Owner-only

Per-tenant configurable checklist auto-seeded into a new project's own Timeline tab (`project_milestone_templates`
table) — replaces a single hardcoded 6-item list shared by every tenant. Not routed through the
`/projects/{id}/...` dispatcher — a standalone tenant-level resource, un-paginated (expected to stay
a short list). See `DB_SCHEMA.md`.

| Method | Path | Notes |
|---|---|---|
| GET | `/milestone-templates` | Owner-only — `[{id, name, daysBeforeEvent, sortOrder}]` |
| POST | `/milestone-templates` | Owner-only. Body `{name, daysBeforeEvent}` — `daysBeforeEvent` can be **negative** ("Setelah Acara"/H+, e.g. `-3` for 3 days after the event); the frontend form represents this as a Sebelum Acara/Setelah Acara toggle + non-negative day count, converting to a signed integer only at the form boundary — this API body is unchanged |
| PATCH | `/milestone-templates` | Owner-only — reorder. Body `{orderedIds}`, rejected (422) unless an exact permutation of the tenant's current template IDs |
| PATCH | `/milestone-templates/{id}` | Owner-only — update `{name, daysBeforeEvent}`, same H+ support as `POST` above |
| DELETE | `/milestone-templates/{id}` | Owner-only — **hard delete** (one of five deliberate hard-delete exceptions in this codebase, see `knowledge/DATABASE_GUIDE.md` and ADR-0018): a template row is only ever copied into a project's own `project_milestones` at creation time, never referenced again afterward, so there's nothing to orphan |

A newly registered tenant gets the same starter 6-item template via `platform`'s tenant-registration
flow (`SeedDefaultMilestoneTemplate`, called the same moment `vendors.SeedDefaultCategories` is) —
already-registered tenants were backfilled directly by migration `000024`.

## WO Console dashboard — **implemented, Fase 5**

| Method | Path | Notes |
|---|---|---|
| GET | `/dashboard` | staff-only; single aggregate payload — `totalProjects`, `activeProjects`, `activeVendorCount`, `openIssues[]`, `overdueVendorMilestones[]`, `incompletePayments[]`, `incompleteVenuePayments[]`, `nearDDayProjects[]`, `laggingProjects[]` (project + computed `overallPercent`), `upcomingProjects[]`, `recentActivity[]` (8 most recent, across all tenant projects), `revenue` (`{total, previousTotal, deltaPercent}`), `projectTrend[]` and `revenueTrend[]` (12 trailing months). Every array field is always `[]`, never `null`, when empty. Issue/milestone/payment rows carry `vendorId` only (no vendor name) — the frontend resolves display names from the already-fetched `vendors` store, the same pattern used throughout the `projects` module since Fase 4. `incompleteVenuePayments[]` has no `vendorId` (a venue payment isn't tied to any vendor) — kept as its own array rather than merged into `incompletePayments[]`; the frontend merges the two into one labeled feed at render time (`dashboard/lib/attention.ts`). | **Fase 4c additions:** `overdueTerms[]` + `overdueTermsTotal` (unpaid client installments past their due date, with the Rp still owed), `dueSoonTerms[]` (unpaid, due within 30 days), `tasksDueThisWeek[]` (open project Timeline items due in the next 7 days), `stageCounts{draft,preparation,ready}` — computed from `client_payment_terms` against real payments (ADR-0028).
| GET | `/dashboard/mine` | **any staff role** (scoped to projects where the caller is PIC or PIC Sales; clients 403). `[{id, name, brideName, groomName, eventDate, status, progressPercent, openIssues, overdueTerms, overdueTotal, nextTermLabel?, nextTermDueDate?, nextTermRemaining?, tasksDueThisWeek[]}]`, open projects only, soonest event first, max 30. The landing page for roles limited to their own projects, which cannot have the tenant-wide `/dashboard` — ADR-0029 |

No `GET /search?q=` endpoint exists or is planned — `GlobalSearch.tsx` calls the existing
`GET /projects`, `GET /vendors`, and `GET /clients` (no `projectId` — see below) endpoints directly
and filters client-side, since all three datasets are already small enough to fetch in full.

Platform dashboard has **no new endpoint** (Fase 5) — its data (`GET /tenants` +
`GET /subscription-transactions`) was already real since Fase 2; only the frontend's date reference
changed (real `todayISO()` instead of the mock's hardcoded `TODAY`).

## Client Portal (read-scoped subset — same `projects` endpoints, `client` principal) — **implemented, Fase 6**

`client` principals hit the exact same `/projects/{id}/...` GET endpoints as staff (milestones,
vendor engagements + their milestones, payments, client-payments, venue-payments, issues, evidence +
file download, activity, `documents`, and `venue` — the public-safe summary, ADR-0016) — there is no
separate client-portal endpoint set. The Pembayaran tab specifically now reads `client-payments` (the
client's own payment history), not `payments`/`venue-payments` (the WO's spend on vendors/venue) —
"Uang Masuk dari Client" is a distinct concept from money spent on vendors/venue. `venue-payments`
is reachable by a `client` principal via the same blanket GET allowance as `payments` above, but
nothing in the Client Portal UI calls it today. The Client Portal's own "Dokumen" tab (appended after
Kendala, ADR-0020) reads `GET .../documents` specifically — never `GET .../evidence` directly, which
stays unfiltered and would leak every other evidence kind if the Client Portal ever called it (see
`DB_SCHEMA.md`'s `evidence.is_client_visible` note). The
one exception is `venues/{id}/attachment`, a `venues`-module route (not under `/projects/{id}/...`)
that `client` principals also reach, gated by mime type (image only) rather than by project
ownership, since it isn't itself project-scoped. Enforcement, checked server-side on every request
(not just a frontend routing convention):
- Only `GET` is ever allowed for a `client` principal — any `POST`/`PATCH` is rejected 403 before
  reaching any business logic, even for the client's own project.
- `{id}` must equal the one project resolved for that client via `clients.Contracts.ProjectIDForClient`
  (new in Fase 6) — any other `{id}` (including a real project belonging to a *different* client in
  the same tenant) is rejected 403.
- The client's own project id is learned via `GET /projects/me` (see above) — there is no other way,
  by design (login doesn't return one; `identity` stays profile-agnostic per ADR-0005).

A deactivated client (`clients.is_active = false`) is also rejected 403, even for their own project.
