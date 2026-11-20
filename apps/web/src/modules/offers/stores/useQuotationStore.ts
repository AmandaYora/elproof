import { create } from "zustand";
import axios from "axios";
import { httpClient } from "@/shared/services/http-client";
import { API } from "@/shared/services/api-endpoints";
import { ifMatch } from "@/shared/lib/concurrency";
import { downloadFile } from "@/shared/lib/download";
import { uploadFileVia } from "@/shared/lib/direct-upload";
import { toPaginationMeta, EMPTY_PAGINATION_META, type PaginationMeta, type RawPaginationMeta } from "@/shared/types/pagination";
import type { EventInput, LineDraft, Rincian } from "@/modules/rincian";
import {
  toActivity,
  toAgreement,
  toCompare,
  toConversionPreview,
  toDetail,
  toQuotationListItem,
  toRevision,
  toRevisionDetail,
  type RawActivity,
  type RawAgreement,
  type RawCompare,
  type RawConversionPreview,
  type RawDetail,
  type RawQuotationListItem,
  type RawRevision,
} from "@/modules/offers/stores/quotation-wire";
import {
  rincianFromDetail,
  rincianFromPreview,
  renameLineErrors,
  rewriteLinesConflict,
  toLinesBody,
  type RawLinesPreview,
} from "@/modules/offers/lib/rincian-adapter";
import type {
  AcceptChannel,
  ConversionInput,
  ConversionPreview,
  ConversionResult,
  ProjectAgreement,
  QuotationActivity,
  QuotationCompare,
  QuotationDetail,
  QuotationListItem,
  QuotationRevision,
  QuotationRevisionDetail,
  QuotationSummary,
  TermDueBasis,
} from "@/modules/offers/types";

type LoadStatus = "loading" | "error" | "ready";

export interface QuotationHeaderInput {
  coupleNames: string;
  contactWhatsapp: string;
  /** "" = the person creating it. */
  picStaffId: string;
  validUntil: string;
  notes: string;
}

export interface QuotationTermInput {
  label: string;
  percent: number | null;
  amount: number | null;
  dueBasis: TermDueBasis;
  dueOffsetDays: number;
  dueEventId: string;
}

export interface QuotationSupplierInput {
  vendorId: string;
  paidBy: "WO" | "Client";
  showName: boolean;
  /** Sent only by a role that sees costs; "" = not filled in. */
  agreedCost?: number | "";
  notes: string;
}

export type QuotationStatusRequest =
  | { status: "Accepted"; revisionN: number | null; acceptedOn: string; channel: AcceptChannel; extendUntil: string }
  | { status: "Rejected"; reason: string }
  | { status: "Draft" };

const idNum = (id: string): number | null => (id ? Number(id) : null);

function eventBody(input: EventInput) {
  return {
    eventKindId: Number(input.eventKindId),
    eventDate: input.eventDate || null,
    startTime: input.startTime || null,
    endTime: input.endTime || null,
    venueId: idNum(input.venueId),
    layout: input.layout,
    sessions: input.sessions.map((s) => ({ label: s.label, startTime: s.startTime || null, endTime: s.endTime || null, guests: s.guests })),
  };
}

function headerBody(v: QuotationHeaderInput) {
  return {
    coupleNames: v.coupleNames,
    contactWhatsapp: v.contactWhatsapp,
    picStaffId: v.picStaffId ? Number(v.picStaffId) : 0,
    validUntil: v.validUntil || null,
    notes: v.notes,
  };
}

function supplierBody(v: QuotationSupplierInput) {
  return {
    vendorId: v.vendorId ? Number(v.vendorId) : 0,
    paidBy: v.paidBy,
    showName: v.showName,
    ...(v.agreedCost === undefined ? {} : { agreedCost: v.agreedCost === "" ? null : v.agreedCost }),
    notes: v.notes,
  };
}

function conversionBody(input: ConversionInput) {
  return {
    name: input.name,
    brideName: input.brideName,
    groomName: input.groomName,
    events: input.eventDates.map((eventDate) => ({ eventDate })),
    prepStartDate: input.prepStartDate,
    picStaffId: input.picStaffId ? Number(input.picStaffId) : 0,
    picSalesStaffId: input.picSalesStaffId ? Number(input.picSalesStaffId) : null,
    description: input.description,
    downPayment: input.downPayment,
  };
}

/** The `code` a 422 carries in `errors.code` (e.g. "quotation_needs_revision"), or "". */
export function apiErrorCode(err: unknown): string {
  if (!axios.isAxiosError(err)) return "";
  const code = (err.response?.data as { errors?: { code?: unknown } } | undefined)?.errors?.code;
  return typeof code === "string" ? code : "";
}

function readFileBase64(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => resolve(String(reader.result).replace(/^data:[^,]*,/, ""));
    reader.onerror = () => reject(reader.error ?? new Error("File belum bisa dibaca"));
    reader.readAsDataURL(file);
  });
}

interface QuotationState {
  // The list (server-side paging, filtering and sorting).
  rows: QuotationListItem[];
  meta: PaginationMeta;
  listStatus: LoadStatus;
  summary: QuotationSummary | null;
  fetchQuotations: (params: Record<string, string | number>) => Promise<void>;
  fetchSummary: () => Promise<void>;

  // One quotation (the page). Every write answers the full detail, which replaces this.
  detail: QuotationDetail | null;
  detailFor: string | null;
  fetchDetail: (id: string) => Promise<QuotationDetail>;
  createQuotation: (input: QuotationHeaderInput) => Promise<QuotationDetail>;
  updateHeader: (id: string, input: QuotationHeaderInput, version: string) => Promise<QuotationDetail>;
  deleteQuotation: (id: string) => Promise<void>;
  saveTermsText: (id: string, termsText: string, version: string) => Promise<QuotationDetail>;
  saveTerms: (id: string, terms: QuotationTermInput[], version: string) => Promise<QuotationDetail>;

  createEvent: (id: string, input: EventInput) => Promise<QuotationDetail>;
  updateEvent: (id: string, eventId: string, input: EventInput, version: string) => Promise<QuotationDetail>;
  deleteEvent: (id: string, eventId: string) => Promise<QuotationDetail>;
  orderEvents: (id: string, ids: string[]) => Promise<QuotationDetail>;

  addSupplier: (id: string, input: QuotationSupplierInput) => Promise<QuotationDetail>;
  updateSupplier: (id: string, supplierId: string, input: QuotationSupplierInput, version: string) => Promise<QuotationDetail>;
  deleteSupplier: (id: string, supplierId: string) => Promise<QuotationDetail>;
  refreshSupplier: (id: string, supplierId: string) => Promise<QuotationDetail>;

  /** The rincian editor's save: drafts → quotation lines, answered as a Rincian. */
  saveLines: (id: string, drafts: LineDraft[], version: string, canMargin: boolean) => Promise<Rincian>;
  previewLines: (id: string, drafts: LineDraft[], signal: AbortSignal, canMargin: boolean) => Promise<Rincian>;
  /** "Perbarui dari katalog": the saved lines again, this package copied afresh from the catalog. */
  refreshPackage: (id: string, lineId: string) => Promise<QuotationDetail>;

  revisions: QuotationRevision[];
  revisionsFor: string | null;
  fetchRevisions: (id: string) => Promise<QuotationRevision[]>;
  /** "Buat revisi untuk klien": created false when nothing changed since the last one. */
  createRevision: (id: string) => Promise<{ revision: QuotationRevision; created: boolean; message: string }>;
  fetchRevision: (id: string, n: number) => Promise<QuotationRevisionDetail>;
  compare: (id: string, from: number, to: number) => Promise<QuotationCompare>;
  shareRevision: (id: string, n: number) => Promise<void>;
  downloadRevisionPdf: (id: string, n: number, number: string) => Promise<void>;
  /** The PDF as a blob (preview DRAF or one revision), for the in-app viewer. */
  fetchPdf: (url: string) => Promise<Blob>;

  activity: QuotationActivity[];
  activityFor: string | null;
  fetchActivity: (id: string) => Promise<void>;

  setStatus: (id: string, request: QuotationStatusRequest) => Promise<QuotationDetail>;
  copyQuotation: (id: string) => Promise<QuotationDetail>;
  createAlternative: (id: string) => Promise<QuotationDetail>;
  extend: (id: string, validUntil: string) => Promise<QuotationDetail>;
  uploadAcceptance: (id: string, file: File, version: string) => Promise<QuotationDetail>;
  downloadAcceptance: (id: string, name: string) => Promise<void>;

  // Jadikan proyek (the projects module's endpoints, used from this screen).
  previewConversion: (quotationId: string, input: ConversionInput) => Promise<ConversionPreview>;
  convert: (quotationId: string, input: ConversionInput, idempotencyKey: string) => Promise<ConversionResult>;
  uploadConversionProof: (quotationId: string, file: File) => Promise<string>;

  // Kesepakatan awal (read on a project made from a quotation).
  fetchAgreement: (projectId: string) => Promise<ProjectAgreement>;
}

/** Penawaran: the list, one quotation with everything on its page, and the conversion into a project. */
export const useQuotationStore = create<QuotationState>((set, get) => {
  /** A write answered the full detail: it becomes the page's. */
  function keep(raw: RawDetail): QuotationDetail {
    const detail = toDetail(raw);
    set({ detail, detailFor: detail.quotation.id });
    return detail;
  }
  let listRequest = 0;

  return {
    rows: [],
    meta: EMPTY_PAGINATION_META,
    listStatus: "loading",
    summary: null,

    fetchQuotations: async (params) => {
      const token = ++listRequest;
      set({ listStatus: "loading" });
      try {
        const res = await httpClient.get(API.quotations.base, { params });
        if (token !== listRequest) return;
        set({
          rows: ((res.data.data ?? []) as RawQuotationListItem[]).map(toQuotationListItem),
          meta: res.data.meta ? toPaginationMeta(res.data.meta as RawPaginationMeta) : EMPTY_PAGINATION_META,
          listStatus: "ready",
        });
      } catch (err) {
        if (token === listRequest) set({ listStatus: "error" });
        throw err;
      }
    },

    fetchSummary: async () => {
      const res = await httpClient.get(API.quotations.summary);
      set({ summary: res.data.data as QuotationSummary });
    },

    detail: null,
    detailFor: null,

    fetchDetail: async (id) => {
      const res = await httpClient.get(API.quotations.item(id));
      return keep(res.data.data as RawDetail);
    },

    createQuotation: async (input) => {
      const res = await httpClient.post(API.quotations.base, headerBody(input));
      return keep(res.data.data as RawDetail);
    },

    updateHeader: async (id, input, version) => {
      const res = await httpClient.put(API.quotations.item(id), headerBody(input), { headers: ifMatch(version) });
      return keep(res.data.data as RawDetail);
    },

    deleteQuotation: async (id) => {
      await httpClient.delete(API.quotations.item(id));
      set((s) => (s.detailFor === id ? { detail: null, detailFor: null } : s));
    },

    saveTermsText: async (id, termsText, version) => {
      const res = await httpClient.put(API.quotations.termsText(id), { termsText }, { headers: ifMatch(version) });
      return keep(res.data.data as RawDetail);
    },

    saveTerms: async (id, terms, version) => {
      const body = {
        terms: terms.map((t) => ({
          label: t.label,
          percent: t.percent,
          amount: t.amount,
          dueBasis: t.dueBasis,
          dueOffsetDays: t.dueOffsetDays,
          dueEventId: t.dueBasis === "Event" ? idNum(t.dueEventId) : null,
        })),
      };
      const res = await httpClient.put(API.quotations.terms(id), body, { headers: ifMatch(version) });
      return keep(res.data.data as RawDetail);
    },

    createEvent: async (id, input) => {
      const res = await httpClient.post(API.quotations.events(id), eventBody(input));
      return keep(res.data.data as RawDetail);
    },

    updateEvent: async (id, eventId, input, version) => {
      const res = await httpClient.put(API.quotations.event(id, eventId), eventBody(input), { headers: ifMatch(version) });
      return keep(res.data.data as RawDetail);
    },

    deleteEvent: async (id, eventId) => {
      const res = await httpClient.delete(API.quotations.event(id, eventId));
      return keep(res.data.data as RawDetail);
    },

    orderEvents: async (id, ids) => {
      const res = await httpClient.post(API.quotations.eventsOrder(id), { ids: ids.map(Number) });
      return keep(res.data.data as RawDetail);
    },

    addSupplier: async (id, input) => {
      const res = await httpClient.post(API.quotations.suppliers(id), supplierBody(input));
      return keep(res.data.data as RawDetail);
    },

    updateSupplier: async (id, supplierId, input, version) => {
      const res = await httpClient.put(API.quotations.supplier(id, supplierId), supplierBody(input), { headers: ifMatch(version) });
      return keep(res.data.data as RawDetail);
    },

    deleteSupplier: async (id, supplierId) => {
      const res = await httpClient.delete(API.quotations.supplier(id, supplierId));
      return keep(res.data.data as RawDetail);
    },

    refreshSupplier: async (id, supplierId) => {
      const res = await httpClient.post(API.quotations.supplierRefresh(id, supplierId));
      return keep(res.data.data as RawDetail);
    },

    saveLines: async (id, drafts, version, canMargin) => {
      const { lines, index } = toLinesBody(drafts, get().detailFor === id ? get().detail : null);
      try {
        const res = await httpClient.put(API.quotations.lines(id), { lines }, { headers: ifMatch(version), ownConflict: true });
        return rincianFromDetail(keep(res.data.data as RawDetail), canMargin);
      } catch (err) {
        // The editor shows its own "changed by someone else" panel from data.current, in its own shape.
        throw renameLineErrors(rewriteLinesConflict(err, canMargin), index);
      }
    },

    previewLines: async (id, drafts, signal, canMargin) => {
      const detail = get().detailFor === id ? get().detail : null;
      const { lines, index } = toLinesBody(drafts, detail);
      const res = await httpClient.post(API.quotations.linesPreview(id), { lines }, { signal });
      const current = get().detail;
      if (!current) throw new Error("Penawaran belum dimuat");
      return rincianFromPreview(res.data.data as RawLinesPreview, index, current, canMargin);
    },

    refreshPackage: async (id, lineId) => {
      const detail = get().detail;
      if (!detail || detail.quotation.id !== id) throw new Error("Penawaran belum dimuat");
      const lines = detail.lines
        .filter((l) => l.kind === "Package" || l.kind === "Item" || l.kind === "Discount")
        .map((l) => ({
          id: Number(l.id),
          kind: l.kind,
          packageId: idNum(l.packageId),
          refreshFromCatalog: l.id === lineId,
          description: l.id === lineId ? "" : l.description,
          details: l.details,
          categoryId: idNum(l.categoryId),
          eventId: idNum(l.eventId),
          quantity: l.kind === "Item" ? l.quantity : null,
          unit: l.kind === "Item" ? l.unit : null,
          unitPrice: l.kind === "Item" ? (l.unitPrice ?? 0) : null,
          amount: l.kind === "Discount" ? (l.amount ?? 0) : null,
          isBonus: l.isBonus,
          supplierRef: idNum(l.supplierRef),
        }));
      const res = await httpClient.put(API.quotations.lines(id), { lines }, { headers: ifMatch(detail.quotation.linesUpdatedAt) });
      return keep(res.data.data as RawDetail);
    },

    revisions: [],
    revisionsFor: null,

    fetchRevisions: async (id) => {
      const res = await httpClient.get(API.quotations.revisions(id));
      const revisions = ((res.data.data ?? []) as RawRevision[]).map(toRevision);
      set({ revisions, revisionsFor: id });
      return revisions;
    },

    createRevision: async (id) => {
      const res = await httpClient.post(API.quotations.revisions(id));
      const revision = toRevision(res.data.data as RawRevision);
      // The working copy now matches a revision: the page's banner and buttons follow.
      await Promise.all([get().fetchDetail(id), get().fetchRevisions(id)]);
      return { revision, created: res.status === 201, message: String(res.data.message ?? "") };
    },

    fetchRevision: async (id, n) => {
      const res = await httpClient.get(API.quotations.revision(id, n));
      return toRevisionDetail(res.data.data as RawRevision);
    },

    compare: async (id, from, to) => {
      const res = await httpClient.get(API.quotations.compare(id), { params: { from, to } });
      return toCompare(res.data.data as RawCompare);
    },

    shareRevision: async (id, n) => {
      await httpClient.post(API.quotations.revisionShare(id, n));
    },

    downloadRevisionPdf: async (id, n, number) => {
      await downloadFile(API.quotations.revisionPdf(id, n), {}, `${number}-Rev${n}.pdf`);
    },

    fetchPdf: async (url) => {
      const res = await httpClient.get<Blob>(url, { responseType: "blob" });
      return res.data;
    },

    activity: [],
    activityFor: null,

    fetchActivity: async (id) => {
      const res = await httpClient.get(API.quotations.activity(id));
      set({ activity: ((res.data.data ?? []) as RawActivity[]).map(toActivity), activityFor: id });
    },

    setStatus: async (id, request) => {
      const body =
        request.status === "Accepted"
          ? {
              status: "Accepted",
              revisionN: request.revisionN,
              acceptedOn: request.acceptedOn || null,
              channel: request.channel,
              extendUntil: request.extendUntil || null,
            }
          : request.status === "Rejected"
            ? { status: "Rejected", reason: request.reason }
            : { status: "Draft" };
      const res = await httpClient.post(API.quotations.status(id), body);
      return keep(res.data.data as RawDetail);
    },

    copyQuotation: async (id) => {
      const res = await httpClient.post(API.quotations.copy(id));
      return toDetail(res.data.data as RawDetail);
    },

    createAlternative: async (id) => {
      const res = await httpClient.post(API.quotations.alternatives(id));
      return toDetail(res.data.data as RawDetail);
    },

    extend: async (id, validUntil) => {
      const res = await httpClient.post(API.quotations.extend(id), { validUntil });
      return keep(res.data.data as RawDetail);
    },

    uploadAcceptance: async (id, file, version) => {
      const base64Data = await readFileBase64(file);
      const res = await httpClient.put(
        API.quotations.acceptanceAttachment(id),
        { fileName: file.name, mimeType: file.type, base64Data },
        { headers: ifMatch(version) },
      );
      return keep(res.data.data as RawDetail);
    },

    downloadAcceptance: async (id, name) => {
      await downloadFile(API.quotations.acceptanceAttachment(id), {}, name || "bukti-penerimaan");
    },

    previewConversion: async (quotationId, input) => {
      const res = await httpClient.post(API.projects.fromQuotationPreview(quotationId), conversionBody(input));
      return toConversionPreview(res.data.data as RawConversionPreview);
    },

    convert: async (quotationId, input, idempotencyKey) => {
      const res = await httpClient.post(API.projects.fromQuotation(quotationId), conversionBody(input), {
        headers: { "Idempotency-Key": idempotencyKey },
      });
      const data = res.data.data as { id: number; created: boolean; downPaymentId: number | null };
      return { projectId: String(data.id), created: data.created, downPaymentId: data.downPaymentId ? String(data.downPaymentId) : "" };
    },

    uploadConversionProof: (quotationId, file) => uploadFileVia(API.projects.fromQuotationUploads(quotationId), file),

    fetchAgreement: async (projectId) => {
      const res = await httpClient.get(API.projects.agreement(projectId));
      return toAgreement(res.data.data as RawAgreement);
    },
  };
});
