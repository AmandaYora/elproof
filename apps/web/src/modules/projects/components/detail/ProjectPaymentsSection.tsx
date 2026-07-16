import { useEffect, useState } from "react";
import { Plus } from "lucide-react";
import { Card, CardHeader, CardContent } from "@/shared/components/ui/Card";
import { Badge } from "@/shared/components/ui/Badge";
import { Button } from "@/shared/components/ui/Button";
import { Modal } from "@/shared/components/ui/Modal";
import { Input, Textarea, Select, Field } from "@/shared/components/ui/Input";
import { Table, THead, TBody, TR, TH, TD } from "@/shared/components/ui/Table";
import { Pagination } from "@/shared/components/ui/Pagination";
import { usePagination } from "@/shared/hooks/usePagination";
import { useProjectStore } from "@/modules/projects/stores/useProjectStore";
import { useVendorStore } from "@/modules/vendors/stores/useVendorStore";
import { paymentSchema, PAYMENT_TYPE_OPTIONS, type PaymentFormValues } from "@/modules/projects/schemas/payment.schema";
import type { ProjectVendor } from "@/modules/projects/types";
import { todayISO } from "@/modules/projects/lib/dates";
import { getApiErrorMessage } from "@/shared/lib/api-error";
import { formatCurrency, formatDate } from "@/shared/lib/formatters";

export function ProjectPaymentsSection({ projectId }: { projectId: string }) {
  const payments = useProjectStore((s) => s.payments);
  const vendorEngagements = useProjectStore((s) => s.vendorEngagements);
  const fetchPayments = useProjectStore((s) => s.fetchPayments);
  const fetchVendorSection = useProjectStore((s) => s.fetchVendorSection);
  const createPayment = useProjectStore((s) => s.createPayment);
  const vendors = useVendorStore((s) => s.vendors);
  const fetchVendors = useVendorStore((s) => s.fetchVendors);

  const [modalOpen, setModalOpen] = useState(false);
  const [actionError, setActionError] = useState<string | null>(null);

  useEffect(() => {
    void fetchPayments(projectId);
    void fetchVendorSection(projectId);
    void fetchVendors();
  }, [projectId, fetchPayments, fetchVendorSection, fetchVendors]);

  const totalContractValue = vendorEngagements.reduce((sum, pv) => sum + pv.contractValue, 0);
  const totalPaid = vendorEngagements.reduce((sum, pv) => sum + pv.paidAmount, 0);
  const totalRemaining = vendorEngagements.reduce((sum, pv) => sum + (pv.contractValue - pv.paidAmount), 0);
  const { page, setPage, totalPages, totalItems, pageSize, pageItems } = usePagination(payments);

  async function handleAddPayment(values: PaymentFormValues) {
    setActionError(null);
    try {
      await createPayment(projectId, values);
      setModalOpen(false);
    } catch (err) {
      setActionError(getApiErrorMessage(err, "Gagal mencatat pembayaran"));
    }
  }

  return (
    <div id="pembayaran">
      <Card>
        <CardHeader
          title="Pembayaran Vendor"
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
                    </TR>
                  );
                })}
              </TBody>
            </Table>
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
          onClose={() => setModalOpen(false)}
          onSubmit={(values) => void handleAddPayment(values)}
          vendorEngagements={vendorEngagements}
          vendors={vendors}
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
  };
}

function AddPaymentModal({
  open,
  onClose,
  onSubmit,
  vendorEngagements,
  vendors,
}: {
  open: boolean;
  onClose: () => void;
  onSubmit: (values: PaymentFormValues) => void;
  vendorEngagements: ProjectVendor[];
  vendors: { id: string; name: string }[];
}) {
  const defaultVendorId = vendorEngagements[0]?.id ?? "";
  const [values, setValues] = useState<PaymentFormValues>(() => toFormValues(defaultVendorId));
  const [errors, setErrors] = useState<Partial<Record<keyof PaymentFormValues, string>>>({});

  function set<K extends keyof PaymentFormValues>(key: K, value: PaymentFormValues[K]) {
    setValues((prev) => ({ ...prev, [key]: value }));
  }

  function handleSubmit() {
    const result = paymentSchema.safeParse(values);
    if (!result.success) {
      const fieldErrors: Partial<Record<keyof PaymentFormValues, string>> = {};
      for (const issue of result.error.issues) {
        fieldErrors[issue.path[0] as keyof PaymentFormValues] = issue.message;
      }
      setErrors(fieldErrors);
      return;
    }
    onSubmit(result.data);
    setValues(toFormValues(defaultVendorId));
    setErrors({});
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
          <Button onClick={handleSubmit}>Simpan Pembayaran</Button>
        </>
      }
    >
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <Field label="Vendor" required hint={errors.projectVendorId}>
          <Select value={values.projectVendorId} onChange={(e) => set("projectVendorId", e.target.value)}>
            {vendorEngagements.map((pv) => (
              <option key={pv.id} value={pv.id}>
                {vendors.find((v) => v.id === pv.vendorId)?.name ?? "Vendor tidak diketahui"}
              </option>
            ))}
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
        <div className="sm:col-span-2">
          <Field label="Catatan">
            <Textarea rows={2} value={values.notes} onChange={(e) => set("notes", e.target.value)} />
          </Field>
        </div>
      </div>
    </Modal>
  );
}
