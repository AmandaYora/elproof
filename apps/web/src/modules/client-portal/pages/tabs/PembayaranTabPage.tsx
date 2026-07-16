import { useEffect, useState } from "react";
import { useOutletContext } from "react-router-dom";
import { Wallet, FileText } from "lucide-react";
import { Badge } from "@/shared/components/ui/Badge";
import { EvidenceViewerModal } from "@/shared/components/ui/EvidenceViewerModal";
import { useProjectStore } from "@/modules/projects/stores/useProjectStore";
import { useVendorStore } from "@/modules/vendors/stores/useVendorStore";
import type { Evidence } from "@/modules/projects/types";
import { formatCurrency, formatDate } from "@/shared/lib/formatters";
import type { ClientPortalContext } from "@/modules/client-portal/layouts/ClientPortalLayout";

export default function PembayaranTabPage() {
  const { projectId } = useOutletContext<ClientPortalContext>();
  const vendorEngagements = useProjectStore((s) => s.vendorEngagements);
  const payments = useProjectStore((s) => s.payments);
  const evidence = useProjectStore((s) => s.evidence);
  const fetchVendorSection = useProjectStore((s) => s.fetchVendorSection);
  const fetchPayments = useProjectStore((s) => s.fetchPayments);
  const fetchEvidence = useProjectStore((s) => s.fetchEvidence);
  const vendors = useVendorStore((s) => s.vendors);
  const fetchVendors = useVendorStore((s) => s.fetchVendors);
  const [viewingEvidence, setViewingEvidence] = useState<Evidence | null>(null);

  useEffect(() => {
    void fetchVendorSection(projectId);
    void fetchPayments(projectId);
    void fetchEvidence(projectId);
    void fetchVendors();
  }, [projectId, fetchVendorSection, fetchPayments, fetchEvidence, fetchVendors]);

  const sortedPayments = [...payments].sort((a, b) => (a.paymentDate < b.paymentDate ? 1 : -1));
  const totalContract = vendorEngagements.reduce((sum, pv) => sum + pv.contractValue, 0);
  const totalPaid = vendorEngagements.reduce((sum, pv) => sum + pv.paidAmount, 0);
  const totalRemaining = totalContract - totalPaid;

  return (
    <div className="flex flex-col gap-6 sm:gap-8">
      <section>
        <div className="mb-4 flex items-center gap-2 sm:mb-5">
          <Wallet className="h-5 w-5 shrink-0 text-navy-900" />
          <h2 className="text-base font-bold text-text-primary sm:text-lg">Transparansi Pembayaran</h2>
        </div>
        <p className="mb-5 max-w-2xl text-[13px] leading-relaxed text-text-secondary sm:mb-6 sm:text-[13.5px]">
          Halaman ini menampilkan setiap pembayaran yang telah dilakukan tim kami kepada vendor untuk pernikahan
          Anda, lengkap dengan dokumen pendukungnya — tidak ada yang kami sembunyikan.
        </p>

        <div className="grid grid-cols-1 gap-3 sm:grid-cols-3 sm:gap-4">
          <div className="rounded-xl border border-border bg-surface p-4 sm:p-6">
            <p className="text-[12px] font-medium text-text-secondary sm:text-[12.5px]">Total Nilai Kerja Sama</p>
            <p className="mt-1.5 text-lg font-bold tabular-nums text-text-primary sm:text-xl">{formatCurrency(totalContract)}</p>
          </div>
          <div className="rounded-xl border border-border bg-surface p-4 sm:p-6">
            <p className="text-[12px] font-medium text-text-secondary sm:text-[12.5px]">Total Sudah Dibayar</p>
            <p className="mt-1.5 text-lg font-bold tabular-nums text-success sm:text-xl">{formatCurrency(totalPaid)}</p>
          </div>
          <div className="rounded-xl border border-border bg-surface p-4 sm:p-6">
            <p className="text-[12px] font-medium text-text-secondary sm:text-[12.5px]">Sisa Pembayaran</p>
            <p className="mt-1.5 text-lg font-bold tabular-nums text-text-primary sm:text-xl">{formatCurrency(totalRemaining)}</p>
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
          <div className="flex flex-col gap-3">
            {sortedPayments.map((payment) => {
              const pv = vendorEngagements.find((v) => v.id === payment.projectVendorId);
              const vendorName = pv ? vendors.find((v) => v.id === pv.vendorId)?.name : null;
              const invoiceEvidence = payment.invoiceEvidenceId ? evidence.find((e) => e.id === payment.invoiceEvidenceId) : null;
              const proofEvidence = payment.proofEvidenceId ? evidence.find((e) => e.id === payment.proofEvidenceId) : null;

              return (
                <div key={payment.id} className="flex flex-col gap-2.5 rounded-xl border border-border bg-surface p-4 sm:p-5">
                  <div className="flex flex-col items-start gap-2 sm:flex-row sm:items-center sm:justify-between sm:gap-3">
                    <div>
                      <p className="text-[13.5px] font-semibold text-text-primary sm:text-[14px]">
                        {vendorName ?? "Vendor tidak diketahui"}
                      </p>
                      <p className="mt-0.5 text-[12px] text-text-secondary sm:text-[12.5px]">
                        {payment.type} · {formatDate(payment.paymentDate)}
                      </p>
                    </div>
                    <div className="flex items-center gap-2 sm:gap-3">
                      <span className="text-[13.5px] font-semibold tabular-nums text-text-primary sm:text-[14px]">
                        {formatCurrency(payment.amount)}
                      </span>
                      {payment.evidenceComplete ? (
                        <Badge tone="success">Bukti pembayaran tersedia</Badge>
                      ) : (
                        <Badge tone="warning">Bukti pembayaran belum lengkap</Badge>
                      )}
                    </div>
                  </div>
                  {(invoiceEvidence || proofEvidence) && (
                    <div className="flex flex-wrap items-center gap-1.5 border-t border-border-light pt-2.5">
                      <FileText className="h-3 w-3 shrink-0 text-text-secondary" />
                      {invoiceEvidence && (
                        <button
                          onClick={() => setViewingEvidence(invoiceEvidence)}
                          className="rounded-full border border-border bg-white px-2.5 py-1 text-[11.5px] font-medium text-navy-900 hover:bg-navy-900/10"
                        >
                          Lihat Invoice
                        </button>
                      )}
                      {proofEvidence && (
                        <button
                          onClick={() => setViewingEvidence(proofEvidence)}
                          className="rounded-full border border-border bg-white px-2.5 py-1 text-[11.5px] font-medium text-navy-900 hover:bg-navy-900/10"
                        >
                          Lihat Bukti Transfer
                        </button>
                      )}
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
