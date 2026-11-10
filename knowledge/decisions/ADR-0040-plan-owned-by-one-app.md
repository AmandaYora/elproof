# ADR-0040: A subscription plan belongs to exactly one App

## Status
Accepted — implemented (T-20261110-16). Supersedes the visibility-mapping part of ADR-0021 (the
`subscription_plan_apps` join table, "unmapped = public", the `appIds` field and `/app-plans`). The
rest of ADR-0021 — the external catalog, the subscription-charge endpoint and its ledger — stands.

## Context

ADR-0021 kept plans global and shared them with Apps through a many-to-many mapping edited in
Manajemen Aplikasi › Detail › Paket (checkboxes). A plan without a mapping was ElProof's own public
catalog. In use this was hard to read: the same plan could be ticked for several Apps, "public" was
the absence of a row, and the Paket form never said who a plan was for. The owner asked for plans to
be made *per App*: several Apps use the payment gateway and each sells its own plans. ElProof is still
in development and its data is dummy data, so no compatibility path was needed.

## Decision

- `subscription_plans.app_id VARCHAR(100) NOT NULL` (migration `000073`): the one App that sells the
  plan. It is a primitive cross-module ID with **no FK** to `payment_apps` (same as
  `subscription_app_charges.app_id`); `POST`/`PATCH /plans` check it against
  `payment.contracts.AppRegistry.AppExists` and answer `422` on `appId` otherwise.
- **ElProof is an App like the others**: its plans are the plans of its internal App
  `platform-billing` (`paymentcontracts.InternalAppBilling`) — the App `platform` already charges WO
  tenant subscriptions through. No special value, no extra column. `payment` stays unaware of plans:
  only `billing` decides that "ElProof's catalog" = `billing/application.TenantPlansApp`. That App is
  always accepted as an owner without asking the registry (it is bootstrapped at every start, and
  `api seed` empties the registry until then).
- Who sees what:
  - `GET /plans` — `platform_admin`: every App's plans, or one App's with `?appId=` (all and
    paginated). Anyone else, with or without `?appId=`: ElProof's catalog only.
  - `GET /public/plans` — ElProof's active plans.
  - `GET /external/subscription-plans` — the calling App's plans only.
  - `POST /external/subscriptions/charges` — a plan of another App answers `422`, exactly like a plan
    that does not exist.
  - WO tenants: `billing.Contracts.GetTenantPlan` returns only ElProof plans (`422` on `planId`
    otherwise); `platform` uses it for self-service payment (`/subscriptions/pay`) and manual
    activation (`/tenants/{id}/activate-subscription`). Reading the plan a tenant already has still
    uses `GetPlan`.
- `/app-plans` and `subscription_plan_apps` are removed. Migration `000073` moves the data: an
  unmapped plan → `platform-billing`; a mapped plan → its first App (`MIN(app_id)`). The down
  migration recreates the table with one row per non-ElProof plan.
- Console: the Paket form requires choosing the App; the Paket page shows one section per App and
  filters by App on the server (`?aplikasi=` in the URL → `GET /plans?appId=`); Manajemen Aplikasi ›
  Detail › Paket lists the App's own plans and adds one with the App fixed. The manual-activation
  picker and announcement audience list ElProof plans only.

## Consequences

- One plan can no longer be shared by two Apps: an App that needs the same terms gets its own plan
  (that was already the recommendation of ADR-0021 D6 for JWS).
- Changing a plan's App is allowed in the form; a WO tenant already on an ElProof plan that is moved
  away keeps its `plan_id` but cannot renew it (the payment and activation guards refuse it).
- Vouchers (next task) can hang off the same key: a voucher table owned by `billing` with `app_id`
  (and an optional `plan_id`), checked against the plan's own `app_id`.
