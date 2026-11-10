// Real backend types for the `projects` module (Fase 4) — replaces the
// pre-integration mock's project-related types. IDs are stringified to match
// the convention already used by `useVendorStore`/`useStaffStore` (Fase 3).

export type ProjectStatus = "Draft" | "Preparation" | "Ready" | "Completed" | "Cancelled";

/** One explicit step in a project's life; the API refuses any that does not fit the current status. */
export type ProjectStatusAction = "start_preparation" | "mark_ready" | "complete" | "reopen";

/** GET /projects/{id}/closing-checklist — informs the person completing a project. */
export interface ClosingChecklistItem {
  key: string;
  label: string;
  done: boolean;
  detail: string;
}

export interface ClosingChecklist {
  /** True when every item is done. */
  ready: boolean;
  items: ClosingChecklistItem[];
}

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
  | "Other"
  | "Booking Proof";

export type EvidenceRelatedKind =
  | "vendorMilestone"
  | "payment"
  | "projectVendor"
  | "issue"
  | "clientPayment"
  | "venuePayment"
  | "general"
  | "projectMilestone";

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
  // "" means "belum ditugaskan" (0 on the wire) — a project created by Sales
  // may have no Wedding Planner assigned yet, until handover (PLAN.md
  // revisi-timeline-vendor-role-sales).
  picStaffId: string;
  // PIC Sales — "" means "belum ditugaskan" (0 on the wire), same sentinel
  // convention as picStaffId.
  picSalesStaffId: string;
  // Resolved server-side (PLAN.md's Client Portal restructure) — "" when
  // unresolved (e.g. the staff row was hard-deleted). Lets Client Portal
  // show "Penanggung Jawab" without depending on the staff-only
  // /staff/summary endpoint.
  picName: string;
  // The PIC's number, present only in the Client Portal (WhatsApp button).
  picPhone?: string;
  description: string;
  isArchived: boolean;
  progress?: ProjectProgress;
}

/** A row of the project list: the project plus two columns the server derives from its payment schedule. */
export interface ProjectListItem extends Project {
  /** What the client still owes on the contract (never negative). */
  remaining: number;
  /** Due date (YYYY-MM-DD) of the earliest installment not fully paid, or null. */
  nextDueDate: string | null;
}

/** A file that backs something shown on screen (E4): enough to list it and open it. */
export interface EvidenceRef {
  id: string;
  type: EvidenceType;
  fileName: string;
  uploadedAt: string;
}

export interface ProjectMilestone {
  id: string;
  order: number;
  name: string;
  status: MilestoneStatus;
  targetDate: string;
  completedDate: string | null;
  /** Files attached to this step (the Portal gets only those the WO shared). */
  evidence: EvidenceRef[];
}

export type VendorPricingTier = "Akad" | "AkadResepsi" | "Resepsi";

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
  // "HH:MM", optional (PLAN.md revisi-timeline-vendor-role-sales) — 5 fixed
  // presets or free custom input on the frontend.
  eventStartTime: string | null;
  eventEndTime: string | null;
  dpAmount: number;
  paidAmount: number;
  dueDate: string | null;
  picStaffId: string;
  notes: string;
  /** Resolved by the server (E6); "" when it could not be. */
  vendorName: string;
  categoryName: string;
  /** Staff name of the engagement's PIC, resolved by the server (E2); "" when unresolved. */
  picName: string;
  /** How many files back this engagement, its steps and its payments (what the caller may see). */
  evidenceCount: number;
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
  evidence: EvidenceRef[];
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
  /** Who entered it (never sent to the Portal). */
  recordedBy: string | null;
  /** When and by whom the WO confirmed the money arrived; null = not yet verified. */
  verifiedAt: string | null;
  verifiedBy: string | null;
  evidence: EvidenceRef[];
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
  /** "" = a kendala about the event in general, not tied to a vendor. */
  projectVendorId: string;
  // Null when the kendala is general to the vendor engagement, not tied to
  // one specific deliverable.
  vendorMilestoneId: string | null;
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
  /** Names resolved by the server so the Portal needs no vendor lists (E6). */
  vendorName: string;
  vendorMilestoneName: string;
  /** The WO's PIC for this kendala (never sent to the Portal). */
  picName: string;
  /** Whether the client sees it in the Portal. */
  isClientVisible: boolean;
  evidence: EvidenceRef[];
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
  isClientVisible: boolean;
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
  | "payment_updated"
  | "payment_deleted"
  | "evidence_uploaded"
  | "issue_created"
  | "issue_updated";

export interface ActivityLogEntry {
  id: string;
  type: ActivityType;
  actorStaffId: string;
  /** Resolved by the server; "" for the system (no actor). */
  actorName: string;
  projectId: string | null;
  entityType: string;
  entityId: string;
  entityLabel: string;
  description: string;
  timestamp: string;
}

/** Where one installment of the client's payment schedule stands (derived by the API from real payments). */
export type PaymentTermStatus = "paid" | "overdue" | "partial" | "due_soon" | "upcoming";

export interface PaymentTerm {
  id: string;
  label: string;
  amount: number;
  dueDate: string;
  paid: number;
  remaining: number;
  status: PaymentTermStatus;
}

/** What the schedule editor sends: the whole schedule replaces the old one. */
export interface PaymentTermInput {
  label: string;
  amount: number;
  dueDate: string;
}

/**
 * GET /projects/{id}/overview (PLAN.md E1): everything the project header, side panel and Ringkasan tab show, in one
 * call and computed by the server (finance numbers, what needs doing, who is on the team).
 */
export interface ProjectOverview {
  finance: {
    contractValue: number;
    /** Applied to the contract: DP + Termin + Pelunasan - Refund. */
    paid: number;
    /** Never negative. */
    remaining: number;
    overdueTotal: number;
    nextTerm: { id: string; label: string; amount: number; dueDate: string; status: PaymentTermStatus; remaining: number } | null;
    terms: PaymentTerm[];
    vendorCost: number;
    venueCost: number;
    /** Only present for roles with finance.margin.view. */
    margin?: number;
  };
  actions: OverviewAction[];
  nextMilestones: Array<{ id: string; name: string; status: MilestoneStatus; targetDate: string }>;
  unconfirmedVendors: Array<{ id: string; vendorName: string; engagementStatus: string }>;
  openIssueCount: number;
  team: { picStaffId: string; picName: string; picSalesStaffId: string; picSalesName: string };
  clients: Array<{ id: string; name: string; role: string; hasLogin: boolean }>;
}

export interface OverviewAction {
  kind: "term_overdue" | "issue_open" | "vendor_unconfirmed" | "milestone_late";
  id: string;
  tone: "danger" | "warning";
  title: string;
  detail: string;
  tab: "pembayaran/client" | "kendala" | "vendor" | "milestone";
}
