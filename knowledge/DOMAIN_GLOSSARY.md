# Domain Glossary

> Shared terminology used across the project.

| Term | Meaning |
|---|---|
| **Tenant** | One WO (Wedding Organizer) business subscribed to ElProof. Owns its own staff, clients, vendors, and projects — never shares data with another tenant. |
| **Owner** | The tenant-side role created by the Platform Console at tenant registration time. Exactly one per tenant conceptually; only the Owner can access `Langganan` (subscription management) in the WO Console. Distinct from `platform_admin`. |
| **WO Console** | The tenant-facing app surface (`/dashboard`, `/projects`, `/clients`, `/vendors`, `/vendor-categories`, `/pengguna`, `/langganan`). |
| **Platform Console** | ElProof's own internal admin surface (`/platform/*`), used by ElProof staff (`platform_admin`), never by tenants. |
| **Client Portal** | Read-only surface (`/portal/*`) for a wedding's bride/groom/family to track their own project. |
| **Principal** | Whoever is authenticated: one of `staff` (WO Console), `client` (Client Portal), or `platform_admin` (Platform Console). See ADR-0005. |
| **Project** | One wedding engagement managed by a tenant. Has a lifecycle status (`Draft→Preparation→Ready→Completed`/`Cancelled`) and owns milestones, vendor engagements, payments, issues, evidence, activity. |
| **Project Vendor** | The engagement between a Project and a Vendor (scope, contract value, DP/paid amounts, status) — not the Vendor record itself. |
| **Vendor Milestone** | A checklist item tracked against one Project Vendor engagement (distinct from a Project Milestone, which tracks the wedding's own top-level timeline). |
| **Evidence** | An uploaded document/photo attached polymorphically to a vendor milestone, payment, or issue (`relatedKind` + `relatedId`). |
| **Granted (transaction status)** | A subscription transaction created by Platform Console manual activation — bypasses payment, deliberately excluded from paid-revenue reporting. Distinct from `paid` (real payment). |
| **Plan (Subscription Plan)** | A row in the shared plan catalog (`billing` module) — name, duration, price, feature list. Single source of truth rendered identically on the WO Console's `Langganan` card and the Platform Console's `Paket` CRUD. |
