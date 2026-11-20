import { lazy, Suspense, useEffect, useRef, useState, type ReactNode } from "react";
import { Link } from "react-router-dom";
import { Eye, Pencil, Plus, RefreshCw } from "lucide-react";
import { ISSUE_STATUS_LABELS, evidenceTypeLabel } from "@/shared/lib/labels";
import { Drawer } from "@/shared/components/ui/Drawer";
import { Button } from "@/shared/components/ui/Button";
import { IconActionButton } from "@/shared/components/ui/IconActionButton";
import { useZodForm } from "@/shared/forms/useZodForm";
import { FormErrorSummary } from "@/shared/forms/FormErrorSummary";
import { useCan } from "@/shared/stores/usePermissionStore";
import { useAuthStore } from "@/shared/stores/useAuthStore";
import { toast } from "@/shared/stores/useToastStore";
import { getApiErrorMessage } from "@/shared/lib/api-error";
import { compressFileForUpload } from "@/shared/lib/image-compression";
import { formatCurrency, formatDate } from "@/shared/lib/formatters";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import type { EvidenceUploadFormValues } from "@/modules/projects/schemas/evidence.schema";
import type { IssueFormValues } from "@/modules/projects/schemas/issue.schema";
import { useProjectStore, toIssueUpdateFields, type SaveOptions } from "@/modules/projects/stores/useProjectStore";
import { useStaffStore } from "@/modules/users/stores/useStaffStore";
import { todayISO } from "@/modules/projects/lib/dates";
import { appliedToContract } from "@/modules/projects/lib/payments";
import { EngagementStatusBadge, IssueImpactBadge } from "@/modules/projects/components/StatusBadges";
import type { RecordPaymentValues } from "@/modules/projects/components/detail/RecordPaymentDrawer";
import { SupplierFields, SUPPLIER_FIELD_LABELS, supplierKindLabel, supplierSchema, type SupplierFormValues } from "@/modules/rincian";
import { SupplierStepsPanel } from "@/modules/projects/components/suppliers/SupplierStepsPanel";
import { SupplierPaymentLedger, useSupplierPaymentPerms } from "@/modules/projects/components/suppliers/SupplierPaymentLedger";
import { stepProgress, stepsOf } from "@/modules/projects/components/suppliers/supplier-helpers";
import { SupplierBillsPanel } from "@/modules/projects/components/supplier-bills/SupplierBillsPanel";
import type { Evidence, ProjectSupplier, VendorIssue } from "@/modules/projects/types";

// Everything below opens on a click: each loads then.
const RecordPaymentDrawer = lazy(() => import("@/modules/projects/components/detail/RecordPaymentDrawer").then((m) => ({ default: m.RecordPaymentDrawer })));
const EvidenceUploadModal = lazy(() => import("@/shared/components/ui/EvidenceUploadModal").then((m) => ({ default: m.EvidenceUploadModal })));
const EvidenceViewerModal = lazy(() => import("@/shared/components/ui/EvidenceViewerModal").then((m) => ({ default: m.EvidenceViewerModal })));
const EvidenceListModal = lazy(() => import("@/shared/components/ui/EvidenceListModal").then((m) => ({ default: m.EvidenceListModal })));
const IssueFormModal = lazy(() => import("@/modules/projects/components/IssueFormModal").then((m) => ({ default: m.IssueFormModal })));

/** The form's values for one supplier as the API gave it. */
function toValues(s: ProjectSupplier): SupplierFormValues {
  return {
    vendorId: s.supplierId,
    paidBy: s.paidBy,
    showName: s.showName,
    agreedCost: s.agreedCost ?? "",
    engagementStatus: s.engagementStatus,
    bookingDate: s.bookingDate ?? "",
    dueDate: s.dueDate ?? "",
    picStaffId: s.picStaffId,
    scope: s.scope,
    notes: s.notes,
  };
}

const sameValues = (a: SupplierFormValues, b: SupplierFormValues) => JSON.stringify(a) === JSON.stringify(b);

function Section({ id, title, action, children }: { id: string; title: string; action?: ReactNode; children: ReactNode }) {
  return (
    <section aria-labelledby={id} className="flex flex-col gap-3 border-t border-border pt-5">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h3 id={id} className="text-[14px] font-semibold text-foreground">{title}</h3>
        {action}
      </div>
      {children}
    </section>
  );
}

interface Props {
  projectId: string;
  supplierId: string;
  onClose: () => void;
}

/**
 * One vendor or venue of the project (DESAIN §5.2): its settings, the venue's capacity, its steps, proof, payments by
 * who pays it, and its kendala. Opened from a row of Vendor & venue (`?buka=<id>`), so a link can open it directly.
 */
export function SupplierDetailDrawer({ projectId, supplierId, onClose }: Props) {
  const supplier = useProjectStore((s) => s.suppliers.find((x) => x.id === supplierId));
  const suppliersLoaded = useProjectStore((s) => s.suppliers.length > 0);
  const fetchSupplierPayments = useProjectStore((s) => s.fetchSupplierPayments);
  const fetchEvidence = useProjectStore((s) => s.fetchEvidence);
  const fetchIssues = useProjectStore((s) => s.fetchIssues);
  const fetchVendorMilestoneHistory = useProjectStore((s) => s.fetchVendorMilestoneHistory);
  const updateSupplier = useProjectStore((s) => s.updateSupplier);
  const staff = useStaffStore((s) => s.staffSummaries);
  const fetchStaff = useStaffStore((s) => s.fetchStaffSummaries);
  const canEdit = useCan("project.edit");
  const canMargin = useCan("finance.margin.view");
  const formRef = useRef<HTMLFormElement>(null);

  useEffect(() => {
    void fetchSupplierPayments(projectId);
    void fetchEvidence(projectId);
    void fetchIssues(projectId);
    void fetchVendorMilestoneHistory(projectId);
    void fetchStaff();
  }, [projectId, fetchSupplierPayments, fetchEvidence, fetchIssues, fetchVendorMilestoneHistory, fetchStaff]);

  // The version the form was filled from: saving sends only what changed against it, with its updatedAt (ADR-0036).
  const [before, setBefore] = useState<ProjectSupplier | undefined>(supplier);
  const form = useZodForm({ schema: supplierSchema, initial: supplier ? toValues(supplier) : toValues(PLACEHOLDER) });
  const { values, errors, set, blur } = form;

  // A newer copy came in (a payment, a refreshed capacity, a save): take it while nothing is typed over the old one.
  useEffect(() => {
    if (!supplier) return;
    if (!before || before.id !== supplier.id) {
      setBefore(supplier);
      form.reset(toValues(supplier));
      return;
    }
    if (supplier.updatedAt !== before.updatedAt && sameValues(form.values, toValues(before))) {
      setBefore(supplier);
      form.reset(toValues(supplier));
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [supplier]);

  const dirty = before ? !sameValues(values, toValues(before)) : false;

  const submit = form.handleSubmit(
    async (parsed) => {
      if (!before) return;
      await updateSupplier(projectId, before, parsed, canMargin);
      const fresh = useProjectStore.getState().suppliers.find((x) => x.id === before.id);
      if (fresh) {
        setBefore(fresh);
        form.reset(toValues(fresh));
      }
      toast.success("Perubahan disimpan");
    },
    { root: () => formRef.current, fallback: "Perubahan belum tersimpan. Periksa koneksi Anda, lalu coba lagi.", known: (key) => key in values },
  );

  const title = supplier?.name ?? "Vendor atau venue";

  return (
    <Drawer
      open
      onClose={onClose}
      title={title}
      description={supplier ? `${supplierKindLabel(supplier)} di proyek ini` : undefined}
      size="lg"
      busy={form.submitting}
      footer={
        <>
          <Button variant="secondary" onClick={onClose} disabled={form.submitting}>Tutup</Button>
          {supplier && canEdit && (
            <Button type="submit" form="supplier-settings-form" disabled={form.submitting || !dirty}>
              {form.submitting ? "Menyimpan..." : "Simpan perubahan"}
            </Button>
          )}
        </>
      }
    >
      {!supplier ? (
        <p className="text-[13px] text-muted-foreground">
          {suppliersLoaded ? "Vendor atau venue ini tidak ada lagi di proyek. Mungkin baru saja dihapus." : "Memuat..."}
        </p>
      ) : (
        <div className="flex flex-col gap-6">
          <SupplierOverview projectId={projectId} supplier={supplier} canEdit={canEdit} />

          <form id="supplier-settings-form" ref={formRef} noValidate onSubmit={(e) => void submit(e)} className="flex flex-col gap-4">
            <FormErrorSummary errors={form.errorList} formError={form.formError} labels={SUPPLIER_FIELD_LABELS} root={() => formRef.current} />
            <SupplierFields
              values={values}
              errors={errors}
              set={set}
              blur={blur}
              canMargin={canMargin}
              paidByLocked={supplier.paidByLocked}
              isVenue={supplier.kind === "venue"}
              staff={staff}
              disabled={!canEdit}
            />
          </form>

          <div className="border-t border-border pt-5">
            <SupplierStepsPanel projectId={projectId} supplier={supplier} canEdit={canEdit} />
          </div>
          <SupplierPaymentsPanel projectId={projectId} supplier={supplier} />
          <SupplierProofPanel projectId={projectId} supplier={supplier} />
          <SupplierIssuesPanel projectId={projectId} supplier={supplier} canEdit={canEdit} staff={staff} />
        </div>
      )}
    </Drawer>
  );
}

// Only fills the form's first render before the supplier list arrives; never sent.
const PLACEHOLDER: ProjectSupplier = {
  updatedAt: "", id: "", kind: "vendor", supplierId: "", name: "", categoryId: "", categoryName: "", paidBy: "WO", showName: true,
  capacityStanding: null, capacitySeated: null, agreedCost: null, paidWo: null, paidClient: null, engagementStatus: "Planned",
  bookingDate: null, dueDate: null, picStaffId: "", picName: "", scope: "", notes: "", paidByLocked: "", usedByEvents: 0, evidenceCount: 0,
};

/** Status, progress and, for a venue, its capacity and the events held there. */
function SupplierOverview({ projectId, supplier, canEdit }: { projectId: string; supplier: ProjectSupplier; canEdit: boolean }) {
  const vendorMilestones = useProjectStore((s) => s.vendorMilestones);
  const refreshSupplierCapacity = useProjectStore((s) => s.refreshSupplierCapacity);
  const [refreshing, setRefreshing] = useState(false);
  const { done, total } = stepProgress(stepsOf(vendorMilestones, supplier.id));

  async function refreshCapacity() {
    setRefreshing(true);
    try {
      await refreshSupplierCapacity(projectId, supplier.id);
      toast.success("Kapasitas diperbarui dari direktori venue");
    } catch (err) {
      toast.error(getApiErrorMessage(err, "Kapasitas belum bisa diperbarui"));
    } finally {
      setRefreshing(false);
    }
  }

  const capacity = [
    supplier.capacityStanding !== null ? `${supplier.capacityStanding.toLocaleString("id-ID")} berdiri` : null,
    supplier.capacitySeated !== null ? `${supplier.capacitySeated.toLocaleString("id-ID")} duduk` : null,
  ].filter(Boolean);

  return (
    <div className="flex flex-col gap-3">
      <dl className="grid grid-cols-2 gap-x-4 gap-y-3 text-[13px] sm:grid-cols-3">
        <div>
          <dt className="text-muted-foreground">Status</dt>
          <dd className="mt-1"><EngagementStatusBadge status={supplier.engagementStatus} /></dd>
        </div>
        <div>
          <dt className="text-muted-foreground">Progres</dt>
          <dd className="mt-1 font-medium text-foreground">{total === 0 ? "Belum ada tahapan" : `${done}/${total} tahap`}</dd>
        </div>
        <div>
          <dt className="text-muted-foreground">PIC internal WO</dt>
          <dd className="mt-1 font-medium text-foreground">{supplier.picName || "-"}</dd>
        </div>
      </dl>
      {supplier.kind === "venue" && (
        <div className="flex flex-wrap items-center justify-between gap-3 rounded-md bg-muted px-3.5 py-3 text-[13px]">
          <div className="min-w-0">
            <p className="text-foreground">Kapasitas: {capacity.length > 0 ? capacity.join(", ") : "belum diisi di direktori"}</p>
            <p className="text-muted-foreground">
              {supplier.usedByEvents === 0 ? "Tidak dipakai acara mana pun" : `Dipakai ${supplier.usedByEvents} acara proyek ini`}
            </p>
          </div>
          {canEdit && (
            <Button size="sm" variant="secondary" icon={<RefreshCw className="h-3.5 w-3.5" />} onClick={() => void refreshCapacity()} disabled={refreshing}>
              {refreshing ? "Memperbarui..." : "Perbarui kapasitas"}
            </Button>
          )}
        </div>
      )}
    </div>
  );
}

/**
 * Payments by who pays the supplier: the WO's are seen with the margin permission only; a vendor the client pays
 * directly shows its bills and the couple's proofs to every staff (SupplierBillsPanel, Paket WO Tahap 4).
 */
function SupplierPaymentsPanel({ projectId, supplier }: { projectId: string; supplier: ProjectSupplier }) {
  if (supplier.paidBy === "Client") return <SupplierBillsPanel projectId={projectId} supplier={supplier} />;
  return <WoPaymentsPanel projectId={projectId} supplier={supplier} />;
}

/** What the WO pays this vendor or venue against its "Biaya rekanan" (finance.margin.view). */
function WoPaymentsPanel({ projectId, supplier }: { projectId: string; supplier: ProjectSupplier }) {
  const supplierPayments = useProjectStore((s) => s.supplierPayments);
  const createSupplierPayment = useProjectStore((s) => s.createSupplierPayment);
  const perms = useSupplierPaymentPerms();
  const [recording, setRecording] = useState(false);
  if (!perms.margin) return null;

  const own = supplierPayments.filter((p) => p.projectSupplierId === supplier.id && p.payer === "WO");
  const paid = appliedToContract(own);
  const cost = supplier.agreedCost;
  const canRecord = perms.canRecord("WO");

  async function record(values: RecordPaymentValues, options: SaveOptions) {
    await createSupplierPayment(projectId, values, options);
    setRecording(false);
  }

  return (
    <Section
      id="supplier-payments"
      title="Pembayaran ke rekanan"
      action={
        canRecord && (
          <Button size="sm" variant="secondary" icon={<Plus className="h-3.5 w-3.5" />} onClick={() => setRecording(true)}>
            Catat pembayaran
          </Button>
        )
      }
    >
      <dl className="grid grid-cols-3 gap-3 rounded-md border border-border px-3.5 py-3 text-[13px]">
        <div>
          <dt className="text-muted-foreground">Biaya rekanan</dt>
          <dd className={cost === null ? "mt-0.5 font-medium text-warning-text" : "mt-0.5 font-semibold tabular-nums text-foreground"}>
            {cost === null ? "Belum diisi" : formatCurrency(cost)}
          </dd>
        </div>
        <div>
          <dt className="text-muted-foreground">Sudah dibayar</dt>
          <dd className="mt-0.5 font-semibold tabular-nums text-foreground">{formatCurrency(paid)}</dd>
        </div>
        <div>
          <dt className="text-muted-foreground">Sisa</dt>
          <dd className="mt-0.5 font-semibold tabular-nums text-foreground">{cost === null ? "-" : formatCurrency(Math.max(cost - paid, 0))}</dd>
        </div>
      </dl>
      <SupplierPaymentLedger
        projectId={projectId}
        label={`Pembayaran ${supplier.name}`}
        payments={own}
        suppliers={[supplier]}
        showSupplier={false}
        empty={{ title: "Belum ada pembayaran", description: "Catat setiap pembayaran WO ke rekanan ini." }}
      />
      {recording && (
        <Suspense fallback={null}>
          <RecordPaymentDrawer
            open
            onClose={() => setRecording(false)}
            title="Catat pembayaran ke rekanan"
            description={`Pembayaran WO ke ${supplier.name}.`}
            supplierOptions={[{ value: supplier.id, label: supplier.name }]}
            initialSupplierId={supplier.id}
            slots={{ invoice: true }}
            owed={() => ({ contractValue: cost ?? 0, existing: own, owedLabel: "sisa biaya rekanan" })}
            onSubmit={record}
          />
        </Suspense>
      )}
    </Section>
  );
}

/** Files about this supplier, its steps and its payments (what the caller may see), plus a file about the supplier itself. */
function SupplierProofPanel({ projectId, supplier }: { projectId: string; supplier: ProjectSupplier }) {
  const evidence = useProjectStore((s) => s.evidence);
  const vendorMilestones = useProjectStore((s) => s.vendorMilestones);
  const supplierPayments = useProjectStore((s) => s.supplierPayments);
  const uploadEvidence = useProjectStore((s) => s.uploadEvidence);
  // Proofs may be added by whoever edits the project or records finance (ADR-0026).
  const canEditProject = useCan("project.edit");
  const canRecordFinance = useCan("finance.record");
  const canAdd = canEditProject || canRecordFinance;
  const [uploading, setUploading] = useState(false);
  const [uploadError, setUploadError] = useState<string | null>(null);
  const [viewing, setViewing] = useState<Evidence | null>(null);
  const [listOpen, setListOpen] = useState(false);

  const stepIds = new Set(vendorMilestones.filter((m) => m.projectSupplierId === supplier.id).map((m) => m.id));
  const paymentIds = new Set(supplierPayments.filter((p) => p.projectSupplierId === supplier.id).map((p) => p.id));
  const items = evidence
    .filter(
      (e) =>
        (e.relatedKind === "projectSupplier" && e.relatedId === supplier.id) ||
        (e.relatedKind === "vendorMilestone" && stepIds.has(e.relatedId)) ||
        (e.relatedKind === "supplierPayment" && paymentIds.has(e.relatedId)),
    )
    .sort((a, b) => b.uploadedAt.localeCompare(a.uploadedAt));

  function contextOf(e: Evidence): string {
    if (e.relatedKind === "vendorMilestone") return `Tahapan: ${vendorMilestones.find((m) => m.id === e.relatedId)?.name ?? "-"}`;
    if (e.relatedKind === "supplierPayment") {
      const p = supplierPayments.find((x) => x.id === e.relatedId);
      return p ? `Pembayaran ${p.type} ${formatCurrency(p.amount)}` : "Pembayaran";
    }
    return supplier.name;
  }

  async function upload(file: File, values: EvidenceUploadFormValues) {
    setUploadError(null);
    try {
      const compressed = await compressFileForUpload(file);
      await uploadEvidence(projectId, {
        ...compressed,
        name: values.name,
        type: values.type,
        documentDate: values.documentDate,
        description: values.description,
        relatedKind: "projectSupplier",
        relatedId: supplier.id,
        isClientVisible: false,
      });
      setUploading(false);
    } catch (err) {
      setUploadError(getApiErrorMessage(err, "Bukti belum terunggah. Periksa koneksi Anda, lalu coba lagi."));
    }
  }

  return (
    <Section
      id="supplier-proof"
      title="Bukti"
      action={
        <span className="flex items-center gap-2">
          {items.length > 3 && (
            <Button size="sm" variant="ghost" onClick={() => setListOpen(true)}>
              Lihat semua ({items.length})
            </Button>
          )}
          {canAdd && (
            <Button size="sm" variant="secondary" icon={<Plus className="h-3.5 w-3.5" />} onClick={() => setUploading(true)}>
              Tambah bukti
            </Button>
          )}
        </span>
      }
    >
      {items.length === 0 ? (
        <p className="text-[13px] text-muted-foreground">Belum ada bukti: kontrak, penawaran, atau tanda booking bisa diunggah di sini.</p>
      ) : (
        <ul className="divide-y divide-border rounded-md border border-border">
          {items.slice(0, 3).map((e) => (
            <li key={e.id} className="flex items-center gap-3 px-3.5 py-2.5">
              <div className="min-w-0 flex-1">
                <p className="truncate text-[13px] font-medium text-foreground">{e.name}</p>
                <p className="truncate text-[12px] text-muted-foreground">
                  {evidenceTypeLabel(e.type)}, {formatDate(e.documentDate ?? e.uploadedAt)}. {contextOf(e)}
                </p>
              </div>
              <IconActionButton icon={Eye} readOnly label={`Lihat ${e.name}`} tone="info" onClick={() => setViewing(e)} />
            </li>
          ))}
        </ul>
      )}
      {uploading && (
        <Suspense fallback={null}>
          <EvidenceUploadModal
            open
            onClose={() => setUploading(false)}
            onSubmit={upload}
            error={uploadError}
            lockedRelatedKind="projectSupplier"
            lockedRelatedId={supplier.id}
            defaultType="Contract"
            title="Tambah bukti"
            description={`Dokumen tentang ${supplier.name}, misalnya kontrak, penawaran, atau tanda booking.`}
          />
        </Suspense>
      )}
      {viewing && (
        <Suspense fallback={null}>
          <EvidenceViewerModal open onClose={() => setViewing(null)} projectId={projectId} evidence={viewing} contextLabel={contextOf(viewing)} />
        </Suspense>
      )}
      {listOpen && (
        <Suspense fallback={null}>
          <EvidenceListModal open projectId={projectId} title={`Bukti ${supplier.name}`} items={items} onClose={() => setListOpen(false)} />
        </Suspense>
      )}
    </Section>
  );
}

/** Kendala about this supplier: record one here, or open the Kendala tab for the whole list. */
function SupplierIssuesPanel({
  projectId,
  supplier,
  canEdit,
  staff,
}: {
  projectId: string;
  supplier: ProjectSupplier;
  canEdit: boolean;
  staff: { id: string; name: string; title: string }[];
}) {
  const issues = useProjectStore((s) => s.issues);
  const suppliers = useProjectStore((s) => s.suppliers);
  const vendorMilestones = useProjectStore((s) => s.vendorMilestones);
  const createIssue = useProjectStore((s) => s.createIssue);
  const updateIssue = useProjectStore((s) => s.updateIssue);
  const currentStaffId = useAuthStore((s) => s.currentStaffId);
  const [modal, setModal] = useState<{ mode: "create" | "edit"; issue?: VendorIssue; initial: IssueFormValues } | null>(null);

  const own = issues.filter((i) => i.projectSupplierId === supplier.id);

  function openCreate() {
    setModal({
      mode: "create",
      initial: {
        projectSupplierId: supplier.id,
        vendorMilestoneId: "",
        title: "",
        description: "",
        impact: "Low",
        foundDate: todayISO(),
        resolutionPlan: "",
        picStaffId: staff.some((s) => s.id === currentStaffId) ? currentStaffId : (staff[0]?.id ?? ""),
        targetResolutionDate: "",
        status: "Open",
        isClientVisible: true,
      },
    });
  }

  async function submit(values: IssueFormValues) {
    if (!modal) return;
    if (modal.mode === "edit" && modal.issue) await updateIssue(projectId, modal.issue.id, values, modal.issue.updatedAt);
    else await createIssue(projectId, values);
    setModal(null);
  }

  return (
    <Section
      id="supplier-issues"
      title="Kendala"
      action={
        canEdit && (
          <Button size="sm" variant="secondary" icon={<Plus className="h-3.5 w-3.5" />} onClick={openCreate}>
            Catat kendala
          </Button>
        )
      }
    >
      {own.length === 0 ? (
        <p className="text-[13px] text-muted-foreground">Tidak ada kendala tercatat untuk {supplier.name}.</p>
      ) : (
        <ul className="divide-y divide-border rounded-md border border-border">
          {own.map((i) => (
            <li key={i.id} className="flex items-start gap-3 px-3.5 py-2.5">
              <div className="min-w-0 flex-1">
                <p className="truncate text-[13px] font-medium text-foreground">{i.title}</p>
                <p className="text-[12px] text-muted-foreground">
                  {ISSUE_STATUS_LABELS.staff[i.status]}
                  {i.vendorMilestoneId ? `, tahapan ${vendorMilestones.find((m) => m.id === i.vendorMilestoneId)?.name ?? "-"}` : ""}
                </p>
              </div>
              <IssueImpactBadge impact={i.impact} />
              {canEdit && (
                <IconActionButton
                  icon={Pencil}
                  label={`Ubah kendala ${i.title}`}
                  tone="neutral"
                  onClick={() => setModal({ mode: "edit", issue: i, initial: toIssueUpdateFields(i) })}
                />
              )}
            </li>
          ))}
        </ul>
      )}
      <Link to={ROUTE_PATHS.projectDetail(projectId, "kendala")} className="self-start text-[13px] font-semibold text-primary hover:underline">
        Buka semua kendala proyek
      </Link>
      {modal && (
        <Suspense fallback={null}>
          <IssueFormModal
            open
            onClose={() => setModal(null)}
            onSubmit={submit}
            initialValues={modal.initial}
            mode={modal.mode}
            milestoneLocked={false}
            suppliers={suppliers}
            vendorMilestones={vendorMilestones}
            staff={staff}
          />
        </Suspense>
      )}
    </Section>
  );
}
