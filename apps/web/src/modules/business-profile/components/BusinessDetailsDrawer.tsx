import { useRef } from "react";
import { Drawer } from "@/shared/components/ui/Drawer";
import { Button } from "@/shared/components/ui/Button";
import { Input, Textarea, Field } from "@/shared/components/ui/Input";
import { httpClient } from "@/shared/services/http-client";
import { API } from "@/shared/services/api-endpoints";
import { useZodForm } from "@/shared/forms/useZodForm";
import { FormErrorSummary } from "@/shared/forms/FormErrorSummary";
import { businessDetailsSchema } from "@/modules/business-profile/schemas/business-details.schema";

export interface BusinessDetails {
  address: string;
  bankName: string;
  bankAccountNumber: string;
  bankAccountHolder: string;
}

const FIELD_LABELS: Record<string, string> = {
  address: "Alamat usaha", bankName: "Nama bank", bankAccountNumber: "Nomor rekening", bankAccountHolder: "Nama pemilik rekening",
};

/** Edits the address and the bank account clients pay into (shown in their portal). */
export function BusinessDetailsDrawer({
  initial,
  onClose,
  onSaved,
}: {
  initial: BusinessDetails;
  onClose: () => void;
  onSaved: (saved: BusinessDetails) => void;
}) {
  const form = useZodForm({ schema: businessDetailsSchema, initial });
  const { values, errors, set, blur } = form;
  const formRef = useRef<HTMLFormElement>(null);

  const submit = form.handleSubmit(
    async (parsed) => {
      const res = await httpClient.put(API.platform.tenantMeBusinessDetails, parsed);
      onSaved(res.data.data as BusinessDetails);
    },
    { root: () => formRef.current, fallback: "Belum tersimpan. Periksa koneksi lalu coba lagi." },
  );

  return (
    <Drawer
      open
      onClose={onClose}
      busy={form.submitting}
      title="Alamat dan rekening pembayaran"
      description="Klien melihat rekening ini di portal mereka saat ada tagihan. Kosongkan semuanya bila belum ingin dibagikan."
      footer={
        <>
          <Button variant="secondary" onClick={onClose} disabled={form.submitting}>Batal</Button>
          <Button type="submit" form="business-details-form" disabled={form.submitting}>{form.submitting ? "Menyimpan..." : "Simpan rekening"}</Button>
        </>
      }
    >
      <form id="business-details-form" ref={formRef} noValidate onSubmit={(e) => void submit(e)} className="flex flex-col gap-4">
        <FormErrorSummary errors={form.errorList} formError={form.formError} labels={FIELD_LABELS} root={() => formRef.current} />
        <Field label="Alamat usaha" name="address" error={errors.address}>
          <Textarea rows={3} value={values.address} onChange={(e) => set("address", e.target.value)} onBlur={() => blur("address")} />
        </Field>
        <Field label="Nama bank" name="bankName" error={errors.bankName}>
          <Input value={values.bankName} onChange={(e) => set("bankName", e.target.value)} onBlur={() => blur("bankName")} placeholder="cth. BCA" />
        </Field>
        <Field label="Nomor rekening" name="bankAccountNumber" error={errors.bankAccountNumber}>
          <Input
            value={values.bankAccountNumber}
            onChange={(e) => set("bankAccountNumber", e.target.value)}
            onBlur={() => blur("bankAccountNumber")}
            inputMode="numeric"
            autoComplete="off"
          />
        </Field>
        <Field label="Nama pemilik rekening" name="bankAccountHolder" error={errors.bankAccountHolder}>
          <Input value={values.bankAccountHolder} onChange={(e) => set("bankAccountHolder", e.target.value)} onBlur={() => blur("bankAccountHolder")} />
        </Field>
      </form>
    </Drawer>
  );
}
