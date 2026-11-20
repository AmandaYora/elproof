import { lazy, Suspense, useEffect, useState } from "react";
import { useCan } from "@/shared/stores/usePermissionStore";
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
import { IconActionButton } from "@/shared/components/ui/IconActionButton";
import { useProjectStore } from "@/modules/projects/stores/useProjectStore";
import { useStaffStore } from "@/modules/users/stores/useStaffStore";
import { EVIDENCE_TYPE_OPTIONS, type EvidenceUploadFormValues } from "@/modules/projects/schemas/evidence.schema";
import { compressFileForUpload } from "@/shared/lib/image-compression";
import { getApiErrorMessage } from "@/shared/lib/api-error";
import type { Evidence, EvidenceRelatedKind, EvidenceType } from "@/modules/projects/types";
import { formatCurrency, formatDate } from "@/shared/lib/formatters";
import { evidenceTypeLabel } from "@/shared/lib/labels";

// Opened by a click only: the upload form and the viewer load then.
const EvidenceUploadModal = lazy(() => import("@/shared/components/ui/EvidenceUploadModal").then((m) => ({ default: m.EvidenceUploadModal })));
const EvidenceViewerModal = lazy(() => import("@/shared/components/ui/EvidenceViewerModal").then((m) => ({ default: m.EvidenceViewerModal })));

/** The kinds of file the client may be shown (the API refuses the flag on the others). */
const SHAREABLE_KINDS = new Set<EvidenceRelatedKind>(["general", "projectMilestone", "vendorMilestone"]);

export function ProjectEvidenceSection({ projectId }: { projectId: string }) {
  // Proofs may be added by whoever edits the project or records finance;
  // sharing a document with the client is a project decision (ADR-0026).
  const canEdit = useCan("project.edit");
  const canRecordFinance = useCan("finance.record");
  const evidence = useProjectStore((s) => s.evidence);
  const milestones = useProjectStore((s) => s.milestones);
  const suppliers = useProjectStore((s) => s.suppliers);
  const vendorMilestones = useProjectStore((s) => s.vendorMilestones);
  const supplierPayments = useProjectStore((s) => s.supplierPayments);
  const clientPayments = useProjectStore((s) => s.clientPayments);
  const issues = useProjectStore((s) => s.issues);
  const fetchEvidence = useProjectStore((s) => s.fetchEvidence);
  const fetchMilestones = useProjectStore((s) => s.fetchMilestones);
  const fetchSuppliers = useProjectStore((s) => s.fetchSuppliers);
  const fetchSupplierPayments = useProjectStore((s) => s.fetchSupplierPayments);
  const fetchClientPayments = useProjectStore((s) => s.fetchClientPayments);
  const fetchIssues = useProjectStore((s) => s.fetchIssues);
  const uploadEvidence = useProjectStore((s) => s.uploadEvidence);
  const setEvidenceClientVisible = useProjectStore((s) => s.setEvidenceClientVisible);
  const staff = useStaffStore((s) => s.staffSummaries);
  const fetchStaff = useStaffStore((s) => s.fetchStaffSummaries);

  const [typeFilter, setTypeFilter] = useState<"Semua" | EvidenceType>("Semua");
  const [modalOpen, setModalOpen] = useState(false);
  const [modalError, setModalError] = useState<string | null>(null);
  const [viewingEvidence, setViewingEvidence] = useState<Evidence | null>(null);

  useEffect(() => {
    void fetchEvidence(projectId);
    void fetchMilestones(projectId);
    void fetchSuppliers(projectId);
    void fetchSupplierPayments(projectId);
    void fetchClientPayments(projectId);
    void fetchIssues(projectId);
    void fetchStaff();
  }, [projectId, fetchEvidence, fetchMilestones, fetchSuppliers, fetchSupplierPayments, fetchClientPayments, fetchIssues, fetchStaff]);

  const filteredEvidence = typeFilter === "Semua" ? evidence : evidence.filter((e) => e.type === typeFilter);
  const { page, setPage, totalPages, totalItems, pageSize, pageItems } = usePagination(filteredEvidence);

  function supplierNameFor(supplierId: string): string {
    return suppliers.find((s) => s.id === supplierId)?.name || "vendor atau venue terhapus";
  }

  // Who uploaded a file: the staff member (from the team list, or the name the API sent), or for a proof the couple
  // sent from the Client Portal the client, marked as such (RT-07).
  function uploaderLabel(item: Evidence): string {
    if (item.uploadedBySource === "client") return `${item.uploadedByName || "Klien"} (klien)`;
    return staff.find((s) => s.id === item.uploadedByStaffId)?.name || item.uploadedByName || "-";
  }

  function contextLabel(item: Evidence): string {
    if (item.relatedKind === "vendorMilestone") {
      const milestone = vendorMilestones.find((m) => m.id === item.relatedId);
      return milestone ? `Tahapan ${supplierNameFor(milestone.projectSupplierId)}: ${milestone.name}` : "Tahapan vendor & venue";
    }
    if (item.relatedKind === "supplierPayment") {
      const payment = supplierPayments.find((p) => p.id === item.relatedId);
      return payment
        ? `Pembayaran ${payment.type} ke ${supplierNameFor(payment.projectSupplierId)} (dibayar ${payment.payer === "WO" ? "WO" : "klien langsung"})`
        : "Pembayaran vendor & venue";
    }
    if (item.relatedKind === "projectSupplier") {
      return `Vendor & venue: ${supplierNameFor(item.relatedId)}`;
    }
    if (item.relatedKind === "issue") {
      const issue = issues.find((i) => i.id === item.relatedId);
      if (!issue) return "Kendala";
      return issue.projectSupplierId ? `Kendala ${issue.supplierName || supplierNameFor(issue.projectSupplierId)}: ${issue.title}` : `Kendala: ${issue.title}`;
    }
    if (item.relatedKind === "clientPayment") {
      const payment = clientPayments.find((p) => p.id === item.relatedId);
      return payment ? `Pembayaran klien ${payment.type} (${formatDate(payment.paymentDate)})` : "Pembayaran klien";
    }
    if (item.relatedKind === "general") {
      return "Dokumen umum";
    }
    if (item.relatedKind === "projectMilestone") {
      const milestone = milestones.find((m) => m.id === item.relatedId);
      return milestone ? `Tahapan: ${milestone.name}` : "Tahapan";
    }
    return "-";
  }

  function relatedOptionsFor(kind: EvidenceRelatedKind): { id: string; label: string }[] {
    if (kind === "general") {
      return [];
    }
    if (kind === "vendorMilestone") {
      return vendorMilestones.map((m) => ({ id: m.id, label: `${supplierNameFor(m.projectSupplierId)}: ${m.name}` }));
    }
    if (kind === "supplierPayment") {
      return supplierPayments.map((p) => ({
        id: p.id,
        label: `${supplierNameFor(p.projectSupplierId)}: ${p.type} ${formatCurrency(p.amount)} (${formatDate(p.paymentDate)})`,
      }));
    }
    if (kind === "projectSupplier") {
      return suppliers.map((s) => ({ id: s.id, label: s.name }));
    }
    if (kind === "clientPayment") {
      return clientPayments.map((p) => ({ id: p.id, label: `${p.type} — ${formatCurrency(p.amount)} (${formatDate(p.paymentDate)})` }));
    }
    if (kind === "projectMilestone") {
      return milestones.map((m) => ({ id: m.id, label: m.name }));
    }
    return issues.map((i) => ({ id: i.id, label: i.projectSupplierId ? `${i.title} (${i.supplierName || supplierNameFor(i.projectSupplierId)})` : i.title }));
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
      setModalError(getApiErrorMessage(err, "Gagal mengunggah bukti"));
    }
  }

  return (
    <div id="dokumen">
      <Card>
        <CardHeader
          title="Dokumen dan bukti"
          subtitle="Seluruh dokumen dan bukti pendukung jadwal, pembayaran, vendor & venue, dan kendala pada proyek ini."
          action={
            (canEdit || canRecordFinance) && (
              <Button size="sm" icon={<Plus className="h-3.5 w-3.5" />} onClick={() => setModalOpen(true)}>
                Tambah dokumen
              </Button>
            )
          }
        />
        <CardContent className="flex flex-col gap-4">
          {evidence.length > 0 && (
            <Select
              className="w-56"
              value={typeFilter}
              onChange={(e) => setTypeFilter(e.target.value as "Semua" | EvidenceType)}
            >
              <option value="Semua">Semua jenis</option>
              {EVIDENCE_TYPE_OPTIONS.map((t) => (
                <option key={t} value={t}>{evidenceTypeLabel(t)}</option>
              ))}
            </Select>
          )}

          {evidence.length === 0 ? (
            <EmptyState
              title="Belum ada dokumen"
              description="Dokumen dan bukti muncul di sini setelah diunggah untuk jadwal, pembayaran, vendor & venue, atau kendala pada proyek ini."
            />
          ) : filteredEvidence.length === 0 ? (
            <p className="rounded-md border border-dashed border-border px-4 py-6 text-center text-[13px] text-text-secondary">
              Tidak ada dokumen dengan jenis ini.
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
                    <Badge tone="neutral">{evidenceTypeLabel(item.type)}</Badge>
                  </div>
                  <div className="flex flex-col gap-1.5">
                    <CardListField label="Tanggal dokumen" value={formatDate(item.documentDate)} />
                    <CardListField label="Diunggah oleh" value={uploaderLabel(item)} />
                    <CardListField label="Konteks" value={contextLabel(item)} />
                    {canEdit && SHAREABLE_KINDS.has(item.relatedKind) && (
                      <CardListField
                        label="Terlihat klien"
                        value={item.isClientVisible ? "Ya" : "Tidak"}
                      />
                    )}
                  </div>
                  <div className="flex items-center gap-2 pt-1">
                    <IconActionButton icon={Eye} readOnly label="Lihat bukti" tone="info" onClick={() => setViewingEvidence(item)} />
                    {canEdit && SHAREABLE_KINDS.has(item.relatedKind) && (
                      <IconActionButton
                        icon={item.isClientVisible ? EyeOff : Eye}
                        label={item.isClientVisible ? "Sembunyikan dari klien" : "Tampilkan ke klien"}
                        tone={item.isClientVisible ? "success" : "neutral"}
                        onClick={() => void setEvidenceClientVisible(projectId, item.id, !item.isClientVisible)}
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
                  <TH>Nama bukti</TH>
                  <TH>Jenis</TH>
                  <TH>Tanggal dokumen</TH>
                  <TH>Diunggah oleh</TH>
                  <TH>Konteks</TH>
                  <TH>Aksi</TH>
                </TR>
              </THead>
              <TBody>
                {pageItems.map((item) => (
                  <TR key={item.id}>
                    <TD className="font-medium">{item.name}</TD>
                    <TD><Badge tone="neutral">{evidenceTypeLabel(item.type)}</Badge></TD>
                    <TD>{formatDate(item.documentDate)}</TD>
                    <TD>{uploaderLabel(item)}</TD>
                    <TD className="text-text-secondary">
                      <div className="flex items-center gap-2">
                        <span>{contextLabel(item)}</span>
                        {canEdit && SHAREABLE_KINDS.has(item.relatedKind) && (
                          <Badge tone={item.isClientVisible ? "success" : "neutral"}>
                            {item.isClientVisible ? "Terlihat klien" : "Tersembunyi"}
                          </Badge>
                        )}
                      </div>
                    </TD>
                    <TD>
                      <div className="flex items-center gap-2">
                        <IconActionButton icon={Eye} readOnly label="Lihat bukti" tone="info" onClick={() => setViewingEvidence(item)} />
                        {canEdit && SHAREABLE_KINDS.has(item.relatedKind) && (
                          <IconActionButton
                            icon={item.isClientVisible ? EyeOff : Eye}
                            label={item.isClientVisible ? "Sembunyikan dari klien" : "Tampilkan ke klien"}
                            tone={item.isClientVisible ? "success" : "neutral"}
                            onClick={() => void setEvidenceClientVisible(projectId, item.id, !item.isClientVisible)}
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
        <Suspense fallback={null}>
          <EvidenceUploadModal
            open
            onClose={() => setModalOpen(false)}
            onSubmit={handleAddEvidence}
            error={modalError}
            relatedOptionsFor={relatedOptionsFor}
            canShareWithClient={canEdit}
          />
        </Suspense>
      )}

      {viewingEvidence && (
        <Suspense fallback={null}>
          <EvidenceViewerModal
            open
            onClose={() => setViewingEvidence(null)}
            projectId={projectId}
            evidence={viewingEvidence}
            contextLabel={contextLabel(viewingEvidence)}
          />
        </Suspense>
      )}
    </div>
  );
}

