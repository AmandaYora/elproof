import type { NavigateFunction } from "react-router-dom";
import { httpClient } from "@/shared/services/http-client";
import { API } from "@/shared/services/api-endpoints";
import { useAuthStore } from "@/shared/stores/useAuthStore";
import { ROUTE_PATHS } from "@/app/routes/route-paths";

// Shared by every console's logout button (WO Console, Client Portal, Platform
// Console) so the three don't drift: clear local session immediately for a
// snappy redirect, then best-effort revoke the refresh token server-side.
export async function logoutAndRedirect(navigate: NavigateFunction) {
  const refreshToken = useAuthStore.getState().session?.refreshToken;
  useAuthStore.getState().logout();
  navigate(ROUTE_PATHS.login);

  if (refreshToken) {
    try {
      await httpClient.post(API.auth.logout, { refreshToken });
    } catch {
      // Best-effort — local session is already cleared either way.
    }
  }
}
