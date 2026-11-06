import { create } from "zustand";
import { httpClient } from "@/shared/services/http-client";
import { API } from "@/shared/services/api-endpoints";
import type { StaffMember, StaffSummary } from "@/modules/users/types";
import type { UserFormValues } from "@/modules/users/schemas/user.schema";
import type { AccessLink } from "@/shared/lib/access-link";
import { toPaginationMeta, EMPTY_PAGINATION_META, type PaginationMeta, type RawPaginationMeta } from "@/shared/types/pagination";
import { toAffectedProjects, type AffectedProject, type RawDeleteImpact } from "@/shared/types/delete-impact";

interface RawStaffMember {
  id: number;
  name: string;
  title: string;
  initials: string;
  role: string;
  roleId: number;
  username: string;
  email: string;
  phone: string;
  isActive: boolean;
  loginStatus?: StaffMember["loginStatus"];
}

function toStaffMember(raw: RawStaffMember): StaffMember {
  return { ...raw, id: String(raw.id) };
}

interface RawStaffSummary {
  id: number;
  name: string;
  title: string;
}

function toStaffSummary(raw: RawStaffSummary): StaffSummary {
  return { id: String(raw.id), name: raw.name, title: raw.title };
}

export interface CreateStaffResult {
  staff: StaffMember;
  invitation: AccessLink;
}

interface RawInvitation {
  token: string;
  expiresAt: string;
}

interface StaffState {
  staff: StaffMember[];
  staffPage: StaffMember[];
  staffPageMeta: PaginationMeta;
  staffSummaries: StaffSummary[];
  fetchStaff: () => Promise<void>;
  /** role: a role id ("" = every role). */
  fetchStaffPage: (page: number, search: string, role: string) => Promise<void>;
  // Public-safe {id, name, title} list (any staff role) — what every PIC
  // picker/label across the `projects` module should use; `staff` above is
  // Owner-only now (Pengguna management, see staff_handler.go's
  // requireOwnerTenant).
  fetchStaffSummaries: () => Promise<void>;
  /** Invites a member; returns the one-time invitation link to send. */
  createStaff: (values: UserFormValues) => Promise<CreateStaffResult>;
  /** New invitation link (not accepted yet, or lost Google account). */
  reinviteStaff: (id: string) => Promise<AccessLink>;
  updateStaff: (id: string, values: UserFormValues) => Promise<void>;
  toggleStaffActive: (id: string) => Promise<void>;
  // Hard delete (PLAN.md) — informed-consent, not a blocking precondition.
  // The tenant's Owner account is rejected server-side unconditionally
  // (403), never reaching a confirmation dialog.
  fetchStaffDeleteImpact: (id: string) => Promise<AffectedProject[]>;
  deleteStaff: (id: string) => Promise<void>;
}

// Backed by the real `staff` module (Fase 3) — tenant-scoped, fetch-then-set
// (ADR-0009).
export const useStaffStore = create<StaffState>((set, get) => ({
  staff: [],
  staffPage: [],
  staffPageMeta: EMPTY_PAGINATION_META,
  staffSummaries: [],

  fetchStaff: async () => {
    const res = await httpClient.get(API.staff.base, { params: { all: true } });
    set({ staff: (res.data.data as RawStaffMember[]).map(toStaffMember) });
  },

  fetchStaffSummaries: async () => {
    const res = await httpClient.get(API.staff.summary);
    set({ staffSummaries: (res.data.data as RawStaffSummary[]).map(toStaffSummary) });
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
    const res = await httpClient.post(API.staff.base, values);
    const data = res.data.data as { staff: RawStaffMember; invitation: RawInvitation };
    // The one-time link must reach the screen even if the list refresh fails.
    await get().fetchStaff().catch(() => undefined);
    return { staff: toStaffMember(data.staff), invitation: { ...data.invitation, purpose: "staff_invite" } };
  },

  reinviteStaff: async (id) => {
    const res = await httpClient.post(API.staff.invite(id));
    return { ...(res.data.data as RawInvitation), purpose: "staff_invite" };
  },

  updateStaff: async (id, values) => {
    await httpClient.patch(API.staff.item(id), values);
    await get().fetchStaff();
  },

  toggleStaffActive: async (id) => {
    await httpClient.post(API.staff.toggleActive(id));
    await get().fetchStaff();
  },

  fetchStaffDeleteImpact: async (id) => {
    const res = await httpClient.get(API.staff.deleteImpact(id));
    return toAffectedProjects(res.data.data as RawDeleteImpact);
  },

  deleteStaff: async (id) => {
    await httpClient.delete(API.staff.item(id));
    await get().fetchStaff();
  },
}));
