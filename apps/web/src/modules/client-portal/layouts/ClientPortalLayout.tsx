import { Suspense, useEffect } from "react";
import { NavLink, Navigate, Outlet, useLocation, useNavigate } from "react-router-dom";
import { CalendarDays, ConciergeBell, Ellipsis, House, Loader2, LogOut, MessageCircle, Wallet, type LucideIcon } from "lucide-react";
import { AppLogoMark } from "@/shared/components/AppLogoMark";
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
import { askMessage, usePortalContact } from "@/modules/client-portal/lib/usePortalContact";

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
  { tab: "layanan", label: "Layanan", icon: ConciergeBell, requires: "portal.vendor.view" },
  { tab: "lainnya", label: "Lainnya", icon: Ellipsis, alsoActiveOn: ["dokumen", "kendala"] },
];

// Pages a role may not open (an old bookmark) land on Beranda.
const GUARDED: Partial<Record<ClientPortalTab, string>> = {
  timeline: "portal.timeline.view",
  pembayaran: "portal.finance.view",
  layanan: "portal.vendor.view",
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

/** A count on a menu item (open kendala on Lainnya). Navy, never red: a kendala is being handled, not an alarm. */
function NavCount({ count, inverted = false, className }: { count: number; inverted?: boolean; className?: string }) {
  if (count <= 0) return null;
  return (
    <span
      className={cn(
        "inline-flex h-[18px] min-w-[18px] items-center justify-center rounded-full px-1 text-[11px] leading-none font-semibold tabular-nums ring-2 ring-card",
        inverted ? "bg-primary-foreground text-primary ring-transparent" : "bg-primary text-primary-foreground",
        className,
      )}
    >
      {count}
      <span className="sr-only"> kendala sedang ditangani</span>
    </span>
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
  const contact = usePortalContact();

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
  // Open kendala live under Lainnya: its menu item carries their count (RISET #7).
  const openIssues = hasPermission(grants, "portal.issues.view") ? (home.openIssueCount ?? 0) : 0;
  const badgeFor = (item: NavItem) => (item.tab === "lainnya" ? openIssues : 0);

  return (
    <div className="min-h-screen bg-background" data-density="comfortable">
      <Toaster mobileOffset={{ bottom: "calc(5rem + env(safe-area-inset-bottom))" }} />
      <OfflineBanner />

      <header className="sticky top-0 z-30 border-b border-border bg-card/90 backdrop-blur">
        <div className="mx-auto flex max-w-6xl items-center justify-between gap-3 px-4 py-2 sm:px-6">
          <NavLink
            to={ROUTE_PATHS.portal("beranda")}
            className="flex min-w-0 items-center gap-2.5 rounded-md text-[15px] font-semibold text-foreground focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-solid focus-visible:outline-ring"
          >
            {logoUrl ? (
              // The WO's own logo carries its name; "Didukung ElProof" sits on Lainnya.
              <img src={logoUrl} alt={brandName ?? APP_NAME} className="h-9 w-auto max-w-[150px] object-contain" />
            ) : (
              <>
                <AppLogoMark className="size-9 rounded-lg" />
                <span className="truncate">{brandName ?? APP_NAME}</span>
              </>
            )}
          </NavLink>

          <nav aria-label="Menu portal" className="hidden items-center gap-1 md:flex">
            {items.map((item) => (
              <NavLink
                key={item.tab}
                to={ROUTE_PATHS.portal(item.tab)}
                className={cn(
                  "inline-flex items-center gap-1.5 rounded-full px-3.5 py-2 text-[14px] font-medium transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-solid focus-visible:outline-ring",
                  isActive(item) ? "bg-primary text-primary-foreground" : "text-text-secondary hover:bg-surface-muted hover:text-text-primary",
                )}
              >
                {item.label}
                <NavCount count={badgeFor(item)} inverted={isActive(item)} />
              </NavLink>
            ))}
          </nav>

          <div className="flex shrink-0 items-center gap-1">
            {contact.phone && (
              <a
                href={whatsappUrl(contact.phone, askMessage(contact))}
                target="_blank"
                rel="noopener noreferrer"
                aria-label={`Hubungi ${contact.name} lewat WhatsApp`}
                title={`Hubungi ${contact.name} lewat WhatsApp`}
                className="flex size-11 items-center justify-center rounded-full text-whatsapp hover:bg-muted focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-solid focus-visible:outline-ring"
              >
                <MessageCircle className="size-5" aria-hidden />
              </a>
            )}
            <button
              type="button"
              onClick={() => void logoutAndRedirect(navigate)}
              className="hidden items-center gap-1.5 rounded-full px-3 py-2 text-[13px] font-medium text-text-secondary hover:bg-danger-soft hover:text-danger-text focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-solid focus-visible:outline-ring md:flex"
            >
              <LogOut className="h-3.5 w-3.5" aria-hidden /> Keluar
            </button>
          </div>
        </div>
      </header>

      <main className="mx-auto max-w-6xl px-4 py-5 pb-28 sm:px-6 sm:py-8 md:pb-10">
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
                    "relative flex min-h-16 flex-col items-center justify-center gap-1 px-1 text-[11.5px] font-medium focus-visible:outline focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-ring",
                    active ? "text-primary" : "text-text-secondary",
                  )}
                >
                  <item.icon className="h-5 w-5" strokeWidth={active ? 2.4 : 1.8} aria-hidden />
                  {item.label}
                  {/* After the label so it is read "Lainnya, 1 kendala…"; drawn on the icon's corner. */}
                  <NavCount count={badgeFor(item)} className="absolute top-2 left-1/2 ml-1" />
                </NavLink>
              </li>
            );
          })}
        </ul>
      </nav>
    </div>
  );
}
