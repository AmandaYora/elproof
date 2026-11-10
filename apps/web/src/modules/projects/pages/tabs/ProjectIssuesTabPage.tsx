import { useEffect, useState } from "react";
import { Link, useOutletContext } from "react-router-dom";
import { CircleCheck, Eye, EyeOff, Plus } from "lucide-react";
import { Card, CardHeader, CardContent } from "@/shared/components/ui/Card";
import { Button } from "@/shared/components/ui/Button";
import { Badge } from "@/shared/components/ui/Badge";
import { Select } from "@/shared/components/ui/Input";
import { EmptyState } from "@/shared/components/feedback/EmptyState";
import { IssueImpactBadge } from "@/modules/projects/components/StatusBadges";
import { IssueFormModal } from "@/modules/projects/components/IssueFormModal";
import type { IssueFormValues } from "@/modules/projects/schemas/issue.schema";
import { ISSUE_STATUS_OPTIONS } from "@/modules/projects/schemas/issue.schema";
import { useProjectStore, toIssueUpdateFields } from "@/modules/projects/stores/useProjectStore";
import { useStaffStore } from "@/modules/users/stores/useStaffStore";
import { useAuthStore } from "@/shared/stores/useAuthStore";
import { useCan } from "@/shared/stores/usePermissionStore";
import { ISSUE_STATUS_LABELS } from "@/shared/lib/labels";
import { toast } from "@/shared/stores/useToastStore";
import { getApiErrorMessage } from "@/shared/lib/api-error";
import { formatDate } from "@/shared/lib/formatters";
import { todayISO } from "@/modules/projects/lib/dates";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import type { IssueStatus, VendorIssue } from "@/modules/projects/types";
import type { ProjectDetailContext } from "@/modules/projects/pages/ProjectDetailLayout";

const OPEN_STATUSES = new Set(["Open", "In Review", "In Resolution"]);

/**
 * Every kendala of the project, open ones first: about a vendor or about the event itself. The status changes in place
 * (no form), and each kendala says whether the client sees it. Two calls: the kendala and the vendors they point to.
 */
export default function ProjectIssuesTabPage() {
  const { projectId } = useOutletContext<ProjectDetailContext>();
  const canEdit = useCan("project.edit");
  const issues = useProjectStore((s) => s.issues);
  const engagements = useProjectStore((s) => s.vendorEngagements);
  const vendorMilestones = useProjectStore((s) => s.vendorMilestones);
  const fetchIssues = useProjectStore((s) => s.fetchIssues);
  const fetchVendorSection = useProjectStore((s) => s.fetchVendorSection);
  const createIssue = useProjectStore((s) => s.createIssue);
  const updateIssue = useProjectStore((s) => s.updateIssue);
  const setIssueStatus = useProjectStore((s) => s.setIssueStatus);
  const staff = useStaffStore((s) => s.staffSummaries);
  const fetchStaff = useStaffStore((s) => s.fetchStaffSummaries);
  const currentStaffId = useAuthStore((s) => s.currentStaffId);
  const [modal, setModal] = useState<{ mode: "create" | "edit"; issueId?: string; version?: string; initialValues: IssueFormValues } | null>(null);

  useEffect(() => {
    void fetchIssues(projectId);
    void fetchVendorSection(projectId);
  }, [projectId, fetchIssues, fetchVendorSection]);

  // The people list is only needed for the PIC picker of the form.
  useEffect(() => {
    if (modal) void fetchStaff();
  }, [modal, fetchStaff]);

  const sorted = [...issues].sort(
    (a, b) => Number(OPEN_STATUSES.has(b.status)) - Number(OPEN_STATUSES.has(a.status)) || b.foundDate.localeCompare(a.foundDate),
  );
  const openCount = sorted.filter((i) => OPEN_STATUSES.has(i.status)).length;
  const vendorLookup = engagements.map((e) => ({ id: e.vendorId, name: e.vendorName || "Vendor", city: null }));

  function openCreate() {
    setModal({
      mode: "create",
      initialValues: {
        projectVendorId: "", vendorMilestoneId: "", title: "", description: "", impact: "Low", foundDate: todayISO(),
        resolutionPlan: "", picStaffId: currentStaffId ?? "", targetResolutionDate: "", status: "Open", isClientVisible: true,
      },
    });
  }

  async function handleSubmit(values: IssueFormValues) {
    if (!modal) return;
    if (modal.mode === "edit" && modal.issueId) await updateIssue(projectId, modal.issueId, values, modal.version);
    else await createIssue(projectId, values);
    setModal(null);
    toast.success(modal.mode === "edit" ? "Kendala diperbarui" : "Kendala dicatat");
  }

  async function changeStatus(issue: VendorIssue, status: IssueStatus) {
    try {
      await setIssueStatus(projectId, issue.id, status);
      toast.success("Status kendala diperbarui");
    } catch (err) {
      toast.error(getApiErrorMessage(err, "Status kendala belum berubah. Coba lagi."));
    }
  }

  return (
    <Card>
      <CardHeader
        title="Kendala"
        subtitle={openCount > 0 ? `${openCount} kendala masih terbuka.` : "Kendala proyek ini, tentang vendor atau tentang acaranya."}
        action={canEdit && <Button size="sm" icon={<Plus className="h-3.5 w-3.5" />} onClick={openCreate}>Tambah kendala</Button>}
      />
      <CardContent>
        {sorted.length === 0 ? (
          <EmptyState
            icon={<CircleCheck className="h-6 w-6" />}
            title="Belum ada kendala"
            description="Catat kendala begitu muncul, tentang vendor atau tentang acaranya. Klien bisa melihatnya dan penanganannya."
          />
        ) : (
          <ul className="flex flex-col divide-y divide-border-light">
            {sorted.map((issue) => (
              <li key={issue.id} className="flex flex-col gap-2 py-4 first:pt-0 last:pb-0">
                <div className="flex flex-wrap items-center gap-2">
                  {canEdit ? (
                    <Select
                      aria-label={`Status kendala ${issue.title}`}
                      value={issue.status}
                      onChange={(e) => void changeStatus(issue, e.target.value as IssueStatus)}
                      className="w-44"
                    >
                      {ISSUE_STATUS_OPTIONS.map((s) => (
                        <option key={s} value={s}>{ISSUE_STATUS_LABELS.staff[s]}</option>
                      ))}
                    </Select>
                  ) : (
                    <Badge tone="neutral">{ISSUE_STATUS_LABELS.staff[issue.status]}</Badge>
                  )}
                  <IssueImpactBadge impact={issue.impact} />
                  <span className="text-[12.5px] text-text-secondary">
                    {issue.vendorName || "Tentang acara"} · ditemukan {formatDate(issue.foundDate)}
                    {issue.picName && ` · PIC ${issue.picName}`}
                  </span>
                  <span className="flex items-center gap-1 text-[12.5px] text-text-secondary">
                    {issue.isClientVisible ? <Eye className="size-3.5" aria-hidden /> : <EyeOff className="size-3.5" aria-hidden />}
                    {issue.isClientVisible ? "Terlihat klien" : "Hanya internal"}
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
                <div className="flex flex-wrap gap-3">
                  {canEdit && (
                    <button
                      type="button"
                      onClick={() => setModal({ mode: "edit", issueId: issue.id, version: issue.updatedAt, initialValues: toIssueUpdateFields(issue) })}
                      className="text-[13px] font-medium text-navy-900 underline-offset-2 hover:underline"
                    >
                      Ubah kendala
                    </button>
                  )}
                  {issue.projectVendorId && (
                    <Link to={ROUTE_PATHS.projectDetail(projectId, "vendor")} className="text-[13px] font-medium text-navy-900 underline-offset-2 hover:underline">
                      Buka di tab Vendor
                    </Link>
                  )}
                </div>
              </li>
            ))}
          </ul>
        )}
      </CardContent>

      {modal && (
        <IssueFormModal
          open
          onClose={() => setModal(null)}
          onSubmit={handleSubmit}
          initialValues={modal.initialValues}
          mode={modal.mode}
          milestoneLocked={false}
          vendorChoice
          vendorEngagements={engagements}
          vendors={vendorLookup}
          vendorMilestones={vendorMilestones}
          staff={staff}
        />
      )}
    </Card>
  );
}
