import { create } from "zustand";
import { httpClient } from "@/shared/services/http-client";
import { API } from "@/shared/services/api-endpoints";
import type { StaffMember } from "@/modules/users/types";
import type { UserFormValues } from "@/modules/users/schemas/user.schema";
import { toPaginationMeta, EMPTY_PAGINATION_META, type PaginationMeta, type RawPaginationMeta } from "@/shared/types/pagination";

interface RawStaffMember {
  id: number;
  name: string;
  title: string;
  initials: string;
  role: StaffMember["role"];
  email: string;
  phone: string;
  isActive: boolean;
}

function toStaffMember(raw: RawStaffMember): StaffMember {
  return { ...raw, id: String(raw.id) };
}

interface StaffState {
  staff: StaffMember[];
  staffPage: StaffMember[];
  staffPageMeta: PaginationMeta;
  fetchStaff: () => Promise<void>;
  fetchStaffPage: (page: number, search: string, role: string) => Promise<void>;
  createStaff: (values: UserFormValues) => Promise<void>;
  updateStaff: (id: string, values: UserFormValues) => Promise<void>;
  toggleStaffActive: (id: string) => Promise<void>;
}

// Backed by the real `staff` module (Fase 3) — tenant-scoped, fetch-then-set
// (ADR-0009). Also the single source of truth for PIC pickers elsewhere
// (e.g. `projects`' ProjectFormModal).
export const useStaffStore = create<StaffState>((set, get) => ({
  staff: [],
  staffPage: [],
  staffPageMeta: EMPTY_PAGINATION_META,

  fetchStaff: async () => {
    const res = await httpClient.get(API.staff.base, { params: { all: true } });
    set({ staff: (res.data.data as RawStaffMember[]).map(toStaffMember) });
  },

  // Backs UserListPage's table — real server-side pagination + search/role
  // filtering, separate from the `staff` full-roster cache above (which PIC
  // pickers elsewhere still rely on).
  fetchStaffPage: async (page, search, role) => {
    const res = await httpClient.get(API.staff.base, { params: { page, search: search || undefined, role: role || undefined } });
    set({
      staffPage: (res.data.data as RawStaffMember[]).map(toStaffMember),
      staffPageMeta: toPaginationMeta(res.data.meta as RawPaginationMeta),
    });
  },

  createStaff: async (values) => {
    await httpClient.post(API.staff.base, values);
    await get().fetchStaff();
  },

  updateStaff: async (id, values) => {
    await httpClient.patch(API.staff.item(id), values);
    await get().fetchStaff();
  },

  toggleStaffActive: async (id) => {
    await httpClient.post(API.staff.toggleActive(id));
    await get().fetchStaff();
  },
}));
