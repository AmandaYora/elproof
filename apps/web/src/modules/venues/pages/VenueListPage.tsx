import { useEffect, useState } from "react";
import { Plus, Pencil, CheckCircle2, Ban, Upload, Download, Trash2, AlertTriangle } from "lucide-react";
import { Card } from "@/shared/components/ui/Card";
import { Button } from "@/shared/components/ui/Button";
import { SearchInput } from "@/shared/components/ui/SearchInput";
import { Combobox } from "@/shared/components/ui/Combobox";
import { CITIES } from "@/shared/constants/cities";
import { ImportBulkModal } from "@/shared/components/ui/ImportBulkModal";
import { Modal } from "@/shared/components/ui/Modal";
import { Table, THead, TBody, TR, TH, TD } from "@/shared/components/ui/Table";
import { CardList, CardListField } from "@/shared/components/ui/CardList";
import { Pagination } from "@/shared/components/ui/Pagination";
import { EmptyState } from "@/shared/components/feedback/EmptyState";
import { IconActionButton } from "@/shared/components/ui/IconActionButton";
import { VenueFormModal } from "@/modules/venues/components/VenueFormModal";
import { VenueStatusBadge } from "@/modules/venues/components/VenueStatusBadge";
import type { VenueFormValues, VenueCreateFormValues } from "@/modules/venues/schemas/venue.schema";
import { useVenueStore } from "@/modules/venues/stores/useVenueStore";
import type { Venue } from "@/modules/venues/types";
import { getApiErrorMessage } from "@/shared/lib/api-error";
import { formatCurrency } from "@/shared/lib/formatters";
import { useDebouncedValue } from "@/shared/hooks/useDebouncedValue";
import { useAuthStore } from "@/shared/stores/useAuthStore";
import type { AffectedProject } from "@/shared/types/delete-impact";

export default function VenueListPage() {
  const venues = useVenueStore((s) => s.venuePage);
  const meta = useVenueStore((s) => s.venuePageMeta);
  const fetchVenuePage = useVenueStore((s) => s.fetchVenuePage);
  const createVenue = useVenueStore((s) => s.createVenue);
  const updateVenue = useVenueStore((s) => s.updateVenue);
  const toggleVenueActive = useVenueStore((s) => s.toggleVenueActive);
  const downloadVenueTemplate = useVenueStore((s) => s.downloadVenueTemplate);
  const importVenues = useVenueStore((s) => s.importVenues);
  const exportVenues = useVenueStore((s) => s.exportVenues);
  const fetchVenueDeleteImpact = useVenueStore((s) => s.fetchVenueDeleteImpact);
  const deleteVenue = useVenueStore((s) => s.deleteVenue);
  const isOwner = useAuthStore((s) => s.session?.role === "Owner");

  const [query, setQuery] = useState("");
  const debouncedQuery = useDebouncedValue(query);
  const [cityFilter, setCityFilter] = useState<string>("Semua");
  const [page, setPage] = useState(1);
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
    setPage(1);
  }, [debouncedQuery, cityFilter]);

  useEffect(() => {
    void fetchVenuePage(page, debouncedQuery, cityFilter === "Semua" ? "" : cityFilter);
  }, [fetchVenuePage, page, debouncedQuery, cityFilter]);

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

  async function handleSubmit(values: VenueFormValues | VenueCreateFormValues) {
    setActionError(null);
    try {
      if (editingVenue) {
        await updateVenue(editingVenue.id, values as VenueFormValues);
      } else {
        await createVenue(values as VenueCreateFormValues);
      }
      closeModal();
      await fetchVenuePage(page, query, cityFilter === "Semua" ? "" : cityFilter);
    } catch (err) {
      setActionError(getApiErrorMessage(err, "Gagal menyimpan venue"));
    }
  }

  async function handleToggleActive(venue: Venue) {
    setActionError(null);
    try {
      await toggleVenueActive(venue.id);
      await fetchVenuePage(page, query, cityFilter === "Semua" ? "" : cityFilter);
    } catch (err) {
      setActionError(getApiErrorMessage(err, "Gagal mengubah status venue"));
    }
  }

  async function handleImportVenues(file: File) {
    const result = await importVenues(file);
    await fetchVenuePage(page, query, cityFilter === "Semua" ? "" : cityFilter);
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
      await fetchVenuePage(page, query, cityFilter === "Semua" ? "" : cityFilter);
    } catch (err) {
      setActionError(getApiErrorMessage(err, "Gagal menghapus venue"));
      closeDeleteConfirm();
    } finally {
      setIsDeleting(false);
    }
  }

  async function handleExportVenues() {
    setActionError(null);
    setIsExporting(true);
    try {
      const blob = await exportVenues(query, cityFilter === "Semua" ? "" : cityFilter);
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
          <Button variant="secondary" icon={<Upload className="h-4 w-4" />} onClick={() => setImportModalOpen(true)}>
            Import Bulk
          </Button>
          <Button variant="secondary" icon={<Download className="h-4 w-4" />} onClick={() => void handleExportVenues()} disabled={isExporting}>
            {isExporting ? "Mengekspor..." : "Export Excel"}
          </Button>
          <Button icon={<Plus className="h-4 w-4" />} onClick={openCreateModal}>
            Tambah Venue
          </Button>
        </div>
      </div>

      {actionError && (
        <p className="rounded-md border border-danger/30 bg-danger-soft px-3.5 py-2.5 text-[13px] font-medium text-danger">{actionError}</p>
      )}

      <div className="flex flex-wrap gap-3">
        <SearchInput className="max-w-xs" placeholder="Cari nama venue atau PIC..." value={query} onChange={(e) => setQuery(e.target.value)} />
        <Combobox className="w-52" value={cityFilter} onChange={(e) => setCityFilter(e.target.value)} placeholder="Cari kota...">
          <option value="Semua">Semua Kota</option>
          {CITIES.map((city) => (
            <option key={city} value={city}>
              {city}
            </option>
          ))}
        </Combobox>
      </div>

      <Card>
        {venues.length === 0 ? (
          <EmptyState title="Tidak ada venue ditemukan" description="Ubah kata kunci pencarian, atau tambah venue baru." />
        ) : (
          <>
            <CardList
              className="sm:hidden"
              items={venues}
              keyFor={(venue) => venue.id}
              renderItem={(venue) => (
                <>
                  <div className="flex items-start justify-between gap-3">
                    <span className="font-semibold text-text-primary">{venue.name}</span>
                    <VenueStatusBadge isActive={venue.isActive} />
                  </div>
                  <div className="flex flex-col gap-1.5">
                    <CardListField label="PIC" value={venue.picName} />
                    <CardListField label="No Tlp PIC" value={venue.phonePic} />
                    <CardListField label="Kota" value={venue.city ?? "-"} />
                    <CardListField label="Harga Sewa" value={venue.rentalPrice ? formatCurrency(venue.rentalPrice) : "Belum diisi"} />
                    <CardListField label="Kapasitas" value={venue.capacity ? `${venue.capacity} orang` : "-"} />
                  </div>
                  <div className="flex items-center gap-1.5 pt-1">
                    <IconActionButton icon={Pencil} label="Ubah Venue" tone="neutral" onClick={() => openEditModal(venue)} />
                    {venue.isActive ? (
                      <IconActionButton icon={Ban} label="Nonaktifkan" tone="danger" onClick={() => void handleToggleActive(venue)} />
                    ) : (
                      <IconActionButton icon={CheckCircle2} label="Aktifkan" tone="success" onClick={() => void handleToggleActive(venue)} />
                    )}
                    {isOwner && (
                      <IconActionButton icon={Trash2} label="Hapus Permanen" tone="danger" onClick={() => void openDeleteConfirm(venue)} />
                    )}
                  </div>
                </>
              )}
            />
            <div className="hidden sm:block">
              <Table>
                <THead>
                  <TR>
                    <TH>Nama Venue</TH>
                    <TH>PIC</TH>
                    <TH>Kota</TH>
                    <TH>Harga Sewa</TH>
                    <TH>Kapasitas</TH>
                    <TH>Status</TH>
                    <TH>Aksi</TH>
                  </TR>
                </THead>
                <TBody>
                  {venues.map((venue) => (
                    <TR key={venue.id}>
                      <TD className="font-semibold text-text-primary">{venue.name}</TD>
                      <TD>
                        <span className="block">{venue.picName}</span>
                        <span className="block text-[12.5px] text-text-secondary">{venue.phonePic}</span>
                      </TD>
                      <TD>{venue.city ?? <span className="text-text-secondary">-</span>}</TD>
                      <TD>{venue.rentalPrice ? formatCurrency(venue.rentalPrice) : <span className="text-text-secondary">Belum diisi</span>}</TD>
                      <TD>{venue.capacity ? `${venue.capacity} orang` : "-"}</TD>
                      <TD>
                        <VenueStatusBadge isActive={venue.isActive} />
                      </TD>
                      <TD>
                        <div className="flex items-center gap-1.5">
                          <IconActionButton icon={Pencil} label="Ubah Venue" tone="neutral" onClick={() => openEditModal(venue)} />
                          {venue.isActive ? (
                            <IconActionButton icon={Ban} label="Nonaktifkan" tone="danger" onClick={() => void handleToggleActive(venue)} />
                          ) : (
                            <IconActionButton icon={CheckCircle2} label="Aktifkan" tone="success" onClick={() => void handleToggleActive(venue)} />
                          )}
                          {isOwner && (
                            <IconActionButton icon={Trash2} label="Hapus Permanen" tone="danger" onClick={() => void openDeleteConfirm(venue)} />
                          )}
                        </div>
                      </TD>
                    </TR>
                  ))}
                </TBody>
              </Table>
            </div>
            <Pagination page={meta.page} totalPages={meta.totalPages} totalItems={meta.total} pageSize={meta.limit} onPageChange={setPage} />
          </>
        )}
      </Card>

      <VenueFormModal
        key={editingVenue?.id ?? "new"}
        open={modalOpen}
        onClose={closeModal}
        onSubmitCreate={(values) => void handleSubmit(values)}
        onSubmitEdit={(values) => void handleSubmit(values)}
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
              <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0 text-danger" />
              {deleteImpact && deleteImpact.length > 0 ? (
                <span>
                  Venue ini masih digunakan oleh project berikut:{" "}
                  <strong className="text-text-primary">{deleteImpact.map((p) => p.name).join(", ")}</strong>. Jika dihapus, data
                  venue pada project tersebut akan kehilangan sumber datanya dan tidak dapat ditelusuri lagi. Apakah Anda yakin ingin
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
