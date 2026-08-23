import { useEffect, useState } from "react";
import { ShieldCheck } from "lucide-react";
import { Modal } from "@/shared/components/ui/Modal";
import { Button } from "@/shared/components/ui/Button";
import { Select, Field } from "@/shared/components/ui/Input";
import {
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
  onSubmit: (values: ActivateSubscriptionFormValues) => void;
}

export function ActivateSubscriptionModal({ open, onClose, tenant, plans, isSubmitting = false, onSubmit }: ActivateSubscriptionModalProps) {
  // appIds.length === 0 is what closes T16: without it, a plan private to
  // some external App (visible here because this store also feeds
  // platform_admin's own Paket page, A12) could be assigned to a WO tenant
  // — who would then be silently filtered out of their own plan by
  // ListPublic on the very next page load.
  const activePlans = plans.filter((p) => p.isActive && p.appIds.length === 0);
  const [values, setValues] = useState<ActivateSubscriptionFormValues>({ planId: tenant?.planId ?? activePlans[0]?.id ?? "" });
  const [errors, setErrors] = useState<Partial<Record<keyof ActivateSubscriptionFormValues, string>>>({});

  // Same async-plans backfill as TenantFormModal — plans may still be empty
  // at first mount if this modal opens before the fetch resolves.
  useEffect(() => {
    if (!values.planId && activePlans.length > 0) {
      setValues({ planId: tenant?.planId ?? activePlans[0].id });
    }
  }, [activePlans.length, tenant?.planId]);

  function handleSubmit() {
    if (activePlans.length === 0) return;
    const result = activateSubscriptionSchema.safeParse(values);
    if (!result.success) {
      const fieldErrors: Partial<Record<keyof ActivateSubscriptionFormValues, string>> = {};
      for (const issue of result.error.issues) {
        fieldErrors[issue.path[0] as keyof ActivateSubscriptionFormValues] = issue.message;
      }
      setErrors(fieldErrors);
      return;
    }
    onSubmit(result.data);
    setErrors({});
  }

  function handleClose() {
    if (isSubmitting) return;
    setValues({ planId: tenant?.planId ?? activePlans[0]?.id ?? "" });
    setErrors({});
    onClose();
  }

  return (
    <Modal
      open={open}
      onClose={handleClose}
      title="Aktifkan Langganan"
      description={tenant ? `Aktifkan langganan ${tenant.businessName} langsung tanpa menunggu pembayaran tenant.` : undefined}
      size="sm"
      footer={
        <>
          <Button variant="secondary" onClick={handleClose} disabled={isSubmitting}>
            Batal
          </Button>
          <Button onClick={handleSubmit} disabled={isSubmitting || activePlans.length === 0}>
            {isSubmitting ? "Memproses..." : "Aktifkan Langganan"}
          </Button>
        </>
      }
    >
      <div className="flex flex-col gap-4">
        {activePlans.length === 0 ? (
          <p className="rounded-md border border-warning/30 bg-warning-soft px-3.5 py-2.5 text-[13px] font-medium text-warning">
            Belum ada paket publik yang aktif — tambahkan dulu di menu Paket.
          </p>
        ) : (
          <Field label="Paket Langganan" required hint={errors.planId}>
            <Select value={values.planId} onChange={(e) => setValues({ planId: e.target.value })}>
              {activePlans.map((plan) => (
                <option key={plan.id} value={plan.id}>
                  {plan.name} — {formatCurrency(plan.price)} / {plan.durationMonths} bulan
                </option>
              ))}
            </Select>
          </Field>
        )}
        <p className="flex items-start gap-1.5 text-[12px] text-text-secondary">
          <ShieldCheck className="mt-0.5 h-3.5 w-3.5 shrink-0" />
          Aktivasi ini dicatat sebagai transaksi &quot;Diaktifkan Admin&quot; — terpisah dari pembayaran Tripay dan tidak dihitung
          sebagai pendapatan.
        </p>
      </div>
    </Modal>
  );
}
