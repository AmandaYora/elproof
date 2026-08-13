# ADR-0020: General project documents + per-document Client Portal visibility

## Status
Accepted — implemented.

## Context

The internal "Dokumen & Evidence" tab required every evidence row to attach to a specific existing
context: `relatedKind` + `relatedId` pointing at a vendor milestone, payment, vendor engagement, or
issue (`relatedId` was `NOT NULL`, and the backend enum had exactly those four/six fixed values).
Free-standing operational documents with no such link — rundown, buku acara, teks juru bicara,
banquet order, rekap order, rekap dekor, etc. — had nowhere to live.

Extending this to also show up in Client Portal surfaced a second, unrelated gap: evidence had **no
client-visibility distinction at all**. Every evidence row was (and, for the six pre-existing kinds,
still is) unconditionally visible to a `client` principal the instant it's attached to something
inside their own project — there was no per-document opt-in/opt-out anywhere. Making a new
general-documents feature visible in Client Portal without addressing this first would have meant
either exposing every general document by default (no staff review step) or building something
narrower without ever naming the underlying gap.

## Decision

### A 7th `related_kind` value, `'general'`, with a sentinel `related_id = 0`
Reuses the entire existing evidence machinery (upload, list, download, storage) rather than a new
table — the only structural addition is one more enum value (migration `000035`,
`ALTER TABLE evidence MODIFY COLUMN related_kind ENUM(...)`, same pattern already used twice before
for `clientPayment`/`venuePayment`). `related_id = 0` is the sentinel for "nothing to point at" —
`AUTO_INCREMENT` never starts at 0, mirroring `projects.venue_id`'s own "0 = detach" convention,
rather than making the column nullable.

### A new `is_client_visible BOOLEAN NOT NULL DEFAULT FALSE` column — safe-by-default, opt-in only
Meaningless for the six pre-existing kinds (they stay unconditionally client-visible, unchanged) —
only ever read/enforced for `related_kind = 'general'`. Defaults `FALSE`: a staff member must
explicitly toggle a general document visible before a client can ever see it. This was an explicit
choice between two options put to the user (toggle per document, default hidden vs. all general
documents automatically visible) — the narrower, opt-in option was chosen specifically because the
user's own examples spanned genuinely mixed sensitivity (a rundown is fine to share; a rekap dekor
or internal banquet-order breakdown may not be).

Toggled via `POST /projects/{id}/evidence/{evidenceId}/toggle-client-visible` — a narrow,
single-purpose endpoint (same `toggle-*` naming convention as `toggle-active`/`toggle-suspension`
elsewhere), not folded into a general evidence-edit endpoint (none exists). Rejects `422` for any
`related_kind` other than `general` — visibility has no meaning for the other six kinds, so
attempting to toggle one is a client error, not a silent no-op. The upload path independently,
server-side, re-normalizes `isClientVisible` (`input.IsClientVisible && input.RelatedKind ==
RelatedGeneral`) — never trusted blindly from the request body, so it can never end up `true` for
any other kind even if a caller tries to set it directly.

### A new, dedicated `GET /projects/{id}/documents` endpoint — not a filter bolted onto `listEvidence`
Client Portal's own "Dokumen" tab reads this endpoint, which **unconditionally** filters to
`related_kind = 'general' AND is_client_visible = TRUE`, regardless of which principal calls it
(reachable by staff too, harmlessly). The alternative — adding principal-type branching logic inside
the existing `listEvidence` handler — was deliberately rejected: the safety property this endpoint
exists for ("this can never return anything unsafe to show a client") needs to be trivially
auditable by reading one query in isolation, not entangled with staff-only branches inside a shared,
more general-purpose handler. `GET /projects/{id}/evidence` itself stays exactly as unfiltered as it
already was — client-visibility scoping was never retrofitted onto it.

### Frontend: reuses `EvidenceViewerModal` as-is; new tab appended at the end, not interleaved
Client Portal's tab order (Vendor, Timeline, Pembayaran, Kendala, **Dokumen**) deliberately keeps
"Dokumen" last — a new, separate, opt-in capability, not a reordering of anything already there. WO
Console's own evidence section gained a 7th "Umum" context option (hides the now-irrelevant
"Terkait" picker for that context) and a visibility badge/toggle button per general-kind row; no
existing UI for the six other kinds changed.

## Consequences
- Migration `000035`: `related_kind` enum gains `'general'`; new `is_client_visible` column. No
  other schema change — every other table/column this feature touches already existed.
- Routes: `POST /projects/{id}/evidence/{evidenceId}/toggle-client-visible`,
  `GET /projects/{id}/documents`. See `docs/API_CONTRACT.md`.
- The pre-existing "zero client-visibility distinction" gap this ADR surfaced is now closed **only**
  for `general`-kind documents. The six pre-existing kinds remain unconditionally client-visible —
  narrowing *those* was explicitly out of scope for this change and would be its own, separate
  decision if ever revisited.
- Every future hard-delete/cascade touching `evidence` (see ADR-0019) treats `general`-kind rows
  identically to every other kind — `EvidenceService.DeleteForRelated`/`DeleteByRelated` key off
  `related_kind` + `related_id` generically, with no special-casing for the sentinel `related_id = 0`
  (nothing ever calls that cascade with `kind='general'` today, since nothing hard-deletes "the
  project itself, but keep its documents" as a distinct concept — flagged here only so a future
  change touching general documents' own lifecycle doesn't assume otherwise).
