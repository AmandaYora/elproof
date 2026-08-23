import { Gantt, type ITask } from "@svar-ui/react-gantt";
import "@svar-ui/react-gantt/all.css";
import { isMilestoneOverdue } from "@/modules/projects/lib/dates";
import type { ProjectMilestone } from "@/modules/projects/types";

function parseISODate(dateISO: string): Date {
  const [y, m, d] = dateISO.split("-").map(Number);
  return new Date(y, (m ?? 1) - 1, d ?? 1);
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
        Belum ada timeline untuk ditampilkan.
      </p>
    );
  }

  return (
    <div className="h-[480px] overflow-hidden rounded-md border border-border">
      <Gantt
        tasks={tasks}
        readonly
        columns={[
          { id: "text", header: "Timeline", flexGrow: 2 },
          {
            id: "status",
            header: "Status",
            width: 130,
            cell: ({ row }: { row: ITask }) => (
              <span
                className={
                  row.overdue
                    ? "font-semibold text-danger"
                    : row.status === "Completed"
                      ? "font-medium text-success"
                      : "text-text-secondary"
                }
              >
                {row.overdue ? `${row.status} · terlambat` : String(row.status)}
              </span>
            ),
          },
        ]}
      />
    </div>
  );
}
