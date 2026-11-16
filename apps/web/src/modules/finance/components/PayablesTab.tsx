import { useEffect } from "react";
import { useNavigate } from "react-router-dom";
import { FolderOpen, HandCoins } from "lucide-react";
import { Badge, type BadgeTone } from "@/shared/components/ui/Badge";
import { Select } from "@/shared/components/ui/Input";
import { RowActions } from "@/shared/components/ui/RowActions";
import { DataTable, type Column } from "@/shared/data-table/DataTable";
import { ListToolbar } from "@/shared/data-table/ListToolbar";
import { useListState } from "@/shared/data-table/useListState";
import { PAYABLE_KIND_LABELS, PAYABLE_STATUS_LABELS } from "@/shared/lib/labels";
import { formatCurrency, formatDate } from "@/shared/lib/formatters";
import { cn } from "@/shared/lib/cn";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import { useStaffStore } from "@/modules/users/stores/useStaffStore";
import { useFinanceStore } from "@/modules/finance/stores/useFinanceStore";
import { usePagedList } from "@/modules/finance/lib/usePagedList";
import { coupleOf, projectFinanceHref } from "@/modules/finance/lib/links";
import type { Payable, PayableStatus } from "@/modules/finance/types";

const FILTERS = { status: "open", kind: "", picStaffId: "" };

const STATUS_TONE: Record<PayableStatus, BadgeTone> = { paid: "success", unpaid: "neutral", overdue: "danger" };

/** Money going out: every vendor engagement with a price and every venue cost, with what is still owed. */
export function PayablesTab() {
  const navigate = useNavigate();
  const list = useListState({ filters: FILTERS });
  const fetchPayables = useFinanceStore((s) => s.fetchPayables);
  const staff = useStaffStore((s) => s.staffSummaries);
  const fetchStaff = useStaffStore((s) => s.fetchStaffSummaries);

  useEffect(() => {
    void fetchStaff().catch(() => undefined);
  }, [fetchStaff]);

  const params = { ...list.params };
  if (params.status === "all") delete params.status;
  const { rows, meta, status, retry } = usePagedList(() => fetchPayables(params), list.paramsKey);

  const columns: Column<Payable>[] = [
    {
      id: "payee",
      header: "Dibayar ke",
      primary: true,
      truncate: true,
      className: "w-[24%] min-w-48",
      cell: (p) => (
        <>
          <span className="block truncate font-semibold">{p.payeeName || PAYABLE_KIND_LABELS[p.kind]}</span>
          <span className="block truncate text-[12.5px] font-normal text-muted-foreground">
            {PAYABLE_KIND_LABELS[p.kind]}
            {p.categoryName && `, ${p.categoryName}`}
          </span>
        </>
      ),
    },
    {
      id: "project",
      header: "Proyek",
      sortKey: "projectName",
      card: true,
      truncate: true,
      className: "w-[20%]",
      cell: (p) => <span title={coupleOf(p)}>{p.projectName}</span>,
    },
    { id: "contractValue", header: "Nilai kontrak", sortKey: "contractValue", align: "right", cell: (p) => formatCurrency(p.contractValue) },
    { id: "paid", header: "Dibayar", align: "right", cell: (p) => (p.paid > 0 ? formatCurrency(p.paid) : <span className="text-muted-foreground">-</span>) },
    {
      id: "remaining",
      header: "Sisa",
      sortKey: "remaining",
      align: "right",
      card: true,
      cell: (p) => (p.remaining > 0 ? <span className="font-semibold">{formatCurrency(p.remaining)}</span> : <span className="text-muted-foreground">-</span>),
    },
    {
      id: "dueDate",
      header: "Jatuh tempo",
      sortKey: "dueDate",
      card: true,
      className: "whitespace-nowrap",
      cell: (p) =>
        p.dueDate ? (
          <span className={cn(p.daysLate > 0 && "font-medium text-danger-text")}>
            {formatDate(p.dueDate)}
            {p.daysLate > 0 && <span className="text-[12.5px]"> · terlambat {p.daysLate} hari</span>}
          </span>
        ) : (
          <span className="text-muted-foreground">Belum ditentukan</span>
        ),
    },
    { id: "status", header: "Status", card: true, className: "whitespace-nowrap", cell: (p) => <Badge tone={STATUS_TONE[p.status]}>{PAYABLE_STATUS_LABELS[p.status]}</Badge> },
  ];

  return (
    <div className="flex flex-col gap-4">
      <ListToolbar search={{ value: list.q, onChange: list.setQ, placeholder: "Cari proyek, pasangan, atau venue..." }}>
        <Select className="w-40" aria-label="Status pembayaran" value={list.filters.status} onChange={(e) => list.setFilter("status", e.target.value)}>
          <option value="open">Belum lunas</option>
          <option value="overdue">Terlambat</option>
          <option value="paid">Lunas</option>
          <option value="all">Semua status</option>
        </Select>
        <Select className="w-36" aria-label="Jenis" value={list.filters.kind || "Semua"} onChange={(e) => list.setFilter("kind", e.target.value === "Semua" ? "" : e.target.value)}>
          <option value="Semua">Vendor & venue</option>
          <option value="vendor">Vendor</option>
          <option value="venue">Venue</option>
        </Select>
        <Select className="w-44" aria-label="PIC" value={list.filters.picStaffId || "Semua"} onChange={(e) => list.setFilter("picStaffId", e.target.value === "Semua" ? "" : e.target.value)}>
          <option value="Semua">Semua PIC</option>
          {staff.map((m) => (
            <option key={m.id} value={m.id}>
              {m.name}
            </option>
          ))}
        </Select>
      </ListToolbar>

      <DataTable
        label="Daftar pembayaran ke vendor dan venue"
        columns={columns}
        rows={rows}
        rowId={(p) => `${p.kind}-${p.id}`}
        rowHref={(p) => projectFinanceHref(p.projectId, p.kind)}
        status={status}
        onRetry={retry}
        meta={meta}
        onPageChange={list.setPage}
        sort={list.sort}
        onSortChange={list.setSort}
        filtered={list.isFiltered}
        onResetFilters={() => list.applyQueryString("tab=keluar")}
        empty={{
          title: "Tidak ada yang perlu dibayar",
          description: "Vendor dengan nilai kontrak dan biaya venue dari semua proyek muncul di sini sampai lunas.",
        }}
        actions={(p) => (
          <RowActions
            rowLabel={`${p.payeeName || PAYABLE_KIND_LABELS[p.kind]} ${p.projectName}`}
            primary={{ label: "Catat pembayaran", icon: <HandCoins aria-hidden />, onClick: () => navigate(projectFinanceHref(p.projectId, p.kind)), hidden: p.remaining <= 0 }}
            items={[{ label: "Buka proyek", icon: <FolderOpen aria-hidden />, onSelect: () => navigate(ROUTE_PATHS.projectDetail(p.projectId)) }]}
          />
        )}
      />
    </div>
  );
}
