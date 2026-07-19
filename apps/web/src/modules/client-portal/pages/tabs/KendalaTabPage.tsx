import { useEffect, useState } from "react";
import { useOutletContext } from "react-router-dom";
import { CheckCircle2, HeartHandshake } from "lucide-react";
import { EvidenceViewerModal } from "@/shared/components/ui/EvidenceViewerModal";
import { IssueCard } from "@/modules/client-portal/components/IssueCard";
import { useProjectStore } from "@/modules/projects/stores/useProjectStore";
import { useVendorStore } from "@/modules/vendors/stores/useVendorStore";
import type { Evidence } from "@/modules/projects/types";
import type { ClientPortalContext } from "@/modules/client-portal/layouts/ClientPortalLayout";

export default function KendalaTabPage() {
  const { projectId } = useOutletContext<ClientPortalContext>();
  const issues = useProjectStore((s) => s.issues);
  const evidence = useProjectStore((s) => s.evidence);
  const vendorEngagements = useProjectStore((s) => s.vendorEngagements);
  const fetchIssues = useProjectStore((s) => s.fetchIssues);
  const fetchEvidence = useProjectStore((s) => s.fetchEvidence);
  const fetchVendorSection = useProjectStore((s) => s.fetchVendorSection);
  const vendors = useVendorStore((s) => s.vendors);
  const fetchVendors = useVendorStore((s) => s.fetchVendors);
  const [viewingEvidence, setViewingEvidence] = useState<Evidence | null>(null);

  useEffect(() => {
    void fetchIssues(projectId);
    void fetchEvidence(projectId);
    void fetchVendorSection(projectId);
    void fetchVendors();
  }, [projectId, fetchIssues, fetchEvidence, fetchVendorSection, fetchVendors]);

  const sortedIssues = [...issues].sort((a, b) => {
    const aOpen = a.status !== "Resolved" && a.status !== "Closed";
    const bOpen = b.status !== "Resolved" && b.status !== "Closed";
    if (aOpen !== bOpen) return aOpen ? -1 : 1;
    return a.foundDate < b.foundDate ? 1 : -1;
  });

  return (
    <div className="flex flex-col gap-6 sm:gap-8">
      <section>
        <div className="mb-4 flex items-center gap-3 border-b border-border pb-4 sm:mb-6">
          <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-warning-soft text-warning-strong">
            <HeartHandshake className="h-5 w-5 shrink-0" />
          </div>
          <div>
            <h2 className="text-[18px] font-bold text-navy-950 sm:text-[20px]">Kendala yang Sedang Kami Tangani</h2>
            <p className="mt-0.5 text-[13px] text-text-secondary sm:text-[14px]">
              Kami percaya Anda berhak mengetahui setiap kendala dan penanganannya — sekecil apa pun itu.
            </p>
          </div>
        </div>

        {sortedIssues.length === 0 ? (
          <div className="flex flex-col items-center gap-4 rounded-3xl border border-border bg-white p-10 text-center shadow-sm sm:p-16">
            <div className="flex h-20 w-20 items-center justify-center rounded-full bg-emerald-50 text-emerald-500 shadow-sm border border-emerald-100 mb-2">
              <CheckCircle2 className="h-10 w-10" />
            </div>
            <p className="max-w-md text-[15px] font-bold text-navy-950 sm:text-[16px]">
              Semuanya Berjalan Lancar
            </p>
            <p className="max-w-sm text-[14px] text-text-secondary mt-1">
              Tidak ada kendala yang tercatat saat ini. Tim kami terus memantau persiapan pernikahan Anda.
            </p>
          </div>
        ) : (
          <div className="flex flex-col gap-4 sm:gap-5">
            {sortedIssues.map((issue) => {
              const pv = vendorEngagements.find((v) => v.id === issue.projectVendorId);
              const vendorName = pv ? vendors.find((v) => v.id === pv.vendorId)?.name : undefined;
              return (
                <IssueCard
                  key={issue.id}
                  issue={issue}
                  evidence={evidence.filter((e) => e.relatedKind === "issue" && e.relatedId === issue.id)}
                  onViewEvidence={setViewingEvidence}
                  vendorName={vendorName}
                />
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
