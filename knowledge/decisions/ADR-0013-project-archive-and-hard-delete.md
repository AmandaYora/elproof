# ADR-0013: Project archive (soft, reversible) and hard delete (permanent, guarded)

## Status
Accepted

## Context
The user asked for two features on the `Project` entity (a wedding engagement, `projects` module):
an "Archive" to get old/inactive projects out of the way of day-to-day work, and a "Hard Delete" to
permanently remove one. This runs directly against an existing, explicit house rule
(`knowledge/DATABASE_GUIDE.md`, "Conventions"): *"Soft state over hard delete... don't introduce
hard deletes the frontend never asks for."* The one precedent anywhere in this codebase,
`ClientService.Delete` (`clients` module), is framed in its own doc comment as a narrow escape
hatch for one bug-workaround scenario, not a template to generalize. This ADR is the deliberate,
explicit exception the house rule anticipates — recorded here rather than silently deviating.

## Decision

### Archive: a separate, orthogonal boolean — not a `Status` value
`projects.is_archived` (default `FALSE`) is independent of the existing `Status` enum
(`Draft/Preparation/Ready/Completed/Cancelled`) — a project of *any* status can be archived or
unarchived. Reversible, single toggle (`POST /projects/{id}/toggle-archive`, no body, mirrors the
existing vendor/staff/category `SetActive` convention exactly), no confirmation dialog, no role
restriction. The project list page shows two genuinely disjoint views (active vs. archived, never
merged) rather than a mixed list with archived rows visually deprioritized — the point is getting
them fully out of the way, not just dimming them.

### Hard delete: guarded on three axes, all enforced server-side
Confirmed with the user before implementing (not assumed):
1. **Owner-only.** `projects` had zero role-gating anywhere before this — every other action (even
   `Cancel`) is available to any staff role. This is the first role check in the module, added
   inline in the handler (`claims.role != "Owner"`), matching `platform`'s tenant_handler.go idiom
   since no shared `middleware.RequireRole()` helper exists in this codebase.
2. **Must already be archived or cancelled.** Enforced in `ProjectService.Delete` itself (422 if
   neither), not just a frontend button-hide — prevents permanently destroying a project that's
   still actively being worked on.
3. **Client cleanup is best-effort, not transactional.** A hard delete's success is never
   contingent on every linked client cleaning up successfully — matches the user's explicit choice
   and `ClientService.Delete`'s own existing best-effort precedent (log and continue past an
   individual failure).

### Cascade scope and ordering
Everything in `projects`' own 8 sub-entity tables is deleted in one DB transaction
(`ProjectRepository.DeleteCascade`) — real FK constraints exist for all of them (same-module, per
`.claude/rules/database.md`), so ordering matters: `vendor_payments` references `evidence`
(`invoice_evidence_id`/`proof_evidence_id`), so it's deleted first; everything referencing
`project_vendors` goes before it; everything goes before the `projects` row itself. `client_payments`
(added later, PLAN.md "Uang Masuk dari Client") has its own direct FK straight to `projects` — no
`project_vendor_id`, unlike `vendor_payments` — so it has no ordering dependency on anything else
here; it just has to go before the final `projects` delete. Full order: `activity_log →
vendor_payments → client_payments → vendor_issues → vendor_milestones → evidence → project_vendors →
project_milestones → projects`.

Two things happen **outside** that transaction, both only after it commits, both best-effort:
- **Evidence's object-storage files** (`EvidenceService.DeleteStorageObjects`) — this is the first
  time `internal/shared/storage`'s `Delete` method (added in ADR-0006, never actually called
  anywhere until now) gets wired up to anything. A failure here is logged, never propagated — the
  worst case is an orphaned S3 object, which is strictly safer than the alternative ordering (S3
  deleted first, DB transaction fails, leaving a dangling reference to a file that's already gone).
- **Every client tied to the project** (`clients.Contracts.DeleteAllForProject`, new) — bridged via
  a second local interface, `application.ClientCleaner`, following the exact same two-phase
  wiring pattern already established for `ClientAccessResolver` (`clients` depends on
  `projects.Contracts()`, so `clients` must be built after `projects`, so `projects` can't take
  this as a constructor argument — see `MODULE_MAP.md`'s circular-dependency note). Internally loops
  `ClientService.Delete` (the codebase's one other hard-delete) per client, best-effort.

No activity-log entry is recorded for the deletion event itself — every `activity_log` row for this
project is deleted in the same transaction, so one would be wiped immediately after being written.

## Consequences
- `apps/api/internal/shared/middleware/cors.go` needed no change (unlike ADR-0012's `PUT`) — `DELETE`
  was already in the dev-mode CORS allow-list.
- `ProjectService` gained two new dependencies: `*EvidenceService` (constructor argument — evidence
  is built before `projectService` now, reordered in `projects.module.go`) and `ClientCleaner`
  (two-phase setter, like `ClientAccessResolver`).
- `GET /projects?all=true` (dashboard/global-search) is **not** archive-filtered by this change —
  archived projects still count in dashboard stats and global search. Only the real list page
  (`GET /projects`, paginated) splits into active/archived views. Revisit if dashboard stats should
  exclude archived projects too — not done here to avoid an unaudited change to dashboard math
  beyond what was asked.
- Verified end-to-end (interactive + direct API): archive/unarchive toggles and reactively
  shows/hides the delete button; the delete button is genuinely absent from the DOM (not just
  disabled) for a non-Owner or a not-yet-archived/cancelled project; a direct API bypass attempt
  for either guard is rejected server-side (403 / 422 respectively); a real hard delete confirmed
  via direct DB query that `projects`, `project_milestones`, `activity_log`, `evidence`, and the
  linked `clients` row are all gone, and confirmed via server log that the evidence file's S3
  object was deleted with no error.
