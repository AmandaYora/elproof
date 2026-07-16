import { useEffect, useState } from "react";
import { Pencil, AlertTriangle } from "lucide-react";
import { Card, CardContent } from "@/shared/components/ui/Card";
import { Button } from "@/shared/components/ui/Button";
import { ProgressMeter } from "@/shared/components/ui/ProgressMeter";
import { ProjectStatusBadge, ConditionBadge } from "@/modules/projects/components/StatusBadges";
import { ProjectFormModal } from "@/modules/projects/components/ProjectFormModal";
import type { ProjectFormValues } from "@/modules/projects/schemas/project.schema";
import { useProjectStore } from "@/modules/projects/stores/useProjectStore";
import { useStaffStore } from "@/modules/users/stores/useStaffStore";
import { daysUntil } from "@/modules/projects/lib/dates";
import { formatCurrency, formatDate } from "@/shared/lib/formatters";
import { getApiErrorMessage } from "@/shared/lib/api-error";

export function ProjectHeaderCard({ projectId }: { projectId: string }) {
  const project = useProjectStore((s) => s.currentProject);
  const milestones = useProjectStore((s) => s.milestones);
  const vendorMilestones = useProjectStore((s) => s.vendorMilestones);
  const fetchMilestones = useProjectStore((s) => s.fetchMilestones);
  const fetchVendorSection = useProjectStore((s) => s.fetchVendorSection);
  const updateProject = useProjectStore((s) => s.updateProject);
  const cancelProject = useProjectStore((s) => s.cancelProject);
  const staff = useStaffStore((s) => s.staff);
  const fetchStaff = useStaffStore((s) => s.fetchStaff);

  const [editOpen, setEditOpen] = useState(false);
  const [confirmingCancel, setConfirmingCancel] = useState(false);
  const [actionError, setActionError] = useState<string | null>(null);

  useEffect(() => {
    void fetchMilestones(projectId);
    void fetchVendorSection(projectId);
    void fetchStaff();
  }, [projectId, fetchMilestones, fetchVendorSection, fetchStaff]);

  if (!project) {
    return <div className="text-sm text-text-secondary">Project tidak ditemukan.</div>;
  }

  const progress = project.progress;
  const segments = [...milestones, ...vendorMilestones];
  const totalMilestones = (progress?.projectMilestoneStats.total ?? 0) + (progress?.vendorMilestoneStats.total ?? 0);
  const completedMilestones = (progress?.projectMilestoneStats.completed ?? 0) + (progress?.vendorMilestoneStats.completed ?? 0);
  const pic = staff.find((s) => s.id === project.picStaffId);
  const d = daysUntil(project.eventDate);
  const isOpenProject = project.status !== "Completed" && project.status !== "Cancelled";

  async function handleEdit(values: ProjectFormValues) {
    setActionError(null);
    try {
      await updateProject(projectId, values);
      setEditOpen(false);
    } catch (err) {
      setActionError(getApiErrorMessage(err, "Gagal menyimpan perubahan project"));
    }
  }

  async function handleCancelProject() {
    setActionError(null);
    try {
      await cancelProject(projectId);
      setConfirmingCancel(false);
    } catch (err) {
      setActionError(getApiErrorMessage(err, "Gagal membatalkan project"));
    }
  }

  return (
    <Card>
      <CardContent className="flex flex-col gap-5 py-5">
        <div className="flex flex-wrap items-start justify-between gap-4">
          <div>
            <div className="flex flex-wrap items-center gap-2">
              <h1 className="text-xl font-bold text-text-primary">{project.name}</h1>
              <ProjectStatusBadge status={project.status} />
            </div>
            <p className="mt-1 text-[13px] text-text-secondary">
              {project.brideName} &amp; {project.groomName} · {project.venue}
            </p>
          </div>
          <div className="flex shrink-0 items-center gap-2">
            <Button variant="secondary" size="sm" icon={<Pencil className="h-3.5 w-3.5" />} onClick={() => setEditOpen(true)}>
              Ubah Project
            </Button>
            {isOpenProject && (
              <Button variant="danger" size="sm" onClick={() => setConfirmingCancel(true)}>
                Batalkan Project
              </Button>
            )}
          </div>
        </div>

        {actionError && (
          <p className="rounded-md border border-danger/30 bg-danger-soft px-3.5 py-2.5 text-[13px] font-medium text-danger">{actionError}</p>
        )}

        {confirmingCancel && (
          <div className="flex flex-wrap items-center justify-between gap-3 rounded-md border border-danger/30 bg-danger-soft px-4 py-3">
            <span className="flex items-center gap-2 text-[13px] font-medium text-danger">
              <AlertTriangle className="h-4 w-4" /> Yakin ingin membatalkan project ini? Data yang sudah ada tidak akan dihapus.
            </span>
            <span className="flex gap-2">
              <Button variant="secondary" size="sm" onClick={() => setConfirmingCancel(false)}>Batal</Button>
              <Button variant="danger" size="sm" onClick={() => void handleCancelProject()}>Ya, Batalkan</Button>
            </span>
          </div>
        )}

        <div className="grid grid-cols-2 gap-4 border-t border-border-light pt-4 sm:grid-cols-3 lg:grid-cols-6">
          <InfoField label="Tanggal Acara" value={formatDate(project.eventDate)} />
          <InfoField
            label="Countdown"
            value={isOpenProject ? (d >= 0 ? `H-${d}` : `H+${Math.abs(d)}`) : project.status === "Completed" ? "Selesai" : "Dibatalkan"}
            emphasize
          />
          <InfoField label="Mulai Persiapan" value={formatDate(project.prepStartDate)} />
          <InfoField label="Paket / Layanan" value={project.packageName} />
          <InfoField label="Nilai Kontrak" value={formatCurrency(project.contractValue)} />
          <InfoField label="Penanggung Jawab" value={pic?.name ?? "-"} />
        </div>

        {project.description && (
          <p className="rounded-md bg-surface-muted px-4 py-3 text-[13px] text-text-secondary">{project.description}</p>
        )}

        {progress && (
          <div className="rounded-md bg-surface-muted/60 p-4">
            <div className="mb-2 flex flex-wrap items-center justify-between gap-2">
              <div className="flex items-center gap-3">
                <span className="text-2xl font-bold leading-none tabular-nums text-navy-900">{progress.overallPercent}%</span>
                <ConditionBadge condition={progress.condition} />
              </div>
              <span className="text-[12.5px] text-text-secondary">
                Milestone project {progress.projectMilestoneStats.completed}/{progress.projectMilestoneStats.total} · Milestone vendor{" "}
                {progress.vendorMilestoneStats.completed}/{progress.vendorMilestoneStats.total}
                {progress.overdueMilestoneCount > 0 && (
                  <span className="font-semibold text-danger"> · {progress.overdueMilestoneCount} terlambat</span>
                )}
              </span>
            </div>
            <ProgressMeter
              percent={progress.overallPercent}
              segments={segments}
              caption={`${completedMilestones}/${totalMilestones} milestone selesai berdasarkan pencapaian nyata — bukan estimasi manual.`}
            />
          </div>
        )}
      </CardContent>

      <ProjectFormModal open={editOpen} onClose={() => setEditOpen(false)} onSubmit={(values) => void handleEdit(values)} initialProject={project} />
    </Card>
  );
}

function InfoField({ label, value, emphasize }: { label: string; value: string; emphasize?: boolean }) {
  return (
    <div>
      <p className="text-[11.5px] font-medium uppercase tracking-wide text-text-secondary">{label}</p>
      <p className={emphasize ? "mt-0.5 text-[15px] font-bold tabular-nums text-navy-900" : "mt-0.5 text-[13.5px] font-medium text-text-primary"}>
        {value}
      </p>
    </div>
  );
}
