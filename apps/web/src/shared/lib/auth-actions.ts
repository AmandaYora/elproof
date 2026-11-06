import type { NavigateFunction } from "react-router-dom";
import { httpClient } from "@/shared/services/http-client";
import { API } from "@/shared/services/api-endpoints";
import { useAuthStore, type AuthSession } from "@/shared/stores/useAuthStore";
import { useTenantBrandingStore } from "@/shared/stores/useTenantBrandingStore";
import { useSubscriptionStatusStore } from "@/shared/stores/useSubscriptionStatusStore";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import { disableGoogleAutoSelect } from "@/shared/lib/google-identity";

/** Where a freshly signed-in principal lands. */
export function landingPathFor(session: AuthSession): string {
  if (session.principalType === "staff") {
    // Neither Wedding Planner ("Staff") nor Sales has Dashboard access.
    return session.role === "Staff" || session.role === "Sales" ? ROUTE_PATHS.projects : ROUTE_PATHS.dashboard;
  }
  if (session.principalType === "client") return ROUTE_PATHS.portal();
  return session.mfa ? ROUTE_PATHS.platformDashboard : ROUTE_PATHS.platformSecurity;
}

// Shared by every console's logout button (WO Console, Client Portal, Platform
// Console) so the three don't drift: clear local session immediately for a
// snappy redirect, then best-effort revoke the refresh token server-side.
export async function logoutAndRedirect(navigate: NavigateFunction) {
  const refreshToken = useAuthStore.getState().session?.refreshToken;
  useAuthStore.getState().logout();
  useTenantBrandingStore.getState().reset();
  useSubscriptionStatusStore.getState().reset();
  disableGoogleAutoSelect();
  navigate(ROUTE_PATHS.login);

  if (refreshToken) {
    try {
      await httpClient.post(API.auth.logout, { refreshToken });
    } catch {
      // Best-effort — local session is already cleared either way.
    }
  }
}
