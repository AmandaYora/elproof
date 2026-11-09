# ADR-0032: Client proof — payment verification, milestone attachments, portal home, closed vendor directories

## Status
Accepted — implemented 2026-11-09 (docs/plan/revamp-ui-shadcn/PLAN.md Tahap 3, endpoints E4, E5, E6).

## Context
The Client Portal made a client trust the WO on its word: a payment showed as "received" with no sign of who
checked it or what proof backs it, a step's photos were never visible to the couple, the home screen needed ~9
calls and assembled money and news in the browser, and two endpoints the portal used for *names*
(`/vendors/summary`, `/vendor-categories`) handed every client the names of **all** vendors and categories of the
workspace.

## Decision
- **Verification model.** `client_payments` gains `recorded_by_staff_id`, `verified_at`, `verified_by_staff_id`
  (migration 65, all `NULL`able, primitive staff ids, no foreign key: a module never owns another module's table).
  `POST /projects/{id}/client-payments/{pid}/verify` (staff, `finance.record`) sets `verified_at = now` (whole
  seconds) and the actor. It is **idempotent** (an already verified payment is returned unchanged; the write is
  conditional on `verified_at IS NULL`, so two simultaneous verifies keep the first) and it **requires at least one
  evidence file**: otherwise 422 on `errors.evidence`. A verified payment must always have proof, so verification is
  **cleared** when what was paid changes (type, amount, date, method or reference; a note-only edit keeps it) and
  when its last evidence file is removed (`EvidenceService.Delete` notifies the payment service; no HTTP route deletes
  a single file yet, the rule is in place for when one is added). Both clearings are written to the activity log.
  There is no un-verify endpoint: editing the payment is the way to undo. `recordedBy` is internal and never sent to a
  client; `verifiedAt`/`verifiedBy` are.
- **Milestone attachment visibility.** A `projectMilestone` file is visible to a client iff
  `portal.timeline.view` **and** `is_client_visible`. Staff set the flag at upload or with the existing
  `toggle-client-visible`, which now accepts `general` and `projectMilestone` (other kinds still 422: their visibility
  follows what they hang off). Default is hidden. The rule lives in `domain.ClientCanSeeEvidence`, used by the list
  responses, the `/file` and `/link` routes and the portal home alike. Single-file routes are no longer gated up front
  by `portal.documents.view` (that would have made a payment proof unopenable for a role with finance but no
  documents); the per-file rule decides, and a shared general document now also needs `portal.documents.view` there.
  The `/file` and `/link` handlers judge visibility before touching storage, so a hidden file is a 404 whether or not
  the object exists.
- **One home call.** `GET /projects/me/home` assembles project, countdown, progress, finance (+ `payTo`), contact,
  actions, open-issue count, recent updates and PIC in one pass, computing only the sections the client's portal
  role may see; a forbidden section is absent, not `null`. Pure rules (remaining, next term, action selection,
  update ordering, proof choice) are unit-tested in `projects/domain`; the bank account and WhatsApp number are read
  from `platform` through a port (`TenantContactReader`), wired in `cmd/server`.
- **Names and proof inside filtered responses.** Vendor/category names (batched, workspace-scoped, for the project's
  own engagements only), issue vendor and step names, and `evidence[]` are added to the existing list responses.
- **Closed:** `GET /vendors/summary` and `GET /vendor-categories` are removed from the client allowlist
  (`cmd/server/client_scope.go`): a client gets 403. The isolation suite proves a client of one workspace cannot read
  another's project or home, sees only its own project's vendor names, and that neither directory is readable.

## Consequences
- Frontend can render "Diterima, diverifikasi oleh X" with the proof, and a Beranda in ≤ 2 calls.
- Verification depends on proof existing; a payment recorded before its proof was uploaded must be verified after.
- Staff must flag a step's attachment before the couple sees it; nothing is shared by accident.
- Any future vendor/category name a portal screen needs must come inside a project-scoped response, never from a
  directory endpoint.
