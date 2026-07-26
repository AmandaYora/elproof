# PLAN — Plan-less tenant creation + card-based subscription + in-history charge recovery

Status: **Implemented.** Backend and frontend changes described below have been built,
build/type-checked, and interactively verified (Playwright). Kept here as the design record.

## 1. Background

Today, creating a tenant (Platform Console → Tenant → Tambah Tenant) requires picking a
subscription plan and a password in the same form. `TenantService.Register` immediately records a
transaction row with `status = unpaid` (label "Menunggu Pembayaran") for that plan — before the
Owner has done anything and before any real money/gateway charge exists.

This creates a permanent data-hygiene bug: if the Platform Console later activates that tenant's
subscription manually (`ActivateSubscription`), it records a **second**, independent transaction
row (`status = granted`, method "Aktivasi Manual (Super Admin)") — but never touches or resolves
the original "unpaid" row, because that row was never tracked in `pending_subscription_charges`
(only `Pay()` inserts there). The result: **every tenant activated by an admin ends up with two
permanent transaction rows for one subscription event** — one real, one a "ghost" that will show
"Menunggu Pembayaran" forever, with no expiry and no way to resolve it.

Root cause: the plan (and therefore the placeholder transaction) is chosen too early — at tenant
creation — instead of at the moment a real subscription event actually happens (Owner pays, or
admin activates).

## 2. New mechanism

Tenant creation no longer takes a plan. A tenant is created "empty" (`plan_id = NULL`,
`subscription_status = pending_payment`) and **no transaction row is created at registration at
all**. The first transaction row a tenant ever gets is the one created by whichever of the two
real subscription events happens first:

**Flow A — Owner self-service:**
1. Superadmin creates the tenant (no plan input).
2. Tenant exists, unbound to any plan.
3. Owner logs in to WO Console.
4. Owner opens Langganan, sees a grid of plan **selection cards** (see §3) and picks one.
5. Confirm modal → `Pay()` creates a real QRIS charge → Owner pays.
6. Webhook confirms → plan becomes active. One transaction row (`paid`).

**Flow B — Admin manual activation:**
1. Superadmin creates the tenant (no plan input).
2. Tenant exists, unbound to any plan.
3. Superadmin opens the tenant's row in Tenant list → "Aktifkan Langganan".
4. Picks a plan in the existing `ActivateSubscriptionModal`.
5. Confirms → `ActivateSubscription` activates synchronously. One transaction row (`granted`).

Either way: **exactly one transaction row ever exists for that first subscription event** — never
two.

## 3. Plan selection cards (WO Console → Langganan)

Confirmed design decisions (asked and answered in conversation):
- Cards are **always** shown, regardless of tenant state — whether the tenant has no plan yet,
  already has an active plan, is expiring soon, or expired. This lets an Owner switch to a
  different plan at any time, not just renew the same one (today's page hard-codes a single
  pre-assigned plan with no way to change it).
- The card matching the tenant's current active plan gets a visual marker (e.g. "Paket Aktif
  Anda" badge).
- Clicking a card does **not** immediately create a charge. It opens the existing confirm modal
  (paket, harga, durasi, tombol "Bayar Sekarang") — same guard rail as today, just triggered from
  a card instead of a single button. Only confirming in that modal calls `Pay()`.
- The pending-charge guard on `Pay()` (409 if a charge is already pending) and its resolution in
  `ActivateSubscription` are unaffected by this change — they operate on `pending_subscription_charges`
  regardless of how the plan was chosen.

## 4. Pending-charge recovery moves into Riwayat Transaksi

The current standalone "Anda memiliki pembayaran tertunda" banner (added earlier this session,
above the plan card, populated via `GET /subscriptions/pending-charge` on page load) is **removed
entirely**.

Instead, the transaction row in Riwayat Transaksi whose `status === "pending"` ("Menunggu
Konfirmasi") gets its own row-level actions: **"Lihat QR"** and **"Batalkan"**. Since `Pay()`
already guards against more than one pending charge existing for a tenant at a time, there is at
most one such row, so this is unambiguous — no need to disambiguate which pending row a click
refers to.

No new backend endpoints are needed. `GET /subscriptions/pending-charge` and
`POST /subscriptions/pending-charge/cancel` (added earlier this session) already resolve "the
tenant's one pending charge" server-side — the row click in Riwayat Transaksi just becomes the new
UI trigger for the same two calls, replacing the banner's buttons. The QR/Batalkan modal and
`handleViewPendingCharge`/`handleCancelPendingCharge` handlers already built in `SubscriptionPage.tsx`
this session are reused as-is; only their trigger location moves.

Open item to confirm during implementation: whether "Lihat QR" and "Batalkan" render as two
inline buttons directly on the transaction row (desktop table + mobile CardList), or as a single
"Kelola" action that opens a small menu — pick whichever fits the existing row's width/density
better once actually laid out.

## 5. Concrete file-level changes

### Backend

- **`apps/api/internal/modules/platform/application/tenant_service.go`**
  - `RegisterTenantInput`: remove `PlanID int64`.
  - `Register`: remove the `s.billing.GetPlan(ctx, input.PlanID)` call and the
    `s.billing.RecordTransaction(...)` call that records the `unpaid` placeholder (lines ~116,
    162-167 as of this writing). Tenant is created with `PlanID: nil` and
    `SubscriptionStatus: domain.StatusPendingPayment` only.
  - No change needed to `ActivateSubscription`, `Pay`, `txTypeFor`, `computeNewExpiry`,
    `GetPendingCharge`, `CancelPendingCharge` — all already tolerate a tenant with no plan
    (`txTypeFor` branches on `SubscriptionStatus`, not `PlanID`; `tenant.PlanID` is already
    `*int64`, nullable at the domain and DB level).
- **`apps/api/internal/modules/platform/presentation/tenant_handler.go`**
  - `registerTenantBody`: remove `PlanID int64 \`json:"planId"\`` and its pass-through in
    `register()`.
- **Database**: no migration required — `tenants.plan_id` has been `NULL`-able since
  `000005_create_platform_tables.up.sql`. Nothing else changes shape.
- **`docs/API_CONTRACT.md`**: update the `POST /tenants` request body row to drop `planId`.

### Frontend — Platform Console (tenant creation)

- **`apps/web/src/modules/platform-admin/schemas/tenant.schema.ts`**: remove `planId` from
  `tenantCreateSchema`.
- **`apps/web/src/modules/platform-admin/components/TenantFormModal.tsx`**: remove the "Paket
  Langganan" `Field`/`Select` block (create-mode only); remove the now-unused `plans`/`activePlans`
  plumbing from this component (`FormState.planId`, `toFormState`'s `defaultPlanId` param, the
  backfill `useEffect`). This component no longer needs a `plans` prop at all.
- **`apps/web/src/modules/platform-admin/pages/TenantListPage.tsx`**:
  - Stop passing `plans` to `<TenantFormModal>` (still needed elsewhere on this page — table's
    `planName()` column and `<ActivateSubscriptionModal>` — so `fetchPlans`/`useSubscriptionPlanStore`
    stay).
  - Update the post-create `credentialReveal` success message (currently "Tagihan paket {X} sebesar
    {Y} telah diterbitkan dan menunggu pembayaran") — it must no longer reference a plan/tagihan
    that doesn't exist yet. Replace with something like: "Owner dapat login dan memilih paket
    langganannya sendiri, atau tunggu Anda mengaktifkan langganannya dari sini."
- **`apps/web/src/modules/platform-admin/stores/usePlatformAdminStore.ts`**: `registerTenant`
  currently does `httpClient.post(API.platform.tenants, { ...values, planId: Number(values.planId) })`
  — simplify to `httpClient.post(API.platform.tenants, values)` since `values` no longer has a
  `planId`.

### Frontend — WO Console (Owner subscription page)

- **`apps/web/src/modules/subscription/pages/SubscriptionPage.tsx`** (the big one):
  - Replace the single plan `<Card>` (name/price/features of `tenant.planId`'s plan, one CTA
    button) with a grid of plan cards — one per `plans.filter(p => p.isActive)` — each showing
    name, price, duration, features, and its own "Pilih Paket ini" (or similar) action. Mark the
    card matching `tenant?.planId` as the active one.
  - Selecting a card opens the existing confirm `<Modal>` (reuse as-is, just change what sets
    `confirmOpen`/which plan it's confirming — today `plan` is derived from `tenant?.planId` only;
    it needs a piece of state for "the plan the Owner just clicked", independent of
    `tenant.planId`, since a plan-less tenant has no `tenant.planId` to derive from and an
    existing-plan tenant might click a *different* card than their current one).
  - Remove the `pendingCharge`/`confirmingCancel`/`isCancelling` banner block (§4) and its
    `isPendingChargeDegraded` handling — this state/logic moves to the transaction table instead.
  - In the Riwayat Transaksi `<Table>`/`<CardList>` rendering, add the row-level "Lihat QR" /
    "Batalkan" actions for the row where `tx.status === "pending"`, wired to the same
    `handleViewPendingCharge`/`handleCancelPendingCharge` (adjusted to take the specific
    transaction rather than reading from a separately-fetched `pendingCharge` state — or keep
    `fetchPendingCharge`/`pendingCharge` as the data source but move *where* it's rendered from
    the banner into this row; either is fine, pick whichever keeps the diff smaller once in the
    code).
  - The degraded-fallback handling added earlier this session (backend `GetPendingCharge`
    returning a partial `ChargeResult` when the live gateway check fails, and the frontend hiding
    "Lihat QR" / the bogus expiry in that case) still applies — it just needs to render inside the
    transaction row instead of the banner.

## 6. Testing plan (once implemented)

Mirror this session's established pattern — build/type-check, then interactive Playwright
verification via a temporary `DevPreview.tsx` (mocked stores, no real backend needed for the
frontend-only pieces; a local `npm run dev:api` + local MySQL for the backend pieces), covering:

1. Register a tenant with no plan input → confirm no transaction row is created and
   `tenant.planId` is `null`.
2. WO Console Langganan page for that plan-less tenant → confirm the card grid renders, no card is
   marked "active", clicking a card opens the confirm modal with that card's plan/price/duration.
3. Confirm → QRIS charge created → transaction row appears as `pending` with row-level "Lihat QR"
   / "Batalkan" actions, no standalone banner.
4. "Lihat QR" reopens the same QR modal used today; "Batalkan" cancels it (existing backend
   behavior, already covered by prior tests this session — just re-verify the new trigger wiring).
5. Platform Console "Aktifkan Langganan" on a plan-less tenant → confirm it still works exactly as
   today (plan picker already lives in that modal) and produces exactly one `granted` row.
6. A tenant that already has an active plan → card grid still shows all plans, current plan
   marked, picking a *different* plan card still works (upgrade/switch path), picking the same
   plan renews as today.
7. Full regression pass on existing subscription/transaction features already built this session
   (guards on `Pay`/`ActivateSubscriptionModal`, `cancelled` status label/tone, degraded-charge
   fallback) to confirm none of them regressed from moving the banner into the table.

## 7. Explicitly out of scope for this change

- No change to `ActivateSubscriptionModal`'s own plan picker — it already lets admin choose a plan
  independent of what the tenant currently has.
- No change to the payment gateway integration, webhook handling, or reconciliation sweep.
- No change to billing/plan CRUD in Platform Console.
