# ADR-0042: The Timeline follows the event date, the payment schedule can follow the contract value

## Status
Accepted — implemented 2026-11-15 (QA findings EP-05, EP-06, EP-10). Amends ADR-0014 §2 (duplicate) for
Timeline steps that follow the event date; extends ADR-0028 (schedule) and ADR-0036 (milestone PATCH).

## Context
QA on production found three things a WO cannot rely on:
- **EP-06** — every Timeline step of a project could sit on the booking date, Hari-H included, and none
  moved when the event date changed. Steps were dated once at creation (`event − offset`, pulled to the
  booking date when earlier), the offset was not kept, and the pull also applied when the event itself
  was on or before the booking (a past or mistyped event date), so every step landed on the booking date
  and stayed there after the event date was corrected. A step set to Selesai had no completed date
  (nothing stamped it when only the status was sent), and a step due today counted as late: the target
  day's midnight was compared with the current instant.
- **EP-05** — changing the contract value left the 30/40/30 installments computed from the old value, with
  no warning anywhere; in the new-project wizard the installments did not follow a contract value typed
  after they were filled.
- **EP-10** — a past event date was saved without a word and a typed minus sign on the contract value was
  silently dropped; the server did not refuse a negative contract value on create.

## Decision
- **A step remembers its offset.** `project_milestones.days_before_event` (migration `000075`, nullable):
  the Jadwal bawaan offset a step's target follows; `NULL` = a fixed date (added by hand, or its target set
  by hand). The target is `domain.MilestoneTarget`: `event − offset`, pulled to the booking date only when
  it falls before the booking **and** the booking is on or before the event — Hari H is always the event day.
- **Dates move with the event.** `PATCH /projects/{id}` that changes `eventDate` or `prepStartDate` re-dates
  every step that follows the event date and is not Selesai/Dibatalkan, in the same transaction as the
  project write, and logs how many moved. Setting a step's target by hand detaches it (`daysBeforeEvent →
  null`). A duplicate keeps each step's offset and dates it from the new project's own dates.
- **Completed date follows the status.** A milestone PATCH (project or vendor step) that changes `status`
  without a `completedDate` stamps today (WIB) on becoming Completed and clears the date on leaving it; a
  date sent in the body wins.
- **Late is a date comparison in WIB.** `IsMilestoneOverdue` and the dashboard's late vendor steps compare
  the target day with today's Indonesian calendar day: late only from the day after the due day. The web's
  `todayISO()` is the WIB day too.
- **The schedule is fitted on request, never silently.** A new contract value does not rewrite the
  schedule. The web warns wherever the installments do not add up to the contract (the schedule editor,
  the Jadwal Termin card, and right after a project edit that changed the value) and offers *Hitung ulang
  termin* → `POST /projects/{id}/payment-terms/recalculate`. Installments with any payment on them keep
  their amount; the ones with nothing paid share the rest in proportion to their current amounts, whole
  rupiah, remainder on the last (`domain.RecalculateTerms`). It is a server endpoint because "has money on
  it" is derived from the payments as they are at that moment (ADR-0028), which a schedule rebuilt in the
  browser cannot guarantee. It refuses (422 on `terms`, saying what to do) when nothing can be fitted.
- **The wizard's schedule is live until edited.** On the new-project page the installments are the 30/40/30
  template of the current contract value and dates until the person edits a row; *Pakai templat 30/40/30*
  makes them follow again.
- **Invalid numbers are refused, not changed.** The contract field keeps a typed minus sign (`CurrencyInput
  signed`) so the form shows "Nilai kontrak tidak boleh negatif"; the server refuses it with 422
  `contractValue` on create, duplicate and edit. A past event date stays allowed (historical projects are
  legitimate): the wizard warns under the field and asks once for confirmation, the edit form warns.

## Consequences
- Migration `000075` links existing steps that still look as seeded (same name as a Jadwal bawaan row of
  the workspace, not finished, dated `event − offset` or on the booking date) and re-dates them with the
  rule above — this repairs projects whose steps were stuck on the booking date. Steps dated by hand are
  left alone. The down migration drops the column and keeps the dates. Completed steps that already lack a
  completed date are not back-filled (the real date is unknown).
- A step whose target was moved by hand no longer follows the event date; there is no "follow again" yet.
- Vendor steps do not follow the event date (they have no template); only their completed date is stamped.
