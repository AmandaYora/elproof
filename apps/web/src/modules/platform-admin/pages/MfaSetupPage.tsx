import { useState } from "react";
import { Navigate, useNavigate } from "react-router-dom";
import axios from "axios";
import { Check, Copy, ShieldCheck } from "lucide-react";
import { Button } from "@/shared/components/ui/Button";
import { Field, Input } from "@/shared/components/ui/Input";
import { httpClient } from "@/shared/services/http-client";
import { API } from "@/shared/services/api-endpoints";
import { useAuthStore, type AuthSession } from "@/shared/stores/useAuthStore";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import { getApiErrorMessage } from "@/shared/lib/api-error";
import { logoutAndRedirect } from "@/shared/lib/auth-actions";

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
  const [code, setCode] = useState("");
  const [error, setError] = useState<string | null>(null);
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

  async function enable() {
    if (code.trim().length !== 6) {
      setError("Masukkan 6 digit kode dari aplikasi authenticator.");
      return;
    }
    setError(null);
    setBusy(true);
    try {
      const res = await httpClient.post(API.auth.mfaEnable, { code: code.trim() });
      useAuthStore.getState().login(res.data.data as AuthSession);
      navigate(ROUTE_PATHS.platformDashboard, { replace: true });
    } catch (err) {
      setError(getApiErrorMessage(err, "Kode verifikasi salah atau sudah kedaluwarsa"));
    } finally {
      setBusy(false);
    }
  }

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
        <div className="mb-6 flex h-12 w-12 items-center justify-center rounded-xl bg-slate-800 text-white">
          <ShieldCheck className="h-5 w-5" />
        </div>
        <h1 className="text-2xl font-bold text-text-primary">Aktifkan verifikasi dua langkah</h1>
        <p className="mt-2 text-[13.5px] leading-relaxed text-text-secondary">
          Platform Console membuka data semua WO, jadi setiap admin wajib memakai kode dari aplikasi authenticator,
          misalnya Google Authenticator, Microsoft Authenticator, atau 1Password.
        </p>

        {error && (
          <p role="alert" className="mt-6 rounded-md border border-danger/30 bg-danger-soft px-3.5 py-2.5 text-[13px] font-medium text-danger">
            {error}
          </p>
        )}

        {alreadyEnabled ? (
          <div className="mt-6 flex flex-col gap-4">
            <p className="text-[13.5px] leading-relaxed text-text-secondary">
              Verifikasi dua langkah untuk akun ini sudah aktif. Masuk ulang dan masukkan kode dari aplikasi authenticator
              Anda.
            </p>
            <Button variant="neutral" className="w-full justify-center" onClick={() => void logoutAndRedirect(navigate)}>
              Masuk ulang
            </Button>
          </div>
        ) : !setup ? (
          <div className="mt-8 flex flex-col gap-3">
            <Button variant="neutral" className="w-full justify-center" onClick={() => void startSetup()} disabled={busy}>
              {busy ? "Membuat kunci..." : "Mulai penyiapan"}
            </Button>
          </div>
        ) : (
          <ol className="mt-8 flex flex-col gap-6">
            <li className="flex flex-col gap-3">
              <span className="text-[13.5px] font-semibold text-text-primary">1. Tambahkan ElProof ke aplikasi authenticator</span>
              <span className="text-[13px] leading-relaxed text-text-secondary">
                Di aplikasi, pilih tambah akun lalu masukkan kunci ini secara manual (jenis: berbasis waktu). Di HP, Anda
                juga bisa membuka tautan di bawah.
              </span>
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
                  className="inline-flex h-8 items-center rounded-md border border-border bg-white px-3 text-[13px] font-semibold text-text-primary hover:bg-surface-muted"
                >
                  Buka di aplikasi authenticator
                </a>
              </div>
            </li>
            <li className="flex flex-col gap-3">
              <span className="text-[13.5px] font-semibold text-text-primary">2. Masukkan kode yang muncul</span>
              <Field label="Kode 6 digit">
                <Input
                  value={code}
                  onChange={(e) => setCode(e.target.value.replace(/\D/g, "").slice(0, 6))}
                  inputMode="numeric"
                  autoComplete="one-time-code"
                  placeholder="123456"
                  className="font-mono tracking-[0.3em]"
                  onKeyDown={(e) => e.key === "Enter" && void enable()}
                />
              </Field>
              <Button variant="neutral" className="w-full justify-center" onClick={() => void enable()} disabled={busy}>
                {busy ? "Memeriksa kode..." : "Aktifkan verifikasi"}
              </Button>
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
