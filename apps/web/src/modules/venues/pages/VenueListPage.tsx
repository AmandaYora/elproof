import { useCallback, useEffect, useState } from "react";
import { useCan, useWriteLocked } from "@/shared/stores/usePermissionStore";
import { Plus, Pencil, CheckCircle2, Ban, Upload, Download, Trash2, AlertTriangle } from "lucide-react";
import { Button } from "@/shared/components/ui/Button";
import { Select } from "@/shared/components/ui/Input";
import { Combobox } from "@/shared/components/ui/Combobox";
import { CITIES } from "@/shared/constants/cities";
import { ImportBulkModal } from "@/shared/components/ui/ImportBulkModal";
import { Modal } from "@/shared/components/ui/Modal";
import { DataTable, type Column } from "@/shared/data-table/DataTable";
import { ListToolbar } from "@/shared/data-table/ListToolbar";
import { useListState } from "@/shared/data-table/useListState";
import { useListDensity } from "@/shared/data-table/useListDensity";
import { useSavedViews, viewQuery } from "@/shared/data-table/useSavedViews";
import { toast } from "@/shared/stores/useToastStore";
import { IconActionButton } from "@/shared/components/ui/IconActionButton";
import { VenueFormModal } from "@/modules/venues/components/VenueFormModal";
import { VenueStatusBadge } from "@/modules/venues/components/VenueStatusBadge";
import type { VenueFormValues, VenueCreateFormValues } from "@/modules/venues/schemas/venue.schema";
import { useVenueStore } from "@/modules/venues/stores/useVenueStore";
import type { Venue } from "@/modules/venues/types";
import { getApiErrorMessage } from "@/shared/lib/api-error";
import { formatCurrency } from "@/shared/lib/formatters";
import type { AffectedProject } from "@/shared/types/delete-impact";

export default function VenueListPage() {
  // Read-only workspace (subscription): no new records until it is renewed.
  const writeLocked = useWriteLocked();
  const venues = useVenueStore((s) => s.venuePage);
  const meta = useVenueStore((s) => s.venuePageMeta);
  const bulkVenues = useVenueStore((s) => s.bulkVenues);
  const fetchVenuePage = useVenueStore((s) => s.fetchVenuePage);
  const createVenue = useVenueStore((s) => s.createVenue);
  const updateVenue = useVenueStore((s) => s.updateVenue);
  const toggleVenueActive = useVenueStore((s) => s.toggleVenueActive);
  const downloadVenueTemplate = useVenueStore((s) => s.downloadVenueTemplate);
  const importVenues = useVenueStore((s) => s.importVenues);
  const exportVenues = useVenueStore((s) => s.exportVenues);
  const fetchVenueDeleteImpact = useVenueStore((s) => s.fetchVenueDeleteImpact);
  const deleteVenue = useVenueStore((s) => s.deleteVenue);
  // Destructive/management actions follow the role's permissions (ADR-0026).
  const isOwner = useCan("vendor.delete");

  // Search, filters, sort and page live in the URL (shareable, survives reload and Back).
  const list = useListState({ filters: { city: "", isActive: "" } });
  const { density, setDensity } = useListDensity();
  const savedViews = useSavedViews("venues");
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [status, setStatus] = useState<"loading" | "error" | "ready">("loading");
  const [attempt, setAttempt] = useState(0);
  const [modalOpen, setModalOpen] = useState(false);
  const [editingVenue, setEditingVenue] = useState<Venue | undefined>(undefined);
  const [actionError, setActionError] = useState<string | null>(null);
  const [importModalOpen, setImportModalOpen] = useState(false);
  const [isExporting, setIsExporting] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<Venue | null>(null);
  const [deleteImpact, setDeleteImpact] = useState<AffectedProject[] | null>(null);
  const [impactLoading, setImpactLoading] = useState(false);
  const [isDeleting, setIsDeleting] = useState(false);

  useEffect(() => {
    let cancelled = false;
    setStatus("loading");
    fetchVenuePage(list.params)
      .then(() => !cancelled && setStatus("ready"))
      .catch(() => !cancelled && setStatus("error"));
    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [fetchVenuePage, list.paramsKey, attempt]);

  // What was selected on another page or before a filter changed is no longer in view: start clean.
  useEffect(() => {
    setSelected(new Set());
  }, [list.paramsKey]);

  const reload = useCallback(() => fetchVenuePage(list.params), [fetchVenuePage, list.params]);

  function openCreateModal() {
    setEditingVenue(undefined);
    setModalOpen(true);
  }

  function openEditModal(venue: Venue) {
    setEditingVenue(venue);
    setModalOpen(true);
  }

  function closeModal() {
    setModalOpen(false);
    setEditingVenue(undefined);
  }

  // Errors are not caught here: they propagate to the form, which shows a 422 on its fields.
  async function handleSubmit(values: VenueFormValues | VenueCreateFormValues) {
    setActionError(null);
    if (editingVenue) {
      await updateVenue(editingVenue.id, values as VenueFormValues);
    } else {
      await createVenue(values as VenueCreateFormValues);
    }
    closeModal();
    await reload();
  }

  async function handleToggleActive(venue: Venue) {
    setActionError(null);
    try {
      await toggleVenueActive(venue.id);
      await reload();
    } catch (err) {
      setActionError(getApiErrorMessage(err, "Gagal mengubah status venue"));
    }
  }

  async function handleImportVenues(file: File) {
    const result = await importVenues(file);
    await reload();
    return result;
  }

  // Hard delete (PLAN.md) — informed-consent, not a blocking precondition.
  async function openDeleteConfirm(venue: Venue) {
    setDeleteTarget(venue);
    setDeleteImpact(null);
    setImpactLoading(true);
    try {
      setDeleteImpact(await fetchVenueDeleteImpact(venue.id));
    } catch {
      setDeleteImpact([]);
    } finally {
      setImpactLoading(false);
    }
  }

  function closeDeleteConfirm() {
    setDeleteTarget(null);
    setDeleteImpact(null);
  }

  async function handleConfirmDelete() {
    if (!deleteTarget) return;
    setIsDeleting(true);
    try {
      await deleteVenue(deleteTarget.id);
      closeDeleteConfirm();
      await reload();
    } catch (err) {
      setActionError(getApiErrorMessage(err, "Gagal menghapus venue"));
      closeDeleteConfirm();
    } finally {
      setIsDeleting(false);
    }
  }

  async function handleBulk(ids: string[], action: "activate" | "deactivate") {
    setActionError(null);
    try {
      const result = await bulkVenues(ids, action);
      setSelected(new Set());
      await reload();
      const verb = action === "activate" ? "diaktifkan" : "dinonaktifkan";
      toast.success(result.updated > 0 ? `${result.updated} venue ${verb}` : `Semua venue terpilih sudah ${action === "activate" ? "aktif" : "nonaktif"}`);
    } catch (err) {
      setActionError(getApiErrorMessage(err, "Venue terpilih belum bisa diubah. Coba lagi."));
    }
  }

  const columns: Column<Venue>[] = [
    { id: "name", header: "Venue", sortKey: "name", primary: true, cell: (v) => <span className="font-semibold">{v.name}</span> },
    {
      id: "pic",
      header: "PIC",
      cell: (v) => (
        <>
          <span className="block">{v.picName}</span>
          <span className="block text-[12.5px] text-muted-foreground">{v.phonePic}</span>
        </>
      ),
    },
    { id: "city", header: "Kota", sortKey: "city", cell: (v) => v.city ?? <span className="text-muted-foreground">-</span> },
    { id: "capacity", header: "Kapasitas", sortKey: "capacity", align: "right", cell: (v) => (v.capacity ? `${v.capacity} orang` : "-") },
    {
      id: "rentalPrice",
      header: "Harga sewa",
      sortKey: "rentalPrice",
      align: "right",
      cell: (v) => (v.rentalPrice ? formatCurrency(v.rentalPrice) : <span className="text-muted-foreground">Belum diisi</span>),
    },
    { id: "status", header: "Status", sortKey: "isActive", cell: (v) => <VenueStatusBadge isActive={v.isActive} /> },
  ];

  async function handleExportVenues() {
    setActionError(null);
    setIsExporting(true);
    try {
      const blob = await exportVenues(list.q, list.filters.city);
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = "venues-export.xlsx";
      a.click();
      URL.revokeObjectURL(url);
    } catch (err) {
      setActionError(getApiErrorMessage(err, "Gagal mengekspor data venue"));
    } finally {
      setIsExporting(false);
    }
  }

  return (
    <div className="flex flex-col gap-5">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-xl font-bold text-text-primary">Venue</h1>
          <p className="mt-1 text-[13px] text-text-secondary">Kelola direktori gedung/lokasi acara yang bekerja sama dengan WO.</p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          {!writeLocked && (
            <Button variant="secondary" icon={<Upload className="h-4 w-4" />} onClick={() => setImportModalOpen(true)}>
              Import Bulk
            </Button>
          )}
          <Button variant="secondary" icon={<Download className="h-4 w-4" />} onClick={() => void handleExportVenues()} disabled={isExporting}>
            {isExporting ? "Mengekspor..." : "Export Excel"}
          </Button>
          {!writeLocked && (
            <Button icon={<Plus className="h-4 w-4" />} onClick={openCreateModal}>
              Tambah Venue
            </Button>
          )}
        </div>
      </div>

      {actionError && (
        <p className="rounded-md border border-danger/30 bg-danger-soft px-3.5 py-2.5 text-[13px] font-medium text-danger-text">{actionError}</p>
      )}

      <ListToolbar
        search={{ value: list.q, onChange: list.setQ, placeholder: "Cari nama venue atau PIC..." }}
        density={{ value: density, onChange: setDensity }}
        views={{
          items: savedViews.views,
          current: viewQuery(list.queryString),
          onApply: list.applyQueryString,
          onSave: savedViews.save,
          onRemove: savedViews.remove,
        }}
      >
        <Combobox className="w-52" value={list.filters.city || "Semua"} onChange={(e) => list.setFilter("city", e.target.value === "Semua" ? "" : e.target.value)} placeholder="Cari kota...">
          <option value="Semua">Semua kota</option>
          {CITIES.map((city) => (
            <option key={city} value={city}>
              {city}
            </option>
          ))}
        </Combobox>
        <Select className="w-40" aria-label="Status" value={list.filters.isActive || "Semua"} onChange={(e) => list.setFilter("isActive", e.target.value === "Semua" ? "" : e.target.value)}>
          <option value="Semua">Semua status</option>
          <option value="true">Aktif</option>
          <option value="false">Nonaktif</option>
        </Select>
      </ListToolbar>

      <DataTable
        label="Daftar venue"
        columns={columns}
        rows={venues}
        rowId={(v) => v.id}
        rowLabel={(v) => v.name}
        status={status}
        onRetry={() => setAttempt((n) => n + 1)}
        meta={meta}
        onPageChange={list.setPage}
        sort={list.sort}
        onSortChange={list.setSort}
        filtered={list.isFiltered}
        onResetFilters={list.reset}
        empty={{
          title: "Belum ada venue",
          description: "Catat gedung dan lokasi acara yang sering dipakai agar bisa dipilih saat membuat proyek.",
          action: !writeLocked ? <Button icon={<Plus className="h-4 w-4" />} onClick={openCreateModal}>Tambah venue</Button> : undefined,
        }}
        density={density}
        selection={writeLocked ? undefined : { selected, onChange: setSelected }}
        bulkActions={(ids) => (
          <>
            <Button variant="secondary" size="sm" onClick={() => void handleBulk(ids, "activate")}>Aktifkan</Button>
            <Button variant="secondary" size="sm" onClick={() => void handleBulk(ids, "deactivate")}>Nonaktifkan</Button>
          </>
        )}
        actions={(venue) => (
          <div className="flex items-center gap-1.5">
            <IconActionButton icon={Pencil} label="Ubah Venue" tone="neutral" onClick={() => openEditModal(venue)} />
            {venue.isActive ? (
              <IconActionButton icon={Ban} label="Nonaktifkan" tone="danger" onClick={() => void handleToggleActive(venue)} />
            ) : (
              <IconActionButton icon={CheckCircle2} label="Aktifkan" tone="success" onClick={() => void handleToggleActive(venue)} />
            )}
            {isOwner && <IconActionButton icon={Trash2} label="Hapus Permanen" tone="danger" onClick={() => void openDeleteConfirm(venue)} />}
          </div>
        )}
      />

      <VenueFormModal
        key={editingVenue?.id ?? "new"}
        open={modalOpen}
        onClose={closeModal}
        onSubmitCreate={handleSubmit}
        onSubmitEdit={handleSubmit}
        initialVenue={editingVenue}
      />

      <Modal
        open={deleteTarget !== null}
        onClose={closeDeleteConfirm}
        title={deleteTarget ? `Hapus Venue — ${deleteTarget.name}` : "Hapus Venue"}
        size="sm"
      >
        {impactLoading ? (
          <p className="text-[13px] text-text-secondary">Memeriksa penggunaan...</p>
        ) : (
          <div className="flex flex-col gap-4">
            <p className="flex items-start gap-2 text-[13px] text-text-secondary">
              <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0 text-danger-text" />
              {deleteImpact && deleteImpact.length > 0 ? (
                <span>
                  Venue ini masih digunakan oleh proyek berikut:{" "}
                  <strong className="text-text-primary">{deleteImpact.map((p) => p.name).join(", ")}</strong>. Jika dihapus, data
                  venue pada proyek tersebut akan kehilangan sumber datanya dan tidak dapat ditelusuri lagi. Apakah Anda yakin ingin
                  menghapus venue ini secara permanen?
                </span>
              ) : (
                <span>Data ini akan dihapus secara permanen dan tidak dapat dikembalikan. Apakah Anda yakin?</span>
              )}
            </p>
            <div className="flex justify-end gap-2">
              <Button variant="secondary" onClick={closeDeleteConfirm} disabled={isDeleting}>
                Batal
              </Button>
              <Button variant="danger" onClick={() => void handleConfirmDelete()} disabled={isDeleting}>
                {isDeleting ? "Menghapus..." : "Ya, Hapus Permanen"}
              </Button>
            </div>
          </div>
        )}
      </Modal>

      {importModalOpen && (
        <ImportBulkModal
          open={importModalOpen}
          onClose={() => setImportModalOpen(false)}
          title="Import Bulk Venue"
          description="Download template Excel, isi datanya, lalu unggah kembali untuk menambah atau memperbarui banyak venue sekaligus."
          templateFileName="template-venue.xlsx"
          entityLabel="venue"
          matchCriteriaLabel="nama+kota"
          onDownloadTemplate={downloadVenueTemplate}
          onImport={handleImportVenues}
        />
      )}
    </div>
  );
}
