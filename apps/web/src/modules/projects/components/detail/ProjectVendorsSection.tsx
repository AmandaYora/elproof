import { lazy, Suspense, useEffect, useMemo, useState } from "react";
import { useSearchParams } from "react-router-dom";
import { Eye, Plus, RefreshCw, Trash2 } from "lucide-react";
import { SUPPLIER_PAYER_LABELS } from "@/shared/lib/labels";
import { Button } from "@/shared/components/ui/Button";
import { Select } from "@/shared/components/ui/Input";
import { RowActions } from "@/shared/components/ui/RowActions";
import { DataTable, type Column } from "@/shared/data-table/DataTable";
import { ListToolbar } from "@/shared/data-table/ListToolbar";
import { usePagination } from "@/shared/hooks/usePagination";
import { useCan } from "@/shared/stores/usePermissionStore";
import { toast } from "@/shared/stores/useToastStore";
import { deleteWithUndo } from "@/shared/lib/undo-delete";
import { getApiErrorMessage } from "@/shared/lib/api-error";
import { formatCurrency } from "@/shared/lib/formatters";
import { cn } from "@/shared/lib/cn";
import { EngagementStatusBadge } from "@/modules/projects/components/StatusBadges";
import {
  isSupplierFilter,
  matchesFilter,
  sortSuppliers,
  stepProgress,
  stepsOf,
  SUPPLIER_FILTER_LABELS,
  SUPPLIER_FILTER_PARAM,
  SUPPLIER_OPEN_PARAM,
  type SupplierFilter,
} from "@/modules/projects/components/suppliers/supplier-helpers";
import { supplierKindLabel } from "@/modules/rincian";
import { useProjectStore } from "@/modules/projects/stores/useProjectStore";
import type { ProjectSupplier } from "@/modules/projects/types";

// Opened by a click (or a link with ?buka=): each loads then.
const SupplierDetailDrawer = lazy(() =>
  import("@/modules/projects/components/suppliers/SupplierDetailDrawer").then((m) => ({ default: m.SupplierDetailDrawer })),
);
const AddVendorModal = lazy(() => import("@/modules/projects/components/suppliers/AddVendorModal").then((m) => ({ default: m.AddVendorModal })));

/**
 * Vendor & venue of a project (DESAIN §5.2): one table of every vendor and venue, filtered by `?saring=`, a row opening
 * its drawer (`?buka=<id>`). Venues come in through events; "Tambah vendor" adds one from the directory.
 */
export function ProjectVendorsSection({ projectId }: { projectId: string }) {
  const suppliers = useProjectStore((s) => s.suppliers);
  const vendorMilestones = useProjectStore((s) => s.vendorMilestones);
  const fetchSuppliers = useProjectStore((s) => s.fetchSuppliers);
  const deleteSupplier = useProjectStore((s) => s.deleteSupplier);
  const refreshSupplierCapacity = useProjectStore((s) => s.refreshSupplierCapacity);
  const canEdit = useCan("project.edit");
  // "Biaya rekanan" is the other half of the margin: the column is not there at all without the permission.
  const canMargin = useCan("finance.margin.view");
  const [params, setParams] = useSearchParams();
  const [status, setStatus] = useState<"loading" | "error" | "ready">("loading");
  const [search, setSearch] = useState("");
  const [adding, setAdding] = useState(false);

  const rawFilter = params.get(SUPPLIER_FILTER_PARAM);
  const filter: SupplierFilter = isSupplierFilter(rawFilter) ? rawFilter : "semua";
  const openId = params.get(SUPPLIER_OPEN_PARAM);

  function load() {
    setStatus("loading");
    fetchSuppliers(projectId).then(
      () => setStatus("ready"),
      () => setStatus("error"),
    );
  }
  useEffect(load, [projectId]); // eslint-disable-line react-hooks/exhaustive-deps

  function setParam(key: string, value: string | null) {
    setParams(
      (prev) => {
        const next = new URLSearchParams(prev);
        if (value === null) next.delete(key);
        else next.set(key, value);
        return next;
      },
      { replace: key === SUPPLIER_FILTER_PARAM },
    );
  }

  function hrefFor(s: ProjectSupplier): string {
    const next = new URLSearchParams(params);
    next.set(SUPPLIER_OPEN_PARAM, s.id);
    return `?${next.toString()}`;
  }

  const visible = useMemo(() => {
    const q = search.trim().toLowerCase();
    // "Biaya rekanan belum diisi" means nothing to someone who cannot see the cost.
    const effective = filter === "tanpa-biaya" && !canMargin ? "semua" : filter;
    return sortSuppliers(suppliers).filter(
      (s) => matchesFilter(s, effective) && (q === "" || s.name.toLowerCase().includes(q) || supplierKindLabel(s).toLowerCase().includes(q)),
    );
  }, [suppliers, filter, search, canMargin]);
  const { page, setPage, totalPages, totalItems, pageSize, pageItems } = usePagination(visible);

  function remove(s: ProjectSupplier) {
    void deleteWithUndo(s.kind === "venue" ? "Venue" : "Vendor", () => deleteSupplier(projectId, s.id));
  }

  async function refreshCapacity(s: ProjectSupplier) {
    try {
      await refreshSupplierCapacity(projectId, s.id);
      toast.success(`Kapasitas ${s.name} diperbarui dari direktori venue`);
    } catch (err) {
      toast.error(getApiErrorMessage(err, "Kapasitas belum bisa diperbarui"));
    }
  }

  const columns: Column<ProjectSupplier>[] = [
    {
      id: "name",
      header: "Nama",
      primary: true,
      className: "min-w-48",
      cell: (s) => (
        <span className="flex min-w-0 flex-col">
          <span className={cn("truncate font-medium text-foreground", s.engagementStatus === "Cancelled" && "line-through")}>{s.name}</span>
          <span className="truncate text-[12.5px] text-muted-foreground">{supplierKindLabel(s)}</span>
        </span>
      ),
    },
    { id: "paidBy", header: "Dibayar oleh", card: true, className: "whitespace-nowrap", cell: (s) => SUPPLIER_PAYER_LABELS[s.paidBy] },
    {
      id: "shown",
      header: "Tampil ke klien",
      className: "whitespace-nowrap",
      // A venue, and anything the client pays itself, is always named to the client (domain.ProjectSupplier.NameShown).
      cell: (s) => (s.kind === "venue" || s.paidBy === "Client" ? "Selalu" : s.showName ? "Ya" : "Tidak"),
    },
    { id: "status", header: "Status", card: true, className: "whitespace-nowrap", cell: (s) => <EngagementStatusBadge status={s.engagementStatus} /> },
    ...(canMargin
      ? [
          {
            id: "cost",
            header: "Biaya rekanan",
            align: "right",
            card: true,
            className: "whitespace-nowrap",
            cell: (s: ProjectSupplier) =>
              s.paidBy === "Client" ? (
                <span className="text-muted-foreground">—</span>
              ) : s.agreedCost === null ? (
                <span className="text-warning-text">Belum diisi</span>
              ) : (
                <span className="tabular-nums">{formatCurrency(s.agreedCost)}</span>
              ),
          } satisfies Column<ProjectSupplier>,
        ]
      : []),
    {
      id: "progress",
      header: "Progres",
      card: true,
      className: "whitespace-nowrap",
      cell: (s) => {
        const { done, total } = stepProgress(stepsOf(vendorMilestones, s.id));
        return total === 0 ? <span className="text-muted-foreground">—</span> : <span className="tabular-nums">{done}/{total} tahap</span>;
      },
    },
  ];

  const filterOptions = (Object.keys(SUPPLIER_FILTER_LABELS) as SupplierFilter[]).filter((f) => f !== "tanpa-biaya" || canMargin);

  return (
    <div id="vendor" className="flex flex-col gap-4">
      <ListToolbar
        search={{ value: search, onChange: setSearch, placeholder: "Cari vendor atau venue" }}
        actions={
          canEdit && (
            <Button size="sm" icon={<Plus className="h-3.5 w-3.5" />} onClick={() => setAdding(true)}>
              Tambah vendor
            </Button>
          )
        }
      >
        <Select
          aria-label="Saring vendor dan venue"
          className="w-56"
          value={filter}
          onChange={(e) => setParam(SUPPLIER_FILTER_PARAM, e.target.value === "semua" ? null : e.target.value)}
        >
          {filterOptions.map((f) => (
            <option key={f} value={f}>{f === "semua" ? "Saring: semua" : SUPPLIER_FILTER_LABELS[f]}</option>
          ))}
        </Select>
      </ListToolbar>

      <DataTable
        label="Vendor & venue"
        columns={columns}
        rows={pageItems}
        rowId={(s) => s.id}
        rowHref={hrefFor}
        rowLabel={(s) => s.name}
        status={status === "ready" || suppliers.length > 0 ? "ready" : status}
        onRetry={load}
        meta={{ page, limit: pageSize, total: totalItems, totalPages }}
        onPageChange={setPage}
        sort={null}
        onSortChange={() => undefined}
        filtered={filter !== "semua" || search.trim() !== ""}
        onResetFilters={() => {
          setSearch("");
          setParam(SUPPLIER_FILTER_PARAM, null);
        }}
        empty={{
          title: "Belum ada vendor atau venue",
          description: "Tambahkan vendor dari direktori. Venue masuk otomatis saat dipilih di acara.",
        }}
        actions={(s) => (
          <RowActions
            rowLabel={s.name}
            items={[
              { label: "Buka detail", icon: <Eye className="h-4 w-4" />, onSelect: () => setParam(SUPPLIER_OPEN_PARAM, s.id) },
              {
                label: "Perbarui kapasitas",
                icon: <RefreshCw className="h-4 w-4" />,
                onSelect: () => void refreshCapacity(s),
                hidden: !canEdit || s.kind !== "venue",
              },
              { label: "Hapus", icon: <Trash2 className="h-4 w-4" />, tone: "danger", onSelect: () => remove(s), hidden: !canEdit },
            ]}
          />
        )}
      />

      {openId && (
        <Suspense fallback={null}>
          <SupplierDetailDrawer projectId={projectId} supplierId={openId} onClose={() => setParam(SUPPLIER_OPEN_PARAM, null)} />
        </Suspense>
      )}
      {adding && (
        <Suspense fallback={null}>
          <AddVendorModal projectId={projectId} onClose={() => setAdding(false)} />
        </Suspense>
      )}
    </div>
  );
}
