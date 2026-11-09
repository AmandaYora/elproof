import type { ActivityLogEntry, IssueImpact, IssueStatus, PaymentType, Project } from "@/modules/projects/types";

export interface DashboardIssueRow {
  id: string;
  projectId: string;
  projectName: string;
  vendorId: string;
  /** Resolved by the server; "" for a kendala about the event itself. */
  vendorName: string;
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
  vendorName: string;
  name: string;
  targetDate: string;
}

export interface DashboardPaymentRow {
  id: string;
  projectId: string;
  projectName: string;
  vendorId: string;
  vendorName: string;
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

/** One week of expected money: what clients still owe (in) and what vendors are still owed (out). */
export interface CashFlowWeek {
  /** Monday of the week, YYYY-MM-DD. */
  weekStart: string;
  expectedIn: number;
  expectedOut: number;
}

/** A vendor payment that is due within 30 days or already late. */
export interface VendorDueRow {
  projectId: string;
  projectName: string;
  vendorName: string;
  dueDate: string;
  remaining: number;
}

/** A vendor still only planned or negotiated although the event is within 30 days. */
export interface UnconfirmedVendorRow {
  projectId: string;
  projectName: string;
  vendorName: string;
  engagementStatus: string;
  eventDate: string;
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
  /** Only for roles that may see money (dashboard.revenue.view). */
  cashFlow?: CashFlowWeek[];
  vendorPaymentsDue?: VendorDueRow[];
  unconfirmedVendors: UnconfirmedVendorRow[];
}
