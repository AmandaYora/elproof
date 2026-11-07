import type { PaymentTermInput } from "@/modules/projects/types";

function addDays(iso: string, days: number): string {
  const d = new Date(`${iso}T00:00:00Z`);
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
export function defaultTermTemplate(contractValue: number, bookingDate: string, eventDate: string): PaymentTermInput[] {
  const dp = Math.round(contractValue * 0.3);
  const termin = Math.round(contractValue * 0.4);
  const clamp = (iso: string) => (iso < bookingDate ? bookingDate : iso);
  return [
    { label: "DP", amount: dp, dueDate: bookingDate },
    { label: "Termin 1", amount: termin, dueDate: clamp(addDays(eventDate, -60)) },
    { label: "Pelunasan", amount: contractValue - dp - termin, dueDate: clamp(addDays(eventDate, -7)) },
  ];
}
