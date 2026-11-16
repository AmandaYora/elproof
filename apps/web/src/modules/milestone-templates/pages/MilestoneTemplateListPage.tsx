import { useEffect, useState } from "react";
import { Plus, Pencil, Trash2, ArrowUp, ArrowDown, AlertTriangle } from "lucide-react";
import { Button } from "@/shared/components/ui/Button";
import { PageHeader } from "@/shared/components/ui/PageHeader";
import { RowActions } from "@/shared/components/ui/RowActions";
import { DataTable, type Column } from "@/shared/data-table/DataTable";
import { MilestoneTemplateFormModal } from "@/modules/milestone-templates/components/MilestoneTemplateFormModal";
import type { MilestoneTemplateSubmitValues } from "@/modules/milestone-templates/schemas/milestone-template.schema";
import { useMilestoneTemplateStore } from "@/modules/milestone-templates/stores/useMilestoneTemplateStore";
import type { MilestoneTemplate } from "@/modules/milestone-templates/types";
import { formatDayCountdown } from "@/shared/lib/formatters";
import { getApiErrorMessage } from "@/shared/lib/api-error";
import { useWriteLocked } from "@/shared/stores/usePermissionStore";

function formatDaysLabel(daysBeforeEvent: number): string {
  return formatDayCountdown(daysBeforeEvent);
}

export default function MilestoneTemplateListPage() {
  // Read-only workspace (subscription): no new records until it is renewed.
  const writeLocked = useWriteLocked();
  const templates = useMilestoneTemplateStore((s) => s.templates);
  const fetchTemplates = useMilestoneTemplateStore((s) => s.fetchTemplates);
  const createTemplate = useMilestoneTemplateStore((s) => s.createTemplate);
  const updateTemplate = useMilestoneTemplateStore((s) => s.updateTemplate);
  const deleteTemplate = useMilestoneTemplateStore((s) => s.deleteTemplate);
  const reorderTemplates = useMilestoneTemplateStore((s) => s.reorderTemplates);

  const [modalOpen, setModalOpen] = useState(false);
  const [editingTemplate, setEditingTemplate] = useState<MilestoneTemplate | undefined>(undefined);
  const [confirmingDeleteId, setConfirmingDeleteId] = useState<string | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);
  const [status, setStatus] = useState<"loading" | "error" | "ready">("loading");
  const [attempt, setAttempt] = useState(0);

  useEffect(() => {
    let cancelled = false;
    setStatus("loading");
    fetchTemplates()
      .then(() => !cancelled && setStatus("ready"))
      .catch(() => !cancelled && setStatus("error"));
    return () => {
      cancelled = true;
    };
  }, [fetchTemplates, attempt]);

  function openCreateModal() {
    setEditingTemplate(undefined);
    setModalOpen(true);
  }

  function openEditModal(template: MilestoneTemplate) {
    setEditingTemplate(template);
    setModalOpen(true);
  }

  function closeModal() {
    setModalOpen(false);
    setEditingTemplate(undefined);
  }

  // Errors are not caught here: they propagate to the form, which shows a 422 on its fields.
  async function handleSubmit(values: MilestoneTemplateSubmitValues) {
    setActionError(null);
    if (editingTemplate) {
      await updateTemplate(editingTemplate.id, values);
    } else {
      await createTemplate(values);
    }
    closeModal();
  }

  async function handleDelete(id: string) {
    setActionError(null);
    try {
      await deleteTemplate(id);
      setConfirmingDeleteId(null);
    } catch (err) {
      setActionError(getApiErrorMessage(err, "Tahapan belum bisa dihapus. Periksa koneksi Anda, lalu coba lagi."));
    }
  }

  async function handleMove(id: string, direction: "up" | "down") {
    setActionError(null);
    const idx = templates.findIndex((t) => t.id === id);
    const swapIdx = direction === "up" ? idx - 1 : idx + 1;
    if (idx === -1 || swapIdx < 0 || swapIdx >= templates.length) return;
    const reordered = [...templates];
    [reordered[idx], reordered[swapIdx]] = [reordered[swapIdx], reordered[idx]];
    try {
      await reorderTemplates(reordered.map((t) => t.id));
    } catch (err) {
      setActionError(getApiErrorMessage(err, "Urutan belum bisa diubah. Periksa koneksi Anda, lalu coba lagi."));
    }
  }

  const confirmingTemplate = templates.find((t) => t.id === confirmingDeleteId);

  const columns: Column<MilestoneTemplate>[] = [
    { id: "order", header: "Urutan", align: "right", className: "w-20", hideOnCard: true, cell: (t) => t.sortOrder },
    {
      id: "name",
      header: "Nama tahapan",
      primary: true,
      truncate: true,
      cell: (t) =>
        writeLocked ? (
          <span className="font-medium"><span className="md:hidden">{t.sortOrder}. </span>{t.name}</span>
        ) : (
          <button
            type="button"
            onClick={() => openEditModal(t)}
            className="max-w-full truncate text-left font-medium underline-offset-2 hover:underline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-solid focus-visible:outline-ring"
          >
            <span className="md:hidden">{t.sortOrder}. </span>{t.name}
          </button>
        ),
    },
    { id: "days", header: "Waktu terhadap hari H", card: true, className: "w-56 whitespace-nowrap", cell: (t) => formatDaysLabel(t.daysBeforeEvent) },
  ];

  return (
    <div className="flex flex-col gap-5">
      <PageHeader
        className="mb-1"
        title="Jadwal bawaan"
        description="Tahapan ini otomatis masuk ke tab Jadwal setiap proyek baru, dalam urutan di bawah. Proyek yang sudah ada tidak berubah."
        actions={
          !writeLocked && (
            <Button icon={<Plus className="h-4 w-4" />} onClick={openCreateModal}>
              Tambah tahapan
            </Button>
          )
        }
      />

      {actionError && (
        <p className="rounded-md border border-danger/30 bg-danger-soft px-3.5 py-2.5 text-[13px] font-medium text-danger-text">{actionError}</p>
      )}

      {confirmingTemplate && (
        <div className="flex flex-wrap items-center justify-between gap-3 rounded-md border border-danger/30 bg-danger-soft px-4 py-3">
          <span className="flex items-center gap-2 text-[13px] font-medium text-danger-text">
            <AlertTriangle className="h-4 w-4 shrink-0" />
            Yakin ingin menghapus "{confirmingTemplate.name}"? Proyek yang sudah dibuat tidak akan terpengaruh.
          </span>
          <span className="flex shrink-0 gap-2">
            <Button variant="secondary" size="sm" onClick={() => setConfirmingDeleteId(null)}>Batal</Button>
            <Button variant="danger" size="sm" onClick={() => void handleDelete(confirmingTemplate.id)}>Hapus tahapan</Button>
          </span>
        </div>
      )}

      <DataTable
        label="Daftar tahapan jadwal bawaan"
        columns={columns}
        rows={templates}
        rowId={(t) => t.id}
        rowLabel={(t) => t.name}
        status={status}
        onRetry={() => setAttempt((n) => n + 1)}
        meta={{ page: 1, limit: templates.length, total: templates.length, totalPages: 1 }}
        onPageChange={() => undefined}
        sort={null}
        onSortChange={() => undefined}
        filtered={false}
        onResetFilters={() => undefined}
        empty={{
          title: "Belum ada tahapan di jadwal bawaan",
          description: "Tambahkan tahapan pertama supaya proyek baru tidak dimulai dengan tab Jadwal kosong.",
          action: !writeLocked ? <Button icon={<Plus className="h-4 w-4" />} onClick={openCreateModal}>Tambah tahapan</Button> : undefined,
        }}
        actions={(t) => {
          const idx = templates.findIndex((x) => x.id === t.id);
          return (
            <RowActions
              rowLabel={t.name}
              primary={{ label: "Ubah", icon: <Pencil />, onClick: () => openEditModal(t), hidden: writeLocked }}
              items={[
                { label: "Naikkan urutan", icon: <ArrowUp />, onSelect: () => void handleMove(t.id, "up"), hidden: writeLocked || idx <= 0 },
                { label: "Turunkan urutan", icon: <ArrowDown />, onSelect: () => void handleMove(t.id, "down"), hidden: writeLocked || idx >= templates.length - 1 },
                { label: "Hapus tahapan", icon: <Trash2 />, tone: "danger", onSelect: () => setConfirmingDeleteId(t.id), hidden: writeLocked },
              ]}
            />
          );
        }}
      />

      <MilestoneTemplateFormModal
        key={editingTemplate?.id ?? "new"}
        open={modalOpen}
        onClose={closeModal}
        onSubmit={handleSubmit}
        initialTemplate={editingTemplate}
      />
    </div>
  );
}
