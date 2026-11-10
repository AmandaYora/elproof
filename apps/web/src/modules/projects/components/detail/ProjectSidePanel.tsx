import { Link } from "react-router-dom";
import { Card, CardContent } from "@/shared/components/ui/Card";
import { ProjectStatusBadge } from "@/modules/projects/components/StatusBadges";
import { useProjectStore } from "@/modules/projects/stores/useProjectStore";
import { formatCurrency, formatDate } from "@/shared/lib/formatters";
import { ROUTE_PATHS } from "@/app/routes/route-paths";

/**
 * Right-hand panel of the project page: the few facts needed at a glance on every tab.
 * Every figure comes from the server's overview (E1), so the panel needs no calls of its own and agrees with the header.
 */
export function ProjectSidePanel({ projectId }: { projectId: string }) {
  const project = useProjectStore((s) => s.currentProject);
  const overview = useProjectStore((s) => s.overview);
  if (!project || !overview) return null;

  const { finance, team, openIssueCount, clients } = overview;
  const progress = project.progress;

  return (
    <aside aria-label="Ringkasan proyek" className="flex flex-col gap-4 lg:sticky lg:top-20 lg:self-start">
      <Card>
        <CardContent className="flex flex-col gap-4 py-4">
          <PanelItem label="Tahap">
            <div className="flex items-center gap-2">
              <ProjectStatusBadge status={project.status} />
              {progress && <span className="text-[13px] font-semibold tabular-nums text-primary">{progress.overallPercent}% jadwal selesai</span>}
            </div>
          </PanelItem>

          <PanelItem label="Sisa tagihan klien">
            <p className="text-[15px] font-bold tabular-nums text-primary">{formatCurrency(finance.remaining)}</p>
            {finance.nextTerm && (
              <p className="text-[12.5px] text-text-secondary">
                {finance.nextTerm.label} jatuh tempo {formatDate(finance.nextTerm.dueDate)}
              </p>
            )}
          </PanelItem>

          <PanelItem label="Kendala terbuka">
            <Link
              to={ROUTE_PATHS.projectDetail(projectId, "kendala")}
              className={`text-[13.5px] font-semibold underline-offset-2 hover:underline ${openIssueCount > 0 ? "text-danger-text" : "text-text-primary"}`}
            >
              {openIssueCount > 0 ? `${openIssueCount} kendala` : "Tidak ada"}
            </Link>
          </PanelItem>

          <PanelItem label="Tim">
            <p className="text-[13px] text-text-primary">
              <span className="text-text-secondary">Planner: </span>
              {team.picName || "Belum ditugaskan"}
            </p>
            <p className="text-[13px] text-text-primary">
              <span className="text-text-secondary">Sales: </span>
              {team.picSalesName || "Belum ditugaskan"}
            </p>
          </PanelItem>

          <Link to={ROUTE_PATHS.projectDetail(projectId, "client")} className="text-[13px] font-medium text-primary underline-offset-2 hover:underline">
            {clients.length > 0 ? `Kelola akses klien (${clients.length})` : "Undang klien"}
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
