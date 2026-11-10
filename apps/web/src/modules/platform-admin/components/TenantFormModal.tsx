import { useEffect, useRef, useState } from "react";
import { Modal } from "@/shared/components/ui/Modal";
import { Button } from "@/shared/components/ui/Button";
import { Input, Field } from "@/shared/components/ui/Input";
import { useZodForm } from "@/shared/forms/useZodForm";
import { FormErrorSummary } from "@/shared/forms/FormErrorSummary";
import {
  tenantSchema,
  tenantCreateSchema,
  type TenantFormValues,
  type TenantCreateFormValues,
} from "@/modules/platform-admin/schemas/tenant.schema";
import type { Tenant } from "@/modules/platform-admin/data/types";
import { usePlatformCan } from "@/modules/platform-admin/lib/capabilities";
import { httpClient } from "@/shared/services/http-client";
import { API } from "@/shared/services/api-endpoints";
import { compressFileForUpload, type CompressedFilePayload } from "@/shared/lib/image-compression";
import { getApiErrorMessage } from "@/shared/lib/api-error";

type FormState = TenantFormValues;

const FIELD_LABELS: Record<string, string> = {
  businessName: "Nama WO", city: "Kota", ownerName: "Nama owner", email: "Email akun Google Owner", phone: "No. HP Owner",
};

function toFormState(tenant?: Tenant): FormState {
  if (!tenant) {
    return {
      businessName: "", ownerName: "", email: "", phone: "", city: "",
    };
  }
  return {
    businessName: tenant.businessName,
    ownerName: tenant.ownerName,
    email: tenant.email,
    phone: tenant.phone,
    city: tenant.city,
  };
}

interface TenantFormModalProps {
  open: boolean;
  onClose: () => void;
  initialTenant?: Tenant;
  /** Saves, then closes the modal. Throw to keep it open: a 422 is shown on its fields, anything else inside the modal. */
  onSubmitCreate: (values: TenantCreateFormValues) => Promise<void>;
  onSubmitEdit: (values: TenantFormValues) => Promise<void>;
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
  const canChangeEmail = usePlatformCan("super-admin");
  const form = useZodForm({ schema: isEditing ? tenantSchema : tenantCreateSchema, initial: toFormState(initialTenant) });
  const { values, errors, set, blur } = form;
  const formRef = useRef<HTMLFormElement>(null);
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

  // The modal stays mounted while closed: start clean each time it opens.
  useEffect(() => {
    form.reset(toFormState(initialTenant));
    setLogoError(null);
  }, [initialTenant, open]);

  const submit = form.handleSubmit(
    async (parsed) => {
      if (isEditing) await onSubmitEdit(parsed as TenantFormValues);
      else await onSubmitCreate(parsed as TenantCreateFormValues);
    },
    { root: () => formRef.current, fallback: "Tenant belum tersimpan. Periksa koneksi lalu coba lagi." },
  );

  return (
    <Modal
      open={open}
      onClose={onClose}
      busy={form.submitting}
      title={isEditing ? "Ubah Tenant" : "Daftarkan Tenant Baru"}
      description={
        isEditing
          ? "Perbarui informasi bisnis dan owner tenant."
          : "Tenant beserta akun owner akan didaftarkan agar dapat mengakses WO Console."
      }
      footer={
        <>
          <Button variant="secondary" onClick={onClose} disabled={form.submitting}>
            Batal
          </Button>
          <Button type="submit" form="tenant-form" disabled={form.submitting}>
            {form.submitting ? "Menyimpan..." : isEditing ? "Simpan perubahan" : "Daftarkan tenant"}
          </Button>
        </>
      }
    >
      <form id="tenant-form" ref={formRef} noValidate onSubmit={(e) => void submit(e)} className="grid grid-cols-1 gap-4">
        <FormErrorSummary errors={form.errorList} formError={form.formError} labels={FIELD_LABELS} root={() => formRef.current} />
        <Field label="Nama WO" required name="businessName" error={errors.businessName}>
          <Input
            value={values.businessName}
            onChange={(e) => set("businessName", e.target.value)}
            onBlur={() => blur("businessName")}
            placeholder="cth. Anisa Wedding Organizer"
          />
        </Field>
        <Field label="Kota" required name="city" error={errors.city}>
          <Input value={values.city} onChange={(e) => set("city", e.target.value)} onBlur={() => blur("city")} placeholder="cth. Jakarta" />
        </Field>
        <Field label="Nama Owner" required name="ownerName" error={errors.ownerName}>
          <Input value={values.ownerName} onChange={(e) => set("ownerName", e.target.value)} onBlur={() => blur("ownerName")} placeholder="Nama owner WO" />
        </Field>
        <Field label="Email akun Google Owner" required name="email" error={errors.email} hint={isEditing ? (canChangeEmail ? undefined : "Hanya Super Admin yang dapat mengganti email Owner, karena email ini menentukan siapa yang menjadi Owner.") : "Owner menerima link undangan dan masuk dengan akun Google ini"}>
          <Input type="email" value={values.email} disabled={isEditing && !canChangeEmail} onChange={(e) => set("email", e.target.value)} onBlur={() => blur("email")} placeholder="owner@gmail.com" />
        </Field>
        <Field label="No. HP Owner" required name="phone" error={errors.phone}>
          <Input value={values.phone} onChange={(e) => set("phone", e.target.value)} onBlur={() => blur("phone")} placeholder="08xx-xxxx-xxxx" />
        </Field>
        {isEditing && (
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
        )}
      </form>
    </Modal>
  );
}
