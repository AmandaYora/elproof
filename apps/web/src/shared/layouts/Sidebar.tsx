import { useEffect, useState } from "react";
import { NavLink, useLocation, useNavigate } from "react-router-dom";
import type { LucideIcon } from "lucide-react";
import { LayoutDashboard, FolderKanban, Users, Tags, Store, Building2, UserCog, Sparkles, Settings, CalendarClock, ChevronDown, LogOut, X } from "lucide-react";
import { cn } from "@/shared/lib/cn";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import { Avatar } from "@/shared/components/ui/Avatar";
import { useAuthStore } from "@/shared/stores/useAuthStore";
import { useTenantBrandingStore } from "@/shared/stores/useTenantBrandingStore";
import { APP_NAME } from "@/shared/constants/brand";
import { logoutAndRedirect } from "@/shared/lib/auth-actions";
import { ROLE_LABELS, type StaffRole } from "@/modules/users/types";

interface NavLinkItem {
  kind: "link";
  to: string;
  label: string;
  icon: LucideIcon;
  // undefined = every role; otherwise only the listed roles see this item.
  allowedRoles?: StaffRole[];
}

interface NavGroupItem {
  kind: "group";
  label: string;
  icon: LucideIcon;
  children: NavLinkItem[];
}

type NavEntry = NavLinkItem | NavGroupItem;

// "Pengaturan" groups the 3 lower-frequency admin pages behind one collapsible
// entry instead of each sitting flat in the list. Role matrix (confirmed):
// Owner sees everything; Admin sees everything except Pengaturan entirely;
// Wedding Planner ("Staff") sees only Project.
const NAV_ITEMS: NavEntry[] = [
  { kind: "link", to: ROUTE_PATHS.dashboard, label: "Dashboard", icon: LayoutDashboard, allowedRoles: ["Owner", "Admin"] },
  { kind: "link", to: ROUTE_PATHS.projects, label: "Project", icon: FolderKanban },
  { kind: "link", to: ROUTE_PATHS.clients, label: "Client", icon: Users, allowedRoles: ["Owner", "Admin"] },
  { kind: "link", to: ROUTE_PATHS.vendors, label: "Vendor", icon: Store, allowedRoles: ["Owner", "Admin"] },
  { kind: "link", to: ROUTE_PATHS.venues, label: "Venue", icon: Building2, allowedRoles: ["Owner", "Admin"] },
  {
    kind: "group",
    label: "Pengaturan",
    icon: Settings,
    children: [
      { kind: "link", to: ROUTE_PATHS.users, label: "Pengguna", icon: UserCog, allowedRoles: ["Owner"] },
      { kind: "link", to: ROUTE_PATHS.subscription, label: "Langganan", icon: Sparkles, allowedRoles: ["Owner"] },
      { kind: "link", to: ROUTE_PATHS.vendorCategories, label: "Kategori Vendor", icon: Tags, allowedRoles: ["Owner"] },
      { kind: "link", to: ROUTE_PATHS.milestoneTemplates, label: "Timeline Default", icon: CalendarClock, allowedRoles: ["Owner"] },
    ],
  },
];

interface SidebarProps {
  open: boolean;
  onClose: () => void;
}

export function Sidebar({ open, onClose }: SidebarProps) {
  const navigate = useNavigate();
  const location = useLocation();
  const session = useAuthStore((s) => s.session);
  const logoUrl = useTenantBrandingStore((s) => s.logoUrl);
  const brandName = useTenantBrandingStore((s) => s.businessName);
  const role = session?.role as StaffRole | undefined;
  const canSee = (allowedRoles: StaffRole[] | undefined) => !allowedRoles || (role !== undefined && allowedRoles.includes(role));

  // Flat items filtered by their own allowedRoles; a group is filtered by
  // its children's allowedRoles and dropped entirely if nothing remains
  // visible (e.g. Pengaturan vanishes for Admin/Wedding Planner, since every
  // one of its children is Owner-only).
  const visibleEntries: NavEntry[] = NAV_ITEMS.flatMap((entry): NavEntry[] => {
    if (entry.kind === "link") {
      return canSee(entry.allowedRoles) ? [entry] : [];
    }
    const children = entry.children.filter((child) => canSee(child.allowedRoles));
    return children.length > 0 ? [{ ...entry, children }] : [];
  });

  // "Pengaturan" is collapsed by default and only ever auto-opens (never
  // auto-closes) when the current route lands on one of its children -- e.g.
  // a direct link/reload to /langganan should still show its nav highlighted,
  // not hidden behind a collapsed group. Sidebar is a persistent element
  // (see AppLayout.tsx), so this state survives in-app navigation elsewhere.
  const [settingsOpen, setSettingsOpen] = useState(false);
  useEffect(() => {
    const settingsGroup = NAV_ITEMS.find((entry): entry is NavGroupItem => entry.kind === "group");
    if (settingsGroup?.children.some((child) => location.pathname.startsWith(child.to))) {
      setSettingsOpen(true);
    }
  }, [location.pathname]);

  return (
    <>
      {open && (
        <div
          className="fixed inset-0 z-30 bg-navy-950/50 lg:hidden"
          onClick={onClose}
          aria-hidden="true"
        />
      )}
      <aside
        className={cn(
          "fixed inset-y-0 left-0 z-40 flex w-64 flex-col bg-navy-950 text-white transition-transform duration-200 ease-in-out lg:translate-x-0",
          open ? "translate-x-0" : "-translate-x-full"
        )}
      >
        <div className="flex items-center justify-between gap-2 px-5 py-6">
          <div className="flex min-w-0 flex-col gap-0.5">
            {logoUrl ? (
              <>
                <img src={logoUrl} alt={brandName ?? APP_NAME} className="h-7 w-auto max-w-[160px] object-contain" />
                <span className="truncate text-[12px] font-medium text-white/55">{brandName ?? APP_NAME}</span>
              </>
            ) : (
              <span className="truncate text-[15px] font-bold tracking-tight">{brandName ?? APP_NAME}</span>
            )}
          </div>
          <button
            onClick={onClose}
            aria-label="Tutup menu"
            className="shrink-0 rounded-md p-1.5 text-white/55 hover:bg-navy-800 hover:text-white lg:hidden"
          >
            <X className="h-4.5 w-4.5" />
          </button>
        </div>

        <nav className="flex flex-1 flex-col gap-1 px-3">
          {visibleEntries.map((entry) =>
            entry.kind === "link" ? (
              <NavLink
                key={entry.to}
                to={entry.to}
                onClick={onClose}
                className={({ isActive }) =>
                  cn(
                    "flex items-center gap-3 rounded-md px-3 py-2.5 text-[13.5px] font-medium transition-colors",
                    isActive ? "bg-navy-800 text-white" : "text-white/70 hover:bg-navy-800/60 hover:text-white"
                  )
                }
              >
                <entry.icon className="h-[18px] w-[18px] shrink-0" />
                {entry.label}
              </NavLink>
            ) : (
              <div key={entry.label}>
                <button
                  type="button"
                  onClick={() => setSettingsOpen((v) => !v)}
                  aria-expanded={settingsOpen}
                  className="flex w-full items-center gap-3 rounded-md px-3 py-2.5 text-[13.5px] font-medium text-white/70 transition-colors hover:bg-navy-800/60 hover:text-white"
                >
                  <entry.icon className="h-[18px] w-[18px] shrink-0" />
                  <span className="flex-1 text-left">{entry.label}</span>
                  <ChevronDown className={cn("h-4 w-4 shrink-0 transition-transform", settingsOpen && "rotate-180")} />
                </button>
                {settingsOpen && (
                  <div className="mt-1 flex flex-col gap-1">
                    {entry.children.map((child) => (
                      <NavLink
                        key={child.to}
                        to={child.to}
                        onClick={onClose}
                        className={({ isActive }) =>
                          cn(
                            "flex items-center gap-3 rounded-md py-2.5 pl-10 pr-3 text-[13.5px] font-medium transition-colors",
                            isActive ? "bg-navy-800 text-white" : "text-white/70 hover:bg-navy-800/60 hover:text-white"
                          )
                        }
                      >
                        <child.icon className="h-[16px] w-[16px] shrink-0" />
                        {child.label}
                      </NavLink>
                    ))}
                  </div>
                )}
              </div>
            )
          )}
        </nav>

        <div className="flex items-center gap-2.5 border-t border-white/10 px-5 py-4">
          <Avatar name={session?.displayName ?? "?"} size="md" className="bg-white/15 text-white" />
          <div className="min-w-0 flex-1">
            <p className="truncate text-[13px] font-semibold text-white">{session?.displayName ?? "Tidak diketahui"}</p>
            <p className="truncate text-[11.5px] text-white/55">
              {session?.role ? (ROLE_LABELS[session.role as StaffRole] ?? session.role) : ""}
            </p>
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
    </>
  );
}
