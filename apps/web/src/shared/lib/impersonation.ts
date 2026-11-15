import { httpClient } from "@/shared/services/http-client";
import { API } from "@/shared/services/api-endpoints";
import { clearSupportMarker, handOffSupportSession, takeInterruptedSupportSession, useAuthStore } from "@/shared/stores/useAuthStore";
import { useTenantBrandingStore } from "@/shared/stores/useTenantBrandingStore";
import { useSubscriptionStatusStore } from "@/shared/stores/useSubscriptionStatusStore";
import { usePermissionStore } from "@/shared/stores/usePermissionStore";
import { useContactStore } from "@/shared/stores/useContactStore";
import { runSessionResets } from "@/shared/lib/session-reset";
import { toast } from "@/shared/stores/useToastStore";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import type { ImpersonationResult } from "@/modules/platform-admin/data/console.types";

function resetWorkspaceStores() {
  useTenantBrandingStore.getState().reset();
  useSubscriptionStatusStore.getState().reset();
  usePermissionStore.getState().reset();
  useContactStore.getState().reset();
  runSessionResets();
}

/**
 * Switches this tab into the workspace as its Owner, read-only. A full page
 * load follows, so no admin state leaks in; the support token crosses it in
 * this tab's sessionStorage and is deleted as soon as the new page reads it
 * (tokens are kept in memory only, ADR-0044). The admin's own refresh token
 * stays in its HttpOnly cookie, out of reach of the tenant-written pages.
 */
export function enterSupportSession(result: ImpersonationResult, tenantName: string) {
  const current = useAuthStore.getState().session;
  if (!current || current.principalType !== "platform_admin") return;
  handOffSupportSession({
    accessToken: result.accessToken,
    principalType: "staff",
    principalId: "",
    tenantId: result.tenantId,
    role: result.role,
    displayName: "Owner (sesi dukungan)",
    impersonation: { sessionId: result.sessionId, tenantName, expiresAt: result.expiresAt, returnTo: { ...current, accessToken: "" } },
  });
  resetWorkspaceStores();
  window.location.assign(ROUTE_PATHS.dashboard);
}

async function tellServerSupportEnded(sessionId: number) {
  try {
    await httpClient.post(API.console.supportSessionEnd(sessionId));
  } catch {
    // The session expires on its own within the hour; the admin is already back.
  }
}

/** Ends the support session: back to the admin's session, then tell the server. */
export async function endSupportSession() {
  const session = useAuthStore.getState().session;
  const info = session?.impersonation;
  if (!info) return;
  const tenantId = session?.tenantId;
  useAuthStore.getState().login(info.returnTo);
  clearSupportMarker();
  resetWorkspaceStores();
  await tellServerSupportEnded(info.sessionId);
  window.location.assign(tenantId ? ROUTE_PATHS.platformTenantDetail(String(tenantId)) : ROUTE_PATHS.platformTenants);
}

/**
 * The support token lives only in memory, so reloading the tab during a
 * support session leaves the admin's own session (the store already moved
 * the address to the workspace's page). End that support session on the
 * server too and say why. Called once at start-up.
 */
export function finishInterruptedSupportSession() {
  const interrupted = takeInterruptedSupportSession();
  if (!interrupted || useAuthStore.getState().session?.principalType !== "platform_admin") return;
  toast.info("Sesi dukungan berakhir karena halaman dimuat ulang. Mulai sesi baru bila masih diperlukan.");
  void tellServerSupportEnded(interrupted.sessionId);
}
