import { useCallback, useEffect, useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { Plus, Pencil, CheckCircle2, Ban, Trash2, AlertTriangle, Sparkles } from "lucide-react";
import { Button } from "@/shared/components/ui/Button";
import { Badge } from "@/shared/components/ui/Badge";
import { Modal } from "@/shared/components/ui/Modal";
import { Select } from "@/shared/components/ui/Input";
import { PageHeader } from "@/shared/components/ui/PageHeader";
import { RowActions } from "@/shared/components/ui/RowActions";
import { DataTable, type Column } from "@/shared/data-table/DataTable";
import { ListToolbar } from "@/shared/data-table/ListToolbar";
import { ListFilters, activeFilterCount } from "@/shared/data-table/ListFilters";
import { useListState } from "@/shared/data-table/useListState";
import { useCan } from "@/shared/stores/usePermissionStore";
import { toast } from "@/shared/stores/useToastStore";
import { getApiErrorMessage } from "@/shared/lib/api-error";
import { formatCurrency } from "@/shared/lib/formatters";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import { usePackageStore } from "@/modules/packages/stores/usePackageStore";
import { extraGuestText } from "@/modules/packages/lib/package-display";
import type { Package } from "@/modules/packages/types";

const FILTERS = { isActive: "" };

/** Katalog › Paket (DESAIN.md §6.6): the WO's products, sorted, searched and paged on the server. */
export default function PackageListPage() {
  const navigate = useNavigate();
  // package.manage, and "no" while the workspace is read-only (subscription).
  const canManage = useCan("package.manage");
  const packages = usePackageStore((s) => s.packagePage);
  const meta = usePackageStore((s) => s.packagePageMeta);
  const fetchPackagePage = usePackageStore((s) => s.fetchPackagePage);
  const setPackageActive = usePackageStore((s) => s.setPackageActive);
  const deletePackage = usePackageStore((s) => s.deletePackage);

  const list = useListState({ filters: FILTERS });
  const [status, setStatus] = useState<"loading" | "error" | "ready">("loading");
  const [attempt, setAttempt] = useState(0);
  const [actionError, setActionError] = useState<string | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<Package | null>(null);
  const [deleteError, setDeleteError] = useState<string | null>(null);
  const [isDeleting, setIsDeleting] = useState(false);

  useEffect(() => {
    let cancelled = false;
    setStatus("loading");
    fetchPackagePage(list.params)
      .then(() => !cancelled && setStatus("ready"))
      .catch(() => !cancelled && setStatus("error"));
    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [fetchPackagePage, list.paramsKey, attempt]);

  const reload = useCallback(() => fetchPackagePage(list.params), [fetchPackagePage, list.params]);

  async function handleSetActive(pkg: Package, isActive: boolean) {
    setActionError(null);
    try {
      await setPackageActive(pkg.id, isActive);
      toast.success(isActive ? "Paket diaktifkan" : "Paket dinonaktifkan");
      await reload();
    } catch (err) {
      setActionError(getApiErrorMessage(err, "Status paket belum bisa diubah. Coba lagi."));
    }
  }

  function closeDelete() {
    setDeleteTarget(null);
    setDeleteError(null);
  }

  async function handleDelete() {
    if (!deleteTarget) return;
    setIsDeleting(true);
    setDeleteError(null);
    try {
      await deletePackage(deleteTarget.id);
      closeDelete();
      toast.success("Paket dihapus");
      await reload();
    } catch (err) {
      setDeleteError(getApiErrorMessage(err, "Paket belum bisa dihapus. Periksa koneksi Anda, lalu coba lagi."));
    } finally {
      setIsDeleting(false);
    }
  }

  const columns: Column<Package>[] = [
    {
      id: "name",
      header: "Paket",
      sortKey: "name",
      primary: true,
      truncate: true,
      className: "w-[30%] min-w-56",
      cell: (p) => (
        <>
          <Link
            to={ROUTE_PATHS.packageDetail(p.id)}
            className="block max-w-full truncate font-semibold underline-offset-2 hover:underline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-solid focus-visible:outline-ring"
          >
            {p.name}
          </Link>
          <span className="block truncate text-[12.5px] font-normal text-muted-foreground">
            {p.componentCount > 0 ? `${p.componentCount} isi paket` : "Belum ada isi paket"}
            {p.defaultVenue ? ` · ${p.defaultVenue.name}` : ""}
          </span>
        </>
      ),
    },
    {
      id: "kinds",
      header: "Acara",
      card: true,
      truncate: true,
      className: "w-[20%]",
      cell: (p) => p.eventKinds.map((k) => k.name).join(", "),
    },
    { id: "price", header: "Harga", sortKey: "price", align: "right", card: true, className: "whitespace-nowrap", cell: (p) => formatCurrency(p.price) },
    {
      id: "guests",
      header: "Tamu termasuk",
      sortKey: "includedGuests",
      align: "right",
      card: true,
      className: "whitespace-nowrap",
      cell: (p) =>
        p.includedGuests === null ? (
          <span className="text-muted-foreground">Tidak bergantung tamu</span>
        ) : (
          <span className="flex flex-col items-end leading-snug">
            <span>{p.includedGuests.toLocaleString("id-ID")}</span>
            {extraGuestText(p) && <span className="text-[12.5px] text-muted-foreground">{extraGuestText(p)}</span>}
          </span>
        ),
    },
    {
      id: "status",
      header: "Status",
      card: true,
      className: "whitespace-nowrap",
      cell: (p) => (p.isActive ? <Badge tone="success">Aktif</Badge> : <Badge tone="neutral">Nonaktif</Badge>),
    },
  ];

  return (
    <div className="flex flex-col gap-5">
      <PageHeader
        className="mb-1"
        title="Paket"
        description="Produk yang Anda tawarkan ke klien: harga, acara yang dicakup, jumlah tamu, dan isinya."
        actions={
          canManage && (
            <Button icon={<Plus className="h-4 w-4" />} onClick={() => navigate(ROUTE_PATHS.packageNew)}>
              Buat paket
            </Button>
          )
        }
      />

      {actionError && (
        <p role="alert" className="rounded-md border border-danger/30 bg-danger-soft px-3.5 py-2.5 text-[13px] font-medium text-danger-text">
          {actionError}
        </p>
      )}

      <ListToolbar search={{ value: list.q, onChange: list.setQ, placeholder: "Cari nama paket..." }}>
        <ListFilters count={activeFilterCount(list.filters)} onClear={list.clearFilters}>
          <Select className="w-36" aria-label="Status" value={list.filters.isActive || "Semua"} onChange={(e) => list.setFilter("isActive", e.target.value === "Semua" ? "" : e.target.value)}>
            <option value="Semua">Semua status</option>
            <option value="true">Aktif</option>
            <option value="false">Nonaktif</option>
          </Select>
        </ListFilters>
      </ListToolbar>

      <DataTable
        label="Daftar paket"
        columns={columns}
        rows={packages}
        rowId={(p) => p.id}
        rowLabel={(p) => p.name}
        status={status}
        onRetry={() => setAttempt((n) => n + 1)}
        meta={meta}
        onPageChange={list.setPage}
        sort={list.sort}
        onSortChange={list.setSort}
        filtered={list.isFiltered}
        onResetFilters={list.reset}
        empty={{
          title: "Belum ada paket",
          description: "Paket adalah produk yang Anda tawarkan ke klien: harga, acara yang dicakup, jumlah tamu, dan isinya.",
          action: canManage ? (
            <span className="flex flex-wrap justify-center gap-2">
              <Button icon={<Plus className="h-4 w-4" />} onClick={() => navigate(ROUTE_PATHS.packageNew)}>
                Buat paket
              </Button>
              <Button variant="secondary" icon={<Sparkles className="h-4 w-4" />} onClick={() => navigate(`${ROUTE_PATHS.packageNew}?contoh=1`)}>
                Isi dengan contoh
              </Button>
            </span>
          ) : undefined,
        }}
        actions={(pkg) => (
          <RowActions
            rowLabel={pkg.name}
            primary={{ label: "Ubah", icon: <Pencil />, onClick: () => navigate(ROUTE_PATHS.packageDetail(pkg.id)), hidden: !canManage }}
            items={[
              pkg.isActive
                ? { label: "Nonaktifkan", icon: <Ban />, onSelect: () => void handleSetActive(pkg, false), hidden: !canManage }
                : { label: "Aktifkan", icon: <CheckCircle2 />, onSelect: () => void handleSetActive(pkg, true), hidden: !canManage },
              { label: "Hapus permanen", icon: <Trash2 />, tone: "danger", onSelect: () => setDeleteTarget(pkg), hidden: !canManage },
            ]}
          />
        )}
      />

      <Modal open={deleteTarget !== null} onClose={closeDelete} size="sm" busy={isDeleting} error={deleteError} title={deleteTarget ? `Hapus paket — ${deleteTarget.name}` : "Hapus paket"}>
        <div className="flex flex-col gap-4">
          <p className="flex items-start gap-2 text-[13px] text-text-secondary">
            <AlertTriangle aria-hidden className="mt-0.5 h-4 w-4 shrink-0 text-danger-text" />
            <span>
              Paket dan isinya dihapus permanen dan tidak bisa dikembalikan. Bila paket hanya tidak dijual lagi, nonaktifkan saja: paket tetap tersimpan dan
              tidak muncul saat menyusun penawaran.
            </span>
          </p>
          <div className="flex flex-wrap justify-end gap-2">
            <Button variant="secondary" onClick={closeDelete} disabled={isDeleting}>
              Batal
            </Button>
            {deleteTarget?.isActive && (
              <Button
                variant="secondary"
                disabled={isDeleting}
                onClick={() => {
                  const target = deleteTarget;
                  closeDelete();
                  void handleSetActive(target, false);
                }}
              >
                Nonaktifkan saja
              </Button>
            )}
            <Button variant="danger" onClick={() => void handleDelete()} disabled={isDeleting}>
              {isDeleting ? "Menghapus..." : "Hapus permanen"}
            </Button>
          </div>
        </div>
      </Modal>
    </div>
  );
}
