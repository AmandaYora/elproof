import { useEffect, useState } from "react";
import { Plus } from "lucide-react";
import { Button } from "@/shared/components/ui/Button";
import { SearchInput } from "@/shared/components/ui/SearchInput";
import { Select } from "@/shared/components/ui/Input";
import { Pagination } from "@/shared/components/ui/Pagination";
import { EmptyState } from "@/shared/components/feedback/EmptyState";
import { ProjectFormModal } from "@/modules/projects/components/ProjectFormModal";
import { ProjectCard } from "@/modules/projects/components/ProjectCard";
import { PROJECT_STATUS_OPTIONS } from "@/modules/projects/schemas/project.schema";
import type { ProjectFormValues } from "@/modules/projects/schemas/project.schema";
import { useProjectStore } from "@/modules/projects/stores/useProjectStore";
import { getApiErrorMessage } from "@/shared/lib/api-error";

export default function ProjectListPage() {
  const projects = useProjectStore((s) => s.projectPage);
  const meta = useProjectStore((s) => s.projectPageMeta);
  const fetchProjectPage = useProjectStore((s) => s.fetchProjectPage);
  const createProject = useProjectStore((s) => s.createProject);

  const [query, setQuery] = useState("");
  const [statusFilter, setStatusFilter] = useState<string>("Semua");
  const [page, setPage] = useState(1);
  const [modalOpen, setModalOpen] = useState(false);
  const [actionError, setActionError] = useState<string | null>(null);

  useEffect(() => {
    setPage(1);
  }, [query, statusFilter]);

  useEffect(() => {
    void fetchProjectPage(page, query, statusFilter === "Semua" ? "" : statusFilter);
  }, [fetchProjectPage, page, query, statusFilter]);

  async function handleCreate(values: ProjectFormValues) {
    setActionError(null);
    try {
      await createProject(values);
      setModalOpen(false);
      await fetchProjectPage(page, query, statusFilter === "Semua" ? "" : statusFilter);
    } catch (err) {
      setActionError(getApiErrorMessage(err, "Gagal membuat project"));
    }
  }

  return (
    <div className="flex flex-col gap-5">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-xl font-bold text-text-primary">Project</h1>
          <p className="mt-1 text-[13px] text-text-secondary">Kelola seluruh project pernikahan yang sedang dan pernah ditangani.</p>
        </div>
        <Button icon={<Plus className="h-4 w-4" />} onClick={() => setModalOpen(true)}>
          Tambah Project
        </Button>
      </div>

      {actionError && (
        <p className="rounded-md border border-danger/30 bg-danger-soft px-3.5 py-2.5 text-[13px] font-medium text-danger">{actionError}</p>
      )}

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
            <option key={s} value={s}>{s}</option>
          ))}
        </Select>
      </div>

      {projects.length === 0 ? (
        <div className="rounded-lg border border-border bg-surface">
          <EmptyState title="Tidak ada project ditemukan" description="Ubah kata kunci pencarian atau filter status." />
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

      <ProjectFormModal open={modalOpen} onClose={() => setModalOpen(false)} onSubmit={(values) => void handleCreate(values)} />
    </div>
  );
}
