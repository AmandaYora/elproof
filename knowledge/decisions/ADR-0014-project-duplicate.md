# ADR-0014: Project duplicate (structural template, not a full clone)

## Status
Accepted

## Context
The user often builds a new project that's similar to one already handled — same kind of vendor
lineup, same milestone checklist — and wanted a "Duplicate Project" feature so they don't have to
set that structure up from scratch every time.

A `Project` (see `MODULE_MAP.md`) owns 7 sub-entity tables. Some of that data is structural/reusable
(milestone checklist, vendor lineup) and some is inherently historical/transactional (payments,
issues, evidence, activity log) or identity-specific (clients). Blindly cloning everything would
carry over data that has no meaning for a project that hasn't actually happened yet — e.g. a cloned
project would appear to already have received payments, or come with photos/documents belonging to
someone else's wedding.

## Decision

Confirmed with the user before implementing (three open questions, not assumed):

### 1. Scope: copy the structural template, exclude history
`ProjectService.Duplicate` clones:
- The new project's own core fields — supplied by the caller (see below), not copied server-side.
- Every **Project Milestone**, reset to `Not Started` / no `CompletedDate`.
- Every **Vendor Engagement** (`project_vendors`), reset to `EngagementStatus: Planned`,
  `DPAmount`/`PaidAmount: 0` — and every one of *its* **Vendor Milestones**, reset the same way as
  Project Milestones.

Deliberately **excluded**: `vendor_payments`, `vendor_issues`, `evidence`, `activity_log`, and every
`clients` row tied to the source project. All five are either genuinely historical (what actually
happened on the source project) or identity-specific (the source couple's own contacts) — neither
has any meaning attached to a not-yet-started duplicate. This mirrors the same exclusion list
`ProjectRepository.DeleteCascade` touches for the *opposite* reason (ADR-0013) — same boundary,
different direction.

### 2. Dates: copied verbatim, not shifted
Every date on every cloned row (the new project's own `eventDate`/`prepStartDate`, every milestone's
`targetDate`, every vendor engagement's `bookingDate`/`dueDate`) is copied as-is — no automatic
"shift by N days relative to the new event date" logic. The user edits whatever's stale by hand
afterward, the same way they'd edit any other field. This was a deliberate simplicity trade-off
(the alternative — requiring a new event date upfront and recomputing every downstream date's
offset — was considered and explicitly not chosen) rather than an oversight.

One field is *not* "copied verbatim" despite this: `ProjectVendor.EventDate` is a denormalized
mirror of the parent project's own event date (every other creation path sets it that way — see
`useProjectStore.ts`'s `vendorEngagementInputBody`, which always fills it from
`currentProject.eventDate`), not independent information. Cloning it from the *new* project
(`p.EventDate`) rather than the source engagement keeps that existing invariant intact instead of
introducing a fresh inconsistency the moment the duplicate is created.

### 3. Identity fields: pre-filled, not blanked
The frontend reuses the existing `ProjectFormModal` (the same component behind create/edit) in a
new `mode="duplicate"`, pre-filled with every field from the source project — including
`name`/`brideName`/`groomName`/dates — rather than forcing the user to re-type everything from a
blank form. Two fields get a safer default than a literal copy: `name` gets a `" (Salinan)"` suffix
(so submitting without editing anything doesn't silently create two identically-named projects),
and `status` resets to `Draft` (copying the source's live status — e.g. `Completed`/`Cancelled` —
would be a confusing starting point for a project that hasn't started). Both remain fully editable
in the same form before submit, exactly like every other field.

## Consequences
- New endpoint `POST /projects/{id}/duplicate`, same request/response shape as create — see
  `docs/API_CONTRACT.md`. At the time this shipped, no role restriction applied (matched `Create`'s
  permission model — any staff, not an Owner-only action like ADR-0013's hard delete, since
  duplicating destroys nothing). **Superseded by ADR-0017**: both `Create` and `Duplicate` are now
  Owner/Admin only — a Wedding Planner only ever operates within a project already assigned to
  them, never creates the resulting copy themselves.
- `ProjectService.Duplicate` orchestrates: create the new `projects` row from the caller's input,
  then clone milestones, then clone vendor engagements (and their vendor milestones) — sequential,
  not wrapped in one cross-repository DB transaction. This matches the existing precedent already
  set by `Create` itself (which also isn't transactional across its own `Create` +
  `seedDefaultMilestones` steps) — a partial failure leaves, at worst, a new project with fewer
  cloned rows than intended, not a corrupted/orphaned reference, and is recoverable by re-adding the
  missing rows by hand.
- `Duplicate` does not call `seedDefaultMilestones` — the whole point is carrying over the *source*
  project's actual current milestones (which may already differ from the 6-item default template
  `Create` seeds), not reseeding the generic default again.
- No new database migration — no schema change, purely new application/presentation-layer code plus
  a new frontend mode on an existing component.
- Frontend: `ProjectHeaderCard` gained a "Duplikat Project" button (next to "Ubah Project"),
  opening `ProjectFormModal` in `mode="duplicate"`; on success the user is redirected straight to
  the new project's detail page to start adjusting whatever's different.
