import { useRef } from "react";
import { Drawer } from "@/shared/components/ui/Drawer";
import { Button } from "@/shared/components/ui/Button";
import { Field, Input, Textarea } from "@/shared/components/ui/Input";
import { httpClient } from "@/shared/services/http-client";
import { API } from "@/shared/services/api-endpoints";
import { useZodForm } from "@/shared/forms/useZodForm";
import { FormErrorSummary } from "@/shared/forms/FormErrorSummary";
import { cn } from "@/shared/lib/cn";
import { quoteSettingsSchema } from "@/modules/business-profile/schemas/quote-settings.schema";
import { exampleQuoteNumber, type QuoteSettings } from "@/modules/business-profile/components/quote-settings";

const FIELD_LABELS: Record<string, string> = {
  documentCode: "Kode usaha",
  isPkp: "Status pajak",
  defaultValidityDays: "Masa berlaku bawaan",
  defaultTermsText: "Syarat & ketentuan",
};

const TAX_OPTIONS = [
  { value: false, title: "Bukan PKP", detail: "Penawaran tanpa PPN." },
  { value: true, title: "PKP", detail: "PPN 12% × 11/12 ditambahkan otomatis ke setiap penawaran." },
];

/** Edits the Penawaran part of the business profile (DESAIN.md §6.8). */
export function QuoteSettingsDrawer({ initial, onClose, onSaved }: { initial: QuoteSettings; onClose: () => void; onSaved: (saved: QuoteSettings) => void }) {
  const form = useZodForm({
    schema: quoteSettingsSchema,
    initial: { ...initial, defaultValidityDays: String(initial.defaultValidityDays) },
  });
  const { values, errors, set, blur } = form;
  const formRef = useRef<HTMLFormElement>(null);

  const submit = form.handleSubmit(
    async (parsed) => {
      const res = await httpClient.put(API.platform.tenantMeQuoteSettings, { ...parsed, defaultValidityDays: Number(parsed.defaultValidityDays) });
      onSaved(res.data.data as QuoteSettings);
    },
    { root: () => formRef.current, fallback: "Pengaturan penawaran belum tersimpan. Periksa koneksi lalu coba lagi." },
  );

  return (
    <Drawer
      open
      onClose={onClose}
      busy={form.submitting}
      title="Penawaran"
      description="Dipakai setiap kali tim menyusun penawaran baru. Penawaran yang sudah dibuat tidak berubah."
      footer={
        <>
          <Button variant="secondary" onClick={onClose} disabled={form.submitting}>
            Batal
          </Button>
          <Button type="submit" form="quote-settings-form" disabled={form.submitting}>
            {form.submitting ? "Menyimpan..." : "Simpan pengaturan"}
          </Button>
        </>
      }
    >
      <form id="quote-settings-form" ref={formRef} noValidate onSubmit={(e) => void submit(e)} className="flex flex-col gap-5">
        <FormErrorSummary errors={form.errorList} formError={form.formError} labels={FIELD_LABELS} root={() => formRef.current} />
        <div className="flex flex-col gap-1.5">
          <Field label="Kode usaha" required name="documentCode" error={errors.documentCode} hint="2–5 huruf atau angka. Nomor yang sudah dibuat tidak berubah.">
            <Input
              className="w-28 font-semibold tracking-wide"
              value={values.documentCode}
              maxLength={5}
              autoCapitalize="characters"
              onChange={(e) => set("documentCode", e.target.value.toUpperCase().replace(/[^A-Z0-9]/g, ""))}
              onBlur={() => blur("documentCode")}
            />
          </Field>
          <p className="text-[13px] text-muted-foreground" aria-live="polite">
            Contoh nomor: <span className="font-semibold text-foreground tabular-nums">{exampleQuoteNumber(values.documentCode)}</span>
          </p>
        </div>

        <fieldset data-field="isPkp" className="flex flex-col gap-2">
          <legend className="mb-1.5 text-[13px] font-medium text-foreground">Status pajak</legend>
          {TAX_OPTIONS.map((o) => (
            <label
              key={String(o.value)}
              className={cn(
                "flex cursor-pointer items-start gap-3 rounded-lg border p-3 transition-colors",
                "has-[:focus-visible]:outline-2 has-[:focus-visible]:outline-offset-2 has-[:focus-visible]:outline-solid has-[:focus-visible]:outline-ring",
                values.isPkp === o.value ? "border-primary bg-primary/5" : "border-border",
              )}
            >
              <input type="radio" name="isPkp" className="mt-0.5 size-4 accent-primary" checked={values.isPkp === o.value} onChange={() => set("isPkp", o.value)} />
              <span className="flex flex-col">
                <span className="text-[13.5px] font-semibold text-foreground">{o.title}</span>
                <span className="text-[12.5px] text-muted-foreground">{o.detail}</span>
              </span>
            </label>
          ))}
        </fieldset>

        <Field label="Masa berlaku bawaan" required name="defaultValidityDays" error={errors.defaultValidityDays} hint="Berapa hari penawaran berlaku sejak dibuat; bisa diubah per penawaran.">
          <span className="flex items-center gap-2">
            <Input
              className="w-24 text-right tabular-nums"
              inputMode="numeric"
              aria-label="Masa berlaku bawaan dalam hari"
              value={values.defaultValidityDays}
              onChange={(e) => set("defaultValidityDays", e.target.value.replace(/\D/g, "").slice(0, 3))}
              onBlur={() => blur("defaultValidityDays")}
            />
            <span className="text-[13px] text-muted-foreground">hari</span>
          </span>
        </Field>

        <Field label="Syarat & ketentuan" name="defaultTermsText" error={errors.defaultTermsText} hint="Satu ketentuan per baris, bernomor. Tampil di bagian akhir setiap penawaran.">
          <Textarea rows={9} value={values.defaultTermsText} onChange={(e) => set("defaultTermsText", e.target.value)} onBlur={() => blur("defaultTermsText")} />
        </Field>
      </form>
    </Drawer>
  );
}
