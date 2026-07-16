import type { DashboardStats } from "@/modules/dashboard/types";
import { daysUntil } from "@/modules/projects/lib/dates";
import { formatDate, formatCurrency } from "@/shared/lib/formatters";
import { ROUTE_PATHS } from "@/app/routes/route-paths";

export type AttentionTone = "danger" | "warning" | "info";

export interface AttentionItem {
  id: string;
  tone: AttentionTone;
  category: string;
  title: string;
  description: string;
  to: string;
}

interface VendorLookup {
  id: string;
  name: string;
}

const TONE_RANK: Record<AttentionTone, number> = { danger: 0, warning: 1, info: 2 };

export function buildAttentionItems(stats: DashboardStats, vendors: VendorLookup[]): { items: AttentionItem[]; counts: Record<string, number> } {
  const items: AttentionItem[] = [];
  const vendorName = (id: string) => vendors.find((v) => v.id === id)?.name ?? "";

  for (const issue of stats.openIssues) {
    items.push({
      id: `issue-${issue.id}`,
      tone: issue.impact === "Critical" || issue.impact === "High" ? "danger" : "warning",
      category: "Kendala Aktif",
      title: `Kendala ${issue.impact}: ${issue.title}`,
      description: `${issue.projectName}${vendorName(issue.vendorId) ? ` — ${vendorName(issue.vendorId)}` : ""}`,
      to: ROUTE_PATHS.projectDetail(issue.projectId, "kendala"),
    });
  }

  for (const m of stats.overdueVendorMilestones) {
    items.push({
      id: `vm-${m.id}`,
      tone: "warning",
      category: "Milestone Terlambat",
      title: `Milestone "${m.name}" terlambat`,
      description: `${m.projectName} — ${vendorName(m.vendorId)} · target ${formatDate(m.targetDate)}`,
      to: ROUTE_PATHS.projectDetail(m.projectId, "vendor"),
    });
  }

  for (const p of stats.incompletePayments) {
    items.push({
      id: `pay-${p.id}`,
      tone: "warning",
      category: "Evidence Pembayaran",
      title: `Pembayaran ${p.type} belum memiliki evidence lengkap`,
      description: `${p.projectName} — ${vendorName(p.vendorId)} · ${formatCurrency(p.amount)}`,
      to: ROUTE_PATHS.projectDetail(p.projectId, "pembayaran"),
    });
  }

  for (const project of stats.nearDDayProjects) {
    const d = daysUntil(project.eventDate);
    items.push({
      id: `dday-${project.id}`,
      tone: d <= 7 ? "danger" : "info",
      category: "Mendekati Hari H",
      title: `H-${d} menuju hari H`,
      description: `${project.name} · ${formatDate(project.eventDate)}`,
      to: ROUTE_PATHS.projectDetail(project.id),
    });
  }

  for (const row of stats.laggingProjects) {
    items.push({
      id: `lag-${row.project.id}`,
      tone: "warning",
      category: "Progress Tertinggal",
      title: "Progress persiapan tertinggal dari rencana",
      description: `${row.project.name} · ${row.overallPercent}% milestone selesai, H-${daysUntil(row.project.eventDate)}`,
      to: ROUTE_PATHS.projectDetail(row.project.id),
    });
  }

  items.sort((a, b) => TONE_RANK[a.tone] - TONE_RANK[b.tone]);

  const counts: Record<string, number> = {};
  for (const item of items) counts[item.category] = (counts[item.category] ?? 0) + 1;

  return { items, counts };
}
