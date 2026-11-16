import type { PaymentTerm, PaymentTermInput } from "@/modules/projects/types";

function addDays(iso: string, days: number): string {
  const d = new Date(`${iso}T00:00:00Z`);
  // No date yet (the wizard fills the schedule while it is being typed): no due date either.
  if (Number.isNaN(d.getTime())) return "";
  d.setUTCDate(d.getUTCDate() + days);
  return d.toISOString().slice(0, 10);
}

/**
 * The usual three-step schedule, so a WO starts from something instead of an
 * empty list: DP 30% on the booking date, Termin 40% 60 days before the event,
 * Pelunasan 30% 7 days before it. Rounded to the rupiah; the last installment
 * takes the remainder so the three always add up to the contract exactly. A
 * date never falls before the booking date.
 */
export function defaultTermTemplate(contract: number, bookingDate: string, eventDate: string): PaymentTermInput[] {
  const contractValue = Math.max(0, Math.trunc(contract)); // a value the form will still refuse never makes negative installments
  const dp = Math.round(contractValue * 0.3);
  const termin = Math.round(contractValue * 0.4);
  const clamp = (iso: string) => (iso < bookingDate ? bookingDate : iso);
  return [
    { label: "DP", amount: dp, dueDate: bookingDate },
    { label: "Termin 1", amount: termin, dueDate: clamp(addDays(eventDate, -60)) },
    { label: "Pelunasan", amount: contractValue - dp - termin, dueDate: clamp(addDays(eventDate, -7)) },
  ];
}

/** The sum of the installments. */
export function termsTotal(rows: Array<{ amount: number }>): number {
  return rows.reduce((sum, r) => sum + r.amount, 0);
}

/**
 * How the schedule stands against the contract (EP-05): "match" when the installments add up to it exactly, "under"
 * or "over" otherwise. With no contract value or no installments there is nothing to compare ("none").
 */
export function termsVsContract(total: number, contractValue: number, count: number): "none" | "match" | "under" | "over" {
  if (contractValue <= 0 || count === 0) return "none";
  if (total === contractValue) return "match";
  return total < contractValue ? "under" : "over";
}

/** Whether moving the event date leaves installments behind (QA RT-04): any installment not paid in full. */
export function hasUnpaidTerms(terms: PaymentTerm[]): boolean {
  return terms.some((t) => t.remaining > 0);
}

/**
 * The whole schedule (for PUT /payment-terms) with every installment not paid in full due `days` later (earlier when
 * negative), after the event date moved by that much. Paid installments, labels, amounts and order stay as they are.
 */
export function shiftUnpaidTerms(terms: PaymentTerm[], days: number): PaymentTermInput[] {
  return terms.map((t) => ({ label: t.label, amount: t.amount, dueDate: t.remaining > 0 ? addDays(t.dueDate, days) : t.dueDate }));
}
