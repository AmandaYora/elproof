import { useEffect, useRef } from "react";
import { useNavigate } from "react-router-dom";
import { Modal } from "@/shared/components/ui/Modal";
import { Button } from "@/shared/components/ui/Button";
import { Field, Input, Select } from "@/shared/components/ui/Input";
import { FormErrorSummary } from "@/shared/forms/FormErrorSummary";
import { useZodForm } from "@/shared/forms/useZodForm";
import { useAuthStore } from "@/shared/stores/useAuthStore";
import { toast } from "@/shared/stores/useToastStore";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import { useStaffStore } from "@/modules/users/stores/useStaffStore";
import { quotationHeaderSchema } from "@/modules/offers/schemas/quotation.schema";
import { useQuotationStore } from "@/modules/offers/stores/useQuotationStore";

const FIELD_LABELS: Record<string, string> = { coupleNames: "Nama pasangan", contactWhatsapp: "WhatsApp", picStaffId: "PIC" };

interface Props {
  open: boolean;
  onClose: () => void;
}

/**
 * "Penawaran baru" (DESAIN A1): only who it is for and who handles it. The number, the validity (today + the
 * workspace's default) and the default terms are made by the server; everything else is filled in on the page.
 */
export default function QuotationCreateModal({ open, onClose }: Props) {
  const navigate = useNavigate();
  const currentStaffId = useAuthStore((s) => s.currentStaffId);
  const staff = useStaffStore((s) => s.staffSummaries);
  const fetchStaffSummaries = useStaffStore((s) => s.fetchStaffSummaries);
  const createQuotation = useQuotationStore((s) => s.createQuotation);
  const formRef = useRef<HTMLFormElement>(null);
  const initial = { coupleNames: "", contactWhatsapp: "", picStaffId: currentStaffId, validUntil: "", notes: "" };
  const form = useZodForm({ schema: quotationHeaderSchema, initial });
  const { values, errors, set, blur } = form;

  useEffect(() => {
    void fetchStaffSummaries().catch(() => undefined);
  }, [fetchStaffSummaries]);

  useEffect(() => {
    if (open) form.reset({ ...initial, picStaffId: currentStaffId });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, currentStaffId]);

  const submit = form.handleSubmit(
    async (v) => {
      const detail = await createQuotation(v);
      toast.success("Penawaran dibuat");
      onClose();
      navigate(ROUTE_PATHS.quotationDetail(detail.quotation.id));
    },
    { root: () => formRef.current, fallback: "Penawaran belum dibuat. Periksa koneksi lalu coba lagi." },
  );

  // The person creating it is always a choice, even before the team list arrives.
  const choices = staff.some((m) => m.id === currentStaffId) ? staff : [{ id: currentStaffId, name: "Saya", title: "" }, ...staff];

  return (
    <Modal
      open={open}
      onClose={onClose}
      title="Penawaran baru"
      description="Nomor penawaran dan masa berlakunya dibuat otomatis. Acara, rincian, dan termin diisi di halaman berikutnya."
      busy={form.submitting}
      footer={
        <>
          <Button variant="secondary" onClick={onClose} disabled={form.submitting}>
            Batal
          </Button>
          <Button type="submit" form="quotation-create-form" disabled={form.submitting}>
            {form.submitting ? "Membuat..." : "Buat penawaran"}
          </Button>
        </>
      }
    >
      <form id="quotation-create-form" ref={formRef} noValidate onSubmit={(e) => void submit(e)} className="grid grid-cols-1 gap-4">
        <FormErrorSummary errors={form.errorList} formError={form.formError} labels={FIELD_LABELS} root={() => formRef.current} />
        <Field label="Nama pasangan" required name="coupleNames" error={errors.coupleNames}>
          <Input
            value={values.coupleNames}
            onChange={(e) => set("coupleNames", e.target.value)}
            onBlur={() => blur("coupleNames")}
            placeholder="cth. Citra & Bima"
            autoComplete="off"
          />
        </Field>
        <Field label="WhatsApp" name="contactWhatsapp" error={errors.contactWhatsapp} hint="Boleh dikosongkan. Dipakai saat membagikan PDF ke klien.">
          <Input
            type="tel"
            inputMode="tel"
            value={values.contactWhatsapp}
            onChange={(e) => set("contactWhatsapp", e.target.value)}
            onBlur={() => blur("contactWhatsapp")}
            placeholder="0812 3456 7890"
          />
        </Field>
        <Field label="PIC" required name="picStaffId" error={errors.picStaffId}>
          <Select aria-label="PIC" value={values.picStaffId} onChange={(e) => set("picStaffId", e.target.value)}>
            {choices.map((m) => (
              <option key={m.id} value={m.id}>
                {m.id === currentStaffId && m.name !== "Saya" ? `${m.name} (saya)` : m.name}
              </option>
            ))}
          </Select>
        </Field>
      </form>
    </Modal>
  );
}
