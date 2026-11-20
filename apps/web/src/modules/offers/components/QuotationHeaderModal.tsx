import { useEffect, useRef } from "react";
import { Modal } from "@/shared/components/ui/Modal";
import { Button } from "@/shared/components/ui/Button";
import { Field, Input, Select, Textarea } from "@/shared/components/ui/Input";
import { FormErrorSummary } from "@/shared/forms/FormErrorSummary";
import { useZodForm } from "@/shared/forms/useZodForm";
import { isStaleConflict } from "@/shared/lib/concurrency";
import { todayISO } from "@/shared/lib/formatters";
import { toast } from "@/shared/stores/useToastStore";
import { useStaffStore } from "@/modules/users/stores/useStaffStore";
import { quotationHeaderSchema } from "@/modules/offers/schemas/quotation.schema";
import { useQuotationStore } from "@/modules/offers/stores/useQuotationStore";
import type { Quotation } from "@/modules/offers/types";

const FIELD_LABELS: Record<string, string> = {
  coupleNames: "Nama pasangan",
  contactWhatsapp: "WhatsApp",
  picStaffId: "PIC",
  validUntil: "Berlaku sampai",
  notes: "Catatan",
};

/** "Ubah data penawaran": the couple, the contact, the PIC, how long it holds and internal notes. */
export default function QuotationHeaderModal({ quotation, onClose }: { quotation: Quotation; onClose: () => void }) {
  const updateHeader = useQuotationStore((s) => s.updateHeader);
  const staff = useStaffStore((s) => s.staffSummaries);
  const fetchStaffSummaries = useStaffStore((s) => s.fetchStaffSummaries);
  const formRef = useRef<HTMLFormElement>(null);
  const form = useZodForm({
    schema: quotationHeaderSchema,
    initial: {
      coupleNames: quotation.coupleNames,
      contactWhatsapp: quotation.contactWhatsapp,
      picStaffId: quotation.picStaffId,
      validUntil: quotation.validUntil,
      notes: quotation.notes,
    },
  });
  const { values, errors, set, blur } = form;

  useEffect(() => {
    void fetchStaffSummaries().catch(() => undefined);
  }, [fetchStaffSummaries]);

  const submit = form.handleSubmit(
    async (v) => {
      try {
        await updateHeader(quotation.id, v, quotation.updatedAt);
      } catch (err) {
        if (isStaleConflict(err)) return;
        throw err;
      }
      toast.success("Data penawaran disimpan");
      onClose();
    },
    { root: () => formRef.current, fallback: "Data penawaran belum tersimpan. Periksa koneksi lalu coba lagi.", known: (key) => key in values },
  );

  const choices = staff.some((m) => m.id === quotation.picStaffId)
    ? staff
    : [{ id: quotation.picStaffId, name: quotation.picName || "PIC sekarang", title: "" }, ...staff];

  return (
    <Modal
      open
      onClose={onClose}
      title="Ubah data penawaran"
      busy={form.submitting}
      footer={
        <>
          <Button variant="secondary" onClick={onClose} disabled={form.submitting}>
            Batal
          </Button>
          <Button type="submit" form="quotation-header-form" disabled={form.submitting}>
            {form.submitting ? "Menyimpan..." : "Simpan perubahan"}
          </Button>
        </>
      }
    >
      <form id="quotation-header-form" ref={formRef} noValidate onSubmit={(e) => void submit(e)} className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <div className="sm:col-span-2">
          <FormErrorSummary errors={form.errorList} formError={form.formError} labels={FIELD_LABELS} root={() => formRef.current} />
        </div>
        <div className="sm:col-span-2">
          <Field label="Nama pasangan" required name="coupleNames" error={errors.coupleNames}>
            <Input value={values.coupleNames} onChange={(e) => set("coupleNames", e.target.value)} onBlur={() => blur("coupleNames")} />
          </Field>
        </div>
        <Field label="WhatsApp" name="contactWhatsapp" error={errors.contactWhatsapp}>
          <Input type="tel" inputMode="tel" value={values.contactWhatsapp} onChange={(e) => set("contactWhatsapp", e.target.value)} onBlur={() => blur("contactWhatsapp")} />
        </Field>
        <Field label="PIC" required name="picStaffId" error={errors.picStaffId}>
          <Select aria-label="PIC" value={values.picStaffId} onChange={(e) => set("picStaffId", e.target.value)}>
            {choices.map((m) => (
              <option key={m.id} value={m.id}>
                {m.name}
              </option>
            ))}
          </Select>
        </Field>
        <Field label="Berlaku sampai" name="validUntil" error={errors.validUntil} hint="Lewat tanggal ini penawaran menjadi Kedaluwarsa.">
          <Input type="date" min={todayISO()} value={values.validUntil} onChange={(e) => set("validUntil", e.target.value)} onBlur={() => blur("validUntil")} />
        </Field>
        <div className="sm:col-span-2">
          <Field label="Catatan internal" name="notes" error={errors.notes} hint="Hanya untuk tim, tidak tampil di PDF.">
            <Textarea rows={3} value={values.notes} onChange={(e) => set("notes", e.target.value)} onBlur={() => blur("notes")} />
          </Field>
        </div>
      </form>
    </Modal>
  );
}
