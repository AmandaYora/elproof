import { useCallback, useEffect, useState } from "react";
import { useCan, useWriteLocked } from "@/shared/stores/usePermissionStore";
import { Link } from "react-router-dom";
import { Plus, Pencil, CheckCircle2, Ban, Eye, Upload, Download, Trash2, AlertTriangle } from "lucide-react";
import { Button } from "@/shared/components/ui/Button";
import { Select } from "@/shared/components/ui/Input";
import { Combobox } from "@/shared/components/ui/Combobox";
import { CITIES } from "@/shared/constants/cities";
import { Badge } from "@/shared/components/ui/Badge";
import { Modal } from "@/shared/components/ui/Modal";
import { ImportBulkModal } from "@/shared/components/ui/ImportBulkModal";
import { DataTable, type Column } from "@/shared/data-table/DataTable";
import { ListToolbar } from "@/shared/data-table/ListToolbar";
import { useListState } from "@/shared/data-table/useListState";
import { useListDensity } from "@/shared/data-table/useListDensity";
import { useSavedViews, viewQuery } from "@/shared/data-table/useSavedViews";
import { toast } from "@/shared/stores/useToastStore";
import { IconActionButton } from "@/shared/components/ui/IconActionButton";
import { VendorFormModal } from "@/modules/vendors/components/VendorFormModal";
import { VendorStatusBadge } from "@/modules/vendors/components/VendorStatusBadge";
import type { VendorFormValues, VendorCreateFormValues } from "@/modules/vendors/schemas/vendor.schema";
import { useVendorStore } from "@/modules/vendors/stores/useVendorStore";
import type { Vendor, VendorProjectHistoryItem } from "@/modules/vendors/types";
import { useVendorCategoryStore } from "@/modules/vendor-categories/stores/useVendorCategoryStore";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import { getApiErrorMessage } from "@/shared/lib/api-error";
import { formatCurrency, formatDate } from "@/shared/lib/formatters";
import type { AffectedProject } from "@/shared/types/delete-impact";

export default function VendorListPage() {
  // Read-only workspace (subscription): no new records until it is renewed.
  const writeLocked = useWriteLocked();
  const vendors = useVendorStore((s) => s.vendorPage);
  const bulkVendors = useVendorStore((s) => s.bulkVendors);
  const meta = useVendorStore((s) => s.vendorPageMeta);
  const fetchVendorPage = useVendorStore((s) => s.fetchVendorPage);
  const createVendor = useVendorStore((s) => s.createVendor);
  const updateVendor = useVendorStore((s) => s.updateVendor);
  const setVendorActive = useVendorStore((s) => s.setVendorActive);
  const fetchVendorProjectHistory = useVendorStore((s) => s.fetchVendorProjectHistory);
  const downloadVendorTemplate = useVendorStore((s) => s.downloadVendorTemplate);
  const importVendors = useVendorStore((s) => s.importVendors);
  const exportVendors = useVendorStore((s) => s.exportVendors);
  const fetchVendorDeleteImpact = useVendorStore((s) => s.fetchVendorDeleteImpact);
  const deleteVendor = useVendorStore((s) => s.deleteVendor);
  // Destructive/management actions follow the role's permissions (ADR-0026).
  const isOwner = useCan("vendor.delete");
  const categories = useVendorCategoryStore((s) => s.categories);
  const fetchCategories = useVendorCategoryStore((s) => s.fetchCategories);

  // Search, filters, sort and page live in the URL (shareable, survives reload and Back).
  const list = useListState({ filters: { categoryId: "", city: "", isActive: "" } });
  const { density, setDensity } = useListDensity();
  const savedViews = useSavedViews("vendors");
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [status, setStatus] = useState<"loading" | "error" | "ready">("loading");
  const [attempt, setAttempt] = useState(0);
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
    let cancelled = false;
    setStatus("loading");
    fetchVendorPage(list.params)
      .then(() => !cancelled && setStatus("ready"))
      .catch(() => !cancelled && setStatus("error"));
    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [fetchVendorPage, list.paramsKey, attempt]);

  // What was selected on another page or before a filter changed is no longer in view: start clean.
  useEffect(() => {
    setSelected(new Set());
  }, [list.paramsKey]);

  const reload = useCallback(() => fetchVendorPage(list.params), [fetchVendorPage, list.params]);

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

  // Errors are not caught here: they propagate to the form, which shows a 422 on its fields.
  async function handleSubmit(values: VendorFormValues | VendorCreateFormValues) {
    setActionError(null);
    if (editingVendor) {
      await updateVendor(editingVendor.id, values as VendorFormValues);
    } else {
      await createVendor(values as VendorCreateFormValues);
    }
    closeModal();
    await reload();
  }

  async function handleToggleActive(vendor: Vendor) {
    setActionError(null);
    try {
      await setVendorActive(vendor.id, !vendor.isActive);
      await reload();
    } catch (err) {
      setActionError(getApiErrorMessage(err, "Gagal mengubah status vendor"));
    }
  }

  async function handleImportVendors(file: File) {
    const result = await importVendors(file);
    await reload();
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
      await reload();
    } catch (err) {
      setActionError(getApiErrorMessage(err, "Gagal menghapus vendor"));
      closeDeleteConfirm();
    } finally {
      setIsDeleting(false);
    }
  }

  async function handleBulk(ids: string[], action: "activate" | "deactivate") {
    setActionError(null);
    try {
      const result = await bulkVendors(ids, action);
      setSelected(new Set());
      await reload();
      const verb = action === "activate" ? "diaktifkan" : "dinonaktifkan";
      toast.success(result.updated > 0 ? `${result.updated} vendor ${verb}` : `Semua vendor terpilih sudah ${action === "activate" ? "aktif" : "nonaktif"}`);
    } catch (err) {
      setActionError(getApiErrorMessage(err, "Vendor terpilih belum bisa diubah. Coba lagi."));
    }
  }

  const columns: Column<Vendor>[] = [
    { id: "name", header: "Nama vendor", sortKey: "name", primary: true, cell: (v) => <span className="font-semibold">{v.name}</span> },
    { id: "category", header: "Kategori", sortKey: "categoryName", cell: (v) => v.categoryName || categories.find((c) => c.id === v.categoryId)?.name || "-" },
    { id: "pic", header: "PIC", cell: (v) => v.picName },
    {
      id: "contact",
      header: "Kontak",
      hideOnCard: false,
      cell: (v) => (
        <>
          <span className="block">{v.phone}</span>
          <span className="block text-[12.5px] text-muted-foreground">{v.email ?? "-"}</span>
        </>
      ),
    },
    { id: "city", header: "Kota", sortKey: "city", cell: (v) => v.city ?? <span className="text-muted-foreground">-</span> },
    { id: "price", header: "Harga akad", sortKey: "priceAkad", align: "right", cell: (v) => (v.priceAkad ? formatCurrency(v.priceAkad) : "-") },
    { id: "status", header: "Status", sortKey: "isActive", cell: (v) => <VendorStatusBadge isActive={v.isActive} /> },
  ];

  async function handleExportVendors() {
    setActionError(null);
    setIsExporting(true);
    try {
      const blob = await exportVendors({ q: list.q, ...list.filters });
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
          {!writeLocked && (
            <Button variant="secondary" icon={<Upload className="h-4 w-4" />} onClick={() => setImportModalOpen(true)}>
              Impor bulk
            </Button>
          )}
          <Button variant="secondary" icon={<Download className="h-4 w-4" />} onClick={() => void handleExportVendors()} disabled={isExporting}>
            {isExporting ? "Mengekspor..." : "Ekspor ke Excel"}
          </Button>
          {!writeLocked && (
            <Button icon={<Plus className="h-4 w-4" />} onClick={openCreateModal}>
              Tambah vendor
            </Button>
          )}
        </div>
      </div>

      {actionError && (
        <p className="rounded-md border border-danger/30 bg-danger-soft px-3.5 py-2.5 text-[13px] font-medium text-danger-text">{actionError}</p>
      )}

      <ListToolbar
        search={{ value: list.q, onChange: list.setQ, placeholder: "Cari nama vendor atau PIC..." }}
        density={{ value: density, onChange: setDensity }}
        views={{
          items: savedViews.views,
          current: viewQuery(list.queryString),
          onApply: list.applyQueryString,
          onSave: savedViews.save,
          onRemove: savedViews.remove,
        }}
      >
        <Select className="w-52" aria-label="Kategori" value={list.filters.categoryId || "Semua"} onChange={(e) => list.setFilter("categoryId", e.target.value === "Semua" ? "" : e.target.value)}>
          <option value="Semua">Semua kategori</option>
          {categories.map((c) => (
            <option key={c.id} value={c.id}>
              {c.name}
            </option>
          ))}
        </Select>
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
        label="Daftar vendor"
        columns={columns}
        rows={vendors}
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
          title: "Belum ada vendor",
          description: "Daftarkan vendor katering, dekorasi, foto, dan lainnya agar bisa dipasang ke proyek.",
          action: !writeLocked ? <Button icon={<Plus className="h-4 w-4" />} onClick={openCreateModal}>Tambah vendor</Button> : undefined,
        }}
        density={density}
        selection={writeLocked ? undefined : { selected, onChange: setSelected }}
        bulkActions={(ids) => (
          <>
            <Button variant="secondary" size="sm" onClick={() => void handleBulk(ids, "activate")}>Aktifkan</Button>
            <Button variant="secondary" size="sm" onClick={() => void handleBulk(ids, "deactivate")}>Nonaktifkan</Button>
          </>
        )}
        actions={(vendor) => (
          <div className="flex items-center gap-1.5">
            <IconActionButton icon={Pencil} label="Ubah vendor" tone="neutral" onClick={() => openEditModal(vendor)} />
            {vendor.isActive ? (
              <IconActionButton icon={Ban} label="Nonaktifkan" tone="danger" onClick={() => void handleToggleActive(vendor)} />
            ) : (
              <IconActionButton icon={CheckCircle2} label="Aktifkan" tone="success" onClick={() => void handleToggleActive(vendor)} />
            )}
            <IconActionButton icon={Eye} readOnly label="Lihat proyek" tone="info" onClick={() => setHistoryVendor(vendor)} />
            {isOwner && <IconActionButton icon={Trash2} label="Hapus permanen" tone="danger" onClick={() => void openDeleteConfirm(vendor)} />}
          </div>
        )}
      />

      <VendorFormModal
        key={editingVendor?.id ?? "new"}
        open={modalOpen}
        onClose={closeModal}
        onSubmitCreate={handleSubmit}
        onSubmitEdit={handleSubmit}
        initialVendor={editingVendor}
        categories={categories}
      />

      <Modal
        open={historyVendor !== null}
        onClose={() => setHistoryVendor(null)}
        title={historyVendor ? `Riwayat proyek — ${historyVendor.name}` : "Riwayat proyek"}
        description="Daftar proyek yang pernah menggunakan vendor ini."
        size="sm"
      >
        {historyLoading ? (
          <p className="text-[13px] text-text-secondary">Memuat...</p>
        ) : history.length === 0 ? (
          <p className="text-[13px] text-text-secondary">Belum digunakan di proyek mana pun.</p>
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
        title={deleteTarget ? `Hapus vendor — ${deleteTarget.name}` : "Hapus vendor"}
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
                  Vendor ini masih digunakan oleh proyek berikut:{" "}
                  <strong className="text-text-primary">{deleteImpact.map((p) => p.name).join(", ")}</strong>. Jika dihapus, data
                  vendor pada proyek tersebut akan kehilangan sumber datanya dan tidak dapat ditelusuri lagi. Apakah Anda yakin ingin
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
                {isDeleting ? "Menghapus..." : "Hapus vendor permanen"}
              </Button>
            </div>
          </div>
        )}
      </Modal>

      {importModalOpen && (
        <ImportBulkModal
          open={importModalOpen}
          onClose={() => setImportModalOpen(false)}
          title="Impor vendor sekaligus"
          description="Unduh template Excel, isi datanya, lalu unggah kembali untuk menambah atau memperbarui banyak vendor sekaligus."
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
