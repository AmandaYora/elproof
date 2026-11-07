# ADR-0031: Scale — server-side lists, composite indexes, direct file links, isolation tests

## Status
Accepted — implemented 2026-11-08 (PLAN.md Fase 7).

## Context
Three habits worked for a handful of tenants and would not for hundreds: consoles that loaded every row to
count or search it in the browser, list queries that sorted a whole tenant in memory, and every evidence
download streamed through the one app container. Separately, "a token from tenant A cannot touch tenant B" was
enforced in every query but proven by no test.

## Decision
- **The Platform Console never loads a whole table.** Dashboard charts come from `GET /platform/trends`
  (pure `ComputeTrends`, buckets in WIB); list pages name their rows' tenants with `GET /tenants/names?ids=`
  (≤ 100 ids); the layout reads only its own admin (`GET /platform-admins/{id}`); the receipt reads one
  transaction (`GET /subscription-transactions/{id}`, tenant-scoped: another tenant's id is 404). `all=true`
  is removed from `/tenants`, `/platform-admins` and `/subscription-transactions`.
- **Global search is server-side.** It asks the paginated project, vendor and client endpoints for four hits
  each, debounced, with stale requests aborted. It respects each endpoint's own permission: a role refused
  tenant-wide clients still gets projects and vendors.
- **WO rosters keep `all=true`** (vendors, venues, categories, clients, staff, projects, plans). They feed
  pickers, are bounded by one tenant and would need a typeahead everywhere to remove. Revisit when a
  tenant's roster passes a few thousand rows; the SQL is already filtered by `tenant_id`.
- **Indexes follow the queries** (migration 64): `projects (tenant_id, is_archived, event_date, id)`, three
  composite indexes and one paid-window index on `subscription_transactions`, `tenants (joined_at, id)`. Each
  replaces the single-column index it covers. Measured on 80 000 projects and 60 000 transactions: project page
  46 ms → 0.2 ms, platform transaction list 23 ms → 0.1 ms, and `EXPLAIN` shows an ordered index scan instead
  of a filesort or table scan. Child tables keyed by `project_id` are left alone: a project has dozens of
  rows, not thousands.
- **Evidence opens through a direct link.** `GET /projects/{id}/evidence/{evidenceId}/link` returns a
  two-minute presigned URL (the same project scope and client-visibility rule as `/file`; `?disposition=
  attachment` for a download). The browser loads the PDF or image from object storage, so no bytes pass through
  the API. Where the deployment cannot do it (`S3_PRESIGN=false`) the endpoint answers 501 `presign_unavailable`
  and the web streams through `/file` as before. Presigning is a local computation, tested against a real
  MinIO: the link serves the object, applies the filename and disposition, is refused for another key and
  refused after expiry. Logos and login photos are small and stay proxied.
- **Isolation is a test, not a convention.** `apps/api/tests/isolation` (build tag `integration`) starts the
  real binary on a throwaway database, signs up two workspaces, creates a full set of resources in one and
  attacks every id-bearing route family with the other's token: it must get 403/404, never see a marker string
  from the victim, and the victim's data must be intact afterwards. Each route also has a control request from
  the owner, so a mistyped route cannot pass by returning 404. It also tries 19 ids a caller could smuggle in a
  request body, each paired with a control using the caller's own ids, plus a client principal's reach: 233
  requests in about two seconds. Run with `TEST_MYSQL_ADMIN_DSN=... go test -tags integration ./tests/isolation/`
  (it skips without the DSN; MinIO is optional for the evidence cases).
- **What the first run found, all fixed.** Query-level isolation held (every id-bearing route answered
  404), but the *ids inside request bodies* were trusted: a workspace could name another workspace's staff
  as a project's PIC or PIC Sales (and as PIC of vendor engagements, milestones and issues), another's venue
  on a project, another's vendor or category on an engagement, an engagement of a different project on an
  issue, and attach evidence to another project's milestone or issue (`relatedId` was never checked). The
  referenced name then came back in the caller's own responses. Each reference is now checked to belong to the
  caller's tenant (and, for sub-resources, to the same project) in the service layer: `projects/application/
  references.go`, `checkPICs`, `IssueService.validateEngagement`, `EvidenceRepository.RelatedBelongsToProject`,
  and `VendorExists`/`CategoryExists` on the vendors contract. Two smaller finds: any workspace's staff token
  reached the login-photo handler (the role policy only judges platform admins; the handler now insists on a
  platform admin) and clients were not allowed the evidence `/link` route (a second allowlist existed).
- **Also fixed on the way:** an unknown evidence type used to upload the file and then fail with a 500;
  it is now a 422 before anything is stored. LIKE wildcards in every search box are escaped.

## Consequences
- The suite does not read xlsx exports/imports or vendor/venue attachments (nothing to compare against);
  those routes are covered by the same tenant-scoped lookups but not by this test.
- The metrics still read every tenant in memory (`ComputePlatformMetrics`). Fine for the first hundreds;
  move the counts to SQL aggregates once the tenant table is in the tens of thousands.
- A presigned link is a bearer secret for two minutes: it can be shared or end up in browser history. The window
  is short and the link is only made after authorising that exact document, but a deployment that cannot accept
  this keeps `S3_PRESIGN=false`.
- The browser must reach the storage endpoint. No CORS rule is needed (images, frames and navigation), but a
  private bucket behind a VPN needs the fallback.
- `LIKE '%term%'` searches still scan the tenant's rows; the tenant filter bounds them. Full-text search would
  be a separate decision.
- Not done: row-level security in MySQL (not available), per-tenant rate limits, and cursor pagination for
  very deep pages (OFFSET is fine at these sizes).
