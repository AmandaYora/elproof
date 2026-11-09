import { useEffect, useRef, useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { ArrowLeft } from "lucide-react";
import { Card, CardContent } from "@/shared/components/ui/Card";
import { Button } from "@/shared/components/ui/Button";
import { Input, Textarea, Select, Field } from "@/shared/components/ui/Input";
import { CurrencyInput } from "@/shared/components/ui/CurrencyInput";
import { useAllProjects } from "@/shared/stores/usePermissionStore";
import { useAuthStore } from "@/shared/stores/useAuthStore";
import { toast } from "@/shared/stores/useToastStore";
import { getApiFieldErrors, getApiErrorMessage } from "@/shared/lib/api-error";
import { newIdempotencyKey } from "@/shared/lib/direct-upload";
import { formatCurrency } from "@/shared/lib/formatters";
import { useZodForm, focusField } from "@/shared/forms/useZodForm";
import { FormErrorSummary } from "@/shared/forms/FormErrorSummary";
import { projectSchema } from "@/modules/projects/schemas/project.schema";
import { useProjectStore } from "@/modules/projects/stores/useProjectStore";
import { useStaffStore } from "@/modules/users/stores/useStaffStore";
import { useVenueStore } from "@/modules/venues/stores/useVenueStore";
import { PaymentTermsEditor, validateTermRows } from "@/modules/projects/components/PaymentTermsEditor";
import { defaultTermTemplate } from "@/modules/projects/lib/terms";
import { todayISO } from "@/modules/projects/lib/dates";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import type { PaymentTermInput } from "@/modules/projects/types";

// Step 1 owns these; the venue text and status are decided when the project is created.
const step1Schema = projectSchema.pick({
  name: true, brideName: true, groomName: true, eventDate: true, prepStartDate: true, packageName: true,
  contractValue: true, picStaffId: true, picSalesStaffId: true, description: true,
});
const STEP_1_KEYS = new Set(Object.keys(step1Schema.shape));

const LABELS: Record<string, string> = {
  brideName: "Nama mempelai wanita", groomName: "Nama mempelai pria", name: "Nama project", eventDate: "Tanggal acara",
  prepStartDate: "Tanggal booking", packageName: "Paket / layanan", contractValue: "Nilai kontrak",
};

/**
 * Creating a project takes two screens (FLOW-IDEAL 2.6): 1) who is marrying,
 * when, which package and how much; 2) where, and when the client pays. The
 * project, its venue and its payment schedule are saved by ONE request that
 * either creates everything or nothing (PLAN.md E7): there is no "project made,
 * but the schedule failed" state. A project is always created as Draf.
 */
export default function ProjectCreatePage() {
  const navigate = useNavigate();
  const createProject = useProjectStore((s) => s.createProject);
  const staffList = useStaffStore((s) => s.staffSummaries);
  const fetchStaff = useStaffStore((s) => s.fetchStaffSummaries);
  const venues = useVenueStore((s) => s.venues);
  const fetchVenues = useVenueStore((s) => s.fetchVenues);
  const currentStaffId = useAuthStore((s) => s.currentStaffId);
  const assignedOnly = !useAllProjects();

  const [step, setStep] = useState<1 | 2>(1);
  const form = useZodForm({
    schema: step1Schema,
    initial: {
      name: "", brideName: "", groomName: "", eventDate: "", prepStartDate: todayISO(), packageName: "",
      contractValue: 0, picStaffId: "", picSalesStaffId: assignedOnly ? currentStaffId : "", description: "",
    },
  });
  const { values, errors } = form;
  const nameTouched = useRef(false);
  const [venueId, setVenueId] = useState("");
  const [manualVenue, setManualVenue] = useState("");
  const [rentalPrice, setRentalPrice] = useState(0);
  const [charge, setCharge] = useState(0);
  const [useSchedule, setUseSchedule] = useState(true);
  const [terms, setTerms] = useState<PaymentTermInput[]>([]);
  const seededTerms = useRef<PaymentTermInput[]>([]);
  const [termErrors, setTermErrors] = useState<Record<number, string>>({});
  const [venueErrors, setVenueErrors] = useState<Record<string, string>>({});
  const [submitting, setSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);
  const formRef = useRef<HTMLFormElement>(null);
  const step2Ref = useRef<HTMLDivElement>(null);

  // One Idempotency-Key per distinct request: retrying the same one after a lost answer cannot create two projects.
  const keyRef = useRef(newIdempotencyKey());
  const requestFingerprint = JSON.stringify([values, venueId, manualVenue, rentalPrice, charge, useSchedule, terms]);
  useEffect(() => {
    keyRef.current = newIdempotencyKey();
  }, [requestFingerprint]);

  useEffect(() => {
    void fetchStaff();
    void fetchVenues();
  }, [fetchStaff, fetchVenues]);

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
      // Start the schedule from the usual template for the values just entered.
      // Going back and changing the contract or dates re-seeds it, unless the
      // user already edited the rows by hand (then their edits stay).
      const template = defaultTermTemplate(values.contractValue, values.prepStartDate, values.eventDate);
      const untouched = terms.length === 0 || JSON.stringify(terms) === JSON.stringify(seededTerms.current);
      if (untouched) setTerms(template);
      seededTerms.current = template;
      setStep(2);
      window.scrollTo({ top: 0 });
    },
    { root: () => formRef.current },
  );

  function pickVenue(id: string) {
    setVenueId(id);
    const venue = venues.find((v) => v.id === id);
    // The venue's current price is only a starting point for this project.
    setRentalPrice(venue?.rentalPrice ?? 0);
    setCharge(venue?.charge ?? 0);
  }

  async function handleCreate() {
    setSubmitError(null);
    setVenueErrors({});
    if (useSchedule) {
      const problems = validateTermRows(terms);
      setTermErrors(problems);
      if (Object.keys(problems).length > 0) {
        setSubmitError("Lengkapi dulu termin yang ditandai, atau matikan jadwal termin.");
        return;
      }
      const total = terms.reduce((sum, t) => sum + t.amount, 0);
      if (values.contractValue > 0 && total > values.contractValue) {
        setSubmitError(`Jumlah termin melebihi nilai kontrak ${formatCurrency(values.contractValue)}.`);
        return;
      }
    }
    setTermErrors({});
    const venueName = venues.find((v) => v.id === venueId)?.name ?? (manualVenue.trim() || "Belum ditentukan");
    setSubmitting(true);
    try {
      const project = await createProject(
        { ...values, venue: venueName, status: "Draft" },
        { venueId: venueId || undefined, venueRentalPrice: rentalPrice, venueCharge: charge, terms: useSchedule ? terms : [] },
        { idempotencyKey: keyRef.current },
      );
      toast.success("Proyek dibuat");
      navigate(ROUTE_PATHS.projectDetail(project.id));
    } catch (err) {
      // Nothing was created, so everything typed stays and the person can fix it and press again.
      const fields = getApiFieldErrors(err);
      const pinnedToStep1 = Object.keys(fields).filter((k) => STEP_1_KEYS.has(k));
      const rowErrors: Record<number, string> = {};
      const venueProblems: Record<string, string> = {};
      let stray: string | null = null;
      for (const [key, message] of Object.entries(fields)) {
        const row = /^terms\[(\d+)\]/.exec(key);
        if (STEP_1_KEYS.has(key)) continue;
        if (row) rowErrors[Number(row[1])] ??= message;
        else if (key === "venueId" || key === "venueRentalPrice" || key === "venueCharge") venueProblems[key] = message;
        else stray ??= message;
      }
      if (pinnedToStep1.length > 0) {
        form.applyServerError(err, "Proyek belum dibuat. Periksa isian lalu coba lagi.", (k) => STEP_1_KEYS.has(k));
        setStep(1);
        requestAnimationFrame(() => focusField(formRef.current, pinnedToStep1[0]));
      } else {
        setTermErrors(rowErrors);
        setVenueErrors(venueProblems);
        const pinnedSomewhere = Object.keys(rowErrors).length > 0 || Object.keys(venueProblems).length > 0;
        setSubmitError(stray ?? (pinnedSomewhere ? "Perbaiki isian yang ditandai, lalu buat proyek lagi." : getApiErrorMessage(err, "Proyek belum dibuat. Periksa koneksi lalu coba lagi.")));
        requestAnimationFrame(() => {
          if (Object.keys(venueProblems).length > 0) focusField(step2Ref.current, Object.keys(venueProblems)[0]);
        });
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
          Langkah {step} dari 2: {step === 1 ? "Pasangan, tanggal, dan paket" : "Venue dan jadwal pembayaran klien"}
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
              <div className="sm:col-span-2">
                <FormErrorSummary errors={form.errorList} formError={form.formError} labels={LABELS} root={() => formRef.current} />
              </div>
              <Field label="Nama mempelai wanita" required name="brideName" error={errors.brideName}>
                <Input value={values.brideName} onChange={(e) => set("brideName", e.target.value)} onBlur={() => form.blur("brideName")} aria-invalid={invalid("brideName")} autoFocus />
              </Field>
              <Field label="Nama mempelai pria" required name="groomName" error={errors.groomName}>
                <Input value={values.groomName} onChange={(e) => set("groomName", e.target.value)} onBlur={() => form.blur("groomName")} aria-invalid={invalid("groomName")} />
              </Field>
              <div className="sm:col-span-2">
                <Field label="Nama project" required name="name" error={errors.name} hint="Terisi otomatis dari nama pasangan; boleh diubah.">
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
              <Field label="Tanggal acara" required name="eventDate" error={errors.eventDate}>
                <Input type="date" value={values.eventDate} onChange={(e) => set("eventDate", e.target.value)} aria-invalid={invalid("eventDate")} />
              </Field>
              <Field label="Tanggal booking" required name="prepStartDate" error={errors.prepStartDate}>
                <Input type="date" value={values.prepStartDate} onChange={(e) => set("prepStartDate", e.target.value)} aria-invalid={invalid("prepStartDate")} />
              </Field>
              <Field label="Paket / layanan" required name="packageName" error={errors.packageName}>
                <Input value={values.packageName} onChange={(e) => set("packageName", e.target.value)} onBlur={() => form.blur("packageName")} aria-invalid={invalid("packageName")} />
              </Field>
              <Field label="Nilai kontrak (Rp)" required name="contractValue" error={errors.contractValue}>
                <CurrencyInput value={values.contractValue} onChange={(n) => set("contractValue", n)} aria-invalid={invalid("contractValue")} />
              </Field>
              <Field label="Penanggung jawab WO (PIC Wedding Planner)">
                <Select value={values.picStaffId} onChange={(e) => set("picStaffId", e.target.value)}>
                  <option value="">Belum ditugaskan</option>
                  {staffOptions}
                </Select>
              </Field>
              <Field label="PIC Sales">
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
                <Field label="Catatan project">
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
              <section aria-labelledby="venue-heading" className="flex flex-col gap-4">
                <div>
                  <h2 id="venue-heading" className="text-[15px] font-semibold text-text-primary">Venue</h2>
                  <p className="mt-0.5 text-[13px] text-text-secondary">Pilih dari direktori venue, atau lewati dan atur nanti dari tab Venue.</p>
                </div>
                <Field label="Venue dari direktori" name="venueId" error={venueErrors.venueId}>
                  <Select value={venueId} onChange={(e) => pickVenue(e.target.value)}>
                    <option value="">Belum dipilih</option>
                    {venues.filter((v) => v.isActive).map((v) => (
                      <option key={v.id} value={v.id}>{v.city ? `${v.name} — ${v.city}` : v.name}</option>
                    ))}
                  </Select>
                </Field>
                {venueId ? (
                  <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                    <Field label="Harga sewa (Rp)" name="venueRentalPrice" error={venueErrors.venueRentalPrice} hint="Hanya berlaku untuk project ini; harga venue di direktori tidak berubah.">
                      <CurrencyInput value={rentalPrice} onChange={setRentalPrice} />
                    </Field>
                    <Field label="Biaya tambahan venue (Rp)" name="venueCharge" error={venueErrors.venueCharge}>
                      <CurrencyInput value={charge} onChange={setCharge} />
                    </Field>
                  </div>
                ) : (
                  <Field label="Atau tulis lokasi sementara" hint="Dipakai sebagai nama lokasi sampai venue dipilih dari direktori.">
                    <Input value={manualVenue} onChange={(e) => setManualVenue(e.target.value)} placeholder="cth. Gedung Serbaguna, Bandung" />
                  </Field>
                )}
              </section>

              <section aria-labelledby="terms-heading" className="flex flex-col gap-4 border-t border-border-light pt-6">
                <div className="flex flex-wrap items-start justify-between gap-3">
                  <div>
                    <h2 id="terms-heading" className="text-[15px] font-semibold text-text-primary">Jadwal pembayaran klien</h2>
                    <p className="mt-0.5 text-[13px] text-text-secondary">Dengan jadwal, termin yang terlambat langsung terlihat di Beranda.</p>
                  </div>
                  <label className="flex items-center gap-2 text-[13px] text-text-primary">
                    <input type="checkbox" checked={useSchedule} onChange={(e) => setUseSchedule(e.target.checked)} className="h-4 w-4" />
                    Atur jadwal sekarang
                  </label>
                </div>
                {useSchedule && (
                  <PaymentTermsEditor
                    rows={terms}
                    setRows={setTerms}
                    rowErrors={termErrors}
                    contractValue={values.contractValue}
                    bookingDate={values.prepStartDate}
                    eventDate={values.eventDate}
                  />
                )}
              </section>

              {submitError && (
                <p role="alert" className="rounded-md border border-danger/30 bg-danger-soft px-3.5 py-2.5 text-[13px] font-medium text-danger-text">
                  {submitError}
                </p>
              )}
              <div className="flex justify-between gap-2">
                <Button variant="secondary" onClick={() => setStep(1)} disabled={submitting}>Kembali</Button>
                <Button onClick={() => void handleCreate()} disabled={submitting}>{submitting ? "Membuat proyek..." : "Buat proyek"}</Button>
              </div>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
