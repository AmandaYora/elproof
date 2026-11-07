import type { BadgeTone } from "@/shared/components/ui/Badge";
import type { DataRequest, DataRequestStatus } from "@/modules/platform-admin/data/console.types";

export const DATA_REQUEST_KIND_LABEL = { export: "Salinan data", delete: "Hapus data" } as const;

export const DATA_REQUEST_STATUS: Record<DataRequestStatus, { label: string; tone: BadgeTone }> = {
  open: { label: "Terbuka", tone: "warning" },
  done: { label: "Selesai", tone: "success" },
  rejected: { label: "Ditolak", tone: "neutral" },
};

/** The statutory deadline, in words: how long is left, or how late it is. */
export function dueText(r: Pick<DataRequest, "status" | "daysUntilDue">): { text: string; late: boolean } {
  if (r.status !== "open") return { text: "", late: false };
  if (r.daysUntilDue < 0) return { text: `terlambat ${Math.abs(r.daysUntilDue)} hari`, late: true };
  if (r.daysUntilDue === 0) return { text: "batas hari ini", late: true };
  return { text: `${r.daysUntilDue} hari lagi`, late: r.daysUntilDue <= 5 };
}
