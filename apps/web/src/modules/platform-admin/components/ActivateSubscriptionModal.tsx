import { useEffect, useState } from "react";
import { ShieldCheck } from "lucide-react";
import { Modal } from "@/shared/components/ui/Modal";
import { Button } from "@/shared/components/ui/Button";
import { Select, Field, Input, Textarea } from "@/shared/components/ui/Input";
import {
  ACTIVATION_REASON_OPTIONS,
  activateSubscriptionSchema,
  type ActivateSubscriptionFormValues,
} from "@/modules/platform-admin/schemas/activate-subscription.schema";
import type { Tenant } from "@/modules/platform-admin/data/types";
import type { SubscriptionPlan } from "@/shared/data/subscriptionPlans";
import { formatCurrency } from "@/shared/lib/formatters";

interface ActivateSubscriptionModalProps {
  open: boolean;
  onClose: () => void;
  tenant?: Tenant;
  plans: SubscriptionPlan[];
  isSubmitting?: boolean;
  /** Server-side failure, shown inside the dialog so nothing is lost. */
  error?: string | null;
  onSubmit: (values: ActivateSubscriptionFormValues) => void;
}

type Errors = Partial<Record<keyof ActivateSubscriptionFormValues, string>>;

function initialValues(tenant: Tenant | undefined, plans: SubscriptionPlan[]): ActivateSubscriptionFormValues {
  return { planId: tenant?.planId ?? plans[0]?.id ?? "", reason: "", proofReference: "", note: "" };
}

export function ActivateSubscriptionModal({
  open,
  onClose,
  tenant,
  plans,
  isSubmitting = false,
  error,
  onSubmit,
}: ActivateSubscriptionModalProps) {
  // appIds.length === 0 is what closes T16: without it, a plan private to
  // some external App (visible here because this store also feeds
  // platform_admin's own Paket page, A12) could be assigned to a WO tenant
  // — who would then be silently filtered out of their own plan by
  // ListPublic on the very next page load.
  const activePlans = plans.filter((p) => p.isActive && p.appIds.length === 0);
  const [values, setValues] = useState<ActivateSubscriptionFormValues>(() => initialValues(tenant, activePlans));
  const [errors, setErrors] = useState<Errors>({});

  // Reset whenever the dialog opens for a (possibly different) tenant, and
  // backfill the plan once plans have loaded.
  useEffect(() => {
    if (open) {
      setValues(initialValues(tenant, activePlans));
      setErrors({});
    }
  }, [open, tenant?.id]);
  useEffect(() => {
    if (!values.planId && activePlans.length > 0) {
      setValues((v) => ({ ...v, planId: tenant?.planId ?? activePlans[0].id }));
    }
  }, [activePlans.length, tenant?.planId]);

  function set<K extends keyof ActivateSubscriptionFormValues>(key: K, value: ActivateSubscriptionFormValues[K]) {
    setValues((prev) => ({ ...prev, [key]: value }));
    setErrors((prev) => ({ ...prev, [key]: undefined }));
  }

  function handleSubmit() {
    if (activePlans.length === 0 || isSubmitting) return;
    const result = activateSubscriptionSchema.safeParse(values);
    if (!result.success) {
      const fieldErrors: Errors = {};
      for (const issue of result.error.issues) {
        fieldErrors[issue.path[0] as keyof ActivateSubscriptionFormValues] ??= issue.message;
      }
      setErrors(fieldErrors);
      return;
    }
    onSubmit(result.data);
  }

  function handleClose() {
    if (isSubmitting) return;
    onClose();
  }

  return (
    <Modal
      open={open}
      onClose={handleClose}
      title="Aktifkan Langganan Manual"
      description={
        tenant
          ? `Aktifkan langganan ${tenant.businessName} tanpa pembayaran lewat gateway. Alasan dan bukti dicatat atas nama Anda.`
          : undefined
      }
      size="sm"
      footer={
        <>
          <Button variant="secondary" onClick={handleClose} disabled={isSubmitting}>
            Batal
          </Button>
          <Button onClick={handleSubmit} disabled={isSubmitting || activePlans.length === 0}>
            {isSubmitting ? "Mengaktifkan..." : "Aktifkan Langganan"}
          </Button>
        </>
      }
    >
      <div className="flex flex-col gap-4">
        {error && (
          <p role="alert" className="rounded-md border border-danger/30 bg-danger-soft px-3.5 py-2.5 text-[13px] font-medium text-danger">
            {error}
          </p>
        )}
        {activePlans.length === 0 ? (
          <p className="rounded-md border border-warning/30 bg-warning-soft px-3.5 py-2.5 text-[13px] font-medium text-warning">
            Belum ada paket publik yang aktif. Tambahkan dulu di menu Paket.
          </p>
        ) : (
          <Field label="Paket Langganan" required error={errors.planId}>
            <Select value={values.planId} onChange={(e) => set("planId", e.target.value)}>
              {activePlans.map((plan) => (
                <option key={plan.id} value={plan.id}>
                  {plan.name} ({formatCurrency(plan.price)} / {plan.durationMonths} bulan)
                </option>
              ))}
            </Select>
          </Field>
        )}
        <Field label="Alasan Aktivasi" required error={errors.reason}>
          <Select value={values.reason} onChange={(e) => set("reason", e.target.value)}>
            <option value="">Pilih alasan</option>
            {ACTIVATION_REASON_OPTIONS.map((r) => (
              <option key={r.value} value={r.value}>
                {r.label}
              </option>
            ))}
          </Select>
        </Field>
        <Field
          label="Bukti"
          required
          error={errors.proofReference}
          hint="No. transfer, no. invoice, atau tautan dokumen kesepakatan."
        >
          <Input
            value={values.proofReference}
            onChange={(e) => set("proofReference", e.target.value)}
            placeholder="cth. TRF-BCA-20261106-001"
            maxLength={255}
          />
        </Field>
        <Field label="Catatan" required={values.reason === "other"} error={errors.note}>
          <Textarea
            value={values.note}
            onChange={(e) => set("note", e.target.value)}
            rows={3}
            maxLength={500}
            placeholder="Keterangan tambahan untuk tim"
          />
        </Field>
        <p className="flex items-start gap-1.5 text-[12px] text-text-secondary">
          <ShieldCheck className="mt-0.5 h-3.5 w-3.5 shrink-0" />
          Dicatat sebagai transaksi &quot;Diaktifkan Admin&quot;, tidak dihitung sebagai pendapatan, dan masuk audit log.
        </p>
      </div>
    </Modal>
  );
}
