import { useEffect } from "react";
import { Link } from "react-router-dom";
import { Card, CardContent } from "@/shared/components/ui/Card";
import { ProjectStatusBadge } from "@/modules/projects/components/StatusBadges";
import { useProjectStore } from "@/modules/projects/stores/useProjectStore";
import { useStaffStore } from "@/modules/users/stores/useStaffStore";
import { remainingToPay } from "@/modules/projects/lib/payments";
import { formatCurrency, formatDate } from "@/shared/lib/formatters";
import { ROUTE_PATHS } from "@/app/routes/route-paths";

const OPEN_ISSUE = new Set(["Open", "In Review", "In Resolution"]);

/** Right-hand panel of the project page: the few facts needed at a glance on every tab. */
export function ProjectSidePanel({ projectId }: { projectId: string }) {
  const project = useProjectStore((s) => s.currentProject);
  const clientPayments = useProjectStore((s) => s.clientPayments);
  const terms = useProjectStore((s) => s.paymentTerms);
  const issues = useProjectStore((s) => s.issues);
  const fetchIssues = useProjectStore((s) => s.fetchIssues);
  const fetchPaymentTerms = useProjectStore((s) => s.fetchPaymentTerms);
  const staff = useStaffStore((s) => s.staffSummaries);

  useEffect(() => {
    void fetchIssues(projectId);
    void fetchPaymentTerms(projectId);
  }, [projectId, fetchIssues, fetchPaymentTerms, clientPayments]);

  if (!project) return null;
  const remaining = remainingToPay(project.contractValue, clientPayments);
  const nextTerm = terms.filter((t) => t.status !== "paid").sort((a, b) => a.dueDate.localeCompare(b.dueDate))[0];
  const openIssues = issues.filter((i) => OPEN_ISSUE.has(i.status)).length;
  const planner = staff.find((s) => s.id === project.picStaffId)?.name;
  const sales = staff.find((s) => s.id === project.picSalesStaffId)?.name;
  const progress = project.progress;

  return (
    <aside aria-label="Ringkasan project" className="flex flex-col gap-4 lg:sticky lg:top-20 lg:self-start">
      <Card>
        <CardContent className="flex flex-col gap-4 py-4">
          <PanelItem label="Tahap">
            <div className="flex items-center gap-2">
              <ProjectStatusBadge status={project.status} />
              {progress && <span className="text-[13px] font-semibold tabular-nums text-navy-900">{progress.overallPercent}% timeline</span>}
            </div>
          </PanelItem>

          <PanelItem label="Sisa tagihan klien">
            <p className="text-[15px] font-bold tabular-nums text-navy-900">
              {remaining < 0 ? `Lebih bayar ${formatCurrency(-remaining)}` : formatCurrency(remaining)}
            </p>
            {nextTerm && (
              <p className="text-[12.5px] text-text-secondary">
                {nextTerm.label} jatuh tempo {formatDate(nextTerm.dueDate)}
              </p>
            )}
          </PanelItem>

          <PanelItem label="Kendala terbuka">
            <Link
              to={ROUTE_PATHS.projectDetail(projectId, "kendala")}
              className={`text-[13.5px] font-semibold underline-offset-2 hover:underline ${openIssues > 0 ? "text-danger-text" : "text-text-primary"}`}
            >
              {openIssues > 0 ? `${openIssues} kendala` : "Tidak ada"}
            </Link>
          </PanelItem>

          <PanelItem label="Tim">
            <p className="text-[13px] text-text-primary">
              <span className="text-text-secondary">Planner: </span>
              {planner ?? "Belum ditugaskan"}
            </p>
            <p className="text-[13px] text-text-primary">
              <span className="text-text-secondary">Sales: </span>
              {sales ?? "Belum ditugaskan"}
            </p>
          </PanelItem>

          <Link
            to={ROUTE_PATHS.projectDetail(projectId, "client")}
            className="text-[13px] font-medium text-navy-900 underline-offset-2 hover:underline"
          >
            Kelola akses klien
          </Link>
        </CardContent>
      </Card>
    </aside>
  );
}

function PanelItem({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div>
      <p className="mb-1 text-[12px] font-medium text-text-secondary">{label}</p>
      {children}
    </div>
  );
}
