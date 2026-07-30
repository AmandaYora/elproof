import { useEffect, useRef, useState } from "react";
import { Eye, EyeOff, Check } from "lucide-react";
import { Modal } from "@/shared/components/ui/Modal";
import { Button } from "@/shared/components/ui/Button";
import { Input, Field } from "@/shared/components/ui/Input";
import {
  tenantSchema,
  tenantCreateSchema,
  type TenantFormValues,
  type TenantCreateFormValues,
} from "@/modules/platform-admin/schemas/tenant.schema";
import type { Tenant } from "@/modules/platform-admin/data/types";
import { httpClient } from "@/shared/services/http-client";
import { API } from "@/shared/services/api-endpoints";
import { compressFileForUpload, type CompressedFilePayload } from "@/shared/lib/image-compression";
import { getApiErrorMessage } from "@/shared/lib/api-error";
import {
  BRAND_COLOR_PRESET_KEYS,
  BRAND_COLOR_PRESETS,
  BRAND_COLOR_PRESET_LABELS,
  type BrandColorPresetKey,
} from "@/theme/brandPresets";

interface FormState extends TenantFormValues {
  username: string;
  password: string;
  confirmPassword: string;
}

function toFormState(tenant?: Tenant): FormState {
  if (!tenant) {
    return {
      businessName: "", ownerName: "", email: "", phone: "", city: "",
      brandColorPreset: "navy", customDomain: "", username: "", password: "", confirmPassword: "",
    };
  }
  return {
    businessName: tenant.businessName,
    ownerName: tenant.ownerName,
    email: tenant.email,
    phone: tenant.phone,
    city: tenant.city,
    brandColorPreset: (tenant.brandColorPreset as BrandColorPresetKey) || "navy",
    customDomain: tenant.customDomain ?? "",
    username: tenant.username,
    password: "",
    confirmPassword: "",
  };
}

interface TenantFormModalProps {
  open: boolean;
  onClose: () => void;
  initialTenant?: Tenant;
  onSubmitCreate: (values: TenantCreateFormValues) => void;
  onSubmitEdit: (values: TenantFormValues) => void;
  onUploadLogo: (id: string, file: CompressedFilePayload) => Promise<void>;
}

export function TenantFormModal({
  open,
  onClose,
  initialTenant,
  onSubmitCreate,
  onSubmitEdit,
  onUploadLogo,
}: TenantFormModalProps) {
  const isEditing = Boolean(initialTenant);
  const [values, setValues] = useState<FormState>(() => toFormState(initialTenant));
  const [errors, setErrors] = useState<Partial<Record<keyof FormState, string>>>({});
  const [showPassword, setShowPassword] = useState(false);
  const [logoPreviewUrl, setLogoPreviewUrl] = useState<string | null>(null);
  const [logoUploading, setLogoUploading] = useState(false);
  const [logoError, setLogoError] = useState<string | null>(null);
  // Tracks the object URL created locally from a freshly-picked file (as
  // opposed to the one fetched from the server, which the effect below
  // manages on its own) so re-picking a file — or unmounting — revokes the
  // previous one instead of leaking it.
  const localPreviewUrlRef = useRef<string | null>(null);
  const mountedRef = useRef(true);
  useEffect(() => {
    mountedRef.current = true;
    return () => {
      mountedRef.current = false;
      if (localPreviewUrlRef.current) URL.revokeObjectURL(localPreviewUrlRef.current);
    };
  }, []);

  // Loads the tenant's existing logo (if any) as a preview — the download
  // endpoint requires the platform_admin Bearer token (same reasoning as
  // EvidenceViewerModal), so it's fetched as a blob rather than a bare <img src>.
  useEffect(() => {
    if (!open || !initialTenant?.hasLogo) {
      setLogoPreviewUrl(null);
      return;
    }
    let objectUrl: string | null = null;
    let cancelled = false;
    httpClient
      .get(API.platform.tenantLogo(initialTenant.id), { responseType: "blob" })
      .then((res) => {
        if (cancelled) return;
        objectUrl = URL.createObjectURL(res.data as Blob);
        setLogoPreviewUrl(objectUrl);
      })
      .catch(() => {
        if (!cancelled) setLogoPreviewUrl(null);
      });
    return () => {
      cancelled = true;
      if (objectUrl) URL.revokeObjectURL(objectUrl);
    };
  }, [open, initialTenant?.id, initialTenant?.hasLogo]);

  async function handleLogoChange(file: File | undefined) {
    if (!file || !initialTenant) return;
    setLogoError(null);
    setLogoUploading(true);
    if (localPreviewUrlRef.current) URL.revokeObjectURL(localPreviewUrlRef.current);
    const localPreview = URL.createObjectURL(file);
    localPreviewUrlRef.current = localPreview;
    setLogoPreviewUrl(localPreview);
    try {
      const compressed = await compressFileForUpload(file);
      await onUploadLogo(initialTenant.id, compressed);
    } catch (err) {
      if (mountedRef.current) setLogoError(getApiErrorMessage(err, "Gagal mengunggah logo"));
    } finally {
      if (mountedRef.current) setLogoUploading(false);
    }
  }

  function set<K extends keyof FormState>(key: K, value: FormState[K]) {
    setValues((prev) => ({ ...prev, [key]: value }));
  }

  function handleSubmit() {
    if (isEditing) {
      const result = tenantSchema.safeParse(values);
      if (!result.success) {
        const fieldErrors: Partial<Record<keyof FormState, string>> = {};
        for (const issue of result.error.issues) {
          fieldErrors[issue.path[0] as keyof FormState] = issue.message;
        }
        setErrors(fieldErrors);
        return;
      }
      onSubmitEdit(result.data);
      setErrors({});
      return;
    }

    const result = tenantCreateSchema.safeParse(values);
    if (!result.success) {
      const fieldErrors: Partial<Record<keyof FormState, string>> = {};
      for (const issue of result.error.issues) {
        fieldErrors[issue.path[0] as keyof FormState] = issue.message;
      }
      setErrors(fieldErrors);
      return;
    }
    onSubmitCreate(result.data);
    setErrors({});
  }

  function handleClose() {
    setValues(toFormState(initialTenant));
    setErrors({});
    setLogoError(null);
    onClose();
  }

  return (
    <Modal
      open={open}
      onClose={handleClose}
      title={isEditing ? "Ubah Tenant" : "Daftarkan Tenant Baru"}
      description={
        isEditing
          ? "Perbarui informasi bisnis dan owner tenant."
          : "Tenant beserta akun owner akan didaftarkan agar dapat mengakses WO Console."
      }
      footer={
        <>
          <Button variant="secondary" onClick={handleClose}>
            Batal
          </Button>
          <Button onClick={handleSubmit}>{isEditing ? "Simpan Perubahan" : "Daftarkan Tenant"}</Button>
        </>
      }
    >
      <div className="grid grid-cols-1 gap-4">
        <Field label="Nama WO" required hint={errors.businessName}>
          <Input
            value={values.businessName}
            onChange={(e) => set("businessName", e.target.value)}
            placeholder="cth. Anisa Wedding Organizer"
          />
        </Field>
        <Field label="Kota" required hint={errors.city}>
          <Input value={values.city} onChange={(e) => set("city", e.target.value)} placeholder="cth. Jakarta" />
        </Field>
        <Field label="Nama Owner" required hint={errors.ownerName}>
          <Input value={values.ownerName} onChange={(e) => set("ownerName", e.target.value)} placeholder="Nama owner WO" />
        </Field>
        <Field label="Email Owner" required hint={errors.email}>
          <Input type="email" value={values.email} onChange={(e) => set("email", e.target.value)} placeholder="owner@contoh.id" />
        </Field>
        <Field label="No. HP Owner" required hint={errors.phone}>
          <Input value={values.phone} onChange={(e) => set("phone", e.target.value)} placeholder="08xx-xxxx-xxxx" />
        </Field>
        <Field label="Username" required={!isEditing} hint={errors.username}>
          <Input
            value={values.username}
            disabled={isEditing}
            onChange={(e) => set("username", e.target.value)}
            placeholder="cth. budi.rahman"
          />
        </Field>

        {!isEditing && (
          <>
            <Field label="Password Owner" required hint={errors.password}>
              <div className="relative">
                <Input
                  type={showPassword ? "text" : "password"}
                  value={values.password}
                  onChange={(e) => set("password", e.target.value)}
                  placeholder="Minimal 8 karakter"
                  className="pr-10"
                />
                <button
                  type="button"
                  onClick={() => setShowPassword((v) => !v)}
                  aria-label={showPassword ? "Sembunyikan password" : "Tampilkan password"}
                  className="absolute right-3 top-1/2 -translate-y-1/2 text-text-secondary hover:text-text-primary"
                >
                  {showPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                </button>
              </div>
            </Field>
            <Field label="Konfirmasi Password" required hint={errors.confirmPassword}>
              <Input
                type={showPassword ? "text" : "password"}
                value={values.confirmPassword}
                onChange={(e) => set("confirmPassword", e.target.value)}
                placeholder="Ulangi password"
              />
            </Field>
          </>
        )}

        {isEditing && (
          <>
            <Field label="Warna Brand" required hint={errors.brandColorPreset}>
              <div className="grid grid-cols-5 gap-2">
                {BRAND_COLOR_PRESET_KEYS.map((key) => {
                  const selected = values.brandColorPreset === key;
                  return (
                    <button
                      key={key}
                      type="button"
                      onClick={() => set("brandColorPreset", key)}
                      title={BRAND_COLOR_PRESET_LABELS[key]}
                      aria-label={BRAND_COLOR_PRESET_LABELS[key]}
                      aria-pressed={selected}
                      className={`flex h-9 w-9 items-center justify-center rounded-full ring-offset-2 transition-shadow ${
                        selected ? "ring-2 ring-navy-900" : "hover:ring-2 hover:ring-border"
                      }`}
                      style={{ backgroundColor: BRAND_COLOR_PRESETS[key][900] }}
                    >
                      {selected && <Check className="h-4 w-4 text-white" />}
                    </button>
                  );
                })}
              </div>
            </Field>

            <Field label="Domain Kustom" hint={errors.customDomain}>
              <Input
                value={values.customDomain}
                onChange={(e) => set("customDomain", e.target.value)}
                placeholder="cth. app.namabisnis.com"
              />
              <p className="mt-1.5 text-[12px] text-text-secondary">
                Kosongkan jika tenant belum punya domainnya sendiri. Arahkan DNS domain ke server terlebih dahulu.
              </p>
            </Field>

            <Field label="Logo" hint={logoError ?? undefined}>
              <div className="flex items-center gap-3">
                {logoPreviewUrl && (
                  <img
                    src={logoPreviewUrl}
                    alt="Logo tenant"
                    className="h-12 w-12 shrink-0 rounded-md border border-border object-contain p-1"
                  />
                )}
                <input
                  type="file"
                  accept="image/png,image/jpeg,image/webp"
                  disabled={logoUploading}
                  onChange={(e) => void handleLogoChange(e.target.files?.[0])}
                  className="block w-full text-[13px] text-text-secondary file:mr-3 file:rounded-md file:border-0 file:bg-navy-900 file:px-3 file:py-1.5 file:text-[12.5px] file:font-semibold file:text-white"
                />
              </div>
              <p className="mt-1.5 text-[12px] text-text-secondary">
                PNG, JPEG, atau WebP, maksimal 2 MB. Diunggah langsung setelah dipilih.
              </p>
            </Field>
          </>
        )}
      </div>
    </Modal>
  );
}
