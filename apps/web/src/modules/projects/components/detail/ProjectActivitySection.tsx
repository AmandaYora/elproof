import { useEffect, useState } from "react";
import { Card, CardHeader, CardContent } from "@/shared/components/ui/Card";
import { EmptyState } from "@/shared/components/feedback/EmptyState";
import { Pagination } from "@/shared/components/ui/Pagination";
import { useProjectStore } from "@/modules/projects/stores/useProjectStore";
import { formatDateTime } from "@/shared/lib/formatters";

/** Riwayat aktivitas, page by page from the server; the actor's name comes with each entry. */
export function ProjectActivitySection({ projectId }: { projectId: string }) {
  const activity = useProjectStore((s) => s.activity);
  const meta = useProjectStore((s) => s.activityMeta);
  const fetchActivity = useProjectStore((s) => s.fetchActivity);
  const [page, setPage] = useState(1);

  useEffect(() => {
    void fetchActivity(projectId, page);
  }, [projectId, page, fetchActivity]);

  return (
    <div id="aktivitas">
      <Card>
        <CardHeader title="Aktivitas terbaru" subtitle="Riwayat perubahan yang tercatat otomatis pada proyek ini." />
        <CardContent>
          {activity.length === 0 ? (
            <EmptyState title="Belum ada aktivitas" description="Aktivitas akan tercatat otomatis seiring perubahan pada proyek ini." />
          ) : (
            <>
              <ul className="flex flex-col">
                {activity.map((entry, index) => {
                  const isLast = index === activity.length - 1;
                  return (
                    <li key={entry.id} className="flex gap-3">
                      <div className="flex flex-col items-center">
                        <span className="mt-1.5 h-2.5 w-2.5 shrink-0 rounded-full bg-primary" />
                        {!isLast && <span className="w-px flex-1 bg-border" />}
                      </div>
                      <div className="pb-5">
                        <p className="text-[13.5px] text-text-primary">
                          <strong className="font-semibold">{entry.actorName || "Sistem"}</strong> {entry.description}
                        </p>
                        <p className="mt-0.5 text-[12px] text-text-secondary">{formatDateTime(entry.timestamp)}</p>
                      </div>
                    </li>
                  );
                })}
              </ul>
              <Pagination page={meta.page} totalPages={meta.totalPages} totalItems={meta.total} pageSize={meta.limit} onPageChange={setPage} />
            </>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
