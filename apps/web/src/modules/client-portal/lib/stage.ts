import type { ProjectStatus } from "@/modules/projects/types";

/** The four steps a couple sees (FLOW-IDEAL 3.2), like an order tracker. */
export const STAGE_LABELS = ["Sedang dipersiapkan", "Persiapan akhir", "Hari bahagia Anda", "Selesai"] as const;

/**
 * Which step the project is on, or null when it was cancelled.
 * `daysToEvent` is days until the event (negative once it has passed).
 */
export function stageIndex(status: ProjectStatus, daysToEvent: number): number | null {
  if (status === "Cancelled") return null;
  if (status === "Completed") return 3;
  // The day itself and the days after it, until the WO marks the project done.
  if (daysToEvent <= 0) return 2;
  if (status === "Ready") return 1;
  return 0;
}

export interface Countdown {
  /** The big figure: "H-45", "Hari ini", "H+3"… */
  figure: string;
  caption: string;
}

/** The hero's countdown, including what to say once the day has passed. */
export function countdown(status: ProjectStatus, daysToEvent: number): Countdown {
  if (status === "Cancelled") return { figure: "—", caption: "Persiapan ini dibatalkan. Hubungi wedding organizer Anda bila ada yang ingin ditanyakan." };
  if (status === "Completed") {
    return daysToEvent < 0
      ? { figure: "Selesai", caption: `Sudah ${Math.abs(daysToEvent)} hari sejak hari bahagia Anda. Terima kasih sudah mempercayakannya kepada kami.` }
      : { figure: "Selesai", caption: "Seluruh persiapan telah selesai. Terima kasih." };
  }
  if (daysToEvent > 0) return { figure: `H-${daysToEvent}`, caption: daysToEvent === 1 ? "Besok hari bahagia Anda" : "hari menuju hari bahagia Anda" };
  if (daysToEvent === 0) return { figure: "Hari ini", caption: "Selamat menempuh hidup baru!" };
  return { figure: `H+${Math.abs(daysToEvent)}`, caption: "Selamat atas pernikahan Anda. Kami sedang merapikan hal-hal terakhir." };
}
