import { httpClient } from "@/shared/services/http-client";
import { API } from "@/shared/services/api-endpoints";
import { useAuthStore } from "@/shared/stores/useAuthStore";
import { useTenantBrandingStore } from "@/shared/stores/useTenantBrandingStore";
import { useSubscriptionStatusStore } from "@/shared/stores/useSubscriptionStatusStore";
import { usePermissionStore } from "@/shared/stores/usePermissionStore";
import { useContactStore } from "@/shared/stores/useContactStore";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import type { ImpersonationResult } from "@/modules/platform-admin/data/console.types";

function resetWorkspaceStores() {
  useTenantBrandingStore.getState().reset();
  useSubscriptionStatusStore.getState().reset();
  usePermissionStore.getState().reset();
  useContactStore.getState().reset();
}

/**
 * Switches this browser into the workspace as its Owner, read-only. The
 * admin's own session is kept inside the new one and restored by
 * endSupportSession. A full page load follows, so no admin state leaks in.
 */
export function enterSupportSession(result: ImpersonationResult, tenantName: string) {
  const current = useAuthStore.getState().session;
  if (!current || current.principalType !== "platform_admin") return;
  useAuthStore.getState().login({
    accessToken: result.accessToken,
    refreshToken: "",
    principalType: "staff",
    principalId: "",
    tenantId: result.tenantId,
    role: result.role,
    displayName: "Owner (sesi dukungan)",
    // The admin's refresh token is deliberately not carried into the WO
    // Console: that page renders tenant-written content, and a script in it
    // must not be able to read a long-lived admin credential. Coming back, the
    // admin has the access token for as long as it lives, then signs in again.
    impersonation: { sessionId: result.sessionId, tenantName, expiresAt: result.expiresAt, returnTo: { ...current, refreshToken: "" } },
  });
  resetWorkspaceStores();
  window.location.assign(ROUTE_PATHS.dashboard);
}

/** Ends the support session: back to the admin's session, then tell the server. */
export async function endSupportSession() {
  const session = useAuthStore.getState().session;
  const info = session?.impersonation;
  if (!info) return;
  const tenantId = session?.tenantId;
  useAuthStore.getState().login(info.returnTo);
  resetWorkspaceStores();
  try {
    await httpClient.post(API.console.supportSessionEnd(info.sessionId));
  } catch {
    // The session expires on its own within the hour; the admin is already back.
  }
  window.location.assign(tenantId ? ROUTE_PATHS.platformTenantDetail(String(tenantId)) : ROUTE_PATHS.platformTenants);
}
