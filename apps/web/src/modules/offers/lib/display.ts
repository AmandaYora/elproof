import { formatDate } from "@/shared/lib/formatters";
import type { Quotation } from "@/modules/offers/types";

/** How long a quotation still holds, in words ("Berlaku sampai 12 Des 2026 · 5 hari lagi", "Lewat sejak …"). */
export function validityText(q: Pick<Quotation, "validUntil" | "daysLeft" | "displayStatus">): string {
  if (!q.validUntil) return "Masa berlaku belum diisi";
  if (q.displayStatus === "Expired" || q.daysLeft < 0) return `Lewat sejak ${formatDate(q.validUntil)}`;
  if (q.displayStatus !== "Draft") return `Berlaku sampai ${formatDate(q.validUntil)}`;
  const left = q.daysLeft === 0 ? "berakhir hari ini" : `${q.daysLeft} hari lagi`;
  return `Berlaku sampai ${formatDate(q.validUntil)} · ${left}`;
}

/** A draft that ends within a week: worth a nudge (the list's "Segera kedaluwarsa" filter uses the same 7 days). */
export function expiresSoon(q: Pick<Quotation, "daysLeft" | "displayStatus">): boolean {
  return q.displayStatus === "Draft" && q.daysLeft >= 0 && q.daysLeft <= 7;
}

/** "Revisi 2" or "Belum ada revisi". */
export function revisionWord(n: number | null | undefined): string {
  return n ? `Revisi ${n}` : "Belum ada revisi";
}

/** The day after tomorrow-style default for a new validity: today plus `days`, as YYYY-MM-DD. */
export function daysFromToday(days: number): string {
  const d = new Date();
  d.setDate(d.getDate() + days);
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
}

/** When a term falls due, in words: "Saat diterima", "14 hari setelah diterima", "7 hari sebelum Resepsi". */
export function termDueText(t: { dueBasis: "Accepted" | "Event"; dueOffsetDays: number }, eventName: string): string {
  const n = Math.abs(t.dueOffsetDays);
  if (t.dueBasis === "Accepted") return n === 0 ? "Saat diterima" : `${n} hari setelah diterima`;
  if (t.dueOffsetDays === 0) return `Pada hari ${eventName}`;
  return t.dueOffsetDays < 0 ? `${n} hari sebelum ${eventName}` : `${n} hari sesudah ${eventName}`;
}

/** A WhatsApp number as wa.me wants it: digits only, a leading 0 becomes Indonesia's 62. "" when there is none. */
export function waNumber(raw: string): string {
  const digits = raw.replace(/[^0-9]/g, "");
  if (!digits) return "";
  return digits.startsWith("0") ? `62${digits.slice(1)}` : digits;
}
