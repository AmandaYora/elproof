# ADR-0030: Platform Console back office (metrics, tenant 360°, support sessions, PDP requests)

## Status
Accepted — implemented 2026-11-07 (PLAN.md Fase 6, FLOW-IDEAL 1.2–1.5). Supersedes ADR-0015 (tenant custom
domain), which is removed rather than kept alongside.

## Context
The Platform Console was a set of lists: tenants, plans, transactions, admins. ElProof's own team could not
answer "how is the business doing", "what is wrong with this customer", "who changed that", or "can we look at
their workspace to help" without database access, and the only way to see a WO's screen was to ask for its
password. Two obligations were also open: answer personal-data requests within 30 days (UU PDP) and keep an
audit trail of what staff of ElProof and of WOs change.

## Decision
- **Numbers come from the server.** `GET /platform/metrics` returns MRR, ARPA, trial conversion, churn and
  30-day revenue from one pure function (`platform/domain/platform_metrics.go`, unit-tested). Trial tenants
  and free grants (compensation, trial extension, internal) are never counted as revenue. Expansion and
  contraction MRR are not computed: the product has one plan and no plan-change history, and the page says so
  rather than showing a made-up figure. Percentages are `null` until there is a base to divide by.
- **Tenant 360° is one read model.** `GET /console/tenants/{id}/overview` bundles profile, signup consent,
  manual activations, usage against trial caps, members (via the `staff` contract), open data requests and
  support consent. Tabs that are lists (billing, notes, data requests, activity) load on demand. Internal notes
  are for the ElProof team only.
- **Support sessions are consented, short, read-only and visible.** The Owner grants access for 24 hours
  (revocable) at Pengaturan → Profil Usaha. A Support or Super Admin then starts a session with a written
  reason (10–255 characters), which mints a 60-minute token for the Owner's credential carrying `imp`
  (admin id) and `sid` (session id), with no refresh token. `RestrictImpersonation` refuses every non-GET with
  `impersonation_read_only`; the web hides write controls and shows a banner with an "Akhiri sesi" button.
  **A token is not enough:** `RequireLiveSupportSession` asks the platform module on every request whether the
  session is still open (not ended, not past 60 minutes), the Owner's consent is still valid and the admin is
  still active as Support or Super Admin; otherwise 401 `support_session_ended`. Ending the session, withdrawing
  consent or deactivating the admin therefore stops a copied token at once. The admin's refresh token is not
  carried into the WO Console page (it renders tenant-written content); on the way back the admin has their
  access token and signs in again once it lapses.
  Every session is recorded (admin, reason, start, expiry, end) and listed to the Owner. The identity module
  refuses to issue the token for an inactive credential or a frozen tenant.
- **Roles follow least privilege, enforced once.** `cmd/server/platform_scope.go` classifies every console
  route. Read-only keeps the summaries but cannot read a workspace's internals or export the customer list;
  Billing activates subscriptions and edits plans; Support keeps notes, logs data requests and starts support
  sessions; announcements, the subscription policy, the audit log and login photos are Super Admin only.
  Anything unclassified is a Super Admin write.
- **Changing the Owner's email is Super Admin only.** The email is the Google account that becomes Owner after
  an invitation, so letting the Support role edit it (then re-invite) would bypass the consent above.
- **Audit covers the platform module and exports.** The platform routes now pass through `AuditTrail`
  (no request bodies); for a platform admin the tenant id is read from the path, so a tenant's activity includes
  what the team did to it. The exports write their audit record *before* any data is sent (no record, no export). Tenant and audit CSV exports write their own audit record and guard against spreadsheet
  formula injection; the audit export is capped at 10 000 rows (the tenant export is not: the table is small).
- **Subscription policy is data.** Trial, reminder, grace and read-only lengths (14/7/7/23) live in
  `subscription_policy`, are validated, and are loaded into `domain.CurrentSubscriptionPolicy()` at startup and
  on every save. Changing it takes effect for the next lifecycle computation of every tenant.
- **Personal-data requests have a clock.** `data_requests` records export/delete requests with a 30-day due
  date, who asked, and mandatory proof of completion. The queue shows days left or days late.
- **Announcements** target all tenants, a plan or a lifecycle status, with a start and optional end; the WO
  Console shows them as dismissible banners (dismissal is per browser).
- **Custom domains are gone (K15).** Migration 62 drops `tenants.custom_domain`; the Host-based public branding
  endpoints, site-meta injection into `index.html` and `useDomainBranding()` are deleted. Per-page title and
  description now come from `usePageMeta`, and `robots.txt` keeps the app out of search.
- **Smaller pieces:** the 2FA setup page draws a QR code in the browser (`qrcode`); QRIS shows a countdown, the
  paid period the payment will buy (it starts where current access ends) and different wording for expired
  versus cancelled; paid subscriptions have a printable receipt (the browser's "Save as PDF", no PDF library);
  a Super Admin can replace the three login photos from the console.

## Consequences
- A support session uses the Owner's real credential, so anything keyed on `principal_id` (activity log
  authors, "last seen") would attribute reads to the Owner; reads are not logged today, and the admin's
  identity is in the session record and in the `imp` claim. Reads of the 360° pages are not audited either.
- A support request costs three indexed lookups (session, grant, admin). Freezing a tenant also ends the
  session at once (the tenant gate runs on every request).
- Free grants: a tenant counts as free only if a free grant (compensation, trial extension, internal) matches
  its current expiry **and it never paid**; a free grant stacked on a customer is a bonus period. A customer who
  lapsed long ago and then got a free grant is therefore still counted as paying. Payments made outside the
  gateway (`offline_payment`, `sales_deal`) count in MRR but not in the 30-day revenue figure, which is
  gateway-paid only; the page says so.
- Every timestamp leaves the API in WIB (+07:00) whatever the server's time zone; the web shows them as-is.
- The audit tab on Tenant 360° is Super Admin only because the audit log is.
- Price versions and plan entitlements (FLOW-IDEAL 1.4) are not built: with no paying customers yet there is no
  history to protect (K13). Revisit before the first price change.
- The metrics read all paid transactions of the last 60 days and every tenant in memory. That is fine for the
  first hundreds of tenants; move to SQL aggregates before it is not.
- A sitemap is not published: it needs the production domain, which is not recorded in this repo.
