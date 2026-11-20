import { useEffect, useMemo, useRef, useState } from "react";
import { Check, CircleAlert, MapPin, Search } from "lucide-react";
import { Button } from "@/shared/components/ui/Button";
import { Input } from "@/shared/components/ui/Input";
import { cn } from "@/shared/lib/cn";
import { EVENT_LAYOUT_LABELS } from "@/shared/lib/labels";
import type { EventLayout } from "@/modules/rincian/types";
import { capacityFor, formatCount, type VenueCaps } from "@/modules/rincian/components/events/event-display";

/** How many matches are drawn at once; the search narrows the rest. */
const SHOWN = 30;

interface VenuePickerProps {
  /** The chosen venue (null = none yet). */
  selected: VenueCaps | null;
  /** Active venues of the directory. */
  venues: VenueCaps[];
  status: "loading" | "ready" | "error";
  onRetry: () => void;
  layout: EventLayout;
  /** The largest session: a venue whose number for the layout is below it is marked as not fitting. */
  maxGuests: number;
  onPick: (venueId: string) => void;
  onClear: () => void;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** Field key for focus and error jumps ("venueId", "events[0].venueId"). */
  name: string;
  error?: string;
  disabled?: boolean;
}

type Fit = "fits" | "unknown" | "missing" | "over";

function fitOf(v: VenueCaps, layout: EventLayout, maxGuests: number): Fit {
  const limit = capacityFor(v, layout);
  if (limit === null) return "missing";
  if (maxGuests <= 0) return "unknown";
  return maxGuests <= limit ? "fits" : "over";
}

const FIT_ORDER: Record<Fit, number> = { fits: 0, unknown: 0, missing: 1, over: 2 };

export function capacityLine(v: VenueCaps): string {
  const parts = [
    v.capacityStanding !== null ? `Berdiri ${formatCount(v.capacityStanding)}` : null,
    v.capacitySeated !== null ? `Duduk ${formatCount(v.capacitySeated)}` : null,
  ].filter(Boolean);
  return parts.length > 0 ? parts.join(" · ") : "Kapasitas belum diisi";
}

/**
 * The venue of an event: the chosen one with its capacities, and a searchable list of the directory that shows every
 * venue's numbers and marks the ones the largest session does not fit (DESAIN §5.1, "Ganti venue").
 */
export function VenuePicker({
  selected, venues, status, onRetry, layout, maxGuests, onPick, onClear, open, onOpenChange, name, error, disabled,
}: VenuePickerProps) {
  const [query, setQuery] = useState("");
  const searchRef = useRef<HTMLInputElement>(null);
  const triggerRef = useRef<HTMLButtonElement>(null);
  const errorId = `${name}-venue-error`;

  useEffect(() => {
    if (open) searchRef.current?.focus();
  }, [open]);

  const matches = useMemo(() => {
    const q = query.trim().toLowerCase();
    const found = q === "" ? venues : venues.filter((v) => `${v.name} ${v.city ?? ""}`.toLowerCase().includes(q));
    // Venues that take the guests come first; within a group the directory's order stays.
    return [...found].sort((a, b) => FIT_ORDER[fitOf(a, layout, maxGuests)] - FIT_ORDER[fitOf(b, layout, maxGuests)]);
  }, [venues, query, layout, maxGuests]);

  function close() {
    onOpenChange(false);
    setQuery("");
    requestAnimationFrame(() => triggerRef.current?.focus());
  }

  return (
    <div data-field={name} className="flex flex-col gap-2">
      <span className="text-[13px] font-medium text-foreground" id={`${name}-venue-label`}>Venue</span>
      <div
        className={cn(
          "flex flex-wrap items-center justify-between gap-x-3 gap-y-2 rounded-lg border px-3 py-2.5",
          error ? "border-danger" : "border-border",
        )}
      >
        <div className="flex min-w-0 items-start gap-2">
          <MapPin aria-hidden className="mt-0.5 size-4 shrink-0 text-muted-foreground" />
          {selected ? (
            <div className="min-w-0">
              <p className="truncate text-[13.5px] font-semibold text-foreground">
                {selected.name}
                {selected.city && <span className="font-normal text-muted-foreground"> · {selected.city}</span>}
              </p>
              <p className="text-[12.5px] text-muted-foreground tabular-nums">{capacityLine(selected)}</p>
            </div>
          ) : (
            <p className="text-[13px] text-muted-foreground">Belum ada venue: tamu belum dicek ke kapasitas</p>
          )}
        </div>
        {!disabled && (
          <div className="flex shrink-0 gap-2">
            {selected && !open && (
              <Button type="button" variant="ghost" size="sm" onClick={onClear}>
                Lepas venue
              </Button>
            )}
            <Button
              ref={triggerRef}
              type="button"
              variant="secondary"
              size="sm"
              aria-expanded={open}
              aria-describedby={error ? errorId : undefined}
              aria-invalid={error ? true : undefined}
              onClick={() => (open ? close() : onOpenChange(true))}
            >
              {open ? "Tutup daftar" : selected ? "Ganti venue" : "Pilih venue"}
            </Button>
          </div>
        )}
      </div>
      {error && (
        <span id={errorId} role="alert" className="flex items-start gap-1 text-xs font-medium text-danger-text">
          <CircleAlert aria-hidden className="mt-px size-3.5 shrink-0" />
          {error}
        </span>
      )}

      {open && (
        <div
          className="flex flex-col gap-2 rounded-lg border border-border bg-surface-muted p-2"
          onKeyDown={(e) => {
            if (e.key === "Escape") {
              // Closes the list only, not the drawer it sits in.
              e.stopPropagation();
              close();
            }
          }}
        >
          <div className="relative">
            <Search aria-hidden className="pointer-events-none absolute top-1/2 left-2.5 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              ref={searchRef}
              type="search"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder="Cari nama atau kota venue"
              aria-label="Cari venue"
              className="bg-card pl-8"
            />
          </div>
          {status === "loading" && <p className="px-2 py-3 text-[13px] text-muted-foreground">Memuat venue…</p>}
          {status === "error" && (
            <div className="flex items-center justify-between gap-3 px-2 py-2 text-[13px] text-danger-text">
              Daftar venue belum bisa dimuat.
              <Button type="button" size="sm" variant="secondary" onClick={onRetry}>
                Coba lagi
              </Button>
            </div>
          )}
          {status === "ready" && matches.length === 0 && (
            <p className="px-2 py-3 text-[13px] text-muted-foreground">
              {venues.length === 0 ? "Belum ada venue aktif di direktori." : "Tidak ada venue yang cocok dengan pencarian ini."}
            </p>
          )}
          {status === "ready" && matches.length > 0 && (
            <ul aria-label="Venue di direktori" className="flex max-h-72 flex-col gap-0.5 overflow-y-auto">
              {matches.slice(0, SHOWN).map((v) => {
                const fit = fitOf(v, layout, maxGuests);
                const isSelected = selected?.venueId === v.venueId;
                const limit = capacityFor(v, layout);
                const layoutWord = EVENT_LAYOUT_LABELS[layout].toLowerCase();
                return (
                  <li key={v.venueId}>
                    <button
                      type="button"
                      aria-pressed={isSelected}
                      onClick={() => {
                        onPick(v.venueId);
                        close();
                      }}
                      className={cn(
                        "flex w-full items-start justify-between gap-3 rounded-md px-2.5 py-2 text-left transition-colors hover:bg-card",
                        "focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-solid focus-visible:outline-ring",
                        isSelected && "bg-card",
                      )}
                    >
                      <span className="min-w-0">
                        <span className="block truncate text-[13.5px] font-medium text-foreground">
                          {v.name}
                          {v.city && <span className="font-normal text-muted-foreground"> · {v.city}</span>}
                        </span>
                        <span className="block text-[12.5px] text-muted-foreground tabular-nums">{capacityLine(v)}</span>
                      </span>
                      <span className="shrink-0 pt-0.5 text-[12px] font-medium">
                        {isSelected ? (
                          <span className="flex items-center gap-1 text-primary">
                            <Check aria-hidden className="size-3.5" /> Dipilih
                          </span>
                        ) : fit === "over" && limit !== null ? (
                          <span className="flex items-center gap-1 text-warning-text">
                            <CircleAlert aria-hidden className="size-3.5" /> Tidak muat {formatCount(maxGuests)} tamu
                          </span>
                        ) : fit === "missing" ? (
                          <span className="text-muted-foreground">Belum ada kapasitas {layoutWord}</span>
                        ) : fit === "fits" ? (
                          <span className="text-success-text">Muat</span>
                        ) : null}
                      </span>
                    </button>
                  </li>
                );
              })}
            </ul>
          )}
          {status === "ready" && matches.length > SHOWN && (
            <p className="px-2 pb-1 text-[12px] text-muted-foreground">
              Menampilkan {SHOWN} dari {matches.length} venue. Ketik nama untuk mempersempit.
            </p>
          )}
        </div>
      )}
    </div>
  );
}
