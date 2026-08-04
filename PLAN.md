# PLAN — Retire the standalone "Kendala" tab; move issues into Vendor/timeline context

## 0. Goal

Remove "Kendala" (Issue) as its own top-level tab in the Project detail page, and
instead let a kendala be recorded directly against a specific vendor engagement
and, optionally, one specific milestone of that vendor's own timeline —
eliminating the duplicate/disconnected experience where the same data is
today shown twice (a flat cross-vendor table on its own tab, and a read-only
mini-list repeated inside the Vendor tab's accordion) with no way to tell
*which* milestone a kendala is actually blocking.

## 1. Current state, confirmed precisely

**Tabs** (`ProjectDetailLayout.tsx:12-21`): Vendor → Venue → Timeline → Client →
Pembayaran → **Kendala** → Dokumen → Aktivitas. Each is a real route
(`/projects/{id}/{tab}`), not an in-page anchor — `ProjectIssuesTabPage.tsx`
just renders `<ProjectIssuesSection projectId={projectId} />`.

**Two unrelated "timeline" concepts already exist** and this matters for where
a kendala can sensibly attach:
- `ProjectMilestone` — the WO's own internal prep checklist (Timeline tab,
  `ProjectMilestonesSection.tsx`). **Has no vendor reference at all.**
- `VendorMilestone` — a specific vendor engagement's own deliverable timeline
  (`vendor_milestones` table, `project_vendor_id` FK), rendered inside the
  **Vendor** tab's per-vendor accordion (`ProjectVendorsSection.tsx:486-561`,
  a real `<tr key={m.id}>` per milestone with Timeline/Status/Target/Selesai/
  Evidence/Aksi columns).

`VendorIssue` (`apps/api/internal/modules/projects/domain/issue.go:24-38`)
already has `ProjectVendorID` — a kendala is already conceptually a
vendor-scoped thing, never a project-milestone-scoped thing. It has **no**
milestone-level linkage of any kind today.

**Duplication that already exists**: `ProjectVendorsSection.tsx:582-599`
renders a read-only "Kendala" mini-list per vendor (title + impact/status
badge only), right next to a "Pembayaran" mini-list, filtered by
`issues.filter(i => i.projectVendorId === pv.id)` — the exact same data the
standalone tab shows, just without found-date/PIC/plan and with no way to
add/edit from here.

**Kendala tab today** (`ProjectIssuesSection.tsx`) supports only **Create**
(`AddIssueModal`) and **status change** (a `<Select>` per row calling
`updateIssueStatus`) — there is no full edit and no delete. Create requires
picking a vendor (`projectVendorId`, required) but has no milestone field at
all (`issue.schema.ts:6-15`).

**Backend mutation surface**: `issue_endpoints.go` exposes `POST
.../issues` (create, all fields) and `PATCH .../issues/{id}` → `updateIssueStatus`
→ `IssueService.UpdateStatus`, which only ever touches `status`/`resolved_date`
(auto-stamped once) — the repository's `Update()` (`mysql_issue_repository.go:115-121`)
literally only writes `status, resolved_date, resolution_notes`, nothing else.
There is no service/repo method today that updates title/description/impact/
vendor/PIC/target-date after creation.

**`vendor_issues` schema** (`000008_create_project_tables.up.sql:89-108`):
`id, project_id, project_vendor_id, title, description, impact, found_date,
status, resolution_plan, pic_staff_id, target_resolution_date, resolved_date,
resolution_notes` — no milestone column.

**Consumers beyond the tab itself** (blast radius of removing the route):
- **Dashboard attention queue** (`attention.ts:28-37`) hard-links every open
  kendala card to `ROUTE_PATHS.projectDetail(issue.projectId, "kendala")`.
- **`ProjectEvidenceSection.tsx`** lets evidence be attached with
  `relatedKind: "issue"`, picking from a dropdown built from `issues` — this
  is unaffected by *where* issues are created/edited, only by the `issues`
  list continuing to exist, which it will.
- **Client Portal** has the *exact same duplication*, one layer further:
  a standalone `KendalaTabPage.tsx` (`/portal/kendala`, read-only `IssueCard`
  list) **and** `VendorProgressTabPage.tsx:73,115,166-179` already renders the
  same per-vendor issue cards inline, plus `RingkasanTabPage.tsx:31,55-62`
  shows just an open-issue *count* banner. This is a different portal/menu
  than what was asked about, so it's tracked as an explicit, separate
  decision (Phase H) rather than folded silently into this plan.
- **`ComputeProjectProgress`** (`domain/progress.go:89-140`) aggregates
  `openIssues` at the whole-project level only (count, critical/high count,
  feeding the On Track/Attention/At Risk condition) — this is untouched by
  everything in this plan; milestone linkage is purely descriptive metadata,
  it does not change scoring.

**Design decisions already confirmed with the user**:
1. Milestone link is **optional** (nullable) — a kendala can be "umum" (general
   to the vendor) or tied to one specific `VendorMilestone`. Forcing every
   kendala to pick a milestone would break the common "vendor tidak
   responsif" / "minta ubah termin" case that has nothing to do with any one
   deliverable.
2. Full **edit** capability is added now (not just status change) — since the
   whole create/status-only flow is being touched anyway, and "mudah
   mengelola" implies being able to fix a wrongly-picked vendor/milestone
   later without going to the DB directly.

## 2. Design

### Phase A — Migration: nullable milestone link

New `apps/api/migrations/000032_add_vendor_milestone_id_to_vendor_issues.up.sql`:
```sql
ALTER TABLE vendor_issues
  ADD COLUMN vendor_milestone_id BIGINT UNSIGNED NULL,
  ADD KEY idx_vendor_issues_vendor_milestone (vendor_milestone_id),
  ADD CONSTRAINT fk_vendor_issues_vendor_milestone
    FOREIGN KEY (vendor_milestone_id) REFERENCES vendor_milestones (id);
```
Same module (`projects` owns both `vendor_issues` and `vendor_milestones`), so
this FK is allowed under the modular-monolith rule. Existing rows backfill to
`NULL` automatically — no data migration needed, no behavior change for
existing kendala until a user explicitly links one.
`.down.sql` drops the FK/key/column in reverse order.

### Phase B — Backend domain + repository

- `domain.VendorIssue`: add `VendorMilestoneID *int64`.
- `mysql_issue_repository.go`: add `vendor_milestone_id` to `issueColumns`,
  `scanIssue` (nullable int64 scan), `Create` (insert the new column), and
  **broaden `Update`** to persist every editable field instead of just
  status/resolved_date/resolution_notes:
  ```go
  UPDATE vendor_issues SET title=?, description=?, project_vendor_id=?,
    vendor_milestone_id=?, impact=?, target_resolution_date=?, pic_staff_id=?,
    resolution_plan=?, status=?, resolved_date=?, resolution_notes=? WHERE id=?
  ```

### Phase C — Backend application service

- `IssueInput` gains `VendorMilestoneID *int64`.
- New **ownership validation**: if `VendorMilestoneID` is set, fetch it via
  the already-existing `VendorMilestoneRepository.FindByID(ctx, projectVendorID, id)`
  (same ownership-check idiom already used everywhere else in this module)
  and reject with a validation error if it doesn't belong to the issue's own
  `ProjectVendorID` — this is the one rule that must never be skippable,
  otherwise a kendala could be silently misattributed to a different
  vendor's milestone.
- Replace `IssueService.UpdateStatus` with a general
  `IssueService.Update(ctx, projectID, id, actorStaffID int64, input IssueInput) (*domain.VendorIssue, error)`
  that overwrites every field from `input` (mirroring how
  `VendorEngagementService.Update`/`UpdateMilestone` already work — full
  overwrite, not a partial patch) — the frontend's "quick status dropdown"
  will just call this with every existing field spread plus the new status,
  exactly like `ProjectVendorsSection.tsx`'s existing `quickStatusChange` /
  `toMilestoneUpdateFields` pattern for vendor milestones. One endpoint,
  one code path, no special-cased "status-only" branch to keep in sync.
- `resolvedDate` auto-stamp-once logic (`shouldStamp`, current
  `UpdateStatus:79-82`) moves into this new `Update`, unchanged in behavior.

### Phase D — Backend presentation

- `issueInputBody`/response DTOs: add `vendorMilestoneId *int64` (create) and
  to `issueResponse`/`toIssueResponse` (nullable int in JSON).
- Replace `updateIssueStatus` handler with `updateIssue`, decoding the full
  `issueInputBody` shape (same body shape as create, since it's a full
  overwrite) — same `PATCH /projects/{id}/issues/{issueId}` route, just a
  richer body and a richer response. `handler.go:165-166`'s dispatch line
  changes its target function only, not its route pattern.

### Phase E — Frontend types, schema, store

- `types.ts`: `VendorIssue.vendorMilestoneId: string | null`.
- `issue.schema.ts`: add `vendorMilestoneId: z.string().optional()` (empty
  string / omitted = "umum", not tied to any milestone).
- `useProjectStore.ts`: `RawIssue`/`toIssue` add the field; replace
  `updateIssueStatus` action with `updateIssue(projectId, issueId, values: IssueFormValues)`
  (full body); add a `toIssueUpdateFields(issue): IssueFormValues` helper next
  to the existing `toMilestoneUpdateFields`, used by the quick-status dropdown
  to spread-then-override `status` only, keeping today's one-click UX
  identical from the user's point of view.

### Phase F — Frontend UI: fold Kendala into the Vendor tab

All of this lands inside `ProjectVendorsSection.tsx` (no new top-level file):

1. **Per-milestone "+Kendala"**: add an icon action to each milestone row's
   existing "Aksi" cell (alongside Edit/Cancel) that opens the issue form
   modal with `projectVendorId` and `vendorMilestoneId` **pre-filled and
   locked** from row context — zero extra picking for the common case.
2. **Vendor-level "+Kendala"**: keep one add-entry-point at the vendor's
   "Kendala" mini-list header (where the section already lives) for the
   general/not-tied-to-one-milestone case — same modal, milestone field left
   as an explicit "Umum / tidak terikat ke timeline" option instead of being
   hidden.
3. **Read affordance** (the actual "mudah membaca" ask): a small "⚠ N"
   badge on any milestone row that has ≥1 open kendala tied to it (same visual
   slot as the existing Evidence badge), so a blocked deliverable is visible
   without opening anything. The vendor-level mini-list keeps showing every
   kendala for that vendor (both milestone-tied and general) but now also
   shows which milestone each one belongs to, or "Umum" if none.
4. **Edit**: add a pencil icon per row in the mini-list opening the same
   form component pre-filled with every field (title/description/impact/
   vendor/milestone/PIC/target date/resolution plan) — mirrors the existing
   create/edit dual-mode pattern already used by `ProjectVendorFormModal`
   (an optional `initial*` prop switches the same component between modes).
5. Remove the tab: delete the `{ to: "kendala", label: "Kendala" }` entry
   from `ProjectDetailLayout.tsx:18`, delete the `kendala` route from
   `protected.routes.tsx:58`, delete `ProjectIssuesTabPage.tsx`. Fold whatever
   of `ProjectIssuesSection.tsx` is still needed (the modal, mostly) into
   `ProjectVendorsSection.tsx` and delete the rest.
6. `route-paths.ts`: drop `"kendala"` from `ProjectDetailTab`.

### Phase G — Dashboard link fix

`attention.ts:35` — change the open-kendala attention card's `to` from
`ROUTE_PATHS.projectDetail(issue.projectId, "kendala")` to
`ROUTE_PATHS.projectDetail(issue.projectId, "vendor")`, matching how the
existing "Timeline Terlambat" attention card (`attention.ts:46`) already links
to the `"vendor"` tab for the same reason. No deep-link-to-specific-vendor-row
mechanism exists anywhere else in the app today (the overdue-milestone card
doesn't have one either) — not introducing one here either, to stay
consistent and avoid scope creep.

### Phase H — Client Portal (explicit decision point, not auto-applied)

`KendalaTabPage.tsx` (`/portal/kendala`) is read-only and duplicates
`VendorProgressTabPage.tsx`'s own per-vendor `IssueCard` list — the identical
redundancy this whole plan removes on the WO Console side. Recommended
mirror treatment: drop `"kendala"` from `client-portal.routes.tsx` and
`ClientPortalTab`, delete `KendalaTabPage.tsx`, and have
`VendorProgressTabPage.tsx`'s existing per-vendor issue cards show the
milestone name (Phase I) so the client sees the same "which deliverable is
this about" context. **This is scoped separately from "menu Project" that was
asked about — confirm before implementing**, since it touches the
client-facing portal, a different audience than the WO Console.

### Phase I — `IssueCard` / mini-list milestone display

`IssueCard.tsx` (shared by Client Portal's Kendala tab today and
`VendorProgressTabPage.tsx`) and the WO Console's vendor mini-list both gain
a milestone name line — "{vendor} · {milestone name}" or "{vendor} · Kendala
Umum" when `vendorMilestoneId` is null. Purely additive/display-only, no
behavior change.

## 3. Order of implementation

1. Phase A (migration) — pure schema addition, zero code risk, ship first.
2. Phase B + C (domain/repo/service) — `go build`/`go vet` before moving on.
3. Phase D (presentation/DTO) — same endpoint path, richer body; `go build`/`go vet`.
4. Phase E (frontend types/schema/store) — `tsc --noEmit`.
5. Phase F (the actual UI move: per-milestone + per-vendor add, edit, badges,
   tab removal) — the one step users will actually see change.
6. Phase G (dashboard link) — one-line fix, do right after F so the link is
   never briefly dangling.
7. Phase I (milestone name display in IssueCard/mini-list) — small, additive.
8. Phase H (Client Portal) — **only after explicit confirmation**, since it's
   a separate portal from what was originally asked about.

## 4. Verification

- `go build ./...` / `go vet ./...` after every backend phase.
- `npx tsc --noEmit` / `npm run build` after every frontend phase.
- Manual check: create a kendala from a milestone row → confirm it's
  pre-filled/locked to that vendor+milestone, appears as a badge on that row,
  and appears in that vendor's mini-list labeled with the milestone name.
- Manual check: create a "Kendala Umum" from the vendor-level button →
  confirm it has no milestone badge anywhere and shows "Kendala Umum" in the
  mini-list.
- Manual check: edit an existing kendala (change vendor, change milestone,
  change impact) → confirm the ownership validation (Phase C) rejects a
  milestone that doesn't belong to the newly-chosen vendor.
- Manual check: dashboard's "Kendala Aktif" attention card still navigates
  successfully (to the Vendor tab, not a 404/blank "kendala" route).
- Confirm `ComputeProjectProgress`'s On Track/Attention/At Risk output is
  unchanged for a fixed project before/after — milestone linkage must stay
  purely descriptive.
- Regression check on `ProjectEvidenceSection.tsx`'s issue-picker dropdown
  (evidence attached to a kendala) — still lists every issue correctly by
  title + vendor regardless of where it was created.
