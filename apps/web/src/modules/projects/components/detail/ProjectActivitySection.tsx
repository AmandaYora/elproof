import { useEffect } from "react";
import { Card, CardHeader, CardContent } from "@/shared/components/ui/Card";
import { EmptyState } from "@/shared/components/feedback/EmptyState";
import { useProjectStore } from "@/modules/projects/stores/useProjectStore";
import { useStaffStore } from "@/modules/users/stores/useStaffStore";
import { formatDateTime } from "@/shared/lib/formatters";

export function ProjectActivitySection({ projectId }: { projectId: string }) {
  const activity = useProjectStore((s) => s.activity);
  const fetchActivity = useProjectStore((s) => s.fetchActivity);
  const staff = useStaffStore((s) => s.staff);
  const fetchStaff = useStaffStore((s) => s.fetchStaff);

  useEffect(() => {
    void fetchActivity(projectId);
    void fetchStaff();
  }, [projectId, fetchActivity, fetchStaff]);

  return (
    <div id="aktivitas">
      <Card>
        <CardHeader title="Aktivitas Terbaru" subtitle="Riwayat aktivitas otomatis yang tercatat pada project ini." />
        <CardContent>
          {activity.length === 0 ? (
            <EmptyState
              title="Belum ada aktivitas"
              description="Aktivitas akan tercatat otomatis seiring perubahan pada project ini."
            />
          ) : (
            <ul className="flex flex-col">
              {activity.map((entry, index) => {
                const actor = staff.find((s) => s.id === entry.actorStaffId);
                const isLast = index === activity.length - 1;
                return (
                  <li key={entry.id} className="flex gap-3">
                    <div className="flex flex-col items-center">
                      <span className="mt-1.5 h-2.5 w-2.5 shrink-0 rounded-full bg-navy-900" />
                      {!isLast && <span className="w-px flex-1 bg-border" />}
                    </div>
                    <div className="pb-5">
                      <p className="text-[13.5px] text-text-primary">
                        <strong className="font-semibold">{actor?.name ?? "Sistem"}</strong> {entry.description}
                      </p>
                      <p className="mt-0.5 text-[12px] text-text-secondary">{formatDateTime(entry.timestamp)}</p>
                    </div>
                  </li>
                );
              })}
            </ul>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
