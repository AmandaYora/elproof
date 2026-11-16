import { lazy, Suspense, useState } from "react";
import { useOutletContext } from "react-router-dom";
import { Badge, type BadgeTone } from "@/shared/components/ui/Badge";
import { Button } from "@/shared/components/kit/button";
import { ProofStamp } from "@/shared/components/ui/ProofStamp";
import { LoadError } from "@/shared/components/feedback/LoadError";
import { SkeletonRows } from "@/shared/components/ui/Skeleton";
import { toast } from "@/shared/stores/useToastStore";
import { useCan } from "@/shared/stores/usePermissionStore";
import { useProjectStore } from "@/modules/projects/stores/useProjectStore";
import { useClientHomeStore } from "@/modules/client-portal/stores/useClientHomeStore";
import { useProofViewer } from "@/modules/client-portal/components/useProofViewer";
import { PageIntro } from "@/modules/client-portal/components/PageIntro";
import { PaymentSummaryCard } from "@/modules/client-portal/components/payments/PaymentSummaryCard";
import { useLoad } from "@/modules/client-portal/lib/useLoad";
import { usePortalContact } from "@/modules/client-portal/lib/usePortalContact";
import { verifiedLabel } from "@/modules/projects/lib/proof";
import { TERM_STATUS_LABELS } from "@/shared/lib/labels";
import { formatCurrency, formatDate } from "@/shared/lib/formatters";
import type { PaymentTermStatus } from "@/modules/projects/types";
import type { ClientPortalContext } from "@/modules/client-portal/layouts/ClientPortalLayout";

// Only needed once someone taps "Kirim bukti transfer".
const SubmitProofDrawer = lazy(() => import("@/modules/client-portal/components/payments/SubmitProofDrawer"));

// Red is kept for the "you need to act" list on Beranda; here a late installment is amber.
const TERM_TONE: Record<PaymentTermStatus, BadgeTone> = { paid: "success", overdue: "warning", partial: "warning", due_soon: "warning", upcoming: "neutral" };

/**
 * The client's own money: one summary card (paid of contract, a bar per term, what is next), the schedule, and each
 * payment with its proof. A payment the WO has verified carries a stamp that opens the receipt; one the couple sent
 * themselves waits as "Menunggu verifikasi WO" until then.
 */
export default function PembayaranTabPage() {
  const { projectId } = useOutletContext<ClientPortalContext>();
  const finance = useClientHomeStore((s) => s.home?.finance);
  const payTo = useClientHomeStore((s) => s.home?.payTo);
  const payments = useProjectStore((s) => s.clientPayments);
  const terms = useProjectStore((s) => s.paymentTerms);
  const fetchClientPayments = useProjectStore((s) => s.fetchClientPayments);
  const fetchPaymentTerms = useProjectStore((s) => s.fetchPaymentTerms);
  const canSubmit = useCan("portal.finance.submit");
  const contact = usePortalContact();
  const viewer = useProofViewer(projectId);
  const [drawer, setDrawer] = useState<"unopened" | "open" | "closed">("unopened");
  const { status, retry } = useLoad(() => Promise.all([fetchClientPayments(projectId), fetchPaymentTerms(projectId)]), [projectId, fetchClientPayments, fetchPaymentTerms]);

  const sorted = [...payments].sort((a, b) => (a.paymentDate < b.paymentDate ? 1 : -1));
  const ordered = [...terms].sort((a, b) => a.dueDate.localeCompare(b.dueDate));

  function sent() {
    setDrawer("closed");
    toast.success("Bukti transfer terkirim. Wedding organizer Anda akan memverifikasinya.");
    void fetchClientPayments(projectId).catch(() => undefined);
  }

  return (
    <div className="mx-auto flex w-full max-w-3xl flex-col gap-6 sm:gap-8">
      <PageIntro title="Pembayaran">Pembayaran yang sudah dicek wedding organizer Anda diberi tanda terverifikasi.</PageIntro>

      {finance && (
        <PaymentSummaryCard
          finance={finance}
          terms={status === "ready" ? ordered : []}
          payTo={payTo}
          contact={contact}
          onSubmitProof={canSubmit ? () => setDrawer("open") : undefined}
        />
      )}

      {status === "loading" && <div className="rounded-2xl border border-border bg-card p-5"><SkeletonRows rows={3} label="Memuat pembayaran" /></div>}
      {status === "error" && <LoadError message="Pembayaran belum bisa dimuat. Periksa koneksi, lalu coba lagi." onRetry={retry} />}

      {status === "ready" && (
        <>
          {ordered.length > 0 && (
            <section aria-labelledby="terms-heading">
              <h2 id="terms-heading" className="mb-3 text-base font-semibold text-foreground">Jadwal pembayaran</h2>
              <ul className="overflow-hidden rounded-2xl border border-border bg-card">
                {ordered.map((t) => (
                  <li key={t.id} className="flex flex-wrap items-center justify-between gap-x-4 gap-y-2 border-b border-border-light px-4 py-3.5 last:border-b-0 sm:px-5">
                    <span className="min-w-0">
                      <span className="block text-[14px] font-semibold text-foreground">{t.label}</span>
                      <span className="block text-[13px] text-muted-foreground">
                        Jatuh tempo {formatDate(t.dueDate)}
                        {t.status !== "paid" && t.paid > 0 && ` · sisa ${formatCurrency(t.remaining)}`}
                      </span>
                    </span>
                    <span className="flex items-center gap-3">
                      <span className="text-[14px] font-semibold text-foreground tabular-nums">{formatCurrency(t.amount)}</span>
                      <Badge tone={TERM_TONE[t.status]}>{TERM_STATUS_LABELS.client[t.status]}</Badge>
                    </span>
                  </li>
                ))}
              </ul>
            </section>
          )}

          <section aria-labelledby="history-heading">
            <h2 id="history-heading" className="mb-3 text-base font-semibold text-foreground">Riwayat pembayaran</h2>
            {sorted.length === 0 ? (
              <p className="rounded-2xl border border-dashed border-border bg-card p-6 text-center text-[14px] text-muted-foreground">
                Pembayaran akan muncul di sini begitu wedding organizer Anda mencatatnya{canSubmit ? " atau Anda mengirim bukti transfer" : ""}.
              </p>
            ) : (
              <ul className="overflow-hidden rounded-2xl border border-border bg-card">
                {sorted.map((p) => {
                  const proof = p.evidence[0];
                  const title = `${p.type} ${formatDate(p.paymentDate)}`;
                  const mine = p.source === "client";
                  return (
                    <li key={p.id} className="border-b border-border-light px-4 py-4 last:border-b-0 sm:px-5">
                      <div className="flex flex-wrap items-start justify-between gap-x-4 gap-y-1">
                        <div className="min-w-0">
                          <p className="text-[14px] font-semibold text-foreground">{mine && !p.verifiedAt ? "Transfer dari Anda" : p.type}</p>
                          <p className="text-[13px] text-muted-foreground">
                            {formatDate(p.paymentDate)} · {p.method}
                          </p>
                        </div>
                        <p className="text-[16px] font-semibold text-foreground tabular-nums">{formatCurrency(p.amount)}</p>
                      </div>
                      <div className="mt-3 flex flex-wrap items-center gap-2">
                        {p.verifiedAt && proof ? (
                          <ProofStamp proof={proof} label={verifiedLabel(p.verifiedAt, p.verifiedBy)} seenKey={`payment-proof-${proof.id}`} onOpen={() => viewer.open(proof, title)} />
                        ) : (
                          <>
                            {!p.verifiedAt && (mine || proof) && <Badge tone="info">Menunggu verifikasi WO</Badge>}
                            {proof ? (
                              <Button variant="outline" size="sm" onClick={() => viewer.open(proof, title)}>Lihat bukti transfer</Button>
                            ) : (
                              <span className="text-[13px] text-muted-foreground">Bukti transfer belum diunggah</span>
                            )}
                          </>
                        )}
                      </div>
                    </li>
                  );
                })}
              </ul>
            )}
          </section>
        </>
      )}
      {viewer.element}
      {drawer !== "unopened" && (
        <Suspense fallback={null}>
          <SubmitProofDrawer
            open={drawer === "open"}
            onClose={() => setDrawer("closed")}
            projectId={projectId}
            suggestedAmount={finance?.nextTerm?.remaining ?? 0}
            onSent={sent}
          />
        </Suspense>
      )}
    </div>
  );
}
