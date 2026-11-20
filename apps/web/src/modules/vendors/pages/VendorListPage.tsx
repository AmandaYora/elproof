import { useCallback, useEffect, useState } from "react";
import { useCan } from "@/shared/stores/usePermissionStore";
import { Link } from "react-router-dom";
import { Plus, Pencil, CheckCircle2, Ban, FolderKanban, Upload, Download, Trash2, AlertTriangle } from "lucide-react";
import { PageHeader } from "@/shared/components/ui/PageHeader";
import { RowActions } from "@/shared/components/ui/RowActions";
import { ActionMenu } from "@/shared/components/ui/ActionMenu";
import { ListFilters, activeFilterCount } from "@/shared/data-table/ListFilters";
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
import { VendorFormModal } from "@/modules/vendors/components/VendorFormModal";
import { VendorStatusBadge } from "@/modules/vendors/components/VendorStatusBadge";
import type { VendorFormValues, VendorCreateFormValues } from "@/modules/vendors/schemas/vendor.schema";
import { useVendorStore } from "@/modules/vendors/stores/useVendorStore";
import type { Vendor, VendorProjectHistoryItem } from "@/modules/vendors/types";
import { useVendorCategoryStore } from "@/modules/vendor-categories/stores/useVendorCategoryStore";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import { getApiErrorMessage } from "@/shared/lib/api-error";
import { formatCurrency, formatDate } from "@/shared/lib/formatters";
import { describeAffectedProjects, inUseProjectsOf, isInOffersRefusal, type AffectedProject } from "@/shared/types/delete-impact";

export default function VendorListPage() {
  // Adding, editing, importing and (de)activating need vendor.manage (ADR-0026); useCan also reads "no" while the
  // workspace is read-only (subscription), so the write buttons disappear then too.
  const canManage = useCan("vendor.manage");
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
  // A package or quotation still refers to it (PLAN paket-wo §3.3): the server refuses the delete, like a project does.
  const [usedInOffers, setUsedInOffers] = useState(false);
  // Why the server refused the delete (e.g. 409: still used by a project), shown inside the dialog that asked.
  const [deleteError, setDeleteError] = useState<string | null>(null);
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
    setUsedInOffers(false);
    setImpactLoading(true);
    try {
      const impact = await fetchVendorDeleteImpact(vendor.id);
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
      await deleteVendor(deleteTarget.id);
      closeDeleteConfirm();
      await reload();
    } catch (err) {
      // A project started using it since the check (409 vendor_in_use): the dialog turns into "deactivate instead".
      const inUse = inUseProjectsOf(err);
      if (inUse) setDeleteImpact(inUse);
      // Or a package or quotation refers to it now (409 *_in_offers): the same "deactivate instead".
      if (isInOffersRefusal(err)) setUsedInOffers(true);
      setDeleteError(getApiErrorMessage(err, "Vendor belum bisa dihapus. Periksa koneksi Anda, lalu coba lagi."));
    } finally {
      setIsDeleting(false);
    }
  }

  /** What the dialog offers for a vendor a project still uses (QA FQ-02): it stays in those projects, out of new ones. */
  async function handleDeactivateInstead() {
    if (!deleteTarget) return;
    setIsDeleting(true);
    setDeleteError(null);
    try {
      await setVendorActive(deleteTarget.id, false);
      closeDeleteConfirm();
      toast.success("Vendor dinonaktifkan");
      await reload();
    } catch (err) {
      setDeleteError(getApiErrorMessage(err, "Vendor belum bisa dinonaktifkan. Periksa koneksi Anda, lalu coba lagi."));
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

  // There is no vendor page of its own: the name opens the vendor's details (the form) for someone who may change them,
  // and the projects it worked on for everyone else.
  const openVendor = (vendor: Vendor) => (canManage ? openEditModal(vendor) : setHistoryVendor(vendor));

  const columns: Column<Vendor>[] = [
    {
      id: "name",
      header: "Nama vendor",
      sortKey: "name",
      primary: true,
      truncate: true,
      className: "w-[34%] min-w-60",
      cell: (v) => (
        <>
          <button
            type="button"
            onClick={() => openVendor(v)}
            className="block max-w-full truncate text-left font-semibold underline-offset-2 hover:underline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-solid focus-visible:outline-ring"
          >
            {v.name}
          </button>
          <span className="block truncate text-[12.5px] font-normal text-muted-foreground">
            {[v.picName, v.phone].filter(Boolean).join(" · ")}
          </span>
        </>
      ),
    },
    {
      id: "category",
      header: "Kategori",
      sortKey: "categoryName",
      card: true,
      truncate: true,
      className: "w-[20%]",
      cell: (v) => v.categoryName || categories.find((c) => c.id === v.categoryId)?.name || "-",
    },
    { id: "city", header: "Kota", sortKey: "city", card: true, truncate: true, className: "w-[14%]", cell: (v) => v.city ?? <span className="text-muted-foreground">-</span> },
    { id: "price", header: "Harga Akad", sortKey: "priceAkad", align: "right", card: true, className: "whitespace-nowrap", cell: (v) => (v.priceAkad ? formatCurrency(v.priceAkad) : "-") },
    { id: "status", header: "Status", sortKey: "isActive", card: true, className: "whitespace-nowrap", cell: (v) => <VendorStatusBadge isActive={v.isActive} /> },
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
      <PageHeader
        className="mb-1"
        title="Vendor"
        description="Katering, dekorasi, foto, dan rekanan lain yang bisa Anda pasang ke proyek."
        actions={
          <>
            {canManage && (
              <Button variant="secondary" className="max-sm:hidden" icon={<Upload className="h-4 w-4" />} onClick={() => setImportModalOpen(true)}>
                Impor bulk
              </Button>
            )}
            <Button variant="secondary" className="max-sm:hidden" icon={<Download className="h-4 w-4" />} onClick={() => void handleExportVendors()} disabled={isExporting}>
              {isExporting ? "Mengekspor..." : "Ekspor ke Excel"}
            </Button>
            {canManage && (
              <Button icon={<Plus className="h-4 w-4" />} onClick={openCreateModal}>
                Tambah vendor
              </Button>
            )}
            {/* On a phone, Impor and Ekspor sit behind "⋯" so the header stays one row. */}
            <div className="sm:hidden">
              <ActionMenu
                label="Impor dan ekspor vendor"
                disabled={isExporting}
                items={[
                  { label: "Impor bulk", icon: <Upload />, onSelect: () => setImportModalOpen(true), hidden: !canManage },
                  { label: "Ekspor ke Excel", icon: <Download />, onSelect: () => void handleExportVendors() },
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
        <ListFilters count={activeFilterCount(list.filters)} onClear={list.clearFilters}>
          <Select className="w-44" aria-label="Kategori" value={list.filters.categoryId || "Semua"} onChange={(e) => list.setFilter("categoryId", e.target.value === "Semua" ? "" : e.target.value)}>
            <option value="Semua">Semua kategori</option>
            {categories.map((c) => (
              <option key={c.id} value={c.id}>
                {c.name}
              </option>
            ))}
          </Select>
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
          action: canManage ? <Button icon={<Plus className="h-4 w-4" />} onClick={openCreateModal}>Tambah vendor</Button> : undefined,
        }}
        density={density}
        selection={!canManage ? undefined : { selected, onChange: setSelected }}
        bulkActions={(ids) => (
          <>
            <Button variant="secondary" size="sm" onClick={() => void handleBulk(ids, "activate")}>Aktifkan</Button>
            <Button variant="secondary" size="sm" onClick={() => void handleBulk(ids, "deactivate")}>Nonaktifkan</Button>
          </>
        )}
        actions={(vendor) => (
          <RowActions
            rowLabel={vendor.name}
            primary={{ label: "Ubah", icon: <Pencil />, onClick: () => openEditModal(vendor), hidden: !canManage }}
            items={[
              { label: "Lihat proyek", icon: <FolderKanban />, onSelect: () => setHistoryVendor(vendor) },
              vendor.isActive
                ? { label: "Nonaktifkan", icon: <Ban />, onSelect: () => void handleToggleActive(vendor), hidden: !canManage }
                : { label: "Aktifkan", icon: <CheckCircle2 />, onSelect: () => void handleToggleActive(vendor), hidden: !canManage },
              { label: "Hapus permanen", icon: <Trash2 />, tone: "danger", onSelect: () => void openDeleteConfirm(vendor), hidden: !isOwner },
            ]}
          />
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
                    className="text-[13.5px] font-semibold text-primary hover:underline"
                    onClick={() => setHistoryVendor(null)}
                  >
                    {item.projectName}
                  </Link>
                  <p className="text-[12px] text-text-secondary">{[formatDate(item.eventDate), item.venue].filter(Boolean).join(" · ")}</p>
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
        title={deleteTarget ? `${deleteImpact?.length || usedInOffers ? "Vendor masih dipakai" : "Hapus vendor"} — ${deleteTarget.name}` : "Hapus vendor"}
        size="sm"
        busy={isDeleting}
        error={deleteError}
      >
        {impactLoading ? (
          <p className="text-[13px] text-text-secondary">Memeriksa penggunaan...</p>
        ) : (deleteImpact && deleteImpact.length > 0) || usedInOffers ? (
          // A vendor a project still uses cannot be deleted (the server answers 409, QA FQ-02): say where, offer to deactivate.
          <div className="flex flex-col gap-4">
            <p className="flex items-start gap-2 text-[13px] text-text-secondary">
              <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0 text-warning-text" />
              <span>
                Vendor ini masih dipakai di{" "}
                {deleteImpact && deleteImpact.length > 0 && <strong className="text-text-primary">{describeAffectedProjects(deleteImpact)}</strong>}
                {deleteImpact && deleteImpact.length > 0 && usedInOffers && " dan di "}
                {usedInOffers && <strong className="text-text-primary">paket atau penawaran</strong>}, jadi belum bisa dihapus.{" "}
                {deleteTarget?.isActive
                  ? usedInOffers
                    ? "Nonaktifkan saja: vendor tidak bisa dipilih lagi untuk proyek, paket, atau penawaran baru, dan data yang sudah ada tetap utuh."
                    : "Nonaktifkan saja: vendor tidak muncul lagi saat menambah vendor ke proyek, dan data di proyek tadi tetap utuh."
                  : usedInOffers
                    ? "Vendor ini sudah nonaktif, jadi tidak bisa dipilih lagi untuk proyek, paket, atau penawaran baru."
                    : "Vendor ini sudah nonaktif, jadi tidak muncul lagi saat menambah vendor ke proyek."}
              </span>
            </p>
            <div className="flex justify-end gap-2">
              <Button variant="secondary" onClick={closeDeleteConfirm} disabled={isDeleting}>
                {deleteTarget?.isActive && canManage ? "Batal" : "Tutup"}
              </Button>
              {deleteTarget?.isActive && canManage && (
                <Button onClick={() => void handleDeactivateInstead()} disabled={isDeleting}>
                  {isDeleting ? "Menyimpan..." : "Nonaktifkan vendor"}
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
