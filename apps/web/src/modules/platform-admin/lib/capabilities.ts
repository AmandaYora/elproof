import { useAuthStore } from "@/shared/stores/useAuthStore";

/**
 * What each Platform Console role may do, mirroring the API's least-privilege
 * policy (cmd/server/platform_scope.go). The API is the real gate; this only
 * keeps buttons and menu entries a role cannot use out of its way.
 */
export type PlatformCapability =
  | "internal-read" // one workspace's back office: notes, members, data requests
  | "tenant-manage" // register/edit tenants, notes, data requests
  | "activate-subscription"
  | "support-access" // start a read-only support session
  | "plans"
  | "super-admin";

const GRANTS: Record<string, PlatformCapability[]> = {
  "Super Admin": ["internal-read", "tenant-manage", "activate-subscription", "support-access", "plans", "super-admin"],
  Billing: ["internal-read", "activate-subscription", "plans"],
  Support: ["internal-read", "tenant-manage", "support-access"],
  "Read-only": [],
};

export function roleCan(role: string | undefined, capability: PlatformCapability): boolean {
  return Boolean(role && GRANTS[role]?.includes(capability));
}

export function usePlatformCan(capability: PlatformCapability): boolean {
  return useAuthStore((s) => roleCan(s.session?.role, capability));
}
