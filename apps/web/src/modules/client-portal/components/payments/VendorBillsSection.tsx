import { lazy, Suspense, useEffect, useState } from "react";
import { useLocation } from "react-router-dom";
import { Badge, type BadgeTone } from "@/shared/components/ui/Badge";
import { Button } from "@/shared/components/kit/button";
import { LoadError } from "@/shared/components/feedback/LoadError";
import { SkeletonRows } from "@/shared/components/ui/Skeleton";
import { toast } from "@/shared/stores/useToastStore";
import { SUPPLIER_PAYMENT_STATUS_LABELS, TERM_STATUS_LABELS } from "@/shared/lib/labels";
import { formatCurrency, formatDate } from "@/shared/lib/formatters";
import { useSupplierBillStore } from "@/modules/projects/stores/useSupplierBillStore";
import { useLoad } from "@/modules/client-portal/lib/useLoad";
import type { ProofToOpen } from "@/modules/client-portal/components/useProofViewer";
import type { PaymentTermStatus, SupplierBill, SupplierBillTerm, SupplierPaymentStatus } from "@/modules/projects/types";

// Only needed once someone taps "Kirim bukti bayar".
const SubmitProofDrawer = lazy(() => import("@/modules/client-portal/components/payments/SubmitProofDrawer"));

// Red is kept for the "you need to act" list on Beranda; here a late bill is amber.
const TERM_TONE: Record<PaymentTermStatus, BadgeTone> = { paid: "success", overdue: "warning", partial: "warning", due_soon: "warning", upcoming: "neutral" };
const PAYMENT_TONE: Record<SupplierPaymentStatus, BadgeTone> = { recorded: "neutral", pending: "info", verified: "success", rejected: "warning" };

interface Props {
  projectId: string;
  /** portal.finance.submit: the couple may send a proof of paying a bill. */
  canSubmit: boolean;
  onOpenProof: (proof: ProofToOpen, title: string) => void;
}

/**
 * Pembayaran › "Tagihan vendor" (Paket WO Tahap 4): every vendor or venue the couple pays directly, by its real name,
 * with its bills, what is paid and what is left, and each proof they sent with where it stands. "Kirim bukti bayar"
 * on a bill sends the transfer proof for the WO to check; it counts as paid once the WO marks it as matching. Hidden
 * when no vendor is paid directly. Only a role with portal.finance.view reaches this page (Keluarga does not).
 */
export function VendorBillsSection({ projectId, canSubmit, onOpenProof }: Props) {
  const bills = useSupplierBillStore((s) => s.bills);
  const billsProject = useSupplierBillStore((s) => s.projectId);
  const fetchBills = useSupplierBillStore((s) => s.fetchBills);
  const submitVendorProof = useSupplierBillStore((s) => s.submitVendorProof);
  const { status, retry } = useLoad(() => fetchBills(projectId), [projectId, fetchBills]);
  const [paying, setPaying] = useState<{ term: SupplierBillTerm; supplierName: string } | null>(null);
  const [drawerOpen, setDrawerOpen] = useState(false);

  const { hash } = useLocation();
  // Beranda's "Lihat tagihan vendor" lands here (#tagihan-vendor): scroll once the list is in.
  useEffect(() => {
    if (status === "ready" && hash === "#tagihan-vendor") document.getElementById("tagihan-vendor")?.scrollIntoView({ block: "start" });
  }, [status, hash]);

  const shown = billsProject === projectId ? bills.filter((b) => b.terms.length > 0 || b.payments.length > 0) : [];

  if (status === "ready" && shown.length === 0) return null;

  function startPaying(term: SupplierBillTerm, supplierName: string) {
    setPaying({ term, supplierName });
    setDrawerOpen(true);
  }

  function sent() {
    setDrawerOpen(false);
    toast.success("Bukti pembayaran terkirim. Wedding organizer Anda akan memeriksanya.");
  }

  return (
    <section id="tagihan-vendor" aria-labelledby="vendor-bills-heading" className="scroll-mt-20">
      <h2 id="vendor-bills-heading" className="text-base font-semibold text-foreground">Tagihan vendor</h2>
      <p className="mt-1 mb-3 text-[13.5px] text-muted-foreground">Anda membayar vendor ini langsung, di luar nilai kontrak.</p>

      {status === "loading" && (
        <div className="rounded-2xl border border-border bg-card p-5">
          <SkeletonRows rows={2} label="Memuat tagihan vendor" />
        </div>
      )}
      {status === "error" && <LoadError message="Tagihan vendor belum bisa dimuat. Periksa koneksi, lalu coba lagi." onRetry={retry} />}

      {status === "ready" && (
        <ul className="flex flex-col gap-4">
          {shown.map((b) => (
            <VendorCard key={b.projectSupplierId} bill={b} canSubmit={canSubmit} onPay={(t) => startPaying(t, b.name)} onOpenProof={onOpenProof} />
          ))}
        </ul>
      )}

      {paying && (
        <Suspense fallback={null}>
          <SubmitProofDrawer
            key={paying.term.id}
            open={drawerOpen}
            onClose={() => setDrawerOpen(false)}
            projectId={projectId}
            suggestedAmount={Math.max(paying.term.remaining - paying.term.pending, 0)}
            onSent={sent}
            purpose={{
              title: "Kirim bukti bayar",
              description: `Bukti pembayaran ${paying.term.label} ke ${paying.supplierName}. Wedding organizer Anda memeriksanya dulu; setelah sesuai, tagihan ini dihitung terbayar.`,
              submitLabel: "Kirim bukti bayar",
              notesHint: "Opsional, mis. nama pengirim bila berbeda.",
              send: (values, key) => submitVendorProof(projectId, paying.term.id, values, key),
            }}
          />
        </Suspense>
      )}
    </section>
  );
}

function VendorCard({
  bill,
  canSubmit,
  onPay,
  onOpenProof,
}: {
  bill: SupplierBill;
  canSubmit: boolean;
  onPay: (term: SupplierBillTerm) => void;
  onOpenProof: (proof: ProofToOpen, title: string) => void;
}) {
  const terms = [...bill.terms].sort((a, b) => a.dueDate.localeCompare(b.dueDate));
  const payments = [...bill.payments].sort((a, b) => (a.paymentDate < b.paymentDate ? 1 : -1));
  const labelOf = (termId: string | null) => (termId ? terms.find((t) => t.id === termId)?.label : undefined);

  return (
    <li className="overflow-hidden rounded-2xl border border-border bg-card">
      <div className="px-4 pt-4 pb-3 sm:px-5">
        <p className="text-[15px] font-semibold text-foreground">{bill.name}</p>
        <p className="text-[12.5px] text-muted-foreground">{bill.kind === "venue" ? "Venue" : "Vendor"}</p>
        <dl className="mt-3 grid grid-cols-3 gap-2 text-[12.5px]">
          <div>
            <dt className="text-muted-foreground">Total</dt>
            <dd className="text-[14px] font-semibold text-foreground tabular-nums">{formatCurrency(bill.total)}</dd>
          </div>
          <div>
            <dt className="text-muted-foreground">Sudah dibayar</dt>
            <dd className="text-[14px] font-semibold text-foreground tabular-nums">{formatCurrency(bill.paid)}</dd>
          </div>
          <div>
            <dt className="text-muted-foreground">Sisa</dt>
            <dd className="text-[14px] font-semibold text-foreground tabular-nums">{formatCurrency(bill.remaining)}</dd>
          </div>
        </dl>
        {bill.pending > 0 && (
          <p className="mt-2 text-[12.5px] text-muted-foreground">{formatCurrency(bill.pending)} yang Anda kirim sedang diperiksa wedding organizer Anda.</p>
        )}
      </div>

      {terms.length > 0 && (
        <ul aria-label={`Tagihan ${bill.name}`} className="border-t border-border-light">
          {terms.map((t) => {
            const left = Math.max(t.remaining - t.pending, 0);
            return (
              <li key={t.id} className="flex flex-col gap-2 border-b border-border-light px-4 py-3.5 last:border-b-0 sm:px-5">
                <div className="flex items-start justify-between gap-3">
                  <span className="min-w-0">
                    <span className="block text-[14px] font-semibold text-foreground">{t.label}</span>
                    <span className="block text-[13px] text-muted-foreground">
                      Jatuh tempo {formatDate(t.dueDate)}
                      {t.status !== "paid" && t.paid > 0 && ` · sisa ${formatCurrency(t.remaining)}`}
                    </span>
                  </span>
                  <span className="flex shrink-0 flex-col items-end gap-1">
                    <span className="text-[14px] font-semibold text-foreground tabular-nums">{formatCurrency(t.amount)}</span>
                    <Badge tone={TERM_TONE[t.status]}>{TERM_STATUS_LABELS.client[t.status]}</Badge>
                  </span>
                </div>
                {t.pending > 0 && <p className="text-[12.5px] text-muted-foreground">{formatCurrency(t.pending)} menunggu diperiksa wedding organizer Anda.</p>}
                {canSubmit && t.status !== "paid" && left > 0 && (
                  <Button variant="outline" className="h-11 w-full sm:w-auto sm:self-start" onClick={() => onPay(t)} aria-label={`Kirim bukti bayar ${t.label} ${bill.name}`}>
                    Kirim bukti bayar
                  </Button>
                )}
              </li>
            );
          })}
        </ul>
      )}

      {payments.length > 0 && (
        <div className="border-t border-border bg-muted/40 px-4 py-3.5 sm:px-5">
          <h3 className="text-[13px] font-semibold text-foreground">Riwayat pembayaran ke {bill.name}</h3>
          <ul className="mt-2 flex flex-col gap-3">
            {payments.map((p) => {
              const proof = p.evidence[0];
              const what = labelOf(p.billTermId) ?? p.type;
              return (
                <li key={p.id} className="flex flex-col gap-1.5">
                  <div className="flex items-start justify-between gap-3">
                    <span className="min-w-0 text-[13px]">
                      <span className="block font-medium text-foreground">{what}</span>
                      <span className="block text-muted-foreground">
                        {formatDate(p.paymentDate)} · {p.method}
                      </span>
                    </span>
                    <span className="shrink-0 text-[14px] font-semibold text-foreground tabular-nums">{formatCurrency(p.amount)}</span>
                  </div>
                  <div className="flex flex-wrap items-center gap-2">
                    <Badge tone={PAYMENT_TONE[p.status]}>{SUPPLIER_PAYMENT_STATUS_LABELS.client[p.status]}</Badge>
                    {proof && (
                      <Button variant="outline" size="sm" onClick={() => onOpenProof(proof, `${what} ${bill.name} ${formatDate(p.paymentDate)}`)}>
                        Lihat bukti
                      </Button>
                    )}
                  </div>
                  {p.status === "rejected" && (
                    <p className="text-[12.5px] text-warning-text">
                      Ditolak: {p.rejectionReason || "tanpa alasan"}.{canSubmit && " Kirim bukti yang benar lewat tombol Kirim bukti bayar di atas."}
                    </p>
                  )}
                </li>
              );
            })}
          </ul>
        </div>
      )}
    </li>
  );
}
