import { useEffect, useMemo, useRef, useState, type ReactNode } from "react";
import { Link, useNavigate } from "react-router-dom";
import { ArrowLeft, Plus, Trash2 } from "lucide-react";
import { Card, CardContent } from "@/shared/components/ui/Card";
import { Button } from "@/shared/components/ui/Button";
import { Badge } from "@/shared/components/ui/Badge";
import { Input, Textarea, Select, Field } from "@/shared/components/ui/Input";
import { ConfirmDialog } from "@/shared/components/ui/ConfirmDialog";
import { IconActionButton } from "@/shared/components/ui/IconActionButton";
import { useAllProjects } from "@/shared/stores/usePermissionStore";
import { useAuthStore } from "@/shared/stores/useAuthStore";
import { toast } from "@/shared/stores/useToastStore";
import { getApiFieldErrors, getApiErrorMessage } from "@/shared/lib/api-error";
import { newIdempotencyKey } from "@/shared/lib/direct-upload";
import { formatDate, todayISO } from "@/shared/lib/formatters";
import { useZodForm, focusField } from "@/shared/forms/useZodForm";
import { FormErrorSummary } from "@/shared/forms/FormErrorSummary";
import { projectSchema } from "@/modules/projects/schemas/project.schema";
import { useProjectStore } from "@/modules/projects/stores/useProjectStore";
import { useStaffStore } from "@/modules/users/stores/useStaffStore";
import {
  EventFields, eventSchema, useEventPickers, useServerRefusal, emptyEventDraft, formatEventDay, toEventInput, venueLookup,
  type EventFormValues,
} from "@/modules/rincian";
import { ROUTE_PATHS } from "@/app/routes/route-paths";

// Step 1 owns these. The date is the main event's (step 2) and the status is always Draf on create.
const step1Schema = projectSchema.pick({
  name: true, brideName: true, groomName: true, prepStartDate: true, picStaffId: true, picSalesStaffId: true,
  description: true,
});
const STEP_1_KEYS = new Set(Object.keys(step1Schema.shape));

const LABELS: Record<string, string> = {
  brideName: "Nama mempelai wanita", groomName: "Nama mempelai pria", name: "Nama proyek",
  prepStartDate: "Tanggal booking",
};

const EVENT_FIELD_LABELS: Record<string, string> = {
  eventKindId: "jenis acara", eventDate: "tanggal", startTime: "jam mulai", endTime: "jam selesai", venueId: "venue",
  layout: "tata letak", sessions: "sesi",
};

const FALLBACK = "Proyek belum dibuat. Periksa koneksi Anda, lalu coba lagi.";

// The API's name of the events list: its errors come as "events.0.eventDate", the form names them "events[0].eventDate".
const EVENTS_KEY = "events";
const eventPrefix = (i: number) => `${EVENTS_KEY}[${i}].`;

/** "Acara 2, sesi 1" for a key such as "events[1].sessions[0].guests": the label of a summary line. */
function eventKeyLabel(key: string): string {
  const m = /^events\[(\d+)\]\.?(.*)$/.exec(key);
  if (!m) return "";
  const where = `Acara ${Number(m[1]) + 1}`;
  const session = /^sessions\[(\d+)\]/.exec(m[2]);
  if (session) return `${where}, sesi ${Number(session[1]) + 1}`;
  const field = EVENT_FIELD_LABELS[m[2]];
  return field ? `${where}, ${field}` : where;
}

/** Zod's path as the form names it: ["sessions", 1, "guests"] → "sessions[1].guests". */
function pathKey(path: Array<string | number>): string {
  return path.map((p, i) => (typeof p === "number" ? `[${p}]` : i === 0 ? p : `.${p}`)).join("");
}

/** What the browser can already say about the events, keyed as the server names them ("events[0].eventDate"). */
function validateEvents(events: EventFormValues[]): Record<string, string> {
  const out: Record<string, string> = {};
  events.forEach((e, i) => {
    const parsed = eventSchema.safeParse(e);
    if (parsed.success) return;
    for (const issue of parsed.error.issues) out[eventPrefix(i) + pathKey(issue.path)] ??= issue.message;
  });
  return out;
}

/** The part of `errors` that belongs to event `i`, with its keys relative to that event. */
function errorsOfEvent(errors: Record<string, string>, i: number): Record<string, string> {
  const prefix = eventPrefix(i);
  const out: Record<string, string> = {};
  for (const [k, v] of Object.entries(errors)) if (k.startsWith(prefix)) out[k.slice(prefix.length)] = v;
  return out;
}

/**
 * Creating a project takes two screens (DESAIN A6): 1) who is marrying, the booking date and the PIC; 2) its events, at
 * least one, each with a kind and a date (venue, layout and sessions may wait). The package and the contract value
 * come from the rincian, built afterwards on Acara & rincian (PLAN §2.2). The project and its events
 * are saved by ONE request that creates everything or nothing, so there is no "project made, events failed" state.
 * The payment schedule is set afterwards on Keuangan. A project is always created as Draf.
 */
export default function ProjectCreatePage() {
  const navigate = useNavigate();
  const createProject = useProjectStore((s) => s.createProject);
  const staffList = useStaffStore((s) => s.staffSummaries);
  const fetchStaff = useStaffStore((s) => s.fetchStaffSummaries);
  const currentStaffId = useAuthStore((s) => s.currentStaffId);
  const assignedOnly = !useAllProjects();

  const [step, setStep] = useState<1 | 2>(1);
  const form = useZodForm({
    schema: step1Schema,
    initial: {
      name: "", brideName: "", groomName: "", prepStartDate: todayISO(), picStaffId: "",
      picSalesStaffId: assignedOnly ? currentStaffId : "", description: "",
    },
  });
  const { values, errors } = form;
  const nameTouched = useRef(false);

  const [events, setEvents] = useState<EventFormValues[]>(() => [emptyEventDraft()]);
  // Client messages show once "Buat proyek" was pressed, and follow every change from then on.
  const [attempted, setAttempted] = useState(false);
  const refusal = useServerRefusal(events);
  // A past event date is allowed (recording a project already held) but is confirmed once per set of dates (EP-10).
  const [confirmingPast, setConfirmingPast] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);
  const formRef = useRef<HTMLFormElement>(null);
  const step2Ref = useRef<HTMLDivElement>(null);

  // One Idempotency-Key per distinct request: retrying the same one after a lost answer cannot create two projects.
  const keyRef = useRef(newIdempotencyKey());
  const requestFingerprint = JSON.stringify([values, events]);
  useEffect(() => {
    keyRef.current = newIdempotencyKey();
  }, [requestFingerprint]);

  useEffect(() => {
    void fetchStaff();
  }, [fetchStaff]);

  function set<K extends keyof typeof values & string>(key: K, value: (typeof values)[K]) {
    form.set(key, value);
    // Suggest a name from the couple until the user types their own.
    if ((key === "brideName" || key === "groomName") && !nameTouched.current) {
      const next = { ...values, [key]: value };
      if (next.brideName && next.groomName) form.set("name", `${next.brideName} & ${next.groomName} Wedding`);
    }
  }

  const goToStep2 = form.handleSubmit(
    () => {
      setStep(2);
      window.scrollTo({ top: 0 });
    },
    { root: () => formRef.current },
  );

  const clientErrors = useMemo(() => (attempted ? validateEvents(events) : {}), [attempted, events]);
  const eventErrors = { ...clientErrors, ...(refusal.active?.fields ?? {}) };
  const eventErrorList = Object.entries(eventErrors)
    .filter(([k]) => k.startsWith(EVENTS_KEY))
    .map(([key, message]) => ({ key, message }));
  const eventLabels = Object.fromEntries(eventErrorList.map((e) => [e.key, eventKeyLabel(e.key)]));
  const pastEvents = events.filter((e) => e.eventDate !== "" && e.eventDate < todayISO());

  function focusFirst(keys: string[]) {
    requestAnimationFrame(() => {
      if (keys.length >= 3) document.getElementById("form-error-summary")?.focus();
      else if (keys.length > 0) focusField(step2Ref.current, keys[0]);
    });
  }

  async function handleCreate(pastConfirmed = false) {
    setAttempted(true);
    setSubmitError(null);
    const problems = validateEvents(events);
    if (Object.keys(problems).length > 0) {
      focusFirst(Object.keys(problems));
      return;
    }
    if (pastEvents.length > 0 && !pastConfirmed) {
      setConfirmingPast(true);
      return;
    }
    const sent = events;
    setSubmitting(true);
    try {
      const project = await createProject(
        { ...values, eventDate: "", status: "Draft" },
        { events: sent.map(toEventInput) },
        { idempotencyKey: keyRef.current },
      );
      toast.success("Proyek dibuat");
      navigate(ROUTE_PATHS.projectDetail(project.id, "acara-rincian"));
    } catch (err) {
      // Nothing was created, so everything typed stays and the person can fix it and press again.
      const keys = Object.keys(getApiFieldErrors(err));
      const step1Keys = keys.filter((k) => STEP_1_KEYS.has(k));
      // Messages about the events stay on their inputs, also after a detour to step 1.
      refusal.record(err, sent, FALLBACK);
      if (step1Keys.length > 0) {
        form.applyServerError(err, FALLBACK, (k) => STEP_1_KEYS.has(k));
        setStep(1);
        requestAnimationFrame(() => focusField(formRef.current, step1Keys[0]));
        return;
      }
      const raw = getApiFieldErrors(err);
      const onEvents = keys.filter((k) => k.startsWith(`${EVENTS_KEY}.`));
      const stray = keys.find((k) => !k.startsWith(`${EVENTS_KEY}.`) && !STEP_1_KEYS.has(k));
      if (onEvents.length > 0) {
        setSubmitError(stray ? raw[stray] : onEvents.length >= 3 ? null : "Perbaiki isian yang ditandai, lalu buat proyek lagi.");
        focusFirst(onEvents.map((k) => k.replace(/\.(\d+)(?=\.|$)/g, "[$1]")));
      } else {
        setSubmitError(stray ? raw[stray] : getApiErrorMessage(err, FALLBACK));
      }
    } finally {
      setSubmitting(false);
    }
  }

  const staffOptions = staffList.map((s) => (
    <option key={s.id} value={s.id}>{s.name} — {s.title}</option>
  ));
  const selfName = staffList.find((s) => s.id === currentStaffId)?.name ?? "...";
  const invalid = (key: keyof typeof values) => (errors[key] ? true : undefined);

  return (
    <div className="mx-auto flex w-full max-w-3xl flex-col gap-5">
      <div>
        <Link to={ROUTE_PATHS.projects} className="inline-flex items-center gap-1.5 text-[13px] font-medium text-text-secondary hover:text-text-primary">
          <ArrowLeft className="h-3.5 w-3.5" /> Kembali ke daftar proyek
        </Link>
        <h1 className="mt-2 text-xl font-bold text-text-primary">Proyek baru</h1>
        <p className="mt-1 text-[13px] text-text-secondary" aria-live="polite">
          Langkah {step} dari 2: {step === 1 ? "Pasangan dan PIC" : "Acara"}
        </p>
      </div>

      <Card>
        <CardContent className="py-5">
          {step === 1 ? (
            <form
              ref={formRef}
              noValidate
              onSubmit={(e) => void goToStep2(e)}
              className="grid grid-cols-1 gap-4 sm:grid-cols-2"
            >
              <div className="sm:col-span-2 empty:hidden">
                <FormErrorSummary errors={form.errorList} formError={form.formError} labels={LABELS} root={() => formRef.current} />
              </div>
              <Field label="Nama mempelai wanita" required name="brideName" error={errors.brideName}>
                <Input value={values.brideName} onChange={(e) => set("brideName", e.target.value)} onBlur={() => form.blur("brideName")} aria-invalid={invalid("brideName")} autoFocus />
              </Field>
              <Field label="Nama mempelai pria" required name="groomName" error={errors.groomName}>
                <Input value={values.groomName} onChange={(e) => set("groomName", e.target.value)} onBlur={() => form.blur("groomName")} aria-invalid={invalid("groomName")} />
              </Field>
              <div className="sm:col-span-2">
                <Field label="Nama proyek" required name="name" error={errors.name} hint="Terisi otomatis dari nama pasangan; boleh diubah.">
                  <Input
                    value={values.name}
                    onChange={(e) => {
                      nameTouched.current = true;
                      set("name", e.target.value);
                    }}
                    aria-invalid={invalid("name")}
                    placeholder="cth. Aurelia & Bagas Wedding"
                  />
                </Field>
              </div>
              <Field label="Tanggal booking" required name="prepStartDate" error={errors.prepStartDate}>
                <Input type="date" value={values.prepStartDate} onChange={(e) => set("prepStartDate", e.target.value)} aria-invalid={invalid("prepStartDate")} />
              </Field>
              <Field label="Penanggung jawab WO (PIC Wedding Planner)" name="picStaffId" error={errors.picStaffId}>
                <Select value={values.picStaffId} onChange={(e) => set("picStaffId", e.target.value)}>
                  <option value="">Belum ditugaskan</option>
                  {staffOptions}
                </Select>
              </Field>
              <Field label="PIC Sales" name="picSalesStaffId" error={errors.picSalesStaffId}>
                {assignedOnly ? (
                  <Input value={selfName} disabled />
                ) : (
                  <Select value={values.picSalesStaffId} onChange={(e) => set("picSalesStaffId", e.target.value)}>
                    <option value="">Belum ditugaskan</option>
                    {staffOptions}
                  </Select>
                )}
              </Field>
              <div className="sm:col-span-2">
                <Field label="Catatan proyek" name="description">
                  <Textarea rows={3} value={values.description} onChange={(e) => set("description", e.target.value)} />
                </Field>
              </div>
              <div className="flex justify-end gap-2 sm:col-span-2">
                <Button variant="secondary" type="button" onClick={() => navigate(ROUTE_PATHS.projects)}>Batal</Button>
                <Button type="submit">Lanjut</Button>
              </div>
            </form>
          ) : (
            <div ref={step2Ref} className="flex flex-col gap-6">
              <EventsStep
                events={events}
                onChange={setEvents}
                errors={eventErrors}
                summary={
                  <FormErrorSummary errors={eventErrorList} formError={submitError} labels={eventLabels} root={() => step2Ref.current} />
                }
              />
              <div className="flex justify-between gap-2 border-t border-border-light pt-5">
                <Button variant="secondary" onClick={() => setStep(1)} disabled={submitting}>Kembali</Button>
                <Button onClick={() => void handleCreate()} disabled={submitting}>{submitting ? "Membuat proyek..." : "Buat proyek"}</Button>
              </div>
            </div>
          )}
        </CardContent>
      </Card>

      <ConfirmDialog
        open={confirmingPast}
        title="Tanggal acara sudah lewat"
        confirmLabel="Buat proyek"
        tone="primary"
        onConfirm={() => {
          setConfirmingPast(false);
          void handleCreate(true);
        }}
        onClose={() => setConfirmingPast(false)}
      >
        <p>
          {pastEvents.length === 1
            ? `Tanggal acara ${formatDate(pastEvents[0].eventDate)} sudah lewat.`
            : `${pastEvents.length} acara bertanggal yang sudah lewat.`}{" "}
          Lanjutkan bila Anda memang mencatat proyek yang sudah berlangsung; bila salah ketik, pilih Batal lalu perbaiki
          tanggalnya.
        </p>
      </ConfirmDialog>
    </div>
  );
}

interface EventsStepProps {
  events: EventFormValues[];
  onChange: (next: EventFormValues[]) => void;
  /** Messages keyed "events[i]...." (client and server). */
  errors: Record<string, string>;
  summary: ReactNode;
}

/** Step 2: the events. The kinds and the venue directory load only once this step shows. */
function EventsStep({ events, onChange, errors, summary }: EventsStepProps) {
  const pickers = useEventPickers();
  const venueFor = useMemo(() => venueLookup([], pickers.venues), [pickers.venues]);
  // Only one venue list is open at a time.
  const [openPicker, setOpenPicker] = useState<number | null>(null);
  // A steady key per event, so removing one does not hand its open list or typed search to the next.
  const nextId = useRef(events.length);
  const [ids, setIds] = useState(() => events.map((_, i) => i));

  function update(i: number, next: EventFormValues) {
    onChange(events.map((e, j) => (j === i ? next : e)));
  }

  function remove(i: number) {
    onChange(events.filter((_, j) => j !== i));
    setIds((all) => all.filter((_, j) => j !== i));
    setOpenPicker(null);
  }

  function add() {
    onChange([...events, emptyEventDraft()]);
    setIds((all) => [...all, nextId.current++]);
  }

  return (
    <section aria-labelledby="events-heading" className="flex flex-col gap-5">
      <div>
        <h2 id="events-heading" className="text-[15px] font-semibold text-text-primary">Acara</h2>
        <p className="mt-0.5 text-[13px] text-text-secondary">
          Isi minimal satu acara dengan jenis dan tanggalnya. Venue, tata letak, dan tamu boleh menyusul.
        </p>
      </div>
      <div className="empty:hidden">{summary}</div>
      {pickers.kindStatus === "error" && (
        <div role="alert" className="flex items-center justify-between gap-3 rounded-md border border-danger/30 bg-danger-soft px-3.5 py-2.5 text-[13px] text-danger-text">
          Daftar jenis acara belum bisa dimuat.
          <Button type="button" size="sm" variant="secondary" onClick={pickers.retryKinds}>
            Coba lagi
          </Button>
        </div>
      )}
      <ol className="flex flex-col gap-4">
        {events.map((e, i) => {
          const kindName = pickers.kinds.find((k) => k.id === e.eventKindId)?.name;
          const heading = kindName ?? `Acara ${i + 1}`;
          return (
            <li key={ids[i] ?? `extra-${i}`} className="rounded-lg border border-border-light p-4 sm:p-5">
              <div className="mb-4 flex items-start justify-between gap-3">
                <div className="min-w-0">
                  <h3 className="flex flex-wrap items-center gap-2 text-[14px] font-semibold text-text-primary">
                    {heading}
                    {i === 0 && (
                      <Badge tone="brand" dot={false}>
                        Acara utama
                      </Badge>
                    )}
                  </h3>
                  <p className="mt-0.5 text-[13px] text-text-secondary">
                    {i === 0
                      ? "Tanggalnya menjadi Hari H proyek. Acara utama bisa diganti nanti di tab Acara & rincian."
                      : e.eventDate
                        ? formatEventDay(e.eventDate)
                        : "Tanggal belum dipilih"}
                  </p>
                </div>
                {events.length > 1 && <IconActionButton icon={Trash2} label={`Hapus ${heading}`} tone="danger" onClick={() => remove(i)} />}
              </div>
              <EventFields
                value={e}
                onChange={(next) => update(i, next)}
                errors={errorsOfEvent(errors, i)}
                prefix={eventPrefix(i)}
                kinds={pickers.kinds}
                venues={pickers.venues}
                venueStatus={pickers.venueStatus}
                onRetryVenues={pickers.retryVenues}
                venueFor={venueFor}
                pickerOpen={openPicker === i}
                onPickerOpenChange={(open) => setOpenPicker(open ? i : null)}
              />
            </li>
          );
        })}
      </ol>
      <div>
        <Button type="button" variant="secondary" icon={<Plus className="size-3.5" />} onClick={add}>
          Tambah acara
        </Button>
      </div>
    </section>
  );
}
