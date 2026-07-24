import { useEffect, useState } from "react";
import { Plus, Ban, CheckCircle2 } from "lucide-react";
import { Card, CardHeader, CardContent } from "@/shared/components/ui/Card";
import { Button } from "@/shared/components/ui/Button";
import { Select } from "@/shared/components/ui/Input";
import { Table, THead, TBody, TR, TH, TD } from "@/shared/components/ui/Table";
import { CardList, CardListField } from "@/shared/components/ui/CardList";
import { Pagination } from "@/shared/components/ui/Pagination";
import { usePagination } from "@/shared/hooks/usePagination";
import { MilestoneRail, MilestoneRailLegend } from "@/shared/components/ui/MilestoneRail";
import { IconActionButton } from "@/shared/components/ui/IconActionButton";
import { ProjectMilestoneFormModal } from "@/modules/projects/components/ProjectMilestoneFormModal";
import type { ProjectMilestoneFormValues } from "@/modules/projects/schemas/project-milestone.schema";
import { useProjectStore } from "@/modules/projects/stores/useProjectStore";
import { computeMilestoneStats, isMilestoneOverdue } from "@/modules/projects/lib/dates";
import type { ProjectMilestone, MilestoneStatus } from "@/modules/projects/types";
import { formatDate } from "@/shared/lib/formatters";
import { getApiErrorMessage } from "@/shared/lib/api-error";
import { cn } from "@/shared/lib/cn";

const STATUS_OPTIONS: MilestoneStatus[] = ["Not Started", "In Progress", "Completed", "Blocked", "Cancelled"];

function sortMilestones(list: ProjectMilestone[]): ProjectMilestone[] {
  return [...list].sort((a, b) => {
    const aCancelled = a.status === "Cancelled";
    const bCancelled = b.status === "Cancelled";
    if (aCancelled !== bCancelled) return aCancelled ? 1 : -1;
    return a.order - b.order;
  });
}

export function ProjectMilestonesSection({ projectId }: { projectId: string }) {
  const milestones = useProjectStore((s) => s.milestones);
  const fetchMilestones = useProjectStore((s) => s.fetchMilestones);
  const createMilestone = useProjectStore((s) => s.createMilestone);
  const updateMilestoneStatus = useProjectStore((s) => s.updateMilestoneStatus);
  const [addOpen, setAddOpen] = useState(false);
  const [actionError, setActionError] = useState<string | null>(null);
  const sortedMilestones = sortMilestones(milestones);
  const stats = computeMilestoneStats(milestones);
  const { page, setPage, totalPages, totalItems, pageSize, pageItems } = usePagination(sortedMilestones);

  useEffect(() => {
    void fetchMilestones(projectId);
  }, [projectId, fetchMilestones]);

  async function updateStatus(id: string, status: MilestoneStatus) {
    setActionError(null);
    try {
      await updateMilestoneStatus(projectId, id, status);
    } catch (err) {
      setActionError(getApiErrorMessage(err, "Gagal memperbarui status milestone"));
    }
  }

  async function handleAddMilestone(values: ProjectMilestoneFormValues) {
    setActionError(null);
    try {
      await createMilestone(projectId, values);
      setAddOpen(false);
    } catch (err) {
      setActionError(getApiErrorMessage(err, "Gagal menambahkan milestone"));
    }
  }

  return (
    <div id="milestone">
      <Card>
        <CardHeader
          title="Milestone Persiapan Acara"
          subtitle="Progress keseluruhan didasarkan pada milestone yang benar-benar telah diselesaikan, bukan angka manual."
          action={
            <Button size="sm" icon={<Plus className="h-3.5 w-3.5" />} onClick={() => setAddOpen(true)}>
              Tambah Milestone
            </Button>
          }
        />
        <CardContent>
          {actionError && (
            <p className="mb-3 rounded-md border border-danger/30 bg-danger-soft px-3.5 py-2.5 text-[13px] font-medium text-danger">{actionError}</p>
          )}
          <div className="flex flex-col gap-2 border-b border-border-light pb-4">
            <MilestoneRail milestones={milestones} size="md" />
            <div className="flex flex-wrap items-center justify-between gap-2">
              <MilestoneRailLegend />
              {stats.overdue > 0 && (
                <span className="text-[12.5px] font-semibold text-danger">{stats.overdue} milestone terlambat dari target</span>
              )}
            </div>
          </div>

          <CardList
            className="sm:hidden"
            items={pageItems}
            keyFor={(m) => m.id}
            renderItem={(m) => {
              const overdue = isMilestoneOverdue(m.status, m.targetDate);
              const cancelled = m.status === "Cancelled";
              return (
                <>
                  <div className="flex items-start justify-between gap-3">
                    <span className={cn("font-medium text-text-primary", cancelled && "line-through")}>{m.order}. {m.name}</span>
                    {cancelled ? (
                      <IconActionButton
                        icon={CheckCircle2}
                        label="Aktifkan kembali"
                        tone="success"
                        onClick={() => void updateStatus(m.id, "Not Started")}
                      />
                    ) : (
                      <IconActionButton
                        icon={Ban}
                        label="Batalkan milestone"
                        tone="danger"
                        onClick={() => void updateStatus(m.id, "Cancelled")}
                      />
                    )}
                  </div>
                  <div className="flex flex-col gap-1.5">
                    <CardListField
                      label="Target Tanggal"
                      value={
                        <span className={overdue ? "font-semibold text-danger" : undefined}>
                          {formatDate(m.targetDate)}
                          {overdue && " · terlambat"}
                        </span>
                      }
                    />
                    <CardListField label="Tanggal Selesai" value={formatDate(m.completedDate)} />
                  </div>
                  <Select
                    value={m.status}
                    onChange={(e) => void updateStatus(m.id, e.target.value as MilestoneStatus)}
                    className="h-9 w-full text-[13px]"
                  >
                    {STATUS_OPTIONS.map((s) => (
                      <option key={s} value={s}>{s}</option>
                    ))}
                  </Select>
                </>
              );
            }}
          />
          <div className="hidden sm:block">
          <Table className="mt-2">
            <THead>
              <TR>
                <TH>Milestone</TH>
                <TH>Status</TH>
                <TH>Target Tanggal</TH>
                <TH>Tanggal Selesai</TH>
                <TH>Aksi</TH>
              </TR>
            </THead>
            <TBody>
              {pageItems.map((m) => {
                const overdue = isMilestoneOverdue(m.status, m.targetDate);
                const cancelled = m.status === "Cancelled";
                return (
                  <TR key={m.id} className={cancelled ? "opacity-50" : undefined}>
                    <TD className={cn("font-medium", cancelled && "line-through")}>{m.order}. {m.name}</TD>
                    <TD>
                      <Select
                        value={m.status}
                        onChange={(e) => void updateStatus(m.id, e.target.value as MilestoneStatus)}
                        className="h-8 w-40 text-[13px]"
                      >
                        {STATUS_OPTIONS.map((s) => (
                          <option key={s} value={s}>{s}</option>
                        ))}
                      </Select>
                    </TD>
                    <TD className={overdue ? "font-semibold text-danger" : undefined}>
                      {formatDate(m.targetDate)}
                      {overdue && " · terlambat"}
                    </TD>
                    <TD>{formatDate(m.completedDate)}</TD>
                    <TD>
                      {cancelled ? (
                        <IconActionButton
                          icon={CheckCircle2}
                          label="Aktifkan kembali"
                          tone="success"
                          onClick={() => void updateStatus(m.id, "Not Started")}
                        />
                      ) : (
                        <IconActionButton
                          icon={Ban}
                          label="Batalkan milestone"
                          tone="danger"
                          onClick={() => void updateStatus(m.id, "Cancelled")}
                        />
                      )}
                    </TD>
                  </TR>
                );
              })}
            </TBody>
          </Table>
          </div>
          <Pagination
            page={page}
            totalPages={totalPages}
            totalItems={totalItems}
            pageSize={pageSize}
            onPageChange={setPage}
            className="-mx-5 -mb-4 mt-1"
          />
        </CardContent>
      </Card>

      <ProjectMilestoneFormModal open={addOpen} onClose={() => setAddOpen(false)} onSubmit={(values) => void handleAddMilestone(values)} />
    </div>
  );
}
