import { useEffect, useState } from "react";
import { Plus } from "lucide-react";
import { Card, CardHeader, CardContent } from "@/shared/components/ui/Card";
import { Button } from "@/shared/components/ui/Button";
import { Modal } from "@/shared/components/ui/Modal";
import { Input, Textarea, Select, Field } from "@/shared/components/ui/Input";
import { Table, THead, TBody, TR, TH, TD } from "@/shared/components/ui/Table";
import { CardList, CardListField } from "@/shared/components/ui/CardList";
import { Pagination } from "@/shared/components/ui/Pagination";
import { usePagination } from "@/shared/hooks/usePagination";
import { IssueImpactBadge } from "@/modules/projects/components/StatusBadges";
import { useProjectStore } from "@/modules/projects/stores/useProjectStore";
import { useVendorStore } from "@/modules/vendors/stores/useVendorStore";
import { useStaffStore } from "@/modules/users/stores/useStaffStore";
import {
  issueSchema,
  ISSUE_IMPACT_OPTIONS,
  ISSUE_STATUS_OPTIONS,
  type IssueFormValues,
} from "@/modules/projects/schemas/issue.schema";
import type { IssueImpact, IssueStatus, ProjectVendor } from "@/modules/projects/types";
import { todayISO } from "@/modules/projects/lib/dates";
import { getApiErrorMessage } from "@/shared/lib/api-error";
import { formatDate } from "@/shared/lib/formatters";

export function ProjectIssuesSection({ projectId }: { projectId: string }) {
  const issues = useProjectStore((s) => s.issues);
  const vendorEngagements = useProjectStore((s) => s.vendorEngagements);
  const fetchIssues = useProjectStore((s) => s.fetchIssues);
  const fetchVendorSection = useProjectStore((s) => s.fetchVendorSection);
  const createIssue = useProjectStore((s) => s.createIssue);
  const updateIssueStatus = useProjectStore((s) => s.updateIssueStatus);
  const vendors = useVendorStore((s) => s.vendors);
  const fetchVendors = useVendorStore((s) => s.fetchVendors);
  const staff = useStaffStore((s) => s.staff);
  const fetchStaff = useStaffStore((s) => s.fetchStaff);

  const [modalOpen, setModalOpen] = useState(false);
  const [actionError, setActionError] = useState<string | null>(null);
  const { page, setPage, totalPages, totalItems, pageSize, pageItems } = usePagination(issues);

  useEffect(() => {
    void fetchIssues(projectId);
    void fetchVendorSection(projectId);
    void fetchVendors();
    void fetchStaff();
  }, [projectId, fetchIssues, fetchVendorSection, fetchVendors, fetchStaff]);

  async function handleStatusChange(issueId: string, status: IssueStatus) {
    setActionError(null);
    try {
      await updateIssueStatus(projectId, issueId, status);
    } catch (err) {
      setActionError(getApiErrorMessage(err, "Gagal memperbarui status kendala"));
    }
  }

  async function handleAddIssue(values: IssueFormValues) {
    setActionError(null);
    try {
      await createIssue(projectId, values);
      setModalOpen(false);
    } catch (err) {
      setActionError(getApiErrorMessage(err, "Gagal mencatat kendala"));
    }
  }

  return (
    <div id="kendala">
      <Card>
        <CardHeader
          title="Kendala"
          subtitle="Kendala yang tercatat pada vendor-vendor di project ini beserta status penanganannya."
          action={
            <Button size="sm" icon={<Plus className="h-3.5 w-3.5" />} onClick={() => setModalOpen(true)}>
              Tambah Kendala
            </Button>
          }
        />
        <CardContent>
          {actionError && (
            <p className="mb-3 rounded-md border border-danger/30 bg-danger-soft px-3.5 py-2.5 text-[13px] font-medium text-danger">{actionError}</p>
          )}
          {issues.length === 0 ? (
            <p className="rounded-md border border-dashed border-border px-4 py-6 text-center text-[13px] text-text-secondary">
              Tidak ada kendala tercatat untuk project ini.
            </p>
          ) : (
            <>
            <CardList
              className="sm:hidden"
              items={pageItems}
              keyFor={(issue) => issue.id}
              renderItem={(issue) => {
                const pv = vendorEngagements.find((v) => v.id === issue.projectVendorId);
                const vendor = pv ? vendors.find((v) => v.id === pv.vendorId) : null;
                return (
                  <>
                    <div className="flex items-start justify-between gap-3">
                      <span className="font-medium text-text-primary">{issue.title}</span>
                      <IssueImpactBadge impact={issue.impact} />
                    </div>
                    <div className="flex flex-col gap-1.5">
                      <CardListField label="Vendor" value={vendor?.name ?? "Vendor tidak diketahui"} />
                      <CardListField label="Ditemukan" value={formatDate(issue.foundDate)} />
                      <CardListField label="Target Selesai" value={formatDate(issue.targetResolutionDate)} />
                      <CardListField label="PIC" value={staff.find((s) => s.id === issue.picStaffId)?.name ?? "-"} />
                    </div>
                    <Select
                      value={issue.status}
                      onChange={(e) => void handleStatusChange(issue.id, e.target.value as IssueStatus)}
                      className="h-9 w-full text-[13px]"
                    >
                      {ISSUE_STATUS_OPTIONS.map((s) => (
                        <option key={s} value={s}>{s}</option>
                      ))}
                    </Select>
                  </>
                );
              }}
            />
            <div className="hidden sm:block">
            <Table>
              <THead>
                <TR>
                  <TH>Judul Kendala</TH>
                  <TH>Vendor</TH>
                  <TH>Dampak</TH>
                  <TH>Status</TH>
                  <TH>Ditemukan</TH>
                  <TH>Target Selesai</TH>
                  <TH>PIC</TH>
                </TR>
              </THead>
              <TBody>
                {pageItems.map((issue) => {
                  const pv = vendorEngagements.find((v) => v.id === issue.projectVendorId);
                  const vendor = pv ? vendors.find((v) => v.id === pv.vendorId) : null;
                  return (
                    <TR key={issue.id}>
                      <TD className="font-medium">{issue.title}</TD>
                      <TD>{vendor?.name ?? "Vendor tidak diketahui"}</TD>
                      <TD><IssueImpactBadge impact={issue.impact} /></TD>
                      <TD>
                        <Select
                          value={issue.status}
                          onChange={(e) => void handleStatusChange(issue.id, e.target.value as IssueStatus)}
                          className="h-8 w-40 text-[13px]"
                        >
                          {ISSUE_STATUS_OPTIONS.map((s) => (
                            <option key={s} value={s}>{s}</option>
                          ))}
                        </Select>
                      </TD>
                      <TD>{formatDate(issue.foundDate)}</TD>
                      <TD>{formatDate(issue.targetResolutionDate)}</TD>
                      <TD>{staff.find((s) => s.id === issue.picStaffId)?.name ?? "-"}</TD>
                    </TR>
                  );
                })}
              </TBody>
            </Table>
            </div>
            <Pagination
              page={page}
              totalPages={totalPages}
              totalItems={totalItems}
              pageSize={pageSize}
              onPageChange={setPage}
              className="-mx-5 -mb-4 mt-1"
            />
            </>
          )}
        </CardContent>
      </Card>

      {modalOpen && (
        <AddIssueModal
          open={modalOpen}
          onClose={() => setModalOpen(false)}
          onSubmit={(values) => void handleAddIssue(values)}
          vendorEngagements={vendorEngagements}
          vendors={vendors}
          staff={staff}
        />
      )}
    </div>
  );
}

function toFormValues(defaultProjectVendorId: string, defaultStaffId: string): IssueFormValues {
  return {
    title: "",
    description: "",
    projectVendorId: defaultProjectVendorId,
    impact: "Low",
    foundDate: todayISO(),
    resolutionPlan: "",
    picStaffId: defaultStaffId,
    targetResolutionDate: "",
  };
}

function AddIssueModal({
  open,
  onClose,
  onSubmit,
  vendorEngagements,
  vendors,
  staff,
}: {
  open: boolean;
  onClose: () => void;
  onSubmit: (values: IssueFormValues) => void;
  vendorEngagements: ProjectVendor[];
  vendors: { id: string; name: string }[];
  staff: { id: string; name: string; title: string }[];
}) {
  const defaultVendorId = vendorEngagements[0]?.id ?? "";
  const defaultStaffId = staff[0]?.id ?? "";
  const [values, setValues] = useState<IssueFormValues>(() => toFormValues(defaultVendorId, defaultStaffId));
  const [errors, setErrors] = useState<Partial<Record<keyof IssueFormValues, string>>>({});

  function set<K extends keyof IssueFormValues>(key: K, value: IssueFormValues[K]) {
    setValues((prev) => ({ ...prev, [key]: value }));
  }

  function handleSubmit() {
    const result = issueSchema.safeParse(values);
    if (!result.success) {
      const fieldErrors: Partial<Record<keyof IssueFormValues, string>> = {};
      for (const issue of result.error.issues) {
        fieldErrors[issue.path[0] as keyof IssueFormValues] = issue.message;
      }
      setErrors(fieldErrors);
      return;
    }
    onSubmit(result.data);
    setValues(toFormValues(defaultVendorId, defaultStaffId));
    setErrors({});
  }

  return (
    <Modal
      open={open}
      onClose={onClose}
      title="Tambah Kendala"
      description="Catat kendala baru terkait vendor pada project ini."
      size="lg"
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>Batal</Button>
          <Button onClick={handleSubmit}>Simpan Kendala</Button>
        </>
      }
    >
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <div className="sm:col-span-2">
          <Field label="Judul Kendala" required hint={errors.title}>
            <Input value={values.title} onChange={(e) => set("title", e.target.value)} />
          </Field>
        </div>
        <div className="sm:col-span-2">
          <Field label="Deskripsi" required hint={errors.description}>
            <Textarea rows={2} value={values.description} onChange={(e) => set("description", e.target.value)} />
          </Field>
        </div>
        <Field label="Vendor" required hint={errors.projectVendorId}>
          <Select value={values.projectVendorId} onChange={(e) => set("projectVendorId", e.target.value)}>
            {vendorEngagements.map((pv) => (
              <option key={pv.id} value={pv.id}>
                {vendors.find((v) => v.id === pv.vendorId)?.name ?? "Vendor tidak diketahui"}
              </option>
            ))}
          </Select>
        </Field>
        <Field label="Dampak" required>
          <Select value={values.impact} onChange={(e) => set("impact", e.target.value as IssueImpact)}>
            {ISSUE_IMPACT_OPTIONS.map((i) => (
              <option key={i} value={i}>{i}</option>
            ))}
          </Select>
        </Field>
        <Field label="Tanggal Ditemukan" required hint={errors.foundDate}>
          <Input type="date" value={values.foundDate} onChange={(e) => set("foundDate", e.target.value)} />
        </Field>
        <Field label="Target Penyelesaian">
          <Input
            type="date"
            value={values.targetResolutionDate}
            onChange={(e) => set("targetResolutionDate", e.target.value)}
          />
        </Field>
        <Field label="PIC" required hint={errors.picStaffId}>
          <Select value={values.picStaffId} onChange={(e) => set("picStaffId", e.target.value)}>
            {staff.map((s) => (
              <option key={s.id} value={s.id}>{s.name} — {s.title}</option>
            ))}
          </Select>
        </Field>
        <div className="sm:col-span-2">
          <Field label="Rencana Penanganan">
            <Textarea rows={2} value={values.resolutionPlan} onChange={(e) => set("resolutionPlan", e.target.value)} />
          </Field>
        </div>
      </div>
    </Modal>
  );
}
