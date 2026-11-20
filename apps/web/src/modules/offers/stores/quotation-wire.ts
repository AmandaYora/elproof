// The quotation API's shapes (numbers, nulls) and their mapping to what the screens hold (string ids, "" for none).
import type { EventLayout, PackageRule, RincianTotals, SupplierPayer } from "@/modules/rincian";
import type {
  AcceptChannel,
  ConversionPreview,
  ProjectAgreement,
  Quotation,
  QuotationActivity,
  QuotationCompare,
  QuotationDetail,
  QuotationDisplayStatus,
  QuotationEvent,
  QuotationLine,
  QuotationLineKind,
  QuotationListItem,
  QuotationRevision,
  QuotationRevisionDetail,
  QuotationStatus,
  QuotationSupplier,
  QuotationTerm,
  SnapshotContent,
  SnapshotEvent,
  SnapshotLine,
  SnapshotSupplier,
  SnapshotTerm,
  TermDueBasis,
} from "@/modules/offers/types";

type Id = number | null | undefined;
const idOf = (n: Id): string => (n ? String(n) : "");
const str = (s: string | null | undefined): string => s ?? "";

export interface RawQuotation {
  id: number;
  number: string;
  docDate: string;
  coupleNames: string;
  contactWhatsapp: string;
  picStaffId: number;
  picName: string | null;
  validUntil: string;
  daysLeft: number;
  notes: string;
  status: QuotationStatus;
  displayStatus: QuotationDisplayStatus;
  alternativeOfId: number | null;
  supersededById: number | null;
  acceptedRevisionN: number | null;
  acceptedOn: string | null;
  acceptedChannel: AcceptChannel | null;
  acceptanceAttachmentName: string | null;
  hasAcceptanceAttachment: boolean;
  rejectedReason: string | null;
  rejectedAt: string | null;
  convertedProjectId: number | null;
  convertedAt: string | null;
  projectDeleted: boolean;
  editable: boolean;
  isPkp: boolean;
  revisionCount: number;
  contractTotal: number;
  mainEventDate: string | null;
  createdAt: string;
  updatedAt: string;
  linesUpdatedAt: string;
  termsUpdatedAt: string;
  changedAt: string;
}

export function toQuotation(r: RawQuotation): Quotation {
  return {
    id: String(r.id),
    number: r.number,
    docDate: r.docDate,
    coupleNames: r.coupleNames,
    contactWhatsapp: str(r.contactWhatsapp),
    picStaffId: idOf(r.picStaffId),
    picName: str(r.picName),
    validUntil: r.validUntil,
    daysLeft: r.daysLeft,
    notes: str(r.notes),
    status: r.status,
    displayStatus: r.displayStatus,
    alternativeOfId: idOf(r.alternativeOfId),
    supersededById: idOf(r.supersededById),
    acceptedRevisionN: r.acceptedRevisionN ?? null,
    acceptedOn: str(r.acceptedOn),
    acceptedChannel: r.acceptedChannel ?? "",
    acceptanceAttachmentName: str(r.acceptanceAttachmentName),
    hasAcceptanceAttachment: r.hasAcceptanceAttachment,
    rejectedReason: str(r.rejectedReason),
    rejectedAt: str(r.rejectedAt),
    convertedProjectId: idOf(r.convertedProjectId),
    convertedAt: str(r.convertedAt),
    projectDeleted: r.projectDeleted,
    editable: r.editable,
    isPkp: r.isPkp,
    revisionCount: r.revisionCount,
    contractTotal: r.contractTotal,
    mainEventDate: str(r.mainEventDate),
    createdAt: r.createdAt,
    updatedAt: r.updatedAt,
    linesUpdatedAt: r.linesUpdatedAt,
    termsUpdatedAt: r.termsUpdatedAt,
    changedAt: r.changedAt,
  };
}

export interface RawQuotationListItem extends RawQuotation {
  alternatives: RawQuotation[] | null;
}

export function toQuotationListItem(r: RawQuotationListItem): QuotationListItem {
  return { ...toQuotation(r), alternatives: (r.alternatives ?? []).map(toQuotation) };
}

export interface RawPackageRule {
  packageId: number;
  name: string;
  price?: number;
  eventKindIds: number[] | null;
  includedGuests: number | null;
  extraGuestStep: number | null;
  extraGuestStepPrice?: number | null;
}

export function toPackageRule(r: RawPackageRule | null | undefined): PackageRule | null {
  if (!r) return null;
  return {
    packageId: String(r.packageId),
    name: r.name,
    price: r.price,
    eventKindIds: (r.eventKindIds ?? []).map(String),
    includedGuests: r.includedGuests ?? 0,
    extraGuestStep: r.extraGuestStep ?? 0,
    extraGuestStepPrice: r.extraGuestStepPrice ?? undefined,
  };
}

interface RawSession {
  id: number;
  label: string;
  startTime: string | null;
  endTime: string | null;
  guests: number;
}

export interface RawEvent {
  id: number;
  eventKindId: number;
  kindName: string;
  eventDate: string | null;
  startTime: string | null;
  endTime: string | null;
  venueSupplierId: number | null;
  venueId: number | null;
  venueName: string | null;
  layout: EventLayout;
  sortOrder: number;
  sessions: RawSession[] | null;
  totalGuests: number;
  updatedAt: string;
}

export function toEvent(r: RawEvent): QuotationEvent {
  return {
    id: String(r.id),
    eventKindId: String(r.eventKindId),
    kindName: r.kindName,
    eventDate: str(r.eventDate),
    startTime: r.startTime,
    endTime: r.endTime,
    venueSupplierId: idOf(r.venueSupplierId),
    venueId: idOf(r.venueId),
    venueName: str(r.venueName),
    layout: r.layout,
    sortOrder: r.sortOrder,
    sessions: (r.sessions ?? []).map((s) => ({ ...s, id: String(s.id) })),
    totalGuests: r.totalGuests,
    updatedAt: r.updatedAt,
  };
}

export interface RawSupplier {
  id: number;
  kind: "vendor" | "venue";
  supplierId: number;
  name: string;
  categoryId: number | null;
  categoryName: string | null;
  paidBy: SupplierPayer;
  showName: boolean;
  capacityStanding: number | null;
  capacitySeated: number | null;
  agreedCost?: number | null;
  notes: string;
  sortOrder: number;
  updatedAt: string;
}

export function toSupplier(r: RawSupplier): QuotationSupplier {
  return {
    id: String(r.id),
    kind: r.kind,
    supplierId: String(r.supplierId),
    name: r.name,
    categoryId: idOf(r.categoryId),
    categoryName: str(r.categoryName),
    paidBy: r.paidBy,
    showName: r.showName,
    capacityStanding: r.capacityStanding,
    capacitySeated: r.capacitySeated,
    agreedCost: r.agreedCost,
    notes: str(r.notes),
    sortOrder: r.sortOrder,
    updatedAt: r.updatedAt,
  };
}

export interface RawLine {
  id: number;
  kind: QuotationLineKind;
  parentLineId: number | null;
  packageId: number | null;
  packageRule: RawPackageRule | null;
  copiedAt: string | null;
  catalogChanged: boolean;
  description: string;
  details: string;
  categoryId: number | null;
  categoryName: string | null;
  eventId: number | null;
  quantity: number | null;
  unit: string | null;
  unitPrice?: number | null;
  amount?: number;
  isBonus: boolean;
  supplierRef: number | null;
  sortOrder: number;
}

export function toLine(r: RawLine): QuotationLine {
  return {
    id: String(r.id),
    kind: r.kind,
    parentLineId: idOf(r.parentLineId),
    packageId: idOf(r.packageId),
    packageRule: toPackageRule(r.packageRule),
    copiedAt: str(r.copiedAt),
    catalogChanged: Boolean(r.catalogChanged),
    description: r.description,
    details: str(r.details),
    categoryId: idOf(r.categoryId),
    categoryName: str(r.categoryName),
    eventId: idOf(r.eventId),
    quantity: r.quantity,
    unit: str(r.unit),
    unitPrice: r.unitPrice ?? undefined,
    amount: r.amount,
    isBonus: r.isBonus,
    supplierRef: idOf(r.supplierRef),
    sortOrder: r.sortOrder,
  };
}

interface RawTerm {
  id: number;
  label: string;
  percent: number | null;
  amount: number | null;
  dueBasis: TermDueBasis;
  dueOffsetDays: number;
  dueEventId: number | null;
  computed: number;
  sortOrder: number;
}

function toTerm(r: RawTerm): QuotationTerm {
  return { ...r, id: String(r.id), dueEventId: idOf(r.dueEventId) };
}

export interface RawDetail {
  quotation: RawQuotation;
  events: RawEvent[] | null;
  suppliers: RawSupplier[] | null;
  lines: RawLine[] | null;
  terms: RawTerm[] | null;
  termsText: string;
  totals: RincianTotals | null;
  totalsProblem: string | null;
  termsMismatch: boolean;
  matchingRevisionN: number | null;
  latestRevisionN: number | null;
  hasUnsentChanges: boolean;
  group: Array<{ id: number; number: string; status: QuotationStatus; displayStatus: QuotationDisplayStatus; contractTotal: number; isCurrent: boolean }> | null;
}

export function toDetail(r: RawDetail): QuotationDetail {
  return {
    quotation: toQuotation(r.quotation),
    events: (r.events ?? []).map(toEvent),
    suppliers: (r.suppliers ?? []).map(toSupplier),
    lines: (r.lines ?? []).map(toLine),
    terms: (r.terms ?? []).map(toTerm),
    termsText: str(r.termsText),
    totals: r.totals,
    totalsProblem: str(r.totalsProblem),
    termsMismatch: r.termsMismatch,
    matchingRevisionN: r.matchingRevisionN ?? null,
    // 0 on the wire = no revision yet.
    latestRevisionN: r.latestRevisionN ? r.latestRevisionN : null,
    hasUnsentChanges: r.hasUnsentChanges,
    group: (r.group ?? []).map((g) => ({ ...g, id: String(g.id) })),
  };
}

export interface RawRevision {
  n: number;
  contractTotal: number;
  contentHash: string;
  createdAt: string;
  createdByStaffId: number | null;
  createdByName: string | null;
  isAccepted: boolean;
  matchesWorkingCopy?: boolean;
  snapshot?: RawSnapshot;
}

export function toRevision(r: RawRevision): QuotationRevision {
  return {
    n: r.n,
    contractTotal: r.contractTotal,
    contentHash: r.contentHash,
    createdAt: r.createdAt,
    createdByStaffId: idOf(r.createdByStaffId),
    createdByName: str(r.createdByName),
    isAccepted: r.isAccepted,
    matchesWorkingCopy: Boolean(r.matchesWorkingCopy),
  };
}

interface RawSnapshotEvent {
  eventKindId: number;
  kindName: string;
  eventDate: string | null;
  startTime: string | null;
  endTime: string | null;
  venueIndex: number | null;
  layout: EventLayout;
  sessions: Array<{ label: string; startTime: string | null; endTime: string | null; guests: number }> | null;
}

interface RawSnapshotSupplier {
  kind: "vendor" | "venue";
  supplierId: number;
  name: string;
  categoryId: number | null;
  paidBy: SupplierPayer;
  showName: boolean;
  capacityStanding: number | null;
  capacitySeated: number | null;
  agreedCost: number | null;
  notes: string;
}

interface RawSnapshotLine {
  kind: QuotationLineKind;
  parentIndex: number | null;
  packageId: number | null;
  packageRule: RawPackageRule | null;
  description: string;
  details: string;
  categoryId: number | null;
  eventIndex: number | null;
  quantity: number | null;
  unit: string | null;
  unitPrice: number | null;
  amount: number;
  isBonus: boolean;
  supplierIndex: number | null;
}

interface RawSnapshotTerm {
  label: string;
  percent: number | null;
  amount: number | null;
  dueBasis: TermDueBasis;
  dueOffsetDays: number;
  dueEventIndex: number | null;
  computed: number;
}

export interface RawContent {
  coupleNames: string;
  contactWhatsapp: string;
  isPkp: boolean;
  events: RawSnapshotEvent[] | null;
  suppliers: RawSnapshotSupplier[] | null;
  lines: RawSnapshotLine[] | null;
  terms: RawSnapshotTerm[] | null;
  termsText: string;
  totals: RincianTotals | null;
}

interface RawSnapshot {
  number: string;
  revisionN: number;
  issuedAt: string;
  validUntil: string;
  picName: string;
  categoryNames: Record<string, string> | null;
  content: RawContent;
}

export function toSnapshotEvent(e: RawSnapshotEvent): SnapshotEvent {
  return { ...e, eventKindId: String(e.eventKindId), eventDate: str(e.eventDate), sessions: e.sessions ?? [] };
}

export function toSnapshotSupplier(s: RawSnapshotSupplier): SnapshotSupplier {
  return { ...s, supplierId: String(s.supplierId), categoryId: idOf(s.categoryId), notes: str(s.notes), agreedCost: s.agreedCost ?? null };
}

export function toSnapshotLine(l: RawSnapshotLine): SnapshotLine {
  return {
    ...l,
    packageId: idOf(l.packageId),
    packageRule: toPackageRule(l.packageRule),
    details: str(l.details),
    categoryId: idOf(l.categoryId),
    unit: str(l.unit),
  };
}

function toSnapshotTerm(t: RawSnapshotTerm): SnapshotTerm {
  return t;
}

export function toContent(c: RawContent): SnapshotContent {
  return {
    coupleNames: c.coupleNames,
    contactWhatsapp: str(c.contactWhatsapp),
    isPkp: c.isPkp,
    events: (c.events ?? []).map(toSnapshotEvent),
    suppliers: (c.suppliers ?? []).map(toSnapshotSupplier),
    lines: (c.lines ?? []).map(toSnapshotLine),
    terms: (c.terms ?? []).map(toSnapshotTerm),
    termsText: str(c.termsText),
    totals: c.totals,
  };
}

export function toRevisionDetail(r: RawRevision): QuotationRevisionDetail {
  const snap = r.snapshot;
  return {
    ...toRevision(r),
    issuedAt: snap?.issuedAt ?? r.createdAt,
    validUntil: snap?.validUntil ?? "",
    picName: snap?.picName ?? "",
    categoryNames: snap?.categoryNames ?? {},
    content: toContent(
      snap?.content ?? { coupleNames: "", contactWhatsapp: "", isPkp: false, events: [], suppliers: [], lines: [], terms: [], termsText: "", totals: null },
    ),
  };
}

interface RawCompareRow<T> {
  change: "added" | "removed" | "changed";
  before: T | null;
  after: T | null;
}

export interface RawCompare {
  from: number;
  to: number;
  lines: Array<RawCompareRow<RawSnapshotLine> & { parent: string }> | null;
  unchangedLines: number;
  events: RawCompareRow<RawSnapshotEvent>[] | null;
  unchangedEvents: number;
  suppliers: RawCompareRow<RawSnapshotSupplier>[] | null;
  unchangedSuppliers: number;
  termsChanged: boolean;
  termsTextChanged: boolean;
  headerChanged: boolean;
  totalsBefore: RincianTotals | null;
  totalsAfter: RincianTotals | null;
}

function row<R, T>(r: RawCompareRow<R>, map: (x: R) => T) {
  return { change: r.change, before: r.before ? map(r.before) : null, after: r.after ? map(r.after) : null };
}

export function toCompare(r: RawCompare): QuotationCompare {
  return {
    from: r.from,
    to: r.to,
    lines: (r.lines ?? []).map((l) => ({ ...row(l, toSnapshotLine), parent: str(l.parent) })),
    unchangedLines: r.unchangedLines,
    events: (r.events ?? []).map((e) => row(e, toSnapshotEvent)),
    unchangedEvents: r.unchangedEvents,
    suppliers: (r.suppliers ?? []).map((s) => row(s, toSnapshotSupplier)),
    unchangedSuppliers: r.unchangedSuppliers,
    termsChanged: r.termsChanged,
    termsTextChanged: r.termsTextChanged,
    headerChanged: r.headerChanged,
    totalsBefore: r.totalsBefore,
    totalsAfter: r.totalsAfter,
  };
}

export interface RawActivity {
  id: number;
  type: string;
  actorStaffId: number | null;
  actorName: string | null;
  summary: string;
  createdAt: string;
}

export function toActivity(r: RawActivity): QuotationActivity {
  return { id: String(r.id), type: r.type, actorStaffId: idOf(r.actorStaffId), actorName: str(r.actorName), summary: r.summary, createdAt: r.createdAt };
}

export interface RawConversionPreview {
  quotationId: number;
  quotationNumber: string;
  revisionN: number;
  eventDate: string;
  contractTotal: number;
  isPkp: boolean;
  terms: Array<{ label: string; amount: number; dueDate: string }> | null;
  counts: { events: number; suppliers: number; lines: number; hasTermsText: boolean };
  existingProjectId: number | null;
}

export function toConversionPreview(r: RawConversionPreview): ConversionPreview {
  return {
    quotationId: String(r.quotationId),
    quotationNumber: r.quotationNumber,
    revisionN: r.revisionN,
    eventDate: str(r.eventDate),
    contractTotal: r.contractTotal,
    isPkp: r.isPkp,
    terms: r.terms ?? [],
    counts: r.counts,
    existingProjectId: idOf(r.existingProjectId),
  };
}

export interface RawAgreement {
  quotationId: number;
  quotationNumber: string;
  revisionId: number;
  revisionN: number;
  revisionCreatedAt: string;
  acceptedOn: string | null;
  acceptedChannel: AcceptChannel | "" | null;
  contractTotal: number;
  content: RawContent;
}

export function toAgreement(r: RawAgreement): ProjectAgreement {
  return {
    quotationId: String(r.quotationId),
    quotationNumber: r.quotationNumber,
    revisionId: String(r.revisionId),
    revisionN: r.revisionN,
    revisionCreatedAt: r.revisionCreatedAt,
    acceptedOn: str(r.acceptedOn),
    acceptedChannel: r.acceptedChannel ?? "",
    contractTotal: r.contractTotal,
    content: toContent(r.content),
  };
}
