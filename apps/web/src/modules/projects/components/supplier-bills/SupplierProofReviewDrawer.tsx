import { lazy, Suspense, useRef, useState, type ReactNode } from "react";
import { BadgeCheck, Eye, XCircle } from "lucide-react";
import { Drawer } from "@/shared/components/ui/Drawer";
import { Button } from "@/shared/components/ui/Button";
import { Badge } from "@/shared/components/ui/Badge";
import { Textarea, Field } from "@/shared/components/ui/Input";
import { useZodForm } from "@/shared/forms/useZodForm";
import { FormErrorSummary } from "@/shared/forms/FormErrorSummary";
import { getApiErrorMessage } from "@/shared/lib/api-error";
import { SUPPLIER_PAYMENT_STATUS_LABELS, evidenceTypeLabel } from "@/shared/lib/labels";
import { formatCurrency, formatDate } from "@/shared/lib/formatters";
import { toast } from "@/shared/stores/useToastStore";
import { useCan } from "@/shared/stores/usePermissionStore";
import { jakartaDay } from "@/modules/projects/lib/proof";
import { SUPPLIER_PAYMENT_STATUS_TONE } from "@/modules/projects/lib/status";
import { supplierProofRejectSchema } from "@/modules/projects/schemas/supplier-bill.schema";
import { useSupplierBillStore } from "@/modules/projects/stores/useSupplierBillStore";
import type { SupplierPayment } from "@/modules/projects/types";

const EvidenceListModal = lazy(() => import("@/shared/components/ui/EvidenceListModal").then((m) => ({ default: m.EvidenceListModal })));

interface Props {
  projectId: string;
  payment: SupplierPayment;
  supplierName: string;
  /** The bill the couple paid ("DP", "Pelunasan"); null when it was not tied to one. */
  billLabel: string | null;
  onClose: () => void;
}

/**
 * "Periksa bukti" of a payment the couple made to a vendor they pay directly (Paket WO Tahap 4, the same check as a
 * client payment, ADR-0047): the WO looks at the transfer proof and either marks it as matching ("Tandai sesuai", it
 * then counts as paid to that vendor) or refuses it with a reason the couple reads ("Tolak dengan alasan"). The money
 * never passes through the WO, so none of this changes Keuangan. Loaded on demand.
 */
export default function SupplierProofReviewDrawer({ projectId, payment, supplierName, billLabel, onClose }: Props) {
  const verify = useSupplierBillStore((s) => s.verifySubmission);
  const reject = useSupplierBillStore((s) => s.rejectSubmission);
  const canRecord = useCan("finance.record");
  const [mode, setMode] = useState<"review" | "reject">("review");
  const [verifying, setVerifying] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [viewing, setViewing] = useState(false);
  const formRef = useRef<HTMLFormElement>(null);
  const form = useZodForm({ schema: supplierProofRejectSchema, initial: { reason: "" } });
  const pending = payment.status === "pending";
  const busy = verifying || form.submitting;
  const what = `${formatCurrency(payment.amount)} ke ${supplierName}`;

  async function handleVerify() {
    setError(null);
    setVerifying(true);
    try {
      await verify(projectId, payment.id);
      toast.success("Bukti pembayaran ditandai sesuai");
      onClose();
    } catch (err) {
      setError(getApiErrorMessage(err, "Bukti belum bisa ditandai sesuai. Periksa koneksi Anda, lalu coba lagi."));
    } finally {
      setVerifying(false);
    }
  }

  const submitReject = form.handleSubmit(
    async (parsed) => {
      await reject(projectId, payment.id, parsed.reason);
      toast.success("Bukti pembayaran ditolak");
      onClose();
    },
    {
      root: () => formRef.current,
      fallback: "Bukti belum bisa ditolak. Periksa koneksi Anda, lalu coba lagi.",
      known: (key) => key === "reason",
    },
  );

  const footer = !pending || !canRecord ? (
    <Button variant="secondary" onClick={onClose}>Tutup</Button>
  ) : mode === "reject" ? (
    <>
      <Button variant="secondary" onClick={() => setMode("review")} disabled={busy}>Kembali</Button>
      <Button variant="danger" type="submit" form="reject-proof-form" disabled={busy} icon={<XCircle className="h-4 w-4" />}>
        {form.submitting ? "Menolak..." : "Tolak bukti"}
      </Button>
    </>
  ) : (
    <>
      <Button variant="secondary" onClick={() => setMode("reject")} disabled={busy} icon={<XCircle className="h-4 w-4" />}>
        Tolak dengan alasan
      </Button>
      <Button onClick={() => void handleVerify()} disabled={busy} icon={<BadgeCheck className="h-4 w-4" />}>
        {verifying ? "Menyimpan..." : "Tandai sesuai"}
      </Button>
    </>
  );

  return (
    <>
      <Drawer
        open
        onClose={onClose}
        title="Periksa bukti bayar vendor"
        description={`Pasangan mengirim bukti pembayaran ${what}. Uang ini dibayar klien langsung ke vendor, tidak masuk kas WO.`}
        busy={busy}
        footer={footer}
      >
        <div className="flex flex-col gap-5">
          <dl className="grid grid-cols-2 gap-x-4 gap-y-3 rounded-lg border border-border bg-card px-4 py-3.5 text-[13px]">
            <Item label="Vendor atau venue" value={supplierName} wide />
            <Item label="Tagihan" value={billLabel ?? "-"} />
            <Item label="Nominal" value={<span className="font-semibold tabular-nums">{formatCurrency(payment.amount)}</span>} />
            <Item label="Tanggal bayar" value={formatDate(payment.paymentDate)} />
            <Item label="Metode" value={payment.method || "-"} />
            <Item label="No. referensi" value={payment.referenceNumber || "-"} wide />
            <Item
              label="Status"
              value={<Badge tone={SUPPLIER_PAYMENT_STATUS_TONE[payment.status]}>{SUPPLIER_PAYMENT_STATUS_LABELS.staff[payment.status]}</Badge>}
              wide
            />
          </dl>

          <section aria-labelledby="proof-files">
            <h3 id="proof-files" className="text-[13px] font-semibold text-text-primary">Bukti dari pasangan</h3>
            {payment.evidence.length === 0 ? (
              <p className="mt-1.5 text-[13px] text-text-secondary">Belum ada berkas bukti. Bukti tanpa berkas belum bisa ditandai sesuai.</p>
            ) : (
              <ul className="mt-2 flex flex-col gap-1.5">
                {payment.evidence.map((e) => (
                  <li key={e.id} className="flex items-center justify-between gap-3 rounded-md border border-border-light px-3 py-2 text-[13px]">
                    <span className="min-w-0 truncate">
                      <span className="text-text-secondary">{evidenceTypeLabel(e.type)}:</span> {e.fileName}
                    </span>
                  </li>
                ))}
                <li>
                  <Button size="sm" variant="secondary" icon={<Eye className="h-3.5 w-3.5" />} onClick={() => setViewing(true)}>
                    Lihat bukti
                  </Button>
                </li>
              </ul>
            )}
          </section>

          {payment.status === "verified" && payment.verifiedAt && (
            <p className="rounded-md bg-success-soft px-3.5 py-2.5 text-[13px] text-success-text">
              Ditandai sesuai {formatDate(jakartaDay(payment.verifiedAt))}
              {payment.verifiedBy ? ` oleh ${payment.verifiedBy}` : ""}. Pembayaran ini dihitung terbayar ke {supplierName}.
            </p>
          )}
          {payment.status === "rejected" && (
            <p className="rounded-md bg-danger-soft px-3.5 py-2.5 text-[13px] text-danger-text">
              Ditolak{payment.rejectedAt ? ` ${formatDate(jakartaDay(payment.rejectedAt))}` : ""}
              {payment.rejectedBy ? ` oleh ${payment.rejectedBy}` : ""}: {payment.rejectionReason || "tanpa alasan"}. Pasangan melihat alasan ini dan bisa mengirim bukti baru.
            </p>
          )}

          {pending && canRecord && mode === "review" && (
            <p className="text-[13px] text-text-secondary">
              Cocokkan nominal, tanggal dan rekening tujuan di bukti dengan tagihan {supplierName}. Bila sesuai, pembayaran ini dihitung terbayar ke vendor
              tersebut. Bila tidak, tolak dengan alasan agar pasangan tahu apa yang perlu diperbaiki.
            </p>
          )}
          {pending && canRecord && mode === "reject" && (
            <form id="reject-proof-form" ref={formRef} noValidate onSubmit={(e) => void submitReject(e)} className="flex flex-col gap-3">
              <FormErrorSummary errors={form.errorList} formError={form.formError} labels={{ reason: "Alasan" }} root={() => formRef.current} />
              <Field label="Alasan" required name="reason" error={form.errors.reason} hint="Pasangan membaca alasan ini di portal. Paling banyak 500 karakter.">
                <Textarea rows={3} maxLength={500} value={form.values.reason} onChange={(e) => form.set("reason", e.target.value)} onBlur={() => form.blur("reason")} />
              </Field>
            </form>
          )}
          {error && (
            <p role="alert" className="rounded-md bg-danger-soft px-3.5 py-2.5 text-[13px] text-danger-text">
              {error}
            </p>
          )}
        </div>
      </Drawer>

      {viewing && (
        <Suspense fallback={null}>
          <EvidenceListModal open projectId={projectId} title={`Bukti — ${supplierName}`} items={payment.evidence} onClose={() => setViewing(false)} />
        </Suspense>
      )}
    </>
  );
}

function Item({ label, value, wide }: { label: string; value: ReactNode; wide?: boolean }) {
  return (
    <div className={wide ? "col-span-2" : undefined}>
      <dt className="text-[12px] text-text-secondary">{label}</dt>
      <dd className="mt-0.5 text-text-primary">{value}</dd>
    </div>
  );
}
