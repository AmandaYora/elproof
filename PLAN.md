# PLAN — Milestone editability/reordering + Combobox scroll-closes bug

Status: **Implemented.** All three items below have been built, build/type-checked, and
interactively verified (Playwright + direct backend API testing). Kept here as the design record.

Covers three verified issues, found and confirmed by direct code inspection this session:

1. A project milestone's Target Date and Completed Date cannot be changed once created (true for
   every milestone — seeded defaults and manually-created ones alike; there is nothing special
   about the seeded "Persiapan Acara" row).
2. There is no way to reorder milestones after creation (same for both project milestones and the
   separate vendor-milestone entity).
3. The shared searchable dropdown component (`Select`/`Combobox`, used all over the app) closes
   itself the instant its own options list is scrolled, making any list too long to fit its
   `max-h-52` panel partly unreachable.

Items 1–2 are gaps, not intentional restrictions — the codebase already has a working precedent for
full milestone editing (vendor milestones' `UpdateMilestone` + `VendorMilestoneEditModal`) that
project milestones never got. Item 3 is a genuine regression-class bug in shared UI, unrelated to
milestones — grouped here only because it was found and verified in the same session.

## 1. Editable Target Date / Completed Date

### Current behavior (confirmed)

- `apps/api/internal/modules/projects/application/project_service.go:189` — `UpdateMilestoneStatus`
  is the only mutation path after creation. It sets `Status` only; `CompletedDate` is auto-stamped
  with `time.Now()` the first time status becomes `Completed` and can never be supplied or
  corrected by the client; `TargetDate` is never touched.
- `apps/api/internal/modules/projects/infrastructure/mysql_project_repository.go:211-217` — the
  repository's `Update` SQL (`SET name = ?, status = ?, target_date = ?, completed_date = ?`)
  already has the columns needed — the application layer just never populates new values for
  `target_date`/`completed_date` before calling it.
- `apps/web/src/modules/projects/components/detail/ProjectMilestonesSection.tsx:120-129/173-177` —
  both dates render as plain read-only text; the only interactive controls per row are the Status
  `<Select>` and a Cancel/Reactivate icon button.
- Working precedent already in the same module: `vendor_engagement_service.go:174-190`
  (`UpdateMilestone`) sets `Status`, `TargetDate`, and `CompletedDate` directly from client input,
  paired with `VendorMilestoneEditModal.tsx` on the frontend (`<Input type="date">` fields, empty
  string = no completed date, auto-fills `completedDate` to today when status flips to `Completed`
  if not already set — `VendorMilestoneEditModal.tsx:85-86`).

### Design

Broaden the *existing* `PATCH /api/v1/projects/{id}/milestones/{milestoneId}` endpoint (currently
`updateMilestoneStatus`) into a full `updateMilestone`, mirroring the vendor-milestone shape as
closely as project milestones' simpler domain allows (no `PICStaffID`/`Description`/`Notes` fields
exist on `domain.ProjectMilestone` — out of scope, not part of the verified gap). Do **not** add a
separate new route for this — same path/method, richer body, same as how vendor milestones already
work. `Name` editing is deliberately **out of scope**: the verified gap is specifically the two
dates; renaming was never reported as broken and isn't touched here.

- **Backend**
  - `apps/api/internal/modules/projects/application/project_service.go`: replace
    `UpdateMilestoneStatus`'s narrow signature with `UpdateMilestone(ctx, tenantID, projectID,
    milestoneID, actorStaffID int64, input MilestoneUpdateInput) (*domain.ProjectMilestone, error)`,
    where `MilestoneUpdateInput{ Status domain.MilestoneStatus; TargetDate time.Time; CompletedDate
    *time.Time }`. Sets all three fields directly from input — no more server-side auto-stamping of
    `CompletedDate`; the client (frontend) decides what to send (mirroring
    `VendorMilestoneEditModal`'s auto-fill-today-but-editable UX instead of a hard server-side
    stamp).
  - `apps/api/internal/modules/projects/presentation/project_endpoints.go:216-237`: broaden
    `milestoneStatusBody` into `milestoneUpdateBody{ Status string; TargetDate string;
    CompletedDate *string }`, parse both dates (empty/`nil` → no completed date), call the new
    service method.
  - No migration needed — `target_date`/`completed_date` columns and the repository `UPDATE`
    already support this (`000008_create_project_tables.up.sql`).
- **Frontend**
  - New `ProjectMilestoneEditModal.tsx` (sibling to `ProjectMilestoneFormModal.tsx`, closely
    mirroring `VendorMilestoneEditModal.tsx`): shows the milestone name as read-only context, Status
    `<Select>`, Target Date `<Input type="date">`, Completed Date `<Input type="date">` (clearable),
    same "auto-fill today when flipped to Completed, still editable" convenience.
  - `ProjectMilestonesSection.tsx`: add a pencil "Edit" `IconActionButton` per row (both the mobile
    `CardList` and desktop `Table` renderings) opening the new modal. Keep the existing inline
    Status `<Select>` as-is for quick status-only changes — it still works fine and isn't part of
    the verified gap; the new modal is for when dates need correcting too.
  - `useProjectStore.ts`: rename/replace `updateMilestoneStatus` with `updateMilestone(projectId,
    milestoneId, fields: MilestoneUpdateFields)` (`{status, targetDate, completedDate}`), PATCH to
    the same `API.projects.milestone(projectId, milestoneId)` endpoint, mirroring
    `updateVendorMilestone`'s existing shape exactly.

## 2. Milestone reordering

### Current behavior (confirmed)

- `SortOrder` is set exactly once, at creation, via `NextSortOrder` (MAX+1) —
  `project_service.go:173`, `mysql_project_repository.go:219-226`.
- The repository's `Update` SQL does not include `sort_order` in its `SET` clause — structurally
  impossible to change post-creation via the existing path.
- No `Reorder`/`Move`/`Swap` method or route exists anywhere in the module. Same true for vendor
  milestones (separate entity, identical gap, not covered here — out of scope, not requested).
- No drag-and-drop library is installed (`apps/web/package.json` has neither `@dnd-kit` nor
  `react-beautiful-dnd` nor similar).

### Design decision: up/down buttons, not drag-and-drop

Recommending simple up/down arrow icon buttons over adding a drag-and-drop dependency:
- Zero new dependencies (this codebase consistently avoids adding libraries beyond what's already
  established — Zustand/Zod/Axios/lazy routes, nothing fancier).
- Drag-and-drop is awkward on touch/mobile, and this app has had a strong mobile-responsiveness
  focus all session (CardList mobile fallback pattern used everywhere already).
- Up/down buttons reuse the exact same `IconActionButton` component already used for
  Cancel/Reactivate on the same rows — no new UI primitive needed.

Cancelled milestones are visually sorted to the bottom regardless of their stored `sort_order`
(`sortMilestones`, `ProjectMilestonesSection.tsx:23-30`) — reordering buttons only make sense
*within* the active (non-cancelled) set, since a cancelled row's on-screen position is fixed
regardless of what its `sort_order` says. **Design call: disable up/down on cancelled milestones**,
and have "up"/"down" move a milestone only relative to other non-cancelled milestones, to avoid a
confusing interaction between the two ordering rules.

- **Backend**
  - New endpoint on the *collection* path (not the per-item one, to avoid any route-matching
    ambiguity with `PATCH /milestones/{milestoneId}`): `PATCH /api/v1/projects/{id}/milestones`
    (currently only `GET`/`POST` are handled at `len(rest)==1 && rest[0]=="milestones"` in
    `handler.go:113-116`) — body `{ orderedIds: number[] }`, the full new order of **all** of that
    project's milestone IDs (simplest to reason about: rewrite `sort_order` to array position,
    rather than a "swap two items" delta that has to handle edge cases).
  - `project_service.go`: new `ReorderMilestones(ctx, tenantID, projectID int64, actorStaffID
    int64, orderedIDs []int64) error` — loads the project's current milestones, validates
    `orderedIDs` is an exact permutation of the existing milestone ID set (same length, no
    missing/extra/duplicate IDs — reject with a validation error otherwise so a stale/corrupted
    client payload can't silently drop or duplicate a row), then persists.
  - New repository method `ReorderMilestones(ctx, projectID int64, orderedIDs []int64) error` —
    inside one transaction, `UPDATE project_milestones SET sort_order = ? WHERE id = ? AND
    project_id = ?` per position (small per-project milestone counts, no need for a fancier
    single-statement `CASE` update).
  - Record one activity log entry ("Urutan milestone diubah") per reorder call, matching the
    existing `s.activity.Record(...)` convention used by every other milestone mutation.
- **Frontend**
  - `useProjectStore.ts`: new `reorderMilestones(projectId, orderedIds: string[])` — PATCH to
    `API.projects.milestones(projectId)`, then refetch (same pattern as every other mutation here).
  - `ProjectMilestonesSection.tsx`: add "Naik"/"Turun" `IconActionButton`s per non-cancelled row
    (both CardList and Table renderings), disabled at the first/last position *within the
    non-cancelled subset*. Clicking swaps that milestone with its non-cancelled neighbor in the
    array, then calls `reorderMilestones` with the resulting full ID order (cancelled IDs kept in
    their existing relative positions in the payload — their absolute `sort_order` values don't
    matter for display, only non-cancelled ones do, but the payload must still include every ID per
    the backend's exact-permutation validation above).

## 3. Testing plan (once implemented)

Same pattern as every other feature this session — build/type-check, then a temporary
`DevPreview.tsx` + interactive Playwright pass:

1. Edit modal: open it on an existing milestone, change Target Date and Completed Date (including
   clearing Completed Date back to empty), save, confirm the table/cardlist reflects the new
   values and confirm status-flip-to-Completed still auto-fills today's date but stays editable.
2. Confirm the existing inline Status dropdown still works unchanged.
3. Reorder: click "Naik"/"Turun" on a middle milestone, confirm it swaps position with its
   neighbor and persists after a refetch/reload; confirm the buttons are disabled at the top/bottom
   of the non-cancelled subset; confirm a cancelled milestone has no up/down buttons and stays
   pinned at the bottom regardless of the active milestones being reordered around it.
4. Backend: directly exercise `PATCH .../milestones` with a payload that omits/duplicates an ID and
   confirm it's rejected with a clear validation error rather than silently corrupting order.

## 4. Explicitly out of scope (milestones)

- Milestone `Name` editing (not part of the verified gap).
- Vendor-milestone reordering (same gap exists there but wasn't part of what was verified/requested
  this round).
- Any drag-and-drop interaction.

## 5. Bug fix: searchable dropdown (`Combobox`/`Select`) closes when scrolling its own list

### Current behavior (confirmed)

- `apps/web/src/shared/components/ui/Combobox.tsx:92-113` — while the dropdown is `open`, a
  capturing (`useCapture: true`) listener is attached: `window.addEventListener("scroll",
  onViewportChange, true)` (line 106). `onViewportChange` (lines 101-103) unconditionally calls
  `closeDropdown()` for **any** scroll event in the document, with no check of `e.target`.
- Because the capture phase dispatches top-down for every scroll event regardless of where it
  originates, scrolling the dropdown's own options list (rendered inside a `createPortal` panel,
  `panelRef`) also fires this handler — closing the dropdown instead of letting the list scroll.
- Confirmed this is unintended: the options `<ul>` (line 181) is explicitly
  `max-h-52 overflow-y-auto`, i.e. built to scroll internally once there are enough options to
  overflow that height. The close-on-any-scroll listener directly defeats that.
- The listener's real purpose (confirmed from `openDropdown`, lines 72-79) is to close the panel if
  the *page* scrolls, since `rect` — the portal's `position: fixed` coordinates — is computed once
  at open time from `triggerRef.getBoundingClientRect()` and never recalculated afterward. If the
  page scrolls without closing the dropdown, the fixed panel would visually drift away from its
  trigger button. This outer-scroll-closes behavior is correct and should be preserved — only the
  internal-list case needs excluding.
- Only one component has this pattern (confirmed via a repo-wide grep for
  `addEventListener("scroll"`) — the fix is isolated to this single file, and since `Select` is a
  shared component, fixing it here fixes every screen that uses it at once (vendor/PIC pickers in
  `ProjectVendorFormModal.tsx`, category/role filters in `VendorListPage.tsx`/`ClientListPage.tsx`,
  and form modals across `platform-admin`/`projects`/`users`).

### Fix

Minimal, surgical change to `onViewportChange` — ignore scroll events whose target is inside the
dropdown's own panel, so only genuine outer/page scroll still closes it:

```tsx
function onViewportChange(e: Event) {
  if (panelRef.current?.contains(e.target as Node)) return;
  closeDropdown();
}
```

No other change needed:
- The same handler is also used for the `resize` listener (line 107); a `resize` event's target is
  always `window`, never inside `panelRef`, so the new guard is a no-op there — resize-closes
  behavior is unaffected.
- No CSS/markup change — `max-h-52 overflow-y-auto` on the `<ul>` already does the right thing once
  the listener stops fighting it.
- No change needed anywhere else — this is a one-component, one-function fix.

### Testing plan (once implemented)

1. Open a `Select` with enough options to overflow `max-h-52` (e.g. a plan/category picker seeded
   with many entries, or temporarily widen a `DevPreview.tsx` mock's option list) — confirm
   scrolling the options list scrolls it and does not close the dropdown.
2. Confirm scrolling the *page* behind an open dropdown still closes it (regression check for the
   preserved outer-scroll-closes behavior).
3. Confirm click-outside-closes and `Escape`-closes still work unchanged (lines 96-100, 133-136 —
   untouched by this fix).
4. Spot-check on at least one real usage site end-to-end (e.g. the vendor picker in
   `ProjectVendorFormModal.tsx`) rather than only a synthetic test, since this is a shared component
   used across many screens.
