import { Toaster } from "@/shared/components/ui/Toaster";
import { Suspense, useEffect, useState } from "react";
import { Navigate, Outlet, useNavigate } from "react-router-dom";
import { LayoutDashboard, Building2, Package, Receipt, UserCog, ShieldCheck, LogOut, Wallet, Boxes, Menu, Megaphone, FileLock2, ScrollText, SlidersHorizontal, ImageIcon } from "lucide-react";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import { Avatar } from "@/shared/components/ui/Avatar";
import { Button } from "@/shared/components/kit/button";
import { AppSidebar } from "@/shared/layouts/nav/AppSidebar";
import { SidebarProvider, useSidebar } from "@/shared/layouts/nav/SidebarContext";
import { useRouteTitle } from "@/shared/hooks/useRouteTitle";
import { CommandPalette, CommandTrigger } from "@/shared/layouts/CommandPalette";
import type { NavSection } from "@/shared/layouts/nav/types";
import { useAuthStore } from "@/shared/stores/useAuthStore";
import { APP_NAME } from "@/shared/constants/brand";
import { getPlatformAdmin } from "@/modules/platform-admin/data";
import { usePlatformAdminStore } from "@/modules/platform-admin/stores/usePlatformAdminStore";
import { formatDate, todayISO } from "@/shared/lib/formatters";
import { logoutAndRedirect } from "@/shared/lib/auth-actions";
import { roleCan, type PlatformCapability } from "@/modules/platform-admin/lib/capabilities";

interface NavItem {
  to: string;
  label: string;
  icon: typeof LayoutDashboard;
  /** Hidden from roles without it (the API refuses them anyway). */
  needs?: PlatformCapability;
}

const NAV_GROUPS: { label: string | null; items: NavItem[] }[] = [
  {
    label: null,
    items: [{ to: ROUTE_PATHS.platformDashboard, label: "Ringkasan", icon: LayoutDashboard }],
  },
  {
    label: "Pelanggan",
    items: [
      { to: ROUTE_PATHS.platformTenants, label: "Tenant", icon: Building2 },
      { to: ROUTE_PATHS.platformTransactions, label: "Tagihan", icon: Receipt },
      { to: ROUTE_PATHS.platformAnnouncements, label: "Pengumuman", icon: Megaphone },
      { to: ROUTE_PATHS.platformDataRequests, label: "Data & Privasi", icon: FileLock2, needs: "internal-read" },
    ],
  },
  {
    label: "Produk",
    items: [
      { to: ROUTE_PATHS.platformPlans, label: "Paket", icon: Package },
      { to: ROUTE_PATHS.platformPolicy, label: "Kebijakan Langganan", icon: SlidersHorizontal },
    ],
  },
  {
    label: "Sistem",
    items: [
      { to: ROUTE_PATHS.platformUsers, label: "Pengguna", icon: UserCog },
      { to: ROUTE_PATHS.platformAudit, label: "Audit Log", icon: ScrollText, needs: "super-admin" },
      { to: ROUTE_PATHS.platformLoginSlides, label: "Tampilan Login", icon: ImageIcon, needs: "super-admin" },
      { to: ROUTE_PATHS.platformGatewayConfig, label: "Gateway Pembayaran", icon: Wallet },
      { to: ROUTE_PATHS.platformApps, label: "Manajemen Aplikasi", icon: Boxes },
    ],
  },
];

export default function PlatformLayout() {
  const session = useAuthStore((s) => s.session);
  // The API refuses every console call without a two-factor session.
  if (session?.principalType === "platform_admin" && !session.mfa) {
    return <Navigate to={ROUTE_PATHS.platformSecurity} replace />;
  }
  return <PlatformShell />;
}

function PlatformShell() {
  useRouteTitle("ElProof Platform");
  const navigate = useNavigate();
  const currentPlatformAdminId = useAuthStore((s) => s.currentPlatformAdminId);
  const platformAdmins = usePlatformAdminStore((s) => s.platformAdmins);
  const fetchPlatformAdmin = usePlatformAdminStore((s) => s.fetchPlatformAdmin);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    fetchPlatformAdmin(currentPlatformAdminId)
      .catch(() => undefined)
      .finally(() => setIsLoading(false));
  }, [fetchPlatformAdmin, currentPlatformAdminId]);

  const admin = getPlatformAdmin(platformAdmins, currentPlatformAdminId);

  if (isLoading) {
    return <div className="py-20 text-center text-sm text-text-secondary">Memuat...</div>;
  }
  if (!admin) {
    return <Navigate to={ROUTE_PATHS.login} replace />;
  }

  // The sections this admin may see, in the shared menu's shape.
  const sections: NavSection[] = NAV_GROUPS.map((group, i) => ({
    id: group.label ?? `main-${i}`,
    label: group.label ?? undefined,
    items: group.items.filter((item) => !item.needs || roleCan(admin.role, item.needs)).map(({ to, label, icon }) => ({ to, label, icon })),
  })).filter((section) => section.items.length > 0);

  return (
    <SidebarProvider>
      <div className="min-h-screen bg-background" data-density="compact">
        <Toaster />
        <AppSidebar
          sections={sections}
          brand={(collapsed) =>
            collapsed ? (
              <ShieldCheck aria-hidden className="size-6 text-warning-text" />
            ) : (
              <span className="flex flex-col gap-0.5">
                <span className="text-[15px] font-bold tracking-tight">{APP_NAME}</span>
                <span className="flex items-center gap-1.5 text-[12px] font-medium text-warning-text">
                  <ShieldCheck aria-hidden className="size-3.5" /> Platform Console
                </span>
              </span>
            )
          }
          footer={(collapsed) => (
            <div className={collapsed ? "flex flex-col items-center gap-2" : "flex items-center gap-2.5"}>
              <Avatar name={admin.name} size="md" />
              {!collapsed && (
                <div className="min-w-0 flex-1">
                  <p className="truncate text-[13px] font-semibold">{admin.name}</p>
                  <p className="truncate text-[11.5px] text-muted-foreground">{admin.title}</p>
                </div>
              )}
              <Button variant="ghost" size="icon-sm" onClick={() => void logoutAndRedirect(navigate)}>
                <LogOut aria-hidden />
                <span className="sr-only">Keluar</span>
              </Button>
            </div>
          )}
        />

        <div className="transition-[padding] duration-200 lg:pl-(--sidebar-w)">
          <PlatformTopbar pages={sections.flatMap((s) => s.items)} title={`${admin.role} ElProof`} />
          <main className="mx-auto max-w-[1400px] px-4 py-5 sm:px-6 sm:py-6">
            <Suspense fallback={<div className="py-20 text-center text-sm text-text-secondary">Memuat halaman...</div>}>
              <Outlet />
            </Suspense>
          </main>
        </div>
      </div>
    </SidebarProvider>
  );
}

function PlatformTopbar({ pages, title }: { pages: NavSection["items"]; title: string }) {
  const { toggle } = useSidebar();
  return (
    <header className="sticky top-0 z-20 flex h-16 items-center justify-between gap-3 border-b bg-card px-4 sm:px-6">
      <Button variant="ghost" size="icon" onClick={toggle} className="lg:hidden">
        <Menu aria-hidden />
        <span className="sr-only">Buka menu</span>
      </Button>
      <span className="hidden min-w-0 truncate text-[13.5px] font-semibold text-foreground sm:block">{title}</span>
      <div className="ml-auto w-full max-w-xs sm:max-w-sm">
        <CommandPalette pages={pages} searchRecords={false} trigger={(open) => <CommandTrigger onOpen={open} placeholder="Cari halaman..." />} />
      </div>
      <span className="hidden shrink-0 text-[13px] font-medium text-muted-foreground md:block">{formatDate(todayISO())}</span>
    </header>
  );
}
