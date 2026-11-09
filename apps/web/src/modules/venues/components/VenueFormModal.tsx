import { useEffect, useRef, useState } from "react";
import { FileText, ExternalLink } from "lucide-react";
import { Modal } from "@/shared/components/ui/Modal";
import { Button } from "@/shared/components/ui/Button";
import { Input, Textarea, Field } from "@/shared/components/ui/Input";
import { CurrencyInput } from "@/shared/components/ui/CurrencyInput";
import { Combobox } from "@/shared/components/ui/Combobox";
import { venueSchema, venueCreateSchema, type VenueFormValues, type VenueCreateFormValues } from "@/modules/venues/schemas/venue.schema";
import { CITIES } from "@/shared/constants/cities";
import type { Venue } from "@/modules/venues/types";
import { useVenueStore } from "@/modules/venues/stores/useVenueStore";
import { httpClient } from "@/shared/services/http-client";
import { API } from "@/shared/services/api-endpoints";
import { compressFileForUpload } from "@/shared/lib/image-compression";
import { getApiErrorMessage } from "@/shared/lib/api-error";
import { useZodForm } from "@/shared/forms/useZodForm";
import { FormErrorSummary } from "@/shared/forms/FormErrorSummary";

const FIELD_LABELS: Record<string, string> = {
  name: "Nama venue", picName: "Nama PIC", phonePic: "No. telepon PIC", phoneVenue: "No. telepon venue", email: "Email", city: "Kota",
  capacity: "Kapasitas", rentalPrice: "Harga sewa", charge: "Charge", address: "Alamat", facilities: "Fasilitas", socialMedia: "Sosial media",
};

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
  /** Saves, then closes the modal. Throw to keep it open: a 422 is shown on its fields, anything else inside the modal. */
  onSubmitCreate: (values: VenueCreateFormValues) => Promise<void>;
  onSubmitEdit: (values: VenueFormValues) => Promise<void>;
  initialVenue?: Venue;
}

export function VenueFormModal({ open, onClose, onSubmitCreate, onSubmitEdit, initialVenue }: VenueFormModalProps) {
  const isEditing = Boolean(initialVenue);
  const form = useZodForm({ schema: isEditing ? venueSchema : venueCreateSchema, initial: toFormValues(initialVenue) });
  const { values, errors, set, blur } = form;
  const formRef = useRef<HTMLFormElement>(null);

  const uploadVenueAttachment = useVenueStore((s) => s.uploadVenueAttachment);

  const [attachmentUploading, setAttachmentUploading] = useState(false);
  const [attachmentError, setAttachmentError] = useState<string | null>(null);
  const [hasAttachment, setHasAttachment] = useState(initialVenue?.hasAttachment ?? false);
  const [attachmentIsImage, setAttachmentIsImage] = useState(initialVenue?.attachmentIsImage ?? false);

  useEffect(() => {
    form.reset(toFormValues(initialVenue));
    setHasAttachment(initialVenue?.hasAttachment ?? false);
    setAttachmentIsImage(initialVenue?.attachmentIsImage ?? false);
    setAttachmentError(null);
  }, [initialVenue, open]);

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

  const submit = form.handleSubmit(
    async (parsed) => {
      if (isEditing) await onSubmitEdit(parsed as VenueFormValues);
      else await onSubmitCreate(parsed as VenueCreateFormValues);
    },
    { root: () => formRef.current, fallback: "Venue belum tersimpan. Periksa koneksi lalu coba lagi." },
  );

  return (
    <Modal
      open={open}
      onClose={onClose}
      title={isEditing ? "Ubah Venue" : "Tambah Venue Baru"}
      description="Informasi gedung/lokasi acara yang bekerja sama dengan WO."
      size="lg"
      busy={form.submitting}
      footer={
        <>
          <Button variant="secondary" onClick={onClose} disabled={form.submitting}>
            Batal
          </Button>
          <Button type="submit" form="venue-form" disabled={form.submitting}>
            {form.submitting ? "Menyimpan..." : isEditing ? "Simpan perubahan" : "Simpan venue"}
          </Button>
        </>
      }
    >
      <form id="venue-form" ref={formRef} noValidate onSubmit={(e) => void submit(e)} className="flex flex-col gap-4">
      <FormErrorSummary errors={form.errorList} formError={form.formError} labels={FIELD_LABELS} root={() => formRef.current} />
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <Field label="Nama Venue" required name="name" error={errors.name}>
          <Input value={values.name} onChange={(e) => set("name", e.target.value)} onBlur={() => blur("name")} placeholder="cth. Grand Ballroom Kemang" />
        </Field>
        <Field label="Nama PIC" required name="picName" error={errors.picName}>
          <Input value={values.picName} onChange={(e) => set("picName", e.target.value)} onBlur={() => blur("picName")} />
        </Field>
        <Field label="No Tlp PIC" required name="phonePic" error={errors.phonePic}>
          <Input value={values.phonePic} onChange={(e) => set("phonePic", e.target.value)} onBlur={() => blur("phonePic")} placeholder="0812-xxxx-xxxx (kontak personal)" />
        </Field>
        <Field label="No Tlp Venue" name="phoneVenue" error={errors.phoneVenue}>
          <Input value={values.phoneVenue} onChange={(e) => set("phoneVenue", e.target.value)} onBlur={() => blur("phoneVenue")} placeholder="Nomor resmi venue (opsional)" />
        </Field>
        <Field label="Email" name="email" error={errors.email}>
          <Input type="email" value={values.email} onChange={(e) => set("email", e.target.value)} onBlur={() => blur("email")} placeholder="Opsional" />
        </Field>
        <Field label="Kota" required={!isEditing} name="city" error={errors.city}>
          <Combobox value={values.city} onChange={(e) => set("city", e.target.value)} placeholder="Cari kota/kabupaten...">
            {CITIES.map((city) => (
              <option key={city} value={city}>
                {city}
              </option>
            ))}
          </Combobox>
        </Field>
        <Field label="Kapasitas (orang)" name="capacity" error={errors.capacity}>
          <Input type="number" min={0} value={values.capacity} onChange={(e) => set("capacity", Number(e.target.value))} onBlur={() => blur("capacity")} />
        </Field>
        <Field label="Harga Sewa (Rp)" required={!isEditing} name="rentalPrice" error={errors.rentalPrice}>
          <CurrencyInput value={values.rentalPrice} onChange={(n) => set("rentalPrice", n)} />
        </Field>
        <Field label="Charge (Rp)" name="charge" error={errors.charge}>
          <CurrencyInput value={values.charge} onChange={(n) => set("charge", n)} />
        </Field>
        <div className="sm:col-span-2">
          <Field label="Alamat" name="address" error={errors.address}>
            <Textarea rows={2} value={values.address} onChange={(e) => set("address", e.target.value)} onBlur={() => blur("address")} placeholder="Opsional" />
          </Field>
        </div>

        <div className="sm:col-span-2">
          <Field label="Fasilitas" name="facilities" error={errors.facilities}>
            <Textarea
              rows={2}
              value={values.facilities}
              onChange={(e) => set("facilities", e.target.value)} onBlur={() => blur("facilities")}
              placeholder="cth. Parkir, AC, Genset, kapasitas 500 kursi"
            />
          </Field>
        </div>

        <div className="sm:col-span-2">
          <Field label="Sosial Media" name="socialMedia" error={errors.socialMedia}>
            <Textarea
              rows={2}
              value={values.socialMedia}
              onChange={(e) => set("socialMedia", e.target.value)} onBlur={() => blur("socialMedia")}
              placeholder="cth. Instagram @grandballroom, grandballroom.com"
            />
          </Field>
        </div>

        <div className="sm:col-span-2">
          <Field label="Catatan" name="notes">
            <Textarea rows={3} value={values.notes} onChange={(e) => set("notes", e.target.value)} onBlur={() => blur("notes")} placeholder="Catatan tambahan mengenai venue ini" />
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
      </form>
    </Modal>
  );
}
