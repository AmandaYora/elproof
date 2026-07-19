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
          <div className="rounded-3xl border border-dashed border-border bg-white p-12 text-center shadow-sm">
            <div className="mx-auto flex h-16 w-16 items-center justify-center rounded-full bg-surface-muted mb-4">
              <Users className="h-8 w-8 text-text-secondary opacity-50" />
            </div>
            <p className="text-[14px] text-text-secondary sm:text-[15px] font-medium">
              Belum ada vendor yang tercatat untuk pernikahan Anda.
            </p>
          </div>
        ) : (
          <div className="grid grid-cols-1 gap-5 md:grid-cols-2">
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
    <div className="flex flex-col overflow-hidden rounded-3xl border border-border bg-white shadow-sm transition-all hover:shadow-xl hover:shadow-navy-900/5 hover:-translate-y-1">
      <div className="flex flex-wrap items-start justify-between gap-3 bg-gradient-to-br from-surface-muted/50 to-white p-5 sm:p-6 border-b border-border/50">
        <div>
          <h3 className="text-[16px] font-bold text-navy-950 sm:text-[18px]">{vendorName}</h3>
          <p className="mt-1 text-[13px] font-medium text-text-secondary sm:text-[13.5px] inline-flex items-center gap-1.5 rounded-full bg-white px-2.5 py-0.5 border border-border">
            <span className="w-1.5 h-1.5 rounded-full bg-blue-400"></span> {categoryName} <span className="text-border mx-1">|</span> {projectVendor.scope}
          </p>
        </div>
        <EngagementStatusBadge status={projectVendor.engagementStatus} />
      </div>

      <div className="flex flex-col p-5 sm:p-6">
        {openIssueCount > 0 && (
          <div className="mb-5 flex items-center gap-2 rounded-xl border border-warning/20 bg-warning-soft/50 px-3.5 py-2.5 text-[13px] font-semibold text-warning-strong shadow-sm">
            <AlertTriangle className="h-4 w-4 shrink-0" />
            <span>{openIssueCount} kendala aktif sedang kami tangani</span>
          </div>
        )}

        <div>
          {milestones.length === 0 ? (
            <p className="text-[13px] text-text-secondary sm:text-[13.5px] text-center bg-surface py-4 rounded-xl border border-dashed border-border">Belum ada progress yang tercatat untuk vendor ini.</p>
          ) : (
            <div className="rounded-2xl border border-border bg-surface/30 p-4">
              <MilestoneRail milestones={relevantMilestones} size="md" showCount={false} />
              <p className="mt-3 text-[13px] font-medium text-navy-900 text-center">
                {completedCount} dari {relevantMilestones.length} tahapan selesai
              </p>
            </div>
          )}
        </div>

        {milestones.length > 0 && (
          <details className="group mt-5 rounded-2xl border border-border bg-white shadow-sm overflow-hidden [&_summary::-webkit-details-marker]:hidden">
            <summary className="flex cursor-pointer items-center justify-between bg-surface-muted/30 px-4 py-3 text-[13.5px] font-bold text-navy-950 hover:bg-surface-muted/60 transition-colors">
              Lihat rincian tahapan
              <span className="flex h-6 w-6 items-center justify-center rounded-full bg-white border border-border text-navy-900 transition-transform group-open:rotate-180">
                <ChevronDown className="h-4 w-4 shrink-0" />
              </span>
            </summary>
            <ul className="flex flex-col gap-3 p-4 bg-surface-muted/10 border-t border-border">
              {milestones.map((m) => (
                <MilestoneRow key={m.id} milestone={m} evidence={evidence.filter((e) => e.relatedKind === "vendorMilestone" && e.relatedId === m.id)} onViewEvidence={onViewEvidence} />
              ))}
            </ul>
          </details>
        )}

        {sortedIssues.length > 0 && (
          <details className="group mt-3 rounded-2xl border border-warning/30 bg-white shadow-sm overflow-hidden [&_summary::-webkit-details-marker]:hidden">
            <summary className="flex cursor-pointer items-center justify-between bg-warning-soft/30 px-4 py-3 text-[13.5px] font-bold text-warning-strong hover:bg-warning-soft/60 transition-colors">
              Lihat detail kendala ({sortedIssues.length})
              <span className="flex h-6 w-6 items-center justify-center rounded-full bg-white border border-warning/30 text-warning-strong transition-transform group-open:rotate-180">
                <ChevronDown className="h-4 w-4 shrink-0" />
              </span>
            </summary>
            <div className="flex flex-col gap-4 p-4 border-t border-warning/20 bg-warning-soft/10">
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
    <li className="flex flex-col gap-2.5 rounded-xl border border-border bg-white px-4 py-3.5 shadow-sm transition-colors hover:border-navy-900/20">
      <div className="flex flex-col items-start gap-2 sm:flex-row sm:flex-wrap sm:items-center sm:justify-between">
        <span className="text-[13.5px] font-semibold text-navy-950">{m.name}</span>
        <span className="flex items-center gap-2">
          <MilestoneStatusBadge status={m.status} />
          <span className="whitespace-nowrap text-[12px] font-medium text-text-secondary bg-surface-muted px-2 py-1 rounded-md">
            {m.status === "Completed" ? `Selesai ${formatDate(m.completedDate)}` : `Target ${formatDate(m.targetDate)}`}
          </span>
        </span>
      </div>
      {evidence.length > 0 && (
        <div className="flex flex-wrap items-center gap-2 mt-1">
          <FileText className="h-3.5 w-3.5 shrink-0 text-text-tertiary" />
          {evidence.map((e) => (
            <button
              key={e.id}
              onClick={() => onViewEvidence(e)}
              className="group flex items-center gap-1.5 rounded-lg border border-border bg-surface px-2.5 py-1 text-[11.5px] font-semibold text-navy-900 hover:bg-navy-50 hover:border-navy-200 transition-colors"
            >
              Lihat {e.type}
            </button>
          ))}
        </div>
      )}
    </li>
  );
}
