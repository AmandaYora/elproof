// Calendar arithmetic on plain YYYY-MM-DD strings (local Date objects at noon would also do; strings keep every
// comparison a calendar-day comparison, the same rule the API uses in WIB).

export const MONTH_NAMES = [
  "Januari", "Februari", "Maret", "April", "Mei", "Juni", "Juli", "Agustus", "September", "Oktober", "November", "Desember",
];
/** Monday first, as Indonesian calendars print it. */
export const DAY_NAMES = ["Senin", "Selasa", "Rabu", "Kamis", "Jumat", "Sabtu", "Minggu"];
export const DAY_SHORT = ["Sen", "Sel", "Rab", "Kam", "Jum", "Sab", "Min"];

const pad = (n: number) => String(n).padStart(2, "0");

function parts(iso: string): [number, number, number] {
  const [y, m, d] = iso.split("-").map(Number);
  return [y, m, d];
}

function toISO(date: Date): string {
  return `${date.getUTCFullYear()}-${pad(date.getUTCMonth() + 1)}-${pad(date.getUTCDate())}`;
}

function utc(iso: string): Date {
  const [y, m, d] = parts(iso);
  return new Date(Date.UTC(y, m - 1, d));
}

export function addDays(iso: string, days: number): string {
  const date = utc(iso);
  date.setUTCDate(date.getUTCDate() + days);
  return toISO(date);
}

/** 0 = Monday … 6 = Sunday. */
export function weekdayIndex(iso: string): number {
  return (utc(iso).getUTCDay() + 6) % 7;
}

/** "2026-11" of a date. */
export function monthOf(iso: string): string {
  return iso.slice(0, 7);
}

export function isMonthKey(value: string | null): value is string {
  return !!value && /^\d{4}-(0[1-9]|1[0-2])$/.test(value);
}

export function shiftMonth(month: string, delta: number): string {
  const [y, m] = month.split("-").map(Number);
  const date = new Date(Date.UTC(y, m - 1 + delta, 1));
  return `${date.getUTCFullYear()}-${pad(date.getUTCMonth() + 1)}`;
}

export function daysInMonth(month: string): number {
  const [y, m] = month.split("-").map(Number);
  return new Date(Date.UTC(y, m, 0)).getUTCDate();
}

/** The same day number in another month, pulled to that month's last day when it has fewer days. */
export function sameDayIn(iso: string, month: string): string {
  const day = Math.min(parts(iso)[2], daysInMonth(month));
  return `${month}-${pad(day)}`;
}

/** "November 2026" */
export function monthLabel(month: string): string {
  const [y, m] = month.split("-").map(Number);
  return `${MONTH_NAMES[m - 1]} ${y}`;
}

/** "Sabtu, 21 November 2026" (DESIGN.md §7 long form). */
export function longDate(iso: string): string {
  const [y, m, d] = parts(iso);
  return `${DAY_NAMES[weekdayIndex(iso)]}, ${d} ${MONTH_NAMES[m - 1]} ${y}`;
}

/** "Sabtu, 21 November" */
export function dayHeading(iso: string): string {
  const [, m, d] = parts(iso);
  return `${DAY_NAMES[weekdayIndex(iso)]}, ${d} ${MONTH_NAMES[m - 1]}`;
}

export function dayNumber(iso: string): number {
  return parts(iso)[2];
}

/**
 * The weeks a month grid shows: from the Monday on or before the 1st to the Sunday on or after the last day
 * (4 to 6 rows, at most 42 days — inside the API's 93-day limit).
 */
export function monthGrid(month: string): { from: string; to: string; weeks: string[][] } {
  const first = `${month}-01`;
  const last = `${month}-${pad(daysInMonth(month))}`;
  const from = addDays(first, -weekdayIndex(first));
  const to = addDays(last, 6 - weekdayIndex(last));
  const weeks: string[][] = [];
  for (let day = from; day <= to; day = addDays(day, 7)) {
    weeks.push(Array.from({ length: 7 }, (_, i) => addDays(day, i)));
  }
  return { from, to, weeks };
}

/** "Hari ini" / "Besok" / "Kemarin" for the agenda's day headings, otherwise null. */
export function relativeDay(iso: string, today: string): string | null {
  if (iso === today) return "Hari ini";
  if (iso === addDays(today, 1)) return "Besok";
  if (iso === addDays(today, -1)) return "Kemarin";
  return null;
}
