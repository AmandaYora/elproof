import { useState } from "react";
import { Drawer } from "@/shared/components/ui/Drawer";
import { Button } from "@/shared/components/ui/Button";
import { Input, Textarea, Field } from "@/shared/components/ui/Input";
import { httpClient } from "@/shared/services/http-client";
import { API } from "@/shared/services/api-endpoints";
import { getApiErrorMessage, getApiFieldErrors } from "@/shared/lib/api-error";

export interface BusinessDetails {
  address: string;
  bankName: string;
  bankAccountNumber: string;
  bankAccountHolder: string;
}

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
  const [values, setValues] = useState<BusinessDetails>(initial);
  const [saving, setSaving] = useState(false);
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const [error, setError] = useState<string | null>(null);

  function set<K extends keyof BusinessDetails>(key: K, value: string) {
    setValues((prev) => ({ ...prev, [key]: value }));
  }

  async function save() {
    setSaving(true);
    setError(null);
    setFieldErrors({});
    try {
      const res = await httpClient.put(API.platform.tenantMeBusinessDetails, values);
      onSaved(res.data.data as BusinessDetails);
    } catch (err) {
      const fields = getApiFieldErrors(err);
      setFieldErrors(fields);
      setError(Object.keys(fields).length > 0 ? "Lengkapi dulu bagian yang ditandai." : getApiErrorMessage(err, "Belum tersimpan. Periksa koneksi lalu coba lagi."));
    } finally {
      setSaving(false);
    }
  }

  const invalid = (key: string) => (fieldErrors[key] ? true : undefined);

  return (
    <Drawer
      open
      onClose={onClose}
      busy={saving}
      title="Alamat dan rekening pembayaran"
      description="Klien melihat rekening ini di portal mereka saat ada tagihan. Kosongkan semuanya bila belum ingin dibagikan."
      footer={
        <>
          <Button variant="secondary" onClick={onClose} disabled={saving}>Batal</Button>
          <Button onClick={() => void save()} disabled={saving}>{saving ? "Menyimpan..." : "Simpan"}</Button>
        </>
      }
    >
      <form
        noValidate
        onSubmit={(e) => {
          e.preventDefault();
          void save();
        }}
        className="flex flex-col gap-4"
      >
        <Field label="Alamat usaha" error={fieldErrors.address}>
          <Textarea rows={3} value={values.address} onChange={(e) => set("address", e.target.value)} aria-invalid={invalid("address")} />
        </Field>
        <Field label="Nama bank" error={fieldErrors.bankName}>
          <Input value={values.bankName} onChange={(e) => set("bankName", e.target.value)} aria-invalid={invalid("bankName")} placeholder="cth. BCA" />
        </Field>
        <Field label="Nomor rekening" error={fieldErrors.bankAccountNumber}>
          <Input
            value={values.bankAccountNumber}
            onChange={(e) => set("bankAccountNumber", e.target.value)}
            aria-invalid={invalid("bankAccountNumber")}
            inputMode="numeric"
            autoComplete="off"
          />
        </Field>
        <Field label="Nama pemilik rekening" error={fieldErrors.bankAccountHolder}>
          <Input value={values.bankAccountHolder} onChange={(e) => set("bankAccountHolder", e.target.value)} aria-invalid={invalid("bankAccountHolder")} />
        </Field>
        {error && (
          <p role="alert" className="rounded-md border border-danger/30 bg-danger-soft px-3.5 py-2.5 text-[13px] font-medium text-danger">{error}</p>
        )}
        <button type="submit" className="hidden" aria-hidden tabIndex={-1} />
      </form>
    </Drawer>
  );
}
