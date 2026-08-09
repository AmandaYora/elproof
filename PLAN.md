# PLAN — Export to Excel, hard delete (Vendor/Venue/Pengguna/Kategori Vendor), city filter

## 0. Goal

Three additions to the `vendors` (Vendor, Venue, Vendor Category) and `staff` (Pengguna) areas:

1. **Export to Excel**, next to the existing "Import Bulk" flow, for Vendor and Venue.
2. **Hard delete** (permanent, not the existing deactivate) for Vendor, Venue, Pengguna, and Kategori
   Vendor — gated by an **informed-consent confirmation dialog** that names exactly what will lose
   its data source, not by a "must be inactive first" precondition.
3. **Filter by Kota (city)** on the Vendor and Venue list pages.
4. **Rename "Tanggal Mulai Persiapan" to "Tanggal Booking"** on the Project form/detail — same date
   field, new label and framing, not a new field.
5. **Restructure Client Portal's project view** to match the WO Console's `ProjectHeaderCard` layout
   (persistent header: name/status, info grid, progress bar) — read-only, internal-only fields
   dropped, folding the current standalone "Ringkasan" tab into that persistent header plus a new
   "Timeline" tab.

## 1. Current state, confirmed precisely

### Import/Template (the pattern Export must sit next to)
`apps/api/internal/modules/vendors/presentation/vendor_handler.go` and `venue_handler.go` already
implement **Template** (blank `.xlsx` header download) and **Import** (base64-JSON upload, parsed
with `github.com/xuri/excelize/v2`, capped at 1000 rows, batched inserts of 200) — see
`vendorTemplateHeaders`/`venueTemplateHeaders` (fixed column order), `ImportVendors`/`ImportVenues`
in `application/vendor_service.go` / `venue_service.go`. **No export capability exists anywhere in
the backend today** — confirmed via a full-repo grep for `excelize|xlsx|Export`; `excelize` is
already a dependency, so Export needs no new library.

Frontend: a single domain-agnostic `apps/web/src/shared/components/ui/ImportBulkModal.tsx` powers
both pages' "Import Bulk" button (`VendorListPage.tsx`, `VenueListPage.tsx`). Template download is a
plain `httpClient.get(url, { responseType: "blob" })` → `URL.createObjectURL` → synthetic
`<a download>` click, done inline inside that modal — the exact mechanic an Export button reuses.

### City — already a real column, not a new field
Both `Vendor` (`domain/vendor.go:14`) and `Venue` (`domain/venue.go:20`) already have `City *string`,
validated against a fixed 128-entry enum: `domain/city.go`'s `AllowedCities`/`IsValidCity`, mirrored
on the frontend at `shared/constants/cities.ts`. Import already **requires** a valid city per row.
Columns exist since `000023_alter_vendors_add_fields.up.sql` (vendors) and from creation
(`000020_create_venue_tables.up.sql`, venues) — **no migration needed for the field**, only
(optionally) an index. Neither table currently has an index on `city`.

List/filter today: `VendorHandler.Collection` / `VenueHandler.Collection` already thread a
`categoryId` (vendor only) and free-text `search` through to `ListPaginated` in
`mysql_vendor_repository.go` / `mysql_venue_repository.go`, which append `AND category_id = ?` /
`AND (name LIKE ? OR pic_name LIKE ? OR email LIKE ?)` conditionally. A `city` filter is a mechanical
repeat of this exact pattern.

### Delete — today only `SetActive` exists; the house rule this plan deliberately overrides
Vendor, Venue, and Vendor Category each have only a `SetActive` (deactivate/reactivate) service +
repo method — **no `Delete` anywhere in any of the three**. Staff's repository does have a `Delete`
(`mysql_staff_repository.go:138`), but it is explicitly documented and only ever called as an
internal rollback when credential creation fails mid-`Create` — never reachable via HTTP.

This repo has an explicit, named house rule against exactly what's being asked for —
`knowledge/DATABASE_GUIDE.md` "Conventions": *"Soft state over hard delete... vendors, venues,
clients, staff, plans all 'deactivate' instead of delete... don't introduce hard deletes the frontend
never asks for."* It names exactly three sanctioned exceptions (`clients.Delete`, `projects.Delete`
— ADR-0013 —, `MilestoneTemplateService.Delete`) and **explicitly calls out this plan's own risk
scenario by name**: *"deleting a `vendors`/`venues` row referenced by `project_vendors`/
`projects.venue_id` would [orphan it]."* This plan is the deliberate, explicit fourth exception the
rule anticipates (recorded here, same as ADR-0013 was) — not a silent deviation.

Unlike ADR-0013 (which *blocks* the delete via a precondition), the user has explicitly chosen a
different guardrail here: **inform, don't block** — show the person deleting exactly what will be
left sourceless, by name, and let them decide. §2 designs that.

**What would actually break if a row just vanished** (cross-module, all primitive-ID references, no
SQL FK — per the modular-monolith rule):
- `project_vendors.vendor_id` (no FK) — a deleted vendor would silently disappear from
  `vendors/summary` (already deliberately includes inactive vendors so engaged-vendor names don't
  vanish); the frontend already degrades this gracefully in half a dozen places (`?? "Vendor tidak
  dikenal"` etc. in `ProjectVendorsSection.tsx`, `ProjectEvidenceSection.tsx`,
  `VendorProgressTabPage.tsx`, `IssueFormModal.tsx`, `ProjectVendorFormModal.tsx`,
  `ProjectPaymentsSection.tsx`) — this existing fallback UI is exactly what will render after a
  confirmed delete goes through; it's the reason a "confirm and proceed" design (rather than a hard
  block) is safe to ship at all.
- `projects.venue_id` (no FK) **is** resolved by a real backend contract call —
  `vendors.Contracts.GetVenueSummary` → `VenueService.Get`, consumed by
  `ProjectService.GetVenue` (`project_service.go:126-139`). Today a missing venue there propagates as
  a real `NotFound` error, not a graceful "no venue attached" — **this is a required fix, not
  optional**, since after this plan ships, a confirmed venue delete is a real, reachable path that
  will hit this exact code (see §2/§3).
- `pic_staff_id` — six distinct reference points, four are **live "responsible person" assignments**
  (`projects.pic_staff_id`, `project_vendors.pic_staff_id`, `vendor_milestones.pic_staff_id`,
  `vendor_issues.pic_staff_id`) and two are **immutable historical audit trail**
  (`activity_log.actor_staff_id`, `evidence.uploaded_by_staff_id`) — see §2 for why only the first
  four inform the confirmation dialog.
- `vendors.category_id → vendor_categories.id` **does** have a real SQL FK
  (`000006_create_vendor_tables.up.sql:29`, same-module, RESTRICT by default, column is `NOT NULL`).
  This is structurally different from the three above — see §2's explicit carve-out for Vendor
  Category.

### "Mulai Persiapan" → "Tanggal Booking" — this is a label change on a field with real behavior behind it
`prepStartDate`/`PrepStartDate`/`prep_start_date` is not decorative — it's a `NOT NULL DATE` column
(`000008_create_project_tables.up.sql:9`) feeding two real calculations, both in
`apps/api/internal/modules/projects/domain/dashboard.go` and `application/default_milestones.go`:
- **`IsLagging`** (`dashboard.go:99-114`) — a project's "Progress Tertinggal" dashboard flag compares
  `elapsed / total` days **since `prepStartDate`** against actual Timeline completion %.
- **`seedDefaultMilestones`** (`default_milestones.go:17-26`) — a new project's auto-seeded Timeline
  items are clamped so no default due date falls before `prepStartDate`.

There is already a **direct precedent in this exact codebase** for "the UI should call this something
different than what the code models it as," recorded in `knowledge/DOMAIN_GLOSSARY.md`'s **Timeline**
and **Vendor Milestone** entries: *"display label only; the underlying identifier, DB table, and API
shape are still 'milestone' everywhere (deliberate, display-text-only rename)."* That precedent is
followed here rather than reinvented — see §2.

**Reusable wiring already in place** (no new cross-module dependency needs to be invented):
- `vendors.NewModule(db, projectsModule.Contracts(), storageClient)` — `vendors` **already** holds a
  reference to `projects.Contracts()` at construction time (`cmd/server/main.go`). Adding new methods
  to that interface for the Vendor/Venue impact lookups needs **zero new wiring** — no bridge, no
  two-phase setter.
- `staff.NewModule(db, identityModule.Contracts())` — `staff` is built **before** `projectsModule` in
  `main.go`, so a Staff→`projects.Contracts()` impact lookup needs the same two-phase
  setter-after-construction bridge already used twice (`ClientCleaner`, `VenueResolver`): a new
  `StaffService.SetProjectReferenceLookup(lookup)`, called from `main.go` right after
  `projectsModule` exists.
- `ProjectHeaderCard.tsx` (ADR-0013's frontend half) is the closest existing confirm-then-delete UX,
  though it confirms a *known, static* consequence ("this project and everything in it"). This
  plan's dialogs need one more step first — fetching the *specific* impact (which projects, by name)
  before the confirmation text can be written — so the pattern is extended, not copied verbatim (see
  §2's frontend flow). No shared `ConfirmDialog` component exists in this codebase — inline local
  state per page remains the right call, not a new shared modal.
- `requireOwnerRole` already exists (`vendor_category_handler.go:62`, same `presentation` package as
  `vendor_handler.go`/`venue_handler.go`) — reusable as-is for Vendor/Venue's new Delete handlers,
  no duplicate helper needed.

### Client Portal restructure — current state
`ProjectHeaderCard.tsx` (WO Console, 283 lines) has four internal-only action buttons (Ubah/
Duplikat/Arsipkan/Batalkan/Hapus Permanen — all mutations, none applicable read-only), an info grid
(Tanggal Acara, Countdown, Mulai Persiapan, Paket/Layanan, Nilai Kontrak, Sisa Tagihan Client,
Margin/Keuntungan, Penanggung Jawab), and a progress section (`{overallPercent}%`, a condition badge,
a compact `ProgressMeter`, and a "project X/Y · vendor X/Y · N terlambat" caption) fed entirely by
`project.progress` (`ComputeProjectProgress`, `domain/progress.go:89-140` — already delivered whole
on the same `GET /projects/{id}` / `GET /projects/me` payload Client Portal already fetches; nothing
in this section needs new backend work). Margin is **already** documented and gated as sensitive
even for internal staff (`DOMAIN_GLOSSARY.md`: "shown only to Owner/Admin... hidden from Wedding
Planner") — the precedent this plan extends one step further to "hidden from client entirely."

Client Portal today: `ClientPortalLayout.tsx` has only a thin header (branding, names, a countdown
pill computed independently via the same `daysUntil` helper) above a 5-tab strip (`ringkasan`,
`vendor`, `venue`, `pembayaran`, `kendala`). The **`ringkasan` tab** (`RingkasanTabPage.tsx`) is the
closest existing analog to the header card's progress section, but computes its own milestone count
client-side (project milestones only, no vendor-milestone weighting) rather than reusing
`progress.projectMilestoneStats`/`.vendorMilestoneStats` directly, and additionally renders an
open-issues banner and a much richer vertical "Perjalanan Persiapan" milestone stepper that has no
equivalent in the WO header card at all (the WO header's `ProgressMeter` is compact; the WO
equivalent of a full milestone list is its own separate **Timeline** tab).

**No response-shape split by principal type exists anywhere in the `projects` module** — confirmed by
reading `project_endpoints.go`'s `me()` (client) handler: it calls the exact same `getProject`/
`toProjectResponse` staff's `GET /projects/{id}` uses. `VenueRentalPrice`/`VenueCharge`/`PICStaffID`
already ride along in the JSON sent to a client principal today; only the frontend chooses not to
render them. This mirrors the already-accepted Margin-hidden-from-Staff pattern (also UI-only, since
`vendorEngagements`/venue cost fields are independently visible via the Vendor/Venue tabs) — not a
new gap this plan introduces, just the same shape of caveat one layer further out.

**"Penanggung Jawab" (PIC name) is currently unreachable by a client principal**: WO Console resolves
it client-side against `GET /staff/summary`, which is hard-gated staff-only
(`requireTenant`, `staff/presentation/*.go:68-70` rejects any non-`"staff"` principal). Showing it to
Client Portal needs a small, real backend addition (see §2) — everything else in the info grid is
already present in the payload today.

**"Client" and "Aktivitas" WO Console tabs have no client-safe equivalent** and are out of scope
here: "Client" is pure credential/account management (reset password, reassign representative) —
nonsensical inside the portal that credential controls; "Aktivitas" is a raw staff-attributed system
log (actor names, every field-level edit) that would need new curation/filtering work to be
client-appropriate, which nothing today provides.

## 2. Design decisions

### Export to Excel (Vendor, Venue)
- New `GET /api/v1/vendors/export`, `GET /api/v1/venues/export`, same `requireManagerRole`
  (Owner/Admin) gate as Template/Import — same handler family, same risk level (bulk data egress).
- **Accepts the same filter params as the list endpoint** (`search`, `categoryId`, and the new `city`
  from below) and exports the matching set, unpaginated — so "filter then export" works, matching
  what a WO staff member actually wants ("export vendor kategori Catering di Bandung"), not just a
  full-roster-only dump. At this app's real data volumes (SMB wedding-organizer rosters, not
  thousands of rows) an unpaginated in-memory export is safely proportionate — no streaming/async-job
  machinery needed.
- **Column order matches the import template exactly**, with one appended `Status` column
  (Aktif/Nonaktif) at the end — `parseVendorImportRow`/`parseVenueImportRow` read fixed columns by
  index and never validate a max column count, so this file **round-trips through Import unmodified**
  (a real, useful property: export, edit in Excel, re-import).
- Implementation: a new `application.VendorService.Export(ctx, tenantID, filters) ([]domain.Vendor,
  error)` (reuses the existing filtered query path, no pagination arguments) and a presentation-layer
  `buildVendorWorkbook([]domain.Vendor) *excelize.File` sibling to the existing template-building
  code — same library, same response headers/content-type as `Template`, different filename
  (`vendors-export-{date}.xlsx`) and populated rows instead of a blank sheet.
- Frontend: one "Export Excel" button placed beside the existing "Import Bulk" button on both list
  pages (no modal — direct blob download on click, mirroring the Template download's own
  no-modal-needed simplicity), passing the page's *currently active* filters as query params.

### City filter (Vendor, Venue)
- Query param `city` (exact string match against the closed `AllowedCities` enum — not `LIKE`, since
  it's a closed set), threaded through `Collection` handler → service → `ListPaginated` repository
  method exactly like `categoryId` is today, appended as `AND city = ?` conditioned on non-empty.
- Frontend: a `cityFilter` state (`"Semua"` sentinel, same convention as the category dropdown) on
  both `VendorListPage.tsx` and `VenueListPage.tsx`. Given 128 options, use the existing `Combobox`
  (already used in `VendorFormModal.tsx:158` for the same city enum) instead of a plain `<Select>`,
  so it's searchable rather than a 128-row scroll.
- Optional, low-priority migration: composite index `(tenant_id, city)` on both `vendors` and
  `venues` — cheap to add now while touching this area, not required at current data volumes (a plain
  tenant-scoped table scan is fine today; this just avoids revisiting the schema later once it isn't).

### "Mulai Persiapan" → "Tanggal Booking" (display-text-only rename)
Mirrors the Timeline/Vendor Milestone precedent exactly (§1): **change the label everywhere a human
reads it, change nothing a machine reads it as.**
- `prepStartDate` / `PrepStartDate` / `prep_start_date` stay **completely unchanged** — column name,
  Go struct field, JSON key, Zod schema key, store field, request/response DTOs. No migration, no
  API contract change, no risk to `IsLagging`/`seedDefaultMilestones` (both keep working exactly as
  before — "days since booking" is, if anything, a more defensible progress baseline than "days since
  an operationally-fuzzy prep-start date," since a booking date is a concrete event every project
  unambiguously has).
- Three display-string edits only:
  - `apps/web/src/modules/projects/components/ProjectFormModal.tsx:136` —
    `<Field label="Tanggal Mulai Persiapan" ...>` → `<Field label="Tanggal Booking" ...>`.
  - `apps/web/src/modules/projects/schemas/project.schema.ts:11` — validation message
    `"Tanggal mulai persiapan wajib diisi"` → `"Tanggal booking wajib diisi"`.
  - `apps/web/src/modules/projects/components/detail/ProjectHeaderCard.tsx:231` —
    `<InfoField label="Mulai Persiapan" .../>` → `<InfoField label="Tanggal Booking" .../>`.
- `knowledge/DOMAIN_GLOSSARY.md`: add one line for **Tanggal Booking**, same shape as the existing
  Timeline/Vendor Milestone entries — "display label only; underlying identifier/column/API field is
  still `prepStartDate` everywhere — see PLAN.md" — so a future reader who greps the backend for
  "booking" and finds nothing isn't left confused.
- One-line doc-comment touch-up (not a behavior change) on `IsLagging` and `seedDefaultMilestones`
  noting the field now displays as "Tanggal Booking," so the "prep time"/"short-notice" phrasing in
  their existing comments isn't read as contradicting the UI.
- **Explicitly not doing**, and why: renaming the identifier itself (`booking_date` column, etc.) or
  changing what the date is validated/used for (e.g., allowing it to sit further from the event than
  today, or decoupling it from the milestone-clamp/lagging math). Nothing about the request asks for
  new date semantics — only a different name for the one that exists — so the smallest-blast-radius
  option that already has a codebase precedent is the right one. If a genuine semantic change is
  wanted later (e.g., a separate real "booking date" distinct from "prep start," tracked
  independently), that's a new field and a bigger, separate decision — flag it explicitly rather than
  smuggling it into a rename.

### Client Portal restructure — confirmed with the user before designing further
1. **The standalone "Ringkasan" tab is absorbed into a persistent header**, not kept alongside an
   enriched header (the user's explicit choice over the lower-risk "keep both, accept duplication"
   option) — this is a real structural change, not just new header content:
   - New `apps/web/src/modules/client-portal/components/ClientProjectHeaderCard.tsx` — read-only
     sibling of `ProjectHeaderCard.tsx`, rendered once inside `ClientPortalLayout.tsx` above `TabNav`
     (replacing that layout's current thin branding/countdown header), so it stays visible across
     every tab exactly like the WO Console screenshot.
   - Contains: branding (existing), title + `ProjectStatusBadge` (new for Client Portal, reused
     verbatim from `projects`'s components — cross-module frontend imports between `client-portal`
     and `projects` are already an established pattern, e.g. `IssueCard`/`EngagementStatusBadge`),
     info grid (below), and the progress section reusing `project.progress` directly —
     `overallPercent`, a condition badge, and the "project X/Y · vendor X/Y · N terlambat" caption —
     **keeping Client Portal's existing softer `CLIENT_CONDITION_COPY` wording** ("Berjalan sesuai
     rencana"/"Ada beberapa hal yang sedang kami tangani"/"Membutuhkan perhatian lebih dari tim
     kami"), not WO Console's blunter "Perlu Perhatian"/"Berisiko" labels — that softer copy was
     already a deliberate client-facing design choice and this restructure doesn't reverse it.
   - The open-issues banner (currently `RingkasanTabPage.tsx` lines 55-72) **moves into this
     persistent header too** — strictly more visible than being one tab away, which is an
     improvement, not a regression, and needs no new data (same `kendala` count already fetched).
   - The richer vertical "Perjalanan Persiapan" milestone stepper — which has no equivalent in the WO
     header card at all (WO's own compact meter is the header's version; WO's *detailed* list lives
     in its own separate **Timeline** tab) — **becomes a new "Timeline" tab** in Client Portal
     (`apps/web/src/modules/client-portal/pages/tabs/TimelineTabPage.tsx`, replacing
     `RingkasanTabPage.tsx`), completing the tab-parity with WO Console's own Vendor/Venue/Timeline/
     Pembayaran set (minus Client/Aktivitas/Dokumen, per §1's "no client-safe equivalent" finding).
   - Resulting tab strip: **Vendor, Venue, Timeline, Pembayaran, Kendala** (drops "Ringkasan" as a
     name entirely). `client-portal.routes.tsx`, `ClientPortalTab` (`route-paths.ts`), and the
     index-route redirect all need updating — land on `vendor` (matching the reference screenshot's
     default-active tab), not the now-removed `ringkasan`.
2. **Margin/Keuntungan, Mulai Persiapan, and all five action buttons are dropped, not just
   unrendered-but-present** — the new `ClientProjectHeaderCard.tsx` never receives or reads
   `venueRentalPrice`/`venueCharge`/`vendorEngagements` cost fields at all (unlike
   `ProjectHeaderCard.tsx`, which fetches them specifically to compute Margin), so there's no
   client-side margin math to accidentally leak in this component even though the raw fields still
   ride along elsewhere in the shared `projectResponse` payload (§1's existing caveat, unchanged by
   this plan). "Mulai Persiapan" (now "Tanggal Booking," §2 above) is simply not one of the six info
   grid fields shown to a client — confirmed as the user's choice, not an oversight.
3. **"Penanggung Jawab" is shown, resolved server-side** (the user's explicit choice over hiding it,
   since it's a real new backend addition, not free): `staff` is already constructed *before*
   `projects` in `cmd/server/main.go`, so `projects.NewModule` can take a narrow new interface —
   `type StaffNameResolver interface { GetSummary(ctx, tenantID, staffID int64) (*Summary, error) }`
   (same narrow-local-interface convention as `VenueResolver`/`ClientCleaner`) — as a **direct
   constructor argument**, no two-phase setter bridge needed (no circular build order exists here,
   unlike those two). `projectResponse` (`dto.go`) gains a `picName string` field, populated once
   server-side in `toProjectResponse`/`getProject` for **both** principal types (simplest option —
   no principal-type branching needed inside the DTO builder; staff already gets the same name via
   `/staff/summary` client-side today, so this is redundant-but-harmless for them, not a new
   exposure). Client Portal's header simply reads `project.picName` — no new client-side fetch, no
   dependency on the staff-only summary endpoint at all.
4. **`project.description` is shown, confirmed with the user.** The screenshot's "CLIENT CONCERN
   MAKANAN" banner turned out under investigation to be `project.description` (a free-text field WO
   staff fill in) rendered inline, not a computed concern indicator — `ProjectHeaderCard.tsx` has no
   dedicated "client-concern" mechanism today; it's just this one field, shown verbatim. Confirmed:
   this field is written with the client as an intended reader, so `ClientProjectHeaderCard.tsx`
   renders it the same way `ProjectHeaderCard.tsx` does — a plain note below the info grid, shown only
   when non-empty (no special "concern" styling/label beyond what WO Console itself already uses,
   since it's the same field, not a distinct feature).

### Hard delete (Vendor, Venue, Pengguna, Kategori Vendor)
Explicit exception to `DATABASE_GUIDE.md`'s soft-state rule. Two axes, both server-side:

1. **Owner-only**, every entity — stricter than these entities' existing `SetActive` gate
   (Vendor/Venue today allow Owner **and** Admin to deactivate; Staff's whole module and Vendor
   Category's `SetActive` are already Owner-only). Delete is irreversible, so it gets the strictest
   gate uniformly. Reuses the existing `requireOwnerRole`/`requireOwnerTenant` helpers verbatim.
2. **Informed-consent confirmation, not a precondition.** There is no "must be inactive first" gate —
   an active row can be hard-deleted directly, as long as the person deleting it has been shown,
   concretely, what will lose its data source as a result. This shifts the safety mechanism from
   *server-enforced blocking* to *server-provided information, user-driven decision* — the server's
   job is to make sure the dialog can never lie by omission, not to refuse the action.

   **New read-only "impact" endpoints**, called by the frontend the moment "Hapus" is clicked
   (before any dialog renders), never gating the delete itself:
   - `GET /api/v1/vendors/{id}/delete-impact` → `{ "affectedProjects": [{"id","name"}, ...] }`
     (project rows where `project_vendors.vendor_id = id`).
   - `GET /api/v1/venues/{id}/delete-impact` → same shape (`projects.venue_id = id`).
   - `GET /api/v1/staff/{id}/delete-impact` → same shape, deduplicated across the four **live
     PIC-assignment** columns only: `projects.pic_staff_id`, `project_vendors.pic_staff_id`,
     `vendor_milestones.pic_staff_id`, `vendor_issues.pic_staff_id`. The two **audit-trail** columns
     (`activity_log.actor_staff_id`, `evidence.uploaded_by_staff_id`) are deliberately **excluded** —
     audit history must stay immutable and readable-by-fallback, and a staff member who merely
     uploaded one evidence file years ago must not show up as "still in use" for a role they no
     longer hold. (Follow-up, in scope: confirm activity-log/evidence author-name rendering already
     tolerates a staff ID with no matching row, same "tidak dikenal" convention vendors/venues have;
     add the fallback if missing — this is the first time a staff row can actually disappear.)

   Backed by new methods on `projects.Contracts()` — `ListAffectedProjectsForVendor`,
   `ListAffectedProjectsForVenue`, `ListAffectedProjectsForStaff` (each `(ctx, tenantID, id)
   ([]ProjectRef, error)`, `ProjectRef{ID, Name}`) — same-module queries inside `projects`' own
   repositories, exposed through the interface `vendors` **already** depends on (zero new wiring for
   Vendor/Venue) and bridged into `staff` via the two-phase `SetProjectReferenceLookup` setter (§1).

   **Frontend dialog copy** (fills in the specific names from the impact call — exact Indonesian
   wording, not a placeholder):
   - *Venue, referenced:* "Venue ini masih digunakan oleh project berikut: **{daftar nama project}**.
     Jika dihapus, data venue pada project tersebut akan kehilangan sumber datanya dan tidak dapat
     ditelusuri lagi. Apakah Anda yakin ingin menghapus venue ini secara permanen?"
   - *Vendor, referenced:* "Vendor ini masih digunakan oleh project berikut: **{daftar nama
     project}**. Jika dihapus, data vendor pada project tersebut akan kehilangan sumber datanya dan
     tidak dapat ditelusuri lagi. Apakah Anda yakin ingin menghapus vendor ini secara permanen?"
   - *Pengguna, referenced:* "Pengguna ini masih ditugaskan sebagai PIC pada project berikut:
     **{daftar nama project}**. Jika dihapus, penugasan PIC pada project tersebut akan kehilangan
     sumber datanya dan tidak dapat ditelusuri lagi. Apakah Anda yakin ingin menghapus pengguna ini
     secara permanen?"
   - *Any entity, not referenced (`affectedProjects` empty):* "Data ini akan dihapus secara permanen
     dan tidak dapat dikembalikan. Apakah Anda yakin?" — no manufactured urgency when there's
     genuinely nothing at stake.
   - Both branches end in the same Batal / **Ya, Hapus Permanen** button pair, `isDeleting` disabling
     both mid-request — same mechanics `ProjectHeaderCard.tsx` already uses.

   The actual `DELETE` call carries no confirmation flag and performs no reference check — by the
   time it fires, informed consent has already happened client-side. This keeps the delete endpoints
   themselves simple (Owner check, then delete) and puts 100% of the "what does this affect" logic in
   one place (the impact endpoint), rather than duplicating it into a blocking check too.

3. **One absolute, separate rule that is *not* about orphaning and is *not* skippable by any
   dialog**: **the tenant's Owner-role staff account can never be hard-deleted.** This used to fall
   out as a side effect of the old "must be inactive first" precondition (Owner can never be
   deactivated); now that precondition is gone, this must be its own explicit, unconditional check in
   `StaffService.Delete` — a tenant losing its only Owner account is a structural break (no one left
   who can perform Owner-only actions), categorically different from "a project's PIC label goes
   stale," so it stays a hard 403, never a confirm-and-proceed dialog.

4. **Vendor Category is a deliberate carve-out, kept fully blocked — not given a confirm-and-proceed
   dialog.** Unlike Vendor/Venue/Staff, `vendors.category_id → vendor_categories.id` is a real,
   same-module SQL foreign key on a `NOT NULL` column (§1) — there is no "orphaned but readable via
   ID" state a vendor row can be left in the way a project's stale `vendor_id` can; the vendor row
   itself cannot exist in the database without a category. Deleting a category still assigned to any
   vendor is therefore kept as a hard block: `VendorRepository.ExistsByCategoryID(ctx, tenantID,
   categoryID) (bool, error)` (same-module, no cross-module call needed) returns `apperror.Conflict`
   with **"Kategori masih digunakan oleh {N} vendor dan tidak dapat dihapus. Ubah kategori pada
   vendor tersebut terlebih dahulu."** — an inline error, not a dialog, since there is no "proceed
   anyway" option that wouldn't either violate the FK or require a separate, out-of-scope decision
   (making `category_id` nullable and defining what an "uncategorized" vendor means everywhere it's
   displayed/filtered/imported). Flagging this now so it isn't mistaken for an inconsistency: the
   other three entities get the informed-consent treatment because their references are already
   soft/nullable-in-spirit cross-module IDs; this one is a hard schema constraint on a column that
   isn't optional today.
5. **No cross-module cleanup, no cascade.** A confirmed delete removes exactly the one row (plus,
   best-effort and non-blocking, its object-storage attachment for Vendor/Venue — same
   log-and-continue pattern as `EvidenceService.DeleteStorageObjects` in ADR-0013). Nothing that
   referenced it is touched, updated, or cleaned up — the whole point of the dialog is that the user
   is choosing to accept those references going stale.
- Routes: `DELETE /api/v1/vendors/{id}`, `DELETE /api/v1/venues/{id}`, `DELETE
  /api/v1/vendor-categories/{id}`, `DELETE /api/v1/staff/{id}`, plus the four `GET .../delete-impact`
  reads above. `DELETE` is already in the dev CORS allow-list (confirmed unchanged since ADR-0013).
- Frontend: "Hapus" is available on **every** row (active or inactive) for an Owner, unlike the
  original design — clicking it fires the impact fetch, then renders the appropriate dialog copy
  above inline (same local-state convention as `ProjectHeaderCard.tsx`, replicated per page rather
  than extracted into a shared component, matching this codebase's existing convention).

### Required fix riding along with this plan: graceful venue-not-found
Because a confirmed venue delete is now a real, reachable path (no precondition blocking it),
`ProjectService.GetVenue` (`project_service.go:126-139`) must be updated so a `NotFound` from
`vendors.Contracts.GetVenueSummary` is treated the same as `p.VenueID == nil` (i.e., "no venue
attached / no longer resolvable") rather than propagated as a real error — otherwise the Project
Detail "Venue" tab and Client Portal's Venue tab would break for any project whose venue was deleted.
This was already a latent gap (§1); this plan is what makes it reachable, so fixing it is in scope,
not a nice-to-have.

## 3. Order of implementation

1. **City filter** (smallest, no new cross-module surface) — handler + repository + frontend filter
   for Vendor and Venue. `go build`/`go vet`, `tsc --noEmit`, manual check: filtering by a city with
   known seed data returns only matching rows, "Semua" clears it.
2. **Export to Excel** (depends on the filter params from step 1 being available, but builds
   independently otherwise) — service `Export` methods, presentation workbook builders + routes,
   frontend buttons. `go build`/`go vet`, `tsc --noEmit`, manual check: exported file opens correctly,
   column order matches Import's template, and the exported file re-imports cleanly with zero
   diffs against the original rows.
3. **`GetVenue` graceful-not-found fix** — small, isolated, and a prerequisite for venue delete being
   safe to ship at all; do it before any delete route exists.
4. **Hard delete — impact-lookup plumbing**: add `ListAffectedProjectsForVendor/Venue/Staff` to
   `projects.Contracts()` and their backing repository queries, wire
   `StaffService.SetProjectReferenceLookup` in `main.go`. `go build`/`go vet` (no new route exposed
   yet at this step).
5. **Hard delete — per-entity `delete-impact` + `Delete` service/repository/handler/route**, one
   entity at a time, simplest first: Vendor Category (blocked-only, same-module, no impact endpoint
   needed) → Vendor → Venue → Staff (needs the step-4 bridge, plus the standalone Owner-account
   guard). `go build`/`go vet` after each.
6. **Hard delete — frontend**, one page at a time, same order: impact fetch on click → dynamic dialog
   copy → confirm → delete call → refetch list. `tsc --noEmit` after each.
7. **Audit pass**: confirm `activity_log`/`evidence` actor/uploader name resolution tolerates a
   deleted staff ID; add the "tidak dikenal" fallback if missing.
8. **"Tanggal Booking" rename** — independent of everything above, safe to do in any order relative
   to it: the three display-string edits + glossary entry + doc-comment touch-up. `tsc --noEmit`.
9. **Client Portal restructure — backend first**: add `StaffNameResolver`, wire it as a direct
   constructor argument on `projects.NewModule` (`main.go` — `staffModule` is already built earlier,
   no reordering needed), add `picName` to `projectResponse`. `go build`/`go vet`; manual check:
   `GET /projects/me` as a client principal now includes a real `picName`, not blocked by any
   staff-only guard.
10. **Client Portal restructure — frontend**: build `ClientProjectHeaderCard.tsx`, wire it into
    `ClientPortalLayout.tsx`, extract the milestone stepper into the new `TimelineTabPage.tsx`,
    update `client-portal.routes.tsx`/`ClientPortalTab`/index redirect, delete the old
    `RingkasanTabPage.tsx`. `tsc --noEmit`.

## 4. Verification

- `go build ./...` / `go vet ./...` after every backend step; `npx tsc --noEmit` after every frontend
  step (existing repo convention).
- City filter: filter Vendor and Venue lists by a city with known rows, confirm exact-match results
  and pagination/count stay correct; confirm Export honors the same filter.
- Export: download, open in Excel, confirm column headers/order match the Import template plus the
  appended Status column; re-import the exported file unmodified and confirm zero new/updated rows
  (a true round-trip).
- Hard delete, per entity, both via the UI and a direct API call:
  - Delete action is genuinely unavailable (not just disabled) for a non-Owner.
  - Deleting a vendor/venue/staff member that **is** referenced shows the correct dialog with the
    actual affected project name(s), proceeds to a real delete on confirmation, and the referencing
    rows are confirmed to degrade gracefully afterward (existing "tidak dikenal" fallback for
    vendor/venue; the Venue tab specifically, per the §2 fix, shows "tidak ada venue" rather than an
    error; new fallback for staff if the audit step in §3.7 found a gap).
  - Deleting an unreferenced row of each type shows the plain "tidak dapat dikembalikan" dialog,
    succeeds, and is confirmed gone from the DB; Vendor/Venue's object-storage attachment is confirmed
    deleted with no error logged.
  - Attempting to hard-delete the tenant's Owner staff account is rejected outright (403), with no
    dialog offered at all, regardless of role-check bypass attempts via direct API call.
  - Vendor Category deletion while any vendor still references it is rejected with the count-based
    message above; no dialog is ever shown for this entity when in use.
- Regression: existing Import Bulk flow for Vendor/Venue still works unchanged after the service-layer
  refactor needed to share filtering logic with Export.
- "Tanggal Booking" rename: Project create/edit form and Project header both show "Tanggal Booking"
  nowhere still says "Mulai Persiapan"; dashboard's "Progress Tertinggal" flag and a newly created
  project's seeded Timeline dates are numerically unchanged before/after (same field, same value,
  same math) — confirms this really was label-only.
- Client Portal restructure: log in as a client, confirm the header (name/status/info grid/progress)
  stays visible while switching between Vendor/Venue/Timeline/Pembayaran/Kendala; confirm no Margin,
  no Mulai Persiapan, and no Ubah/Duplikat/Arsipkan/Batalkan/Hapus action ever renders anywhere in the
  portal; confirm Penanggung Jawab shows a real name (not blank/error); confirm the moved "Perjalanan
  Persiapan" stepper renders correctly under the new Timeline tab; confirm `/portal/ringkasan` no
  longer 404s ungracefully (redirects to `vendor` or an equivalent valid tab); confirm a project with
  a non-empty `description` shows it in the client header, and a project with an empty `description`
  shows no stray empty note element.
