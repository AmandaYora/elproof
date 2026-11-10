import { useEffect, useRef, useState } from "react";
import { Card, CardHeader } from "@/shared/components/ui/Card";
import { Button } from "@/shared/components/ui/Button";
import { Field, Input } from "@/shared/components/ui/Input";
import { httpClient } from "@/shared/services/http-client";
import { API } from "@/shared/services/api-endpoints";
import { getApiErrorMessage } from "@/shared/lib/api-error";
import { useAuthStore } from "@/shared/stores/useAuthStore";
import { trialLimitsSchema } from "@/modules/platform-admin/schemas/console.schema";
import { useZodForm } from "@/shared/forms/useZodForm";
import { FormErrorSummary } from "@/shared/forms/FormErrorSummary";

interface TrialLimits {
  maxStaff: number;
  maxActiveProjects: number;
  maxClients: number;
}

const FIELDS: { key: keyof TrialLimits; label: string; hint: string }[] = [
  { key: "maxStaff", label: "Anggota tim selain Owner", hint: "Akun anggota tim aktif" },
  { key: "maxActiveProjects", label: "Proyek aktif", hint: "Belum selesai, dibatalkan, atau diarsipkan" },
  { key: "maxClients", label: "Akun klien", hint: "Akun klien aktif" },
];

function toDraft(l: TrialLimits): Record<keyof TrialLimits, string> {
  return { maxStaff: String(l.maxStaff), maxActiveProjects: String(l.maxActiveProjects), maxClients: String(l.maxClients) };
}

const FIELD_LABELS: Record<string, string> = { maxStaff: "Anggota tim selain Owner", maxActiveProjects: "Proyek aktif", maxClients: "Akun klien" };

/**
 * Free-trial caps (decision K6): only trialing workspaces are limited;
 * paid plans are unlimited. Billing and Super Admin may change them.
 */
export function TrialLimitsCard() {
  const role = useAuthStore((s) => s.session?.role);
  const canEdit = role === "Super Admin" || role === "Billing";
  const [saved, setSaved] = useState<TrialLimits | null>(null);
  const form = useZodForm({ schema: trialLimitsSchema, initial: { maxStaff: "", maxActiveProjects: "", maxClients: "" } });
  const { values: draft, errors } = form;
  const { reset } = form;
  const formRef = useRef<HTMLFormElement>(null);
  const [message, setMessage] = useState<{ tone: "ok" | "error"; text: string } | null>(null);

  useEffect(() => {
    httpClient
      .get(API.platform.trialLimits)
      .then((res) => {
        const l = res.data.data as TrialLimits;
        setSaved(l);
        reset(toDraft(l));
      })
      .catch((err) => setMessage({ tone: "error", text: getApiErrorMessage(err, "Batas masa coba gagal dimuat.") }));
  }, []);

  const dirty =
    saved !== null && FIELDS.some((f) => draft[f.key] !== String(saved[f.key]));

  // A failure (a 422 on a field, or anything else) is shown above the fields by the form.
  const save = form.handleSubmit(
    async (next) => {
      setMessage(null);
      const res = await httpClient.put(API.platform.trialLimits, next);
      const limits = res.data.data as TrialLimits;
      setSaved(limits);
      reset(toDraft(limits));
      setMessage({ tone: "ok", text: "Batas masa coba tersimpan. Berlaku untuk penambahan berikutnya di semua workspace masa coba." });
    },
    { root: () => formRef.current, fallback: "Batas masa coba belum tersimpan.", known: (key) => key in FIELD_LABELS },
  );

  return (
    <Card>
      <CardHeader
        title="Batas masa coba"
        subtitle="Berlaku hanya untuk workspace yang masih masa coba 14 hari. Paket berbayar tidak dibatasi."
      />
      <form ref={formRef} noValidate onSubmit={(e) => void save(e)} className="flex flex-col gap-4 p-5">
        <FormErrorSummary errors={form.errorList} formError={form.formError} labels={FIELD_LABELS} root={() => formRef.current} />
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
          {FIELDS.map((f) => (
            <Field key={f.key} label={f.label} htmlFor={`trial-${f.key}`} name={f.key} hint={f.hint} error={errors[f.key]}>
              <Input
                id={`trial-${f.key}`}
                type="number"
                inputMode="numeric"
                min={1}
                max={1000}
                value={draft[f.key]}
                disabled={!canEdit || saved === null}
                onChange={(e) => {
                  form.set(f.key, e.target.value);
                  setMessage(null);
                }}
                onBlur={() => form.blur(f.key)}
              />
            </Field>
          ))}
        </div>
        <div className="flex flex-wrap items-center justify-between gap-3">
          {message ? (
            <p role="status" className={message.tone === "ok" ? "text-[13px] text-success-text" : "text-[13px] font-medium text-danger-text"}>
              {message.text}
            </p>
          ) : (
            <p className="text-[12.5px] text-text-secondary">
              {canEdit ? "Workspace yang sudah melewati batas tidak terhapus datanya; hanya penambahan baru yang ditolak." : "Hanya Super Admin dan Billing yang dapat mengubah batas ini."}
            </p>
          )}
          {canEdit && (
            <Button type="submit" disabled={!dirty || form.submitting}>
              {form.submitting ? "Menyimpan..." : "Simpan batas"}
            </Button>
          )}
        </div>
      </form>
    </Card>
  );
}
