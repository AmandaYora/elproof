import { useEffect, useState } from "react";
import { Eye, EyeOff } from "lucide-react";
import { Modal } from "@/shared/components/ui/Modal";
import { Button } from "@/shared/components/ui/Button";
import { Input, Select, Field } from "@/shared/components/ui/Input";
import {
  tenantSchema,
  tenantCreateSchema,
  type TenantFormValues,
  type TenantCreateFormValues,
} from "@/modules/platform-admin/schemas/tenant.schema";
import type { Tenant } from "@/modules/platform-admin/data/types";
import type { SubscriptionPlan } from "@/shared/data/subscriptionPlans";
import { formatCurrency } from "@/shared/lib/formatters";

interface FormState extends TenantFormValues {
  planId: string;
  password: string;
  confirmPassword: string;
}

function toFormState(tenant?: Tenant, defaultPlanId = ""): FormState {
  if (!tenant) {
    return { businessName: "", ownerName: "", email: "", phone: "", city: "", planId: defaultPlanId, password: "", confirmPassword: "" };
  }
  return {
    businessName: tenant.businessName,
    ownerName: tenant.ownerName,
    email: tenant.email,
    phone: tenant.phone,
    city: tenant.city,
    planId: tenant.planId ?? defaultPlanId,
    password: "",
    confirmPassword: "",
  };
}

interface TenantFormModalProps {
  open: boolean;
  onClose: () => void;
  initialTenant?: Tenant;
  plans: SubscriptionPlan[];
  onSubmitCreate: (values: TenantCreateFormValues) => void;
  onSubmitEdit: (values: TenantFormValues) => void;
}

export function TenantFormModal({ open, onClose, initialTenant, plans, onSubmitCreate, onSubmitEdit }: TenantFormModalProps) {
  const isEditing = Boolean(initialTenant);
  const activePlans = plans.filter((p) => p.isActive);
  const [values, setValues] = useState<FormState>(() => toFormState(initialTenant, activePlans[0]?.id ?? ""));
  const [errors, setErrors] = useState<Partial<Record<keyof FormState, string>>>({});
  const [showPassword, setShowPassword] = useState(false);

  function set<K extends keyof FormState>(key: K, value: FormState[K]) {
    setValues((prev) => ({ ...prev, [key]: value }));
  }

  // Plans load asynchronously (Fase 2) — this modal is mounted once and kept
  // alive while closed, so its initial `useState` can capture an empty plan
  // list before the fetch resolves. Backfill the default once real plans
  // arrive, without clobbering a value the user already picked.
  useEffect(() => {
    if (!isEditing && !values.planId && activePlans.length > 0) {
      setValues((prev) => ({ ...prev, planId: activePlans[0].id }));
    }
  }, [activePlans.length, isEditing]);

  function handleSubmit() {
    if (isEditing) {
      const result = tenantSchema.safeParse(values);
      if (!result.success) {
        const fieldErrors: Partial<Record<keyof FormState, string>> = {};
        for (const issue of result.error.issues) {
          fieldErrors[issue.path[0] as keyof FormState] = issue.message;
        }
        setErrors(fieldErrors);
        return;
      }
      onSubmitEdit(result.data);
      setErrors({});
      return;
    }

    const result = tenantCreateSchema.safeParse(values);
    if (!result.success) {
      const fieldErrors: Partial<Record<keyof FormState, string>> = {};
      for (const issue of result.error.issues) {
        fieldErrors[issue.path[0] as keyof FormState] = issue.message;
      }
      setErrors(fieldErrors);
      return;
    }
    onSubmitCreate(result.data);
    setErrors({});
  }

  function handleClose() {
    setValues(toFormState(initialTenant, activePlans[0]?.id ?? ""));
    setErrors({});
    onClose();
  }

  return (
    <Modal
      open={open}
      onClose={handleClose}
      title={isEditing ? "Ubah Tenant" : "Daftarkan Tenant Baru"}
      description={
        isEditing
          ? "Perbarui informasi bisnis dan owner tenant."
          : "Tenant beserta akun owner akan didaftarkan agar dapat mengakses WO Console."
      }
      footer={
        <>
          <Button variant="secondary" onClick={handleClose}>
            Batal
          </Button>
          <Button onClick={handleSubmit}>{isEditing ? "Simpan Perubahan" : "Daftarkan Tenant"}</Button>
        </>
      }
    >
      <div className="grid grid-cols-1 gap-4">
        <Field label="Nama WO" required hint={errors.businessName}>
          <Input
            value={values.businessName}
            onChange={(e) => set("businessName", e.target.value)}
            placeholder="cth. Anisa Wedding Organizer"
          />
        </Field>
        <Field label="Kota" required hint={errors.city}>
          <Input value={values.city} onChange={(e) => set("city", e.target.value)} placeholder="cth. Jakarta" />
        </Field>
        <Field label="Nama Owner" required hint={errors.ownerName}>
          <Input value={values.ownerName} onChange={(e) => set("ownerName", e.target.value)} placeholder="Nama owner WO" />
        </Field>
        <Field label="Email Owner" required hint={errors.email}>
          <Input type="email" value={values.email} onChange={(e) => set("email", e.target.value)} placeholder="owner@contoh.id" />
        </Field>
        <Field label="No. HP Owner" required hint={errors.phone}>
          <Input value={values.phone} onChange={(e) => set("phone", e.target.value)} placeholder="08xx-xxxx-xxxx" />
        </Field>

        {!isEditing && (
          <>
            <Field label="Paket Langganan" required hint={errors.planId}>
              <Select value={values.planId} onChange={(e) => set("planId", e.target.value)}>
                {activePlans.map((plan) => (
                  <option key={plan.id} value={plan.id}>
                    {plan.name} — {formatCurrency(plan.price)} / {plan.durationMonths} bulan
                  </option>
                ))}
              </Select>
            </Field>
            <Field label="Password Owner" required hint={errors.password}>
              <div className="relative">
                <Input
                  type={showPassword ? "text" : "password"}
                  value={values.password}
                  onChange={(e) => set("password", e.target.value)}
                  placeholder="Minimal 8 karakter"
                  className="pr-10"
                />
                <button
                  type="button"
                  onClick={() => setShowPassword((v) => !v)}
                  aria-label={showPassword ? "Sembunyikan password" : "Tampilkan password"}
                  className="absolute right-3 top-1/2 -translate-y-1/2 text-text-secondary hover:text-text-primary"
                >
                  {showPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                </button>
              </div>
            </Field>
            <Field label="Konfirmasi Password" required hint={errors.confirmPassword}>
              <Input
                type={showPassword ? "text" : "password"}
                value={values.confirmPassword}
                onChange={(e) => set("confirmPassword", e.target.value)}
                placeholder="Ulangi password"
              />
            </Field>
          </>
        )}
      </div>
    </Modal>
  );
}
