import type { LineDraft, LineKind, Rincian, RincianLine } from "@/modules/rincian/types";

/** A line as the API sends it: ids are numbers, and null (or absent) where nothing is set. */
interface RawLine {
  id: number;
  key?: string;
  kind: LineKind;
  parentId: number | null;
  parentKey?: string;
  packageRule?: {
    packageId: number;
    name: string;
    price?: number;
    eventKindIds: number[] | null;
    includedGuests: number | null;
    extraGuestStep: number | null;
    extraGuestStepPrice?: number;
  };
  description: string;
  categoryId: number | null;
  eventId: number | null;
  quantity: number | null;
  unit: string;
  unitPrice?: number;
  amount?: number;
  isBonus: boolean;
  supplierRef?: number | null;
  sortOrder: number;
  extraGuests?: {
    eventId: number | null;
    eventKindName: string;
    guests: number;
    includedGuests: number;
    over: number;
    step: number;
    stepPrice?: number;
    steps: number;
    noGuests: boolean;
  };
  updatedAt?: string;
}

export interface RawRincian {
  projectId: number;
  lines: RawLine[] | null;
  totals?: Rincian["totals"];
  isPkp: boolean;
  linesVersion?: string;
  termsTotal?: number;
  hasTerms?: boolean;
  termsMismatch?: boolean;
  margin?: Rincian["margin"];
}

const idOf = (n: number | null | undefined): string => (n ? String(n) : "");

function toLine(raw: RawLine): RincianLine {
  return {
    id: idOf(raw.id),
    key: raw.key || undefined,
    kind: raw.kind,
    parentId: idOf(raw.parentId),
    parentKey: raw.parentKey || undefined,
    packageRule: raw.packageRule
      ? {
          packageId: String(raw.packageRule.packageId),
          name: raw.packageRule.name,
          price: raw.packageRule.price,
          eventKindIds: (raw.packageRule.eventKindIds ?? []).map(String),
          includedGuests: raw.packageRule.includedGuests ?? 0,
          extraGuestStep: raw.packageRule.extraGuestStep ?? 0,
          extraGuestStepPrice: raw.packageRule.extraGuestStepPrice,
        }
      : undefined,
    description: raw.description,
    categoryId: idOf(raw.categoryId),
    eventId: idOf(raw.eventId),
    quantity: raw.quantity ?? 0,
    unit: raw.unit ?? "",
    unitPrice: raw.unitPrice,
    amount: raw.amount,
    isBonus: raw.isBonus,
    supplierRef: raw.supplierRef === undefined ? undefined : idOf(raw.supplierRef),
    sortOrder: raw.sortOrder,
    extraGuests: raw.extraGuests ? { ...raw.extraGuests, eventId: idOf(raw.extraGuests.eventId) } : undefined,
    updatedAt: raw.updatedAt,
  };
}

/** GET …/lines (and every write and preview) as the screens hold it. */
export function toRincian(raw: RawRincian): Rincian {
  return {
    projectId: String(raw.projectId),
    lines: (raw.lines ?? []).map(toLine),
    totals: raw.totals,
    isPkp: raw.isPkp,
    linesVersion: raw.linesVersion,
    termsTotal: raw.termsTotal,
    hasTerms: raw.hasTerms,
    termsMismatch: raw.termsMismatch,
    margin: raw.margin,
  };
}

const numOrNull = (id: string | undefined): number | null => (id ? Number(id) : null);

/** One draft as PUT / preview take it. */
export function lineDraftBody(d: LineDraft) {
  return {
    id: d.id ? Number(d.id) : 0,
    key: d.key ?? "",
    parentId: d.parentId ? Number(d.parentId) : 0,
    parentKey: d.parentKey ?? "",
    kind: d.kind,
    packageId: d.packageId ? Number(d.packageId) : 0,
    description: d.description,
    categoryId: numOrNull(d.categoryId),
    eventId: numOrNull(d.eventId),
    quantity: d.quantity ?? null,
    unit: d.unit ?? "",
    unitPrice: d.unitPrice ?? null,
    amount: d.amount ?? 0,
    isBonus: d.isBonus ?? false,
    supplierRef: numOrNull(d.supplierRef),
  };
}
