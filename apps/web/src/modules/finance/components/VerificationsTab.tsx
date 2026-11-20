import { useNavigate } from "react-router-dom";
import { FileCheck2, FolderOpen } from "lucide-react";
import { Badge } from "@/shared/components/ui/Badge";
import { RowActions } from "@/shared/components/ui/RowActions";
import { DataTable, type Column } from "@/shared/data-table/DataTable";
import { useListState } from "@/shared/data-table/useListState";
import { formatCurrency, formatDate } from "@/shared/lib/formatters";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import { useFinanceStore } from "@/modules/finance/stores/useFinanceStore";
import { usePagedList } from "@/modules/finance/lib/usePagedList";
import { coupleOf, verificationHref, verificationRowId } from "@/modules/finance/lib/links";
import type { PendingVerification } from "@/modules/finance/types";

const FILTERS = {};

/**
 * Proofs nobody has checked yet: client payments recorded with a transfer proof, and the couple's proofs of paying a
 * vendor they pay directly ("Bukti bayar vendor", Paket WO Tahap 4). Checking happens in the project (where the proof
 * opens next to the payment), so each row leads there; a vendor proof opens its review straight away.
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
    {
      id: "type",
      header: "Jenis",
      card: true,
      cell: (v) =>
        v.kind === "supplierPayment" ? (
          <span className="flex min-w-0 flex-col items-start gap-1">
            <Badge tone="brand">Bukti bayar vendor</Badge>
            <span className="block max-w-48 truncate text-[12.5px]" title={[v.supplierName, v.billTermLabel].filter(Boolean).join(", ")}>
              {v.supplierName ?? "Vendor"}
              {v.billTermLabel ? `, ${v.billTermLabel}` : ""}
            </span>
          </span>
        ) : (
          v.type
        ),
    },
    { id: "paymentDate", header: "Tanggal bayar", sortKey: "paymentDate", card: true, className: "whitespace-nowrap", cell: (v) => formatDate(v.paymentDate) },
    { id: "amount", header: "Nominal", sortKey: "amount", align: "right", card: true, cell: (v) => <span className="font-semibold">{formatCurrency(v.amount)}</span> },
    { id: "reference", header: "Referensi", cell: (v) => (v.referenceNumber ? <span className="block max-w-36 truncate" title={v.referenceNumber}>{v.referenceNumber}</span> : <span className="text-muted-foreground">-</span>) },
    {
      id: "recordedBy",
      header: "Dicatat oleh",
      card: true,
      cell: (v) =>
        v.source === "client" ? (
          <Badge tone="info">Dikirim pasangan</Badge>
        ) : v.recordedByName ? <span className="block max-w-36 truncate" title={v.recordedByName}>{v.recordedByName}</span> : <span className="text-muted-foreground">-</span>,
    },
  ];

  return (
    <DataTable
      label="Bukti pembayaran yang menunggu diperiksa"
      columns={columns}
      rows={rows}
      rowId={verificationRowId}
      rowHref={verificationHref}
      status={status}
      onRetry={retry}
      meta={meta}
      onPageChange={list.setPage}
      sort={list.sort}
      onSortChange={list.setSort}
      filtered={false}
      onResetFilters={() => list.applyQueryString("tab=verifikasi")}
      empty={{ title: "Semua bukti sudah diperiksa", description: "Pembayaran klien dengan bukti transfer dan bukti bayar vendor dari pasangan muncul di sini sampai diperiksa." }}
      actions={(v) => (
        <RowActions
          rowLabel={`${v.kind === "supplierPayment" ? `Bukti bayar ${v.supplierName ?? "vendor"}` : v.type} ${v.projectName}`}
          primary={{ label: "Periksa bukti", icon: <FileCheck2 aria-hidden />, onClick: () => navigate(verificationHref(v)) }}
          items={[{ label: "Buka proyek", icon: <FolderOpen aria-hidden />, onSelect: () => navigate(ROUTE_PATHS.projectDetail(v.projectId)) }]}
        />
      )}
    />
  );
}
