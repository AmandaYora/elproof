# ADR-0018: Hard delete for Vendor, Venue, Pengguna — informed consent, not block-then-cascade

## Status
Accepted — implemented.

## Context

`DATABASE_GUIDE.md`'s "Conventions" section names an explicit house rule: soft state
(`is_active`/status-toggle) over hard delete, with exactly three sanctioned exceptions at the time
(`clients.Delete`, `projects.Delete` — ADR-0013 — and `MilestoneTemplateService.Delete`), each
narrowly scoped and guarded. The user asked for a fourth: permanent delete for Vendor, Venue,
Pengguna (staff), and Kategori Vendor — none of which had anything beyond `SetActive` before this.

This is the deliberate, explicit exception the house rule itself anticipated by name ("don't
introduce hard deletes the frontend never asks for" — this one was asked for), recorded here the
same way ADR-0013 was for `projects.Delete`.

## Decision

### Guardrail model: inform, don't block — deliberately different from ADR-0013
ADR-0013's `projects.Delete` *blocks* via a precondition (must already be archived/cancelled).
For this ADR, the user explicitly chose the opposite: no "must be inactive first" gate. An active
row can be hard-deleted directly, as long as the person deleting it has first been shown,
concretely and by name, what will lose its data source as a result. The safety mechanism shifts
from *server-enforced blocking* to *server-provided information, user-driven decision* — the
server's job is making sure the confirmation dialog can never lie by omission, not refusing the
action outright.

**New read-only `GET .../{id}/delete-impact` endpoints**, called by the frontend the instant
"Hapus" is clicked, before any dialog renders (never gating the delete itself):
- `GET /api/v1/vendors/{id}/delete-impact` / `GET /api/v1/venues/{id}/delete-impact` →
  `{ "affectedProjects": [{"id","name"}, ...] }`, sourced from `project_vendors.vendor_id` /
  `projects.venue_id`.
- `GET /api/v1/staff/{id}/delete-impact` → same shape, deduplicated across the four **live
  PIC-assignment** columns only (`projects.pic_staff_id`, `project_vendors.pic_staff_id`,
  `vendor_milestones.pic_staff_id`, `vendor_issues.pic_staff_id`). The two **audit-trail** columns
  (`activity_log.actor_staff_id`, `evidence.uploaded_by_staff_id`) are deliberately excluded —
  history must stay immutable, and someone who merely uploaded one file years ago shouldn't read
  as "still in use" for a role they no longer hold.

Backed by `projects.Contracts()`'s `ListAffectedProjectsForVendor/Venue/Staff` (same-module
queries inside `projects`, exposed through the interface `vendors` already depended on — zero new
wiring there — and bridged into `staff` via the same two-phase `SetProjectReferenceLookup` setter
idiom `MODULE_MAP.md` documents for every other cross-module callback).

The actual `DELETE` call carries no confirmation flag and performs no reference check — informed
consent already happened client-side by the time it fires. This keeps the delete endpoints
themselves simple (Owner check, then delete) and keeps 100% of "what does this affect" logic in
one place, not duplicated into a blocking check too.

### Owner-only, uniformly — stricter than the existing SetActive gate
Every one of these four entities' `SetActive` already allows Owner+Admin (Vendor/Venue) or is
already Owner-only (Staff, Vendor Category). Delete is irreversible, so it gets the strictest gate
uniformly: Owner only, everywhere, via the existing `requireOwnerRole` helper.

### The one absolute exception with no dialog at all: a tenant's Owner account
The tenant's Owner-role staff account can never be hard-deleted, full stop — not a
confirm-and-proceed dialog, a hard 403 regardless of what the impact lookup would show. This used
to fall out as a side effect of the old "must be inactive first" precondition (an Owner can never
be deactivated); with that precondition gone, `StaffService.Delete` checks it explicitly. A tenant
losing its only Owner is a structural break (no one left who can perform Owner-only actions),
categorically different from "a project's PIC label goes stale."

### Vendor Category is a deliberate carve-out — stays fully blocked, no dialog
Unlike Vendor/Venue/Staff, `vendors.category_id → vendor_categories.id` is a real, same-module SQL
foreign key on a `NOT NULL` column — there is no "orphaned but readable via stale ID" state a
vendor row can be left in the way a project's stale `vendor_id`/`venue_id`/`pic_staff_id` can; the
vendor row itself cannot exist without a category. Deleting a category still in use is kept as a
hard block (`VendorRepository.ExistsByCategoryID` → `apperror.Conflict`, "Kategori masih digunakan
oleh {N} vendor dan tidak dapat dihapus. Ubah kategori pada vendor tersebut terlebih dahulu.") —
an inline error, not a dialog, since there is no "proceed anyway" option that wouldn't either
violate the FK or require a separate out-of-scope decision (making `category_id` nullable and
defining what an "uncategorized" vendor means everywhere it's displayed/filtered/imported). The
other three entities get informed-consent because their references are already soft,
nullable-in-spirit cross-module IDs; this one is a hard schema constraint on a non-optional column.

### No cross-module cleanup, no cascade
A confirmed delete removes exactly the one row (plus, best-effort and non-blocking, its
object-storage attachment for Vendor/Venue — same log-and-continue pattern as
`EvidenceService.DeleteStorageObjects`, ADR-0013). Nothing that referenced it is touched, updated,
or cleaned up — the whole point of the dialog is that the user is choosing to accept those
references going stale, same "tidak dikenal" degrade-gracefully fallback the frontend already had
in half a dozen places for a deactivated (not deleted) vendor.

### A required fix this ADR made reachable: graceful venue-not-found
Before this ADR, `ProjectService.GetVenue` propagated a missing `venue_id` as a real `NotFound`
error, harmless only because a venue could never actually disappear (soft-state only). Once venue
hard-delete became a real, reachable path, that same `NotFound` had to be treated the same as
"no venue attached" (`p.VenueID == nil`) instead of a hard error — otherwise Project Detail's
Venue tab and Client Portal's venue view would break for any project whose venue was deleted.

## Consequences
- Routes: `DELETE /api/v1/vendors/{id}`, `DELETE /api/v1/venues/{id}`, `DELETE
  /api/v1/vendor-categories/{id}`, `DELETE /api/v1/staff/{id}`, plus the three `GET
  .../delete-impact` reads above (Vendor Category needs none — it's block-only).
- `DATABASE_GUIDE.md`'s "Conventions" section names four exceptions now, not three; this ADR is
  the fourth.
- Frontend: "Hapus" is available on every row (active or inactive) for an Owner — fetches impact,
  renders dialog copy dynamically from the actual affected project names (or the plain
  "tidak dapat dikembalikan" copy when `affectedProjects` is empty), same local-state
  confirm-dialog convention `ProjectHeaderCard.tsx` (ADR-0013) already established, not extracted
  into a shared component (matches this codebase's existing per-page-dialog convention).
- No migration needed — every affected table already stored its cross-module references as
  primitive, un-enforced IDs (per the modular-monolith rule), which is exactly what makes
  informed-consent delete safe to ship without a schema change.
