import { create } from "zustand";
import { httpClient } from "@/shared/services/http-client";
import { API } from "@/shared/services/api-endpoints";
import { toPaginationMeta, type PaginationMeta, type RawPaginationMeta } from "@/shared/types/pagination";
import type { FinanceSummary, Payable, PendingVerification, Receivable } from "@/modules/finance/types";

type Params = Record<string, string | number>;

export interface Page<T> {
  rows: T[];
  meta: PaginationMeta;
}

// The API sends numeric ids; the app keys rows and routes by string ids (as every other module does).
type Raw<T> = Omit<T, "id" | "projectId"> & { id: number; projectId: number };

function ids<T extends { id: string; projectId: string }>(raw: Raw<T>): T {
  return { ...raw, id: String(raw.id), projectId: String(raw.projectId) } as unknown as T;
}

async function page<T extends { id: string; projectId: string }>(url: string, params: Params): Promise<Page<T>> {
  const res = await httpClient.get(url, { params });
  return {
    rows: (res.data.data as Raw<T>[]).map((r) => ids<T>(r)),
    meta: toPaginationMeta(res.data.meta as RawPaginationMeta),
  };
}

interface FinanceState {
  summary: FinanceSummary | null;
  /** The headline figures and the cash flow (one call). */
  fetchSummary: () => Promise<void>;
  // The lists live in their tab (one page at a time, never cached across tabs).
  fetchReceivables: (params: Params) => Promise<Page<Receivable>>;
  fetchPayables: (params: Params) => Promise<Page<Payable>>;
  fetchVerifications: (params: Params) => Promise<Page<PendingVerification>>;
}

export const useFinanceStore = create<FinanceState>((set) => ({
  summary: null,
  fetchSummary: async () => {
    const res = await httpClient.get(API.finance.summary);
    set({ summary: res.data.data as FinanceSummary });
  },
  fetchReceivables: (params) => page<Receivable>(API.finance.receivables, params),
  fetchPayables: (params) => page<Payable>(API.finance.payables, params),
  fetchVerifications: async (params) => {
    const res = await page<PendingVerification>(API.finance.verifications, params);
    // The supplier keys come as numbers (or null on a client payment row); everything else is keyed by string ids.
    const str = (v: unknown) => (v === null || v === undefined ? null : String(v));
    return {
      ...res,
      rows: res.rows.map((r) => ({
        ...r,
        kind: r.kind ?? "clientPayment",
        projectSupplierId: str(r.projectSupplierId),
        supplierName: r.supplierName ?? null,
        billTermId: str(r.billTermId),
        billTermLabel: r.billTermLabel ?? null,
      })),
    };
  },
}));
