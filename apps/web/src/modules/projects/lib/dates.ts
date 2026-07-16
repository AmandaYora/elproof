import { daysBetween, todayISO } from "@/shared/lib/formatters";
import type { Evidence, MilestoneStats, MilestoneStatus } from "@/modules/projects/types";

// Re-exported for the many existing call sites in this module — the actual
// implementation now lives in shared/lib/formatters.ts so other modules
// (e.g. platform-admin) don't have to import across module boundaries.
export { todayISO };

export function daysUntil(dateISO: string): number {
  return daysBetween(todayISO(), dateISO);
}

export function isPastDate(dateISO: string | null): boolean {
  if (!dateISO) return false;
  return daysUntil(dateISO) < 0;
}

export function isMilestoneOverdue(status: MilestoneStatus, targetDate: string): boolean {
  if (status === "Completed" || status === "Cancelled") return false;
  return isPastDate(targetDate);
}

// Mirrors the backend's domain.ComputeMilestoneStats exactly (same field
// names, same ratio formula) — used for tab-local stats (e.g. the project
// milestone tab) without needing a dedicated stats endpoint.
export function computeMilestoneStats(milestones: { status: MilestoneStatus; targetDate: string }[]): MilestoneStats {
  const relevant = milestones.filter((m) => m.status !== "Cancelled");
  const total = relevant.length;
  const completed = relevant.filter((m) => m.status === "Completed").length;
  const inProgress = relevant.filter((m) => m.status === "In Progress").length;
  const blocked = relevant.filter((m) => m.status === "Blocked").length;
  const notStarted = relevant.filter((m) => m.status === "Not Started").length;
  const cancelled = milestones.length - relevant.length;
  const overdue = milestones.filter((m) => isMilestoneOverdue(m.status, m.targetDate)).length;
  return {
    total,
    completed,
    inProgress,
    blocked,
    notStarted,
    cancelled,
    overdue,
    ratio: total === 0 ? 0 : completed / total,
  };
}

// The backend doesn't model "does this milestone require evidence" — this is
// a pure display heuristic ported as-is from the pre-integration mock
// (mock/selectors.ts's getMilestoneEvidenceCompleteness).
export function getMilestoneEvidenceCompleteness(
  milestoneName: string,
  linkedEvidence: Evidence[]
): { requiresEvidence: boolean; complete: boolean } {
  const requiresEvidence = milestoneName.toLowerCase().includes("dp dibayarkan") || milestoneName.toLowerCase().includes("pelunasan");
  if (!requiresEvidence) return { requiresEvidence: false, complete: true };
  const hasInvoice = linkedEvidence.some((e) => e.type === "Invoice" || e.type === "Receipt");
  const hasProof = linkedEvidence.some((e) => e.type === "Transfer Proof");
  return { requiresEvidence: true, complete: hasInvoice && hasProof };
}
