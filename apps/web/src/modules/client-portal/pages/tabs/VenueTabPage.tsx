import { useEffect, useState } from "react";
import { useOutletContext } from "react-router-dom";
import { Building2, MapPin, Users } from "lucide-react";
import { useProjectStore } from "@/modules/projects/stores/useProjectStore";
import type { ProjectVenueSummary } from "@/modules/projects/types";
import { httpClient } from "@/shared/services/http-client";
import { API } from "@/shared/services/api-endpoints";
import type { ClientPortalContext } from "@/modules/client-portal/layouts/ClientPortalLayout";

// Client-facing (ADR-0016) — only ever renders the public-safe summary
// (name/address/city/capacity/facilities/social media/attachment) via GET
// /projects/{id}/venue. Never fetches GET /venues/{id} directly (that's the
// staff-only full record with rental price/charge/PIC contact — see
// ProjectVenueTabPage.tsx, the WO Console equivalent of this tab). There's
// no more photo gallery — at most one image, only fetched/rendered when
// hasVisibleAttachment is true (the field name documents the mime-type gate
// already enforced server-side).
export default function VenueTabPage() {
  const { projectId } = useOutletContext<ClientPortalContext>();
  const fetchProjectVenueSummary = useProjectStore((s) => s.fetchProjectVenueSummary);
  const [venue, setVenue] = useState<ProjectVenueSummary | null>(null);
  const [photoUrl, setPhotoUrl] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;
    let createdUrl: string | null = null;
    setLoading(true);
    void fetchProjectVenueSummary(projectId)
      .then(async (summary) => {
        if (cancelled) return;
        setVenue(summary);
        if (!summary || !summary.hasVisibleAttachment) return;
        try {
          const res = await httpClient.get(API.venues.attachment(summary.id), { responseType: "blob" });
          if (cancelled) return;
          createdUrl = URL.createObjectURL(res.data as Blob);
          setPhotoUrl(createdUrl);
        } catch {
          // Swallow — the summary card still renders fine without a photo.
        }
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
      if (createdUrl) URL.revokeObjectURL(createdUrl);
    };
  }, [projectId, fetchProjectVenueSummary]);

  if (loading) {
    return <p className="py-16 text-center text-sm text-text-secondary animate-pulse">Memuat...</p>;
  }

  return (
    <section>
      <div className="mb-4 flex items-center gap-2 sm:mb-5">
        <Building2 className="h-5 w-5 shrink-0 text-navy-900" />
        <h2 className="text-base font-bold text-text-primary sm:text-lg">Venue Pernikahan Anda</h2>
      </div>

      {!venue ? (
        <div className="rounded-3xl border border-dashed border-border bg-white p-12 text-center shadow-sm">
          <div className="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-surface-muted">
            <Building2 className="h-8 w-8 text-text-secondary opacity-50" />
          </div>
          <p className="text-[14px] font-medium text-text-secondary sm:text-[15px]">Venue untuk pernikahan Anda belum ditentukan.</p>
        </div>
      ) : (
        <div className="overflow-hidden rounded-3xl border border-border bg-white shadow-sm">
          <div className="flex flex-wrap items-start justify-between gap-3 border-b border-border/50 bg-gradient-to-br from-surface-muted/50 to-white p-5 sm:p-6">
            <div>
              <h3 className="text-[16px] font-bold text-navy-950 sm:text-[18px]">{venue.name}</h3>
              <p className="mt-1.5 flex items-center gap-1.5 text-[13px] text-text-secondary sm:text-[13.5px]">
                <MapPin className="h-3.5 w-3.5 shrink-0" /> {[venue.address, venue.city].filter(Boolean).join(", ") || "-"}
              </p>
              {venue.capacity !== null && (
                <p className="mt-1 flex items-center gap-1.5 text-[13px] text-text-secondary sm:text-[13.5px]">
                  <Users className="h-3.5 w-3.5 shrink-0" /> Kapasitas {venue.capacity} orang
                </p>
              )}
            </div>
          </div>

          <div className="flex flex-col gap-5 p-5 sm:p-6">
            {photoUrl && <img src={photoUrl} alt={venue.name} className="w-full rounded-xl object-cover" />}

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
          </div>
        </div>
      )}
    </section>
  );
}
