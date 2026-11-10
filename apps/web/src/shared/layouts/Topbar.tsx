import { Menu } from "lucide-react";
import { CommandPalette, CommandTrigger } from "@/shared/layouts/CommandPalette";
import { useSidebar } from "@/shared/layouts/nav/SidebarContext";
import { visibleSections } from "@/shared/layouts/Sidebar";
import { usePermissionStore } from "@/shared/stores/usePermissionStore";
import { Button } from "@/shared/components/kit/button";
import { formatDate, todayISO } from "@/shared/lib/formatters";

/** The WO Console's top bar: menu button (phone), search/shortcut palette and today's date. */
export function Topbar() {
  const { toggle } = useSidebar();
  const grants = usePermissionStore((s) => s.grants);
  const pages = visibleSections(grants).flatMap((s) => s.items);
  return (
    <header className="print:hidden sticky top-0 z-20 flex h-16 items-center justify-between gap-3 border-b bg-card px-4 sm:px-6">
      <Button variant="ghost" size="icon" onClick={toggle} className="lg:hidden">
        <Menu aria-hidden />
        <span className="sr-only">Buka menu</span>
      </Button>
      <CommandPalette pages={pages} searchRecords trigger={(open) => <CommandTrigger onOpen={open} />} />
      <div className="hidden shrink-0 text-[13px] font-medium text-muted-foreground sm:block">{formatDate(todayISO())}</div>
    </header>
  );
}
