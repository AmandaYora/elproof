# ADR-0019: Edit + hard delete for vendor/client/venue payments — Owner-or-Admin, evidence-first cascade

## Status
Accepted — implemented.

## Context

`vendor_payments`/`client_payments`/`venue_payments` (the three payment ledgers inside `projects`)
only ever supported Create + List — no way to correct a mistake (wrong amount, wrong type, wrong
date) or remove a bad entry once recorded. Unlike every entity ADR-0013/ADR-0018 already cover,
these three have **evidence rows hanging off them** (`evidence.related_kind = 'payment'` /
`'clientPayment'` / `'venuePayment'`) that themselves have no delete capability at all anywhere in
the codebase — so a naive hard delete of the payment row alone would leave those evidence rows (and
their S3 objects) as permanent, un-reachable ghosts: nothing references them anymore, and there is no
endpoint that could ever remove them afterward.

## Decision

### Edit: unconditional full-overwrite, no role gate — same convention as everything else in `projects`
`PATCH .../payments/{id}` (and the `client-payments`/`venue-payments` equivalents) overwrites every
ledger field (`type`, `amount`, `paymentDate`, `method`, `referenceNumber`, `notes`) unconditionally
— same shape as `updateProject`/`updateVendorEngagement`/`updateIssue`, none of which gate on role or
status. `projectVendorId` (vendor payments) is deliberately **not** editable — re-parenting a payment
to a different engagement isn't supported; delete and recreate under the correct engagement instead.

### Delete: hard delete, no precondition block — but Owner-or-Admin, not Owner-only
Two existing hard-delete guardrail models already exist to choose between:
- ADR-0013 (`projects.Delete`): blocks via precondition (must already be archived/cancelled), Owner-only.
- ADR-0018 (Vendor/Venue/Staff): informed-consent, no block, but **Owner-only, uniformly** ("Owner
  only, everywhere, via the existing `requireOwnerRole` helper").

This ADR follows ADR-0018's *no-block* shape (a payment has no status to block on, and the blast
radius is one row plus its own evidence — nowhere near Project's 8-table cascade) but deliberately
**widens the role bar to Owner-or-Admin**, matching the bar `vendors`/`venues` already use for their
own *non-destructive* writes (`requireManagerRole`) rather than the stricter Owner-only bar every
prior hard-delete used. This was an explicit user request, not a default — checked inline
(`isOwnerOrAdmin(claims.role)` in `apps/api/internal/modules/projects/presentation/util.go`) rather
than through `vendors`/`venues`' `requireOwnerRole` helper, since `projects` already resolves role
off its own `staffClaims` and has no equivalent helper of its own to reuse. **No `delete-impact`
endpoint** exists for payments (unlike Vendor/Venue/Staff) — the only thing a payment delete affects
is its own evidence, and the frontend already has that count in memory (the `evidence` array it
already fetched for the same tab), so no extra round-trip is needed to show it in the confirm dialog.

### Cascade: evidence deleted FIRST, then the payment row — deliberately reversed from intuition
`PaymentService.Delete` (and its two siblings) call `EvidenceService.DeleteForRelated(ctx, kind,
paymentID)` — a new method that deletes every matching evidence row, then best-effort deletes their
S3 objects (same log-and-continue pattern as `EvidenceService.DeleteStorageObjects`, ADR-0013) —
**before** deleting the payment row itself. This ordering is intentional, not arbitrary:
- If deleting the evidence rows fails (DB error), the whole operation aborts and the payment is
  untouched — nothing is left half-deleted.
- If evidence deletion succeeds but the payment-row delete then fails, the payment survives with no
  evidence attached — already a normal, expected state (a payment can legitimately have 0 evidence
  today, e.g. a partially-failed two-slot upload at creation time), fully recoverable by retrying
  the delete.
- The reverse order (payment first) would risk the opposite failure: payment gone, evidence rows
  orphaned forever with a `related_id` that no longer resolves to anything — and since evidence has
  no delete capability of its own, that ghost state would be permanent. Evidence-first ordering
  makes the *recoverable* failure mode the only one that can actually happen.

This is also the reason a **new** capability had to be added rather than reused: `EvidenceRepository`
had `ListByRelated` already (used for completeness checks) but no `DeleteByRelated` — added
specifically for this cascade, one `DELETE FROM evidence WHERE related_kind = ? AND related_id = ?`
statement.

### Activity logging: two new generic types, same convention as `payment_recorded`
`ActivityPaymentUpdated`/`ActivityPaymentDeleted` (`payment_updated`/`payment_deleted`) are shared
across all three payment kinds — distinguished by `entityType`/`entityLabel`/description at the
call site, not six separate per-kind constants, mirroring `ActivityPaymentRecorded`'s own existing
convention.

## Consequences
- `DATABASE_GUIDE.md`'s "Conventions" section now names **five** deliberate hard-delete exceptions,
  not four — this ADR is the fifth, and the first one that isn't Owner-only. Read ADR-0018's own
  "Owner-only, uniformly" section as scoped to Vendor/Venue/Staff specifically, not as a codebase-wide
  invariant, going forward.
- Routes added: `PATCH`/`DELETE /api/v1/projects/{id}/payments/{paymentId}` and the
  `client-payments`/`venue-payments` equivalents. See `docs/API_CONTRACT.md`.
- No migration needed — every field these mutations touch already existed; the cascade needed a new
  repository method (`EvidenceRepository.DeleteByRelated`) but no new column or table.
- Frontend: a new domain-agnostic shared component, `EvidenceListModal.tsx`
  (`apps/web/src/shared/components/ui/`), lets a payment row show its attached evidence inline
  ("Lihat Bukti") without leaving the Pembayaran tab — reuses the existing `EvidenceViewerModal`
  for the actual file view rather than duplicating that logic. Not itself part of the Edit/Delete
  decision, but shipped alongside it and reads the same `evidence` array the Delete confirm dialog
  already needed.
