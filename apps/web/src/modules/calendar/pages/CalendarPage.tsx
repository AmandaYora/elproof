import { Suspense, lazy, useCallback, useEffect, useId, useMemo, useState } from "react";
import { useSearchParams } from "react-router-dom";
import { CalendarDays, ChevronLeft, ChevronRight, CircleAlert, CircleCheck, CircleDashed, Clock } from "lucide-react";
import { PageHeader } from "@/shared/components/ui/PageHeader";
import { Button } from "@/shared/components/ui/Button";
import { Select } from "@/shared/components/ui/Input";
import { LoadError } from "@/shared/components/feedback/LoadError";
import { useMediaQuery } from "@/shared/hooks/useMediaQuery";
import { useAllProjects } from "@/shared/stores/usePermissionStore";
import { todayISO } from "@/shared/lib/formatters";
import { cn } from "@/shared/lib/cn";
import { useStaffStore } from "@/modules/users/stores/useStaffStore";
import { MonthGrid } from "@/modules/calendar/components/MonthGrid";
import { AgendaList } from "@/modules/calendar/components/AgendaList";
import { calendarQueryKey, useCalendarStore, type CalendarQuery } from "@/modules/calendar/stores/useCalendarStore";
import { isMonthKey, monthGrid, monthLabel, monthOf, shiftMonth } from "@/modules/calendar/lib/dates";
import { KIND_ICON, KIND_LABELS } from "@/modules/calendar/lib/items";
import { CALENDAR_KINDS, type CalendarItem, type CalendarKind, type CalendarView } from "@/modules/calendar/types";

// The day panel is only needed after a click: it loads on demand (frontend rule, entry budget).
const DayDrawer = lazy(() => import("@/modules/calendar/components/DayDrawer"));

function parseKinds(raw: string | null): CalendarKind[] {
  if (!raw) return CALENDAR_KINDS;
  const picked = CALENDAR_KINDS.filter((k) => raw.split(",").includes(k));
  return picked.length > 0 ? picked : CALENDAR_KINDS;
}

/**
 * Kalender: every wedding day, open schedule step and client installment of the projects the person reaches, by day
 * (RISET WO Console §3 #2, design brief §7.7). Month grid on wide screens, agenda list on phones; month, view and
 * filters live in the URL so a link opens the same view.
 */
export default function CalendarPage() {
  const today = todayISO();
  const [params, setParams] = useSearchParams();
  const wide = useMediaQuery("(min-width: 768px)");
  const seesAll = useAllProjects();

  const month = isMonthKey(params.get("bulan")) ? (params.get("bulan") as string) : monthOf(today);
  const viewParam = params.get("tampilan");
  const view: CalendarView = viewParam === "bulan" || viewParam === "agenda" ? viewParam : wide ? "bulan" : "agenda";
  const kinds = parseKinds(params.get("jenis"));
  const pic = seesAll ? (params.get("pic") ?? "") : "";
  const filtersActive = kinds.length < CALENDAR_KINDS.length || pic !== "";

  const grid = useMemo(() => monthGrid(month), [month]);
  // The grid needs its padding days too; the agenda only the month itself.
  const query: CalendarQuery = useMemo(
    () => ({ from: grid.from, to: grid.to, picStaffId: pic || undefined, kinds }),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [grid.from, grid.to, pic, kinds.join(",")],
  );
  const key = calendarQueryKey(query);

  const items = useCalendarStore((s) => s.items);
  const loadedKey = useCalendarStore((s) => s.loadedKey);
  const status = useCalendarStore((s) => s.status);
  const fetchCalendar = useCalendarStore((s) => s.fetchCalendar);
  const staff = useStaffStore((s) => s.staffSummaries);
  const fetchStaffSummaries = useStaffStore((s) => s.fetchStaffSummaries);

  const load = useCallback(() => {
    void fetchCalendar(query).catch(() => undefined);
  }, [fetchCalendar, query]);
  useEffect(load, [load]);

  // The PIC filter needs the team's names: a small list, loaded once, only for roles that see every project.
  useEffect(() => {
    if (seesAll) void fetchStaffSummaries().catch(() => undefined);
  }, [seesAll, fetchStaffSummaries]);

  const fresh = loadedKey === key;
  const loading = !fresh && status !== "error";
  const failed = status === "error" && !fresh;

  const byDay = useMemo(() => {
    const map = new Map<string, CalendarItem[]>();
    if (!fresh) return map;
    for (const item of items) {
      const list = map.get(item.date);
      if (list) list.push(item);
      else map.set(item.date, [item]);
    }
    return map;
  }, [items, fresh]);

  const monthDays = useMemo(
    () => [...byDay.entries()].filter(([day]) => monthOf(day) === month).sort(([a], [b]) => (a < b ? -1 : 1)),
    [byDay, month],
  );
  const summary = useMemo(() => {
    const all = monthDays.flatMap(([, list]) => list);
    return {
      total: all.length,
      events: all.filter((i) => i.kind === "event").length,
      due: all.filter((i) => i.kind === "payment" && i.status !== "paid").length,
      overdue: all.filter((i) => i.kind === "payment" && i.status === "overdue").length,
    };
  }, [monthDays]);

  // Grid focus (roving tabindex) and the day panel.
  const [activeDay, setActiveDay] = useState(() => (monthOf(today) === month ? today : `${month}-01`));
  const [focusRequested, setFocusRequested] = useState(false);
  const [drawerDay, setDrawerDay] = useState<string | null>(null);
  const [drawerUsed, setDrawerUsed] = useState(false);
  const onFocusHandled = useCallback(() => setFocusRequested(false), []);
  useEffect(() => {
    // A keyboard move to another month is waiting for that month to render: not an orphan day.
    if (focusRequested) return;
    if (activeDay < grid.from || activeDay > grid.to) setActiveDay(monthOf(today) === month ? today : `${month}-01`);
  }, [activeDay, focusRequested, grid.from, grid.to, month, today]);

  const setParam = useCallback(
    (changes: Record<string, string | null>) => {
      setParams(
        (prev) => {
          const next = new URLSearchParams(prev);
          for (const [k, v] of Object.entries(changes)) {
            if (v === null || v === "") next.delete(k);
            else next.set(k, v);
          }
          return next;
        },
        { replace: true },
      );
    },
    [setParams],
  );

  const goToMonth = (m: string) => {
    setParam({ bulan: m === monthOf(today) ? null : m });
    setActiveDay(monthOf(today) === m ? today : `${m}-01`);
  };

  const toggleKind = (kind: CalendarKind) => {
    const on = kinds.includes(kind);
    if (on && kinds.length === 1) return; // at least one kind stays shown
    const next = on ? kinds.filter((k) => k !== kind) : CALENDAR_KINDS.filter((k) => k === kind || kinds.includes(k));
    setParam({ jenis: next.length === CALENDAR_KINDS.length ? null : next.join(",") });
  };

  const headingId = useId();
  const label = monthLabel(month);
  const isCurrentMonth = month === monthOf(today);

  const openDay = (day: string) => {
    setDrawerUsed(true);
    setDrawerDay(day);
  };

  return (
    <div>
      <PageHeader
        title="Kalender"
        actions={
          <>
            <div className="flex items-center gap-1">
              <Button variant="ghost" size="sm" aria-label="Bulan sebelumnya" onClick={() => goToMonth(shiftMonth(month, -1))} className="pointer-coarse:size-11">
                <ChevronLeft aria-hidden />
              </Button>
              <h2 id={headingId} aria-live="polite" className="min-w-[9.5rem] text-center text-[16px] font-semibold tabular-nums text-foreground">
                {label}
              </h2>
              <Button variant="ghost" size="sm" aria-label="Bulan berikutnya" onClick={() => goToMonth(shiftMonth(month, 1))} className="pointer-coarse:size-11">
                <ChevronRight aria-hidden />
              </Button>
            </div>
            <Button variant="secondary" size="sm" onClick={() => goToMonth(monthOf(today))} disabled={isCurrentMonth && activeDay === today}>
              Hari ini
            </Button>
            <div role="group" aria-label="Tampilan" className="inline-flex rounded-md border border-border bg-muted p-0.5">
              {(["bulan", "agenda"] as const).map((v) => (
                <button
                  key={v}
                  type="button"
                  aria-pressed={view === v}
                  onClick={() => setParam({ tampilan: v })}
                  className={cn(
                    "h-8 rounded-[6px] px-3 text-[13px] font-medium pointer-coarse:h-10",
                    "focus-visible:outline-2 focus-visible:outline-offset-1 focus-visible:outline-ring focus-visible:outline-solid",
                    view === v ? "bg-card text-foreground shadow-xs" : "text-muted-foreground hover:text-foreground",
                  )}
                >
                  {v === "bulan" ? "Bulan" : "Agenda"}
                </button>
              ))}
            </div>
          </>
        }
      />

      <div className="mb-4 flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
        <div className="flex flex-wrap items-center gap-2">
          <div role="group" aria-label="Jenis" className="flex flex-wrap gap-1.5">
            {CALENDAR_KINDS.map((kind) => {
              const Icon = KIND_ICON[kind];
              const on = kinds.includes(kind);
              return (
                <button
                  key={kind}
                  type="button"
                  aria-pressed={on}
                  onClick={() => toggleKind(kind)}
                  title={on && kinds.length === 1 ? "Minimal satu jenis tetap tampil" : undefined}
                  className={cn(
                    "inline-flex h-8 items-center gap-1.5 rounded-full border px-3 text-[13px] font-medium pointer-coarse:h-10",
                    "focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring focus-visible:outline-solid",
                    on
                      ? kind === "event"
                        ? "border-primary bg-primary text-primary-foreground"
                        : "border-input bg-card text-foreground"
                      : "border-dashed border-input bg-transparent text-muted-foreground line-through decoration-muted-foreground/60",
                  )}
                >
                  <Icon aria-hidden className="size-3.5" />
                  {KIND_LABELS[kind]}
                </button>
              );
            })}
          </div>
          {seesAll && (
            <Select className="w-48" aria-label="PIC" value={pic || "Semua"} onChange={(e) => setParam({ pic: e.target.value === "Semua" ? null : e.target.value })}>
              <option value="Semua">Semua PIC</option>
              {staff.map((m) => (
                <option key={m.id} value={m.id}>
                  {m.name}
                </option>
              ))}
            </Select>
          )}
          {filtersActive && (
            <Button variant="ghost" size="sm" onClick={() => setParam({ jenis: null, pic: null })}>
              Reset filter
            </Button>
          )}
        </div>
        {fresh && summary.total > 0 && (
          <p className="text-[13px] text-muted-foreground" aria-live="polite">
            {[`${summary.total} agenda`, summary.events > 0 && `${summary.events} hari acara`, summary.due > 0 && `${summary.due} termin belum lunas`]
              .filter(Boolean)
              .join(" · ")}
            {summary.overdue > 0 && <span className="font-semibold text-danger-text"> · {summary.overdue} termin terlambat</span>}
          </p>
        )}
      </div>

      {failed ? (
        <LoadError message="Kalender belum bisa dimuat. Periksa koneksi, lalu coba lagi." onRetry={load} />
      ) : (
        <>
          {fresh && summary.total === 0 && (
            <div role="status" className="mb-4 flex items-start gap-3 rounded-lg border border-border bg-card px-4 py-3">
              <CalendarDays aria-hidden className="mt-0.5 size-5 shrink-0 text-muted-foreground" />
              <div className="min-w-0 flex-1">
                <p className="text-[14px] font-semibold text-foreground">Tidak ada agenda di {label}</p>
                <p className="text-[13px] text-muted-foreground">
                  {filtersActive
                    ? "Saringan sedang aktif. Tampilkan semua jenis dan PIC untuk melihat agenda lain."
                    : "Hari acara, tahapan jadwal, dan jatuh tempo termin dari proyek Anda muncul di sini."}
                </p>
              </div>
              {filtersActive && (
                <Button variant="secondary" size="sm" onClick={() => setParam({ jenis: null, pic: null })}>
                  Reset filter
                </Button>
              )}
            </div>
          )}

          {view === "bulan" ? (
            <>
              <MonthGrid
                month={month}
                weeks={grid.weeks}
                today={today}
                byDay={byDay}
                loading={loading}
                activeDay={activeDay}
                onActiveDayChange={(day, focus) => {
                  setActiveDay(day);
                  if (focus) setFocusRequested(true);
                }}
                onMonthChange={(m, day) => {
                  setParam({ bulan: m === monthOf(today) ? null : m });
                  setActiveDay(day);
                  setFocusRequested(true);
                }}
                onOpenDay={openDay}
                focusRequested={focusRequested}
                onFocusHandled={onFocusHandled}
                labelledBy={headingId}
              />
              <Legend />
            </>
          ) : (
            (loading || monthDays.length > 0) && <AgendaList key={month} days={monthDays} today={today} loading={loading} />
          )}
        </>
      )}

      {drawerUsed && (
        <Suspense fallback={null}>
          <DayDrawer day={drawerDay} items={drawerDay ? (byDay.get(drawerDay) ?? []) : []} today={today} onClose={() => setDrawerDay(null)} />
        </Suspense>
      )}
    </div>
  );
}

/** What the payment icons mean, and how to use the grid from the keyboard. */
function Legend() {
  const marks = [
    { icon: CircleAlert, text: "Terlambat", cls: "text-danger-text" },
    { icon: Clock, text: "Segera jatuh tempo", cls: "text-warning-text" },
    { icon: CircleDashed, text: "Dibayar sebagian", cls: "text-warning-text" },
    { icon: CircleCheck, text: "Lunas", cls: "text-success-text" },
  ];
  return (
    <div className="mt-3 flex flex-col gap-2 text-[12px] text-muted-foreground sm:flex-row sm:items-center sm:justify-between">
      <ul className="flex flex-wrap items-center gap-x-4 gap-y-1" aria-label="Arti tanda termin">
        {marks.map(({ icon: Icon, text, cls }) => (
          <li key={text} className="inline-flex items-center gap-1.5">
            <Icon aria-hidden className={cn("size-3.5", cls)} />
            {text}
          </li>
        ))}
      </ul>
      <p className="max-md:hidden">Panah untuk berpindah hari, Enter untuk membuka semua agenda hari itu.</p>
    </div>
  );
}
