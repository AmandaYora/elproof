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
  onSubmit: (values: ActivateSubscriptionFormValues) => void;
}

export function ActivateSubscriptionModal({ open, onClose, tenant, plans, onSubmit }: ActivateSubscriptionModalProps) {
  const activePlans = plans.filter((p) => p.isActive);
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
          <Button variant="secondary" onClick={handleClose}>
            Batal
          </Button>
          <Button onClick={handleSubmit}>Aktifkan Langganan</Button>
        </>
      }
    >
      <div className="flex flex-col gap-4">
        <Field label="Paket Langganan" required hint={errors.planId}>
          <Select value={values.planId} onChange={(e) => setValues({ planId: e.target.value })}>
            {activePlans.map((plan) => (
              <option key={plan.id} value={plan.id}>
                {plan.name} — {formatCurrency(plan.price)} / {plan.durationMonths} bulan
              </option>
            ))}
          </Select>
        </Field>
        <p className="flex items-start gap-1.5 text-[12px] text-text-secondary">
          <ShieldCheck className="mt-0.5 h-3.5 w-3.5 shrink-0" />
          Aktivasi ini dicatat sebagai transaksi &quot;Diaktifkan Admin&quot; — terpisah dari pembayaran Tripay dan tidak dihitung
          sebagai pendapatan.
        </p>
      </div>
    </Modal>
  );
}
