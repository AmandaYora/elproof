import { useEffect, useRef } from "react";
import { Modal } from "@/shared/components/ui/Modal";
import { Button } from "@/shared/components/ui/Button";
import { Input, Textarea, Field } from "@/shared/components/ui/Input";
import {
  vendorCategorySchema,
  type VendorCategoryFormValues,
} from "@/modules/vendor-categories/schemas/vendor-category.schema";
import type { VendorCategory } from "@/modules/vendor-categories/types";
import { useZodForm } from "@/shared/forms/useZodForm";
import { FormErrorSummary } from "@/shared/forms/FormErrorSummary";

const FIELD_LABELS: Record<string, string> = { name: "Nama kategori", description: "Deskripsi" };

interface VendorCategoryFormModalProps {
  open: boolean;
  onClose: () => void;
  /** Saves, then closes the modal. Throw to keep it open: a 422 is shown on its fields, anything else inside the modal. */
  onSubmit: (values: VendorCategoryFormValues) => Promise<void>;
  initialCategory?: VendorCategory;
}

function toFormValues(category?: VendorCategory): VendorCategoryFormValues {
  if (!category) {
    return {
      name: "",
      description: "",
    };
  }
  return {
    name: category.name,
    description: category.description,
  };
}

export function VendorCategoryFormModal({ open, onClose, onSubmit, initialCategory }: VendorCategoryFormModalProps) {
  const form = useZodForm({ schema: vendorCategorySchema, initial: toFormValues(initialCategory) });
  const { values, errors, set, blur } = form;
  const formRef = useRef<HTMLFormElement>(null);

  // Mounted once and kept alive while closed: every opening starts from the category as it is now.
  useEffect(() => {
    form.reset(toFormValues(initialCategory));
  }, [initialCategory, open]);

  const submit = form.handleSubmit(onSubmit, { root: () => formRef.current, fallback: "Kategori belum tersimpan. Periksa koneksi lalu coba lagi." });

  return (
    <Modal
      open={open}
      onClose={onClose}
      title={initialCategory ? "Ubah Kategori Vendor" : "Tambah Kategori Vendor"}
      description="Kategori digunakan untuk mengelompokkan vendor berdasarkan jenis layanan."
      busy={form.submitting}
      footer={
        <>
          <Button variant="secondary" onClick={onClose} disabled={form.submitting}>
            Batal
          </Button>
          <Button type="submit" form="vendor-category-form" disabled={form.submitting}>
            {form.submitting ? "Menyimpan..." : initialCategory ? "Simpan perubahan" : "Simpan kategori"}
          </Button>
        </>
      }
    >
      <form id="vendor-category-form" ref={formRef} noValidate onSubmit={(e) => void submit(e)} className="grid grid-cols-1 gap-4">
        <FormErrorSummary errors={form.errorList} formError={form.formError} labels={FIELD_LABELS} root={() => formRef.current} />
        <Field label="Nama Kategori" required name="name" error={errors.name}>
          <Input value={values.name} onChange={(e) => set("name", e.target.value)} onBlur={() => blur("name")} placeholder="cth. Dekorasi" />
        </Field>
        <Field label="Deskripsi" required name="description" error={errors.description}>
          <Textarea
            rows={3}
            value={values.description}
            onChange={(e) => set("description", e.target.value)}
            onBlur={() => blur("description")}
            placeholder="Jelaskan cakupan kategori vendor ini"
          />
        </Field>
      </form>
    </Modal>
  );
}
