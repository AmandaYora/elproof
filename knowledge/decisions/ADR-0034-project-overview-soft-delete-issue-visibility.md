# ADR-0034: Project overview, names in responses, fresh progress on every mutation, soft delete with restore and purge, kendala without a vendor, dashboard sections

## Status
Accepted — implemented 2026-11-10 (docs/plan/revamp-ui-shadcn/PLAN.md Tahap 5, backend half: endpoints E1, E2, E8, E12, E13).

## Context
The project page loaded ~9 endpoints and computed money, "next term", margin and the "perlu tindakan" queue in the browser; the vendor
tab and the activity log needed names the API did not carry (the web looked them up in full directories, and showed a made-up "Sistem"
for actors it could not find); after any mutation the header stayed stale until a reload; deleting a payment, a client or a vendor was
irreversible (no "Urungkan"); a kendala needed a vendor and could not be kept from the couple; and the Beranda had no cash flow, no
vendor payments due and no unconfirmed-vendor list.

## Decision

### E1 — `GET /projects/{id}/overview`
One staff-only call (a client principal gets 403: it is not on the portal allowlist, because it carries margin and vendor costs).
`data` = `project` (exactly `GET /projects/{id}`), `finance` (contract, paid, remaining, overdue total, next term, the very objects of
`GET /payment-terms`, vendor cost, venue cost, and `margin` **only with `finance.margin.view`** — the key is absent otherwise),
`actions`, `nextMilestones`, `unconfirmedVendors`, `openIssueCount`, `team`, `clients`. The rules (`domain/overview.go`: actions,
next milestones, finance, margin) are pure and unit-tested; `application.OverviewService` only fetches. Progress, the actions and the
counts are computed from the **same seven progress sources** (`ProjectService.loadProgressData`, which `ComputeProgress` now uses too,
so the single-project progress no longer costs one query per vendor). Statements: 14 in the handler's own path plus one login lookup per
client — above the plan's "~10": the seven progress reads are separate queries, and merging them would have meant a bespoke SQL
view; not done. The clients come through a local `ClientDirectory` port (`projects/application`), adapted from `clients.Contracts` in
`cmd/server` (the same bridge idiom as `ClientCleaner`), so `projects` still never imports `clients`; a failure there is logged and the
overview answers with `clients: []` rather than failing.

### E2, E12 — names, vendor tab, activity
`GET …/vendors` items gain `picName` and `evidenceCount` (the engagement's own proof, its milestones' and its payments', judged per
caller with `domain.ClientCanSeeEvidence`; a client gets `picName: ""`). `GET …/activity` is **always paginated** (default 20, max 100,
the shared `meta`) and filterable by `entityType`/`entityId`; entries gain `actorName` from one batched staff lookup. **An actor that
does not resolve, or `0`, has `actorName: ""`: the server never invents a label** (the brief allowed "Sistem" only "when the entry
really has no actor" and in the same breath said the system actor is `""`; the server emits `""` for both and the web decides what to
show for `actorStaffId = 0`). Evidence rows carry `uploadedByName`, issues `picName` (both `""` for a client), and the dashboard rows
carry `vendorName` and `recentActivity[].actorName` — the brief said the dashboard rows "already resolve names"; they only had ids, so
this adds them.

### E8 — a fresh `projectProgress` on every mutation
One helper (`presentation/mutation.go`, `withExtras` + `Handler.mutated`) serialises the item exactly as before and appends
`"projectProgress"` (the progress after the change) and, for payments, `"ledgerSummary"`. A DELETE or a reorder that answered `null`
now answers `{"projectProgress": …}`. If the progress cannot be computed after a committed mutation the key is left out (a 5xx would
make a client retry a non-idempotent create). Covered: project and vendor milestones, kendala, vendor engagements (create, update,
cancel, delete, restore), the three payment kinds (create, update, delete, restore; `ledgerSummary` now on update and delete too), and
evidence (upload, toggle, **delete**). The brief lists "milestones … delete" and "evidence … delete": there is no milestone delete
route, and there was no evidence delete route, so `DELETE /projects/{id}/evidence/{eid}` was added (`project.edit`/`finance.record`,
plus `finance.payment.delete` for a payment's proof; `EvidenceService.Delete` already existed with the verification invariant).

### E13 — soft delete, restore, purge
* `deleted_at DATETIME NULL` (+ index) on `client_payments`, `vendor_payments`, `venue_payments`, `project_vendors`, `clients`
  (migration 68). `DELETE` stamps it (written by the application, so `restorableUntil = deleted_at + 24 h` is exact) and answers
  `{id, restorableUntil, projectProgress, ledgerSummary?}`; `POST …/restore` answers the item as its list shows it. Permissions are
  unchanged (`finance.payment.delete`; restore needs the same; vendors `project.edit`; clients `client.manage`). A verified payment is
  deletable like any other; its verification stays on the hidden row and returns on restore.
* **Every read of a live row filters the stamp** (audited: all repositories, the batched progress sources, the dashboard queries, the
  portal home, `CountActive`, the vendor's project history, the staff delete-impact). A deleted engagement hides its milestones,
  payments, kendala and their evidence by the engagement's own stamp (the children are not stamped, so restore brings everything back).
  Evidence rows have no foreign key to what they belong to, so the evidence repository adds, per `related_kind`, a "not in the deleted
  ids" predicate to every read (list, per project, per related item, by id, documents tab) and to the "may I attach to this?" check: a
  deleted payment's files are not listed, opened (`/file`, `/link` → 404) or attachable by anyone until restored.
* **Restore** re-checks what can now fail and answers 404 when the item is not deleted, never existed, is another workspace's, or was
  deleted ≥ 24 h ago (a row past retention that the purge has not reached counts as purged: the SQL for restore is
  `deleted_at > now − 24 h`, the exact complement of the purge's `deleted_at <= now − 24 h`). 422 with the usual keys: `amount`
  (`payment_exceeds_contract`) when adding a payment back would overpay what is owed (checked under the project lock in one
  transaction, like a create), `projectVendorId` when a vendor payment's engagement is itself deleted, `vendorId`/`categoryId` when an
  engagement's vendor or category was hard-deleted meanwhile, `projectId`/the trial-quota message for a client.
* **Uniqueness decisions.** `evidence.storage_path` stays UNIQUE: a deleted payment keeps its upload claimed, so its token cannot be
  reused by a new payment until the purge (or restore) — wrong re-use would otherwise attach one file to two payments. A deleted client
  keeps its login **deactivated, not removed**: restore is then always possible and the person keeps their password; the price is that
  the username and email stay reserved for up to 24 h (a re-created client with an explicit taken username gets the usual 422; a derived
  username picks another suffix). The purge deletes the login, which frees both.
* **Purge.** `api purge` and an in-process ticker (`PURGE_INTERVAL`, default 30 m, `0` = off) call each module's `PurgeExpired`. It is
  idempotent and takes a MySQL `GET_LOCK` per module on a dedicated connection (released with the connection, so a crash leaves
  nothing), so several instances can share one database. `projects` deletes, in batches of 200 per kind and one transaction each, the
  evidence rows first and then the expired payments; an expired engagement takes its kendala, payments, milestones and their evidence
  with it; stored objects are deleted after the commit (a storage failure leaves an orphan object, never a row pointing at a missing
  file). `clients` removes the login first (a failure leaves the row for the next run) and then the row. It logs the counts.
  The project hard delete (ADR-0013) now lists *all* evidence (hidden included) to remove the objects, and deletes clients including
  deleted ones.
* The activity log records `payment_deleted`, `vendor_deleted` and `item_restored`. **Clients are not logged**: the log belongs to
  `projects` and the clients module has no port to it (client changes were never logged); not done.
* Down migrations hard-delete what the old shape cannot hold (soft-deleted rows with their evidence, kendala without a vendor).

### E13 — kendala
`vendor_issues.project_vendor_id` is NULLable and `is_client_visible BOOLEAN NOT NULL DEFAULT 1` (migration 69; existing rows stay
visible). Create/update accept both (`isClientVisible` omitted: shown on create, unchanged on update; a `vendorMilestoneId` without a
vendor is a 422). Answers carry `projectVendorId` (`null` without a vendor), `isClientVisible`, `vendorName`, `vendorMilestoneName`,
`picName`. **A client only ever sees shared kendala**: the list, the home (`openIssueCount`, the `open_issues` action, `issue_resolved`
news), the progress count **and condition** that `GET /projects/{id}` and the home show (`ComputeClientProgress*`: a hidden Critical
kendala must not turn the client's condition to "At Risk"), and the proof files of a hidden kendala (a computed `ClientHidden` flag on
the evidence row, so `ClientCanSeeEvidence`, which every list/file/link/home path already uses, answers false). Places that assumed a
vendor: the dashboard's open-issues query (now a LEFT JOIN; the row has `vendorId: 0`, `vendorName: ""`), the issue repository, the
evidence ownership check.

### E13 — dashboard
`cashFlow` (8 Monday-based weeks from the week containing today; `expectedIn` = unpaid remainder of payment-schedule installments by
due date, from the same allocation the term status uses; `expectedOut` = unpaid remainder of non-cancelled vendor engagements that have
a due date; overdue counts in week 1, beyond week 8 is left out), `vendorPaymentsDue` (the same vendor balances, overdue or due ≤ 30
days, nearest first, ≤ 10) and `unconfirmedVendors` (Planned/Negotiation, event 0–30 days away, ≤ 10). Open projects only. `cashFlow` and
`vendorPaymentsDue` need `dashboard.revenue.view`: without it the keys are absent and the money is not even computed (the pointer-to-slice
makes "permitted but empty" `[]` and "not permitted" absent). The vendor balance of an engagement does not count its deleted
payments.

## Consequences
- The web can build the project page from one call plus the vendor tab from one, and update its header from any mutation's answer.
- "Urungkan" is real for payments, vendors and clients; the cost is a hidden-row filter in every read of those tables, enforced by
  review and by the isolation suite (a deleted payment disappears from the list, ledger, overview, portal home and dashboard, and its
  proof stops opening); a new query on these tables must add `deleted_at IS NULL` (or the helpers in `infrastructure/softdelete.go`).
- A deleted client's username/email and a deleted payment's upload token stay reserved for up to 24 h.
- `GET /activity` without parameters now returns the newest 20 entries only.
- Open: the overview costs 14+ statements (target ~10); the purge has no metrics beyond the log line; client delete/restore is not in the
  activity log.
