import { useRef } from "react";
import { Modal } from "@/shared/components/ui/Modal";
import { Button } from "@/shared/components/ui/Button";
import { Input, Select, Field } from "@/shared/components/ui/Input";
import { CurrencyInput } from "@/shared/components/ui/CurrencyInput";
import { useZodForm } from "@/shared/forms/useZodForm";
import { FormErrorSummary } from "@/shared/forms/FormErrorSummary";
import { newIdempotencyKey } from "@/shared/lib/direct-upload";
import { toast } from "@/shared/stores/useToastStore";
import { supplierBillTermSchema, type SupplierBillTermFormValues } from "@/modules/projects/schemas/supplier-bill.schema";
import { useSupplierBillStore } from "@/modules/projects/stores/useSupplierBillStore";
import type { SupplierBillTerm } from "@/modules/projects/types";

const LABELS: Record<string, string> = {
  projectSupplierId: "Vendor atau venue", label: "Nama tagihan", amount: "Nominal", dueDate: "Jatuh tempo",
};

interface Props {
  projectId: string;
  /** Vendors and venues the client pays directly, to pick from when adding. */
  supplierOptions: { value: string; label: string }[];
  /** Opened from one vendor (its drawer or its row): that one is chosen already. */
  initialSupplierId?: string;
  /** Ubah tagihan: the bill as read earlier (its version goes back as If-Match). */
  term?: SupplierBillTerm;
  onClose: () => void;
}

/**
 * "Tambah tagihan" / "Ubah tagihan" of a vendor or venue the client pays directly (Paket WO Tahap 4): what the couple
 * owes that vendor and by when. Loaded on demand. Adding is safe to retry (Idempotency-Key); a change sends the
 * version it was read at (If-Match).
 */
export default function BillTermFormModal({ projectId, supplierOptions, initialSupplierId, term, onClose }: Props) {
  const createTerm = useSupplierBillStore((s) => s.createTerm);
  const updateTerm = useSupplierBillStore((s) => s.updateTerm);
  const formRef = useRef<HTMLFormElement>(null);
  const keyRef = useRef(newIdempotencyKey());
  const editing = Boolean(term);
  const form = useZodForm({
    schema: supplierBillTermSchema,
    initial: {
      projectSupplierId: term?.projectSupplierId ?? initialSupplierId ?? supplierOptions[0]?.value ?? "",
      label: term?.label ?? "",
      amount: term?.amount ?? 0,
      dueDate: term?.dueDate ?? "",
    },
  });
  const { values, errors, blur } = form;
  const supplierName = supplierOptions.find((o) => o.value === values.projectSupplierId)?.label ?? "";

  function set<K extends keyof SupplierBillTermFormValues & string>(key: K, value: (typeof values)[K]) {
    // A different request must not reuse the earlier key.
    keyRef.current = newIdempotencyKey();
    form.set(key, value);
  }

  const submit = form.handleSubmit(
    async (parsed) => {
      if (term) {
        await updateTerm(projectId, term.id, parsed, term.updatedAt);
        toast.success("Tagihan vendor disimpan");
      } else {
        await createTerm(projectId, parsed, keyRef.current);
        toast.success("Tagihan vendor ditambahkan");
      }
      onClose();
    },
    {
      root: () => formRef.current,
      fallback: "Tagihan belum tersimpan. Periksa koneksi lalu coba lagi.",
      known: (key) => key in values,
    },
  );

  return (
    <Modal
      open
      onClose={onClose}
      title={editing ? "Ubah tagihan" : "Tambah tagihan"}
      description={
        supplierName
          ? `Tagihan dari ${supplierName} yang dibayar klien langsung. Tidak masuk kas WO dan tidak menambah nilai kontrak.`
          : "Tagihan vendor yang dibayar klien langsung. Tidak masuk kas WO dan tidak menambah nilai kontrak."
      }
      size="sm"
      busy={form.submitting}
      footer={
        <>
          <Button variant="secondary" onClick={onClose} disabled={form.submitting}>Batal</Button>
          <Button type="submit" form="bill-term-form" disabled={form.submitting}>
            {form.submitting ? "Menyimpan..." : editing ? "Simpan perubahan" : "Simpan tagihan"}
          </Button>
        </>
      }
    >
      <form id="bill-term-form" ref={formRef} noValidate onSubmit={(e) => void submit(e)} className="flex flex-col gap-4">
        <FormErrorSummary errors={form.errorList} formError={form.formError} labels={LABELS} root={() => formRef.current} />
        {!editing && supplierOptions.length > 1 && (
          <Field label="Vendor atau venue" required name="projectSupplierId" error={errors.projectSupplierId}>
            <Select aria-label="Vendor atau venue" value={values.projectSupplierId} onChange={(e) => set("projectSupplierId", e.target.value)}>
              {supplierOptions.map((o) => (
                <option key={o.value} value={o.value}>{o.label}</option>
              ))}
            </Select>
          </Field>
        )}
        <Field label="Nama tagihan" required name="label" error={errors.label} hint="Mis. DP, Termin 2, Pelunasan.">
          <Input value={values.label} onChange={(e) => set("label", e.target.value)} onBlur={() => blur("label")} maxLength={100} />
        </Field>
        <Field label="Nominal (Rp)" required name="amount" error={errors.amount}>
          <CurrencyInput value={Number(values.amount) || 0} onChange={(n) => set("amount", n)} aria-invalid={errors.amount ? true : undefined} />
        </Field>
        <Field label="Jatuh tempo" required name="dueDate" error={errors.dueDate}>
          <Input type="date" value={values.dueDate} onChange={(e) => set("dueDate", e.target.value)} onBlur={() => blur("dueDate")} />
        </Field>
      </form>
    </Modal>
  );
}
