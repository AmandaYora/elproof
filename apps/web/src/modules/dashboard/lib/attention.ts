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
      to: ROUTE_PATHS.projectDetail(t.projectId, "keuangan/klien"),
    });
  }

  for (const issue of stats.openIssues) {
    items.push({
      id: `issue-${issue.id}`,
      tone: issue.impact === "Critical" || issue.impact === "High" ? "danger" : "warning",
      group: "kendala",
      category: "Kendala aktif",
      title: `Kendala dampak ${ISSUE_IMPACT_LABELS.staff[issue.impact].toLowerCase()}: ${issue.title}`,
      description: `${issue.projectName}${issue.supplierName ? ` · ${issue.supplierName}` : ""}`,
      to: ROUTE_PATHS.projectDetail(issue.projectId, "kendala"),
    });
  }

  for (const v of stats.supplierPaymentsDue ?? []) {
    const d = daysUntil(v.dueDate);
    if (d >= 0) continue;
    items.push({
      id: `supplierdue-${v.projectId}-${v.supplierKind}-${v.supplierName}-${v.dueDate}`,
      tone: "warning",
      group: "lainnya",
      category: `Pembayaran ${v.supplierKind} terlambat`,
      title: `Pembayaran ke ${v.supplierName || v.supplierKind} terlambat ${Math.abs(d)} hari`,
      description: `${v.projectName} · sisa ${formatCurrency(v.remaining)}, jatuh tempo ${formatDate(v.dueDate)}`,
      to: ROUTE_PATHS.projectDetail(v.projectId, "keuangan/vendor-venue"),
    });
  }

  for (const m of stats.overdueVendorMilestones) {
    items.push({
      id: `vm-${m.id}`,
      tone: "warning",
      group: "lainnya",
      category: "Tahapan terlambat",
      title: `Tahapan "${m.name}" terlambat`,
      description: `${m.projectName}${m.supplierName ? ` · ${m.supplierName}` : ""} · target ${formatDate(m.targetDate)}`,
      to: ROUTE_PATHS.projectDetail(m.projectId, "vendor"),
    });
  }

  // One feed for vendors and venues, whoever paid: a payment the client made straight to a supplier lives in
  // Keuangan › Dibayar klien, the WO's own in Keuangan › Vendor & venue.
  for (const p of stats.incompletePayments) {
    items.push({
      id: `pay-${p.id}`,
      tone: "warning",
      group: "bukti",
      category: "Bukti pembayaran",
      title: `Pembayaran ${p.type} belum memiliki bukti lengkap`,
      description: `${p.projectName} · ${p.supplierName || p.supplierKind} · ${formatCurrency(p.amount)}`,
      to: ROUTE_PATHS.projectDetail(p.projectId, p.payer === "Client" ? "keuangan/dibayar-klien" : "keuangan/vendor-venue"),
    });
  }

  // The couple's proofs of paying a vendor they pay directly (Paket WO Tahap 4): the link opens the check itself.
  for (const p of stats.supplierProofsPending ?? []) {
    items.push({
      id: `supplierproof-${p.supplierPaymentId}`,
      tone: "warning",
      group: "bukti",
      category: "Bukti bayar vendor",
      title: `Bukti bayar ${p.supplierName || p.supplierKind} dari pasangan menunggu diperiksa`,
      description: `${p.projectName} · ${p.billTermLabel || p.type} · ${formatCurrency(p.amount)}`,
      to: `${ROUTE_PATHS.projectDetail(p.projectId, "keuangan/dibayar-klien")}?periksa=${encodeURIComponent(p.supplierPaymentId)}`,
    });
  }

  // Bills the couple pays a vendor directly, late or due within 7 days: the WO reminds them (never WO cash).
  for (const b of stats.clientBillsDue ?? []) {
    const d = daysUntil(b.dueDate);
    const late = b.status === "overdue" || d < 0;
    const name = b.supplierName || b.supplierKind;
    items.push({
      id: `clientbill-${b.billTermId}`,
      tone: late ? "warning" : "info",
      group: "tagihan",
      category: "Tagihan vendor",
      title: late
        ? `Tagihan ${name} (${b.label}) lewat jatuh tempo ${Math.abs(d)} hari`
        : `Tagihan ${name} (${b.label}) jatuh tempo ${d === 0 ? "hari ini" : formatDate(b.dueDate)}`,
      description: `${b.projectName} · dibayar klien langsung, sisa ${formatCurrency(b.remaining)}${b.pending > 0 ? `, ${formatCurrency(b.pending)} menunggu diperiksa` : ""}`,
      to: ROUTE_PATHS.projectDetail(b.projectId, "keuangan/dibayar-klien"),
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

  for (const v of stats.unconfirmedSuppliers) {
    items.push({
      id: `unconfirmed-${v.projectId}-${v.supplierKind}-${v.supplierName}`,
      tone: "warning",
      group: "lainnya",
      category: v.supplierKind === "venue" ? "Venue belum pasti" : "Vendor belum pasti",
      title: `${v.supplierName} belum terkonfirmasi`,
      description: `${v.projectName} · acara ${formatDate(v.eventDate)} (${formatDayCountdown(daysUntil(v.eventDate))})`,
      to: ROUTE_PATHS.projectDetail(v.projectId, "vendor"),
    });
  }

  const counts: Record<AttentionGroup, number> = { tagihan: 0, kendala: 0, bukti: 0, lainnya: 0 };
  for (const item of items) counts[item.group] += 1;

  return { items, counts };
}
