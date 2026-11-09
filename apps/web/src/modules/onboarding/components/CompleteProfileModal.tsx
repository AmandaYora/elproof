import { useEffect, useRef, useState } from "react";
import { ImagePlus } from "lucide-react";
import { Modal } from "@/shared/components/ui/Modal";
import { Button } from "@/shared/components/ui/Button";
import { Field, Input } from "@/shared/components/ui/Input";
import { httpClient } from "@/shared/services/http-client";
import { API } from "@/shared/services/api-endpoints";
import { getApiErrorMessage } from "@/shared/lib/api-error";
import { useZodForm } from "@/shared/forms/useZodForm";
import { FormErrorSummary } from "@/shared/forms/FormErrorSummary";
import { compressFileForUpload } from "@/shared/lib/image-compression";
import { useTenantBrandingStore } from "@/shared/stores/useTenantBrandingStore";
import { profileSchema } from "@/modules/onboarding/schemas/profile.schema";

const FIELD_LABELS: Record<string, string> = { businessName: "Nama WO", phone: "Nomor WhatsApp", city: "Kota" };
// The form's own keys: a server message about anything else (the logo's file type) is shown on top.
const FORM_KEYS = new Set(["businessName", "phone", "city"]);

const LOGO_TYPES = ["image/png", "image/jpeg", "image/webp"];
const MAX_LOGO_BYTES = 2 * 1024 * 1024;

interface TenantProfile {
  businessName: string;
  phone: string;
  city: string;
  hasLogo: boolean;
}

/** Shows 62812… as 0812… — the way people write their own number. */
function displayPhone(stored: string) {
  return stored.startsWith("62") ? "0" + stored.slice(2) : stored;
}

/**
 * "Lengkapi Profil" (PLAN.md §3 step 4): the Owner's WhatsApp number, city
 * and optional logo — asked after signup instead of on the signup form.
 */
export function CompleteProfileModal({ open, onClose, onSaved }: { open: boolean; onClose: () => void; onSaved: () => void }) {
  const form = useZodForm({ schema: profileSchema, initial: { businessName: "", phone: "", city: "" } });
  const { values, errors, set, blur } = form;
  const formRef = useRef<HTMLFormElement>(null);
  const [hasLogo, setHasLogo] = useState(false);
  const [logo, setLogo] = useState<File | null>(null);
  const [logoError, setLogoError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const fileRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (!open) return;
    let cancelled = false;
    form.reset({ businessName: "", phone: "", city: "" });
    setLogo(null);
    setLogoError(null);
    setLoading(true);
    httpClient
      .get(API.platform.tenantMe)
      .then((res) => {
        if (cancelled) return;
        const t = res.data.data as TenantProfile;
        form.reset({ businessName: t.businessName, phone: displayPhone(t.phone), city: t.city });
        setHasLogo(t.hasLogo);
      })
      .catch((err) => !cancelled && form.setFormError(getApiErrorMessage(err, "Profil WO gagal dimuat.")))
      .finally(() => !cancelled && setLoading(false));
    return () => {
      cancelled = true;
    };
  }, [open]);

  function pickLogo(file: File | undefined) {
    if (!file) return;
    if (!LOGO_TYPES.includes(file.type)) {
      setLogoError("Gunakan gambar PNG, JPEG, atau WebP");
      return;
    }
    if (file.size > MAX_LOGO_BYTES) {
      setLogoError("Ukuran logo maksimal 2 MB");
      return;
    }
    setLogoError(null);
    setLogo(file);
  }

  const submit = form.handleSubmit(
    async (parsed) => {
      await httpClient.patch(API.platform.tenantMeProfile, parsed);
      if (logo) {
        const payload = await compressFileForUpload(logo);
        await httpClient.put(API.platform.tenantMeLogo, payload);
      }
      void useTenantBrandingStore.getState().hydrate("WO Console");
      onSaved();
      onClose();
    },
    { root: () => formRef.current, fallback: "Profil belum tersimpan. Silakan coba lagi.", known: (key) => FORM_KEYS.has(key) },
  );

  return (
    <Modal
      open={open}
      onClose={onClose}
      title="Lengkapi profil WO"
      description="Nomor WhatsApp dipakai untuk tautan undangan dan pengingat ke client."
      size="sm"
      busy={form.submitting}
      footer={
        <>
          <Button variant="secondary" onClick={onClose} disabled={form.submitting}>
            Batal
          </Button>
          <Button type="submit" form="complete-profile-form" disabled={form.submitting || loading}>
            {form.submitting ? "Menyimpan..." : "Simpan profil"}
          </Button>
        </>
      }
    >
      {loading ? (
        <p className="py-6 text-center text-sm text-text-secondary">Memuat profil...</p>
      ) : (
        <form id="complete-profile-form" ref={formRef} noValidate onSubmit={(e) => void submit(e)} className="flex flex-col gap-4">
          <FormErrorSummary errors={form.errorList} formError={form.formError} labels={FIELD_LABELS} root={() => formRef.current} />
          <Field label="Nama WO" htmlFor="profile-name" required name="businessName" error={errors.businessName}>
            <Input id="profile-name" value={values.businessName} onChange={(e) => set("businessName", e.target.value)} onBlur={() => blur("businessName")} maxLength={100} />
          </Field>
          <Field label="Nomor WhatsApp" htmlFor="profile-phone" required name="phone" error={errors.phone} hint="Contoh: 0812 3456 7890">
            <Input
              id="profile-phone"
              value={values.phone}
              onChange={(e) => set("phone", e.target.value)}
              onBlur={() => blur("phone")}
              inputMode="tel"
              autoComplete="tel"
              placeholder="0812 3456 7890"
            />
          </Field>
          <Field label="Kota" htmlFor="profile-city" required name="city" error={errors.city}>
            <Input
              id="profile-city"
              value={values.city}
              onChange={(e) => set("city", e.target.value)}
              onBlur={() => blur("city")}
              autoComplete="address-level2"
              placeholder="Contoh: Yogyakarta"
              maxLength={100}
            />
          </Field>
          <div className="flex flex-col gap-1.5">
            <span className="text-[13px] font-medium text-text-primary">Logo (opsional)</span>
            <div className="flex items-center gap-3">
              <Button type="button" variant="secondary" size="sm" icon={<ImagePlus className="h-4 w-4" />} onClick={() => fileRef.current?.click()}>
                {logo || hasLogo ? "Ganti logo" : "Pilih logo"}
              </Button>
              <span className="truncate text-[12.5px] text-text-secondary">
                {logo ? logo.name : hasLogo ? "Logo sudah terpasang" : "PNG, JPEG, atau WebP, maks. 2 MB"}
              </span>
            </div>
            <input
              ref={fileRef}
              type="file"
              accept={LOGO_TYPES.join(",")}
              className="hidden"
              onChange={(e) => {
                pickLogo(e.target.files?.[0]);
                e.target.value = "";
              }}
            />
            {logoError && (
              <span role="alert" className="text-xs font-medium text-danger-text">
                {logoError}
              </span>
            )}
          </div>
        </form>
      )}
    </Modal>
  );
}
