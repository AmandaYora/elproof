import type { ActivityLogEntry, IssueImpact, IssueStatus, PaymentType, Project } from "@/modules/projects/types";

export interface DashboardIssueRow {
  id: string;
  projectId: string;
  projectName: string;
  vendorId: string;
  title: string;
  impact: IssueImpact;
  foundDate: string;
  status: IssueStatus;
}

export interface DashboardMilestoneRow {
  id: string;
  projectId: string;
  projectName: string;
  vendorId: string;
  name: string;
  targetDate: string;
}

export interface DashboardPaymentRow {
  id: string;
  projectId: string;
  projectName: string;
  vendorId: string;
  type: PaymentType;
  amount: number;
  paymentDate: string;
}

export interface LaggingProjectRow {
  project: Project;
  overallPercent: number;
}

export interface ProjectTrendPoint {
  key: string;
  label: string;
  count: number;
}

export interface RevenueTrendPoint {
  key: string;
  label: string;
  total: number;
}

export interface RevenueSummary {
  total: number;
  previousTotal: number;
  deltaPercent: number | null;
}

export interface DashboardStats {
  totalProjects: number;
  activeProjects: number;
  activeVendorCount: number;
  openIssues: DashboardIssueRow[];
  overdueVendorMilestones: DashboardMilestoneRow[];
  incompletePayments: DashboardPaymentRow[];
  nearDDayProjects: Project[];
  laggingProjects: LaggingProjectRow[];
  upcomingProjects: Project[];
  recentActivity: ActivityLogEntry[];
  revenue: RevenueSummary;
  projectTrend: ProjectTrendPoint[];
  revenueTrend: RevenueTrendPoint[];
}
