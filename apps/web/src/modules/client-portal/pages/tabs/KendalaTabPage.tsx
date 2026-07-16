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
        <div className="mb-4 flex items-center gap-2 sm:mb-5">
          <HeartHandshake className="h-5 w-5 shrink-0 text-navy-900" />
          <h2 className="text-base font-bold text-text-primary sm:text-lg">Kendala yang Sedang Kami Tangani</h2>
        </div>
        <p className="mb-5 max-w-2xl text-[13px] leading-relaxed text-text-secondary sm:mb-6 sm:text-[13.5px]">
          Kami percaya Anda berhak mengetahui setiap kendala dan bagaimana kami menanganinya — sekecil apa pun itu.
        </p>

        {sortedIssues.length === 0 ? (
          <div className="flex flex-col items-center gap-3 rounded-xl border border-border bg-surface p-6 text-center sm:p-10">
            <CheckCircle2 className="h-8 w-8 text-success" />
            <p className="max-w-md text-[13.5px] font-medium text-text-primary sm:text-[14px]">
              Tidak ada kendala yang tercatat saat ini — persiapan pernikahan Anda berjalan lancar.
            </p>
          </div>
        ) : (
          <div className="flex flex-col gap-3 sm:gap-4">
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
