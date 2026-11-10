import { create } from "zustand";
import { httpClient } from "@/shared/services/http-client";
import { API } from "@/shared/services/api-endpoints";
import { useAuthStore } from "@/shared/stores/useAuthStore";
import { useSubscriptionStatusStore, type SubscriptionLifecycle } from "@/shared/stores/useSubscriptionStatusStore";

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

/**
 * Whether these grants cover a staff role: the same permissions or fewer, and the every-project scope only when held.
 * The API's rule (Grants.Covers, ADR-0026) for handing out, editing or deleting a role and for managing its members, so
 * team.manage / role.manage never become a way to gain access. The Owner covers every role.
 */
export function coversRole(grants: Grants | null, role: { permissions: string[]; projectScope: "all" | "assigned" }): boolean {
  if (!grants) return false;
  if (grants.isOwner) return true;
  if (role.projectScope === "all" && grants.projectScope !== "all") return false;
  return role.permissions.every((p) => grants.permissions.includes(p));
}

/** True while the workspace may only be read (grace period over, unpaid or frozen). */
export function isWriteLocked(lifecycle: SubscriptionLifecycle | null): boolean {
  return lifecycle !== null && lifecycle.access !== "full";
}

/** What stays usable in read-only mode besides every ".view" permission: paying. */
const ALWAYS_ALLOWED = new Set(["subscription.manage"]);

/** Permissions that only read data; they survive read-only mode. */
function isReadPermission(permission: string): boolean {
  return permission.endsWith(".view") || ALWAYS_ALLOWED.has(permission);
}

/** True while the workspace may only be read; use for write buttons with no permission of their own. */
export function useWriteLocked(): boolean {
  const locked = useSubscriptionStatusStore((s) => isWriteLocked(s.lifecycle));
  // A support session only looks: nothing may be changed, whatever the plan.
  const supportSession = useAuthStore((s) => s.session?.impersonation !== undefined);
  return locked || supportSession;
}

/**
 * Whether the signed-in person's role grants a permission, for buttons and
 * other in-page controls. In read-only mode every permission that changes
 * data reads as "no", so write buttons disappear (the API refuses them
 * anyway with `subscription_read_only`). Routes and the sidebar use
 * `hasPermission` directly, so pages stay reachable and readable.
 */
export function useCan(permission: string): boolean {
  const granted = usePermissionStore((s) => hasPermission(s.grants, permission));
  const locked = useSubscriptionStatusStore((s) => isWriteLocked(s.lifecycle));
  const supportSession = useAuthStore((s) => s.session?.impersonation !== undefined);
  if (supportSession) return granted && permission.endsWith(".view");
  return granted && !(locked && !isReadPermission(permission));
}

/** Whether their project permissions reach every project. */
export function useAllProjects(): boolean {
  return usePermissionStore((s) => s.grants?.projectScope === "all");
}
