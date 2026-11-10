import { useEffect, useState } from "react";
import { useCan } from "@/shared/stores/usePermissionStore";
import { Plus, Pencil, Trash2, Eye } from "lucide-react";
import { Card, CardHeader, CardContent } from "@/shared/components/ui/Card";
import { Badge } from "@/shared/components/ui/Badge";
import { Button } from "@/shared/components/ui/Button";
import { RecordPaymentDrawer } from "@/modules/projects/components/detail/RecordPaymentDrawer";
import type { PaymentFormValues } from "@/modules/projects/schemas/payment.schema";
import { deleteWithUndo } from "@/shared/lib/undo-delete";
import { Modal } from "@/shared/components/ui/Modal";
import { Input, Textarea, Select, Field } from "@/shared/components/ui/Input";
import { CurrencyInput } from "@/shared/components/ui/CurrencyInput";
import { Table, THead, TBody, TR, TH, TD } from "@/shared/components/ui/Table";
import { CardList, CardListField } from "@/shared/components/ui/CardList";
import { IconActionButton } from "@/shared/components/ui/IconActionButton";
import { EvidenceListModal } from "@/shared/components/ui/EvidenceListModal";
import { Pagination } from "@/shared/components/ui/Pagination";
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
export function VenuePaymentsSection({ projectId }: { projectId: string }) {
  const project = useProjectStore((s) => s.currentProject);
  const venuePayments = useProjectStore((s) => s.venuePayments);
  const evidence = useProjectStore((s) => s.evidence);
  const fetchVenuePayments = useProjectStore((s) => s.fetchVenuePayments);
  const fetchEvidence = useProjectStore((s) => s.fetchEvidence);
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
    void fetchEvidence(projectId);
  }, [projectId, fetchVenuePayments, fetchEvidence]);

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
      <Card>
        <CardHeader title="Pembayaran ke venue" subtitle="Ringkasan nilai sewa dan riwayat pembayaran ke venue pada proyek ini." />
        <CardContent>
          <p className="rounded-md border border-dashed border-border px-4 py-6 text-center text-[13px] text-text-secondary">
            Belum ada venue terpasang untuk proyek ini. Pasang venue terlebih dahulu di tab Venue.
          </p>
        </CardContent>
      </Card>
    );
  }

  return (
    <div>
      <Card>
        <CardHeader
          title="Pembayaran ke venue"
          subtitle="Ringkasan nilai sewa dan seluruh riwayat pembayaran ke venue pada proyek ini."
          action={
            hasVenue && canRecord ? (
              <Button size="sm" icon={<Plus className="h-3.5 w-3.5" />} onClick={() => setModalOpen(true)}>
                Tambah pembayaran
              </Button>
            ) : undefined
          }
        />
        <CardContent className="flex flex-col gap-5">
          {!hasVenue && (
            <p className="rounded-md border border-dashed border-border px-3.5 py-2.5 text-[13px] text-text-secondary">
              Venue sudah dilepas dari proyek ini. Riwayat pembayaran berikut tetap ditampilkan sebagai catatan historis.
            </p>
          )}
          <div className="grid grid-cols-1 gap-3 xl:grid-cols-3">
            <SummaryStat label="Nilai sewa venue" value={formatCurrency(venueValue)} />
            <SummaryStat label="Total sudah dibayar" value={formatCurrency(totalPaid)} />
            <SummaryStat label="Sisa pembayaran" value={totalRemaining < 0 ? `Lebih bayar ${formatCurrency(-totalRemaining)}` : formatCurrency(totalRemaining)} />
          </div>

          {venuePayments.length === 0 ? (
            <p className="rounded-md border border-dashed border-border px-4 py-6 text-center text-[13px] text-text-secondary">
              Belum ada pembayaran venue tercatat untuk proyek ini.
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
                    {payment.evidenceComplete ? <Badge tone="success">Lengkap</Badge> : <Badge tone="warning">Belum lengkap</Badge>}
                  </div>
                  <div className="flex flex-col gap-1.5">
                    <CardListField label="Nominal" value={formatCurrency(payment.amount)} />
                    <CardListField label="Tanggal" value={formatDate(payment.paymentDate)} />
                    <CardListField label="Metode" value={payment.method} />
                    <CardListField label="No. referensi" value={payment.referenceNumber} />
                  </div>
                  <div className="flex items-center gap-2 pt-1">
                    <IconActionButton icon={Eye} readOnly label="Lihat bukti" tone="info" onClick={() => setViewingEvidencePayment(payment)} />
                    {canRecord && <IconActionButton icon={Pencil} label="Ubah pembayaran" tone="neutral" onClick={() => setEditingPayment(payment)} />}
                    {canDelete && (
                      <IconActionButton icon={Trash2} label="Hapus pembayaran" tone="danger" onClick={() => handleDeletePayment(payment)} />
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
                  <TH>Kelengkapan bukti</TH>
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
                    <TD>{payment.evidenceComplete ? <Badge tone="success">Lengkap</Badge> : <Badge tone="warning">Belum lengkap</Badge>}</TD>
                    <TD>
                      <div className="flex items-center gap-2">
                        <IconActionButton icon={Eye} readOnly label="Lihat bukti" tone="info" onClick={() => setViewingEvidencePayment(payment)} />
                        {canRecord && <IconActionButton icon={Pencil} label="Ubah pembayaran" tone="neutral" onClick={() => setEditingPayment(payment)} />}
                        {canDelete && (
                          <IconActionButton icon={Trash2} label="Hapus pembayaran" tone="danger" onClick={() => handleDeletePayment(payment)} />
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
        <EvidenceListModal
          open
          projectId={projectId}
          title={`Bukti — ${viewingEvidencePayment.type}`}
          items={evidence.filter((e) => e.relatedKind === "venuePayment" && e.relatedId === viewingEvidencePayment.id)}
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
