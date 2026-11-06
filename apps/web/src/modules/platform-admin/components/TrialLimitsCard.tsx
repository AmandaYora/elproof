import { useEffect, useState } from "react";
import axios from "axios";
import { Card, CardHeader } from "@/shared/components/ui/Card";
import { Button } from "@/shared/components/ui/Button";
import { Field, Input } from "@/shared/components/ui/Input";
import { httpClient } from "@/shared/services/http-client";
import { API } from "@/shared/services/api-endpoints";
import { getApiErrorMessage } from "@/shared/lib/api-error";
import { useAuthStore } from "@/shared/stores/useAuthStore";

interface TrialLimits {
  maxStaff: number;
  maxActiveProjects: number;
  maxClients: number;
}

const FIELDS: { key: keyof TrialLimits; label: string; hint: string }[] = [
  { key: "maxStaff", label: "Staf selain Owner", hint: "Akun staf aktif" },
  { key: "maxActiveProjects", label: "Project aktif", hint: "Belum selesai, dibatalkan, atau diarsipkan" },
  { key: "maxClients", label: "Akun client", hint: "Akun client aktif" },
];

/**
 * Free-trial caps (decision K6): only trialing workspaces are limited;
 * paid plans are unlimited. Billing and Super Admin may change them.
 */
export function TrialLimitsCard() {
  const role = useAuthStore((s) => s.session?.role);
  const canEdit = role === "Super Admin" || role === "Billing";
  const [saved, setSaved] = useState<TrialLimits | null>(null);
  const [draft, setDraft] = useState<Record<keyof TrialLimits, string>>({ maxStaff: "", maxActiveProjects: "", maxClients: "" });
  const [errors, setErrors] = useState<Partial<Record<keyof TrialLimits, string>>>({});
  const [message, setMessage] = useState<{ tone: "ok" | "error"; text: string } | null>(null);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    httpClient
      .get(API.platform.trialLimits)
      .then((res) => {
        const l = res.data.data as TrialLimits;
        setSaved(l);
        setDraft({ maxStaff: String(l.maxStaff), maxActiveProjects: String(l.maxActiveProjects), maxClients: String(l.maxClients) });
      })
      .catch((err) => setMessage({ tone: "error", text: getApiErrorMessage(err, "Batas masa coba gagal dimuat.") }));
  }, []);

  const dirty =
    saved !== null && FIELDS.some((f) => draft[f.key] !== String(saved[f.key]));

  async function save() {
    const next: Partial<TrialLimits> = {};
    const fieldErrors: Partial<Record<keyof TrialLimits, string>> = {};
    for (const f of FIELDS) {
      const n = Number(draft[f.key]);
      if (!Number.isInteger(n) || n < 1 || n > 1000) fieldErrors[f.key] = "Isi angka 1–1000";
      else next[f.key] = n;
    }
    setErrors(fieldErrors);
    if (Object.keys(fieldErrors).length > 0) return;
    setSaving(true);
    setMessage(null);
    try {
      const res = await httpClient.put(API.platform.trialLimits, next);
      setSaved(res.data.data as TrialLimits);
      setMessage({ tone: "ok", text: "Batas masa coba tersimpan. Berlaku untuk penambahan berikutnya di semua workspace masa coba." });
    } catch (err) {
      if (axios.isAxiosError(err) && err.response?.status === 422) {
        const f = (err.response.data as { errors?: Record<string, string[]> }).errors ?? {};
        setErrors({ maxStaff: f.maxStaff?.[0], maxActiveProjects: f.maxActiveProjects?.[0], maxClients: f.maxClients?.[0] });
      }
      setMessage({ tone: "error", text: getApiErrorMessage(err, "Batas masa coba belum tersimpan.") });
    } finally {
      setSaving(false);
    }
  }

  return (
    <Card>
      <CardHeader
        title="Batas masa coba"
        subtitle="Berlaku hanya untuk workspace yang masih masa coba 14 hari. Paket berbayar tidak dibatasi."
      />
      <div className="flex flex-col gap-4 p-5">
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
          {FIELDS.map((f) => (
            <Field key={f.key} label={f.label} htmlFor={`trial-${f.key}`} hint={f.hint} error={errors[f.key]}>
              <Input
                id={`trial-${f.key}`}
                type="number"
                inputMode="numeric"
                min={1}
                max={1000}
                value={draft[f.key]}
                disabled={!canEdit || saved === null}
                onChange={(e) => {
                  setDraft((d) => ({ ...d, [f.key]: e.target.value }));
                  setErrors((er) => ({ ...er, [f.key]: undefined }));
                  setMessage(null);
                }}
              />
            </Field>
          ))}
        </div>
        <div className="flex flex-wrap items-center justify-between gap-3">
          {message ? (
            <p role="status" className={message.tone === "ok" ? "text-[13px] text-success" : "text-[13px] font-medium text-danger"}>
              {message.text}
            </p>
          ) : (
            <p className="text-[12.5px] text-text-secondary">
              {canEdit ? "Workspace yang sudah melewati batas tidak terhapus datanya; hanya penambahan baru yang ditolak." : "Hanya Super Admin dan Billing yang dapat mengubah batas ini."}
            </p>
          )}
          {canEdit && (
            <Button onClick={() => void save()} disabled={!dirty || saving}>
              {saving ? "Menyimpan..." : "Simpan batas"}
            </Button>
          )}
        </div>
      </div>
    </Card>
  );
}
