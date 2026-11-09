import { create } from "zustand";
import { httpClient } from "@/shared/services/http-client";
import { API } from "@/shared/services/api-endpoints";
import type { ProjectFormValues } from "@/modules/projects/schemas/project.schema";
import type { ProjectMilestoneFormValues } from "@/modules/projects/schemas/project-milestone.schema";
import type { ProjectVendorFormValues } from "@/modules/projects/schemas/project-vendor.schema";
import type { VendorMilestoneFormValues } from "@/modules/projects/schemas/vendor-milestone.schema";
import type { PaymentFormValues, PaymentUpdateFormValues } from "@/modules/projects/schemas/payment.schema";
import type { ClientPaymentFormValues, ClientPaymentUpdateFormValues } from "@/modules/projects/schemas/client-payment.schema";
import type { VenuePaymentFormValues, VenuePaymentUpdateFormValues } from "@/modules/projects/schemas/venue-payment.schema";
import type { IssueFormValues } from "@/modules/projects/schemas/issue.schema";
import type { CompressedFilePayload } from "@/shared/lib/image-compression";
import { uploadFile } from "@/shared/lib/direct-upload";
import type {
  PaymentTerm,
  PaymentTermInput,
  ClosingChecklist,
  ProjectStatusAction,
  ActivityLogEntry,
  ClientPayment,
  Evidence,
  EvidenceRef,
  EvidenceRelatedKind,
  EvidenceType,
  IssueStatus,
  MilestoneStatus,
  Project,
  ProjectMilestone,
  ProjectProgress,
  ProjectVendor,
  ProjectVenueSummary,
  VendorIssue,
  VendorMilestone,
  VendorPayment,
  VenuePayment,
} from "@/modules/projects/types";
import { toPaginationMeta, EMPTY_PAGINATION_META, type PaginationMeta, type RawPaginationMeta } from "@/shared/types/pagination";

/** What a new project can carry besides its own fields: the venue it takes place at and the client's payment schedule. */
export interface NewProjectExtras {
  venueId?: string;
  venueRentalPrice?: number;
  venueCharge?: number;
  terms?: PaymentTermInput[];
}

/** Retrying the same save reuses the key, so the server replays the answer instead of recording the payment twice. */
export interface SaveOptions {
  idempotencyKey?: string;
}

/**
 * Records a payment together with its proof files as ONE business call (PLAN.md
 * E3). Files go to storage first and come back as upload tokens; the payment
 * request then attaches them and the server saves payment + proof in a single
 * transaction. If anything fails nothing is saved, so the form can simply stay
 * open for a retry: there is no "saved, but the proof failed" state.
 */
async function savePaymentWithProof(
  projectId: string,
  url: string,
  body: Record<string, unknown>,
  slots: Array<{ file: File | undefined; type: "Invoice" | "Transfer Proof" }>,
  idempotencyKey?: string,
): Promise<void> {
  const evidence: Array<{ uploadToken: string; type: string }> = [];
  for (const slot of slots) {
    if (slot.file) evidence.push({ uploadToken: await uploadFile(projectId, slot.file), type: slot.type });
  }
  await httpClient.post(url, evidence.length > 0 ? { ...body, evidence } : body, {
    headers: idempotencyKey ? { "Idempotency-Key": idempotencyKey } : undefined,
  });
}

// --- Raw wire shapes (see apps/api .../projects/presentation/dto.go) ---

interface RawMilestoneStats {
  total: number;
  completed: number;
  inProgress: number;
  blocked: number;
  notStarted: number;
  cancelled: number;
  overdue: number;
  ratio: number;
}

interface RawProgress {
  projectMilestoneStats: RawMilestoneStats;
  vendorMilestoneStats: RawMilestoneStats;
  overallPercent: number;
  condition: ProjectProgress["condition"];
  openIssueCount: number;
  criticalOrHighOpenIssueCount: number;
  overdueMilestoneCount: number;
  incompleteEvidenceCount: number;
}

export interface RawProject {
  id: number;
  name: string;
  brideName: string;
  groomName: string;
  eventDate: string;
  venue: string;
  venueId: number | null;
  venueRentalPrice: number | null;
  venueCharge: number | null;
  prepStartDate: string;
  packageName: string;
  contractValue: number;
  status: Project["status"];
  picStaffId: number;
  picSalesStaffId: number;
  picName: string;
  picPhone?: string;
  description: string;
  isArchived: boolean;
  progress?: RawProgress;
}

function toProgress(raw: RawProgress): ProjectProgress {
  return {
    projectMilestoneStats: raw.projectMilestoneStats,
    vendorMilestoneStats: raw.vendorMilestoneStats,
    overallPercent: raw.overallPercent,
    condition: raw.condition,
    openIssueCount: raw.openIssueCount,
    criticalOrHighOpenIssueCount: raw.criticalOrHighOpenIssueCount,
    overdueMilestoneCount: raw.overdueMilestoneCount,
    incompleteEvidenceCount: raw.incompleteEvidenceCount,
  };
}

export function toProject(raw: RawProject): Project {
  return {
    id: String(raw.id),
    name: raw.name,
    brideName: raw.brideName,
    groomName: raw.groomName,
    eventDate: raw.eventDate,
    venue: raw.venue,
    venueId: raw.venueId !== null ? String(raw.venueId) : null,
    venueRentalPrice: raw.venueRentalPrice,
    venueCharge: raw.venueCharge,
    prepStartDate: raw.prepStartDate,
    packageName: raw.packageName,
    contractValue: raw.contractValue,
    status: raw.status,
    picStaffId: raw.picStaffId ? String(raw.picStaffId) : "",
    picSalesStaffId: raw.picSalesStaffId ? String(raw.picSalesStaffId) : "",
    picName: raw.picName,
    picPhone: raw.picPhone,
    description: raw.description,
    isArchived: raw.isArchived,
    progress: raw.progress ? toProgress(raw.progress) : undefined,
  };
}

function projectInputBody(values: ProjectFormValues) {
  return {
    name: values.name,
    brideName: values.brideName,
    groomName: values.groomName,
    eventDate: values.eventDate,
    venue: values.venue,
    prepStartDate: values.prepStartDate,
    packageName: values.packageName,
    contractValue: values.contractValue,
    status: values.status,
    // "" (belum ditugaskan) -> 0, the backend's sentinel for both PIC slots.
    picStaffId: values.picStaffId ? Number(values.picStaffId) : 0,
    picSalesStaffId: values.picSalesStaffId ? Number(values.picSalesStaffId) : 0,
    description: values.description,
  };
}

interface RawProjectVenueSummary {
  id: number;
  name: string;
  address: string | null;
  city: string | null;
  capacity: number | null;
  facilities: string | null;
  socialMedia: string | null;
  hasVisibleAttachment: boolean;
}

function toProjectVenueSummary(raw: RawProjectVenueSummary): ProjectVenueSummary {
  return {
    id: String(raw.id), name: raw.name, address: raw.address, city: raw.city, capacity: raw.capacity,
    facilities: raw.facilities, socialMedia: raw.socialMedia, hasVisibleAttachment: raw.hasVisibleAttachment,
  };
}

interface RawEvidenceRef {
  id: number;
  type: EvidenceRef["type"];
  fileName: string;
  uploadedAt: string;
}

function toEvidenceRefs(raw: RawEvidenceRef[] | null | undefined): EvidenceRef[] {
  return (raw ?? []).map((e) => ({ id: String(e.id), type: e.type, fileName: e.fileName, uploadedAt: e.uploadedAt }));
}

interface RawMilestone {
  id: number;
  order: number;
  name: string;
  status: MilestoneStatus;
  targetDate: string;
  completedDate: string | null;
  evidence?: RawEvidenceRef[] | null;
}

function toMilestone(raw: RawMilestone): ProjectMilestone {
  return {
    id: String(raw.id),
    order: raw.order,
    name: raw.name,
    status: raw.status,
    targetDate: raw.targetDate,
    completedDate: raw.completedDate,
    evidence: toEvidenceRefs(raw.evidence),
  };
}

export interface ProjectMilestoneUpdateFields {
  status: MilestoneStatus;
  targetDate: string;
  completedDate: string;
}

interface RawProjectVendor {
  id: number;
  vendorId: number;
  categoryId: number;
  scope: string;
  contractValue: number;
  pricingTier: ProjectVendor["pricingTier"];
  engagementStatus: ProjectVendor["engagementStatus"];
  bookingDate: string | null;
  eventDate: string;
  eventStartTime: string | null;
  eventEndTime: string | null;
  dpAmount: number;
  paidAmount: number;
  dueDate: string | null;
  picStaffId: number;
  notes: string;
  vendorName?: string;
  categoryName?: string;
  // Only populated by GET .../vendors (listVendorEngagements) — see
  // fetchVendorSection, which reads this instead of issuing its own
  // per-engagement GET .../milestones request.
  milestones?: RawVendorMilestone[];
}

function toProjectVendor(raw: RawProjectVendor): ProjectVendor {
  return {
    id: String(raw.id),
    vendorId: String(raw.vendorId),
    categoryId: String(raw.categoryId),
    scope: raw.scope,
    contractValue: raw.contractValue,
    pricingTier: raw.pricingTier,
    engagementStatus: raw.engagementStatus,
    bookingDate: raw.bookingDate,
    eventDate: raw.eventDate,
    eventStartTime: raw.eventStartTime,
    eventEndTime: raw.eventEndTime,
    dpAmount: raw.dpAmount,
    paidAmount: raw.paidAmount,
    dueDate: raw.dueDate,
    picStaffId: String(raw.picStaffId),
    notes: raw.notes,
    vendorName: raw.vendorName ?? "",
    categoryName: raw.categoryName ?? "",
  };
}

function vendorEngagementInputBody(values: ProjectVendorFormValues) {
  return {
    vendorId: Number(values.vendorId),
    // categoryId/eventDate are not part of the form — callers below fill
    // these in from the vendor's own category and the project's event date.
    categoryId: 0,
    scope: values.scope,
    contractValue: values.contractValue,
    pricingTier: values.pricingTier,
    engagementStatus: values.engagementStatus,
    bookingDate: values.bookingDate || "",
    eventDate: "",
    eventStartTime: values.eventStartTime || null,
    eventEndTime: values.eventEndTime || null,
    dpAmount: values.dpAmount,
    dueDate: values.dueDate || "",
    picStaffId: Number(values.picStaffId),
    notes: values.notes,
  };
}

interface RawVendorMilestone {
  id: number;
  order: number;
  name: string;
  description: string;
  status: MilestoneStatus;
  targetDate: string;
  completedDate: string | null;
  picStaffId: number;
  notes: string;
  evidence?: RawEvidenceRef[] | null;
}

function toVendorMilestone(raw: RawVendorMilestone, projectVendorId: string): VendorMilestone {
  return {
    id: String(raw.id),
    projectVendorId,
    order: raw.order,
    name: raw.name,
    description: raw.description,
    status: raw.status,
    targetDate: raw.targetDate,
    completedDate: raw.completedDate,
    picStaffId: String(raw.picStaffId),
    notes: raw.notes,
    evidence: toEvidenceRefs(raw.evidence),
  };
}

export interface VendorMilestoneUpdateFields {
  status: MilestoneStatus;
  targetDate: string;
  completedDate: string;
  picStaffId: string;
  description: string;
  notes: string;
}

interface RawPayment {
  id: number;
  projectVendorId: number;
  type: VendorPayment["type"];
  amount: number;
  paymentDate: string;
  method: string;
  referenceNumber: string;
  notes: string;
  evidenceComplete: boolean;
}

function toPayment(raw: RawPayment): VendorPayment {
  return {
    id: String(raw.id),
    projectVendorId: String(raw.projectVendorId),
    type: raw.type,
    amount: raw.amount,
    paymentDate: raw.paymentDate,
    method: raw.method,
    referenceNumber: raw.referenceNumber,
    notes: raw.notes,
    evidenceComplete: raw.evidenceComplete,
  };
}

interface RawClientPayment {
  id: number;
  type: ClientPayment["type"];
  amount: number;
  paymentDate: string;
  method: string;
  referenceNumber: string;
  notes: string;
  evidenceComplete: boolean;
  recordedBy?: { name: string } | null;
  verifiedAt?: string | null;
  verifiedBy?: { name: string } | null;
  evidence?: RawEvidenceRef[] | null;
}

function toClientPayment(raw: RawClientPayment): ClientPayment {
  return {
    recordedBy: raw.recordedBy?.name ?? null,
    verifiedAt: raw.verifiedAt ?? null,
    verifiedBy: raw.verifiedBy?.name ?? null,
    evidence: toEvidenceRefs(raw.evidence),
    id: String(raw.id),
    type: raw.type,
    amount: raw.amount,
    paymentDate: raw.paymentDate,
    method: raw.method,
    referenceNumber: raw.referenceNumber,
    notes: raw.notes,
    evidenceComplete: raw.evidenceComplete,
  };
}

interface RawVenuePayment {
  id: number;
  type: VenuePayment["type"];
  amount: number;
  paymentDate: string;
  method: string;
  referenceNumber: string;
  notes: string;
  evidenceComplete: boolean;
}

function toVenuePayment(raw: RawVenuePayment): VenuePayment {
  return {
    id: String(raw.id),
    type: raw.type,
    amount: raw.amount,
    paymentDate: raw.paymentDate,
    method: raw.method,
    referenceNumber: raw.referenceNumber,
    notes: raw.notes,
    evidenceComplete: raw.evidenceComplete,
  };
}

interface RawIssue {
  id: number;
  projectVendorId: number;
  vendorMilestoneId: number | null;
  title: string;
  description: string;
  impact: VendorIssue["impact"];
  foundDate: string;
  status: IssueStatus;
  resolutionPlan: string;
  picStaffId: number;
  targetResolutionDate: string | null;
  resolvedDate: string | null;
  resolutionNotes: string;
  vendorName?: string;
  vendorMilestoneName?: string;
  evidence?: RawEvidenceRef[] | null;
}

function toPaymentTerms(raw: Array<Omit<PaymentTerm, "id"> & { id: number }>): PaymentTerm[] {
  return raw.map((t) => ({ ...t, id: String(t.id) }));
}

function toIssue(raw: RawIssue): VendorIssue {
  return {
    id: String(raw.id),
    projectVendorId: String(raw.projectVendorId),
    vendorMilestoneId: raw.vendorMilestoneId !== null ? String(raw.vendorMilestoneId) : null,
    title: raw.title,
    description: raw.description,
    impact: raw.impact,
    foundDate: raw.foundDate,
    status: raw.status,
    resolutionPlan: raw.resolutionPlan,
    picStaffId: String(raw.picStaffId),
    targetResolutionDate: raw.targetResolutionDate,
    resolvedDate: raw.resolvedDate,
    resolutionNotes: raw.resolutionNotes,
    vendorName: raw.vendorName ?? "",
    vendorMilestoneName: raw.vendorMilestoneName ?? "",
    evidence: toEvidenceRefs(raw.evidence),
  };
}

// toIssueUpdateFields lets the vendor accordion's quick status-change
// dropdown reuse the same full-overwrite updateIssue action -- spread every
// existing field, override just status -- mirroring
// toMilestoneUpdateFields's role for vendor milestones.
export function toIssueUpdateFields(issue: VendorIssue): IssueFormValues {
  return {
    projectVendorId: issue.projectVendorId,
    vendorMilestoneId: issue.vendorMilestoneId ?? "",
    title: issue.title,
    description: issue.description,
    impact: issue.impact,
    foundDate: issue.foundDate,
    resolutionPlan: issue.resolutionPlan,
    picStaffId: issue.picStaffId,
    targetResolutionDate: issue.targetResolutionDate ?? "",
    status: issue.status,
  };
}

interface RawEvidence {
  id: number;
  name: string;
  type: EvidenceType;
  fileName: string;
  documentDate: string | null;
  uploadedAt: string;
  description: string;
  uploadedByStaffId: number;
  relatedKind: EvidenceRelatedKind;
  relatedId: number;
  isClientVisible: boolean;
}

function toEvidence(raw: RawEvidence): Evidence {
  return {
    id: String(raw.id),
    name: raw.name,
    type: raw.type,
    fileName: raw.fileName,
    documentDate: raw.documentDate,
    uploadedAt: raw.uploadedAt,
    description: raw.description,
    uploadedByStaffId: String(raw.uploadedByStaffId),
    relatedKind: raw.relatedKind,
    relatedId: String(raw.relatedId),
    isClientVisible: raw.isClientVisible,
  };
}

export interface UploadEvidenceInput extends CompressedFilePayload {
  name: string;
  type: EvidenceType;
  documentDate: string;
  description: string;
  relatedKind: EvidenceRelatedKind;
  relatedId: string;
  isClientVisible?: boolean;
}

export interface RawActivity {
  id: number;
  type: ActivityLogEntry["type"];
  actorStaffId: number;
  projectId: number | null;
  entityType: string;
  entityId: string;
  entityLabel: string;
  description: string;
  timestamp: string;
}

export function toActivity(raw: RawActivity): ActivityLogEntry {
  return {
    id: String(raw.id),
    type: raw.type,
    actorStaffId: String(raw.actorStaffId),
    projectId: raw.projectId !== null ? String(raw.projectId) : null,
    entityType: raw.entityType,
    entityId: raw.entityId,
    entityLabel: raw.entityLabel,
    description: raw.description,
    timestamp: raw.timestamp,
  };
}

interface ProjectState {
  projects: Project[];
  currentProject: Project | null;
  milestones: ProjectMilestone[];
  vendorEngagements: ProjectVendor[];
  vendorMilestones: VendorMilestone[];
  payments: VendorPayment[];
  clientPayments: ClientPayment[];
  venuePayments: VenuePayment[];
  issues: VendorIssue[];
  evidence: Evidence[];
  // documents is Client Portal's own "Dokumen" tab feed — always only
  // general-kind, client-visible documents (GET /projects/{id}/documents),
  // deliberately its own cache slice rather than a filtered view over
  // `evidence` above (which staff-only screens populate via a different,
  // unfiltered endpoint).
  documents: Evidence[];
  activity: ActivityLogEntry[];

  projectPage: Project[];
  projectPageMeta: PaginationMeta;

  fetchProjects: () => Promise<void>;
  // Backs ProjectListPage's table — real server-side pagination + search/
  // status filtering, separate from the `projects` full-roster cache above
  // (which the dashboard, global search, and client grouping still rely on).
  // showArchived splits active/archived into two disjoint views, never
  // merged (see ADR-0013) — omit or pass false for the normal active view.
  fetchProjectPage: (page: number, search: string, status: string, showArchived?: boolean) => Promise<void>;
  /**
   * One call creates the project together with its venue and payment schedule (PLAN.md E7): all or nothing, every
   * problem reported at once. Retrying with the same Idempotency-Key returns the project the first try made.
   */
  createProject: (values: ProjectFormValues, extra?: NewProjectExtras, options?: SaveOptions) => Promise<Project>;
  updateProject: (id: string, values: ProjectFormValues) => Promise<Project>;
  // Attaches/detaches this project's structured venue (ADR-0016) — pass
  // null to detach. Sends a full-replace PATCH body sourced from the
  // already-loaded currentProject (same wire shape as updateProject), since
  // the backend's Update endpoint doesn't support a true partial patch.
  // rentalPrice/charge are the per-project cost snapshot (PLAN.md
  // "Financial Calculation Correctness") -- omitted/ignored on detach
  // (venueId: null), since the backend force-clears them regardless.
  setProjectVenue: (
    id: string,
    venueId: string | null,
    rentalPrice?: number | null,
    charge?: number | null
  ) => Promise<Project>;
  // Public-safe summary (ADR-0016) — not cached in store state, same
  // "return directly, don't stash" convention as fetchVendorProjectHistory,
  // since it's only ever needed by whichever single tab is currently open.
  fetchProjectVenueSummary: (id: string) => Promise<ProjectVenueSummary | null>;
  cancelProject: (id: string) => Promise<void>;
  // One explicit step of the project's life (Draf → Persiapan → Siap →
  // Selesai, or reopen). The API refuses a step that does not fit the
  // current status, so a stale page cannot skip one.
  transitionProject: (id: string, action: ProjectStatusAction) => Promise<void>;
  fetchClosingChecklist: (id: string) => Promise<ClosingChecklist>;

  // The client's payment schedule (installments); status comes from the API.
  paymentTerms: PaymentTerm[];
  /** The project the loaded schedule belongs to (null until it has loaded). */
  paymentTermsFor: string | null;
  fetchPaymentTerms: (projectId: string) => Promise<void>;
  savePaymentTerms: (projectId: string, terms: PaymentTermInput[]) => Promise<void>;
  toggleArchiveProject: (id: string) => Promise<void>;
  // Hard delete (ADR-0013) — Owner-only and requires the project already be
  // archived or cancelled, both enforced server-side, not just by the UI
  // hiding the button.
  deleteProject: (id: string) => Promise<void>;
  // Duplicate (ADR-0014) — clones the source project's milestones and vendor
  // lineup into a brand-new project; `values` is the new project's own
  // (possibly user-edited) fields, same shape as create.
  duplicateProject: (sourceId: string, values: ProjectFormValues) => Promise<Project>;

  fetchProjectDetail: (projectId: string) => Promise<void>;
  fetchMyProject: () => Promise<string>;

  fetchMilestones: (projectId: string) => Promise<void>;
  createMilestone: (projectId: string, values: ProjectMilestoneFormValues) => Promise<void>;
  updateMilestoneStatus: (projectId: string, milestoneId: string, status: MilestoneStatus) => Promise<void>;
  updateMilestone: (projectId: string, milestoneId: string, fields: ProjectMilestoneUpdateFields) => Promise<void>;
  reorderMilestones: (projectId: string, orderedIds: string[]) => Promise<void>;

  fetchVendorSection: (projectId: string) => Promise<void>;
  createVendorEngagement: (projectId: string, values: ProjectVendorFormValues) => Promise<void>;
  updateVendorEngagement: (projectId: string, pvId: string, values: ProjectVendorFormValues) => Promise<void>;
  cancelVendorEngagement: (projectId: string, pvId: string) => Promise<void>;
  createVendorMilestone: (projectId: string, pvId: string, values: VendorMilestoneFormValues) => Promise<void>;
  updateVendorMilestone: (
    projectId: string,
    pvId: string,
    milestoneId: string,
    fields: VendorMilestoneUpdateFields
  ) => Promise<void>;

  fetchPayments: (projectId: string) => Promise<void>;
  createPayment: (projectId: string, values: PaymentFormValues, options?: SaveOptions) => Promise<void>;
  // Edit scope is the ledger fields only (Type/Amount/PaymentDate/Method/
  // ReferenceNumber/Notes) — no file re-upload, no re-parenting to a
  // different vendor engagement (PLAN.md §2.5). Both also refetch
  // `fetchVendorSection` alongside `fetchPayments` — PLAN.md §2.7's fix for
  // the pre-existing `pv.paidAmount` staleness bug (that cached field lives
  // on a sibling array populated only by fetchVendorSection, not derived
  // from `payments` itself).
  updatePayment: (projectId: string, paymentId: string, values: PaymentUpdateFormValues) => Promise<void>;
  deletePayment: (projectId: string, paymentId: string) => Promise<void>;

  fetchClientPayments: (projectId: string) => Promise<void>;
  // Orchestrates POST .../client-payments then, only if values.proofFile is
  // set, compress+POST .../evidence with relatedKind "clientPayment" — one
  // store action instead of a separate manual "Tambah Evidence" step (PLAN.md
  // §1.4), closing the "always incomplete in practice" gap vendor_payments
  // has today (createPayment above never attaches evidence at creation time).
  createClientPayment: (projectId: string, values: ClientPaymentFormValues, options?: SaveOptions) => Promise<void>;
  updateClientPayment: (projectId: string, paymentId: string, values: ClientPaymentUpdateFormValues) => Promise<void>;
  deleteClientPayment: (projectId: string, paymentId: string) => Promise<void>;
  /** The WO confirms the money arrived (E4). Needs a proof file; the client then sees the stamp. */
  verifyClientPayment: (projectId: string, paymentId: string) => Promise<void>;

  fetchVenuePayments: (projectId: string) => Promise<void>;
  // Same two-slot evidence orchestration as createPayment (vendor's own),
  // just against .../venue-payments and relatedKind "venuePayment".
  createVenuePayment: (projectId: string, values: VenuePaymentFormValues, options?: SaveOptions) => Promise<void>;
  updateVenuePayment: (projectId: string, paymentId: string, values: VenuePaymentUpdateFormValues) => Promise<void>;
  deleteVenuePayment: (projectId: string, paymentId: string) => Promise<void>;

  fetchIssues: (projectId: string) => Promise<void>;
  createIssue: (projectId: string, values: IssueFormValues) => Promise<void>;
  updateIssue: (projectId: string, issueId: string, values: IssueFormValues) => Promise<void>;

  fetchEvidence: (projectId: string) => Promise<void>;
  uploadEvidence: (projectId: string, values: UploadEvidenceInput) => Promise<void>;
  toggleEvidenceClientVisible: (projectId: string, evidenceId: string) => Promise<void>;
  fetchDocuments: (projectId: string) => Promise<void>;

  fetchActivity: (projectId: string) => Promise<void>;
}

// Backed by the real `projects` module (Fase 4) — tenant-scoped, fetch-then-set
// (ADR-0009). Every fetch* function replaces its slice wholesale; there is no
// per-project namespacing in the store itself, so callers always re-fetch
// when the viewed project changes (matches how the tab pages remount).
export const useProjectStore = create<ProjectState>((set, get) => ({
  projects: [],
  currentProject: null,
  milestones: [],
  vendorEngagements: [],
  vendorMilestones: [],
  payments: [],
  clientPayments: [],
  venuePayments: [],
  issues: [],
  paymentTerms: [],
  paymentTermsFor: null,
  evidence: [],
  documents: [],
  activity: [],

  projectPage: [],
  projectPageMeta: EMPTY_PAGINATION_META,

  fetchProjects: async () => {
    const res = await httpClient.get(API.projects.base, { params: { all: true } });
    const list = (res.data.data as RawProject[]).map(toProject);
    set({ projects: list });
  },

  fetchProjectPage: async (page, search, status, showArchived = false) => {
    const res = await httpClient.get(API.projects.base, {
      params: { page, search: search || undefined, status: status || undefined, archived: showArchived || undefined },
    });
    const list = (res.data.data as RawProject[]).map(toProject);
    set({ projectPage: list, projectPageMeta: toPaginationMeta(res.data.meta as RawPaginationMeta) });
  },

  createProject: async (values, extra, options) => {
    const body: Record<string, unknown> = projectInputBody(values);
    if (extra?.venueId) {
      body.venueId = Number(extra.venueId);
      body.venueRentalPrice = extra.venueRentalPrice;
      body.venueCharge = extra.venueCharge;
    }
    if (extra?.terms && extra.terms.length > 0) body.terms = extra.terms;
    const res = await httpClient.post(API.projects.base, body, {
      headers: options?.idempotencyKey ? { "Idempotency-Key": options.idempotencyKey } : undefined,
    });
    const project = toProject(res.data.data as RawProject);
    // The project exists now. Refreshing the list is a courtesy: if it fails,
    // the caller must still see success, or the user retries and gets a duplicate.
    await get().fetchProjects().catch(() => undefined);
    return project;
  },

  updateProject: async (id, values) => {
    const res = await httpClient.patch(API.projects.item(id), projectInputBody(values));
    const project = toProject(res.data.data as RawProject);
    set((state) => ({
      projects: state.projects.map((p) => (p.id === id ? project : p)),
      currentProject: state.currentProject?.id === id ? { ...project, progress: state.currentProject.progress } : state.currentProject,
    }));
    return project;
  },

  setProjectVenue: async (id, venueId, rentalPrice, charge) => {
    const current = get().currentProject;
    if (!current || current.id !== id) {
      throw new Error("Project belum dimuat");
    }
    const res = await httpClient.patch(API.projects.item(id), {
      name: current.name,
      brideName: current.brideName,
      groomName: current.groomName,
      eventDate: current.eventDate,
      venue: current.venue,
      prepStartDate: current.prepStartDate,
      packageName: current.packageName,
      contractValue: current.contractValue,
      status: current.status,
      picStaffId: current.picStaffId ? Number(current.picStaffId) : 0,
      picSalesStaffId: current.picSalesStaffId ? Number(current.picSalesStaffId) : 0,
      description: current.description,
      // 0 is never a real venue id (AUTO_INCREMENT starts at 1) — the
      // backend's sentinel for "detach" (see ADR-0016). The backend also
      // force-clears venueRentalPrice/venueCharge on detach regardless of
      // what's sent here — these are just the belt-and-suspenders mirror.
      venueId: venueId !== null ? Number(venueId) : 0,
      venueRentalPrice: venueId !== null ? rentalPrice ?? null : null,
      venueCharge: venueId !== null ? charge ?? null : null,
    });
    const project = toProject(res.data.data as RawProject);
    set((state) => ({
      projects: state.projects.map((p) => (p.id === id ? project : p)),
      currentProject: state.currentProject?.id === id ? { ...project, progress: state.currentProject.progress } : state.currentProject,
    }));
    return project;
  },

  fetchProjectVenueSummary: async (id) => {
    const res = await httpClient.get(API.projects.venue(id));
    if (!res.data.data) return null;
    return toProjectVenueSummary(res.data.data as RawProjectVenueSummary);
  },

  transitionProject: async (id, action) => {
    const res = await httpClient.post(API.projects.transition(id), { action });
    const project = toProject(res.data.data as RawProject);
    set((state) => ({
      projects: state.projects.map((p) => (p.id === id ? project : p)),
      currentProject: state.currentProject?.id === id ? { ...project, progress: state.currentProject.progress } : state.currentProject,
    }));
  },

  fetchPaymentTerms: async (projectId) => {
    const res = await httpClient.get(API.projects.paymentTerms(projectId));
    set({ paymentTerms: toPaymentTerms(res.data.data), paymentTermsFor: projectId });
  },

  savePaymentTerms: async (projectId, terms) => {
    const res = await httpClient.put(API.projects.paymentTerms(projectId), { terms });
    set({ paymentTerms: toPaymentTerms(res.data.data), paymentTermsFor: projectId });
  },

  fetchClosingChecklist: async (id) => {
    const res = await httpClient.get(API.projects.closingChecklist(id));
    return res.data.data as ClosingChecklist;
  },

  cancelProject: async (id) => {
    const res = await httpClient.post(API.projects.cancel(id));
    const project = toProject(res.data.data as RawProject);
    set((state) => ({
      projects: state.projects.map((p) => (p.id === id ? project : p)),
      currentProject: state.currentProject?.id === id ? { ...project, progress: state.currentProject.progress } : state.currentProject,
    }));
  },

  toggleArchiveProject: async (id) => {
    const res = await httpClient.post(API.projects.toggleArchive(id));
    const project = toProject(res.data.data as RawProject);
    set((state) => ({
      projects: state.projects.map((p) => (p.id === id ? project : p)),
      currentProject: state.currentProject?.id === id ? { ...project, progress: state.currentProject.progress } : state.currentProject,
    }));
  },

  deleteProject: async (id) => {
    await httpClient.delete(API.projects.item(id));
    set((state) => ({
      projects: state.projects.filter((p) => p.id !== id),
      currentProject: state.currentProject?.id === id ? null : state.currentProject,
    }));
  },

  duplicateProject: async (sourceId, values) => {
    const res = await httpClient.post(API.projects.duplicate(sourceId), projectInputBody(values));
    const project = toProject(res.data.data as RawProject);
    await get().fetchProjects();
    return project;
  },

  fetchProjectDetail: async (projectId) => {
    // vendorEngagements/vendorMilestones are separately fetched (fetchVendorSection)
    // and, unlike currentProject, never scoped by project id in the store —
    // cleared here, synchronously, the moment a project switch starts so a
    // still-loading ProjectHeaderCard can never compute its Margin/Keuntungan
    // (or anything else) against the PREVIOUS project's vendor costs while
    // this project's own fetchVendorSection call is still in flight.
    // Everything scoped to a project is cleared with it: the side panel, the
    // Ringkasan and the schedule must never show (or, worse, edit) the previous
    // project's milestones, payments, issues or installments.
    set({
      vendorEngagements: [], vendorMilestones: [], milestones: [], payments: [], clientPayments: [], venuePayments: [],
      evidence: [], issues: [], paymentTerms: [], paymentTermsFor: null,
    });
    const res = await httpClient.get(API.projects.item(projectId));
    set({ currentProject: toProject(res.data.data as RawProject) });
  },

  // Client Portal's entry point (Fase 6) — a client principal has no other
  // way to learn its own project id, so this resolves + returns it in one
  // call (mirrors GET /projects/me on the backend).
  fetchMyProject: async () => {
    const res = await httpClient.get(API.projects.me);
    const project = toProject(res.data.data as RawProject);
    set({ currentProject: project });
    return project.id;
  },

  fetchMilestones: async (projectId) => {
    const res = await httpClient.get(API.projects.milestones(projectId));
    set({ milestones: (res.data.data as RawMilestone[]).map(toMilestone) });
  },

  createMilestone: async (projectId, values) => {
    await httpClient.post(API.projects.milestones(projectId), values);
    await get().fetchMilestones(projectId);
  },

  updateMilestoneStatus: async (projectId, milestoneId, status) => {
    // The backend endpoint is a full update (mirrors vendor milestones) — the
    // quick inline status dropdown resends this milestone's current dates
    // unchanged alongside the new status, rather than needing its own
    // partial-update endpoint.
    const existing = get().milestones.find((m) => m.id === milestoneId);
    await httpClient.patch(API.projects.milestone(projectId, milestoneId), {
      status,
      targetDate: existing?.targetDate ?? "",
      completedDate: existing?.completedDate ?? "",
    });
    await get().fetchMilestones(projectId);
  },

  updateMilestone: async (projectId, milestoneId, fields) => {
    await httpClient.patch(API.projects.milestone(projectId, milestoneId), {
      status: fields.status,
      targetDate: fields.targetDate,
      completedDate: fields.completedDate,
    });
    await get().fetchMilestones(projectId);
  },

  reorderMilestones: async (projectId, orderedIds) => {
    await httpClient.patch(API.projects.milestones(projectId), { orderedIds: orderedIds.map(Number) });
    await get().fetchMilestones(projectId);
  },

  fetchVendorSection: async (projectId) => {
    const res = await httpClient.get(API.projects.vendors(projectId));
    const raw = res.data.data as RawProjectVendor[];
    const engagements = raw.map(toProjectVendor);
    const allMilestones = raw.flatMap((pv) => (pv.milestones ?? []).map((m) => toVendorMilestone(m, String(pv.id))));
    set({ vendorEngagements: engagements, vendorMilestones: allMilestones });
  },

  createVendorEngagement: async (projectId, values) => {
    // The category comes from the vendor and the date from the project: the server derives both (E7), so a stale
    // page cannot send a date the project no longer has.
    await httpClient.post(API.projects.vendors(projectId), vendorEngagementInputBody(values));
    await get().fetchVendorSection(projectId);
  },

  updateVendorEngagement: async (projectId, pvId, values) => {
    const existing = get().vendorEngagements.find((pv) => pv.id === pvId);
    await httpClient.patch(API.projects.vendor(projectId, pvId), {
      ...vendorEngagementInputBody(values),
      categoryId: existing ? Number(existing.categoryId) : 0,
      eventDate: existing?.eventDate ?? "",
    });
    await get().fetchVendorSection(projectId);
  },

  cancelVendorEngagement: async (projectId, pvId) => {
    await httpClient.post(API.projects.vendorCancel(projectId, pvId));
    await get().fetchVendorSection(projectId);
  },

  createVendorMilestone: async (projectId, pvId, values) => {
    await httpClient.post(API.projects.vendorMilestones(projectId, pvId), {
      name: values.name,
      description: values.description,
      targetDate: values.targetDate,
      picStaffId: Number(values.picStaffId),
    });
    await get().fetchVendorSection(projectId);
  },

  updateVendorMilestone: async (projectId, pvId, milestoneId, fields) => {
    await httpClient.patch(API.projects.vendorMilestone(projectId, pvId, milestoneId), {
      status: fields.status,
      targetDate: fields.targetDate,
      completedDate: fields.completedDate,
      picStaffId: Number(fields.picStaffId),
      description: fields.description,
      notes: fields.notes,
    });
    await get().fetchVendorSection(projectId);
  },

  fetchPayments: async (projectId) => {
    const res = await httpClient.get(API.projects.payments(projectId));
    set({ payments: (res.data.data as RawPayment[]).map(toPayment) });
  },

  createPayment: async (projectId, values, options) => {
    await savePaymentWithProof(
      projectId,
      API.projects.payments(projectId),
      {
        projectVendorId: Number(values.projectVendorId),
        type: values.type,
        amount: values.amount,
        paymentDate: values.paymentDate,
        method: values.method,
        referenceNumber: values.referenceNumber,
        notes: values.notes,
      },
      [
        { file: values.invoiceFile, type: "Invoice" },
        { file: values.proofFile, type: "Transfer Proof" },
      ],
      options?.idempotencyKey,
    );
    // The engagement's paid amount is part of the vendor section.
    await Promise.all([get().fetchPayments(projectId), get().fetchVendorSection(projectId)]);
  },

  updatePayment: async (projectId, paymentId, values) => {
    await httpClient.patch(API.projects.payment(projectId, paymentId), {
      type: values.type,
      amount: values.amount,
      paymentDate: values.paymentDate,
      method: values.method,
      referenceNumber: values.referenceNumber,
      notes: values.notes,
    });
    await Promise.all([get().fetchPayments(projectId), get().fetchVendorSection(projectId)]);
  },

  deletePayment: async (projectId, paymentId) => {
    await httpClient.delete(API.projects.payment(projectId, paymentId));
    await Promise.all([get().fetchPayments(projectId), get().fetchVendorSection(projectId)]);
  },

  fetchClientPayments: async (projectId) => {
    const res = await httpClient.get(API.projects.clientPayments(projectId));
    set({ clientPayments: (res.data.data as RawClientPayment[]).map(toClientPayment) });
  },

  createClientPayment: async (projectId, values, options) => {
    await savePaymentWithProof(
      projectId,
      API.projects.clientPayments(projectId),
      {
        type: values.type,
        amount: values.amount,
        paymentDate: values.paymentDate,
        method: values.method,
        referenceNumber: values.referenceNumber,
        notes: values.notes,
      },
      [{ file: values.proofFile, type: "Transfer Proof" }],
      options?.idempotencyKey,
    );
    await get().fetchClientPayments(projectId);
  },

  updateClientPayment: async (projectId, paymentId, values) => {
    await httpClient.patch(API.projects.clientPayment(projectId, paymentId), {
      type: values.type,
      amount: values.amount,
      paymentDate: values.paymentDate,
      method: values.method,
      referenceNumber: values.referenceNumber,
      notes: values.notes,
    });
    await get().fetchClientPayments(projectId);
  },

  deleteClientPayment: async (projectId, paymentId) => {
    await httpClient.delete(API.projects.clientPayment(projectId, paymentId));
    await get().fetchClientPayments(projectId);
  },

  verifyClientPayment: async (projectId, paymentId) => {
    await httpClient.post(API.projects.clientPaymentVerify(projectId, paymentId));
    await get().fetchClientPayments(projectId);
  },

  fetchVenuePayments: async (projectId) => {
    const res = await httpClient.get(API.projects.venuePayments(projectId));
    set({ venuePayments: (res.data.data as RawVenuePayment[]).map(toVenuePayment) });
  },

  createVenuePayment: async (projectId, values, options) => {
    await savePaymentWithProof(
      projectId,
      API.projects.venuePayments(projectId),
      {
        type: values.type,
        amount: values.amount,
        paymentDate: values.paymentDate,
        method: values.method,
        referenceNumber: values.referenceNumber,
        notes: values.notes,
      },
      [
        { file: values.invoiceFile, type: "Invoice" },
        { file: values.proofFile, type: "Transfer Proof" },
      ],
      options?.idempotencyKey,
    );
    await get().fetchVenuePayments(projectId);
  },

  updateVenuePayment: async (projectId, paymentId, values) => {
    await httpClient.patch(API.projects.venuePayment(projectId, paymentId), {
      type: values.type,
      amount: values.amount,
      paymentDate: values.paymentDate,
      method: values.method,
      referenceNumber: values.referenceNumber,
      notes: values.notes,
    });
    await get().fetchVenuePayments(projectId);
  },

  deleteVenuePayment: async (projectId, paymentId) => {
    await httpClient.delete(API.projects.venuePayment(projectId, paymentId));
    await get().fetchVenuePayments(projectId);
  },

  fetchIssues: async (projectId) => {
    const res = await httpClient.get(API.projects.issues(projectId));
    set({ issues: (res.data.data as RawIssue[]).map(toIssue) });
  },

  createIssue: async (projectId, values) => {
    await httpClient.post(API.projects.issues(projectId), {
      ...values,
      projectVendorId: Number(values.projectVendorId),
      vendorMilestoneId: values.vendorMilestoneId ? Number(values.vendorMilestoneId) : null,
      picStaffId: Number(values.picStaffId),
    });
    await get().fetchIssues(projectId);
  },

  updateIssue: async (projectId, issueId, values) => {
    await httpClient.patch(API.projects.issue(projectId, issueId), {
      ...values,
      projectVendorId: Number(values.projectVendorId),
      vendorMilestoneId: values.vendorMilestoneId ? Number(values.vendorMilestoneId) : null,
      picStaffId: Number(values.picStaffId),
    });
    await get().fetchIssues(projectId);
  },

  fetchEvidence: async (projectId) => {
    const res = await httpClient.get(API.projects.evidence(projectId));
    set({ evidence: (res.data.data as RawEvidence[]).map(toEvidence) });
  },

  uploadEvidence: async (projectId, values) => {
    await httpClient.post(API.projects.evidence(projectId), {
      name: values.name,
      type: values.type,
      fileName: values.fileName,
      mimeType: values.mimeType,
      base64Data: values.base64Data,
      documentDate: values.documentDate,
      description: values.description,
      relatedKind: values.relatedKind,
      relatedId: values.relatedId ? Number(values.relatedId) : 0,
      isClientVisible: values.isClientVisible ?? false,
    });
    await get().fetchEvidence(projectId);
  },

  toggleEvidenceClientVisible: async (projectId, evidenceId) => {
    await httpClient.post(API.projects.evidenceToggleClientVisible(projectId, evidenceId));
    await get().fetchEvidence(projectId);
  },

  fetchDocuments: async (projectId) => {
    const res = await httpClient.get(API.projects.documents(projectId));
    set({ documents: (res.data.data as RawEvidence[]).map(toEvidence) });
  },

  fetchActivity: async (projectId) => {
    const res = await httpClient.get(API.projects.activity(projectId));
    set({ activity: (res.data.data as RawActivity[]).map(toActivity) });
  },
}));
