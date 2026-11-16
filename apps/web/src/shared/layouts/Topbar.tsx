import { useNavigate } from "react-router-dom";
import { Menu, Plus } from "lucide-react";
import { CommandPalette, CommandTrigger } from "@/shared/layouts/CommandPalette";
import { useSidebar } from "@/shared/layouts/nav/SidebarContext";
import { visibleSections } from "@/shared/layouts/Sidebar";
import { usePermissionStore, hasPermission } from "@/shared/stores/usePermissionStore";
import { Button } from "@/shared/components/kit/button";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import { formatDate, todayISO } from "@/shared/lib/formatters";

/**
 * The WO Console's top bar: menu button (phone), search/shortcut palette, today's date and the one thing people start
 * most often, a new project, reachable from every page (RISET WO Console §3 #10).
 */
export function Topbar() {
  const { toggle } = useSidebar();
  const navigate = useNavigate();
  const grants = usePermissionStore((s) => s.grants);
  const canCreate = hasPermission(grants, "project.create");
  const pages = visibleSections(grants).flatMap((s) => s.items);
  return (
    <header className="print:hidden sticky top-0 z-20 flex h-16 items-center justify-between gap-3 border-b bg-card px-4 sm:px-6">
      <Button variant="ghost" size="icon" onClick={toggle} className="lg:hidden">
        <Menu aria-hidden />
        <span className="sr-only">Buka menu</span>
      </Button>
      <CommandPalette pages={pages} searchRecords trigger={(open) => <CommandTrigger onOpen={open} />} />
      <div className="flex shrink-0 items-center gap-3">
        <span className="hidden text-[13px] font-medium text-muted-foreground xl:block">{formatDate(todayISO())}</span>
        {canCreate && (
          <Button onClick={() => navigate(ROUTE_PATHS.projectNew)} className="hidden sm:inline-flex">
            <Plus aria-hidden /> Proyek baru
          </Button>
        )}
      </div>
    </header>
  );
}
