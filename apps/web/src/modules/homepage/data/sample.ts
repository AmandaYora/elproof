import { daysBetween, formatDate, formatDayCountdown } from "@/shared/lib/formatters";

/*
 * The one example wedding every illustration on the Beranda draws (QA FQ-11): the hero's WO Console and Portal Klien,
 * the "Cara kerja" cards (H-180 …), and the try-it preview. Amounts, percentages, dates, the verifier and the H-numbers
 * all come from here, so two pictures can never disagree. Dates are ISO days; every label is derived with the same
 * formatters the app uses (string arithmetic only, so the prerendered HTML and the browser agree).
 *
 * The story: contract signed at H-180 (15 Jun 2026), wedding on Saturday 12 Dec 2026, and the pictures are "taken" on
 * 10 Nov 2026 (H-32): the down payment and Termin 2 are paid and verified by Rina, Pelunasan is due 5 Dec (H-7).
 */

const MONTHS_LONG = ["Januari", "Februari", "Maret", "April", "Mei", "Juni", "Juli", "Agustus", "September", "Oktober", "November", "Desember"];
const WEEKDAYS = ["Minggu", "Senin", "Selasa", "Rabu", "Kamis", "Jumat", "Sabtu"];

/** "Sabtu, 12 Desember 2026" (DESIGN.md §7, long form), from an ISO day. */
export function formatLongDate(iso: string): string {
  const [y, m, d] = iso.split("-").map(Number);
  return `${WEEKDAYS[new Date(Date.UTC(y, m - 1, d)).getUTCDay()]}, ${d} ${MONTHS_LONG[m - 1]} ${y}`;
}

/** "2 Nov": the short date without the year, for tight rows. */
export function formatDayMonth(iso: string): string {
  return formatDate(iso).replace(/ \d{4}$/, "");
}

/** "2 Nov 2026, 19.42" (Indonesian time separator), from "2026-11-01T19:42". */
function formatUploadTime(isoDateTime: string): string {
  const [day, time] = isoDateTime.split("T");
  return `${formatDate(day)}, ${time.replace(":", ".")}`;
}

interface SampleTerm {
  id: string;
  label: string;
  percent: number;
  due: string;
  /** When the couple sent the transfer proof and when the team verified it; null while unpaid. */
  paid: { uploaded: string; verified: string } | null;
}

const CONTRACT = 45_000_000;
const EVENT_DATE = "2026-12-12";
const TODAY = "2026-11-10";
const VERIFIER = "Rina";

const TERMS: SampleTerm[] = [
  { id: "dp", label: "Uang muka", percent: 30, due: "2026-06-15", paid: { uploaded: "2026-06-15T10:15", verified: "2026-06-16" } },
  { id: "t2", label: "Termin 2", percent: 40, due: "2026-11-01", paid: { uploaded: "2026-11-01T19:42", verified: "2026-11-02" } },
  { id: "lunas", label: "Pelunasan", percent: 30, due: "2026-12-05", paid: null },
];

/** "H-32", "Hari H": how far a day is from the wedding. */
export function countdownTo(iso: string): string {
  return formatDayCountdown(daysBetween(iso, EVENT_DATE));
}

/** "Besok" for the sample's next day, otherwise the distance to the wedding. */
function whenLabel(iso: string): string {
  return daysBetween(TODAY, iso) === 1 ? "Besok" : countdownTo(iso);
}

const terms = TERMS.map((t) => {
  const amount = (CONTRACT * t.percent) / 100;
  return {
    ...t,
    amount,
    /** "Uang muka (30%)". */
    labelWithPercent: `${t.label} (${t.percent}%)`,
    dueLabel: formatDate(t.due),
    dueShort: formatDayMonth(t.due),
    paidLabel: t.paid ? formatDate(t.paid.uploaded.split("T")[0]) : null,
    uploadedLabel: t.paid ? formatUploadTime(t.paid.uploaded) : null,
    /** "Rina, 2 Nov 2026". */
    verifiedLabel: t.paid ? `${VERIFIER}, ${formatDate(t.paid.verified)}` : null,
    /** "Rina, 2 Nov". */
    verifiedShort: t.paid ? `${VERIFIER}, ${formatDayMonth(t.paid.verified)}` : null,
  };
});

const paidTotal = terms.filter((t) => t.paid).reduce((sum, t) => sum + t.amount, 0);
const milestonesDone = 18;
const milestonesTotal = 24;

/** The upcoming steps of the Jadwal on the sample day, nearest first. */
const UPCOMING = [
  { name: "Konfirmasi menu tasting", who: "Katering Rasa Nusantara", pic: "Rina", date: "2026-11-11" },
  { name: "Fitting baju kedua", who: "Sanggar Kebaya Laras", pic: "Yoga", date: "2026-11-21" },
  { name: "Technical meeting", who: "Semua vendor", pic: "Rina", date: "2026-11-28" },
];

export const SAMPLE = {
  couple: "Raka & Dinda",
  /** The partner the portal greets. */
  greeted: "Dinda",
  venue: "Gedung Sasana Kriya",
  eventDate: EVENT_DATE,
  /** "Sabtu, 12 Desember 2026". */
  eventDateLong: formatLongDate(EVENT_DATE),
  eventDateShort: formatDate(EVENT_DATE),
  /** The day the pictures are "taken": "H-32". */
  today: TODAY,
  countdown: countdownTo(TODAY),
  daysLeft: daysBetween(TODAY, EVENT_DATE),
  contract: CONTRACT,
  verifier: VERIFIER,
  terms,
  /** Look up one term by id ("dp", "t2", "lunas"). */
  term: (id: string) => terms.find((t) => t.id === id)!,
  paidTotal,
  remaining: CONTRACT - paidTotal,
  milestones: { done: milestonesDone, total: milestonesTotal, percent: Math.round((milestonesDone / milestonesTotal) * 100) },
  /** The step finished last before the sample day. */
  lastDone: { name: "Tasting katering", who: "Katering Rasa Nusantara", pic: "Rina" },
  upcoming: UPCOMING.map((u) => ({ ...u, when: whenLabel(u.date) })),
  /** The one open kendala on the sample day. */
  issue: {
    title: "Venue minta jadwal loading dimajukan",
    short: "Jadwal loading venue",
    reported: "2026-11-08",
    reportedLabel: formatDate("2026-11-08"),
    reportedAgo: `${daysBetween("2026-11-08", TODAY)} hari lalu`,
    countdown: countdownTo("2026-11-08"),
  },
  /** The days of the "Cara kerja" cards, as distances to the wedding. */
  journey: {
    contract: countdownTo("2026-06-15"),
    schedule: countdownTo("2026-07-15"),
    vendors: countdownTo("2026-08-14"),
    proof: countdownTo("2026-11-01"),
  },
  /** Steps of the Jadwal shown in the "Jadwal ikut bergeser" card. */
  schedule: [
    { name: "Survei venue dan vendor", date: "2026-09-13" },
    { name: "Technical meeting", date: "2026-11-28" },
  ].map((s) => ({ ...s, when: countdownTo(s.date) })),
} as const;
