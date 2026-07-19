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
    <div className="group relative rounded-2xl border border-border bg-white p-5 sm:p-6 shadow-sm hover:shadow-md transition-shadow overflow-hidden">
      {/* Decorative side bar based on resolution status */}
      <div className={`absolute left-0 top-0 bottom-0 w-1.5 transition-opacity opacity-70 group-hover:opacity-100 ${isResolved ? 'bg-emerald-500' : 'bg-warning-strong'}`}></div>

      <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between sm:gap-4 border-b border-border/50 pb-4">
        <div>
          <p className="text-[15px] font-bold text-navy-950 sm:text-[16px]">{issue.title}</p>
          <p className="mt-1 text-[12.5px] font-medium text-text-secondary sm:text-[13px] bg-surface-muted inline-block px-2.5 py-1 rounded-md">
            {showVendorName && <span className="text-navy-700 font-semibold">{vendorName ?? "Vendor tidak diketahui"}</span>}
            {showVendorName && <span className="mx-1.5 text-border">|</span>}
            Ditemukan {formatDate(issue.foundDate)}
          </p>
        </div>
        <div className="flex shrink-0 flex-wrap items-center gap-2">
          <IssueImpactBadge impact={issue.impact} />
          <IssueStatusBadge status={issue.status} />
        </div>
      </div>

      <div className="mt-4">
        <p className="text-[13.5px] leading-relaxed text-text-secondary sm:text-[14px]">{issue.description}</p>
      </div>

      <div className="mt-4 rounded-xl border border-blue-100 bg-blue-50/40 p-4 relative overflow-hidden">
        <div className="absolute top-0 left-0 bottom-0 w-1 bg-blue-400 opacity-50"></div>
        <p className="text-[12.5px] font-bold text-navy-900 uppercase tracking-wider">Rencana penanganan</p>
        <p className="mt-1.5 text-[13.5px] leading-relaxed text-navy-900/80">{issue.resolutionPlan}</p>
      </div>

      {isResolved && (
        <div className="mt-3 rounded-xl border border-emerald-100 bg-emerald-50/40 p-4 relative overflow-hidden">
          <div className="absolute top-0 left-0 bottom-0 w-1 bg-emerald-400 opacity-50"></div>
          <p className="text-[12.5px] font-bold text-emerald-800 uppercase tracking-wider">
            Diselesaikan {formatDate(issue.resolvedDate)}
          </p>
          {issue.resolutionNotes && (
            <p className="mt-1.5 text-[13.5px] leading-relaxed text-emerald-900/80">{issue.resolutionNotes}</p>
          )}
        </div>
      )}

      {evidence.length > 0 && (
        <div className="mt-5 flex flex-wrap items-center gap-2 border-t border-border/50 pt-4">
          <FileText className="h-4 w-4 shrink-0 text-text-tertiary" />
          <span className="text-[13px] font-medium text-text-secondary mr-2">Dokumen Pendukung:</span>
          {evidence.map((e) => (
            <button
              key={e.id}
              onClick={() => onViewEvidence(e)}
              className="flex items-center gap-1.5 rounded-lg border border-border bg-surface px-3 py-1.5 text-[12px] font-semibold text-navy-900 hover:bg-navy-50 hover:border-navy-200 transition-colors shadow-sm"
            >
              Lihat {e.type}
            </button>
          ))}
        </div>
      )}
    </div>
  );
}
