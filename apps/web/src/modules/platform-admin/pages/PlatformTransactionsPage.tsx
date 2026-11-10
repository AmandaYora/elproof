import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { Badge } from "@/shared/components/ui/Badge";
import { Input, Select } from "@/shared/components/ui/Input";
import { DataTable, type Column } from "@/shared/data-table/DataTable";
import { ListToolbar } from "@/shared/data-table/ListToolbar";
import { useListState } from "@/shared/data-table/useListState";
import { useListDensity } from "@/shared/data-table/useListDensity";
import { useSavedViews, viewQuery } from "@/shared/data-table/useSavedViews";
import { usePlatformAdminStore } from "@/modules/platform-admin/stores/usePlatformAdminStore";
import type { SubscriptionTransaction, SubscriptionTransactionStatus } from "@/modules/platform-admin/data";
import { TRANSACTION_STATUS_TONE, TRANSACTION_STATUS_LABEL, TRANSACTION_TYPE_LABEL } from "@/modules/platform-admin/lib/status";
import { formatCurrency, formatDate } from "@/shared/lib/formatters";
import { usePlatformCan } from "@/modules/platform-admin/lib/capabilities";
import { ROUTE_PATHS } from "@/app/routes/route-paths";

const STATUS_FILTERS: SubscriptionTransactionStatus[] = ["unpaid", "pending", "paid", "expired", "granted", "cancelled"];
const LIST_FILTERS = { status: "", from: "", to: "" };

export default function PlatformTransactionsPage() {
  const transactions = usePlatformAdminStore((s) => s.transactionPage);
  const meta = usePlatformAdminStore((s) => s.transactionPageMeta);
  const fetchTransactionList = usePlatformAdminStore((s) => s.fetchTransactionList);
  const canOpen = usePlatformCan("internal-read");

  // Search, filters, sort and page live in the URL (shareable, survives reload and Back).
  const list = useListState({ filters: LIST_FILTERS });
  const { density, setDensity } = useListDensity();
  const savedViews = useSavedViews("platform-transactions");
  const [loadState, setLoadState] = useState<"loading" | "error" | "ready">("loading");
  const [attempt, setAttempt] = useState(0);

  useEffect(() => {
    let cancelled = false;
    setLoadState("loading");
    fetchTransactionList(list.params)
      .then(() => !cancelled && setLoadState("ready"))
      .catch(() => !cancelled && setLoadState("error"));
    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [fetchTransactionList, list.paramsKey, attempt]);

  function tenantCell(tx: SubscriptionTransaction) {
    const name = tx.tenantName || `Tenant ${tx.tenantId}`;
    if (!canOpen) return <span className="font-semibold text-text-primary">{name}</span>;
    return (
      <Link to={ROUTE_PATHS.platformTenantDetail(tx.tenantId, "tagihan")} className="font-semibold text-text-primary hover:underline">
        {name}
      </Link>
    );
  }

  const columns: Column<SubscriptionTransaction>[] = [
    { id: "createdAt", header: "Tanggal", sortKey: "createdAt", cell: (tx) => formatDate(tx.createdAt) },
    { id: "tenant", header: "Tenant", sortKey: "tenantName", primary: true, cell: tenantCell },
    { id: "type", header: "Jenis", cell: (tx) => TRANSACTION_TYPE_LABEL[tx.type] },
    { id: "amount", header: "Nominal", sortKey: "amount", align: "right", cell: (tx) => formatCurrency(tx.amount) },
    { id: "method", header: "Metode", cell: (tx) => tx.paymentMethod },
    {
      id: "status",
      header: "Status",
      sortKey: "status",
      cell: (tx) => <Badge tone={TRANSACTION_STATUS_TONE[tx.status]}>{TRANSACTION_STATUS_LABEL[tx.status]}</Badge>,
    },
    { id: "reference", header: "Referensi", cell: (tx) => <span className="font-mono text-[12.5px]">{tx.paymentReference}</span> },
    { id: "paidAt", header: "Dibayar", cell: (tx) => (tx.paidAt ? formatDate(tx.paidAt) : <span className="text-muted-foreground">-</span>) },
  ];

  return (
    <div className="flex flex-col gap-5">
      <div>
        <h1 className="text-xl font-bold text-text-primary">Tagihan</h1>
        <p className="mt-1 text-[13px] text-text-secondary">
          Semua tagihan langganan tenant. Pembayaran diproses otomatis oleh sistem pembayaran; pilih status "Menunggu Pembayaran" untuk tagihan yang perlu ditindaklanjuti.
        </p>
      </div>

      <ListToolbar
        search={{ value: list.q, onChange: list.setQ, placeholder: "Cari nama tenant atau referensi..." }}
        density={{ value: density, onChange: setDensity }}
        views={{
          items: savedViews.views,
          current: viewQuery(list.queryString),
          onApply: list.applyQueryString,
          onSave: savedViews.save,
          onRemove: savedViews.remove,
        }}
      >
        <Select className="w-56" aria-label="Status" value={list.filters.status || "Semua"} onChange={(e) => list.setFilter("status", e.target.value === "Semua" ? "" : e.target.value)}>
          <option value="Semua">Semua status</option>
          {STATUS_FILTERS.map((status) => (
            <option key={status} value={status}>
              {TRANSACTION_STATUS_LABEL[status]}
            </option>
          ))}
        </Select>
        <Input type="date" className="w-40" aria-label="Dari tanggal" value={list.filters.from} max={list.filters.to || undefined} onChange={(e) => list.setFilter("from", e.target.value)} />
        <Input type="date" className="w-40" aria-label="Sampai tanggal" value={list.filters.to} min={list.filters.from || undefined} onChange={(e) => list.setFilter("to", e.target.value)} />
      </ListToolbar>

      <DataTable
        label="Daftar tagihan"
        columns={columns}
        rows={transactions}
        rowId={(tx) => tx.id}
        status={loadState}
        onRetry={() => setAttempt((n) => n + 1)}
        meta={meta}
        onPageChange={list.setPage}
        sort={list.sort}
        onSortChange={list.setSort}
        filtered={list.isFiltered}
        onResetFilters={list.reset}
        empty={{
          title: "Belum ada tagihan",
          description: "Tagihan muncul di sini saat tenant membayar langganan atau tim ElProof mengaktifkannya.",
        }}
        density={density}
      />
    </div>
  );
}
