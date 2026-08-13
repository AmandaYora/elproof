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
import { useProjectStore, VendorPaymentEvidenceError } from "@/modules/projects/stores/useProjectStore";
import { useVendorStore } from "@/modules/vendors/stores/useVendorStore";
import { useAuthStore } from "@/shared/stores/useAuthStore";
import {
  paymentSchema,
  paymentUpdateSchema,
  PAYMENT_TYPE_OPTIONS,
  PAYMENT_METHOD_OPTIONS,
  type PaymentFormValues,
  type PaymentUpdateFormValues,
} from "@/modules/projects/schemas/payment.schema";
import type { ProjectVendor, VendorPayment } from "@/modules/projects/types";
import { todayISO } from "@/modules/projects/lib/dates";
import { getApiErrorMessage } from "@/shared/lib/api-error";
import { formatCurrency, formatDate } from "@/shared/lib/formatters";

export function ProjectPaymentsSection({ projectId }: { projectId: string }) {
  const payments = useProjectStore((s) => s.payments);
  const vendorEngagements = useProjectStore((s) => s.vendorEngagements);
  const evidence = useProjectStore((s) => s.evidence);
  const fetchPayments = useProjectStore((s) => s.fetchPayments);
  const fetchVendorSection = useProjectStore((s) => s.fetchVendorSection);
  const fetchEvidence = useProjectStore((s) => s.fetchEvidence);
  const createPayment = useProjectStore((s) => s.createPayment);
  const updatePayment = useProjectStore((s) => s.updatePayment);
  const deletePayment = useProjectStore((s) => s.deletePayment);
  const vendors = useVendorStore((s) => s.vendors);
  const fetchVendors = useVendorStore((s) => s.fetchVendors);
  // Hard delete is Owner-or-Admin (broadened from Owner-only per explicit
  // user request) -- mirrors ProjectHeaderCard.tsx's own "!== \"Staff\"" idiom
  // for the same Owner-or-Admin bar, since those are the only 3 roles. Edit
  // has no role restriction at all.
  const canDelete = useAuthStore((s) => s.session?.role) !== "Staff";

  const [modalOpen, setModalOpen] = useState(false);
  const [actionError, setActionError] = useState<string | null>(null);
  const [editingPayment, setEditingPayment] = useState<VendorPayment | null>(null);
  const [deletingPayment, setDeletingPayment] = useState<VendorPayment | null>(null);
  const [deleteError, setDeleteError] = useState<string | null>(null);
  const [deleting, setDeleting] = useState(false);
  const [viewingEvidencePayment, setViewingEvidencePayment] = useState<VendorPayment | null>(null);

  useEffect(() => {
    void fetchPayments(projectId);
    void fetchVendorSection(projectId);
    void fetchEvidence(projectId);
    void fetchVendors();
  }, [projectId, fetchPayments, fetchVendorSection, fetchEvidence, fetchVendors]);

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

  async function handleAddPayment(values: PaymentFormValues) {
    setActionError(null);
    try {
      await createPayment(projectId, values);
      setModalOpen(false);
    } catch (err) {
      if (err instanceof VendorPaymentEvidenceError) {
        // The payment itself was saved — close the modal so the user isn't
        // invited to resubmit (which would create a duplicate payment), just
        // surface that one or both evidence slots didn't make it.
        setActionError(err.message);
        setModalOpen(false);
        return;
      }
      setActionError(getApiErrorMessage(err, "Gagal mencatat pembayaran"));
      throw err;
    }
  }

  async function handleEditPayment(values: PaymentUpdateFormValues) {
    if (!editingPayment) return;
    await updatePayment(projectId, editingPayment.id, values);
    setEditingPayment(null);
  }

  async function handleDeletePayment() {
    if (!deletingPayment) return;
    setDeleteError(null);
    setDeleting(true);
    try {
      await deletePayment(projectId, deletingPayment.id);
      setDeletingPayment(null);
    } catch (err) {
      setDeleteError(getApiErrorMessage(err, "Gagal menghapus pembayaran"));
    } finally {
      setDeleting(false);
    }
  }

  return (
    <div id="pembayaran">
      <Card>
        <CardHeader
          title="Pembayaran ke Vendor"
          subtitle="Ringkasan nilai kerja sama dan seluruh riwayat pembayaran ke vendor pada project ini."
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
            <SummaryStat label="Total Nilai Kerja Sama" value={formatCurrency(totalContractValue)} />
            <SummaryStat label="Total Sudah Dibayar" value={formatCurrency(totalPaid)} />
            <SummaryStat label="Sisa Pembayaran" value={formatCurrency(totalRemaining)} />
          </div>

          {payments.length === 0 ? (
            <p className="rounded-md border border-dashed border-border px-4 py-6 text-center text-[13px] text-text-secondary">
              Belum ada pembayaran tercatat untuk project ini.
            </p>
          ) : (
            <>
            <CardList
              className="sm:hidden"
              items={pageItems}
              keyFor={(payment) => payment.id}
              renderItem={(payment) => {
                const pv = vendorEngagements.find((v) => v.id === payment.projectVendorId);
                const vendor = pv ? vendors.find((v) => v.id === pv.vendorId) : null;
                return (
                  <>
                    <div className="flex items-start justify-between gap-3">
                      <span className="font-medium text-text-primary">{vendor?.name ?? "Vendor tidak diketahui"}</span>
                      {payment.evidenceComplete ? <Badge tone="success">Lengkap</Badge> : <Badge tone="warning">Belum Lengkap</Badge>}
                    </div>
                    <div className="flex flex-col gap-1.5">
                      <CardListField label="Jenis" value={payment.type} />
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
                );
              }}
            />
            <div className="hidden sm:block">
            <Table>
              <THead>
                <TR>
                  <TH>Vendor</TH>
                  <TH>Jenis</TH>
                  <TH className="text-right">Nominal</TH>
                  <TH>Tanggal</TH>
                  <TH>Metode</TH>
                  <TH>No. Referensi</TH>
                  <TH>Kelengkapan Evidence</TH>
                  <TH>Aksi</TH>
                </TR>
              </THead>
              <TBody>
                {pageItems.map((payment) => {
                  const pv = vendorEngagements.find((v) => v.id === payment.projectVendorId);
                  const vendor = pv ? vendors.find((v) => v.id === pv.vendorId) : null;
                  return (
                    <TR key={payment.id}>
                      <TD className="font-medium">{vendor?.name ?? "Vendor tidak diketahui"}</TD>
                      <TD>{payment.type}</TD>
                      <TD className="text-right tabular-nums">{formatCurrency(payment.amount)}</TD>
                      <TD>{formatDate(payment.paymentDate)}</TD>
                      <TD>{payment.method}</TD>
                      <TD>{payment.referenceNumber}</TD>
                      <TD>
                        {payment.evidenceComplete ? <Badge tone="success">Lengkap</Badge> : <Badge tone="warning">Belum Lengkap</Badge>}
                      </TD>
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
                  );
                })}
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
        <AddPaymentModal
          open={modalOpen}
          onClose={() => { setModalOpen(false); setActionError(null); }}
          onSubmit={handleAddPayment}
          error={actionError}
          vendorEngagements={vendorEngagements}
          vendors={vendors}
        />
      )}

      {editingPayment && (
        <EditPaymentModal
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
          title="Hapus Pembayaran"
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
                Yakin ingin menghapus pembayaran <strong>{formatCurrency(deletingPayment.amount)}</strong> ({deletingPayment.type}) ini
                secara permanen?
                {(() => {
                  const count = evidence.filter((e) => e.relatedKind === "payment" && e.relatedId === deletingPayment.id).length;
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
          items={evidence.filter((e) => e.relatedKind === "payment" && e.relatedId === viewingEvidencePayment.id)}
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

function toFormValues(defaultProjectVendorId: string): PaymentFormValues {
  return {
    projectVendorId: defaultProjectVendorId,
    type: "DP",
    amount: 0,
    paymentDate: todayISO(),
    method: "Transfer Bank",
    referenceNumber: "",
    notes: "",
    invoiceFile: undefined,
    proofFile: undefined,
  };
}

function AddPaymentModal({
  open,
  onClose,
  onSubmit,
  error,
  vendorEngagements,
  vendors,
}: {
  open: boolean;
  onClose: () => void;
  onSubmit: (values: PaymentFormValues) => Promise<void>;
  error: string | null;
  vendorEngagements: ProjectVendor[];
  vendors: { id: string; name: string; city: string | null }[];
}) {
  const defaultVendorId = vendorEngagements[0]?.id ?? "";
  const [values, setValues] = useState<PaymentFormValues>(() => toFormValues(defaultVendorId));
  const [errors, setErrors] = useState<Partial<Record<keyof PaymentFormValues, string>>>({});
  const [submitting, setSubmitting] = useState(false);

  function set<K extends keyof PaymentFormValues>(key: K, value: PaymentFormValues[K]) {
    setValues((prev) => ({ ...prev, [key]: value }));
  }

  async function handleSubmit() {
    const result = paymentSchema.safeParse(values);
    if (!result.success) {
      const fieldErrors: Partial<Record<keyof PaymentFormValues, string>> = {};
      for (const issue of result.error.issues) {
        fieldErrors[issue.path[0] as keyof PaymentFormValues] = issue.message;
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
      setValues(toFormValues(defaultVendorId));
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
      title="Tambah Pembayaran"
      description="Catat pembayaran baru ke vendor untuk project ini."
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
        <Field label="Vendor" required hint={errors.projectVendorId}>
          <Select value={values.projectVendorId} onChange={(e) => set("projectVendorId", e.target.value)}>
            {vendorEngagements.map((pv) => {
              const vendor = vendors.find((v) => v.id === pv.vendorId);
              const label = vendor ? (vendor.city ? `${vendor.name} — ${vendor.city}` : vendor.name) : "Vendor tidak diketahui";
              return (
                <option key={pv.id} value={pv.id}>{label}</option>
              );
            })}
          </Select>
        </Field>
        <Field label="Jenis Pembayaran" required>
          <Select value={values.type} onChange={(e) => set("type", e.target.value as PaymentFormValues["type"])}>
            {PAYMENT_TYPE_OPTIONS.map((t) => (
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
          <Select value={values.method} onChange={(e) => set("method", e.target.value as PaymentFormValues["method"])}>
            {PAYMENT_METHOD_OPTIONS.map((m) => (
              <option key={m} value={m}>{m}</option>
            ))}
          </Select>
        </Field>
        <Field label="No. Referensi" hint={errors.referenceNumber}>
          <Input value={values.referenceNumber} onChange={(e) => set("referenceNumber", e.target.value)} />
        </Field>
        <Field label="Invoice">
          <input
            type="file"
            accept="image/jpeg,image/png,application/pdf"
            onChange={(e) => set("invoiceFile", e.target.files?.[0] ?? undefined)}
            className="block w-full text-[13px] text-text-secondary file:mr-3 file:rounded-md file:border-0 file:bg-navy-900 file:px-3 file:py-1.5 file:text-[12.5px] file:font-semibold file:text-white"
          />
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

// Edit scope is the ledger fields only -- no vendor picker (re-parenting a
// payment isn't supported, PLAN.md §2.5) and no file inputs (already
// covered by the existing generic evidence tab).
function EditPaymentModal({
  open,
  payment,
  onClose,
  onSubmit,
}: {
  open: boolean;
  payment: VendorPayment;
  onClose: () => void;
  onSubmit: (values: PaymentUpdateFormValues) => Promise<void>;
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
    if (!result.success) {
      const fieldErrors: Partial<Record<keyof PaymentUpdateFormValues, string>> = {};
      for (const issue of result.error.issues) {
        fieldErrors[issue.path[0] as keyof PaymentUpdateFormValues] = issue.message;
      }
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
      title="Ubah Pembayaran"
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
          <Select value={values.type} onChange={(e) => set("type", e.target.value as PaymentUpdateFormValues["type"])}>
            {PAYMENT_TYPE_OPTIONS.map((t) => (
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
          <Select value={values.method} onChange={(e) => set("method", e.target.value as PaymentUpdateFormValues["method"])}>
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
