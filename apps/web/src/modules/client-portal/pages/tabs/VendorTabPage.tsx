import { useEffect, useState } from "react";
import { useOutletContext } from "react-router-dom";
import { ChevronDown, Users, FileText, AlertTriangle, Building2, MapPin } from "lucide-react";
import { EngagementStatusBadge, MilestoneStatusBadge } from "@/modules/projects/components/StatusBadges";
import { MilestoneRail } from "@/shared/components/ui/MilestoneRail";
import { EvidenceViewerModal } from "@/shared/components/ui/EvidenceViewerModal";
import { IssueCard } from "@/modules/client-portal/components/IssueCard";
import { useProjectStore } from "@/modules/projects/stores/useProjectStore";
import { useVendorStore } from "@/modules/vendors/stores/useVendorStore";
import { useVendorCategoryStore } from "@/modules/vendor-categories/stores/useVendorCategoryStore";
import type { ProjectVendor, VendorMilestone, VendorIssue, Evidence, ProjectVenueSummary } from "@/modules/projects/types";
import { formatDate } from "@/shared/lib/formatters";
import { httpClient } from "@/shared/services/http-client";
import { API } from "@/shared/services/api-endpoints";
import type { ClientPortalContext } from "@/modules/client-portal/layouts/ClientPortalLayout";

// Venue merged into this same tab (client-facing only — WO Console keeps its
// own separate Vendor/Venue tabs, and the underlying data model is
// untouched). Venue has no engagement/milestone/issue concept at all, so it
// renders as one pinned, full-width card ahead of the vendor grid rather
// than pretending to have progress it doesn't have — see VenueCard below.
export default function VendorTabPage() {
  const { projectId } = useOutletContext<ClientPortalContext>();
  const vendorEngagements = useProjectStore((s) => s.vendorEngagements);
  const vendorMilestones = useProjectStore((s) => s.vendorMilestones);
  const issues = useProjectStore((s) => s.issues);
  const evidence = useProjectStore((s) => s.evidence);
  const fetchVendorSection = useProjectStore((s) => s.fetchVendorSection);
  const fetchIssues = useProjectStore((s) => s.fetchIssues);
  const fetchEvidence = useProjectStore((s) => s.fetchEvidence);
  const fetchProjectVenueSummary = useProjectStore((s) => s.fetchProjectVenueSummary);
  // Public-safe {id, name} only (ADR-0016-style split) — every other vendor
  // field (PIC, phone, email, price, lampiran) is staff-only now, and this
  // tab never needed more than the name anyway (category name comes from
  // the separate `vendor-categories` store below).
  const vendors = useVendorStore((s) => s.vendorSummaries);
  const fetchVendorSummaries = useVendorStore((s) => s.fetchVendorSummaries);
  const categories = useVendorCategoryStore((s) => s.categories);
  const fetchCategories = useVendorCategoryStore((s) => s.fetchCategories);
  const [viewingEvidence, setViewingEvidence] = useState<Evidence | null>(null);

  const [venue, setVenue] = useState<ProjectVenueSummary | null>(null);
  const [venuePhotoUrl, setVenuePhotoUrl] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  // One combined loading gate for venue + vendor data -- waiting on both
  // avoids flashing "belum ada venue/vendor" while either fetch is still in
  // flight (the pre-merge Vendor tab had this exact gap: it rendered
  // immediately off an empty store).
  useEffect(() => {
    let cancelled = false;
    let createdUrl: string | null = null;
    setLoading(true);

    const venueLoad = fetchProjectVenueSummary(projectId).then(async (summary) => {
      if (cancelled) return;
      setVenue(summary);
      if (!summary || !summary.hasVisibleAttachment) return;
      try {
        const res = await httpClient.get(API.venues.attachment(summary.id), { responseType: "blob" });
        if (cancelled) return;
        createdUrl = URL.createObjectURL(res.data as Blob);
        setVenuePhotoUrl(createdUrl);
      } catch {
        // Swallow -- the venue card still renders fine without a photo.
      }
    });

    void Promise.all([
      venueLoad,
      fetchVendorSection(projectId),
      fetchIssues(projectId),
      fetchEvidence(projectId),
      fetchVendorSummaries(),
      fetchCategories(),
    ]).finally(() => {
      if (!cancelled) setLoading(false);
    });

    return () => {
      cancelled = true;
      if (createdUrl) URL.revokeObjectURL(createdUrl);
    };
  }, [projectId, fetchProjectVenueSummary, fetchVendorSection, fetchIssues, fetchEvidence, fetchVendorSummaries, fetchCategories]);

  const isEmpty = !venue && vendorEngagements.length === 0;

  return (
    <div className="flex flex-col gap-6 sm:gap-8">
      <section>
        <div className="mb-4 flex items-center gap-2 sm:mb-5">
          <Users className="h-5 w-5 shrink-0 text-navy-900" />
          <h2 className="text-base font-bold text-text-primary sm:text-lg">Venue &amp; Vendor yang Mempersiapkan Hari Bahagia Anda</h2>
        </div>
        <p className="mb-5 max-w-2xl text-[13px] leading-relaxed text-text-secondary sm:mb-6 sm:text-[13.5px]">
          Venue dan seluruh vendor yang bekerja untuk mewujudkan hari bahagia Anda, beserta progress tahapan kerja
          masing-masing yang benar-benar telah diselesaikan oleh tim kami — lengkap dengan dokumen pendukungnya.
        </p>

        {loading ? (
          <p className="py-16 text-center text-sm text-text-secondary animate-pulse">Memuat...</p>
        ) : isEmpty ? (
          <div className="rounded-3xl border border-dashed border-border bg-white p-12 text-center shadow-sm">
            <div className="mx-auto flex h-16 w-16 items-center justify-center rounded-full bg-surface-muted mb-4">
              <Users className="h-8 w-8 text-text-secondary opacity-50" />
            </div>
            <p className="text-[14px] text-text-secondary sm:text-[15px] font-medium">
              Venue dan vendor untuk pernikahan Anda belum ditentukan.
            </p>
          </div>
        ) : (
          <div className="grid grid-cols-1 gap-5 md:grid-cols-2">
            {venue ? (
              <VenueCard venue={venue} photoUrl={venuePhotoUrl} />
            ) : (
              vendorEngagements.length > 0 && <VenuePlaceholderCard />
            )}

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

// Pinned ahead of the vendor grid, spanning both columns -- venue has no
// engagement status, milestones, or issues in the data model, so this
// deliberately doesn't reuse VendorCard's progress rail/status badge/
// accordion slots, only its outer card shell (rounded-3xl border shadow-sm)
// so the two read as one visual family. "Venue Pernikahan" fills the
// pill-badge slot a vendor card would use for its category.
function VenueCard({ venue, photoUrl }: { venue: ProjectVenueSummary; photoUrl: string | null }) {
  const hasExtra = Boolean(photoUrl || venue.facilities || venue.socialMedia);
  return (
    <div className="flex flex-col overflow-hidden rounded-3xl border border-border bg-white shadow-sm transition-all hover:shadow-xl hover:shadow-navy-900/5 md:col-span-2 sm:flex-row">
      <div className="flex shrink-0 flex-col gap-3 border-b border-border/50 bg-gradient-to-br from-surface-muted/50 to-white p-5 sm:w-[320px] sm:border-b-0 sm:border-r sm:p-6">
        <div>
          <h3 className="text-[16px] font-bold text-navy-950 sm:text-[18px]">{venue.name}</h3>
          <p className="mt-1.5 inline-flex items-center gap-1.5 rounded-full border border-border bg-white px-2.5 py-0.5 text-[13px] font-medium text-text-secondary sm:text-[13.5px]">
            <span className="h-1.5 w-1.5 rounded-full bg-navy-900" /> Venue Pernikahan
          </p>
        </div>
        <div className="flex flex-col gap-1.5">
          <p className="flex items-start gap-1.5 text-[13px] text-text-secondary sm:text-[13.5px]">
            <MapPin className="h-3.5 w-3.5 shrink-0 translate-y-0.5" /> {[venue.address, venue.city].filter(Boolean).join(", ") || "-"}
          </p>
          {venue.capacity !== null && (
            <p className="flex items-center gap-1.5 text-[13px] text-text-secondary sm:text-[13.5px]">
              <Users className="h-3.5 w-3.5 shrink-0" /> Kapasitas {venue.capacity} orang
            </p>
          )}
        </div>
      </div>

      <div className="flex flex-1 flex-col gap-5 p-5 sm:p-6">
        {photoUrl && <img src={photoUrl} alt={venue.name} className="max-h-56 w-full rounded-xl object-cover" />}

        {venue.facilities && (
          <div>
            <p className="mb-1.5 text-[12px] font-semibold text-text-secondary">Fasilitas</p>
            <p className="whitespace-pre-line text-[13px] text-text-primary">{venue.facilities}</p>
          </div>
        )}

        {venue.socialMedia && (
          <div>
            <p className="mb-1.5 text-[12px] font-semibold text-text-secondary">Sosial Media</p>
            <p className="whitespace-pre-line text-[13px] text-text-primary">{venue.socialMedia}</p>
          </div>
        )}

        {!hasExtra && <p className="text-[13px] text-text-secondary">Tidak ada informasi tambahan untuk venue ini.</p>}
      </div>
    </div>
  );
}

// Shown in the venue's pinned slot only when at least one vendor already
// exists but no venue has been picked yet -- a visible "not decided yet"
// placeholder, not a silently missing card, so the client isn't left
// wondering whether it was simply forgotten.
function VenuePlaceholderCard() {
  return (
    <div className="rounded-3xl border border-dashed border-border bg-white p-8 text-center shadow-sm md:col-span-2">
      <div className="mx-auto mb-3 flex h-12 w-12 items-center justify-center rounded-full bg-surface-muted">
        <Building2 className="h-6 w-6 text-text-secondary opacity-50" />
      </div>
      <p className="text-[13.5px] font-medium text-text-secondary">Venue untuk pernikahan Anda belum ditentukan.</p>
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
                  milestoneName={milestones.find((m) => m.id === issue.vendorMilestoneId)?.name}
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
