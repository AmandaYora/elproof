import { lazy, Suspense, useEffect, useState } from "react";
import { useCan } from "@/shared/stores/usePermissionStore";
import { Plus, Pencil, Trash2, Eye } from "lucide-react";
import { Button } from "@/shared/components/ui/Button";
import { RecordPaymentDrawer } from "@/modules/projects/components/detail/RecordPaymentDrawer";
import { deleteWithUndo } from "@/shared/lib/undo-delete";
import { Modal } from "@/shared/components/ui/Modal";
import { Input, Textarea, Select, Field } from "@/shared/components/ui/Input";
import { CurrencyInput } from "@/shared/components/ui/CurrencyInput";
import { RowActions } from "@/shared/components/ui/RowActions";
import type { Column } from "@/shared/data-table/DataTable";
import { EvidenceBadge, FinanceSection, FinanceSummary, LedgerTable } from "@/modules/projects/components/detail/finance-ui";
import { usePagination } from "@/shared/hooks/usePagination";
import { useProjectStore, type SaveOptions } from "@/modules/projects/stores/useProjectStore";
import { useVendorStore } from "@/modules/vendors/stores/useVendorStore";
import {
  paymentUpdateSchema,
  PAYMENT_TYPE_OPTIONS,
  PAYMENT_METHOD_OPTIONS,
  type PaymentFormValues,
  type PaymentUpdateFormValues,
} from "@/modules/projects/schemas/payment.schema";
import type { VendorPayment } from "@/modules/projects/types";
import { getApiErrorMessage } from "@/shared/lib/api-error";
import { overpaymentMessage } from "@/modules/projects/lib/payments";
import { formatCurrency, formatDate } from "@/shared/lib/formatters";

// The proof viewer is only needed once someone opens a payment's proof.
const EvidenceListModal = lazy(() => import("@/shared/components/ui/EvidenceListModal").then((m) => ({ default: m.EvidenceListModal })));

export function ProjectPaymentsSection({ projectId }: { projectId: string }) {
  const payments = useProjectStore((s) => s.payments);
  const vendorEngagements = useProjectStore((s) => s.vendorEngagements);
  const fetchPayments = useProjectStore((s) => s.fetchPayments);
  const fetchVendorSection = useProjectStore((s) => s.fetchVendorSection);
  const createPayment = useProjectStore((s) => s.createPayment);
  const updatePayment = useProjectStore((s) => s.updatePayment);
  const deletePayment = useProjectStore((s) => s.deletePayment);
  const vendors = useVendorStore((s) => s.vendors);
  const fetchVendors = useVendorStore((s) => s.fetchVendors);
  // What the workspace role allows (ADR-0026): hard delete needs
  // finance.payment.delete; adding and editing payments finance.record.
  const canDelete = useCan("finance.payment.delete");
  const canRecord = useCan("finance.record");

  const [modalOpen, setModalOpen] = useState(false);
  const [editingPayment, setEditingPayment] = useState<VendorPayment | null>(null);
  const [viewingEvidencePayment, setViewingEvidencePayment] = useState<VendorPayment | null>(null);

  useEffect(() => {
    void fetchPayments(projectId);
    void fetchVendorSection(projectId);
    void fetchVendors();
  }, [projectId, fetchPayments, fetchVendorSection, fetchVendors]);

  // Excludes Cancelled engagements, same as Margin/Keuntungan's own
  // vendorCost already does — a cancelled engagement's contract value isn't
  // still owed to anyone. Individual payment history rows below are NOT
  // filtered by this: a payment that already happened stays a historical
  // fact regardless of the engagement's current status. See PLAN.md
  // "Financial Calculation Correctness".
  const activeEngagements = vendorEngagements.filter((pv) => pv.engagementStatus !== "Cancelled");
  const totalContractValue = activeEngagements.reduce((sum, pv) => sum + pv.contractValue, 0);
  const totalPaid = activeEngagements.reduce((sum, pv) => sum + pv.paidAmount, 0);
  const totalRemaining = totalContractValue - totalPaid;
  const { page, setPage, totalPages, totalItems, pageSize, pageItems } = usePagination(payments);

  // Payment and proof are one request: if it fails nothing was saved and the drawer stays open (errors show inside it).
  async function handleAddPayment(values: PaymentFormValues, options: SaveOptions) {
    await createPayment(projectId, values, options);
    setModalOpen(false);
  }

  async function handleEditPayment(values: PaymentUpdateFormValues) {
    if (!editingPayment) return;
    await updatePayment(projectId, editingPayment.id, values, editingPayment.updatedAt);
    setEditingPayment(null);
  }

  // No "Yakin?" step: it disappears at once and the toast offers "Urungkan" (the server keeps it 24 hours).
  function handleDeletePayment(payment: { id: string }) {
    void deleteWithUndo("Pembayaran vendor", () => deletePayment(projectId, payment.id));
  }

  function vendorNameOf(payment: VendorPayment): string {
    const pv = vendorEngagements.find((v) => v.id === payment.projectVendorId);
    const vendor = pv ? vendors.find((v) => v.id === pv.vendorId) : null;
    return vendor?.name ?? (pv?.vendorName || "Vendor tidak diketahui");
  }

  const columns: Column<VendorPayment>[] = [
    { id: "vendor", header: "Vendor", primary: true, truncate: true, className: "w-[28%] min-w-40", cell: (p) => <span className="font-medium" title={vendorNameOf(p)}>{vendorNameOf(p)}</span> },
    { id: "type", header: "Jenis", card: true, className: "whitespace-nowrap", cell: (p) => p.type },
    { id: "date", header: "Tanggal", card: true, className: "w-32 whitespace-nowrap", cell: (p) => formatDate(p.paymentDate) },
    { id: "amount", header: "Nominal", align: "right", card: true, cell: (p) => formatCurrency(p.amount) },
    { id: "method", header: "Metode", className: "whitespace-nowrap", cell: (p) => p.method || "-" },
    { id: "ref", header: "No. referensi", truncate: true, className: "w-36 tabular-nums", cell: (p) => <span title={p.referenceNumber || undefined}>{p.referenceNumber || "-"}</span> },
    { id: "proof", header: "Bukti", className: "whitespace-nowrap", cell: (p) => <EvidenceBadge complete={p.evidenceComplete} /> },
  ];

  return (
    <div id="pembayaran">
      <FinanceSection
        title="Pembayaran ke vendor"
        action={
          canRecord && (
            <Button size="sm" icon={<Plus className="h-3.5 w-3.5" />} onClick={() => setModalOpen(true)}>
              Tambah pembayaran
            </Button>
          )
        }
      >
        <FinanceSummary
          items={[
            { label: "Total nilai kerja sama", value: formatCurrency(totalContractValue) },
            { label: "Sudah dibayar", value: formatCurrency(totalPaid) },
            { label: "Sisa pembayaran", value: formatCurrency(totalRemaining) },
          ]}
        />
        <LedgerTable
          label="Pembayaran ke vendor"
          columns={columns}
          rows={pageItems}
          rowId={(p) => p.id}
          page={page}
          pageSize={pageSize}
          totalPages={totalPages}
          totalItems={totalItems}
          onPageChange={setPage}
          empty={{ title: "Belum ada pembayaran ke vendor", description: "Catat setiap pembayaran ke vendor beserta tagihan dan bukti transfernya." }}
          actions={(p) => (
            <RowActions
              rowLabel={`${vendorNameOf(p)} ${p.type} ${formatCurrency(p.amount)}`}
              primary={{ label: "Ubah pembayaran", icon: <Pencil />, onClick: () => setEditingPayment(p), hidden: !canRecord }}
              items={[
                { label: "Lihat bukti", icon: <Eye className="h-4 w-4" />, onSelect: () => setViewingEvidencePayment(p) },
                { label: "Hapus pembayaran", icon: <Trash2 className="h-4 w-4" />, tone: "danger", onSelect: () => handleDeletePayment(p), hidden: !canDelete },
              ]}
            />
          )}
        />
      </FinanceSection>

      <RecordPaymentDrawer
        open={modalOpen}
        onClose={() => setModalOpen(false)}
        title="Catat pembayaran vendor"
        description="Catat pembayaran baru ke vendor untuk proyek ini, beserta tagihan dan bukti transfernya."
        slots={{ invoice: true }}
        vendorOptions={vendorEngagements.map((pv) => {
          const vendor = vendors.find((v) => v.id === pv.vendorId);
          return { value: pv.id, label: vendor ? (vendor.city ? `${vendor.name} — ${vendor.city}` : vendor.name) : pv.vendorName || "Vendor" };
        })}
        owed={(projectVendorId) => ({
          contractValue: vendorEngagements.find((e) => e.id === projectVendorId)?.contractValue ?? 0,
          existing: payments.filter((p) => p.projectVendorId === projectVendorId),
          owedLabel: "sisa pembayaran vendor",
        })}
        onSubmit={handleAddPayment}
      />

      {editingPayment && (
        <EditPaymentModal
          open={Boolean(editingPayment)}
          payment={editingPayment}
          onClose={() => setEditingPayment(null)}
          onSubmit={handleEditPayment}
          contractValue={vendorEngagements.find((e) => e.id === editingPayment.projectVendorId)?.contractValue ?? 0}
          others={payments.filter((p) => p.projectVendorId === editingPayment.projectVendorId && p.id !== editingPayment.id)}
        />
      )}

      {viewingEvidencePayment && (
        <Suspense fallback={null}>
          <EvidenceListModal
            open
            projectId={projectId}
            title={`Bukti — ${viewingEvidencePayment.type}`}
            items={viewingEvidencePayment.evidence}
            onClose={() => setViewingEvidencePayment(null)}
          />
        </Suspense>
      )}
    </div>
  );
}

// Edit scope is the ledger fields only -- no vendor picker (re-parenting a
// payment isn't supported, PLAN.md §2.5) and no file inputs (already
// covered by the existing generic evidence tab).
function EditPaymentModal({
  open,
  payment,
  onClose,
  onSubmit,
  contractValue,
  others,
}: {
  open: boolean;
  payment: VendorPayment;
  onClose: () => void;
  onSubmit: (values: PaymentUpdateFormValues) => Promise<void>;
  contractValue: number;
  /** This engagement's other payments (the one being edited left out). */
  others: VendorPayment[];
}) {
  const [values, setValues] = useState<PaymentUpdateFormValues>({
    type: payment.type,
    amount: payment.amount,
    paymentDate: payment.paymentDate,
    method: payment.method as PaymentUpdateFormValues["method"],
    referenceNumber: payment.referenceNumber,
    notes: payment.notes,
  });
  const [errors, setErrors] = useState<Partial<Record<keyof PaymentUpdateFormValues, string>>>({});
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  function set<K extends keyof PaymentUpdateFormValues>(key: K, value: PaymentUpdateFormValues[K]) {
    setValues((prev) => ({ ...prev, [key]: value }));
  }

  async function handleSubmit() {
    const result = paymentUpdateSchema.safeParse(values);
    const fieldErrors: Partial<Record<keyof PaymentUpdateFormValues, string>> = {};
    if (!result.success) {
      for (const issue of result.error.issues) {
        fieldErrors[issue.path[0] as keyof PaymentUpdateFormValues] ??= issue.message;
      }
    } else {
      const over = overpaymentMessage(contractValue, others, result.data, formatCurrency, "sisa pembayaran vendor", payment);
      if (over) fieldErrors.amount = over;
    }
    if (!result.success || fieldErrors.amount) {
      setErrors(fieldErrors);
      return;
    }
    setSubmitting(true);
    setError(null);
    try {
      await onSubmit(result.data);
    } catch (err) {
      setError(getApiErrorMessage(err, "Gagal memperbarui pembayaran"));
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <Modal
      open={open}
      onClose={onClose}
      title="Ubah pembayaran"
      description="Perbarui detail pembayaran ini. Bukti yang sudah terlampir tidak berubah."
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>Batal</Button>
          <Button onClick={() => void handleSubmit()} disabled={submitting}>
            {submitting ? "Menyimpan..." : "Simpan Perubahan"}
          </Button>
        </>
      }
    >
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        {error && (
          <p className="sm:col-span-2 rounded-md border border-danger/30 bg-danger-soft px-3.5 py-2.5 text-[13px] font-medium text-danger-text">{error}</p>
        )}
        <Field label="Jenis pembayaran" required>
          <Select value={values.type} onChange={(e) => set("type", e.target.value as PaymentUpdateFormValues["type"])}>
            {PAYMENT_TYPE_OPTIONS.map((t) => (
              <option key={t} value={t}>{t}</option>
            ))}
          </Select>
        </Field>
        <Field label="Nominal (Rp)" required error={errors.amount}>
          <CurrencyInput value={values.amount} onChange={(n) => set("amount", n)} />
        </Field>
        <Field label="Tanggal" required hint={errors.paymentDate}>
          <Input type="date" value={values.paymentDate} onChange={(e) => set("paymentDate", e.target.value)} />
        </Field>
        <Field label="Metode" required hint={errors.method}>
          <Select value={values.method} onChange={(e) => set("method", e.target.value as PaymentUpdateFormValues["method"])}>
            {PAYMENT_METHOD_OPTIONS.map((m) => (
              <option key={m} value={m}>{m}</option>
            ))}
          </Select>
        </Field>
        <Field label="No. referensi" hint={errors.referenceNumber}>
          <Input value={values.referenceNumber} onChange={(e) => set("referenceNumber", e.target.value)} />
        </Field>
        <div className="sm:col-span-2">
          <Field label="Catatan">
            <Textarea rows={2} value={values.notes} onChange={(e) => set("notes", e.target.value)} />
          </Field>
        </div>
      </div>
    </Modal>
  );
}
