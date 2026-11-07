import { useCallback, useEffect, useState } from "react";
import { Card, CardHeader, CardContent } from "@/shared/components/ui/Card";
import { Button } from "@/shared/components/ui/Button";
import { Field, Input } from "@/shared/components/ui/Input";
import { LoadError } from "@/shared/components/feedback/LoadError";
import { toast } from "@/shared/stores/useToastStore";
import { getApiErrorMessage, getApiFieldErrors } from "@/shared/lib/api-error";
import { useConsoleStore } from "@/modules/platform-admin/stores/useConsoleStore";
import { usePlatformCan } from "@/modules/platform-admin/lib/capabilities";
import { subscriptionPolicySchema } from "@/modules/platform-admin/schemas/console.schema";

type Draft = Record<"trialDays" | "expiringSoonDays" | "graceDays" | "readOnlyDays", string>;

const FIELDS: { key: keyof Draft; label: string; hint: string }[] = [
  { key: "trialDays", label: "Masa coba (hari)", hint: "Lama trial gratis untuk WO yang baru mendaftar. Berlaku untuk pendaftar berikutnya." },
  { key: "expiringSoonDays", label: "Pengingat sebelum berakhir (hari)", hint: "Mulai kapan banner \"Segera berakhir\" tampil di WO Console." },
  { key: "graceDays", label: "Masa tenggang (hari)", hint: "Setelah langganan berakhir, semua fitur masih bisa dipakai selama ini." },
  { key: "readOnlyDays", label: "Masa baca-saja (hari)", hint: "Setelah masa tenggang, data hanya bisa dilihat. Lewat ini, akses ditangguhkan sampai membayar." },
];

export default function SubscriptionPolicyPage() {
  const policy = useConsoleStore((s) => s.policy);
  const fetchPolicy = useConsoleStore((s) => s.fetchPolicy);
  const savePolicy = useConsoleStore((s) => s.savePolicy);
  const canEdit = usePlatformCan("super-admin");
  const [failed, setFailed] = useState(false);
  const [draft, setDraft] = useState<Draft | null>(null);
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [saving, setSaving] = useState(false);

  const load = useCallback(() => {
    setFailed(false);
    fetchPolicy().catch(() => setFailed(true));
  }, [fetchPolicy]);

  useEffect(() => {
    load();
  }, [load]);

  useEffect(() => {
    if (policy) {
      setDraft({
        trialDays: String(policy.trialDays),
        expiringSoonDays: String(policy.expiringSoonDays),
        graceDays: String(policy.graceDays),
        readOnlyDays: String(policy.readOnlyDays),
      });
    }
  }, [policy]);

  if (failed && !policy) return <LoadError onRetry={load} />;
  if (!policy || !draft) return <div className="py-20 text-center text-sm text-text-secondary">Memuat kebijakan...</div>;

  const dirty = FIELDS.some((f) => draft[f.key] !== String(policy[f.key]));
  const graceEnd = Number(draft.graceDays) || 0;
  const readOnlyEnd = graceEnd + (Number(draft.readOnlyDays) || 0);

  async function handleSave() {
    if (!draft) return;
    const parsed = subscriptionPolicySchema.safeParse(draft);
    if (!parsed.success) {
      const next: Record<string, string> = {};
      for (const issue of parsed.error.issues) next[String(issue.path[0])] = issue.message;
      setErrors(next);
      return;
    }
    setErrors({});
    setSaving(true);
    try {
      await savePolicy(parsed.data);
      toast.success("Kebijakan langganan disimpan");
    } catch (err) {
      setErrors(getApiFieldErrors(err));
      toast.error(getApiErrorMessage(err, "Kebijakan belum bisa disimpan"));
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className="flex max-w-3xl flex-col gap-5">
      <div>
        <h1 className="text-xl font-bold text-text-primary">Kebijakan Langganan</h1>
        <p className="mt-1 text-[13px] text-text-secondary">
          Aturan waktu yang sama untuk semua tenant: berapa lama trial, kapan mulai diingatkan, dan kapan akses dibatasi setelah langganan berakhir.
        </p>
      </div>

      <Card>
        <CardHeader title="Durasi" subtitle={canEdit ? "Perubahan langsung berlaku untuk semua tenant." : "Hanya Super Admin yang dapat mengubah kebijakan ini."} />
        <CardContent className="flex flex-col gap-5">
          {FIELDS.map((f) => (
            <Field key={f.key} label={f.label} hint={f.hint} error={errors[f.key]} htmlFor={`policy-${f.key}`}>
              <Input
                id={`policy-${f.key}`}
                type="number"
                inputMode="numeric"
                className="max-w-40"
                value={draft[f.key]}
                disabled={!canEdit || saving}
                aria-invalid={errors[f.key] ? true : undefined}
                onChange={(e) => setDraft({ ...draft, [f.key]: e.target.value })}
              />
            </Field>
          ))}
        </CardContent>
      </Card>

      <Card>
        <CardHeader title="Yang dialami tenant" subtitle="Contoh urutan setelah langganan berakhir pada hari ke-0." />
        <CardContent>
          <ol className="flex flex-col gap-2 text-[13.5px] text-text-primary">
            <li>Mulai {draft.expiringSoonDays || 0} hari sebelum berakhir: banner pengingat.</li>
            <li>Hari ke-0 sampai ke-{graceEnd}: masa tenggang, semua fitur masih bisa dipakai.</li>
            <li>Hari ke-{graceEnd} sampai ke-{readOnlyEnd}: baca-saja, data bisa dilihat tetapi tidak diubah.</li>
            <li>Setelah hari ke-{readOnlyEnd}: ditangguhkan, hanya halaman pembayaran yang bisa dibuka.</li>
          </ol>
        </CardContent>
      </Card>

      {canEdit && (
        <div className="flex gap-3">
          <Button onClick={() => void handleSave()} disabled={!dirty || saving}>
            {saving ? "Menyimpan..." : "Simpan kebijakan"}
          </Button>
          {dirty && (
            <Button variant="secondary" disabled={saving} onClick={() => { setDraft(null); setErrors({}); void fetchPolicy(); }}>
              Batalkan perubahan
            </Button>
          )}
        </div>
      )}
    </div>
  );
}
