import { useEffect, useState } from "react";
import { useCan } from "@/shared/stores/usePermissionStore";
import { Plus, Pencil, Trash2, Eye, AlertTriangle } from "lucide-react";
import { Card, CardHeader, CardContent } from "@/shared/components/ui/Card";
import { Badge } from "@/shared/components/ui/Badge";
import { Button } from "@/shared/components/ui/Button";
import { Modal } from "@/shared/components/ui/Modal";
import { Input, Textarea, Select, Field } from "@/shared/components/ui/Input";
import { CurrencyInput } from "@/shared/components/ui/CurrencyInput";
import { Table, THead, TBody, TR, TH, TD } from "@/shared/components/ui/Table";
import { CardList, CardListField } from "@/shared/components/ui/CardList";
import { IconActionButton } from "@/shared/components/ui/IconActionButton";
import { EvidenceListModal } from "@/shared/components/ui/EvidenceListModal";
import { RecordPaymentDrawer } from "@/modules/projects/components/detail/RecordPaymentDrawer";
import type { PaymentFormValues } from "@/modules/projects/schemas/payment.schema";
import { ConfirmDialog } from "@/shared/components/ui/ConfirmDialog";
import { ProofStamp } from "@/shared/components/ui/ProofStamp";
import { Pagination } from "@/shared/components/ui/Pagination";
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
import { remainingToPay, overpaymentMessage, countsAgainstContract } from "@/modules/projects/lib/payments";
import { toast } from "@/shared/stores/useToastStore";
import { formatCurrency, formatDate } from "@/shared/lib/formatters";

// Simpler sibling of ProjectPaymentsSection (PLAN.md "Uang Masuk dari
// Client") — no vendor picker (nothing to pick), a single Bukti Ada/Belum
// Ada badge instead of the vendor version's two-part Lengkap/Belum Lengkap
// (there's only one evidence slot to begin with, nothing to be "partially"
// complete about), and the proof file is attached in the same submit that
// records the payment.
export function ClientPaymentsSection({ projectId }: { projectId: string }) {
  const project = useProjectStore((s) => s.currentProject);
  const clientPayments = useProjectStore((s) => s.clientPayments);
  const evidence = useProjectStore((s) => s.evidence);
  const fetchClientPayments = useProjectStore((s) => s.fetchClientPayments);
  const fetchEvidence = useProjectStore((s) => s.fetchEvidence);
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
  const [deleteError, setDeleteError] = useState<string | null>(null);
  const [deleting, setDeleting] = useState(false);
  const [viewingEvidencePayment, setViewingEvidencePayment] = useState<ClientPayment | null>(null);
  const [verifying, setVerifying] = useState<ClientPayment | null>(null);
  const [verifyError, setVerifyError] = useState<string | null>(null);
  const [verifySubmitting, setVerifySubmitting] = useState(false);

  useEffect(() => {
    void fetchClientPayments(projectId);
    void fetchEvidence(projectId);
  }, [projectId, fetchClientPayments, fetchEvidence]);

  const contractValue = project?.contractValue ?? 0;
  const totalReceived = clientPayments.reduce(
    (sum, p) => (p.type === "Refund" ? sum - p.amount : sum + p.amount),
    0
  );
  // Sisa tagihan counts only DP/Termin/Pelunasan (minus refunds): Tambahan is
  // outside the contract, so it must not shrink what the client still owes.
  const totalRemaining = remainingToPay(contractValue, clientPayments);
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
    await updateClientPayment(projectId, editingPayment.id, values);
    setEditingPayment(null);
    toast.success("Pembayaran diperbarui");
  }

  async function handleDeletePayment() {
    if (!deletingPayment) return;
    setDeleteError(null);
    setDeleting(true);
    try {
      await deleteClientPayment(projectId, deletingPayment.id);
      setDeletingPayment(null);
      toast.success("Pembayaran dihapus");
    } catch (err) {
      setDeleteError(getApiErrorMessage(err, "Gagal menghapus pembayaran client"));
    } finally {
      setDeleting(false);
    }
  }

  return (
    <div id="pembayaran-client">
      <Card>
        <CardHeader
          title="Uang Masuk dari Client"
          subtitle="Ringkasan nilai kontrak dan seluruh riwayat pembayaran dari client untuk project ini."
          action={
            canRecord && (<Button size="sm" icon={<Plus className="h-3.5 w-3.5" />} onClick={() => setModalOpen(true)}>
              Tambah Pembayaran
            </Button>)
          }
        />
        <CardContent className="flex flex-col gap-5">
          <div className="grid grid-cols-1 gap-3 xl:grid-cols-3">
            <SummaryStat label="Nilai Kontrak" value={formatCurrency(contractValue)} />
            <SummaryStat label="Total Diterima" value={formatCurrency(totalReceived)} />
            <SummaryStat
              label="Sisa Tagihan"
              value={formatCurrency(Math.max(totalRemaining, 0))}
              badge={totalRemaining < 0 ? `Lebih bayar ${formatCurrency(-totalRemaining)}` : undefined}
            />
          </div>

          {clientPayments.length === 0 ? (
            <p className="rounded-md border border-dashed border-border px-4 py-6 text-center text-[13px] text-text-secondary">
              Belum ada pembayaran client tercatat untuk project ini.
            </p>
          ) : (
            <>
            <CardList
              className="sm:hidden"
              items={pageItems}
              keyFor={(payment) => payment.id}
              renderItem={(payment) => (
                <>
                  <div className="flex items-start justify-between gap-3">
                    <span className="font-medium text-text-primary">{payment.type}</span>
                    <PaymentProofCell payment={payment} canVerify={canRecord} onOpen={() => setViewingEvidencePayment(payment)} onVerify={() => setVerifying(payment)} />
                  </div>
                  <div className="flex flex-col gap-1.5">
                    <CardListField label="Nominal" value={formatCurrency(payment.amount)} />
                    <CardListField label="Tanggal" value={formatDate(payment.paymentDate)} />
                    <CardListField label="Metode" value={payment.method} />
                    <CardListField label="No. Referensi" value={payment.referenceNumber} />
                  </div>
                  <div className="flex items-center gap-2 pt-1">
                    <IconActionButton icon={Eye} readOnly label="Lihat Bukti" tone="info" onClick={() => setViewingEvidencePayment(payment)} />
                    {canRecord && <IconActionButton icon={Pencil} label="Ubah Pembayaran" tone="neutral" onClick={() => setEditingPayment(payment)} />}
                    {canDelete && (
                      <IconActionButton icon={Trash2} label="Hapus Pembayaran" tone="danger" onClick={() => { setDeletingPayment(payment); setDeleteError(null); }} />
                    )}
                  </div>
                </>
              )}
            />
            <div className="hidden sm:block">
            <Table>
              <THead>
                <TR>
                  <TH>Jenis</TH>
                  <TH className="text-right">Nominal</TH>
                  <TH>Tanggal</TH>
                  <TH>Metode</TH>
                  <TH>No. Referensi</TH>
                  <TH>Bukti</TH>
                  <TH>Aksi</TH>
                </TR>
              </THead>
              <TBody>
                {pageItems.map((payment) => (
                  <TR key={payment.id}>
                    <TD className="font-medium">{payment.type}</TD>
                    <TD className="text-right tabular-nums">{formatCurrency(payment.amount)}</TD>
                    <TD>{formatDate(payment.paymentDate)}</TD>
                    <TD>{payment.method}</TD>
                    <TD>{payment.referenceNumber}</TD>
                    <TD><PaymentProofCell payment={payment} canVerify={canRecord} onOpen={() => setViewingEvidencePayment(payment)} onVerify={() => setVerifying(payment)} /></TD>
                    <TD>
                      <div className="flex items-center gap-2">
                        <IconActionButton icon={Eye} readOnly label="Lihat Bukti" tone="info" onClick={() => setViewingEvidencePayment(payment)} />
                        {canRecord && <IconActionButton icon={Pencil} label="Ubah Pembayaran" tone="neutral" onClick={() => setEditingPayment(payment)} />}
                        {canDelete && (
                          <IconActionButton icon={Trash2} label="Hapus Pembayaran" tone="danger" onClick={() => { setDeletingPayment(payment); setDeleteError(null); }} />
                        )}
                      </div>
                    </TD>
                  </TR>
                ))}
              </TBody>
            </Table>
            </div>
            <Pagination
              page={page}
              totalPages={totalPages}
              totalItems={totalItems}
              pageSize={pageSize}
              onPageChange={setPage}
              className="-mx-5 -mb-4 mt-1"
            />
            </>
          )}
        </CardContent>
      </Card>

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

      {deletingPayment && (
        <Modal
          open={Boolean(deletingPayment)}
          onClose={() => setDeletingPayment(null)}
          title="Hapus Pembayaran Client"
          description="Tindakan ini permanen dan tidak dapat dibatalkan."
          footer={
            <>
              <Button variant="secondary" onClick={() => setDeletingPayment(null)} disabled={deleting}>Batal</Button>
              <Button variant="danger" onClick={() => void handleDeletePayment()} disabled={deleting}>
                {deleting ? "Menghapus..." : "Ya, Hapus Permanen"}
              </Button>
            </>
          }
        >
          <div className="flex flex-col gap-3">
            {deleteError && (
              <p className="rounded-md border border-danger/30 bg-danger-soft px-3.5 py-2.5 text-[13px] font-medium text-danger-text">{deleteError}</p>
            )}
            <p className="flex items-start gap-2 text-[13.5px] text-text-primary">
              <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0 text-danger-text" />
              <span>
                Yakin ingin menghapus pembayaran client <strong>{formatCurrency(deletingPayment.amount)}</strong> ({deletingPayment.type})
                ini secara permanen?
                {(() => {
                  const count = evidence.filter((e) => e.relatedKind === "clientPayment" && e.relatedId === deletingPayment.id).length;
                  return count > 0
                    ? ` ${count} berkas bukti yang terlampir pada pembayaran ini akan ikut terhapus.`
                    : "";
                })()}
              </span>
            </p>
          </div>
        </Modal>
      )}

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
        <EvidenceListModal
          open
          projectId={projectId}
          title={`Bukti — ${viewingEvidencePayment.type}`}
          items={evidence.filter((e) => e.relatedKind === "clientPayment" && e.relatedId === viewingEvidencePayment.id)}
          onClose={() => setViewingEvidencePayment(null)}
        />
      )}
    </div>
  );
}

/**
 * The proof state of one payment: a stamp when verified (opens the file), else
 * whether a file is attached and, for roles that may, the button to verify it.
 */
function PaymentProofCell({ payment, canVerify, onOpen, onVerify }: { payment: ClientPayment; canVerify: boolean; onOpen: () => void; onVerify: () => void }) {
  const proof = payment.evidence[0];
  if (payment.verifiedAt && proof) {
    return <ProofStamp proof={proof} label={verifiedLabel(payment.verifiedAt, payment.verifiedBy)} onOpen={onOpen} />;
  }
  if (!proof) return <Badge tone="warning">Belum ada bukti</Badge>;
  return (
    <span className="flex flex-wrap items-center gap-2">
      <Badge tone="neutral">Bukti ada</Badge>
      {canVerify && (
        <Button size="sm" variant="secondary" onClick={onVerify}>
          Verifikasi
        </Button>
      )}
    </span>
  );
}

function SummaryStat({ label, value, badge }: { label: string; value: string; badge?: string }) {
  return (
    <div className="rounded-md border border-border bg-surface-muted/40 px-4 py-3">
      <p className="text-[11.5px] font-medium uppercase tracking-wide text-text-secondary">{label}</p>
      <p className="mt-1 flex flex-wrap items-center gap-2 text-[15px] font-bold tabular-nums text-navy-900">
        {value}
        {badge && <Badge tone="warning">{badge}</Badge>}
      </p>
    </div>
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
      setError(getApiErrorMessage(err, "Gagal memperbarui pembayaran client"));
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <Modal
      open={open}
      onClose={onClose}
      title="Ubah Pembayaran Client"
      description="Perbarui detail pembayaran ini. Berkas evidence yang sudah terlampir tidak berubah."
      busy={submitting}
      error={error}
      footer={
        <>
          <Button variant="secondary" onClick={onClose} disabled={submitting}>Batal</Button>
          <Button onClick={() => void handleSubmit()} disabled={submitting}>
            {submitting ? "Menyimpan..." : "Simpan Perubahan"}
          </Button>
        </>
      }
    >
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <Field label="Jenis Pembayaran" required>
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
        <Field label="No. Referensi" hint={errors.referenceNumber}>
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
