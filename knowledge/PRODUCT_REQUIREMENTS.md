# Product Requirements

> Features, scope, and user stories.

See [PROJECT_BRIEF.md](PROJECT_BRIEF.md) for who uses ElProof and why. This file lists what each
surface actually does, at the feature level, as already implemented in the frontend.

## Login (`/login`) — shared entry point for every principal type

A full-bleed, auto-advancing photo slider (global marketing images, not tenant-scoped — see
`MODULE_PLATFORM.md` §9) behind the headline/subtext. The page is always neutral: no tenant is known
before sign-in, and tenant custom domains no longer exist (ADR-0030).

## WO Console (`/dashboard`, `/projects`, `/clients`, `/vendors`, `/venues`, `/pengguna`, `/peran-akses`, `/langganan`, `/vendor-categories`, `/timeline-default`)

**Role-based access (RBAC — see ADR-0026):** each workspace has editable roles built from a fixed
permission catalog (`access/domain/catalog.go`, Pengaturan → Peran & Akses). Owner (locked) sees and
can do everything below. Default templates: **Admin** (every project, client, vendor and payment;
no workspace settings — team, roles, subscription, business profile, vendor categories, Timeline
Default — no permanent deletes of projects/vendors/venues), **Wedding Planner** (only projects they
are PIC or PIC Sales of: edit them and record payments), **Sales** (projects they hold: create —
becoming PIC Sales — edit, assign PIC, archive, record payments, see the margin, manage their
clients), **Keuangan** (every project: dashboard with revenue, record and delete payments, margin,
read the vendor/venue directory). A WO may edit these (except Owner) or create its own roles; the
rules below name the permission, the default holders follow from the templates. A role with the
"assigned" project scope never sees a tenant-wide list (dashboard, all clients, a vendor's project
history, the names in a delete warning beyond its own projects). Menu entries and buttons follow the
role's permissions, and the API refuses the same things (`403 forbidden_permission`).

- **Dashboard** (`dashboard.view` with every-project scope — Owner, Admin, Keuangan; revenue figures
  need `dashboard.revenue.view`): stat cards + revenue/project trend charts + attention queue (overdue
  Timeline items, open issues, incomplete payment evidence, near-D-day projects, lagging progress) +
  recent activity + upcoming events. Every other role gets "Proyek saya" (the projects they hold).
- **Projects**: full lifecycle per wedding project — status (`Draft→Preparation→Ready→Completed`,
  or `Cancelled`), 8 tabs per project: vendor engagements, Timeline, Venue, client contacts, payments,
  issues, evidence documents, activity log. Editing needs `project.edit`, assigning PIC/PIC Sales
  `project.assign_pic`, archiving `project.archive`, duplicating `project.duplicate`, permanently
  deleting an archived/cancelled project `project.delete` (Owner by default). Creating one needs
  `project.create` (Owner, Admin, Sales; with "assigned" scope the creator becomes PIC Sales). The
  header also shows a computed **Margin/Keuntungan** (contract value minus vendor costs minus venue
  cost) to `finance.margin.view` (Owner, Admin, Sales, Keuangan; not Wedding Planner). The figures it
  is computed from still reach everyone who works on the project (ADR-0026 › Consequences).
  - **Pembayaran** (vendor/client/venue, all three): recording, editing and verifying need
    `finance.record`; deleting — and bringing back with "Urungkan" — needs `finance.payment.delete`
    (Owner, Admin, Keuangan). A deleted payment stays restorable for 24 hours before it is purged
    (ADR-0034); the dialog names the amount and how many evidence files go with it (ADR-0019).
    A "Lihat Bukti" button on every row shows its attached evidence inline, without leaving the tab.
  - **Dokumen & Evidence**: gained a 7th "Umum" context for documents with no specific
    vendor/payment/timeline link (rundown, buku acara, teks juru bicara, banquet order, rekap dekor,
    etc.), each with its own Client Portal visibility toggle — default hidden, opt-in only (ADR-0020).
    Adding a file needs `project.edit` or `finance.record` (payment proofs); showing a document to
    the client — the toggle, or ticking it while uploading — needs `project.edit`; deleting a
    payment's proof also needs `finance.payment.delete`.
- **Clients** (`client.manage`: Owner, Admin, and Sales for the projects they hold): contacts per
  project (Bride/Groom/Family Representative) with a portal role (Pengantin/Keluarga), contact
  edit, invitation and credential-reset links, active/inactive toggle, replacing the family
  representative, and delete (restorable for 24 hours, ADR-0034). The tenant-wide client list (and
  the client search) needs `client.manage` with every-project scope; anyone who can open a project
  reads its clients.
- **Vendors** (list page, export and project history: `vendor.view` — Owner, Admin, Keuangan; add,
  edit, import, (de)activate: `vendor.manage` — Owner, Admin; permanent delete: `vendor.delete` —
  Owner): directory with project history, each with a pair of preset prices ("Harga Akad" / "Harga
  Akad + Resepsi") a project engagement can auto-fill its own Nilai Kerja Sama from, plus social
  media/city/a single document-or-photo attachment; filterable by Kota; bulk Excel import
  (dropdown-validated Kategori/Kota columns, styled header/column widths — no free-text typo can
  reach an import row) and export (same filters as the list, column order matches Import so an
  exported file round-trips back through it); permanent delete (informed-consent dialog naming the
  affected projects — see ADR-0018), alongside the deactivate toggle. The picker list inside a
  project (`GET /vendors?all=true`) is also open to whoever works on projects (`project.edit`,
  `project.create`, `finance.record` or `dashboard.view`), because the project's vendor form needs
  the prices.
- **Venues** (ADR-0016; same permissions as Vendors): its own directory, separate from Vendors —
  rental price, a separate charge, capacity, facilities, a single attachment; filterable by Kota;
  bulk Excel import/export and permanent delete, same shape as Vendors above (ADR-0018); attaches
  to a Project 1:1 from the Project's own Venue tab. Single venues are looked up by whoever works on
  projects, like the vendor picker.
- **Pengguna / Tim** (`team.manage`, Owner by default): invite staff by email — each person accepts
  the invitation with their own Google account (ADR-0024/0025), no username or password is handed
  out — edit, re-invite, deactivate and permanently delete members (informed-consent dialog, same
  pattern as Vendor/Venue — ADR-0018). Someone who is not the Owner may only hand out, or manage a
  member holding, a role their own role covers, and never changes their own role (ADR-0026 › No
  escalation). The Owner row is seeded once at tenant registration; the Owner edits their own
  name/title/contact details here, nobody else touches it, the Owner role is never handed out, and
  the Owner account can never be deactivated or deleted.
- **Peran & Akses** (`role.manage`, Owner by default): create, edit and copy roles, delete custom roles nobody holds; the
  same no-escalation rule applies to staff roles (portal roles grant nothing in the console).
- **Langganan** (Subscription, `subscription.manage`, Owner by default): Shows the tenant's current
  plan, features, expiry, and transaction history; "Bayar Sekarang" creates a real QRIS charge
  through Tripay (`payment` module, Fase 9) and polls the transaction status until the gateway
  confirms it — see [MODULE_PAYMENT.md](MODULE_PAYMENT.md).
- **Kategori Vendor** (`vendor.category.manage`, Owner by default, to manage; the plain list stays
  readable by every staff role for the Vendor form's category picker and each project's Vendor
  tab). Permanent delete is blocked (not informed-consent) while any vendor still references the
  category — a real, same-module SQL foreign key, unlike Vendor/Venue/Staff's soft cross-module
  references (ADR-0018).
- **Timeline Default** (`template.manage`, Owner by default): configure the per-tenant checklist (name + a day offset, either
  Sebelum Acara/"H-" or Setelah Acara/"H+" — e.g. an item scheduled a few days *after* the event,
  like returning the seserahan box) auto-seeded into every new project's own Timeline tab, replacing
  what used to be one hardcoded 6-item list shared by every tenant. Editing it never retroactively
  changes an already-created project, and has no effect on Duplikat Project (which clones the source
  project's actual Timeline).

## Client Portal (`/portal/*`)

Read-only for the logged-in client, everything scoped to their own project only — no
create/update/delete anywhere in this surface. What each client sees follows their portal role
(ADR-0026): **Pengantin** sees everything, **Keluarga** everything except money — without
`portal.finance.view` the contract value and remaining bill are left out, the Pembayaran tab, the
bank account and payment receipts are not there, and the API refuses them. A WO may edit both roles
or make its own from the five portal permissions (finance, vendor, timeline, documents, issues). A persistent header (`ClientProjectHeaderCard.tsx`)
stays visible above every tab: project name/status, an info grid (Tanggal Acara, Paket/Layanan,
Nilai Kontrak, Sisa Tagihan, Penanggung Jawab), the free-text project note when non-empty, and a
compact progress summary (percent, condition badge, open-issues banner) — this absorbed what used
to be a standalone "Ringkasan" tab. Internal-only fields (Margin/Keuntungan, Tanggal Booking, and
every WO-side action button) are never sent to this surface at all, not just hidden in the UI.

Five tabs below that header:
- **Vendor** — every vendor engaged on the project (progress rail, engagement status, open-kendala
  banner, collapsible tahapan/kendala detail) *plus* the project's venue, pinned as the grid's first
  card, presentation-merged in client-facing views only (WO Console keeps Vendor and Venue as
  separate pages/tabs) — see ADR-0016's Revision. The venue card has no progress rail or status
  badge (nothing backs one), just its public-safe summary (address, capacity, facilities, social
  media, photo).
- **Timeline** — the project's own milestone stepper (the detailed view; the header's own progress
  summary is the compact one).
- **Pembayaran** — payments received against the project's contract value.
- **Kendala** — every open/resolved issue across every vendor on the project, in one place (distinct
  from the per-vendor kendala accordion on the Vendor tab's own cards).
- **Dokumen** (ADR-0020) — general project documents the WO has explicitly opted to share (rundown,
  buku acara, etc.) — never every document uploaded internally, only the ones a staff member has
  individually toggled visible. Appended last, deliberately not interleaved with the four tabs above.

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
