import type { NavigateFunction } from "react-router-dom";
import { httpClient } from "@/shared/services/http-client";
import { API } from "@/shared/services/api-endpoints";
import { useAuthStore, type AuthSession } from "@/shared/stores/useAuthStore";
import { useTenantBrandingStore } from "@/shared/stores/useTenantBrandingStore";
import { useSubscriptionStatusStore } from "@/shared/stores/useSubscriptionStatusStore";
import { usePermissionStore } from "@/shared/stores/usePermissionStore";
import { useContactStore } from "@/shared/stores/useContactStore";
import { runSessionResets } from "@/shared/lib/session-reset";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import { disableGoogleAutoSelect } from "@/shared/lib/google-identity";

/** Where a freshly signed-in principal lands. */
export function landingPathFor(session: AuthSession): string {
  // Staff land on the Dashboard; a role without it is sent on to Project
  // by the route guard (RequirePermission).
  if (session.principalType === "staff") return ROUTE_PATHS.dashboard;
  if (session.principalType === "client") return ROUTE_PATHS.portal();
  return session.mfa ? ROUTE_PATHS.platformDashboard : ROUTE_PATHS.platformSecurity;
}

// Shared by every console's logout button (WO Console, Client Portal, Platform
// Console) so the three don't drift: clear local session immediately for a
// snappy redirect, then best-effort revoke the sign-in server-side. The
// refresh token is an HttpOnly cookie (ADR-0044): the logout call carries it
// and the API removes it.
export async function logoutAndRedirect(navigate: NavigateFunction) {
  useAuthStore.getState().logout();
  useTenantBrandingStore.getState().reset();
  useSubscriptionStatusStore.getState().reset();
  usePermissionStore.getState().reset();
  useContactStore.getState().reset();
  runSessionResets();
  disableGoogleAutoSelect();
  navigate(ROUTE_PATHS.login);

  // Always sent, also from a support session: signing out of the browser
  // ends the admin's own sign-in too, as it did when the token was stored.
  try {
    await httpClient.post(API.auth.logout);
  } catch {
    // Best-effort — local session is already cleared either way.
  }
}
