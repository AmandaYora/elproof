import { MapPin } from "lucide-react";
import { Button } from "@/shared/components/kit/button";
import { Badge } from "@/shared/components/ui/Badge";
import { EVENT_LAYOUT_LABELS } from "@/shared/lib/labels";
import type { EventVenue, ProjectEvent } from "@/modules/projects/types";

const MONTHS_SHORT = ["Jan", "Feb", "Mar", "Apr", "Mei", "Jun", "Jul", "Agu", "Sep", "Okt", "Nov", "Des"];
const MONTHS = ["Januari", "Februari", "Maret", "April", "Mei", "Juni", "Juli", "Agustus", "September", "Oktober", "November", "Desember"];
const DAYS = ["Minggu", "Senin", "Selasa", "Rabu", "Kamis", "Jumat", "Sabtu"];

function dateParts(iso: string) {
  const [y, m, d] = iso.split("T")[0].split("-").map(Number);
  return { y, m, d, weekday: new Date(y, m - 1, d).getDay() };
}

/** "Sabtu, 13 Maret 2027": the full date a couple tells their guests. */
function fullDate(iso: string): string {
  const { y, m, d, weekday } = dateParts(iso);
  return `${DAYS[weekday]}, ${d} ${MONTHS[m - 1]} ${y}`;
}

/** "08.00–10.00" (PUEBI writes the hour with a dot); one end alone when only one is set; "" for none. */
function timeRange(start: string | null, end: string | null): string {
  const t = (v: string | null) => (v ? v.slice(0, 5).replace(":", ".") : "");
  if (start && end) return `${t(start)}–${t(end)}`;
  if (start) return `Mulai ${t(start)}`;
  if (end) return `Selesai ${t(end)}`;
  return "";
}

/** A map search for the venue (no coordinates are stored): its name, address and city as one query. */
function mapUrl(venue: EventVenue): string {
  const query = [venue.name, venue.address, venue.city].filter(Boolean).join(", ");
  return `https://www.google.com/maps/search/?api=1&query=${encodeURIComponent(query)}`;
}

/**
 * One acara on the portal's Jadwal (DESAIN §6.11): its full date and hours, the venue with its address and a map
 * link, and each session with its hours and guests. The date tile on the left is how a couple scans for "which day".
 */
export function EventCard({ event, showMain }: { event: ProjectEvent; showMain: boolean }) {
  const { m, d } = dateParts(event.eventDate);
  const hours = timeRange(event.startTime, event.endTime);
  const venue = event.venue;
  const place = venue ? [venue.address, venue.city].filter(Boolean).join(", ") : "";
  const headingId = `event-${event.id}`;

  return (
    <article aria-labelledby={headingId} className="overflow-hidden rounded-2xl border border-border bg-card">
      <div className="flex items-start gap-4 p-5">
        <div aria-hidden className="flex w-14 shrink-0 flex-col items-center rounded-xl border border-border bg-background py-2 leading-none">
          <span className="text-[24px] font-extrabold tracking-[-0.02em] text-primary tabular-nums">{d}</span>
          <span className="mt-1 text-[12px] font-medium text-muted-foreground">{MONTHS_SHORT[m - 1]}</span>
        </div>
        <div className="min-w-0 flex-1">
          <div className="flex flex-wrap items-center gap-x-2 gap-y-1">
            <h3 id={headingId} className="text-base font-semibold text-foreground">{event.kindName}</h3>
            {showMain && event.isMain && <Badge tone="brand">Acara utama</Badge>}
          </div>
          <p className="mt-0.5 text-[14px] text-muted-foreground">
            {fullDate(event.eventDate)}
            {hours && <span className="tabular-nums">, {hours}</span>}
          </p>
        </div>
      </div>

      <div className="border-t border-border-light px-5 py-4">
        {venue ? (
          <div className="flex items-start gap-3">
            <MapPin className="mt-0.5 size-5 shrink-0 text-primary" aria-hidden />
            <div className="min-w-0 flex-1">
              <p className="text-[14px] font-semibold text-foreground">{venue.name}</p>
              {place && <p className="mt-0.5 text-[14px] leading-relaxed text-muted-foreground">{place}</p>}
              <Button
                variant="outline"
                size="sm"
                className="mt-3"
                nativeButton={false}
                render={<a href={mapUrl(venue)} target="_blank" rel="noopener noreferrer" aria-label={`Buka peta ${venue.name}`} />}
              >
                Buka peta
              </Button>
            </div>
          </div>
        ) : (
          <p className="flex items-center gap-3 text-[14px] text-muted-foreground">
            <MapPin className="size-5 shrink-0" aria-hidden /> Venue untuk acara ini belum ditentukan.
          </p>
        )}
      </div>

      {event.sessions.length > 0 && (
        <div className="border-t border-border-light px-5 py-4">
          <ul className="flex flex-col gap-2.5">
            {event.sessions.map((s, i) => {
              const sessionHours = timeRange(s.startTime, s.endTime);
              return (
                <li key={s.id} className="flex items-baseline justify-between gap-4 text-[14px]">
                  <span className="min-w-0">
                    <span className="font-medium text-foreground">{s.label || `Sesi ${i + 1}`}</span>
                    {sessionHours && <span className="ml-2 text-muted-foreground tabular-nums">{sessionHours}</span>}
                  </span>
                  <span className="shrink-0 text-foreground tabular-nums">{s.guests.toLocaleString("id-ID")} tamu</span>
                </li>
              );
            })}
          </ul>
          <p className="mt-3 flex items-baseline justify-between gap-4 border-t border-border-light pt-3 text-[13px] text-muted-foreground">
            <span>Tata letak {EVENT_LAYOUT_LABELS[event.layout].toLowerCase()}</span>
            {event.sessions.length > 1 && <span className="font-semibold text-foreground tabular-nums">Total {event.totalGuests.toLocaleString("id-ID")} tamu</span>}
          </p>
        </div>
      )}
    </article>
  );
}
