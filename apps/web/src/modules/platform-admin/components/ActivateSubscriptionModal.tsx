import { useEffect, useRef } from "react";
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
import { ELPROOF_APP_ID, type SubscriptionPlan } from "@/shared/data/subscriptionPlans";
import { formatCurrency } from "@/shared/lib/formatters";
import { useZodForm } from "@/shared/forms/useZodForm";
import { FormErrorSummary } from "@/shared/forms/FormErrorSummary";

const FIELD_LABELS: Record<string, string> = { planId: "Paket langganan", reason: "Alasan aktivasi", proofReference: "Bukti", note: "Catatan" };

interface ActivateSubscriptionModalProps {
  open: boolean;
  onClose: () => void;
  tenant?: Tenant;
  plans: SubscriptionPlan[];
  /** Activates, then closes the dialog. Throw to keep it open: the message is shown inside it, a 422 on its fields. */
  onSubmit: (values: ActivateSubscriptionFormValues) => Promise<void>;
}

function initialValues(tenant: Tenant | undefined, plans: SubscriptionPlan[]): ActivateSubscriptionFormValues {
  return { planId: tenant?.planId ?? plans[0]?.id ?? "", reason: "", proofReference: "", note: "" };
}

export function ActivateSubscriptionModal({
  open,
  onClose,
  tenant,
  plans,
  onSubmit,
}: ActivateSubscriptionModalProps) {
  // Only ElProof's own plans can be given to a WO tenant (ADR-0040); the
  // server refuses any other App's plan with a 422 on planId as well.
  const activePlans = plans.filter((p) => p.isActive && p.appId === ELPROOF_APP_ID);
  const form = useZodForm({ schema: activateSubscriptionSchema, initial: initialValues(tenant, activePlans) });
  const { values, errors, set } = form;
  const formRef = useRef<HTMLFormElement>(null);

  // Reset whenever the dialog opens for a (possibly different) tenant, and
  // backfill the plan once plans have loaded.
  useEffect(() => {
    if (open) form.reset(initialValues(tenant, activePlans));
  }, [open, tenant?.id]);
  useEffect(() => {
    if (!values.planId && activePlans.length > 0) {
      set("planId", tenant?.planId ?? activePlans[0].id);
    }
  }, [activePlans.length, tenant?.planId]);

  const submit = form.handleSubmit(onSubmit, {
    root: () => formRef.current,
    fallback: "Langganan belum aktif. Periksa koneksi lalu coba lagi.",
  });

  return (
    <Modal
      open={open}
      onClose={onClose}
      title="Aktifkan Langganan Manual"
      description={
        tenant
          ? `Aktifkan langganan ${tenant.businessName} tanpa pembayaran lewat gateway. Alasan dan bukti dicatat atas nama Anda.`
          : undefined
      }
      size="sm"
      busy={form.submitting}
      footer={
        <>
          <Button variant="secondary" onClick={onClose} disabled={form.submitting}>
            Batal
          </Button>
          <Button type="submit" form="activate-subscription-form" disabled={form.submitting || activePlans.length === 0}>
            {form.submitting ? "Mengaktifkan..." : "Aktifkan langganan"}
          </Button>
        </>
      }
    >
      <form id="activate-subscription-form" ref={formRef} noValidate onSubmit={(e) => void submit(e)} className="flex flex-col gap-4">
        <FormErrorSummary errors={form.errorList} formError={form.formError} labels={FIELD_LABELS} root={() => formRef.current} />
        {activePlans.length === 0 ? (
          <p className="rounded-md border border-warning/30 bg-warning-soft px-3.5 py-2.5 text-[13px] font-medium text-warning-text">
            Belum ada paket ElProof yang aktif. Tambahkan dulu di menu Paket.
          </p>
        ) : (
          <Field label="Paket Langganan" required name="planId" error={errors.planId}>
            <Select value={values.planId} onChange={(e) => set("planId", e.target.value)}>
              {activePlans.map((plan) => (
                <option key={plan.id} value={plan.id}>
                  {plan.name} ({formatCurrency(plan.price)} / {plan.durationMonths} bulan)
                </option>
              ))}
            </Select>
          </Field>
        )}
        <Field label="Alasan Aktivasi" required name="reason" error={errors.reason}>
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
          name="proofReference"
          error={errors.proofReference}
          hint="No. transfer, no. tagihan, atau tautan dokumen kesepakatan."
        >
          <Input
            value={values.proofReference}
            onChange={(e) => set("proofReference", e.target.value)}
            onBlur={() => form.blur("proofReference")}
            placeholder="cth. TRF-BCA-20261106-001"
            maxLength={255}
          />
        </Field>
        <Field label="Catatan" required={values.reason === "other"} name="note" error={errors.note}>
          <Textarea
            value={values.note}
            onChange={(e) => set("note", e.target.value)}
            onBlur={() => form.blur("note")}
            rows={3}
            maxLength={500}
            placeholder="Keterangan tambahan untuk tim"
          />
        </Field>
        <p className="flex items-start gap-1.5 text-[12px] text-text-secondary">
          <ShieldCheck className="mt-0.5 h-3.5 w-3.5 shrink-0" />
          Dicatat sebagai transaksi &quot;Diaktifkan Admin&quot;, tidak dihitung sebagai pendapatan, dan masuk audit log.
        </p>
      </form>
    </Modal>
  );
}
