import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { Plus, Pencil, CheckCircle2, Ban, Eye } from "lucide-react";
import { Card } from "@/shared/components/ui/Card";
import { Button } from "@/shared/components/ui/Button";
import { SearchInput } from "@/shared/components/ui/SearchInput";
import { Select } from "@/shared/components/ui/Input";
import { Badge } from "@/shared/components/ui/Badge";
import { Modal } from "@/shared/components/ui/Modal";
import { Table, THead, TBody, TR, TH, TD } from "@/shared/components/ui/Table";
import { CardList, CardListField } from "@/shared/components/ui/CardList";
import { Pagination } from "@/shared/components/ui/Pagination";
import { EmptyState } from "@/shared/components/feedback/EmptyState";
import { IconActionButton } from "@/shared/components/ui/IconActionButton";
import { VendorFormModal } from "@/modules/vendors/components/VendorFormModal";
import { VendorStatusBadge } from "@/modules/vendors/components/VendorStatusBadge";
import type { VendorFormValues } from "@/modules/vendors/schemas/vendor.schema";
import { useVendorStore } from "@/modules/vendors/stores/useVendorStore";
import type { Vendor, VendorProjectHistoryItem } from "@/modules/vendors/types";
import { useVendorCategoryStore } from "@/modules/vendor-categories/stores/useVendorCategoryStore";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import { getApiErrorMessage } from "@/shared/lib/api-error";
import { formatDate } from "@/shared/lib/formatters";

export default function VendorListPage() {
  const vendors = useVendorStore((s) => s.vendorPage);
  const meta = useVendorStore((s) => s.vendorPageMeta);
  const fetchVendorPage = useVendorStore((s) => s.fetchVendorPage);
  const createVendor = useVendorStore((s) => s.createVendor);
  const updateVendor = useVendorStore((s) => s.updateVendor);
  const toggleVendorActive = useVendorStore((s) => s.toggleVendorActive);
  const fetchVendorProjectHistory = useVendorStore((s) => s.fetchVendorProjectHistory);
  const categories = useVendorCategoryStore((s) => s.categories);
  const fetchCategories = useVendorCategoryStore((s) => s.fetchCategories);

  const [query, setQuery] = useState("");
  const [categoryFilter, setCategoryFilter] = useState<string>("Semua");
  const [page, setPage] = useState(1);
  const [modalOpen, setModalOpen] = useState(false);
  const [editingVendor, setEditingVendor] = useState<Vendor | undefined>(undefined);
  const [historyVendor, setHistoryVendor] = useState<Vendor | null>(null);
  const [history, setHistory] = useState<VendorProjectHistoryItem[]>([]);
  const [historyLoading, setHistoryLoading] = useState(false);
  const [actionError, setActionError] = useState<string | null>(null);

  useEffect(() => {
    void fetchCategories();
  }, [fetchCategories]);

  useEffect(() => {
    setPage(1);
  }, [query, categoryFilter]);

  useEffect(() => {
    void fetchVendorPage(page, query, categoryFilter === "Semua" ? "" : categoryFilter);
  }, [fetchVendorPage, page, query, categoryFilter]);

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

  async function handleSubmit(values: VendorFormValues) {
    setActionError(null);
    try {
      if (editingVendor) {
        await updateVendor(editingVendor.id, values);
      } else {
        await createVendor(values);
      }
      closeModal();
      await fetchVendorPage(page, query, categoryFilter === "Semua" ? "" : categoryFilter);
    } catch (err) {
      setActionError(getApiErrorMessage(err, "Gagal menyimpan vendor"));
    }
  }

  async function handleToggleActive(vendor: Vendor) {
    setActionError(null);
    try {
      await toggleVendorActive(vendor.id);
      await fetchVendorPage(page, query, categoryFilter === "Semua" ? "" : categoryFilter);
    } catch (err) {
      setActionError(getApiErrorMessage(err, "Gagal mengubah status vendor"));
    }
  }

  return (
    <div className="flex flex-col gap-5">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-xl font-bold text-text-primary">Vendor</h1>
          <p className="mt-1 text-[13px] text-text-secondary">Kelola seluruh vendor yang bekerja sama dengan WO.</p>
        </div>
        <Button icon={<Plus className="h-4 w-4" />} onClick={openCreateModal}>
          Tambah Vendor
        </Button>
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
                    <CardListField label="Telepon" value={vendor.phone} />
                    <CardListField label="Email" value={vendor.email} />
                  </div>
                  <div className="flex items-center gap-1.5 pt-1">
                    <IconActionButton icon={Pencil} label="Ubah Vendor" tone="neutral" onClick={() => openEditModal(vendor)} />
                    {vendor.isActive ? (
                      <IconActionButton icon={Ban} label="Nonaktifkan" tone="danger" onClick={() => void handleToggleActive(vendor)} />
                    ) : (
                      <IconActionButton icon={CheckCircle2} label="Aktifkan" tone="success" onClick={() => void handleToggleActive(vendor)} />
                    )}
                    <IconActionButton icon={Eye} label="Lihat Project" tone="info" onClick={() => setHistoryVendor(vendor)} />
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
                      <span className="block text-[12.5px] text-text-secondary">{vendor.email}</span>
                    </TD>
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
        onSubmit={(values) => void handleSubmit(values)}
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
    </div>
  );
}
