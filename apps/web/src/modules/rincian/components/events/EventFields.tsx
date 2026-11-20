import { useId } from "react";
import { CircleAlert, Plus, Trash2, TriangleAlert } from "lucide-react";
import { Button } from "@/shared/components/ui/Button";
import { Field, Input, Select } from "@/shared/components/ui/Input";
import { IconActionButton } from "@/shared/components/ui/IconActionButton";
import { cn } from "@/shared/lib/cn";
import { todayISO } from "@/shared/lib/formatters";
import { EVENT_LAYOUT_LABELS } from "@/shared/lib/labels";
import { EVENT_LAYOUT_OPTIONS, type EventFormValues } from "@/modules/rincian/schemas/event.schema";
import type { EventKind } from "@/modules/event-kinds/types";
import type { EventLayout } from "@/modules/rincian/types";
import { VenuePicker } from "@/modules/rincian/components/events/VenuePicker";
import {
  capacityFor, emptySession, formatCount, maxSessionGuests, otherLayout, totalGuests, type VenueCaps,
} from "@/modules/rincian/components/events/event-display";

export interface EventFieldsProps {
  value: EventFormValues;
  onChange: (next: EventFormValues) => void;
  /** A field was left: its key relative to the event ("eventDate", "sessions[1].guests"). */
  onBlur?: (key: string) => void;
  /** Messages by key relative to the event. */
  errors: Record<string, string>;
  /** Put before every key in `data-field`, so an error summary of a bigger form can jump here ("events[0]."). */
  prefix?: string;
  kinds: EventKind[];
  venues: VenueCaps[];
  venueStatus: "loading" | "ready" | "error";
  onRetryVenues: () => void;
  venueFor: (venueId: string) => VenueCaps | null;
  pickerOpen: boolean;
  onPickerOpenChange: (open: boolean) => void;
  /** The date the event had when the form opened: only a date moved into the past is pointed out. */
  initialDate?: string;
  /** The date may stay empty (a quotation's event before the couple has picked a day). */
  dateOptional?: boolean;
}

/**
 * One event's fields (DESAIN §5.1): kind, date and times, venue with its capacities, layout, and the sessions with the
 * capacity left per session, worked out on screen from the numbers shown. Typing over the limit is never cut off: the
 * refusal is the server's, on save.
 */
export function EventFields({
  value, onChange, onBlur, errors, prefix = "", kinds, venues, venueStatus, onRetryVenues, venueFor, pickerOpen,
  onPickerOpenChange, initialDate, dateOptional = false,
}: EventFieldsProps) {
  const uid = useId();
  const venue = venueFor(value.venueId);
  const limit = capacityFor(venue, value.layout);
  const maxGuests = maxSessionGuests(value.sessions);
  const movedToPast = value.eventDate !== "" && value.eventDate !== initialDate && value.eventDate < todayISO();
  // A kind switched off in Katalog still shows on the event that uses it.
  const kindOptions = kinds.filter((k) => k.isActive || k.id === value.eventKindId);

  function patch(next: Partial<EventFormValues>) {
    onChange({ ...value, ...next });
  }

  function pickVenue(venueId: string) {
    const v = venueFor(venueId);
    // A venue with a number only for the other layout takes that layout: the event could not be saved otherwise.
    const layout: EventLayout =
      v && capacityFor(v, value.layout) === null && capacityFor(v, otherLayout(value.layout)) !== null ? otherLayout(value.layout) : value.layout;
    patch({ venueId, layout });
  }

  function setSession(i: number, next: Partial<EventFormValues["sessions"][number]>) {
    patch({ sessions: value.sessions.map((s, j) => (j === i ? { ...s, ...next } : s)) });
  }

  return (
    <div className="flex flex-col gap-5">
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <div className="sm:col-span-2">
          <Field label="Jenis acara" required name={`${prefix}eventKindId`} error={errors.eventKindId}>
            <Select
              value={value.eventKindId}
              aria-label="Jenis acara"
              placeholder="Pilih jenis acara"
              onChange={(e) => {
                patch({ eventKindId: e.target.value });
                onBlur?.("eventKindId");
              }}
            >
              {kindOptions.map((k) => (
                <option key={k.id} value={k.id}>{k.name}</option>
              ))}
            </Select>
          </Field>
        </div>
        <div className="sm:col-span-2">
          <Field
            label="Tanggal"
            required={!dateOptional}
            name={`${prefix}eventDate`}
            error={errors.eventDate}
            hint={
              movedToPast
                ? "Tanggal ini sudah lewat. Pastikan memang benar sebelum menyimpan."
                : dateOptional && !value.eventDate
                  ? "Boleh dikosongkan bila tanggalnya belum pasti."
                  : undefined
            }
          >
            <Input type="date" value={value.eventDate} onChange={(e) => patch({ eventDate: e.target.value })} onBlur={() => onBlur?.("eventDate")} className="sm:max-w-56" />
          </Field>
        </div>
        <Field label="Mulai" name={`${prefix}startTime`} error={errors.startTime}>
          <Input type="time" value={value.startTime} onChange={(e) => patch({ startTime: e.target.value })} onBlur={() => onBlur?.("startTime")} />
        </Field>
        <Field label="Selesai" name={`${prefix}endTime`} error={errors.endTime}>
          <Input type="time" value={value.endTime} onChange={(e) => patch({ endTime: e.target.value })} onBlur={() => onBlur?.("endTime")} />
        </Field>
      </div>

      <VenuePicker
        name={`${prefix}venueId`}
        selected={venue}
        venues={venues}
        status={venueStatus}
        onRetry={onRetryVenues}
        layout={value.layout}
        maxGuests={maxGuests}
        onPick={pickVenue}
        onClear={() => patch({ venueId: "" })}
        open={pickerOpen}
        onOpenChange={onPickerOpenChange}
        error={errors.venueId}
      />

      <fieldset data-field={`${prefix}layout`} className="flex flex-col gap-2">
        <legend className="mb-1.5 text-[13px] font-medium text-foreground">Tata letak</legend>
        <div className="flex flex-wrap gap-2">
          {EVENT_LAYOUT_OPTIONS.map((layout) => {
            const n = capacityFor(venue, layout);
            const checked = value.layout === layout;
            return (
              <label
                key={layout}
                className={cn(
                  "flex cursor-pointer items-center gap-2.5 rounded-lg border px-3 py-2 text-[13.5px] transition-colors",
                  "has-[:focus-visible]:outline-2 has-[:focus-visible]:outline-offset-2 has-[:focus-visible]:outline-solid has-[:focus-visible]:outline-ring",
                  checked ? "border-primary bg-primary/5" : "border-border",
                )}
              >
                <input
                  type="radio"
                  name={`${uid}-layout`}
                  className="size-4 accent-primary"
                  checked={checked}
                  onChange={() => patch({ layout })}
                />
                <span className="font-medium text-foreground">{EVENT_LAYOUT_LABELS[layout]}</span>
                {venue && (
                  <span className="text-muted-foreground tabular-nums">
                    {n !== null ? `muat ${formatCount(n)}` : "belum diisi"}
                  </span>
                )}
              </label>
            );
          })}
        </div>
        {errors.layout && <p role="alert" className="text-xs font-medium text-danger-text">{errors.layout}</p>}
        {venue && limit === null && (
          <p className="flex items-start gap-1.5 text-[12.5px] text-warning-text">
            <TriangleAlert aria-hidden className="mt-px size-3.5 shrink-0" />
            {venue.name} belum punya kapasitas {EVENT_LAYOUT_LABELS[value.layout].toLowerCase()}. Isi di Direktori › Venue, atau pilih tata letak lain.
          </p>
        )}
      </fieldset>

      <SessionsEditor
        sessions={value.sessions}
        limit={limit}
        hasVenue={venue !== null}
        prefix={prefix}
        errors={errors}
        onBlur={onBlur}
        onSet={setSession}
        onAdd={() => patch({ sessions: [...value.sessions, emptySession()] })}
        onRemove={(i) => patch({ sessions: value.sessions.filter((_, j) => j !== i) })}
      />
    </div>
  );
}

interface SessionsEditorProps {
  sessions: EventFormValues["sessions"];
  limit: number | null;
  hasVenue: boolean;
  prefix: string;
  errors: Record<string, string>;
  onBlur?: (key: string) => void;
  onSet: (i: number, next: Partial<EventFormValues["sessions"][number]>) => void;
  onAdd: () => void;
  onRemove: (i: number) => void;
}

const ROW = "grid grid-cols-3 gap-x-2 gap-y-1.5 sm:grid-cols-[minmax(0,1fr)_6.5rem_6.5rem_5rem_8.5rem_2rem] sm:items-start";

function SessionsEditor({ sessions, limit, hasVenue, prefix, errors, onBlur, onSet, onAdd, onRemove }: SessionsEditorProps) {
  const uid = useId();
  const total = totalGuests(sessions);
  return (
    <fieldset className="flex flex-col gap-2" data-field={`${prefix}sessions`}>
      <legend className="mb-1.5 text-[13px] font-medium text-foreground">Sesi</legend>
      <div aria-hidden className={cn(ROW, "hidden text-[12px] font-medium text-muted-foreground sm:grid")}>
        <span>Nama</span>
        <span>Mulai</span>
        <span>Selesai</span>
        <span>Tamu</span>
        <span>Sisa kapasitas</span>
        <span />
      </div>
      <ol className="flex flex-col gap-3 sm:gap-2">
        {sessions.map((s, i) => {
          const n = i + 1;
          const key = (field: string) => `sessions[${i}].${field}`;
          const guestsError = errors[key("guests")];
          const rowError = errors[key("label")] ?? errors[key("startTime")] ?? errors[key("endTime")];
          const left = limit !== null ? limit - s.guests : null;
          return (
            <li key={i} className="flex flex-col gap-1">
              <div className={ROW}>
                <span className="col-span-3 sm:col-span-1" data-field={`${prefix}${key("label")}`}>
                  <Input
                    value={s.label}
                    placeholder={`Sesi ${n}`}
                    aria-label={`Nama sesi ${n}`}
                    maxLength={60}
                    aria-invalid={errors[key("label")] ? true : undefined}
                    onChange={(e) => onSet(i, { label: e.target.value })}
                    onBlur={() => onBlur?.(key("label"))}
                  />
                </span>
                <span data-field={`${prefix}${key("startTime")}`}>
                  <Input
                    type="time"
                    value={s.startTime}
                    aria-label={`Jam mulai sesi ${n}`}
                    aria-invalid={errors[key("startTime")] ? true : undefined}
                    onChange={(e) => onSet(i, { startTime: e.target.value })}
                    onBlur={() => onBlur?.(key("startTime"))}
                  />
                </span>
                <span data-field={`${prefix}${key("endTime")}`}>
                  <Input
                    type="time"
                    value={s.endTime}
                    aria-label={`Jam selesai sesi ${n}`}
                    aria-invalid={errors[key("endTime")] ? true : undefined}
                    onChange={(e) => onSet(i, { endTime: e.target.value })}
                    onBlur={() => onBlur?.(key("endTime"))}
                  />
                </span>
                <span data-field={`${prefix}${key("guests")}`}>
                  <Input
                    inputMode="numeric"
                    autoComplete="off"
                    value={s.guests > 0 ? String(s.guests) : ""}
                    placeholder="0"
                    aria-label={`Tamu sesi ${n}`}
                    aria-required
                    aria-invalid={guestsError ? true : undefined}
                    aria-describedby={guestsError ? `${uid}-guests-${i}` : `${uid}-left-${i}`}
                    className="text-right tabular-nums"
                    onChange={(e) => onSet(i, { guests: Number(e.target.value.replace(/\D/g, "").slice(0, 6)) || 0 })}
                    onBlur={() => onBlur?.(key("guests"))}
                  />
                </span>
                <span id={`${uid}-left-${i}`} className="col-span-2 flex min-h-(--control-h) items-center text-[12.5px] tabular-nums sm:col-span-1">
                  {left === null ? (
                    <span className="text-muted-foreground">{hasVenue ? "—" : ""}</span>
                  ) : left >= 0 ? (
                    <span className="text-muted-foreground">{left === 0 ? "Penuh" : `${formatCount(left)} lagi`}</span>
                  ) : (
                    <span className="flex items-center gap-1 font-medium text-warning-text">
                      <TriangleAlert aria-hidden className="size-3.5 shrink-0" />
                      {formatCount(-left)} di atas kapasitas
                    </span>
                  )}
                </span>
                <span className="flex min-h-(--control-h) items-center justify-end">
                  {sessions.length > 1 && <IconActionButton icon={Trash2} label={`Hapus sesi ${n}`} onClick={() => onRemove(i)} />}
                </span>
              </div>
              {(guestsError || rowError) && (
                <p id={`${uid}-guests-${i}`} role="alert" className="flex items-start gap-1 text-xs font-medium text-danger-text">
                  <CircleAlert aria-hidden className="mt-px size-3.5 shrink-0" />
                  {guestsError ?? rowError}
                </p>
              )}
            </li>
          );
        })}
      </ol>
      {errors.sessions && <p role="alert" className="text-xs font-medium text-danger-text">{errors.sessions}</p>}
      <div className="flex flex-wrap items-center justify-between gap-3 pt-1">
        <Button type="button" variant="secondary" size="sm" icon={<Plus className="size-3.5" />} onClick={onAdd}>
          Tambah sesi
        </Button>
        <p className="text-[13px] text-muted-foreground" aria-live="polite">
          Total tamu <span className="font-semibold text-foreground tabular-nums">{formatCount(total)}</span>
        </p>
      </div>
    </fieldset>
  );
}
