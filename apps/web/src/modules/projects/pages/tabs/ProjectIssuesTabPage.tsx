import { useEffect } from "react";
import { Link, useOutletContext } from "react-router-dom";
import { CircleCheck } from "lucide-react";
import { Card, CardHeader, CardContent } from "@/shared/components/ui/Card";
import { EmptyState } from "@/shared/components/feedback/EmptyState";
import { IssueImpactBadge, IssueStatusBadge } from "@/modules/projects/components/StatusBadges";
import { useProjectStore } from "@/modules/projects/stores/useProjectStore";
import { useVendorStore } from "@/modules/vendors/stores/useVendorStore";
import { formatDate } from "@/shared/lib/formatters";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import type { ProjectDetailContext } from "@/modules/projects/pages/ProjectDetailLayout";

const OPEN_STATUSES = new Set(["Open", "In Review", "In Resolution"]);

/** Every kendala of the project across all its vendors, open ones first. */
export default function ProjectIssuesTabPage() {
  const { projectId } = useOutletContext<ProjectDetailContext>();
  const issues = useProjectStore((s) => s.issues);
  const engagements = useProjectStore((s) => s.vendorEngagements);
  const fetchIssues = useProjectStore((s) => s.fetchIssues);
  const fetchVendorSection = useProjectStore((s) => s.fetchVendorSection);
  const vendors = useVendorStore((s) => s.vendors);
  const fetchVendors = useVendorStore((s) => s.fetchVendors);

  useEffect(() => {
    void fetchIssues(projectId);
    void fetchVendorSection(projectId);
    void fetchVendors();
  }, [projectId, fetchIssues, fetchVendorSection, fetchVendors]);

  const vendorName = (engagementId: string) => {
    const engagement = engagements.find((e) => e.id === engagementId);
    return vendors.find((v) => v.id === engagement?.vendorId)?.name ?? "Vendor";
  };
  const sorted = [...issues].sort(
    (a, b) => Number(OPEN_STATUSES.has(b.status)) - Number(OPEN_STATUSES.has(a.status)) || b.foundDate.localeCompare(a.foundDate),
  );
  const openCount = sorted.filter((i) => OPEN_STATUSES.has(i.status)).length;

  return (
    <Card>
      <CardHeader
        title="Kendala"
        subtitle={openCount > 0 ? `${openCount} kendala masih terbuka. Ubah atau tambah kendala dari tab Vendor.` : "Semua kendala proyek ini, dari semua vendor."}
      />
      <CardContent>
        {sorted.length === 0 ? (
          <EmptyState
            icon={<CircleCheck className="h-6 w-6" />}
            title="Belum ada kendala"
            description="Kendala dicatat dari tab Vendor, pada vendor atau timeline yang bermasalah."
          />
        ) : (
          <ul className="flex flex-col divide-y divide-border-light">
            {sorted.map((issue) => (
              <li key={issue.id} className="flex flex-col gap-2 py-4 first:pt-0 last:pb-0">
                <div className="flex flex-wrap items-center gap-2">
                  <IssueStatusBadge status={issue.status} />
                  <IssueImpactBadge impact={issue.impact} />
                  <span className="text-[12.5px] text-text-secondary">
                    {vendorName(issue.projectVendorId)} · ditemukan {formatDate(issue.foundDate)}
                  </span>
                </div>
                <p className="text-[14px] font-semibold text-text-primary">{issue.title}</p>
                {issue.description && <p className="max-w-prose text-[13px] text-text-secondary">{issue.description}</p>}
                {issue.resolutionPlan && (
                  <p className="max-w-prose text-[13px] text-text-secondary">
                    <span className="font-medium text-text-primary">Rencana penyelesaian: </span>
                    {issue.resolutionPlan}
                    {issue.targetResolutionDate && ` (target ${formatDate(issue.targetResolutionDate)})`}
                  </p>
                )}
                <Link
                  to={ROUTE_PATHS.projectDetail(projectId, "vendor")}
                  className="w-fit text-[13px] font-medium text-navy-900 underline-offset-2 hover:underline"
                >
                  Buka di tab Vendor
                </Link>
              </li>
            ))}
          </ul>
        )}
      </CardContent>
    </Card>
  );
}
