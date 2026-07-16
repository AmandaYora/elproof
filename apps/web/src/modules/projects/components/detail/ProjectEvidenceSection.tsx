import { useEffect, useState } from "react";
import { Plus, Eye } from "lucide-react";
import { Card, CardHeader, CardContent } from "@/shared/components/ui/Card";
import { Badge } from "@/shared/components/ui/Badge";
import { Button } from "@/shared/components/ui/Button";
import { Modal } from "@/shared/components/ui/Modal";
import { Input, Textarea, Select, Field } from "@/shared/components/ui/Input";
import { Table, THead, TBody, TR, TH, TD } from "@/shared/components/ui/Table";
import { Pagination } from "@/shared/components/ui/Pagination";
import { usePagination } from "@/shared/hooks/usePagination";
import { EmptyState } from "@/shared/components/feedback/EmptyState";
import { EvidenceViewerModal } from "@/shared/components/ui/EvidenceViewerModal";
import { IconActionButton } from "@/shared/components/ui/IconActionButton";
import { useProjectStore } from "@/modules/projects/stores/useProjectStore";
import { useVendorStore } from "@/modules/vendors/stores/useVendorStore";
import { useStaffStore } from "@/modules/users/stores/useStaffStore";
import {
  evidenceUploadSchema,
  EVIDENCE_TYPE_OPTIONS,
  EVIDENCE_RELATED_KIND_OPTIONS,
  type EvidenceUploadFormValues,
} from "@/modules/projects/schemas/evidence.schema";
import { compressFileForUpload } from "@/shared/lib/image-compression";
import { todayISO } from "@/modules/projects/lib/dates";
import { getApiErrorMessage } from "@/shared/lib/api-error";
import type { Evidence, EvidenceRelatedKind, EvidenceType } from "@/modules/projects/types";
import { formatDate } from "@/shared/lib/formatters";

const RELATED_KIND_LABEL: Record<EvidenceRelatedKind, string> = {
  vendorMilestone: "Milestone Vendor",
  payment: "Pembayaran",
  projectVendor: "Kerja Sama Vendor",
  issue: "Kendala",
};

export function ProjectEvidenceSection({ projectId }: { projectId: string }) {
  const evidence = useProjectStore((s) => s.evidence);
  const vendorEngagements = useProjectStore((s) => s.vendorEngagements);
  const vendorMilestones = useProjectStore((s) => s.vendorMilestones);
  const payments = useProjectStore((s) => s.payments);
  const issues = useProjectStore((s) => s.issues);
  const fetchEvidence = useProjectStore((s) => s.fetchEvidence);
  const fetchVendorSection = useProjectStore((s) => s.fetchVendorSection);
  const fetchPayments = useProjectStore((s) => s.fetchPayments);
  const fetchIssues = useProjectStore((s) => s.fetchIssues);
  const uploadEvidence = useProjectStore((s) => s.uploadEvidence);
  const vendors = useVendorStore((s) => s.vendors);
  const fetchVendors = useVendorStore((s) => s.fetchVendors);
  const staff = useStaffStore((s) => s.staff);
  const fetchStaff = useStaffStore((s) => s.fetchStaff);

  const [typeFilter, setTypeFilter] = useState<"Semua" | EvidenceType>("Semua");
  const [modalOpen, setModalOpen] = useState(false);
  const [modalError, setModalError] = useState<string | null>(null);
  const [viewingEvidence, setViewingEvidence] = useState<Evidence | null>(null);

  useEffect(() => {
    void fetchEvidence(projectId);
    void fetchVendorSection(projectId);
    void fetchPayments(projectId);
    void fetchIssues(projectId);
    void fetchVendors();
    void fetchStaff();
  }, [projectId, fetchEvidence, fetchVendorSection, fetchPayments, fetchIssues, fetchVendors, fetchStaff]);

  const filteredEvidence = typeFilter === "Semua" ? evidence : evidence.filter((e) => e.type === typeFilter);
  const { page, setPage, totalPages, totalItems, pageSize, pageItems } = usePagination(filteredEvidence);

  function vendorNameFor(pvId: string): string {
    const pv = vendorEngagements.find((v) => v.id === pvId);
    return pv ? vendors.find((v) => v.id === pv.vendorId)?.name ?? "Vendor tidak diketahui" : "Vendor tidak diketahui";
  }

  function contextLabel(item: Evidence): string {
    if (item.relatedKind === "vendorMilestone") {
      const milestone = vendorMilestones.find((m) => m.id === item.relatedId);
      return milestone ? `Milestone: ${milestone.name} — ${vendorNameFor(milestone.projectVendorId)}` : "Milestone";
    }
    if (item.relatedKind === "payment") {
      const payment = payments.find((p) => p.id === item.relatedId);
      return payment ? `Pembayaran ${payment.type} — ${vendorNameFor(payment.projectVendorId)}` : "Pembayaran";
    }
    if (item.relatedKind === "projectVendor") {
      return `Kerja Sama: ${vendorNameFor(item.relatedId)}`;
    }
    if (item.relatedKind === "issue") {
      const issue = issues.find((i) => i.id === item.relatedId);
      return issue ? `Kendala: ${issue.title} — ${vendorNameFor(issue.projectVendorId)}` : "Kendala";
    }
    return "-";
  }

  function relatedOptionsFor(kind: EvidenceRelatedKind): { id: string; label: string }[] {
    if (kind === "vendorMilestone") {
      return vendorMilestones.map((m) => ({ id: m.id, label: `${vendorNameFor(m.projectVendorId)} — ${m.name}` }));
    }
    if (kind === "payment") {
      return payments.map((p) => ({ id: p.id, label: `${vendorNameFor(p.projectVendorId)} — ${p.type} (${formatDate(p.paymentDate)})` }));
    }
    if (kind === "projectVendor") {
      return vendorEngagements.map((pv) => ({ id: pv.id, label: vendorNameFor(pv.id) }));
    }
    return issues.map((i) => ({ id: i.id, label: `${i.title} — ${vendorNameFor(i.projectVendorId)}` }));
  }

  async function handleAddEvidence(file: File, values: EvidenceUploadFormValues) {
    setModalError(null);
    try {
      const compressed = await compressFileForUpload(file);
      await uploadEvidence(projectId, {
        ...compressed,
        name: values.name,
        type: values.type,
        documentDate: values.documentDate,
        description: values.description,
        relatedKind: values.relatedKind,
        relatedId: values.relatedId,
      });
      setModalOpen(false);
    } catch (err) {
      setModalError(getApiErrorMessage(err, "Gagal mengunggah evidence"));
    }
  }

  return (
    <div id="dokumen">
      <Card>
        <CardHeader
          title="Dokumen & Evidence"
          subtitle="Seluruh dokumen pendukung milestone, pembayaran, kerja sama vendor, dan kendala pada project ini."
          action={
            <Button size="sm" icon={<Plus className="h-3.5 w-3.5" />} onClick={() => setModalOpen(true)}>
              Tambah Evidence
            </Button>
          }
        />
        <CardContent className="flex flex-col gap-4">
          {evidence.length > 0 && (
            <Select
              className="w-56"
              value={typeFilter}
              onChange={(e) => setTypeFilter(e.target.value as "Semua" | EvidenceType)}
            >
              <option value="Semua">Semua Jenis</option>
              {EVIDENCE_TYPE_OPTIONS.map((t) => (
                <option key={t} value={t}>{t}</option>
              ))}
            </Select>
          )}

          {evidence.length === 0 ? (
            <EmptyState
              title="Belum ada evidence"
              description="Evidence akan muncul di sini setelah diunggah untuk milestone, pembayaran, kerja sama vendor, atau kendala pada project ini."
            />
          ) : filteredEvidence.length === 0 ? (
            <p className="rounded-md border border-dashed border-border px-4 py-6 text-center text-[13px] text-text-secondary">
              Tidak ada evidence dengan jenis ini.
            </p>
          ) : (
            <>
            <Table>
              <THead>
                <TR>
                  <TH>Nama Evidence</TH>
                  <TH>Jenis</TH>
                  <TH>Tanggal Dokumen</TH>
                  <TH>Diunggah Oleh</TH>
                  <TH>Konteks</TH>
                  <TH>Aksi</TH>
                </TR>
              </THead>
              <TBody>
                {pageItems.map((item) => (
                  <TR key={item.id}>
                    <TD className="font-medium">{item.name}</TD>
                    <TD><Badge tone="neutral">{item.type}</Badge></TD>
                    <TD>{formatDate(item.documentDate)}</TD>
                    <TD>{staff.find((s) => s.id === item.uploadedByStaffId)?.name ?? "-"}</TD>
                    <TD className="text-text-secondary">{contextLabel(item)}</TD>
                    <TD>
                      <IconActionButton icon={Eye} label="Lihat Evidence" tone="info" onClick={() => setViewingEvidence(item)} />
                    </TD>
                  </TR>
                ))}
              </TBody>
            </Table>
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
        <AddEvidenceModal
          open={modalOpen}
          onClose={() => setModalOpen(false)}
          onSubmit={(file, values) => void handleAddEvidence(file, values)}
          error={modalError}
          relatedOptionsFor={relatedOptionsFor}
        />
      )}

      {viewingEvidence && (
        <EvidenceViewerModal
          open={Boolean(viewingEvidence)}
          onClose={() => setViewingEvidence(null)}
          projectId={projectId}
          evidence={viewingEvidence}
          contextLabel={contextLabel(viewingEvidence)}
        />
      )}
    </div>
  );
}

function emptyValues(defaultRelatedId: string): EvidenceUploadFormValues {
  return {
    name: "",
    type: "Document",
    relatedKind: "vendorMilestone",
    relatedId: defaultRelatedId,
    documentDate: todayISO(),
    description: "",
  };
}

function AddEvidenceModal({
  open,
  onClose,
  onSubmit,
  error,
  relatedOptionsFor,
}: {
  open: boolean;
  onClose: () => void;
  onSubmit: (file: File, values: EvidenceUploadFormValues) => void;
  error: string | null;
  relatedOptionsFor: (kind: EvidenceRelatedKind) => { id: string; label: string }[];
}) {
  const [values, setValues] = useState<EvidenceUploadFormValues>(() => emptyValues(relatedOptionsFor("vendorMilestone")[0]?.id ?? ""));
  const [file, setFile] = useState<File | null>(null);
  const [errors, setErrors] = useState<Partial<Record<keyof EvidenceUploadFormValues, string>>>({});
  const [fileError, setFileError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  const options = relatedOptionsFor(values.relatedKind);

  function set<K extends keyof EvidenceUploadFormValues>(key: K, value: EvidenceUploadFormValues[K]) {
    setValues((prev) => ({ ...prev, [key]: value }));
  }

  function handleRelatedKindChange(kind: EvidenceRelatedKind) {
    const firstOption = relatedOptionsFor(kind)[0]?.id ?? "";
    setValues((prev) => ({ ...prev, relatedKind: kind, relatedId: firstOption }));
  }

  async function handleSubmit() {
    if (!file) {
      setFileError("Berkas wajib dipilih");
      return;
    }
    const result = evidenceUploadSchema.safeParse(values);
    if (!result.success) {
      const fieldErrors: Partial<Record<keyof EvidenceUploadFormValues, string>> = {};
      for (const issue of result.error.issues) {
        fieldErrors[issue.path[0] as keyof EvidenceUploadFormValues] = issue.message;
      }
      setErrors(fieldErrors);
      return;
    }
    setSubmitting(true);
    try {
      onSubmit(file, result.data);
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <Modal
      open={open}
      onClose={onClose}
      title="Tambah Evidence"
      description="Unggah dokumen pendukung dan kaitkan dengan milestone, pembayaran, kerja sama vendor, atau kendala pada project ini."
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>Batal</Button>
          <Button onClick={() => void handleSubmit()} disabled={submitting || options.length === 0}>
            {submitting ? "Mengunggah..." : "Simpan Evidence"}
          </Button>
        </>
      }
    >
      <div className="flex flex-col gap-4">
        {error && <p className="text-[12.5px] font-medium text-danger">{error}</p>}
        <Field label="Berkas" required hint={fileError ?? undefined}>
          <input
            type="file"
            accept="image/jpeg,image/png,application/pdf"
            onChange={(e) => {
              setFile(e.target.files?.[0] ?? null);
              setFileError(null);
            }}
            className="block w-full text-[13px] text-text-secondary file:mr-3 file:rounded-md file:border-0 file:bg-navy-900 file:px-3 file:py-1.5 file:text-[12.5px] file:font-semibold file:text-white"
          />
        </Field>
        <Field label="Nama Evidence" required hint={errors.name}>
          <Input value={values.name} onChange={(e) => set("name", e.target.value)} placeholder="cth. Invoice DP Venue" />
        </Field>
        <Field label="Jenis Evidence" required>
          <Select value={values.type} onChange={(e) => set("type", e.target.value as EvidenceType)}>
            {EVIDENCE_TYPE_OPTIONS.map((t) => (
              <option key={t} value={t}>{t}</option>
            ))}
          </Select>
        </Field>
        <Field label="Konteks">
          <Select value={values.relatedKind} onChange={(e) => handleRelatedKindChange(e.target.value as EvidenceRelatedKind)}>
            {EVIDENCE_RELATED_KIND_OPTIONS.map((k) => (
              <option key={k} value={k}>{RELATED_KIND_LABEL[k]}</option>
            ))}
          </Select>
        </Field>
        <Field label="Terkait" required hint={errors.relatedId}>
          <Select value={values.relatedId} onChange={(e) => set("relatedId", e.target.value)}>
            {options.length === 0 && <option value="">Tidak ada data tersedia</option>}
            {options.map((o) => (
              <option key={o.id} value={o.id}>{o.label}</option>
            ))}
          </Select>
        </Field>
        <Field label="Tanggal Dokumen" required hint={errors.documentDate}>
          <Input type="date" value={values.documentDate} onChange={(e) => set("documentDate", e.target.value)} />
        </Field>
        <Field label="Deskripsi">
          <Textarea rows={2} value={values.description} onChange={(e) => set("description", e.target.value)} />
        </Field>
      </div>
    </Modal>
  );
}
