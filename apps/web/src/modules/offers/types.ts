// Penawaran (paket WO Tahap 5, ADR-0050) as the screens hold it. Ids are strings like everywhere in the web app; the
// wire shapes (numbers, nulls) live in stores/quotation-wire.ts.
import type { EventLayout, PackageRule, RincianTotals, SupplierPayer } from "@/modules/rincian";

/** What is stored. */
export type QuotationStatus = "Draft" | "Accepted" | "Rejected" | "Superseded";
/** What is shown: a draft past its validity reads "Kedaluwarsa". */
export type QuotationDisplayStatus = QuotationStatus | "Expired";
/** The list's tabs ("" = Semua). */
export type QuotationTab = "" | QuotationDisplayStatus;
export type AcceptChannel = "WhatsApp" | "InPerson" | "Email" | "Other";
export type TermDueBasis = "Accepted" | "Event";

export interface Quotation {
  id: string;
  /** "KODE-PNW-YYMMDD-NNN". */
  number: string;
  docDate: string;
  coupleNames: string;
  contactWhatsapp: string;
  picStaffId: string;
  picName: string;
  validUntil: string;
  /** Days until validUntil (negative once past). */
  daysLeft: number;
  notes: string;
  status: QuotationStatus;
  displayStatus: QuotationDisplayStatus;
  alternativeOfId: string;
  supersededById: string;
  acceptedRevisionN: number | null;
  acceptedOn: string;
  acceptedChannel: AcceptChannel | "";
  acceptanceAttachmentName: string;
  hasAcceptanceAttachment: boolean;
  rejectedReason: string;
  rejectedAt: string;
  convertedProjectId: string;
  convertedAt: string;
  /** The project it became was deleted afterwards. */
  projectDeleted: boolean;
  /** The working copy can still be changed (a draft, not converted). */
  editable: boolean;
  isPkp: boolean;
  revisionCount: number;
  contractTotal: number;
  mainEventDate: string;
  createdAt: string;
  updatedAt: string;
  linesUpdatedAt: string;
  termsUpdatedAt: string;
  changedAt: string;
}

/** A row of the list: the quotation and its alternatives (shown as child rows). */
export interface QuotationListItem extends Quotation {
  alternatives: Quotation[];
}

export interface QuotationSummary {
  all: number;
  draft: number;
  expired: number;
  accepted: number;
  rejected: number;
  superseded: number;
  expiringSoon: number;
}

export interface QuotationSession {
  id: string;
  label: string;
  startTime: string | null;
  endTime: string | null;
  guests: number;
}

export interface QuotationEvent {
  id: string;
  eventKindId: string;
  kindName: string;
  /** "" = not decided yet. */
  eventDate: string;
  startTime: string | null;
  endTime: string | null;
  venueSupplierId: string;
  venueId: string;
  venueName: string;
  layout: EventLayout;
  sortOrder: number;
  sessions: QuotationSession[];
  totalGuests: number;
  updatedAt: string;
}

export interface QuotationSupplier {
  id: string;
  kind: "vendor" | "venue";
  supplierId: string;
  name: string;
  categoryId: string;
  categoryName: string;
  paidBy: SupplierPayer;
  showName: boolean;
  capacityStanding: number | null;
  capacitySeated: number | null;
  /** Biaya rekanan: undefined without finance.margin.view, null when not filled in. */
  agreedCost?: number | null;
  notes: string;
  sortOrder: number;
  updatedAt: string;
}

export type QuotationLineKind = "Package" | "Component" | "ExtraGuests" | "Item" | "Discount";

export interface QuotationLine {
  id: string;
  kind: QuotationLineKind;
  parentLineId: string;
  packageId: string;
  packageRule: PackageRule | null;
  copiedAt: string;
  /** The package changed in the catalog after it was copied here. */
  catalogChanged: boolean;
  description: string;
  details: string;
  categoryId: string;
  categoryName: string;
  eventId: string;
  quantity: number | null;
  unit: string;
  unitPrice?: number;
  amount?: number;
  isBonus: boolean;
  supplierRef: string;
  sortOrder: number;
}

export interface QuotationTerm {
  id: string;
  label: string;
  percent: number | null;
  amount: number | null;
  dueBasis: TermDueBasis;
  dueOffsetDays: number;
  dueEventId: string;
  /** What the term comes to today. */
  computed: number;
  sortOrder: number;
}

export interface QuotationGroupMember {
  id: string;
  number: string;
  status: QuotationStatus;
  displayStatus: QuotationDisplayStatus;
  contractTotal: number;
  isCurrent: boolean;
}

export interface QuotationDetail {
  quotation: Quotation;
  events: QuotationEvent[];
  suppliers: QuotationSupplier[];
  lines: QuotationLine[];
  terms: QuotationTerm[];
  termsText: string;
  totals: RincianTotals | null;
  /** The totals cannot be worked out (a discount over the subtotal): the message to show. */
  totalsProblem: string;
  termsMismatch: boolean;
  /** The revision the working copy is equal to, or null. */
  matchingRevisionN: number | null;
  latestRevisionN: number | null;
  /** The working copy differs from every revision ("Ada perubahan yang belum dikirim"). */
  hasUnsentChanges: boolean;
  /** The quotation and its alternatives (empty when there are none). */
  group: QuotationGroupMember[];
}

export interface QuotationRevision {
  n: number;
  contractTotal: number;
  contentHash: string;
  createdAt: string;
  createdByStaffId: string;
  createdByName: string;
  isAccepted: boolean;
  matchesWorkingCopy: boolean;
}

export interface SnapshotEvent {
  eventKindId: string;
  kindName: string;
  /** "" = not decided yet. */
  eventDate: string;
  startTime: string | null;
  endTime: string | null;
  /** Position in the revision's suppliers. */
  venueIndex: number | null;
  layout: EventLayout;
  sessions: Array<{ label: string; startTime: string | null; endTime: string | null; guests: number }>;
}

export interface SnapshotSupplier {
  kind: "vendor" | "venue";
  supplierId: string;
  name: string;
  categoryId: string;
  paidBy: SupplierPayer;
  showName: boolean;
  capacityStanding: number | null;
  capacitySeated: number | null;
  /** null without finance.margin.view or when not filled in. */
  agreedCost: number | null;
  notes: string;
}

export interface SnapshotLine {
  kind: QuotationLineKind;
  /** Positions in the revision's lists. */
  parentIndex: number | null;
  packageId: string;
  packageRule: PackageRule | null;
  description: string;
  details: string;
  categoryId: string;
  eventIndex: number | null;
  quantity: number | null;
  unit: string;
  unitPrice: number | null;
  amount: number;
  isBonus: boolean;
  supplierIndex: number | null;
}

export interface SnapshotTerm {
  label: string;
  percent: number | null;
  amount: number | null;
  dueBasis: TermDueBasis;
  dueOffsetDays: number;
  dueEventIndex: number | null;
  computed: number;
}

/** A revision's frozen content (also the "Kesepakatan awal" of a project). */
export interface SnapshotContent {
  coupleNames: string;
  contactWhatsapp: string;
  isPkp: boolean;
  events: SnapshotEvent[];
  suppliers: SnapshotSupplier[];
  lines: SnapshotLine[];
  terms: SnapshotTerm[];
  termsText: string;
  totals: RincianTotals | null;
}

export interface QuotationRevisionDetail extends QuotationRevision {
  issuedAt: string;
  validUntil: string;
  picName: string;
  categoryNames: Record<string, string>;
  content: SnapshotContent;
}

export type CompareChange = "added" | "removed" | "changed";

export interface CompareRow<T> {
  change: CompareChange;
  before: T | null;
  after: T | null;
}

export interface QuotationCompare {
  from: number;
  to: number;
  /** `parent`: the package a component belongs to ("" for a top-level row). */
  lines: Array<CompareRow<SnapshotLine> & { parent: string }>;
  unchangedLines: number;
  events: CompareRow<SnapshotEvent>[];
  unchangedEvents: number;
  suppliers: CompareRow<SnapshotSupplier>[];
  unchangedSuppliers: number;
  termsChanged: boolean;
  termsTextChanged: boolean;
  headerChanged: boolean;
  totalsBefore: RincianTotals | null;
  totalsAfter: RincianTotals | null;
}

export interface QuotationActivity {
  id: string;
  type: string;
  actorStaffId: string;
  actorName: string;
  summary: string;
  createdAt: string;
}

/* ----------------------------------------------------------------- Jadikan proyek -------------------------------- */

export interface ConversionPreview {
  quotationId: string;
  quotationNumber: string;
  revisionN: number;
  eventDate: string;
  contractTotal: number;
  isPkp: boolean;
  terms: Array<{ label: string; amount: number; dueDate: string }>;
  counts: { events: number; suppliers: number; lines: number; hasTermsText: boolean };
  /** The quotation is already a project: POST answers that project. */
  existingProjectId: string;
}

export interface ConversionInput {
  name: string;
  brideName: string;
  groomName: string;
  /** By position of the accepted revision's events; "" keeps the revision's date. */
  eventDates: string[];
  prepStartDate: string;
  picStaffId: string;
  /** "" = the quotation's PIC. */
  picSalesStaffId: string;
  description: string;
  downPayment: null | {
    amount: number;
    paymentDate: string;
    method: string;
    referenceNumber: string;
    notes: string;
    evidence: Array<{ uploadToken: string; type: string; name: string }>;
  };
}

export interface ConversionResult {
  projectId: string;
  /** false: the quotation was already a project, and this is it. */
  created: boolean;
  downPaymentId: string;
}

/** GET /projects/{id}/agreement: the accepted revision the project was made from. */
export interface ProjectAgreement {
  quotationId: string;
  quotationNumber: string;
  revisionId: string;
  revisionN: number;
  revisionCreatedAt: string;
  acceptedOn: string;
  acceptedChannel: AcceptChannel | "";
  contractTotal: number;
  content: SnapshotContent;
}
