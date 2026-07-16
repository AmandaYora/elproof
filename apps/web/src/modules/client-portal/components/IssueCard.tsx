import { FileText } from "lucide-react";
import { IssueImpactBadge, IssueStatusBadge } from "@/modules/projects/components/StatusBadges";
import type { VendorIssue, Evidence } from "@/modules/projects/types";
import { formatDate } from "@/shared/lib/formatters";

interface IssueCardProps {
  issue: VendorIssue;
  evidence: Evidence[];
  onViewEvidence: (evidence: Evidence) => void;
  showVendorName?: boolean;
  vendorName?: string;
}

export function IssueCard({ issue, evidence, onViewEvidence, showVendorName = true, vendorName }: IssueCardProps) {
  const isResolved = issue.status === "Resolved" || issue.status === "Closed";

  return (
    <div className="rounded-xl border border-border bg-surface p-4 sm:p-6">
      <div className="flex flex-col gap-2 sm:flex-row sm:items-start sm:justify-between sm:gap-3">
        <div>
          <p className="text-[14px] font-bold text-text-primary sm:text-[15px]">{issue.title}</p>
          <p className="mt-0.5 text-[12px] text-text-secondary sm:text-[12.5px]">
            {showVendorName && `${vendorName ?? "Vendor tidak diketahui"} · `}Ditemukan {formatDate(issue.foundDate)}
          </p>
        </div>
        <div className="flex shrink-0 flex-wrap items-center gap-2">
          <IssueImpactBadge impact={issue.impact} />
          <IssueStatusBadge status={issue.status} />
        </div>
      </div>

      <p className="mt-3 text-[13px] leading-relaxed text-text-secondary sm:mt-4 sm:text-[13.5px]">{issue.description}</p>

      <div className="mt-3 rounded-lg bg-neutral-soft px-3 py-2.5 sm:mt-4 sm:px-4 sm:py-3">
        <p className="text-[12px] font-semibold text-text-primary sm:text-[12.5px]">Rencana penanganan:</p>
        <p className="mt-1 text-[12.5px] leading-relaxed text-text-secondary sm:text-[13px]">{issue.resolutionPlan}</p>
      </div>

      {isResolved && (
        <div className="mt-3 rounded-lg border border-success/30 bg-success-soft px-3 py-2.5 sm:px-4 sm:py-3">
          <p className="text-[12px] font-semibold text-success sm:text-[12.5px]">
            Sudah diselesaikan pada {formatDate(issue.resolvedDate)}
          </p>
          {issue.resolutionNotes && (
            <p className="mt-1 text-[12.5px] leading-relaxed text-text-secondary sm:text-[13px]">{issue.resolutionNotes}</p>
          )}
        </div>
      )}

      {evidence.length > 0 && (
        <div className="mt-3 flex flex-wrap items-center gap-1.5 border-t border-border-light pt-2.5 sm:mt-4">
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
    </div>
  );
}
