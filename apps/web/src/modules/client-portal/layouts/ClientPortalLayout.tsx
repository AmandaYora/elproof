import { Suspense, useEffect } from "react";
import { NavLink, Navigate, Outlet, useLocation, useNavigate } from "react-router-dom";
import { CalendarDays, Ellipsis, Heart, House, Loader2, LogOut, Store, Wallet, type LucideIcon } from "lucide-react";
import { Toaster } from "@/shared/components/ui/Toaster";
import { Button } from "@/shared/components/ui/Button";
import { cn } from "@/shared/lib/cn";
import { APP_NAME } from "@/shared/constants/brand";
import { ROUTE_PATHS, type ClientPortalTab } from "@/app/routes/route-paths";
import { logoutAndRedirect } from "@/shared/lib/auth-actions";
import { registerPortalServiceWorker } from "@/shared/lib/pwa";
import { whatsappUrl } from "@/shared/lib/access-link";
import { useTenantBrandingStore } from "@/shared/stores/useTenantBrandingStore";
import { useRouteTitle } from "@/shared/hooks/useRouteTitle";
import { hasPermission, usePermissionStore } from "@/shared/stores/usePermissionStore";
import { useContactStore } from "@/shared/stores/useContactStore";
import { useClientHomeStore } from "@/modules/client-portal/stores/useClientHomeStore";
import { OfflineBanner } from "@/modules/client-portal/components/OfflineBanner";

/** What every portal page gets: which project it is looking at. Each page loads its own data. */
export interface ClientPortalContext {
  projectId: string;
}

interface NavItem {
  tab: ClientPortalTab;
  label: string;
  icon: LucideIcon;
  /** Portal permission needed to see it; none = always. */
  requires?: string;
  /** Other pages that belong under this item (Lainnya holds Dokumen and Kendala). */
  alsoActiveOn?: ClientPortalTab[];
}

// Bottom bar on a phone, top bar on a larger screen (FLOW-IDEAL 3.2). Each item
// shows only when the client's portal role allows it (ADR-0026), e.g. Keluarga
// has no "Pembayaran".
const NAV: NavItem[] = [
  { tab: "beranda", label: "Beranda", icon: House },
  { tab: "timeline", label: "Jadwal", icon: CalendarDays, requires: "portal.timeline.view" },
  { tab: "pembayaran", label: "Pembayaran", icon: Wallet, requires: "portal.finance.view" },
  { tab: "vendor", label: "Vendor", icon: Store, requires: "portal.vendor.view" },
  { tab: "lainnya", label: "Lainnya", icon: Ellipsis, alsoActiveOn: ["dokumen", "kendala"] },
];

// Pages a role may not open (an old bookmark) land on Beranda.
const GUARDED: Partial<Record<ClientPortalTab, string>> = {
  timeline: "portal.timeline.view",
  pembayaran: "portal.finance.view",
  vendor: "portal.vendor.view",
  kendala: "portal.issues.view",
  dokumen: "portal.documents.view",
};

/** "Hubungi lewat WhatsApp" for the screens shown when there is no project to read a contact from. */
function HelpLink({ message, label }: { message: string; label: string }) {
  const contact = useContactStore((s) => s.contact);
  const fetchContact = useContactStore((s) => s.fetch);
  useEffect(() => {
    void fetchContact();
  }, [fetchContact]);
  if (!contact?.whatsapp) return null;
  return (
    <a href={whatsappUrl(contact.whatsapp, message)} target="_blank" rel="noopener noreferrer" className="font-semibold text-primary underline underline-offset-2">
      {label}
    </a>
  );
}

function FullScreenMessage({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div className="flex min-h-screen flex-col items-center justify-center gap-3 bg-card px-6 text-center">
      <h1 className="text-lg font-bold text-text-primary">{title}</h1>
      <div className="flex max-w-sm flex-col items-center gap-4 text-[14px] text-text-secondary">{children}</div>
    </div>
  );
}

export default function ClientPortalLayout() {
  const navigate = useNavigate();
  const location = useLocation();
  const home = useClientHomeStore((s) => s.home);
  const state = useClientHomeStore((s) => s.status);
  const fetchHome = useClientHomeStore((s) => s.fetch);
  const logoUrl = useTenantBrandingStore((s) => s.logoUrl);
  const brandName = useTenantBrandingStore((s) => s.businessName);
  useRouteTitle(brandName);
  const hydrateBranding = useTenantBrandingStore((s) => s.hydrate);
  const brandingHydrated = useTenantBrandingStore((s) => s.hydrated);
  const grants = usePermissionStore((s) => s.grants);
  const grantsLoaded = usePermissionStore((s) => s.loaded);
  const fetchGrants = usePermissionStore((s) => s.fetch);

  // The three calls the shell needs, in parallel: permissions, branding and the
  // landing page's data (which also tells every page which project this is).
  useEffect(() => {
    void fetchGrants();
    void hydrateBranding("Portal Klien");
    void fetchHome();
    registerPortalServiceWorker();
  }, [fetchGrants, hydrateBranding, fetchHome]);

  if (state === "no-project") {
    return (
      <FullScreenMessage title="Akun Anda belum terhubung ke persiapan pernikahan">
        <p>Hubungi wedding organizer Anda agar akun ini dihubungkan ke proyek Anda.</p>
        <HelpLink message="Halo, akun portal saya belum terhubung ke proyek. Mohon dibantu." label="Hubungi lewat WhatsApp" />
        <Button variant="secondary" onClick={() => void logoutAndRedirect(navigate)}>Keluar</Button>
      </FullScreenMessage>
    );
  }
  if (state === "error") {
    return (
      <FullScreenMessage title="Gagal memuat">
        <p>Portal belum bisa dimuat. Periksa koneksi internet Anda, lalu coba lagi.</p>
        <div className="flex flex-wrap items-center justify-center gap-3">
          <Button onClick={() => void fetchHome()}>Coba lagi</Button>
          <HelpLink message="Halo, portal saya tidak bisa dibuka. Mohon dibantu." label="Hubungi WO" />
        </div>
      </FullScreenMessage>
    );
  }
  if (state === "idle" || state === "loading" || !home || !brandingHydrated || !grantsLoaded) {
    // Waits on branding too: the header shows the WO's logo and name, and
    // painting before they land would swap them in visibly.
    return (
      <div className="flex min-h-screen items-center justify-center bg-card" role="status" aria-label="Memuat portal">
        <Loader2 className="h-6 w-6 animate-spin text-text-secondary" />
      </div>
    );
  }

  const items = NAV.filter((n) => !n.requires || hasPermission(grants, n.requires));
  const here = location.pathname.split("/")[2] as ClientPortalTab | undefined;
  const guard = here ? GUARDED[here] : undefined;
  if (guard && !hasPermission(grants, guard)) return <Navigate to={ROUTE_PATHS.portal("beranda")} replace />;

  const isActive = (item: NavItem) => here === item.tab || (here !== undefined && item.alsoActiveOn?.includes(here) === true);

  return (
    <div className="min-h-screen bg-background" data-density="comfortable">
      <Toaster mobileOffset={{ bottom: "calc(5rem + env(safe-area-inset-bottom))" }} />
      <OfflineBanner />

      <header className="sticky top-0 z-30 border-b border-border bg-card/90 backdrop-blur">
        <div className="mx-auto flex max-w-5xl items-center justify-between gap-4 px-4 py-2.5 sm:px-6">
          <NavLink to={ROUTE_PATHS.portal("beranda")} className="flex shrink-0 items-center gap-2 text-[14px] font-bold text-foreground">
            {logoUrl ? (
              <img src={logoUrl} alt={brandName ?? APP_NAME} className="h-8 w-auto max-w-[140px] object-contain" />
            ) : (
              <>
                <span className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary text-primary-foreground">
                  <Heart className="h-4 w-4" fill="currentColor" aria-hidden />
                </span>
                {brandName ?? APP_NAME}
              </>
            )}
          </NavLink>

          <nav aria-label="Menu portal" className="hidden items-center gap-1 md:flex">
            {items.map((item) => (
              <NavLink
                key={item.tab}
                to={ROUTE_PATHS.portal(item.tab)}
                className={cn(
                  "rounded-full px-3.5 py-2 text-[13.5px] font-medium transition-colors",
                  isActive(item) ? "bg-primary text-primary-foreground" : "text-text-secondary hover:bg-surface-muted hover:text-text-primary",
                )}
              >
                {item.label}
              </NavLink>
            ))}
          </nav>

          <button
            type="button"
            onClick={() => void logoutAndRedirect(navigate)}
            className="hidden items-center gap-1.5 rounded-full px-3 py-2 text-[13px] font-medium text-text-secondary hover:bg-danger-soft hover:text-danger-text focus-visible:outline focus-visible:outline-2 focus-visible:outline-ring md:flex"
          >
            <LogOut className="h-3.5 w-3.5" aria-hidden /> Keluar
          </button>
        </div>
      </header>

      <main className="mx-auto max-w-5xl px-4 py-5 pb-28 sm:px-6 sm:py-8 md:pb-10">
        <Suspense fallback={<div className="py-16 text-center text-sm text-text-secondary">Memuat...</div>}>
          <Outlet context={{ projectId: home.project.id } satisfies ClientPortalContext} />
        </Suspense>
      </main>

      <nav
        aria-label="Menu portal"
        className="fixed inset-x-0 bottom-0 z-30 border-t border-border bg-card/95 pb-[env(safe-area-inset-bottom)] backdrop-blur md:hidden"
      >
        <ul className="mx-auto grid max-w-md" style={{ gridTemplateColumns: `repeat(${items.length}, minmax(0, 1fr))` }}>
          {items.map((item) => {
            const active = isActive(item);
            return (
              <li key={item.tab}>
                <NavLink
                  to={ROUTE_PATHS.portal(item.tab)}
                  aria-current={active ? "page" : undefined}
                  className={cn(
                    "flex min-h-16 flex-col items-center justify-center gap-1 px-1 text-[11.5px] font-medium focus-visible:outline focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-ring",
                    active ? "text-primary" : "text-text-secondary",
                  )}
                >
                  <item.icon className="h-5 w-5" strokeWidth={active ? 2.4 : 1.8} aria-hidden />
                  {item.label}
                </NavLink>
              </li>
            );
          })}
        </ul>
      </nav>
    </div>
  );
}
