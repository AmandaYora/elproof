# ADR-0029: Client Portal home, payment information and "Proyek saya"

## Status
Accepted — implemented 2026-11-07 (PLAN.md Fase 5, FLOW-IDEAL 3.2–3.3).

## Context
The portal opened on the Vendor tab, never said what the couple should do, never showed how to pay,
leaked internal wording, and sent every error to the login page. Meanwhile roles limited to their own
projects (Wedding Planner, Sales) had no landing page at all: the dashboard aggregates every project.

## Decision
- **Beranda first.** The portal lands on `/portal/beranda`: countdown, what to do now, next
  installment with the account to pay into and a one-tap WhatsApp confirmation, four-step progress,
  latest finished steps, contact. Bottom tab bar on phones, top bar from `md`. Red appears only on the
  "Yang perlu Anda lakukan" card, and only for a payment past due; badges the WO is handling are amber
  for clients. Issues always answer: what happened, what we are doing, when to expect news.
- **Payment information is the Owner's, in a separate table.** `tenant_business_details` holds the
  address and bank account (all-or-nothing). `GET /tenants/me/contact` serves the business WhatsApp and
  address to any tenant principal and the bank account **only** to staff or a client with
  `portal.finance.view`. The first-line client allowlist (`cmd/server/client_scope.go`) and the
  handler-level permission both have to agree; both have tests.
- **The client reads their schedule.** `GET /projects/{id}/payment-terms` is open to a client with
  `portal.finance.view` (with the labels from `TERM_STATUS_LABELS.client`); `PUT` stays staff-only.
- **Contact the PIC directly.** The staff contract's `Summary` carries `Phone`; `getProject` adds
  `picPhone` only for a client principal (their own project). The business WhatsApp is the fallback.
- **Errors do not log anyone out.** 401 is the HTTP client's concern (refresh, then login). "No project"
  (403/404) and "could not load" (network/5xx) are their own full-screen messages with retry and, when
  known, a WhatsApp link to the WO.
- **PWA, minimally.** Manifest, icons and a service worker that keeps one offline page and nothing else:
  API calls and scripts always go to the network, so nothing can go stale or leak between accounts.
- **"Proyek saya".** `GET /dashboard/mine` (any staff role) returns the open projects the caller is PIC
  or PIC Sales of, with progress, overdue installments, open issues and items due this week. Beranda
  shows the tenant-wide dashboard to roles with `dashboard.view` and all-project scope, "Proyek saya" to
  everyone else.

## Consequences
- Whoever holds `business_profile.manage` can change the account clients are told to pay into. That is
  the most sensitive setting a WO has, so every write by staff through the platform module (profile,
  logo, brand color, business details) is recorded in `audit_log` (who, role, workspace, path, status,
  IP; bodies are never logged, so old values are not kept) and the permission's description in
  Peran & Akses says what it controls. Default holder: Owner only.
- The Beranda never shows money before the payments and schedule have really loaded (a client who has
  paid in full would otherwise briefly see the whole contract as owed), says "belum ada tagihan" when no
  contract amount is set, and never shows another account's bank details: the contact store is cleared on
  logout and on every load.
- A PIC's phone that is not a usable number (free text such as "-") falls back to the business
  WhatsApp instead of producing a `wa.me/` link with no recipient.
- Personal staff numbers become visible to that project's client: the WO decides who PIC is, and the
  business number is used when a PIC has none.
- Not done: password change by the client, "Bagikan ke keluarga" (family invitations stay with the WO),
  per-issue "Terlihat klien" switch, per-tenant PWA manifest (name/icon are generic ElProof ones).
