import { useEffect, useRef } from "react";
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
import { useZodForm } from "@/shared/forms/useZodForm";
import { FormErrorSummary } from "@/shared/forms/FormErrorSummary";

const FIELD_LABELS: Record<string, string> = {
  title: "Judul", body: "Isi pengumuman", kind: "Jenis", audience: "Ditujukan ke", audienceValue: "Tujuan pengumuman", startsAt: "Mulai tayang", endsAt: "Selesai tayang",
};

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
  /** Saves, then closes the modal. Throw to keep it open: the message is shown inside it, a 422 on its fields. */
  onSubmit: (input: AnnouncementInput) => Promise<void>;
}

export function AnnouncementFormModal({ open, onClose, initial, plans, onSubmit }: Props) {
  const form = useZodForm({ schema: announcementSchema, initial: toDraft(initial) });
  const { values, errors, set, blur } = form;
  const formRef = useRef<HTMLFormElement>(null);

  // The modal stays mounted while closed: start clean (and with a fresh start time) each time it opens.
  useEffect(() => {
    form.reset(toDraft(initial));
  }, [initial, open]);

  const submit = form.handleSubmit(
    async (v) => {
      await onSubmit({
        title: v.title,
        body: v.body,
        kind: v.kind,
        audience: v.audience,
        audienceValue: v.audience === "all" ? "" : v.audienceValue,
        startsAt: localToApiTime(v.startsAt),
        endsAt: v.endsAt ? localToApiTime(v.endsAt) : null,
      });
    },
    { root: () => formRef.current, fallback: "Pengumuman belum tersimpan. Periksa koneksi lalu coba lagi." },
  );

  return (
    <Modal
      open={open}
      onClose={onClose}
      busy={form.submitting}
      title={initial ? "Ubah pengumuman" : "Buat pengumuman"}
      description="Tampil sebagai banner di WO Console tenant yang dituju. Setiap orang bisa menutupnya di browsernya."
      footer={
        <>
          <Button variant="secondary" onClick={onClose} disabled={form.submitting}>
            Batal
          </Button>
          <Button type="submit" form="announcement-form" disabled={form.submitting}>
            {form.submitting ? "Menyimpan..." : initial ? "Simpan perubahan" : "Terbitkan pengumuman"}
          </Button>
        </>
      }
    >
      <form id="announcement-form" ref={formRef} noValidate onSubmit={(e) => void submit(e)} className="grid grid-cols-1 gap-4">
        <FormErrorSummary errors={form.errorList} formError={form.formError} labels={FIELD_LABELS} root={() => formRef.current} />
        <Field label="Judul" required name="title" error={errors.title}>
          <Input value={values.title} onChange={(e) => set("title", e.target.value)} onBlur={() => blur("title")} placeholder="cth. Pemeliharaan sistem Sabtu malam" />
        </Field>
        <Field label="Isi pengumuman" required name="body" error={errors.body} hint="Tulis tanggal, jam (WIB), dan dampaknya bagi pengguna.">
          <Textarea rows={4} value={values.body} onChange={(e) => set("body", e.target.value)} onBlur={() => blur("body")} />
        </Field>
        <Field label="Jenis" required name="kind" error={errors.kind}>
          <Select value={values.kind} onChange={(e) => set("kind", e.target.value as AnnouncementFormValues["kind"])}>
            {ANNOUNCEMENT_KINDS.map((k) => (
              <option key={k.value} value={k.value}>
                {k.label}
              </option>
            ))}
          </Select>
        </Field>
        <Field label="Ditujukan ke" required name="audience" error={errors.audience}>
          <Select
            value={values.audience}
            onChange={(e) => {
              set("audience", e.target.value as AnnouncementFormValues["audience"]);
              set("audienceValue", "");
            }}
          >
            {ANNOUNCEMENT_AUDIENCES.map((a) => (
              <option key={a.value} value={a.value}>
                {a.label}
              </option>
            ))}
          </Select>
        </Field>
        {values.audience === "status" && (
          <Field label="Status langganan" required name="audienceValue" error={errors.audienceValue}>
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
          <Field label="Paket" required name="audienceValue" error={errors.audienceValue}>
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
          <Field label="Mulai tayang (WIB)" required name="startsAt" error={errors.startsAt}>
            <Input type="datetime-local" value={values.startsAt} onChange={(e) => set("startsAt", e.target.value)} onBlur={() => blur("startsAt")} />
          </Field>
          <Field label="Selesai tayang (WIB)" name="endsAt" error={errors.endsAt} hint="Kosongkan agar tayang sampai dihapus.">
            <Input type="datetime-local" value={values.endsAt} onChange={(e) => set("endsAt", e.target.value)} onBlur={() => blur("endsAt")} />
          </Field>
        </div>
      </form>
    </Modal>
  );
}
