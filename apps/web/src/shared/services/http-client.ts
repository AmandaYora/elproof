import axios, { AxiosError, type InternalAxiosRequestConfig } from "axios";
import { API } from "@/shared/services/api-endpoints";
import { useAuthStore, type AuthSession } from "@/shared/stores/useAuthStore";
import { useTenantBrandingStore } from "@/shared/stores/useTenantBrandingStore";
import { useSubscriptionStatusStore } from "@/shared/stores/useSubscriptionStatusStore";
import { usePermissionStore } from "@/shared/stores/usePermissionStore";
import { setLoginNotice } from "@/shared/lib/session-notice";
import { toast } from "@/shared/stores/useToastStore";
import { useContactStore } from "@/shared/stores/useContactStore";
import { useConflictStore } from "@/shared/stores/useConflictStore";
import { isStaleConflict } from "@/shared/lib/concurrency";
import { ROUTE_PATHS } from "@/app/routes/route-paths";

declare module "axios" {
  interface AxiosRequestConfig {
    /**
     * The screen that made this save shows its own "changed by someone else" state (the rincian editor keeps the
     * person's edits and offers the newer version), so the global conflict dialog stays closed.
     */
    ownConflict?: boolean;
  }
}

/** Codes the API attaches to a 403 when access depends on account state. */
type AccessErrorCode =
  | "subscription_read_only"
  | "subscription_suspended"
  | "tenant_frozen"
  | "mfa_required"
  | "impersonation_read_only"
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
    case "impersonation_read_only":
      toast.error("Sesi dukungan hanya untuk melihat. Akhiri sesi untuk mengubah data.");
      break;
    case "mfa_required":
      if (window.location.pathname !== ROUTE_PATHS.platformSecurity) window.location.assign(ROUTE_PATHS.platformSecurity);
      break;
  }
}

// The refresh token is an HttpOnly cookie scoped to /api/v1/auth (ADR-0044):
// `withCredentials` lets the browser store and send it when the API is on
// another origin (local development); X-Requested-With is the header the
// API demands on the endpoints that read that cookie (CSRF).
const clientDefaults = {
  baseURL: import.meta.env.VITE_API_BASE_URL,
  timeout: 30000,
  withCredentials: true,
  headers: { "Content-Type": "application/json", "X-Requested-With": "XMLHttpRequest" },
};

export const httpClient = axios.create(clientDefaults);

/** Endpoints that sign in or out: they never carry or wait for an access token. */
function isAuthEndpoint(url: string | undefined): boolean {
  return (
    url === API.auth.login ||
    url === API.auth.refresh ||
    url === API.auth.logout ||
    url === API.auth.mfaVerify ||
    url === API.auth.google ||
    url === API.auth.googleLink ||
    url === API.public.signupWorkspace ||
    Boolean(url?.startsWith("/api/v1/auth/links/"))
  );
}

httpClient.interceptors.request.use(async (config) => {
  const session = useAuthStore.getState().session;
  let token = session?.accessToken;
  // After a page load the access token is not known yet (it is never
  // stored): get one from the refresh cookie before the first call that
  // needs it (public endpoints, e.g. the homepage's price list, do not).
  if (session && !token && !isAuthEndpoint(config.url) && !config.url?.startsWith("/api/v1/public/")) {
    token = (await refreshAccessToken()) ?? undefined;
  }
  if (token) {
    config.headers.set("Authorization", `Bearer ${token}`);
  }
  return config;
});

// A bare instance (no interceptors) used only for the refresh call itself, so
// it never recurses into the 401 handler below.
const refreshClient = axios.create(clientDefaults);

let refreshInFlight: Promise<string | null> | null = null;

/** A support session has no refresh token: when it lapses, go back to the admin's own session. */
let leavingSupportSession = false;

function leaveExpiredSupportSession(returnTo: NonNullable<ReturnType<typeof useAuthStore.getState>["session"]>) {
  // Several requests can hit the 401 at once; only the first one leaves.
  if (leavingSupportSession) return;
  leavingSupportSession = true;
  useAuthStore.getState().login(returnTo);
  useTenantBrandingStore.getState().reset();
  useSubscriptionStatusStore.getState().reset();
  usePermissionStore.getState().reset();
  useContactStore.getState().reset();
  window.location.assign(ROUTE_PATHS.platformTenants);
}

async function refreshAccessToken(): Promise<string | null> {
  const session = useAuthStore.getState().session;
  if (!session) return null;
  if (leavingSupportSession) return null;
  if (session.impersonation) {
    leaveExpiredSupportSession(session.impersonation.returnTo);
    return null;
  }

  if (!refreshInFlight) {
    // No body: the refresh token travels only in the HttpOnly cookie. Two
    // tabs refreshing at once is fine (the API accepts a token rotated a
    // moment ago), so no cross-tab lock is needed.
    refreshInFlight = refreshClient
      .post(API.auth.refresh)
      .then((res) => {
        const data = res.data.data as AuthSession;
        useAuthStore.getState().login({ ...session, ...data });
        return data.accessToken;
      })
      .catch((error: unknown) => {
        // No answer or a server fault: keep the session and let this call
        // fail, the next one tries again. Only a refusal ends the session.
        const status = axios.isAxiosError(error) ? error.response?.status : undefined;
        if (status === undefined || status >= 500) return null;
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

// A write under /projects/{id}/... that the server accepted. Stores that show a project listen, so a header or a
// side panel is never left showing numbers from before the change (PLAN.md E8).
type ProjectWriteListener = (projectId: string, data: unknown) => void;
const projectWriteListeners = new Set<ProjectWriteListener>();

/** Registers a listener for accepted writes under a project; call once at module load. */
export function onProjectWrite(listener: ProjectWriteListener): void {
  projectWriteListeners.add(listener);
}

const PROJECT_PATH = /^\/api\/v1\/projects\/(\d+)(?:\/|$)/;

httpClient.interceptors.response.use(
  (response) => {
    const method = response.config.method?.toLowerCase();
    if (method && method !== "get") {
      const path = (response.config.url ?? "").split("?")[0];
      const match = PROJECT_PATH.exec(path);
      // A preview computes and writes nothing: nothing on screen needs reloading.
      if (match && !path.endsWith("/preview")) {
        const data = (response.data as { data?: unknown } | undefined)?.data;
        projectWriteListeners.forEach((listen) => listen(match[1], data));
      }
    }
    return response;
  },
  async (error: AxiosError) => {
    const originalRequest = error.config as (InternalAxiosRequestConfig & { _retried?: boolean }) | undefined;

    handleAccessError(accessErrorCode(error));
    // Saved over someone else's change: say so once, here, whichever form made the save (ADR-0036).
    if (isStaleConflict(error) && !error.config?.ownConflict) useConflictStore.getState().show();

    if (error.response?.status === 401 && originalRequest && !originalRequest._retried && !isAuthEndpoint(originalRequest.url)) {
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
