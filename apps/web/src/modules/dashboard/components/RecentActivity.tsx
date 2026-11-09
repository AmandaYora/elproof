import { useEffect } from "react";
import { Link } from "react-router-dom";
import { Card, CardHeader, CardContent } from "@/shared/components/ui/Card";
import type { ActivityLogEntry } from "@/modules/projects/types";
import { useStaffStore } from "@/modules/users/stores/useStaffStore";
import { formatDateTime } from "@/shared/lib/formatters";
import { ROUTE_PATHS } from "@/app/routes/route-paths";

export function RecentActivity({ activity }: { activity: ActivityLogEntry[] }) {
  const staff = useStaffStore((s) => s.staffSummaries);
  const fetchStaffSummaries = useStaffStore((s) => s.fetchStaffSummaries);

  // The page never loaded staff names itself, so on a first visit every actor
  // read "Sistem". Load them here (the API will return actor names directly: PLAN E12).
  useEffect(() => {
    if (staff.length === 0) void fetchStaffSummaries().catch(() => undefined);
  }, [staff.length, fetchStaffSummaries]);

  return (
    <Card>
      <CardHeader title="Aktivitas terbaru" subtitle="Riwayat perubahan penting di seluruh proyek." />
      <CardContent className="p-0">
        <ul className="divide-y divide-border-light">
          {activity.map((entry) => {
            const actor = staff.find((s) => s.id === entry.actorStaffId);
            const row = (
              <span className="block px-5 py-3">
                <span className="block text-[13px] text-text-primary">
                  <span className="font-semibold">{actor?.name ?? "Sistem"}</span> {entry.description}
                </span>
                <span className="mt-0.5 block text-[12px] text-text-secondary">{formatDateTime(entry.timestamp)}</span>
              </span>
            );
            return (
              <li key={entry.id}>
                {entry.projectId ? (
                  <Link to={ROUTE_PATHS.projectDetail(entry.projectId, "aktivitas")} className="block hover:bg-surface-muted">
                    {row}
                  </Link>
                ) : (
                  row
                )}
              </li>
            );
          })}
        </ul>
      </CardContent>
    </Card>
  );
}
