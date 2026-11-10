import { useEffect, useRef, useState } from "react";
import { Modal } from "@/shared/components/ui/Modal";
import { Button } from "@/shared/components/ui/Button";
import { Input, Field, Select } from "@/shared/components/ui/Input";
import { CurrencyInput } from "@/shared/components/ui/CurrencyInput";
import { useZodForm, type FormValues } from "@/shared/forms/useZodForm";
import { FormErrorSummary } from "@/shared/forms/FormErrorSummary";
import { isStaleConflict } from "@/shared/lib/concurrency";
import { formatCurrency } from "@/shared/lib/formatters";
import { useConflictStore } from "@/shared/stores/useConflictStore";
import { useSubscriptionPlanStore } from "@/shared/stores/useSubscriptionPlanStore";
import { appLabel, ELPROOF_APP_ID, type SubscriptionPlan } from "@/shared/data/subscriptionPlans";
import { voucherSchema, type VoucherFormValues } from "@/modules/platform-admin/schemas/voucher.schema";
import type { PaymentApp } from "@/modules/platform-admin/stores/useAppsStore";
import type { Voucher, VoucherBody } from "@/modules/platform-admin/stores/useVoucherStore";

const FIELD_LABELS: Record<string, string> = {
  code: "Kode",
  appId: "Aplikasi",
  planId: "Paket",
  discountType: "Jenis potongan",
  discountValue: "Nilai potongan",
  maxDiscount: "Batas maks. potongan",
  validFrom: "Berlaku mulai",
  validUntil: "Berlaku sampai",
  quota: "Kuota total",
};

type Values = FormValues<typeof voucherSchema>;

function toFormValues(v: Voucher | undefined, defaultAppId: string): Values {
  if (!v) {
    return {
      code: "", appId: defaultAppId, planId: "", discountType: "percent", discountValue: 0, maxDiscount: 0,
      validFrom: "", validUntil: "", quota: "", oncePerCustomer: false,
    };
  }
  return {
    code: v.code,
    appId: v.appId,
    planId: v.planId !== null ? String(v.planId) : "",
    discountType: v.discountType,
    discountValue: v.discountValue,
    maxDiscount: v.maxDiscount ?? 0,
    validFrom: v.validFrom ?? "",
    validUntil: v.validUntil ?? "",
    quota: v.quota !== null ? String(v.quota) : "",
    oncePerCustomer: v.oncePerCustomer,
  };
}

/** The request body for the parsed form: empty optional fields become null ("no limit"). */
export function toVoucherBody(v: VoucherFormValues): VoucherBody {
  return {
    code: v.code,
    appId: v.appId,
    planId: v.planId ? Number(v.planId) : null,
    discountType: v.discountType,
    discountValue: v.discountValue,
    maxDiscount: v.discountType === "percent" && v.maxDiscount > 0 ? v.maxDiscount : null,
    validFrom: v.validFrom,
    validUntil: v.validUntil,
    quota: v.quota !== "" ? Number(v.quota) : null,
    oncePerCustomer: v.oncePerCustomer,
  };
}

type PlanOptions = { appId: string; status: "loading" | "ready" | "error"; plans: SubscriptionPlan[] };

interface VoucherFormModalProps {
  open: boolean;
  onClose: () => void;
  /** Saves, then closes. Throw to keep it open: a 422 lands on its fields. */
  onSubmit: (body: VoucherBody) => Promise<void>;
  /** The voucher being edited (its updatedAt is the If-Match of the save); none = a new one. */
  initialVoucher?: Voucher;
  /** The edit was refused because someone saved first: this is the voucher as it is now. */
  onStale?: (latest: Voucher) => void;
  apps: PaymentApp[];
  /** App chosen for a new voucher ("" = the person picks one). */
  defaultAppId?: string;
}

export function VoucherFormModal({ open, onClose, onSubmit, initialVoucher, onStale, apps, defaultAppId = "" }: VoucherFormModalProps) {
  const form = useZodForm({ schema: voucherSchema, initial: toFormValues(initialVoucher, defaultAppId) });
  const { values, errors, set, blur } = form;
  const formRef = useRef<HTMLFormElement>(null);
  const [planOptions, setPlanOptions] = useState<PlanOptions>({ appId: "", status: "ready", plans: [] });
  // The App whose plans were asked for last: an older answer for another App must not fill this picker.
  const plansFor = useRef("");

  // The modal stays mounted while closed: start clean each time it opens, or when another voucher is opened. Keyed on the
  // id, not the object, so taking the latest version after a refused save does not wipe the message about it.
  useEffect(() => {
    if (open) form.reset(toFormValues(initialVoucher, defaultAppId));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, initialVoucher?.id, defaultAppId]);

  // The plans of the chosen App, asked of the server for that App alone (never read from a list filtered elsewhere).
  useEffect(() => {
    const appId = values.appId;
    plansFor.current = appId;
    if (!open || !appId) {
      setPlanOptions({ appId, status: "ready", plans: [] });
      return;
    }
    setPlanOptions({ appId, status: "loading", plans: [] });
    useSubscriptionPlanStore
      .getState()
      .fetchPlansForApp(appId)
      .then((plans) => {
        if (plansFor.current === appId) setPlanOptions({ appId, status: "ready", plans: plans.filter((p) => p.appId === appId) });
      })
      .catch(() => {
        if (plansFor.current === appId) setPlanOptions({ appId, status: "error", plans: [] });
      });
  }, [open, values.appId]);

  const appOptions = [
    { appId: ELPROOF_APP_ID, label: appLabel(undefined, ELPROOF_APP_ID) },
    ...apps.filter((a) => a.appId !== ELPROOF_APP_ID).map((a) => ({ appId: a.appId, label: a.isActive ? a.name : `${a.name} (nonaktif)` })),
  ];
  if (values.appId && !appOptions.some((o) => o.appId === values.appId)) {
    appOptions.push({ appId: values.appId, label: appLabel(undefined, values.appId) });
  }

  const plans = planOptions.appId === values.appId ? planOptions.plans : [];
  const planChoices = plans.map((p) => ({ value: p.id, label: p.isActive ? `${p.name} · ${formatCurrency(p.price)}` : `${p.name} (nonaktif)` }));
  // While the plans load (or if they could not), the voucher's own plan still shows by name.
  if (values.planId && !planChoices.some((c) => c.value === values.planId)) {
    const own = initialVoucher && String(initialVoucher.planId) === values.planId ? initialVoucher.planName : null;
    planChoices.push({ value: values.planId, label: own ?? `Paket #${values.planId}` });
  }
  const chosenPlan = plans.find((p) => p.id === values.planId);
  const isPercent = values.discountType === "percent";

  function chooseApp(appId: string) {
    if (appId === values.appId) return;
    set("appId", appId);
    set("planId", ""); // a plan belongs to one App: another App's plan never stays chosen
  }

  const submit = form.handleSubmit(
    async (parsed) => {
      try {
        await onSubmit(toVoucherBody(parsed));
      } catch (err) {
        if (initialVoucher && isStaleConflict(err)) {
          const latest = (err as { response?: { data?: { data?: Voucher } } }).response?.data?.data;
          // Answered here, in the form, with the latest data in place (the console has no global conflict dialog).
          useConflictStore.getState().dismiss();
          if (latest) {
            onStale?.(latest);
            form.reset(toFormValues(latest, defaultAppId));
          }
          form.setFormError(
            "Voucher ini baru saja diubah orang lain, jadi perubahan Anda belum tersimpan. Isian di bawah sudah diganti dengan data terbaru; ulangi perubahan Anda lalu simpan lagi."
          );
          return;
        }
        throw err;
      }
    },
    {
      root: () => formRef.current,
      fallback: "Voucher belum tersimpan. Periksa koneksi lalu coba lagi.",
      known: (key) => key in FIELD_LABELS,
    }
  );

  return (
    <Modal
      open={open}
      onClose={onClose}
      busy={form.submitting}
      title={initialVoucher ? "Ubah Voucher" : "Tambah Voucher"}
      description="Voucher memotong harga paket satu aplikasi saat pelanggan membayar. Harga akhir tidak pernah Rp0."
      footer={
        <>
          <Button variant="secondary" onClick={onClose} disabled={form.submitting}>
            Batal
          </Button>
          <Button type="submit" form="voucher-form" disabled={form.submitting}>
            {form.submitting ? "Menyimpan..." : initialVoucher ? "Simpan perubahan" : "Simpan voucher"}
          </Button>
        </>
      }
    >
      <form id="voucher-form" ref={formRef} noValidate onSubmit={(e) => void submit(e)} className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <div className="sm:col-span-2">
          <FormErrorSummary errors={form.errorList} formError={form.formError} labels={FIELD_LABELS} root={() => formRef.current} />
        </div>
        <div className="sm:col-span-2">
          <Field label="Kode" required name="code" error={errors.code} hint="Huruf besar, angka, - atau _ (3–32 karakter). Pelanggan mengetik kode ini saat membayar.">
            <Input
              value={values.code}
              onChange={(e) => set("code", e.target.value.toUpperCase())}
              onBlur={() => blur("code")}
              placeholder="cth. HEMAT10"
              autoComplete="off"
              spellCheck={false}
              maxLength={32}
              className="font-mono uppercase placeholder:normal-case placeholder:font-sans"
            />
          </Field>
        </div>
        <Field label="Aplikasi" required name="appId" error={errors.appId}>
          <Select value={values.appId} onChange={(e) => chooseApp(e.target.value)} aria-label="Aplikasi" placeholder="Pilih aplikasi...">
            {appOptions.map((o) => (
              <option key={o.appId} value={o.appId}>
                {o.label}
              </option>
            ))}
          </Select>
        </Field>
        <Field
          label="Paket"
          name="planId"
          error={errors.planId}
          hint={
            !values.appId
              ? "Pilih aplikasi dulu."
              : planOptions.status === "loading"
                ? "Memuat paket aplikasi ini..."
                : planOptions.status === "error"
                  ? "Daftar paket belum bisa dimuat. Voucher tetap bisa berlaku untuk semua paket."
                  : "Kosongkan agar berlaku untuk semua paket aplikasi ini."
          }
        >
          <Select value={values.planId} onChange={(e) => set("planId", e.target.value)} aria-label="Paket" disabled={!values.appId}>
            <option value="">Semua paket aplikasi ini</option>
            {planChoices.map((c) => (
              <option key={c.value} value={c.value}>
                {c.label}
              </option>
            ))}
          </Select>
        </Field>
        <Field label="Jenis potongan" required name="discountType" error={errors.discountType}>
          <Select value={values.discountType} onChange={(e) => set("discountType", e.target.value as Values["discountType"])} aria-label="Jenis potongan">
            <option value="percent">Persen (%)</option>
            <option value="amount">Nominal (Rp)</option>
          </Select>
        </Field>
        <Field
          label={isPercent ? "Nilai potongan (%)" : "Nilai potongan (Rp)"}
          required
          name="discountValue"
          error={errors.discountValue}
          hint={chosenPlan ? `Harga paket ${formatCurrency(chosenPlan.price)}; potongan harus di bawahnya.` : isPercent ? "1–99%" : undefined}
        >
          {isPercent ? (
            <Input
              type="number"
              inputMode="numeric"
              min={1}
              max={99}
              step={1}
              value={values.discountValue === 0 ? "" : values.discountValue}
              onChange={(e) => set("discountValue", e.target.value === "" ? 0 : Math.trunc(Number(e.target.value)))}
              onBlur={() => blur("discountValue")}
              placeholder="10"
            />
          ) : (
            <CurrencyInput value={values.discountValue} onChange={(n) => set("discountValue", n)} placeholder="50.000" aria-invalid={Boolean(errors.discountValue)} />
          )}
        </Field>
        {isPercent && (
          <div className="sm:col-span-2">
            <Field label="Batas maks. potongan (Rp)" name="maxDiscount" error={errors.maxDiscount} hint="Opsional. 0 = tanpa batas.">
              <CurrencyInput value={values.maxDiscount} onChange={(n) => set("maxDiscount", n)} aria-invalid={Boolean(errors.maxDiscount)} />
            </Field>
          </div>
        )}
        <Field label="Berlaku mulai" name="validFrom" error={errors.validFrom} hint="Opsional. Kosong = langsung berlaku.">
          <Input
            type="date"
            value={values.validFrom}
            max={values.validUntil || undefined}
            onChange={(e) => set("validFrom", e.target.value)}
            onBlur={() => blur("validFrom")}
          />
        </Field>
        <Field label="Berlaku sampai" name="validUntil" error={errors.validUntil} hint="Opsional, termasuk hari itu. Kosong = tanpa batas.">
          <Input
            type="date"
            value={values.validUntil}
            min={values.validFrom || undefined}
            onChange={(e) => set("validUntil", e.target.value)}
            onBlur={() => blur("validUntil")}
          />
        </Field>
        <Field label="Kuota total" name="quota" error={errors.quota} hint="Opsional. Kosong = tanpa batas. Pembayaran yang masih menunggu ikut dihitung.">
          <Input
            type="number"
            inputMode="numeric"
            min={1}
            step={1}
            value={values.quota}
            onChange={(e) => set("quota", e.target.value)}
            onBlur={() => blur("quota")}
            placeholder="Tanpa batas"
          />
        </Field>
        <div className="flex flex-col gap-1.5 sm:col-span-2">
          <label className="flex items-start gap-2.5 rounded-md border border-border px-3 py-2.5 text-[13px] text-foreground">
            <input
              type="checkbox"
              className="mt-0.5 size-4 shrink-0 accent-primary"
              checked={values.oncePerCustomer}
              onChange={(e) => set("oncePerCustomer", e.target.checked)}
              aria-describedby="voucher-once-hint"
            />
            <span className="flex flex-col gap-0.5">
              <span className="font-medium">Satu kali per pelanggan</span>
              <span id="voucher-once-hint" className="text-xs text-muted-foreground">
                Pelanggan adalah usaha WO (tenant) untuk ElProof, atau aplikasi itu sendiri untuk aplikasi lain.
              </span>
            </span>
          </label>
        </div>
      </form>
    </Modal>
  );
}
