import { useState } from "react";
import { Plus, X } from "lucide-react";
import { Modal } from "@/shared/components/ui/Modal";
import { Button } from "@/shared/components/ui/Button";
import { Input, Field } from "@/shared/components/ui/Input";
import { planSchema, type PlanFormValues } from "@/modules/platform-admin/schemas/plan.schema";
import type { SubscriptionPlan } from "@/shared/data/subscriptionPlans";

function toFormValues(plan?: SubscriptionPlan): PlanFormValues {
  if (!plan) return { name: "", durationMonths: 12, price: 0, features: [""] };
  return {
    name: plan.name,
    durationMonths: plan.durationMonths,
    price: plan.price,
    features: plan.features.length > 0 ? plan.features : [""],
  };
}

interface PlanFormModalProps {
  open: boolean;
  onClose: () => void;
  onSubmit: (values: PlanFormValues) => void;
  initialPlan?: SubscriptionPlan;
}

export function PlanFormModal({ open, onClose, onSubmit, initialPlan }: PlanFormModalProps) {
  const [values, setValues] = useState<PlanFormValues>(() => toFormValues(initialPlan));
  const [errors, setErrors] = useState<Partial<Record<keyof PlanFormValues, string>>>({});

  function set<K extends keyof PlanFormValues>(key: K, value: PlanFormValues[K]) {
    setValues((prev) => ({ ...prev, [key]: value }));
  }

  function setFeature(index: number, value: string) {
    setValues((prev) => ({ ...prev, features: prev.features.map((f, i) => (i === index ? value : f)) }));
  }

  function addFeature() {
    setValues((prev) => ({ ...prev, features: [...prev.features, ""] }));
  }

  function removeFeature(index: number) {
    setValues((prev) => ({ ...prev, features: prev.features.filter((_, i) => i !== index) }));
  }

  function handleSubmit() {
    const cleanedFeatures = values.features.map((f) => f.trim()).filter((f) => f.length > 0);
    const result = planSchema.safeParse({ ...values, features: cleanedFeatures });
    if (!result.success) {
      const fieldErrors: Partial<Record<keyof PlanFormValues, string>> = {};
      for (const issue of result.error.issues) {
        fieldErrors[issue.path[0] as keyof PlanFormValues] = issue.message;
      }
      setErrors(fieldErrors);
      return;
    }
    onSubmit(result.data);
    setErrors({});
  }

  function handleClose() {
    setValues(toFormValues(initialPlan));
    setErrors({});
    onClose();
  }

  return (
    <Modal
      open={open}
      onClose={handleClose}
      title={initialPlan ? "Ubah Paket Langganan" : "Tambah Paket Langganan"}
      description="Paket ini akan tersedia untuk dipilih saat registrasi tenant atau aktivasi langganan."
      footer={
        <>
          <Button variant="secondary" onClick={handleClose}>
            Batal
          </Button>
          <Button onClick={handleSubmit}>{initialPlan ? "Simpan Perubahan" : "Simpan Paket"}</Button>
        </>
      }
    >
      <div className="grid grid-cols-1 gap-4">
        <Field label="Nama Paket" required hint={errors.name}>
          <Input value={values.name} onChange={(e) => set("name", e.target.value)} placeholder="cth. Paket 1 Tahun" />
        </Field>
        <Field label="Durasi (bulan)" required hint={errors.durationMonths}>
          <Input
            type="number"
            min={1}
            value={values.durationMonths}
            onChange={(e) => set("durationMonths", Number(e.target.value))}
            placeholder="12"
          />
        </Field>
        <Field label="Harga (Rp)" required hint={errors.price}>
          <Input
            type="number"
            min={0}
            value={values.price}
            onChange={(e) => set("price", Number(e.target.value))}
            placeholder="2000000"
          />
        </Field>
        <Field label="Fitur Paket" required hint={errors.features}>
          <div className="flex flex-col gap-2">
            {values.features.map((feature, index) => (
              <div key={index} className="flex items-center gap-2">
                <Input
                  value={feature}
                  onChange={(e) => setFeature(index, e.target.value)}
                  placeholder="cth. Backup data otomatis setiap hari"
                />
                <button
                  type="button"
                  onClick={() => removeFeature(index)}
                  disabled={values.features.length <= 1}
                  aria-label="Hapus fitur"
                  className="shrink-0 rounded-md p-2 text-text-secondary hover:bg-surface-muted hover:text-danger disabled:pointer-events-none disabled:opacity-40"
                >
                  <X className="h-4 w-4" />
                </button>
              </div>
            ))}
            <Button type="button" variant="secondary" size="sm" icon={<Plus className="h-3.5 w-3.5" />} onClick={addFeature}>
              Tambah Fitur
            </Button>
          </div>
        </Field>
      </div>
    </Modal>
  );
}
