import axios, { AxiosError, type InternalAxiosRequestConfig } from "axios";
import { API } from "@/shared/services/api-endpoints";
import { useAuthStore } from "@/shared/stores/useAuthStore";

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
        useAuthStore.getState().logout();
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
      originalRequest?.url === API.auth.logout;

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
