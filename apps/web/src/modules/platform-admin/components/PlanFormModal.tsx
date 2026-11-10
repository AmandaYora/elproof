import { useEffect, useRef } from "react";
import { Plus, X } from "lucide-react";
import { Modal } from "@/shared/components/ui/Modal";
import { Button } from "@/shared/components/ui/Button";
import { Input, Field, Select } from "@/shared/components/ui/Input";
import { planSchema, type PlanFormValues } from "@/modules/platform-admin/schemas/plan.schema";
import { appLabel, ELPROOF_APP_ID, type SubscriptionPlan } from "@/shared/data/subscriptionPlans";
import type { PaymentApp } from "@/modules/platform-admin/stores/useAppsStore";
import { useZodForm } from "@/shared/forms/useZodForm";
import { FormErrorSummary } from "@/shared/forms/FormErrorSummary";

const FIELD_LABELS: Record<string, string> = { appId: "Aplikasi", name: "Nama paket", durationMonths: "Durasi", price: "Harga", features: "Fitur paket" };

function toFormValues(plan: SubscriptionPlan | undefined, defaultAppId: string): PlanFormValues {
  if (!plan) return { appId: defaultAppId, name: "", durationMonths: 12, price: 0, features: [""] };
  return {
    appId: plan.appId,
    name: plan.name,
    durationMonths: plan.durationMonths,
    price: plan.price,
    features: plan.features.length > 0 ? plan.features : [""],
  };
}

interface PlanFormModalProps {
  open: boolean;
  onClose: () => void;
  /** Saves, then closes the modal. Throw to keep it open: the message is shown inside it, a 422 on its fields. */
  onSubmit: (values: PlanFormValues) => Promise<void>;
  initialPlan?: SubscriptionPlan;
  /** The Apps a plan can belong to (Manajemen Aplikasi). */
  apps: PaymentApp[];
  /** App chosen for a new plan; "" makes the person pick one. */
  defaultAppId?: string;
  /** Fixes the App (adding a plan from that App's own detail): shown, not editable. */
  lockApp?: boolean;
}

export function PlanFormModal({ open, onClose, onSubmit, initialPlan, apps, defaultAppId = "", lockApp = false }: PlanFormModalProps) {
  const form = useZodForm({ schema: planSchema, initial: toFormValues(initialPlan, defaultAppId) });
  const { values, errors, set, blur } = form;
  const formRef = useRef<HTMLFormElement>(null);

  // The modal stays mounted while closed: start clean each time it opens.
  useEffect(() => {
    form.reset(toFormValues(initialPlan, defaultAppId));
  }, [initialPlan, open, defaultAppId]);

  // ElProof's own App first, then the external ones as Manajemen Aplikasi lists them.
  const appOptions = [
    { appId: ELPROOF_APP_ID, label: appLabel(undefined, ELPROOF_APP_ID) },
    ...apps.filter((a) => a.appId !== ELPROOF_APP_ID).map((a) => ({ appId: a.appId, label: a.isActive ? a.name : `${a.name} (nonaktif)` })),
  ];
  const lockedLabel = appLabel(apps.find((a) => a.appId === values.appId), values.appId);

  function setFeature(index: number, value: string) {
    set("features", values.features.map((f, i) => (i === index ? value : f)));
  }

  function addFeature() {
    set("features", [...values.features, ""]);
  }

  function removeFeature(index: number) {
    set("features", values.features.filter((_, i) => i !== index));
  }

  const submit = form.handleSubmit(onSubmit, {
    root: () => formRef.current,
    fallback: "Paket belum tersimpan. Periksa koneksi lalu coba lagi.",
    known: (key) => key in FIELD_LABELS,
  });

  return (
    <Modal
      open={open}
      onClose={onClose}
      busy={form.submitting}
      title={initialPlan ? "Ubah Paket Langganan" : "Tambah Paket Langganan"}
      description="Setiap paket milik satu aplikasi. Hanya aplikasi itu yang melihat dan menjual paket ini; paket ElProof untuk tenant WO."
      footer={
        <>
          <Button variant="secondary" onClick={onClose} disabled={form.submitting}>
            Batal
          </Button>
          <Button type="submit" form="plan-form" disabled={form.submitting}>
            {form.submitting ? "Menyimpan..." : initialPlan ? "Simpan perubahan" : "Simpan paket"}
          </Button>
        </>
      }
    >
      <form id="plan-form" ref={formRef} noValidate onSubmit={(e) => void submit(e)} className="grid grid-cols-1 gap-4">
        <FormErrorSummary errors={form.errorList} formError={form.formError} labels={FIELD_LABELS} root={() => formRef.current} />
        <Field label="Aplikasi" required name="appId" error={errors.appId}>
          {lockApp ? (
            <p className="rounded-md border border-border bg-surface-muted px-3 py-2 text-[13.5px] font-medium text-text-primary">{lockedLabel}</p>
          ) : (
            <Select value={values.appId} onChange={(e) => set("appId", e.target.value)} aria-label="Aplikasi" placeholder="Pilih aplikasi...">
              {appOptions.map((o) => (
                <option key={o.appId} value={o.appId}>
                  {o.label}
                </option>
              ))}
            </Select>
          )}
        </Field>
        <Field label="Nama Paket" required name="name" error={errors.name}>
          <Input value={values.name} onChange={(e) => set("name", e.target.value)} onBlur={() => blur("name")} placeholder="cth. Paket 1 Tahun" />
        </Field>
        <Field label="Durasi (bulan)" required name="durationMonths" error={errors.durationMonths}>
          <Input
            type="number"
            min={1}
            value={values.durationMonths}
            onChange={(e) => set("durationMonths", Number(e.target.value))}
            onBlur={() => blur("durationMonths")}
            placeholder="12"
          />
        </Field>
        <Field label="Harga (Rp)" required name="price" error={errors.price}>
          <Input
            type="number"
            min={0}
            value={values.price}
            onChange={(e) => set("price", Number(e.target.value))}
            onBlur={() => blur("price")}
            placeholder="2000000"
          />
        </Field>
        <Field label="Fitur Paket" required name="features" error={errors.features}>
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
                  className="shrink-0 rounded-md p-2 text-text-secondary hover:bg-surface-muted hover:text-danger-text disabled:pointer-events-none disabled:opacity-40"
                >
                  <X className="h-4 w-4" />
                </button>
              </div>
            ))}
            <Button type="button" variant="secondary" size="sm" icon={<Plus className="h-3.5 w-3.5" />} onClick={addFeature}>
              Tambah fitur
            </Button>
          </div>
        </Field>
      </form>
    </Modal>
  );
}
