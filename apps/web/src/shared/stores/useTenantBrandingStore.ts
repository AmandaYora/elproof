import { create } from "zustand";
import { httpClient } from "@/shared/services/http-client";
import { API } from "@/shared/services/api-endpoints";
import {
  applyBrandColorPreset,
  resetBrandColorPreset,
  isBrandColorPresetKey,
  type BrandColorPresetKey,
} from "@/theme/brandPresets";
import { applyTabIdentity, resetTabIdentity } from "@/theme/tabIdentity";

interface RawBranding {
  businessName: string;
  brandColorPreset: string;
  hasLogo: boolean;
}

interface TenantBrandingState {
  colorPreset: BrandColorPresetKey;
  logoUrl: string | null;
  /** The tenant's own business name — used as the text fallback in place of
   * "ElProof" wherever no logo is configured yet (Sidebar, ClientPortalLayout),
   * so an unbranded tenant still sees its own name, never the platform's. */
  businessName: string | null;
  /** Fetches the caller's own tenant's branding (GET /tenants/me/branding,
   * self-service — open to any staff role or client, unlike GET /tenants/me
   * which stays Owner-only) and applies it by overriding the app's brand CSS
   * variables at runtime, plus the browser tab title/favicon. Call once per
   * authenticated WO Console/Client Portal session (fresh login or page
   * reload) — never for a platform_admin session, which has no tenant to
   * brand. `consoleLabel` (e.g. "WO Console", "Portal Klien") is appended to
   * the tab title so it still reads like a real page title, not just a bare
   * company name. Failures are swallowed: branding is a visual nicety, not
   * something that should block the app. */
  hydrate: (consoleLabel?: string) => Promise<void>;
  /** Reverts to the app's default (navy) look, tab title, and favicon, and
   * revokes the logo object URL — called alongside every session teardown
   * (logoutAndRedirect). */
  reset: () => void;
}

// Guards against a stale hydrate() call winning a race against a newer one
// (React StrictMode's double-invoked effects, or a fast logout->login-as-
// different-tenant sequence) — only the hydrate()/reset() call that's still
// the latest by the time its async work resolves is allowed to apply.
let hydrateToken = 0;

export const useTenantBrandingStore = create<TenantBrandingState>((set, get) => ({
  colorPreset: "navy",
  logoUrl: null,
  businessName: null,

  hydrate: async (consoleLabel) => {
    const token = ++hydrateToken;
    try {
      const res = await httpClient.get(API.platform.tenantMeBranding);
      if (token !== hydrateToken) return; // superseded by a newer hydrate()/reset()
      const raw = res.data.data as RawBranding;
      const colorPreset = isBrandColorPresetKey(raw.brandColorPreset) ? raw.brandColorPreset : "navy";

      let logoUrl: string | null = null;
      if (raw.hasLogo) {
        const fileRes = await httpClient.get(API.platform.tenantMeLogo, { responseType: "blob" });
        if (token !== hydrateToken) return; // superseded while the logo itself was loading
        logoUrl = URL.createObjectURL(fileRes.data as Blob);
      }

      applyBrandColorPreset(colorPreset);
      applyTabIdentity(raw.businessName, logoUrl, consoleLabel);
      const previousLogoUrl = get().logoUrl;
      set({ colorPreset, logoUrl, businessName: raw.businessName });
      if (previousLogoUrl) URL.revokeObjectURL(previousLogoUrl);
    } catch {
      // No tenant to brand (e.g. platform_admin) or a transient failure —
      // the app keeps rendering with its default navy look.
    }
  },

  reset: () => {
    hydrateToken++; // invalidate any hydrate() still in flight
    resetBrandColorPreset();
    resetTabIdentity();
    const previousLogoUrl = get().logoUrl;
    if (previousLogoUrl) URL.revokeObjectURL(previousLogoUrl);
    set({ colorPreset: "navy", logoUrl: null, businessName: null });
  },
}));
