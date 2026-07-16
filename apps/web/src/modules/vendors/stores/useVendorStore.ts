import { create } from "zustand";
import { httpClient } from "@/shared/services/http-client";
import { API } from "@/shared/services/api-endpoints";
import type { Vendor, VendorProjectHistoryItem } from "@/modules/vendors/types";
import type { VendorFormValues } from "@/modules/vendors/schemas/vendor.schema";
import { toPaginationMeta, EMPTY_PAGINATION_META, type PaginationMeta, type RawPaginationMeta } from "@/shared/types/pagination";

interface RawVendor {
  id: number;
  categoryId: number;
  name: string;
  picName: string;
  phone: string;
  email: string;
  address: string;
  notes: string;
  isActive: boolean;
  createdAt: string;
}

function toVendor(raw: RawVendor): Vendor {
  return { ...raw, id: String(raw.id), categoryId: String(raw.categoryId) };
}

interface RawVendorProjectHistoryItem {
  projectId: number;
  projectName: string;
  eventDate: string;
  venue: string;
  engagementStatus: string;
}

function toVendorProjectHistoryItem(raw: RawVendorProjectHistoryItem): VendorProjectHistoryItem {
  return { ...raw, projectId: String(raw.projectId) };
}

interface VendorState {
  vendors: Vendor[];
  vendorPage: Vendor[];
  vendorPageMeta: PaginationMeta;
  fetchVendors: () => Promise<void>;
  fetchVendorPage: (page: number, search: string, categoryId: string) => Promise<void>;
  createVendor: (values: VendorFormValues) => Promise<void>;
  updateVendor: (id: string, values: VendorFormValues) => Promise<void>;
  toggleVendorActive: (id: string) => Promise<void>;
  // Backs VendorListPage's "Lihat Project" modal — resolved server-side via
  // the `vendors` module calling into `projects.Contracts` (project_vendors
  // is owned by `projects`, not `vendors`). Not cached in store state since
  // it's only ever needed for whichever single vendor's modal is open.
  fetchVendorProjectHistory: (vendorId: string) => Promise<VendorProjectHistoryItem[]>;
}

// Backed by the real `vendors` module (Fase 3) — tenant-scoped. Also the
// single source of truth for vendor pickers elsewhere (e.g. `projects`'
// ProjectVendorFormModal).
export const useVendorStore = create<VendorState>((set, get) => ({
  vendors: [],
  vendorPage: [],
  vendorPageMeta: EMPTY_PAGINATION_META,

  fetchVendors: async () => {
    const res = await httpClient.get(API.vendors.base, { params: { all: true } });
    set({ vendors: (res.data.data as RawVendor[]).map(toVendor) });
  },

  // Backs VendorListPage's table — real server-side pagination + search/
  // category filtering, separate from the `vendors` full-roster cache above.
  fetchVendorPage: async (page, search, categoryId) => {
    const res = await httpClient.get(API.vendors.base, {
      params: { page, search: search || undefined, categoryId: categoryId || undefined },
    });
    set({
      vendorPage: (res.data.data as RawVendor[]).map(toVendor),
      vendorPageMeta: toPaginationMeta(res.data.meta as RawPaginationMeta),
    });
  },

  createVendor: async (values) => {
    await httpClient.post(API.vendors.base, { ...values, categoryId: Number(values.categoryId) });
    await get().fetchVendors();
  },

  updateVendor: async (id, values) => {
    await httpClient.patch(API.vendors.item(id), { ...values, categoryId: Number(values.categoryId) });
    await get().fetchVendors();
  },

  toggleVendorActive: async (id) => {
    await httpClient.post(API.vendors.toggleActive(id));
    await get().fetchVendors();
  },

  fetchVendorProjectHistory: async (vendorId) => {
    const res = await httpClient.get(API.vendors.projectHistory(vendorId));
    return (res.data.data as RawVendorProjectHistoryItem[]).map(toVendorProjectHistoryItem);
  },
}));
