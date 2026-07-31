import { useEffect, useState } from "react";
import { FileText, ExternalLink } from "lucide-react";
import { Modal } from "@/shared/components/ui/Modal";
import { Button } from "@/shared/components/ui/Button";
import { Input, Textarea, Field } from "@/shared/components/ui/Input";
import { Combobox } from "@/shared/components/ui/Combobox";
import { venueSchema, venueCreateSchema, type VenueFormValues, type VenueCreateFormValues } from "@/modules/venues/schemas/venue.schema";
import { CITIES } from "@/shared/constants/cities";
import type { Venue } from "@/modules/venues/types";
import { useVenueStore } from "@/modules/venues/stores/useVenueStore";
import { httpClient } from "@/shared/services/http-client";
import { API } from "@/shared/services/api-endpoints";
import { compressFileForUpload } from "@/shared/lib/image-compression";
import { getApiErrorMessage } from "@/shared/lib/api-error";

function toFormValues(venue?: Venue): VenueFormValues {
  if (!venue) {
    return {
      name: "", picName: "", phonePic: "", phoneVenue: "", email: "", address: "", city: "",
      rentalPrice: 0, charge: 0, capacity: 0, facilities: "", socialMedia: "", notes: "",
    };
  }
  return {
    name: venue.name, picName: venue.picName, phonePic: venue.phonePic,
    phoneVenue: venue.phoneVenue ?? "", email: venue.email ?? "", address: venue.address ?? "", city: venue.city ?? "",
    rentalPrice: venue.rentalPrice ?? 0, charge: venue.charge ?? 0, capacity: venue.capacity ?? 0,
    facilities: venue.facilities ?? "", socialMedia: venue.socialMedia ?? "", notes: venue.notes,
  };
}

interface VenueFormModalProps {
  open: boolean;
  onClose: () => void;
  onSubmitCreate: (values: VenueCreateFormValues) => void;
  onSubmitEdit: (values: VenueFormValues) => void;
  initialVenue?: Venue;
}

export function VenueFormModal({ open, onClose, onSubmitCreate, onSubmitEdit, initialVenue }: VenueFormModalProps) {
  const isEditing = Boolean(initialVenue);
  const [values, setValues] = useState<VenueFormValues>(() => toFormValues(initialVenue));
  const [errors, setErrors] = useState<Partial<Record<keyof VenueFormValues, string>>>({});

  const uploadVenueAttachment = useVenueStore((s) => s.uploadVenueAttachment);

  const [attachmentUploading, setAttachmentUploading] = useState(false);
  const [attachmentError, setAttachmentError] = useState<string | null>(null);
  const [hasAttachment, setHasAttachment] = useState(initialVenue?.hasAttachment ?? false);
  const [attachmentIsImage, setAttachmentIsImage] = useState(initialVenue?.attachmentIsImage ?? false);

  useEffect(() => {
    setValues(toFormValues(initialVenue));
    setErrors({});
    setHasAttachment(initialVenue?.hasAttachment ?? false);
    setAttachmentIsImage(initialVenue?.attachmentIsImage ?? false);
    setAttachmentError(null);
  }, [initialVenue, open]);

  function set<K extends keyof VenueFormValues>(key: K, value: VenueFormValues[K]) {
    setValues((prev) => ({ ...prev, [key]: value }));
  }

  async function handleAttachmentChange(file: File | undefined) {
    if (!file || !initialVenue) return;
    setAttachmentError(null);
    setAttachmentUploading(true);
    try {
      const compressed = await compressFileForUpload(file);
      await uploadVenueAttachment(initialVenue.id, compressed);
      setHasAttachment(true);
      setAttachmentIsImage(compressed.mimeType.startsWith("image/"));
    } catch (err) {
      setAttachmentError(getApiErrorMessage(err, "Gagal mengunggah lampiran"));
    } finally {
      setAttachmentUploading(false);
    }
  }

  async function handleViewAttachment() {
    if (!initialVenue) return;
    try {
      const res = await httpClient.get(API.venues.attachment(initialVenue.id), { responseType: "blob" });
      window.open(URL.createObjectURL(res.data as Blob), "_blank");
    } catch (err) {
      setAttachmentError(getApiErrorMessage(err, "Gagal membuka lampiran"));
    }
  }

  function handleSubmit() {
    const schema = isEditing ? venueSchema : venueCreateSchema;
    const result = schema.safeParse(values);
    if (!result.success) {
      const fieldErrors: Partial<Record<keyof VenueFormValues, string>> = {};
      for (const issue of result.error.issues) {
        fieldErrors[issue.path[0] as keyof VenueFormValues] = issue.message;
      }
      setErrors(fieldErrors);
      return;
    }
    setErrors({});
    if (isEditing) {
      onSubmitEdit(result.data);
    } else {
      onSubmitCreate(result.data as VenueCreateFormValues);
    }
  }

  return (
    <Modal
      open={open}
      onClose={onClose}
      title={isEditing ? "Ubah Venue" : "Tambah Venue Baru"}
      description="Informasi gedung/lokasi acara yang bekerja sama dengan WO."
      size="lg"
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>
            Batal
          </Button>
          <Button onClick={handleSubmit}>{isEditing ? "Simpan Perubahan" : "Simpan Venue"}</Button>
        </>
      }
    >
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <Field label="Nama Venue" required hint={errors.name}>
          <Input value={values.name} onChange={(e) => set("name", e.target.value)} placeholder="cth. Grand Ballroom Kemang" />
        </Field>
        <Field label="Nama PIC" required hint={errors.picName}>
          <Input value={values.picName} onChange={(e) => set("picName", e.target.value)} />
        </Field>
        <Field label="No Tlp PIC" required hint={errors.phonePic}>
          <Input value={values.phonePic} onChange={(e) => set("phonePic", e.target.value)} placeholder="0812-xxxx-xxxx (kontak personal)" />
        </Field>
        <Field label="No Tlp Venue" hint={errors.phoneVenue}>
          <Input value={values.phoneVenue} onChange={(e) => set("phoneVenue", e.target.value)} placeholder="Nomor resmi venue (opsional)" />
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
        <Field label="Kapasitas (orang)" hint={errors.capacity}>
          <Input type="number" min={0} value={values.capacity} onChange={(e) => set("capacity", Number(e.target.value))} />
        </Field>
        <Field label={`Harga Sewa (Rp)${isEditing ? "" : " *"}`} required={!isEditing} hint={errors.rentalPrice}>
          <Input type="number" min={0} value={values.rentalPrice} onChange={(e) => set("rentalPrice", Number(e.target.value))} />
        </Field>
        <Field label="Charge (Rp)" hint={errors.charge}>
          <Input type="number" min={0} value={values.charge} onChange={(e) => set("charge", Number(e.target.value))} />
        </Field>
        <div className="sm:col-span-2">
          <Field label="Alamat" hint={errors.address}>
            <Textarea rows={2} value={values.address} onChange={(e) => set("address", e.target.value)} placeholder="Opsional" />
          </Field>
        </div>

        <div className="sm:col-span-2">
          <Field label="Fasilitas" hint={errors.facilities}>
            <Textarea
              rows={2}
              value={values.facilities}
              onChange={(e) => set("facilities", e.target.value)}
              placeholder="cth. Parkir, AC, Genset, kapasitas 500 kursi"
            />
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
            <Textarea rows={3} value={values.notes} onChange={(e) => set("notes", e.target.value)} placeholder="Catatan tambahan mengenai venue ini" />
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
