// The bridge between a quotation and the shared rincian editor (`@/modules/rincian`): the editor speaks `Rincian` and
// `LineDraft`, the quotation API speaks its own detail, preview and line body. Nothing here prices a line: every
// amount comes from the server.
import axios from "axios";
import { toRincian, type LineDraft, type RawRincian, type Rincian, type RincianEvent, type RincianTotals } from "@/modules/rincian";
import type { QuotationDetail, QuotationEvent, QuotationLine, QuotationLineKind } from "@/modules/offers/types";
import type { RawDetail, RawPackageRule } from "@/modules/offers/stores/quotation-wire";
import { toDetail } from "@/modules/offers/stores/quotation-wire";
import type { PackageRule } from "@/modules/rincian";

type RawRincianLine = NonNullable<RawRincian["lines"]>[number];

const num = (id: string | undefined): number | null => (id ? Number(id) : null);

function rawRule(rule: PackageRule | null): RawRincianLine["packageRule"] {
  if (!rule) return undefined;
  return {
    packageId: Number(rule.packageId),
    name: rule.name,
    price: rule.price,
    eventKindIds: rule.eventKindIds.map(Number),
    includedGuests: rule.includedGuests,
    extraGuestStep: rule.extraGuestStep,
    extraGuestStepPrice: rule.extraGuestStepPrice,
  };
}

/** The covered event with the most guests: what "Tambahan tamu" is counted from (PLAN §1). */
function largestCovered(rule: PackageRule, events: QuotationEvent[]): QuotationEvent | null {
  let best: QuotationEvent | null = null;
  for (const e of events) {
    if (!rule.eventKindIds.includes(e.eventKindId)) continue;
    if (!best || e.totalGuests > best.totalGuests) best = e;
  }
  return best;
}

/** How the "Tambahan tamu" amount was worked out, for the editor's explanation (display only; the amount is the server's). */
function extraGuestsInfo(rule: PackageRule | null, events: QuotationEvent[], amount: number, known?: { guests: number; over: number; steps: number }) {
  if (!rule) return undefined;
  const event = largestCovered(rule, events);
  const guests = known?.guests ?? event?.totalGuests ?? 0;
  const over = known?.over ?? Math.max(0, guests - rule.includedGuests);
  const stepPrice = rule.extraGuestStepPrice;
  const steps = known?.steps ?? (stepPrice ? Math.round(amount / stepPrice) : rule.extraGuestStep > 0 ? Math.ceil(over / rule.extraGuestStep) : 0);
  return {
    eventId: event ? Number(event.id) : null,
    eventKindName: event?.kindName ?? "",
    guests,
    includedGuests: rule.includedGuests,
    over,
    step: rule.extraGuestStep,
    stepPrice,
    steps,
    noGuests: guests === 0,
  };
}

/** The margin panel, only for a viewer who receives costs (finance.margin.view). */
function marginOf(detail: QuotationDetail, totals: RincianTotals | null, canMargin: boolean): Rincian["margin"] {
  if (!canMargin || !totals) return undefined;
  const paid = detail.suppliers.filter((s) => s.paidBy === "WO");
  const supplierCost = paid.reduce((sum, s) => sum + (s.agreedCost ?? 0), 0);
  const missingCost = paid.filter((s) => s.agreedCost === null || s.agreedCost === undefined).length;
  return { subtotal: totals.subtotal, supplierCost, missingCost, incomplete: missingCost > 0, margin: totals.subtotal - supplierCost };
}

function rawLine(l: QuotationLine, byId: Map<string, QuotationLine>, events: QuotationEvent[]): RawRincianLine {
  const parent = l.parentLineId ? byId.get(l.parentLineId) : undefined;
  return {
    id: Number(l.id),
    kind: l.kind,
    parentId: num(l.parentLineId),
    packageRule: rawRule(l.packageRule),
    description: l.description,
    categoryId: num(l.categoryId),
    eventId: num(l.eventId),
    quantity: l.quantity,
    unit: l.unit,
    unitPrice: l.unitPrice,
    amount: l.amount,
    isBonus: l.isBonus,
    supplierRef: num(l.supplierRef),
    sortOrder: l.sortOrder,
    extraGuests: l.kind === "ExtraGuests" ? extraGuestsInfo(parent?.packageRule ?? null, events, l.amount ?? 0) : undefined,
  };
}

/** The quotation's saved lines as the editor's wire shape (also what a 409 hands the editor as `data.current`). */
export function rawRincianFromDetail(detail: QuotationDetail, canMargin: boolean): RawRincian {
  const byId = new Map(detail.lines.map((l) => [l.id, l]));
  return {
    projectId: Number(detail.quotation.id),
    lines: detail.lines.map((l) => rawLine(l, byId, detail.events)),
    totals: detail.totals ?? undefined,
    isPkp: detail.quotation.isPkp,
    linesVersion: detail.quotation.linesUpdatedAt,
    termsTotal: detail.terms.reduce((sum, t) => sum + t.computed, 0),
    hasTerms: detail.terms.length > 0,
    termsMismatch: detail.termsMismatch,
    margin: marginOf(detail, detail.totals, canMargin),
  };
}

export function rincianFromDetail(detail: QuotationDetail, canMargin: boolean): Rincian {
  return toRincian(rawRincianFromDetail(detail, canMargin));
}

/* ------------------------------------------------------------------ drafts → PUT/preview body ------------------- */

export interface QuotationLineBody {
  id: number;
  kind: "Package" | "Item" | "Discount";
  packageId: number | null;
  refreshFromCatalog: boolean;
  /** Absent: keep the saved isi (or copy the catalog's for a new package). */
  components?: Array<{ description: string; details: string; categoryId: number | null }>;
  description: string;
  details: string;
  categoryId: number | null;
  eventId: number | null;
  quantity: number | null;
  unit: string | null;
  unitPrice: number | null;
  amount: number | null;
  isBonus: boolean;
  supplierRef: number | null;
}

/** Which draft each body row (and each of a package's components) came from: to put server messages on the right row. */
export interface BodyIndex {
  top: number[];
  components: number[][];
  /** The top-level drafts, in body order. */
  drafts: LineDraft[];
  /** The component drafts of each top-level row (null: absent, the server copies them). */
  componentDrafts: Array<LineDraft[] | null>;
}

/**
 * The editor's drafts as PUT/preview take them: only top-level rows (a package carries its isi as `components`), a
 * bonus is an Item with isBonus. A saved package always sends its isi as the editor shows it, so a removed or added
 * row is saved; a new one sends none and the server copies the catalog's.
 */
export function toLinesBody(drafts: LineDraft[], detail: QuotationDetail | null): { lines: QuotationLineBody[]; index: BodyIndex } {
  const saved = new Map((detail?.lines ?? []).map((l) => [l.id, l]));
  const details = (id: string | undefined) => (id ? saved.get(id)?.details ?? "" : "");
  const lines: QuotationLineBody[] = [];
  const index: BodyIndex = { top: [], components: [], drafts: [], componentDrafts: [] };
  drafts.forEach((d, i) => {
    if (d.kind === "Component") return;
    const base = {
      id: d.id ? Number(d.id) : 0,
      description: d.description,
      details: details(d.id),
      categoryId: num(d.categoryId),
      eventId: num(d.eventId),
      supplierRef: num(d.supplierRef),
      refreshFromCatalog: false,
    };
    let comps: number[] = [];
    let compDrafts: LineDraft[] | null = null;
    if (d.kind === "Package") {
      const packageId = d.packageId ? Number(d.packageId) : num(saved.get(d.id ?? "")?.packageId);
      const body: QuotationLineBody = { ...base, kind: "Package", packageId, quantity: null, unit: null, unitPrice: null, amount: null, isBonus: false };
      if (d.id) {
        comps = drafts.flatMap((c, j) => (c.kind === "Component" && ((c.parentId && c.parentId === d.id) || (c.parentKey && c.parentKey === d.key)) ? [j] : []));
        compDrafts = comps.map((j) => drafts[j]);
        body.components = compDrafts.map((c) => ({ description: c.description, details: details(c.id), categoryId: num(c.categoryId) }));
      }
      lines.push(body);
    } else if (d.kind === "Item") {
      lines.push({
        ...base,
        kind: "Item",
        packageId: null,
        quantity: d.quantity ?? 1,
        unit: d.unit ?? "",
        unitPrice: d.isBonus ? 0 : (d.unitPrice ?? 0),
        amount: null,
        isBonus: d.isBonus ?? false,
      });
    } else {
      lines.push({ ...base, kind: "Discount", packageId: null, quantity: null, unit: null, unitPrice: null, amount: d.amount ?? 0, isBonus: false, supplierRef: null });
    }
    index.top.push(i);
    index.components.push(comps);
    index.drafts.push(d);
    index.componentDrafts.push(compDrafts);
  });
  return { lines, index };
}

/**
 * The server's row keys ("lines.2.unitPrice", "lines.0.components.1.description") renamed to the draft the editor sent
 * ("lines.<draft index>.<field>"), so its own error mapping finds the row.
 */
export function renameLineErrors(err: unknown, index: BodyIndex): unknown {
  if (!axios.isAxiosError(err)) return err;
  const data = err.response?.data as { errors?: Record<string, unknown> } | undefined;
  if (!data?.errors || typeof data.errors !== "object") return err;
  const next: Record<string, unknown> = {};
  for (const [key, value] of Object.entries(data.errors)) {
    const comp = /^lines\.(\d+)\.components\.(\d+)\.(\w+)$/.exec(key);
    const top = /^lines\.(\d+)\.(\w+)$/.exec(key);
    let to = key;
    if (comp) {
      const draft = index.components[Number(comp[1])]?.[Number(comp[2])];
      if (draft !== undefined) to = `lines.${draft}.${comp[3]}`;
    } else if (top) {
      const draft = index.top[Number(top[1])];
      if (draft !== undefined) to = `lines.${draft}.${top[2]}`;
    }
    if (!(to in next)) next[to] = value;
  }
  data.errors = next;
  return err;
}

/** A stale save: the editor reads `data.current` as its own wire shape, so the quotation detail is converted for it. */
export function rewriteLinesConflict(err: unknown, canMargin: boolean): unknown {
  if (!axios.isAxiosError(err) || err.response?.status !== 409) return err;
  // The quotation API puts the full detail straight in `data`; the editor reads `data.current`.
  const data = err.response.data as { data?: unknown } | undefined;
  const current = data?.data as RawDetail | undefined;
  if (data && current && typeof current === "object" && "quotation" in current) {
    data.data = { current: rawRincianFromDetail(toDetail(current), canMargin) };
  }
  return err;
}

/* ------------------------------------------------------------------ preview → Rincian --------------------------- */

export interface RawLinesPreview {
  lines: Array<{
    index: number;
    parentIndex: number | null;
    kind: QuotationLineKind;
    packageId: number | null;
    packageRule: RawPackageRule | null;
    description: string;
    quantity: number | null;
    unitPrice: number | null;
    amount: number;
    isBonus: boolean;
    extraGuests: { guests: number; over: number; steps: number; amount: number } | null;
  }> | null;
  totals: RincianTotals;
}

function toRule(r: RawPackageRule | null): PackageRule | null {
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

/**
 * What a preview comes to, keyed the way the editor finds its rows again: each top-level row by its draft's key, a
 * saved package's isi by the component drafts sent, a new package's isi (copied from the catalog) without a key under
 * its package, and the "Tambahan tamu" row under its package.
 */
export function rincianFromPreview(raw: RawLinesPreview, index: BodyIndex, detail: QuotationDetail, canMargin: boolean): Rincian {
  const rows = raw.lines ?? [];
  const lines: RawRincianLine[] = [];
  let top = -1;
  const compSeen = new Map<number, number>();
  const topOfRow = new Map<number, number>();
  for (const r of rows) {
    if (r.parentIndex === null || r.parentIndex === undefined) {
      top += 1;
      topOfRow.set(r.index, top);
      const draft = index.drafts[top];
      lines.push({
        id: draft?.id ? Number(draft.id) : 0,
        key: draft?.key,
        kind: r.kind,
        parentId: null,
        packageRule: rawRule(toRule(r.packageRule)),
        description: r.description,
        categoryId: num(draft?.categoryId),
        eventId: num(draft?.eventId),
        quantity: r.quantity,
        unit: draft?.unit ?? "",
        unitPrice: r.unitPrice ?? undefined,
        amount: r.amount,
        isBonus: r.isBonus,
        supplierRef: num(draft?.supplierRef),
        sortOrder: lines.length,
      });
      continue;
    }
    const parentTop = topOfRow.get(r.parentIndex) ?? -1;
    const parentDraft = index.drafts[parentTop];
    const parentRow = rows.find((x) => x.index === r.parentIndex);
    if (r.kind === "ExtraGuests") {
      lines.push({
        id: 0,
        kind: "ExtraGuests",
        parentId: parentDraft?.id ? Number(parentDraft.id) : null,
        parentKey: parentDraft?.key,
        description: r.description,
        categoryId: null,
        eventId: null,
        quantity: null,
        unit: "",
        amount: r.amount,
        isBonus: false,
        sortOrder: lines.length,
        extraGuests: extraGuestsInfo(toRule(parentRow?.packageRule ?? null), detail.events, r.amount, parentRow?.extraGuests ?? undefined),
      });
      continue;
    }
    const j = compSeen.get(parentTop) ?? 0;
    compSeen.set(parentTop, j + 1);
    const compDraft = index.componentDrafts[parentTop]?.[j];
    lines.push({
      id: compDraft?.id ? Number(compDraft.id) : 0,
      key: compDraft?.key,
      kind: "Component",
      parentId: parentDraft?.id ? Number(parentDraft.id) : null,
      parentKey: parentDraft?.key,
      description: r.description,
      categoryId: num(compDraft?.categoryId),
      eventId: null,
      quantity: null,
      unit: "",
      amount: 0,
      isBonus: false,
      sortOrder: lines.length,
    });
  }
  return toRincian({
    projectId: Number(detail.quotation.id),
    lines,
    totals: raw.totals,
    isPkp: detail.quotation.isPkp,
    linesVersion: detail.quotation.linesUpdatedAt,
    termsTotal: detail.terms.reduce((sum, t) => sum + t.computed, 0),
    hasTerms: detail.terms.length > 0,
    termsMismatch: false,
    margin: marginOf(detail, raw.totals, canMargin),
  });
}

/** The quotation's events as the shared event list and drawer read them (the venue's capacities come from its supplier row). */
export function rincianEventsFromDetail(detail: QuotationDetail): RincianEvent[] {
  const suppliers = new Map(detail.suppliers.map((s) => [s.id, s]));
  return detail.events.map((e) => {
    const sup = e.venueSupplierId ? suppliers.get(e.venueSupplierId) : undefined;
    return {
      updatedAt: e.updatedAt,
      id: e.id,
      eventKindId: e.eventKindId,
      kindName: e.kindName,
      eventDate: e.eventDate,
      startTime: e.startTime,
      endTime: e.endTime,
      venue: e.venueId
        ? {
            supplierId: e.venueSupplierId,
            venueId: e.venueId,
            name: e.venueName || sup?.name || "",
            capacityStanding: sup?.capacityStanding ?? null,
            capacitySeated: sup?.capacitySeated ?? null,
            address: null,
            city: null,
          }
        : null,
      layout: e.layout,
      isMain: false,
      order: e.sortOrder,
      sessions: e.sessions,
      totalGuests: e.totalGuests,
    };
  });
}
