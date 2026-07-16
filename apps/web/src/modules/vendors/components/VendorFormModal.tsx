import { useEffect, useState } from "react";
import { Modal } from "@/shared/components/ui/Modal";
import { Button } from "@/shared/components/ui/Button";
import { Input, Textarea, Select, Field } from "@/shared/components/ui/Input";
import { vendorSchema, type VendorFormValues } from "@/modules/vendors/schemas/vendor.schema";
import type { Vendor } from "@/modules/vendors/types";
import type { VendorCategory } from "@/modules/vendor-categories/types";

interface VendorFormModalProps {
  open: boolean;
  onClose: () => void;
  onSubmit: (values: VendorFormValues) => void;
  initialVendor?: Vendor;
  categories: VendorCategory[];
}

function toFormValues(vendor?: Vendor, defaultCategoryId = ""): VendorFormValues {
  if (!vendor) {
    return {
      name: "",
      categoryId: defaultCategoryId,
      picName: "",
      phone: "",
      email: "",
      address: "",
      notes: "",
    };
  }
  return {
    name: vendor.name,
    categoryId: vendor.categoryId,
    picName: vendor.picName,
    phone: vendor.phone,
    email: vendor.email,
    address: vendor.address,
    notes: vendor.notes,
  };
}

export function VendorFormModal({ open, onClose, onSubmit, initialVendor, categories }: VendorFormModalProps) {
  const [values, setValues] = useState<VendorFormValues>(() => toFormValues(initialVendor, categories[0]?.id ?? ""));
  const [errors, setErrors] = useState<Partial<Record<keyof VendorFormValues, string>>>({});

  // Categories load asynchronously (Fase 3) — this modal is mounted once and
  // kept alive while closed, so backfill the default once real categories
  // arrive, without clobbering a value the user already picked.
  useEffect(() => {
    if (!initialVendor && !values.categoryId && categories.length > 0) {
      setValues((prev) => ({ ...prev, categoryId: categories[0].id }));
    }
  }, [categories.length, initialVendor]);

  function set<K extends keyof VendorFormValues>(key: K, value: VendorFormValues[K]) {
    setValues((prev) => ({ ...prev, [key]: value }));
  }

  function handleSubmit() {
    const result = vendorSchema.safeParse(values);
    if (!result.success) {
      const fieldErrors: Partial<Record<keyof VendorFormValues, string>> = {};
      for (const issue of result.error.issues) {
        const key = issue.path[0] as keyof VendorFormValues;
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
      title={initialVendor ? "Ubah Vendor" : "Tambah Vendor Baru"}
      description="Informasi vendor yang bekerja sama dengan WO."
      size="lg"
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>
            Batal
          </Button>
          <Button onClick={handleSubmit}>{initialVendor ? "Simpan Perubahan" : "Simpan Vendor"}</Button>
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
        <Field label="Nomor Telepon" required hint={errors.phone}>
          <Input value={values.phone} onChange={(e) => set("phone", e.target.value)} placeholder="0812-xxxx-xxxx" />
        </Field>
        <Field label="Email" required hint={errors.email}>
          <Input type="email" value={values.email} onChange={(e) => set("email", e.target.value)} />
        </Field>
        <div className="sm:col-span-2">
          <Field label="Alamat" required hint={errors.address}>
            <Textarea rows={2} value={values.address} onChange={(e) => set("address", e.target.value)} />
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
      </div>
    </Modal>
  );
}
