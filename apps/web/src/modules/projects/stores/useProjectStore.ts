import { create } from "zustand";
import { httpClient } from "@/shared/services/http-client";
import { API } from "@/shared/services/api-endpoints";
import { useVendorStore } from "@/modules/vendors/stores/useVendorStore";
import type { ProjectFormValues } from "@/modules/projects/schemas/project.schema";
import type { ProjectMilestoneFormValues } from "@/modules/projects/schemas/project-milestone.schema";
import type { ProjectVendorFormValues } from "@/modules/projects/schemas/project-vendor.schema";
import type { VendorMilestoneFormValues } from "@/modules/projects/schemas/vendor-milestone.schema";
import type { PaymentFormValues } from "@/modules/projects/schemas/payment.schema";
import type { ClientPaymentFormValues } from "@/modules/projects/schemas/client-payment.schema";
import type { VenuePaymentFormValues } from "@/modules/projects/schemas/venue-payment.schema";
import type { IssueFormValues } from "@/modules/projects/schemas/issue.schema";
import type { CompressedFilePayload } from "@/shared/lib/image-compression";
import { compressFileForUpload } from "@/shared/lib/image-compression";
import type {
  ActivityLogEntry,
  ClientPayment,
  Evidence,
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

// Thrown by createClientPayment when the payment itself was saved but its
// proof upload failed — distinct from a full failure so the caller can close
// the modal (the payment is real) instead of leaving it open for a retry,
// which would otherwise risk creating a duplicate payment.
export class ClientPaymentEvidenceError extends Error {}

// Same purpose as ClientPaymentEvidenceError, for createPayment's two
// possible evidence slots (Invoice, Bukti Transfer) instead of one.
export class VendorPaymentEvidenceError extends Error {}

// Same purpose as VendorPaymentEvidenceError, for createVenuePayment.
export class VenuePaymentEvidenceError extends Error {}

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
    picStaffId: String(raw.picStaffId),
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
    picStaffId: Number(values.picStaffId),
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

interface RawMilestone {
  id: number;
  order: number;
  name: string;
  status: MilestoneStatus;
  targetDate: string;
  completedDate: string | null;
}

function toMilestone(raw: RawMilestone): ProjectMilestone {
  return { id: String(raw.id), order: raw.order, name: raw.name, status: raw.status, targetDate: raw.targetDate, completedDate: raw.completedDate };
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
  dpAmount: number;
  paidAmount: number;
  dueDate: string | null;
  picStaffId: number;
  notes: string;
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
    dpAmount: raw.dpAmount,
    paidAmount: raw.paidAmount,
    dueDate: raw.dueDate,
    picStaffId: String(raw.picStaffId),
    notes: raw.notes,
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
}

function toClientPayment(raw: RawClientPayment): ClientPayment {
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
  };
}

export interface UploadEvidenceInput extends CompressedFilePayload {
  name: string;
  type: EvidenceType;
  documentDate: string;
  description: string;
  relatedKind: EvidenceRelatedKind;
  relatedId: string;
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
  createProject: (values: ProjectFormValues) => Promise<Project>;
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
  createPayment: (projectId: string, values: PaymentFormValues) => Promise<void>;

  fetchClientPayments: (projectId: string) => Promise<void>;
  // Orchestrates POST .../client-payments then, only if values.proofFile is
  // set, compress+POST .../evidence with relatedKind "clientPayment" — one
  // store action instead of a separate manual "Tambah Evidence" step (PLAN.md
  // §1.4), closing the "always incomplete in practice" gap vendor_payments
  // has today (createPayment above never attaches evidence at creation time).
  createClientPayment: (projectId: string, values: ClientPaymentFormValues) => Promise<void>;

  fetchVenuePayments: (projectId: string) => Promise<void>;
  // Same two-slot evidence orchestration as createPayment (vendor's own),
  // just against .../venue-payments and relatedKind "venuePayment".
  createVenuePayment: (projectId: string, values: VenuePaymentFormValues) => Promise<void>;

  fetchIssues: (projectId: string) => Promise<void>;
  createIssue: (projectId: string, values: IssueFormValues) => Promise<void>;
  updateIssue: (projectId: string, issueId: string, values: IssueFormValues) => Promise<void>;

  fetchEvidence: (projectId: string) => Promise<void>;
  uploadEvidence: (projectId: string, values: UploadEvidenceInput) => Promise<void>;

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
  evidence: [],
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

  createProject: async (values) => {
    const res = await httpClient.post(API.projects.base, projectInputBody(values));
    const project = toProject(res.data.data as RawProject);
    await get().fetchProjects();
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
      picStaffId: Number(current.picStaffId),
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
    set({ vendorEngagements: [], vendorMilestones: [] });
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
    const vendor = useVendorStore.getState().vendors.find((v) => v.id === values.vendorId);
    await httpClient.post(API.projects.vendors(projectId), {
      ...vendorEngagementInputBody(values),
      categoryId: vendor ? Number(vendor.categoryId) : 0,
      eventDate: get().currentProject?.eventDate ?? "",
    });
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

  createPayment: async (projectId, values) => {
    const res = await httpClient.post(API.projects.payments(projectId), {
      projectVendorId: Number(values.projectVendorId),
      type: values.type,
      amount: values.amount,
      paymentDate: values.paymentDate,
      method: values.method,
      referenceNumber: values.referenceNumber,
      notes: values.notes,
    });
    const created = toPayment(res.data.data as RawPayment);
    // The payment itself is already persisted at this point -- each of the
    // two evidence slots (Invoice, Bukti Transfer) is uploaded independently
    // so one failing doesn't abort the other; a failure in either must NOT
    // look like the whole submission failed (the caller would otherwise
    // invite a resubmit, creating a duplicate payment). Refetch regardless,
    // then signal any partial failure via a distinct error type naming
    // which slot(s) didn't make it -- same pattern as createClientPayment.
    const failedSlots: string[] = [];
    if (values.invoiceFile) {
      try {
        const compressed = await compressFileForUpload(values.invoiceFile);
        await httpClient.post(API.projects.evidence(projectId), {
          name: values.referenceNumber ? `Invoice - ${values.referenceNumber}` : "Invoice",
          type: "Invoice",
          fileName: compressed.fileName,
          mimeType: compressed.mimeType,
          base64Data: compressed.base64Data,
          documentDate: values.paymentDate,
          description: "",
          relatedKind: "payment",
          relatedId: Number(created.id),
        });
      } catch {
        failedSlots.push("invoice");
      }
    }
    if (values.proofFile) {
      try {
        const compressed = await compressFileForUpload(values.proofFile);
        await httpClient.post(API.projects.evidence(projectId), {
          name: values.referenceNumber ? `Bukti Transfer - ${values.referenceNumber}` : "Bukti Transfer",
          type: "Transfer Proof",
          fileName: compressed.fileName,
          mimeType: compressed.mimeType,
          base64Data: compressed.base64Data,
          documentDate: values.paymentDate,
          description: "",
          relatedKind: "payment",
          relatedId: Number(created.id),
        });
      } catch {
        failedSlots.push("bukti transfer");
      }
    }
    await get().fetchPayments(projectId);
    if (failedSlots.length > 0) {
      throw new VendorPaymentEvidenceError(
        `Pembayaran tersimpan, tapi ${failedSlots.join(" dan ")} gagal diunggah. Anda bisa melampirkannya nanti melalui tab Dokumen.`
      );
    }
  },

  fetchClientPayments: async (projectId) => {
    const res = await httpClient.get(API.projects.clientPayments(projectId));
    set({ clientPayments: (res.data.data as RawClientPayment[]).map(toClientPayment) });
  },

  createClientPayment: async (projectId, values) => {
    const res = await httpClient.post(API.projects.clientPayments(projectId), {
      type: values.type,
      amount: values.amount,
      paymentDate: values.paymentDate,
      method: values.method,
      referenceNumber: values.referenceNumber,
      notes: values.notes,
    });
    const created = toClientPayment(res.data.data as RawClientPayment);
    // The payment itself is already persisted at this point — if the proof
    // upload fails (network/storage error), that must NOT look like the
    // whole submission failed (the caller would otherwise invite a resubmit,
    // creating a duplicate payment). Refetch regardless, then signal the
    // partial failure via a distinct error type so the caller can close the
    // modal instead of leaving it open for a retry.
    if (values.proofFile) {
      try {
        const compressed = await compressFileForUpload(values.proofFile);
        await httpClient.post(API.projects.evidence(projectId), {
          name: values.referenceNumber ? `Bukti Transfer - ${values.referenceNumber}` : "Bukti Transfer",
          type: "Transfer Proof",
          fileName: compressed.fileName,
          mimeType: compressed.mimeType,
          base64Data: compressed.base64Data,
          documentDate: values.paymentDate,
          description: "",
          relatedKind: "clientPayment",
          relatedId: Number(created.id),
        });
      } catch {
        await get().fetchClientPayments(projectId);
        throw new ClientPaymentEvidenceError(
          "Pembayaran tersimpan, tapi bukti transfer gagal diunggah. Anda bisa melampirkan buktinya nanti melalui tab Dokumen."
        );
      }
    }
    await get().fetchClientPayments(projectId);
  },

  fetchVenuePayments: async (projectId) => {
    const res = await httpClient.get(API.projects.venuePayments(projectId));
    set({ venuePayments: (res.data.data as RawVenuePayment[]).map(toVenuePayment) });
  },

  createVenuePayment: async (projectId, values) => {
    const res = await httpClient.post(API.projects.venuePayments(projectId), {
      type: values.type,
      amount: values.amount,
      paymentDate: values.paymentDate,
      method: values.method,
      referenceNumber: values.referenceNumber,
      notes: values.notes,
    });
    const created = toVenuePayment(res.data.data as RawVenuePayment);
    // Same independent-slot orchestration as createPayment (vendor's own) --
    // see its own comment for why each slot is caught separately and the
    // failure surfaced via a distinct error type instead of aborting.
    const failedSlots: string[] = [];
    if (values.invoiceFile) {
      try {
        const compressed = await compressFileForUpload(values.invoiceFile);
        await httpClient.post(API.projects.evidence(projectId), {
          name: values.referenceNumber ? `Invoice - ${values.referenceNumber}` : "Invoice",
          type: "Invoice",
          fileName: compressed.fileName,
          mimeType: compressed.mimeType,
          base64Data: compressed.base64Data,
          documentDate: values.paymentDate,
          description: "",
          relatedKind: "venuePayment",
          relatedId: Number(created.id),
        });
      } catch {
        failedSlots.push("invoice");
      }
    }
    if (values.proofFile) {
      try {
        const compressed = await compressFileForUpload(values.proofFile);
        await httpClient.post(API.projects.evidence(projectId), {
          name: values.referenceNumber ? `Bukti Transfer - ${values.referenceNumber}` : "Bukti Transfer",
          type: "Transfer Proof",
          fileName: compressed.fileName,
          mimeType: compressed.mimeType,
          base64Data: compressed.base64Data,
          documentDate: values.paymentDate,
          description: "",
          relatedKind: "venuePayment",
          relatedId: Number(created.id),
        });
      } catch {
        failedSlots.push("bukti transfer");
      }
    }
    await get().fetchVenuePayments(projectId);
    if (failedSlots.length > 0) {
      throw new VenuePaymentEvidenceError(
        `Pembayaran tersimpan, tapi ${failedSlots.join(" dan ")} gagal diunggah. Anda bisa melampirkannya nanti melalui tab Dokumen.`
      );
    }
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
      relatedId: Number(values.relatedId),
    });
    await get().fetchEvidence(projectId);
  },

  fetchActivity: async (projectId) => {
    const res = await httpClient.get(API.projects.activity(projectId));
    set({ activity: (res.data.data as RawActivity[]).map(toActivity) });
  },
}));
