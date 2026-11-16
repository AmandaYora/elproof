import { useEffect } from "react";
import { useNavigate } from "react-router-dom";
import { FolderOpen, HandCoins } from "lucide-react";
import { Badge } from "@/shared/components/ui/Badge";
import { Input, Select } from "@/shared/components/ui/Input";
import { RowActions } from "@/shared/components/ui/RowActions";
import { DataTable, type Column } from "@/shared/data-table/DataTable";
import { ListToolbar } from "@/shared/data-table/ListToolbar";
import { useListState } from "@/shared/data-table/useListState";
import { TERM_STATUS_LABELS } from "@/shared/lib/labels";
import { formatCurrency, formatDate } from "@/shared/lib/formatters";
import { cn } from "@/shared/lib/cn";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import { TERM_STATUS_TONE } from "@/modules/projects/lib/status";
import { useStaffStore } from "@/modules/users/stores/useStaffStore";
import { useFinanceStore } from "@/modules/finance/stores/useFinanceStore";
import { usePagedList } from "@/modules/finance/lib/usePagedList";
import { coupleOf, projectFinanceHref } from "@/modules/finance/lib/links";
import type { Receivable, ReceivableTab } from "@/modules/finance/types";

const FILTERS = { status: "overdue", picStaffId: "", dueFrom: "", dueTo: "" };

const TABS: { value: ReceivableTab; label: string }[] = [
  { value: "overdue", label: "Terlambat" },
  { value: "due_30d", label: "Jatuh tempo 30 hari" },
  { value: "all", label: "Semua" },
  { value: "paid", label: "Lunas" },
];

const EMPTY: Record<ReceivableTab, { title: string; description: string }> = {
  overdue: { title: "Tidak ada tagihan terlambat", description: "Semua termin yang sudah jatuh tempo sudah dibayar klien." },
  due_30d: { title: "Tidak ada termin jatuh tempo 30 hari ke depan", description: "Termin berikutnya ada di tab Semua." },
  all: {
    title: "Belum ada termin pembayaran",
    description: "Termin dibuat di tab Keuangan setiap proyek: jadwal kapan klien membayar DP, termin, dan pelunasan.",
  },
  paid: { title: "Belum ada termin yang lunas", description: "Termin muncul di sini setelah pembayaran klien menutupnya." },
};

/** "Terlambat" first: the tab people open the page for (RISET WO Console §7.6). */
export function ReceivablesTab({ counts }: { counts?: Partial<Record<ReceivableTab, number>> }) {
  const navigate = useNavigate();
  const list = useListState({ filters: FILTERS });
  const fetchReceivables = useFinanceStore((s) => s.fetchReceivables);
  const staff = useStaffStore((s) => s.staffSummaries);
  const fetchStaff = useStaffStore((s) => s.fetchStaffSummaries);
  const tab = (TABS.some((t) => t.value === list.filters.status) ? list.filters.status : "overdue") as ReceivableTab;

  useEffect(() => {
    void fetchStaff().catch(() => undefined);
  }, [fetchStaff]);

  // "Semua" is the API's unfiltered list; the other tabs are its status views.
  const params = { ...list.params };
  if (params.status === "all") delete params.status;
  const { rows, meta, status, retry } = usePagedList(() => fetchReceivables(params), list.paramsKey);

  const columns: Column<Receivable>[] = [
    {
      id: "project",
      header: "Proyek",
      sortKey: "projectName",
      primary: true,
      truncate: true,
      className: "w-[22%] min-w-44",
      cell: (r) => (
        <>
          <span className="block truncate font-semibold">{r.projectName}</span>
          <span className="block truncate text-[12.5px] font-normal text-muted-foreground">{coupleOf(r) || "-"}</span>
        </>
      ),
    },
    { id: "label", header: "Termin", card: true, cell: (r) => <span className="block max-w-40 truncate" title={r.label}>{r.label}</span> },
    {
      id: "dueDate",
      header: "Jatuh tempo",
      sortKey: "dueDate",
      card: true,
      className: "whitespace-nowrap",
      cell: (r) => (
        <span className={cn(r.daysLate > 0 && "font-medium text-danger-text")}>
          {formatDate(r.dueDate)}
          {r.daysLate > 0 && <span className="text-[12.5px]"> · terlambat {r.daysLate} hari</span>}
        </span>
      ),
    },
    { id: "amount", header: "Nominal", sortKey: "amount", align: "right", cell: (r) => formatCurrency(r.amount) },
    {
      id: "remaining",
      header: "Sisa",
      sortKey: "remaining",
      align: "right",
      card: true,
      cell: (r) => (r.remaining > 0 ? <span className="font-semibold">{formatCurrency(r.remaining)}</span> : <span className="text-muted-foreground">-</span>),
    },
    { id: "status", header: "Status", card: true, className: "whitespace-nowrap", cell: (r) => <Badge tone={TERM_STATUS_TONE[r.status]}>{TERM_STATUS_LABELS.staff[r.status]}</Badge> },
    { id: "pic", header: "PIC", hideOnCard: true, cell: (r) => (r.picName ? <span className="block max-w-36 truncate" title={r.picName}>{r.picName}</span> : <span className="text-muted-foreground">-</span>) },
  ];

  return (
    <div className="flex flex-col gap-4">
      <div role="group" aria-label="Saring menurut status termin" className="flex flex-wrap gap-1.5">
        {TABS.map((t) => {
          const active = t.value === tab;
          const count = counts?.[t.value];
          return (
            <button
              key={t.value}
              type="button"
              aria-pressed={active}
              onClick={() => list.setFilter("status", t.value)}
              className={cn(
                "inline-flex h-8 items-center gap-1.5 rounded-full border px-3 text-[13px] font-medium transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-solid focus-visible:outline-ring",
                active ? "border-primary bg-primary text-primary-foreground" : "border-border bg-card text-muted-foreground hover:text-foreground",
              )}
            >
              {t.label}
              {count !== undefined && (
                <span className={cn("rounded-full px-1.5 text-[12px] tabular-nums", active ? "bg-primary-foreground/15" : "bg-muted text-foreground")}>{count}</span>
              )}
            </button>
          );
        })}
      </div>

      <ListToolbar search={{ value: list.q, onChange: list.setQ, placeholder: "Cari proyek, pasangan, atau termin..." }}>
        <Select className="w-44" aria-label="PIC" value={list.filters.picStaffId || "Semua"} onChange={(e) => list.setFilter("picStaffId", e.target.value === "Semua" ? "" : e.target.value)}>
          <option value="Semua">Semua PIC</option>
          {staff.map((m) => (
            <option key={m.id} value={m.id}>
              {m.name}
            </option>
          ))}
        </Select>
        <label className="flex items-center gap-2 text-[13px] text-muted-foreground">
          Jatuh tempo dari
          <Input type="date" className="w-40" value={list.filters.dueFrom} max={list.filters.dueTo || undefined} onChange={(e) => list.setFilter("dueFrom", e.target.value)} />
        </label>
        <label className="flex items-center gap-2 text-[13px] text-muted-foreground">
          sampai
          <Input type="date" className="w-40" value={list.filters.dueTo} min={list.filters.dueFrom || undefined} onChange={(e) => list.setFilter("dueTo", e.target.value)} />
        </label>
      </ListToolbar>

      <DataTable
        label="Daftar tagihan klien"
        columns={columns}
        rows={rows}
        rowId={(r) => r.id}
        rowHref={(r) => projectFinanceHref(r.projectId, "klien")}
        status={status}
        onRetry={retry}
        meta={meta}
        onPageChange={list.setPage}
        sort={list.sort}
        onSortChange={list.setSort}
        // The status tab is a view, not a filter to reset: only search, PIC and dates count.
        filtered={list.q !== "" || list.filters.picStaffId !== "" || list.filters.dueFrom !== "" || list.filters.dueTo !== ""}
        onResetFilters={() => list.applyQueryString(tab === "overdue" ? "" : `status=${tab}`)}
        empty={EMPTY[tab]}
        actions={(r) => (
          <RowActions
            rowLabel={`${r.label} ${r.projectName}`}
            primary={{ label: "Catat pembayaran", icon: <HandCoins aria-hidden />, onClick: () => navigate(projectFinanceHref(r.projectId, "klien")), hidden: r.remaining <= 0 }}
            items={[{ label: "Buka proyek", icon: <FolderOpen aria-hidden />, onSelect: () => navigate(ROUTE_PATHS.projectDetail(r.projectId)) }]}
          />
        )}
      />
    </div>
  );
}
