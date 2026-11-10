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

const TONE_RANK: Record<AttentionTone, number> = { danger: 0, warning: 1, info: 2 };

export function buildAttentionItems(stats: DashboardStats): { items: AttentionItem[]; counts: Record<string, number> } {
  const items: AttentionItem[] = [];

  for (const t of stats.overdueTerms) {
    items.push({
      id: `term-${t.projectId}-${t.label}-${t.dueDate}`,
      tone: "danger",
      category: "Tagihan terlambat",
      title: `Termin ${t.label} terlambat ${Math.abs(daysUntil(t.dueDate))} hari`,
      description: `${t.projectName} · sisa ${formatCurrency(t.remaining)}, jatuh tempo ${formatDate(t.dueDate)}`,
      to: ROUTE_PATHS.projectDetail(t.projectId, "pembayaran/client"),
    });
  }

  for (const issue of stats.openIssues) {
    items.push({
      id: `issue-${issue.id}`,
      tone: issue.impact === "Critical" || issue.impact === "High" ? "danger" : "warning",
      category: "Kendala aktif",
      title: `Kendala ${issue.impact}: ${issue.title}`,
      description: `${issue.projectName}${issue.vendorName ? ` — ${issue.vendorName}` : ""}`,
      to: ROUTE_PATHS.projectDetail(issue.projectId, "vendor"),
    });
  }

  for (const m of stats.overdueVendorMilestones) {
    items.push({
      id: `vm-${m.id}`,
      tone: "warning",
      category: "Tahapan terlambat",
      title: `Tahapan "${m.name}" terlambat`,
      description: `${m.projectName} — ${m.vendorName} · target ${formatDate(m.targetDate)}`,
      to: ROUTE_PATHS.projectDetail(m.projectId, "vendor"),
    });
  }

  for (const p of stats.incompletePayments) {
    items.push({
      id: `pay-${p.id}`,
      tone: "warning",
      category: "Bukti pembayaran",
      title: `Pembayaran ${p.type} belum memiliki bukti lengkap`,
      description: `${p.projectName} — ${p.vendorName} · ${formatCurrency(p.amount)}`,
      to: ROUTE_PATHS.projectDetail(p.projectId, "pembayaran/vendor"),
    });
  }

  // Same category as vendor's own incomplete-payment items above -- the two
  // sources merge into one feed here (rather than a second widget), each row
  // labeled by which ledger it's from via its description instead of a
  // vendor name (a venue payment has none). See PLAN.md "Venue Payments +
  // Pembayaran tab restructuring".
  for (const p of stats.incompleteVenuePayments) {
    items.push({
      id: `venuepay-${p.id}`,
      tone: "warning",
      category: "Bukti pembayaran",
      title: `Pembayaran ${p.type} (venue) belum memiliki bukti lengkap`,
      description: `${p.projectName} — Venue · ${formatCurrency(p.amount)}`,
      to: ROUTE_PATHS.projectDetail(p.projectId, "pembayaran/venue"),
    });
  }

  for (const project of stats.nearDDayProjects) {
    const d = daysUntil(project.eventDate);
    items.push({
      id: `dday-${project.id}`,
      tone: d <= 7 ? "danger" : "info",
      category: "Mendekati hari H",
      title: `H-${d} menuju hari H`,
      description: `${project.name} · ${formatDate(project.eventDate)}`,
      to: ROUTE_PATHS.projectDetail(project.id),
    });
  }

  for (const row of stats.laggingProjects) {
    items.push({
      id: `lag-${row.project.id}`,
      tone: "warning",
      category: "Progres tertinggal",
      title: "Progress persiapan tertinggal dari rencana",
      description: `${row.project.name} · ${row.overallPercent}% tahapan selesai, H-${daysUntil(row.project.eventDate)}`,
      to: ROUTE_PATHS.projectDetail(row.project.id),
    });
  }

  items.sort((a, b) => TONE_RANK[a.tone] - TONE_RANK[b.tone]);

  for (const v of stats.unconfirmedVendors) {
    items.push({
      id: `unconfirmed-${v.projectId}-${v.vendorName}`,
      tone: "warning",
      category: "Vendor belum pasti",
      title: `${v.vendorName} belum terkonfirmasi`,
      description: `${v.projectName} · acara ${formatDate(v.eventDate)} (H-${Math.max(daysUntil(v.eventDate), 0)})`,
      to: ROUTE_PATHS.projectDetail(v.projectId, "vendor"),
    });
  }

  const counts: Record<string, number> = {};
  for (const item of items) counts[item.category] = (counts[item.category] ?? 0) + 1;

  return { items, counts };
}
