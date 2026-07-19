# Project Brief

> What this project is and why it exists.

ElProof is a SaaS platform for Wedding Organizer (WO) businesses to run their operations —
projects (weddings), vendor engagements, client communication, payments, and evidence/documentation
— from one console, plus a client-facing portal so the bride/groom/family can track progress
without calling the WO for updates.

## Who uses it

- **WO staff** (tenant side) — "WO Console". Three roles: `Owner`, `Admin`, `Staff`. The Owner
  account is special: it is the account created by ElProof's own platform team when the tenant is
  registered, and only the Owner can manage the tenant's ElProof subscription (`Langganan`).
- **Clients** (the bride/groom/family of a wedding project) — "Client Portal", read-only view of
  their own project's vendor progress, payments, and issues.
- **ElProof's own internal team** (SaaS operator) — "Platform Console". Manages tenant onboarding,
  subscription plans, manual subscription activation (bypassing payment when needed), platform
  admin accounts, and platform-wide metrics.

## Why a Platform Console exists

ElProof itself is the product being sold to WO businesses as tenants. Someone at ElProof needs to
register new tenants, see who's paying/expiring, manage the plan catalog, and grant access without
always going through the payment gateway (e.g. sales-negotiated deals, trials, manual invoicing
outside Tripay). This is a distinct bounded context from any one tenant's data — it operates
*across* tenants, never scoped to one.

## Current state

ElProof is **live in production** at <https://elproof.elcodelabs.com>. `apps/api` is a real Go
backend across all three consoles plus a `payment` module (Tripay-backed subscription billing, and
a Payment Gateway as a Service API other SaaS products can integrate against — see
[MODULE_PAYMENT.md](MODULE_PAYMENT.md)); the mock-data era this brief originally described is gone
— `apps/web/src/mock/` was deleted once every real feature had a backend to call. A public
marketing site also exists at `/homepage/*` for FAQ/Terms/Privacy/Refund/Contact pages, separate
from the three consoles above. See [ARCHITECTURE_OVERVIEW.md](ARCHITECTURE_OVERVIEW.md) and
`PLAN.md` at the repo root (a living phase-by-phase log, not a one-time plan) for the full history
and current status of each capability.
