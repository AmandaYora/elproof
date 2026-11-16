import { useEffect, useRef } from "react";
import { cn } from "@/shared/lib/cn";
import { SkeletonRows } from "@/shared/components/ui/Skeleton";
import { CalendarRow } from "@/modules/calendar/components/CalendarEntry";
import { dayHeading, dayNumber, DAY_SHORT, relativeDay, weekdayIndex } from "@/modules/calendar/lib/dates";
import type { CalendarItem } from "@/modules/calendar/types";

interface AgendaListProps {
  /** Only the days of the shown month that have entries, in order. */
  days: [string, CalendarItem[]][];
  today: string;
  loading: boolean;
}

/**
 * The month as a list of days (the phone default, and the linear alternative to the grid for keyboard and screen
 * reader users): one heading per day, its entries as full-width links. In the current month the list opens at today.
 */
export function AgendaList({ days, today, loading }: AgendaListProps) {
  const anchorRef = useRef<HTMLLIElement>(null);
  const scrolledRef = useRef(false);

  // Open at today (or the next day with entries) once, the first time the current month is shown.
  const anchorDay = days.find(([day]) => day >= today)?.[0];
  useEffect(() => {
    if (loading || scrolledRef.current || !anchorRef.current) return;
    scrolledRef.current = true;
    if (days[0]?.[0] < today) anchorRef.current.scrollIntoView({ block: "start" });
  }, [loading, days, today]);

  if (loading) {
    return (
      <div className="rounded-lg border border-border bg-card p-4">
        <SkeletonRows rows={4} label="Memuat agenda" />
      </div>
    );
  }

  return (
    <ol className="flex flex-col gap-3">
      {days.map(([day, items]) => {
        const rel = relativeDay(day, today);
        const past = day < today;
        return (
          <li key={day} ref={day === anchorDay ? anchorRef : undefined} className="scroll-mt-20 rounded-lg border border-border bg-card">
            <h3 className="flex items-center gap-3 border-b border-border-light px-3 py-2.5">
              <span
                aria-hidden
                className={cn(
                  "flex size-10 shrink-0 flex-col items-center justify-center rounded-md leading-none tabular-nums",
                  day === today ? "bg-primary text-primary-foreground" : past ? "bg-muted text-muted-foreground" : "bg-accent text-accent-foreground",
                )}
              >
                <span className="text-[11px] font-medium">{DAY_SHORT[weekdayIndex(day)]}</span>
                <span className="mt-0.5 text-[16px] font-bold">{dayNumber(day)}</span>
              </span>
              <span className="min-w-0">
                <span className={cn("block text-[14px] font-semibold", past ? "text-muted-foreground" : "text-foreground")}>{dayHeading(day)}</span>
                <span className="block text-[12px] text-muted-foreground">
                  {rel ? `${rel} · ` : ""}
                  {items.length} agenda
                </span>
              </span>
            </h3>
            <ul className="flex flex-col p-1.5">
              {items.map((item, i) => (
                <li key={`${item.kind}-${item.projectId}-${i}`}>
                  <CalendarRow item={item} today={today} />
                </li>
              ))}
            </ul>
          </li>
        );
      })}
    </ol>
  );
}
