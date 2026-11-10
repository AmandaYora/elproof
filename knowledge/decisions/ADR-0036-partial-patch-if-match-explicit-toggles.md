# ADR-0036: Partial PATCH, optimistic concurrency with `If-Match`, and explicit state instead of toggles

## Status
Accepted — 2026-11-10 (docs/plan/revamp-ui-shadcn/PLAN.md Tahap 8, backend half: endpoint E9).

## Context
Three things made an edit unsafe or clumsy:

1. **Full-body PATCH.** `PATCH` on a project, a project milestone, a vendor milestone and a kendala wanted every field again (the issue endpoint's own comment says "a full
   overwrite, not a partial patch"). To change one thing the web spread the old row and replaced one field, so an in-place edit also re-sent — and silently
   overwrote — every other field another person might have changed meanwhile.
2. **Toggles.** `POST …/toggle-archive`, `…/toggle-active`, `…/toggle-client-visible` flip whatever the state is now. A double click, a retry after a timeout, or an undo that
   arrives after someone else already flipped it, undoes itself or the other person's choice.
3. **No version.** Nothing told the server which version of a row the person was looking at, so two staff editing the same item overwrote each other with no sign.

## Decision

### 1. Partial PATCH (JSON merge by presence)
`PATCH` on the resources below changes **only the fields present in the body**. A field that is absent keeps its value; a field that is present is validated
exactly as on create (same rules, same messages, same 422 `errors` keyed by field). `null` clears a field only where the field is nullable today (dates that may be
empty, `vendorMilestoneId`, `picStaffId` where optional); sending `null` for a required field is a 422 for that field. An empty body `{}` is a 422 (`errors.body`) so a
client bug does not look like success. Unknown fields are ignored, as today.

The handlers decode into pointer fields (or a `map[string]json.RawMessage` plus the typed struct) so "absent" and "zero value" are different things; **the service layer
receives only what was sent** (an input struct of pointers/`Optional[T]`), never a full row built in the handler, and applies it to the row it just loaded.
Business rules that look at several fields (a milestone's dates, a payment's amount against the contract, an issue's resolution rules) are checked against the **merged**
result, not the body alone.

Resources: project (`PATCH /projects/{id}`), project milestone (`/projects/{id}/milestones/{mid}`), project vendor (`/projects/{id}/vendors/{pvid}`), vendor milestone
(`/projects/{id}/vendors/{pvid}/milestones/{mid}`), kendala (`/projects/{id}/issues/{iid}`), client payment, vendor payment and venue payment
(`/projects/{id}/client-payments|payments|venue-payments/{pid}`).

### 2. Optimistic concurrency: `If-Match`
* **The version is the row's `updated_at`**, now `TIMESTAMP(6)` (microseconds). A `TIMESTAMP` of whole seconds is not a version: two edits within one second would look
  identical. Migration `000071` gives `updated_at` the definition `TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6)` on `projects`,
  `project_milestones`, `project_vendors`, `vendor_milestones`, `vendor_issues`, `client_payments`, `vendor_payments` (the table behind `/payments`), `venue_payments`
  and `evidence` (table names verified against `apps/api/migrations`). MySQL bumps it on any change to the row, whichever code path made it, so nobody has to remember to.
  **Only `projects` had the column** (whole seconds); the other eight tables never did, so the migration *adds* it there (every existing row starts at the instant of the
  migration) and *modifies* it on `projects`.
* **Every item response in scope carries `updatedAt`** as RFC 3339 with microseconds in UTC (`2026-11-10T08:15:30.123456Z`) and sets `ETag: "<same value>"`. Lists carry
  `updatedAt` per item. (Where an item response already had `updatedAt` its format is aligned to this.)
* **A PATCH may send `If-Match: "<updatedAt>"`** (quotes optional). The write is a compare-and-set in the same statement — `UPDATE … SET … WHERE id = ? AND tenant_id = ? AND
  updated_at = ?` — never read-then-write, so it is safe under concurrency. Zero rows affected is disambiguated by one more read: row gone or other tenant → `404`
  exactly as today; row present → **`409`**:
  `{ success: false, message: "Data ini sudah diubah orang lain. Muat ulang untuk melihat versi terbaru.", errors: { _conflict: ["stale"] }, data: { current: <the item as a GET returns it> } }`.
  Returning the current item lets the web show what changed without a second call.
* **The header was optional in this transition (superseded: required for field edits since Tahap 9, see the last section).** Without it the write is last-write-wins as before (callers not yet moved keep working; this is the only break in
  E9 and it is transitional). A malformed value is `400`. Tahap 9 decides whether to make it required (`428 Precondition Required`) once every caller sends it.
* **A successful PATCH answers with the new item (and the new `ETag`)**, in the same envelope the mutation already returns (`h.mutated(...)` extras such as
  `projectProgress` and `ledgerSummary` stay).
* Soft-deleted rows behave as today (not found for PATCH; restore is its own call). Restore and the state transitions (`cancel`, `transition`, `verify`) are explicit
  actions already and are not touched.

### 3. Explicit state instead of toggles
| Old (removed 2026-11-10) | New |
|---|---|
| `POST /projects/{id}/toggle-archive` | `PATCH /projects/{id}` with `{ "isArchived": true\|false }` (part of the partial patch; the project response gains `isArchived` if it lacks it) |
| `POST /projects/{id}/evidence/{eid}/toggle-client-visible` | `PATCH /projects/{id}/evidence/{eid}` with `{ "isClientVisible": true\|false }` (new route) |
| `POST /vendors/{id}/toggle-active`, `/venues/{id}/…`, `/vendor-categories/{id}/…`, `/clients/{id}/…`, `/staff/{id}/…` | `PATCH /<resource>/{id}` with `{ "isActive": true\|false }` (all five already had a `PATCH` for the ordinary edit, so this is not a new route: a body that is `isActive` alone is the state change, see Implementation notes) |

Setting a state that already holds is **not an error and not a write**: `200` with the item as it is (idempotent). The effects the toggle already had (a deactivated
client may not sign in, a deactivated staff loses sessions, an archived project leaves default lists) are unchanged and tied to the transition `false → true` /
`true → false` only. Platform-console toggles (`toggle-suspension`, plans, payment apps, platform admins) are out of scope: they are not edited by two people at once and
have no in-place edit UI; they stay.

**Removal of the old routes:** this ADR's backend half shipped the new routes **and kept the old ones** (a toggle route read the current state and delegated to the
same service call, so behaviour was identical). **The old routes were removed on 2026-11-10**, in the same Tahap, once `apps/web` no longer called any of them (grep
proved it): `POST /projects/{id}/toggle-archive`, `POST /projects/{id}/evidence/{eid}/toggle-client-visible` and `POST /vendors|venues|vendor-categories|clients|staff/{id}/toggle-active`,
with their handlers and the service helper only they used (`EvidenceService.ToggleClientVisible`) and the permission routing for them. A request to one of them now
gets exactly the answer any unknown route of its subtree gets: `404` `Endpoint tidak ditemukan` (for the projects subtree and the clients and staff subtrees after the
usual ownership/permission gates, so a caller without the gate's permission may still see its `403`, as for any other unknown path). `docs/API_CONTRACT.md` was brought in line in Tahap 9 (the Postman collection covers only the external payment gateway, which this did not touch).

### 4. Audit and activity
A PATCH writes one activity/audit entry naming only the fields that actually changed (as the full-body update did for the fields it compared). A no-op PATCH (every
sent value equal to the stored one) writes nothing and returns `200` with the unchanged item.

## Consequences
* A person editing one field no longer re-sends, and cannot overwrite, the others. Two people on the same item get a clear 409 and the current data instead of a silent loss.
* Retried and double-clicked state changes are harmless.
* `updated_at` is `TIMESTAMP(6)` on nine tables (added on eight, widened on `projects`): an in-place `ALTER` rebuild, trivial at this size. The repositories hand-write
  their SQL with `database/sql` (the repo has no sqlc queries, `sqlc generate` has nothing to generate and fails with "no queries" before and after this change), so
  there were no generated models to refresh.
* One more column of the response to keep (`updatedAt`) and an `ETag` header to set; the web keeps the last seen `updatedAt` per edited item.
* The `If-Match` check covers the rows in section 1 only. A person who adds a payment while another edits the project is not a conflict (different rows); two people editing
  different fields of the **same** row **are** a conflict by design (the second must reload) — finer granularity (per field) was rejected as complexity without a user need.
* Making `If-Match` mandatory is done (Tahap 9, last section); the old toggle routes are gone from `docs/API_CONTRACT.md` (Tahap 9).

## Implementation notes
Backend half shipped in the commit that adds this section. What the code forced or decided beyond the text above:

* **Verified facts.** Table names in the Decision were right (`vendor_issues`, `vendor_payments`). The one wrong assumption was that the column existed: only `projects`
  had `updated_at`. No sqlc query files exist (`sqlc generate` exits with "no queries contained in paths" before and after), so nothing was regenerated.
* **Version round trip.** The DSN already has `parseTime=true&loc=Local`. The driver sends a `time.Time` parameter with its microseconds, so `WHERE updated_at = ?`
  matches exactly (twenty chained `If-Match` edits in a test prove it). The value is an opaque token: it equals the true UTC instant when the Go process' local zone
  and MySQL's session zone agree (the normal deployment); when they differ the token is shifted, but equality, which is all that matters, still holds. A zone with
  daylight saving could in theory make the one ambiguous wall-clock hour of the year compare ambiguously; the app runs in zones without it.
  `internal/shared/etag` (purely technical: a time and a header) formats `updatedAt` as `2006-01-02T15:04:05.000000Z` and parses `If-Match` (quotes optional; `W/…`,
  `*`, lists and anything that is not one RFC 3339 timestamp are malformed → 400).
* **ETag is set by `response.OK`/`Created`** for any item that implements `response.Versioned` (all in-scope item DTOs do, and `withExtras` delegates to its item), so
  `GET /projects/{id}`, create answers and PATCH answers all carry `ETag: "<updatedAt>"`. List items carry `updatedAt` only.
* **Only the fields that changed are written.** The service applies the patch onto the loaded row, records which fields really changed, and the repository writes exactly
  those columns (`UPDATE … SET <changed columns> WHERE id = ? AND tenant/project scope [AND updated_at = ?]`). Two people who edit *different* fields of one item without
  `If-Match` therefore both land (a test proves it); with `If-Match` the second is a 409 by design.
* **After a write the row is read again**, so the answer (and the `ETag`) carry the version MySQL just stamped, together with whatever else changed under it.
  A write that matched no row is told apart by that read: the row is gone (`404`, unchanged), still at the version sent (MySQL had nothing to change: success), or at
  another version (`409`). With `If-Match`, a change that would be judged on a row that has already moved past the version sent is answered `409` *before* any rule runs
  (so the caller never gets a 422 about data it has not seen).
* **A patch that changes nothing** (every sent value equals the stored one) writes nothing, logs nothing and answers `200` with the item as it is, **even when its
  `If-Match` is stale**: nothing is written, so nothing can be lost.
* **Payments** run under the project lock (`SELECT … FOR UPDATE` on the project row, like create/delete), so two edits cannot overpay together; the overpayment rule is
  judged on the merged `type`/`amount` and only when one of them changes (a note can still be added to an old overpaid row). A client payment's verification is cleared
  in the same write when what was paid changes, and the **one** activity entry says so ("…: nominal. Verifikasi pembayaran dibatalkan karena data pembayaran diubah").
* **Activity:** exactly one entry per effective PATCH, e.g. `Informasi project diperbarui: nama, tanggal acara`; none for a no-op. Labels are Indonesian field names.
* **Project PATCH specifics.** `status` is not patchable (still `transition`/`cancel`). `isArchived` needs `project.archive`, every other field needs `project.edit`; a
  request carrying both needs both (the route lets in a caller holding either, `updateProject` judges each field). PIC changes still need `project.assign_pic`. Attaching a
  venue (`venueId` positive) without prices in the same body snapshots the venue's current prices and name (as create does); `venueId` null/0 detaches and clears the
  snapshot; sending a non-null price for a project with no venue is a 422 on `venueId`. Un-archiving still counts against the trial cap.
* **Directory resources.** `PATCH /vendors|venues|vendor-categories|clients|staff/{id}` with a body that is `{"isActive": true|false}` **alone** is the state change; any
  other body is the unchanged ordinary edit; `isActive` mixed with other fields, or not a boolean, is a 422 on `isActive`. The state change reads the current state, writes
  nothing when it already holds, and otherwise calls the same `SetActive` as the toggle (client: login switched off and trial quota on activation; staff: sessions revoked,
  the Owner cannot be deactivated, trial quota on activation). The old `POST …/toggle-active`, `…/toggle-archive` and `…/toggle-client-visible` routes were removed on
  2026-11-10 (see section 3); the handlers' `setActive` takes the explicit target and is the only path left.
* **Evidence.** `PATCH /projects/{id}/evidence/{eid}` `{"isClientVisible": bool}` needs `project.edit` only (a `finance.record`-only role is refused, as for the toggle). It
  also honours `If-Match`. Only `general` documents and timeline attachments carry the flag (422 `relatedKind` otherwise, as before).
* **Blank text.** On PATCH a required text field (name, brideName, groomName, packageName, scope, title) cannot be blank (422 on that key) and enum/format values are
  validated (milestone/issue/engagement status, impact, pricing tier, payment type, `HH:MM`): create never checked these and a bad value used to surface as a database error.
* **Not done / left for later:** (`If-Match` was optional here; Tahap 9 made it required, see the last section); `docs/API_CONTRACT.md` was updated in Tahap 9; platform-console toggles are untouched
  (the old toggle routes themselves were removed on 2026-11-10). Any write to a row bumps its version, so a *non-PATCH* change
  (reordering milestones, cancelling or deleting an engagement, verifying a client payment, a status transition on the project) also makes a held `ETag` stale, which is
  intended.

## Contract for the web (backend half, verified by `apps/api/tests/isolation/scenarios_tahap8_test.go`)
Common to every PATCH below:
* Body is a JSON object; only keys present change. `null` clears a field **only where the table says "nullable"**; `null` on any other key is `422` with `errors.<key>: ["Wajib diisi"]`.
  A body with no recognised key (`{}`, or only unknown keys) is `422` with `errors.body`. A body that is not a JSON object or not valid JSON is `400`.
  A wrongly typed or badly formatted value is `422` on its own key (dates `YYYY-MM-DD`, message `Gunakan format YYYY-MM-DD`). Several problems come back together.
* Header `If-Match: "<updatedAt>"` (quotes optional, value exactly as returned in `updatedAt`). **Required for field edits since Tahap 9**: absent → `428` (see the last section; the
  archive-only, evidence-visibility and `isActive` calls are exempt). Malformed → `400` with `errors["If-Match"]`.
* Success `200`: `{ success, message, data: <the item, as the list shows it, with its new updatedAt>, … }` and header `ETag: "<updatedAt>"`. Most answers go through the existing
  mutation envelope, so `data` also carries `projectProgress` (and `ledgerSummary` on payments). The project PATCH answer is the project itself (with `progress` and
  `picName`, as `GET /projects/{id}`) and has no `projectProgress` key. A repeated state or a no-op patch is a `200` with the unchanged item and unchanged `updatedAt`.
* Stale `If-Match` → `409`, no `ETag` header:
  ```json
  { "success": false,
    "message": "Data ini sudah diubah orang lain. Muat ulang untuk melihat versi terbaru.",
    "errors": { "_conflict": ["stale"] },
    "data": { "current": { /* the item as it is now, same shape as the 200 answer's item, incl. updatedAt */ } } }
  ```
  `404` stays `404` (row gone, soft-deleted, or another workspace's: indistinguishable), with or without `If-Match`.
* Every item and every list item in scope has `updatedAt` (RFC 3339, six fractional digits, UTC, e.g. `2026-11-10T08:15:30.123456Z`): projects (list and detail), milestones, project vendors
  (and their nested milestones), vendor milestones, kendala, client/vendor/venue payments, evidence.

| Resource | Path (`/api/v1` prefix) | Needs | Fields accepted (JSON name: type) | Nullable (null clears) |
|---|---|---|---|---|
| Project | `PATCH /projects/{id}` | `project.edit` for details, `project.archive` for `isArchived` | `name`, `brideName`, `groomName`, `packageName`: string, not blank · `eventDate`, `prepStartDate`: date · `venue`: string · `contractValue`: int 0..1e15 · `picStaffId`: int (0 = none) · `picSalesStaffId`: int · `description`: string · `venueId`: int · `venueRentalPrice`, `venueCharge`: int 0..1e15 · `isArchived`: bool | `venue`, `description` (→ `""`), `picSalesStaffId` (null or 0 = none), `venueId` (null or 0 = detach, clears both prices), `venueRentalPrice`, `venueCharge` |
| Project milestone | `PATCH /projects/{id}/milestones/{mid}` | `project.edit` | `status`: `Not Started`\|`In Progress`\|`Completed`\|`Blocked`\|`Cancelled` · `targetDate`: date · `completedDate`: date | `completedDate` (null or `""`) |
| Project vendor | `PATCH /projects/{id}/vendors/{pvid}` | `project.edit` | `vendorId`, `categoryId`: int (this workspace's) · `scope`: string, not blank · `contractValue`, `dpAmount`: int 0..1e15 · `pricingTier`: `Akad`\|`AkadResepsi`\|`Resepsi` · `engagementStatus`: `Planned`\|`Negotiation`\|`Booked`\|`DP Paid`\|`In Progress`\|`Fully Paid`\|`Ready`\|`Completed`\|`Cancelled` · `bookingDate`, `eventDate`, `dueDate`: date · `eventStartTime`, `eventEndTime`: `HH:MM` · `picStaffId`: int · `notes`: string | `bookingDate`, `dueDate`, `eventStartTime`, `eventEndTime` (null or `""`), `notes` (→ `""`). Answer: the engagement as `GET …/vendors` lists it (names, milestones, paidAmount) |
| Vendor milestone | `PATCH /projects/{id}/vendors/{pvid}/milestones/{mid}` | `project.edit` | `status` (as above) · `targetDate`: date · `completedDate`: date · `picStaffId`: int · `description`, `notes`: string | `completedDate`, `description`, `notes` (→ `""`) |
| Kendala | `PATCH /projects/{id}/issues/{iid}` | `project.edit` | `projectVendorId`: int · `vendorMilestoneId`: int · `title`: string, not blank · `description`: string · `impact`: `Low`\|`Medium`\|`High`\|`Critical` · `foundDate`: date · `status`: `Open`\|`In Review`\|`In Resolution`\|`Resolved`\|`Closed` · `resolutionPlan`: string · `picStaffId`: int · `targetResolutionDate`: date · `isClientVisible`: bool | `projectVendorId` (null or 0 = about the event in general), `vendorMilestoneId`, `resolutionPlan` (→ `""`), `targetResolutionDate`. Merged rule: the milestone must belong to the (merged) vendor (422 `vendorMilestoneId`). Status → Resolved/Closed stamps `resolvedDate` once |
| Client payment | `PATCH /projects/{id}/client-payments/{pid}` | `finance.record` | `type`: `DP`\|`Termin`\|`Pelunasan`\|`Tambahan`\|`Refund` · `amount`: int 1..1e15 · `paymentDate`: date · `method`, `referenceNumber`: string · `notes`: string | `notes` (→ `""`). Merged overpayment rule → 422 `amount`, code `payment_exceeds_contract`. Changing type/amount/paymentDate/method/referenceNumber clears a verification |
| Vendor payment | `PATCH /projects/{id}/payments/{pid}` | `finance.record` | same as client payment (`projectVendorId` is ignored: not patchable) | `notes` |
| Venue payment | `PATCH /projects/{id}/venue-payments/{pid}` | `finance.record` | same as client payment | `notes` |
| Evidence visibility | `PATCH /projects/{id}/evidence/{eid}` | `project.edit` | `isClientVisible`: bool (only `general` documents and timeline attachments; else 422 `relatedKind`) | none |
| Vendor | `PATCH /vendors/{id}` | `vendor.manage` | `{ "isActive": bool }` alone | none |
| Venue | `PATCH /venues/{id}` | `vendor.manage` | `{ "isActive": bool }` alone | none |
| Vendor category | `PATCH /vendor-categories/{id}` | `vendor.category.manage` | `{ "isActive": bool }` alone | none |
| Client | `PATCH /clients/{id}` | `client.manage` with every-project scope (`requireClientManager`) | `{ "isActive": bool }` alone | none |
| Staff | `PATCH /staff/{id}` | team manager (and the target's role must not exceed the caller's) | `{ "isActive": bool }` alone | none |

The five directory routes keep their ordinary edit on the same `PATCH` (a body without `isActive`); they have **no `If-Match`** (not in scope), and their answers are the
item as the old toggle answered. `isActive` not a boolean, or sent together with other keys, is `422` on `isActive`. Removed on 2026-11-10 (they answer `404`
`Endpoint tidak ditemukan`; `scenarios_tahap8_test.go` checks each of the seven): `POST /projects/{id}/toggle-archive`, `POST /projects/{id}/evidence/{eid}/toggle-client-visible`,
`POST /vendors|venues|vendor-categories|clients|staff/{id}/toggle-active`.

## Tahap 9: `If-Match` is now required for field edits (428)
Accepted — 2026-11-10 (docs/plan/revamp-ui-shadcn/PLAN.md Tahap 9, backend hardening). This settles the open point in section 2 ("Tahap 9 decides whether to make it
required"): every caller (the web) now sends the version it saw, so the transitional last-write-wins is closed.

**Rule.** A `PATCH` that edits fields of a **project, project milestone, project vendor, vendor milestone, kendala, client payment, vendor payment or venue payment**
must carry `If-Match`. Without it: `428 Precondition Required`, nothing written, nothing logged, no `ETag`:
`{ success: false, message: "Sertakan versi data (If-Match) agar perubahan tidak menimpa orang lain. Muat ulang data lalu coba lagi.", errors: { "If-Match": ["wajib"] } }`.
A malformed value is still `400`, a stale one still `409` with `data.current`, a valid current one is the compare-and-set of section 2.

**Exempt (header stays optional; they are idempotent explicit-state calls, so a retry or a double click cannot overwrite anyone's data):**
* a project `PATCH` whose body is **exactly** `{ "isArchived": bool }`. `isArchived` mixed with any other key (a field, or even an unknown key) is a field edit and needs the header;
* the evidence `PATCH` `{ "isClientVisible": bool }` (it still honours the header when it is sent, 409 on a stale one);
* the five directory `{ "isActive": bool }` PATCHes (vendor, venue, vendor category, client, staff). Their ordinary edit (a body without `isActive`) was never in scope and is unchanged;
* the collection `PATCH /projects/{id}/milestones` (reorder, body `orderedIds`) is not an item edit and is unchanged.

**Order of answers** (so the 428 never tells a caller more than a 404 would):
1. Auth, the workspace gate on the project (a project that is not this workspace's is `404`) and the route permission run first, as before: they decide who may talk about the row at all.
2. A malformed `If-Match` is `400`.
3. Header present: it is used (compare-and-set), the body is judged as before.
4. Header absent and required: the item is looked up first. A row that does not exist, is soft-deleted, or belongs to another workspace/project is the **same `404` it is with a
   header**; only a row the caller may see gets the `428`. The body is not read, validated or permission-checked in detail before this (an empty or invalid body without a header
   is a `428`, not a `422`), so a missing header can never be used to probe validation or per-field permission rules. The project PATCH has to read the body to know whether it is the
   archive-only call, but it still answers a missing header first: an unparseable body without a header is a `428`, with a header a `400`; nothing is validated before the 428.

**Where it lives.** One function, `readIfMatch` in `internal/modules/projects/presentation/patch.go`, owns the policy (`requireIfMatch` for field edits, `ifMatch` for the
optional ones, `requireIfMatch`'s existence probe is a one-line closure per handler over the owning service's `Get`/`Exists`/`MilestoneExists`). Handlers carry no
"required" logic of their own, and the services still receive a plain optional `expected`, so the application layer is unchanged.

**Consequences.** Any client that edits these fields without first reading `updatedAt` now gets 428 (the web already sends it; `docs/API_CONTRACT.md` was updated in the same Tahap, and the Postman collection
only covers the external payment gateway, which is not affected). The old "no header = last write wins" test became the 428 test; two writers who both send the version they saw conflict by design
(second one `409`), they no longer both land silently.
