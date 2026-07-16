import { useState } from "react";
import { Modal } from "@/shared/components/ui/Modal";
import { Button } from "@/shared/components/ui/Button";
import { Input, Textarea, Field } from "@/shared/components/ui/Input";
import {
  vendorCategorySchema,
  type VendorCategoryFormValues,
} from "@/modules/vendor-categories/schemas/vendor-category.schema";
import type { VendorCategory } from "@/modules/vendor-categories/types";

interface VendorCategoryFormModalProps {
  open: boolean;
  onClose: () => void;
  onSubmit: (values: VendorCategoryFormValues) => void;
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
  const [values, setValues] = useState<VendorCategoryFormValues>(() => toFormValues(initialCategory));
  const [errors, setErrors] = useState<Partial<Record<keyof VendorCategoryFormValues, string>>>({});

  function set<K extends keyof VendorCategoryFormValues>(key: K, value: VendorCategoryFormValues[K]) {
    setValues((prev) => ({ ...prev, [key]: value }));
  }

  function handleSubmit() {
    const result = vendorCategorySchema.safeParse(values);
    if (!result.success) {
      const fieldErrors: Partial<Record<keyof VendorCategoryFormValues, string>> = {};
      for (const issue of result.error.issues) {
        const key = issue.path[0] as keyof VendorCategoryFormValues;
        fieldErrors[key] = issue.message;
      }
      setErrors(fieldErrors);
      return;
    }
    onSubmit(result.data);
    setValues(toFormValues());
    setErrors({});
  }

  return (
    <Modal
      open={open}
      onClose={onClose}
      title={initialCategory ? "Ubah Kategori Vendor" : "Tambah Kategori Vendor"}
      description="Kategori digunakan untuk mengelompokkan vendor berdasarkan jenis layanan."
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>
            Batal
          </Button>
          <Button onClick={handleSubmit}>{initialCategory ? "Simpan Perubahan" : "Simpan Kategori"}</Button>
        </>
      }
    >
      <div className="grid grid-cols-1 gap-4">
        <Field label="Nama Kategori" required hint={errors.name}>
          <Input value={values.name} onChange={(e) => set("name", e.target.value)} placeholder="cth. Venue" />
        </Field>
        <Field label="Deskripsi" required hint={errors.description}>
          <Textarea
            rows={3}
            value={values.description}
            onChange={(e) => set("description", e.target.value)}
            placeholder="Jelaskan cakupan kategori vendor ini"
          />
        </Field>
      </div>
    </Modal>
  );
}
