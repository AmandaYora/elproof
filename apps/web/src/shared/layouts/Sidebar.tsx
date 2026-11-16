import { lazy, Suspense, useState } from "react";
import { Link } from "react-router-dom";
import {
  BookUser, Briefcase, Building2, CalendarClock, CalendarDays, ChevronsUpDown, FolderKanban, LayoutDashboard, LayoutTemplate,
  Settings, ShieldCheck, Sparkles, Store, Tags, UserCog, Users, Wallet,
} from "lucide-react";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import { Avatar } from "@/shared/components/ui/Avatar";
import { useAuthStore } from "@/shared/stores/useAuthStore";
import { useTenantBrandingStore } from "@/shared/stores/useTenantBrandingStore";
import { useSubscriptionStatusStore } from "@/shared/stores/useSubscriptionStatusStore";
import { APP_NAME, SUPPORT_WHATSAPP } from "@/shared/constants/brand";
import { cn } from "@/shared/lib/cn";
import { usePermissionStore, hasPermission, type Grants } from "@/shared/stores/usePermissionStore";
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

const HELP_WHATSAPP = `https://wa.me/${SUPPORT_WHATSAPP}?text=${encodeURIComponent("Halo ElProof, saya butuh bantuan di WO Console.")}`;

// The menu (RISET WO Console §4): daily work first (Beranda, Proyek, Kalender, Keuangan, Klien), then the directories,
// the templates a new project is built from, and the workspace settings. Items follow the role's permissions
// (ADR-0026), the same ones the pages and their API calls require.
const SECTIONS: WoSection[] = [
  {
    id: "main",
    items: [
      { to: ROUTE_PATHS.dashboard, label: "Beranda", icon: LayoutDashboard },
      { to: ROUTE_PATHS.projects, label: "Proyek", icon: FolderKanban },
      { to: ROUTE_PATHS.calendar, label: "Kalender", icon: CalendarDays },
      { to: ROUTE_PATHS.finance, label: "Keuangan", icon: Wallet, requires: ["finance.record"] },
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
    id: "templates",
    collapsible: { label: "Templat", icon: LayoutTemplate },
    items: [
      { to: ROUTE_PATHS.milestoneTemplates, label: "Jadwal bawaan", icon: CalendarClock, requires: ["template.manage"] },
      { to: ROUTE_PATHS.vendorCategories, label: "Kategori vendor", icon: Tags, requires: ["vendor.category.manage"] },
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

/** The WO's own mark: its logo, or its initial on a navy square. */
function WorkspaceMark({ name, logoUrl, size = "md" }: { name: string; logoUrl: string | null; size?: "md" | "sm" }) {
  const box = size === "md" ? "size-9 rounded-lg text-[15px]" : "size-8 rounded-md text-[14px]";
  if (logoUrl) {
    return (
      <span className={cn("grid shrink-0 place-items-center overflow-hidden border border-border bg-card", box)}>
        <img src={logoUrl} alt="" className="size-full object-contain p-0.5" />
      </span>
    );
  }
  return (
    <span aria-hidden className={cn("grid shrink-0 place-items-center bg-primary font-bold text-primary-foreground", box)}>
      {name.trim().charAt(0).toUpperCase()}
    </span>
  );
}

/**
 * The trial in one quiet line above the user (instead of a banner on every page): days left and the way to a plan.
 * Shown only while the trial is calm; its last days and every other subscription state stay a banner (AppLayout).
 */
function TrialChip({ collapsed }: { collapsed: boolean }) {
  const lifecycle = useSubscriptionStatusStore((s) => s.lifecycle);
  const canManage = usePermissionStore((s) => hasPermission(s.grants, "subscription.manage"));
  if (!lifecycle || lifecycle.status !== "trialing" || lifecycle.daysLeft <= 3 || collapsed) return null;
  const days = Math.max(lifecycle.daysLeft, 0);
  return (
    <div className="mx-2.5 mb-2 rounded-lg border border-border bg-card px-3 py-2.5 text-[12.5px]">
      <p className="font-semibold text-foreground">Masa coba: {days} hari lagi</p>
      {canManage ? (
        <Link to={ROUTE_PATHS.subscription} className="mt-0.5 inline-block font-medium text-muted-foreground underline-offset-2 hover:text-foreground hover:underline">
          Pilih paket langganan
        </Link>
      ) : (
        <p className="mt-0.5 text-muted-foreground">Owner bisa memilih paket di menu Langganan.</p>
      )}
    </div>
  );
}

/** What the user button shows: avatar, name and role (the same before and after the menu code has loaded). */
function UserButtonFace({ name, role, collapsed }: { name: string; role: string; collapsed: boolean }) {
  return (
    <>
      <Avatar name={name} size="md" />
      {!collapsed && (
        <>
          <span className="min-w-0 flex-1">
            <span className="block truncate text-[13px] font-semibold">{name}</span>
            <span className="block truncate text-[12px] text-muted-foreground">{role}</span>
          </span>
          <ChevronsUpDown aria-hidden className="size-4 shrink-0 text-muted-foreground" />
        </>
      )}
    </>
  );
}

export const USER_BUTTON = (collapsed: boolean) =>
  cn(
    "flex w-full items-center gap-2.5 rounded-lg p-1.5 text-left outline-none hover:bg-muted focus-visible:outline-2 focus-visible:outline-solid focus-visible:outline-ring",
    collapsed && "justify-center",
  );

// The dropdown is only needed after a click: its code is fetched then, keeping it out of the entry chunk (npm run budget).
const UserMenuDropdown = lazy(() => import("@/shared/layouts/UserMenuDropdown"));

/** The person signed in, as a menu: who they are, help, and signing out (personal, not workspace, actions). */
function UserMenu({ collapsed }: { collapsed: boolean }) {
  const session = useAuthStore((s) => s.session);
  const grants = usePermissionStore((s) => s.grants);
  const [wanted, setWanted] = useState(false);
  const name = session?.displayName ?? "Tidak diketahui";
  const role = grants?.roleName ?? session?.role ?? "";
  const face = <UserButtonFace name={name} role={role} collapsed={collapsed} />;
  const plain = (
    <button type="button" aria-label={`Menu akun ${name}`} aria-haspopup="menu" onClick={() => setWanted(true)} className={USER_BUTTON(collapsed)}>
      {face}
    </button>
  );
  if (!wanted) return plain;
  return (
    <Suspense fallback={plain}>
      <UserMenuDropdown name={name} role={role} collapsed={collapsed} face={face} helpUrl={HELP_WHATSAPP} />
    </Suspense>
  );
}

/** The WO Console's menu. */
export function Sidebar() {
  const logoUrl = useTenantBrandingStore((s) => s.logoUrl);
  const brandName = useTenantBrandingStore((s) => s.businessName);
  const grants = usePermissionStore((s) => s.grants);
  const name = brandName ?? APP_NAME;

  return (
    <AppSidebar
      sections={visibleSections(grants)}
      brand={(collapsed) =>
        collapsed ? (
          <WorkspaceMark name={name} logoUrl={logoUrl} size="sm" />
        ) : (
          <span className="flex min-w-0 items-center gap-2.5">
            <WorkspaceMark name={name} logoUrl={logoUrl} />
            <span className="min-w-0">
              <span className="block truncate text-[14px] font-semibold tracking-tight">{name}</span>
              <span className="block truncate text-[12px] text-muted-foreground">WO Console</span>
            </span>
          </span>
        )
      }
      aboveFooter={(collapsed) => <TrialChip collapsed={collapsed} />}
      footer={(collapsed) => <UserMenu collapsed={collapsed} />}
    />
  );
}
