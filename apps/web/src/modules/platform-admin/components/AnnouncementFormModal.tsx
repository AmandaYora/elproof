import { useState } from "react";
import { Modal } from "@/shared/components/ui/Modal";
import { Button } from "@/shared/components/ui/Button";
import { Field, Input, Select, Textarea } from "@/shared/components/ui/Input";
import type { Announcement } from "@/modules/platform-admin/data/console.types";
import type { SubscriptionPlan } from "@/shared/data/subscriptionPlans";
import { TENANT_STATUS_FILTERS, TENANT_STATUS_LABEL } from "@/modules/platform-admin/lib/status";
import {
  ANNOUNCEMENT_AUDIENCES,
  ANNOUNCEMENT_KINDS,
  announcementSchema,
  apiTimeToLocal,
  localToApiTime,
  type AnnouncementFormValues,
} from "@/modules/platform-admin/schemas/console.schema";
import type { AnnouncementInput } from "@/modules/platform-admin/stores/useConsoleStore";

function nowLocal(): string {
  const d = new Date();
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

function toDraft(a?: Announcement): AnnouncementFormValues {
  if (!a) return { title: "", body: "", kind: "info", audience: "all", audienceValue: "", startsAt: nowLocal(), endsAt: "" };
  return {
    title: a.title,
    body: a.body,
    kind: a.kind,
    audience: a.audience,
    audienceValue: a.audienceValue,
    startsAt: apiTimeToLocal(a.startsAt),
    endsAt: apiTimeToLocal(a.endsAt),
  };
}

interface Props {
  open: boolean;
  onClose: () => void;
  initial?: Announcement;
  plans: SubscriptionPlan[];
  onSubmit: (input: AnnouncementInput) => Promise<void>;
  error: string | null;
  isSubmitting: boolean;
}

export function AnnouncementFormModal({ open, onClose, initial, plans, onSubmit, error, isSubmitting }: Props) {
  const [values, setValues] = useState<AnnouncementFormValues>(() => toDraft(initial));
  const [errors, setErrors] = useState<Record<string, string>>({});

  function set<K extends keyof AnnouncementFormValues>(key: K, value: AnnouncementFormValues[K]) {
    setValues((prev) => ({ ...prev, [key]: value }));
  }

  async function handleSubmit() {
    const parsed = announcementSchema.safeParse(values);
    if (!parsed.success) {
      const next: Record<string, string> = {};
      for (const issue of parsed.error.issues) next[String(issue.path[0])] ??= issue.message;
      setErrors(next);
      return;
    }
    setErrors({});
    const v = parsed.data;
    await onSubmit({
      title: v.title,
      body: v.body,
      kind: v.kind,
      audience: v.audience,
      audienceValue: v.audience === "all" ? "" : v.audienceValue,
      startsAt: localToApiTime(v.startsAt),
      endsAt: v.endsAt ? localToApiTime(v.endsAt) : null,
    });
  }

  return (
    <Modal
      open={open}
      onClose={onClose}
      busy={isSubmitting}
      error={error}
      title={initial ? "Ubah pengumuman" : "Buat pengumuman"}
      description="Tampil sebagai banner di WO Console tenant yang dituju. Setiap orang bisa menutupnya di browsernya."
      footer={
        <>
          <Button variant="secondary" onClick={onClose} disabled={isSubmitting}>
            Batal
          </Button>
          <Button onClick={() => void handleSubmit()} disabled={isSubmitting}>
            {isSubmitting ? "Menyimpan..." : initial ? "Simpan perubahan" : "Terbitkan"}
          </Button>
        </>
      }
    >
      <div className="grid grid-cols-1 gap-4">
        <Field label="Judul" required error={errors.title}>
          <Input value={values.title} onChange={(e) => set("title", e.target.value)} placeholder="cth. Pemeliharaan sistem Sabtu malam" aria-invalid={errors.title ? true : undefined} />
        </Field>
        <Field label="Isi pengumuman" required error={errors.body} hint="Tulis tanggal, jam (WIB), dan dampaknya bagi pengguna.">
          <Textarea rows={4} value={values.body} onChange={(e) => set("body", e.target.value)} aria-invalid={errors.body ? true : undefined} />
        </Field>
        <Field label="Jenis" required error={errors.kind}>
          <Select value={values.kind} onChange={(e) => set("kind", e.target.value as AnnouncementFormValues["kind"])}>
            {ANNOUNCEMENT_KINDS.map((k) => (
              <option key={k.value} value={k.value}>
                {k.label}
              </option>
            ))}
          </Select>
        </Field>
        <Field label="Ditujukan ke" required error={errors.audience}>
          <Select
            value={values.audience}
            onChange={(e) => setValues((prev) => ({ ...prev, audience: e.target.value as AnnouncementFormValues["audience"], audienceValue: "" }))}
          >
            {ANNOUNCEMENT_AUDIENCES.map((a) => (
              <option key={a.value} value={a.value}>
                {a.label}
              </option>
            ))}
          </Select>
        </Field>
        {values.audience === "status" && (
          <Field label="Status langganan" required error={errors.audienceValue}>
            <Select value={values.audienceValue} onChange={(e) => set("audienceValue", e.target.value)}>
              <option value="">Pilih status</option>
              {TENANT_STATUS_FILTERS.map((s) => (
                <option key={s} value={s}>
                  {TENANT_STATUS_LABEL[s]}
                </option>
              ))}
            </Select>
          </Field>
        )}
        {values.audience === "plan" && (
          <Field label="Paket" required error={errors.audienceValue}>
            <Select value={values.audienceValue} onChange={(e) => set("audienceValue", e.target.value)}>
              <option value="">Pilih paket</option>
              {plans.map((p) => (
                <option key={p.id} value={p.id}>
                  {p.name}
                </option>
              ))}
            </Select>
          </Field>
        )}
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <Field label="Mulai tayang (WIB)" required error={errors.startsAt}>
            <Input type="datetime-local" value={values.startsAt} onChange={(e) => set("startsAt", e.target.value)} aria-invalid={errors.startsAt ? true : undefined} />
          </Field>
          <Field label="Selesai tayang (WIB)" error={errors.endsAt} hint="Kosongkan agar tayang sampai dihapus.">
            <Input type="datetime-local" value={values.endsAt} onChange={(e) => set("endsAt", e.target.value)} aria-invalid={errors.endsAt ? true : undefined} />
          </Field>
        </div>
      </div>
    </Modal>
  );
}
