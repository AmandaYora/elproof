import { create } from "zustand";
import { httpClient } from "@/shared/services/http-client";
import { API } from "@/shared/services/api-endpoints";
import { toProject, toActivity, type RawProject, type RawActivity } from "@/modules/projects/stores/useProjectStore";
import type {
  DashboardIssueRow,
  DashboardPaymentRow,
  DashboardStats,
  DashboardVenuePaymentRow,
  LaggingProjectRow,
  DashboardTermRow,
  DashboardTaskRow,
  StageCounts,
  ProjectTrendPoint,
  RevenueTrendPoint,
  CashFlowWeek,
  VendorDueRow,
  UnconfirmedVendorRow,
} from "@/modules/dashboard/types";

interface RawDashboardIssue {
  id: number;
  projectId: number;
  projectName: string;
  vendorId: number;
  vendorName?: string;
  title: string;
  impact: DashboardIssueRow["impact"];
  foundDate: string;
  status: DashboardIssueRow["status"];
}

interface RawDashboardMilestone {
  id: number;
  projectId: number;
  projectName: string;
  vendorId: number;
  vendorName?: string;
  name: string;
  targetDate: string;
}

interface RawDashboardPayment {
  id: number;
  projectId: number;
  projectName: string;
  vendorId: number;
  vendorName?: string;
  type: DashboardPaymentRow["type"];
  amount: number;
  paymentDate: string;
}

interface RawDashboardVenuePayment {
  id: number;
  projectId: number;
  projectName: string;
  type: DashboardVenuePaymentRow["type"];
  amount: number;
  paymentDate: string;
}

interface RawLaggingProject {
  project: RawProject;
  overallPercent: number;
}

interface RawDashboard {
  totalProjects: number;
  activeProjects: number;
  activeVendorCount: number;
  openIssues: RawDashboardIssue[];
  overdueVendorMilestones: RawDashboardMilestone[];
  incompletePayments: RawDashboardPayment[];
  incompleteVenuePayments: RawDashboardVenuePayment[];
  nearDDayProjects: RawProject[];
  laggingProjects: RawLaggingProject[];
  upcomingProjects: RawProject[];
  overdueTerms: Array<Omit<DashboardTermRow, "projectId"> & { projectId: number }>;
  overdueTermsTotal: number;
  dueSoonTerms: Array<Omit<DashboardTermRow, "projectId"> & { projectId: number }>;
  tasksDueThisWeek: Array<Omit<DashboardTaskRow, "projectId"> & { projectId: number }>;
  stageCounts: StageCounts;
  recentActivity: RawActivity[];
  revenue: { total: number; previousTotal: number; deltaPercent: number | null };
  projectTrend: ProjectTrendPoint[];
  revenueTrend: RevenueTrendPoint[];
  cashFlow?: CashFlowWeek[] | null;
  vendorPaymentsDue?: Array<Omit<VendorDueRow, "projectId"> & { projectId: number }> | null;
  unconfirmedVendors?: Array<Omit<UnconfirmedVendorRow, "projectId"> & { projectId: number }> | null;
}

function toDashboardStats(raw: RawDashboard): DashboardStats {
  return {
    totalProjects: raw.totalProjects,
    activeProjects: raw.activeProjects,
    activeVendorCount: raw.activeVendorCount,
    openIssues: raw.openIssues.map((i) => ({
      id: String(i.id), projectId: String(i.projectId), projectName: i.projectName, vendorId: String(i.vendorId), vendorName: i.vendorName ?? "",
      title: i.title, impact: i.impact, foundDate: i.foundDate, status: i.status,
    })),
    overdueVendorMilestones: raw.overdueVendorMilestones.map((m) => ({
      id: String(m.id), projectId: String(m.projectId), projectName: m.projectName, vendorId: String(m.vendorId), vendorName: m.vendorName ?? "",
      name: m.name, targetDate: m.targetDate,
    })),
    incompletePayments: raw.incompletePayments.map((p) => ({
      id: String(p.id), projectId: String(p.projectId), projectName: p.projectName, vendorId: String(p.vendorId), vendorName: p.vendorName ?? "",
      type: p.type, amount: p.amount, paymentDate: p.paymentDate,
    })),
    incompleteVenuePayments: raw.incompleteVenuePayments.map((p) => ({
      id: String(p.id), projectId: String(p.projectId), projectName: p.projectName,
      type: p.type, amount: p.amount, paymentDate: p.paymentDate,
    })),
    nearDDayProjects: raw.nearDDayProjects.map(toProject),
    laggingProjects: raw.laggingProjects.map((row): LaggingProjectRow => ({ project: toProject(row.project), overallPercent: row.overallPercent })),
    upcomingProjects: raw.upcomingProjects.map(toProject),
    overdueTerms: raw.overdueTerms.map((t) => ({ ...t, projectId: String(t.projectId) })),
    overdueTermsTotal: raw.overdueTermsTotal,
    dueSoonTerms: raw.dueSoonTerms.map((t) => ({ ...t, projectId: String(t.projectId) })),
    tasksDueThisWeek: raw.tasksDueThisWeek.map((t) => ({ ...t, projectId: String(t.projectId) })),
    stageCounts: raw.stageCounts,
    recentActivity: raw.recentActivity.map(toActivity),
    revenue: raw.revenue,
    projectTrend: raw.projectTrend,
    revenueTrend: raw.revenueTrend,
    cashFlow: raw.cashFlow ?? undefined,
    vendorPaymentsDue: raw.vendorPaymentsDue?.map((v) => ({ ...v, projectId: String(v.projectId) })),
    unconfirmedVendors: (raw.unconfirmedVendors ?? []).map((v) => ({ ...v, projectId: String(v.projectId) })),
  };
}

interface DashboardState {
  stats: DashboardStats | null;
  fetchDashboard: () => Promise<void>;
}

// Backed by the real WO dashboard aggregation endpoint (Fase 5) —
// GET /api/v1/dashboard, computed server-side from real project/vendor/issue/
// payment/activity data with the real server clock (replaces mock/selectors.ts's
// getDashboardStats + hardcoded TODAY).
export const useDashboardStore = create<DashboardState>((set) => ({
  stats: null,

  fetchDashboard: async () => {
    const res = await httpClient.get(API.dashboard);
    set({ stats: toDashboardStats(res.data.data as RawDashboard) });
  },
}));
