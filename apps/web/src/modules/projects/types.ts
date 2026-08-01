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

// Public-safe subset backing GET /projects/{id}/venue (ADR-0016) — shared by
// the WO Console's own summary card and Client Portal's Venue tab. Never
// carries rental price/charge/PIC contact (staff-only, fetched directly from
// the venues module instead — see ProjectVenueTabPage.tsx). facilities/
// socialMedia are free text now, not arrays/structured lists; there's no
// more photo gallery, just hasVisibleAttachment (true only when an
// attachment exists and it's an image — see ADR-0016's Revisi).
export interface ProjectVenueSummary {
  id: string;
  name: string;
  address: string | null;
  city: string | null;
  capacity: number | null;
  facilities: string | null;
  socialMedia: string | null;
  hasVisibleAttachment: boolean;
}

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

export type EvidenceRelatedKind = "vendorMilestone" | "payment" | "projectVendor" | "issue" | "clientPayment" | "venuePayment";

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
  // Cross-module reference into the vendors module's Venue directory
  // (ADR-0016) — null means no structured venue attached yet; `venue`
  // (free text) stays the fallback display in that case.
  venueId: string | null;
  // Per-project cost SNAPSHOT captured at attach time (freely editable
  // afterward, same lifecycle as a vendor engagement's own contractValue)
  // — never the venue's live master-data price. Both null whenever
  // venueId is null. See PLAN.md "Financial Calculation Correctness".
  venueRentalPrice: number | null;
  venueCharge: number | null;
  prepStartDate: string;
  packageName: string;
  contractValue: number;
  status: ProjectStatus;
  picStaffId: string;
  description: string;
  isArchived: boolean;
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

export type VendorPricingTier = "Akad" | "AkadResepsi";

export interface ProjectVendor {
  id: string;
  vendorId: string;
  categoryId: string;
  scope: string;
  contractValue: number;
  pricingTier: VendorPricingTier;
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
  notes: string;
  evidenceComplete: boolean;
}

// Money coming IN from the client against the project's own contractValue —
// the opposite accounting direction from VendorPayment, and structurally
// simpler: no projectVendorId (it belongs to the project itself, not any
// vendor), and only one evidence slot (a transfer proof), so evidenceComplete
// here is a plain existence check, not a two-field rule.
export interface ClientPayment {
  id: string;
  type: PaymentType;
  amount: number;
  paymentDate: string;
  method: string;
  referenceNumber: string;
  notes: string;
  evidenceComplete: boolean;
}

// Money going OUT to the project's venue -- same accounting direction and
// same two-slot evidence rule (Invoice + Transfer Proof) as VendorPayment,
// but structurally simpler like ClientPayment: no projectVendorId-equivalent
// field, since a project has at most one venue.
export interface VenuePayment {
  id: string;
  type: PaymentType;
  amount: number;
  paymentDate: string;
  method: string;
  referenceNumber: string;
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
