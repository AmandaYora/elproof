import { ArrowDown, ArrowUp, Copy, Info, Pencil, Star, Trash2 } from "lucide-react";
import { Button } from "@/shared/components/ui/Button";
import { Badge } from "@/shared/components/ui/Badge";
import { ActionMenu } from "@/shared/components/ui/ActionMenu";
import { cn } from "@/shared/lib/cn";
import { EVENT_LAYOUT_LABELS } from "@/shared/lib/labels";
import type { RincianEvent } from "@/modules/rincian/types";
import { capacityFor, dayAndMonth, formatCount, formatEventDay, formatTimeRange } from "@/modules/rincian/components/events/event-display";

export type EventRowAction = "edit" | "main" | "up" | "down" | "duplicate" | "delete";

interface EventListProps {
  /** In screen order (the server's: by date, then the order set with "Pindah"). */
  events: RincianEvent[];
  canEdit: boolean;
  /** The event an action is running for: its buttons wait. */
  busyId: string | null;
  onAction: (event: RincianEvent, action: EventRowAction) => void;
  /** Actions this screen does not offer (a quotation has no "acara utama" to set by hand). */
  hide?: EventRowAction[];
}

/** The project's events as one list with dividers, not a card per event (DESAIN §5.1). */
export function EventList({ events, canEdit, busyId, onAction, hide = [] }: EventListProps) {
  return (
    <ol className="divide-y divide-border-light">
      {events.map((e, i) => {
        const prev = events[i - 1];
        const next = events[i + 1];
        // The order follows the date; "Pindah" only swaps events on the same day (the server refuses the rest).
        const canUp = prev !== undefined && prev.eventDate === e.eventDate;
        const canDown = next !== undefined && next.eventDate === e.eventDate;
        return (
          <li key={e.id}>
            <EventRow
              event={e}
              canEdit={canEdit}
              busy={busyId === e.id}
              canUp={canUp}
              canDown={canDown}
              canDelete={events.length > 1}
              hide={hide}
              onAction={(action) => onAction(e, action)}
            />
          </li>
        );
      })}
    </ol>
  );
}

interface EventRowProps {
  event: RincianEvent;
  canEdit: boolean;
  busy: boolean;
  canUp: boolean;
  canDown: boolean;
  canDelete: boolean;
  hide: EventRowAction[];
  onAction: (action: EventRowAction) => void;
}

function EventRow({ event: e, canEdit, busy, canUp, canDown, canDelete, hide, onAction }: EventRowProps) {
  // An event without a date yet (a quotation's) shows a dash in the day tile.
  const { day, month } = e.eventDate ? dayAndMonth(e.eventDate) : { day: "—", month: "" };
  const time = formatTimeRange(e.startTime, e.endTime);
  const limit = capacityFor(e.venue, e.layout);
  const layoutWord = EVENT_LAYOUT_LABELS[e.layout];

  return (
    <article aria-label={`${e.kindName}, ${formatEventDay(e.eventDate)}`} className="flex gap-4 px-5 py-4">
      {/* The day at a glance; the main event's tile is filled, because its date is the project's Hari H. */}
      <div
        aria-hidden
        className={cn(
          "flex h-13 w-12 shrink-0 flex-col items-center justify-center rounded-lg leading-none",
          e.isMain ? "bg-primary text-primary-foreground" : "bg-muted text-foreground",
        )}
      >
        <span className="text-[19px] font-semibold tabular-nums">{day}</span>
        <span className={cn("mt-1 text-[12px] font-medium", e.isMain ? "text-on-primary-muted" : "text-muted-foreground")}>{month}</span>
      </div>

      <div className="flex min-w-0 flex-1 flex-col gap-1">
        <div className="flex flex-wrap items-start justify-between gap-x-3 gap-y-2">
          <h3 className="flex min-w-0 flex-wrap items-center gap-x-2 gap-y-1 text-[14px] leading-snug">
            <span className="font-semibold text-foreground">{e.kindName}</span>
            <span className="text-muted-foreground">
              {formatEventDay(e.eventDate)}
              {time && <span className="tabular-nums"> · {time}</span>}
            </span>
            {e.isMain && (
              <Badge tone="brand" dot={false}>
                Acara utama
              </Badge>
            )}
          </h3>
          {canEdit && (
            <div className="flex shrink-0 items-center gap-1.5">
              <Button variant="secondary" size="sm" disabled={busy} icon={<Pencil className="size-3.5" />} onClick={() => onAction("edit")} aria-label={`Ubah acara ${e.kindName}`}>
                Ubah
              </Button>
              <ActionMenu
                variant="ghost"
                label={`Aksi lain untuk acara ${e.kindName}`}
                disabled={busy}
                items={[
                  { label: "Jadikan acara utama", icon: <Star className="size-4" />, onSelect: () => onAction("main"), hidden: e.isMain || hide.includes("main") },
                  { label: "Pindah ke atas", icon: <ArrowUp className="size-4" />, onSelect: () => onAction("up"), hidden: !canUp || hide.includes("up") },
                  { label: "Pindah ke bawah", icon: <ArrowDown className="size-4" />, onSelect: () => onAction("down"), hidden: !canDown || hide.includes("down") },
                  { label: "Duplikat acara", icon: <Copy className="size-4" />, onSelect: () => onAction("duplicate"), hidden: hide.includes("duplicate") },
                  { label: "Hapus acara", icon: <Trash2 className="size-4" />, tone: "danger", onSelect: () => onAction("delete"), hidden: !canDelete || hide.includes("delete") },
                ]}
              />
            </div>
          )}
        </div>

        {e.venue ? (
          <p className="text-[13px] text-foreground">
            {e.venue.name}
            <span className="text-muted-foreground">
              {" · "}
              {layoutWord} ({limit !== null ? `kapasitas ${formatCount(limit)}` : `kapasitas ${layoutWord.toLowerCase()} belum diisi`})
            </span>
          </p>
        ) : (
          <p className="flex items-center gap-1.5 text-[13px] text-muted-foreground">
            <Info aria-hidden className="size-3.5 shrink-0" />
            Belum ada venue: tamu belum dicek ke kapasitas
          </p>
        )}

        <SessionsLine event={e} />
      </div>
    </article>
  );
}

function SessionsLine({ event: e }: { event: RincianEvent }) {
  if (e.sessions.length === 0) return null;
  if (e.sessions.length === 1) {
    const s = e.sessions[0];
    const time = formatTimeRange(s.startTime, s.endTime);
    return (
      <p className="text-[13px] text-muted-foreground tabular-nums">
        1 sesi{time && ` ${time}`} · {formatCount(s.guests)} tamu
      </p>
    );
  }
  return (
    <ul aria-label="Sesi" className="flex flex-wrap gap-x-4 gap-y-0.5 text-[13px] text-muted-foreground tabular-nums">
      {e.sessions.map((s) => {
        const time = formatTimeRange(s.startTime, s.endTime);
        return (
          <li key={s.id}>
            <span className="text-foreground">{s.label}</span>
            {time && ` ${time}`} · {formatCount(s.guests)} tamu
          </li>
        );
      })}
      <li>Total {formatCount(e.totalGuests)} tamu</li>
    </ul>
  );
}
