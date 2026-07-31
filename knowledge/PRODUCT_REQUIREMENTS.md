# Product Requirements

> Features, scope, and user stories.

See [PROJECT_BRIEF.md](PROJECT_BRIEF.md) for who uses ElProof and why. This file lists what each
surface actually does, at the feature level, as already implemented in the frontend.

## WO Console (`/dashboard`, `/projects`, `/clients`, `/vendors`, `/venues`, `/pengguna`, `/langganan`, `/vendor-categories`, `/timeline-default`)

**Role-based access (RBAC — see ADR-0017):** Owner sees and can do everything below. Admin sees
everything except the "Pengaturan" group (Pengguna, Langganan, Kategori Vendor, Timeline Default —
collapsed under one sidebar entry, invisible to Admin entirely). Wedding Planner (backend role
`"Staff"`) only sees Project, scoped to just the projects they're PIC of — never creates or
duplicates a project, and never reassigns a project's PIC (Owner/Admin only).

- **Dashboard** (Owner/Admin): stat cards + revenue/project trend charts + attention queue (overdue
  Timeline items, open issues, incomplete payment evidence, near-D-day projects, lagging progress) +
  recent activity + upcoming events.
- **Projects**: full lifecycle per wedding project — status (`Draft→Preparation→Ready→Completed`,
  or `Cancelled`), 8 tabs per project: vendor engagements, Timeline, Venue, client contacts, payments,
  issues, evidence documents, activity log. The header also shows a computed **Margin/Keuntungan**
  (contract value minus vendor costs minus venue cost) — Owner/Admin only, hidden from Wedding
  Planner. Creating a new project or duplicating one is Owner/Admin only.
- **Clients** (Owner/Admin): contacts per project (Bride/Groom/Family Representative), contact edit,
  credential reset, active/inactive toggle, and a self-service permanent delete for a client left
  without a working login credential (frees up its role slot on the project — deactivating alone
  doesn't).
- **Vendors** (Owner/Admin write, read open to every staff role for pickers): directory with project
  history, each with a pair of preset prices ("Harga Akad" / "Harga Akad + Resepsi") a project
  engagement can auto-fill its own Nilai Kerja Sama from, plus social media/city/a single
  document-or-photo attachment; bulk Excel import/export.
- **Venues** (ADR-0016; Owner/Admin write, read open to every staff role): its own directory,
  separate from Vendors — rental price, a separate charge, capacity, facilities, a single
  attachment; bulk Excel import/export; attaches to a Project 1:1 from the Project's own Venue tab.
- **Pengguna** (Users, Owner-only): create/edit Admin and Staff accounts, each provisioned with a
  real login credential (username + password) the same way Clients are. The Owner row is seeded once
  by Platform Console at tenant registration, but the Owner can edit their own name/title/contact
  details here afterward — no other staff member (even Admin) can touch the Owner's row, and
  nobody can reassign the Owner's role or username from this page.
- **Langganan** (Subscription, Owner-only): Shows the tenant's current plan, features, expiry, and
  transaction history; "Bayar Sekarang" creates a real QRIS charge through Tripay (`payment` module,
  Fase 9) and polls the transaction status until the gateway confirms it — see
  [MODULE_PAYMENT.md](MODULE_PAYMENT.md).
- **Kategori Vendor** (Owner-only to manage; the plain list stays readable by every staff role for
  the Vendor form's category picker and each project's Vendor tab).
- **Timeline Default** (Owner-only): configure the per-tenant checklist (name + days-before-event)
  auto-seeded into every new project's own Timeline tab, replacing what used to be one hardcoded
  6-item list shared by every tenant. Editing it never retroactively changes an already-created
  project, and has no effect on Duplikat Project (which clones the source project's actual Timeline).

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
