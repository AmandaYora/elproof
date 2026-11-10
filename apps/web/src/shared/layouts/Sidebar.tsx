import { useNavigate } from "react-router-dom";
import { Briefcase, Building2, CalendarClock, FolderKanban, LayoutDashboard, LogOut, Settings, ShieldCheck, Sparkles, Store, Tags, UserCog, Users, BookUser } from "lucide-react";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import { Avatar } from "@/shared/components/ui/Avatar";
import { Button } from "@/shared/components/kit/button";
import { useAuthStore } from "@/shared/stores/useAuthStore";
import { useTenantBrandingStore } from "@/shared/stores/useTenantBrandingStore";
import { APP_NAME } from "@/shared/constants/brand";
import { logoutAndRedirect } from "@/shared/lib/auth-actions";
import { usePermissionStore, type Grants } from "@/shared/stores/usePermissionStore";
import { AppSidebar } from "@/shared/layouts/nav/AppSidebar";
import type { NavItem, NavSection } from "@/shared/layouts/nav/types";

interface WoItem extends NavItem {
  /** Permissions the role needs to see this item (undefined = everyone). */
  requires?: string[];
  /** Also needs a role that reaches every project (tenant-wide pages). */
  allProjects?: boolean;
}

interface WoSection extends Omit<NavSection, "items"> {
  items: WoItem[];
}

// The menu of FLOW-IDEAL §2.2 in the glossary's words. Items follow the workspace role's permissions (ADR-0026), the same
// ones the pages and their API calls require. "Keuangan" joins when the finance views exist (PLAN.md E13, backlog).
const SECTIONS: WoSection[] = [
  {
    id: "main",
    items: [
      { to: ROUTE_PATHS.dashboard, label: "Beranda", icon: LayoutDashboard },
      { to: ROUTE_PATHS.projects, label: "Proyek", icon: FolderKanban },
      { to: ROUTE_PATHS.clients, label: "Klien", icon: Users, requires: ["client.manage"], allProjects: true },
    ],
  },
  {
    id: "directory",
    collapsible: { label: "Direktori", icon: BookUser },
    items: [
      { to: ROUTE_PATHS.vendors, label: "Vendor", icon: Store, requires: ["vendor.view"] },
      { to: ROUTE_PATHS.venues, label: "Venue", icon: Building2, requires: ["vendor.view"] },
    ],
  },
  {
    id: "settings",
    collapsible: { label: "Pengaturan", icon: Settings },
    items: [
      { to: ROUTE_PATHS.businessProfile, label: "Profil usaha", icon: Briefcase, requires: ["business_profile.manage"] },
      { to: ROUTE_PATHS.users, label: "Tim", icon: UserCog, requires: ["team.manage"] },
      { to: ROUTE_PATHS.roles, label: "Peran dan akses", icon: ShieldCheck, requires: ["role.manage"] },
      { to: ROUTE_PATHS.subscription, label: "Langganan", icon: Sparkles, requires: ["subscription.manage"] },
      { to: ROUTE_PATHS.vendorCategories, label: "Kategori vendor", icon: Tags, requires: ["vendor.category.manage"] },
      { to: ROUTE_PATHS.milestoneTemplates, label: "Jadwal bawaan", icon: CalendarClock, requires: ["template.manage"] },
    ],
  },
];

function canSee(item: WoItem, grants: Grants | null): boolean {
  if (item.requires?.some((p) => !grants?.permissions.includes(p))) return false;
  return !item.allProjects || grants?.projectScope === "all";
}

/** The sections this role may see; a group with nothing left to show disappears entirely. */
export function visibleSections(grants: Grants | null): NavSection[] {
  return SECTIONS.map((s) => ({ ...s, items: s.items.filter((i) => canSee(i, grants)) })).filter((s) => s.items.length > 0);
}

/** The WO Console's menu. */
export function Sidebar() {
  const navigate = useNavigate();
  const session = useAuthStore((s) => s.session);
  const logoUrl = useTenantBrandingStore((s) => s.logoUrl);
  const brandName = useTenantBrandingStore((s) => s.businessName);
  const grants = usePermissionStore((s) => s.grants);
  const name = brandName ?? APP_NAME;

  return (
    <AppSidebar
      sections={visibleSections(grants)}
      brand={(collapsed) =>
        collapsed ? (
          <span aria-hidden className="flex size-9 items-center justify-center rounded-lg bg-primary text-[15px] font-bold text-primary-foreground">
            {name.trim().charAt(0).toUpperCase()}
          </span>
        ) : logoUrl ? (
          <span className="flex min-w-0 flex-col gap-0.5">
            <img src={logoUrl} alt={name} className="h-7 w-auto max-w-[160px] object-contain" />
            <span className="truncate text-[12px] font-medium text-muted-foreground">{name}</span>
          </span>
        ) : (
          <span className="truncate text-[15px] font-bold tracking-tight">{name}</span>
        )
      }
      footer={(collapsed) => (
        <div className={collapsed ? "flex flex-col items-center gap-2" : "flex items-center gap-2.5"}>
          <Avatar name={session?.displayName ?? "?"} size="md" />
          {!collapsed && (
            <div className="min-w-0 flex-1">
              <p className="truncate text-[13px] font-semibold">{session?.displayName ?? "Tidak diketahui"}</p>
              <p className="truncate text-[11.5px] text-muted-foreground">{grants?.roleName ?? session?.role ?? ""}</p>
            </div>
          )}
          <Button variant="ghost" size="icon-sm" onClick={() => void logoutAndRedirect(navigate)}>
            <LogOut aria-hidden />
            <span className="sr-only">Keluar</span>
          </Button>
        </div>
      )}
    />
  );
}
