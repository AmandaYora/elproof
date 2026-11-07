# ADR-0028: Client payment schedule and the action-first dashboard

## Status
Accepted — implemented 2026-11-07 (PLAN.md Fase 4b/4c, FLOW-IDEAL 2.3, 2.4, 2.6).

## Context
A project had a contract value and a ledger of payments, but nothing said when the client should
pay what. "Overdue" could not exist, so the dashboard showed omzet and counts rather than what a
WO has to do today. Creating a project was one long modal, and the venue was recorded twice (a
free-text field and a directory attachment) that could disagree.

## Decision
- **Schedule, not status, is stored.** `client_payment_terms` holds label, amount and due date.
  Status (`paid`, `overdue`, `partial`, `due_soon`, `upcoming`) is derived on read by
  `domain.AllocateTerms`: real client payments (DP + Termin + Pelunasan − Refund; Tambahan is
  outside the contract) are spread over the installments, earliest due date first. A part-paid
  installment past its due date is still `overdue`; one due today is `due_soon`, not overdue.
- **Whole-schedule replace.** `PUT /projects/{id}/payment-terms` swaps the list in one transaction
  (max 20, total ≤ contract value). One call keeps the editor simple and a failed save never leaves
  half a schedule. Needs `finance.record`; not readable by client principals (portal comes in Fase 5).
- **Dashboard answers "what is late, what is next".** One big number (Rp of client installments
  overdue), then the attention queue (overdue installments first), events in 14 days, Timeline items
  due this week, client payments due in 30 days, and project counts by stage. The payload gains
  `overdueTerms`, `overdueTermsTotal`, `dueSoonTerms`, `tasksDueThisWeek`, `stageCounts`.
- **Project page is two columns.** Tabs: Ringkasan (first; what needs action, next five Timeline
  items, next installment), Timeline, Vendor, Venue, Keuangan, Kendala, Dokumen, Klien, Aktivitas;
  a side panel carries stage, outstanding amount, open issues and team on every tab. Kendala lists
  all issues across vendors; they are still edited from the Vendor tab.
- **Create project in two steps** on `/projects/baru`: (1) couple, dates, package, value, PIC;
  (2) venue from the directory (with a per-project price snapshot) and the schedule (starting from a
  30/40/30 template: DP at booking date, Termin 1 at H-60, Pelunasan at H-7). A project is always
  created as Draf. If venue or schedule fail after the project exists, the user lands on the project
  with a message naming what is missing — nothing is typed twice.
- **One place for the venue.** While a directory venue is attached, the server sets the project's
  venue text to the venue's name (`ProjectService.Update`), and the edit form shows it read-only.
- **Owner picks the brand color** (`PATCH /tenants/me/brand-color`, `business_profile.manage`) on the
  new Profil Usaha page, next to name, WhatsApp, city and logo (the existing profile modal).

## Consequences
- Existing projects have no schedule; the dashboard shows nothing overdue for them until one is set.
- Address and the bank account clients should pay into are **not** stored yet: they only matter once
  the Client Portal shows them (Fase 5), so they are added there.
- The Wedding Planner/Sales "Proyek saya" dashboard is still open: the dashboard stays tenant-wide
  (`dashboard.view` + all-projects scope), as in ADR-0026.
