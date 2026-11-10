# API inventory — routes registered in `apps/api` (2026-11-10)

> Companion to [`docs/API_CONTRACT.md`](../../API_CONTRACT.md), written while bringing that contract in line with the code after the UI revamp
> (ADR-0032…ADR-0036 and the late decision that `If-Match` is required on field edits). Nothing here is a decision to delete anything: the
> *Candidates for removal* section is a reading aid for whoever decides.

## How this was produced

- **Routes.** Every `mux.Handle`/`mux.HandleFunc` in `cmd/server/main.go` and each module's `RegisterRoutes`/`RegisterPublicRoutes`/`RegisterAuthedRoutes`
  (`apps/api/internal/modules/*/*.module.go`), then the per-module dispatchers that fan one prefix out to many method+path pairs (`Item`/`Collection`/`Console`/`Links`
  in `presentation/*.go`). One row per method + path; a path that serves several methods has one row each. The permission/scope notes were read from the handlers and from
  `cmd/server/{client_scope,platform_scope}.go`. The `/` single-page-app file server is not an API route and is left out.
- **`apps/web` usage.** From `apps/web/src/shared/services/api-endpoints.ts` and every caller of those keys, checked **by method** and, for store actions, by whether the
  action itself has a caller (a store method that nobody calls counts as *no*). `yes` = a screen reaches it today; `no` = nothing in `apps/web/src` calls that method + path
  (the e2e suite in `apps/web/e2e` and the Go isolation suite call more routes than the web does and are not counted); `unsure` is not used for any route below: where there is
  doubt about *removal* it is expressed in the candidates section, not in this column.
- **Caller** is who may call it: `public` (no token), `staff` (a WO's team, further narrowed by permission keys), `client` (Client Portal, GET only, fixed allowlist), `platform`
  (ElProof's own team, two-factor required; role in the note), `external` (an integrated application with an `app` token, or the payment gateway).

## Summary

- **211** registered method + path routes (the contract, `docs/API_CONTRACT.md`, lists exactly these 211, and nothing else; 213 before the two removals of Tahap 9, see *Removed in Tahap 9*).
- By module: `cmd/server` 3, `identity` 11, `access` 7, `payment` 11, `billing` 11, `platform` 61, `audit` 2, `staff` 7, `clients` 7, `vendors` 28, `projects` 63.
- `apps/web` calls **194** of them, does not call **17** (7 of those are the external integration surface: the payment gateway callback, `/auth/app/token`, `/external/*`).
- Removed from the API on 2026-11-10 and **not** in the table: `POST /projects/{id}/toggle-archive`, `POST /projects/{id}/evidence/{eid}/toggle-client-visible`,
  `POST /vendors|venues|vendor-categories|clients|staff/{id}/toggle-active` (ADR-0036; they answer 404), and `POST /tenants/{id}/reset-credential`, `POST /clients/{id}/reset-credential`
  (replaced by `owner-invite` and `access-link`, ADR-0025), and, in Tahap 9, `GET /vendors/summary` and `GET /projects/{id}/vendors/{projectVendorId}/milestones` (see *Removed in Tahap 9*).

## Every registered route

| # | Method | Path (under `/api/v1` unless noted) | Module | Caller | `apps/web` calls it? | Notes |
|---|---|---|---|---|---|---|
| 1 | GET | `/health` | cmd/server | public (deploy probe) | no | docker healthcheck + e2e use it (docs/DEPLOYMENT.md) |
| 2 | GET | `/public/config` | cmd/server | public | yes | googleClientId |
| 3 | GET | `/auth/me` | cmd/server | any authenticated | no | token claims; on the client allowlist and the MFA-enrolment allowlist |
| 4 | POST | `/auth/login` | identity | public | yes |  |
| 5 | POST | `/auth/refresh` | identity | public | yes |  |
| 6 | POST | `/auth/logout` | identity | public | yes |  |
| 7 | POST | `/auth/mfa/verify` | identity | public | yes |  |
| 8 | POST | `/auth/mfa/setup` | identity | platform | yes |  |
| 9 | POST | `/auth/mfa/enable` | identity | platform | yes |  |
| 10 | POST | `/auth/google` | identity | public | yes |  |
| 11 | POST | `/auth/google/link` | identity | public | yes |  |
| 12 | GET | `/auth/links/{token}` | identity | public | yes |  |
| 13 | POST | `/auth/links/{token}/google` | identity | public | yes |  |
| 14 | POST | `/auth/links/{token}/password` | identity | public | yes |  |
| 15 | GET | `/access/me` | access | staff, client | yes |  |
| 16 | GET | `/access/catalog` | access | staff | yes | any staff role |
| 17 | GET | `/access/roles` | access | staff | yes | role.manage, team.manage or client.manage |
| 18 | POST | `/access/roles` | access | staff | yes | role.manage |
| 19 | GET | `/access/roles/{id}` | access | staff | no | web edits from the list it already holds |
| 20 | PATCH | `/access/roles/{id}` | access | staff | yes | role.manage |
| 21 | DELETE | `/access/roles/{id}` | access | staff | yes | role.manage |
| 22 | POST | `/webhooks/payment` (no `/api/v1` prefix) | payment | external (payment gateway, HMAC) | no | not under /api/v1; Tripay callback |
| 23 | GET | `/payment/gateway-config` | payment | platform | yes |  |
| 24 | PATCH | `/payment/gateway-config` | payment | platform | yes | Super Admin |
| 25 | GET | `/payment/apps` | payment | platform | yes |  |
| 26 | POST | `/payment/apps` | payment | platform | yes | Super Admin |
| 27 | POST | `/payment/apps/{appId}/reset-secret` | payment | platform | yes | Super Admin |
| 28 | POST | `/payment/apps/{appId}/toggle-active` | payment | platform | yes | Super Admin; Console toggle that stays |
| 29 | POST | `/auth/app/token` | payment | external (app) | no | public, rate limited; external integration |
| 30 | POST | `/external/payments/charges` | payment | external (app) | no | external integration |
| 31 | GET | `/external/payments/charges/{orderRef}/status` | payment | external (app) | no | external integration |
| 32 | GET | `/external/payments/channels` | payment | external (app) | no | external integration |
| 33 | GET | `/plans` | billing | platform, staff | yes | principal-aware catalog |
| 34 | POST | `/plans` | billing | platform | yes | Billing / Super Admin |
| 35 | PATCH | `/plans/{id}` | billing | platform | yes | Billing / Super Admin |
| 36 | POST | `/plans/{id}/toggle-active` | billing | platform | yes | Billing / Super Admin; Console toggle that stays |
| 37 | GET | `/subscription-transactions` | billing | platform, staff | yes | staff need subscription.manage |
| 38 | GET | `/subscription-transactions/{id}` | billing | platform, staff | yes | receipt page |
| 39 | GET | `/external/subscription-plans` | billing | external (app) | no | external integration |
| 40 | POST | `/external/subscriptions/charges` | billing | external (app) | no | external integration |
| 41 | GET | `/app-plans` | billing | platform | yes | PUT: Billing / Super Admin |
| 42 | PUT | `/app-plans` | billing | platform | yes | PUT: Billing / Super Admin |
| 43 | GET | `/app-subscriptions` | billing | platform | yes |  |
| 44 | GET | `/tenants` | platform | platform | yes |  |
| 45 | POST | `/tenants` | platform | platform | yes | Support / Super Admin |
| 46 | GET | `/tenants/{id}` | platform | platform | no | list rows are enough for the web |
| 47 | PATCH | `/tenants/{id}` | platform | platform | yes | Support / Super Admin |
| 48 | PUT | `/tenants/{id}/logo` | platform | platform | yes |  |
| 49 | GET | `/tenants/{id}/logo` | platform | platform | yes |  |
| 50 | POST | `/tenants/{id}/toggle-suspension` | platform | platform | yes | Super Admin; Console toggle that stays |
| 51 | POST | `/tenants/{id}/owner-invite` | platform | platform | yes | Support / Super Admin |
| 52 | POST | `/tenants/{id}/activate-subscription` | platform | platform | yes | Billing / Super Admin |
| 53 | GET | `/tenants/export` | platform | platform | yes | CSV |
| 54 | GET | `/tenants/names` | platform | platform | no | replaced by tenantName on /subscription-transactions |
| 55 | GET | `/tenants/me` | platform | staff | yes | subscription.manage or business_profile.manage |
| 56 | GET | `/tenants/me/subscription` | platform | staff | yes | any staff role |
| 57 | PATCH | `/tenants/me/profile` | platform | staff | yes | business_profile.manage |
| 58 | PUT | `/tenants/me/logo` | platform | staff | yes | business_profile.manage |
| 59 | GET | `/tenants/me/logo` | platform | staff, client | yes |  |
| 60 | GET | `/tenants/me/business-details` | platform | staff | yes | business_profile.manage |
| 61 | PUT | `/tenants/me/business-details` | platform | staff | yes | business_profile.manage |
| 62 | GET | `/tenants/me/contact` | platform | staff, client | yes |  |
| 63 | PATCH | `/tenants/me/brand-color` | platform | staff | yes | business_profile.manage |
| 64 | GET | `/tenants/me/branding` | platform | staff, client | yes |  |
| 65 | GET | `/tenants/me/onboarding` | platform | staff | yes | business_profile.manage |
| 66 | POST | `/tenants/me/onboarding/dismiss` | platform | staff | yes | business_profile.manage |
| 67 | GET | `/tenants/me/support-access` | platform | staff | yes | business_profile.manage |
| 68 | PUT | `/tenants/me/support-access` | platform | staff | yes | business_profile.manage |
| 69 | DELETE | `/tenants/me/support-access` | platform | staff | yes | business_profile.manage |
| 70 | POST | `/subscriptions/pay` | platform | staff | yes | subscription.manage |
| 71 | GET | `/subscriptions/pending-charge` | platform | staff | yes | subscription.manage |
| 72 | POST | `/subscriptions/pending-charge/cancel` | platform | staff | yes | subscription.manage |
| 73 | GET | `/trial-limits` | platform | platform | yes | PUT: Billing / Super Admin |
| 74 | PUT | `/trial-limits` | platform | platform | yes | PUT: Billing / Super Admin |
| 75 | GET | `/platform/metrics` | platform | platform | yes |  |
| 76 | GET | `/platform/trends` | platform | platform | yes |  |
| 77 | GET | `/announcements/active` | platform | staff | yes |  |
| 78 | GET | `/public/login-slides` | platform | public | yes |  |
| 79 | GET | `/public/login-slides/{index}` | platform | public | yes |  |
| 80 | POST | `/public/signup/workspace` | platform | public | yes | rate limited |
| 81 | GET | `/console/tenants/{id}/overview` | platform | platform | yes |  |
| 82 | GET | `/console/tenants/{id}/notes` | platform | platform | yes |  |
| 83 | POST | `/console/tenants/{id}/notes` | platform | platform | yes |  |
| 84 | GET | `/console/tenants/{id}/data-requests` | platform | platform | yes |  |
| 85 | POST | `/console/tenants/{id}/data-requests` | platform | platform | yes |  |
| 86 | GET | `/console/tenants/{id}/support` | platform | platform | yes |  |
| 87 | POST | `/console/tenants/{id}/impersonate` | platform | platform | yes | Support / Super Admin |
| 88 | GET | `/console/data-requests` | platform | platform | yes |  |
| 89 | POST | `/console/data-requests/{id}/complete` | platform | platform | yes |  |
| 90 | GET | `/console/announcements` | platform | platform | yes | Super Admin to write |
| 91 | POST | `/console/announcements` | platform | platform | yes | Super Admin to write |
| 92 | PATCH | `/console/announcements/{id}` | platform | platform | yes | Super Admin |
| 93 | DELETE | `/console/announcements/{id}` | platform | platform | yes | Super Admin |
| 94 | GET | `/console/subscription-policy` | platform | platform | yes | PUT: Super Admin |
| 95 | PUT | `/console/subscription-policy` | platform | platform | yes | PUT: Super Admin |
| 96 | POST | `/console/support-sessions/{id}/end` | platform | platform | yes |  |
| 97 | PUT | `/console/login-slides/{index}` | platform | platform | yes | Super Admin |
| 98 | GET | `/platform-admins` | platform | platform | yes |  |
| 99 | POST | `/platform-admins` | platform | platform | yes | Super Admin |
| 100 | GET | `/platform-admins/{id}` | platform | platform | yes | PATCH: Super Admin |
| 101 | PATCH | `/platform-admins/{id}` | platform | platform | yes | PATCH: Super Admin |
| 102 | POST | `/platform-admins/{id}/toggle-active` | platform | platform | yes | Super Admin; Console toggle that stays |
| 103 | POST | `/platform-admins/{id}/reset-password` | platform | platform | yes | Super Admin |
| 104 | POST | `/platform-admins/{id}/reset-mfa` | platform | platform | yes | Super Admin |
| 105 | GET | `/audit-log` | audit | platform | yes | Super Admin |
| 106 | GET | `/audit-log/export` | audit | platform | yes | Super Admin; CSV |
| 107 | GET | `/staff` | staff | staff | yes | team.manage |
| 108 | POST | `/staff` | staff | staff | yes | team.manage |
| 109 | GET | `/staff/summary` | staff | staff | yes | any staff role |
| 110 | PATCH | `/staff/{id}` | staff | staff | yes | team.manage; PATCH {isActive} = state |
| 111 | DELETE | `/staff/{id}` | staff | staff | yes | team.manage; PATCH {isActive} = state |
| 112 | POST | `/staff/{id}/invite` | staff | staff | yes | team.manage |
| 113 | GET | `/staff/{id}/delete-impact` | staff | staff | yes | team.manage |
| 114 | GET | `/clients` | clients | staff | yes | client.manage; GET has three shapes |
| 115 | POST | `/clients` | clients | staff | yes | client.manage; GET has three shapes |
| 116 | PATCH | `/clients/{id}` | clients | staff | yes | client.manage; PATCH {isActive} = state; DELETE is soft |
| 117 | DELETE | `/clients/{id}` | clients | staff | yes | client.manage; PATCH {isActive} = state; DELETE is soft |
| 118 | POST | `/clients/{id}/restore` | clients | staff | yes | client.manage |
| 119 | POST | `/clients/{id}/access-link` | clients | staff | yes | client.manage |
| 120 | POST | `/clients/{id}/replace-representative` | clients | staff | yes | client.manage |
| 121 | GET | `/vendor-categories` | vendors | staff | yes | vendor.category.manage (paginated, POST); ?all=true any staff |
| 122 | POST | `/vendor-categories` | vendors | staff | yes | vendor.category.manage (paginated, POST); ?all=true any staff |
| 123 | PATCH | `/vendor-categories/{id}` | vendors | staff | yes | vendor.category.manage |
| 124 | DELETE | `/vendor-categories/{id}` | vendors | staff | yes | vendor.category.manage |
| 125 | GET | `/vendors` | vendors | staff | yes | vendor.view / vendor.manage |
| 126 | POST | `/vendors` | vendors | staff | yes | vendor.view / vendor.manage |
| 127 | POST | `/vendors/bulk` | vendors | staff | yes | vendor.manage |
| 128 | GET | `/vendors/template` | vendors | staff | yes | vendor.manage |
| 129 | POST | `/vendors/import` | vendors | staff | yes | vendor.manage |
| 130 | GET | `/vendors/export` | vendors | staff | yes | vendor.view |
| 131 | PATCH | `/vendors/{id}` | vendors | staff | yes | vendor.manage / vendor.delete; PATCH {isActive} = state |
| 132 | DELETE | `/vendors/{id}` | vendors | staff | yes | vendor.manage / vendor.delete; PATCH {isActive} = state |
| 133 | GET | `/vendors/{id}/project-history` | vendors | staff | yes | vendor.view |
| 134 | GET | `/vendors/{id}/attachment` | vendors | staff | yes | PUT: vendor.manage |
| 135 | PUT | `/vendors/{id}/attachment` | vendors | staff | yes | PUT: vendor.manage |
| 136 | GET | `/vendors/{id}/delete-impact` | vendors | staff | yes | vendor.delete |
| 137 | GET | `/venues` | vendors | staff | yes | vendor.view / vendor.manage |
| 138 | POST | `/venues` | vendors | staff | yes | vendor.view / vendor.manage |
| 139 | POST | `/venues/bulk` | vendors | staff | yes | vendor.manage |
| 140 | GET | `/venues/template` | vendors | staff | yes | vendor.manage |
| 141 | POST | `/venues/import` | vendors | staff | yes | vendor.manage |
| 142 | GET | `/venues/export` | vendors | staff | yes | vendor.view |
| 143 | GET | `/venues/{id}` | vendors | staff | yes | PATCH {isActive} = state; DELETE vendor.delete |
| 144 | PATCH | `/venues/{id}` | vendors | staff | yes | PATCH {isActive} = state; DELETE vendor.delete |
| 145 | DELETE | `/venues/{id}` | vendors | staff | yes | PATCH {isActive} = state; DELETE vendor.delete |
| 146 | GET | `/venues/{id}/attachment` | vendors | staff, client | yes | client: images only |
| 147 | PUT | `/venues/{id}/attachment` | vendors | staff, client | yes | client: images only |
| 148 | GET | `/venues/{id}/delete-impact` | vendors | staff | yes | vendor.delete |
| 149 | GET | `/projects` | projects | staff | yes | project.create; Idempotency-Key on POST |
| 150 | POST | `/projects` | projects | staff | yes | project.create; Idempotency-Key on POST |
| 151 | GET | `/projects/me` | projects | client | no | replaced by /projects/me/home; e2e asserts the web does not call it |
| 152 | GET | `/projects/me/home` | projects | client | yes |  |
| 153 | GET | `/projects/{id}` | projects | staff, client | no | web uses /overview (staff) and /me/home (client) |
| 154 | PATCH | `/projects/{id}` | projects | staff | yes | partial; If-Match required unless only isArchived |
| 155 | DELETE | `/projects/{id}` | projects | staff | yes | project.delete; hard delete |
| 156 | GET | `/projects/{id}/overview` | projects | staff | yes |  |
| 157 | POST | `/projects/{id}/cancel` | projects | staff | yes |  |
| 158 | POST | `/projects/{id}/transition` | projects | staff | yes |  |
| 159 | GET | `/projects/{id}/payment-terms` | projects | staff, client | yes | GET: client with portal.finance.view; PUT: finance.record |
| 160 | PUT | `/projects/{id}/payment-terms` | projects | staff, client | yes | GET: client with portal.finance.view; PUT: finance.record |
| 161 | GET | `/projects/{id}/closing-checklist` | projects | staff | yes |  |
| 162 | POST | `/projects/{id}/duplicate` | projects | staff | yes | project.duplicate |
| 163 | GET | `/projects/{id}/venue` | projects | staff, client | yes |  |
| 164 | GET | `/projects/{id}/milestones` | projects | staff, client | yes |  |
| 165 | POST | `/projects/{id}/milestones` | projects | staff, client | yes |  |
| 166 | PATCH | `/projects/{id}/milestones` | projects | staff | yes | reorder |
| 167 | PATCH | `/projects/{id}/milestones/{milestoneId}` | projects | staff | yes | partial; If-Match required |
| 168 | GET | `/projects/{id}/vendors` | projects | staff, client | yes |  |
| 169 | POST | `/projects/{id}/vendors` | projects | staff, client | yes |  |
| 170 | PATCH | `/projects/{id}/vendors/{projectVendorId}` | projects | staff | yes | DELETE is soft; no web UI for deleting an engagement yet |
| 171 | DELETE | `/projects/{id}/vendors/{projectVendorId}` | projects | staff | no | DELETE is soft; no web UI for deleting an engagement yet |
| 172 | POST | `/projects/{id}/vendors/{projectVendorId}/restore` | projects | staff | no | no web UI for deleting an engagement yet |
| 173 | POST | `/projects/{id}/vendors/{projectVendorId}/cancel` | projects | staff | yes |  |
| 174 | POST | `/projects/{id}/vendors/{projectVendorId}/milestones` | projects | staff | yes | create only; the GET was removed in Tahap 9, steps are read from milestones[] in GET /projects/{id}/vendors |
| 175 | PATCH | `/projects/{id}/vendors/{projectVendorId}/milestones/{milestoneId}` | projects | staff | yes | partial; If-Match required |
| 176 | GET | `/projects/{id}/payments` | projects | staff | yes | POST: Idempotency-Key, evidence[] with upload tokens |
| 177 | POST | `/projects/{id}/payments` | projects | staff | yes | POST: Idempotency-Key, evidence[] with upload tokens |
| 178 | PATCH | `/projects/{id}/payments/{paymentId}` | projects | staff | yes | PATCH partial + If-Match required; DELETE soft |
| 179 | DELETE | `/projects/{id}/payments/{paymentId}` | projects | staff | yes | PATCH partial + If-Match required; DELETE soft |
| 180 | POST | `/projects/{id}/payments/{paymentId}/restore` | projects | staff | yes |  |
| 181 | GET | `/projects/{id}/client-payments` | projects | staff, client | yes |  |
| 182 | POST | `/projects/{id}/client-payments` | projects | staff, client | yes |  |
| 183 | PATCH | `/projects/{id}/client-payments/{paymentId}` | projects | staff | yes |  |
| 184 | DELETE | `/projects/{id}/client-payments/{paymentId}` | projects | staff | yes |  |
| 185 | POST | `/projects/{id}/client-payments/{paymentId}/restore` | projects | staff | yes |  |
| 186 | POST | `/projects/{id}/client-payments/{paymentId}/verify` | projects | staff | yes | finance.record |
| 187 | GET | `/projects/{id}/venue-payments` | projects | staff | yes |  |
| 188 | POST | `/projects/{id}/venue-payments` | projects | staff | yes |  |
| 189 | PATCH | `/projects/{id}/venue-payments/{paymentId}` | projects | staff | yes |  |
| 190 | DELETE | `/projects/{id}/venue-payments/{paymentId}` | projects | staff | yes |  |
| 191 | POST | `/projects/{id}/venue-payments/{paymentId}/restore` | projects | staff | yes |  |
| 192 | GET | `/projects/{id}/issues` | projects | staff, client | yes |  |
| 193 | POST | `/projects/{id}/issues` | projects | staff, client | yes |  |
| 194 | PATCH | `/projects/{id}/issues/{issueId}` | projects | staff | yes | partial; If-Match required |
| 195 | GET | `/projects/{id}/evidence` | projects | staff, client | yes | POST is the base64 route |
| 196 | POST | `/projects/{id}/evidence` | projects | staff, client | yes | POST is the base64 route |
| 197 | PATCH | `/projects/{id}/evidence/{evidenceId}` | projects | staff | yes | {isClientVisible}; If-Match optional |
| 198 | DELETE | `/projects/{id}/evidence/{evidenceId}` | projects | staff | no | added by ADR-0034 so a proof file can be removed; no web UI yet |
| 199 | GET | `/projects/{id}/evidence/{evidenceId}/file` | projects | staff, client | yes |  |
| 200 | GET | `/projects/{id}/evidence/{evidenceId}/link` | projects | staff, client | yes | 501 presign_unavailable without S3_PRESIGN |
| 201 | POST | `/projects/{id}/uploads` | projects | staff | yes |  |
| 202 | PUT | `/projects/{id}/uploads/{uploadToken}` | projects | staff | yes | only when the grant says direct:false |
| 203 | GET | `/projects/{id}/documents` | projects | staff, client | yes |  |
| 204 | GET | `/projects/{id}/activity` | projects | staff | yes |  |
| 205 | GET | `/dashboard` | projects | staff | yes | dashboard.view + all-projects scope |
| 206 | GET | `/dashboard/mine` | projects | staff | yes |  |
| 207 | GET | `/milestone-templates` | projects | staff | yes | template.manage |
| 208 | POST | `/milestone-templates` | projects | staff | yes | template.manage |
| 209 | PATCH | `/milestone-templates` | projects | staff | yes | reorder |
| 210 | PATCH | `/milestone-templates/{id}` | projects | staff | yes | template.manage |
| 211 | DELETE | `/milestone-templates/{id}` | projects | staff | yes | template.manage |

Query shapes that are one route but several behaviours (relevant for the web column):

| Route | Shape | `apps/web` calls it? |
|---|---|---|
| `GET /projects` | paginated + `sort` + filters | yes |
| `GET /projects` | `?all=true` (plain array) | yes — only as a refetch after create/duplicate in `useProjectStore.fetchProjects` |
| `GET /clients` | paginated flat list | yes |
| `GET /clients` | `?projectId=` alone (project tab) | yes |
| `GET /clients` | `?all=true` (tenant-wide plain array) | **no** — the command palette and the Klien page use the paginated shape; see candidates |
| `GET /vendors`, `/venues`, `/staff`, `/vendor-categories`, `/plans` | `?all=true` pickers | yes |

## Candidates for removal

Routes that `apps/web` does not call (column above = *no*), that are **not** part of the external payment integration (`docs/PAYMENT_INTEGRATION_GUIDE.md`, the Postman
collection, `/external/*`, `/auth/app/token`, `/webhooks/payment`) and not needed by the infrastructure (`GET /health` is the deploy health check). Each says what replaced it and
how sure the reading is. **Nothing was deleted when this list was written** (the two routes removed since are under *Removed in Tahap 9*). Be conservative: where an ADR or the contract says the route stays, it is `unsure`, not a candidate.

| Route | Replaced by | Reading | Why not simply remove it |
|---|---|---|---|
| `GET /clients?all=true` (a query shape of `GET /clients`, not a route) | the paginated `GET /clients?page&limit&q&…` (ADR-0035), which the command palette also uses for global search | **candidate** | The contract used to say it powered `GlobalSearch`; that component no longer exists. Needs `client.manage` with every-project scope either way |
| `GET /projects/me` | `GET /projects/me/home` (`data.project.id` and the rest of the home in one call, ADR-0032) | **unsure** | Nothing in the web calls it (`useProjectStore.fetchMyProject` has no caller) and `e2e/tests/proof.spec.ts` asserts the portal does *not* call it. But ADR-0032 and the contract say "`GET /projects/me` stays", and it is on the client allowlist |
| `GET /projects/{id}` | `GET /projects/{id}/overview` (staff project page, which embeds exactly this object as `data.project`) and `GET /projects/me/home` (client) | **unsure** | Not called by the web (`useProjectStore.fetchProjectDetail` has no caller). It is the canonical item answer that carries the `ETag` and the shape of a 409's `data.current` (ADR-0036), is on the client allowlist, and the isolation suite calls it (`scenarios_test.go:312`, `:366`) |
| `GET /tenants/names` | `tenantName` on every row of `GET /subscription-transactions` (ADR-0035) | **unsure** | `useConsoleStore.fetchTenantNames` has no caller, but ADR-0035 says explicitly "`GET /tenants/names` stays", and the isolation suite checks its workspace scoping (`scenarios_tahap6_test.go:1242`) |
| `GET /tenants/{id}` | the row already held from `GET /tenants` | **unsure** | The Console edits from the list row; nobody calls the single read. It is the obvious read counterpart of `PATCH /tenants/{id}`; the isolation suite calls it (`isolation_test.go:231`) |
| `GET /access/roles/{id}` | the role already held from `GET /access/roles` | **unsure** | Same: the web edits from the list. Counterpart of `PATCH`/`DELETE /access/roles/{id}`; the isolation suite calls it (`isolation_test.go:221`) |
| `DELETE /projects/{id}/vendors/{projectVendorId}` and `POST …/restore` | — (no replacement; the web has no "remove vendor from project" action yet) | **unsure** | Added deliberately by ADR-0034 / PLAN D18 (soft delete with "Urungkan"). The back half is finished, the web half for vendor engagements is not. Removing it would undo a decision, not a leftover |
| `DELETE /projects/{id}/evidence/{evidenceId}` | — | **unsure** | Added by ADR-0034 so that `EvidenceService.Delete` (and the rule "a verified payment loses its verification with its last file", ADR-0032) has an HTTP route; no web screen deletes a single proof file yet |
| `GET /auth/me` | — (nothing in the web asks for it; `GET /access/me` supplies the permissions) | **unsure** | It is the one route a platform admin without two-factor may call besides `mfa/setup` and `mfa/enable`, and it is on the client allowlist; the isolation suite calls it (`scenarios_tahap5_test.go:241`) |

### Removed in Tahap 9

Both were *candidates* above, nothing in `apps/web` called them any more, and they are no longer registered (the row numbers of the table were shifted down accordingly).

| Route | Replaced by | What stays |
|---|---|---|
| `GET /vendors/summary` | `vendorName` / `categoryName` inside `GET /projects/{id}/vendors` and `/issues` (ADR-0032); the staff pickers use `GET /vendors?all=true` | The path stays off the Client Portal allowlist (`cmd/server/client_scope.go`, tested in `client_scope_test.go` and in the isolation suite): a client still gets 403 on it, staff get no 200 (`GET /vendors/{id}` answers 400 for the non-numeric id). `GET /staff/summary` is untouched |
| `GET /projects/{id}/vendors/{projectVendorId}/milestones` | `milestones[]` embedded in every item of `GET /projects/{id}/vendors` | `POST` (create) and `PATCH …/{id}` on the same path. The isolation suite now reads vendor milestones from the embedded list |

Also in Tahap 9: `GET /vendors/export` and `GET /venues/export` read the same filters as their lists (`q`/`search`, `categoryId` (vendors), `city`, `isActive`).

Dead code on the web side that these findings point to (not touched here): `useProjectStore.fetchMyProject`, `useProjectStore.fetchProjectDetail`,
`useVendorStore.fetchVendorSummaries`, `useConsoleStore.fetchTenantNames`.

### Not candidates although the web does not call them

- The external surface: `POST /webhooks/payment`, `POST /auth/app/token`, `GET|POST /external/payments/*`, `GET /external/subscription-plans`, `POST /external/subscriptions/charges` (7 rows marked *no*, caller `external`).
- `GET /health` — the container health check (`./api healthcheck`) and the e2e harness poll it (`docs/DEPLOYMENT.md`).
