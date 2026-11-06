import axios, { AxiosError, type InternalAxiosRequestConfig } from "axios";
import { API } from "@/shared/services/api-endpoints";
import { useAuthStore } from "@/shared/stores/useAuthStore";
import { useTenantBrandingStore } from "@/shared/stores/useTenantBrandingStore";
import { useSubscriptionStatusStore } from "@/shared/stores/useSubscriptionStatusStore";
import { usePermissionStore } from "@/shared/stores/usePermissionStore";
import { setLoginNotice } from "@/shared/lib/session-notice";
import { ROUTE_PATHS } from "@/app/routes/route-paths";

/** Codes the API attaches to a 403 when access depends on account state. */
type AccessErrorCode =
  | "subscription_read_only"
  | "subscription_suspended"
  | "tenant_frozen"
  | "mfa_required"
  | "forbidden_permission"
  | "role_missing";

function accessErrorCode(error: AxiosError): AccessErrorCode | undefined {
  if (error.response?.status !== 403) return undefined;
  const errors = (error.response.data as { errors?: { code?: string } } | undefined)?.errors;
  return errors?.code as AccessErrorCode | undefined;
}

/**
 * Reacts to account-state refusals in one place, so every page behaves the
 * same: refresh the subscription banner, or end the session with a reason.
 */
function handleAccessError(code: AccessErrorCode | undefined) {
  switch (code) {
    case "subscription_read_only":
    case "subscription_suspended":
      void useSubscriptionStatusStore.getState().fetch();
      break;
    case "tenant_frozen":
      setLoginNotice("Akun WO ini sedang dibekukan oleh ElProof. Hubungi tim ElProof untuk bantuan.");
      useAuthStore.getState().logout();
      useTenantBrandingStore.getState().reset();
      useSubscriptionStatusStore.getState().reset();
      window.location.assign(ROUTE_PATHS.login);
      break;
    case "forbidden_permission":
      // The role may have changed since the page loaded: refresh what the
      // UI offers so it stops showing actions the role no longer has.
      void usePermissionStore.getState().fetch();
      break;
    case "role_missing":
      setLoginNotice("Peran Anda di workspace ini sudah dihapus. Hubungi Owner WO Anda.");
      useAuthStore.getState().logout();
      usePermissionStore.getState().reset();
      window.location.assign(ROUTE_PATHS.login);
      break;
    case "mfa_required":
      if (window.location.pathname !== ROUTE_PATHS.platformSecurity) window.location.assign(ROUTE_PATHS.platformSecurity);
      break;
  }
}

export const httpClient = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL,
  timeout: 30000,
  headers: { "Content-Type": "application/json" },
});

httpClient.interceptors.request.use((config) => {
  const token = useAuthStore.getState().session?.accessToken;
  if (token) {
    config.headers.set("Authorization", `Bearer ${token}`);
  }
  return config;
});

// A bare instance (no interceptors) used only for the refresh call itself, so
// it never recurses into the 401 handler below.
const refreshClient = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL,
  timeout: 30000,
  headers: { "Content-Type": "application/json" },
});

let refreshInFlight: Promise<string | null> | null = null;

async function refreshAccessToken(): Promise<string | null> {
  const session = useAuthStore.getState().session;
  if (!session) return null;

  if (!refreshInFlight) {
    refreshInFlight = refreshClient
      .post(API.auth.refresh, { refreshToken: session.refreshToken })
      .then((res) => {
        const data = res.data.data as Omit<typeof session, never>;
        useAuthStore.getState().login({ ...session, ...data });
        return data.accessToken as string;
      })
      .catch(() => {
        setLoginNotice("Sesi Anda sudah berakhir. Silakan masuk kembali.");
        useAuthStore.getState().logout();
        useTenantBrandingStore.getState().reset();
        useSubscriptionStatusStore.getState().reset();
        return null;
      })
      .finally(() => {
        refreshInFlight = null;
      });
  }
  return refreshInFlight;
}

httpClient.interceptors.response.use(
  (response) => response,
  async (error: AxiosError) => {
    const originalRequest = error.config as (InternalAxiosRequestConfig & { _retried?: boolean }) | undefined;
    const isAuthEndpoint =
      originalRequest?.url === API.auth.login ||
      originalRequest?.url === API.auth.refresh ||
      originalRequest?.url === API.auth.logout ||
      originalRequest?.url === API.auth.mfaVerify ||
      originalRequest?.url === API.auth.google ||
      originalRequest?.url === API.auth.googleLink ||
      originalRequest?.url === API.public.signupWorkspace ||
      Boolean(originalRequest?.url?.startsWith("/api/v1/auth/links/"));

    handleAccessError(accessErrorCode(error));

    if (error.response?.status === 401 && originalRequest && !originalRequest._retried && !isAuthEndpoint) {
      originalRequest._retried = true;
      const newToken = await refreshAccessToken();
      if (newToken) {
        originalRequest.headers.set("Authorization", `Bearer ${newToken}`);
        return httpClient(originalRequest);
      }
    }
    return Promise.reject(error);
  }
);
