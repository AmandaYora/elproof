import { useRef, useState } from "react";
import { Navigate, useNavigate } from "react-router-dom";
import axios from "axios";
import { Check, Copy, ShieldCheck } from "lucide-react";
import { Button } from "@/shared/components/ui/Button";
import { Field, Input } from "@/shared/components/ui/Input";
import { QrCode } from "@/shared/components/ui/QrCode";
import { httpClient } from "@/shared/services/http-client";
import { API } from "@/shared/services/api-endpoints";
import { useAuthStore, type AuthSession } from "@/shared/stores/useAuthStore";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import { getApiErrorMessage } from "@/shared/lib/api-error";
import { logoutAndRedirect } from "@/shared/lib/auth-actions";
import { mfaCodeSchema } from "@/modules/auth/schemas/login.schema";
import { useZodForm } from "@/shared/forms/useZodForm";
import { FormErrorSummary } from "@/shared/forms/FormErrorSummary";

interface MfaSetup {
  secret: string;
  otpauthUrl: string;
}

/** "JBSWY3DPEHPK3PXP" -> "JBSW Y3DP EHPK 3PXP" — easier to type by hand. */
function groupKey(secret: string) {
  return secret.replace(/(.{4})/g, "$1 ").trim();
}

/**
 * Mandatory two-factor enrolment for Platform Console admins. Reached right
 * after a password login while the session has no second factor; the API
 * refuses every other console call until this is done.
 */
export default function MfaSetupPage() {
  const navigate = useNavigate();
  const session = useAuthStore((s) => s.session);
  const [setup, setSetup] = useState<MfaSetup | null>(null);
  // One message for the whole page (a failed key, a wrong code), shown at the top.
  const form = useZodForm({ schema: mfaCodeSchema, initial: { code: "" } });
  const formRef = useRef<HTMLFormElement>(null);
  const setError = form.setFormError;
  const [alreadyEnabled, setAlreadyEnabled] = useState(false);
  const [busy, setBusy] = useState(false);
  const [copied, setCopied] = useState(false);

  if (session?.mfa) {
    return <Navigate to={ROUTE_PATHS.platformDashboard} replace />;
  }

  async function startSetup() {
    setError(null);
    setBusy(true);
    try {
      const res = await httpClient.post(API.auth.mfaSetup);
      setSetup(res.data.data as MfaSetup);
    } catch (err) {
      if (axios.isAxiosError(err) && err.response?.status === 409) {
        // Enrolled earlier, but this session predates it: sign in again.
        setAlreadyEnabled(true);
      } else {
        setError(getApiErrorMessage(err, "Kunci verifikasi gagal dibuat"));
      }
    } finally {
      setBusy(false);
    }
  }

  const enable = form.handleSubmit(
    async ({ code }) => {
      const res = await httpClient.post(API.auth.mfaEnable, { code });
      useAuthStore.getState().login(res.data.data as AuthSession);
      navigate(ROUTE_PATHS.platformDashboard, { replace: true });
    },
    { root: () => formRef.current, fallback: "Kode verifikasi salah atau sudah kedaluwarsa", known: (key) => key === "code" },
  );

  async function copyKey() {
    if (!setup) return;
    try {
      await navigator.clipboard.writeText(setup.secret);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      setCopied(false);
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-background px-4 py-12">
      <div className="w-full max-w-md rounded-xl border border-border bg-surface p-8 shadow-sm sm:p-10">
        <div className="mb-6 flex h-12 w-12 items-center justify-center rounded-xl bg-primary text-primary-foreground">
          <ShieldCheck className="h-5 w-5" />
        </div>
        <h1 className="text-2xl font-bold text-text-primary">Aktifkan verifikasi dua langkah</h1>
        <p className="mt-2 text-[13.5px] leading-relaxed text-text-secondary">
          Platform Console membuka data semua WO, jadi setiap admin wajib memakai kode dari aplikasi authenticator,
          misalnya Google Authenticator, Microsoft Authenticator, atau 1Password.
        </p>

        <div className="mt-6 empty:hidden">
          <FormErrorSummary errors={form.errorList} formError={form.formError} root={() => formRef.current} />
        </div>

        {alreadyEnabled ? (
          <div className="mt-6 flex flex-col gap-4">
            <p className="text-[13.5px] leading-relaxed text-text-secondary">
              Verifikasi dua langkah untuk akun ini sudah aktif. Masuk ulang dan masukkan kode dari aplikasi authenticator
              Anda.
            </p>
            <Button className="w-full justify-center" onClick={() => void logoutAndRedirect(navigate)}>
              Masuk ulang
            </Button>
          </div>
        ) : !setup ? (
          <div className="mt-8 flex flex-col gap-3">
            <Button className="w-full justify-center" onClick={() => void startSetup()} disabled={busy}>
              {busy ? "Membuat kunci..." : "Mulai penyiapan"}
            </Button>
          </div>
        ) : (
          <ol className="mt-8 flex flex-col gap-6">
            <li className="flex flex-col gap-3">
              <span className="text-[13.5px] font-semibold text-text-primary">1. Tambahkan ElProof ke aplikasi authenticator</span>
              <span className="text-[13px] leading-relaxed text-text-secondary">
                Di aplikasi authenticator, pilih tambah akun lalu pindai kode QR ini. Jika tidak bisa memindai, masukkan
                kunci di bawahnya secara manual (jenis: berbasis waktu).
              </span>
              <QrCode value={setup.otpauthUrl} label="Kode QR untuk menambahkan ElProof ke aplikasi authenticator" className="mx-auto" />
              <code className="select-all rounded-md bg-surface-muted px-3 py-2.5 text-center font-mono text-[14px] font-semibold leading-relaxed tracking-wide text-text-primary [word-spacing:0.35em]">
                {groupKey(setup.secret)}
              </code>
              <div className="flex flex-wrap gap-2">
                <Button
                  variant="secondary"
                  size="sm"
                  icon={copied ? <Check className="h-3.5 w-3.5" /> : <Copy className="h-3.5 w-3.5" />}
                  onClick={() => void copyKey()}
                >
                  {copied ? "Kunci tersalin" : "Salin kunci"}
                </Button>
                <a
                  href={setup.otpauthUrl}
                  className="inline-flex h-8 items-center rounded-md border border-border bg-card px-3 text-[13px] font-semibold text-text-primary hover:bg-surface-muted"
                >
                  Buka di aplikasi authenticator (HP)
                </a>
              </div>
            </li>
            <li className="flex flex-col gap-3">
              <span className="text-[13.5px] font-semibold text-text-primary">2. Masukkan kode yang muncul</span>
              <form ref={formRef} noValidate onSubmit={(e) => void enable(e)} className="flex flex-col gap-3">
                <Field label="Kode 6 digit" required name="code" error={form.errors.code}>
                  <Input
                    value={form.values.code}
                    onChange={(e) => {
                      form.set("code", e.target.value.replace(/\D/g, "").slice(0, 6));
                      setError(null);
                    }}
                    inputMode="numeric"
                    autoComplete="one-time-code"
                    placeholder="123456"
                    className="font-mono tracking-[0.3em]"
                  />
                </Field>
                <Button type="submit" className="w-full justify-center" disabled={form.submitting}>
                  {form.submitting ? "Memeriksa kode..." : "Aktifkan verifikasi"}
                </Button>
              </form>
            </li>
          </ol>
        )}

        <button
          type="button"
          onClick={() => void logoutAndRedirect(navigate)}
          className="mt-6 text-[13px] font-medium text-text-secondary hover:text-text-primary"
        >
          Keluar
        </button>
      </div>
    </div>
  );
}
