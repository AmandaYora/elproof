import { lazy, Suspense, useCallback, useEffect, useMemo, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
import { BadgeCheck, Plus } from "lucide-react";
import { Button } from "@/shared/components/ui/Button";
import type { Column } from "@/shared/data-table/DataTable";
import { usePagination } from "@/shared/hooks/usePagination";
import { formatCurrency, formatDate } from "@/shared/lib/formatters";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import { FinanceSection, FinanceSummary, LedgerTable } from "@/modules/projects/components/detail/finance-ui";
import type { RecordPaymentValues, SupplierOption } from "@/modules/projects/components/detail/RecordPaymentDrawer";
import { EngagementStatusBadge } from "@/modules/projects/components/StatusBadges";
import { SupplierPaymentLedger, useSupplierPaymentPerms } from "@/modules/projects/components/suppliers/SupplierPaymentLedger";
import { SUPPLIER_OPEN_PARAM } from "@/modules/projects/components/suppliers/supplier-helpers";
import { BillTermsTable, useBillTermEditor, useCanEditBills } from "@/modules/projects/components/supplier-bills/BillTermsTable";
import { supplierKindLabel } from "@/modules/rincian";
import { useProjectStore, type SaveOptions } from "@/modules/projects/stores/useProjectStore";
import { useSupplierBillStore } from "@/modules/projects/stores/useSupplierBillStore";
import type { SupplierBill } from "@/modules/projects/types";

// Opened by a click (or a "?periksa=" link from Beranda or the verification queue): they load then.
const RecordPaymentDrawer = lazy(() => import("@/modules/projects/components/detail/RecordPaymentDrawer").then((m) => ({ default: m.RecordPaymentDrawer })));
const SupplierProofReviewDrawer = lazy(() => import("@/modules/projects/components/supplier-bills/SupplierProofReviewDrawer"));

/** The address parameter that opens "Periksa bukti" for one payment the couple sent to a vendor. */
export const REVIEW_PROOF_PARAM = "periksa";

/**
 * Keuangan › Dibayar klien langsung (Paket WO Tahap 4): the vendors and venues the client pays itself, their bills
 * ("Tagihan vendor") as the couple sees them on the portal, and every payment to them with the proofs the couple sent
 * waiting for "Periksa bukti". None of this is WO cash: the totals here never reach the WO's own Keuangan figures.
 */
export function ClientPaidBook({ projectId }: { projectId: string }) {
  const suppliers = useProjectStore((s) => s.suppliers);
  const supplierPayments = useProjectStore((s) => s.supplierPayments);
  const fetchSuppliers = useProjectStore((s) => s.fetchSuppliers);
  const fetchSupplierPayments = useProjectStore((s) => s.fetchSupplierPayments);
  const createSupplierPayment = useProjectStore((s) => s.createSupplierPayment);
  const billsProject = useSupplierBillStore((s) => s.projectId);
  const allBills = useSupplierBillStore((s) => s.bills);
  const fetchBills = useSupplierBillStore((s) => s.fetchBills);
  const perms = useSupplierPaymentPerms();
  const canRecord = perms.canRecord("Client");
  const canEditBills = useCanEditBills();
  const [recordOpen, setRecordOpen] = useState(false);
  const [status, setStatus] = useState<"loading" | "error" | "ready">("loading");
  const [params, setParams] = useSearchParams();
  const reviewId = params.get(REVIEW_PROOF_PARAM);

  // A money screen never shows Rp0 for a book it could not read: the totals wait for every list, and a failure says so.
  const load = useCallback(() => {
    setStatus("loading");
    Promise.all([fetchSuppliers(projectId), fetchSupplierPayments(projectId), fetchBills(projectId)]).then(
      () => setStatus("ready"),
      () => setStatus("error"),
    );
  }, [projectId, fetchSuppliers, fetchSupplierPayments, fetchBills]);
  useEffect(load, [load]);
  const ready = status === "ready" && billsProject === projectId;
  const bills = useMemo(() => (billsProject === projectId ? allBills : []), [allBills, billsProject, projectId]);
  const editor = useBillTermEditor(projectId, bills);

  const payments = useMemo(() => supplierPayments.filter((p) => p.payer === "Client"), [supplierPayments]);
  const waiting = useMemo(() => payments.filter((p) => p.status === "pending"), [payments]);
  const total = bills.reduce((sum, b) => sum + b.total, 0);
  const paid = bills.reduce((sum, b) => sum + b.paid, 0);
  const remaining = bills.reduce((sum, b) => sum + b.remaining, 0);
  const pending = bills.reduce((sum, b) => sum + b.pending, 0);
  const pagination = usePagination(bills);
  const supplierOf = (id: string) => suppliers.find((s) => s.id === id);
  const nameOf = (id: string) => bills.find((b) => b.projectSupplierId === id)?.name ?? supplierOf(id)?.name ?? "Vendor atau venue terhapus";
  const termLabel = (termId: string | null) => (termId ? (bills.flatMap((b) => b.terms).find((t) => t.id === termId)?.label ?? null) : null);

  const supplierOptions: SupplierOption[] = bills.map((b) => ({ value: b.projectSupplierId, label: `${b.name} (${b.kind === "venue" ? "Venue" : "Vendor"})` }));
  const openHref = (id: string) => `${ROUTE_PATHS.projectDetail(projectId, "vendor")}?${SUPPLIER_OPEN_PARAM}=${encodeURIComponent(id)}`;

  function closeReview() {
    const next = new URLSearchParams(params);
    next.delete(REVIEW_PROOF_PARAM);
    setParams(next, { replace: true });
  }
  function openReview(paymentId: string) {
    const next = new URLSearchParams(params);
    next.set(REVIEW_PROOF_PARAM, paymentId);
    setParams(next, { replace: true });
  }
  const reviewing = reviewId ? payments.find((p) => p.id === reviewId) : undefined;
  // A link to a proof that is gone (deleted, or of another project) just lands on the book.
  useEffect(() => {
    if (ready && reviewId && !reviewing) {
      const next = new URLSearchParams(params);
      next.delete(REVIEW_PROOF_PARAM);
      setParams(next, { replace: true });
    }
  }, [ready, reviewId, reviewing, params, setParams]);

  const columns: Column<SupplierBill>[] = [
    {
      id: "name",
      header: "Nama",
      primary: true,
      truncate: true,
      className: "w-[30%] min-w-44",
      cell: (b) => {
        const s = supplierOf(b.projectSupplierId);
        return (
          <span className="flex min-w-0 flex-col">
            <Link to={openHref(b.projectSupplierId)} className="truncate font-medium text-foreground hover:text-primary hover:underline" title={b.name}>
              {b.name}
            </Link>
            <span className="truncate text-[12.5px] text-muted-foreground">{s ? supplierKindLabel(s) : b.kind === "venue" ? "Venue" : "Vendor"}</span>
          </span>
        );
      },
    },
    {
      id: "status",
      header: "Status",
      card: true,
      className: "whitespace-nowrap",
      cell: (b) => {
        const s = supplierOf(b.projectSupplierId);
        return s ? <EngagementStatusBadge status={s.engagementStatus} /> : <span className="text-muted-foreground">-</span>;
      },
    },
    { id: "total", header: "Total tagihan", align: "right", card: true, cell: (b) => <span className="tabular-nums">{formatCurrency(b.total)}</span> },
    { id: "paid", header: "Dibayar klien", align: "right", card: true, cell: (b) => <span className="tabular-nums">{formatCurrency(b.paid)}</span> },
    {
      id: "left",
      header: "Sisa",
      align: "right",
      card: true,
      cell: (b) => (
        <span className="flex flex-col items-end">
          <span className="tabular-nums">{formatCurrency(b.remaining)}</span>
          {b.pending > 0 && <span className="text-[12px] whitespace-nowrap text-text-secondary">{formatCurrency(b.pending)} menunggu diperiksa</span>}
        </span>
      ),
    },
  ];

  // Payment and proof are one request: if it fails nothing was saved and the drawer stays open (errors show inside it).
  async function handleRecord(values: RecordPaymentValues, options: SaveOptions) {
    await createSupplierPayment(projectId, values, options);
    setRecordOpen(false);
    void fetchBills(projectId).catch(() => undefined);
  }

  const noSuppliers = {
    title: "Belum ada vendor atau venue yang dibayar klien langsung",
    description: "Vendor dan venue dengan Dibayar oleh Klien langsung muncul di sini bersama tagihannya.",
    action: (
      <Link to={ROUTE_PATHS.projectDetail(projectId, "vendor")} className="text-[13px] font-semibold text-primary hover:underline">
        Buka Vendor & venue
      </Link>
    ),
  };

  // One load feeds all three sections: a failure, or a project with no client-paid vendor at all, is said once (in the
  // first section) instead of three times.
  const onlyFirst = status === "error" || (ready && bills.length === 0 && payments.length === 0);

  return (
    <div id="pembayaran" className="flex flex-col gap-8">
      <FinanceSection
        title="Dibayar klien langsung"
        notice={
          ready && waiting.length > 0 ? (
            <div role="status" className="flex flex-col gap-2 rounded-lg border border-warning/30 bg-warning-soft px-4 py-3 text-[13px] text-warning-text">
              <p className="font-semibold">
                {waiting.length} bukti bayar vendor menunggu diperiksa
              </p>
              <ul className="flex flex-col gap-1.5">
                {waiting.map((p) => (
                  <li key={p.id} className="flex flex-wrap items-center justify-between gap-2">
                    <span className="min-w-0">
                      {nameOf(p.projectSupplierId)}
                      {termLabel(p.billTermId) ? ` · ${termLabel(p.billTermId)}` : ""}: <span className="tabular-nums">{formatCurrency(p.amount)}</span>, dibayar{" "}
                      {formatDate(p.paymentDate)}
                    </span>
                    <Button size="sm" variant="secondary" icon={<BadgeCheck className="h-3.5 w-3.5" />} onClick={() => openReview(p.id)}>
                      Periksa bukti
                    </Button>
                  </li>
                ))}
              </ul>
            </div>
          ) : undefined
        }
      >
        {ready && (
          <FinanceSummary
            items={[
              { label: "Total tagihan vendor", value: formatCurrency(total) },
              {
                label: "Sudah dibayar klien",
                value: formatCurrency(paid),
                note: pending > 0 ? `${formatCurrency(pending)} menunggu diperiksa` : undefined,
                noteTone: "muted",
              },
              { label: "Sisa tagihan vendor", value: formatCurrency(remaining) },
            ]}
          />
        )}
        <LedgerTable
          label="Vendor & venue dibayar klien langsung"
          columns={columns}
          rows={pagination.pageItems}
          rowId={(b) => b.projectSupplierId}
          page={pagination.page}
          pageSize={pagination.pageSize}
          totalPages={pagination.totalPages}
          totalItems={pagination.totalItems}
          onPageChange={pagination.setPage}
          status={ready ? "ready" : status === "error" ? "error" : "loading"}
          onRetry={load}
          empty={noSuppliers}
          actions={
            canEditBills
              ? (b) => (
                  <Button size="sm" variant="ghost" icon={<Plus className="h-3.5 w-3.5" />} onClick={() => editor.openAdd(b.projectSupplierId)} aria-label={`Tambah tagihan ${b.name}`}>
                    Tambah tagihan
                  </Button>
                )
              : undefined
          }
        />
      </FinanceSection>

      {!onlyFirst && (
      <FinanceSection
        title="Tagihan vendor"
        action={
          canEditBills &&
          ready &&
          bills.length > 0 && (
            <Button size="sm" icon={<Plus className="h-3.5 w-3.5" />} onClick={() => editor.openAdd()}>
              Tambah tagihan
            </Button>
          )
        }
      >
        <p className="-mt-2 text-[13px] text-muted-foreground">
          Pasangan melihat tagihan ini di portal, membayar vendornya langsung, lalu mengirim bukti bayar untuk Anda periksa.
        </p>
        <BillTermsTable
          projectId={projectId}
          bills={bills}
          showSupplier
          onEdit={editor.openEdit}
          status={ready ? "ready" : "loading"}
          onRetry={load}
          empty={
            bills.length === 0
              ? noSuppliers
              : {
                  title: "Belum ada tagihan vendor",
                  description: "Catat tagihan tiap vendor yang dibayar klien langsung agar pasangan tahu berapa dan kapan membayarnya.",
                }
          }
        />
      </FinanceSection>
      )}

      {!onlyFirst && (
      <FinanceSection
        title="Pembayaran klien ke vendor & venue"
        action={
          canRecord &&
          ready &&
          supplierOptions.length > 0 && (
            <Button size="sm" variant="secondary" icon={<Plus className="h-3.5 w-3.5" />} onClick={() => setRecordOpen(true)}>
              Catat pembayaran
            </Button>
          )
        }
      >
        <SupplierPaymentLedger
          projectId={projectId}
          label="Pembayaran klien ke vendor & venue"
          payments={payments}
          suppliers={suppliers}
          showSupplier
          empty={{
            title: "Belum ada pembayaran klien ke vendor atau venue",
            description: "Bukti bayar yang dikirim pasangan dari portal muncul di sini untuk diperiksa. Anda juga bisa mencatatnya sendiri.",
          }}
          status={ready ? "ready" : "loading"}
          onRetry={load}
        />
      </FinanceSection>
      )}

      {editor.element}

      {recordOpen && (
        <Suspense fallback={null}>
          <RecordPaymentDrawer
            open
            onClose={() => setRecordOpen(false)}
            title="Catat pembayaran klien ke vendor atau venue"
            description="Yang dibayar klien langsung ke vendor atau venue, beserta tagihan dan bukti transfernya. Tidak masuk kas WO."
            supplierOptions={supplierOptions}
            slots={{ invoice: true }}
            owed={() => ({ contractValue: 0, existing: [], owedLabel: "" })}
            onSubmit={handleRecord}
          />
        </Suspense>
      )}

      {reviewing && (
        <Suspense fallback={null}>
          <SupplierProofReviewDrawer
            projectId={projectId}
            payment={reviewing}
            supplierName={nameOf(reviewing.projectSupplierId)}
            billLabel={termLabel(reviewing.billTermId)}
            onClose={closeReview}
          />
        </Suspense>
      )}
    </div>
  );
}
