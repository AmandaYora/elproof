import { useEffect, useMemo } from "react";
import { Link, useOutletContext } from "react-router-dom";
import { CircleCheck } from "lucide-react";
import { Card, CardHeader, CardContent } from "@/shared/components/ui/Card";
import { Badge } from "@/shared/components/ui/Badge";
import { EmptyState } from "@/shared/components/feedback/EmptyState";
import { MilestoneStatusBadge } from "@/modules/projects/components/StatusBadges";
import { useProjectStore } from "@/modules/projects/stores/useProjectStore";
import { useVendorStore } from "@/modules/vendors/stores/useVendorStore";
import { daysUntil } from "@/modules/projects/lib/dates";
import { TERM_STATUS_TONE } from "@/modules/projects/lib/status";
import { TERM_STATUS_LABELS } from "@/shared/lib/labels";
import { formatCurrency, formatDate } from "@/shared/lib/formatters";
import { ROUTE_PATHS, type ProjectDetailTab } from "@/app/routes/route-paths";
import type { ProjectDetailContext } from "@/modules/projects/pages/ProjectDetailLayout";

interface ActionItem {
  id: string;
  tone: "danger" | "warning";
  title: string;
  detail: string;
  tab: ProjectDetailTab;
}

const OPEN_ISSUE = new Set(["Open", "In Review", "In Resolution"]);

function relativeDay(dateISO: string): string {
  const d = daysUntil(dateISO);
  if (d === 0) return "hari ini";
  return d > 0 ? `H-${d}` : `terlambat ${Math.abs(d)} hari`;
}

/** First tab of a project: what needs doing now, what comes next. */
export default function ProjectSummaryTabPage() {
  const { projectId } = useOutletContext<ProjectDetailContext>();
  const project = useProjectStore((s) => s.currentProject);
  const milestones = useProjectStore((s) => s.milestones);
  const engagements = useProjectStore((s) => s.vendorEngagements);
  const issues = useProjectStore((s) => s.issues);
  const terms = useProjectStore((s) => s.paymentTerms);
  const fetchIssues = useProjectStore((s) => s.fetchIssues);
  const fetchPaymentTerms = useProjectStore((s) => s.fetchPaymentTerms);
  const vendors = useVendorStore((s) => s.vendors);
  const fetchVendors = useVendorStore((s) => s.fetchVendors);

  useEffect(() => {
    void fetchIssues(projectId);
    void fetchPaymentTerms(projectId);
    void fetchVendors();
  }, [projectId, fetchIssues, fetchPaymentTerms, fetchVendors]);

  const actions = useMemo<ActionItem[]>(() => {
    const items: ActionItem[] = [];
    for (const t of terms) {
      if (t.status === "overdue") {
        items.push({
          id: `term-${t.id}`,
          tone: "danger",
          title: `Termin ${t.label} terlambat`,
          detail: `Sisa ${formatCurrency(t.remaining)}, jatuh tempo ${formatDate(t.dueDate)}`,
          tab: "pembayaran/client",
        });
      }
    }
    for (const i of issues) {
      if (OPEN_ISSUE.has(i.status)) {
        items.push({
          id: `issue-${i.id}`,
          tone: i.impact === "High" || i.impact === "Critical" ? "danger" : "warning",
          title: i.title,
          detail: "Kendala masih terbuka",
          tab: "kendala",
        });
      }
    }
    // A vendor still only "planned" or "negotiating" a month before the event.
    const daysToEvent = project ? daysUntil(project.eventDate) : null;
    if (daysToEvent !== null && daysToEvent >= 0 && daysToEvent <= 30) {
      for (const e of engagements) {
        if (e.engagementStatus === "Planned" || e.engagementStatus === "Negotiation") {
          const name = vendors.find((v) => v.id === e.vendorId)?.name ?? "Vendor";
          items.push({
            id: `vendor-${e.id}`,
            tone: "warning",
            title: `${name} belum terkonfirmasi`,
            detail: `Acara H-${daysToEvent}, status masih belum dipesan`,
            tab: "vendor",
          });
        }
      }
    }
    for (const m of milestones) {
      if (m.status !== "Completed" && m.status !== "Cancelled" && daysUntil(m.targetDate) < 0) {
        items.push({ id: `ms-${m.id}`, tone: "warning", title: `Timeline "${m.name}" lewat tenggat`, detail: `Target ${formatDate(m.targetDate)}`, tab: "milestone" });
      }
    }
    return items.sort((a, b) => Number(b.tone === "danger") - Number(a.tone === "danger"));
  }, [terms, issues, engagements, milestones, vendors, project]);

  const nextMilestones = useMemo(
    () =>
      milestones
        .filter((m) => m.status !== "Completed" && m.status !== "Cancelled")
        .sort((a, b) => a.targetDate.localeCompare(b.targetDate))
        .slice(0, 5),
    [milestones],
  );
  const nextTerm = terms.filter((t) => t.status !== "paid").sort((a, b) => a.dueDate.localeCompare(b.dueDate))[0];

  return (
    <div className="flex flex-col gap-5">
      <Card>
        <CardHeader title="Perlu tindakan" subtitle="Yang sebaiknya ditangani lebih dulu." />
        <CardContent>
          {actions.length === 0 ? (
            <EmptyState
              icon={<CircleCheck className="h-6 w-6 text-success" />}
              title="Tidak ada yang perlu ditindak"
              description="Termin, kendala, vendor, dan timeline proyek ini sejauh ini berjalan sesuai rencana."
            />
          ) : (
            <ul className="flex flex-col divide-y divide-border-light">
              {actions.map((a) => (
                <li key={a.id}>
                  <Link
                    to={ROUTE_PATHS.projectDetail(projectId, a.tab)}
                    className="flex items-start gap-3 py-3 first:pt-0 last:pb-0 hover:bg-surface-muted/50 focus-visible:outline focus-visible:outline-2 focus-visible:outline-navy-900"
                  >
                    <span aria-hidden className={`mt-1.5 h-2 w-2 shrink-0 rounded-full ${a.tone === "danger" ? "bg-danger" : "bg-warning"}`} />
                    <span className="min-w-0">
                      <span className="block text-[13.5px] font-semibold text-text-primary">{a.title}</span>
                      <span className="block text-[12.5px] text-text-secondary">{a.detail}</span>
                    </span>
                  </Link>
                </li>
              ))}
            </ul>
          )}
        </CardContent>
      </Card>

      <div className="grid grid-cols-1 gap-5 xl:grid-cols-2">
        <Card>
          <CardHeader title="Timeline berikutnya" subtitle="Lima item terdekat yang belum selesai." />
          <CardContent>
            {nextMilestones.length === 0 ? (
              <p className="text-[13px] text-text-secondary">Semua item timeline sudah selesai.</p>
            ) : (
              <ul className="flex flex-col divide-y divide-border-light">
                {nextMilestones.map((m) => (
                  <li key={m.id} className="flex items-center justify-between gap-3 py-2.5 first:pt-0 last:pb-0">
                    <div className="min-w-0">
                      <p className="truncate text-[13.5px] font-medium text-text-primary">{m.name}</p>
                      <p className="text-[12.5px] text-text-secondary">
                        {formatDate(m.targetDate)} · {relativeDay(m.targetDate)}
                      </p>
                    </div>
                    <MilestoneStatusBadge status={m.status} />
                  </li>
                ))}
              </ul>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader title="Termin berikutnya" subtitle="Tagihan klien yang belum lunas." />
          <CardContent>
            {nextTerm ? (
              <div className="flex flex-wrap items-center justify-between gap-3">
                <div>
                  <p className="text-[14px] font-semibold text-text-primary">{nextTerm.label}</p>
                  <p className="text-[12.5px] text-text-secondary">
                    Jatuh tempo {formatDate(nextTerm.dueDate)} · {relativeDay(nextTerm.dueDate)}
                  </p>
                  <p className="mt-1 text-[15px] font-bold tabular-nums text-navy-900">{formatCurrency(nextTerm.remaining)}</p>
                </div>
                <Badge tone={TERM_STATUS_TONE[nextTerm.status]}>{TERM_STATUS_LABELS.staff[nextTerm.status]}</Badge>
              </div>
            ) : terms.length === 0 ? (
              <p className="text-[13px] text-text-secondary">
                Belum ada jadwal termin.{" "}
                <Link to={ROUTE_PATHS.projectDetail(projectId, "pembayaran/client")} className="font-medium text-navy-900 underline-offset-2 hover:underline">
                  Buat jadwal termin
                </Link>
              </p>
            ) : (
              <p className="text-[13px] text-text-secondary">Semua termin sudah lunas.</p>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
