import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { Plus, Pencil, CheckCircle2, Ban, Eye, Upload, Download, Trash2, AlertTriangle } from "lucide-react";
import { Card } from "@/shared/components/ui/Card";
import { Button } from "@/shared/components/ui/Button";
import { SearchInput } from "@/shared/components/ui/SearchInput";
import { Select } from "@/shared/components/ui/Input";
import { Combobox } from "@/shared/components/ui/Combobox";
import { CITIES } from "@/shared/constants/cities";
import { Badge } from "@/shared/components/ui/Badge";
import { Modal } from "@/shared/components/ui/Modal";
import { ImportBulkModal } from "@/shared/components/ui/ImportBulkModal";
import { Table, THead, TBody, TR, TH, TD } from "@/shared/components/ui/Table";
import { CardList, CardListField } from "@/shared/components/ui/CardList";
import { Pagination } from "@/shared/components/ui/Pagination";
import { EmptyState } from "@/shared/components/feedback/EmptyState";
import { IconActionButton } from "@/shared/components/ui/IconActionButton";
import { VendorFormModal } from "@/modules/vendors/components/VendorFormModal";
import { VendorStatusBadge } from "@/modules/vendors/components/VendorStatusBadge";
import type { VendorFormValues, VendorCreateFormValues } from "@/modules/vendors/schemas/vendor.schema";
import { useVendorStore } from "@/modules/vendors/stores/useVendorStore";
import type { Vendor, VendorProjectHistoryItem } from "@/modules/vendors/types";
import { useVendorCategoryStore } from "@/modules/vendor-categories/stores/useVendorCategoryStore";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import { getApiErrorMessage } from "@/shared/lib/api-error";
import { formatDate } from "@/shared/lib/formatters";
import { useDebouncedValue } from "@/shared/hooks/useDebouncedValue";
import { useAuthStore } from "@/shared/stores/useAuthStore";
import type { AffectedProject } from "@/shared/types/delete-impact";

export default function VendorListPage() {
  const vendors = useVendorStore((s) => s.vendorPage);
  const meta = useVendorStore((s) => s.vendorPageMeta);
  const fetchVendorPage = useVendorStore((s) => s.fetchVendorPage);
  const createVendor = useVendorStore((s) => s.createVendor);
  const updateVendor = useVendorStore((s) => s.updateVendor);
  const toggleVendorActive = useVendorStore((s) => s.toggleVendorActive);
  const fetchVendorProjectHistory = useVendorStore((s) => s.fetchVendorProjectHistory);
  const downloadVendorTemplate = useVendorStore((s) => s.downloadVendorTemplate);
  const importVendors = useVendorStore((s) => s.importVendors);
  const exportVendors = useVendorStore((s) => s.exportVendors);
  const fetchVendorDeleteImpact = useVendorStore((s) => s.fetchVendorDeleteImpact);
  const deleteVendor = useVendorStore((s) => s.deleteVendor);
  const isOwner = useAuthStore((s) => s.session?.role === "Owner");
  const categories = useVendorCategoryStore((s) => s.categories);
  const fetchCategories = useVendorCategoryStore((s) => s.fetchCategories);

  const [query, setQuery] = useState("");
  const debouncedQuery = useDebouncedValue(query);
  const [categoryFilter, setCategoryFilter] = useState<string>("Semua");
  const [cityFilter, setCityFilter] = useState<string>("Semua");
  const [page, setPage] = useState(1);
  const [modalOpen, setModalOpen] = useState(false);
  const [editingVendor, setEditingVendor] = useState<Vendor | undefined>(undefined);
  const [historyVendor, setHistoryVendor] = useState<Vendor | null>(null);
  const [history, setHistory] = useState<VendorProjectHistoryItem[]>([]);
  const [historyLoading, setHistoryLoading] = useState(false);
  const [actionError, setActionError] = useState<string | null>(null);
  const [importModalOpen, setImportModalOpen] = useState(false);
  const [isExporting, setIsExporting] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<Vendor | null>(null);
  const [deleteImpact, setDeleteImpact] = useState<AffectedProject[] | null>(null);
  const [impactLoading, setImpactLoading] = useState(false);
  const [isDeleting, setIsDeleting] = useState(false);

  useEffect(() => {
    void fetchCategories();
  }, [fetchCategories]);

  useEffect(() => {
    setPage(1);
  }, [debouncedQuery, categoryFilter, cityFilter]);

  useEffect(() => {
    void fetchVendorPage(
      page,
      debouncedQuery,
      categoryFilter === "Semua" ? "" : categoryFilter,
      cityFilter === "Semua" ? "" : cityFilter
    );
  }, [fetchVendorPage, page, debouncedQuery, categoryFilter, cityFilter]);

  useEffect(() => {
    if (!historyVendor) {
      setHistory([]);
      return;
    }
    setHistoryLoading(true);
    void fetchVendorProjectHistory(historyVendor.id)
      .then(setHistory)
      .finally(() => setHistoryLoading(false));
  }, [historyVendor, fetchVendorProjectHistory]);

  function openCreateModal() {
    setEditingVendor(undefined);
    setModalOpen(true);
  }

  function openEditModal(vendor: Vendor) {
    setEditingVendor(vendor);
    setModalOpen(true);
  }

  function closeModal() {
    setModalOpen(false);
    setEditingVendor(undefined);
  }

  async function handleSubmit(values: VendorFormValues | VendorCreateFormValues) {
    setActionError(null);
    try {
      if (editingVendor) {
        await updateVendor(editingVendor.id, values as VendorFormValues);
      } else {
        await createVendor(values as VendorCreateFormValues);
      }
      closeModal();
      await fetchVendorPage(page, query, categoryFilter === "Semua" ? "" : categoryFilter, cityFilter === "Semua" ? "" : cityFilter);
    } catch (err) {
      setActionError(getApiErrorMessage(err, "Gagal menyimpan vendor"));
    }
  }

  async function handleToggleActive(vendor: Vendor) {
    setActionError(null);
    try {
      await toggleVendorActive(vendor.id);
      await fetchVendorPage(page, query, categoryFilter === "Semua" ? "" : categoryFilter, cityFilter === "Semua" ? "" : cityFilter);
    } catch (err) {
      setActionError(getApiErrorMessage(err, "Gagal mengubah status vendor"));
    }
  }

  async function handleImportVendors(file: File) {
    const result = await importVendors(file);
    await fetchVendorPage(page, query, categoryFilter === "Semua" ? "" : categoryFilter, cityFilter === "Semua" ? "" : cityFilter);
    return result;
  }

  // Hard delete (PLAN.md) — informed-consent, not a blocking precondition:
  // fetch exactly which projects would lose this vendor's data source before
  // the confirmation dialog renders, so its wording can name them.
  async function openDeleteConfirm(vendor: Vendor) {
    setDeleteTarget(vendor);
    setDeleteImpact(null);
    setImpactLoading(true);
    try {
      setDeleteImpact(await fetchVendorDeleteImpact(vendor.id));
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
      await deleteVendor(deleteTarget.id);
      closeDeleteConfirm();
      await fetchVendorPage(page, query, categoryFilter === "Semua" ? "" : categoryFilter, cityFilter === "Semua" ? "" : cityFilter);
    } catch (err) {
      setActionError(getApiErrorMessage(err, "Gagal menghapus vendor"));
      closeDeleteConfirm();
    } finally {
      setIsDeleting(false);
    }
  }

  async function handleExportVendors() {
    setActionError(null);
    setIsExporting(true);
    try {
      const blob = await exportVendors(query, categoryFilter === "Semua" ? "" : categoryFilter, cityFilter === "Semua" ? "" : cityFilter);
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = "vendors-export.xlsx";
      a.click();
      URL.revokeObjectURL(url);
    } catch (err) {
      setActionError(getApiErrorMessage(err, "Gagal mengekspor data vendor"));
    } finally {
      setIsExporting(false);
    }
  }

  return (
    <div className="flex flex-col gap-5">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-xl font-bold text-text-primary">Vendor</h1>
          <p className="mt-1 text-[13px] text-text-secondary">Kelola seluruh vendor yang bekerja sama dengan WO.</p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <Button variant="secondary" icon={<Upload className="h-4 w-4" />} onClick={() => setImportModalOpen(true)}>
            Import Bulk
          </Button>
          <Button variant="secondary" icon={<Download className="h-4 w-4" />} onClick={() => void handleExportVendors()} disabled={isExporting}>
            {isExporting ? "Mengekspor..." : "Export Excel"}
          </Button>
          <Button icon={<Plus className="h-4 w-4" />} onClick={openCreateModal}>
            Tambah Vendor
          </Button>
        </div>
      </div>

      {actionError && (
        <p className="rounded-md border border-danger/30 bg-danger-soft px-3.5 py-2.5 text-[13px] font-medium text-danger">{actionError}</p>
      )}

      <div className="flex flex-wrap gap-3">
        <SearchInput
          className="max-w-xs"
          placeholder="Cari nama vendor atau PIC..."
          value={query}
          onChange={(e) => setQuery(e.target.value)}
        />
        <Select className="w-52" value={categoryFilter} onChange={(e) => setCategoryFilter(e.target.value)}>
          <option value="Semua">Semua Kategori</option>
          {categories.map((c) => (
            <option key={c.id} value={c.id}>
              {c.name}
            </option>
          ))}
        </Select>
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
        {vendors.length === 0 ? (
          <EmptyState title="Tidak ada vendor ditemukan" description="Ubah kata kunci pencarian atau filter kategori." />
        ) : (
          <>
          <CardList
            className="sm:hidden"
            items={vendors}
            keyFor={(vendor) => vendor.id}
            renderItem={(vendor) => {
              const category = categories.find((c) => c.id === vendor.categoryId);
              return (
                <>
                  <div className="flex items-start justify-between gap-3">
                    <span className="font-semibold text-text-primary">{vendor.name}</span>
                    <VendorStatusBadge isActive={vendor.isActive} />
                  </div>
                  <div className="flex flex-col gap-1.5">
                    <CardListField label="Kategori" value={category?.name ?? "-"} />
                    <CardListField label="PIC" value={vendor.picName} />
                    <CardListField label="No Tlp Vendor" value={vendor.phone} />
                    <CardListField label="Kota" value={vendor.city ?? "-"} />
                  </div>
                  <div className="flex items-center gap-1.5 pt-1">
                    <IconActionButton icon={Pencil} label="Ubah Vendor" tone="neutral" onClick={() => openEditModal(vendor)} />
                    {vendor.isActive ? (
                      <IconActionButton icon={Ban} label="Nonaktifkan" tone="danger" onClick={() => void handleToggleActive(vendor)} />
                    ) : (
                      <IconActionButton icon={CheckCircle2} label="Aktifkan" tone="success" onClick={() => void handleToggleActive(vendor)} />
                    )}
                    <IconActionButton icon={Eye} label="Lihat Project" tone="info" onClick={() => setHistoryVendor(vendor)} />
                    {isOwner && (
                      <IconActionButton icon={Trash2} label="Hapus Permanen" tone="danger" onClick={() => void openDeleteConfirm(vendor)} />
                    )}
                  </div>
                </>
              );
            }}
          />
          <div className="hidden sm:block">
          <Table>
            <THead>
              <TR>
                <TH>Nama Vendor</TH>
                <TH>Kategori</TH>
                <TH>PIC</TH>
                <TH>Kontak</TH>
                <TH>Kota</TH>
                <TH>Status</TH>
                <TH>Aksi</TH>
              </TR>
            </THead>
            <TBody>
              {vendors.map((vendor) => {
                const category = categories.find((c) => c.id === vendor.categoryId);
                return (
                  <TR key={vendor.id}>
                    <TD className="font-semibold text-text-primary">{vendor.name}</TD>
                    <TD>{category?.name ?? "-"}</TD>
                    <TD>{vendor.picName}</TD>
                    <TD>
                      <span className="block">{vendor.phone}</span>
                      <span className="block text-[12.5px] text-text-secondary">{vendor.email ?? "-"}</span>
                    </TD>
                    <TD>{vendor.city ?? <span className="text-text-secondary">-</span>}</TD>
                    <TD>
                      <VendorStatusBadge isActive={vendor.isActive} />
                    </TD>
                    <TD>
                      <div className="flex items-center gap-1.5">
                        <IconActionButton icon={Pencil} label="Ubah Vendor" tone="neutral" onClick={() => openEditModal(vendor)} />
                        {vendor.isActive ? (
                          <IconActionButton icon={Ban} label="Nonaktifkan" tone="danger" onClick={() => void handleToggleActive(vendor)} />
                        ) : (
                          <IconActionButton icon={CheckCircle2} label="Aktifkan" tone="success" onClick={() => void handleToggleActive(vendor)} />
                        )}
                        <IconActionButton icon={Eye} label="Lihat Project" tone="info" onClick={() => setHistoryVendor(vendor)} />
                        {isOwner && (
                          <IconActionButton icon={Trash2} label="Hapus Permanen" tone="danger" onClick={() => void openDeleteConfirm(vendor)} />
                        )}
                      </div>
                    </TD>
                  </TR>
                );
              })}
            </TBody>
          </Table>
          </div>
          <Pagination page={meta.page} totalPages={meta.totalPages} totalItems={meta.total} pageSize={meta.limit} onPageChange={setPage} />
          </>
        )}
      </Card>

      <VendorFormModal
        key={editingVendor?.id ?? "new"}
        open={modalOpen}
        onClose={closeModal}
        onSubmitCreate={(values) => void handleSubmit(values)}
        onSubmitEdit={(values) => void handleSubmit(values)}
        initialVendor={editingVendor}
        categories={categories}
      />

      <Modal
        open={historyVendor !== null}
        onClose={() => setHistoryVendor(null)}
        title={historyVendor ? `Riwayat Project — ${historyVendor.name}` : "Riwayat Project"}
        description="Daftar project yang pernah menggunakan vendor ini."
        size="sm"
      >
        {historyLoading ? (
          <p className="text-[13px] text-text-secondary">Memuat...</p>
        ) : history.length === 0 ? (
          <p className="text-[13px] text-text-secondary">Belum digunakan di project manapun.</p>
        ) : (
          <ul className="flex flex-col gap-3">
            {history.map((item) => (
              <li
                key={item.projectId}
                className="flex items-center justify-between gap-3 border-b border-border-light pb-3 last:border-b-0 last:pb-0"
              >
                <div>
                  <Link
                    to={ROUTE_PATHS.projectDetail(item.projectId)}
                    className="text-[13.5px] font-semibold text-navy-900 hover:underline"
                    onClick={() => setHistoryVendor(null)}
                  >
                    {item.projectName}
                  </Link>
                  <p className="text-[12px] text-text-secondary">{formatDate(item.eventDate)} · {item.venue}</p>
                </div>
                <Badge tone="neutral">{item.engagementStatus}</Badge>
              </li>
            ))}
          </ul>
        )}
      </Modal>

      <Modal
        open={deleteTarget !== null}
        onClose={closeDeleteConfirm}
        title={deleteTarget ? `Hapus Vendor — ${deleteTarget.name}` : "Hapus Vendor"}
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
                  Vendor ini masih digunakan oleh project berikut:{" "}
                  <strong className="text-text-primary">{deleteImpact.map((p) => p.name).join(", ")}</strong>. Jika dihapus, data
                  vendor pada project tersebut akan kehilangan sumber datanya dan tidak dapat ditelusuri lagi. Apakah Anda yakin ingin
                  menghapus vendor ini secara permanen?
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
          title="Import Bulk Vendor"
          description="Download template Excel, isi datanya, lalu unggah kembali untuk menambah atau memperbarui banyak vendor sekaligus."
          templateFileName="template-vendor.xlsx"
          entityLabel="vendor"
          matchCriteriaLabel="nama+kota+kategori"
          onDownloadTemplate={downloadVendorTemplate}
          onImport={handleImportVendors}
        />
      )}
    </div>
  );
}
