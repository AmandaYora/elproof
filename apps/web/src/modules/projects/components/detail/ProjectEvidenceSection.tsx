import { useEffect, useState } from "react";
import { Plus, Eye, EyeOff } from "lucide-react";
import { Card, CardHeader, CardContent } from "@/shared/components/ui/Card";
import { Badge } from "@/shared/components/ui/Badge";
import { Button } from "@/shared/components/ui/Button";
import { Select } from "@/shared/components/ui/Input";
import { Table, THead, TBody, TR, TH, TD } from "@/shared/components/ui/Table";
import { CardList, CardListField } from "@/shared/components/ui/CardList";
import { Pagination } from "@/shared/components/ui/Pagination";
import { usePagination } from "@/shared/hooks/usePagination";
import { EmptyState } from "@/shared/components/feedback/EmptyState";
import { EvidenceViewerModal } from "@/shared/components/ui/EvidenceViewerModal";
import { EvidenceUploadModal } from "@/shared/components/ui/EvidenceUploadModal";
import { IconActionButton } from "@/shared/components/ui/IconActionButton";
import { useProjectStore } from "@/modules/projects/stores/useProjectStore";
import { useVendorStore } from "@/modules/vendors/stores/useVendorStore";
import { useStaffStore } from "@/modules/users/stores/useStaffStore";
import { EVIDENCE_TYPE_OPTIONS, type EvidenceUploadFormValues } from "@/modules/projects/schemas/evidence.schema";
import { compressFileForUpload } from "@/shared/lib/image-compression";
import { getApiErrorMessage } from "@/shared/lib/api-error";
import type { Evidence, EvidenceRelatedKind, EvidenceType } from "@/modules/projects/types";
import { formatCurrency, formatDate } from "@/shared/lib/formatters";

export function ProjectEvidenceSection({ projectId }: { projectId: string }) {
  const evidence = useProjectStore((s) => s.evidence);
  const milestones = useProjectStore((s) => s.milestones);
  const vendorEngagements = useProjectStore((s) => s.vendorEngagements);
  const vendorMilestones = useProjectStore((s) => s.vendorMilestones);
  const payments = useProjectStore((s) => s.payments);
  const clientPayments = useProjectStore((s) => s.clientPayments);
  const venuePayments = useProjectStore((s) => s.venuePayments);
  const issues = useProjectStore((s) => s.issues);
  const fetchEvidence = useProjectStore((s) => s.fetchEvidence);
  const fetchMilestones = useProjectStore((s) => s.fetchMilestones);
  const fetchVendorSection = useProjectStore((s) => s.fetchVendorSection);
  const fetchPayments = useProjectStore((s) => s.fetchPayments);
  const fetchClientPayments = useProjectStore((s) => s.fetchClientPayments);
  const fetchVenuePayments = useProjectStore((s) => s.fetchVenuePayments);
  const fetchIssues = useProjectStore((s) => s.fetchIssues);
  const uploadEvidence = useProjectStore((s) => s.uploadEvidence);
  const toggleEvidenceClientVisible = useProjectStore((s) => s.toggleEvidenceClientVisible);
  const vendors = useVendorStore((s) => s.vendors);
  const fetchVendors = useVendorStore((s) => s.fetchVendors);
  const staff = useStaffStore((s) => s.staffSummaries);
  const fetchStaff = useStaffStore((s) => s.fetchStaffSummaries);

  const [typeFilter, setTypeFilter] = useState<"Semua" | EvidenceType>("Semua");
  const [modalOpen, setModalOpen] = useState(false);
  const [modalError, setModalError] = useState<string | null>(null);
  const [viewingEvidence, setViewingEvidence] = useState<Evidence | null>(null);

  useEffect(() => {
    void fetchEvidence(projectId);
    void fetchMilestones(projectId);
    void fetchVendorSection(projectId);
    void fetchPayments(projectId);
    void fetchClientPayments(projectId);
    void fetchVenuePayments(projectId);
    void fetchIssues(projectId);
    void fetchVendors();
    void fetchStaff();
  }, [projectId, fetchEvidence, fetchMilestones, fetchVendorSection, fetchPayments, fetchClientPayments, fetchVenuePayments, fetchIssues, fetchVendors, fetchStaff]);

  const filteredEvidence = typeFilter === "Semua" ? evidence : evidence.filter((e) => e.type === typeFilter);
  const { page, setPage, totalPages, totalItems, pageSize, pageItems } = usePagination(filteredEvidence);

  function vendorNameFor(pvId: string): string {
    const pv = vendorEngagements.find((v) => v.id === pvId);
    return pv ? vendors.find((v) => v.id === pv.vendorId)?.name ?? "Vendor tidak diketahui" : "Vendor tidak diketahui";
  }

  function contextLabel(item: Evidence): string {
    if (item.relatedKind === "vendorMilestone") {
      const milestone = vendorMilestones.find((m) => m.id === item.relatedId);
      return milestone ? `Timeline: ${milestone.name} — ${vendorNameFor(milestone.projectVendorId)}` : "Timeline";
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
    if (item.relatedKind === "clientPayment") {
      const payment = clientPayments.find((p) => p.id === item.relatedId);
      return payment ? `Pembayaran Client ${payment.type} (${formatDate(payment.paymentDate)})` : "Pembayaran Client";
    }
    if (item.relatedKind === "venuePayment") {
      const payment = venuePayments.find((p) => p.id === item.relatedId);
      return payment ? `Pembayaran Venue ${payment.type} (${formatDate(payment.paymentDate)})` : "Pembayaran Venue";
    }
    if (item.relatedKind === "general") {
      return "Dokumen Umum";
    }
    if (item.relatedKind === "projectMilestone") {
      const milestone = milestones.find((m) => m.id === item.relatedId);
      return milestone ? `Timeline: ${milestone.name}` : "Timeline";
    }
    return "-";
  }

  function relatedOptionsFor(kind: EvidenceRelatedKind): { id: string; label: string }[] {
    if (kind === "general") {
      return [];
    }
    if (kind === "vendorMilestone") {
      return vendorMilestones.map((m) => ({ id: m.id, label: `${vendorNameFor(m.projectVendorId)} — ${m.name}` }));
    }
    if (kind === "payment") {
      return payments.map((p) => ({ id: p.id, label: `${vendorNameFor(p.projectVendorId)} — ${p.type} (${formatDate(p.paymentDate)})` }));
    }
    if (kind === "projectVendor") {
      return vendorEngagements.map((pv) => ({ id: pv.id, label: vendorNameFor(pv.id) }));
    }
    if (kind === "clientPayment") {
      return clientPayments.map((p) => ({ id: p.id, label: `${p.type} — ${formatCurrency(p.amount)} (${formatDate(p.paymentDate)})` }));
    }
    if (kind === "venuePayment") {
      return venuePayments.map((p) => ({ id: p.id, label: `${p.type} — ${formatCurrency(p.amount)} (${formatDate(p.paymentDate)})` }));
    }
    if (kind === "projectMilestone") {
      return milestones.map((m) => ({ id: m.id, label: m.name }));
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
        isClientVisible: values.isClientVisible,
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
          subtitle="Seluruh dokumen pendukung timeline, pembayaran, kerja sama vendor, dan kendala pada project ini."
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
              description="Evidence akan muncul di sini setelah diunggah untuk timeline, pembayaran, kerja sama vendor, atau kendala pada project ini."
            />
          ) : filteredEvidence.length === 0 ? (
            <p className="rounded-md border border-dashed border-border px-4 py-6 text-center text-[13px] text-text-secondary">
              Tidak ada evidence dengan jenis ini.
            </p>
          ) : (
            <>
            <CardList
              className="sm:hidden"
              items={pageItems}
              keyFor={(item) => item.id}
              renderItem={(item) => (
                <>
                  <div className="flex items-start justify-between gap-3">
                    <span className="font-medium text-text-primary">{item.name}</span>
                    <Badge tone="neutral">{item.type}</Badge>
                  </div>
                  <div className="flex flex-col gap-1.5">
                    <CardListField label="Tanggal Dokumen" value={formatDate(item.documentDate)} />
                    <CardListField label="Diunggah Oleh" value={staff.find((s) => s.id === item.uploadedByStaffId)?.name ?? "-"} />
                    <CardListField label="Konteks" value={contextLabel(item)} />
                    {item.relatedKind === "general" && (
                      <CardListField
                        label="Terlihat Client"
                        value={item.isClientVisible ? "Ya" : "Tidak"}
                      />
                    )}
                  </div>
                  <div className="flex items-center gap-2 pt-1">
                    <IconActionButton icon={Eye} label="Lihat Evidence" tone="info" onClick={() => setViewingEvidence(item)} />
                    {item.relatedKind === "general" && (
                      <IconActionButton
                        icon={item.isClientVisible ? EyeOff : Eye}
                        label={item.isClientVisible ? "Sembunyikan dari Client" : "Tampilkan ke Client"}
                        tone={item.isClientVisible ? "success" : "neutral"}
                        onClick={() => void toggleEvidenceClientVisible(projectId, item.id)}
                      />
                    )}
                  </div>
                </>
              )}
            />
            <div className="hidden sm:block">
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
                    <TD className="text-text-secondary">
                      <div className="flex items-center gap-2">
                        <span>{contextLabel(item)}</span>
                        {item.relatedKind === "general" && (
                          <Badge tone={item.isClientVisible ? "success" : "neutral"}>
                            {item.isClientVisible ? "Terlihat Client" : "Tersembunyi"}
                          </Badge>
                        )}
                      </div>
                    </TD>
                    <TD>
                      <div className="flex items-center gap-2">
                        <IconActionButton icon={Eye} label="Lihat Evidence" tone="info" onClick={() => setViewingEvidence(item)} />
                        {item.relatedKind === "general" && (
                          <IconActionButton
                            icon={item.isClientVisible ? EyeOff : Eye}
                            label={item.isClientVisible ? "Sembunyikan dari Client" : "Tampilkan ke Client"}
                            tone={item.isClientVisible ? "success" : "neutral"}
                            onClick={() => void toggleEvidenceClientVisible(projectId, item.id)}
                          />
                        )}
                      </div>
                    </TD>
                  </TR>
                ))}
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
        <EvidenceUploadModal
          open={modalOpen}
          onClose={() => setModalOpen(false)}
          onSubmit={handleAddEvidence}
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

