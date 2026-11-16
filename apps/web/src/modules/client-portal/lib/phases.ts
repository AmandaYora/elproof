import { daysBetween, todayISO } from "@/shared/lib/formatters";

export type PhaseId = "bulan-ini" | "1-3-bulan" | "lebih-3-bulan" | "menjelang-hari-h";

const LABEL: Record<PhaseId, string> = {
  "bulan-ini": "Bulan ini",
  "1-3-bulan": "1–3 bulan lagi",
  "lebih-3-bulan": "Lebih dari 3 bulan lagi",
  "menjelang-hari-h": "Menjelang hari H",
};

/** The last stretch before the day gets its own group, whatever month it falls in. */
export const FINAL_DAYS = 14;

/**
 * Which phase an open step belongs to, counted from today: within 30 days (late and running steps included) is "Bulan
 * ini", then up to 90 days, then further; a step in the last 14 days before the event is "Menjelang hari H", and so is
 * every step up to the day once the event itself is within 30 days. Steps after the day go by their date.
 */
export function phaseOf(targetDate: string, eventDate: string | undefined, today = todayISO()): PhaseId {
  const fromToday = daysBetween(today, targetDate);
  if (eventDate) {
    const beforeEvent = daysBetween(targetDate, eventDate);
    if (beforeEvent >= 0 && (beforeEvent <= FINAL_DAYS || daysBetween(today, eventDate) <= 30)) return "menjelang-hari-h";
  }
  if (fromToday <= 30) return "bulan-ini";
  if (fromToday <= 90) return "1-3-bulan";
  return "lebih-3-bulan";
}

interface Step {
  status: string;
  targetDate: string;
}

/** Open steps in their phases, phases in time order (by their earliest step), steps by date with running/blocked ones first. */
export function phaseSteps<T extends Step>(steps: T[], eventDate: string | undefined): Array<{ id: PhaseId; label: string; steps: T[] }> {
  const groups = new Map<PhaseId, T[]>();
  for (const s of steps) {
    const id = phaseOf(s.targetDate, eventDate);
    groups.set(id, [...(groups.get(id) ?? []), s]);
  }
  const running = (s: T) => Number(s.status === "In Progress" || s.status === "Blocked");
  const earliest = (list: T[]) => list.reduce((min, s) => (s.targetDate < min ? s.targetDate : min), list[0].targetDate);
  return [...groups.entries()]
    .sort(([, a], [, b]) => earliest(a).localeCompare(earliest(b)))
    .map(([id, list]) => ({
      id,
      label: LABEL[id],
      steps: list.sort((a, b) => running(b) - running(a) || a.targetDate.localeCompare(b.targetDate)),
    }));
}
