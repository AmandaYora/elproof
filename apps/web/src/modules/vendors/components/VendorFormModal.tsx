import { useEffect, useState } from "react";
import { FileText, ExternalLink } from "lucide-react";
import { Modal } from "@/shared/components/ui/Modal";
import { Button } from "@/shared/components/ui/Button";
import { Input, Textarea, Select, Field } from "@/shared/components/ui/Input";
import { Combobox } from "@/shared/components/ui/Combobox";
import { vendorSchema, vendorCreateSchema, type VendorFormValues, type VendorCreateFormValues } from "@/modules/vendors/schemas/vendor.schema";
import { CITIES } from "@/shared/constants/cities";
import type { Vendor } from "@/modules/vendors/types";
import type { VendorCategory } from "@/modules/vendor-categories/types";
import { useVendorStore } from "@/modules/vendors/stores/useVendorStore";
import { httpClient } from "@/shared/services/http-client";
import { API } from "@/shared/services/api-endpoints";
import { compressFileForUpload } from "@/shared/lib/image-compression";
import { getApiErrorMessage } from "@/shared/lib/api-error";

function toFormValues(vendor?: Vendor, defaultCategoryId = ""): VendorFormValues {
  if (!vendor) {
    return {
      name: "", categoryId: defaultCategoryId, picName: "", phone: "", email: "", socialMedia: "",
      city: "", address: "", priceAkad: 0, priceAkadResepsi: 0, notes: "",
    };
  }
  return {
    name: vendor.name, categoryId: vendor.categoryId, picName: vendor.picName, phone: vendor.phone,
    email: vendor.email ?? "", socialMedia: vendor.socialMedia ?? "", city: vendor.city ?? "",
    address: vendor.address ?? "", priceAkad: vendor.priceAkad ?? 0, priceAkadResepsi: vendor.priceAkadResepsi ?? 0,
    notes: vendor.notes,
  };
}

interface VendorFormModalProps {
  open: boolean;
  onClose: () => void;
  onSubmitCreate: (values: VendorCreateFormValues) => void;
  onSubmitEdit: (values: VendorFormValues) => void;
  initialVendor?: Vendor;
  categories: VendorCategory[];
}

export function VendorFormModal({ open, onClose, onSubmitCreate, onSubmitEdit, initialVendor, categories }: VendorFormModalProps) {
  const isEditing = Boolean(initialVendor);
  const [values, setValues] = useState<VendorFormValues>(() => toFormValues(initialVendor, categories[0]?.id ?? ""));
  const [errors, setErrors] = useState<Partial<Record<keyof VendorFormValues, string>>>({});

  const uploadVendorAttachment = useVendorStore((s) => s.uploadVendorAttachment);

  const [attachmentUploading, setAttachmentUploading] = useState(false);
  const [attachmentError, setAttachmentError] = useState<string | null>(null);
  const [hasAttachment, setHasAttachment] = useState(initialVendor?.hasAttachment ?? false);
  const [attachmentIsImage, setAttachmentIsImage] = useState(initialVendor?.attachmentIsImage ?? false);

  // Categories load asynchronously (Fase 3) — this modal is mounted once and
  // kept alive while closed, so backfill the default once real categories
  // arrive, without clobbering a value the user already picked.
  useEffect(() => {
    if (!initialVendor && !values.categoryId && categories.length > 0) {
      setValues((prev) => ({ ...prev, categoryId: categories[0].id }));
    }
  }, [categories.length, initialVendor]);

  useEffect(() => {
    setValues(toFormValues(initialVendor, categories[0]?.id ?? ""));
    setErrors({});
    setHasAttachment(initialVendor?.hasAttachment ?? false);
    setAttachmentIsImage(initialVendor?.attachmentIsImage ?? false);
    setAttachmentError(null);
  }, [initialVendor, open]);

  function set<K extends keyof VendorFormValues>(key: K, value: VendorFormValues[K]) {
    setValues((prev) => ({ ...prev, [key]: value }));
  }

  async function handleAttachmentChange(file: File | undefined) {
    if (!file || !initialVendor) return;
    setAttachmentError(null);
    setAttachmentUploading(true);
    try {
      const compressed = await compressFileForUpload(file);
      await uploadVendorAttachment(initialVendor.id, compressed);
      setHasAttachment(true);
      setAttachmentIsImage(compressed.mimeType.startsWith("image/"));
    } catch (err) {
      setAttachmentError(getApiErrorMessage(err, "Gagal mengunggah lampiran"));
    } finally {
      setAttachmentUploading(false);
    }
  }

  async function handleViewAttachment() {
    if (!initialVendor) return;
    try {
      const res = await httpClient.get(API.vendors.attachment(initialVendor.id), { responseType: "blob" });
      window.open(URL.createObjectURL(res.data as Blob), "_blank");
    } catch (err) {
      setAttachmentError(getApiErrorMessage(err, "Gagal membuka lampiran"));
    }
  }

  function handleSubmit() {
    const schema = isEditing ? vendorSchema : vendorCreateSchema;
    const result = schema.safeParse(values);
    if (!result.success) {
      const fieldErrors: Partial<Record<keyof VendorFormValues, string>> = {};
      for (const issue of result.error.issues) {
        fieldErrors[issue.path[0] as keyof VendorFormValues] = issue.message;
      }
      setErrors(fieldErrors);
      return;
    }
    setErrors({});
    if (isEditing) {
      onSubmitEdit(result.data);
    } else {
      onSubmitCreate(result.data as VendorCreateFormValues);
    }
  }

  return (
    <Modal
      open={open}
      onClose={onClose}
      title={isEditing ? "Ubah Vendor" : "Tambah Vendor Baru"}
      description="Informasi vendor yang bekerja sama dengan WO."
      size="lg"
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>
            Batal
          </Button>
          <Button onClick={handleSubmit}>{isEditing ? "Simpan Perubahan" : "Simpan Vendor"}</Button>
        </>
      }
    >
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <Field label="Nama Vendor" required hint={errors.name}>
          <Input value={values.name} onChange={(e) => set("name", e.target.value)} placeholder="cth. Grand Ballroom Kemang" />
        </Field>
        <Field label="Kategori" required hint={errors.categoryId}>
          <Select value={values.categoryId} onChange={(e) => set("categoryId", e.target.value)}>
            {categories.map((c) => (
              <option key={c.id} value={c.id}>
                {c.name}
              </option>
            ))}
          </Select>
        </Field>
        <Field label="Nama PIC" required hint={errors.picName}>
          <Input value={values.picName} onChange={(e) => set("picName", e.target.value)} />
        </Field>
        <Field label="No Tlp Vendor" required hint={errors.phone}>
          <Input value={values.phone} onChange={(e) => set("phone", e.target.value)} placeholder="0812-xxxx-xxxx" />
        </Field>
        <Field label="Email" hint={errors.email}>
          <Input type="email" value={values.email} onChange={(e) => set("email", e.target.value)} placeholder="Opsional" />
        </Field>
        <Field label={`Kota${isEditing ? "" : " *"}`} required={!isEditing} hint={errors.city}>
          <Combobox value={values.city} onChange={(e) => set("city", e.target.value)} placeholder="Cari kota/kabupaten...">
            {CITIES.map((city) => (
              <option key={city} value={city}>
                {city}
              </option>
            ))}
          </Combobox>
        </Field>
        <Field label={`Harga Akad (Rp)${isEditing ? "" : " *"}`} required={!isEditing} hint={errors.priceAkad}>
          <Input type="number" min={0} value={values.priceAkad} onChange={(e) => set("priceAkad", Number(e.target.value))} />
        </Field>
        <Field label={`Harga Akad+Resepsi (Rp)${isEditing ? "" : " *"}`} required={!isEditing} hint={errors.priceAkadResepsi}>
          <Input type="number" min={0} value={values.priceAkadResepsi} onChange={(e) => set("priceAkadResepsi", Number(e.target.value))} />
        </Field>
        <div className="sm:col-span-2">
          <Field label="Alamat" hint={errors.address}>
            <Textarea rows={2} value={values.address} onChange={(e) => set("address", e.target.value)} placeholder="Opsional" />
          </Field>
        </div>

        <div className="sm:col-span-2">
          <Field label="Sosial Media" hint={errors.socialMedia}>
            <Textarea
              rows={2}
              value={values.socialMedia}
              onChange={(e) => set("socialMedia", e.target.value)}
              placeholder="cth. Instagram @grandballroom, grandballroom.com"
            />
          </Field>
        </div>

        <div className="sm:col-span-2">
          <Field label="Catatan">
            <Textarea
              rows={3}
              value={values.notes}
              onChange={(e) => set("notes", e.target.value)}
              placeholder="Catatan tambahan mengenai vendor ini"
            />
          </Field>
        </div>

        {isEditing && (
          <div className="sm:col-span-2">
            <Field label="Lampiran" hint={attachmentError ?? undefined}>
              <div className="flex items-center gap-3">
                {hasAttachment && (
                  <Button
                    type="button"
                    variant="secondary"
                    size="sm"
                    icon={<ExternalLink className="h-3.5 w-3.5" />}
                    onClick={() => void handleViewAttachment()}
                  >
                    {attachmentIsImage ? "Lihat Foto" : "Lihat Dokumen"}
                  </Button>
                )}
                <label className="flex cursor-pointer items-center gap-2 text-[12.5px] font-semibold text-navy-900 hover:underline">
                  <FileText className="h-4 w-4" />
                  {attachmentUploading ? "Mengunggah..." : hasAttachment ? "Ganti Lampiran" : "Unggah Lampiran"}
                  <input
                    type="file"
                    accept="image/png,image/jpeg,image/webp,application/pdf"
                    disabled={attachmentUploading}
                    className="hidden"
                    onChange={(e) => void handleAttachmentChange(e.target.files?.[0])}
                  />
                </label>
              </div>
              <p className="mt-1.5 text-[12px] text-text-secondary">Dokumen atau foto, PNG/JPEG/WebP/PDF, maksimal 15 MB.</p>
            </Field>
          </div>
        )}
      </div>
    </Modal>
  );
}
