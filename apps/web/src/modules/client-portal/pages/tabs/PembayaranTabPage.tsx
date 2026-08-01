import { useEffect, useState } from "react";
import { useOutletContext } from "react-router-dom";
import { Wallet, FileText } from "lucide-react";
import { EvidenceViewerModal } from "@/shared/components/ui/EvidenceViewerModal";
import { useProjectStore } from "@/modules/projects/stores/useProjectStore";
import type { Evidence } from "@/modules/projects/types";
import { formatCurrency, formatDate } from "@/shared/lib/formatters";
import type { ClientPortalContext } from "@/modules/client-portal/layouts/ClientPortalLayout";

// Repurposed (PLAN.md "Uang Masuk dari Client", §3.8) — shows the client's
// OWN payment history against the project's contractValue, not the WO's
// spend on vendors (not relevant to a client, and a cost-management detail
// they were never meant to see). Same three-card visual layout as before,
// new source numbers; each card now shows a single "Bukti Transfer" button
// (a client payment has only one evidence slot) instead of an Invoice/Bukti
// Transfer pair, and drops the vendor-name line entirely — there is no
// vendor on a client payment.
export default function PembayaranTabPage() {
  const { projectId } = useOutletContext<ClientPortalContext>();
  const project = useProjectStore((s) => s.currentProject);
  const clientPayments = useProjectStore((s) => s.clientPayments);
  const evidence = useProjectStore((s) => s.evidence);
  const fetchClientPayments = useProjectStore((s) => s.fetchClientPayments);
  const fetchEvidence = useProjectStore((s) => s.fetchEvidence);
  const [viewingEvidence, setViewingEvidence] = useState<Evidence | null>(null);

  useEffect(() => {
    void fetchClientPayments(projectId);
    void fetchEvidence(projectId);
  }, [projectId, fetchClientPayments, fetchEvidence]);

  const sortedPayments = [...clientPayments].sort((a, b) => (a.paymentDate < b.paymentDate ? 1 : -1));
  const contractValue = project?.contractValue ?? 0;
  const totalReceived = clientPayments.reduce(
    (sum, p) => (p.type === "Refund" ? sum - p.amount : sum + p.amount),
    0
  );
  const totalRemaining = contractValue - totalReceived;

  return (
    <div className="flex flex-col gap-6 sm:gap-8">
      <section>
        <div className="mb-4 flex items-center gap-2 sm:mb-5">
          <Wallet className="h-5 w-5 shrink-0 text-navy-900" />
          <h2 className="text-base font-bold text-text-primary sm:text-lg">Transparansi Pembayaran</h2>
        </div>
        <p className="mb-5 max-w-2xl text-[13px] leading-relaxed text-text-secondary sm:mb-6 sm:text-[13.5px]">
          Halaman ini menampilkan riwayat pembayaran yang telah Anda lakukan untuk pernikahan Anda, lengkap
          dengan bukti transfernya — tidak ada yang kami sembunyikan.
        </p>

        <div className="grid grid-cols-1 gap-4 sm:grid-cols-3 sm:gap-6">
          <div className="rounded-2xl border border-border bg-white p-5 sm:p-6 shadow-sm hover:shadow-md transition-shadow">
            <p className="text-[12.5px] font-bold text-text-secondary uppercase tracking-wider">Nilai Kontrak</p>
            <p className="mt-3 text-2xl font-bold tabular-nums text-navy-950">{formatCurrency(contractValue)}</p>
          </div>
          <div className="rounded-2xl border border-emerald-100 bg-emerald-50/30 p-5 sm:p-6 shadow-sm hover:shadow-md transition-shadow">
            <p className="text-[12.5px] font-bold text-emerald-700 uppercase tracking-wider">Total Diterima</p>
            <p className="mt-3 text-2xl font-bold tabular-nums text-emerald-600">{formatCurrency(totalReceived)}</p>
          </div>
          <div className="rounded-2xl border border-amber-100 bg-amber-50/30 p-5 sm:p-6 shadow-sm hover:shadow-md transition-shadow">
            <p className="text-[12.5px] font-bold text-amber-700 uppercase tracking-wider">Sisa Tagihan</p>
            <p className="mt-3 text-2xl font-bold tabular-nums text-amber-600">{formatCurrency(totalRemaining)}</p>
          </div>
        </div>
      </section>

      <section>
        <h3 className="mb-3 text-[14px] font-bold text-text-primary sm:mb-4 sm:text-[15px]">Riwayat Pembayaran</h3>
        {sortedPayments.length === 0 ? (
          <p className="rounded-xl border border-dashed border-border bg-surface p-4 text-center text-[13px] text-text-secondary sm:p-6 sm:text-[13.5px]">
            Pembayaran akan muncul di sini seiring proses persiapan berjalan.
          </p>
        ) : (
          <div className="flex flex-col gap-4">
            {sortedPayments.map((payment) => {
              const proofEvidence = evidence.find((e) => e.relatedKind === "clientPayment" && e.relatedId === payment.id);

              return (
                <div key={payment.id} className="group relative flex flex-col gap-4 rounded-2xl border border-border bg-white p-5 sm:p-6 shadow-sm hover:shadow-md transition-shadow overflow-hidden">
                  <div className="absolute left-0 top-0 bottom-0 w-1.5 bg-navy-900 opacity-0 group-hover:opacity-100 transition-opacity"></div>

                  <div className="flex flex-col items-start gap-3 sm:flex-row sm:items-center sm:justify-between sm:gap-4">
                    <div>
                      <p className="text-[15px] font-bold text-navy-950 sm:text-[16px]">{payment.type}</p>
                      <p className="mt-1 text-[13px] font-medium text-text-secondary bg-surface-muted inline-block px-2.5 py-1 rounded-md">
                        {formatDate(payment.paymentDate)} · {payment.method}
                      </p>
                    </div>
                    <div className="flex flex-col items-start sm:items-end gap-2">
                      <span className="text-[18px] font-bold tabular-nums text-navy-950 sm:text-[20px]">
                        {formatCurrency(payment.amount)}
                      </span>
                      {payment.evidenceComplete ? (
                        <span className="inline-flex items-center gap-1.5 rounded-full bg-emerald-50 px-2.5 py-1 text-[11px] font-bold text-emerald-700 ring-1 ring-emerald-500/20">
                          BUKTI TERSEDIA
                        </span>
                      ) : (
                        <span className="inline-flex items-center gap-1.5 rounded-full bg-warning-soft px-2.5 py-1 text-[11px] font-bold text-warning-strong ring-1 ring-warning/30">
                          BUKTI BELUM ADA
                        </span>
                      )}
                    </div>
                  </div>

                  {proofEvidence && (
                    <div className="flex flex-wrap items-center gap-2 border-t border-border/50 pt-4 mt-2">
                      <FileText className="h-4 w-4 shrink-0 text-text-tertiary" />
                      <span className="text-[13px] font-medium text-text-secondary mr-2">Dokumen Pendukung:</span>
                      <button
                        onClick={() => setViewingEvidence(proofEvidence)}
                        className="flex items-center gap-1.5 rounded-lg border border-border bg-surface px-3 py-1.5 text-[12px] font-semibold text-navy-900 hover:bg-navy-50 hover:border-navy-200 transition-colors shadow-sm"
                      >
                        Bukti Transfer
                      </button>
                    </div>
                  )}
                </div>
              );
            })}
          </div>
        )}
      </section>

      {viewingEvidence && (
        <EvidenceViewerModal open onClose={() => setViewingEvidence(null)} projectId={projectId} evidence={viewingEvidence} />
      )}
    </div>
  );
}
