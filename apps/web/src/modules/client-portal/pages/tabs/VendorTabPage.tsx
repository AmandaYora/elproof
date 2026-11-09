import { useEffect, useState } from "react";
import { useOutletContext } from "react-router-dom";
import { AlertTriangle, Building2, MapPin, Users } from "lucide-react";
import { EngagementStatusBadge } from "@/modules/projects/components/StatusBadges";
import { LoadError } from "@/shared/components/feedback/LoadError";
import { SkeletonRows } from "@/shared/components/ui/Skeleton";
import { IssueCard } from "@/modules/client-portal/components/IssueCard";
import { MilestoneRail } from "@/modules/client-portal/components/MilestoneRail";
import { useProofViewer, type ProofToOpen } from "@/modules/client-portal/components/useProofViewer";
import { useLoad } from "@/modules/client-portal/lib/useLoad";
import { useProjectStore } from "@/modules/projects/stores/useProjectStore";
import type { ProjectVendor, ProjectVenueSummary, VendorIssue, VendorMilestone } from "@/modules/projects/types";
import { httpClient } from "@/shared/services/http-client";
import { API } from "@/shared/services/api-endpoints";
import type { ClientPortalContext } from "@/modules/client-portal/layouts/ClientPortalLayout";

const isOpenIssue = (i: VendorIssue) => i.status !== "Resolved" && i.status !== "Closed";

/**
 * Venue and the vendors preparing the day, each with its steps and proof.
 * Vendor and category names arrive with the vendors (E6): this page asks for
 * the vendors, the issues and the venue, and never for the tenant's vendor list.
 */
export default function VendorTabPage() {
  const { projectId } = useOutletContext<ClientPortalContext>();
  const engagements = useProjectStore((s) => s.vendorEngagements);
  const vendorMilestones = useProjectStore((s) => s.vendorMilestones);
  const issues = useProjectStore((s) => s.issues);
  const fetchVendorSection = useProjectStore((s) => s.fetchVendorSection);
  const fetchIssues = useProjectStore((s) => s.fetchIssues);
  const fetchProjectVenueSummary = useProjectStore((s) => s.fetchProjectVenueSummary);
  const viewer = useProofViewer(projectId);
  const [venue, setVenue] = useState<ProjectVenueSummary | null>(null);
  const [venuePhotoUrl, setVenuePhotoUrl] = useState<string | null>(null);

  const { status, retry } = useLoad(async () => {
    const [summary] = await Promise.all([fetchProjectVenueSummary(projectId), fetchVendorSection(projectId), fetchIssues(projectId)]);
    setVenue(summary);
  }, [projectId, fetchProjectVenueSummary, fetchVendorSection, fetchIssues]);

  // The venue photo is a nicety: fetched after the page is up, never blocks it.
  useEffect(() => {
    if (!venue?.hasVisibleAttachment) return;
    let cancelled = false;
    let createdUrl: string | null = null;
    httpClient
      .get(API.venues.attachment(venue.id), { responseType: "blob" })
      .then((res) => {
        if (cancelled) return;
        createdUrl = URL.createObjectURL(res.data as Blob);
        setVenuePhotoUrl(createdUrl);
      })
      .catch(() => undefined);
    return () => {
      cancelled = true;
      if (createdUrl) URL.revokeObjectURL(createdUrl);
    };
  }, [venue]);

  const empty = !venue && engagements.length === 0;

  return (
    <div className="mx-auto flex w-full max-w-3xl flex-col gap-6">
      <header>
        <h1 className="text-xl font-semibold text-foreground">Vendor dan venue</h1>
        <p className="mt-1.5 max-w-prose text-[14px] leading-relaxed text-muted-foreground">
          Venue dan vendor yang menyiapkan hari bahagia Anda, beserta tahapan kerja yang sudah selesai dan buktinya.
        </p>
      </header>

      {status === "loading" && <div className="rounded-2xl border border-border bg-card p-5"><SkeletonRows rows={4} label="Memuat vendor" /></div>}
      {status === "error" && <LoadError message="Vendor belum bisa dimuat. Periksa koneksi, lalu coba lagi." onRetry={retry} />}

      {status === "ready" && empty && (
        <div className="flex flex-col items-center gap-3 rounded-2xl border border-dashed border-border bg-card p-10 text-center">
          <Users className="size-8 text-muted-foreground" aria-hidden />
          <p className="text-[14px] text-muted-foreground">Venue dan vendor untuk pernikahan Anda belum ditentukan.</p>
        </div>
      )}

      {status === "ready" && !empty && (
        <div className="flex flex-col gap-4">
          {venue ? <VenueCard venue={venue} photoUrl={venuePhotoUrl} /> : <VenuePlaceholder />}
          {engagements.map((pv) => (
            <VendorCard
              key={pv.id}
              vendor={pv}
              steps={vendorMilestones.filter((m) => m.projectVendorId === pv.id && m.status !== "Cancelled")}
              issues={issues.filter((i) => i.projectVendorId === pv.id)}
              onOpenProof={viewer.open}
            />
          ))}
        </div>
      )}
      {viewer.element}
    </div>
  );
}

function VenueCard({ venue, photoUrl }: { venue: ProjectVenueSummary; photoUrl: string | null }) {
  const place = [venue.address, venue.city].filter(Boolean).join(", ");
  return (
    <article className="overflow-hidden rounded-2xl border border-border bg-card">
      {photoUrl && <img src={photoUrl} alt={venue.name} className="max-h-56 w-full object-cover" />}
      <div className="flex flex-col gap-3 p-5">
        <div>
          <p className="text-[12.5px] font-medium text-muted-foreground">Venue</p>
          <h2 className="text-[17px] font-semibold text-foreground">{venue.name}</h2>
        </div>
        {place && (
          <p className="flex items-start gap-2 text-[13.5px] text-muted-foreground">
            <MapPin className="mt-0.5 size-4 shrink-0" aria-hidden /> {place}
          </p>
        )}
        {venue.capacity !== null && (
          <p className="flex items-center gap-2 text-[13.5px] text-muted-foreground">
            <Users className="size-4 shrink-0" aria-hidden /> Kapasitas {venue.capacity} orang
          </p>
        )}
        {venue.facilities && (
          <div>
            <p className="text-[12.5px] font-semibold text-foreground">Fasilitas</p>
            <p className="mt-0.5 text-[13.5px] whitespace-pre-line text-muted-foreground">{venue.facilities}</p>
          </div>
        )}
        {venue.socialMedia && (
          <div>
            <p className="text-[12.5px] font-semibold text-foreground">Sosial media</p>
            <p className="mt-0.5 text-[13.5px] whitespace-pre-line text-muted-foreground">{venue.socialMedia}</p>
          </div>
        )}
      </div>
    </article>
  );
}

// Shown when vendors exist but no venue is chosen yet, so a missing card is
// never mistaken for something forgotten.
function VenuePlaceholder() {
  return (
    <div className="flex items-center gap-3 rounded-2xl border border-dashed border-border bg-card p-5 text-[13.5px] text-muted-foreground">
      <Building2 className="size-5 shrink-0" aria-hidden /> Venue untuk pernikahan Anda belum ditentukan.
    </div>
  );
}

function VendorCard({
  vendor,
  steps,
  issues,
  onOpenProof,
}: {
  vendor: ProjectVendor;
  steps: VendorMilestone[];
  issues: VendorIssue[];
  onOpenProof: (proof: ProofToOpen, title: string) => void;
}) {
  const doneCount = steps.filter((m) => m.status === "Completed").length;
  const openCount = issues.filter(isOpenIssue).length;
  const ordered = [...steps].sort((a, b) => a.order - b.order);

  return (
    <article className="overflow-hidden rounded-2xl border border-border bg-card">
      <div className="flex flex-wrap items-start justify-between gap-3 border-b p-5">
        <div className="min-w-0">
          <h2 className="text-[16px] font-semibold text-foreground">{vendor.vendorName || "Vendor"}</h2>
          <p className="mt-0.5 text-[13px] text-muted-foreground">{[vendor.categoryName, vendor.scope].filter(Boolean).join(" · ")}</p>
        </div>
        <EngagementStatusBadge status={vendor.engagementStatus} audience="client" />
      </div>

      <div className="flex flex-col gap-4 p-5">
        {openCount > 0 && (
          <p className="flex items-center gap-2 rounded-xl bg-warning-soft px-3.5 py-2.5 text-[13px] font-medium text-warning-text">
            <AlertTriangle className="size-4 shrink-0" aria-hidden /> {openCount} kendala sedang kami tangani
          </p>
        )}

        {ordered.length === 0 ? (
          <p className="text-[13.5px] text-muted-foreground">Belum ada tahapan yang tercatat untuk vendor ini.</p>
        ) : (
          <details className="group">
            <summary className="flex min-h-11 cursor-pointer items-center justify-between rounded-lg text-[14px] font-semibold text-foreground focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-foreground">
              <span>
                {doneCount} dari {ordered.length} tahapan selesai
              </span>
              <span className="text-[13px] font-medium text-muted-foreground group-open:hidden">Lihat tahapan</span>
              <span className="hidden text-[13px] font-medium text-muted-foreground group-open:inline">Sembunyikan</span>
            </summary>
            <div className="mt-3">
              <MilestoneRail steps={ordered} onOpenProof={onOpenProof} />
            </div>
          </details>
        )}

        {issues.length > 0 && (
          <div className="flex flex-col gap-3">
            {[...issues].sort((a, b) => Number(isOpenIssue(b)) - Number(isOpenIssue(a)) || b.foundDate.localeCompare(a.foundDate)).map((i) => (
              <IssueCard key={i.id} issue={i} showVendorName={false} onOpenProof={onOpenProof} />
            ))}
          </div>
        )}
      </div>
    </article>
  );
}
