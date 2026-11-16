import { useEffect, useState } from "react";
import { useCan, useWriteLocked } from "@/shared/stores/usePermissionStore";
import { Plus, Pencil, CheckCircle2, Ban, Trash2, AlertTriangle } from "lucide-react";
import { Button } from "@/shared/components/ui/Button";
import { Badge } from "@/shared/components/ui/Badge";
import { Modal } from "@/shared/components/ui/Modal";
import { PageHeader } from "@/shared/components/ui/PageHeader";
import { RowActions } from "@/shared/components/ui/RowActions";
import { DataTable, type Column } from "@/shared/data-table/DataTable";
import { ListToolbar } from "@/shared/data-table/ListToolbar";
import { VendorCategoryFormModal } from "@/modules/vendor-categories/components/VendorCategoryFormModal";
import type { VendorCategoryFormValues } from "@/modules/vendor-categories/schemas/vendor-category.schema";
import { useVendorCategoryStore } from "@/modules/vendor-categories/stores/useVendorCategoryStore";
import { useVendorStore } from "@/modules/vendors/stores/useVendorStore";
import type { VendorCategory } from "@/modules/vendor-categories/types";
import { getApiErrorMessage } from "@/shared/lib/api-error";
import { useDebouncedValue } from "@/shared/hooks/useDebouncedValue";

export default function VendorCategoryListPage() {
  // Read-only workspace (subscription): no new records until it is renewed.
  const writeLocked = useWriteLocked();
  const categories = useVendorCategoryStore((s) => s.categoryPage);
  const meta = useVendorCategoryStore((s) => s.categoryPageMeta);
  const fetchCategoryPage = useVendorCategoryStore((s) => s.fetchCategoryPage);
  const createCategory = useVendorCategoryStore((s) => s.createCategory);
  const updateCategory = useVendorCategoryStore((s) => s.updateCategory);
  const setCategoryActive = useVendorCategoryStore((s) => s.setCategoryActive);
  const deleteCategory = useVendorCategoryStore((s) => s.deleteCategory);
  const vendors = useVendorStore((s) => s.vendors);
  const fetchVendors = useVendorStore((s) => s.fetchVendors);
  // Destructive/management actions follow the role's permissions (ADR-0026).
  const isOwner = useCan("vendor.category.manage");

  const [query, setQuery] = useState("");
  const debouncedQuery = useDebouncedValue(query);
  const [page, setPage] = useState(1);
  const [modalOpen, setModalOpen] = useState(false);
  const [editingCategory, setEditingCategory] = useState<VendorCategory | undefined>(undefined);
  const [actionError, setActionError] = useState<string | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<VendorCategory | null>(null);
  const [isDeleting, setIsDeleting] = useState(false);
  const [status, setStatus] = useState<"loading" | "error" | "ready">("loading");
  const [attempt, setAttempt] = useState(0);

  useEffect(() => {
    void fetchVendors();
  }, [fetchVendors]);

  useEffect(() => {
    setPage(1);
  }, [debouncedQuery]);

  useEffect(() => {
    let cancelled = false;
    setStatus("loading");
    fetchCategoryPage(page, debouncedQuery)
      .then(() => !cancelled && setStatus("ready"))
      .catch(() => !cancelled && setStatus("error"));
    return () => {
      cancelled = true;
    };
  }, [fetchCategoryPage, page, debouncedQuery, attempt]);

  function openCreateModal() {
    setEditingCategory(undefined);
    setModalOpen(true);
  }

  function openEditModal(category: VendorCategory) {
    setEditingCategory(category);
    setModalOpen(true);
  }

  function closeModal() {
    setModalOpen(false);
    setEditingCategory(undefined);
  }

  // Errors are not caught here: they propagate to the form, which shows a 422 on its fields.
  async function handleSubmit(values: VendorCategoryFormValues) {
    setActionError(null);
    if (editingCategory) {
      await updateCategory(editingCategory.id, values);
    } else {
      await createCategory(values);
    }
    closeModal();
    await fetchCategoryPage(page, query);
  }

  async function handleToggleActive(category: VendorCategory) {
    setActionError(null);
    try {
      await setCategoryActive(category.id, !category.isActive);
      await fetchCategoryPage(page, query);
    } catch (err) {
      setActionError(getApiErrorMessage(err, "Gagal mengubah status kategori"));
    }
  }

  // Hard delete (PLAN.md) — deliberate carve-out, no confirm-and-proceed
  // dialog: a category still assigned to any vendor is a hard block (real
  // same-module SQL FK), so the vendor count already loaded for this page's
  // own "Jumlah Vendor" column doubles as the pre-check here too.
  async function handleConfirmDelete() {
    if (!deleteTarget) return;
    setIsDeleting(true);
    try {
      await deleteCategory(deleteTarget.id);
      setDeleteTarget(null);
      await fetchCategoryPage(page, query);
    } catch (err) {
      setActionError(getApiErrorMessage(err, "Gagal menghapus kategori vendor"));
      setDeleteTarget(null);
    } finally {
      setIsDeleting(false);
    }
  }

  const vendorCount = (category: VendorCategory) => {
    const inCategory = vendors.filter((v) => v.categoryId === category.id);
    return `${inCategory.length} vendor (${inCategory.filter((v) => v.isActive).length} aktif)`;
  };

  const columns: Column<VendorCategory>[] = [
    {
      id: "name",
      header: "Nama kategori",
      primary: true,
      truncate: true,
      className: "w-[26%] min-w-48",
      cell: (c) =>
        writeLocked ? (
          <span className="font-semibold">{c.name}</span>
        ) : (
          <button
            type="button"
            onClick={() => openEditModal(c)}
            className="max-w-full truncate text-left font-semibold underline-offset-2 hover:underline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-solid focus-visible:outline-ring"
          >
            {c.name}
          </button>
        ),
    },
    {
      id: "description",
      header: "Deskripsi",
      truncate: true,
      card: true,
      className: "w-[40%] text-muted-foreground",
      cell: (c) => c.description || "-",
    },
    { id: "vendors", header: "Jumlah vendor", card: true, className: "whitespace-nowrap", cell: vendorCount },
    {
      id: "status",
      header: "Status",
      card: true,
      className: "whitespace-nowrap",
      cell: (c) => (c.isActive ? <Badge tone="success">Aktif</Badge> : <Badge tone="neutral">Nonaktif</Badge>),
    },
  ];

  return (
    <div className="flex flex-col gap-5">
      <PageHeader
        className="mb-1"
        title="Kategori vendor"
        description="Jenis layanan untuk mengelompokkan vendor, misalnya katering, dekorasi, atau foto."
        actions={
          !writeLocked && (
            <Button icon={<Plus className="h-4 w-4" />} onClick={openCreateModal}>
              Tambah kategori
            </Button>
          )
        }
      />

      {actionError && (
        <p className="rounded-md border border-danger/30 bg-danger-soft px-3.5 py-2.5 text-[13px] font-medium text-danger-text">{actionError}</p>
      )}

      <ListToolbar search={{ value: query, onChange: setQuery, placeholder: "Cari nama kategori..." }} />

      <DataTable
        label="Daftar kategori vendor"
        columns={columns}
        rows={categories}
        rowId={(c) => c.id}
        rowLabel={(c) => c.name}
        status={status}
        onRetry={() => setAttempt((n) => n + 1)}
        meta={meta}
        onPageChange={setPage}
        sort={null}
        onSortChange={() => undefined}
        filtered={query !== ""}
        onResetFilters={() => setQuery("")}
        empty={{
          title: "Belum ada kategori vendor",
          description: "Buat kategori seperti Katering atau Dekorasi agar vendor mudah dikelompokkan dan dicari.",
          action: !writeLocked ? <Button icon={<Plus className="h-4 w-4" />} onClick={openCreateModal}>Tambah kategori</Button> : undefined,
        }}
        actions={(category) => (
          <RowActions
            rowLabel={category.name}
            primary={{ label: "Ubah", icon: <Pencil />, onClick: () => openEditModal(category), hidden: writeLocked }}
            items={[
              category.isActive
                ? { label: "Nonaktifkan", icon: <Ban />, onSelect: () => void handleToggleActive(category), hidden: writeLocked }
                : { label: "Aktifkan", icon: <CheckCircle2 />, onSelect: () => void handleToggleActive(category), hidden: writeLocked },
              { label: "Hapus permanen", icon: <Trash2 />, tone: "danger", onSelect: () => setDeleteTarget(category), hidden: !isOwner },
            ]}
          />
        )}
      />

      <VendorCategoryFormModal
        key={editingCategory?.id ?? "new"}
        open={modalOpen}
        onClose={closeModal}
        onSubmit={handleSubmit}
        initialCategory={editingCategory}
      />

      <Modal
        open={deleteTarget !== null}
        onClose={() => setDeleteTarget(null)}
        title={deleteTarget ? `Hapus kategori — ${deleteTarget.name}` : "Hapus kategori"}
        size="sm"
      >
        {deleteTarget &&
          (() => {
            const inUseCount = vendors.filter((v) => v.categoryId === deleteTarget.id).length;
            if (inUseCount > 0) {
              return (
                <div className="flex flex-col gap-4">
                  <p className="flex items-start gap-2 text-[13px] text-text-secondary">
                    <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0 text-danger-text" />
                    <span>
                      Kategori ini masih digunakan oleh <strong className="text-text-primary">{inUseCount} vendor</strong> dan tidak
                      dapat dihapus. Ubah kategori pada vendor tersebut terlebih dahulu.
                    </span>
                  </p>
                  <div className="flex justify-end">
                    <Button variant="secondary" onClick={() => setDeleteTarget(null)}>
                      Tutup
                    </Button>
                  </div>
                </div>
              );
            }
            return (
              <div className="flex flex-col gap-4">
                <p className="flex items-start gap-2 text-[13px] text-text-secondary">
                  <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0 text-danger-text" />
                  <span>Data ini akan dihapus secara permanen dan tidak dapat dikembalikan. Apakah Anda yakin?</span>
                </p>
                <div className="flex justify-end gap-2">
                  <Button variant="secondary" onClick={() => setDeleteTarget(null)} disabled={isDeleting}>
                    Batal
                  </Button>
                  <Button variant="danger" onClick={() => void handleConfirmDelete()} disabled={isDeleting}>
                    {isDeleting ? "Menghapus..." : "Hapus kategori permanen"}
                  </Button>
                </div>
              </div>
            );
          })()}
      </Modal>
    </div>
  );
}
