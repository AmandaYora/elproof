import { lazy, Suspense, useEffect, useState, type ReactNode } from "react";
import { useLocation } from "react-router-dom";
import { PanelLeftClose, PanelLeftOpen } from "lucide-react";
import { cn } from "@/shared/lib/cn";
import { useSidebar } from "@/shared/layouts/nav/SidebarContext";
import { ITEM, Menu } from "@/shared/layouts/nav/NavMenu";
import type { NavSection } from "@/shared/layouts/nav/types";

// The sheet is a dialog; nobody on a computer ever needs it, so it is fetched when the menu is first opened.
const MobileSheet = lazy(() => import("@/shared/layouts/nav/MobileSheet"));

interface Props {
  /** Top of the menu: logo or name. Receives whether the rail is narrow. */
  brand: (collapsed: boolean) => ReactNode;
  sections: NavSection[];
  /** Bottom of the menu: who is signed in and the sign-out button. */
  footer: (collapsed: boolean) => ReactNode;
}

/**
 * The console menu (PLAN.md Tahap 7): neutral surface with the brand only as an accent, a narrow icon rail on a
 * computer, a sheet on a phone, Cmd/Ctrl+B for both. One component for the WO Console and the Platform Console.
 */
export function AppSidebar({ brand, sections, footer }: Props) {
  const { mobileOpen, setMobileOpen, collapsed, toggle } = useSidebar();
  const { pathname } = useLocation();
  const [sheetWanted, setSheetWanted] = useState(false);

  // Going to a page closes the sheet.
  useEffect(() => {
    setMobileOpen(false);
  }, [pathname, setMobileOpen]);
  useEffect(() => {
    if (mobileOpen) setSheetWanted(true);
  }, [mobileOpen]);

  return (
    <>
      <aside
        aria-label="Menu"
        className="print:hidden fixed inset-y-0 left-0 z-30 hidden w-(--sidebar-w) flex-col border-r bg-sidebar text-sidebar-foreground transition-[width] duration-200 lg:flex"
      >
        <div className={cn("flex min-h-16 items-center", collapsed ? "justify-center px-2" : "px-5")}>{brand(collapsed)}</div>
        <Menu sections={sections} rail={collapsed} />
        <div className="border-t px-2.5 py-2">
          <button
            type="button"
            onClick={toggle}
            aria-label={collapsed ? "Lebarkan menu" : "Ciutkan menu"}
            title="Ctrl+B"
            className={cn(ITEM, "w-full", collapsed && "justify-center px-0")}
          >
            {collapsed ? <PanelLeftOpen aria-hidden className="size-[18px]" /> : <PanelLeftClose aria-hidden className="size-[18px]" />}
            {!collapsed && <span>Ciutkan menu</span>}
          </button>
        </div>
        <div className={cn("border-t py-3", collapsed ? "px-2" : "px-5")}>{footer(collapsed)}</div>
      </aside>

      {(sheetWanted || mobileOpen) && (
        <Suspense fallback={null}>
          <MobileSheet open={mobileOpen} onOpenChange={setMobileOpen} brand={brand(false)} sections={sections} footer={footer(false)} />
        </Suspense>
      )}
    </>
  );
}
