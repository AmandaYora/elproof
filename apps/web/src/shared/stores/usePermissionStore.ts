import { create } from "zustand";
import { httpClient } from "@/shared/services/http-client";
import { API } from "@/shared/services/api-endpoints";

/** GET /access/me — what the signed-in staff member or client may do (ADR-0026). */
export interface Grants {
  roleId: number;
  roleName: string;
  isOwner: boolean;
  /** "all": every project; "assigned": only projects where they are PIC / PIC Sales. */
  projectScope: "all" | "assigned";
  permissions: string[];
}

interface PermissionState {
  grants: Grants | null;
  /** True once the first load settled (success or failure). */
  loaded: boolean;
  fetch: () => Promise<void>;
  reset: () => void;
}

export const usePermissionStore = create<PermissionState>((set) => ({
  grants: null,
  loaded: false,
  fetch: async () => {
    try {
      const res = await httpClient.get(API.access.me);
      set({ grants: res.data.data as Grants, loaded: true });
    } catch {
      // No grants: the UI shows only what needs no permission; the API
      // refuses everything else anyway.
      set({ grants: null, loaded: true });
    }
  },
  reset: () => set({ grants: null, loaded: false }),
}));

/** Pure check, usable outside React. */
export function hasPermission(grants: Grants | null, permission: string): boolean {
  return Boolean(grants?.permissions.includes(permission));
}

/** Whether the signed-in person's role grants a permission. */
export function useCan(permission: string): boolean {
  return usePermissionStore((s) => hasPermission(s.grants, permission));
}

/** Whether their project permissions reach every project. */
export function useAllProjects(): boolean {
  return usePermissionStore((s) => s.grants?.projectScope === "all");
}
