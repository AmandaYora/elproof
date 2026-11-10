import { create } from "zustand";
import { httpClient } from "@/shared/services/http-client";
import { API } from "@/shared/services/api-endpoints";
import { applyTabIdentity, resetTabIdentity } from "@/theme/tabIdentity";

interface RawBranding {
  businessName: string;
  hasLogo: boolean;
}

/** A tenant's identity inside its own session: its name and logo. There is no
 * per-tenant colour (ADR-0039): every console uses the one palette in theme.css. */
interface TenantBrandingState {
  logoUrl: string | null;
  /** The tenant's own business name — used as the text fallback in place of
   * "ElProof" wherever no logo is configured yet (Sidebar, ClientPortalLayout),
   * so an unbranded tenant still sees its own name, never the platform's. */
  businessName: string | null;
  /** False from the moment a fresh session mounts (AppLayout/ClientPortalLayout)
   * until `hydrate()` settles (match, 404, or failure) at least once. Layouts
   * hold off rendering the header and sidebar while this is false, so the
   * tenant's logo and name are painted once instead of swapping in. */
  hydrated: boolean;
  /** Fetches the caller's own tenant's branding (GET /tenants/me/branding,
   * self-service — open to any staff role or client, unlike GET /tenants/me
   * which stays Owner-only), loads the logo, and sets the browser tab
   * title/favicon. Call once per authenticated WO Console/Client Portal
   * session (fresh login or page reload) — never for a platform_admin
   * session, which has no tenant. `consoleLabel` (e.g. "WO Console", "Portal
   * Klien") is appended to the tab title so it still reads like a real page
   * title. Failures are swallowed: branding must never block the app. */
  hydrate: (consoleLabel?: string) => Promise<void>;
  /** Reverts the tab title and favicon and revokes the logo object URL —
   * called alongside every session teardown (logoutAndRedirect). */
  reset: () => void;
}

// Guards against a stale hydrate() call winning a race against a newer one
// (React StrictMode's double-invoked effects, or a fast logout->login-as-
// different-tenant sequence) — only the hydrate()/reset() call that's still
// the latest by the time its async work resolves is allowed to apply.
let hydrateToken = 0;

export const useTenantBrandingStore = create<TenantBrandingState>((set, get) => ({
  logoUrl: null,
  businessName: null,
  hydrated: false,

  hydrate: async (consoleLabel) => {
    const token = ++hydrateToken;
    try {
      const res = await httpClient.get(API.platform.tenantMeBranding);
      if (token !== hydrateToken) return; // superseded by a newer hydrate()/reset()
      const raw = res.data.data as RawBranding;

      let logoUrl: string | null = null;
      if (raw.hasLogo) {
        const fileRes = await httpClient.get(API.platform.tenantMeLogo, { responseType: "blob" });
        if (token !== hydrateToken) return; // superseded while the logo itself was loading
        logoUrl = URL.createObjectURL(fileRes.data as Blob);
      }

      applyTabIdentity(raw.businessName, logoUrl, consoleLabel);
      const previousLogoUrl = get().logoUrl;
      set({ logoUrl, businessName: raw.businessName, hydrated: true });
      if (previousLogoUrl) URL.revokeObjectURL(previousLogoUrl);
    } catch {
      // No tenant (e.g. platform_admin) or a transient failure: the header
      // falls back to plain text. Still unblocks whichever layout is waiting
      // on `hydrated` — a failure here must never hang the app on its
      // loading screen.
      if (token === hydrateToken) set({ hydrated: true });
    }
  },

  reset: () => {
    hydrateToken++; // invalidate any hydrate() still in flight
    resetTabIdentity();
    const previousLogoUrl = get().logoUrl;
    if (previousLogoUrl) URL.revokeObjectURL(previousLogoUrl);
    set({ logoUrl: null, businessName: null, hydrated: false });
  },
}));
