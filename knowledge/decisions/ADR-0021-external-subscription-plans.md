# ADR-0021: Isolated per-App plan catalog and subscription ledger for external Apps

## Status
Accepted — implemented.

## Context

`GET /api/v1/external/subscription-plans` was called by jwswedding's already-shipped client
(`internal/shared/elproofpay/client.go`) and 404'd — nothing on the ElProof side had ever built it.
Closing the 404 alone would not have been correct: the plan catalog (`subscription_plans`) is
global, with no column linking a plan to a specific external App, so a naive implementation would
have returned ElProof's own single demo plan to every App that asked — no isolation between JWS
Wedding, a future "XXX Wedding," and Elkasir (which uses the gateway to bill its own customers and
owns no ElProof plan at all).

A first design-review pass on the plan for this work found the isolation wasn't actually achieved
even with a naive per-App mapping table: `PlanHandler.list` (`GET /plans`) had no principal guard at
all, so a plan marked "private" to one App would still be readable by any authenticated principal,
including a WO tenant's own Owner via `SubscriptionPage` (T9). Separately, feature details never
reached the external contract at all (T10), and ElProof had no record of which external App was
paying for which plan or until when — Manajemen Aplikasi showed registration metadata only, nothing
about subscriptions (T13). A second verification pass, after the isolation fix, found the fix itself
opened a new leak: the same "platform_admin sees every plan" rule the isolation fix required also
fed the tenant-activation picker, so a private App plan could be assigned to a WO tenant, silently
locking that tenant out of its own subscription page on the very next load (T16). Full trace: `docs/
plan/external-subscription-plans/PLAN.md`.

## Decision

### `billing` owns the isolation, not `payment` (D1)
`billing` already owns `subscription_plans`; "which App may see which plan" is a catalog-visibility
question, not a payment-gateway one. Consequence accepted: the external API surface for one App-
token principal is now split across two modules (`payment`'s three `/external/payments/*` routes,
`billing`'s two `/external/subscription-plans` / `/external/subscriptions/charges` routes) —
`billing` receives `payment.contracts.AppRegistry` and `payment.contracts.Client` at construction
time (A5, same direct-contracts-injection shape `platform` already uses for `payment.Client`), not
a duplicated local interface. See `knowledge/MODULE_MAP.md`.

### A join table, not a column — `subscription_plan_apps` (R1)
`app_id` × `plan_id`, `UNIQUE(app_id, plan_id)`, `plan_id` a same-module FK with `ON DELETE CASCADE`,
`app_id` a primitive cross-module ID with **no** FK to `payment_apps` (A2 — `.claude/rules/
database.md` forbids cross-module FKs). The plan catalog itself stays global; only its visibility is
scoped. Elkasir (which owns no ElProof plan) simply has no rows and receives `data: []`, `200` — a
valid state (R2), not an error (A4).

### Visibility is derived from the mapping's existence, not a new column (D3/D6)
A plan with a row in `subscription_plan_apps` is private to that App and hidden from the WO tenant
catalog; a plan with no row is ElProof's own public catalog. An `is_public` boolean was explicitly
rejected (D6): production has exactly one plan today, shared between the WO demo tenant and JWS's
real subscription — a boolean would keep them permanently tied to one price. Instead, JWS was given
its **own** plan (seed migration `000045`, values copied from the shared plan as it stands: A15),
mapped to `app_02ef90c52704`; the original plan (id 1) stays unmapped so the demo tenant keeps
seeing it. The two are independent from that point on.

### `planResponse` gains an additive `appIds` field, populated only for `platform_admin` (D12)
This is the fix for T16, and it's the opposite of the first instinct (hide private plans from
`platform_admin` entirely). That would have broken Platform Console's own ability to manage the
mapping it had just created — `platform_admin` legitimately needs to see every plan (T9's fix,
below). `appIds` lets the *frontend* tell public and private plans apart without the backend ever
hiding data from the one principal that must see it all: the tenant-activation picker filters to
`appIds.length === 0`, and the Paket page groups by `appIds` into two visual sections. Every other
principal always receives `appIds: []` — correct as-is, since the plans they can see are, by
construction, never mapped to anything.

### `GET /plans` becomes principal-aware, not access-denied (D3/A12, closes T9)
`platform_admin` → sees every plan (`List`/`ListPaginated`); every other principal, **including no
claim at all** (the safe default) → sees only unmapped plans (`ListPublic`/`ListPublicPaginated`).
Applied to both the `?all=true` and paginated code paths — a leak fixed in only one is still a leak.
Locking the whole endpoint to `platform_admin` was rejected: WO tenants' own `SubscriptionPage`
reads this same endpoint and must keep working unmodified (D5 — ElProof's own subscription flow must
never break for this).

### A new subscription-purchase endpoint, not a field on the existing charge endpoint (D9)
`POST /external/subscriptions/charges` — records intent (`app_id`, `plan_id`, `order_ref`, a price/
duration snapshot) **before** calling `payment.Client.CreateChannelCharge`, so a process death
between those two steps leaves a recoverable "ghost" row rather than an untracked live charge (A20,
mirroring `docs/plan/migrasi-data-jws/PLAN.md`'s D16 for the same failure class). Price and duration
come from the catalog server-side (A18) — the request body carries no `amount` at all, so an App can
never determine what it pays for a plan it's entitled to buy. `planId` not mapped to the calling App
→ `422`, even if the caller knows a valid id belonging to another App (R1/R2) — this check, not the
catalog endpoint, is the real guard, since `planId` values are guessable integers.

### The ledger reconciles independently of `payment`'s own dispatch, not by extending it (A16, closes T15)
`payment`'s `HandleWebhook`/`reconcileOne` both branch on `app.Kind` to decide whether a terminal
result goes to an in-process consumer or is relayed externally — an **exclusive** branch, duplicated
in two places, that today processes real tenant money through `platform-billing`. Teaching it to
also write to `subscription_app_charges` would mean editing that live path twice, identically. A
`billing`-owned `SubscriptionReconciler` polls `payment.Client.CheckStatusForApp` on its own interval
(`SUBSCRIPTION_RECONCILE_INTERVAL`, independent from `PAYMENT_RECONCILE_INTERVAL`) instead — an
isolated, additive component. Accepted consequence: the two reconcilers call the gateway
independently for the same `order_ref` around the same time; this is expected, not a bug (see
`docs/DEPLOYMENT.md` §7).

### One atomic UPDATE per terminal outcome, not claim-then-write
`MarkPaidIfUnresolved`/`MarkResolvedIfUnresolved` each set `status` (and, for the paid case,
`subscription_until`) **and** `resolved_at` in a single `UPDATE ... WHERE resolved_at IS NULL`,
returning whether this call's `RowsAffected` was 1. A separate `ClaimUnresolved`-then-write step
(the pattern `payment`'s own reconciler uses, safely, because its post-claim action is a dispatch,
not a database write) was rejected here: a process death between claim and write would leave a row
permanently `resolved_at`-set but `status='pending'` — invisible to `FindUnresolved` forever, a
silently lost payment.

### `subscription_app_charges` is a new table, not a reuse of `subscription_transactions`
`subscription_transactions.tenant_id` is `NOT NULL` and is read by the live WO-tenant subscription
flow; an App-keyed row has no tenant. Reusing it would mean loosening that column's nullability on a
table already carrying production traffic — a real gap, not a style preference.

### Handler constructors take narrow interfaces, not the concrete service (A34)
Every new/touched handler (`ExternalPlanHandler`, `AppPlanHandler`, `ExternalSubscriptionHandler`,
`AppSubscriptionHandler`, and `PlanHandler` itself) declares its own small interface in its own file
and accepts that instead of `*application.PlanService`/`*application.SubscriptionService` directly.
`application.PlanService` is a concrete struct with no interface anywhere in `billing/presentation`
today — without this, none of the principal-branching or guard logic this ADR describes could be
unit-tested at all. Wiring in `billing.module.go` is unaffected: the concrete services satisfy these
interfaces implicitly.

## Consequences
- Migrations `000043` (`subscription_plan_apps`), `000044` (`subscription_app_charges`), `000045`
  (seeds JWS's own plan) — three files, not one, specifically so the seed can be rolled back on its
  own without dropping the ledger table (`migrate down` only ever steps back one migration). See
  `docs/DB_SCHEMA.md`.
- New routes: `GET /external/subscription-plans`, `POST /external/subscriptions/charges`,
  `GET`/`PUT /app-plans`, `GET /app-subscriptions`. See `docs/API_CONTRACT.md`.
- `GET /plans` is now principal-aware and `planResponse` carries an additive `appIds` field — every
  existing consumer of this endpoint keeps working unmodified except for the plans it can no longer
  see if it isn't `platform_admin` (the intended fix).
- Platform Console: Manajemen Aplikasi's table dropped from 7 columns to 4 (Nama/Jenis/Status/Aksi)
  in favor of a "Detail" action opening a tabbed modal (Info/Paket/Langganan) — the same information
  is one click away, not gone. The Paket page is now two sections ("Katalog Publik ElProof" /
  "Paket Privat per App") instead of one flat table.
- Two independent reconciler loops now run in the server process (`payment`'s existing one,
  `billing`'s new one) — see `docs/DEPLOYMENT.md` §7 for what that looks like in logs.
- `payment/contracts` gained one new export, `AppRegistry` (`IsAppActive`/`AppExists`) — `payment`'s
  own non-goal of staying unaware of tenants/plans/business concepts (`knowledge/MODULE_PAYMENT.md`
  §2.2) is unaffected, since this is a read-only App-status check, not business logic.
- `contracts.Client`'s `contextID` parameter was confirmed dead code (never read, no `context%`
  column anywhere) but **not removed** — its signature is shared with `platform`, out of scope for
  this change. `knowledge/MODULE_PAYMENT.md` now documents this explicitly so a future reader
  doesn't build something on top of it.
- Everything the jwswedding (or any future App) client needs to actually consume `features` and
  switch its purchase flow to the new endpoint is out of scope here (D10) — recorded as a handoff
  spec in `docs/plan/external-subscription-plans/PLAN.md` §8 for that repo's own, separate plan.
  Every change on the ElProof side is additive, so this is safe to leave pending indefinitely.
