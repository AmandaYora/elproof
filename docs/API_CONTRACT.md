# API Contract — ElProof

> Endpoint list and request/response formats. Conventions: `.claude/rules/api-standard.md` +
> [`knowledge/API_GUIDE.md`](../knowledge/API_GUIDE.md). All paths are under `/api/v1`. Auth header
> required on every endpoint except login/refresh (see API_GUIDE.md).

Envelope reminder — success: `{ "success": true, "message": "...", "data": ... }`; error:
`{ "success": false, "message": "...", "errors": {...} }`; paginated lists additionally return
`meta: { page, limit, total, total_pages }` (plus `sort: "field:dir"` on the lists that accept `?sort=`, see
**Tahap 6** below).

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
| GET | `/tenants` | **paginated** (Fase 7); `?search=` matches business/owner name + email; `?status=` filters `subscriptionStatus`; `?planId=` and `?source=self_service|platform` (Fase 6); `?sort=` (Tahap 6) |
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
| GET | `/audit-log` | **Super Admin**, paginated; filters `actorType`, `actorId`, `tenantId`, `outcome=ok|failed`, `action` (exact label, Tahap 6), `q` or `search` (substring of action or path), `from`, `to` (WIB days); `?sort=` — see Tahap 6 |
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
| GET | `/subscription-transactions` | platform_admin: all tenants (optional `?tenantId=`); staff: forced to own tenant from JWT regardless of query; **paginated** (Fase 7), `?status=` filters transaction status (now includes `pending`, added Fase 9). **Tahap 6 (E14):** rows carry `tenantName`; `?search=` (or `q`) matches the payment reference or the tenant's name, `?from=`/`?to=` bound `createdAt` (WIB days, inclusive), `?sort=` — see Tahap 6 |
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
| GET | `/staff` | Owner-only; **paginated** (Fase 7); `?search=` matches name + email; `?role=` filters exact role; response includes `username`. **Tahap 6:** `?roleId=` (same filter), `?isActive=`, `?sort=`, items carry `roleName` |
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
| GET | `/clients` | tenant-scoped; `?projectId=` filters to one project (unpaginated — small, bounded per-project list). Omit `projectId` to list every client for the tenant instead (added Fase 5, powers `GlobalSearch` via `?all=true`) — this tenant-wide mode is **paginated** (Fase 7), `?search=` matches name + email; response includes `username` **Tahap 6 (E11):** `?page&limit&sort&q&role&loginStatus&projectId` — the flat paginated list with `projectName`/`projectEventDate`; `?all=true` and `?projectId=` alone keep their shapes (see *Tahap 6* below). |
| POST | `/clients` | body `{projectId, role, relationNote, name, phone, email, username, password}` — validates `projectId` via `projects.Contracts.ProjectExists`, provisions a real login credential in the same call; if `identity.CreateCredential` fails (e.g. username or email already taken by *any* principal on the platform — neither is tenant-scoped), the just-inserted `clients` row is rolled back (deleted) rather than left orphaned with no working credential |
| PATCH | `/clients/{id}` | contact edit (`{name, phone, email}`) — no `username` (immutable after creation) |
| POST | `/clients/{id}/toggle-active` | |
| POST | `/clients/{id}/reset-credential` | body `{password}` — sets a new password and stamps `lastCredentialResetAt`. Fails with `404`/"Kredensial tidak ditemukan" for a client left behind by a pre-rollback-fix `Create` failure (no `identity` row ever existed) — use `DELETE` below to clear those out, not this endpoint |
| POST | `/clients/{id}/replace-representative` | body `{name, phone, email, relationNote}`; only valid when `role=Family Representative`; overwrites the same row, no history kept (§6.3) |
| DELETE | `/clients/{id}` | **Tahap 5 (E13, ADR-0034): now a soft delete** — answers 200 `data: {id, restorableUntil}` (RFC 3339, now + 24 h); the client leaves every list, count and the project's client slots at once and their login is switched off (kept, deactivated; username and email stay reserved until the purge). `POST /clients/{id}/restore` brings it back (see "Tahap 5" below). Same gate as every client write (`client.manage`, own projects for a limited role). *Before Tahap 5 this was the module's one hard delete:* the description that follows is the old behaviour, now performed by the purge once a deleted client is older than 24 h — hard delete — the one permanent-delete endpoint in this module (every other mutation elsewhere is a soft toggle). Best-effort deactivates the client's `identity` credential first (harmless no-op if none exists); exists specifically so an operator can self-service clear a client stuck with no working credential, since deactivating alone doesn't free up that client's role slot on the project (the role lookup doesn't filter on `is_active`) |

## `vendors` (tenant-scoped) — **implemented, Fase 3** (Venue directory added ADR-0016; RBAC below added later — see ADR-0017)

**Role gate, applies uniformly across `vendors`/`venues` below (`requireManagerRole`):** reading
(list/get/attachment-download) stays open to any staff role, including Wedding Planner (`"Staff"`),
so the vendor/venue pickers inside their own project keep working — but every write
(create/update/toggle-active/attachment-upload/template/import) is Owner/Admin only.

| Method | Path | Notes |
|---|---|---|
| GET | `/vendor-categories` | tenant-scoped, open to any staff role; **paginated** (Fase 7) view is Owner-only, `?all=true` (picker/display use) stays open to any role; `?search=` matches category name **Staff only: a `client` principal gets 403** (closed by E6 — it listed every category of the workspace; the portal reads `categoryName` inside `GET /projects/{id}/vendors` instead, ADR-0032) **Tahap 6:** `?sort=name|createdAt`. |
| POST | `/vendor-categories` | Owner-only |
| PATCH | `/vendor-categories/{id}` | Owner-only |
| POST | `/vendor-categories/{id}/toggle-active` | Owner-only |
| DELETE | `/vendor-categories/{id}` | Owner-only — **hard delete, no `delete-impact` counterpart** (blocked, not informed-consent): `409` if any vendor still references it (`vendors.category_id` is a real, same-module, `NOT NULL` FK) with a count in the message; see ADR-0018 |
| GET | `/vendors` | `?categoryId=`, `?city=` (exact match against the fixed 128-entry city enum) filters; **paginated** (Fase 7); `?search=` matches name + PIC + email. **Tahap 6:** `?isActive=true\|false`, `?sort=`, items carry `categoryName` |
| POST | `/vendors/bulk` | **Tahap 6 (E13).** body `{ids[1–100], action: activate\|deactivate}`, needs `vendor.manage` (the toggle's permission); one transaction, ids of other workspaces → `notFound`; answers `{updated, unchanged[], notFound[]}`. Details under *Tahap 6* below |
| POST | `/vendors` | Owner/Admin only. Body now also accepts `socialMedia`, `city`, `priceAkad`, `priceAkadResepsi` (migration `000023`) — `email`/`address` are no longer mandatory; validates `categoryId` belongs to the same tenant (422 if not) |
| PATCH | `/vendors/{id}` | Owner/Admin only; same body/`categoryId` validation as create |
| POST | `/vendors/{id}/toggle-active` | Owner/Admin only |
| GET | `/vendors/{id}/delete-impact` | Owner-only — `{affectedProjects: [{id, name}, ...]}`, every project whose `project_vendors.vendor_id` points at this vendor; called by the frontend the instant "Hapus" is clicked, before `DELETE` — see ADR-0018 |
| DELETE | `/vendors/{id}` | Owner-only — hard delete, no precondition (informed-consent model, not block-then-cascade — ADR-0018); best-effort deletes the object-storage attachment too, log-and-continue on failure |
| PUT | `/vendors/{id}/attachment` | Owner/Admin only. Body `{fileName, mimeType, base64Data}` — same base64-JSON shape as evidence/logo upload; single slot (document or photo), replaces any existing attachment |
| GET | `/vendors/{id}/attachment` | any staff role — streams the stored bytes back, `Content-Disposition: inline` |
| GET | `/vendors/{id}/project-history` | Owner/Admin only — backs "Lihat Project," the vendor's engagement history across every project in the tenant, `[{projectId, projectName, eventDate, venue, engagementStatus}]` ordered by `eventDate DESC`. `project_vendors` is owned by `projects`, not `vendors` — resolved through `projects.Contracts.ListVendorEngagementHistory` (`vendors.NewModule` now takes `projects.Contracts` as a constructor arg; `main.go` builds `projects` before `vendors` accordingly). A Wedding Planner has no legitimate need to see a vendor's history across the whole tenant, only their own project's own engagement (lives in `projects` itself) |
| GET | `/vendors/summary` | any staff role — `[{id, name}]`, **includes inactive vendors** (unlike every other `?all=true` listing in this module) so a project already engaging a since-deactivated vendor can still resolve its name. Backs Client Portal's Vendor tab (which also pins the project's venue as its first card, client-facing presentation only — ADR-0016's Revision) and any picker that only ever needs a name **Staff only: a `client` principal gets 403** (closed by E6 — it returned the names of *every* vendor of the workspace; the portal now reads `vendorName` inside `GET /projects/{id}/vendors` and `/issues`, for the project's own vendors only, ADR-0032) |
| GET | `/vendors/template` | Owner/Admin only — downloads an `.xlsx` with the fixed header row `Nama Vendor, Kategori, Nama PIC, No Tlp Vendor, Email, Sosial Media, Kota, Alamat, Harga Akad, Harga Akad+Resepsi, Catatan`, styled (frozen header row, sized columns, thousand-separator number format on the two price columns) and with an in-file dropdown on Kategori (the tenant's own registered categories) and Kota (the fixed 128-entry enum), sourced from a hidden helper sheet since both lists exceed Excel's inline dropdown-formula character cap |
| POST | `/vendors/import` | Owner/Admin only. Body `{base64Data}` (an `.xlsx` matching the template's column order) → partial-success import, `{insertedCount, updatedCount, errors: [{row, message}]}` — invalid rows are reported back by row number, valid rows still commit; capped at 1000 data rows; a price cell formatted with thousand separators (e.g. from the styled template above) is parsed correctly (separators stripped before `ParseInt`) |
| GET | `/vendors/export` | Owner/Admin only — same `?categoryId=`/`?city=`/`?search=` filters as `GET /vendors` (unpaginated, exports the matching set); column order matches `/vendors/template` exactly plus an appended `Status` column, so the file round-trips back through `/vendors/import` unmodified |

## `venues` (tenant-scoped) — **implemented, ADR-0016**

Its own directory inside the `vendors` module (not a new top-level Go module) — see ADR-0016. Same
`requireManagerRole` read/write split as `vendors` above, with one exception: the attachment
download route is reachable by a `client` principal too (see below).

| Method | Path | Notes |
|---|---|---|
| GET | `/venues` | any staff role; `?city=` (exact match against the fixed 128-entry city enum) filter; **paginated** (Fase 7); `?search=`. **Tahap 6:** `?isActive=true\|false`, `?sort=` |
| POST | `/venues/bulk` | **Tahap 6 (E13).** same as `/vendors/bulk` for venues (`vendor.manage`) |
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
| GET | `/projects` | staff-only; **paginated** (Fase 7); `?search=` matches name + bride/groom name + venue, `?status=` filters exact status; includes `progress` for every row (one batched computation per page, `toProjectResponsesWithProgress`). `?archived=true` switches to the archived-only view — the two are disjoint, never merged (ADR-0013); omitted/`false` shows only non-archived projects. Note `GET /projects?all=true` (the full-roster/dashboard consumer) is **not** archive-filtered — it always includes archived projects, a deliberate scope boundary (see ADR-0013). **Wedding Planner** (`role="Staff"`): both paginated and `all=true` results are silently scoped to only projects where `picStaffId` equals the caller — never a 403, just a narrower result set (see ADR-0017). **Tahap 6 (E11):** `?picStaffId=`, `?eventFrom=`, `?eventTo=`, `?sort=`; items carry `remaining` and `nextDueDate` (see *Tahap 6* below). |
| POST | `/projects` | **Owner/Admin only** (403 for `role="Staff"`) — a Wedding Planner only ever operates within a project already assigned to them, never creates one (ADR-0017). **E7 (ADR-0033): one call creates the project, its default milestones, its venue and its payment schedule in one transaction** — see "Tahap 4" below for the extra body fields (`venueId`, `venueRentalPrice`, `venueCharge`, `terms`), the extra response fields (`terms`, `venueSummary`), the all-errors-at-once 422, and the optional `Idempotency-Key` header. A date that is not `YYYY-MM-DD` is now reported under its own key (`eventDate` / `prepStartDate`; before, any bad date was reported as `eventDate`) on create, update and duplicate |
| GET | `/projects/me` | **client-only** (Fase 6) — resolves and returns the calling client's own single project (same shape as `GET /projects/{id}`, `progress` included); the Client Portal's entry point, since a client never learns its own project id any other way (mirrors `GET /tenants/me` from Fase 2). Handled as a special `"me"` segment inside the same dispatcher as `/projects/{id}/...`, not a separate route — see implementation note. |
| GET | `/projects/me/home` | **client-only, GET** (E5, ADR-0032) — the whole Client Portal "Beranda" in one call (replaces ~9). Same project resolution and the same 403/404 answers as `GET /projects/me` (403 for staff, 403 for a client with no project). Each section exists only when the client's portal role may see it (**absent, never `null`**) and nothing is computed for a section the role cannot see. `data`: `project {id, name, brideName, groomName, eventDate (YYYY-MM-DD), venue, packageName, status, picName, picPhone}` (client redaction, **no** `contractValue`), `countdown {daysToEvent}` (event date minus today in Asia/Jakarta, negative after the event), `progress {overallPercent, condition}` (`portal.timeline.view`), `finance {contractValue, paid, remaining, nextTerm {id, label, amount, dueDate, status, remaining} \| null, overdueCount}` (`portal.finance.view`; `paid` = DP+Termin+Pelunasan−Refund, `remaining` = contract − paid and never negative, `nextTerm` = earliest installment that is not paid), `payTo {bankName, accountNumber, accountHolder}` (`portal.finance.view` **and** the WO filled in bank name + number in *Data usaha*), `contact {whatsapp}` (the WO's number from `tenants/me/contact`, absent when empty), `actions [{kind: "pay_term"\|"open_issues", title, detail, dueDate?, amount?, target: "pembayaran"\|"kendala"}]` (always an array, max 3, most urgent first: installments past due oldest first, then installments due within 7 days, then the open-issues count; `pay_term` needs finance and a project that is not cancelled, `open_issues` needs `portal.issues.view`; `dueDate`/`amount` only on `pay_term`, `amount` = what is still owed on that installment), `openIssueCount` (`portal.issues.view`; `0` is sent as `0`), `recentUpdates [{kind: "milestone_done"\|"payment_verified"\|"issue_resolved", title, at (RFC 3339, Asia/Jakarta), proof {evidenceId, type, fileName} \| null, verifiedBy?}]` (always an array, newest first, max 5; sources: completed steps (timeline permission; a step's date has no time of day, so it counts as the start of that day), verified payments (finance; `verifiedBy` = verifier's name, `proof` = the payment's earliest file), resolved issues (issues permission); `proof` is the earliest file of that item that the client may see, `null` otherwise), `pic {name, phone}`. Never contains the Sales PIC, venue commercial terms, the booking date, vendor money, staff ids, `recordedBy` or internal notes |
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
| GET/POST | `/projects/{id}/milestones` | Each item carries `evidence: [{id, type, fileName, uploadedAt}]` (E4, **GET only**; `[]` on create/update responses): all of the step's attachments for staff, and for a **client** only those with `portal.timeline.view` **and** `isClientVisible = true` (staff flag an attachment with `isClientVisible` on upload or the existing `toggle-client-visible`; an unflagged attachment is invisible and its `/file` and `/link` answer 404 — ADR-0032) |
| PATCH | `/projects/{id}/milestones` | body `{orderedIds}` — full new order of all of the project's milestone IDs; rejected (422) unless it's an exact permutation of the existing set |
| PATCH | `/projects/{id}/milestones/{milestoneId}` | body `{status, targetDate, completedDate}` — full update, mirrors vendor milestones below; `completedDate` is client-supplied, not server-stamped |
| GET/POST | `/projects/{id}/vendors` | **E7 (POST):** `categoryId` and `eventDate` are derived on the server when omitted (the vendor's category, the project's event date); if sent they must equal them (422 on `categoryId` / `eventDate`, all problems at once; `vendorId` is required and must be this workspace's). PATCH is unchanged. Older notes: project-vendor engagements; create/update body requires `eventDate` (frontend fills it from the parent project's own `eventDate`) and now also `pricingTier` (`"Akad"` \| `"AkadResepsi"`, migration `000025`) — which of the vendor's own two preset prices (`vendors.priceAkad`/`priceAkadResepsi`) the frontend auto-filled `contractValue` from; purely a stored label, not backend-validated against the two-value set (same lax convention as `engagementStatus`) — `contractValue` itself stays whatever was actually submitted, negotiated or not. The response's `paidAmount` (migration `000028`) is **computed**, not a request body field — the handler sums this engagement's own `vendor_payments` rows (`Refund` netted as a subtraction) fresh on every read; sending `paidAmount` in a create/update body has no effect **GET only, E6/E4:** each item also carries `vendorName` and `categoryName` (resolved inside the caller's workspace for this project's own engagements only; `""` when unresolved and on create/update responses), and each embedded milestone carries `evidence: [{id, type, fileName, uploadedAt}]` (client: only what `portal.vendor.view` allows). Client redaction of money/notes is unchanged |
| PATCH | `/projects/{id}/vendors/{projectVendorId}` | full-body update, same shape as create (including `pricingTier`) |
| POST | `/projects/{id}/vendors/{projectVendorId}/cancel` | one-way; "un-cancelling" is just editing `engagementStatus` back via PATCH |
| DELETE | `/projects/{id}/vendors/{projectVendorId}` | **Tahap 5 (E13):** soft delete (needs `project.edit`) — the engagement and its milestones, payments, kendala and their files disappear everywhere; 200 `{id, restorableUntil, projectProgress}`. `POST …/restore` brings everything back — see "Tahap 5" |
| GET/POST | `/projects/{id}/vendors/{projectVendorId}/milestones` | vendor milestones |
| PATCH | `/projects/{id}/vendors/{projectVendorId}/milestones/{id}` | full-body update (`status`, `targetDate`, `completedDate`, `picStaffId`, `description`, `notes`) — no partial-field PATCH |
| GET/POST | `/projects/{id}/payments` | vendor payments (money **out**). `method` is one of `Tunai`/`Transfer Bank`/`QRIS` (frontend-constrained, not backend-validated); `referenceNumber` is optional. Response's `evidenceComplete` (migration `000029`) is **computed**, cross-referencing `evidence` rows with `relatedKind: "payment"` for this payment's id, split by `type` (`Invoice` vs `Transfer Proof`) — a `Refund` needs only a Transfer Proof, every other type needs both. "Tambah Pembayaran" attaches both file slots (via two separate `POST .../evidence` calls, `relatedKind: "payment"`) in the same submit that records the payment — finally closing what was previously an unreachable "always incomplete in practice" state (the columns this used to read from were never written by any code path) **Tahap 4 (E3, ADR-0033):** POST also accepts `evidence: [{uploadToken, type, name?}]` and the header `Idempotency-Key`, records the payment and its proof in one transaction and answers with the payment plus `ledgerSummary` — see "Tahap 4" below. |
| PATCH | `/projects/{id}/payments/{paymentId}` | full-field overwrite (`type`, `amount`, `paymentDate`, `method`, `referenceNumber`, `notes`) — **no role gate**, same convention as every other Update in this module; `projectVendorId` is not editable (see ADR-0019) |
| DELETE | `/projects/{id}/payments/{paymentId}` | **Tahap 5 (E13): now a soft delete** — 200 `data: {id, restorableUntil, ledgerSummary, projectProgress}`, `POST …/{paymentId}/restore` undoes it; needs `finance.payment.delete` (see "Tahap 5"). *The rest of this row describes the hard delete it was before; evidence is no longer deleted at once — it is hidden with the payment and removed by the purge.* Hard delete — **Owner-or-Admin only** (403 for `role="Staff"`, checked inline, not via `vendors`/`venues`' `requireOwnerRole` helper). Cascades to this payment's own `evidence` rows (deleted first, then the payment row) and their S3 objects, best-effort — see ADR-0019 |
| GET/POST | `/projects/{id}/client-payments` | client payments (money **in** — "Uang Masuk dari Client"), no `projectVendorId`. `method`/`referenceNumber` follow the same fixed-dropdown/optional rules as `/payments` above. Response includes `evidenceComplete`, computed by the handler as a plain existence check against `evidence` (`relatedKind = "clientPayment"`) — a single slot, since there's no "invoice" concept for money coming in (unlike `vendor_payments`, which needs both an Invoice and a Transfer Proof) **E4 — every row (list/create/update/verify) also carries:** `recordedBy: {name} \| null` (internal: **always `null` for a client**; null when unset or the staff row is gone), `verifiedAt: RFC 3339 \| null`, `verifiedBy: {name} \| null`, `evidence: [{id, type, fileName, uploadedAt}]` (oldest first; a client gets only what `portal.finance.view` allows, `[]` otherwise). `recordedBy` is set to the staff actor on create (migration `000065`; older rows are `null`). Existing fields, including `evidenceComplete`, are unchanged **Tahap 4 (E3, ADR-0033):** POST also accepts `evidence: [{uploadToken, type, name?}]` and the header `Idempotency-Key`, records the payment and its proof in one transaction and answers with the payment plus `ledgerSummary` — see "Tahap 4" below. |
| PATCH/DELETE | `/projects/{id}/client-payments/{paymentId}` | **Tahap 5:** `PATCH` also answers `ledgerSummary` + `projectProgress`; `DELETE` is a soft delete (200 `{id, restorableUntil, ledgerSummary, projectProgress}`) with `POST …/{paymentId}/restore`; a verified payment is deletable like any other — see "Tahap 5". Same shape/role-gate as `/payments/{paymentId}` above — see ADR-0019. **Overpayment (ADR-0027), all three payment ledgers:** a `DP`/`Termin`/`Pelunasan` that would take what is paid past what is owed (client: project `contractValue`; vendor: that engagement's `contractValue`; venue: rental price + charge) is refused **422** with the explanation in `message` and `errors.amount`. The rule judges the change: refused only when it exceeds what is owed **and** raises the paid total, so lowering/deleting a payment and editing a reference or date stay allowed, while deleting or shrinking a `Refund` is checked like adding a payment. Amounts must be 1…10^15 (422 otherwise). `DELETE` on a payment can therefore also answer 422. `Tambahan` and `Refund` are never limited; `Tambahan` does not count against the contract; editing a payment leaves itself out of the sum; a contract of 0 (not set) is not limited **Verification (ADR-0032):** a PATCH that changes `type`, `amount`, `paymentDate`, `method` or `referenceNumber` clears `verifiedAt`/`verifiedBy` (and the activity log says so); changing only `notes` keeps it |
| POST | `/projects/{id}/client-payments/{paymentId}/verify` | **E4, staff only, needs `finance.record`** (a `client` principal gets 403 — the portal is GET-only; another workspace's staff gets 404). No body. Sets `verifiedAt = now` and `verifiedBy = the caller`; returns the full client-payment row (200). **Idempotent:** an already verified payment is returned unchanged (same person, same time). **422** with `message: "Lampirkan bukti transfer sebelum memverifikasi pembayaran."` and `errors.evidence` while the payment has no evidence file (`relatedKind = "clientPayment"`). Writes a `payment_updated` activity entry ("Pembayaran client diverifikasi"). A verified payment always has at least one file: removing its last file clears the verification (`EvidenceService.Delete`; there is no HTTP route for deleting a single file yet). There is no un-verify endpoint — edit the payment instead |
| GET/POST | `/projects/{id}/venue-payments` | venue payments (money **out** to the project's venue), no `projectVendorId`-equivalent (a project has at most one venue). `method`/`referenceNumber` follow the same fixed-dropdown/optional rules as `/payments` above. Response's `evidenceComplete` is computed the same two-slot way as `/payments` (`relatedKind = "venuePayment"`, distinct from `"payment"` so `vendor_payments.id`/`venue_payments.id` — independent `AUTO_INCREMENT` sequences that can collide in value — are never ambiguous) **Tahap 4 (E3, ADR-0033):** POST also accepts `evidence: [{uploadToken, type, name?}]` and the header `Idempotency-Key`, records the payment and its proof in one transaction and answers with the payment plus `ledgerSummary` — see "Tahap 4" below. |
| PATCH/DELETE | `/projects/{id}/venue-payments/{paymentId}` | same shape/role-gate as `/payments/{paymentId}` above — see ADR-0019. **Tahap 5:** `PATCH` also answers `ledgerSummary` + `projectProgress`; `DELETE` is a soft delete with `POST …/{paymentId}/restore` — see "Tahap 5" |
| GET/POST | `/projects/{id}/issues` | **Tahap 5 (E13, E12):** `POST` accepts `projectVendorId` as optional and `isClientVisible` (default `true`); every item carries `projectVendorId` (`null` without a vendor), `isClientVisible`, `picName`; **a client lists only `isClientVisible = true` kendala** — see "Tahap 5". **GET only, E6/E4:** each item also carries `vendorName` (the vendor of the issue's engagement, resolved within the workspace for this project's engagements only; `""` when unresolved), `vendorMilestoneName` (`""` when the issue has no vendor milestone) and `evidence: [{id, type, fileName, uploadedAt}]` (client: only what `portal.issues.view` allows). Create/update responses carry `""`/`[]` for these |
| PATCH | `/projects/{id}/issues/{issueId}` | **Tahap 5:** `projectVendorId` is optional (omitted/`null`/`0` = a kendala about the event in general) and `isClientVisible` is accepted (omitted = unchanged); see "Tahap 5". Body **penuh** (`projectVendorId, vendorMilestoneId?, title, description, impact, foundDate, status, resolutionPlan, picStaffId, targetResolutionDate`), bukan hanya `{status}` (PATCH parsial direncanakan di `docs/plan/revamp-ui-shadcn/PLAN.md` E9); auto-stamps `resolvedDate` exactly once when status becomes `Resolved`/`Closed` |
| GET | `/projects/{id}/evidence` | returns every evidence row for the project directly (not derived client-side from milestones/payments/issues) |
| POST | `/projects/{id}/evidence` | JSON body `{name, type, fileName, mimeType, base64Data, documentDate, description, relatedKind, relatedId, isClientVisible}` — base64, not multipart (see ADR-0006, ADR-0010); 15 MB decoded-size cap; `relatedKind` is one of `vendorMilestone`/`payment`/`projectVendor`/`issue`/`clientPayment`/`venuePayment`/`general` (migration `000035`, ADR-0020); `isClientVisible` is server-side re-normalized to `false` unless `relatedKind = "general"` (never trusted blindly from the request body, even for a `general` document it still defaults `false`) |
| POST | `/projects/{id}/evidence/{evidenceId}/toggle-client-visible` | flips `is_client_visible` for one evidence row — allowed for `relatedKind = "general"` **and `"projectMilestone"`** (a step's attachment is shown to the client only once staff flag it, ADR-0032); rejected `422` for every other kind (their visibility follows what they hang off). `isClientVisible` on upload follows the same rule (dropped for other kinds). Needs `project.edit` |
| GET | `/projects/{id}/documents` | Client Portal's own "Dokumen" tab feed (ADR-0020) — **always** only `relatedKind = "general"` **and** `isClientVisible = true`, unconditionally regardless of caller (reachable by staff too, harmlessly — the safety property is what the endpoint returns, not who can call it). Deliberately its own query/endpoint rather than a filter bolted onto `GET .../evidence`, so "only ever returns documents safe to show a client" stays trivially auditable |
| GET | `/projects/{id}/evidence/{evidenceId}/link` | **Fase 7.** `{url, expiresAt}`: a direct, two-minute link to the stored file in object storage, so the bytes do not pass through the API. Same project scope and client-visibility rule as `/file` (an internal document is 404 for a client). `?disposition=attachment` makes the link force a download. `501` with `errors.code = presign_unavailable` when the deployment has `S3_PRESIGN=false`; the web then falls back to `/file`. Allowed for client principals **Client visibility (E4):** judged per file *before* storage is touched, by what the file hangs off — payment proof `portal.finance.view`; step attachment `portal.timeline.view` + `isClientVisible`; vendor deliverable `portal.vendor.view`; issue `portal.issues.view`; general document `isClientVisible` + `portal.documents.view`. A file the client may not see is 404. (The single-file routes no longer ask for `portal.documents.view` up front — only the list `GET .../evidence` and `.../documents` do.) |
| GET | `/projects/{id}/evidence/{evidenceId}/file` | authenticated download (Bearer token required — not a public URL); streams the object exactly as stored (already compressed at upload time); frontend fetches it as a blob via `httpClient`, not a bare `<img src>` **Client visibility (E4):** judged per file *before* storage is touched, by what the file hangs off — payment proof `portal.finance.view`; step attachment `portal.timeline.view` + `isClientVisible`; vendor deliverable `portal.vendor.view`; issue `portal.issues.view`; general document `isClientVisible` + `portal.documents.view`. A file the client may not see is 404. (The single-file routes no longer ask for `portal.documents.view` up front — only the list `GET .../evidence` and `.../documents` do.) |
| GET | `/projects/{id}/activity` | **Tahap 5 (E2): paginated and filterable** — `?page=&limit=&entityType=&entityId=`, default 20 per page (max 100), newest first, `meta {page, limit, total, total_pages}`; each entry also carries `actorName`; new types `item_restored`, `evidence_deleted`, `vendor_deleted` — see "Tahap 5". Append-only, populated server-side on every mutation above; activity types actually emitted: `project_created`, `project_updated`, `project_status_changed`, `vendor_added`, `vendor_status_changed`, `milestone_updated`, `payment_recorded`/`payment_updated`/`payment_deleted` (each shared by vendor, client, and venue payments, distinguished only by `entityType`: `"vendor_payment"` vs. `"client_payment"` vs. `"venue_payment"`), `evidence_uploaded`, `issue_created`, `issue_updated`; response includes `projectId` (needed by the dashboard's cross-project activity feed, not by the per-project tab which already knows it from the URL) |

### Tahap 4 (E3, E7, E10) — signed uploads, a payment recorded with its proof, `Idempotency-Key`, a project created whole (ADR-0033)

All additive: the base64 `POST /projects/{id}/evidence` and the old two-call flow keep working unchanged.

**`POST /projects/{id}/uploads`** — staff; needs `project.edit` **or** `finance.record` (the same as the evidence upload; a `client` principal gets 403, another workspace's staff 404). Body `{fileName, mimeType, size}`. `fileName` ≤ 150 characters with no `/`, `\` or control character; `mimeType` one of `image/jpeg`, `image/png`, `image/webp`, `image/heic`, `image/heif`, `application/pdf`; `size` 1 … 15 MiB (15 728 640). **201** `data`:

```json
{ "uploadToken": "<payload>.<signature>", "method": "PUT", "url": "https://… or /api/v1/projects/{id}/uploads/{uploadToken}",
  "headers": { "Content-Type": "application/pdf" }, "expiresAt": "2026-11-09T21:15:00+07:00", "direct": true }
```

The browser sends the file with `method` to `url` with `headers` (one code path): `direct: true` means `url` is a presigned URL of object storage itself (storage reachable from browsers, `S3_PRESIGN`), `direct: false` means it is this API's own route below. The token is stateless and HMAC-signed (key derived from the JWT secret with a purpose label); it carries workspace, project, object key, declared type, file name, size cap, expiry (15 minutes) and a random nonce. Nothing is stored when it is granted. 422 per field (`fileName`, `mimeType`, `size`), all at once.

**`PUT /projects/{id}/uploads/{uploadToken}`** — the raw bytes (no JSON), same staff authorization. `Content-Type` must equal the declared type (422 `contentType`), the body may not exceed the declared size (413 `size`; never above 15 MiB) nor be empty (422 `size`); a token that is malformed, expired, tampered or for another workspace/project is 422 `uploadToken`. JPEG/PNG go through the usual server-side re-compression. A retried PUT before a payment used the token is fine; once an evidence row owns the object the token is spent and a PUT answers **409** `upload_token_used` (a recorded proof cannot be overwritten). **200** `data: {size}`.

**Payment create — `POST /projects/{id}/client-payments`, `/payments` (vendor), `/venue-payments`.** The existing body plus an optional `evidence: [{uploadToken, type, name?}]` — at most 4 items; `type` is `"Transfer Proof"` for a client payment and `"Invoice"` and/or `"Transfer Proof"` for vendor and venue payments; `name` defaults to `Bukti Transfer - <referenceNumber>` / `Invoice - <referenceNumber>`. Uploads are verified first (token valid and for this workspace + project, not already used, object present in storage, within its size cap and of the declared type — an object that is present but wrong is deleted); then **one database transaction** locks the project row, applies the overpayment rule, inserts the payment and one `evidence` row per file (`relatedKind` = the new payment, `documentDate` = `paymentDate`, uploaded by the caller, `isClientVisible` false), writes the same activity entries as the two-call flow and computes the ledger from the rows just written. Any failure rolls everything back: there is never a payment without the proof it was recorded with, and nothing partial is visible. A token can prove only one payment (unique index on `evidence.storage_path`); two requests racing with one token create exactly one payment. Objects uploaded but never committed are left in storage (see ADR-0033). **201** `data` = the payment object exactly as the list returns it (`evidence: [{id, type, fileName, uploadedAt}]` is now on vendor and venue payments too, in lists as well; `evidenceComplete` is true at once when the proof is complete) plus `ledgerSummary`:

```json
// client payment
{ "id": 12, "type": "DP", "amount": 15000000, "paymentDate": "2026-11-08", "method": "Transfer", "referenceNumber": "REF-1", "notes": "",
  "evidenceComplete": true, "recordedBy": {"name": "Ani"}, "verifiedAt": null, "verifiedBy": null,
  "evidence": [{"id": 7, "type": "Transfer Proof", "fileName": "bukti.pdf", "uploadedAt": "2026-11-09T14:02:11+07:00"}],
  "ledgerSummary": { "contractValue": 60000000, "paid": 15000000, "remaining": 45000000,
    "terms": [{"id": 3, "label": "DP", "amount": 30000000, "dueDate": "2026-11-14", "paid": 15000000, "remaining": 15000000, "status": "partial"}] } }
// vendor payment: the usual paymentResponse (+ evidence) and  "ledgerSummary": {"contractValue": 5000000, "paid": 2000000, "remaining": 3000000}   // that engagement
// venue payment:  the usual venuePaymentResponse (+ evidence) and "ledgerSummary": {"contractValue": 10500000, "paid": 4000000, "remaining": 6500000} // rental price + charge
```

`ledgerSummary.paid` is what counts toward the contract (DP + Termin + Pelunasan − Refund; `Tambahan` is outside it; never negative), `remaining` is never negative, `terms` is the server's own allocation (the same objects `GET /payment-terms` returns, as of today in Asia/Jakarta). Errors are **422** with one key per problem, all at once: `type`, `amount` (with `errors.code = "payment_exceeds_contract"` for an overpayment, as before), and per evidence item `evidence[i]` **and** `evidence[i].uploadToken` (same message: invalid/expired/foreign/never-uploaded/too-large/wrong-type/already-used token), `evidence[i].type` (type not accepted here) and `evidence[i].name`; `evidence` for more than 4 items. The token problems of items that are fine do not appear. An overpayment keeps the uploads usable: fix the amount and resend the same tokens.

**`Idempotency-Key` header** — on the three payment POSTs above and on `POST /projects`. 8–128 characters of `[A-Za-z0-9_-]` (otherwise **422** `errors["Idempotency-Key"]`); without the header nothing changes. Scoped to the workspace and to method + path (so project and call). First time: the request runs, a 2xx answer is stored for 24 hours and returned; any other answer releases the key (the corrected retry with the same key is a new attempt). Same key + same body (JSON key order and whitespace do not matter) and the first request succeeded: the stored status and body are replayed with header `Idempotent-Replay: true` and nothing is created. Same key + a different body (or another person): **422** `{errors: {"Idempotency-Key": ["…"], "code": "idempotency_key_reuse"}}`. Same key while the first request is still running: **409** `{errors: {"code": "idempotency_in_progress"}}`. Table `idempotency_keys` (DB_SCHEMA.md).

**`POST /projects` — E7 additions.** Optional body fields: `venueId` (a positive id of this workspace's venue; omitted or `0` = no venue), `venueRentalPrice`, `venueCharge` (each 0 … 10^15; **omitted = the venue's current price** — the project's own snapshot, as the web did), `terms: [{label, amount, dueDate}]` (the same rules as `PUT /payment-terms`: ≤ 20, label ≤ 100 characters, amount 1 … 10^15, `YYYY-MM-DD`, together not above `contractValue` when one is set). In **one transaction** the server creates the project, its default milestones, attaches the venue (the project's `venue` text becomes the venue's name, as when attaching later) and saves the schedule; it validates everything first and answers **one 422** with every problem of every part — keys `eventDate`, `prepStartDate`, `picStaffId`, `venueId` ("Venue tidak ditemukan", also for another workspace's venue), `venueRentalPrice`, `venueCharge`, `terms` (count / sum above contract), `terms[i].label|amount|dueDate` — and creates nothing. Trial quota and permissions are unchanged (403 `trial_limit_reached` is answered after validation). **201** `data` = the project exactly as before (`venue` stays the text) plus `terms: [{id, label, amount, dueDate, paid, remaining, status}]` (`[]` when none were sent) and `venueSummary: {id, name, address, city, capacity, facilities, socialMedia, hasVisibleAttachment} | null` (the same shape as `GET /projects/{id}/venue`; its own key because `venue` is already the project's text).

**E10 on these routes:** every validation answer here is a 422 `{success:false, message, errors:{<field>: [messages]}}` (plus `errors.code` where a stable reason exists); `message` is the first problem, followed by "Ada N isian lain yang perlu diperbaiki." when there are more.

### Tahap 5 (E1, E2, E8, E12, E13) — project overview, names, fresh progress on every mutation, soft delete with restore, kendala without a vendor, dashboard sections (ADR-0034)

All additive except the deletes (soft now) and `GET /activity` (now a page). Existing fields keep their names and shapes.

**`GET /projects/{id}/overview`** (E1) — staff, same project access and permissions as `GET /projects/{id}`; a `client` principal gets **403** (not on the portal allowlist), another workspace's staff 404, no token 401. One call for the project header, side panel and Ringkasan tab. `data`:

```json
{
  "project": { "...exactly what GET /projects/{id} returns for staff: ids, names, dates, venue, contractValue, status, picStaffId, picSalesStaffId, picName, progress {...}" },
  "finance": {
    "contractValue": 100000000, "paid": 10000000, "remaining": 90000000, "overdueTotal": 20000000,
    "nextTerm": { "id": 7, "label": "DP", "amount": 30000000, "dueDate": "2026-11-05", "status": "overdue", "remaining": 20000000 },
    "terms": [ { "id": 7, "label": "DP", "amount": 30000000, "dueDate": "2026-11-05", "paid": 10000000, "remaining": 20000000, "status": "overdue" } ],
    "vendorCost": 25000000, "venueCost": 4000000,
    "margin": 71000000
  },
  "actions": [ { "kind": "term_overdue", "id": "term-7", "tone": "danger", "title": "Termin DP terlambat", "detail": "Sisa Rp20.000.000, jatuh tempo 5 Nov 2026", "tab": "pembayaran/client" } ],
  "nextMilestones": [ { "id": 31, "name": "Fitting", "status": "Not Started", "targetDate": "2026-11-15" } ],
  "unconfirmedVendors": [ { "id": 12, "vendorName": "Bunga Indah", "engagementStatus": "Planned" } ],
  "openIssueCount": 2,
  "team": { "picStaffId": 3, "picName": "Sari", "picSalesStaffId": 0, "picSalesName": "" },
  "clients": [ { "id": 5, "name": "Rina", "role": "Bride", "hasLogin": true } ]
}
```

* `finance.paid`/`remaining` are what the client's payments apply to the contract (DP + Termin + Pelunasan − Refund; `Tambahan` outside it; `remaining` never negative). `terms` are the very objects `GET /payment-terms` returns; `nextTerm` is the earliest installment not fully paid (`null` when none); `overdueTotal` is the unpaid part of the overdue installments. `vendorCost` = the sum of `contractValue` of engagements that are not `Cancelled` (and not deleted); `venueCost` = rental price + charge; `margin = contractValue − vendorCost − venueCost` (may be negative). **`margin` is absent (the key is not there) unless the caller has `finance.margin.view`.**
* `actions` — the rules the web computed in the browser: an overdue installment (`term_overdue`, `danger`, id `term-<id>`, tab `pembayaran/client`), an open kendala (`issue_open`, id `issue-<id>`, `danger` for High/Critical, else `warning`, tab `kendala`, detail "Kendala masih terbuka"), a vendor still `Planned`/`Negotiation` when the event is 0–30 days away (`vendor_unconfirmed`, `warning`, id `vendor-<engagementId>`, title "<vendor name> belum terkonfirmasi" — "Vendor" if the name does not resolve —, detail "Acara H-<n>, status masih belum dipesan", tab `vendor`), a timeline item past its target date and not Completed/Cancelled (`milestone_late`, `warning`, id `ms-<id>`, title `Timeline "<name>" lewat tenggat`, tab `milestone`). Danger items first, then warnings; inside a tone: installments, kendala, vendors, timeline. Dates read "5 Nov 2026", money "Rp20.000.000". Staff see every kendala, including the ones hidden from the client.
* `nextMilestones` — at most 5 timeline items not Completed/Cancelled, earliest target date first. `unconfirmedVendors` — every engagement still Planned/Negotiation. `openIssueCount` — open kendala (staff view). `team` — names through the staff contract (`""` when unresolved). `clients` — the project's live clients through the clients module's contract; `hasLogin` is true once the client set a password (an invitation not yet accepted is `false`).
* Cost: 14 SQL statements in the handler's own path (the project twice — the access check and the assembler —, seven progress sources, the client's payments, the schedule, vendor names, staff names, the clients) plus one login lookup per client (a project has at most a handful); none per vendor, term or milestone. Above the "~10" target of the plan: the progress sources are seven separate reads.

**Vendor tab data (E2).** `GET /projects/{id}/vendors` items also carry `picName` (the engagement PIC's name; `""` when unresolved and always `""` for a client) and `evidenceCount` (every proof file of the engagement itself, of its milestones and of its payments that the caller may see — staff: all; a client: only what `domain.ClientCanSeeEvidence` allows, i.e. a deliverable's files with `portal.vendor.view`). `vendorName`, `categoryName` and the per-milestone `evidence[]` were already there (ADR-0032).

**`GET /projects/{id}/activity?page=&limit=&entityType=&entityId=`** (E2, E12) — staff only (403 for a client). **Always paginated**: default `limit` 20 (max 100), newest first, `meta: {page, limit, total, total_pages}`; `entityType`/`entityId` narrow it to one entity (e.g. `entityType=client_payment&entityId=12`). Each `data` item keeps its fields (`id, type, actorStaffId, projectId, entityType, entityId, entityLabel, description, timestamp`) and adds `actorName` — the staff member's name, `""` when the actor is unknown, was deleted or is `0`: **the server never invents a label** (the web may show its own text for `actorStaffId = 0`). `GET /dashboard` `recentActivity[]` carries `actorName` too.

**Names in responses (E12).** Evidence rows (`GET …/evidence`, upload, toggle) carry `uploadedByName` (always `""` for a client); issues carry `picName` (always `""` for a client) in lists and in create/update answers; dashboard `openIssues[]`, `overdueVendorMilestones[]` and `incompletePayments[]` rows carry `vendorName` next to `vendorId` (`""`/`0` for a kendala without a vendor).

**Fresh progress on every mutation (E8).** The answer of every mutation of project milestones (create, update, reorder), vendor milestones (create, update), kendala (create, update), vendor engagements (create, update, cancel, **delete, restore**), the three payment kinds (create, update, delete, restore; client payment also `verify`) and evidence (upload, toggle-client-visible, **delete**) is the same `data` as before **plus** `"projectProgress": <the object GET /projects/{id} returns as progress, computed after the change>`. Where `data` used to be `null` (reorder, deletes) it is now `{"projectProgress": …}` (deletes: see below). On update, delete and restore of a client, vendor and venue payment the answer also carries `"ledgerSummary"` the way a create does (`{contractValue, paid, remaining}`; for a client payment also `terms`). If computing the progress fails after a committed mutation the key is omitted (the mutation is not turned into an error).

**New route `DELETE /projects/{id}/evidence/{evidenceId}`** — removes one proof file for good (row and stored object); needs `project.edit` or `finance.record`, plus `finance.payment.delete` when the file proves a payment. 200 `data: {id, projectProgress}`. A verified client payment loses its verification with its last file (ADR-0032). Logged as `evidence_deleted`.

**Soft delete and restore (E13).** `DELETE` of a client payment, vendor payment, venue payment, vendor engagement (`/projects/{id}/vendors/{pvid}`, new) and client (`/clients/{id}`) no longer removes the row: it answers **200**
```json
{ "id": 12, "restorableUntil": "2026-11-11T09:00:00+07:00", "projectProgress": { "...": "..." }, "ledgerSummary": { "contractValue": 50000000, "paid": 0, "remaining": 50000000, "terms": [ "..." ] } }
```
(`ledgerSummary` on payments only — `terms` only on client payments; a client delete answers just `{id, restorableUntil}`: the clients module has no project progress). `restorableUntil` = now + 24 h (RFC 3339, Asia/Jakarta for projects, server local time for clients). From that moment the item **disappears from every list, total and screen**: payment lists, the ledger and the payment schedule, progress, the overview, the portal home and the dashboard, the vendor tab, the vendor's project history, the trial counters; the proof files of a deleted payment/engagement/kendala are not listed, opened (`/file`, `/link` → 404) or attachable by anyone, a deleted engagement takes its milestones, payments and kendala with it, and a deleted client is locked out at once (the login is deactivated, not removed). A verified payment is deletable like any other; its verification is kept on the hidden row and returns on restore.

`POST /projects/{id}/client-payments/{pid}/restore`, `/payments/{pid}/restore`, `/venue-payments/{pid}/restore` (need `finance.payment.delete`), `/vendors/{pvid}/restore` (needs `project.edit`) and `POST /clients/{id}/restore` (needs `client.manage`) answer **200** with the restored item exactly as its list shows it (payments: `evidence[]`, verification and — for a client payment — `recordedBy`/`verifiedBy`; engagement: `vendorName`, `categoryName`, `picName`, `evidenceCount`, `milestones[]`, `paidAmount`; client: `loginStatus`, `accessRoleName`) plus `projectProgress` and, on payments, `ledgerSummary`. Errors: **404** when the item is not deleted, never existed, belongs to another workspace/project or was deleted more than 24 h ago (counts as purged even if the purge has not run); **422** with the usual keys when a rule now fails — `amount` (+ `errors.code = payment_exceeds_contract`) when adding the payment back would overpay what is owed, `projectVendorId` when a vendor payment's engagement is itself deleted (restore it first), `vendorId`/`categoryId` when an engagement's vendor or category no longer exists, `projectId`/the trial-quota message when a client's project is gone or the trial has no room. The activity log records `payment_deleted`, `vendor_deleted` and `item_restored` (clients are not in the project log).

**Purge.** `api purge` (next to `migrate`/`seed`) and a low-frequency in-process ticker (`PURGE_INTERVAL`, default 30 m, `0` = off) hard-delete what has been deleted for 24 h or more: the rows, their `evidence` rows and stored objects, a purged engagement's milestones, kendala and payments, a purged client's login. Idempotent; one database lock (`GET_LOCK`) per module lets several instances share one database; it logs the counts.

**Kendala (E13).** `projectVendorId` is optional on create and update (omitted, `null` or `0` = a kendala about the event in general; a `vendorMilestoneId` then 422s on that key). `isClientVisible` (default `true` on create; omitted on update keeps it) decides whether the couple sees the kendala. Every issue answer carries `projectVendorId` (`null` without a vendor), `isClientVisible`, `vendorName` and `vendorMilestoneName` (`""` without a vendor/milestone), `picName`. **A client only ever gets `isClientVisible = true` kendala:** the list, `/projects/me/home` (`openIssueCount`, the `open_issues` action, `issue_resolved` news), the progress that `GET /projects/{id}` and the home show (count and condition), and the proof files of a hidden kendala (`/evidence`, `/file`, `/link` → 404 for the client; staff unaffected). Existing rows are `isClientVisible = true`.

**Dashboard (E13) — three sections added to `GET /dashboard`** (tenant-wide, as the rest of the dashboard; open projects only, i.e. not completed, cancelled or archived):
```json
{
  "cashFlow": [ { "weekStart": "2026-11-09", "expectedIn": 15000000, "expectedOut": 10000000 } ],
  "vendorPaymentsDue": [ { "projectId": 7, "projectName": "Akad Rina", "vendorName": "Bunga Indah", "dueDate": "2026-11-15", "remaining": 10000000 } ],
  "unconfirmedVendors": [ { "projectId": 7, "projectName": "Akad Rina", "vendorName": "Bunga Indah", "engagementStatus": "Planned", "eventDate": "2026-11-20" } ]
}
```
`cashFlow` has exactly 8 entries, the first being the week (Monday–Sunday) that contains today; `weekStart` is that Monday. `expectedIn` = what the clients still owe on payment-schedule installments due that week (each project's real payments allocated over its schedule, oldest first); `expectedOut` = what is still owed on non-cancelled vendor engagements with a due date (`contractValue − DP − Termin − Pelunasan + Refund`, never negative) by due date; anything already overdue counts in week 1, anything beyond week 8 is left out. `vendorPaymentsDue` — those same unpaid vendor balances that are overdue or due within 30 days, nearest first, at most 10. `unconfirmedVendors` — engagements still `Planned`/`Negotiation` whose event is 0–30 days away, soonest event first, at most 10. **`cashFlow` and `vendorPaymentsDue` need `dashboard.revenue.view`: without it the keys are absent and nothing money-related is computed; `unconfirmedVendors` is always present.**

### Tahap 6 (E11, E13 bulk, E14) — sorting, filters, derived columns, bulk activate/deactivate (ADR-0035)

All additive: with none of the new parameters every list answers in the order it had (`ORDER BY id` lists now read `createdAt:asc`, the same
order), and `?all=true` and the older shapes are untouched.

**`?sort=field:asc|desc`** on every paginated list below. One column; `field` alone means ascending; the direction word is case-insensitive, the
field name exact. The field must be in the endpoint's whitelist (table below) or the answer is **422** with a single message under `errors.sort`:

```json
{ "success": false, "message": "Parameter urutan tidak valid",
  "errors": { "sort": ["Urutkan menurut salah satu dari: contractValue, createdAt, eventDate, name, nextDueDate, remaining, status"] } }
```

A bad direction (`name:up`) answers `"Arah urutan harus asc atau desc"`; two `sort` parameters, a comma list, `field:asc;DROP …`, a backquoted name,
`1` … are all just "not in the whitelist" (nothing of the value reaches the SQL). The primary key is always the last sort key, in the same direction,
so equal values page in a stable order. A column that may be empty (`priceAkad`, `city`, `nextDueDate`, `trialEndsAt` …) puts its empty values **last in
both directions**. The effective sort is echoed in `meta.sort`; the three other keys are unchanged:

```json
{ "success": true, "message": "ok", "data": [ … ],
  "meta": { "page": 1, "limit": 10, "total": 37, "total_pages": 4, "sort": "remaining:desc" } }
```

**`q`** is the common name of the free-text search; `search` (what every list accepted first) stays and means the same; `q` wins when both are sent.
Boolean filters (`isActive`) take exactly `true` or `false` (422 under their key otherwise); dates are `YYYY-MM-DD` (422 under their key otherwise).

| Endpoint | Whitelist (`?sort=`) | Default | New filters / fields |
|---|---|---|---|
| `GET /projects` | `name`, `eventDate`, `status`, `contractValue`, `remaining`, `nextDueDate`, `createdAt` | `eventDate:desc` | `picStaffId`, `eventFrom`, `eventTo` (both inclusive, on `eventDate`; `eventTo` before `eventFrom` → 422 `eventTo`); items gain `remaining`, `nextDueDate` |
| `GET /clients` | `name`, `role`, `projectName`, `projectEventDate`, `loginStatus`, `createdAt` | `createdAt:asc` | `role` (`Bride`/`Groom`/`Family Representative`), `loginStatus` (`active`/`invited`/`inactive`/`none`), `projectId`, `q`; items gain `projectName`, `projectEventDate` |
| `GET /vendors` | `name`, `categoryName`, `city`, `priceAkad`, `isActive`, `createdAt` | `createdAt:asc` | `isActive`; items gain `categoryName` |
| `GET /venues` | `name`, `city`, `capacity`, `rentalPrice`, `isActive`, `createdAt` | `createdAt:asc` | `isActive` |
| `GET /staff` | `name`, `title`, `roleName`, `isActive`, `createdAt` | `createdAt:asc` | `isActive`, `roleId` (the older `role=<id>` still works); items gain `roleName` (the role's current name, the same as `role`) |
| `GET /vendor-categories` | `name`, `createdAt` | `createdAt:asc` | — |
| `GET /subscription-transactions` | `createdAt`, `amount`, `status`, `tenantName` | `createdAt:desc` | `search`/`q`, `from`, `to` (see its row); items gain `tenantName` |
| `GET /tenants` | `name`, `status`, `createdAt`, `trialEndsAt`, `planName` | `createdAt:desc` | the existing `search`/`status`/`planId`/`source`; `q` as an alias of `search` |
| `GET /audit-log` | `createdAt`, `action`, `actor` | `createdAt:desc` | `action` (exact label), `search` as an alias of `q` |

Notes on single columns: `status` of a project or a transaction sorts in the order of its enum (project: Draft, Preparation, Ready, Completed, Cancelled;
transaction: unpaid, pending, paid, expired, granted, cancelled), not alphabetically; `role` of a client likewise (Bride, Groom, Family Representative); tenant
`status` is the stored status the hourly sweeper keeps current; tenant `createdAt` is the registration date (`joinedAt`); `actor` orders by actor type, role,
then the actor id as a number; `planName` and `roleName` order by the plan's/role's current name (those names live in `billing`/`access`, so the SQL ranks the
ids by name), `tenantName`, `projectName`, `projectEventDate` and `loginStatus` order after the other module's data has been read (see ADR-0035).

**`GET /projects` items — `remaining` and `nextDueDate`.** `remaining` is what the client still owes on the contract: `contractValue` minus the payments applied
to it (DP + Termin + Pelunasan − Refund; a Tambahan is outside the contract; a soft-deleted payment is not counted), never negative — the very rule of
`GET /projects/{id}/overview` (`finance.remaining`). `nextDueDate` is the due date of the earliest installment of the payment schedule that is not fully paid
(the schedule is allocated earliest due date first, then id), or `null` when there is no schedule or everything is paid; the date is the installment's own, whether
or not it is already overdue. Both are computed for the page with two batched queries and, when sorted, by an SQL expression of the same rule (the integration
suite checks the order against the displayed values). Scope is unchanged: a Wedding Planner / Sales role still sees only projects where the person is PIC or PIC Sales,
and every new filter is ANDed with that scope (`?picStaffId=` of someone else gives the planner nothing). `?all=true` does not carry the two columns.

```json
{ "id": 12, "name": "Pernikahan Sari & Budi", "brideName": "Sari", "groomName": "Budi", "eventDate": "2026-12-20", "venue": "Gedung Mawar", "venueId": null,
  "venueRentalPrice": null, "venueCharge": null, "prepStartDate": "2026-10-11", "packageName": "Paket Gold", "contractValue": 100000000, "status": "Preparation",
  "picStaffId": 3, "picSalesStaffId": 0, "picName": "", "description": "", "isArchived": false, "progress": { "…": "as before" },
  "remaining": 70000000, "nextDueDate": "2027-01-09" }
```

**`GET /clients` — the flat, paginated list.** `?page&limit&sort&q&role&loginStatus&projectId`. `loginStatus` is what the list already derived from `identity`:
`active` (can sign in), `invited` (no password yet), `inactive` (switched off), `none` (no login at all); an unknown `role` or `loginStatus` is a 422 under its key.
Items add `projectName` and `projectEventDate` (from one batched `projects` lookup per page). **Compatibility:** `?all=true` is unchanged (plain array, owner-wide);
`?projectId=` **alone** (none of `page`, `limit`, `sort`, `q`, `role`, `loginStatus`) is unchanged too (the project tab's plain array, same access rule: a role limited to
its own projects must be PIC or PIC Sales of that project). With any of those parameters `projectId` is a filter on the paginated list, with the same access rule. Without
`projectId` the list is the tenant-wide one, still `client.manage` with every-project scope (403 `forbidden_permission` for a Wedding Planner or Sales). A project id of
another workspace lists nothing.

```json
{ "id": 4, "projectId": 12, "role": "Bride", "username": "sari.4", "relationNote": "", "name": "Sari", "phone": "0812…", "email": "", "isActive": true,
  "lastCredentialResetAt": null, "loginStatus": "invited", "accessRoleId": 6, "accessRoleName": "Pengantin",
  "projectName": "Pernikahan Sari & Budi", "projectEventDate": "2026-12-20" }
```

**`GET /vendors` item** gains `"categoryName": "Katering"` (last key); **`GET /staff` item** gains `"roleName": "Wedding Planner"`;
**`GET /subscription-transactions` item** gains `"tenantName"`:

```json
{ "id": 31, "tenantId": 7, "type": "renewal", "amount": 2000000, "paymentMethod": "QRIS", "paymentReference": "ELP-ABC123", "status": "paid",
  "createdAt": "2026-02-10T23:59:00+07:00", "paidAt": null, "tenantName": "WO Melati" }
```

`tenantName` is `""` when the tenant no longer resolves. `GET /subscription-transactions/{id}` carries it too. `GET /tenants/names` is unchanged. `search` matches the payment
reference (substring) **or** any tenant whose business name contains the text (a workspace's own Owner is still confined to their own tenant, whatever they search); `from`/`to`
are Indonesian calendar days on `createdAt`, both inclusive (`from` after `to` → 422 `to`; a malformed date → 422 on its key); a name search resolves at most 1000 tenants.

**`POST /vendors/bulk` and `POST /venues/bulk`** — body `{"ids": [12, 13, 13, 99], "action": "activate" | "deactivate"}`. Needs what the single `toggle-active` of that
resource needs (`vendor.manage`; 403 otherwise, 401 without a token, 405 for another method). `ids`: 1 to 100 positive integers as sent (duplicates are removed, order
kept; more than 100 as sent is refused). Everything is read, decided and written in **one transaction** scoped to the caller's workspace; ids that are not this workspace's
(another workspace's, or nonexistent) are reported as `notFound`, indistinguishable from each other and never touched:

```json
{ "success": true, "message": "Status vendor diperbarui",
  "data": { "updated": 2, "unchanged": [13], "notFound": [99] } }
```

`updated` is the number of rows changed; `unchanged` lists the ids already in the requested state; both lists keep the request's order and are `[]` when empty. Validation is a
422 with per-field keys, all problems at once: `{"errors": {"ids": ["Pilih 1 sampai 100 data"], "action": ["Aksi harus activate atau deactivate"]}}`; a body that is not
JSON is a 400. The single toggle writes no activity record, so neither does this (the `vendors` module has no activity log).

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

**Tahap 3 additions (E4, E5, E6 — `docs/plan/revamp-ui-shadcn/PLAN.md`, ADR-0032).** All additive to existing responses except the two closed endpoints:
- `GET /projects/me/home` (above) is the one call behind the portal home; `GET /projects/me` stays.
- Names and proof come inside already-filtered responses so the portal needs no directory calls: `vendorName`/`categoryName` on `GET /projects/{id}/vendors` items, `vendorName`/`vendorMilestoneName` on `GET /projects/{id}/issues` items, `evidence[]` on milestones, vendor milestones, client payments and issues, and `verifiedAt`/`verifiedBy` on client payments.
- **Closed for clients:** `GET /vendors/summary` and `GET /vendor-categories` (any query) now answer 403 for a `client` principal. The client allowlist (`cmd/server/client_scope.go`) is: `auth/me`, `access/me`, `tenants/me/branding|logo|contact`, `projects/me`, `projects/me/home`, `projects/{id}` and its allowlisted sub-resources (`milestones`, `vendors`, `client-payments`, `payment-terms`, `issues`, `evidence`, `documents`, `venue`, `evidence/{id}/file|link`) and `venues/{id}/attachment`.
- `recordedBy` (who entered a payment) is never sent to a client; `verifiedBy`/`verifiedAt` are.
- Verification and the staff-only `POST .../client-payments/{pid}/verify` are described in the `projects` table above.
