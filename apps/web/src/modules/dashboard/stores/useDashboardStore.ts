import { create } from "zustand";
import { httpClient } from "@/shared/services/http-client";
import { API } from "@/shared/services/api-endpoints";
import { toProject, toActivity, type RawProject, type RawActivity } from "@/modules/projects/stores/useProjectStore";
import type {
  DashboardIssueRow,
  DashboardPaymentRow,
  DashboardStats,
  LaggingProjectRow,
  ProjectTrendPoint,
  RevenueTrendPoint,
} from "@/modules/dashboard/types";

interface RawDashboardIssue {
  id: number;
  projectId: number;
  projectName: string;
  vendorId: number;
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
  name: string;
  targetDate: string;
}

interface RawDashboardPayment {
  id: number;
  projectId: number;
  projectName: string;
  vendorId: number;
  type: DashboardPaymentRow["type"];
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
  nearDDayProjects: RawProject[];
  laggingProjects: RawLaggingProject[];
  upcomingProjects: RawProject[];
  recentActivity: RawActivity[];
  revenue: { total: number; previousTotal: number; deltaPercent: number | null };
  projectTrend: ProjectTrendPoint[];
  revenueTrend: RevenueTrendPoint[];
}

function toDashboardStats(raw: RawDashboard): DashboardStats {
  return {
    totalProjects: raw.totalProjects,
    activeProjects: raw.activeProjects,
    activeVendorCount: raw.activeVendorCount,
    openIssues: raw.openIssues.map((i) => ({
      id: String(i.id), projectId: String(i.projectId), projectName: i.projectName, vendorId: String(i.vendorId),
      title: i.title, impact: i.impact, foundDate: i.foundDate, status: i.status,
    })),
    overdueVendorMilestones: raw.overdueVendorMilestones.map((m) => ({
      id: String(m.id), projectId: String(m.projectId), projectName: m.projectName, vendorId: String(m.vendorId),
      name: m.name, targetDate: m.targetDate,
    })),
    incompletePayments: raw.incompletePayments.map((p) => ({
      id: String(p.id), projectId: String(p.projectId), projectName: p.projectName, vendorId: String(p.vendorId),
      type: p.type, amount: p.amount, paymentDate: p.paymentDate,
    })),
    nearDDayProjects: raw.nearDDayProjects.map(toProject),
    laggingProjects: raw.laggingProjects.map((row): LaggingProjectRow => ({ project: toProject(row.project), overallPercent: row.overallPercent })),
    upcomingProjects: raw.upcomingProjects.map(toProject),
    recentActivity: raw.recentActivity.map(toActivity),
    revenue: raw.revenue,
    projectTrend: raw.projectTrend,
    revenueTrend: raw.revenueTrend,
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
