import { Link } from "react-router-dom";
import { Card, CardHeader } from "@/shared/components/ui/Card";
import type { ActivityLogEntry } from "@/modules/projects/types";
import { formatDateTime } from "@/shared/lib/formatters";
import { activityText } from "@/shared/lib/labels";
import { ROUTE_PATHS } from "@/app/routes/route-paths";

/** The five latest changes across the workspace; a row about a project opens that project's history. */
export function RecentActivity({ activity }: { activity: ActivityLogEntry[] }) {
  const rows = activity.slice(0, 5);
  return (
    <Card>
      <CardHeader title="Aktivitas terbaru" />
      {rows.length === 0 ? (
        <p className="px-5 py-5 text-[13px] text-muted-foreground">Belum ada aktivitas. Perubahan pada proyek, pembayaran, dan kendala akan tercatat di sini.</p>
      ) : (
        <ul className="divide-y divide-border-light">
          {rows.map((entry) => {
            const row = (
              <span className="block px-5 py-2.5 sm:flex sm:items-baseline sm:justify-between sm:gap-4">
                <span className="line-clamp-2 min-w-0 text-[13px] text-foreground sm:line-clamp-1">
                  <span className="font-semibold">{entry.actorName || "Sistem"}</span> {activityText(entry.description)}
                </span>
                <span className="mt-0.5 block shrink-0 text-[12px] tabular-nums text-muted-foreground sm:mt-0">{formatDateTime(entry.timestamp)}</span>
              </span>
            );
            return (
              <li key={entry.id}>
                {entry.projectId ? (
                  <Link
                    to={ROUTE_PATHS.projectDetail(entry.projectId, "aktivitas")}
                    className="block hover:bg-muted focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-solid focus-visible:outline-ring"
                  >
                    {row}
                  </Link>
                ) : (
                  row
                )}
              </li>
            );
          })}
        </ul>
      )}
    </Card>
  );
}
