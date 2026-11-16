import { useEffect, useState } from "react";
import { EmptyState } from "@/shared/components/feedback/EmptyState";
import { Pagination } from "@/shared/components/ui/Pagination";
import { useProjectStore } from "@/modules/projects/stores/useProjectStore";
import { ActivityList } from "@/modules/projects/components/detail/ActivityList";

/**
 * The project's whole history, page by page from the server (the actor's name comes with each entry). Opened from the
 * Riwayat panel on Ringkasan, inside a drawer that carries the title.
 */
export function ProjectActivitySection({ projectId }: { projectId: string }) {
  const activity = useProjectStore((s) => s.activity);
  const meta = useProjectStore((s) => s.activityMeta);
  const fetchActivity = useProjectStore((s) => s.fetchActivity);
  const [page, setPage] = useState(1);

  useEffect(() => {
    void fetchActivity(projectId, page);
  }, [projectId, page, fetchActivity]);

  if (activity.length === 0) {
    return <EmptyState title="Belum ada riwayat" description="Perubahan pada proyek ini tercatat di sini secara otomatis." />;
  }
  return (
    <div className="flex flex-col gap-4">
      <ActivityList entries={activity} />
      <Pagination page={meta.page} totalPages={meta.totalPages} totalItems={meta.total} pageSize={meta.limit} onPageChange={setPage} />
    </div>
  );
}
