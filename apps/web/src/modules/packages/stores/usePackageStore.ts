import { create } from "zustand";
import { httpClient } from "@/shared/services/http-client";
import { API } from "@/shared/services/api-endpoints";
import { ifMatch } from "@/shared/lib/concurrency";
import { renameApiFieldErrors } from "@/shared/lib/api-error";
import { toPaginationMeta, EMPTY_PAGINATION_META, type PaginationMeta, type RawPaginationMeta } from "@/shared/types/pagination";
import type { Package, PackagePreview } from "@/modules/packages/types";
import type { PackageFormValues } from "@/modules/packages/schemas/package.schema";

interface RawRef {
  id: number;
  name: string;
}

interface RawPackage extends Omit<Package, "id" | "defaultVenue" | "eventKinds" | "components"> {
  id: number;
  defaultVenue: RawRef | null;
  eventKinds: RawRef[];
  components: Array<{
    id: number;
    description: string;
    categoryId: number | null;
    categoryName: string | null;
    vendorId: number | null;
    vendorName: string | null;
    estimatedCost?: number | null;
  }>;
}

const ref = (r: RawRef) => ({ id: String(r.id), name: r.name });
const idOrNull = (id: number | null) => (id === null ? null : String(id));

function toPackage(raw: RawPackage): Package {
  return {
    ...raw,
    id: String(raw.id),
    defaultVenue: raw.defaultVenue ? ref(raw.defaultVenue) : null,
    eventKinds: raw.eventKinds.map(ref),
    components: raw.components.map((c) => ({ ...c, id: String(c.id), categoryId: idOrNull(c.categoryId), vendorId: idOrNull(c.vendorId) })),
  };
}

const numberOrNull = (text: string) => (text === "" ? null : Number(text));
const positiveOrNull = (n: number) => (n > 0 ? n : null);
const idNumber = (id: string) => (id === "" ? null : Number(id));

/** The form as the API takes it. Costs are sent only by a role that sees them; otherwise the server keeps each row's own. */
function toRequestBody(values: PackageFormValues, withCosts: boolean) {
  return {
    name: values.name,
    price: values.price,
    eventKindIds: values.eventKindIds.map(Number),
    includedGuests: numberOrNull(values.includedGuests),
    extraGuestStep: numberOrNull(values.extraGuestStep),
    extraGuestStepPrice: positiveOrNull(values.extraGuestStepPrice),
    defaultVenueId: idNumber(values.defaultVenueId),
    components: values.components.map((c) => ({
      id: c.id === "" ? 0 : Number(c.id),
      description: c.description,
      categoryId: idNumber(c.categoryId),
      vendorId: idNumber(c.vendorId),
      ...(withCosts ? { estimatedCost: positiveOrNull(c.estimatedCost) } : {}),
    })),
  };
}

/** The server names a row "components.2.description"; the form's fields are "components[2].description". */
function rowKeys(err: unknown): unknown {
  return renameApiFieldErrors(err, (key) => key.replace(/^components\.(\d+)\./, "components[$1]."));
}

export interface PreviewRequest {
  price: number;
  includedGuests: number | null;
  extraGuestStep: number | null;
  extraGuestStepPrice: number | null;
  guests: number;
}

interface PackageState {
  packagePage: Package[];
  packagePageMeta: PaginationMeta;
  /** One page of the list. `params` is the list's own query: page, limit, sort, search and filters. */
  fetchPackagePage: (params: Record<string, string | number>) => Promise<void>;
  fetchPackage: (id: string) => Promise<Package>;
  /** Active packages whose name matches (the rincian's "Tambah paket"); leaves the list page's state alone. */
  searchActivePackages: (search: string) => Promise<Package[]>;
  createPackage: (values: PackageFormValues, withCosts: boolean) => Promise<Package>;
  /** Saves over the package as it was read (If-Match): a 409 means someone changed it meanwhile. */
  updatePackage: (pkg: Package, values: PackageFormValues, withCosts: boolean) => Promise<Package>;
  /** Sets the state explicitly (ADR-0036): a double click or a retry cannot flip it back. */
  setPackageActive: (id: string, isActive: boolean) => Promise<void>;
  deletePackage: (id: string) => Promise<void>;
  previewPackage: (body: PreviewRequest, signal?: AbortSignal) => Promise<PackagePreview>;
}

export const usePackageStore = create<PackageState>((set) => ({
  packagePage: [],
  packagePageMeta: EMPTY_PAGINATION_META,

  fetchPackagePage: async (params) => {
    const res = await httpClient.get(API.offers.packages, { params });
    set({
      packagePage: (res.data.data as RawPackage[]).map(toPackage),
      packagePageMeta: toPaginationMeta(res.data.meta as RawPaginationMeta),
    });
  },

  searchActivePackages: async (search) => {
    const params: Record<string, string | number | boolean> = { isActive: true, page: 1, limit: 20, sort: "name:asc" };
    if (search) params.search = search;
    const res = await httpClient.get(API.offers.packages, { params });
    return (res.data.data as RawPackage[]).map(toPackage);
  },

  fetchPackage: async (id) => {
    const res = await httpClient.get(API.offers.package(id));
    return toPackage(res.data.data as RawPackage);
  },

  createPackage: async (values, withCosts) => {
    try {
      const res = await httpClient.post(API.offers.packages, toRequestBody(values, withCosts));
      return toPackage(res.data.data as RawPackage);
    } catch (err) {
      throw rowKeys(err);
    }
  },

  updatePackage: async (pkg, values, withCosts) => {
    try {
      const res = await httpClient.patch(API.offers.package(pkg.id), toRequestBody(values, withCosts), { headers: ifMatch(pkg.updatedAt) });
      return toPackage(res.data.data as RawPackage);
    } catch (err) {
      throw rowKeys(err);
    }
  },

  setPackageActive: async (id, isActive) => {
    await httpClient.patch(API.offers.package(id), { isActive });
  },

  deletePackage: async (id) => {
    await httpClient.delete(API.offers.package(id));
  },

  previewPackage: async (body, signal) => {
    const res = await httpClient.post(API.offers.packagePreview, body, { signal });
    return res.data.data as PackagePreview;
  },
}));
