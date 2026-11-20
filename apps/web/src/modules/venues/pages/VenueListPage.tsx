import { useCallback, useEffect, useState } from "react";
import { useCan } from "@/shared/stores/usePermissionStore";
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
import { PageHeader } from "@/shared/components/ui/PageHeader";
import { RowActions } from "@/shared/components/ui/RowActions";
import { ActionMenu } from "@/shared/components/ui/ActionMenu";
import { ListFilters, activeFilterCount } from "@/shared/data-table/ListFilters";
import { VenueFormModal } from "@/modules/venues/components/VenueFormModal";
import { VenueStatusBadge } from "@/modules/venues/components/VenueStatusBadge";
import type { VenueFormValues, VenueCreateFormValues } from "@/modules/venues/schemas/venue.schema";
import { useVenueStore } from "@/modules/venues/stores/useVenueStore";
import type { Venue } from "@/modules/venues/types";
import { venueCapacityText, venuePriceLines } from "@/modules/venues/lib/venue-display";
import { getApiErrorMessage } from "@/shared/lib/api-error";
import { formatCurrency } from "@/shared/lib/formatters";
import { describeAffectedProjects, inUseProjectsOf, isInOffersRefusal, type AffectedProject } from "@/shared/types/delete-impact";

export default function VenueListPage() {
  // Adding, editing, importing and (de)activating need vendor.manage (ADR-0026); useCan also reads "no" while the
  // workspace is read-only (subscription), so the write buttons disappear then too.
  const canManage = useCan("vendor.manage");
  const venues = useVenueStore((s) => s.venuePage);
  const meta = useVenueStore((s) => s.venuePageMeta);
  const bulkVenues = useVenueStore((s) => s.bulkVenues);
  const fetchVenuePage = useVenueStore((s) => s.fetchVenuePage);
  const createVenue = useVenueStore((s) => s.createVenue);
  const updateVenue = useVenueStore((s) => s.updateVenue);
  const setVenueActive = useVenueStore((s) => s.setVenueActive);
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
  // A package or quotation still refers to it (PLAN paket-wo §3.3): the server refuses the delete, like a project does.
  const [usedInOffers, setUsedInOffers] = useState(false);
  // Why the server refused the delete (e.g. 409: still used by a project), shown inside the dialog that asked.
  const [deleteError, setDeleteError] = useState<string | null>(null);
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
      await setVenueActive(venue.id, !venue.isActive);
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
    setUsedInOffers(false);
    setImpactLoading(true);
    try {
      const impact = await fetchVenueDeleteImpact(venue.id);
      setDeleteImpact(impact.projects);
      setUsedInOffers(impact.usedInOffers);
    } catch {
      setDeleteImpact([]);
    } finally {
      setImpactLoading(false);
    }
  }

  function closeDeleteConfirm() {
    setDeleteTarget(null);
    setDeleteImpact(null);
    setUsedInOffers(false);
    setDeleteError(null);
  }

  async function handleConfirmDelete() {
    if (!deleteTarget) return;
    setIsDeleting(true);
    setDeleteError(null);
    try {
      await deleteVenue(deleteTarget.id);
      closeDeleteConfirm();
      await reload();
    } catch (err) {
      // A project started using it since the check (409 venue_in_use): the dialog turns into "deactivate instead".
      const inUse = inUseProjectsOf(err);
      if (inUse) setDeleteImpact(inUse);
      // Or a package or quotation refers to it now (409 *_in_offers): the same "deactivate instead".
      if (isInOffersRefusal(err)) setUsedInOffers(true);
      setDeleteError(getApiErrorMessage(err, "Venue belum bisa dihapus. Periksa koneksi Anda, lalu coba lagi."));
    } finally {
      setIsDeleting(false);
    }
  }

  /** What the dialog offers for a venue a project still uses (QA FQ-02): it stays in those projects, out of new ones. */
  async function handleDeactivateInstead() {
    if (!deleteTarget) return;
    setIsDeleting(true);
    setDeleteError(null);
    try {
      await setVenueActive(deleteTarget.id, false);
      closeDeleteConfirm();
      toast.success("Venue dinonaktifkan");
      await reload();
    } catch (err) {
      setDeleteError(getApiErrorMessage(err, "Venue belum bisa dinonaktifkan. Periksa koneksi Anda, lalu coba lagi."));
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
    {
      id: "name",
      header: "Venue",
      sortKey: "name",
      primary: true,
      truncate: true,
      className: "w-[34%] min-w-60",
      // There is no venue page of its own: for someone who may change it, the name opens its details (the form).
      cell: (v) => (
        <>
          {canManage ? (
            <button
              type="button"
              onClick={() => openEditModal(v)}
              className="block max-w-full truncate text-left font-semibold underline-offset-2 hover:underline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-solid focus-visible:outline-ring"
            >
              {v.name}
            </button>
          ) : (
            <span className="block truncate font-semibold">{v.name}</span>
          )}
          <span className="block truncate text-[12.5px] font-normal text-muted-foreground">{[v.picName, v.phonePic].filter(Boolean).join(" · ")}</span>
        </>
      ),
    },
    { id: "city", header: "Kota", sortKey: "city", card: true, truncate: true, className: "w-[18%]", cell: (v) => v.city ?? <span className="text-muted-foreground">-</span> },
    {
      id: "capacity",
      header: "Kapasitas",
      sortKey: "capacityStanding",
      card: true,
      className: "whitespace-nowrap",
      cell: (v) => venueCapacityText(v) ?? <span className="text-muted-foreground">-</span>,
    },
    {
      id: "prices",
      header: "Harga",
      sortKey: "priceAkadResepsi",
      card: true,
      cell: (v) => {
        const prices = venuePriceLines(v);
        if (prices.length === 0) return <span className="text-muted-foreground">Belum diisi</span>;
        return (
          <span className="flex flex-col text-[12.5px] leading-snug">
            {prices.map(([label, amount]) => (
              <span key={label} className="whitespace-nowrap">
                <span className="text-muted-foreground">{label}</span> {formatCurrency(amount)}
              </span>
            ))}
          </span>
        );
      },
    },
    { id: "status", header: "Status", sortKey: "isActive", card: true, className: "whitespace-nowrap", cell: (v) => <VenueStatusBadge isActive={v.isActive} /> },
  ];

  async function handleExportVenues() {
    setActionError(null);
    setIsExporting(true);
    try {
      const blob = await exportVenues({ q: list.q, city: list.filters.city, isActive: list.filters.isActive });
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
      <PageHeader
        className="mb-1"
        title="Venue"
        description="Gedung dan lokasi acara yang sering Anda pakai, siap dipilih saat membuat proyek."
        actions={
          <>
            {canManage && (
              <Button variant="secondary" className="max-sm:hidden" icon={<Upload className="h-4 w-4" />} onClick={() => setImportModalOpen(true)}>
                Impor bulk
              </Button>
            )}
            <Button variant="secondary" className="max-sm:hidden" icon={<Download className="h-4 w-4" />} onClick={() => void handleExportVenues()} disabled={isExporting}>
              {isExporting ? "Mengekspor..." : "Ekspor ke Excel"}
            </Button>
            {canManage && (
              <Button icon={<Plus className="h-4 w-4" />} onClick={openCreateModal}>
                Tambah venue
              </Button>
            )}
            {/* On a phone, Impor and Ekspor sit behind "⋯" so the header stays one row. */}
            <div className="sm:hidden">
              <ActionMenu
                label="Impor dan ekspor venue"
                disabled={isExporting}
                items={[
                  { label: "Impor bulk", icon: <Upload />, onSelect: () => setImportModalOpen(true), hidden: !canManage },
                  { label: "Ekspor ke Excel", icon: <Download />, onSelect: () => void handleExportVenues() },
                ]}
              />
            </div>
          </>
        }
      />

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
        <ListFilters count={activeFilterCount(list.filters)} onClear={list.clearFilters}>
          <Combobox className="w-44" value={list.filters.city || "Semua"} onChange={(e) => list.setFilter("city", e.target.value === "Semua" ? "" : e.target.value)} placeholder="Cari kota...">
            <option value="Semua">Semua kota</option>
            {CITIES.map((city) => (
              <option key={city} value={city}>
                {city}
              </option>
            ))}
          </Combobox>
          <Select className="w-36" aria-label="Status" value={list.filters.isActive || "Semua"} onChange={(e) => list.setFilter("isActive", e.target.value === "Semua" ? "" : e.target.value)}>
            <option value="Semua">Semua status</option>
            <option value="true">Aktif</option>
            <option value="false">Nonaktif</option>
          </Select>
        </ListFilters>
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
          action: canManage ? <Button icon={<Plus className="h-4 w-4" />} onClick={openCreateModal}>Tambah venue</Button> : undefined,
        }}
        density={density}
        selection={!canManage ? undefined : { selected, onChange: setSelected }}
        bulkActions={(ids) => (
          <>
            <Button variant="secondary" size="sm" onClick={() => void handleBulk(ids, "activate")}>Aktifkan</Button>
            <Button variant="secondary" size="sm" onClick={() => void handleBulk(ids, "deactivate")}>Nonaktifkan</Button>
          </>
        )}
        actions={(venue) => (
          <RowActions
            rowLabel={venue.name}
            primary={{ label: "Ubah", icon: <Pencil />, onClick: () => openEditModal(venue), hidden: !canManage }}
            items={[
              venue.isActive
                ? { label: "Nonaktifkan", icon: <Ban />, onSelect: () => void handleToggleActive(venue), hidden: !canManage }
                : { label: "Aktifkan", icon: <CheckCircle2 />, onSelect: () => void handleToggleActive(venue), hidden: !canManage },
              { label: "Hapus permanen", icon: <Trash2 />, tone: "danger", onSelect: () => void openDeleteConfirm(venue), hidden: !isOwner },
            ]}
          />
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
        title={deleteTarget ? `${deleteImpact?.length || usedInOffers ? "Venue masih dipakai" : "Hapus venue"} — ${deleteTarget.name}` : "Hapus venue"}
        size="sm"
        busy={isDeleting}
        error={deleteError}
      >
        {impactLoading ? (
          <p className="text-[13px] text-text-secondary">Memeriksa penggunaan...</p>
        ) : (deleteImpact && deleteImpact.length > 0) || usedInOffers ? (
          // A venue a project still uses cannot be deleted (the server answers 409, QA FQ-02): say where, offer to deactivate.
          <div className="flex flex-col gap-4">
            <p className="flex items-start gap-2 text-[13px] text-text-secondary">
              <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0 text-warning-text" />
              <span>
                Venue ini masih dipakai di{" "}
                {deleteImpact && deleteImpact.length > 0 && <strong className="text-text-primary">{describeAffectedProjects(deleteImpact)}</strong>}
                {deleteImpact && deleteImpact.length > 0 && usedInOffers && " dan di "}
                {usedInOffers && <strong className="text-text-primary">paket atau penawaran</strong>}, jadi belum bisa dihapus.{" "}
                {deleteTarget?.isActive
                  ? usedInOffers
                    ? "Nonaktifkan saja: venue tidak bisa dipilih lagi untuk proyek, paket, atau penawaran baru, dan data yang sudah ada tetap utuh."
                    : "Nonaktifkan saja: venue tidak muncul lagi saat memilih venue proyek, dan data di proyek tadi tetap utuh."
                  : usedInOffers
                    ? "Venue ini sudah nonaktif, jadi tidak bisa dipilih lagi untuk proyek, paket, atau penawaran baru."
                    : "Venue ini sudah nonaktif, jadi tidak muncul lagi saat memilih venue proyek."}
              </span>
            </p>
            <div className="flex justify-end gap-2">
              <Button variant="secondary" onClick={closeDeleteConfirm} disabled={isDeleting}>
                {deleteTarget?.isActive && canManage ? "Batal" : "Tutup"}
              </Button>
              {deleteTarget?.isActive && canManage && (
                <Button onClick={() => void handleDeactivateInstead()} disabled={isDeleting}>
                  {isDeleting ? "Menyimpan..." : "Nonaktifkan venue"}
                </Button>
              )}
            </div>
          </div>
        ) : (
          <div className="flex flex-col gap-4">
            <p className="flex items-start gap-2 text-[13px] text-text-secondary">
              <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0 text-danger-text" />
              <span>Data ini akan dihapus secara permanen dan tidak dapat dikembalikan. Apakah Anda yakin?</span>
            </p>
            <div className="flex justify-end gap-2">
              <Button variant="secondary" onClick={closeDeleteConfirm} disabled={isDeleting}>
                Batal
              </Button>
              <Button variant="danger" onClick={() => void handleConfirmDelete()} disabled={isDeleting}>
                {isDeleting ? "Menghapus..." : "Hapus venue permanen"}
              </Button>
            </div>
          </div>
        )}
      </Modal>

      {importModalOpen && (
        <ImportBulkModal
          open={importModalOpen}
          onClose={() => setImportModalOpen(false)}
          title="Impor venue sekaligus"
          description="Unduh template Excel, isi datanya, lalu unggah kembali untuk menambah atau memperbarui banyak venue sekaligus."
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
