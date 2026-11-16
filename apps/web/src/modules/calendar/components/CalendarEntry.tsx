import { Link } from "react-router-dom";
import { cn } from "@/shared/lib/cn";
import { itemHref, lookOf } from "@/modules/calendar/lib/items";
import type { CalendarItem } from "@/modules/calendar/types";

/**
 * One entry inside a month cell: a one-line chip that opens the project. It is a mouse shortcut (tabIndex -1): the
 * keyboard reaches the same links through the day panel (Enter on the day) or the Agenda view, so a busy month is one
 * Tab stop instead of dozens (RISET §7.7).
 */
export function CalendarChip({ item, today }: { item: CalendarItem; today: string }) {
  const look = lookOf(item, today);
  const Icon = look.icon;
  return (
    <Link
      to={itemHref(item)}
      tabIndex={-1}
      title={look.full}
      data-kind={item.kind}
      className={cn(
        "flex h-[22px] min-w-0 items-center gap-1 rounded-[6px] border px-1.5 text-[12px] leading-none font-medium max-sm:justify-center max-sm:px-0",
        "transition-[filter] duration-100 hover:brightness-95",
        look.chip,
      )}
    >
      <Icon aria-hidden className="size-3 shrink-0" />
      <span className="truncate max-sm:sr-only">{look.chipText}</span>
      <span className="sr-only">{look.srExtra}</span>
    </Link>
  );
}

/**
 * One entry as a full row (Agenda view and the day panel): what it is, which project, its status in words, and the
 * amount for an installment. The whole row is the link.
 */
export function CalendarRow({ item, today }: { item: CalendarItem; today: string }) {
  const look = lookOf(item, today);
  const Icon = look.icon;
  return (
    <Link
      to={itemHref(item)}
      data-kind={item.kind}
      className="group flex min-h-11 items-center gap-3 rounded-md px-2 py-2 hover:bg-muted focus-visible:outline-2 focus-visible:outline-offset-0 focus-visible:outline-ring focus-visible:outline-solid"
    >
      <span aria-hidden className={cn("flex size-8 shrink-0 items-center justify-center rounded-full", look.mark)}>
        <Icon className="size-4" />
      </span>
      <span className="min-w-0 flex-1">
        <span className="block truncate text-[14px] font-semibold text-foreground group-hover:underline group-hover:underline-offset-2">{look.heading}</span>
        <span className="block truncate text-[13px] text-muted-foreground">{look.meta}</span>
      </span>
      <span
        className={cn(
          "shrink-0 text-right text-[12px] font-medium",
          look.late ? "text-danger-text" : item.kind === "payment" && item.status === "paid" ? "text-success-text" : "text-muted-foreground",
        )}
      >
        {look.statusText}
      </span>
    </Link>
  );
}
