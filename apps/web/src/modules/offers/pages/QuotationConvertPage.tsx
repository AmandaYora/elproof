import { useEffect, useMemo, useRef, useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { AlertTriangle, CheckCircle2, FolderOpen, Info, Paperclip } from "lucide-react";
import { Button } from "@/shared/components/ui/Button";
import { Breadcrumbs } from "@/shared/components/ui/Breadcrumbs";
import { Card, CardContent, CardHeader } from "@/shared/components/ui/Card";
import { CurrencyInput } from "@/shared/components/ui/CurrencyInput";
import { Field, Input, Select, Textarea } from "@/shared/components/ui/Input";
import { Skeleton, SkeletonRows } from "@/shared/components/ui/Skeleton";
import { LoadError } from "@/shared/components/feedback/LoadError";
import { FormErrorSummary } from "@/shared/forms/FormErrorSummary";
import { useZodForm } from "@/shared/forms/useZodForm";
import { getApiErrorMessage, renameApiFieldErrors } from "@/shared/lib/api-error";
import { newIdempotencyKey } from "@/shared/lib/direct-upload";
import { formatCurrency, formatDate, todayISO } from "@/shared/lib/formatters";
import { useCan } from "@/shared/stores/usePermissionStore";
import { toast } from "@/shared/stores/useToastStore";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import { useStaffStore } from "@/modules/users/stores/useStaffStore";
import { convertSchema, PAYMENT_METHOD_OPTIONS, type ConvertValues } from "@/modules/offers/schemas/quotation.schema";
import { useQuotationStore } from "@/modules/offers/stores/useQuotationStore";
import type { ConversionInput, ConversionPreview, QuotationDetail, SnapshotEvent } from "@/modules/offers/types";

const FIELD_LABELS: Record<string, string> = {
  name: "Nama proyek",
  brideName: "Mempelai wanita",
  groomName: "Mempelai pria",
  prepStartDate: "Mulai persiapan",
  picStaffId: "PIC",
  picSalesStaffId: "PIC Sales",
  description: "Catatan",
  dpAmount: "Jumlah tanda jadi",
  dpDate: "Tanggal bayar",
  dpMethod: "Metode bayar",
  dpReference: "Nomor referensi",
  dpNotes: "Catatan pembayaran",
  dpProof: "Bukti bayar",
};
const PROOF_TYPES = ["image/jpeg", "image/png", "image/webp", "application/pdf"];

/** "Ayu & Bima" / "Ayu dan Bima" → the bride and the groom, as the server does it. */
function splitCouple(couple: string): [string, string] {
  for (const sep of ["&", " dan "]) {
    const i = couple.indexOf(sep);
    if (i > 0) return [couple.slice(0, i).trim(), couple.slice(i + sep.length).trim()];
  }
  return ["", ""];
}

/** Server keys → this form's fields ("events.1.eventDate" → "eventDates[1]", "downPayment.amount" → "dpAmount"). */
function formKey(key: string): string | undefined {
  const event = /^events[.[](\d+)\]?\.eventDate$/.exec(key);
  if (event) return `eventDates[${event[1]}]`;
  if (key.startsWith("downPayment.evidence")) return "dpProof"; // ui-terms-ok: an API key, never shown
  const dp: Record<string, string> = {
    "downPayment.amount": "dpAmount",
    "downPayment.paymentDate": "dpDate",
    "downPayment.method": "dpMethod",
    "downPayment.referenceNumber": "dpReference",
    "downPayment.notes": "dpNotes",
  };
  return dp[key];
}

function toInput(v: ConvertValues, dates: string[], proofToken: string, proof: File | null): ConversionInput {
  return {
    name: v.name,
    brideName: v.brideName,
    groomName: v.groomName,
    eventDates: dates,
    prepStartDate: v.prepStartDate,
    picStaffId: v.picStaffId,
    picSalesStaffId: v.picSalesStaffId,
    description: v.description,
    downPayment: v.withDownPayment
      ? {
          amount: Number(v.dpAmount) || 0,
          paymentDate: v.dpDate,
          method: v.dpMethod,
          referenceNumber: v.dpReference,
          notes: v.dpNotes,
          evidence: proofToken && proof ? [{ uploadToken: proofToken, type: "Transfer Proof", name: proof.name }] : [],
        }
      : null,
  };
}

/**
 * "Jadikan proyek" (DESAIN §6.5, A2): the exact date of each event, preparation and team, an optional down payment,
 * and a check of what will be made before "Buat proyek". The request carries one Idempotency-Key, so a double click
 * or a retry lands on the same project.
 */
export default function QuotationConvertPage() {
  const { quotationId = "" } = useParams<{ quotationId: string }>();
  const navigate = useNavigate();
  const fetchDetail = useQuotationStore((s) => s.fetchDetail);
  const fetchRevision = useQuotationStore((s) => s.fetchRevision);
  const [detail, setDetail] = useState<QuotationDetail | null>(null);
  const [events, setEvents] = useState<SnapshotEvent[] | null>(null);
  const [failed, setFailed] = useState(false);
  const [attempt, setAttempt] = useState(0);

  useEffect(() => {
    let cancelled = false;
    setFailed(false);
    (async () => {
      const d = await fetchDetail(quotationId);
      const n = d.quotation.acceptedRevisionN;
      const rev = n ? await fetchRevision(quotationId, n) : null;
      if (cancelled) return;
      setDetail(d);
      setEvents(rev?.content.events ?? []);
    })().catch(() => !cancelled && setFailed(true));
    return () => {
      cancelled = true;
    };
  }, [fetchDetail, fetchRevision, quotationId, attempt]);

  const crumbs = [
    { label: "Penawaran", to: ROUTE_PATHS.quotations },
    { label: detail?.quotation.number ?? "Penawaran", to: ROUTE_PATHS.quotationDetail(quotationId) },
    { label: "Jadikan proyek" },
  ];

  if (failed) {
    return (
      <div className="flex flex-col gap-4">
        <Breadcrumbs items={crumbs} />
        <LoadError message="Penawaran tidak bisa dimuat." onRetry={() => setAttempt((n) => n + 1)} />
      </div>
    );
  }
  if (!detail || !events) {
    return (
      <div className="flex flex-col gap-4">
        <Breadcrumbs items={crumbs} />
        <div role="status" aria-label="Memuat penawaran" className="flex flex-col gap-4">
          <Skeleton className="h-9 w-2/5" />
          <SkeletonRows rows={5} label="Memuat penawaran" />
        </div>
      </div>
    );
  }

  const q = detail.quotation;
  if (q.status !== "Accepted") {
    return (
      <div className="flex flex-col gap-4">
        <Breadcrumbs items={crumbs} />
        <h1 className="text-[22px] font-semibold text-foreground">Jadikan proyek</h1>
        <p className="flex items-start gap-2 rounded-md border border-warning/30 bg-warning-soft px-3.5 py-2.5 text-[13px] text-warning-text">
          <AlertTriangle aria-hidden className="mt-0.5 h-4 w-4 shrink-0" />
          Hanya penawaran yang sudah diterima yang bisa dijadikan proyek. Tandai diterima dulu di halaman penawaran.
        </p>
        <div>
          <Button variant="secondary" onClick={() => navigate(ROUTE_PATHS.quotationDetail(q.id))}>
            Kembali ke penawaran
          </Button>
        </div>
      </div>
    );
  }

  return <ConvertForm detail={detail} events={events} crumbs={crumbs} />;
}

function ConvertForm({ detail, events, crumbs }: { detail: QuotationDetail; events: SnapshotEvent[]; crumbs: Array<{ label: string; to?: string }> }) {
  const navigate = useNavigate();
  const q = detail.quotation;
  const canRecord = useCan("finance.record");
  const previewConversion = useQuotationStore((s) => s.previewConversion);
  const convert = useQuotationStore((s) => s.convert);
  const uploadConversionProof = useQuotationStore((s) => s.uploadConversionProof);
  const staff = useStaffStore((s) => s.staffSummaries);
  const fetchStaffSummaries = useStaffStore((s) => s.fetchStaffSummaries);
  const formRef = useRef<HTMLFormElement>(null);
  const keyRef = useRef(newIdempotencyKey());
  const [proof, setProof] = useState<File | null>(null);
  const [proofError, setProofError] = useState<string | undefined>();
  const [preview, setPreview] = useState<ConversionPreview | null>(null);
  const [previewError, setPreviewError] = useState<string | null>(null);

  const [bride, groom] = splitCouple(q.coupleNames);
  const form = useZodForm({
    schema: convertSchema,
    initial: {
      name: q.coupleNames,
      brideName: bride,
      groomName: groom,
      eventDates: events.map((e) => e.eventDate),
      prepStartDate: todayISO(),
      picStaffId: q.picStaffId,
      picSalesStaffId: q.picStaffId,
      description: "",
      withDownPayment: false,
      dpAmount: 0,
      dpDate: todayISO(),
      dpMethod: "Transfer Bank",
      dpReference: "",
      dpNotes: "",
    },
  });
  const { values, errors, set, blur } = form;
  const dates = values.eventDates as string[];

  useEffect(() => {
    void fetchStaffSummaries().catch(() => undefined);
  }, [fetchStaffSummaries]);

  // "Periksa sebelum membuat": what the server would make from these values (writes nothing). The dates move the
  // terms' due dates, so it follows them, 500 ms after the last change.
  const previewKey = JSON.stringify([dates, values.prepStartDate]);
  useEffect(() => {
    let cancelled = false;
    const timer = window.setTimeout(() => {
      previewConversion(q.id, toInput(form.values as ConvertValues, dates, "", null)).then(
        (p) => {
          if (cancelled) return;
          setPreview(p);
          setPreviewError(null);
        },
        (err) => !cancelled && setPreviewError(getApiErrorMessage(err, "Ringkasan belum bisa dihitung. Periksa isian, lalu coba lagi.")),
      );
    }, 500);
    return () => {
      cancelled = true;
      window.clearTimeout(timer);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [previewKey, q.id, previewConversion]);

  function setDate(i: number, value: string) {
    set(
      "eventDates",
      dates.map((d, j) => (j === i ? value : d)),
    );
  }

  function pickProof(f: File | undefined) {
    setProofError(undefined);
    if (!f) {
      setProof(null);
      return;
    }
    if (!PROOF_TYPES.includes(f.type)) {
      setProofError("Pilih foto (JPG, PNG, WEBP) atau PDF");
      return;
    }
    setProof(f);
  }

  const submit = form.handleSubmit(
    async (v) => {
      let token = "";
      if (v.withDownPayment && proof) {
        try {
          token = await uploadConversionProof(q.id, proof);
        } catch (err) {
          throw renameApiFieldErrors(err, () => "dpProof");
        }
      }
      try {
        const res = await convert(q.id, toInput(v, dates, token, proof), keyRef.current);
        if (res.created) toast.success("Proyek dibuat");
        else toast.info("Penawaran ini sudah dijadikan proyek");
        navigate(ROUTE_PATHS.projectDetail(res.projectId));
      } catch (err) {
        // A refusal is about these values: the next attempt (after a fix) is a new request.
        keyRef.current = newIdempotencyKey();
        throw renameApiFieldErrors(err, formKey);
      }
    },
    { root: () => formRef.current, fallback: "Proyek belum dibuat. Periksa koneksi Anda, lalu coba lagi." },
  );

  const staffChoices = useMemo(
    () => (staff.some((m) => m.id === q.picStaffId) ? staff : [{ id: q.picStaffId, name: q.picName || "PIC penawaran", title: "" }, ...staff]),
    [staff, q.picStaffId, q.picName],
  );
  const existing = preview?.existingProjectId;
  const fieldError = (key: string) => errors[key];

  return (
    <div className="flex flex-col gap-5">
      <Breadcrumbs items={crumbs} />
      <div>
        <h1 className="text-[22px] leading-tight font-semibold text-foreground">Jadikan proyek</h1>
        <p className="mt-1 text-[13px] text-text-secondary">
          {q.number} Revisi {q.acceptedRevisionN} · {q.coupleNames} · diterima {formatDate(q.acceptedOn)}
        </p>
      </div>

      {existing && (
        <p role="status" className="flex flex-wrap items-center gap-2 rounded-md border border-info/30 bg-info-soft px-3.5 py-2.5 text-[13px] text-info-text">
          <Info aria-hidden className="h-4 w-4 shrink-0" />
          Penawaran ini sudah dijadikan proyek.
          <Link to={ROUTE_PATHS.projectDetail(existing)} className="font-semibold underline underline-offset-2">
            Buka proyek
          </Link>
        </p>
      )}

      <form ref={formRef} noValidate onSubmit={(e) => void submit(e)} className="flex flex-col gap-5">
        <FormErrorSummary errors={form.errorList} formError={form.formError} labels={FIELD_LABELS} root={() => formRef.current} />

        <Card>
          <CardHeader title="Tanggal acara" subtitle="Tanggal pasti setiap acara. Yang kosong di penawaran wajib diisi sekarang." />
          <CardContent className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {events.length === 0 ? (
              <p className="text-[13px] text-danger-text">Penawaran ini belum punya acara.</p>
            ) : (
              events.map((e, i) => (
                <Field key={i} label={e.kindName} required name={`eventDates[${i}]`} error={fieldError(`eventDates[${i}]`)} hint={e.eventDate ? undefined : "Belum ada tanggal di penawaran"}>
                  <Input type="date" value={dates[i] ?? ""} onChange={(ev) => setDate(i, ev.target.value)} />
                </Field>
              ))
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader title="Proyek, persiapan & tim" />
          <CardContent className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <div className="sm:col-span-2">
              <Field label="Nama proyek" required name="name" error={fieldError("name")}>
                <Input value={values.name} onChange={(e) => set("name", e.target.value)} onBlur={() => blur("name")} />
              </Field>
            </div>
            <Field label="Mempelai wanita" required name="brideName" error={fieldError("brideName")}>
              <Input value={values.brideName} onChange={(e) => set("brideName", e.target.value)} onBlur={() => blur("brideName")} />
            </Field>
            <Field label="Mempelai pria" required name="groomName" error={fieldError("groomName")}>
              <Input value={values.groomName} onChange={(e) => set("groomName", e.target.value)} onBlur={() => blur("groomName")} />
            </Field>
            <Field label="Mulai persiapan" required name="prepStartDate" error={fieldError("prepStartDate")}>
              <Input type="date" value={values.prepStartDate} onChange={(e) => set("prepStartDate", e.target.value)} onBlur={() => blur("prepStartDate")} />
            </Field>
            <Field label="PIC" required name="picStaffId" error={fieldError("picStaffId")}>
              <Select aria-label="PIC" value={values.picStaffId} onChange={(e) => set("picStaffId", e.target.value)}>
                {staffChoices.map((m) => (
                  <option key={m.id} value={m.id}>
                    {m.name}
                  </option>
                ))}
              </Select>
            </Field>
            <Field label="PIC Sales" name="picSalesStaffId" error={fieldError("picSalesStaffId")} hint="Bawaan: PIC penawaran.">
              <Select aria-label="PIC Sales" value={values.picSalesStaffId} onChange={(e) => set("picSalesStaffId", e.target.value)}>
                {staffChoices.map((m) => (
                  <option key={m.id} value={m.id}>
                    {m.name}
                  </option>
                ))}
              </Select>
            </Field>
            <div className="sm:col-span-2">
              <Field label="Catatan (opsional)" name="description" error={fieldError("description")}>
                <Textarea rows={2} value={values.description} onChange={(e) => set("description", e.target.value)} onBlur={() => blur("description")} />
              </Field>
            </div>
          </CardContent>
        </Card>

        {canRecord && (
          <Card>
            <CardHeader title="Tanda jadi" subtitle="Bila klien sudah membayar, catat sekarang sebagai pembayaran DP." />
            <CardContent className="flex flex-col gap-4">
              <fieldset className="flex flex-wrap gap-x-6 gap-y-2" data-field="withDownPayment">
                <legend className="sr-only">Tanda jadi</legend>
                <label className="flex items-center gap-2 text-[13px] text-foreground">
                  <input type="radio" name="withDownPayment" checked={!values.withDownPayment} onChange={() => set("withDownPayment", false)} className="size-4 accent-primary" />
                  Belum ada
                </label>
                <label className="flex items-center gap-2 text-[13px] text-foreground">
                  <input type="radio" name="withDownPayment" checked={values.withDownPayment} onChange={() => set("withDownPayment", true)} className="size-4 accent-primary" />
                  Sudah diterima
                </label>
              </fieldset>
              {values.withDownPayment && (
                <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
                  <Field label="Jumlah (Rp)" required name="dpAmount" error={fieldError("dpAmount")}>
                    <CurrencyInput value={Number(values.dpAmount) || 0} onChange={(n) => set("dpAmount", n)} onBlur={() => blur("dpAmount")} aria-invalid={Boolean(fieldError("dpAmount"))} />
                  </Field>
                  <Field label="Tanggal bayar" required name="dpDate" error={fieldError("dpDate")}>
                    <Input type="date" max={todayISO()} value={values.dpDate} onChange={(e) => set("dpDate", e.target.value)} onBlur={() => blur("dpDate")} />
                  </Field>
                  <Field label="Metode bayar" required name="dpMethod" error={fieldError("dpMethod")}>
                    <Select aria-label="Metode bayar" value={values.dpMethod} onChange={(e) => set("dpMethod", e.target.value)}>
                      {PAYMENT_METHOD_OPTIONS.map((m) => (
                        <option key={m} value={m}>
                          {m}
                        </option>
                      ))}
                    </Select>
                  </Field>
                  <Field label="Nomor referensi (opsional)" name="dpReference" error={fieldError("dpReference")}>
                    <Input value={values.dpReference} onChange={(e) => set("dpReference", e.target.value)} onBlur={() => blur("dpReference")} />
                  </Field>
                  <div className="sm:col-span-2">
                    <Field label="Catatan pembayaran (opsional)" name="dpNotes" error={fieldError("dpNotes")}>
                      <Input value={values.dpNotes} onChange={(e) => set("dpNotes", e.target.value)} onBlur={() => blur("dpNotes")} />
                    </Field>
                  </div>
                  <div className="sm:col-span-3">
                    <Field label="Bukti bayar (opsional)" name="dpProof" error={proofError ?? fieldError("dpProof")} hint="Foto atau PDF bukti transfer.">
                      <Input type="file" accept={PROOF_TYPES.join(",")} onChange={(e) => pickProof(e.target.files?.[0])} />
                    </Field>
                    {proof && (
                      <p className="mt-1.5 flex items-center gap-2 text-[13px] text-text-secondary">
                        <Paperclip aria-hidden className="h-3.5 w-3.5" />
                        {proof.name}
                      </p>
                    )}
                  </div>
                </div>
              )}
            </CardContent>
          </Card>
        )}

        <Card>
          <CardHeader title="Periksa sebelum membuat" />
          <CardContent>
            {previewError && !preview ? (
              <p role="alert" className="flex items-start gap-2 text-[13px] text-danger-text">
                <AlertTriangle aria-hidden className="mt-0.5 h-4 w-4 shrink-0" />
                {previewError}
              </p>
            ) : !preview ? (
              <SkeletonRows rows={3} label="Menghitung ringkasan" />
            ) : (
              <div className="flex flex-col gap-3 text-[13.5px]" aria-live="polite">
                <p className="flex flex-wrap items-baseline justify-between gap-2">
                  <span className="font-semibold text-foreground">Nilai kontrak</span>
                  <span className="font-semibold tabular-nums">{formatCurrency(preview.contractTotal)}</span>
                </p>
                {preview.terms.length > 0 && (
                  <ul className="flex flex-col divide-y divide-border-light">
                    {preview.terms.map((t, i) => (
                      <li key={i} className="flex flex-wrap items-baseline justify-between gap-2 py-1.5">
                        <span>
                          {t.label}
                          <span className="text-muted-foreground"> · jatuh tempo {t.dueDate ? formatDate(t.dueDate) : "-"}</span>
                        </span>
                        <span className="ml-auto tabular-nums">{formatCurrency(t.amount)}</span>
                      </li>
                    ))}
                  </ul>
                )}
                {values.withDownPayment && Number(values.dpAmount) > 0 && (
                  <p className="text-text-secondary">Tanda jadi {formatCurrency(Number(values.dpAmount))} tercatat sebagai pembayaran DP.</p>
                )}
                <p className="text-text-secondary">
                  Disalin: {preview.counts.events} acara · {preview.counts.suppliers} vendor & venue · {preview.counts.lines} baris rincian
                  {preview.counts.hasTermsText ? " · syarat & ketentuan tetap terbaca di Kesepakatan awal" : ""}
                </p>
                {previewError ? (
                  <p role="alert" className="flex items-start gap-2 text-danger-text">
                    <AlertTriangle aria-hidden className="mt-0.5 h-4 w-4 shrink-0" />
                    {previewError}
                  </p>
                ) : (
                  <p className="flex items-center gap-2 text-success-text">
                    <CheckCircle2 aria-hidden className="h-4 w-4" />
                    Kapasitas semua sesi dicek ulang saat proyek dibuat.
                  </p>
                )}
              </div>
            )}
          </CardContent>
        </Card>

        <div className="flex flex-wrap justify-end gap-2">
          <Button variant="secondary" onClick={() => navigate(ROUTE_PATHS.quotationDetail(q.id))} disabled={form.submitting}>
            Batal
          </Button>
          {existing ? (
            <Button icon={<FolderOpen className="h-4 w-4" />} onClick={() => navigate(ROUTE_PATHS.projectDetail(existing))}>
              Buka proyek
            </Button>
          ) : (
            <Button type="submit" disabled={form.submitting || events.length === 0}>
              {form.submitting ? "Membuat proyek…" : "Buat proyek"}
            </Button>
          )}
        </div>
      </form>
    </div>
  );
}
