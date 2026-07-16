// Real backend types for the `projects` module (Fase 4) — replaces the
// pre-integration mock's project-related types. IDs are stringified to match
// the convention already used by `useVendorStore`/`useStaffStore` (Fase 3).

export type ProjectStatus = "Draft" | "Preparation" | "Ready" | "Completed" | "Cancelled";

export type MilestoneStatus = "Not Started" | "In Progress" | "Completed" | "Blocked" | "Cancelled";

export type EngagementStatus =
  | "Planned"
  | "Negotiation"
  | "Booked"
  | "DP Paid"
  | "In Progress"
  | "Fully Paid"
  | "Ready"
  | "Completed"
  | "Cancelled";

export type IssueImpact = "Low" | "Medium" | "High" | "Critical";

export type IssueStatus = "Open" | "In Review" | "In Resolution" | "Resolved" | "Closed";

export type PaymentType = "DP" | "Termin" | "Pelunasan" | "Tambahan" | "Refund";

export type EvidenceType =
  | "Quotation"
  | "Invoice"
  | "Contract"
  | "Transfer Proof"
  | "Receipt"
  | "Purchase Order"
  | "Photo"
  | "Document"
  | "Screenshot"
  | "Minutes of Meeting"
  | "Other";

export type EvidenceRelatedKind = "vendorMilestone" | "payment" | "projectVendor" | "issue";

export type ProjectCondition = "On Track" | "Attention" | "At Risk";

export interface MilestoneStats {
  total: number;
  completed: number;
  inProgress: number;
  blocked: number;
  notStarted: number;
  cancelled: number;
  overdue: number;
  ratio: number;
}

export interface ProjectProgress {
  projectMilestoneStats: MilestoneStats;
  vendorMilestoneStats: MilestoneStats;
  overallPercent: number;
  condition: ProjectCondition;
  openIssueCount: number;
  criticalOrHighOpenIssueCount: number;
  overdueMilestoneCount: number;
  incompleteEvidenceCount: number;
}

export interface Project {
  id: string;
  name: string;
  brideName: string;
  groomName: string;
  eventDate: string;
  venue: string;
  prepStartDate: string;
  packageName: string;
  contractValue: number;
  status: ProjectStatus;
  picStaffId: string;
  description: string;
  progress?: ProjectProgress;
}

export interface ProjectMilestone {
  id: string;
  order: number;
  name: string;
  status: MilestoneStatus;
  targetDate: string;
  completedDate: string | null;
}

export interface ProjectVendor {
  id: string;
  vendorId: string;
  categoryId: string;
  scope: string;
  contractValue: number;
  engagementStatus: EngagementStatus;
  bookingDate: string | null;
  eventDate: string;
  dpAmount: number;
  paidAmount: number;
  dueDate: string | null;
  picStaffId: string;
  notes: string;
}

export interface VendorMilestone {
  id: string;
  projectVendorId: string;
  order: number;
  name: string;
  description: string;
  status: MilestoneStatus;
  targetDate: string;
  completedDate: string | null;
  picStaffId: string;
  notes: string;
}

export interface VendorPayment {
  id: string;
  projectVendorId: string;
  type: PaymentType;
  amount: number;
  paymentDate: string;
  method: string;
  referenceNumber: string;
  invoiceEvidenceId: string | null;
  proofEvidenceId: string | null;
  notes: string;
  evidenceComplete: boolean;
}

export interface VendorIssue {
  id: string;
  projectVendorId: string;
  title: string;
  description: string;
  impact: IssueImpact;
  foundDate: string;
  status: IssueStatus;
  resolutionPlan: string;
  picStaffId: string;
  targetResolutionDate: string | null;
  resolvedDate: string | null;
  resolutionNotes: string;
}

export interface Evidence {
  id: string;
  name: string;
  type: EvidenceType;
  fileName: string;
  documentDate: string | null;
  uploadedAt: string;
  description: string;
  uploadedByStaffId: string;
  relatedKind: EvidenceRelatedKind;
  relatedId: string;
}

// Backend only implements a subset of the mock's original activity taxonomy —
// see ADR/Fase 4 implementation notes. Types not listed here never occur.
export type ActivityType =
  | "project_created"
  | "project_updated"
  | "project_status_changed"
  | "vendor_added"
  | "vendor_status_changed"
  | "milestone_updated"
  | "payment_recorded"
  | "evidence_uploaded"
  | "issue_created"
  | "issue_updated";

export interface ActivityLogEntry {
  id: string;
  type: ActivityType;
  actorStaffId: string;
  projectId: string | null;
  entityType: string;
  entityId: string;
  entityLabel: string;
  description: string;
  timestamp: string;
}
