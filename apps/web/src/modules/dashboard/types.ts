import type { ActivityLogEntry, IssueImpact, IssueStatus, PaymentTermStatus, PaymentType, Project, SupplierKind, SupplierPayer } from "@/modules/projects/types";

export interface DashboardIssueRow {
  id: string;
  projectId: string;
  projectName: string;
  /** The vendor or venue it is about; "" for a kendala about the event itself. */
  supplierName: string;
  title: string;
  impact: IssueImpact;
  foundDate: string;
  status: IssueStatus;
}

export interface DashboardMilestoneRow {
  id: string;
  projectId: string;
  projectName: string;
  /** The vendor or venue whose step it is. */
  supplierName: string;
  name: string;
  targetDate: string;
}

/** A payment to a vendor or venue that still lacks its invoice or transfer proof. */
export interface DashboardPaymentRow {
  id: string;
  projectId: string;
  projectName: string;
  supplierKind: SupplierKind;
  supplierName: string;
  /** Who paid it: the WO, or the client straight to the supplier. */
  payer: SupplierPayer;
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

/** One week of expected money: what clients still owe (in) and what the WO still owes its vendors and venues (out). */
export interface CashFlowWeek {
  /** Monday of the week, YYYY-MM-DD. */
  weekStart: string;
  expectedIn: number;
  expectedOut: number;
}

/** What the WO still owes a vendor or venue, due within 30 days or already late. */
export interface SupplierDueRow {
  projectId: string;
  projectName: string;
  supplierKind: SupplierKind;
  supplierName: string;
  dueDate: string;
  remaining: number;
}

/** A vendor or venue still only planned or negotiated although the event is within 30 days. */
export interface UnconfirmedSupplierRow {
  projectId: string;
  projectName: string;
  supplierKind: SupplierKind;
  supplierName: string;
  engagementStatus: string;
  eventDate: string;
}

/**
 * A bill of a vendor or venue the client pays directly that is overdue or due within 7 days (Paket WO Tahap 4): the
 * couple pays it, the WO reminds them. Never WO cash.
 */
export interface ClientBillDueRow {
  billTermId: string;
  projectSupplierId: string;
  projectId: string;
  projectName: string;
  supplierKind: SupplierKind;
  supplierName: string;
  label: string;
  dueDate: string;
  remaining: number;
  /** The couple's proofs for it the WO has not checked yet. */
  pending: number;
  status: PaymentTermStatus;
}

/** A proof the couple sent of paying a vendor they pay directly, waiting for "Periksa bukti" (Paket WO Tahap 4). */
export interface SupplierProofPendingRow {
  supplierPaymentId: string;
  projectId: string;
  projectName: string;
  projectSupplierId: string;
  supplierKind: SupplierKind;
  supplierName: string;
  billTermId: string | null;
  billTermLabel: string;
  type: PaymentType;
  amount: number;
  paymentDate: string;
  submittedAt: string;
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
  supplierPaymentsDue?: SupplierDueRow[];
  unconfirmedSuppliers: UnconfirmedSupplierRow[];
  /** Only for roles that record finance (finance.record): bills the couple pays a vendor directly, due soon or late. */
  clientBillsDue?: ClientBillDueRow[];
  /** Only for finance.record: the couple's proofs of paying a vendor, waiting to be checked. */
  supplierProofsPending?: SupplierProofPendingRow[];
  supplierProofsPendingCount?: number;
}
