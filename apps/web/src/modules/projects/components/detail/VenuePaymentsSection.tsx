import { useEffect, useState } from "react";
import { Plus } from "lucide-react";
import { Card, CardHeader, CardContent } from "@/shared/components/ui/Card";
import { Badge } from "@/shared/components/ui/Badge";
import { Button } from "@/shared/components/ui/Button";
import { Modal } from "@/shared/components/ui/Modal";
import { Input, Textarea, Select, Field } from "@/shared/components/ui/Input";
import { Table, THead, TBody, TR, TH, TD } from "@/shared/components/ui/Table";
import { CardList, CardListField } from "@/shared/components/ui/CardList";
import { Pagination } from "@/shared/components/ui/Pagination";
import { usePagination } from "@/shared/hooks/usePagination";
import { useProjectStore, VenuePaymentEvidenceError } from "@/modules/projects/stores/useProjectStore";
import {
  venuePaymentSchema,
  VENUE_PAYMENT_TYPE_OPTIONS,
  type VenuePaymentFormValues,
} from "@/modules/projects/schemas/venue-payment.schema";
import { PAYMENT_METHOD_OPTIONS } from "@/modules/projects/schemas/payment.schema";
import { todayISO } from "@/modules/projects/lib/dates";
import { getApiErrorMessage } from "@/shared/lib/api-error";
import { formatCurrency, formatDate } from "@/shared/lib/formatters";

// Sibling of ProjectPaymentsSection (Pembayaran ke Vendor) minus the vendor
// picker -- a project has at most one venue (project.venueId), nothing to
// choose. See PLAN.md "Venue Payments + Pembayaran tab restructuring".
export function VenuePaymentsSection({ projectId }: { projectId: string }) {
  const project = useProjectStore((s) => s.currentProject);
  const venuePayments = useProjectStore((s) => s.venuePayments);
  const fetchVenuePayments = useProjectStore((s) => s.fetchVenuePayments);
  const createVenuePayment = useProjectStore((s) => s.createVenuePayment);

  const [modalOpen, setModalOpen] = useState(false);
  const [actionError, setActionError] = useState<string | null>(null);

  useEffect(() => {
    void fetchVenuePayments(projectId);
  }, [projectId, fetchVenuePayments]);

  const hasVenue = Boolean(project?.venueId);
  const venueValue = (project?.venueRentalPrice ?? 0) + (project?.venueCharge ?? 0);
  const totalPaid = venuePayments.reduce(
    (sum, p) => (p.type === "Refund" ? sum - p.amount : sum + p.amount),
    0
  );
  const totalRemaining = venueValue - totalPaid;
  const { page, setPage, totalPages, totalItems, pageSize, pageItems } = usePagination(venuePayments);

  async function handleAddPayment(values: VenuePaymentFormValues) {
    setActionError(null);
    try {
      await createVenuePayment(projectId, values);
      setModalOpen(false);
    } catch (err) {
      if (err instanceof VenuePaymentEvidenceError) {
        setActionError(err.message);
        setModalOpen(false);
        return;
      }
      setActionError(getApiErrorMessage(err, "Gagal mencatat pembayaran venue"));
      throw err;
    }
  }

  // A venue can be detached later (Venue tab's "Lepas") after payments were
  // already recorded against it -- those rows must stay visible as a
  // historical fact, the same principle ProjectPaymentsSection's own
  // payment table already follows for a Cancelled vendor engagement. So the
  // "nothing to show" empty state only applies when there's truly nothing:
  // no venue attached AND no payment history at all.
  if (!hasVenue && venuePayments.length === 0) {
    return (
      <Card>
        <CardHeader title="Pembayaran ke Venue" subtitle="Ringkasan nilai sewa dan riwayat pembayaran ke venue pada project ini." />
        <CardContent>
          <p className="rounded-md border border-dashed border-border px-4 py-6 text-center text-[13px] text-text-secondary">
            Belum ada venue terpasang untuk project ini. Pasang venue terlebih dahulu di tab Venue.
          </p>
        </CardContent>
      </Card>
    );
  }

  return (
    <div>
      <Card>
        <CardHeader
          title="Pembayaran ke Venue"
          subtitle="Ringkasan nilai sewa dan seluruh riwayat pembayaran ke venue pada project ini."
          action={
            hasVenue ? (
              <Button size="sm" icon={<Plus className="h-3.5 w-3.5" />} onClick={() => setModalOpen(true)}>
                Tambah Pembayaran
              </Button>
            ) : undefined
          }
        />
        <CardContent className="flex flex-col gap-5">
          {!hasVenue && (
            <p className="rounded-md border border-dashed border-border px-3.5 py-2.5 text-[13px] text-text-secondary">
              Venue sudah dilepas dari project ini. Riwayat pembayaran berikut tetap ditampilkan sebagai catatan historis.
            </p>
          )}
          {actionError && (
            <p className="rounded-md border border-danger/30 bg-danger-soft px-3.5 py-2.5 text-[13px] font-medium text-danger">{actionError}</p>
          )}
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-3">
            <SummaryStat label="Nilai Sewa Venue" value={formatCurrency(venueValue)} />
            <SummaryStat label="Total Sudah Dibayar" value={formatCurrency(totalPaid)} />
            <SummaryStat label="Sisa Pembayaran" value={formatCurrency(totalRemaining)} />
          </div>

          {venuePayments.length === 0 ? (
            <p className="rounded-md border border-dashed border-border px-4 py-6 text-center text-[13px] text-text-secondary">
              Belum ada pembayaran venue tercatat untuk project ini.
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
                    {payment.evidenceComplete ? <Badge tone="success">Lengkap</Badge> : <Badge tone="warning">Belum Lengkap</Badge>}
                  </div>
                  <div className="flex flex-col gap-1.5">
                    <CardListField label="Nominal" value={formatCurrency(payment.amount)} />
                    <CardListField label="Tanggal" value={formatDate(payment.paymentDate)} />
                    <CardListField label="Metode" value={payment.method} />
                    <CardListField label="No. Referensi" value={payment.referenceNumber} />
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
                  <TH>Kelengkapan Evidence</TH>
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
                    <TD>{payment.evidenceComplete ? <Badge tone="success">Lengkap</Badge> : <Badge tone="warning">Belum Lengkap</Badge>}</TD>
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
        <AddVenuePaymentModal
          open={modalOpen}
          onClose={() => { setModalOpen(false); setActionError(null); }}
          onSubmit={handleAddPayment}
          error={actionError}
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

function emptyValues(): VenuePaymentFormValues {
  return {
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

function AddVenuePaymentModal({
  open,
  onClose,
  onSubmit,
  error,
}: {
  open: boolean;
  onClose: () => void;
  onSubmit: (values: VenuePaymentFormValues) => Promise<void>;
  error: string | null;
}) {
  const [values, setValues] = useState<VenuePaymentFormValues>(emptyValues);
  const [errors, setErrors] = useState<Partial<Record<keyof VenuePaymentFormValues, string>>>({});
  const [submitting, setSubmitting] = useState(false);

  function set<K extends keyof VenuePaymentFormValues>(key: K, value: VenuePaymentFormValues[K]) {
    setValues((prev) => ({ ...prev, [key]: value }));
  }

  async function handleSubmit() {
    const result = venuePaymentSchema.safeParse(values);
    if (!result.success) {
      const fieldErrors: Partial<Record<keyof VenuePaymentFormValues, string>> = {};
      for (const issue of result.error.issues) {
        fieldErrors[issue.path[0] as keyof VenuePaymentFormValues] = issue.message;
      }
      setErrors(fieldErrors);
      return;
    }
    setSubmitting(true);
    try {
      await onSubmit(result.data);
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
      title="Tambah Pembayaran Venue"
      description="Catat pembayaran baru ke venue untuk project ini."
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
          <Select value={values.type} onChange={(e) => set("type", e.target.value as VenuePaymentFormValues["type"])}>
            {VENUE_PAYMENT_TYPE_OPTIONS.map((t) => (
              <option key={t} value={t}>{t}</option>
            ))}
          </Select>
        </Field>
        <Field label="Nominal (Rp)" required hint={errors.amount}>
          <Input type="number" min={0} value={values.amount} onChange={(e) => set("amount", Number(e.target.value))} />
        </Field>
        <Field label="Tanggal" required hint={errors.paymentDate}>
          <Input type="date" value={values.paymentDate} onChange={(e) => set("paymentDate", e.target.value)} />
        </Field>
        <Field label="Metode" required hint={errors.method}>
          <Select value={values.method} onChange={(e) => set("method", e.target.value as VenuePaymentFormValues["method"])}>
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
