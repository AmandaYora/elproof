import { create } from "zustand";
import { httpClient } from "@/shared/services/http-client";
import { API } from "@/shared/services/api-endpoints";

export type RoleKind = "staff" | "client";
export type ProjectScope = "all" | "assigned";

export interface Permission {
  key: string;
  kind: RoleKind;
  group: string;
  label: string;
  description: string;
}

export interface Role {
  id: number;
  kind: RoleKind;
  name: string;
  description: string;
  templateKey?: string;
  isSystem: boolean;
  projectScope: ProjectScope;
  permissions: string[];
  memberCount: number;
}

export interface RoleInput {
  kind: RoleKind;
  name: string;
  description: string;
  projectScope: ProjectScope;
  permissions: string[];
}

interface RolesState {
  catalog: Permission[];
  roles: Record<RoleKind, Role[]>;
  loaded: Record<RoleKind, boolean>;
  fetchCatalog: () => Promise<void>;
  fetchRoles: (kind: RoleKind) => Promise<void>;
  createRole: (input: RoleInput) => Promise<Role>;
  updateRole: (id: number, input: RoleInput) => Promise<Role>;
  deleteRole: (role: Role) => Promise<void>;
}

// Workspace roles (Pengaturan → Peran & Akses, ADR-0026). Also feeds the
// role pickers of the team and client forms.
export const useRolesStore = create<RolesState>((set, get) => ({
  catalog: [],
  roles: { staff: [], client: [] },
  loaded: { staff: false, client: false },
  fetchCatalog: async () => {
    if (get().catalog.length > 0) return;
    const res = await httpClient.get(API.access.catalog);
    set({ catalog: res.data.data as Permission[] });
  },
  fetchRoles: async (kind) => {
    const res = await httpClient.get(API.access.roles, { params: { kind } });
    set((s) => ({ roles: { ...s.roles, [kind]: res.data.data as Role[] }, loaded: { ...s.loaded, [kind]: true } }));
  },
  createRole: async (input) => {
    const res = await httpClient.post(API.access.roles, input);
    await get().fetchRoles(input.kind).catch(() => undefined);
    return res.data.data as Role;
  },
  updateRole: async (id, input) => {
    const res = await httpClient.patch(API.access.role(id), input);
    await get().fetchRoles(input.kind).catch(() => undefined);
    return res.data.data as Role;
  },
  deleteRole: async (role) => {
    await httpClient.delete(API.access.role(role.id));
    await get().fetchRoles(role.kind).catch(() => undefined);
  },
}));
