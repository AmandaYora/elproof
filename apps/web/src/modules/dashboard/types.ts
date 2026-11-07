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

// No vendorId -- a venue payment isn't tied to any vendor. Kept as its own
// type/list (see DashboardStats.incompleteVenuePayments) rather than
// reshaping DashboardPaymentRow into a source-discriminated union.
export interface DashboardVenuePaymentRow {
  id: string;
  projectId: string;
  projectName: string;
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

/** An unpaid client installment that is overdue or due within 30 days. */
export interface DashboardTermRow {
  projectId: string;
  projectName: string;
  label: string;
  dueDate: string;
  remaining: number;
}

/** A project Timeline item due this week. */
export interface DashboardTaskRow {
  projectId: string;
  projectName: string;
  name: string;
  targetDate: string;
}

export interface StageCounts {
  draft: number;
  preparation: number;
  ready: number;
}

export interface DashboardStats {
  totalProjects: number;
  activeProjects: number;
  activeVendorCount: number;
  openIssues: DashboardIssueRow[];
  overdueVendorMilestones: DashboardMilestoneRow[];
  incompletePayments: DashboardPaymentRow[];
  incompleteVenuePayments: DashboardVenuePaymentRow[];
  nearDDayProjects: Project[];
  laggingProjects: LaggingProjectRow[];
  upcomingProjects: Project[];
  overdueTerms: DashboardTermRow[];
  overdueTermsTotal: number;
  dueSoonTerms: DashboardTermRow[];
  tasksDueThisWeek: DashboardTaskRow[];
  stageCounts: StageCounts;
  recentActivity: ActivityLogEntry[];
  revenue: RevenueSummary;
  projectTrend: ProjectTrendPoint[];
  revenueTrend: RevenueTrendPoint[];
}
