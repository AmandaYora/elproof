import { lazy, Suspense, useCallback, useEffect, useMemo, useState } from "react";
import { Link } from "react-router-dom";
import { Plus } from "lucide-react";
import { Button } from "@/shared/components/ui/Button";
import type { Column } from "@/shared/data-table/DataTable";
import { usePagination } from "@/shared/hooks/usePagination";
import { formatCurrency } from "@/shared/lib/formatters";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import { FinanceSection, FinanceSummary, LedgerTable, type SummaryItem } from "@/modules/projects/components/detail/finance-ui";
import type { RecordPaymentValues, SupplierOption } from "@/modules/projects/components/detail/RecordPaymentDrawer";
import { EngagementStatusBadge } from "@/modules/projects/components/StatusBadges";
import { SupplierPaymentLedger, useSupplierPaymentPerms } from "@/modules/projects/components/suppliers/SupplierPaymentLedger";
import { isMissingCost, supplierKindLabel } from "@/modules/rincian";
import { sortSuppliers, SUPPLIER_OPEN_PARAM } from "@/modules/projects/components/suppliers/supplier-helpers";
import { appliedToContract } from "@/modules/projects/lib/payments";
import { useProjectStore, type SaveOptions } from "@/modules/projects/stores/useProjectStore";
import type { ProjectSupplier, SupplierPayer, SupplierPayment } from "@/modules/projects/types";

// "Catat pembayaran" opens on a click: the drawer and its form load then.
const RecordPaymentDrawer = lazy(() => import("@/modules/projects/components/detail/RecordPaymentDrawer").then((m) => ({ default: m.RecordPaymentDrawer })));

interface Row {
  supplier: ProjectSupplier;
  payments: SupplierPayment[];
  paid: number;
}

const COPY = {
  title: "Vendor & venue dibayar WO",
  ledger: "Pembayaran ke rekanan",
  emptySuppliers: {
    title: "Belum ada vendor atau venue yang dibayar WO",
    description: "Vendor dan venue dengan Dibayar oleh WO muncul di sini bersama biaya rekanannya.",
  },
  emptyPayments: { title: "Belum ada pembayaran ke rekanan", description: "Catat setiap pembayaran WO ke vendor atau venue beserta tagihan dan bukti transfernya." },
};

/**
 * The book of payments the WO makes to vendors and venues (DESAIN §5.3): what it pays against each "Biaya rekanan"
 * (needs finance.margin.view). A table per vendor or venue, then every payment, with "Catat pembayaran". What the
 * client pays vendors directly is its own book (ClientPaidBook, "Tagihan vendor").
 */
export function ProjectPaymentsSection({ projectId }: { projectId: string }) {
  const payer: SupplierPayer = "WO";
  const suppliers = useProjectStore((s) => s.suppliers);
  const supplierPayments = useProjectStore((s) => s.supplierPayments);
  const fetchSuppliers = useProjectStore((s) => s.fetchSuppliers);
  const fetchSupplierPayments = useProjectStore((s) => s.fetchSupplierPayments);
  const createSupplierPayment = useProjectStore((s) => s.createSupplierPayment);
  const perms = useSupplierPaymentPerms();
  const canRecord = perms.canRecord(payer);
  const [recordOpen, setRecordOpen] = useState(false);
  const [status, setStatus] = useState<"loading" | "error" | "ready">("loading");
  const copy = COPY;

  // A money screen never shows Rp0 for a book it could not read: the totals wait for both lists, and a failure says so.
  const load = useCallback(() => {
    setStatus("loading");
    Promise.all([fetchSuppliers(projectId), fetchSupplierPayments(projectId)]).then(
      () => setStatus("ready"),
      () => setStatus("error"),
    );
  }, [projectId, fetchSuppliers, fetchSupplierPayments]);
  useEffect(load, [load]);
  const ready = status === "ready";

  const payments = useMemo(() => supplierPayments.filter((p) => p.payer === payer), [supplierPayments, payer]);
  const rows: Row[] = useMemo(
    () =>
      sortSuppliers(suppliers.filter((s) => s.paidBy === payer)).map((supplier) => {
        const own = payments.filter((p) => p.projectSupplierId === supplier.id);
        return { supplier, payments: own, paid: appliedToContract(own) };
      }),
    [suppliers, payments, payer],
  );
  // A cancelled vendor or venue is not owed anything any more; what was paid to it stays in the ledger below.
  const active = rows.filter((r) => r.supplier.engagementStatus !== "Cancelled");
  const pagination = usePagination(rows);

  const totalCost = active.reduce((sum, r) => sum + (r.supplier.agreedCost ?? 0), 0);
  const totalPaid = active.reduce((sum, r) => sum + r.paid, 0);
  const missing = active.filter((r) => isMissingCost(r.supplier)).length;

  const summary: SummaryItem[] = [
    {
      label: "Biaya rekanan",
      value: formatCurrency(totalCost),
      note: missing > 0 ? `${missing} vendor atau venue belum diisi` : undefined,
    },
    { label: "Sudah dibayar", value: formatCurrency(totalPaid) },
    {
      label: "Sisa",
      value: formatCurrency(Math.max(totalCost - totalPaid, 0)),
      note: totalPaid > totalCost && totalCost > 0 ? `Lebih bayar ${formatCurrency(totalPaid - totalCost)}` : undefined,
      tone: totalPaid > totalCost && totalCost > 0 ? "warning" : "neutral",
    },
  ];

  const supplierOptions: SupplierOption[] = rows.map(({ supplier: s }) => ({ value: s.id, label: `${s.name} (${supplierKindLabel(s)})` }));
  const openHref = (s: ProjectSupplier) => `${ROUTE_PATHS.projectDetail(projectId, "vendor")}?${SUPPLIER_OPEN_PARAM}=${encodeURIComponent(s.id)}`;

  const columns: Column<Row>[] = [
    {
      id: "name",
      header: "Nama",
      primary: true,
      truncate: true,
      className: "w-[30%] min-w-44",
      cell: ({ supplier: s }) => (
        <span className="flex min-w-0 flex-col">
          <Link to={openHref(s)} className="truncate font-medium text-foreground hover:text-primary hover:underline" title={s.name}>
            {s.name}
          </Link>
          <span className="truncate text-[12.5px] text-muted-foreground">{supplierKindLabel(s)}</span>
        </span>
      ),
    },
    { id: "status", header: "Status", card: true, className: "whitespace-nowrap", cell: ({ supplier: s }) => <EngagementStatusBadge status={s.engagementStatus} /> },
    {
      id: "cost",
      header: "Biaya rekanan",
      align: "right",
      card: true,
      cell: ({ supplier: s }) =>
        s.agreedCost === null ? <span className="text-warning-text">Belum diisi</span> : <span className="tabular-nums">{formatCurrency(s.agreedCost)}</span>,
    },
    { id: "paid", header: "Dibayar", align: "right", card: true, cell: (r) => <span className="tabular-nums">{formatCurrency(r.paid)}</span> },
    {
      id: "left",
      header: "Sisa",
      align: "right",
      card: true,
      cell: ({ supplier: s, paid }) =>
        s.agreedCost === null ? (
          <span className="text-muted-foreground">-</span>
        ) : (
          <span className={paid > s.agreedCost ? "tabular-nums text-warning-text" : "tabular-nums"}>
            {paid > s.agreedCost ? `Lebih ${formatCurrency(paid - s.agreedCost)}` : formatCurrency(s.agreedCost - paid)}
          </span>
        ),
    },
    { id: "count", header: "Pembayaran", align: "right", className: "whitespace-nowrap", cell: (r) => <span className="tabular-nums">{r.payments.length}</span> },
  ];

  // Payment and proof are one request: if it fails nothing was saved and the drawer stays open (errors show inside it).
  async function handleRecord(values: RecordPaymentValues, options: SaveOptions) {
    await createSupplierPayment(projectId, values, options);
    setRecordOpen(false);
  }

  return (
    <div id="pembayaran" className="flex flex-col gap-8">
      <FinanceSection
        title={copy.title}
        action={
          canRecord &&
          ready &&
          supplierOptions.length > 0 && (
            <Button size="sm" icon={<Plus className="h-3.5 w-3.5" />} onClick={() => setRecordOpen(true)}>
              Catat pembayaran
            </Button>
          )
        }
      >
        {ready && <FinanceSummary items={summary} />}
        <LedgerTable
          label={copy.title}
          columns={columns}
          rows={pagination.pageItems}
          rowId={(r) => r.supplier.id}
          page={pagination.page}
          pageSize={pagination.pageSize}
          totalPages={pagination.totalPages}
          totalItems={pagination.totalItems}
          onPageChange={pagination.setPage}
          status={status}
          onRetry={load}
          empty={{
            ...copy.emptySuppliers,
            action: (
              <Link to={ROUTE_PATHS.projectDetail(projectId, "vendor")} className="text-[13px] font-semibold text-primary hover:underline">
                Buka Vendor & venue
              </Link>
            ),
          }}
        />
      </FinanceSection>

      <FinanceSection title={copy.ledger}>
        <SupplierPaymentLedger
          projectId={projectId}
          label={copy.ledger}
          payments={payments}
          suppliers={suppliers}
          showSupplier
          empty={copy.emptyPayments}
          status={status}
          onRetry={load}
        />
      </FinanceSection>

      {recordOpen && (
        <Suspense fallback={null}>
          <RecordPaymentDrawer
            open
            onClose={() => setRecordOpen(false)}
            title="Catat pembayaran ke rekanan"
            description="Pembayaran WO ke vendor atau venue, beserta tagihan dan bukti transfernya."
            supplierOptions={supplierOptions}
            slots={{ invoice: true }}
            owed={(sid) => ({
              contractValue: suppliers.find((s) => s.id === sid)?.agreedCost ?? 0,
              existing: payments.filter((p) => p.projectSupplierId === sid),
              owedLabel: "sisa biaya rekanan",
            })}
            onSubmit={handleRecord}
          />
        </Suspense>
      )}
    </div>
  );
}
