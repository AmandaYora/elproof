import { useEffect, useMemo, useRef } from "react";
import { Modal } from "@/shared/components/ui/Modal";
import { Button } from "@/shared/components/ui/Button";
import { CurrencyInput } from "@/shared/components/ui/CurrencyInput";
import { Field, Select, Textarea } from "@/shared/components/ui/Input";
import { useZodForm } from "@/shared/forms/useZodForm";
import { FormErrorSummary } from "@/shared/forms/FormErrorSummary";
import { isStaleConflict } from "@/shared/lib/concurrency";
import { SUPPLIER_PAYER_LABELS } from "@/shared/lib/labels";
import { toast } from "@/shared/stores/useToastStore";
import { SUPPLIER_PAYER_OPTIONS } from "@/modules/rincian";
import { useVendorStore } from "@/modules/vendors/stores/useVendorStore";
import { quotationSupplierSchema } from "@/modules/offers/schemas/quotation.schema";
import { useQuotationStore } from "@/modules/offers/stores/useQuotationStore";
import type { QuotationSupplier } from "@/modules/offers/types";

const FIELD_LABELS: Record<string, string> = {
  vendorId: "Vendor",
  paidBy: "Dibayar oleh",
  showName: "Tampilkan nama ke klien",
  agreedCost: "Biaya rekanan",
  notes: "Catatan",
};

interface Props {
  quotationId: string;
  /** The quotation's vendors and venues (a vendor is added once). */
  suppliers: QuotationSupplier[];
  /** The one to change, or null for "Tambah vendor". */
  supplier: QuotationSupplier | null;
  /** finance.margin.view: "Biaya rekanan" is shown and sent. */
  canMargin: boolean;
  onClose: () => void;
}

/** "Tambah vendor" / "Ubah vendor" of a quotation (DESAIN §5.2): who pays it, whether the client sees its name, the biaya rekanan. */
export default function QuotationSupplierModal({ quotationId, suppliers, supplier, canMargin, onClose }: Props) {
  const addSupplier = useQuotationStore((s) => s.addSupplier);
  const updateSupplier = useQuotationStore((s) => s.updateSupplier);
  const vendors = useVendorStore((s) => s.vendors);
  const fetchVendors = useVendorStore((s) => s.fetchVendors);
  const formRef = useRef<HTMLFormElement>(null);
  const isVenue = supplier?.kind === "venue";

  useEffect(() => {
    if (!supplier) void fetchVendors().catch(() => undefined);
  }, [fetchVendors, supplier]);

  const choices = useMemo(() => {
    const taken = new Set(suppliers.filter((s) => s.kind === "vendor").map((s) => s.supplierId));
    return vendors.filter((v) => v.isActive && !taken.has(v.id)).sort((a, b) => a.name.localeCompare(b.name, "id"));
  }, [vendors, suppliers]);

  const form = useZodForm({
    schema: quotationSupplierSchema,
    initial: {
      vendorId: supplier?.supplierId ?? "",
      paidBy: supplier?.paidBy ?? "WO",
      showName: supplier?.showName ?? true,
      agreedCost: supplier?.agreedCost ?? "",
      notes: supplier?.notes ?? "",
    },
  });
  const { values, errors, set, blur } = form;

  const submit = form.handleSubmit(
    async (parsed) => {
      const input = {
        vendorId: parsed.vendorId,
        paidBy: parsed.paidBy,
        showName: isVenue ? true : parsed.showName,
        notes: parsed.notes,
        ...(canMargin ? { agreedCost: parsed.agreedCost } : {}),
      };
      try {
        if (supplier) await updateSupplier(quotationId, supplier.id, input, supplier.updatedAt);
        else await addSupplier(quotationId, input);
      } catch (err) {
        if (isStaleConflict(err)) return;
        throw err;
      }
      toast.success(supplier ? "Vendor disimpan" : "Vendor ditambahkan");
      onClose();
    },
    { root: () => formRef.current, fallback: "Vendor belum tersimpan. Periksa koneksi Anda, lalu coba lagi.", known: (key) => key in values },
  );

  const chosen = vendors.find((v) => v.id === values.vendorId);
  const title = supplier ? `Ubah ${isVenue ? "venue" : "vendor"}: ${supplier.name}` : "Tambah vendor";

  return (
    <Modal
      open
      onClose={onClose}
      title={title}
      description={supplier ? undefined : "Pilih vendor dari direktori. Venue masuk lewat acara."}
      busy={form.submitting}
      footer={
        <>
          <Button variant="secondary" onClick={onClose} disabled={form.submitting}>
            Batal
          </Button>
          <Button type="submit" form="quotation-supplier-form" disabled={form.submitting}>
            {form.submitting ? "Menyimpan..." : supplier ? "Simpan perubahan" : "Tambah vendor"}
          </Button>
        </>
      }
    >
      <form id="quotation-supplier-form" ref={formRef} noValidate onSubmit={(e) => void submit(e)} className="flex flex-col gap-5">
        <FormErrorSummary errors={form.errorList} formError={form.formError} labels={FIELD_LABELS} root={() => formRef.current} />
        {!supplier && (
          <Field
            label="Vendor"
            required
            name="vendorId"
            error={errors.vendorId}
            hint={
              choices.length === 0 && vendors.length > 0
                ? "Semua vendor aktif di direktori sudah ada di penawaran ini."
                : chosen
                  ? [chosen.categoryName, chosen.city].filter(Boolean).join(", ") || undefined
                  : undefined
            }
          >
            <Select aria-label="Vendor" placeholder="Pilih vendor" value={values.vendorId} onChange={(e) => set("vendorId", e.target.value)}>
              {choices.map((v) => (
                <option key={v.id} value={v.id}>
                  {v.name}
                </option>
              ))}
            </Select>
          </Field>
        )}

        <fieldset className="flex flex-col gap-2" data-field="paidBy">
          <legend className="mb-1.5 text-[13px] font-medium text-foreground">Dibayar oleh</legend>
          <div className="flex flex-wrap gap-x-6 gap-y-2">
            {SUPPLIER_PAYER_OPTIONS.map((payer) => (
              <label key={payer} className="flex items-center gap-2 text-[13px] text-foreground">
                <input type="radio" name="paidBy" value={payer} checked={values.paidBy === payer} onChange={() => set("paidBy", payer)} className="size-4 accent-primary" />
                {SUPPLIER_PAYER_LABELS[payer]}
              </label>
            ))}
          </div>
          {errors.paidBy && (
            <span role="alert" className="text-xs font-medium text-danger-text">
              {errors.paidBy}
            </span>
          )}
        </fieldset>

        {!isVenue && (
          <label className="flex items-start gap-2 text-[13px] text-foreground" data-field="showName">
            <input type="checkbox" checked={values.showName} onChange={(e) => set("showName", e.target.checked)} className="mt-0.5 size-4 accent-primary" />
            <span>
              Tampilkan nama ke klien
              <span className="block text-xs text-muted-foreground">Bila tidak, PDF hanya menyebut kategorinya.</span>
            </span>
          </label>
        )}

        {canMargin && values.paidBy === "WO" && (
          <Field label="Biaya rekanan (Rp)" name="agreedCost" error={errors.agreedCost} hint="Yang WO bayar ke rekanan ini. Tidak pernah tampil di PDF.">
            <CurrencyInput
              optional
              value={values.agreedCost === "" ? 0 : Number(values.agreedCost)}
              onChange={(n) => set("agreedCost", n === 0 ? "" : n)}
              onBlur={() => blur("agreedCost")}
              aria-invalid={Boolean(errors.agreedCost)}
            />
          </Field>
        )}

        <Field label="Catatan" name="notes" error={errors.notes}>
          <Textarea rows={3} value={values.notes} onChange={(e) => set("notes", e.target.value)} onBlur={() => blur("notes")} />
        </Field>
      </form>
    </Modal>
  );
}
