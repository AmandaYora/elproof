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
  reset, active/inactive toggle.
- **Vendor Categories** / **Vendors**: reference data + vendor directory with project history.
- **Pengguna** (Users): WO's own internal staff (Admin/Staff only — Owner is managed via Platform
  Console, see below).
- **Langganan** (Subscription): Owner-only. Shows the tenant's current plan, features, expiry, and
  transaction history; "pay" is currently simulated client-side (Tripay wording removed from UI by
  design — see ADR history in `PLAN.md`).

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

## Explicitly out of scope for now

- Real payment gateway integration (Tripay is UI-hidden/mocked; wording deliberately kept
  non-developer-centric).
- Anything beyond a single WO business per tenant (no franchise/multi-branch concept).
