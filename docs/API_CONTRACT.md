# API Contract — ElProof

> Endpoint list and request/response formats. Conventions: `.claude/rules/api-standard.md` +
> [`knowledge/API_GUIDE.md`](../knowledge/API_GUIDE.md). All paths are under `/api/v1`. Auth header
> required on every endpoint except login/refresh (see API_GUIDE.md).

Envelope reminder — success: `{ "success": true, "message": "...", "data": ... }`; error:
`{ "success": false, "message": "...", "errors": {...} }`; paginated lists additionally return
`meta: { page, limit, total, total_pages }`.

**Pagination — Fase 7, applies to every `GET` list endpoint marked "paginated" below:** default
response is paginated (`?page=`/`?limit=`, default `page=1`/`limit=10`, `limit` capped at 100).
Pass `?all=true` to instead get the old unpaginated shape (`response.OK`, no `meta`) — every
dropdown/picker/global-search/dashboard-aggregation consumer in the frontend uses this. `?search=`
does a `LIKE` match (columns vary per endpoint, noted below); other filters (`role`, `categoryId`,
`status`) apply at the SQL level too, so `search` + a filter + a page number combine correctly
instead of a filter being applied only to one already-paginated page.

---

## `identity`

| Method | Path | Notes |
|---|---|---|
| POST | `/auth/login` | body `{username, password}` → `{accessToken, refreshToken, principalType, principalId, tenantId, role, displayName}` |
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
| GET | `/tenants` | **paginated** (Fase 7); `?search=` matches business/owner name + email; `?status=` filters `subscriptionStatus` |
| POST | `/tenants` | orchestrates `staff.CreateOwner` + `identity.CreateCredential` — see ADR-0008. No `planId` in the body and no transaction recorded: the tenant starts unbound to any plan (`planId: null`, `subscriptionStatus: pending_payment`); the first transaction row it ever gets comes from whichever real subscription event happens first — the Owner's own `/subscriptions/pay` or the Platform Console's `/activate-subscription` below — never a placeholder created here |
| GET | `/tenants/{id}` | |
| PATCH | `/tenants/{id}` | |
| POST | `/tenants/{id}/toggle-suspension` | single toggle (flips current state), not separate suspend/reactivate |
| POST | `/tenants/{id}/reset-credential` | body `{password}`, resets the Owner's login via `identity.ResetPasswordByUsername` |
| POST | `/tenants/{id}/activate-subscription` | body `{planId}` → `billing.RecordTransaction` status `granted`, bypasses payment. End state (`subscriptionStatus`/`subscriptionExpiresAt`) is computed by the exact same `UpdateSubscription` call as a real payment confirmation — granted and paid subscriptions are indistinguishable afterward except in the transaction ledger. Also resolves (expires + untracks) any self-service charge still pending for this tenant first, so a late webhook/reconciliation-sweep result for that old charge can't double-extend the expiry this call just set |
| GET | `/tenants/me` | **staff Owner self-service** (not platform_admin) — own tenant's plan/status/expiry, powers WO Console's `Langganan` |
| POST | `/subscriptions/pay` | **staff Owner self-service** — body `{planId}`. **Changed Fase 9:** no longer activates synchronously — creates a real charge via `payment.Client.CreateCharge` (QRIS), records a `pending` transaction, and returns the charge (`{orderRef, providerRef, channel, qrImageUrl, payCode, checkoutUrl, amount, feeAmount, expiresAt, status}`). The subscription only activates once `/webhooks/payment` confirms it — see `payment` module below. Rejects (409) if this tenant already has a pending charge — see `GET`/`cancel` below for how the Owner resolves that instead of getting stuck |
| GET | `/subscriptions/pending-charge` | **staff Owner self-service** — returns `data: null` if nothing is pending, otherwise the same `{orderRef, providerRef, channel, qrImageUrl, payCode, checkoutUrl, amount, feeAmount, expiresAt, status}` shape as `Pay`, re-fetched live from the gateway via `payment.Client.CheckStatusForApp` (never cached — these fields were never persisted after `Pay`'s one-time response). Lets the Owner re-view a still-pending charge (e.g. after accidentally closing its QR modal) instead of losing access to it until it expires |
| POST | `/subscriptions/pending-charge/cancel` | **staff Owner self-service** — 404 if nothing is pending. Marks the pending transaction `cancelled` (distinct from `expired` — this was a deliberate give-up, not a timeout) and stops tracking it, freeing `Pay` to start a fresh charge immediately. Bookkeeping only: Tripay has no cancel/void API, so the old charge is still technically payable at the gateway until its own expiry — if the Owner somehow still completes it, the late webhook/reconciliation result finds no tracking row and is silently ignored rather than reactivating the subscription |
| GET | `/platform-admins` | **paginated** (Fase 7); `?search=` matches name + email; `?role=` filters exact role |
| POST | `/platform-admins` | orchestrates `identity.CreateCredential` |
| PATCH | `/platform-admins/{id}` | |
| POST | `/platform-admins/{id}/toggle-active` | single toggle; self-lockout (an admin deactivating their own account) rejected with 403 |
| POST | `/platform-admins/{id}/reset-password` | |

Platform dashboard has no dedicated endpoint yet — the frontend computes stats/trend client-side
from `GET /tenants` + `GET /subscription-transactions` (`getPlatformStats`, `buildRevenueTrend`),
same as before Fase 2. Moving that computation server-side (with a real clock instead of the
frontend's local "today") is Fase 5 scope.

## `billing` (read by both consoles; writes are `platform_admin`-only) — **implemented, Fase 2**

| Method | Path | Notes |
|---|---|---|
| GET | `/plans` | public to any authenticated principal — WO Console's `Langganan` reads this too; **paginated** (Fase 7), but every current frontend consumer calls `?all=true` since the plan catalog is a small, bounded reference list, not a growing per-tenant one |
| POST | `/plans` | platform_admin only |
| PATCH | `/plans/{id}` | platform_admin only |
| POST | `/plans/{id}/toggle-active` | platform_admin only |
| GET | `/subscription-transactions` | platform_admin: all tenants (optional `?tenantId=`); staff: forced to own tenant from JWT regardless of query; **paginated** (Fase 7), `?status=` filters transaction status (now includes `pending`, added Fase 9) |

No `POST /subscription-transactions` on this module — both "pay" and "activate" mutations live on
`platform`'s routes (`/subscriptions/pay`, `/tenants/{id}/activate-subscription`) since they must
also update the `tenants` row `billing` doesn't own; they call `billing.Contracts.RecordTransaction`
internally. See ADR-0008.

## `staff` (WO Console, tenant-scoped)

| Method | Path | Notes |
|---|---|---|
| GET | `/staff` | tenant-scoped; **paginated** (Fase 7); `?search=` matches name + email; `?role=` filters exact role; response includes `username` |
| POST | `/staff` | body `{name, title, role, username, password, email, phone}` — rejects `role=Owner` (422); provisions a real login credential via `identity.CreateCredential` in the same call (mirrors `clients`' `POST /clients`), rolling back (deleting) the just-inserted row if that fails, e.g. username already taken by *any* principal on the platform (usernames aren't tenant-scoped) |
| PATCH | `/staff/{id}` | body `{name, title, role, email, phone}` — no `username` (immutable after creation). Rejects assigning `role=Owner` to a non-Owner row (403). For a row that's already `role=Owner`: rejected (403) unless the caller's own JWT principal id equals `{id}` — i.e. the Owner may edit their own name/title/contact details, but no other staff member (even Admin) may touch the Owner's row; even the Owner can't change their own role away from `Owner` this way |
| POST | `/staff/{id}/toggle-active` | rejects deactivating the Owner row (403), regardless of caller |

## `clients` — **implemented, Fase 4** (bundled with `projects`, since `clients.project_id` needs
a real `projects` row to reference)

| Method | Path | Notes |
|---|---|---|
| GET | `/clients` | tenant-scoped; `?projectId=` filters to one project (unpaginated — small, bounded per-project list). Omit `projectId` to list every client for the tenant instead (added Fase 5, powers `GlobalSearch` via `?all=true`) — this tenant-wide mode is **paginated** (Fase 7), `?search=` matches name + email; response includes `username` |
| POST | `/clients` | body `{projectId, role, relationNote, name, phone, email, username, password}` — validates `projectId` via `projects.Contracts.ProjectExists`, provisions a real login credential in the same call; if `identity.CreateCredential` fails (e.g. username already taken by *any* principal on the platform — usernames aren't tenant-scoped), the just-inserted `clients` row is rolled back (deleted) rather than left orphaned with no working credential |
| PATCH | `/clients/{id}` | contact edit (`{name, phone, email}`) — no `username` (immutable after creation) |
| POST | `/clients/{id}/toggle-active` | |
| POST | `/clients/{id}/reset-credential` | body `{password}` — sets a new password and stamps `lastCredentialResetAt`. Fails with `404`/"Kredensial tidak ditemukan" for a client left behind by a pre-rollback-fix `Create` failure (no `identity` row ever existed) — use `DELETE` below to clear those out, not this endpoint |
| POST | `/clients/{id}/replace-representative` | body `{name, phone, email, relationNote}`; only valid when `role=Family Representative`; overwrites the same row, no history kept (§6.3) |
| DELETE | `/clients/{id}` | hard delete — the one permanent-delete endpoint in this module (every other mutation elsewhere is a soft toggle). Best-effort deactivates the client's `identity` credential first (harmless no-op if none exists); exists specifically so an operator can self-service clear a client stuck with no working credential, since deactivating alone doesn't free up that client's role slot on the project (the role lookup doesn't filter on `is_active`) |

## `vendors` (tenant-scoped) — **implemented, Fase 3**

| Method | Path | Notes |
|---|---|---|
| GET | `/vendor-categories` | tenant-scoped; **paginated** (Fase 7); `?search=` matches category name |
| POST | `/vendor-categories` | |
| PATCH | `/vendor-categories/{id}` | |
| POST | `/vendor-categories/{id}/toggle-active` | |
| GET | `/vendors` | tenant-scoped, `?categoryId=` filter; **paginated** (Fase 7); `?search=` matches name + PIC + email |
| POST | `/vendors` | validates `categoryId` belongs to the same tenant (422 if not) |
| PATCH | `/vendors/{id}` | same `categoryId` validation |
| POST | `/vendors/{id}/toggle-active` | |
| GET | `/vendors/{id}/project-history` | backs "Lihat Project" — the vendor's engagement history across every project in the tenant, `[{projectId, projectName, eventDate, venue, engagementStatus}]` ordered by `eventDate DESC`. `project_vendors` is owned by `projects`, not `vendors` — resolved through `projects.Contracts.ListVendorEngagementHistory` (`vendors.NewModule` now takes `projects.Contracts` as a constructor arg; `main.go` builds `projects` before `vendors` accordingly). |

## `projects` (tenant-scoped — the largest surface) — **implemented, Fase 4**

| Method | Path | Notes |
|---|---|---|
| GET | `/projects` | staff-only; **paginated** (Fase 7); `?search=` matches name + bride/groom name + venue, `?status=` filters exact status; does **not** include `progress` (see below) |
| POST | `/projects` | staff-only |
| GET | `/projects/me` | **client-only** (Fase 6) — resolves and returns the calling client's own single project (same shape as `GET /projects/{id}`, `progress` included); the Client Portal's entry point, since a client never learns its own project id any other way (mirrors `GET /tenants/me` from Fase 2). Handled as a special `"me"` segment inside the same dispatcher as `/projects/{id}/...`, not a separate route — see implementation note. |
| GET | `/projects/{id}` | includes `progress` (computed on read, not stored). Staff: any project in their tenant. **Client** (Fase 6): only their own project — any other `{id}` is 403, checked server-side via `clients.Contracts.ProjectIDForClient`, not just a frontend convention. |
| PATCH | `/projects/{id}` | |
| POST | `/projects/{id}/cancel` | soft status change |
| GET/POST | `/projects/{id}/milestones` | — |
| PATCH | `/projects/{id}/milestones` | body `{orderedIds}` — full new order of all of the project's milestone IDs; rejected (422) unless it's an exact permutation of the existing set |
| PATCH | `/projects/{id}/milestones/{milestoneId}` | body `{status, targetDate, completedDate}` — full update, mirrors vendor milestones below; `completedDate` is client-supplied, not server-stamped |
| GET/POST | `/projects/{id}/vendors` | project-vendor engagements; create/update body requires `eventDate` (frontend fills it from the parent project's own `eventDate` — there's no separate date field in the vendor form) |
| PATCH | `/projects/{id}/vendors/{projectVendorId}` | full-body update, same shape as create |
| POST | `/projects/{id}/vendors/{projectVendorId}/cancel` | one-way; "un-cancelling" is just editing `engagementStatus` back via PATCH |
| GET/POST | `/projects/{id}/vendors/{projectVendorId}/milestones` | vendor milestones |
| PATCH | `/projects/{id}/vendors/{projectVendorId}/milestones/{id}` | full-body update (`status`, `targetDate`, `completedDate`, `picStaffId`, `description`, `notes`) — no partial-field PATCH |
| GET/POST | `/projects/{id}/payments` | no endpoint to attach `invoiceEvidenceId`/`proofEvidenceId` after creation — a payment's evidence-completeness can only ever be satisfied by whatever was true at creation (always incomplete in practice; matches the pre-integration mock's own limitation) |
| GET/POST | `/projects/{id}/issues` | |
| PATCH | `/projects/{id}/issues/{issueId}` | body `{status}`; auto-stamps `resolvedDate` exactly once when status becomes `Resolved`/`Closed` |
| GET | `/projects/{id}/evidence` | returns every evidence row for the project directly (not derived client-side from milestones/payments/issues) |
| POST | `/projects/{id}/evidence` | JSON body `{name, type, fileName, mimeType, base64Data, documentDate, description, relatedKind, relatedId}` — base64, not multipart (see ADR-0006, ADR-0010); 15 MB decoded-size cap; `relatedKind` is one of `vendorMilestone`/`payment`/`projectVendor`/`issue` |
| GET | `/projects/{id}/evidence/{evidenceId}/file` | authenticated download (Bearer token required — not a public URL); streams the object exactly as stored (already compressed at upload time); frontend fetches it as a blob via `httpClient`, not a bare `<img src>` |
| GET | `/projects/{id}/activity` | append-only, populated server-side on every mutation above; activity types actually emitted: `project_created`, `project_updated`, `project_status_changed`, `vendor_added`, `vendor_status_changed`, `milestone_updated`, `payment_recorded`, `evidence_uploaded`, `issue_created`, `issue_updated`; response includes `projectId` (needed by the dashboard's cross-project activity feed, not by the per-project tab which already knows it from the URL) |

## WO Console dashboard — **implemented, Fase 5**

| Method | Path | Notes |
|---|---|---|
| GET | `/dashboard` | staff-only; single aggregate payload — `totalProjects`, `activeProjects`, `activeVendorCount`, `openIssues[]`, `overdueVendorMilestones[]`, `incompletePayments[]`, `nearDDayProjects[]`, `laggingProjects[]` (project + computed `overallPercent`), `upcomingProjects[]`, `recentActivity[]` (8 most recent, across all tenant projects), `revenue` (`{total, previousTotal, deltaPercent}`), `projectTrend[]` and `revenueTrend[]` (12 trailing months). Every array field is always `[]`, never `null`, when empty. Issue/milestone/payment rows carry `vendorId` only (no vendor name) — the frontend resolves display names from the already-fetched `vendors` store, the same pattern used throughout the `projects` module since Fase 4. |

No `GET /search?q=` endpoint exists or is planned — `GlobalSearch.tsx` calls the existing
`GET /projects`, `GET /vendors`, and `GET /clients` (no `projectId` — see below) endpoints directly
and filters client-side, since all three datasets are already small enough to fetch in full.

Platform dashboard has **no new endpoint** (Fase 5) — its data (`GET /tenants` +
`GET /subscription-transactions`) was already real since Fase 2; only the frontend's date reference
changed (real `todayISO()` instead of the mock's hardcoded `TODAY`).

## Client Portal (read-scoped subset — same `projects` endpoints, `client` principal) — **implemented, Fase 6**

`client` principals hit the exact same `/projects/{id}/...` GET endpoints as staff (milestones,
vendor engagements + their milestones, payments, issues, evidence + file download, activity) —
there is no separate client-portal endpoint set. Enforcement, checked server-side on every request
(not just a frontend routing convention):
- Only `GET` is ever allowed for a `client` principal — any `POST`/`PATCH` is rejected 403 before
  reaching any business logic, even for the client's own project.
- `{id}` must equal the one project resolved for that client via `clients.Contracts.ProjectIDForClient`
  (new in Fase 6) — any other `{id}` (including a real project belonging to a *different* client in
  the same tenant) is rejected 403.
- The client's own project id is learned via `GET /projects/me` (see above) — there is no other way,
  by design (login doesn't return one; `identity` stays profile-agnostic per ADR-0005).

A deactivated client (`clients.is_active = false`) is also rejected 403, even for their own project.
