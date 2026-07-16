import { NavLink, useNavigate } from "react-router-dom";
import { LayoutDashboard, FolderKanban, Users, Tags, Store, UserCog, Sparkles, LogOut } from "lucide-react";
import { cn } from "@/shared/lib/cn";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import { Avatar } from "@/shared/components/ui/Avatar";
import { useAuthStore } from "@/shared/stores/useAuthStore";
import { APP_NAME } from "@/shared/constants/brand";
import { logoutAndRedirect } from "@/shared/lib/auth-actions";

const NAV_ITEMS = [
  { to: ROUTE_PATHS.dashboard, label: "Dashboard", icon: LayoutDashboard, ownerOnly: false },
  { to: ROUTE_PATHS.projects, label: "Project", icon: FolderKanban, ownerOnly: false },
  { to: ROUTE_PATHS.clients, label: "Client", icon: Users, ownerOnly: false },
  { to: ROUTE_PATHS.vendorCategories, label: "Kategori Vendor", icon: Tags, ownerOnly: false },
  { to: ROUTE_PATHS.vendors, label: "Vendor", icon: Store, ownerOnly: false },
  { to: ROUTE_PATHS.users, label: "Pengguna", icon: UserCog, ownerOnly: false },
  { to: ROUTE_PATHS.subscription, label: "Langganan", icon: Sparkles, ownerOnly: true },
];

export function Sidebar() {
  const navigate = useNavigate();
  const session = useAuthStore((s) => s.session);
  const isOwner = session?.role === "Owner";
  const visibleNavItems = NAV_ITEMS.filter((item) => !item.ownerOnly || isOwner);

  return (
    <aside className="fixed inset-y-0 left-0 z-20 flex w-64 flex-col bg-navy-950 text-white">
      <div className="flex flex-col gap-0.5 px-5 py-6">
        <span className="text-[15px] font-bold tracking-tight">{APP_NAME}</span>
        <span className="text-[12px] font-medium text-white/55">WO Console</span>
      </div>

      <nav className="flex flex-1 flex-col gap-1 px-3">
        {visibleNavItems.map(({ to, label, icon: Icon }) => (
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
        <Avatar name={session?.displayName ?? "?"} size="md" className="bg-white/15 text-white" />
        <div className="min-w-0 flex-1">
          <p className="truncate text-[13px] font-semibold text-white">{session?.displayName ?? "Tidak diketahui"}</p>
          <p className="truncate text-[11.5px] text-white/55">{session?.role ?? ""}</p>
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
  );
}
