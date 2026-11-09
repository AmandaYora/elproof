import { useCallback, useEffect, useRef, useState } from "react";
import { Card, CardHeader, CardContent } from "@/shared/components/ui/Card";
import { Button } from "@/shared/components/ui/Button";
import { Field, Input } from "@/shared/components/ui/Input";
import { LoadError } from "@/shared/components/feedback/LoadError";
import { toast } from "@/shared/stores/useToastStore";
import { getApiErrorMessage } from "@/shared/lib/api-error";
import { useConsoleStore } from "@/modules/platform-admin/stores/useConsoleStore";
import { usePlatformCan } from "@/modules/platform-admin/lib/capabilities";
import { subscriptionPolicyFormSchema } from "@/modules/platform-admin/schemas/console.schema";
import { useZodForm } from "@/shared/forms/useZodForm";
import { FormErrorSummary } from "@/shared/forms/FormErrorSummary";

type Draft = Record<"trialDays" | "expiringSoonDays" | "graceDays" | "readOnlyDays", string>;

const FIELDS: { key: keyof Draft; label: string; hint: string }[] = [
  { key: "trialDays", label: "Masa coba (hari)", hint: "Lama trial gratis untuk WO yang baru mendaftar. Berlaku untuk pendaftar berikutnya." },
  { key: "expiringSoonDays", label: "Pengingat sebelum berakhir (hari)", hint: "Mulai kapan banner \"Segera berakhir\" tampil di WO Console." },
  { key: "graceDays", label: "Masa tenggang (hari)", hint: "Setelah langganan berakhir, semua fitur masih bisa dipakai selama ini." },
  { key: "readOnlyDays", label: "Masa baca-saja (hari)", hint: "Setelah masa tenggang, data hanya bisa dilihat. Lewat ini, akses ditangguhkan sampai membayar." },
];

function toDraft(policy: Record<keyof Draft, number>): Draft {
  return {
    trialDays: String(policy.trialDays),
    expiringSoonDays: String(policy.expiringSoonDays),
    graceDays: String(policy.graceDays),
    readOnlyDays: String(policy.readOnlyDays),
  };
}

export default function SubscriptionPolicyPage() {
  const policy = useConsoleStore((s) => s.policy);
  const fetchPolicy = useConsoleStore((s) => s.fetchPolicy);
  const savePolicy = useConsoleStore((s) => s.savePolicy);
  const canEdit = usePlatformCan("super-admin");
  const [failed, setFailed] = useState(false);
  const form = useZodForm({ schema: subscriptionPolicyFormSchema, initial: { trialDays: "", expiringSoonDays: "", graceDays: "", readOnlyDays: "" } });
  const { values: draft, errors, reset } = form;
  const formRef = useRef<HTMLFormElement>(null);
  const [synced, setSynced] = useState(false);

  const load = useCallback(() => {
    setFailed(false);
    fetchPolicy().catch(() => setFailed(true));
  }, [fetchPolicy]);

  useEffect(() => {
    load();
  }, [load]);

  useEffect(() => {
    if (policy) {
      reset(toDraft(policy));
      setSynced(true);
    }
  }, [policy, reset]);

  // A failure (a 422 on a field, or anything else) is shown by the form; the toast is for when the save button is off-screen.
  const handleSave = form.handleSubmit(
    async (parsed) => {
      try {
        await savePolicy(parsed);
        toast.success("Kebijakan langganan disimpan");
      } catch (err) {
        toast.error(getApiErrorMessage(err, "Kebijakan belum bisa disimpan"));
        throw err;
      }
    },
    { root: () => formRef.current, fallback: "Kebijakan belum bisa disimpan", known: (key) => FIELDS.some((f) => f.key === key) },
  );

  if (failed && !policy) return <LoadError onRetry={load} />;
  if (!policy || !synced) return <div className="py-20 text-center text-sm text-text-secondary">Memuat kebijakan...</div>;

  const dirty = FIELDS.some((f) => draft[f.key] !== String(policy[f.key]));
  const graceEnd = Number(draft.graceDays) || 0;
  const readOnlyEnd = graceEnd + (Number(draft.readOnlyDays) || 0);

  return (
    <form ref={formRef} noValidate onSubmit={(e) => void handleSave(e)} className="flex max-w-3xl flex-col gap-5">
      <div>
        <h1 className="text-xl font-bold text-text-primary">Kebijakan Langganan</h1>
        <p className="mt-1 text-[13px] text-text-secondary">
          Aturan waktu yang sama untuk semua tenant: berapa lama trial, kapan mulai diingatkan, dan kapan akses dibatasi setelah langganan berakhir.
        </p>
      </div>

      <FormErrorSummary errors={form.errorList} formError={form.formError} labels={Object.fromEntries(FIELDS.map((f) => [f.key, f.label]))} root={() => formRef.current} />

      <Card>
        <CardHeader title="Durasi" subtitle={canEdit ? "Perubahan langsung berlaku untuk semua tenant." : "Hanya Super Admin yang dapat mengubah kebijakan ini."} />
        <CardContent className="flex flex-col gap-5">
          {FIELDS.map((f) => (
            <Field key={f.key} label={f.label} hint={f.hint} name={f.key} error={errors[f.key]} htmlFor={`policy-${f.key}`}>
              <Input
                id={`policy-${f.key}`}
                type="number"
                inputMode="numeric"
                className="max-w-40"
                value={draft[f.key]}
                disabled={!canEdit || form.submitting}
                onChange={(e) => form.set(f.key, e.target.value)}
                onBlur={() => form.blur(f.key)}
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
          <Button type="submit" disabled={!dirty || form.submitting}>
            {form.submitting ? "Menyimpan..." : "Simpan kebijakan"}
          </Button>
          {dirty && (
            <Button type="button" variant="secondary" disabled={form.submitting} onClick={() => reset(toDraft(policy))}>
              Batalkan perubahan
            </Button>
          )}
        </div>
      )}
    </form>
  );
}
