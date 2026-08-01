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
import { useProjectStore, ClientPaymentEvidenceError } from "@/modules/projects/stores/useProjectStore";
import {
  clientPaymentSchema,
  CLIENT_PAYMENT_TYPE_OPTIONS,
  type ClientPaymentFormValues,
} from "@/modules/projects/schemas/client-payment.schema";
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
  const fetchClientPayments = useProjectStore((s) => s.fetchClientPayments);
  const createClientPayment = useProjectStore((s) => s.createClientPayment);

  const [modalOpen, setModalOpen] = useState(false);
  const [actionError, setActionError] = useState<string | null>(null);

  useEffect(() => {
    void fetchClientPayments(projectId);
  }, [projectId, fetchClientPayments]);

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
          onClose={() => setModalOpen(false)}
          onSubmit={handleAddPayment}
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
}: {
  open: boolean;
  onClose: () => void;
  onSubmit: (values: ClientPaymentFormValues) => Promise<void>;
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
      // Parent already surfaces the error via its own actionError banner —
      // nothing further to do here except leave the form as the user left it.
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
        <Field label="Jenis Pembayaran" required>
          <Select value={values.type} onChange={(e) => set("type", e.target.value as ClientPaymentFormValues["type"])}>
            {CLIENT_PAYMENT_TYPE_OPTIONS.map((t) => (
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
          <Input value={values.method} onChange={(e) => set("method", e.target.value)} />
        </Field>
        <Field label="No. Referensi" required hint={errors.referenceNumber}>
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
