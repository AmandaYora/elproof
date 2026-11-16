import { lazy, Suspense, useEffect, useState } from "react";
import { useCan } from "@/shared/stores/usePermissionStore";
import { Plus, Pencil, Trash2, Eye, BadgeCheck } from "lucide-react";
import { Badge } from "@/shared/components/ui/Badge";
import { Button } from "@/shared/components/ui/Button";
import { Modal } from "@/shared/components/ui/Modal";
import { Input, Textarea, Select, Field } from "@/shared/components/ui/Input";
import { CurrencyInput } from "@/shared/components/ui/CurrencyInput";
import { RowActions } from "@/shared/components/ui/RowActions";
import type { Column } from "@/shared/data-table/DataTable";
import { FinanceSection, FinanceSummary, LedgerTable } from "@/modules/projects/components/detail/finance-ui";
import { RecordPaymentDrawer } from "@/modules/projects/components/detail/RecordPaymentDrawer";
import type { PaymentFormValues } from "@/modules/projects/schemas/payment.schema";
import { deleteWithUndo } from "@/shared/lib/undo-delete";
import { ConfirmDialog } from "@/shared/components/ui/ConfirmDialog";
import { ProofStamp } from "@/shared/components/ui/ProofStamp";
import { usePagination } from "@/shared/hooks/usePagination";
import { useProjectStore, type SaveOptions } from "@/modules/projects/stores/useProjectStore";
import {
  clientPaymentUpdateSchema,
  CLIENT_PAYMENT_TYPE_OPTIONS,
  type ClientPaymentUpdateFormValues,
} from "@/modules/projects/schemas/client-payment.schema";
import { PAYMENT_METHOD_OPTIONS } from "@/modules/projects/schemas/payment.schema";
import type { ClientPayment } from "@/modules/projects/types";
import { verifiedLabel } from "@/modules/projects/lib/proof";
import { getApiErrorMessage } from "@/shared/lib/api-error";
import { remainingToPay, overpaymentMessage, countsAgainstContract, countsTowardContract } from "@/modules/projects/lib/payments";
import { toast } from "@/shared/stores/useToastStore";
import { formatCurrency, formatDate } from "@/shared/lib/formatters";

// Simpler sibling of ProjectPaymentsSection (PLAN.md "Uang Masuk dari
// Client") — no vendor picker (nothing to pick), a single Bukti Ada/Belum
// Ada badge instead of the vendor version's two-part Lengkap/Belum Lengkap
// (there's only one evidence slot to begin with, nothing to be "partially"
// complete about), and the proof file is attached in the same submit that
// records the payment.
// The proof viewer is only needed once someone opens a payment's proof.
const EvidenceListModal = lazy(() => import("@/shared/components/ui/EvidenceListModal").then((m) => ({ default: m.EvidenceListModal })));

export function ClientPaymentsSection({ projectId }: { projectId: string }) {
  const project = useProjectStore((s) => s.currentProject);
  const clientPayments = useProjectStore((s) => s.clientPayments);
  const fetchClientPayments = useProjectStore((s) => s.fetchClientPayments);
  const createClientPayment = useProjectStore((s) => s.createClientPayment);
  const updateClientPayment = useProjectStore((s) => s.updateClientPayment);
  const deleteClientPayment = useProjectStore((s) => s.deleteClientPayment);
  const verifyClientPayment = useProjectStore((s) => s.verifyClientPayment);
  // What the workspace role allows (ADR-0026): hard delete needs
  // finance.payment.delete; adding and editing payments finance.record.
  const canDelete = useCan("finance.payment.delete");
  const canRecord = useCan("finance.record");

  const [modalOpen, setModalOpen] = useState(false);
  const [editingPayment, setEditingPayment] = useState<ClientPayment | null>(null);
  const [deletingPayment, setDeletingPayment] = useState<ClientPayment | null>(null);
  const [deleting, setDeleting] = useState(false);
  const [viewingEvidencePayment, setViewingEvidencePayment] = useState<ClientPayment | null>(null);
  const [verifying, setVerifying] = useState<ClientPayment | null>(null);
  const [verifyError, setVerifyError] = useState<string | null>(null);
  const [verifySubmitting, setVerifySubmitting] = useState(false);

  useEffect(() => {
    void fetchClientPayments(projectId);
  }, [projectId, fetchClientPayments]);

  const contractValue = project?.contractValue ?? 0;
  // A couple's submission from the portal counts as received only once it is verified, the server's rule for the
  // overview and the schedule too (QA RT-03, ADR-0047); until then it is shown apart as waiting for verification.
  const counted = clientPayments.filter(countsTowardContract);
  const totalReceived = counted.reduce((sum, p) => (p.type === "Refund" ? sum - p.amount : sum + p.amount), 0);
  const pendingVerification = clientPayments.filter((p) => !countsTowardContract(p)).reduce((sum, p) => (p.type === "Refund" ? sum - p.amount : sum + p.amount), 0);
  // Sisa tagihan counts only DP/Termin/Pelunasan (minus refunds): Tambahan is
  // outside the contract, so it must not shrink what the client still owes.
  const totalRemaining = remainingToPay(contractValue, counted);
  const { page, setPage, totalPages, totalItems, pageSize, pageItems } = usePagination(clientPayments);

  async function handleVerify() {
    if (!verifying) return;
    setVerifySubmitting(true);
    setVerifyError(null);
    try {
      await verifyClientPayment(projectId, verifying.id);
      toast.success("Pembayaran diverifikasi");
      setVerifying(null);
    } catch (err) {
      setVerifyError(getApiErrorMessage(err, "Pembayaran belum bisa diverifikasi. Coba lagi."));
    } finally {
      setVerifySubmitting(false);
    }
  }

  // Payment and proof are one request: if it fails nothing was saved and the drawer stays open (errors show inside it).
  async function handleAddPayment(values: PaymentFormValues, options: SaveOptions) {
    await createClientPayment(projectId, values, options);
    setModalOpen(false);
    toast.success("Pembayaran dicatat");
  }

  async function handleEditPayment(values: ClientPaymentUpdateFormValues) {
    if (!editingPayment) return;
    await updateClientPayment(projectId, editingPayment.id, values, editingPayment.updatedAt);
    setEditingPayment(null);
    toast.success("Pembayaran diperbarui");
  }

  // Routine payments disappear at once with "Urungkan"; a VERIFIED one is something the client has seen as proven, so it asks first.
  function requestDelete(payment: ClientPayment) {
    if (payment.verifiedAt) setDeletingPayment(payment);
    else void deleteWithUndo("Pembayaran", () => deleteClientPayment(projectId, payment.id));
  }

  async function handleDeleteVerified() {
    if (!deletingPayment) return;
    setDeleting(true);
    const done = await deleteWithUndo("Pembayaran terverifikasi", () => deleteClientPayment(projectId, deletingPayment.id));
    setDeleting(false);
    if (done) setDeletingPayment(null);
  }

  const columns: Column<ClientPayment>[] = [
    {
      id: "type",
      header: "Jenis",
      primary: true,
      className: "w-32",
      cell: (p) => (
        <span className="flex flex-col items-start gap-1">
          <span className="font-medium whitespace-nowrap">{p.type}</span>
          {/* Sent from the Client Portal (ADR-0047): the WO still checks the money arrived before verifying. */}
          {p.source === "client" && <Badge tone="info">Dikirim pasangan</Badge>}
        </span>
      ),
    },
    { id: "date", header: "Tanggal", card: true, className: "w-32 whitespace-nowrap", cell: (p) => formatDate(p.paymentDate) },
    { id: "amount", header: "Nominal", align: "right", card: true, cell: (p) => formatCurrency(p.amount) },
    { id: "method", header: "Metode", className: "whitespace-nowrap", cell: (p) => p.method || "-" },
    { id: "ref", header: "No. referensi", truncate: true, className: "w-40 tabular-nums", cell: (p) => <span title={p.referenceNumber || undefined}>{p.referenceNumber || "-"}</span> },
    {
      id: "proof",
      header: "Bukti",
      card: true,
      className: "whitespace-nowrap",
      cell: (p) => <PaymentProofCell payment={p} canVerify={canRecord} onOpen={() => setViewingEvidencePayment(p)} onVerify={() => setVerifying(p)} />,
    },
  ];

  return (
    <div id="pembayaran-client">
      <FinanceSection
        title="Uang masuk dari klien"
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
            { label: "Nilai kontrak", value: formatCurrency(contractValue) },
            {
              label: "Total diterima",
              value: formatCurrency(totalReceived),
              note: pendingVerification > 0 ? `${formatCurrency(pendingVerification)} menunggu verifikasi` : undefined,
              noteTone: "muted",
            },
            {
              label: "Sisa tagihan",
              value: formatCurrency(Math.max(totalRemaining, 0)),
              note: totalRemaining < 0 ? `Lebih bayar ${formatCurrency(-totalRemaining)}` : undefined,
            },
          ]}
        />
        <LedgerTable
          label="Pembayaran dari klien"
          columns={columns}
          rows={pageItems}
          rowId={(p) => p.id}
          page={page}
          pageSize={pageSize}
          totalPages={totalPages}
          totalItems={totalItems}
          onPageChange={setPage}
          empty={{ title: "Belum ada pembayaran klien", description: "Catat setiap uang yang masuk dari klien beserta bukti transfernya, lalu verifikasi agar klien melihat tandanya." }}
          actions={(p) => (
            <RowActions
              rowLabel={`${p.type} ${formatCurrency(p.amount)}`}
              primary={{ label: "Ubah pembayaran", icon: <Pencil />, onClick: () => setEditingPayment(p), hidden: !canRecord }}
              items={[
                { label: "Lihat bukti", icon: <Eye className="h-4 w-4" />, onSelect: () => setViewingEvidencePayment(p) },
                { label: "Verifikasi pembayaran", icon: <BadgeCheck className="h-4 w-4" />, onSelect: () => setVerifying(p), hidden: !(canRecord && p.evidence.length > 0 && !p.verifiedAt) },
                { label: "Hapus pembayaran", icon: <Trash2 className="h-4 w-4" />, tone: "danger", onSelect: () => requestDelete(p), hidden: !canDelete },
              ]}
            />
          )}
        />
      </FinanceSection>

      <RecordPaymentDrawer
        open={modalOpen}
        onClose={() => setModalOpen(false)}
        title="Catat pembayaran klien"
        description="Catat pembayaran baru dari klien untuk proyek ini, beserta bukti transfernya."
        slots={{ invoice: false }}
        owed={() => ({ contractValue, existing: clientPayments, owedLabel: "sisa tagihan" })}
        onSubmit={handleAddPayment}
      />

      {editingPayment && (
        <EditClientPaymentModal
          open={Boolean(editingPayment)}
          payment={editingPayment}
          onClose={() => setEditingPayment(null)}
          onSubmit={handleEditPayment}
          contractValue={contractValue}
          existing={clientPayments.filter((p) => p.id !== editingPayment.id)}
        />
      )}

      <ConfirmDialog
        open={deletingPayment !== null}
        title="Hapus pembayaran terverifikasi?"
        confirmLabel="Hapus pembayaran"
        isSubmitting={deleting}
        onConfirm={() => void handleDeleteVerified()}
        onClose={() => setDeletingPayment(null)}
      >
        <p>
          Pembayaran {deletingPayment ? `${formatCurrency(deletingPayment.amount)} (${deletingPayment.type})` : ""} sudah ditandai terverifikasi dan klien melihat tandanya. Tanda dan bukti
          transfernya ikut hilang. Anda masih bisa mengurungkannya beberapa detik setelah ini, dan server menyimpannya 24 jam.
        </p>
      </ConfirmDialog>

      <ConfirmDialog
        open={verifying !== null}
        title="Verifikasi pembayaran ini?"
        confirmLabel="Verifikasi pembayaran"
        tone="primary"
        isSubmitting={verifySubmitting}
        error={verifyError}
        onConfirm={() => void handleVerify()}
        onClose={() => setVerifying(null)}
      >
        <p>
          Anda menyatakan {verifying ? `${formatCurrency(verifying.amount)} (${verifying.type})` : "pembayaran ini"} sudah diterima dan buktinya sesuai. Klien akan
          melihat tanda terverifikasi beserta nama Anda. Mengubah pembayaran ini nanti akan menghapus verifikasinya.
        </p>
      </ConfirmDialog>

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

/**
 * The proof state of one payment, on one line: a stamp when verified (who and when on hover; opens the file), else
 * whether a file is attached and, for roles that may, the button to verify it.
 */
function PaymentProofCell({ payment, canVerify, onOpen, onVerify }: { payment: ClientPayment; canVerify: boolean; onOpen: () => void; onVerify: () => void }) {
  const proof = payment.evidence[0];
  if (payment.verifiedAt && proof) {
    return <ProofStamp proof={proof} label="Diverifikasi" detail={verifiedLabel(payment.verifiedAt, payment.verifiedBy)} onOpen={onOpen} className="min-h-7 py-0.5 whitespace-nowrap" />;
  }
  if (!proof) return <Badge tone="warning">Belum ada bukti</Badge>;
  return (
    <span className="inline-flex items-center gap-1.5 whitespace-nowrap">
      <Badge tone="neutral">Bukti ada</Badge>
      {canVerify && (
        <Button size="sm" variant="ghost" onClick={onVerify} aria-label={`Verifikasi ${payment.type} ${formatCurrency(payment.amount)}`}>
          Verifikasi
        </Button>
      )}
    </span>
  );
}

// Edit scope is the ledger fields only -- no file input (already covered by
// the existing generic evidence tab), see PLAN.md §2.5.
function EditClientPaymentModal({
  open,
  payment,
  onClose,
  onSubmit,
  contractValue,
  existing,
}: {
  open: boolean;
  payment: ClientPayment;
  onClose: () => void;
  onSubmit: (values: ClientPaymentUpdateFormValues) => Promise<void>;
  contractValue: number;
  /** Every other payment of the project (this one excluded). */
  existing: ClientPayment[];
}) {
  const remaining = remainingToPay(contractValue, existing);
  const [values, setValues] = useState<ClientPaymentUpdateFormValues>({
    type: payment.type,
    amount: payment.amount,
    paymentDate: payment.paymentDate,
    method: payment.method as ClientPaymentUpdateFormValues["method"],
    referenceNumber: payment.referenceNumber,
    notes: payment.notes,
  });
  const [errors, setErrors] = useState<Partial<Record<keyof ClientPaymentUpdateFormValues, string>>>({});
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  function set<K extends keyof ClientPaymentUpdateFormValues>(key: K, value: ClientPaymentUpdateFormValues[K]) {
    setValues((prev) => ({ ...prev, [key]: value }));
  }

  async function handleSubmit() {
    const result = clientPaymentUpdateSchema.safeParse(values);
    const fieldErrors: Partial<Record<keyof ClientPaymentUpdateFormValues, string>> = {};
    if (!result.success) {
      for (const issue of result.error.issues) {
        fieldErrors[issue.path[0] as keyof ClientPaymentUpdateFormValues] ??= issue.message;
      }
    } else {
      const over = overpaymentMessage(contractValue, existing, result.data, formatCurrency, undefined, payment);
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
      setError(getApiErrorMessage(err, "Pembayaran belum bisa diperbarui. Periksa koneksi Anda, lalu coba lagi."));
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <Modal
      open={open}
      onClose={onClose}
      title="Ubah pembayaran klien"
      description="Perbarui detail pembayaran ini. Bukti yang sudah terlampir tidak berubah."
      busy={submitting}
      error={error}
      footer={
        <>
          <Button variant="secondary" onClick={onClose} disabled={submitting}>Batal</Button>
          <Button onClick={() => void handleSubmit()} disabled={submitting}>
            {submitting ? "Menyimpan..." : "Simpan perubahan"}
          </Button>
        </>
      }
    >
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <Field label="Jenis pembayaran" required>
          <Select value={values.type} onChange={(e) => set("type", e.target.value as ClientPaymentUpdateFormValues["type"])}>
            {CLIENT_PAYMENT_TYPE_OPTIONS.map((t) => (
              <option key={t} value={t}>{t}</option>
            ))}
          </Select>
        </Field>
        <Field
          label="Nominal (Rp)"
          required
          error={errors.amount}
          hint={countsAgainstContract(values.type) && contractValue > 0 ? `Sisa tagihan ${formatCurrency(Math.max(remaining, 0))}` : undefined}
        >
          <CurrencyInput value={values.amount} onChange={(n) => set("amount", n)} aria-invalid={errors.amount ? true : undefined} />
        </Field>
        <Field label="Tanggal" required error={errors.paymentDate}>
          <Input type="date" value={values.paymentDate} onChange={(e) => set("paymentDate", e.target.value)} aria-invalid={errors.paymentDate ? true : undefined} />
        </Field>
        <Field label="Metode" required error={errors.method}>
          <Select value={values.method} onChange={(e) => set("method", e.target.value as ClientPaymentUpdateFormValues["method"])}>
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
