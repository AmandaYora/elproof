// One place for every status label shown to people (FLOW-IDEAL.md §4,
// PLAN.md Fase 3b). The database keeps English enum values; staff and
// clients each read their own wording. `null` for a client means "do not
// show this to the client" (internal detail).
import type { EngagementStatus, IssueImpact, IssueStatus, MilestoneStatus, PaymentTermStatus, ProjectStatus } from "@/modules/projects/types";

export type Audience = "staff" | "client";

interface Labels<T extends string> {
  staff: Record<T, string>;
  client: Record<T, string | null>;
}

export const PROJECT_STATUS_LABELS: Labels<ProjectStatus> = {
  staff: { Draft: "Draf", Preparation: "Persiapan", Ready: "Siap", Completed: "Selesai", Cancelled: "Dibatalkan" },
  client: { Draft: null, Preparation: "Sedang dipersiapkan", Ready: "Persiapan akhir", Completed: "Selesai, terima kasih", Cancelled: "Dibatalkan" },
};

export const ENGAGEMENT_STATUS_LABELS: Labels<EngagementStatus> = {
  staff: {
    Planned: "Direncanakan",
    Negotiation: "Negosiasi",
    Booked: "Dipesan",
    "DP Paid": "DP dibayar",
    "In Progress": "Dikerjakan",
    "Fully Paid": "Lunas",
    Ready: "Siap",
    Completed: "Selesai",
    Cancelled: "Dibatalkan",
  },
  client: {
    Planned: "Sedang dipilih WO",
    Negotiation: "Sedang dipilih WO",
    Booked: "Sudah dipesan",
    "DP Paid": "Sudah dipesan",
    "In Progress": "Sedang disiapkan",
    "Fully Paid": "Sudah dipesan",
    Ready: "Siap",
    Completed: "Selesai",
    Cancelled: "Diganti WO",
  },
};

export const MILESTONE_STATUS_LABELS: Labels<MilestoneStatus> = {
  staff: { "Not Started": "Belum mulai", "In Progress": "Berjalan", Completed: "Selesai", Blocked: "Terhambat", Cancelled: "Dibatalkan" },
  client: { "Not Started": "Akan datang", "In Progress": "Sedang berjalan", Completed: "Selesai", Blocked: "Sedang berjalan", Cancelled: null },
};

export const ISSUE_STATUS_LABELS: Labels<IssueStatus> = {
  staff: { Open: "Baru", "In Review": "Ditinjau", "In Resolution": "Ditangani", Resolved: "Teratasi", Closed: "Ditutup" },
  client: {
    Open: "Sedang kami periksa",
    "In Review": "Sedang kami periksa",
    "In Resolution": "Sedang ditangani WO",
    Resolved: "Sudah beres",
    Closed: "Sudah beres",
  },
};

export const ISSUE_IMPACT_LABELS: Labels<IssueImpact> = {
  staff: { Low: "Rendah", Medium: "Sedang", High: "Tinggi", Critical: "Kritis" },
  client: { Low: null, Medium: null, High: null, Critical: null },
};

/** Payment-schedule installment status (FLOW-IDEAL §4). */
export const TERM_STATUS_LABELS: Labels<PaymentTermStatus> = {
  staff: { paid: "Lunas", overdue: "Terlambat", partial: "Dibayar sebagian", due_soon: "Segera jatuh tempo", upcoming: "Belum jatuh tempo" },
  client: {
    paid: "Lunas",
    overdue: "Lewat jatuh tempo — hubungi PIC",
    partial: "Dibayar sebagian",
    due_soon: "Jatuh tempo segera",
    upcoming: "Menunggu pembayaran",
  },
};

/** The label for one value, or null when the audience must not see it. */
export function labelFor<T extends string>(labels: Labels<T>, value: T, audience: Audience = "staff"): string | null {
  return audience === "client" ? labels.client[value] : labels.staff[value];
}
