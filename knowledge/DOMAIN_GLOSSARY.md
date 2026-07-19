# Domain Glossary

> Shared terminology used across the project.

| Term | Meaning |
|---|---|
| **Tenant** | One WO (Wedding Organizer) business subscribed to ElProof. Owns its own staff, clients, vendors, and projects — never shares data with another tenant. |
| **Owner** | The tenant-side role created by the Platform Console at tenant registration time. Exactly one per tenant conceptually; only the Owner can access `Langganan` (subscription management) in the WO Console. Distinct from `platform_admin`. |
| **WO Console** | The tenant-facing app surface (`/dashboard`, `/projects`, `/clients`, `/vendors`, `/vendor-categories`, `/pengguna`, `/langganan`). |
| **Platform Console** | ElProof's own internal admin surface (`/platform/*`), used by ElProof staff (`platform_admin`), never by tenants. |
| **Client Portal** | Read-only surface (`/portal/*`) for a wedding's bride/groom/family to track their own project. |
| **Principal** | Whoever is authenticated: `staff` (WO Console), `client` (Client Portal), `platform_admin` (Platform Console), or `app` (an external SaaS consumer of `payment`'s Fase 10 API, minted via `POST /auth/app/token` — no `credentials` row, no refresh token). See ADR-0005 and its update. |
| **App** (payment) | A consumer registered in `payment`'s App registry (`payment_apps`), allowed to create charges through ElProof's one gateway wallet. `kind=internal` (in-process, e.g. `platform-billing` — ElProof's own subscription billing) or `kind=external` (HTTP, another SaaS product, managed via Platform Console's "Manajemen Aplikasi"). Not to be confused with the `App` React component or "application" in a general sense — always refers to this registry concept. See `MODULE_PAYMENT.md`. |
| **Konfigurasi Gateway** | Platform Console page (`/platform/pembayaran`) for choosing the active payment provider (Tripay) and its sandbox/production credentials — write-only, never echoes a stored secret back. |
| **Manajemen Aplikasi** | Platform Console page (`/platform/aplikasi`) for registering/managing external Apps (see above) — reveal-once secrets, reset, toggle active/inactive. |
| **Reconciliation sweep** | `payment`'s background safety net (`ReconcilePending`) for a charge whose webhook was never received — periodically re-checks still-open charges directly against the gateway and force-resolves anything left open long past its own expiry. See `MODULE_PAYMENT.md` §6 step 6. |
| **Project** | One wedding engagement managed by a tenant. Has a lifecycle status (`Draft→Preparation→Ready→Completed`/`Cancelled`) and owns milestones, vendor engagements, payments, issues, evidence, activity. |
| **Project Vendor** | The engagement between a Project and a Vendor (scope, contract value, DP/paid amounts, status) — not the Vendor record itself. |
| **Vendor Milestone** | A checklist item tracked against one Project Vendor engagement (distinct from a Project Milestone, which tracks the wedding's own top-level timeline). |
| **Evidence** | An uploaded document/photo attached polymorphically to a vendor milestone, payment, or issue (`relatedKind` + `relatedId`). |
| **Granted (transaction status)** | A subscription transaction created by Platform Console manual activation — bypasses payment, deliberately excluded from paid-revenue reporting. Distinct from `paid` (real payment). |
| **Plan (Subscription Plan)** | A row in the shared plan catalog (`billing` module) — name, duration, price, feature list. Single source of truth rendered identically on the WO Console's `Langganan` card and the Platform Console's `Paket` CRUD. |
