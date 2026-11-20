import { useMemo, useRef, useState } from "react";
import { CircleAlert } from "lucide-react";
import { Drawer } from "@/shared/components/ui/Drawer";
import { Button } from "@/shared/components/ui/Button";
import { useZodForm, focusField } from "@/shared/forms/useZodForm";
import { FormErrorSummary } from "@/shared/forms/FormErrorSummary";
import { isStaleConflict } from "@/shared/lib/concurrency";
import { EVENT_LAYOUT_LABELS } from "@/shared/lib/labels";
import { toast } from "@/shared/stores/useToastStore";
import { useProjectStore } from "@/modules/projects/stores/useProjectStore";
import type { ProjectEvent } from "@/modules/projects/types";
import {
  EventFields, eventSchema, useEventPickers, useServerRefusal, capacityFor, draftFromEvent, emptyEventDraft, formatCount,
  maxSessionGuests, otherLayout, refusedSessions, splitSession, toDraft, toEventInput, venueLookup, type EventFormValues,
} from "@/modules/rincian";

const FIELD_LABELS: Record<string, string> = {
  eventKindId: "Jenis acara",
  eventDate: "Tanggal",
  startTime: "Mulai",
  endTime: "Selesai",
  venueId: "Venue",
  layout: "Tata letak",
  sessions: "Sesi",
};

interface EventDrawerProps {
  projectId: string;
  /** The event to change, or null to add one. */
  event: ProjectEvent | null;
  onClose: () => void;
}

/**
 * "Tambah acara" / "Ubah acara" (DESAIN §5.1, A4). Loaded on the click that opens it. It exists only while open, so
 * each opening starts from the event as it is now and keeps the version it opened with for If-Match (ADR-0036).
 */
export default function EventDrawer({ projectId, event, onClose }: EventDrawerProps) {
  const createEvent = useProjectStore((s) => s.createEvent);
  const updateEvent = useProjectStore((s) => s.updateEvent);
  const projectEvents = useProjectStore((s) => s.events);
  const pickers = useEventPickers();
  const venueFor = useMemo(() => venueLookup(projectEvents, pickers.venues), [projectEvents, pickers.venues]);

  const [opened] = useState(() => ({ initial: event ? draftFromEvent(event) : emptyEventDraft(), version: event?.updatedAt ?? "" }));
  const form = useZodForm({ schema: eventSchema, initial: opened.initial });
  const draft = toDraft(form.values);
  const refusal = useServerRefusal(draft);
  const [pickerOpen, setPickerOpen] = useState(false);
  const formRef = useRef<HTMLFormElement>(null);
  const panelRef = useRef<HTMLDivElement>(null);

  const errors = { ...form.errors, ...(refusal.active?.fields ?? {}) };

  function change(next: EventFormValues) {
    for (const key of Object.keys(next) as Array<keyof EventFormValues>) {
      if (next[key] !== draft[key]) form.set(key, next[key] as never);
    }
  }

  const submit = form.handleSubmit(
    async (parsed) => {
      const sent = draft;
      try {
        if (event) await updateEvent(projectId, event.id, toEventInput(parsed), opened.version);
        else await createEvent(projectId, toEventInput(parsed));
      } catch (err) {
        // Someone saved this event first: the shared "Data diubah orang lain" dialog offers to reload the page.
        if (isStaleConflict(err)) return;
        if (refusal.record(err, sent, "Acara belum tersimpan. Periksa isian yang ditandai.")) {
          requestAnimationFrame(() => panelRef.current?.focus());
          return;
        }
        throw err;
      }
      toast.success(event ? "Acara disimpan" : "Acara ditambahkan");
      onClose();
    },
    { root: () => formRef.current, fallback: "Acara belum tersimpan. Periksa koneksi Anda, lalu coba lagi." },
  );

  // Ways out of a capacity refusal (DESAIN §5.1): split the session, take the other layout when it fits, or pick another venue.
  const active = refusal.active;
  const over = active ? refusedSessions(Object.keys(active.fields)) : [];
  const venue = venueFor(draft.venueId);
  const other = otherLayout(draft.layout);
  const otherLimit = capacityFor(venue, other);
  const fitsOther = venue !== null && otherLimit !== null && maxSessionGuests(draft.sessions) <= otherLimit;
  const venueRefused = Boolean(active?.fields.venueId);
  const messages = active ? [...new Set(Object.values(active.fields))] : [];

  function split(index: number) {
    change(splitSession(draft, index));
    requestAnimationFrame(() => focusField(formRef.current, `sessions[${index + 1}].startTime`));
  }

  function pickAnotherVenue() {
    setPickerOpen(true);
  }

  return (
    <Drawer
      open
      onClose={onClose}
      busy={form.submitting}
      size="lg"
      title={event ? `Ubah acara: ${event.kindName}` : "Tambah acara"}
      description={event?.isMain ? "Acara utama: tanggalnya menjadi Hari H proyek." : undefined}
      footer={
        <>
          <Button variant="secondary" onClick={onClose} disabled={form.submitting}>
            Batal
          </Button>
          <Button type="submit" form="event-form" disabled={form.submitting}>
            {form.submitting ? "Menyimpan…" : "Simpan acara"}
          </Button>
        </>
      }
    >
      <form id="event-form" ref={formRef} noValidate onSubmit={(e) => void submit(e)} className="flex flex-col gap-5">
        {active && (
          <div
            ref={panelRef}
            tabIndex={-1}
            role="alert"
            className="flex flex-col gap-3 rounded-md border border-danger/30 bg-danger-soft px-4 py-3 text-[13px] text-danger-text outline-none focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-solid focus-visible:outline-danger-text"
          >
            <div className="flex items-start gap-2">
              <CircleAlert aria-hidden className="mt-px size-4 shrink-0" />
              {messages.length === 1 ? (
                <p className="font-medium">{messages[0]}</p>
              ) : (
                <div>
                  <p className="font-semibold">Acara belum tersimpan:</p>
                  <ul className="mt-1 list-disc space-y-0.5 pl-5">
                    {messages.map((m) => (
                      <li key={m}>{m}</li>
                    ))}
                  </ul>
                </div>
              )}
            </div>
            {(over.length > 0 || venueRefused) && (
              <div className="flex flex-wrap gap-2 pl-6">
                {over.length > 0 && (
                  <Button type="button" size="sm" variant="secondary" onClick={() => split(over[0])}>
                    Bagi jadi 2 sesi
                  </Button>
                )}
                {fitsOther && (
                  <Button type="button" size="sm" variant="secondary" onClick={() => change({ ...draft, layout: other })}>
                    Ubah ke {EVENT_LAYOUT_LABELS[other].toLowerCase()} — muat {formatCount(otherLimit ?? 0)}
                  </Button>
                )}
                <Button type="button" size="sm" variant="secondary" onClick={pickAnotherVenue}>
                  Ganti venue
                </Button>
              </div>
            )}
          </div>
        )}
        <FormErrorSummary errors={form.errorList} formError={form.formError} labels={FIELD_LABELS} root={() => formRef.current} />
        {pickers.kindStatus === "error" && (
          <div role="alert" className="flex items-center justify-between gap-3 rounded-md border border-danger/30 bg-danger-soft px-3.5 py-2.5 text-[13px] text-danger-text">
            Daftar jenis acara belum bisa dimuat.
            <Button type="button" size="sm" variant="secondary" onClick={pickers.retryKinds}>
              Coba lagi
            </Button>
          </div>
        )}
        <EventFields
          value={draft}
          onChange={change}
          onBlur={form.blur}
          errors={errors}
          kinds={pickers.kinds}
          venues={pickers.venues}
          venueStatus={pickers.venueStatus}
          onRetryVenues={pickers.retryVenues}
          venueFor={venueFor}
          pickerOpen={pickerOpen}
          onPickerOpenChange={setPickerOpen}
          initialDate={opened.initial.eventDate}
        />
      </form>
    </Drawer>
  );
}
