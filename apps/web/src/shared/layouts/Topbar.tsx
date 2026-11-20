import { lazy, Suspense, useState } from "react";
import { useNavigate } from "react-router-dom";
import { ChevronDown, Menu, Plus } from "lucide-react";
import { CommandPalette, CommandTrigger } from "@/shared/layouts/CommandPalette";
import { useSidebar } from "@/shared/layouts/nav/SidebarContext";
import { visibleSections } from "@/shared/layouts/Sidebar";
import { usePermissionStore, hasPermission } from "@/shared/stores/usePermissionStore";
import { Button } from "@/shared/components/kit/button";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import { formatDate, todayISO } from "@/shared/lib/formatters";

export interface CreateItem {
  key: string;
  /** In the "Baru" menu: "Proyek". */
  label: string;
  /** As the only button, when it is the one thing the role can start: "Proyek baru". */
  direct: string;
  permission: string;
  to: string;
}

// What the top bar starts. One entry today; with two or more allowed, the button becomes the "Baru" menu.
const CREATE_ITEMS: CreateItem[] = [
  // "Penawaran baru" opens its dialog on the list page (?baru=1).
  { key: "quotation", label: "Penawaran", direct: "Penawaran baru", permission: "quotation.manage", to: ROUTE_PATHS.quotationNew },
  { key: "project", label: "Proyek", direct: "Proyek baru", permission: "project.create", to: ROUTE_PATHS.projectNew },
];

// The menu is only needed after a click: its code is fetched then, keeping it out of the entry chunk (npm run budget).
const CreateMenuDropdown = lazy(() => import("@/shared/layouts/CreateMenuDropdown"));

/**
 * The WO Console's top bar: menu button (phone), search/shortcut palette, today's date and what people start most
 * often, reachable from every page (RISET WO Console §3 #10).
 */
export function Topbar() {
  const { toggle } = useSidebar();
  const grants = usePermissionStore((s) => s.grants);
  const creatable = CREATE_ITEMS.filter((item) => hasPermission(grants, item.permission));
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
        <CreateButton items={creatable} />
      </div>
    </header>
  );
}

const CREATE_BUTTON = "hidden sm:inline-flex";

/** Nothing when the role starts nothing; a direct button for a single thing; otherwise "Baru ▾" with the list. */
function CreateButton({ items }: { items: CreateItem[] }) {
  const navigate = useNavigate();
  const [wanted, setWanted] = useState(false);
  if (items.length === 0) return null;
  if (items.length === 1) {
    const only = items[0];
    return (
      <Button onClick={() => navigate(only.to)} className={CREATE_BUTTON}>
        <Plus aria-hidden /> {only.direct}
      </Button>
    );
  }
  const plain = (
    <Button aria-haspopup="menu" onClick={() => setWanted(true)} className={CREATE_BUTTON}>
      <Plus aria-hidden /> Baru <ChevronDown aria-hidden />
    </Button>
  );
  if (!wanted) return plain;
  return (
    <Suspense fallback={plain}>
      <CreateMenuDropdown items={items} triggerClassName={CREATE_BUTTON} />
    </Suspense>
  );
}
