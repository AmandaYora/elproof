# ADR-0041: Subscription vouchers (percent or amount, per App)

## Status
Accepted — implemented (T-20261110-17). Builds on ADR-0040 (a plan belongs to one App).

## Context

The owner asked for voucher codes that lower the price of a subscription, as a percentage or a fixed amount. Decisions
taken with the owner (`handoff/T-20261110-17/pemahaman.md`): a voucher belongs to one App and may be narrowed to some
of its plans; it can have a validity period, a total quota, a once-per-customer rule and, for a percentage, a cap; the
final price is never Rp0 (a free period stays the admin's "Aktifkan Langganan"). Codes are used in two places: a WO
tenant paying in Langganan (`/subscriptions/pay`), and an external App creating a subscription charge
(`/external/subscriptions/charges`). Vouchers for general payments (`/external/payments/charges`) are out of scope.

## Decision

- **Owner: `billing`.** Vouchers discount plan prices, which `billing` owns; `payment` stays blind to plans and
  vouchers and only receives the final amount. Two tables (migration `000074`): `subscription_vouchers`
  (`app_id` primitive, `plan_id` FK to `subscription_plans`, NULL = every plan of the App) and
  `subscription_voucher_redemptions` (one row per use, keyed by the charge's `order_ref`).
- **One plan at most.** "Narrowed to some plans" is one plan or all plans of the App — enough for the cases named and
  far simpler than a many-to-many list; two plans = two codes.
- **Who is the customer** (once-per-customer): a WO **tenant** for ElProof's own catalog; for an external App, the
  **App itself** — the subscriber its ledger (`subscription_app_charges`) and its subscription period
  (`CurrentUntilForApp`) are keyed on. The charge contract has no required identity of a person (`customerEmail` is
  optional), so it is not used.
- **When a use counts: from the moment the charge is created.** A use is `reserved` while the charge is pending and
  counts against the quota and the once-per-customer rule; it becomes `used` when paid and `released` (no longer
  counted) when the charge expires, fails or is cancelled. Counting at payment instead would let several pending
  charges all pass the last unit of a quota and then all be paid; counting only at creation without release would let
  abandoned charges eat the quota. The reservation is taken under `SELECT … FOR UPDATE` of the voucher row in a
  READ COMMITTED transaction (lock first, then read the counts in a new statement), so concurrent charges can never
  over-use a quota or a once-per-customer code (tested with 20 concurrent reservations).
- **Settlement.** WO: `UpdateTransactionStatus` (paid → used, expired/cancelled → released). App: the subscription
  reconciler settles a use when it resolves the charge. If no charge comes out of a request (refused code, gateway or
  ledger error), the reservation is deleted at once; a reservation left by a process that died is released by the
  reconciler after an hour when no pending/paid charge has its `order_ref`.
- **Never Rp0.** A percentage is 1–99; a fixed amount must be below the plan price when the voucher names a plan; at
  use time any discount that reaches the whole price is refused (`voucher_too_large`), not capped.
- **Refusals** are a `422` on `voucherCode` with a ready-to-show Indonesian message and a stable code
  (`voucher_not_found | inactive | not_started | expired | exhausted | already_used | wrong_plan | too_large`); external
  routes send it as `errors.reason`. A code of another App answers exactly like an unknown code.
- **Preview endpoints:** `POST /subscriptions/voucher-check` (WO Owner) and `POST /external/subscriptions/voucher-check`
  (App) reserve nothing; paying checks again.
- **Ledgers record the discount:** `subscription_transactions.original_amount/discount_amount/voucher_code` and
  `subscription_app_charges.discount_amount/amount/voucher_code`; `amount` stays what was charged, so revenue reports
  are unchanged.
- **Console:** Platform Console › Voucher (Billing or Super Admin may write, every admin may read): list, form,
  explicit activate/deactivate (ADR-0036; full edit with `If-Match`), and the list of uses per voucher. The App of a
  voucher cannot change once it has been used. There is no delete.
- **Dates** are inclusive WIB calendar days.

## Consequences

- A pending charge holds a unit of the quota until it resolves (WO: the gateway's expiry, the Owner's "Batalkan" or
  the admin's activation; App: `SUBSCRIPTION_CHARGE_MAX_AGE`).
- An external App can use a once-per-customer code only once in total, since the App is the customer.
- A voucher edited after a use does not change that use: amounts are snapshotted on the redemption and the ledger.
