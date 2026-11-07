import { Suspense, useEffect, useState } from "react";
import { Outlet, useLocation } from "react-router-dom";
import { Loader2 } from "lucide-react";
import { Sidebar } from "@/shared/layouts/Sidebar";
import { Topbar } from "@/shared/layouts/Topbar";
import { useAuthStore } from "@/shared/stores/useAuthStore";
import { useTenantBrandingStore } from "@/shared/stores/useTenantBrandingStore";
import { useSubscriptionStatusStore } from "@/shared/stores/useSubscriptionStatusStore";
import { usePermissionStore, hasPermission } from "@/shared/stores/usePermissionStore";
import { SubscriptionBanner, SubscriptionLockedScreen } from "@/shared/layouts/SubscriptionBanner";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import { Toaster } from "@/shared/components/ui/Toaster";
import { SupportSessionBanner } from "@/shared/layouts/SupportSessionBanner";
import { AnnouncementBanner } from "@/shared/layouts/AnnouncementBanner";

export function AppLayout() {
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const hydrateBranding = useTenantBrandingStore((s) => s.hydrate);
  const brandingHydrated = useTenantBrandingStore((s) => s.hydrated);
  const lifecycle = useSubscriptionStatusStore((s) => s.lifecycle);
  const fetchLifecycle = useSubscriptionStatusStore((s) => s.fetch);
  const fetchGrants = usePermissionStore((s) => s.fetch);
  const grantsLoaded = usePermissionStore((s) => s.loaded);
  const supportSession = useAuthStore((s) => s.session?.impersonation !== undefined);
  // A support session only looks, so it offers no way to pay.
  const canManageSubscription = usePermissionStore((s) => hasPermission(s.grants, "subscription.manage")) && !supportSession;
  const { pathname } = useLocation();

  useEffect(() => {
    void hydrateBranding("WO Console");
    void fetchGrants();
  }, [hydrateBranding, fetchGrants]);

  // A role can be edited while someone is signed in (ADR-0026): refresh
  // what they may do when they come back to the tab, so new menu entries
  // show up and removed ones disappear without a reload.
  useEffect(() => {
    const onVisible = () => {
      if (document.visibilityState === "visible") void fetchGrants();
    };
    document.addEventListener("visibilitychange", onVisible);
    return () => document.removeEventListener("visibilitychange", onVisible);
  }, [fetchGrants]);

  // Subscription state for the banner: on entry and whenever the tab
  // regains focus (a payment may have completed in the meantime).
  useEffect(() => {
    void fetchLifecycle();
    const onFocus = () => void fetchLifecycle();
    window.addEventListener("focus", onFocus);
    return () => window.removeEventListener("focus", onFocus);
  }, [fetchLifecycle]);

  const locked = lifecycle?.access === "billing_only" && pathname !== ROUTE_PATHS.subscription;

  if (!brandingHydrated || !grantsLoaded) {
    // Sidebar and every `variant="primary"` Button read their color live off
    // the `--brand-navy-*` CSS vars hydrate() overwrites — painting them
    // before that resolves shows ElProof's own navy default first, then a
    // visible swap to the tenant's real color once it lands. A plain white
    // screen carries no color of its own, so there's nothing left to flash
    // away from once the real chrome mounts (see useTenantBrandingStore).
    return (
      <div className="flex min-h-screen items-center justify-center bg-white">
        <Loader2 className="h-6 w-6 animate-spin text-text-secondary" />
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-background" data-density="compact">
      <Toaster />
      <SupportSessionBanner />
      <Sidebar open={sidebarOpen} onClose={() => setSidebarOpen(false)} />
      <div className="lg:pl-64 print:pl-0">
        <Topbar onMenuClick={() => setSidebarOpen(true)} />
        <main className="mx-auto max-w-[1400px] px-4 py-5 sm:px-6 sm:py-6">
          <AnnouncementBanner />
          {lifecycle && !locked && <SubscriptionBanner lifecycle={lifecycle} canManage={canManageSubscription} />}
          {locked ? (
            <SubscriptionLockedScreen canManage={canManageSubscription} />
          ) : (
            <Suspense fallback={<div className="py-20 text-center text-sm text-text-secondary">Memuat halaman...</div>}>
              <Outlet />
            </Suspense>
          )}
        </main>
      </div>
    </div>
  );
}
