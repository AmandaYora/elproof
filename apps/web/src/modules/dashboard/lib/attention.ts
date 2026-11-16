import type { DashboardStats } from "@/modules/dashboard/types";
import { daysUntil } from "@/modules/projects/lib/dates";
import { formatDate, formatCurrency, formatDayCountdown } from "@/shared/lib/formatters";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import { ISSUE_IMPACT_LABELS } from "@/shared/lib/labels";

export type AttentionTone = "danger" | "warning" | "info";

/** The filter chips of "Perlu ditindaklanjuti"; "lainnya" items show only under Semua. */
export type AttentionGroup = "tagihan" | "kendala" | "bukti" | "lainnya";

export interface AttentionItem {
  id: string;
  tone: AttentionTone;
  group: AttentionGroup;
  category: string;
  title: string;
  description: string;
  to: string;
}

const TONE_RANK: Record<AttentionTone, number> = { danger: 0, warning: 1, info: 2 };

/**
 * The name of an installment in a sentence. A WO usually names them "DP", "Termin 2", "Pelunasan"; "Termin" is added
 * only in front of a label that does not already start with it, so "Termin 2" never reads "Termin Termin 2".
 */
export function termName(label: string): string {
  const trimmed = label.trim();
  return /^termin\b/i.test(trimmed) ? trimmed : `Termin ${trimmed}`;
}

export function buildAttentionItems(stats: DashboardStats): { items: AttentionItem[]; counts: Record<AttentionGroup, number> } {
  const items: AttentionItem[] = [];

  for (const t of stats.overdueTerms) {
    items.push({
      id: `term-${t.projectId}-${t.label}-${t.dueDate}`,
      tone: "danger",
      group: "tagihan",
      category: "Tagihan terlambat",
      title: `${termName(t.label)} terlambat ${Math.abs(daysUntil(t.dueDate))} hari`,
      description: `${t.projectName} · sisa ${formatCurrency(t.remaining)}, jatuh tempo ${formatDate(t.dueDate)}`,
      to: ROUTE_PATHS.projectDetail(t.projectId, "pembayaran/client"),
    });
  }

  for (const issue of stats.openIssues) {
    items.push({
      id: `issue-${issue.id}`,
      tone: issue.impact === "Critical" || issue.impact === "High" ? "danger" : "warning",
      group: "kendala",
      category: "Kendala aktif",
      title: `Kendala dampak ${ISSUE_IMPACT_LABELS.staff[issue.impact].toLowerCase()}: ${issue.title}`,
      description: `${issue.projectName}${issue.vendorName ? ` · ${issue.vendorName}` : ""}`,
      to: ROUTE_PATHS.projectDetail(issue.projectId, "kendala"),
    });
  }

  for (const v of stats.vendorPaymentsDue ?? []) {
    const d = daysUntil(v.dueDate);
    if (d >= 0) continue;
    items.push({
      id: `vendordue-${v.projectId}-${v.vendorName}-${v.dueDate}`,
      tone: "warning",
      group: "lainnya",
      category: "Pembayaran vendor terlambat",
      title: `Pembayaran ke ${v.vendorName || "vendor"} terlambat ${Math.abs(d)} hari`,
      description: `${v.projectName} · sisa ${formatCurrency(v.remaining)}, jatuh tempo ${formatDate(v.dueDate)}`,
      to: ROUTE_PATHS.projectDetail(v.projectId, "pembayaran/vendor"),
    });
  }

  for (const m of stats.overdueVendorMilestones) {
    items.push({
      id: `vm-${m.id}`,
      tone: "warning",
      group: "lainnya",
      category: "Tahapan terlambat",
      title: `Tahapan "${m.name}" terlambat`,
      description: `${m.projectName} · ${m.vendorName} · target ${formatDate(m.targetDate)}`,
      to: ROUTE_PATHS.projectDetail(m.projectId, "vendor"),
    });
  }

  for (const p of stats.incompletePayments) {
    items.push({
      id: `pay-${p.id}`,
      tone: "warning",
      group: "bukti",
      category: "Bukti pembayaran",
      title: `Pembayaran ${p.type} belum memiliki bukti lengkap`,
      description: `${p.projectName} · ${p.vendorName} · ${formatCurrency(p.amount)}`,
      to: ROUTE_PATHS.projectDetail(p.projectId, "pembayaran/vendor"),
    });
  }

  // Same category as the vendor's own incomplete-payment items above: the two ledgers merge into one feed, a venue
  // payment named by "Venue" instead of a vendor name (it has none).
  for (const p of stats.incompleteVenuePayments) {
    items.push({
      id: `venuepay-${p.id}`,
      tone: "warning",
      group: "bukti",
      category: "Bukti pembayaran",
      title: `Pembayaran ${p.type} (venue) belum memiliki bukti lengkap`,
      description: `${p.projectName} · Venue · ${formatCurrency(p.amount)}`,
      to: ROUTE_PATHS.projectDetail(p.projectId, "pembayaran/venue"),
    });
  }

  for (const project of stats.nearDDayProjects) {
    const d = daysUntil(project.eventDate);
    // The server lists only events still ahead; the browser's "today" can be a day later (time zone), and an event
    // that has passed is not "approaching" any more.
    if (d < 0) continue;
    items.push({
      id: `dday-${project.id}`,
      tone: d <= 7 ? "danger" : "info",
      group: "lainnya",
      category: "Mendekati hari H",
      title: d === 0 ? "Hari ini hari H" : `${formatDayCountdown(d)} menuju hari H`,
      description: `${project.name} · ${formatDate(project.eventDate)}`,
      to: ROUTE_PATHS.projectDetail(project.id),
    });
  }

  for (const row of stats.laggingProjects) {
    items.push({
      id: `lag-${row.project.id}`,
      tone: "warning",
      group: "lainnya",
      category: "Progres tertinggal",
      title: "Progres persiapan tertinggal dari rencana",
      description: `${row.project.name} · ${row.overallPercent}% tahapan selesai, ${formatDayCountdown(daysUntil(row.project.eventDate))}`,
      to: ROUTE_PATHS.projectDetail(row.project.id),
    });
  }

  items.sort((a, b) => TONE_RANK[a.tone] - TONE_RANK[b.tone]);

  for (const v of stats.unconfirmedVendors) {
    items.push({
      id: `unconfirmed-${v.projectId}-${v.vendorName}`,
      tone: "warning",
      group: "lainnya",
      category: "Vendor belum pasti",
      title: `${v.vendorName} belum terkonfirmasi`,
      description: `${v.projectName} · acara ${formatDate(v.eventDate)} (${formatDayCountdown(daysUntil(v.eventDate))})`,
      to: ROUTE_PATHS.projectDetail(v.projectId, "vendor"),
    });
  }

  const counts: Record<AttentionGroup, number> = { tagihan: 0, kendala: 0, bukti: 0, lainnya: 0 };
  for (const item of items) counts[item.group] += 1;

  return { items, counts };
}
