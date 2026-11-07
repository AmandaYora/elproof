import { useEffect, useState } from "react";
import { useCan } from "@/shared/stores/usePermissionStore";
import { PROJECT_STATUS_LABELS } from "@/shared/lib/labels";
import { Plus } from "lucide-react";
import { Button } from "@/shared/components/ui/Button";
import { SearchInput } from "@/shared/components/ui/SearchInput";
import { Select } from "@/shared/components/ui/Input";
import { Pagination } from "@/shared/components/ui/Pagination";
import { EmptyState } from "@/shared/components/feedback/EmptyState";
import { LoadError } from "@/shared/components/feedback/LoadError";
import { SkeletonRows } from "@/shared/components/ui/Skeleton";
import { ProjectCard } from "@/modules/projects/components/ProjectCard";
import { PROJECT_STATUS_OPTIONS } from "@/modules/projects/schemas/project.schema";
import { useProjectStore } from "@/modules/projects/stores/useProjectStore";
import { useNavigate } from "react-router-dom";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import { useDebouncedValue } from "@/shared/hooks/useDebouncedValue";

export default function ProjectListPage() {
  // Wedding Planner ("Staff" role) never creates a project — only Owner/Admin
  // do, then assign it to a planner (see PLAN.md's RBAC section). The
  // backend already rejects POST /projects for this role; hiding the button
  // too avoids a confusing 403 after filling out the whole form.
  const canCreate = useCan("project.create");
  const projects = useProjectStore((s) => s.projectPage);
  const meta = useProjectStore((s) => s.projectPageMeta);
  const fetchProjectPage = useProjectStore((s) => s.fetchProjectPage);

  const [query, setQuery] = useState("");
  const debouncedQuery = useDebouncedValue(query);
  const [statusFilter, setStatusFilter] = useState<string>("Semua");
  const [showArchived, setShowArchived] = useState(false);
  const [page, setPage] = useState(1);
  const navigate = useNavigate();

  useEffect(() => {
    setPage(1);
  }, [debouncedQuery, statusFilter, showArchived]);

  const [loading, setLoading] = useState(true);
  const [loadFailed, setLoadFailed] = useState(false);
  const [attempt, setAttempt] = useState(0);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setLoadFailed(false);
    fetchProjectPage(page, debouncedQuery, statusFilter === "Semua" ? "" : statusFilter, showArchived)
      .catch(() => !cancelled && setLoadFailed(true))
      .finally(() => !cancelled && setLoading(false));
    return () => {
      cancelled = true;
    };
  }, [fetchProjectPage, page, debouncedQuery, statusFilter, showArchived, attempt]);

  return (
    <div className="flex flex-col gap-5">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-xl font-bold text-text-primary">Project</h1>
          <p className="mt-1 text-[13px] text-text-secondary">Kelola seluruh project pernikahan yang sedang dan pernah ditangani.</p>
        </div>
        {canCreate && (
          <Button icon={<Plus className="h-4 w-4" />} onClick={() => navigate(ROUTE_PATHS.projectNew)}>
            Tambah Project
          </Button>
        )}
      </div>

      <div className="flex flex-wrap gap-3">
        <SearchInput
          className="max-w-xs"
          placeholder="Cari nama project, pasangan, atau venue..."
          value={query}
          onChange={(e) => setQuery(e.target.value)}
        />
        <Select className="w-48" value={statusFilter} onChange={(e) => setStatusFilter(e.target.value)}>
          <option value="Semua">Semua Status</option>
          {PROJECT_STATUS_OPTIONS.map((s) => (
            <option key={s} value={s}>{PROJECT_STATUS_LABELS.staff[s]}</option>
          ))}
        </Select>
        <label className="flex items-center gap-2 rounded-md border border-border px-3 text-[13px] text-text-secondary">
          <input type="checkbox" checked={showArchived} onChange={(e) => setShowArchived(e.target.checked)} />
          Tampilkan yang diarsipkan
        </label>
      </div>

      {loadFailed ? (
        <LoadError message="Daftar project belum bisa dimuat. Periksa koneksi Anda, lalu coba lagi." onRetry={() => setAttempt((n) => n + 1)} />
      ) : loading && projects.length === 0 ? (
        <div className="rounded-lg border border-border bg-surface p-5"><SkeletonRows rows={4} label="Memuat daftar project" /></div>
      ) : projects.length === 0 ? (
        <div className="rounded-lg border border-border bg-surface">
          <EmptyState
            title={showArchived ? "Tidak ada project yang diarsipkan" : "Tidak ada project ditemukan"}
            description="Ubah kata kunci pencarian atau filter status."
          />
        </div>
      ) : (
        <>
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-3">
            {projects.map((project) => (
              <ProjectCard key={project.id} project={project} />
            ))}
          </div>
          <div className="rounded-lg border border-border bg-surface">
            <Pagination page={meta.page} totalPages={meta.totalPages} totalItems={meta.total} pageSize={meta.limit} onPageChange={setPage} />
          </div>
        </>
      )}

    </div>
  );
}
