import { useEffect, useState } from "react";
import { Plus, Pencil, CheckCircle2, Ban, Upload } from "lucide-react";
import { Card } from "@/shared/components/ui/Card";
import { Button } from "@/shared/components/ui/Button";
import { SearchInput } from "@/shared/components/ui/SearchInput";
import { ImportBulkModal } from "@/shared/components/ui/ImportBulkModal";
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

export default function VenueListPage() {
  const venues = useVenueStore((s) => s.venuePage);
  const meta = useVenueStore((s) => s.venuePageMeta);
  const fetchVenuePage = useVenueStore((s) => s.fetchVenuePage);
  const createVenue = useVenueStore((s) => s.createVenue);
  const updateVenue = useVenueStore((s) => s.updateVenue);
  const toggleVenueActive = useVenueStore((s) => s.toggleVenueActive);
  const downloadVenueTemplate = useVenueStore((s) => s.downloadVenueTemplate);
  const importVenues = useVenueStore((s) => s.importVenues);

  const [query, setQuery] = useState("");
  const [page, setPage] = useState(1);
  const [modalOpen, setModalOpen] = useState(false);
  const [editingVenue, setEditingVenue] = useState<Venue | undefined>(undefined);
  const [actionError, setActionError] = useState<string | null>(null);
  const [importModalOpen, setImportModalOpen] = useState(false);

  useEffect(() => {
    setPage(1);
  }, [query]);

  useEffect(() => {
    void fetchVenuePage(page, query);
  }, [fetchVenuePage, page, query]);

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
      await fetchVenuePage(page, query);
    } catch (err) {
      setActionError(getApiErrorMessage(err, "Gagal menyimpan venue"));
    }
  }

  async function handleToggleActive(venue: Venue) {
    setActionError(null);
    try {
      await toggleVenueActive(venue.id);
      await fetchVenuePage(page, query);
    } catch (err) {
      setActionError(getApiErrorMessage(err, "Gagal mengubah status venue"));
    }
  }

  async function handleImportVenues(file: File) {
    const result = await importVenues(file);
    await fetchVenuePage(page, query);
    return result;
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
