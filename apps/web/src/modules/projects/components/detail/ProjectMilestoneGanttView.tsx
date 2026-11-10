import { Gantt, Willow, type ITask } from "@svar-ui/react-gantt";
import "@svar-ui/react-gantt/all.css";
import "./gantt-theme.css";
import { isMilestoneOverdue } from "@/modules/projects/lib/dates";
import { MILESTONE_STATUS_LABELS } from "@/shared/lib/labels";
import type { ProjectMilestone } from "@/modules/projects/types";

function parseISODate(dateISO: string): Date {
  const [y, m, d] = dateISO.split("-").map(Number);
  return new Date(y, (m ?? 1) - 1, d ?? 1);
}

function statusLabel(status: unknown): string {
  return MILESTONE_STATUS_LABELS.staff[status as keyof typeof MILESTONE_STATUS_LABELS.staff] ?? String(status);
}

function sortForGantt(milestones: ProjectMilestone[]): ProjectMilestone[] {
  return [...milestones].sort((a, b) => a.order - b.order);
}

// Every Timeline item is a single point in time (targetDate, or completedDate
// once done) rather than a date range, so each one maps to a "milestone"
// task (a diamond marker), never a duration bar — see PLAN.md
// revisi-timeline-vendor-role-sales. Status is surfaced via the grid's
// Status column (colored text) rather than the diamond's own fill, since the
// library doesn't expose a documented per-task bar-color hook.
export function ProjectMilestoneGanttView({ milestones }: { milestones: ProjectMilestone[] }) {
  const tasks: ITask[] = sortForGantt(milestones).map((m) => ({
    id: m.id,
    text: `${m.order}. ${m.name}`,
    start: parseISODate(m.completedDate ?? m.targetDate),
    type: "milestone",
    progress: m.status === "Completed" ? 100 : 0,
    status: m.status,
    overdue: isMilestoneOverdue(m.status, m.targetDate),
  }));

  if (tasks.length === 0) {
    return (
      <p className="rounded-md border border-dashed border-border px-4 py-6 text-center text-[13px] text-text-secondary">
        Belum ada tahapan untuk ditampilkan.
      </p>
    );
  }

  return (
    // Below about 900px the library drops into a one-pane mode and hides the timeline, so a narrow card scrolls sideways instead.
    <div className="h-[480px] overflow-x-auto overflow-y-hidden rounded-md border border-border" role="region" aria-label="Jadwal proyek dalam bentuk garis waktu" tabIndex={0}>
      <Willow fonts={false}>
        <div className="milestone-gantt wx-willow-theme h-full min-w-[900px]">
      <Gantt
        tasks={tasks}
        readonly
        columns={[
          // A fixed grid leaves room for the timeline beside it; with a flexible one the grid took the whole width.
          { id: "text", header: "Tahapan", width: 220 },
          {
            id: "status",
            header: "Status",
            width: 120,
            cell: ({ row }: { row: ITask }) => (
              <span
                className={
                  row.overdue
                    ? "font-semibold text-danger-text"
                    : row.status === "Completed"
                      ? "font-medium text-success-text"
                      : "text-text-secondary"
                }
              >
                {row.overdue ? `${statusLabel(row.status)}, terlambat` : statusLabel(row.status)}
              </span>
            ),
          },
        ]}
      />
        </div>
      </Willow>
    </div>
  );
}
