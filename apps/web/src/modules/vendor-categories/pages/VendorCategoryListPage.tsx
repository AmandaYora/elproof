import { useEffect, useState } from "react";
import { Plus, Pencil, CheckCircle2, Ban } from "lucide-react";
import { Card } from "@/shared/components/ui/Card";
import { Button } from "@/shared/components/ui/Button";
import { SearchInput } from "@/shared/components/ui/SearchInput";
import { Badge } from "@/shared/components/ui/Badge";
import { Table, THead, TBody, TR, TH, TD } from "@/shared/components/ui/Table";
import { Pagination } from "@/shared/components/ui/Pagination";
import { EmptyState } from "@/shared/components/feedback/EmptyState";
import { IconActionButton } from "@/shared/components/ui/IconActionButton";
import { VendorCategoryFormModal } from "@/modules/vendor-categories/components/VendorCategoryFormModal";
import type { VendorCategoryFormValues } from "@/modules/vendor-categories/schemas/vendor-category.schema";
import { useVendorCategoryStore } from "@/modules/vendor-categories/stores/useVendorCategoryStore";
import { useVendorStore } from "@/modules/vendors/stores/useVendorStore";
import type { VendorCategory } from "@/modules/vendor-categories/types";
import { getApiErrorMessage } from "@/shared/lib/api-error";

export default function VendorCategoryListPage() {
  const categories = useVendorCategoryStore((s) => s.categoryPage);
  const meta = useVendorCategoryStore((s) => s.categoryPageMeta);
  const fetchCategoryPage = useVendorCategoryStore((s) => s.fetchCategoryPage);
  const createCategory = useVendorCategoryStore((s) => s.createCategory);
  const updateCategory = useVendorCategoryStore((s) => s.updateCategory);
  const toggleCategoryActive = useVendorCategoryStore((s) => s.toggleCategoryActive);
  const vendors = useVendorStore((s) => s.vendors);
  const fetchVendors = useVendorStore((s) => s.fetchVendors);

  const [query, setQuery] = useState("");
  const [page, setPage] = useState(1);
  const [modalOpen, setModalOpen] = useState(false);
  const [editingCategory, setEditingCategory] = useState<VendorCategory | undefined>(undefined);
  const [actionError, setActionError] = useState<string | null>(null);

  useEffect(() => {
    void fetchVendors();
  }, [fetchVendors]);

  useEffect(() => {
    setPage(1);
  }, [query]);

  useEffect(() => {
    void fetchCategoryPage(page, query);
  }, [fetchCategoryPage, page, query]);

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
                    </div>
                  </TD>
                </TR>
                );
              })}
            </TBody>
          </Table>
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
    </div>
  );
}
