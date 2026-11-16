import { useEffect, useRef, type KeyboardEvent } from "react";
import { cn } from "@/shared/lib/cn";
import { Skeleton } from "@/shared/components/ui/Skeleton";
import { CalendarChip } from "@/modules/calendar/components/CalendarEntry";
import { DAY_NAMES, DAY_SHORT, addDays, dayNumber, longDate, monthOf, sameDayIn, shiftMonth, weekdayIndex } from "@/modules/calendar/lib/dates";
import type { CalendarItem } from "@/modules/calendar/types";

/** Entries shown inside a cell before "+N lainnya" (RISET §7.7). */
const MAX_PER_DAY = 3;

interface MonthGridProps {
  month: string;
  weeks: string[][];
  today: string;
  byDay: Map<string, CalendarItem[]>;
  loading: boolean;
  /** The day holding the grid's single Tab stop (roving tabindex). */
  activeDay: string;
  /** Move the active day; `focus` is true when the keyboard moved it (the cell then takes focus). */
  onActiveDayChange: (day: string, focus: boolean) => void;
  /** The keyboard left the shown month (PageUp/PageDown or arrows past the edge). */
  onMonthChange: (month: string, day: string) => void;
  onOpenDay: (day: string) => void;
  /** Set by the page after a keyboard move, so the new cell takes focus once it is rendered. */
  focusRequested: boolean;
  onFocusHandled: () => void;
  labelledBy: string;
}

/**
 * The month as a WAI-ARIA grid (APG date grid): one Tab stop, arrows move a day/week, Home/End go to the start/end of
 * the week, PageUp/PageDown change month, Enter or Space opens the day panel with every entry as a link. Day names are
 * spelled out for screen readers and abbreviated on screen (Roselli, calendar tables).
 */
export function MonthGrid({
  month, weeks, today, byDay, loading, activeDay, onActiveDayChange, onMonthChange, onOpenDay, focusRequested, onFocusHandled, labelledBy,
}: MonthGridProps) {
  const tableRef = useRef<HTMLTableElement>(null);

  useEffect(() => {
    if (!focusRequested) return;
    // After PageUp/PageDown the new month can arrive one render later (the router updates the URL in a transition):
    // keep the request until the target cell exists.
    const cell = tableRef.current?.querySelector<HTMLElement>(`[data-day="${activeDay}"]`);
    if (!cell) return;
    cell.focus();
    onFocusHandled();
  }, [focusRequested, activeDay, weeks, onFocusHandled]);

  const first = weeks[0][0];
  const last = weeks[weeks.length - 1][6];

  function move(day: string) {
    if (day < first || day > last) onMonthChange(monthOf(day), day);
    else onActiveDayChange(day, true);
  }

  function onKeyDown(e: KeyboardEvent<HTMLTableElement>) {
    const wd = weekdayIndex(activeDay);
    switch (e.key) {
      case "ArrowLeft": move(addDays(activeDay, -1)); break;
      case "ArrowRight": move(addDays(activeDay, 1)); break;
      case "ArrowUp": move(addDays(activeDay, -7)); break;
      case "ArrowDown": move(addDays(activeDay, 7)); break;
      case "Home": move(addDays(activeDay, -wd)); break;
      case "End": move(addDays(activeDay, 6 - wd)); break;
      case "PageUp": {
        const m = shiftMonth(month, -1);
        onMonthChange(m, sameDayIn(activeDay, m));
        break;
      }
      case "PageDown": {
        const m = shiftMonth(month, 1);
        onMonthChange(m, sameDayIn(activeDay, m));
        break;
      }
      case "Enter":
      case " ":
        if ((e.target as HTMLElement).dataset.day) onOpenDay(activeDay);
        else return;
        break;
      default:
        return;
    }
    e.preventDefault();
  }

  return (
    <div className="overflow-hidden rounded-lg border border-border bg-card">
      <table
        ref={tableRef}
        role="grid"
        aria-labelledby={labelledBy}
        aria-busy={loading || undefined}
        onKeyDown={onKeyDown}
        className="w-full table-fixed border-collapse"
      >
        <thead>
          <tr>
            {DAY_SHORT.map((short, i) => (
              <th
                key={short}
                scope="col"
                className={cn(
                  "border-b border-border bg-muted/60 px-2 py-2 text-left text-[12px] font-semibold text-muted-foreground",
                  i >= 5 && "text-foreground",
                )}
              >
                <span aria-hidden>{short}</span>
                <span className="sr-only">{DAY_NAMES[i]}</span>
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {weeks.map((week) => (
            <tr key={week[0]}>
              {week.map((day) => {
                const items = byDay.get(day) ?? [];
                const inMonth = monthOf(day) === month;
                const isToday = day === today;
                const hasEvent = items.some((it) => it.kind === "event");
                const shown = items.slice(0, MAX_PER_DAY);
                const more = items.length - shown.length;
                return (
                  <td
                    key={day}
                    role="gridcell"
                    data-day={day}
                    tabIndex={day === activeDay ? 0 : -1}
                    aria-current={isToday ? "date" : undefined}
                    onFocus={() => day !== activeDay && onActiveDayChange(day, false)}
                    className={cn(
                      "h-16 border-r border-b border-border p-1 align-top last:border-r-0 sm:h-[7.25rem] sm:p-1.5",
                      // `focus`, not `focus-visible`: the cell is focused by script after an arrow key, which browsers do not
                      // always mark as focus-visible, and the ring is also the "active day" marker after a click.
                      "focus:relative focus:z-[1] focus:outline-2 focus:-outline-offset-2 focus:outline-ring focus:outline-solid",
                      !inMonth && "bg-muted/45",
                      hasEvent && inMonth && "bg-accent/35",
                    )}
                  >
                    <div className="mb-1 flex items-center justify-between gap-1">
                      <button
                        type="button"
                        tabIndex={-1}
                        aria-hidden
                        onClick={() => {
                          onActiveDayChange(day, false);
                          onOpenDay(day);
                        }}
                        className={cn(
                          "flex size-6 items-center justify-center rounded-full text-[13px] font-semibold tabular-nums hover:bg-muted",
                          isToday ? "bg-primary text-primary-foreground hover:bg-primary-hover" : inMonth ? "text-foreground" : "text-subtle-foreground",
                        )}
                      >
                        {dayNumber(day)}
                      </button>
                      <span className="sr-only">
                        {longDate(day)}
                        {isToday ? ", hari ini" : ""}
                        {items.length === 0 ? ", tidak ada agenda" : `, ${items.length} agenda`}
                      </span>
                    </div>
                    {loading ? (
                      // Deterministic placeholders so the grid does not flicker into a different shape.
                      dayNumber(day) % 4 === 1 && <Skeleton className="h-[22px] w-4/5 rounded-[6px]" />
                    ) : (
                      <ul className="flex flex-col gap-1">
                        {shown.map((item, i) => (
                          <li key={`${item.kind}-${item.projectId}-${i}`} className="min-w-0">
                            <CalendarChip item={item} today={today} />
                          </li>
                        ))}
                        {more > 0 && (
                          <li>
                            <button
                              type="button"
                              tabIndex={-1}
                              onClick={() => {
                                onActiveDayChange(day, false);
                                onOpenDay(day);
                              }}
                              className="h-[22px] rounded-[6px] px-1.5 text-[12px] font-semibold text-muted-foreground hover:bg-muted hover:text-foreground"
                            >
                              +{more} lainnya
                            </button>
                          </li>
                        )}
                      </ul>
                    )}
                  </td>
                );
              })}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
