import { create } from "zustand";
import { httpClient } from "@/shared/services/http-client";
import { API } from "@/shared/services/api-endpoints";
import { ifMatch } from "@/shared/lib/concurrency";
import { toPaginationMeta, EMPTY_PAGINATION_META, type PaginationMeta, type RawPaginationMeta } from "@/shared/types/pagination";

export type VoucherState = "active" | "inactive" | "scheduled" | "expired" | "exhausted";
export type DiscountType = "percent" | "amount";

/** A discount code for one App's plans (ADR-0041). Money is whole rupiah; dates are WIB days (YYYY-MM-DD). */
export interface Voucher {
  id: number;
  code: string;
  appId: string;
  /** null = every plan of the App. */
  planId: number | null;
  planName: string | null;
  discountType: DiscountType;
  /** Percent 1–99, or rupiah. */
  discountValue: number;
  /** Cap on a percent discount, in rupiah; null = no cap. */
  maxDiscount: number | null;
  validFrom: string | null;
  validUntil: string | null;
  /** Total uses; null = unlimited. */
  quota: number | null;
  oncePerCustomer: boolean;
  isActive: boolean;
  state: VoucherState;
  /** Paid uses. */
  usedCount: number;
  /** Uses whose payment is still pending (they count against the quota). */
  reservedCount: number;
  createdAt: string;
  /** The version a full edit sends back as If-Match (ADR-0036). */
  updatedAt: string;
}

/** What a create or a full edit sends: the whole voucher. */
export interface VoucherBody {
  code: string;
  appId: string;
  planId: number | null;
  discountType: DiscountType;
  discountValue: number;
  maxDiscount: number | null;
  validFrom: string;
  validUntil: string;
  quota: number | null;
  oncePerCustomer: boolean;
}

export type RedemptionStatus = "reserved" | "used" | "released";

export interface VoucherRedemption {
  id: number;
  orderRef: string;
  customerType: "tenant" | "app";
  customerRef: string;
  /** The tenant's name; "" for an App (the console names it from Manajemen Aplikasi). */
  customerName: string;
  planId: number;
  originalAmount: number;
  discountAmount: number;
  finalAmount: number;
  status: RedemptionStatus;
  createdAt: string;
  resolvedAt: string | null;
}

export type VoucherListParams = Record<string, string | number>;

interface VoucherStoreState {
  vouchers: Voucher[];
  meta: PaginationMeta;
  /** Loads one page of the list. A slower, older answer never replaces the page asked for since. */
  fetchVouchers: (params: VoucherListParams) => Promise<void>;
  createVoucher: (body: VoucherBody) => Promise<Voucher>;
  /** Full edit, guarded by the version the form was opened with. A 409 carries the voucher as it is now. */
  updateVoucher: (id: number, body: VoucherBody, updatedAt: string) => Promise<Voucher>;
  /** Explicit state, never a flip (ADR-0036). */
  setVoucherActive: (id: number, active: boolean) => Promise<Voucher>;
  /** One page of a voucher's uses. Not kept in the store: each dialog holds its own. */
  fetchRedemptions: (id: number, page: number) => Promise<{ rows: VoucherRedemption[]; meta: PaginationMeta }>;
}

let listSeq = 0;
let lastParams: VoucherListParams = {};

// Backed by the `billing` module's /vouchers endpoints. Fetch-then-set, no client cache (ADR-0009).
export const useVoucherStore = create<VoucherStoreState>((set, get) => {
  /** Reloads the page on screen after a write. Its failure is not the write's: the write already succeeded. */
  const reload = () => get().fetchVouchers(lastParams).catch(() => undefined);

  return {
    vouchers: [],
    meta: EMPTY_PAGINATION_META,

    fetchVouchers: async (params) => {
      const seq = ++listSeq;
      lastParams = params;
      const res = await httpClient.get(API.billing.vouchers, { params });
      if (seq !== listSeq) return;
      set({ vouchers: res.data.data as Voucher[], meta: toPaginationMeta(res.data.meta as RawPaginationMeta) });
    },

    createVoucher: async (body) => {
      const res = await httpClient.post(API.billing.vouchers, body);
      await reload();
      return res.data.data as Voucher;
    },

    updateVoucher: async (id, body, updatedAt) => {
      const res = await httpClient.patch(API.billing.voucher(id), body, { headers: ifMatch(updatedAt) });
      await reload();
      return res.data.data as Voucher;
    },

    setVoucherActive: async (id, active) => {
      const res = await httpClient.patch(API.billing.voucher(id), { isActive: active });
      await reload();
      return res.data.data as Voucher;
    },

    fetchRedemptions: async (id, page) => {
      const res = await httpClient.get(API.billing.voucherRedemptions(id), { params: { page, limit: 10 } });
      return { rows: res.data.data as VoucherRedemption[], meta: toPaginationMeta(res.data.meta as RawPaginationMeta) };
    },
  };
});
