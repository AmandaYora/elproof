# ADR-0050: Quotations — numbering, revisions by content hash, the status machine, and the PDF library

## Status
Accepted — 2026-11-20. Plan: `docs/plan/paket-wo/PLAN.md` §5 (5.1–5.3, 5.5), §6. Builds on ADR-0048 (same shape as a project) and
ADR-0049 (the `offers` module and its ports). "Jadikan proyek" (§5.4) is owned by `projects` and uses the contract described below.

## Context
A WO sends a couple a **Penawaran** before there is a project: the same events, Vendor & venue and Rincian a project has, plus payment
terms, terms & conditions and a validity date. The couple negotiates over WhatsApp, so one quotation goes through several versions; the
WO must be able to say exactly which version was sent and which one was accepted, and the PDF of a version sent last week must never
change because someone edited the quotation since. Several quotations can be offered side by side (alternatives) and only one can win.

## Decision
* **Tables in `offers`** (migration `000090`): `quotations`, `quotation_counters`, `quotation_suppliers`, `quotation_events`,
  `quotation_event_sessions`, `quotation_lines`, `quotation_terms`, `quotation_revisions`, `quotation_activity`. Every table carries
  `tenant_id`, every unique key starts with it, and every query filters by the session's tenant; another workspace's id answers 404.
  Kinds, venues, vendors, categories, packages, staff and projects are primitive ids (no cross-module foreign key).
* **Numbering.** `KODE-PNW-YYMMDD-NNN` — the workspace's `document_code` (2–5 capital letters or digits, otherwise `WO`), the WIB day
  of creation, and a per-workspace-per-day sequence from `quotation_counters` (row-locked `INSERT … ON DUPLICATE KEY UPDATE`). The
  number is frozen at creation: changing the document code later never renames an earlier quotation. A 1000th quotation on one day is
  refused (422 `quotation_daily_limit`).
* **Working copy vs revisions.** The quotation's rows are an editable *working copy*. `POST /quotations/{id}/revisions` freezes it into
  `quotation_revisions.snapshot` (JSON: the content with positions instead of database ids, the WO's letterhead identity incl. the logo
  key, the validity date and PIC name) and stores `content_hash` = sha256 of the canonical content only. **Identical content never adds a
  revision**: the same hash returns the existing revision (200 "Tidak ada perubahan sejak Revisi N"), so concurrent clicks make one
  (`UNIQUE (tenant_id, quotation_id, content_hash)` and `(…, n)`; a duplicate-key race re-reads and reuses). The validity date, notes and
  the PIC's display name are not hashed (extending validity alone is not a new offer). A revision needs at least one line and terms that
  add up to the total.
* **Status machine** (`domain.PlanTransition`; explicit `POST /quotations/{id}/status`, never a toggle). Stored: `Draft`, `Accepted`,
  `Rejected`, `Superseded`. `Kedaluwarsa` is computed (a Draft past its last WIB day) and never stored.
  * Draft → Accepted needs a revision whose hash equals the working copy (`quotation_needs_revision`) and, when expired, a new
    `extendUntil` ≥ today (`quotation_expired_needs_extension`). The accepted revision, day and channel (WhatsApp, Tatap muka, Email,
    Lainnya) are recorded; accepting one quotation of a group supersedes the other Drafts of that group (expired ones included,
    Rejected ones stay Rejected).
  * Draft → Rejected (optional reason); Rejected → Draft (reopen); Accepted → Draft (unaccept) only before conversion
    (`quotation_converted` after). Superseded is final and automatic only (`quotation_superseded`, `quotation_status_automatic`).
  * **Copy as new** makes a new Draft with a new number and supersedes the original (a Rejected original stays Rejected);
    **alternatives** are siblings under the group's first quotation. Only a Draft without revisions or alternatives can be deleted.
  * Only a Draft that is not converted is editable (`quotation_read_only` otherwise).
* **Edits** follow ADR-0036: the header uses `updatedAt`, the lines `linesUpdatedAt`, the terms and T&C text `termsUpdatedAt`, each
  event and supplier its own `updatedAt` — so saving lines never makes an open header form stale. Every content write answers the full
  detail so the screen never re-fetches.
* **PDF library: `github.com/go-pdf/fpdf` v0.9.0** (pure Go, no CGO, no headless browser in the one app container). The PDF is
  set in Plus Jakarta Sans (static Regular and Bold TTF from github.com/tokotype/PlusJakartaSans, SIL OFL 1.1, licence beside the
  files in `offers/infrastructure/pdf/fonts/`), embedded in the binary with `go:embed` and subset per document (a 40-line quotation
  is about 30 kB); text is written as UTF-8, so any letter prints. Tests that read the content streams as text render with the core
  Helvetica font instead, since an embedded font writes glyph ids. A revision's PDF is **rendered again from its snapshot** on every request, deterministically (fixed
  creation date from the snapshot), so the same revision always gives the same bytes; the working-copy preview is stamped "DRAF" and
  makes no revision. The PDF shows client-paid vendors by name only and never an agreed cost or estimated cost.
* **Ports** (no import of `staff`, `platform` or `projects`): `WorkspaceProfile` (document code, PKP, default validity and terms text,
  letterhead — adapter in `cmd/server` over `platform.LetterheadProfile`), `StaffNameResolver`, `FileStore` (logo and acceptance
  attachment, the shared storage client) and `DocumentRenderer` (the fpdf renderer inside `offers/infrastructure/pdf`).
* **Contract for projects** (`offers/contracts.Quotations`): `AcceptedQuotation(ctx, tenantID, quotationID)` (the accepted revision's
  content; 422 `quotation_not_accepted` otherwise), `MarkConverted(ctx, tenantID, quotationID, projectID)` (idempotent; the quotation
  becomes read-only) and `MarkProjectDeleted(ctx, tenantID, projectID)`.
* **Permissions.** `quotation.view` and `quotation.manage`, on by default for Owner, Admin and Sales. A role with `assigned` scope sees
  the quotations whose PIC it is. Agreed costs need `finance.margin.view` in every response, including revision snapshots and
  comparisons.

## Consequences
* A package or event kind used by a quotation can only be deactivated, never deleted (422 `package_in_quotation`; an event kind's
  `quotationCount` feeds "Dipakai di"), and a vendor, venue or category on a quotation is protected through the existing
  `OfferReferences` port (its `…Referenced` checks look at packages, then quotations).
* Revision snapshots carry a `schemaVersion`; the renderer reads older snapshots as they were written. Pre-launch there is no migration
  of older snapshot shapes.
* Switching to an embedded TTF font later changes the PDF bytes of every revision once (the content stays the same).
