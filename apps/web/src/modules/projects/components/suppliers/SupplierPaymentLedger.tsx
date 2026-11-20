import { lazy, Suspense, useState } from "react";
import { BadgeCheck, Eye, Pencil, Trash2 } from "lucide-react";
import { Badge } from "@/shared/components/ui/Badge";
import { RowActions } from "@/shared/components/ui/RowActions";
import type { Column } from "@/shared/data-table/DataTable";
import { useCan, useWriteLocked } from "@/shared/stores/usePermissionStore";
import { usePagination } from "@/shared/hooks/usePagination";
import { deleteWithUndo } from "@/shared/lib/undo-delete";
import { formatCurrency, formatDate } from "@/shared/lib/formatters";
import { SUPPLIER_PAYMENT_STATUS_LABELS } from "@/shared/lib/labels";
import { SUPPLIER_PAYMENT_STATUS_TONE } from "@/modules/projects/lib/status";
import { useSupplierBillStore } from "@/modules/projects/stores/useSupplierBillStore";
import { EvidenceBadge, LedgerTable } from "@/modules/projects/components/detail/finance-ui";
import { canRecordFor } from "@/modules/projects/components/suppliers/supplier-helpers";
import { useProjectStore } from "@/modules/projects/stores/useProjectStore";
import type { SupplierPaymentUpdateFormValues } from "@/modules/projects/schemas/payment.schema";
import type { ProjectSupplier, SupplierPayment } from "@/modules/projects/types";

// Opened by a click only: the edit form and the proof viewer load then.
const EditSupplierPaymentModal = lazy(() =>
  import("@/modules/projects/components/suppliers/EditSupplierPaymentModal").then((m) => ({ default: m.EditSupplierPaymentModal })),
);
const SupplierProofReviewDrawer = lazy(() => import("@/modules/projects/components/supplier-bills/SupplierProofReviewDrawer"));
const EvidenceListModal = lazy(() => import("@/shared/components/ui/EvidenceListModal").then((m) => ({ default: m.EvidenceListModal })));

/** Who may change a payment to a vendor or venue, by who pays it (mirrors the API's canRecordSupplierPayment). */
export function useSupplierPaymentPerms() {
  const writeLocked = useWriteLocked();
  // finance.margin.view ends in ".view", so it survives read-only mode: recording still needs a writable workspace.
  const margin = useCan("finance.margin.view");
  const record = useCan("finance.record");
  return {
    margin,
    canRecord: (payer: ProjectSupplier["paidBy"]) => !writeLocked && canRecordFor(payer, { margin, record }),
    canDelete: useCan("finance.payment.delete"),
  };
}

interface Props {
  projectId: string;
  label: string;
  payments: SupplierPayment[];
  suppliers: ProjectSupplier[];
  /** The book of one payer lists every vendor and venue; a supplier's own drawer leaves the name column out. */
  showSupplier: boolean;
  empty: { title: string; description?: string };
  /** The Keuangan book loads the payments itself: until they arrive, a skeleton or the error with "Coba lagi". */
  status?: "loading" | "error" | "ready";
  onRetry?: () => void;
}

/**
 * The payments to vendors and venues as one ledger: "Ubah pembayaran", "Lihat bukti" and "Hapus pembayaran" on each row
 * (undo through the toast). Used by the Keuangan books and by a supplier's drawer.
 */
export function SupplierPaymentLedger({ projectId, label, payments, suppliers, showSupplier, empty, status, onRetry }: Props) {
  const allPayments = useProjectStore((s) => s.supplierPayments);
  const updateSupplierPayment = useProjectStore((s) => s.updateSupplierPayment);
  const deleteSupplierPayment = useProjectStore((s) => s.deleteSupplierPayment);
  const perms = useSupplierPaymentPerms();
  const [editing, setEditing] = useState<SupplierPayment | null>(null);
  const [viewing, setViewing] = useState<SupplierPayment | null>(null);
  const [reviewing, setReviewing] = useState<SupplierPayment | null>(null);
  const bills = useSupplierBillStore((s) => s.bills);
  // A vendor the client pays directly has proofs the couple sends and the WO checks: its rows show where each stands.
  const showStatus = payments.some((p) => p.payer === "Client");
  const { page, setPage, totalPages, totalItems, pageSize, pageItems } = usePagination(payments);

  const supplierOf = (p: SupplierPayment) => suppliers.find((s) => s.id === p.projectSupplierId);
  const nameOf = (p: SupplierPayment) => supplierOf(p)?.name || "Vendor atau venue terhapus";
  const billLabelOf = (p: SupplierPayment) =>
    p.billTermId ? (bills.flatMap((b) => b.terms).find((t) => t.id === p.billTermId)?.label ?? null) : null;

  async function handleEdit(values: SupplierPaymentUpdateFormValues) {
    if (!editing) return;
    await updateSupplierPayment(projectId, editing.id, values, editing.updatedAt);
    setEditing(null);
  }

  // No "Yakin?" step: it disappears at once and the toast offers "Urungkan" (the server keeps it 24 hours).
  function handleDelete(p: SupplierPayment) {
    void deleteWithUndo("Pembayaran", () => deleteSupplierPayment(projectId, p.id));
  }

  const columns: Column<SupplierPayment>[] = [
    ...(showSupplier
      ? [
          {
            id: "supplier",
            header: "Vendor atau venue",
            primary: true,
            truncate: true,
            className: "w-[28%] min-w-40",
            cell: (p: SupplierPayment) => <span className="font-medium" title={nameOf(p)}>{nameOf(p)}</span>,
          } satisfies Column<SupplierPayment>,
        ]
      : []),
    {
      id: "type",
      header: "Jenis",
      primary: !showSupplier,
      card: true,
      className: "whitespace-nowrap",
      cell: (p) => (
        <span className="flex flex-col items-start gap-1">
          <span>{billLabelOf(p) ?? p.type}</span>
          {/* Sent from the portal (Paket WO Tahap 4): the WO checks it before it counts as paid to the vendor. */}
          {p.source === "client" && <Badge tone="info">Dikirim pasangan</Badge>}
        </span>
      ),
    },
    { id: "date", header: "Tanggal", card: true, className: "w-32 whitespace-nowrap", cell: (p) => formatDate(p.paymentDate) },
    { id: "amount", header: "Nominal", align: "right", card: true, cell: (p) => <span className="tabular-nums">{formatCurrency(p.amount)}</span> },
    { id: "method", header: "Metode", className: "whitespace-nowrap", cell: (p) => p.method || "-" },
    {
      id: "ref",
      header: "No. referensi",
      truncate: true,
      className: "w-36 tabular-nums",
      cell: (p) => <span title={p.referenceNumber || undefined}>{p.referenceNumber || "-"}</span>,
    },
    { id: "proof", header: "Bukti", className: "whitespace-nowrap", cell: (p) => <EvidenceBadge complete={p.evidenceComplete} /> },
    ...(showStatus
      ? [
          {
            id: "status",
            header: "Status",
            card: true,
            className: "whitespace-nowrap",
            cell: (p: SupplierPayment) => (
              <span title={p.status === "rejected" && p.rejectionReason ? `Alasan: ${p.rejectionReason}` : undefined}>
                <Badge tone={SUPPLIER_PAYMENT_STATUS_TONE[p.status]}>{SUPPLIER_PAYMENT_STATUS_LABELS.staff[p.status]}</Badge>
              </span>
            ),
          } satisfies Column<SupplierPayment>,
        ]
      : []),
  ];

  const editingSupplier = editing ? supplierOf(editing) : undefined;

  return (
    <>
      <LedgerTable
        label={label}
        columns={columns}
        rows={pageItems}
        rowId={(p) => p.id}
        page={page}
        pageSize={pageSize}
        totalPages={totalPages}
        totalItems={totalItems}
        onPageChange={setPage}
        empty={empty}
        status={status}
        onRetry={onRetry}
        actions={(p) => (
          <RowActions
            rowLabel={`${nameOf(p)} ${p.type} ${formatCurrency(p.amount)}`}
            primary={
              p.source === "client"
                ? // The couple's proof is checked, not edited: "Tandai sesuai" or "Tolak dengan alasan".
                  { label: "Periksa bukti", icon: <BadgeCheck />, onClick: () => setReviewing(p) }
                : { label: "Ubah pembayaran", icon: <Pencil />, onClick: () => setEditing(p), hidden: !perms.canRecord(p.payer) }
            }
            items={[
              { label: "Lihat bukti", icon: <Eye className="h-4 w-4" />, onSelect: () => setViewing(p) },
              { label: "Hapus pembayaran", icon: <Trash2 className="h-4 w-4" />, tone: "danger", onSelect: () => handleDelete(p), hidden: !perms.canDelete },
            ]}
          />
        )}
      />

      {editing && (
        <Suspense fallback={null}>
          <EditSupplierPaymentModal
            payment={editing}
            supplierName={editingSupplier?.name || "vendor atau venue"}
            // Only the WO's payments are held to the "Biaya rekanan"; the client pays the supplier whatever it agreed.
            agreedCost={editing.payer === "WO" ? (editingSupplier?.agreedCost ?? 0) : 0}
            others={allPayments.filter((x) => x.projectSupplierId === editing.projectSupplierId && x.payer === editing.payer && x.id !== editing.id)}
            onClose={() => setEditing(null)}
            onSubmit={handleEdit}
          />
        </Suspense>
      )}

      {reviewing && (
        <Suspense fallback={null}>
          <SupplierProofReviewDrawer
            projectId={projectId}
            // The row as it is now (a check elsewhere may have changed it), else as it was opened.
            payment={allPayments.find((x) => x.id === reviewing.id) ?? reviewing}
            supplierName={nameOf(reviewing)}
            billLabel={billLabelOf(reviewing)}
            onClose={() => setReviewing(null)}
          />
        </Suspense>
      )}

      {viewing && (
        <Suspense fallback={null}>
          <EvidenceListModal
            open
            projectId={projectId}
            title={`Bukti pembayaran ${viewing.type}: ${nameOf(viewing)}`}
            items={viewing.evidence}
            onClose={() => setViewing(null)}
          />
        </Suspense>
      )}
    </>
  );
}
