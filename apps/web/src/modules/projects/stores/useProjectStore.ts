import { create } from "zustand";
import { httpClient } from "@/shared/services/http-client";
import { API } from "@/shared/services/api-endpoints";
import { useVendorStore } from "@/modules/vendors/stores/useVendorStore";
import type { ProjectFormValues } from "@/modules/projects/schemas/project.schema";
import type { ProjectMilestoneFormValues } from "@/modules/projects/schemas/project-milestone.schema";
import type { ProjectVendorFormValues } from "@/modules/projects/schemas/project-vendor.schema";
import type { VendorMilestoneFormValues } from "@/modules/projects/schemas/vendor-milestone.schema";
import type { PaymentFormValues } from "@/modules/projects/schemas/payment.schema";
import type { IssueFormValues } from "@/modules/projects/schemas/issue.schema";
import type { CompressedFilePayload } from "@/shared/lib/image-compression";
import type {
  ActivityLogEntry,
  Evidence,
  EvidenceRelatedKind,
  EvidenceType,
  IssueStatus,
  MilestoneStatus,
  Project,
  ProjectMilestone,
  ProjectProgress,
  ProjectVendor,
  VendorIssue,
  VendorMilestone,
  VendorPayment,
} from "@/modules/projects/types";
import { toPaginationMeta, EMPTY_PAGINATION_META, type PaginationMeta, type RawPaginationMeta } from "@/shared/types/pagination";

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
  prepStartDate: string;
  packageName: string;
  contractValue: number;
  status: Project["status"];
  picStaffId: number;
  description: string;
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
    prepStartDate: raw.prepStartDate,
    packageName: raw.packageName,
    contractValue: raw.contractValue,
    status: raw.status,
    picStaffId: String(raw.picStaffId),
    description: raw.description,
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

interface RawProjectVendor {
  id: number;
  vendorId: number;
  categoryId: number;
  scope: string;
  contractValue: number;
  engagementStatus: ProjectVendor["engagementStatus"];
  bookingDate: string | null;
  eventDate: string;
  dpAmount: number;
  paidAmount: number;
  dueDate: string | null;
  picStaffId: number;
  notes: string;
}

function toProjectVendor(raw: RawProjectVendor): ProjectVendor {
  return {
    id: String(raw.id),
    vendorId: String(raw.vendorId),
    categoryId: String(raw.categoryId),
    scope: raw.scope,
    contractValue: raw.contractValue,
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
    engagementStatus: values.engagementStatus,
    bookingDate: values.bookingDate || "",
    eventDate: "",
    dpAmount: values.dpAmount,
    paidAmount: values.paidAmount,
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
  invoiceEvidenceId: number | null;
  proofEvidenceId: number | null;
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
    invoiceEvidenceId: raw.invoiceEvidenceId !== null ? String(raw.invoiceEvidenceId) : null,
    proofEvidenceId: raw.proofEvidenceId !== null ? String(raw.proofEvidenceId) : null,
    notes: raw.notes,
    evidenceComplete: raw.evidenceComplete,
  };
}

interface RawIssue {
  id: number;
  projectVendorId: number;
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
  issues: VendorIssue[];
  evidence: Evidence[];
  activity: ActivityLogEntry[];

  projectPage: Project[];
  projectPageMeta: PaginationMeta;

  fetchProjects: () => Promise<void>;
  // Backs ProjectListPage's table — real server-side pagination + search/
  // status filtering, separate from the `projects` full-roster cache above
  // (which the dashboard, global search, and client grouping still rely on).
  fetchProjectPage: (page: number, search: string, status: string) => Promise<void>;
  createProject: (values: ProjectFormValues) => Promise<Project>;
  updateProject: (id: string, values: ProjectFormValues) => Promise<Project>;
  cancelProject: (id: string) => Promise<void>;

  fetchProjectDetail: (projectId: string) => Promise<void>;
  fetchMyProject: () => Promise<string>;

  fetchMilestones: (projectId: string) => Promise<void>;
  createMilestone: (projectId: string, values: ProjectMilestoneFormValues) => Promise<void>;
  updateMilestoneStatus: (projectId: string, milestoneId: string, status: MilestoneStatus) => Promise<void>;

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

  fetchIssues: (projectId: string) => Promise<void>;
  createIssue: (projectId: string, values: IssueFormValues) => Promise<void>;
  updateIssueStatus: (projectId: string, issueId: string, status: IssueStatus) => Promise<void>;

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
  issues: [],
  evidence: [],
  activity: [],

  projectPage: [],
  projectPageMeta: EMPTY_PAGINATION_META,

  fetchProjects: async () => {
    const res = await httpClient.get(API.projects.base, { params: { all: true } });
    const list = (res.data.data as RawProject[]).map(toProject);
    // The list endpoint doesn't include progress — fetch it per project so
    // cards can show percent/condition (small dataset; acceptable N+1).
    const withProgress = await Promise.all(
      list.map(async (p) => {
        try {
          const detail = await httpClient.get(API.projects.item(p.id));
          return toProject(detail.data.data as RawProject);
        } catch {
          return p;
        }
      })
    );
    set({ projects: withProgress });
  },

  fetchProjectPage: async (page, search, status) => {
    const res = await httpClient.get(API.projects.base, {
      params: { page, search: search || undefined, status: status || undefined },
    });
    const list = (res.data.data as RawProject[]).map(toProject);
    // Same per-row progress enrichment as fetchProjects, bounded to just this
    // page's rows instead of the whole tenant roster.
    const withProgress = await Promise.all(
      list.map(async (p) => {
        try {
          const detail = await httpClient.get(API.projects.item(p.id));
          return toProject(detail.data.data as RawProject);
        } catch {
          return p;
        }
      })
    );
    set({ projectPage: withProgress, projectPageMeta: toPaginationMeta(res.data.meta as RawPaginationMeta) });
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

  cancelProject: async (id) => {
    const res = await httpClient.post(API.projects.cancel(id));
    const project = toProject(res.data.data as RawProject);
    set((state) => ({
      projects: state.projects.map((p) => (p.id === id ? project : p)),
      currentProject: state.currentProject?.id === id ? { ...project, progress: state.currentProject.progress } : state.currentProject,
    }));
  },

  fetchProjectDetail: async (projectId) => {
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
    await httpClient.patch(API.projects.milestone(projectId, milestoneId), { status });
    await get().fetchMilestones(projectId);
  },

  fetchVendorSection: async (projectId) => {
    const res = await httpClient.get(API.projects.vendors(projectId));
    const engagements = (res.data.data as RawProjectVendor[]).map(toProjectVendor);
    const milestoneLists = await Promise.all(
      engagements.map((pv) => httpClient.get(API.projects.vendorMilestones(projectId, pv.id)))
    );
    const allMilestones = milestoneLists.flatMap((r, i) =>
      (r.data.data as RawVendorMilestone[]).map((m) => toVendorMilestone(m, engagements[i].id))
    );
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
    await httpClient.post(API.projects.payments(projectId), {
      ...values,
      projectVendorId: Number(values.projectVendorId),
    });
    await get().fetchPayments(projectId);
  },

  fetchIssues: async (projectId) => {
    const res = await httpClient.get(API.projects.issues(projectId));
    set({ issues: (res.data.data as RawIssue[]).map(toIssue) });
  },

  createIssue: async (projectId, values) => {
    await httpClient.post(API.projects.issues(projectId), {
      ...values,
      projectVendorId: Number(values.projectVendorId),
      picStaffId: Number(values.picStaffId),
    });
    await get().fetchIssues(projectId);
  },

  updateIssueStatus: async (projectId, issueId, status) => {
    await httpClient.patch(API.projects.issue(projectId, issueId), { status });
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
