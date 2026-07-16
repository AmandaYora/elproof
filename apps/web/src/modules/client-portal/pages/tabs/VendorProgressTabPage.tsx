import { useEffect, useState } from "react";
import { useOutletContext } from "react-router-dom";
import { ChevronDown, Users, FileText, AlertTriangle } from "lucide-react";
import { EngagementStatusBadge, MilestoneStatusBadge } from "@/modules/projects/components/StatusBadges";
import { MilestoneRail } from "@/shared/components/ui/MilestoneRail";
import { EvidenceViewerModal } from "@/shared/components/ui/EvidenceViewerModal";
import { IssueCard } from "@/modules/client-portal/components/IssueCard";
import { useProjectStore } from "@/modules/projects/stores/useProjectStore";
import { useVendorStore } from "@/modules/vendors/stores/useVendorStore";
import { useVendorCategoryStore } from "@/modules/vendor-categories/stores/useVendorCategoryStore";
import type { ProjectVendor, VendorMilestone, VendorIssue, Evidence } from "@/modules/projects/types";
import { formatDate } from "@/shared/lib/formatters";
import type { ClientPortalContext } from "@/modules/client-portal/layouts/ClientPortalLayout";

export default function VendorProgressTabPage() {
  const { projectId } = useOutletContext<ClientPortalContext>();
  const vendorEngagements = useProjectStore((s) => s.vendorEngagements);
  const vendorMilestones = useProjectStore((s) => s.vendorMilestones);
  const issues = useProjectStore((s) => s.issues);
  const evidence = useProjectStore((s) => s.evidence);
  const fetchVendorSection = useProjectStore((s) => s.fetchVendorSection);
  const fetchIssues = useProjectStore((s) => s.fetchIssues);
  const fetchEvidence = useProjectStore((s) => s.fetchEvidence);
  const vendors = useVendorStore((s) => s.vendors);
  const fetchVendors = useVendorStore((s) => s.fetchVendors);
  const categories = useVendorCategoryStore((s) => s.categories);
  const fetchCategories = useVendorCategoryStore((s) => s.fetchCategories);
  const [viewingEvidence, setViewingEvidence] = useState<Evidence | null>(null);

  useEffect(() => {
    void fetchVendorSection(projectId);
    void fetchIssues(projectId);
    void fetchEvidence(projectId);
    void fetchVendors();
    void fetchCategories();
  }, [projectId, fetchVendorSection, fetchIssues, fetchEvidence, fetchVendors, fetchCategories]);

  return (
    <div className="flex flex-col gap-6 sm:gap-8">
      <section>
        <div className="mb-4 flex items-center gap-2 sm:mb-5">
          <Users className="h-5 w-5 shrink-0 text-navy-900" />
          <h2 className="text-base font-bold text-text-primary sm:text-lg">Vendor yang Mempersiapkan Hari Bahagia Anda</h2>
        </div>
        <p className="mb-5 max-w-2xl text-[13px] leading-relaxed text-text-secondary sm:mb-6 sm:text-[13.5px]">
          Berikut seluruh vendor yang sedang bekerja untuk mewujudkan hari bahagia Anda, beserta progress tahapan
          kerja masing-masing yang benar-benar telah diselesaikan oleh tim kami — lengkap dengan dokumen
          pendukungnya.
        </p>

        {vendorEngagements.length === 0 ? (
          <p className="rounded-xl border border-dashed border-border bg-surface p-4 text-center text-[13px] text-text-secondary sm:p-6 sm:text-[13.5px]">
            Belum ada vendor yang tercatat untuk pernikahan Anda.
          </p>
        ) : (
          <div className="flex flex-col gap-3 sm:gap-4">
            {vendorEngagements.map((pv) => (
              <VendorCard
                key={pv.id}
                projectVendor={pv}
                vendorName={vendors.find((v) => v.id === pv.vendorId)?.name ?? "Vendor tidak diketahui"}
                categoryName={categories.find((c) => c.id === pv.categoryId)?.name ?? "Kategori tidak diketahui"}
                milestones={vendorMilestones.filter((m) => m.projectVendorId === pv.id)}
                issues={issues.filter((i) => i.projectVendorId === pv.id)}
                evidence={evidence}
                onViewEvidence={setViewingEvidence}
              />
            ))}
          </div>
        )}
      </section>

      {viewingEvidence && (
        <EvidenceViewerModal open onClose={() => setViewingEvidence(null)} projectId={projectId} evidence={viewingEvidence} />
      )}
    </div>
  );
}

function VendorCard({
  projectVendor,
  vendorName,
  categoryName,
  milestones,
  issues,
  evidence,
  onViewEvidence,
}: {
  projectVendor: ProjectVendor;
  vendorName: string;
  categoryName: string;
  milestones: VendorMilestone[];
  issues: VendorIssue[];
  evidence: Evidence[];
  onViewEvidence: (evidence: Evidence) => void;
}) {
  const relevantMilestones = milestones.filter((m) => m.status !== "Cancelled");
  const completedCount = relevantMilestones.filter((m) => m.status === "Completed").length;

  const sortedIssues = [...issues].sort((a, b) => {
    const aOpen = a.status !== "Resolved" && a.status !== "Closed";
    const bOpen = b.status !== "Resolved" && b.status !== "Closed";
    if (aOpen !== bOpen) return aOpen ? -1 : 1;
    return a.foundDate < b.foundDate ? 1 : -1;
  });
  const openIssueCount = issues.filter((i) => i.status !== "Resolved" && i.status !== "Closed").length;

  return (
    <div className="rounded-xl border border-border bg-surface p-4 sm:p-6">
      <div className="flex flex-wrap items-start justify-between gap-2 sm:gap-3">
        <div>
          <p className="text-[14px] font-bold text-text-primary sm:text-[15px]">{vendorName}</p>
          <p className="mt-0.5 text-[12px] text-text-secondary sm:text-[12.5px]">
            {categoryName} · {projectVendor.scope}
          </p>
        </div>
        <EngagementStatusBadge status={projectVendor.engagementStatus} />
      </div>

      {openIssueCount > 0 && (
        <div className="mt-3 flex items-center gap-1.5 rounded-md bg-danger-soft px-2.5 py-1.5 text-[12px] font-semibold text-danger sm:text-[12.5px]">
          <AlertTriangle className="h-3.5 w-3.5 shrink-0" />
          {openIssueCount} kendala aktif sedang kami tangani untuk vendor ini
        </div>
      )}

      <div className="mt-3 sm:mt-4">
        {milestones.length === 0 ? (
          <p className="text-[12.5px] text-text-secondary sm:text-[13px]">Belum ada progress yang tercatat untuk vendor ini.</p>
        ) : (
          <>
            <MilestoneRail milestones={relevantMilestones} size="md" showCount={false} />
            <p className="mt-1.5 text-[12px] font-medium text-text-secondary sm:text-[12.5px]">
              {completedCount} dari {relevantMilestones.length} tahapan selesai
            </p>
          </>
        )}
      </div>

      {milestones.length > 0 && (
        <details className="group mt-3 border-t border-border pt-3 sm:mt-4">
          <summary className="flex cursor-pointer list-none items-center gap-1.5 text-[12px] font-semibold text-navy-900 sm:text-[12.5px] [&::-webkit-details-marker]:hidden">
            <ChevronDown className="h-3.5 w-3.5 shrink-0 transition-transform group-open:rotate-180" />
            Lihat detail tahapan
          </summary>
          <ul className="mt-3 flex flex-col gap-2.5">
            {milestones.map((m) => (
              <MilestoneRow key={m.id} milestone={m} evidence={evidence.filter((e) => e.relatedKind === "vendorMilestone" && e.relatedId === m.id)} onViewEvidence={onViewEvidence} />
            ))}
          </ul>
        </details>
      )}

      {sortedIssues.length > 0 && (
        <details className="group mt-3 border-t border-border pt-3 sm:mt-4">
          <summary className="flex cursor-pointer list-none items-center gap-1.5 text-[12px] font-semibold text-navy-900 sm:text-[12.5px] [&::-webkit-details-marker]:hidden">
            <ChevronDown className="h-3.5 w-3.5 shrink-0 transition-transform group-open:rotate-180" />
            Lihat detail kendala ({sortedIssues.length})
          </summary>
          <div className="mt-3 flex flex-col gap-3">
            {sortedIssues.map((issue) => (
              <IssueCard
                key={issue.id}
                issue={issue}
                evidence={evidence.filter((e) => e.relatedKind === "issue" && e.relatedId === issue.id)}
                onViewEvidence={onViewEvidence}
                showVendorName={false}
                vendorName={vendorName}
              />
            ))}
          </div>
        </details>
      )}
    </div>
  );
}

function MilestoneRow({
  milestone: m,
  evidence,
  onViewEvidence,
}: {
  milestone: VendorMilestone;
  evidence: Evidence[];
  onViewEvidence: (evidence: Evidence) => void;
}) {
  return (
    <li className="flex flex-col gap-1.5 rounded-md bg-surface-muted/60 px-3 py-2.5 text-[12.5px] sm:text-[13px]">
      <div className="flex flex-col items-start gap-1.5 sm:flex-row sm:flex-wrap sm:items-center sm:justify-between sm:gap-2">
        <span className="text-text-primary">{m.name}</span>
        <span className="flex items-center gap-1.5 sm:gap-2">
          <MilestoneStatusBadge status={m.status} />
          <span className="whitespace-nowrap text-[12px] text-text-secondary">
            {m.status === "Completed" ? `Selesai ${formatDate(m.completedDate)}` : `Target ${formatDate(m.targetDate)}`}
          </span>
        </span>
      </div>
      {evidence.length > 0 && (
        <div className="flex flex-wrap items-center gap-1.5">
          <FileText className="h-3 w-3 shrink-0 text-text-secondary" />
          {evidence.map((e) => (
            <button
              key={e.id}
              onClick={() => onViewEvidence(e)}
              className="rounded-full border border-border bg-white px-2.5 py-1 text-[11.5px] font-medium text-navy-900 hover:bg-navy-900/10"
            >
              Lihat {e.type}
            </button>
          ))}
        </div>
      )}
    </li>
  );
}
