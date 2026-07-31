import { create } from "zustand";
import { httpClient } from "@/shared/services/http-client";
import { API } from "@/shared/services/api-endpoints";
import type { Vendor, VendorSummary, VendorProjectHistoryItem, VendorImportResult, VendorImportRowError } from "@/modules/vendors/types";
import type { VendorFormValues, VendorCreateFormValues } from "@/modules/vendors/schemas/vendor.schema";
import type { CompressedFilePayload } from "@/shared/lib/image-compression";
import { compressFileForUpload } from "@/shared/lib/image-compression";
import { toPaginationMeta, EMPTY_PAGINATION_META, type PaginationMeta, type RawPaginationMeta } from "@/shared/types/pagination";

interface RawVendor {
  id: number;
  categoryId: number;
  name: string;
  picName: string;
  phone: string;
  email: string | null;
  socialMedia: string | null;
  city: string | null;
  address: string | null;
  priceAkad: number | null;
  priceAkadResepsi: number | null;
  notes: string;
  hasAttachment: boolean;
  attachmentIsImage: boolean;
  isActive: boolean;
  createdAt: string;
}

function toVendor(raw: RawVendor): Vendor {
  return { ...raw, id: String(raw.id), categoryId: String(raw.categoryId) };
}

interface RawVendorSummary {
  id: number;
  name: string;
}

function toVendorSummary(raw: RawVendorSummary): VendorSummary {
  return { id: String(raw.id), name: raw.name };
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

interface RawVendorImportResult {
  insertedCount: number;
  updatedCount: number;
  errors: VendorImportRowError[];
}

function toVendorImportResult(raw: RawVendorImportResult): VendorImportResult {
  return { insertedCount: raw.insertedCount, updatedCount: raw.updatedCount, errors: raw.errors };
}

// priceAkad/priceAkadResepsi are plain numbers in the form (0 == "not set",
// same convention as VenueFormModal's rentalPrice) — converted to null here
// so an unset field round-trips as SQL NULL, not a literal 0.
function toRequestBody(values: VendorFormValues | VendorCreateFormValues) {
  return {
    ...values,
    categoryId: Number(values.categoryId),
    priceAkad: values.priceAkad || null,
    priceAkadResepsi: values.priceAkadResepsi || null,
  };
}

interface VendorState {
  vendors: Vendor[];
  vendorPage: Vendor[];
  vendorPageMeta: PaginationMeta;
  vendorSummaries: VendorSummary[];
  fetchVendors: () => Promise<void>;
  fetchVendorPage: (page: number, search: string, categoryId: string) => Promise<void>;
  // Public-safe {id, name} list — the only vendor data Client Portal is
  // allowed to fetch directly now that commercial fields exist (see
  // vendor_handler.go's Summary handler doc comment).
  fetchVendorSummaries: () => Promise<void>;
  createVendor: (values: VendorCreateFormValues) => Promise<void>;
  updateVendor: (id: string, values: VendorFormValues) => Promise<void>;
  toggleVendorActive: (id: string) => Promise<void>;
  // Backs VendorListPage's "Lihat Project" modal — resolved server-side via
  // the `vendors` module calling into `projects.Contracts` (project_vendors
  // is owned by `projects`, not `vendors`). Not cached in store state since
  // it's only ever needed for whichever single vendor's modal is open.
  fetchVendorProjectHistory: (vendorId: string) => Promise<VendorProjectHistoryItem[]>;
  uploadVendorAttachment: (id: string, file: CompressedFilePayload) => Promise<void>;
  downloadVendorTemplate: () => Promise<Blob>;
  importVendors: (file: File) => Promise<VendorImportResult>;
}

// Backed by the real `vendors` module (Fase 3) — tenant-scoped. Also the
// single source of truth for vendor pickers elsewhere (e.g. `projects`'
// ProjectVendorFormModal). `GET /vendors` is staff-only (ADR-0016-style
// split) — Client Portal must use fetchVendorSummaries instead.
export const useVendorStore = create<VendorState>((set, get) => ({
  vendors: [],
  vendorPage: [],
  vendorPageMeta: EMPTY_PAGINATION_META,
  vendorSummaries: [],

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

  fetchVendorSummaries: async () => {
    const res = await httpClient.get(API.vendors.summary);
    set({ vendorSummaries: (res.data.data as RawVendorSummary[]).map(toVendorSummary) });
  },

  createVendor: async (values) => {
    await httpClient.post(API.vendors.base, toRequestBody(values));
    await get().fetchVendors();
  },

  updateVendor: async (id, values) => {
    await httpClient.patch(API.vendors.item(id), toRequestBody(values));
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

  uploadVendorAttachment: async (id, file) => {
    await httpClient.put(API.vendors.attachment(id), file);
  },

  downloadVendorTemplate: async () => {
    const res = await httpClient.get(API.vendors.template, { responseType: "blob" });
    return res.data as Blob;
  },

  // The .xlsx file itself is never an image, so compressFileForUpload passes
  // it through untouched (ADR-0010's "anything else passes through" rule) —
  // same reuse already established by useVenueStore.importVenues.
  importVendors: async (file) => {
    const compressed = await compressFileForUpload(file);
    const res = await httpClient.post(API.vendors.import, { base64Data: compressed.base64Data });
    return toVendorImportResult(res.data.data as RawVendorImportResult);
  },
}));
