import { useState } from "react";
import { ConfirmDialog } from "@/shared/components/ui/ConfirmDialog";
import type { SubscriptionPlan } from "@/shared/data/subscriptionPlans";
import { useSubscriptionPlanStore } from "@/shared/stores/useSubscriptionPlanStore";
import { toast } from "@/shared/stores/useToastStore";
import { getApiErrorMessage } from "@/shared/lib/api-error";

/**
 * Activating and deactivating a plan, shared by the Paket page and an App's detail dialog (EP-11). Like Voucher:
 * activating happens at once, deactivating asks first, because the plan leaves the catalog (and the homepage) straight away.
 * `onChanged` reloads what the caller shows; `onError` shows an activation failure in the caller's own error slot.
 */
export function usePlanActiveActions(onChanged: () => Promise<void> | void, onError: (message: string | null) => void) {
  const togglePlanActive = useSubscriptionPlanStore((s) => s.togglePlanActive);
  const [deactivating, setDeactivating] = useState<SubscriptionPlan | undefined>(undefined);
  const [open, setOpen] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function activate(plan: SubscriptionPlan) {
    onError(null);
    try {
      await togglePlanActive(plan.id);
      toast.success(`Paket ${plan.name} diaktifkan`);
      await onChanged();
    } catch (err) {
      onError(getApiErrorMessage(err, `Paket ${plan.name} belum bisa diaktifkan. Coba lagi.`));
    }
  }

  function askDeactivate(plan: SubscriptionPlan) {
    onError(null);
    setError(null);
    setDeactivating(plan);
    setOpen(true);
  }

  async function confirmDeactivate() {
    if (!deactivating) return;
    setError(null);
    setBusy(true);
    try {
      await togglePlanActive(deactivating.id);
      toast.success(`Paket ${deactivating.name} dinonaktifkan`);
      setOpen(false);
      await onChanged();
    } catch (err) {
      setError(getApiErrorMessage(err, "Paket belum bisa dinonaktifkan. Coba lagi."));
    } finally {
      setBusy(false);
    }
  }

  // The plan is kept after closing so the title does not empty while the dialog animates out.
  const dialog = (
    <ConfirmDialog
      open={open}
      title={`Nonaktifkan paket ${deactivating?.name ?? ""}?`}
      confirmLabel="Nonaktifkan paket"
      isSubmitting={busy}
      error={error}
      onConfirm={() => void confirmDeactivate()}
      onClose={() => setOpen(false)}
    >
      Paket ini langsung hilang dari pilihan pembelian baru (untuk paket ElProof juga dari halaman Langganan tenant WO dan beranda). Tenant
      yang sudah berlangganan tidak berubah, dan paket bisa diaktifkan kembali kapan saja.
    </ConfirmDialog>
  );

  return { activate, askDeactivate, dialog };
}
