import { create } from "zustand";
import { ifMatch } from "@/shared/lib/concurrency";
import { httpClient, onProjectWrite } from "@/shared/services/http-client";
import { API } from "@/shared/services/api-endpoints";
import type { ProjectFormValues } from "@/modules/projects/schemas/project.schema";
import type { ProjectMilestoneFormValues } from "@/modules/projects/schemas/project-milestone.schema";
import { lineDraftBody, toRincian, type RawRincian, type SupplierFormValues } from "@/modules/rincian";
import type { VendorMilestoneFormValues } from "@/modules/projects/schemas/vendor-milestone.schema";
import type { SupplierPaymentFormValues, SupplierPaymentUpdateFormValues } from "@/modules/projects/schemas/payment.schema";
import type { ClientPaymentFormValues, ClientPaymentUpdateFormValues } from "@/modules/projects/schemas/client-payment.schema";
import type { IssueFormValues } from "@/modules/projects/schemas/issue.schema";
import type { CompressedFilePayload } from "@/shared/lib/image-compression";
import { uploadFile } from "@/shared/lib/direct-upload";
import { renameApiFieldErrors } from "@/shared/lib/api-error";
import type {
  OverviewAction,
  PaymentTerm,
  PaymentTermInput,
  ProjectOverview,
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
  ProjectListItem,
  ProjectMilestone,
  ProjectProgress,
  ProjectEvent,
  EventInput,
  Rincian,
  LineDraft,
  ProjectSupplier,
  SupplierPayment,
  VendorIssue,
  VendorMilestone,
} from "@/modules/projects/types";
import { toPaginationMeta, EMPTY_PAGINATION_META, type PaginationMeta, type RawPaginationMeta } from "@/shared/types/pagination";

/** What a new project carries besides its own fields: its events (at least one; the first is the main event). */
export interface NewProjectExtras {
  events: EventInput[];
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
  // The form names its file inputs; the server names them by position in `evidence`.
  const fieldOf = (type: "Invoice" | "Transfer Proof") => (type === "Invoice" ? "invoiceFile" : "proofFile");
  const evidence: Array<{ uploadToken: string; type: string }> = [];
  const fieldByIndex: string[] = [];
  for (const slot of slots) {
    if (!slot.file) continue;
    try {
      evidence.push({ uploadToken: await uploadFile(projectId, slot.file), type: slot.type });
    } catch (err) {
      // Whatever the upload refused (type, size, content) is about this one file.
      throw renameApiFieldErrors(err, () => fieldOf(slot.type));
    }
    fieldByIndex.push(fieldOf(slot.type));
  }
  try {
    await httpClient.post(url, evidence.length > 0 ? { ...body, evidence } : body, {
      headers: idempotencyKey ? { "Idempotency-Key": idempotencyKey } : undefined,
    });
  } catch (err) {
    // `evidence[1].uploadToken` → the input that file came from, so the message shows under it (EP-01).
    throw renameApiFieldErrors(err, (key) => {
      const m = /^evidence\[(\d+)\](?:\.uploadToken)?$/.exec(key);
      return m ? fieldByIndex[Number(m[1])] : undefined;
    });
  }
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
  /** The version for `If-Match` (ADR-0036). */
  updatedAt?: string;
  id: number;
  name: string;
  brideName: string;
  groomName: string;
  eventDate: string;
  prepStartDate: string;
  packageNames?: string[] | null;
  contractValue: number;
  contractSubtotal?: number;
  contractTax?: number;
  isPkp?: boolean;
  linesVersion?: string;
  status: Project["status"];
  picStaffId: number;
  picSalesStaffId: number;
  picName: string;
  picPhone?: string;
  description: string;
  isArchived: boolean;
  progress?: RawProgress;
  coverUrl?: string | null;
  location?: string;
  eventsLine?: string;
  quotation?: { id: number | string; number: string; revisionN: number } | null;
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
    updatedAt: raw.updatedAt ?? "",
    id: String(raw.id),
    name: raw.name,
    brideName: raw.brideName,
    groomName: raw.groomName,
    eventDate: raw.eventDate,
    prepStartDate: raw.prepStartDate,
    packageNames: raw.packageNames ?? [],
    contractValue: raw.contractValue,
    contractSubtotal: raw.contractSubtotal ?? 0,
    contractTax: raw.contractTax ?? 0,
    isPkp: raw.isPkp ?? false,
    linesVersion: raw.linesVersion ?? "",
    status: raw.status,
    picStaffId: raw.picStaffId ? String(raw.picStaffId) : "",
    picSalesStaffId: raw.picSalesStaffId ? String(raw.picSalesStaffId) : "",
    picName: raw.picName,
    picPhone: raw.picPhone,
    description: raw.description,
    isArchived: raw.isArchived,
    progress: raw.progress ? toProgress(raw.progress) : undefined,
    coverUrl: raw.coverUrl ?? null,
    location: raw.location ?? "",
    eventsLine: raw.eventsLine ?? "",
    quotation: raw.quotation ? { id: String(raw.quotation.id), number: raw.quotation.number, revisionN: raw.quotation.revisionN } : null,
  };
}

/** A row of GET /projects (paginated): the project plus what the client still owes and the next installment due. */
interface RawProjectListItem extends RawProject {
  remaining: number;
  nextDueDate: string | null;
}

function toProjectListItem(raw: RawProjectListItem): ProjectListItem {
  return { ...toProject(raw), remaining: raw.remaining ?? 0, nextDueDate: raw.nextDueDate ?? null };
}

/** The project's own fields as PATCH and duplicate take them (the date only matters to duplicate). */
function projectInputBody(values: ProjectFormValues) {
  return {
    name: values.name,
    brideName: values.brideName,
    groomName: values.groomName,
    prepStartDate: values.prepStartDate,
    status: values.status,
    // "" (belum ditugaskan) -> 0, the backend's sentinel for both PIC slots.
    picStaffId: values.picStaffId ? Number(values.picStaffId) : 0,
    picSalesStaffId: values.picSalesStaffId ? Number(values.picSalesStaffId) : 0,
    description: values.description,
  };
}

interface RawEventSession {
  id: number;
  label: string;
  startTime: string | null;
  endTime: string | null;
  guests: number;
}

interface RawEvent {
  updatedAt?: string;
  id: number;
  eventKindId: number;
  kindName: string;
  eventDate: string;
  startTime: string | null;
  endTime: string | null;
  venue: {
    supplierId?: number;
    venueId?: number;
    name: string;
    capacityStanding?: number | null;
    capacitySeated?: number | null;
    address: string | null;
    city: string | null;
  } | null;
  layout: ProjectEvent["layout"];
  isMain: boolean;
  order: number;
  sessions: RawEventSession[] | null;
  totalGuests: number;
}

export function toEvent(raw: RawEvent): ProjectEvent {
  return {
    updatedAt: raw.updatedAt ?? "",
    id: String(raw.id),
    eventKindId: String(raw.eventKindId),
    kindName: raw.kindName,
    eventDate: raw.eventDate,
    startTime: raw.startTime,
    endTime: raw.endTime,
    venue: raw.venue
      ? {
          supplierId: raw.venue.supplierId ? String(raw.venue.supplierId) : "",
          venueId: raw.venue.venueId ? String(raw.venue.venueId) : "",
          name: raw.venue.name,
          capacityStanding: raw.venue.capacityStanding ?? null,
          capacitySeated: raw.venue.capacitySeated ?? null,
          address: raw.venue.address,
          city: raw.venue.city,
        }
      : null,
    layout: raw.layout,
    isMain: raw.isMain,
    order: raw.order,
    sessions: (raw.sessions ?? []).map((x) => ({ id: String(x.id), label: x.label, startTime: x.startTime, endTime: x.endTime, guests: x.guests })),
    totalGuests: raw.totalGuests,
  };
}

/** An event as the API takes it: the whole event and its sessions. */
export function eventInputBody(e: EventInput) {
  return {
    eventKindId: Number(e.eventKindId),
    eventDate: e.eventDate,
    startTime: e.startTime || null,
    endTime: e.endTime || null,
    venueId: e.venueId ? Number(e.venueId) : null,
    layout: e.layout,
    sessions: e.sessions.map((x) => ({ label: x.label, startTime: x.startTime || null, endTime: x.endTime || null, guests: x.guests })),
  };
}

/** What an event write answers next to the event's own keys: the project's date, which the main event carries. */
interface EventWriteResult {
  projectEventDate?: string;
}

export interface RawEvidenceRef {
  id: number;
  type: EvidenceRef["type"];
  fileName: string;
  uploadedAt: string;
}

export function toEvidenceRefs(raw: RawEvidenceRef[] | null | undefined): EvidenceRef[] {
  return (raw ?? []).map((e) => ({ id: String(e.id), type: e.type, fileName: e.fileName, uploadedAt: e.uploadedAt }));
}

interface RawMilestone {
  /** The version for `If-Match` (ADR-0036). */
  updatedAt?: string;
  id: number;
  order: number;
  name: string;
  status: MilestoneStatus;
  targetDate: string;
  completedDate: string | null;
  daysBeforeEvent?: number | null;
  forClient?: boolean;
  completedByClient?: boolean;
  evidence?: RawEvidenceRef[] | null;
}

function toMilestone(raw: RawMilestone): ProjectMilestone {
  return {
    updatedAt: raw.updatedAt ?? "",
    id: String(raw.id),
    order: raw.order,
    name: raw.name,
    status: raw.status,
    targetDate: raw.targetDate,
    completedDate: raw.completedDate,
    daysBeforeEvent: raw.daysBeforeEvent ?? null,
    forClient: raw.forClient ?? false,
    completedByClient: raw.completedByClient ?? false,
    evidence: toEvidenceRefs(raw.evidence),
  };
}

export interface ProjectMilestoneUpdateFields {
  status: MilestoneStatus;
  targetDate: string;
  completedDate: string;
  forClient: boolean;
}

interface RawSupplier {
  updatedAt?: string;
  id: number;
  kind: ProjectSupplier["kind"];
  supplierId?: number;
  name: string;
  categoryId: number;
  categoryName: string;
  paidBy: ProjectSupplier["paidBy"];
  showName: boolean;
  capacityStanding?: number | null;
  capacitySeated?: number | null;
  agreedCost: number | null;
  paidWo: number | null;
  paidClient: number | null;
  engagementStatus: ProjectSupplier["engagementStatus"];
  bookingDate: string | null;
  dueDate: string | null;
  picStaffId: number;
  picName?: string;
  scope: string;
  notes: string;
  paidByLocked?: string;
  usedByEvents?: number;
  evidenceCount?: number;
  milestones?: RawVendorMilestone[] | null;
}

function toSupplier(raw: RawSupplier): ProjectSupplier {
  return {
    updatedAt: raw.updatedAt ?? "",
    id: String(raw.id),
    kind: raw.kind,
    supplierId: raw.supplierId ? String(raw.supplierId) : "",
    name: raw.name,
    categoryId: String(raw.categoryId),
    categoryName: raw.categoryName,
    paidBy: raw.paidBy,
    showName: raw.showName,
    capacityStanding: raw.capacityStanding ?? null,
    capacitySeated: raw.capacitySeated ?? null,
    agreedCost: raw.agreedCost ?? null,
    paidWo: raw.paidWo ?? null,
    paidClient: raw.paidClient ?? null,
    engagementStatus: raw.engagementStatus,
    bookingDate: raw.bookingDate,
    dueDate: raw.dueDate,
    picStaffId: raw.picStaffId ? String(raw.picStaffId) : "",
    picName: raw.picName ?? "",
    scope: raw.scope,
    notes: raw.notes,
    paidByLocked: raw.paidByLocked ?? "",
    usedByEvents: raw.usedByEvents ?? 0,
    evidenceCount: raw.evidenceCount ?? 0,
  };
}

/** The money and status fields of a supplier form; the agreed cost only when the caller may see it. */
function supplierBody(values: SupplierFormValues, withCost: boolean) {
  return {
    paidBy: values.paidBy,
    showName: values.paidBy === "WO" ? values.showName : true,
    ...(withCost && { agreedCost: values.paidBy === "WO" && values.agreedCost !== "" ? Number(values.agreedCost) : null }),
    engagementStatus: values.engagementStatus,
    bookingDate: values.bookingDate || null,
    dueDate: values.dueDate || null,
    picStaffId: Number(values.picStaffId),
    scope: values.scope,
    notes: values.notes,
  };
}

interface RawVendorMilestone {
  /** The version for `If-Match` (ADR-0036). */
  updatedAt?: string;
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

function toVendorMilestone(raw: RawVendorMilestone, projectSupplierId: string): VendorMilestone {
  return {
    updatedAt: raw.updatedAt ?? "",
    id: String(raw.id),
    projectSupplierId,
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

export interface RawSupplierPayment {
  updatedAt?: string;
  id: number;
  projectSupplierId: number;
  payer: SupplierPayment["payer"];
  type: SupplierPayment["type"];
  amount: number;
  paymentDate: string;
  method: string;
  referenceNumber: string;
  /** The portal never gets it. */
  notes?: string;
  evidenceComplete: boolean;
  evidence?: RawEvidenceRef[] | null;
  billTermId?: number | null;
  source?: SupplierPayment["source"];
  status?: SupplierPayment["status"];
  verifiedAt?: string | null;
  verifiedBy?: { name: string } | null;
  rejectedAt?: string | null;
  rejectedBy?: { name: string } | null;
  rejectionReason?: string | null;
}

export function toSupplierPayment(raw: RawSupplierPayment): SupplierPayment {
  return {
    updatedAt: raw.updatedAt ?? "",
    id: String(raw.id),
    projectSupplierId: String(raw.projectSupplierId),
    payer: raw.payer,
    type: raw.type,
    amount: raw.amount,
    paymentDate: raw.paymentDate,
    method: raw.method,
    referenceNumber: raw.referenceNumber,
    notes: raw.notes ?? "",
    evidenceComplete: raw.evidenceComplete,
    evidence: toEvidenceRefs(raw.evidence),
    billTermId: raw.billTermId != null ? String(raw.billTermId) : null,
    source: raw.source ?? "staff",
    status: raw.status ?? "recorded",
    verifiedAt: raw.verifiedAt ?? null,
    verifiedBy: raw.verifiedBy?.name ?? null,
    rejectedAt: raw.rejectedAt ?? null,
    rejectedBy: raw.rejectedBy?.name ?? null,
    rejectionReason: raw.rejectionReason ?? "",
  };
}

interface RawClientPayment {
  /** The version for `If-Match` (ADR-0036). */
  updatedAt?: string;
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
  source?: ClientPayment["source"];
}

function toClientPayment(raw: RawClientPayment): ClientPayment {
  return {
    updatedAt: raw.updatedAt ?? "",
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
    source: raw.source ?? "staff",
  };
}

interface RawIssue {
  /** The version for `If-Match` (ADR-0036). */
  updatedAt?: string;
  id: number;
  projectSupplierId: number | null;
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
  supplierName?: string;
  vendorMilestoneName?: string;
  picName?: string;
  isClientVisible?: boolean;
  evidence?: RawEvidenceRef[] | null;
}

interface RawOverview {
  project: RawProject;
  finance: Omit<ProjectOverview["finance"], "terms" | "nextTerm"> & {
    nextTerm: (Omit<NonNullable<ProjectOverview["finance"]["nextTerm"]>, "id"> & { id: number }) | null;
    terms: Array<Omit<PaymentTerm, "id"> & { id: number }> | null;
  };
  actions: OverviewAction[] | null;
  nextMilestones: Array<Omit<ProjectOverview["nextMilestones"][number], "id"> & { id: number }> | null;
  unconfirmedSuppliers: Array<Omit<ProjectOverview["unconfirmedSuppliers"][number], "id"> & { id: number }> | null;
  openIssueCount: number;
  team: { picStaffId: number; picName: string; picSalesStaffId: number; picSalesName: string };
  clients: Array<Omit<ProjectOverview["clients"][number], "id"> & { id: number }> | null;
}

function toPaymentTerms(raw: Array<Omit<PaymentTerm, "id"> & { id: number }>): PaymentTerm[] {
  return raw.map((t) => ({ ...t, id: String(t.id) }));
}

function toIssue(raw: RawIssue): VendorIssue {
  return {
    updatedAt: raw.updatedAt ?? "",
    id: String(raw.id),
    projectSupplierId: raw.projectSupplierId ? String(raw.projectSupplierId) : "",
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
    supplierName: raw.supplierName ?? "",
    vendorMilestoneName: raw.vendorMilestoneName ?? "",
    picName: raw.picName ?? "",
    isClientVisible: raw.isClientVisible ?? true,
    evidence: toEvidenceRefs(raw.evidence),
  };
}

// toIssueUpdateFields lets the vendor accordion's quick status-change
// dropdown reuse the same full-overwrite updateIssue action -- spread every
// existing field, override just status -- mirroring
// toMilestoneUpdateFields's role for vendor milestones.
export function toIssueUpdateFields(issue: VendorIssue): IssueFormValues {
  return {
    projectSupplierId: issue.projectSupplierId,
    vendorMilestoneId: issue.vendorMilestoneId ?? "",
    title: issue.title,
    description: issue.description,
    impact: issue.impact,
    foundDate: issue.foundDate,
    resolutionPlan: issue.resolutionPlan,
    picStaffId: issue.picStaffId,
    targetResolutionDate: issue.targetResolutionDate ?? "",
    status: issue.status,
    isClientVisible: issue.isClientVisible,
  };
}

interface RawEvidence {
  /** The version for `If-Match` (ADR-0036). */
  updatedAt?: string;
  id: number;
  name: string;
  type: EvidenceType;
  fileName: string;
  documentDate: string | null;
  uploadedAt: string;
  description: string;
  uploadedByStaffId: number;
  uploadedByName?: string;
  uploadedBySource?: "staff" | "client";
  relatedKind: EvidenceRelatedKind;
  relatedId: number;
  isClientVisible: boolean;
}

function toEvidence(raw: RawEvidence): Evidence {
  return {
    updatedAt: raw.updatedAt ?? "",
    id: String(raw.id),
    name: raw.name,
    type: raw.type,
    fileName: raw.fileName,
    documentDate: raw.documentDate,
    uploadedAt: raw.uploadedAt,
    description: raw.description,
    uploadedByStaffId: String(raw.uploadedByStaffId),
    uploadedByName: raw.uploadedByName ?? "",
    uploadedBySource: raw.uploadedBySource ?? "staff",
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
  actorName?: string;
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
    actorName: raw.actorName ?? "",
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
  overview: ProjectOverview | null;
  milestones: ProjectMilestone[];
  events: ProjectEvent[];
  /** The project the loaded events belong to (null until they have loaded). */
  eventsFor: string | null;
  suppliers: ProjectSupplier[];
  vendorMilestones: VendorMilestone[];
  supplierPayments: SupplierPayment[];
  clientPayments: ClientPayment[];
  issues: VendorIssue[];
  evidence: Evidence[];
  // documents is Client Portal's own "Dokumen" tab feed — always only
  // general-kind, client-visible documents (GET /projects/{id}/documents),
  // deliberately its own cache slice rather than a filtered view over
  // `evidence` above (which staff-only screens populate via a different,
  // unfiltered endpoint).
  documents: Evidence[];
  activity: ActivityLogEntry[];
  activityMeta: PaginationMeta;
  milestoneHistory: ActivityLogEntry[];

  projectPage: ProjectListItem[];
  projectPageMeta: PaginationMeta;

  fetchProjects: () => Promise<void>;
  // Backs ProjectListPage's table — server-side page, sort, search and filters
  // (status, archived, picStaffId, eventFrom, eventTo), separate from the
  // `projects` full-roster cache above. `params` is the list's own query.
  // archived splits active/archived into two disjoint views, never merged
  // (see ADR-0013). Each row carries the derived `remaining` and `nextDueDate`.
  fetchProjectPage: (params: Record<string, string | number | boolean>) => Promise<void>;
  /** The first 100 projects by name as {id, name}: a small picker for filters on other lists. Not cached in the store. */
  fetchProjectOptions: () => Promise<{ id: string; name: string }[]>;
  /**
   * One call creates the project together with its events and the venues they bring in (PLAN.md E7): all or nothing,
   * every problem reported at once. Retrying with the same Idempotency-Key returns the project the first try made.
   */
  createProject: (values: ProjectFormValues, extra: NewProjectExtras, options?: SaveOptions) => Promise<Project>;
  /** `version` is the `updatedAt` the form was opened with; the API refuses the save (409) when the project changed since. */
  updateProject: (id: string, values: ProjectFormValues, version?: string) => Promise<Project>;
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
  /** Fits the schedule to the current contract value: paid installments stay, unpaid ones share the rest (EP-05). */
  recalculatePaymentTerms: (projectId: string) => Promise<void>;

  /** The rincian of `rincianFor` (Tahap 3): its lines, totals, margin and how the schedule stands against it. */
  rincian: Rincian | null;
  rincianFor: string | null;
  fetchRincian: (projectId: string) => Promise<Rincian>;
  /**
   * Saves the whole rincian as the editor holds it (PUT, If-Match = linesVersion). A stale version is left to the
   * editor (409 with data.current), which keeps the person's edits instead of the global conflict dialog.
   */
  saveRincian: (projectId: string, lines: LineDraft[], version: string) => Promise<Rincian>;
  /** What the drafts would come to, computed by the server; writes nothing. */
  previewRincian: (projectId: string, lines: LineDraft[], signal?: AbortSignal) => Promise<Rincian>;
  /** Sets the archive state explicitly (ADR-0036): repeating it, or two people doing it, cannot undo it. */
  setProjectArchived: (id: string, isArchived: boolean) => Promise<void>;
  // Hard delete (ADR-0013) — Owner-only and requires the project already be
  // archived or cancelled, both enforced server-side, not just by the UI
  // hiding the button.
  deleteProject: (id: string) => Promise<void>;
  // Duplicate (ADR-0014) — clones the source project's milestones and vendor
  // lineup into a brand-new project; `values` is the new project's own
  // (possibly user-edited) fields, same shape as create.
  duplicateProject: (sourceId: string, values: ProjectFormValues) => Promise<Project>;

  /**
   * Loads the project page's header, side panel and Ringkasan in ONE call (E1) and sets currentProject and the
   * payment schedule from it; everything else on the page loads when its tab opens.
   */
  fetchOverview: (projectId: string) => Promise<void>;
  /** Uploads (or replaces) the Client Portal cover photo; the payload is already compressed (ADR-0047). */
  saveCover: (projectId: string, file: CompressedFilePayload) => Promise<void>;
  /** Removes the cover photo: the portal hero falls back to navy. */
  removeCover: (projectId: string) => Promise<void>;
  /** The server returns fresh progress with every mutation (E8): putting it here keeps the header from going stale. */
  applyProgress: (projectId: string, progress: unknown) => void;
  /** Reloads the overview without clearing what is on screen. */
  refreshOverview: (projectId: string) => Promise<void>;

  fetchMilestones: (projectId: string) => Promise<void>;
  createMilestone: (projectId: string, values: ProjectMilestoneFormValues) => Promise<void>;
  updateMilestoneStatus: (projectId: string, milestoneId: string, status: MilestoneStatus) => Promise<void>;
  updateMilestone: (projectId: string, milestoneId: string, fields: ProjectMilestoneUpdateFields, version?: string) => Promise<void>;
  reorderMilestones: (projectId: string, orderedIds: string[]) => Promise<void>;

  fetchEvents: (projectId: string) => Promise<void>;
  /** Each event write may move the project's date (the main event's) and bring a venue into Vendor & venue. */
  createEvent: (projectId: string, input: EventInput) => Promise<ProjectEvent>;
  /** Replaces the event and its sessions; `version` is the event's `updatedAt` the drawer opened with (409 when it changed). */
  updateEvent: (projectId: string, eventId: string, input: EventInput, version: string) => Promise<ProjectEvent>;
  deleteEvent: (projectId: string, eventId: string) => Promise<void>;
  setMainEvent: (projectId: string, eventId: string) => Promise<void>;
  moveEvent: (projectId: string, eventId: string, direction: "up" | "down") => Promise<void>;
  duplicateEvent: (projectId: string, eventId: string) => Promise<void>;

  /** Vendor & venue with every step, one call. */
  fetchSuppliers: (projectId: string) => Promise<void>;
  /** Adds a directory vendor (a venue comes in through an event). `withCost`: the caller may send "Biaya rekanan". */
  createSupplier: (projectId: string, values: SupplierFormValues, withCost: boolean) => Promise<void>;
  /** Sends only what changed against `before` (ADR-0036): another person's edit to another field is kept. */
  updateSupplier: (projectId: string, before: ProjectSupplier, values: SupplierFormValues, withCost: boolean) => Promise<void>;
  /** Removes it at once and returns the function that restores it (the server keeps it 24 h). */
  deleteSupplier: (projectId: string, supplierId: string) => Promise<() => Promise<void>>;
  /** A venue: copies the directory's current capacities onto this project. */
  refreshSupplierCapacity: (projectId: string, supplierId: string) => Promise<void>;
  createVendorMilestone: (projectId: string, supplierId: string, values: VendorMilestoneFormValues) => Promise<void>;
  updateVendorMilestone: (
    projectId: string,
    supplierId: string,
    milestoneId: string,
    fields: Partial<VendorMilestoneUpdateFields>,
    version?: string
  ) => Promise<void>;

  /** Payments to vendors and venues the caller may see (the WO's need finance.margin.view). */
  fetchSupplierPayments: (projectId: string) => Promise<void>;
  createSupplierPayment: (projectId: string, values: SupplierPaymentFormValues, options?: SaveOptions) => Promise<void>;
  updateSupplierPayment: (projectId: string, paymentId: string, values: SupplierPaymentUpdateFormValues, version?: string) => Promise<void>;
  /** Removes it at once and returns the function that restores it (the server keeps it 24 h). */
  deleteSupplierPayment: (projectId: string, paymentId: string) => Promise<() => Promise<void>>;

  fetchClientPayments: (projectId: string) => Promise<void>;
  // Orchestrates POST .../client-payments then, only if values.proofFile is
  // set, compress+POST .../evidence with relatedKind "clientPayment" — one
  // store action instead of a separate manual "Tambah Evidence" step (PLAN.md
  // §1.4), closing the "always incomplete in practice" gap vendor_payments
  // has today (createPayment above never attaches evidence at creation time).
  createClientPayment: (projectId: string, values: ClientPaymentFormValues, options?: SaveOptions) => Promise<void>;
  updateClientPayment: (projectId: string, paymentId: string, values: ClientPaymentUpdateFormValues, version?: string) => Promise<void>;
  deleteClientPayment: (projectId: string, paymentId: string) => Promise<() => Promise<void>>;
  /** The WO confirms the money arrived (E4). Needs a proof file; the client then sees the stamp. */
  verifyClientPayment: (projectId: string, paymentId: string) => Promise<void>;

  fetchIssues: (projectId: string) => Promise<void>;
  createIssue: (projectId: string, values: IssueFormValues) => Promise<void>;
  updateIssue: (projectId: string, issueId: string, values: IssueFormValues, version?: string) => Promise<void>;
  /** The status dropdown on a kendala: sends only the status, so nothing else another person changed is touched. */
  setIssueStatus: (projectId: string, issueId: string, status: IssueStatus) => Promise<void>;

  fetchEvidence: (projectId: string) => Promise<void>;
  uploadEvidence: (projectId: string, values: UploadEvidenceInput) => Promise<void>;
  setEvidenceClientVisible: (projectId: string, evidenceId: string, isClientVisible: boolean) => Promise<void>;
  fetchDocuments: (projectId: string) => Promise<void>;

  fetchActivity: (projectId: string, page?: number) => Promise<void>;
  /** The few latest entries for the Riwayat panel on Ringkasan; returned, not stored, so the full list is left alone. */
  fetchRecentActivity: (projectId: string, limit: number) => Promise<ActivityLogEntry[]>;
  /** What happened to the vendor steps (one filtered call, newest first), for the history inside each engagement. */
  fetchVendorMilestoneHistory: (projectId: string) => Promise<void>;
}

// Backed by the real `projects` module (Fase 4) — tenant-scoped, fetch-then-set
// (ADR-0009). Every fetch* function replaces its slice wholesale; there is no
// per-project namespacing in the store itself, so callers always re-fetch
// when the viewed project changes (matches how the tab pages remount).
async function loadOverview(projectId: string): Promise<Pick<ProjectState, "currentProject" | "overview" | "paymentTerms" | "paymentTermsFor">> {
  const res = await httpClient.get(API.projects.overview(projectId));
  const raw = res.data.data as RawOverview;
  const finance = raw.finance;
  const terms = toPaymentTerms(finance.terms ?? []);
  return {
    currentProject: toProject(raw.project),
    overview: {
      finance: { ...finance, nextTerm: finance.nextTerm ? { ...finance.nextTerm, id: String(finance.nextTerm.id) } : null, terms },
      actions: raw.actions ?? [],
      nextMilestones: (raw.nextMilestones ?? []).map((m) => ({ ...m, id: String(m.id) })),
      unconfirmedSuppliers: (raw.unconfirmedSuppliers ?? []).map((v) => ({ ...v, id: String(v.id) })),
      openIssueCount: raw.openIssueCount,
      team: {
        picStaffId: raw.team.picStaffId ? String(raw.team.picStaffId) : "",
        picName: raw.team.picName,
        picSalesStaffId: raw.team.picSalesStaffId ? String(raw.team.picSalesStaffId) : "",
        picSalesName: raw.team.picSalesName,
      },
      clients: (raw.clients ?? []).map((c) => ({ ...c, id: String(c.id) })),
    },
    paymentTerms: terms,
    paymentTermsFor: projectId,
  };
}

// The overview (finance numbers, what needs doing) changes with almost any write on the project, so it is reloaded
// after the last write of a burst; the progress that came back with the write is applied at once.
let overviewTimer: ReturnType<typeof setTimeout> | undefined;

/**
 * After an event write: the list is reloaded (order, main event and sessions may all have moved), the project's date
 * follows the main event, and Vendor & venue is reloaded when it is on screen (a chosen venue joins it at once).
 */
async function afterEventWrite(projectId: string, data: EventWriteResult | null | undefined): Promise<void> {
  const store = useProjectStore.getState();
  const date = data?.projectEventDate;
  if (date && store.currentProject?.id === projectId && store.currentProject.eventDate !== date) {
    useProjectStore.setState({ currentProject: { ...store.currentProject, eventDate: date } });
  }
  await Promise.all([
    store.fetchEvents(projectId),
    // Guests drive "Tambahan tamu": the rincian on screen is read again.
    store.rincianFor === projectId ? store.fetchRincian(projectId).catch(() => undefined) : undefined,
    store.suppliers.length > 0 || store.eventsFor === projectId ? store.fetchSuppliers(projectId).catch(() => undefined) : undefined,
  ]);
}

export const useProjectStore = create<ProjectState>((set, get) => ({
  projects: [],
  currentProject: null,
  overview: null,
  milestones: [],
  events: [],
  eventsFor: null,
  suppliers: [],
  vendorMilestones: [],
  supplierPayments: [],
  clientPayments: [],
  issues: [],
  paymentTerms: [],
  paymentTermsFor: null,
  rincian: null,
  rincianFor: null,
  evidence: [],
  documents: [],
  activity: [],
  activityMeta: EMPTY_PAGINATION_META,
  milestoneHistory: [],

  projectPage: [],
  projectPageMeta: EMPTY_PAGINATION_META,

  fetchProjects: async () => {
    const res = await httpClient.get(API.projects.base, { params: { all: true } });
    const list = (res.data.data as RawProject[]).map(toProject);
    set({ projects: list });
  },

  fetchProjectPage: async (params) => {
    const res = await httpClient.get(API.projects.base, { params });
    const list = (res.data.data as RawProjectListItem[]).map(toProjectListItem);
    set({ projectPage: list, projectPageMeta: toPaginationMeta(res.data.meta as RawPaginationMeta) });
  },

  fetchProjectOptions: async () => {
    const res = await httpClient.get(API.projects.base, { params: { page: 1, limit: 100, sort: "name:asc" } });
    return (res.data.data as { id: number; name: string }[]).map((p) => ({ id: String(p.id), name: p.name }));
  },

  createProject: async (values, extra, options) => {
    const { eventDate: _eventDate, ...own } = projectInputBody(values) as ReturnType<typeof projectInputBody> & { eventDate?: string };
    const body = { ...own, events: extra.events.map(eventInputBody) };
    const res = await httpClient.post(API.projects.base, body, {
      headers: options?.idempotencyKey ? { "Idempotency-Key": options.idempotencyKey } : undefined,
    });
    const project = toProject(res.data.data as RawProject);
    // The project exists now. Refreshing the list is a courtesy: if it fails,
    // the caller must still see success, or the user retries and gets a duplicate.
    await get().fetchProjects().catch(() => undefined);
    return project;
  },

  updateProject: async (id, values, version) => {
    const current = get().currentProject;
    const { status: _status, ...body } = projectInputBody(values); // state changes go through transition/cancel, not PATCH
    const res = await httpClient.patch(API.projects.item(id), body, {
      headers: ifMatch(version ?? (current?.id === id ? current.updatedAt : undefined)),
    });
    const project = toProject(res.data.data as RawProject);
    set((state) => ({
      projects: state.projects.map((p) => (p.id === id ? project : p)),
      currentProject: state.currentProject?.id === id ? { ...project, progress: state.currentProject.progress } : state.currentProject,
    }));
    return project;
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

  recalculatePaymentTerms: async (projectId) => {
    const res = await httpClient.post(API.projects.paymentTermsRecalculate(projectId));
    set({ paymentTerms: toPaymentTerms(res.data.data), paymentTermsFor: projectId });
    // The schedule now adds up to the contract: the rincian's termsMismatch is read again.
    if (get().rincianFor === projectId) await get().fetchRincian(projectId).catch(() => undefined);
  },

  fetchRincian: async (projectId) => {
    const res = await httpClient.get(API.projects.lines(projectId));
    const rincian = toRincian(res.data.data as RawRincian);
    set({ rincian, rincianFor: projectId });
    return rincian;
  },

  saveRincian: async (projectId, lines, version) => {
    const res = await httpClient.put(
      API.projects.lines(projectId),
      { lines: lines.map(lineDraftBody) },
      { headers: ifMatch(version), ownConflict: true },
    );
    const rincian = toRincian(res.data.data as RawRincian);
    set((state) => ({
      rincian,
      rincianFor: projectId,
      currentProject:
        state.currentProject?.id === projectId && rincian.linesVersion
          ? { ...state.currentProject, linesVersion: rincian.linesVersion }
          : state.currentProject,
    }));
    return rincian;
  },

  previewRincian: async (projectId, lines, signal) => {
    const res = await httpClient.post(API.projects.linesPreview(projectId), { lines: lines.map(lineDraftBody) }, { signal });
    return toRincian(res.data.data as RawRincian);
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

  setProjectArchived: async (id, isArchived) => {
    const res = await httpClient.patch(API.projects.item(id), { isArchived });
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
    const res = await httpClient.post(API.projects.duplicate(sourceId), { ...projectInputBody(values), eventDate: values.eventDate });
    const project = toProject(res.data.data as RawProject);
    await get().fetchProjects();
    return project;
  },

  fetchOverview: async (projectId) => {
    // Everything scoped to a project is cleared with it, so nothing of the previous project can show or be edited.
    set({
      suppliers: [], vendorMilestones: [], milestones: [], supplierPayments: [], clientPayments: [], events: [], eventsFor: null,
      evidence: [], issues: [], paymentTerms: [], paymentTermsFor: null, overview: null, rincian: null, rincianFor: null,
    });
    set(await loadOverview(projectId));
  },

  saveCover: async (projectId, file) => {
    const res = await httpClient.put(API.projects.cover(projectId), file);
    const coverUrl = (res.data.data as { coverUrl: string | null }).coverUrl;
    // The cover does not change the project's version (ADR-0047): only this field moves.
    const current = get().currentProject;
    if (current?.id === projectId) set({ currentProject: { ...current, coverUrl } });
  },

  removeCover: async (projectId) => {
    await httpClient.delete(API.projects.cover(projectId));
    const current = get().currentProject;
    if (current?.id === projectId) set({ currentProject: { ...current, coverUrl: null } });
  },

  refreshOverview: async (projectId) => {
    const loaded = await loadOverview(projectId);
    if (get().currentProject?.id === projectId) set(loaded);
  },

  applyProgress: (projectId, progress) => {
    if (!progress) return;
    const next = toProgress(progress as RawProgress);
    set((state) => ({
      currentProject: state.currentProject?.id === projectId ? { ...state.currentProject, progress: next } : state.currentProject,
    }));
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
    // Only the status is sent; dates stay as they are, even if someone changed them a moment ago.
    const existing = get().milestones.find((m) => m.id === milestoneId);
    await httpClient.patch(API.projects.milestone(projectId, milestoneId), { status }, { headers: ifMatch(existing?.updatedAt) });
    await get().fetchMilestones(projectId);
  },

  updateMilestone: async (projectId, milestoneId, fields, version) => {
    const existing = get().milestones.find((m) => m.id === milestoneId);
    await httpClient.patch(
      API.projects.milestone(projectId, milestoneId),
      { status: fields.status, targetDate: fields.targetDate, completedDate: fields.completedDate || null, forClient: fields.forClient },
      { headers: ifMatch(version ?? existing?.updatedAt) },
    );
    await get().fetchMilestones(projectId);
  },

  reorderMilestones: async (projectId, orderedIds) => {
    await httpClient.patch(API.projects.milestones(projectId), { orderedIds: orderedIds.map(Number) });
    await get().fetchMilestones(projectId);
  },

  fetchEvents: async (projectId) => {
    const res = await httpClient.get(API.projects.events(projectId));
    set({ events: (res.data.data as RawEvent[]).map(toEvent), eventsFor: projectId });
  },

  createEvent: async (projectId, input) => {
    const res = await httpClient.post(API.projects.events(projectId), eventInputBody(input));
    const data = res.data.data as RawEvent & EventWriteResult;
    await afterEventWrite(projectId, data);
    return toEvent(data);
  },

  updateEvent: async (projectId, eventId, input, version) => {
    const res = await httpClient.put(API.projects.event(projectId, eventId), eventInputBody(input), { headers: ifMatch(version) });
    const data = res.data.data as RawEvent & EventWriteResult;
    await afterEventWrite(projectId, data);
    return toEvent(data);
  },

  deleteEvent: async (projectId, eventId) => {
    const res = await httpClient.delete(API.projects.event(projectId, eventId));
    await afterEventWrite(projectId, res.data.data as EventWriteResult);
  },

  setMainEvent: async (projectId, eventId) => {
    const res = await httpClient.post(API.projects.eventSetMain(projectId, eventId));
    await afterEventWrite(projectId, res.data.data as EventWriteResult);
  },

  moveEvent: async (projectId, eventId, direction) => {
    const res = await httpClient.post(API.projects.eventMove(projectId, eventId), { direction });
    set({ events: (res.data.data as RawEvent[]).map(toEvent), eventsFor: projectId });
  },

  duplicateEvent: async (projectId, eventId) => {
    const res = await httpClient.post(API.projects.eventDuplicate(projectId, eventId));
    await afterEventWrite(projectId, res.data.data as EventWriteResult);
  },

  fetchSuppliers: async (projectId) => {
    const res = await httpClient.get(API.projects.suppliers(projectId));
    const raw = res.data.data as RawSupplier[];
    set({
      suppliers: raw.map(toSupplier),
      vendorMilestones: raw.flatMap((sp) => (sp.milestones ?? []).map((m) => toVendorMilestone(m, String(sp.id)))),
    });
  },

  createSupplier: async (projectId, values, withCost) => {
    await httpClient.post(API.projects.suppliers(projectId), { vendorId: Number(values.vendorId), ...supplierBody(values, withCost) });
    await get().fetchSuppliers(projectId);
  },

  updateSupplier: async (projectId, before, values, withCost) => {
    const next = supplierBody(values, withCost) as Record<string, unknown>;
    const was: Record<string, unknown> = {
      paidBy: before.paidBy,
      showName: before.showName,
      agreedCost: before.agreedCost,
      engagementStatus: before.engagementStatus,
      bookingDate: before.bookingDate,
      dueDate: before.dueDate,
      picStaffId: before.picStaffId ? Number(before.picStaffId) : 0,
      scope: before.scope,
      notes: before.notes,
    };
    const body = Object.fromEntries(Object.entries(next).filter(([k, v]) => v !== was[k]));
    if (Object.keys(body).length === 0) return;
    await httpClient.patch(API.projects.supplier(projectId, before.id), body, { headers: ifMatch(before.updatedAt) });
    await get().fetchSuppliers(projectId);
  },

  deleteSupplier: async (projectId, supplierId) => {
    const before = get().suppliers;
    set({ suppliers: before.filter((x) => x.id !== supplierId) });
    try {
      await httpClient.delete(API.projects.supplier(projectId, supplierId));
    } catch (err) {
      set({ suppliers: before });
      throw err;
    }
    return async () => {
      await httpClient.post(API.projects.supplierRestore(projectId, supplierId));
      await get().fetchSuppliers(projectId);
    };
  },

  refreshSupplierCapacity: async (projectId, supplierId) => {
    await httpClient.post(API.projects.supplierRefreshCapacity(projectId, supplierId));
    await get().fetchSuppliers(projectId);
    if (get().eventsFor === projectId) await get().fetchEvents(projectId);
  },

  createVendorMilestone: async (projectId, supplierId, values) => {
    await httpClient.post(API.projects.supplierMilestones(projectId, supplierId), {
      name: values.name,
      description: values.description,
      targetDate: values.targetDate,
      picStaffId: Number(values.picStaffId),
    });
    await get().fetchSuppliers(projectId);
  },

  updateVendorMilestone: async (projectId, supplierId, milestoneId, fields, version) => {
    const existing = get().vendorMilestones.find((m) => m.id === milestoneId);
    await httpClient.patch(
      API.projects.supplierMilestone(projectId, supplierId, milestoneId),
      // Only what was passed: a status change from the dropdown sends the status, not the whole step.
      {
        ...(fields.status !== undefined && { status: fields.status }),
        ...(fields.targetDate !== undefined && { targetDate: fields.targetDate }),
        ...(fields.completedDate !== undefined && { completedDate: fields.completedDate || null }),
        ...(fields.picStaffId !== undefined && { picStaffId: Number(fields.picStaffId) }),
        ...(fields.description !== undefined && { description: fields.description }),
        ...(fields.notes !== undefined && { notes: fields.notes }),
      },
      { headers: ifMatch(version ?? existing?.updatedAt) },
    );
    await get().fetchSuppliers(projectId);
  },

  fetchSupplierPayments: async (projectId) => {
    const res = await httpClient.get(API.projects.supplierPayments(projectId));
    set({ supplierPayments: (res.data.data as RawSupplierPayment[]).map(toSupplierPayment) });
  },

  createSupplierPayment: async (projectId, values, options) => {
    await savePaymentWithProof(
      projectId,
      API.projects.supplierPayments(projectId),
      {
        projectSupplierId: Number(values.projectSupplierId),
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
    // What each supplier was paid (and whether "Dibayar oleh" is locked) is part of Vendor & venue.
    await Promise.all([get().fetchSupplierPayments(projectId), get().fetchSuppliers(projectId)]);
  },

  updateSupplierPayment: async (projectId, paymentId, values, version) => {
    const existing = get().supplierPayments.find((p) => p.id === paymentId);
    await httpClient.patch(
      API.projects.supplierPayment(projectId, paymentId),
      {
        type: values.type,
        amount: values.amount,
        paymentDate: values.paymentDate,
        method: values.method,
        referenceNumber: values.referenceNumber,
        notes: values.notes,
      },
      { headers: ifMatch(version ?? existing?.updatedAt) },
    );
    await Promise.all([get().fetchSupplierPayments(projectId), get().fetchSuppliers(projectId)]);
  },

  deleteSupplierPayment: async (projectId, paymentId) => {
    const before = get().supplierPayments;
    set({ supplierPayments: before.filter((p) => p.id !== paymentId) });
    try {
      await httpClient.delete(API.projects.supplierPayment(projectId, paymentId));
    } catch (err) {
      set({ supplierPayments: before });
      throw err;
    }
    void get().fetchSuppliers(projectId);
    return async () => {
      await httpClient.post(API.projects.supplierPaymentRestore(projectId, paymentId));
      await Promise.all([get().fetchSupplierPayments(projectId), get().fetchSuppliers(projectId)]);
    };
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

  updateClientPayment: async (projectId, paymentId, values, version) => {
    const existing = get().clientPayments.find((p) => p.id === paymentId);
    await httpClient.patch(
      API.projects.clientPayment(projectId, paymentId),
      {
        type: values.type,
        amount: values.amount,
        paymentDate: values.paymentDate,
        method: values.method,
        referenceNumber: values.referenceNumber,
        notes: values.notes,
      },
      { headers: ifMatch(version ?? existing?.updatedAt) },
    );
    await get().fetchClientPayments(projectId);
  },

  deleteClientPayment: async (projectId, paymentId) => {
    const before = get().clientPayments;
    set({ clientPayments: before.filter((p) => p.id !== paymentId) });
    try {
      await httpClient.delete(API.projects.clientPayment(projectId, paymentId));
    } catch (err) {
      set({ clientPayments: before });
      throw err;
    }
    return async () => {
      await httpClient.post(API.projects.clientPaymentRestore(projectId, paymentId));
      await get().fetchClientPayments(projectId);
    };
  },

  verifyClientPayment: async (projectId, paymentId) => {
    await httpClient.post(API.projects.clientPaymentVerify(projectId, paymentId));
    await get().fetchClientPayments(projectId);
  },

  fetchIssues: async (projectId) => {
    const res = await httpClient.get(API.projects.issues(projectId));
    set({ issues: (res.data.data as RawIssue[]).map(toIssue) });
  },

  createIssue: async (projectId, values) => {
    await httpClient.post(API.projects.issues(projectId), {
      ...values,
      projectSupplierId: values.projectSupplierId ? Number(values.projectSupplierId) : null,
      vendorMilestoneId: values.projectSupplierId && values.vendorMilestoneId ? Number(values.vendorMilestoneId) : null,
      picStaffId: Number(values.picStaffId),
    });
    await get().fetchIssues(projectId);
  },

  updateIssue: async (projectId, issueId, values, version) => {
    const existing = get().issues.find((i) => i.id === issueId);
    await httpClient.patch(
      API.projects.issue(projectId, issueId),
      {
        ...values,
        projectSupplierId: values.projectSupplierId ? Number(values.projectSupplierId) : null,
        vendorMilestoneId: values.projectSupplierId && values.vendorMilestoneId ? Number(values.vendorMilestoneId) : null,
        picStaffId: Number(values.picStaffId),
      },
      { headers: ifMatch(version ?? existing?.updatedAt) },
    );
    await get().fetchIssues(projectId);
  },

  setIssueStatus: async (projectId, issueId, status) => {
    const existing = get().issues.find((i) => i.id === issueId);
    await httpClient.patch(API.projects.issue(projectId, issueId), { status }, { headers: ifMatch(existing?.updatedAt) });
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

  setEvidenceClientVisible: async (projectId, evidenceId, isClientVisible) => {
    await httpClient.patch(API.projects.evidenceItem(projectId, evidenceId), { isClientVisible });
    await get().fetchEvidence(projectId);
  },

  fetchDocuments: async (projectId) => {
    const res = await httpClient.get(API.projects.documents(projectId));
    set({ documents: (res.data.data as RawEvidence[]).map(toEvidence) });
  },

  fetchVendorMilestoneHistory: async (projectId) => {
    const res = await httpClient.get(API.projects.activity(projectId), { params: { entityType: "vendor_milestone", limit: 100 } });
    set({ milestoneHistory: (res.data.data as RawActivity[]).map(toActivity) });
  },

  fetchActivity: async (projectId, page = 1) => {
    const res = await httpClient.get(API.projects.activity(projectId), { params: { page, limit: 20 } });
    set({ activity: (res.data.data as RawActivity[]).map(toActivity), activityMeta: toPaginationMeta(res.data.meta as RawPaginationMeta) });
  },

  fetchRecentActivity: async (projectId, limit) => {
    const res = await httpClient.get(API.projects.activity(projectId), { params: { page: 1, limit } });
    return (res.data.data as RawActivity[]).map(toActivity);
  },
}));

onProjectWrite((projectId, data) => {
  const store = useProjectStore.getState();
  if (store.currentProject?.id !== projectId) return;
  store.applyProgress(projectId, (data as { projectProgress?: unknown } | null | undefined)?.projectProgress);
  clearTimeout(overviewTimer);
  overviewTimer = setTimeout(() => void store.refreshOverview(projectId).catch(() => undefined), 300);
});
