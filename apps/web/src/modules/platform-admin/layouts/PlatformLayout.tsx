import { Suspense, useEffect, useState } from "react";
import { NavLink, Navigate, Outlet, useNavigate } from "react-router-dom";
import { LayoutDashboard, Building2, Package, Receipt, UserCog, ShieldCheck, LogOut, Wallet, Boxes } from "lucide-react";
import { cn } from "@/shared/lib/cn";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import { Avatar } from "@/shared/components/ui/Avatar";
import { useAuthStore } from "@/shared/stores/useAuthStore";
import { APP_NAME } from "@/shared/constants/brand";
import { getPlatformAdmin } from "@/modules/platform-admin/data";
import { usePlatformAdminStore } from "@/modules/platform-admin/stores/usePlatformAdminStore";
import { formatDate, todayISO } from "@/shared/lib/formatters";
import { logoutAndRedirect } from "@/shared/lib/auth-actions";

const NAV_ITEMS = [
  { to: ROUTE_PATHS.platformDashboard, label: "Dashboard", icon: LayoutDashboard },
  { to: ROUTE_PATHS.platformTenants, label: "Tenant", icon: Building2 },
  { to: ROUTE_PATHS.platformPlans, label: "Paket", icon: Package },
  { to: ROUTE_PATHS.platformTransactions, label: "Transaksi", icon: Receipt },
  { to: ROUTE_PATHS.platformUsers, label: "Pengguna", icon: UserCog },
  { to: ROUTE_PATHS.platformGatewayConfig, label: "Gateway Pembayaran", icon: Wallet },
  { to: ROUTE_PATHS.platformApps, label: "Manajemen Aplikasi", icon: Boxes },
];

export default function PlatformLayout() {
  const navigate = useNavigate();
  const currentPlatformAdminId = useAuthStore((s) => s.currentPlatformAdminId);
  const platformAdmins = usePlatformAdminStore((s) => s.platformAdmins);
  const fetchPlatformAdmins = usePlatformAdminStore((s) => s.fetchPlatformAdmins);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    fetchPlatformAdmins().finally(() => setIsLoading(false));
  }, [fetchPlatformAdmins]);

  const admin = getPlatformAdmin(platformAdmins, currentPlatformAdminId);

  if (isLoading) {
    return <div className="py-20 text-center text-sm text-text-secondary">Memuat...</div>;
  }
  if (!admin) {
    return <Navigate to={ROUTE_PATHS.login} replace />;
  }

  return (
    <div className="min-h-screen bg-background">
      <aside className="fixed inset-y-0 left-0 z-20 flex w-64 flex-col bg-navy-950 text-white">
        <div className="flex flex-col gap-0.5 px-5 py-6">
          <span className="text-[15px] font-bold tracking-tight">{APP_NAME}</span>
          <span className="flex items-center gap-1.5 text-[12px] font-medium text-warning">
            <ShieldCheck className="h-3.5 w-3.5" /> Platform Console
          </span>
        </div>

        <nav className="flex flex-1 flex-col gap-1 px-3">
          {NAV_ITEMS.map(({ to, label, icon: Icon }) => (
            <NavLink
              key={to}
              to={to}
              className={({ isActive }) =>
                cn(
                  "flex items-center gap-3 rounded-md px-3 py-2.5 text-[13.5px] font-medium transition-colors",
                  isActive ? "bg-navy-800 text-white" : "text-white/70 hover:bg-navy-800/60 hover:text-white"
                )
              }
            >
              <Icon className="h-[18px] w-[18px] shrink-0" />
              {label}
            </NavLink>
          ))}
        </nav>

        <div className="flex items-center gap-2.5 border-t border-white/10 px-5 py-4">
          <Avatar name={admin.name} size="md" className="bg-white/15 text-white" />
          <div className="min-w-0 flex-1">
            <p className="truncate text-[13px] font-semibold text-white">{admin.name}</p>
            <p className="truncate text-[11.5px] text-white/55">{admin.title}</p>
          </div>
          <button
            onClick={() => void logoutAndRedirect(navigate)}
            aria-label="Keluar"
            className="shrink-0 rounded-md p-1.5 text-white/55 hover:bg-navy-800 hover:text-white"
          >
            <LogOut className="h-4 w-4" />
          </button>
        </div>
      </aside>

      <div className="pl-64">
        <header className="sticky top-0 z-20 flex h-16 items-center justify-between gap-4 border-b border-border bg-surface px-6">
          <span className="text-[13.5px] font-semibold text-text-primary">{admin.role} ElProof</span>
          <span className="shrink-0 text-[13px] font-medium text-text-secondary">{formatDate(todayISO())}</span>
        </header>
        <main className="mx-auto max-w-[1400px] px-6 py-6">
          <Suspense fallback={<div className="py-20 text-center text-sm text-text-secondary">Memuat halaman...</div>}>
            <Outlet />
          </Suspense>
        </main>
      </div>
    </div>
  );
}
