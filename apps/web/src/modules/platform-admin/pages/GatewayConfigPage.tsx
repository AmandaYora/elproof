import { useEffect, useRef, useState, type FormEvent } from "react";
import { ShieldCheck, KeyRound } from "lucide-react";
import { Card, CardHeader, CardContent } from "@/shared/components/ui/Card";
import { Button } from "@/shared/components/ui/Button";
import { Badge } from "@/shared/components/ui/Badge";
import { Field, Input, Select } from "@/shared/components/ui/Input";
import { usePaymentGatewayStore } from "@/modules/platform-admin/stores/usePaymentGatewayStore";
import { getApiErrorMessage } from "@/shared/lib/api-error";
import { ConfirmDialog } from "@/shared/components/ui/ConfirmDialog";
import { gatewayConfigSchema } from "@/modules/platform-admin/schemas/gateway.schema";
import { useZodForm } from "@/shared/forms/useZodForm";
import { FormErrorSummary } from "@/shared/forms/FormErrorSummary";

const FIELD_LABELS: Record<string, string> = {
  activeProvider: "Provider", isSandbox: "Mode", tripayMerchantCode: "Merchant Code (Tripay)", tripayApiKey: "API Key (Tripay)", tripayPrivateKey: "Private Key (Tripay)",
};

const PROVIDER_OPTIONS = [
  { value: "", label: "Tidak ada (mode simulasi — charge akan ditolak)" },
  { value: "tripay", label: "Tripay" },
];

export default function GatewayConfigPage() {
  const config = usePaymentGatewayStore((s) => s.config);
  const isLoading = usePaymentGatewayStore((s) => s.isLoading);
  const fetchConfig = usePaymentGatewayStore((s) => s.fetchConfig);
  const updateConfig = usePaymentGatewayStore((s) => s.updateConfig);

  const form = useZodForm({
    schema: gatewayConfigSchema,
    initial: { activeProvider: "", isSandbox: true, tripayMerchantCode: "", tripayApiKey: "", tripayPrivateKey: "" },
  });
  const { values, errors, set } = form;
  const { activeProvider, isSandbox } = values;
  const formRef = useRef<HTMLFormElement>(null);
  const [successMessage, setSuccessMessage] = useState<string | null>(null);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [confirmOpen, setConfirmOpen] = useState(false);

  function load() {
    setLoadError(null);
    fetchConfig().catch((err) => setLoadError(getApiErrorMessage(err, "Konfigurasi gateway gagal dimuat")));
  }

  useEffect(() => {
    load();
  }, [fetchConfig]);

  useEffect(() => {
    if (!config) return;
    // Only the stored settings: a key being typed is not touched.
    set("activeProvider", config.activeProvider);
    set("isSandbox", config.isSandbox);
    set("tripayMerchantCode", config.tripayMerchantCode);
  }, [config, set]);

  // Switching provider or mode changes where real payments go: always ask.
  const providerChanged = config !== null && activeProvider !== config.activeProvider;
  const modeChanged = config !== null && isSandbox !== config.isSandbox;
  const needsConfirmation = providerChanged || modeChanged;

  // Errors are not caught here: the form shows a 422 on its fields and any other failure above them
  // (the confirm dialog shows the same message while it is open).
  const save = form.handleSubmit(
    async (v) => {
      setSuccessMessage(null);
      await updateConfig({
        activeProvider: v.activeProvider, isSandbox: v.isSandbox, tripayMerchantCode: v.tripayMerchantCode,
        tripayApiKey: v.tripayApiKey, tripayPrivateKey: v.tripayPrivateKey,
      });
      set("tripayApiKey", "");
      set("tripayPrivateKey", "");
      setConfirmOpen(false);
      setSuccessMessage("Konfigurasi gateway berhasil disimpan.");
    },
    { root: () => formRef.current, fallback: "Gagal menyimpan konfigurasi gateway", known: (key) => key in FIELD_LABELS },
  );

  function requestSave(e: FormEvent) {
    if (needsConfirmation) {
      e.preventDefault();
      form.setFormError(null);
      setConfirmOpen(true);
      return;
    }
    void save(e);
  }

  if (isLoading && !config) {
    return <div className="py-20 text-center text-sm text-text-secondary">Memuat...</div>;
  }

  // Never show an editable form built from defaults: saving it would switch
  // the live gateway off and wipe the merchant code.
  if (!config) {
    return (
      <div className="flex max-w-xl flex-col gap-4">
        <h1 className="text-xl font-bold text-text-primary">Konfigurasi Gateway</h1>
        <p role="alert" className="rounded-md border border-danger/30 bg-danger-soft px-3.5 py-2.5 text-[13px] font-medium text-danger-text">
          {loadError ?? "Konfigurasi gateway gagal dimuat"}. Form dikunci supaya pengaturan yang sedang berjalan tidak tertimpa.
        </p>
        <div>
          <Button variant="secondary" onClick={load}>
            Muat ulang
          </Button>
        </div>
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-5">
      <div>
        <h1 className="text-xl font-bold text-text-primary">Konfigurasi Gateway</h1>
        <p className="mt-1 text-[13px] text-text-secondary">
          Kredensial ini dipakai untuk memproses pembayaran langganan ElProof, dan juga oleh aplikasi eksternal
          yang sudah didaftarkan di halaman Manajemen Aplikasi.
        </p>
      </div>

      {successMessage && (
        <p className="max-w-xl rounded-md border border-success/30 bg-success-soft px-3.5 py-2.5 text-[13px] font-medium text-success-text">
          {successMessage}
        </p>
      )}

      <Card className="max-w-xl">
        <CardHeader
          title="Status Saat Ini"
          subtitle={config?.activeProvider ? `Gateway aktif: ${config.activeProvider}` : "Belum ada gateway aktif — mode simulasi"}
        />
        <CardContent className="flex flex-wrap items-center gap-3 py-4">
          {config?.activeProvider ? <Badge tone="success">Aktif</Badge> : <Badge tone="neutral">Nonaktif</Badge>}
          <Badge tone={config?.isSandbox ? "warning" : "info"}>{config?.isSandbox ? "Sandbox" : "Production"}</Badge>
          <span className="flex items-center gap-1.5 text-[12.5px] text-text-secondary">
            <KeyRound className="h-3.5 w-3.5" />
            API Key: {config?.hasTripayApiKey ? "sudah diatur" : "belum diatur"}
          </span>
          <span className="flex items-center gap-1.5 text-[12.5px] text-text-secondary">
            <KeyRound className="h-3.5 w-3.5" />
            Private Key: {config?.hasTripayPrivateKey ? "sudah diatur" : "belum diatur"}
          </span>
        </CardContent>
      </Card>

      <Card className="max-w-xl">
        <CardHeader title="Ubah Konfigurasi" subtitle="Kredensial disimpan terenkripsi — tidak pernah ditampilkan ulang setelah disimpan." />
        <form ref={formRef} noValidate onSubmit={requestSave} className="flex flex-col gap-4 px-5 py-4">
          <FormErrorSummary errors={form.errorList} formError={form.formError} labels={FIELD_LABELS} root={() => formRef.current} />
          <Field label="Provider" name="activeProvider" error={errors.activeProvider}>
            <Select value={activeProvider} onChange={(e) => set("activeProvider", e.target.value)}>
              {PROVIDER_OPTIONS.map((opt) => (
                <option key={opt.value} value={opt.value}>{opt.label}</option>
              ))}
            </Select>
          </Field>

          <Field label="Mode" name="isSandbox" error={errors.isSandbox}>
            <Select value={isSandbox ? "sandbox" : "production"} onChange={(e) => set("isSandbox", e.target.value === "sandbox")}>
              <option value="sandbox">Sandbox (uji coba)</option>
              <option value="production">Production (transaksi nyata)</option>
            </Select>
          </Field>

          <Field label="Merchant Code (Tripay)" name="tripayMerchantCode" error={errors.tripayMerchantCode}>
            <Input value={values.tripayMerchantCode} onChange={(e) => set("tripayMerchantCode", e.target.value)} onBlur={() => form.blur("tripayMerchantCode")} placeholder="Contoh: T12345" />
          </Field>

          <Field label="API Key (Tripay)" name="tripayApiKey" error={errors.tripayApiKey} hint="Kosongkan untuk mempertahankan nilai yang sudah tersimpan">
            <Input
              type="password"
              value={values.tripayApiKey}
              onChange={(e) => set("tripayApiKey", e.target.value)}
              placeholder={config?.hasTripayApiKey ? "•••••••• (sudah diatur)" : "Masukkan API key"}
            />
          </Field>

          <Field label="Private Key (Tripay)" name="tripayPrivateKey" error={errors.tripayPrivateKey} hint="Kosongkan untuk mempertahankan nilai yang sudah tersimpan">
            <Input
              type="password"
              value={values.tripayPrivateKey}
              onChange={(e) => set("tripayPrivateKey", e.target.value)}
              placeholder={config?.hasTripayPrivateKey ? "•••••••• (sudah diatur)" : "Masukkan private key"}
            />
          </Field>

          <Button type="submit" className="w-full justify-center" disabled={form.submitting} icon={<ShieldCheck className="h-4 w-4" />}>
            {form.submitting ? "Menyimpan..." : "Simpan konfigurasi"}
          </Button>
        </form>
      </Card>

      <ConfirmDialog
        open={confirmOpen}
        title="Ubah gateway pembayaran?"
        confirmLabel="Simpan perubahan gateway"
        confirmText={!isSandbox || !activeProvider ? "UBAH GATEWAY" : undefined}
        isSubmitting={form.submitting}
        error={form.formError}
        onConfirm={() => void save()}
        onClose={() => setConfirmOpen(false)}
      >
        <ul className="flex list-disc flex-col gap-1.5 pl-5">
          {providerChanged && (
            <li>
              Provider berubah dari <b className="text-text-primary">{config.activeProvider || "tidak ada"}</b> menjadi{" "}
              <b className="text-text-primary">{activeProvider || "tidak ada"}</b>.
              {!activeProvider && " Semua pembayaran langganan WO dan aplikasi eksternal akan ditolak."}
            </li>
          )}
          {modeChanged && (
            <li>
              Mode berubah menjadi <b className="text-text-primary">{isSandbox ? "Sandbox" : "Production"}</b>.
              {isSandbox ? " Pembayaran baru tidak akan menggerakkan uang sungguhan." : " Pembayaran baru akan menggerakkan uang sungguhan."}
            </li>
          )}
          <li>Berlaku segera untuk langganan ElProof dan semua aplikasi eksternal.</li>
        </ul>
      </ConfirmDialog>
    </div>
  );
}
