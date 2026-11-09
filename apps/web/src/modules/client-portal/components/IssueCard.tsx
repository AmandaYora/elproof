import { Button } from "@/shared/components/kit/button";
import { IssueImpactBadge, IssueStatusBadge } from "@/modules/projects/components/StatusBadges";
import { formatDate } from "@/shared/lib/formatters";
import type { VendorIssue } from "@/modules/projects/types";
import type { ProofToOpen } from "@/modules/client-portal/components/useProofViewer";

/**
 * One kendala, always answering the same three questions (FLOW-IDEAL 3.2): what
 * happened, what we are doing, when to expect news. Calm colors, never red.
 * Names come with the issue (E6), so the card needs no vendor lists.
 */
export function IssueCard({ issue, onOpenProof, showVendorName = true }: { issue: VendorIssue; onOpenProof: (proof: ProofToOpen, title: string) => void; showVendorName?: boolean }) {
  const resolved = issue.status === "Resolved" || issue.status === "Closed";
  const where = [showVendorName ? issue.vendorName : "", issue.vendorMilestoneName || "Kendala umum"].filter(Boolean).join(" · ");

  return (
    <article className="rounded-2xl border border-border bg-card p-5">
      <div className="flex flex-col gap-3 border-b pb-4 sm:flex-row sm:items-start sm:justify-between">
        <div className="min-w-0">
          <h3 className="text-[15px] font-semibold text-foreground">{issue.title}</h3>
          <p className="mt-1 text-[13px] text-muted-foreground">
            {where} · Ditemukan {formatDate(issue.foundDate)}
          </p>
        </div>
        <div className="flex shrink-0 flex-wrap items-center gap-2">
          <IssueImpactBadge impact={issue.impact} audience="client" />
          <IssueStatusBadge status={issue.status} audience="client" />
        </div>
      </div>

      <dl className="mt-4 flex flex-col gap-4">
        <div>
          <dt className="text-[13px] font-semibold text-foreground">Apa yang terjadi</dt>
          <dd className="mt-1 text-[14px] leading-relaxed text-muted-foreground">{issue.description || "Kami sedang memeriksa hal ini."}</dd>
        </div>
        <div className="rounded-xl bg-info-soft p-4">
          <dt className="text-[13px] font-semibold text-info-text">Yang sedang kami lakukan</dt>
          <dd className="mt-1 text-[14px] leading-relaxed text-foreground">
            {issue.resolutionPlan || "Tim kami sedang menyusun langkah penanganan dan akan memberi tahu Anda."}
          </dd>
        </div>
        {!resolved && (
          <div>
            <dt className="text-[13px] font-semibold text-foreground">Kabar berikutnya</dt>
            <dd className="mt-1 text-[14px] text-muted-foreground">
              {issue.targetResolutionDate ? `Kami targetkan selesai sekitar ${formatDate(issue.targetResolutionDate)}.` : "Kami akan mengabari Anda begitu ada perkembangan."}
            </dd>
          </div>
        )}
      </dl>

      {resolved && (
        <div className="mt-4 rounded-xl bg-success-soft p-4">
          <p className="text-[13px] font-semibold text-success-text">Diselesaikan {formatDate(issue.resolvedDate)}</p>
          {issue.resolutionNotes && <p className="mt-1 text-[14px] leading-relaxed text-foreground">{issue.resolutionNotes}</p>}
        </div>
      )}

      {issue.evidence.length > 0 && (
        <div className="mt-4 flex flex-wrap items-center gap-2 border-t pt-4">
          {issue.evidence.map((e) => (
            <Button key={e.id} variant="outline" size="sm" onClick={() => onOpenProof(e, issue.title)}>
              Lihat {e.type.toLowerCase()}
            </Button>
          ))}
        </div>
      )}
    </article>
  );
}
