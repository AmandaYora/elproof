import { Link } from "react-router-dom";
import { MapPin } from "lucide-react";
import { Avatar } from "@/shared/components/ui/Avatar";
import { ProgressMeter } from "@/shared/components/ui/ProgressMeter";
import { ProjectStatusBadge, ConditionBadge } from "@/modules/projects/components/StatusBadges";
import { useStaffStore } from "@/modules/users/stores/useStaffStore";
import type { Project } from "@/modules/projects/types";
import { daysUntil } from "@/modules/projects/lib/dates";
import { formatCurrency, formatDate } from "@/shared/lib/formatters";
import { ROUTE_PATHS } from "@/app/routes/route-paths";

export function ProjectCard({ project }: { project: Project }) {
  const pic = useStaffStore((s) => s.staff.find((member) => member.id === project.picStaffId));
  const progress = project.progress;
  const d = daysUntil(project.eventDate);
  const isOpenProject = project.status !== "Completed" && project.status !== "Cancelled";

  const totalMilestones = (progress?.projectMilestoneStats.total ?? 0) + (progress?.vendorMilestoneStats.total ?? 0);
  const completedMilestones = (progress?.projectMilestoneStats.completed ?? 0) + (progress?.vendorMilestoneStats.completed ?? 0);
  const caption =
    totalMilestones === 0
      ? "Belum ada milestone"
      : `${completedMilestones}/${totalMilestones} milestone selesai${
          progress && progress.openIssueCount > 0 ? ` · ${progress.openIssueCount} kendala aktif` : ""
        }`;

  return (
    <Link
      to={ROUTE_PATHS.projectDetail(project.id)}
      className="flex flex-col gap-4 rounded-lg border border-border bg-surface p-5 transition-all hover:border-navy-900/30 hover:shadow-md"
    >
      <div className="flex items-center justify-between gap-2">
        <ProjectStatusBadge status={project.status} />
        {isOpenProject ? (
          <span className="rounded-full bg-navy-900/10 px-2.5 py-1 text-[11px] font-bold tabular-nums text-navy-900">
            {d >= 0 ? `H-${d}` : `H+${Math.abs(d)}`}
          </span>
        ) : (
          <span className="text-[11.5px] font-medium text-text-secondary">{formatDate(project.eventDate)}</span>
        )}
      </div>

      <div className="min-w-0">
        <h3 className="truncate text-[15px] font-bold text-text-primary">{project.name}</h3>
        <p className="mt-0.5 truncate text-[12.5px] text-text-secondary">{project.brideName} &amp; {project.groomName}</p>
        <p className="mt-1 flex items-center gap-1 truncate text-[12.5px] text-text-secondary">
          <MapPin className="h-3 w-3 shrink-0" /> {project.venue}
        </p>
      </div>

      {progress && (
        <div className="rounded-md bg-surface-muted/60 p-3">
          <div className="mb-2 flex items-center justify-between gap-2">
            <span className="text-2xl font-bold leading-none tabular-nums text-navy-900">{progress.overallPercent}%</span>
            <ConditionBadge condition={progress.condition} />
          </div>
          <ProgressMeter percent={progress.overallPercent} segments={[]} caption={caption} />
        </div>
      )}

      <div className="flex items-center justify-between gap-2 border-t border-border-light pt-3 text-[12.5px]">
        <span className="font-semibold tabular-nums text-text-primary">{formatCurrency(project.contractValue)}</span>
        {pic && (
          <span className="flex items-center gap-1.5 text-text-secondary">
            <Avatar name={pic.name} size="sm" />
            {pic.name}
          </span>
        )}
      </div>
    </Link>
  );
}
