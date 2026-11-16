import { lazy, Suspense, useEffect, useState } from "react";
import { useCan } from "@/shared/stores/usePermissionStore";
import { Plus, Pencil, Trash2, Eye } from "lucide-react";
import { Link } from "react-router-dom";
import { EmptyState } from "@/shared/components/feedback/EmptyState";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import { Button } from "@/shared/components/ui/Button";
import { RecordPaymentDrawer } from "@/modules/projects/components/detail/RecordPaymentDrawer";
import type { PaymentFormValues } from "@/modules/projects/schemas/payment.schema";
import { deleteWithUndo } from "@/shared/lib/undo-delete";
import { Modal } from "@/shared/components/ui/Modal";
import { Input, Textarea, Select, Field } from "@/shared/components/ui/Input";
import { CurrencyInput } from "@/shared/components/ui/CurrencyInput";
import { RowActions } from "@/shared/components/ui/RowActions";
import type { Column } from "@/shared/data-table/DataTable";
import { EvidenceBadge, FinanceSection, FinanceSummary, LedgerTable } from "@/modules/projects/components/detail/finance-ui";
import { usePagination } from "@/shared/hooks/usePagination";
import { useProjectStore, type SaveOptions } from "@/modules/projects/stores/useProjectStore";
import {
  venuePaymentUpdateSchema,
  VENUE_PAYMENT_TYPE_OPTIONS,
  type VenuePaymentUpdateFormValues,
} from "@/modules/projects/schemas/venue-payment.schema";
import { PAYMENT_METHOD_OPTIONS } from "@/modules/projects/schemas/payment.schema";
import type { VenuePayment } from "@/modules/projects/types";
import { getApiErrorMessage } from "@/shared/lib/api-error";
import { remainingToPay, overpaymentMessage } from "@/modules/projects/lib/payments";
import { formatCurrency, formatDate } from "@/shared/lib/formatters";

// Sibling of ProjectPaymentsSection (Pembayaran ke Vendor) minus the vendor
// picker -- a project has at most one venue (project.venueId), nothing to
// choose. See PLAN.md "Venue Payments + Pembayaran tab restructuring".
// The proof viewer is only needed once someone opens a payment's proof.
const EvidenceListModal = lazy(() => import("@/shared/components/ui/EvidenceListModal").then((m) => ({ default: m.EvidenceListModal })));

export function VenuePaymentsSection({ projectId }: { projectId: string }) {
  const project = useProjectStore((s) => s.currentProject);
  const venuePayments = useProjectStore((s) => s.venuePayments);
  const fetchVenuePayments = useProjectStore((s) => s.fetchVenuePayments);
  const createVenuePayment = useProjectStore((s) => s.createVenuePayment);
  const updateVenuePayment = useProjectStore((s) => s.updateVenuePayment);
  const deleteVenuePayment = useProjectStore((s) => s.deleteVenuePayment);
  // Hard delete is Owner-or-Admin (broadened from Owner-only per explicit
  // user request) -- mirrors ProjectHeaderCard.tsx's own "!== \"Staff\"" idiom
  // for the same Owner-or-Admin bar, since those are the only 3 roles. Edit
  // has no role restriction at all.
  const canDelete = useCan("finance.payment.delete");
  const canRecord = useCan("finance.record");

  const [modalOpen, setModalOpen] = useState(false);
  const [editingPayment, setEditingPayment] = useState<VenuePayment | null>(null);
  const [viewingEvidencePayment, setViewingEvidencePayment] = useState<VenuePayment | null>(null);

  useEffect(() => {
    void fetchVenuePayments(projectId);
  }, [projectId, fetchVenuePayments]);

  const hasVenue = Boolean(project?.venueId);
  const venueValue = (project?.venueRentalPrice ?? 0) + (project?.venueCharge ?? 0);
  const totalPaid = venuePayments.reduce(
    (sum, p) => (p.type === "Refund" ? sum - p.amount : sum + p.amount),
    0
  );
  // Tambahan is outside the venue cost, so it does not reduce what is still owed.
  const totalRemaining = remainingToPay(venueValue, venuePayments);
  const { page, setPage, totalPages, totalItems, pageSize, pageItems } = usePagination(venuePayments);

  // Payment and proof are one request: if it fails nothing was saved and the drawer stays open (errors show inside it).
  async function handleAddPayment(values: PaymentFormValues, options: SaveOptions) {
    await createVenuePayment(projectId, values, options);
    setModalOpen(false);
  }

  async function handleEditPayment(values: VenuePaymentUpdateFormValues) {
    if (!editingPayment) return;
    await updateVenuePayment(projectId, editingPayment.id, values, editingPayment.updatedAt);
    setEditingPayment(null);
  }

  // No "Yakin?" step: it disappears at once and the toast offers "Urungkan" (the server keeps it 24 hours).
  function handleDeletePayment(payment: { id: string }) {
    void deleteWithUndo("Pembayaran venue", () => deleteVenuePayment(projectId, payment.id));
  }

  // A venue can be detached later (Venue tab's "Lepas") after payments were already recorded against it -- those rows
  // must stay visible as a historical fact. So the "nothing to show" empty state only applies when there is truly nothing:
  // no venue attached AND no payment history at all.
  if (!hasVenue && venuePayments.length === 0) {
    return (
      <FinanceSection title="Pembayaran ke venue">
        <div className="rounded-lg border border-border bg-card">
          <EmptyState
            title="Belum ada venue terpasang"
            description="Pasang venue proyek ini dulu, lalu catat pembayaran sewanya di sini."
            action={
              <Link to={ROUTE_PATHS.projectDetail(projectId, "vendor")} className="text-[13px] font-medium text-primary underline-offset-2 hover:underline">
                Buka Vendor & venue
              </Link>
            }
          />
        </div>
      </FinanceSection>
    );
  }

  const columns: Column<VenuePayment>[] = [
    { id: "type", header: "Jenis", primary: true, className: "w-32", cell: (p) => <span className="font-medium whitespace-nowrap">{p.type}</span> },
    { id: "date", header: "Tanggal", card: true, className: "w-32 whitespace-nowrap", cell: (p) => formatDate(p.paymentDate) },
    { id: "amount", header: "Nominal", align: "right", card: true, cell: (p) => formatCurrency(p.amount) },
    { id: "method", header: "Metode", className: "whitespace-nowrap", cell: (p) => p.method || "-" },
    { id: "ref", header: "No. referensi", truncate: true, className: "w-40 tabular-nums", cell: (p) => <span title={p.referenceNumber || undefined}>{p.referenceNumber || "-"}</span> },
    { id: "proof", header: "Bukti", card: true, className: "whitespace-nowrap", cell: (p) => <EvidenceBadge complete={p.evidenceComplete} /> },
  ];

  return (
    <div>
      <FinanceSection
        title="Pembayaran ke venue"
        action={
          hasVenue && canRecord ? (
            <Button size="sm" icon={<Plus className="h-3.5 w-3.5" />} onClick={() => setModalOpen(true)}>
              Tambah pembayaran
            </Button>
          ) : undefined
        }
        notice={
          !hasVenue ? (
            <p className="rounded-md border border-border bg-muted px-3.5 py-2.5 text-[13px] text-text-secondary">
              Venue sudah dilepas dari proyek ini. Riwayat pembayaran berikut tetap ditampilkan sebagai catatan.
            </p>
          ) : undefined
        }
      >
        <FinanceSummary
          items={[
            { label: "Nilai sewa venue", value: formatCurrency(venueValue) },
            { label: "Sudah dibayar", value: formatCurrency(totalPaid) },
            totalRemaining < 0
              ? { label: "Sisa pembayaran", value: formatCurrency(0), note: `Lebih bayar ${formatCurrency(-totalRemaining)}` }
              : { label: "Sisa pembayaran", value: formatCurrency(totalRemaining) },
          ]}
        />
        <LedgerTable
          label="Pembayaran ke venue"
          columns={columns}
          rows={pageItems}
          rowId={(p) => p.id}
          page={page}
          pageSize={pageSize}
          totalPages={totalPages}
          totalItems={totalItems}
          onPageChange={setPage}
          empty={{ title: "Belum ada pembayaran ke venue", description: "Catat setiap pembayaran sewa beserta tagihan dan bukti transfernya." }}
          actions={(p) => (
            <RowActions
              rowLabel={`${p.type} ${formatCurrency(p.amount)}`}
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
        title="Catat pembayaran venue"
        description="Catat pembayaran baru ke venue untuk proyek ini, beserta tagihan dan bukti transfernya."
        slots={{ invoice: true }}
        owed={() => ({ contractValue: venueValue, existing: venuePayments, owedLabel: "sisa biaya venue" })}
        onSubmit={handleAddPayment}
      />

      {editingPayment && (
        <EditVenuePaymentModal
          open={Boolean(editingPayment)}
          payment={editingPayment}
          onClose={() => setEditingPayment(null)}
          onSubmit={handleEditPayment}
          venueValue={venueValue}
          existing={venuePayments.filter((p) => p.id !== editingPayment.id)}
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

// Edit scope is the ledger fields only -- no file inputs (already covered by
// the existing generic evidence tab), see PLAN.md §2.5.
function EditVenuePaymentModal({
  open,
  payment,
  onClose,
  onSubmit,
  venueValue,
  existing,
}: {
  open: boolean;
  payment: VenuePayment;
  onClose: () => void;
  onSubmit: (values: VenuePaymentUpdateFormValues) => Promise<void>;
  venueValue: number;
  /** The other venue payments of the project (this one left out). */
  existing: VenuePayment[];
}) {
  const [values, setValues] = useState<VenuePaymentUpdateFormValues>({
    type: payment.type,
    amount: payment.amount,
    paymentDate: payment.paymentDate,
    method: payment.method as VenuePaymentUpdateFormValues["method"],
    referenceNumber: payment.referenceNumber,
    notes: payment.notes,
  });
  const [errors, setErrors] = useState<Partial<Record<keyof VenuePaymentUpdateFormValues, string>>>({});
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  function set<K extends keyof VenuePaymentUpdateFormValues>(key: K, value: VenuePaymentUpdateFormValues[K]) {
    setValues((prev) => ({ ...prev, [key]: value }));
  }

  async function handleSubmit() {
    const result = venuePaymentUpdateSchema.safeParse(values);
    const fieldErrors: Partial<Record<keyof VenuePaymentUpdateFormValues, string>> = {};
    if (!result.success) {
      for (const issue of result.error.issues) {
        fieldErrors[issue.path[0] as keyof VenuePaymentUpdateFormValues] ??= issue.message;
      }
    } else {
      const over = overpaymentMessage(venueValue, existing, result.data, formatCurrency, "sisa biaya venue", payment);
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
      setError(getApiErrorMessage(err, "Gagal memperbarui pembayaran venue"));
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <Modal
      open={open}
      onClose={onClose}
      title="Ubah pembayaran venue"
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
          <Select value={values.type} onChange={(e) => set("type", e.target.value as VenuePaymentUpdateFormValues["type"])}>
            {VENUE_PAYMENT_TYPE_OPTIONS.map((t) => (
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
          <Select value={values.method} onChange={(e) => set("method", e.target.value as VenuePaymentUpdateFormValues["method"])}>
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
