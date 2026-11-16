import { useNavigate } from "react-router-dom";
import { FileCheck2, FolderOpen } from "lucide-react";
import { RowActions } from "@/shared/components/ui/RowActions";
import { DataTable, type Column } from "@/shared/data-table/DataTable";
import { useListState } from "@/shared/data-table/useListState";
import { formatCurrency, formatDate } from "@/shared/lib/formatters";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import { useFinanceStore } from "@/modules/finance/stores/useFinanceStore";
import { usePagedList } from "@/modules/finance/lib/usePagedList";
import { coupleOf, projectFinanceHref } from "@/modules/finance/lib/links";
import type { PendingVerification } from "@/modules/finance/types";

const FILTERS = {};

/**
 * Client payments recorded with a transfer proof that nobody has checked yet. Verifying happens in the project (where
 * the proof opens next to the payment), so each row leads there.
 */
export function VerificationsTab() {
  const navigate = useNavigate();
  const list = useListState({ filters: FILTERS });
  const fetchVerifications = useFinanceStore((s) => s.fetchVerifications);
  const { rows, meta, status, retry } = usePagedList(() => fetchVerifications(list.params), list.paramsKey);

  const columns: Column<PendingVerification>[] = [
    {
      id: "project",
      header: "Proyek",
      primary: true,
      truncate: true,
      className: "w-[28%] min-w-48",
      cell: (v) => (
        <>
          <span className="block truncate font-semibold">{v.projectName}</span>
          <span className="block truncate text-[12.5px] font-normal text-muted-foreground">{coupleOf(v) || "-"}</span>
        </>
      ),
    },
    { id: "type", header: "Jenis", card: true, cell: (v) => v.type },
    { id: "paymentDate", header: "Tanggal bayar", sortKey: "paymentDate", card: true, className: "whitespace-nowrap", cell: (v) => formatDate(v.paymentDate) },
    { id: "amount", header: "Nominal", sortKey: "amount", align: "right", card: true, cell: (v) => <span className="font-semibold">{formatCurrency(v.amount)}</span> },
    { id: "reference", header: "Referensi", cell: (v) => (v.referenceNumber ? <span className="block max-w-36 truncate" title={v.referenceNumber}>{v.referenceNumber}</span> : <span className="text-muted-foreground">-</span>) },
    { id: "recordedBy", header: "Dicatat oleh", cell: (v) => (v.recordedByName ? <span className="block max-w-36 truncate" title={v.recordedByName}>{v.recordedByName}</span> : <span className="text-muted-foreground">-</span>) },
  ];

  return (
    <DataTable
      label="Pembayaran klien yang menunggu verifikasi bukti"
      columns={columns}
      rows={rows}
      rowId={(v) => v.id}
      rowHref={(v) => projectFinanceHref(v.projectId, "klien")}
      status={status}
      onRetry={retry}
      meta={meta}
      onPageChange={list.setPage}
      sort={list.sort}
      onSortChange={list.setSort}
      filtered={false}
      onResetFilters={() => list.applyQueryString("tab=verifikasi")}
      empty={{ title: "Semua bukti sudah diperiksa", description: "Pembayaran klien yang dicatat dengan bukti transfer muncul di sini sampai diverifikasi." }}
      actions={(v) => (
        <RowActions
          rowLabel={`${v.type} ${v.projectName}`}
          primary={{ label: "Periksa bukti", icon: <FileCheck2 aria-hidden />, onClick: () => navigate(projectFinanceHref(v.projectId, "klien")) }}
          items={[{ label: "Buka proyek", icon: <FolderOpen aria-hidden />, onSelect: () => navigate(ROUTE_PATHS.projectDetail(v.projectId)) }]}
        />
      )}
    />
  );
}
