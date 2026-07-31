import { create } from "zustand";
import { httpClient } from "@/shared/services/http-client";
import { API } from "@/shared/services/api-endpoints";
import type { Venue, VenueImportResult, VenueImportRowError } from "@/modules/venues/types";
import type { VenueFormValues, VenueCreateFormValues } from "@/modules/venues/schemas/venue.schema";
import type { CompressedFilePayload } from "@/shared/lib/image-compression";
import { compressFileForUpload } from "@/shared/lib/image-compression";
import { toPaginationMeta, EMPTY_PAGINATION_META, type PaginationMeta, type RawPaginationMeta } from "@/shared/types/pagination";

interface RawVenue {
  id: number;
  name: string;
  picName: string;
  phonePic: string;
  phoneVenue: string | null;
  email: string | null;
  address: string | null;
  city: string | null;
  rentalPrice: number | null;
  charge: number | null;
  capacity: number | null;
  facilities: string | null;
  socialMedia: string | null;
  notes: string;
  hasAttachment: boolean;
  attachmentIsImage: boolean;
  isActive: boolean;
  createdAt: string;
}

function toVenue(raw: RawVenue): Venue {
  return { ...raw, id: String(raw.id) };
}

interface RawVenueImportResult {
  insertedCount: number;
  updatedCount: number;
  errors: VenueImportRowError[];
}

function toVenueImportResult(raw: RawVenueImportResult): VenueImportResult {
  return { insertedCount: raw.insertedCount, updatedCount: raw.updatedCount, errors: raw.errors };
}

// rentalPrice/charge/capacity are plain numbers in the form (0 == "not
// set", same convention as ProjectFormModal's contractValue) — converted to
// null here so an unset field round-trips as SQL NULL, not a literal 0.
function toRequestBody(values: VenueFormValues | VenueCreateFormValues) {
  return {
    ...values,
    rentalPrice: values.rentalPrice || null,
    charge: values.charge || null,
    capacity: values.capacity || null,
  };
}

interface VenueState {
  venues: Venue[];
  venuePage: Venue[];
  venuePageMeta: PaginationMeta;
  fetchVenues: () => Promise<void>;
  fetchVenuePage: (page: number, search: string) => Promise<void>;
  // Full staff-only record (rental price, charge, PIC contact) — used by
  // Project Detail's Venue tab once it knows which venue is attached, never
  // by Client Portal (see ADR-0016).
  fetchVenue: (id: string) => Promise<Venue>;
  createVenue: (values: VenueCreateFormValues) => Promise<void>;
  updateVenue: (id: string, values: VenueFormValues) => Promise<void>;
  toggleVenueActive: (id: string) => Promise<void>;
  uploadVenueAttachment: (id: string, file: CompressedFilePayload) => Promise<void>;
  downloadVenueTemplate: () => Promise<Blob>;
  importVenues: (file: File) => Promise<VenueImportResult>;
}

// Backed by the `vendors` module's Venue directory (ADR-0016) — tenant-scoped.
// Also the source of truth for the venue picker elsewhere (e.g. Project
// Detail's "Pilih Venue" action).
export const useVenueStore = create<VenueState>((set, get) => ({
  venues: [],
  venuePage: [],
  venuePageMeta: EMPTY_PAGINATION_META,

  fetchVenues: async () => {
    const res = await httpClient.get(API.venues.base, { params: { all: true } });
    set({ venues: (res.data.data as RawVenue[]).map(toVenue) });
  },

  // Backs VenueListPage's table — real server-side pagination + search,
  // separate from the `venues` full-roster cache above.
  fetchVenuePage: async (page, search) => {
    const res = await httpClient.get(API.venues.base, { params: { page, search: search || undefined } });
    set({
      venuePage: (res.data.data as RawVenue[]).map(toVenue),
      venuePageMeta: toPaginationMeta(res.data.meta as RawPaginationMeta),
    });
  },

  fetchVenue: async (id) => {
    const res = await httpClient.get(API.venues.item(id));
    return toVenue(res.data.data as RawVenue);
  },

  createVenue: async (values) => {
    await httpClient.post(API.venues.base, toRequestBody(values));
    await get().fetchVenues();
  },

  updateVenue: async (id, values) => {
    await httpClient.patch(API.venues.item(id), toRequestBody(values));
    await get().fetchVenues();
  },

  toggleVenueActive: async (id) => {
    await httpClient.post(API.venues.toggleActive(id));
    await get().fetchVenues();
  },

  uploadVenueAttachment: async (id, file) => {
    await httpClient.put(API.venues.attachment(id), file);
  },

  downloadVenueTemplate: async () => {
    const res = await httpClient.get(API.venues.template, { responseType: "blob" });
    return res.data as Blob;
  },

  // The .xlsx file itself is never an image, so compressFileForUpload passes
  // it through untouched (ADR-0010's "anything else passes through" rule) —
  // this just reuses that existing file->base64 path, not a new one.
  importVenues: async (file) => {
    const compressed = await compressFileForUpload(file);
    const res = await httpClient.post(API.venues.import, { base64Data: compressed.base64Data });
    return toVenueImportResult(res.data.data as RawVenueImportResult);
  },
}));
