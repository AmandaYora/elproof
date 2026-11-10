import { useOutletContext } from "react-router-dom";
import { Badge, type BadgeTone } from "@/shared/components/ui/Badge";
import { Button } from "@/shared/components/kit/button";
import { ProofStamp } from "@/shared/components/ui/ProofStamp";
import { LoadError } from "@/shared/components/feedback/LoadError";
import { SkeletonRows } from "@/shared/components/ui/Skeleton";
import { useProjectStore } from "@/modules/projects/stores/useProjectStore";
import { useClientHomeStore } from "@/modules/client-portal/stores/useClientHomeStore";
import { useProofViewer } from "@/modules/client-portal/components/useProofViewer";
import { useLoad } from "@/modules/client-portal/lib/useLoad";
import { verifiedLabel } from "@/modules/projects/lib/proof";
import { TERM_STATUS_LABELS } from "@/shared/lib/labels";
import { formatCurrency, formatDate } from "@/shared/lib/formatters";
import type { PaymentTermStatus } from "@/modules/projects/types";
import type { ClientPortalContext } from "@/modules/client-portal/layouts/ClientPortalLayout";

// Red is kept for the one "you need to act" card on Beranda; here a late
// installment is amber.
const TERM_TONE: Record<PaymentTermStatus, BadgeTone> = { paid: "success", overdue: "warning", partial: "warning", due_soon: "warning", upcoming: "neutral" };

function Figure({ label, value, tone }: { label: string; value: number; tone?: "good" | "owed" }) {
  return (
    <div className="rounded-2xl border border-border bg-card p-5">
      <p className="text-[12.5px] font-medium text-muted-foreground">{label}</p>
      <p className={`mt-2 text-2xl font-semibold tabular-nums ${tone === "good" ? "text-success-text" : tone === "owed" ? "text-warning-text" : "text-foreground"}`}>{formatCurrency(value)}</p>
    </div>
  );
}

/**
 * The client's own money: what the contract is, what has been paid, what is
 * still owed, the schedule, and each payment with its proof. A payment the WO
 * has verified carries a stamp that opens the transfer receipt.
 */
export default function PembayaranTabPage() {
  const { projectId } = useOutletContext<ClientPortalContext>();
  const finance = useClientHomeStore((s) => s.home?.finance);
  const payments = useProjectStore((s) => s.clientPayments);
  const terms = useProjectStore((s) => s.paymentTerms);
  const fetchClientPayments = useProjectStore((s) => s.fetchClientPayments);
  const fetchPaymentTerms = useProjectStore((s) => s.fetchPaymentTerms);
  const viewer = useProofViewer(projectId);
  const { status, retry } = useLoad(() => Promise.all([fetchClientPayments(projectId), fetchPaymentTerms(projectId)]), [projectId, fetchClientPayments, fetchPaymentTerms]);

  const sorted = [...payments].sort((a, b) => (a.paymentDate < b.paymentDate ? 1 : -1));

  return (
    <div className="mx-auto flex w-full max-w-3xl flex-col gap-6 sm:gap-8">
      <header>
        <h1 className="text-xl font-semibold text-foreground">Pembayaran</h1>
        <p className="mt-1.5 max-w-prose text-[14px] leading-relaxed text-muted-foreground">
          Riwayat pembayaran Anda beserta bukti transfernya. Pembayaran yang sudah dicek wedding organizer Anda diberi tanda terverifikasi.
        </p>
      </header>

      {finance && (
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-3 sm:gap-4">
          <Figure label="Nilai kontrak" value={finance.contractValue} />
          <Figure label="Sudah dibayar" value={finance.paid} tone="good" />
          <Figure label="Sisa tagihan" value={finance.remaining} tone={finance.remaining > 0 ? "owed" : undefined} />
        </div>
      )}

      {status === "loading" && <div className="rounded-2xl border border-border bg-card p-5"><SkeletonRows rows={3} label="Memuat pembayaran" /></div>}
      {status === "error" && <LoadError message="Pembayaran belum bisa dimuat. Periksa koneksi, lalu coba lagi." onRetry={retry} />}

      {status === "ready" && (
        <>
          {terms.length > 0 && (
            <section aria-labelledby="terms-heading">
              <h2 id="terms-heading" className="mb-3 text-[15px] font-semibold text-foreground">Jadwal pembayaran</h2>
              <ul className="overflow-hidden rounded-2xl border border-border bg-card">
                {terms.map((t) => (
                  <li key={t.id} className="flex flex-wrap items-center justify-between gap-x-4 gap-y-2 border-b px-4 py-3.5 last:border-b-0">
                    <span className="min-w-0">
                      <span className="block text-[14px] font-semibold text-foreground">{t.label}</span>
                      <span className="block text-[12.5px] text-muted-foreground">Jatuh tempo {formatDate(t.dueDate)}</span>
                    </span>
                    <span className="flex items-center gap-3">
                      <span className="text-[14px] font-semibold tabular-nums text-foreground">{formatCurrency(t.amount)}</span>
                      <Badge tone={TERM_TONE[t.status]}>{TERM_STATUS_LABELS.client[t.status]}</Badge>
                    </span>
                  </li>
                ))}
              </ul>
            </section>
          )}

          <section aria-labelledby="history-heading">
            <h2 id="history-heading" className="mb-3 text-[15px] font-semibold text-foreground">Riwayat pembayaran</h2>
            {sorted.length === 0 ? (
              <p className="rounded-2xl border border-dashed border-border bg-card p-6 text-center text-[13.5px] text-muted-foreground">
                Pembayaran akan muncul di sini begitu wedding organizer Anda mencatatnya.
              </p>
            ) : (
              <ul className="flex flex-col gap-3">
                {sorted.map((p) => {
                  const proof = p.evidence[0];
                  const title = `${p.type} ${formatDate(p.paymentDate)}`;
                  return (
                    <li key={p.id} className="rounded-2xl border border-border bg-card p-4 sm:p-5">
                      <div className="flex flex-wrap items-start justify-between gap-x-4 gap-y-1">
                        <div>
                          <p className="text-[15px] font-semibold text-foreground">{p.type}</p>
                          <p className="text-[13px] text-muted-foreground">{formatDate(p.paymentDate)} · {p.method}</p>
                        </div>
                        <p className="text-[18px] font-semibold tabular-nums text-foreground">{formatCurrency(p.amount)}</p>
                      </div>
                      <div className="mt-3 flex flex-wrap items-center gap-2">
                        {p.verifiedAt && proof ? (
                          <ProofStamp
                            proof={proof}
                            label={verifiedLabel(p.verifiedAt, p.verifiedBy)}
                            seenKey={`payment-proof-${proof.id}`}
                            onOpen={() => viewer.open(proof, title)}
                          />
                        ) : proof ? (
                          <>
                            <Button variant="outline" size="sm" onClick={() => viewer.open(proof, title)}>Lihat bukti transfer</Button>
                            <span className="text-[12.5px] text-muted-foreground">Menunggu dicek wedding organizer</span>
                          </>
                        ) : (
                          <span className="text-[12.5px] text-muted-foreground">Bukti transfer belum diunggah</span>
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
    </div>
  );
}
