import { Suspense, useEffect, useState } from "react";
import { Outlet } from "react-router-dom";
import { Loader2 } from "lucide-react";
import { Sidebar } from "@/shared/layouts/Sidebar";
import { Topbar } from "@/shared/layouts/Topbar";
import { useTenantBrandingStore } from "@/shared/stores/useTenantBrandingStore";

export function AppLayout() {
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const hydrateBranding = useTenantBrandingStore((s) => s.hydrate);
  const brandingHydrated = useTenantBrandingStore((s) => s.hydrated);

  useEffect(() => {
    void hydrateBranding("WO Console");
  }, [hydrateBranding]);

  if (!brandingHydrated) {
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
    <div className="min-h-screen bg-background">
      <Sidebar open={sidebarOpen} onClose={() => setSidebarOpen(false)} />
      <div className="lg:pl-64">
        <Topbar onMenuClick={() => setSidebarOpen(true)} />
        <main className="mx-auto max-w-[1400px] px-4 py-5 sm:px-6 sm:py-6">
          <Suspense fallback={<div className="py-20 text-center text-sm text-text-secondary">Memuat halaman...</div>}>
            <Outlet />
          </Suspense>
        </main>
      </div>
    </div>
  );
}
