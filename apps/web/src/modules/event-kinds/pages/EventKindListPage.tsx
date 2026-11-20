import { useCallback, useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { Plus, Pencil, CheckCircle2, Ban, Trash2, AlertTriangle } from "lucide-react";
import { Button } from "@/shared/components/ui/Button";
import { Badge } from "@/shared/components/ui/Badge";
import { Modal } from "@/shared/components/ui/Modal";
import { PageHeader } from "@/shared/components/ui/PageHeader";
import { RowActions } from "@/shared/components/ui/RowActions";
import { DataTable, type Column } from "@/shared/data-table/DataTable";
import { useCan } from "@/shared/stores/usePermissionStore";
import { toast } from "@/shared/stores/useToastStore";
import { getApiErrorMessage } from "@/shared/lib/api-error";
import { isStaleConflict } from "@/shared/lib/concurrency";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import { EventKindFormModal } from "@/modules/event-kinds/components/EventKindFormModal";
import type { EventKindFormValues } from "@/modules/event-kinds/schemas/event-kind.schema";
import { useEventKindStore } from "@/modules/event-kinds/stores/useEventKindStore";
import type { EventKind } from "@/modules/event-kinds/types";

/** Katalog › Jenis acara (DESAIN.md §6.7): the six built-in kinds plus the WO's own, renamed or deactivated in place. */
/** "3 paket · 2 penawaran": where a kind is in use ("" = nowhere, so it can be deleted). */
function usedIn(k: EventKind): string {
  const parts: string[] = [];
  if (k.packageCount > 0) parts.push(`${k.packageCount.toLocaleString("id-ID")} paket`);
  if (k.quotationCount > 0) parts.push(`${k.quotationCount.toLocaleString("id-ID")} penawaran`);
  return parts.join(" · ");
}

export default function EventKindListPage() {
  // package.manage, and "no" while the workspace is read-only (subscription): the write buttons disappear then.
  const canManage = useCan("package.manage");
  const kinds = useEventKindStore((s) => s.kinds);
  const fetchKinds = useEventKindStore((s) => s.fetchKinds);
  const createKind = useEventKindStore((s) => s.createKind);
  const renameKind = useEventKindStore((s) => s.renameKind);
  const setKindActive = useEventKindStore((s) => s.setKindActive);
  const deleteKind = useEventKindStore((s) => s.deleteKind);

  const [status, setStatus] = useState<"loading" | "error" | "ready">("loading");
  const [attempt, setAttempt] = useState(0);
  const [modalOpen, setModalOpen] = useState(false);
  const [editing, setEditing] = useState<EventKind | undefined>(undefined);
  const [actionError, setActionError] = useState<string | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<EventKind | null>(null);
  const [deleteError, setDeleteError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    let cancelled = false;
    setStatus("loading");
    fetchKinds()
      .then(() => !cancelled && setStatus("ready"))
      .catch(() => !cancelled && setStatus("error"));
    return () => {
      cancelled = true;
    };
  }, [fetchKinds, attempt]);

  const openCreate = useCallback(() => {
    setEditing(undefined);
    setModalOpen(true);
  }, []);

  function openRename(kind: EventKind) {
    setEditing(kind);
    setModalOpen(true);
  }

  async function handleSubmit(values: EventKindFormValues) {
    setActionError(null);
    try {
      if (editing) await renameKind(editing, values.name);
      else await createKind(values.name);
    } catch (err) {
      if (!isStaleConflict(err)) throw err;
      // Someone renamed it after this page loaded: show the current name instead of overwriting it.
      await fetchKinds();
      setModalOpen(false);
      setActionError("Jenis acara ini baru saja diubah orang lain. Daftar sudah dimuat ulang; periksa namanya, lalu ubah lagi bila perlu.");
      return;
    }
    setModalOpen(false);
    toast.success(editing ? "Jenis acara disimpan" : "Jenis acara ditambahkan");
  }

  async function handleSetActive(kind: EventKind, isActive: boolean) {
    setActionError(null);
    try {
      await setKindActive(kind.id, isActive);
      toast.success(isActive ? "Jenis acara diaktifkan" : "Jenis acara dinonaktifkan");
    } catch (err) {
      setActionError(getApiErrorMessage(err, "Status jenis acara belum bisa diubah. Coba lagi."));
    }
  }

  function closeDelete() {
    setDeleteTarget(null);
    setDeleteError(null);
  }

  async function handleDelete() {
    if (!deleteTarget) return;
    setBusy(true);
    setDeleteError(null);
    try {
      await deleteKind(deleteTarget.id);
      closeDelete();
      toast.success("Jenis acara dihapus");
    } catch (err) {
      // 409: a package started covering it since the list loaded. The server's sentence says how many and what to do.
      setDeleteError(getApiErrorMessage(err, "Jenis acara belum bisa dihapus. Periksa koneksi Anda, lalu coba lagi."));
      void fetchKinds();
    } finally {
      setBusy(false);
    }
  }

  async function handleDeactivateInstead() {
    if (!deleteTarget) return;
    setBusy(true);
    try {
      await setKindActive(deleteTarget.id, false);
      closeDelete();
      toast.success("Jenis acara dinonaktifkan");
    } catch (err) {
      setDeleteError(getApiErrorMessage(err, "Jenis acara belum bisa dinonaktifkan. Coba lagi."));
    } finally {
      setBusy(false);
    }
  }

  const columns: Column<EventKind>[] = [
    {
      id: "name",
      header: "Jenis acara",
      primary: true,
      className: "w-[40%] min-w-48",
      cell: (k) => (
        <span className="flex flex-wrap items-center gap-2">
          <span className="font-semibold">{k.name}</span>
          {k.isDefault && (
            <Badge tone="neutral" dot={false}>
              Bawaan
            </Badge>
          )}
        </span>
      ),
    },
    {
      id: "used",
      header: "Dipakai di",
      card: true,
      className: "whitespace-nowrap",
      cell: (k) => usedIn(k) || <span className="text-muted-foreground">Belum dipakai</span>,
    },
    {
      id: "status",
      header: "Status",
      card: true,
      className: "whitespace-nowrap",
      cell: (k) => (k.isActive ? <Badge tone="success">Aktif</Badge> : <Badge tone="neutral">Nonaktif</Badge>),
    },
  ];

  const total = kinds.length;

  return (
    <div className="flex flex-col gap-5">
      <PageHeader
        className="mb-1"
        title="Jenis acara"
        description="Acara yang bisa dicakup paket, misalnya Akad dan Resepsi. Nonaktifkan yang tidak Anda layani; paket dan penawaran lama tetap utuh."
        actions={
          canManage && (
            <Button icon={<Plus className="h-4 w-4" />} onClick={openCreate}>
              Tambah jenis acara
            </Button>
          )
        }
      />

      {actionError && (
        <p role="alert" className="rounded-md border border-danger/30 bg-danger-soft px-3.5 py-2.5 text-[13px] font-medium text-danger-text">
          {actionError}
        </p>
      )}

      <DataTable
        label="Daftar jenis acara"
        columns={columns}
        rows={kinds}
        rowId={(k) => k.id}
        rowLabel={(k) => k.name}
        status={status}
        onRetry={() => setAttempt((n) => n + 1)}
        meta={{ page: 1, limit: Math.max(total, 1), total, totalPages: 1 }}
        onPageChange={() => undefined}
        sort={null}
        onSortChange={() => undefined}
        filtered={false}
        onResetFilters={() => undefined}
        empty={{
          title: "Belum ada jenis acara",
          description: "Tambahkan acara yang Anda layani, misalnya Akad atau Resepsi, agar bisa dipilih di paket.",
          action: canManage ? <Button icon={<Plus className="h-4 w-4" />} onClick={openCreate}>Tambah jenis acara</Button> : undefined,
        }}
        actions={(kind) => (
          <RowActions
            rowLabel={kind.name}
            primary={{ label: "Ganti nama", icon: <Pencil />, onClick: () => openRename(kind), hidden: !canManage }}
            items={[
              kind.isActive
                ? { label: "Nonaktifkan", icon: <Ban />, onSelect: () => void handleSetActive(kind, false), hidden: !canManage }
                : { label: "Aktifkan", icon: <CheckCircle2 />, onSelect: () => void handleSetActive(kind, true), hidden: !canManage },
              { label: "Hapus permanen", icon: <Trash2 />, tone: "danger", onSelect: () => setDeleteTarget(kind), hidden: !canManage },
            ]}
          />
        )}
      />

      <p className="text-[13px] text-muted-foreground">
        Jenis acara dipilih saat menyusun <Link to={ROUTE_PATHS.packages} className="font-medium text-foreground underline underline-offset-2">paket</Link>.
      </p>

      <EventKindFormModal key={editing?.id ?? "new"} open={modalOpen} onClose={() => setModalOpen(false)} onSubmit={handleSubmit} kind={editing} />

      <Modal
        open={deleteTarget !== null}
        onClose={closeDelete}
        size="sm"
        busy={busy}
        error={deleteError}
        title={deleteTarget ? `${usedIn(deleteTarget) ? "Jenis acara masih dipakai" : "Hapus jenis acara"} — ${deleteTarget.name}` : "Hapus jenis acara"}
      >
        {deleteTarget &&
          (usedIn(deleteTarget) ? (
            <div className="flex flex-col gap-4">
              <p className="flex items-start gap-2 text-[13px] text-text-secondary">
                <AlertTriangle aria-hidden className="mt-0.5 h-4 w-4 shrink-0 text-warning-text" />
                <span>
                  {deleteTarget.name} dipakai di <strong className="text-text-primary">{usedIn(deleteTarget)}</strong>, jadi belum bisa dihapus.{" "}
                  {deleteTarget.isActive ? "Nonaktifkan saja: paket dan penawaran lama tetap utuh, dan jenis acara ini tidak muncul lagi saat menyusun paket atau penawaran baru." : "Jenis acara ini sudah nonaktif."}
                </span>
              </p>
              <div className="flex justify-end gap-2">
                <Button variant="secondary" onClick={closeDelete} disabled={busy}>
                  Tutup
                </Button>
                {deleteTarget.isActive && (
                  <Button onClick={() => void handleDeactivateInstead()} disabled={busy}>
                    {busy ? "Menyimpan..." : "Nonaktifkan"}
                  </Button>
                )}
              </div>
            </div>
          ) : (
            <div className="flex flex-col gap-4">
              <p className="flex items-start gap-2 text-[13px] text-text-secondary">
                <AlertTriangle aria-hidden className="mt-0.5 h-4 w-4 shrink-0 text-danger-text" />
                <span>Jenis acara ini dihapus permanen dan tidak bisa dikembalikan.</span>
              </p>
              <div className="flex justify-end gap-2">
                <Button variant="secondary" onClick={closeDelete} disabled={busy}>
                  Batal
                </Button>
                <Button variant="danger" onClick={() => void handleDelete()} disabled={busy}>
                  {busy ? "Menghapus..." : "Hapus permanen"}
                </Button>
              </div>
            </div>
          ))}
      </Modal>
    </div>
  );
}
