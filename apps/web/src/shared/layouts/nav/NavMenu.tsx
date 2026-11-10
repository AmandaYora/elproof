import { lazy, Suspense, useEffect, useState } from "react";
import { NavLink, useLocation } from "react-router-dom";
import { ChevronDown } from "lucide-react";
import { cn } from "@/shared/lib/cn";
import type { NavItem, NavSection } from "@/shared/layouts/nav/types";

const RailTooltip = lazy(() => import("@/shared/layouts/nav/RailTooltip"));

export const ITEM =
  "flex min-h-10 items-center gap-3 rounded-md px-3 text-[13.5px] font-medium text-sidebar-foreground/80 transition-colors hover:bg-muted hover:text-sidebar-foreground focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-foreground aria-[current=page]:bg-sidebar-accent aria-[current=page]:font-semibold aria-[current=page]:text-sidebar-accent-foreground";

function Item({ item, rail, nested }: { item: NavItem; rail: boolean; nested?: boolean }) {
  const link = (
    <NavLink to={item.to} className={cn(ITEM, rail && "justify-center px-0", nested && !rail && "pl-10")} aria-label={rail ? item.label : undefined}>
      <item.icon aria-hidden className="size-[18px] shrink-0" />
      {!rail && <span className="truncate">{item.label}</span>}
    </NavLink>
  );
  if (!rail) return link;
  // The tooltip (Base UI positioning) is only needed on the narrow rail, so it loads then; until it does the link stands alone.
  return (
    <Suspense fallback={link}>
      <RailTooltip label={item.label}>{link}</RailTooltip>
    </Suspense>
  );
}

function Collapsible({ section, rail }: { section: NavSection; rail: boolean }) {
  const { pathname } = useLocation();
  const holdsRoute = section.items.some((i) => pathname.startsWith(i.to));
  const [open, setOpen] = useState(holdsRoute);
  // Landing on one of its pages opens it; it never closes by itself.
  useEffect(() => {
    if (holdsRoute) setOpen(true);
  }, [holdsRoute]);

  // On a narrow rail the group has no room for its own row: its pages show as plain icons.
  if (rail) return <>{section.items.map((item) => <Item key={item.to} item={item} rail />)}</>;

  const { label, icon: Icon } = section.collapsible!;
  const panelId = `nav-${section.id}`;
  return (
    <div>
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        aria-expanded={open}
        aria-controls={panelId}
        className={cn(ITEM, "w-full", holdsRoute && !open && "text-sidebar-foreground")}
      >
        <Icon aria-hidden className="size-[18px] shrink-0" />
        <span className="flex-1 text-left">{label}</span>
        <ChevronDown aria-hidden className={cn("size-4 shrink-0 transition-transform", open && "rotate-180")} />
      </button>
      {open && (
        <div id={panelId} className="mt-0.5 flex flex-col gap-0.5">
          {section.items.map((item) => (
            <Item key={item.to} item={item} rail={false} nested />
          ))}
        </div>
      )}
    </div>
  );
}

export function Menu({ sections, rail }: { sections: NavSection[]; rail: boolean }) {
  return (
    <nav aria-label="Menu utama" className="flex flex-1 flex-col gap-4 overflow-y-auto px-2.5 pb-3">
      {sections.map((section) => (
        <div key={section.id} className="flex flex-col gap-0.5">
          {section.label && !rail && <p className="px-3 pb-1 text-[11.5px] font-medium text-muted-foreground">{section.label}</p>}
          {section.collapsible ? (
            <Collapsible section={section} rail={rail} />
          ) : (
            section.items.map((item) => <Item key={item.to} item={item} rail={rail} />)
          )}
        </div>
      ))}
    </nav>
  );
}
