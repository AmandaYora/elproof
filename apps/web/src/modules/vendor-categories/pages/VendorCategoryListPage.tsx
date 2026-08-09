import { useEffect, useState } from "react";
import { Plus, Pencil, CheckCircle2, Ban, Trash2, AlertTriangle } from "lucide-react";
import { Card } from "@/shared/components/ui/Card";
import { Button } from "@/shared/components/ui/Button";
import { SearchInput } from "@/shared/components/ui/SearchInput";
import { Badge } from "@/shared/components/ui/Badge";
import { Modal } from "@/shared/components/ui/Modal";
import { Table, THead, TBody, TR, TH, TD } from "@/shared/components/ui/Table";
import { CardList, CardListField } from "@/shared/components/ui/CardList";
import { Pagination } from "@/shared/components/ui/Pagination";
import { EmptyState } from "@/shared/components/feedback/EmptyState";
import { IconActionButton } from "@/shared/components/ui/IconActionButton";
import { VendorCategoryFormModal } from "@/modules/vendor-categories/components/VendorCategoryFormModal";
import type { VendorCategoryFormValues } from "@/modules/vendor-categories/schemas/vendor-category.schema";
import { useVendorCategoryStore } from "@/modules/vendor-categories/stores/useVendorCategoryStore";
import { useVendorStore } from "@/modules/vendors/stores/useVendorStore";
import type { VendorCategory } from "@/modules/vendor-categories/types";
import { getApiErrorMessage } from "@/shared/lib/api-error";
import { useDebouncedValue } from "@/shared/hooks/useDebouncedValue";
import { useAuthStore } from "@/shared/stores/useAuthStore";

export default function VendorCategoryListPage() {
  const categories = useVendorCategoryStore((s) => s.categoryPage);
  const meta = useVendorCategoryStore((s) => s.categoryPageMeta);
  const fetchCategoryPage = useVendorCategoryStore((s) => s.fetchCategoryPage);
  const createCategory = useVendorCategoryStore((s) => s.createCategory);
  const updateCategory = useVendorCategoryStore((s) => s.updateCategory);
  const toggleCategoryActive = useVendorCategoryStore((s) => s.toggleCategoryActive);
  const deleteCategory = useVendorCategoryStore((s) => s.deleteCategory);
  const vendors = useVendorStore((s) => s.vendors);
  const fetchVendors = useVendorStore((s) => s.fetchVendors);
  const isOwner = useAuthStore((s) => s.session?.role === "Owner");

  const [query, setQuery] = useState("");
  const debouncedQuery = useDebouncedValue(query);
  const [page, setPage] = useState(1);
  const [modalOpen, setModalOpen] = useState(false);
  const [editingCategory, setEditingCategory] = useState<VendorCategory | undefined>(undefined);
  const [actionError, setActionError] = useState<string | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<VendorCategory | null>(null);
  const [isDeleting, setIsDeleting] = useState(false);

  useEffect(() => {
    void fetchVendors();
  }, [fetchVendors]);

  useEffect(() => {
    setPage(1);
  }, [debouncedQuery]);

  useEffect(() => {
    void fetchCategoryPage(page, debouncedQuery);
  }, [fetchCategoryPage, page, debouncedQuery]);

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

  async function handleSubmit(values: VendorCategoryFormValues) {
    setActionError(null);
    try {
      if (editingCategory) {
        await updateCategory(editingCategory.id, values);
      } else {
        await createCategory(values);
      }
      closeModal();
      await fetchCategoryPage(page, query);
    } catch (err) {
      setActionError(getApiErrorMessage(err, "Gagal menyimpan kategori vendor"));
    }
  }

  async function handleToggleActive(category: VendorCategory) {
    setActionError(null);
    try {
      await toggleCategoryActive(category.id);
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

  return (
    <div className="flex flex-col gap-5">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-xl font-bold text-text-primary">Kategori Vendor</h1>
          <p className="mt-1 text-[13px] text-text-secondary">
            Kelola kategori untuk mengelompokkan vendor berdasarkan jenis layanan.
          </p>
        </div>
        <Button icon={<Plus className="h-4 w-4" />} onClick={openCreateModal}>
          Tambah Kategori
        </Button>
      </div>

      {actionError && (
        <p className="rounded-md border border-danger/30 bg-danger-soft px-3.5 py-2.5 text-[13px] font-medium text-danger">{actionError}</p>
      )}

      <div className="flex flex-wrap gap-3">
        <SearchInput
          className="max-w-xs"
          placeholder="Cari nama kategori..."
          value={query}
          onChange={(e) => setQuery(e.target.value)}
        />
      </div>

      <Card>
        {categories.length === 0 ? (
          <EmptyState
            title="Tidak ada kategori ditemukan"
            description="Ubah kata kunci pencarian atau tambahkan kategori baru."
          />
        ) : (
          <>
          <CardList
            className="sm:hidden"
            items={categories}
            keyFor={(category) => category.id}
            renderItem={(category) => {
              const categoryVendors = vendors.filter((v) => v.categoryId === category.id);
              const activeCount = categoryVendors.filter((v) => v.isActive).length;
              return (
                <>
                  <div className="flex items-start justify-between gap-3">
                    <span className="font-semibold text-text-primary">{category.name}</span>
                    {category.isActive ? <Badge tone="success">Aktif</Badge> : <Badge tone="neutral">Nonaktif</Badge>}
                  </div>
                  {category.description && <p className="text-[12.5px] text-text-secondary">{category.description}</p>}
                  <CardListField label="Jumlah Vendor" value={`${categoryVendors.length} vendor (${activeCount} aktif)`} />
                  <div className="flex items-center gap-1.5 pt-1">
                    <IconActionButton icon={Pencil} label="Ubah Kategori" tone="neutral" onClick={() => openEditModal(category)} />
                    {category.isActive ? (
                      <IconActionButton icon={Ban} label="Nonaktifkan" tone="danger" onClick={() => void handleToggleActive(category)} />
                    ) : (
                      <IconActionButton icon={CheckCircle2} label="Aktifkan" tone="success" onClick={() => void handleToggleActive(category)} />
                    )}
                    {isOwner && (
                      <IconActionButton icon={Trash2} label="Hapus Permanen" tone="danger" onClick={() => setDeleteTarget(category)} />
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
                <TH>Nama Kategori</TH>
                <TH>Deskripsi</TH>
                <TH>Jumlah Vendor</TH>
                <TH>Status</TH>
                <TH>Aksi</TH>
              </TR>
            </THead>
            <TBody>
              {categories.map((category) => {
                const categoryVendors = vendors.filter((v) => v.categoryId === category.id);
                const activeCount = categoryVendors.filter((v) => v.isActive).length;
                return (
                <TR key={category.id}>
                  <TD className="font-semibold text-text-primary">{category.name}</TD>
                  <TD className="max-w-sm text-text-secondary">{category.description}</TD>
                  <TD>
                    {categoryVendors.length} vendor ({activeCount} aktif)
                  </TD>
                  <TD>
                    {category.isActive ? <Badge tone="success">Aktif</Badge> : <Badge tone="neutral">Nonaktif</Badge>}
                  </TD>
                  <TD>
                    <div className="flex items-center gap-1.5">
                      <IconActionButton icon={Pencil} label="Ubah Kategori" tone="neutral" onClick={() => openEditModal(category)} />
                      {category.isActive ? (
                        <IconActionButton icon={Ban} label="Nonaktifkan" tone="danger" onClick={() => void handleToggleActive(category)} />
                      ) : (
                        <IconActionButton icon={CheckCircle2} label="Aktifkan" tone="success" onClick={() => void handleToggleActive(category)} />
                      )}
                      {isOwner && (
                        <IconActionButton icon={Trash2} label="Hapus Permanen" tone="danger" onClick={() => setDeleteTarget(category)} />
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

      <VendorCategoryFormModal
        key={editingCategory?.id ?? "new"}
        open={modalOpen}
        onClose={closeModal}
        onSubmit={(values) => void handleSubmit(values)}
        initialCategory={editingCategory}
      />

      <Modal
        open={deleteTarget !== null}
        onClose={() => setDeleteTarget(null)}
        title={deleteTarget ? `Hapus Kategori — ${deleteTarget.name}` : "Hapus Kategori"}
        size="sm"
      >
        {deleteTarget &&
          (() => {
            const inUseCount = vendors.filter((v) => v.categoryId === deleteTarget.id).length;
            if (inUseCount > 0) {
              return (
                <div className="flex flex-col gap-4">
                  <p className="flex items-start gap-2 text-[13px] text-text-secondary">
                    <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0 text-danger" />
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
                  <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0 text-danger" />
                  <span>Data ini akan dihapus secara permanen dan tidak dapat dikembalikan. Apakah Anda yakin?</span>
                </p>
                <div className="flex justify-end gap-2">
                  <Button variant="secondary" onClick={() => setDeleteTarget(null)} disabled={isDeleting}>
                    Batal
                  </Button>
                  <Button variant="danger" onClick={() => void handleConfirmDelete()} disabled={isDeleting}>
                    {isDeleting ? "Menghapus..." : "Ya, Hapus Permanen"}
                  </Button>
                </div>
              </div>
            );
          })()}
      </Modal>
    </div>
  );
}
