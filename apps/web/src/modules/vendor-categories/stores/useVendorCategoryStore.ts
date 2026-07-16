import { create } from "zustand";
import { httpClient } from "@/shared/services/http-client";
import { API } from "@/shared/services/api-endpoints";
import type { VendorCategory } from "@/modules/vendor-categories/types";
import type { VendorCategoryFormValues } from "@/modules/vendor-categories/schemas/vendor-category.schema";
import { toPaginationMeta, EMPTY_PAGINATION_META, type PaginationMeta, type RawPaginationMeta } from "@/shared/types/pagination";

interface RawVendorCategory {
  id: number;
  name: string;
  description: string;
  isActive: boolean;
  createdAt: string;
}

function toVendorCategory(raw: RawVendorCategory): VendorCategory {
  return { ...raw, id: String(raw.id) };
}

interface VendorCategoryState {
  categories: VendorCategory[];
  categoryPage: VendorCategory[];
  categoryPageMeta: PaginationMeta;
  fetchCategories: () => Promise<void>;
  fetchCategoryPage: (page: number, search: string) => Promise<void>;
  createCategory: (values: VendorCategoryFormValues) => Promise<void>;
  updateCategory: (id: string, values: VendorCategoryFormValues) => Promise<void>;
  toggleCategoryActive: (id: string) => Promise<void>;
}

// Backed by the real `vendors` module (Fase 3) — tenant-scoped. Also the
// single source of truth for category pickers/filters elsewhere.
export const useVendorCategoryStore = create<VendorCategoryState>((set, get) => ({
  categories: [],
  categoryPage: [],
  categoryPageMeta: EMPTY_PAGINATION_META,

  fetchCategories: async () => {
    const res = await httpClient.get(API.vendors.categories, { params: { all: true } });
    set({ categories: (res.data.data as RawVendorCategory[]).map(toVendorCategory) });
  },

  // Backs VendorCategoryListPage's table — real server-side pagination +
  // search, separate from the `categories` full-roster cache above (which the
  // vendor category picker/filter dropdowns still rely on).
  fetchCategoryPage: async (page, search) => {
    const res = await httpClient.get(API.vendors.categories, { params: { page, search: search || undefined } });
    set({
      categoryPage: (res.data.data as RawVendorCategory[]).map(toVendorCategory),
      categoryPageMeta: toPaginationMeta(res.data.meta as RawPaginationMeta),
    });
  },

  createCategory: async (values) => {
    await httpClient.post(API.vendors.categories, values);
    await get().fetchCategories();
  },

  updateCategory: async (id, values) => {
    await httpClient.patch(API.vendors.category(id), values);
    await get().fetchCategories();
  },

  toggleCategoryActive: async (id) => {
    await httpClient.post(API.vendors.categoryToggleActive(id));
    await get().fetchCategories();
  },
}));
