import { useEffect, useState } from "react";
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
import { Pagination } from "@/shared/components/ui/Pagination";
import { usePagination } from "@/shared/hooks/usePagination";
import { useProjectStore, ClientPaymentEvidenceError } from "@/modules/projects/stores/useProjectStore";
import { useAuthStore } from "@/shared/stores/useAuthStore";
import {
  clientPaymentSchema,
  clientPaymentUpdateSchema,
  CLIENT_PAYMENT_TYPE_OPTIONS,
  type ClientPaymentFormValues,
  type ClientPaymentUpdateFormValues,
} from "@/modules/projects/schemas/client-payment.schema";
import { PAYMENT_METHOD_OPTIONS } from "@/modules/projects/schemas/payment.schema";
import type { ClientPayment } from "@/modules/projects/types";
import { todayISO } from "@/modules/projects/lib/dates";
import { getApiErrorMessage } from "@/shared/lib/api-error";
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
  // Hard delete is Owner-or-Admin (broadened from Owner-only per explicit
  // user request) -- mirrors ProjectHeaderCard.tsx's own "!== \"Staff\"" idiom
  // for the same Owner-or-Admin bar, since those are the only 3 roles. Edit
  // has no role restriction at all.
  const canDelete = useAuthStore((s) => s.session?.role) !== "Staff";

  const [modalOpen, setModalOpen] = useState(false);
  const [actionError, setActionError] = useState<string | null>(null);
  const [editingPayment, setEditingPayment] = useState<ClientPayment | null>(null);
  const [deletingPayment, setDeletingPayment] = useState<ClientPayment | null>(null);
  const [deleteError, setDeleteError] = useState<string | null>(null);
  const [deleting, setDeleting] = useState(false);
  const [viewingEvidencePayment, setViewingEvidencePayment] = useState<ClientPayment | null>(null);

  useEffect(() => {
    void fetchClientPayments(projectId);
    void fetchEvidence(projectId);
  }, [projectId, fetchClientPayments, fetchEvidence]);

  const contractValue = project?.contractValue ?? 0;
  const totalReceived = clientPayments.reduce(
    (sum, p) => (p.type === "Refund" ? sum - p.amount : sum + p.amount),
    0
  );
  const totalRemaining = contractValue - totalReceived;
  const { page, setPage, totalPages, totalItems, pageSize, pageItems } = usePagination(clientPayments);

  async function handleAddPayment(values: ClientPaymentFormValues) {
    setActionError(null);
    try {
      await createClientPayment(projectId, values);
      setModalOpen(false);
    } catch (err) {
      if (err instanceof ClientPaymentEvidenceError) {
        // The payment itself was saved — close the modal so the user isn't
        // invited to resubmit (which would create a duplicate payment), just
        // surface that the proof didn't make it.
        setActionError(err.message);
        setModalOpen(false);
        return;
      }
      setActionError(getApiErrorMessage(err, "Gagal mencatat pembayaran client"));
      throw err;
    }
  }

  async function handleEditPayment(values: ClientPaymentUpdateFormValues) {
    if (!editingPayment) return;
    await updateClientPayment(projectId, editingPayment.id, values);
    setEditingPayment(null);
  }

  async function handleDeletePayment() {
    if (!deletingPayment) return;
    setDeleteError(null);
    setDeleting(true);
    try {
      await deleteClientPayment(projectId, deletingPayment.id);
      setDeletingPayment(null);
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
            <Button size="sm" icon={<Plus className="h-3.5 w-3.5" />} onClick={() => setModalOpen(true)}>
              Tambah Pembayaran
            </Button>
          }
        />
        <CardContent className="flex flex-col gap-5">
          {actionError && (
            <p className="rounded-md border border-danger/30 bg-danger-soft px-3.5 py-2.5 text-[13px] font-medium text-danger">{actionError}</p>
          )}
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-3">
            <SummaryStat label="Nilai Kontrak" value={formatCurrency(contractValue)} />
            <SummaryStat label="Total Diterima" value={formatCurrency(totalReceived)} />
            <SummaryStat label="Sisa Tagihan" value={formatCurrency(totalRemaining)} />
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
                    {payment.evidenceComplete ? <Badge tone="success">Ada</Badge> : <Badge tone="warning">Belum Ada</Badge>}
                  </div>
                  <div className="flex flex-col gap-1.5">
                    <CardListField label="Nominal" value={formatCurrency(payment.amount)} />
                    <CardListField label="Tanggal" value={formatDate(payment.paymentDate)} />
                    <CardListField label="Metode" value={payment.method} />
                    <CardListField label="No. Referensi" value={payment.referenceNumber} />
                  </div>
                  <div className="flex items-center gap-2 pt-1">
                    <IconActionButton icon={Eye} label="Lihat Bukti" tone="info" onClick={() => setViewingEvidencePayment(payment)} />
                    <IconActionButton icon={Pencil} label="Ubah Pembayaran" tone="neutral" onClick={() => setEditingPayment(payment)} />
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
                    <TD>{payment.evidenceComplete ? <Badge tone="success">Ada</Badge> : <Badge tone="warning">Belum Ada</Badge>}</TD>
                    <TD>
                      <div className="flex items-center gap-2">
                        <IconActionButton icon={Eye} label="Lihat Bukti" tone="info" onClick={() => setViewingEvidencePayment(payment)} />
                        <IconActionButton icon={Pencil} label="Ubah Pembayaran" tone="neutral" onClick={() => setEditingPayment(payment)} />
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

      {modalOpen && (
        <AddClientPaymentModal
          open={modalOpen}
          onClose={() => { setModalOpen(false); setActionError(null); }}
          onSubmit={handleAddPayment}
          error={actionError}
        />
      )}

      {editingPayment && (
        <EditClientPaymentModal
          open={Boolean(editingPayment)}
          payment={editingPayment}
          onClose={() => setEditingPayment(null)}
          onSubmit={handleEditPayment}
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
              <p className="rounded-md border border-danger/30 bg-danger-soft px-3.5 py-2.5 text-[13px] font-medium text-danger">{deleteError}</p>
            )}
            <p className="flex items-start gap-2 text-[13.5px] text-text-primary">
              <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0 text-danger" />
              <span>
                Yakin ingin menghapus pembayaran client <strong>{formatCurrency(deletingPayment.amount)}</strong> ({deletingPayment.type})
                ini secara permanen?
                {(() => {
                  const count = evidence.filter((e) => e.relatedKind === "clientPayment" && e.relatedId === deletingPayment.id).length;
                  return count > 0
                    ? ` ${count} berkas evidence yang terlampir pada pembayaran ini akan ikut terhapus.`
                    : "";
                })()}
              </span>
            </p>
          </div>
        </Modal>
      )}

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

function SummaryStat({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-md border border-border bg-surface-muted/40 px-4 py-3">
      <p className="text-[11.5px] font-medium uppercase tracking-wide text-text-secondary">{label}</p>
      <p className="mt-1 text-[15px] font-bold tabular-nums text-navy-900">{value}</p>
    </div>
  );
}

function emptyValues(): ClientPaymentFormValues {
  return {
    type: "DP",
    amount: 0,
    paymentDate: todayISO(),
    method: "Transfer Bank",
    referenceNumber: "",
    notes: "",
    proofFile: undefined,
  };
}

function AddClientPaymentModal({
  open,
  onClose,
  onSubmit,
  error,
}: {
  open: boolean;
  onClose: () => void;
  onSubmit: (values: ClientPaymentFormValues) => Promise<void>;
  error: string | null;
}) {
  const [values, setValues] = useState<ClientPaymentFormValues>(emptyValues);
  const [errors, setErrors] = useState<Partial<Record<keyof ClientPaymentFormValues, string>>>({});
  const [submitting, setSubmitting] = useState(false);

  function set<K extends keyof ClientPaymentFormValues>(key: K, value: ClientPaymentFormValues[K]) {
    setValues((prev) => ({ ...prev, [key]: value }));
  }

  async function handleSubmit() {
    const result = clientPaymentSchema.safeParse(values);
    if (!result.success) {
      const fieldErrors: Partial<Record<keyof ClientPaymentFormValues, string>> = {};
      for (const issue of result.error.issues) {
        fieldErrors[issue.path[0] as keyof ClientPaymentFormValues] = issue.message;
      }
      setErrors(fieldErrors);
      return;
    }
    setSubmitting(true);
    try {
      await onSubmit(result.data);
      // Only reset on a resolved submit — on failure the modal stays open
      // (parent doesn't close it), so the form must keep the user's input
      // rather than silently wiping it out from under them.
      setValues(emptyValues());
      setErrors({});
    } catch {
      // The `error` prop (parent's actionError, passed back down) surfaces
      // this inside the modal itself -- the modal stays open here, and its
      // own fixed-overlay backdrop would otherwise hide a banner rendered
      // only in the parent's now-obscured page content.
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <Modal
      open={open}
      onClose={onClose}
      title="Tambah Pembayaran Client"
      description="Catat pembayaran baru dari client untuk project ini, beserta bukti transfernya jika ada."
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>Batal</Button>
          <Button onClick={() => void handleSubmit()} disabled={submitting}>
            {submitting ? "Menyimpan..." : "Simpan Pembayaran"}
          </Button>
        </>
      }
    >
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        {error && (
          <p className="sm:col-span-2 rounded-md border border-danger/30 bg-danger-soft px-3.5 py-2.5 text-[13px] font-medium text-danger">{error}</p>
        )}
        <Field label="Jenis Pembayaran" required>
          <Select value={values.type} onChange={(e) => set("type", e.target.value as ClientPaymentFormValues["type"])}>
            {CLIENT_PAYMENT_TYPE_OPTIONS.map((t) => (
              <option key={t} value={t}>{t}</option>
            ))}
          </Select>
        </Field>
        <Field label="Nominal (Rp)" required hint={errors.amount}>
          <CurrencyInput value={values.amount} onChange={(n) => set("amount", n)} />
        </Field>
        <Field label="Tanggal" required hint={errors.paymentDate}>
          <Input type="date" value={values.paymentDate} onChange={(e) => set("paymentDate", e.target.value)} />
        </Field>
        <Field label="Metode" required hint={errors.method}>
          <Select value={values.method} onChange={(e) => set("method", e.target.value as ClientPaymentFormValues["method"])}>
            {PAYMENT_METHOD_OPTIONS.map((m) => (
              <option key={m} value={m}>{m}</option>
            ))}
          </Select>
        </Field>
        <Field label="No. Referensi" hint={errors.referenceNumber}>
          <Input value={values.referenceNumber} onChange={(e) => set("referenceNumber", e.target.value)} />
        </Field>
        <Field label="Bukti Transfer">
          <input
            type="file"
            accept="image/jpeg,image/png,application/pdf"
            onChange={(e) => set("proofFile", e.target.files?.[0] ?? undefined)}
            className="block w-full text-[13px] text-text-secondary file:mr-3 file:rounded-md file:border-0 file:bg-navy-900 file:px-3 file:py-1.5 file:text-[12.5px] file:font-semibold file:text-white"
          />
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

// Edit scope is the ledger fields only -- no file input (already covered by
// the existing generic evidence tab), see PLAN.md §2.5.
function EditClientPaymentModal({
  open,
  payment,
  onClose,
  onSubmit,
}: {
  open: boolean;
  payment: ClientPayment;
  onClose: () => void;
  onSubmit: (values: ClientPaymentUpdateFormValues) => Promise<void>;
}) {
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
    if (!result.success) {
      const fieldErrors: Partial<Record<keyof ClientPaymentUpdateFormValues, string>> = {};
      for (const issue of result.error.issues) {
        fieldErrors[issue.path[0] as keyof ClientPaymentUpdateFormValues] = issue.message;
      }
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
          <p className="sm:col-span-2 rounded-md border border-danger/30 bg-danger-soft px-3.5 py-2.5 text-[13px] font-medium text-danger">{error}</p>
        )}
        <Field label="Jenis Pembayaran" required>
          <Select value={values.type} onChange={(e) => set("type", e.target.value as ClientPaymentUpdateFormValues["type"])}>
            {CLIENT_PAYMENT_TYPE_OPTIONS.map((t) => (
              <option key={t} value={t}>{t}</option>
            ))}
          </Select>
        </Field>
        <Field label="Nominal (Rp)" required hint={errors.amount}>
          <CurrencyInput value={values.amount} onChange={(n) => set("amount", n)} />
        </Field>
        <Field label="Tanggal" required hint={errors.paymentDate}>
          <Input type="date" value={values.paymentDate} onChange={(e) => set("paymentDate", e.target.value)} />
        </Field>
        <Field label="Metode" required hint={errors.method}>
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
