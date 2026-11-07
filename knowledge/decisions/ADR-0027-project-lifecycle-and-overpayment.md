# ADR-0027: Explicit project lifecycle and the overpayment rule

## Status
Accepted — implemented 2026-11-07 (PLAN.md Fase 4a, FLOW-IDEAL 2.4).

## Context
Any editor could set a project to any status through the edit form, so a project could jump from
Draft to Completed, and a finished project was one dropdown away from open again. Nothing stopped
a payment from exceeding what was owed, so "Sisa tagihan" could go negative without anyone
noticing. Closing a project had no moment where the money and the issues were looked at.

## Decision
- **Status changes only through actions.** `POST /projects/{id}/transition` with `start_preparation`,
  `mark_ready`, `complete` or `reopen`; `POST .../cancel` stays. The allowed steps live in one pure
  function (`domain.NextStatus`); anything else is a 409 `invalid_status_transition` (a stale page
  cannot skip a step). `PATCH /projects/{id}` ignores `status`. `reopen` counts as one more active
  project, so the trial cap applies (ADR-0024).
- **Closing checklist is advice, not a gate.** `GET .../closing-checklist` (staff only) reports
  whether the client is paid in full, vendors and venue are paid, and issues are closed.
  `complete` works either way; the UI shows the list first and the button becomes "Tetap
  selesaikan" when something is open. A WO may close knowingly (a vendor waived a balance).
- **One overpayment rule for the three ledgers** (`domain.ExcessAfterChange`). The rule is about
  the *change*, not the payment: an add, edit or delete is refused (422, message in `errors.amount`)
  only when it both makes the applied total exceed what is owed **and** raises it. `DP`, `Termin`
  and `Pelunasan` count towards the total, `Refund` subtracts, `Tambahan` (work outside the contract)
  never counts. Consequences: lowering or deleting a payment is always allowed; editing a reference
  number or date on an already-overpaid project is allowed (the total does not move); deleting or
  shrinking a Refund is checked like adding a payment (otherwise Refund → Termin → delete Refund
  would bypass the limit). Owed amounts: client → project `contractValue`; vendor → that
  engagement's `contractValue`; venue → rental price + charge. A contract of 0 means "not set yet"
  and is not limited. Every amount must be 1 … `MaxMoney` (Rp 10^15) so sums can never wrap. The
  frontend mirrors the rule (`modules/projects/lib/payments.ts`) for the form hint and the
  "Lebih bayar" badge; the API is the authority.
- **Status writes are conditional.** `ChangeStatus(from → to)` only updates a project still in
  `from`; a second person acting on a stale page gets a 409 `invalid_status_transition` instead of
  silently undoing a step. `Update` no longer writes `status` at all. `Cancel` only applies to an
  open project (Completed/Cancelled → 409).
- The services read what they are owed through a narrow `ContractReader` (inside `projects`), not by
  reaching into repositories of other services.

## Consequences
- "Sisa tagihan" now excludes `Tambahan`, so extra charges no longer make the client look less
  indebted for the contract. "Total diterima" is still all cash in.
- Vendor "paidAmount" in the vendor list still counts every payment type; only the overpayment
  check and the closing checklist use the contract-only rule.
- Existing data that is already over the limit stays; it shows "Lebih bayar" and only changes that
  raise the total are refused.
- Known limitation: the check is read-then-write without a lock, so two simultaneous payments that
  each fit alone can both be saved. Acceptable for a WO's handful of editors; a row lock per project
  would close it if it ever matters.
- Lowering the contract value below what is already paid is allowed (the dashboard then shows
  "Lebih bayar"); only further increases are refused.
