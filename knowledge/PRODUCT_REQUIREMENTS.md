# Product Requirements

> Features, scope, and user stories.

See [PROJECT_BRIEF.md](PROJECT_BRIEF.md) for who uses ElProof and why. This file lists what each
surface actually does, at the feature level, as already implemented in the frontend.

## WO Console (`/dashboard`, `/projects`, `/clients`, `/vendor-categories`, `/vendors`, `/pengguna`, `/langganan`)

- **Dashboard**: stat cards + revenue/project trend charts + attention queue (overdue milestones,
  open issues, incomplete payment evidence, near-D-day projects) + recent activity + upcoming events.
- **Projects**: full lifecycle per wedding project — status (`Draft→Preparation→Ready→Completed`,
  or `Cancelled`), 7 tabs per project: vendor engagements, milestones, client contacts, payments,
  issues, evidence documents, activity log.
- **Clients**: contacts per project (Bride/Groom/Family Representative), contact edit, credential
  reset, active/inactive toggle, and a self-service permanent delete for a client left without a
  working login credential (frees up its role slot on the project — deactivating alone doesn't).
- **Vendor Categories** / **Vendors**: reference data + vendor directory with project history.
- **Pengguna** (Users): create/edit Admin and Staff accounts, each provisioned with a real login
  credential (username + password) the same way Clients are. The Owner row is seeded once by
  Platform Console at tenant registration, but the Owner can edit their own name/title/contact
  details here afterward — no other staff member (even Admin) can touch the Owner's row, and
  nobody can reassign the Owner's role or username from this page.
- **Langganan** (Subscription): Owner-only. Shows the tenant's current plan, features, expiry, and
  transaction history; "Bayar Sekarang" creates a real QRIS charge through Tripay (`payment` module,
  Fase 9) and polls the transaction status until the gateway confirms it — see
  [MODULE_PAYMENT.md](MODULE_PAYMENT.md).

## Client Portal (`/portal/*`)

Read-only for the logged-in client: ringkasan (summary/condition), vendor progress, payments,
issues (kendala). No create/update/delete anywhere in this surface — everything scoped to the
client's own project only.

## Platform Console (`/platform/*`) — ElProof's internal team only

- **Dashboard**: platform-wide stats (total/active tenants, unpaid count, paid revenue) + trend
  charts (omzet, tenant baru) filterable by "Bulan Ini"/"Tahun Ini".
- **Tenant**: register new tenant (sets the Owner's login credentials directly — no auto-generated
  password), edit, reset credential, suspend/reactivate, **activate subscription manually** (grants
  a plan without going through payment — recorded as transaction status `granted`, excluded from
  paid-revenue reporting).
- **Paket** (Plans): CRUD the subscription plan catalog (name, duration, price, feature list) — this
  is the single source of truth also rendered on the WO Console's `Langganan` card.
- **Transaksi**: subscription transaction ledger across all tenants.
- **Pengguna**: manage Platform Console's own admin accounts (Super Admin / Support roles).
- **Gateway Pembayaran** (`/platform/pembayaran`): configure the active payment provider (Tripay),
  sandbox/production mode, and merchant credentials (write-only — never echoed back) — see
  [MODULE_PAYMENT.md](MODULE_PAYMENT.md).
- **Manajemen Aplikasi** (`/platform/aplikasi`): register/manage the external Apps allowed to create
  charges through ElProof's payment gateway as a service (Fase 10) — list Apps, register a new
  external App (App ID + secret, shown once), reset a secret, toggle active/inactive. ElProof's own
  billing (`ElProof Billing`, internal) is always listed and can't be disabled here.

## Public Marketing Site (`/homepage/*`)

Frontend-only, no backend module or API calls (see the `homepage` row in
[MODULE_MAP.md](MODULE_MAP.md)) — landing page, Tentang Kami, Syarat & Ketentuan, Kebijakan Privasi,
Kebijakan Refund, FAQ, and Kontak (showing email, phone, and business address, kept in sync with
what's registered with payment-gateway merchants like iPaymu).

## Explicitly out of scope for now

- Anything beyond a single WO business per tenant (no franchise/multi-branch concept).
- Multiple simultaneous active payment gateway providers (one active provider at a time, chosen in
  Gateway Pembayaran — no per-tenant or per-App provider choice).
- Cancelling an already-created payment charge (Tripay itself has no cancel/void endpoint — an
  abandoned charge is left to expire naturally; see [MODULE_PAYMENT.md](MODULE_PAYMENT.md)).
