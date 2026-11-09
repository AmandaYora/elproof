import { formatDate } from "@/shared/lib/formatters";

const JAKARTA_DAY = new Intl.DateTimeFormat("en-CA", { timeZone: "Asia/Jakarta", year: "numeric", month: "2-digit", day: "2-digit" });

/** The calendar day (YYYY-MM-DD) a timestamp falls on in Jakarta, where every WO works. */
export function jakartaDay(iso: string): string {
  const d = new Date(iso);
  return Number.isNaN(d.getTime()) ? iso.split("T")[0] : JAKARTA_DAY.format(d);
}

/**
 * The words on a stamp, the same in the Portal, the console and WhatsApp
 * messages (DESIGN.md §3): "Terverifikasi 7 Nov oleh Rina". The name is left
 * out when the server could not resolve it.
 */
export function verifiedLabel(verifiedAt: string, verifiedBy: string | null | undefined): string {
  const by = verifiedBy ? ` oleh ${verifiedBy}` : "";
  return `Terverifikasi ${formatDate(jakartaDay(verifiedAt))}${by}`;
}
