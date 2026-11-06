import { useEffect, useRef, useState } from "react";
import { ImagePlus } from "lucide-react";
import axios from "axios";
import { Modal } from "@/shared/components/ui/Modal";
import { Button } from "@/shared/components/ui/Button";
import { Field, Input } from "@/shared/components/ui/Input";
import { httpClient } from "@/shared/services/http-client";
import { API } from "@/shared/services/api-endpoints";
import { getApiErrorMessage } from "@/shared/lib/api-error";
import { compressFileForUpload } from "@/shared/lib/image-compression";
import { useTenantBrandingStore } from "@/shared/stores/useTenantBrandingStore";
import { profileSchema, type ProfileFormValues } from "@/modules/onboarding/schemas/profile.schema";

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
  const [values, setValues] = useState<ProfileFormValues>({ businessName: "", phone: "", city: "" });
  const [hasLogo, setHasLogo] = useState(false);
  const [logo, setLogo] = useState<File | null>(null);
  const [errors, setErrors] = useState<Partial<Record<keyof ProfileFormValues | "logo", string>>>({});
  const [formError, setFormError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const fileRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (!open) return;
    let cancelled = false;
    setErrors({});
    setFormError(null);
    setLogo(null);
    setLoading(true);
    httpClient
      .get(API.platform.tenantMe)
      .then((res) => {
        if (cancelled) return;
        const t = res.data.data as TenantProfile;
        setValues({ businessName: t.businessName, phone: displayPhone(t.phone), city: t.city });
        setHasLogo(t.hasLogo);
      })
      .catch((err) => !cancelled && setFormError(getApiErrorMessage(err, "Profil WO gagal dimuat.")))
      .finally(() => !cancelled && setLoading(false));
    return () => {
      cancelled = true;
    };
  }, [open]);

  function set<K extends keyof ProfileFormValues>(key: K, value: string) {
    setValues((prev) => ({ ...prev, [key]: value }));
    setErrors((prev) => ({ ...prev, [key]: undefined }));
  }

  function pickLogo(file: File | undefined) {
    if (!file) return;
    if (!LOGO_TYPES.includes(file.type)) {
      setErrors((e) => ({ ...e, logo: "Gunakan gambar PNG, JPEG, atau WebP" }));
      return;
    }
    if (file.size > MAX_LOGO_BYTES) {
      setErrors((e) => ({ ...e, logo: "Ukuran logo maksimal 2 MB" }));
      return;
    }
    setErrors((e) => ({ ...e, logo: undefined }));
    setLogo(file);
  }

  async function save() {
    const parsed = profileSchema.safeParse(values);
    if (!parsed.success) {
      const fieldErrors: Partial<Record<keyof ProfileFormValues, string>> = {};
      for (const issue of parsed.error.issues) fieldErrors[issue.path[0] as keyof ProfileFormValues] = issue.message;
      setErrors(fieldErrors);
      return;
    }
    setSaving(true);
    setFormError(null);
    try {
      await httpClient.patch(API.platform.tenantMeProfile, parsed.data);
      if (logo) {
        const payload = await compressFileForUpload(logo);
        await httpClient.put(API.platform.tenantMeLogo, payload);
      }
      void useTenantBrandingStore.getState().hydrate("WO Console");
      onSaved();
      onClose();
    } catch (err) {
      if (axios.isAxiosError(err) && err.response?.status === 422) {
        const fields = (err.response.data as { errors?: Record<string, string[]> }).errors ?? {};
        setErrors({ businessName: fields.businessName?.[0], phone: fields.phone?.[0], city: fields.city?.[0], logo: fields.mimeType?.[0] });
      }
      setFormError(getApiErrorMessage(err, "Profil belum tersimpan. Silakan coba lagi."));
    } finally {
      setSaving(false);
    }
  }

  return (
    <Modal
      open={open}
      onClose={onClose}
      title="Lengkapi profil WO"
      description="Nomor WhatsApp dipakai untuk tautan undangan dan pengingat ke client."
      size="sm"
      footer={
        <>
          <Button variant="secondary" onClick={onClose} disabled={saving}>
            Batal
          </Button>
          <Button onClick={() => void save()} disabled={saving || loading}>
            {saving ? "Menyimpan..." : "Simpan profil"}
          </Button>
        </>
      }
    >
      {loading ? (
        <p className="py-6 text-center text-sm text-text-secondary">Memuat profil...</p>
      ) : (
        <div className="flex flex-col gap-4">
          {formError && (
            <p role="alert" className="rounded-md border border-danger/30 bg-danger-soft px-3.5 py-2.5 text-[13px] font-medium text-danger">
              {formError}
            </p>
          )}
          <Field label="Nama WO" htmlFor="profile-name" required error={errors.businessName}>
            <Input id="profile-name" value={values.businessName} onChange={(e) => set("businessName", e.target.value)} maxLength={100} />
          </Field>
          <Field label="Nomor WhatsApp" htmlFor="profile-phone" required error={errors.phone} hint="Contoh: 0812 3456 7890">
            <Input
              id="profile-phone"
              value={values.phone}
              onChange={(e) => set("phone", e.target.value)}
              inputMode="tel"
              autoComplete="tel"
              placeholder="0812 3456 7890"
            />
          </Field>
          <Field label="Kota" htmlFor="profile-city" required error={errors.city}>
            <Input
              id="profile-city"
              value={values.city}
              onChange={(e) => set("city", e.target.value)}
              autoComplete="address-level2"
              placeholder="Contoh: Yogyakarta"
              maxLength={100}
            />
          </Field>
          <div className="flex flex-col gap-1.5">
            <span className="text-[13px] font-medium text-text-primary">Logo (opsional)</span>
            <div className="flex items-center gap-3">
              <Button variant="secondary" size="sm" icon={<ImagePlus className="h-4 w-4" />} onClick={() => fileRef.current?.click()}>
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
            {errors.logo && (
              <span role="alert" className="text-xs font-medium text-danger">
                {errors.logo}
              </span>
            )}
          </div>
        </div>
      )}
    </Modal>
  );
}
