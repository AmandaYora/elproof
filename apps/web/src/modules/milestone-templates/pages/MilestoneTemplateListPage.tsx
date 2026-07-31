import { useEffect, useState } from "react";
import { Plus, Pencil, Trash2, ArrowUp, ArrowDown, AlertTriangle } from "lucide-react";
import { Card, CardHeader, CardContent } from "@/shared/components/ui/Card";
import { Button } from "@/shared/components/ui/Button";
import { Table, THead, TBody, TR, TH, TD } from "@/shared/components/ui/Table";
import { CardList, CardListField } from "@/shared/components/ui/CardList";
import { EmptyState } from "@/shared/components/feedback/EmptyState";
import { IconActionButton } from "@/shared/components/ui/IconActionButton";
import { MilestoneTemplateFormModal } from "@/modules/milestone-templates/components/MilestoneTemplateFormModal";
import type { MilestoneTemplateFormValues } from "@/modules/milestone-templates/schemas/milestone-template.schema";
import { useMilestoneTemplateStore } from "@/modules/milestone-templates/stores/useMilestoneTemplateStore";
import type { MilestoneTemplate } from "@/modules/milestone-templates/types";
import { getApiErrorMessage } from "@/shared/lib/api-error";

export default function MilestoneTemplateListPage() {
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

  useEffect(() => {
    void fetchTemplates();
  }, [fetchTemplates]);

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

  async function handleSubmit(values: MilestoneTemplateFormValues) {
    setActionError(null);
    try {
      if (editingTemplate) {
        await updateTemplate(editingTemplate.id, values);
      } else {
        await createTemplate(values);
      }
      closeModal();
    } catch (err) {
      setActionError(getApiErrorMessage(err, "Gagal menyimpan timeline default"));
    }
  }

  async function handleDelete(id: string) {
    setActionError(null);
    try {
      await deleteTemplate(id);
      setConfirmingDeleteId(null);
    } catch (err) {
      setActionError(getApiErrorMessage(err, "Gagal menghapus timeline default"));
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
      setActionError(getApiErrorMessage(err, "Gagal mengubah urutan timeline default"));
    }
  }

  const confirmingTemplate = templates.find((t) => t.id === confirmingDeleteId);

  return (
    <div className="flex flex-col gap-5">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-xl font-bold text-text-primary">Timeline Default</h1>
          <p className="mt-1 text-[13px] text-text-secondary">
            Checklist ini otomatis ditambahkan ke tab Timeline setiap kali project baru dibuat.
          </p>
        </div>
        <Button icon={<Plus className="h-4 w-4" />} onClick={openCreateModal}>
          Tambah Timeline
        </Button>
      </div>

      <Card>
        <CardHeader
          title="Checklist Timeline"
          subtitle="Urutan di sini menentukan urutan tampil di tab Timeline project baru. Perubahan tidak memengaruhi project yang sudah ada."
        />
        <CardContent>
          {actionError && (
            <p className="mb-3 rounded-md border border-danger/30 bg-danger-soft px-3.5 py-2.5 text-[13px] font-medium text-danger">{actionError}</p>
          )}

          {confirmingTemplate && (
            <div className="mb-3 flex flex-wrap items-center justify-between gap-3 rounded-md border border-danger/30 bg-danger-soft px-4 py-3">
              <span className="flex items-center gap-2 text-[13px] font-medium text-danger">
                <AlertTriangle className="h-4 w-4 shrink-0" />
                Yakin ingin menghapus "{confirmingTemplate.name}"? Project yang sudah dibuat tidak akan terpengaruh.
              </span>
              <span className="flex shrink-0 gap-2">
                <Button variant="secondary" size="sm" onClick={() => setConfirmingDeleteId(null)}>Batal</Button>
                <Button variant="danger" size="sm" onClick={() => void handleDelete(confirmingTemplate.id)}>Ya, Hapus</Button>
              </span>
            </div>
          )}

          {templates.length === 0 ? (
            <EmptyState
              title="Belum ada timeline default"
              description="Tambahkan item pertama supaya project baru tidak lahir dengan tab Timeline kosong."
            />
          ) : (
            <>
              <CardList
                className="sm:hidden"
                items={templates}
                keyFor={(t) => t.id}
                renderItem={(t) => {
                  const idx = templates.findIndex((x) => x.id === t.id);
                  return (
                    <>
                      <div className="flex items-start justify-between gap-3">
                        <span className="font-medium text-text-primary">{t.sortOrder}. {t.name}</span>
                        <div className="flex shrink-0 items-center gap-1">
                          <IconActionButton
                            icon={ArrowUp}
                            label="Naikkan urutan"
                            tone="neutral"
                            disabled={idx <= 0}
                            onClick={() => void handleMove(t.id, "up")}
                          />
                          <IconActionButton
                            icon={ArrowDown}
                            label="Turunkan urutan"
                            tone="neutral"
                            disabled={idx === -1 || idx >= templates.length - 1}
                            onClick={() => void handleMove(t.id, "down")}
                          />
                          <IconActionButton icon={Pencil} label="Ubah timeline" tone="neutral" onClick={() => openEditModal(t)} />
                          <IconActionButton icon={Trash2} label="Hapus timeline" tone="danger" onClick={() => setConfirmingDeleteId(t.id)} />
                        </div>
                      </div>
                      <CardListField label="Hari Sebelum Acara" value={t.daysBeforeEvent === 0 ? "Hari-H" : `H-${t.daysBeforeEvent}`} />
                    </>
                  );
                }}
              />
              <div className="hidden sm:block">
                <Table>
                  <THead>
                    <TR>
                      <TH>Nama Timeline</TH>
                      <TH>Hari Sebelum Acara</TH>
                      <TH>Aksi</TH>
                    </TR>
                  </THead>
                  <TBody>
                    {templates.map((t, idx) => (
                      <TR key={t.id}>
                        <TD className="font-medium text-text-primary">{t.sortOrder}. {t.name}</TD>
                        <TD className="text-text-secondary">{t.daysBeforeEvent === 0 ? "Hari-H" : `H-${t.daysBeforeEvent}`}</TD>
                        <TD>
                          <div className="flex items-center gap-1.5">
                            <IconActionButton
                              icon={ArrowUp}
                              label="Naikkan urutan"
                              tone="neutral"
                              disabled={idx <= 0}
                              onClick={() => void handleMove(t.id, "up")}
                            />
                            <IconActionButton
                              icon={ArrowDown}
                              label="Turunkan urutan"
                              tone="neutral"
                              disabled={idx >= templates.length - 1}
                              onClick={() => void handleMove(t.id, "down")}
                            />
                            <IconActionButton icon={Pencil} label="Ubah timeline" tone="neutral" onClick={() => openEditModal(t)} />
                            <IconActionButton icon={Trash2} label="Hapus timeline" tone="danger" onClick={() => setConfirmingDeleteId(t.id)} />
                          </div>
                        </TD>
                      </TR>
                    ))}
                  </TBody>
                </Table>
              </div>
            </>
          )}
        </CardContent>
      </Card>

      <MilestoneTemplateFormModal
        key={editingTemplate?.id ?? "new"}
        open={modalOpen}
        onClose={closeModal}
        onSubmit={(values) => void handleSubmit(values)}
        initialTemplate={editingTemplate}
      />
    </div>
  );
}
