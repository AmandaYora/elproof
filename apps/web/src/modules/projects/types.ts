// Real backend types for the `projects` module (Fase 4) — replaces the
// pre-integration mock's project-related types. IDs are stringified to match
// the convention already used by `useVendorStore`/`useStaffStore` (Fase 3).

import type { EngagementStatus, SupplierKind, SupplierPayer } from "@/modules/rincian";

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

export type IssueImpact = "Low" | "Medium" | "High" | "Critical";

export type {
  EngagementStatus,
  EventLayout,
  SupplierPayer,
  SupplierKind,
  EventSession,
  EventVenue,
  EventInput,
  RincianEvent as ProjectEvent,
  Rincian,
  RincianLine,
  LineDraft,
  LineKind,
} from "@/modules/rincian";

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
  | "supplierPayment"
  | "projectSupplier"
  | "issue"
  | "clientPayment"
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
  /** Version of this item as the API gave it; sent back as `If-Match` when saving (ADR-0036). */
  updatedAt: string;
  id: string;
  name: string;
  brideName: string;
  groomName: string;
  eventDate: string;
  prepStartDate: string;
  /** Names of the packages in the rincian, in order ([] = "Tanpa paket"). */
  packageNames: string[];
  /** Nilai kontrak: the rincian's subtotal plus PPN (when the WO is PKP). */
  contractValue: number;
  contractSubtotal: number;
  contractTax: number;
  /** The WO collects PPN on this project's contract. */
  isPkp: boolean;
  /** Version of the rincian, sent as `If-Match` when it is saved. */
  linesVersion: string;
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
  /** Where the Client Portal hero photo streams from (`/api/v1/projects/{id}/cover?v=...`, needs the session); null = none (ADR-0047). */
  coverUrl: string | null;
  /** Where the events happen ("Masjid Al-Ikhlas · Gedung Melati"); "" before a venue is chosen. */
  location: string;
  /** The project's events in one line ("Akad, Resepsi"). */
  eventsLine: string;
  /** The accepted quotation this project was made from ("Jadikan proyek"); null for a project made by hand. */
  quotation: ProjectQuotationRef | null;
}

/** "Dari KODE-PNW-… Revisi n": the quotation revision a project was made from (its Kesepakatan awal). */
export interface ProjectQuotationRef {
  id: string;
  number: string;
  revisionN: number;
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
  /** Version of this item as the API gave it; sent back as `If-Match` when saving (ADR-0036). */
  updatedAt: string;
  id: string;
  order: number;
  name: string;
  status: MilestoneStatus;
  targetDate: string;
  completedDate: string | null;
  /**
   * The Jadwal bawaan offset the target follows (H-14 = 14, H+3 = -3): the target moves with the event date. null when
   * the date is fixed (added or dated by hand).
   */
  daysBeforeEvent?: number | null;
  /** A task for the couple: shown on the Client Portal, which may mark it done (ADR-0047). */
  forClient: boolean;
  /** The couple marked it done from the portal. */
  completedByClient: boolean;
  /** Files attached to this step (the Portal gets only those the WO shared). */
  evidence: EvidenceRef[];
}

/** One vendor or venue of a project ("Vendor & venue"). */
export interface ProjectSupplier {
  /** Version of this item as the API gave it; sent back as `If-Match` when saving (ADR-0036). */
  updatedAt: string;
  id: string;
  kind: SupplierKind;
  /** The directory vendor or venue it was chosen from. */
  supplierId: string;
  name: string;
  categoryId: string;
  categoryName: string;
  paidBy: SupplierPayer;
  /** Whether the client sees the name (a vendor the WO pays may be shown by its service only). */
  showName: boolean;
  capacityStanding: number | null;
  capacitySeated: number | null;
  /** "Biaya rekanan" and what the WO paid: null without finance.margin.view. */
  agreedCost: number | null;
  paidWo: number | null;
  /** What the client paid this supplier directly. */
  paidClient: number | null;
  engagementStatus: EngagementStatus;
  bookingDate: string | null;
  dueDate: string | null;
  picStaffId: string;
  picName: string;
  scope: string;
  notes: string;
  /** Why "Dibayar oleh" can no longer change; "" when it can. */
  paidByLocked: string;
  /** How many events take place at this venue (0 = "Tidak dipakai acara mana pun"; always 0 for a vendor). */
  usedByEvents: number;
  /** How many files back this supplier, its steps and its payments (what the caller may see). */
  evidenceCount: number;
}

export interface VendorMilestone {
  /** Version of this item as the API gave it; sent back as `If-Match` when saving (ADR-0036). */
  updatedAt: string;
  id: string;
  projectSupplierId: string;
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

/** A payment to a vendor or venue of the project, by the WO or straight from the client (the supplier's "Dibayar oleh"). */
export interface SupplierPayment {
  /** Version of this item as the API gave it; sent back as `If-Match` when saving (ADR-0036). */
  updatedAt: string;
  id: string;
  projectSupplierId: string;
  payer: SupplierPayer;
  type: PaymentType;
  amount: number;
  paymentDate: string;
  method: string;
  referenceNumber: string;
  notes: string;
  evidenceComplete: boolean;
  /** The payment's own proof files (invoice and/or transfer proof), as the API embeds them. */
  evidence: EvidenceRef[];
  /** The bill of a vendor the client pays that a proof from the portal pays; null otherwise (Paket WO Tahap 4). */
  billTermId: string | null;
  /** "client" when the couple sent the proof from the portal, "staff" when the WO recorded it. */
  source: PaymentSource;
  /** recorded (staff, counts) · pending (the couple's proof, not checked: not counted) · verified (counts) · rejected (not counted). */
  status: SupplierPaymentStatus;
  verifiedAt: string | null;
  /** Who checked the couple's proof (staff view only; the portal never names the WO's staff). */
  verifiedBy: string | null;
  rejectedAt: string | null;
  rejectedBy: string | null;
  /** Why the WO refused the couple's proof; the couple reads it. "" when not refused. */
  rejectionReason: string;
}

export type SupplierPaymentStatus = "recorded" | "pending" | "verified" | "rejected";

/**
 * One bill of a vendor or venue the client pays directly ("Tagihan vendor", Paket WO Tahap 4): what it asks for and by
 * when, with how much of it the counted payments cover (earliest due date first) and what the couple sent that the WO
 * has not checked yet.
 */
export interface SupplierBillTerm {
  /** Version of this item as the API gave it; sent back as `If-Match` when saving (ADR-0036). */
  updatedAt: string;
  id: string;
  projectSupplierId: string;
  label: string;
  amount: number;
  dueDate: string;
  paid: number;
  remaining: number;
  /** The couple's proofs for this bill the WO has not checked yet (not counted as paid). */
  pending: number;
  status: PaymentTermStatus;
}

/** One vendor or venue the client pays directly, always by its real name: its bills, the payments to it and the totals. */
export interface SupplierBill {
  projectSupplierId: string;
  name: string;
  kind: SupplierKind;
  /** Sum of the bills. */
  total: number;
  /** What the counted payments cover (staff-recorded and verified proofs). */
  paid: number;
  remaining: number;
  /** The couple's proofs to this vendor the WO has not checked yet. */
  pending: number;
  terms: SupplierBillTerm[];
  payments: SupplierPayment[];
}

// Money coming IN from the client against the project's own contractValue —
// the opposite accounting direction from SupplierPayment, and structurally
// simpler: it belongs to the project itself, not to a supplier, and only one evidence slot (a transfer proof), so evidenceComplete
// here is a plain existence check, not a two-field rule.
export interface ClientPayment {
  /** Version of this item as the API gave it; sent back as `If-Match` when saving (ADR-0036). */
  updatedAt: string;
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
  /** "client" when the couple sent it from the portal (unverified until the WO checks it), "staff" otherwise (ADR-0047). */
  source: PaymentSource;
}

export type PaymentSource = "client" | "staff";

export interface VendorIssue {
  /** Version of this item as the API gave it; sent back as `If-Match` when saving (ADR-0036). */
  updatedAt: string;
  id: string;
  /** "" = a kendala about the event in general, not tied to a vendor or venue. */
  projectSupplierId: string;
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
  /** Resolved by the server; the Portal gets the service instead of a hidden vendor's name. */
  supplierName: string;
  vendorMilestoneName: string;
  /** The WO's PIC for this kendala (never sent to the Portal). */
  picName: string;
  /** Whether the client sees it in the Portal. */
  isClientVisible: boolean;
  evidence: EvidenceRef[];
}

export interface Evidence {
  /** Version of this item as the API gave it; sent back as `If-Match` when saving (ADR-0036). */
  updatedAt: string;
  id: string;
  name: string;
  type: EvidenceType;
  fileName: string;
  documentDate: string | null;
  uploadedAt: string;
  description: string;
  uploadedByStaffId: string;
  /** The uploader's name from the API: a staff member, or the client who sent the file from the Client Portal. */
  uploadedByName: string;
  /** "client" when the couple sent the file from the Client Portal (uploadedByStaffId is then "0"). */
  uploadedBySource: "staff" | "client";
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
  | "issue_updated"
  | "line_added"
  | "line_updated"
  | "line_deleted";

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
    /** Nilai kontrak: subtotal of the rincian plus PPN. */
    contractValue: number;
    contractSubtotal: number;
    contractTax: number;
    /** What the payment schedule adds up to; termsMismatch when that is not the contract value. */
    termsTotal: number;
    termsMismatch: boolean;
    /** Applied to the contract: DP + Termin + Pelunasan - Refund. */
    paid: number;
    /** Never negative. */
    remaining: number;
    overdueTotal: number;
    nextTerm: { id: string; label: string; amount: number; dueDate: string; status: PaymentTermStatus; remaining: number } | null;
    terms: PaymentTerm[];
    /** What the WO agreed to pay its vendors and venues; only with finance.margin.view. */
    supplierCost?: number;
    /** How many vendors the WO pays have no "Biaya rekanan" yet (the margin is incomplete); only with margin. */
    missingCost?: number;
    /** Contract value minus supplierCost; only with finance.margin.view. */
    margin?: number;
  };
  actions: OverviewAction[];
  nextMilestones: Array<{ id: string; name: string; status: MilestoneStatus; targetDate: string }>;
  unconfirmedSuppliers: Array<{ id: string; kind: SupplierKind; name: string; engagementStatus: string }>;
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
  tab: "keuangan/klien" | "kendala" | "vendor" | "jadwal";
}
