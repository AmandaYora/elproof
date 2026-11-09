import { useCallback, useEffect, useState, type FormEvent } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import axios from "axios";
import { Eye, EyeOff, Loader2 } from "lucide-react";
import { Button } from "@/shared/components/ui/Button";
import { Field, Input } from "@/shared/components/ui/Input";
import { GoogleSignInButton } from "@/shared/components/auth/GoogleSignInButton";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import { useAuthStore, type AuthSession } from "@/shared/stores/useAuthStore";
import { landingPathFor } from "@/shared/lib/auth-actions";
import { httpClient } from "@/shared/services/http-client";
import { API } from "@/shared/services/api-endpoints";
import { getApiErrorMessage } from "@/shared/lib/api-error";
import { formatLinkExpiry, type AccessLinkPurpose } from "@/shared/lib/access-link";
import { AuthSplitLayout } from "@/modules/auth/components/AuthSplitLayout";
import { usePageMeta } from "@/shared/hooks/usePageMeta";

interface LinkInfo {
  purpose: AccessLinkPurpose;
  tenantName: string;
  displayName: string;
  username?: string;
  email?: string;
  expiresAt: string;
}

function errorCode(err: unknown): string | undefined {
  if (!axios.isAxiosError(err)) return undefined;
  return (err.response?.data as { errors?: { code?: string } } | undefined)?.errors?.code;
}

const MIN_PASSWORD = 8;

/**
 * /undangan/:token — one page for every one-time link (PLAN.md Fase 3):
 * a staff/Owner invitation is accepted by signing in with the invited
 * Google account; a client invitation or password reset lets the client
 * choose their own password. Either way the person ends up signed in.
 */
export default function InvitationPage() {
  usePageMeta({ title: "Undangan", description: "Terima undangan untuk bergabung di ElProof.", noindex: true });
  const { token = "" } = useParams();
  const navigate = useNavigate();
  const [info, setInfo] = useState<LinkInfo | null>(null);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [password, setPassword] = useState("");
  const [confirm, setConfirm] = useState("");
  const [showPassword, setShowPassword] = useState(false);
  const [fieldError, setFieldError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    httpClient
      .get(API.auth.link(token))
      .then((res) => !cancelled && setInfo(res.data.data as LinkInfo))
      .catch((err) => {
        if (cancelled) return;
        setLoadError(
          axios.isAxiosError(err) && !err.response
            ? "Tidak dapat terhubung ke server. Periksa koneksi internet Anda lalu muat ulang halaman."
            : getApiErrorMessage(err, "Link ini tidak berlaku. Minta link baru ke WO Anda."),
        );
      });
    return () => {
      cancelled = true;
    };
  }, [token]);

  const finish = useCallback(
    (session: AuthSession) => {
      useAuthStore.getState().login(session);
      navigate(landingPathFor(session), { replace: true });
    },
    [navigate],
  );

  const acceptWithGoogle = useCallback(
    async (credential: string) => {
      setError(null);
      setBusy(true);
      try {
        const res = await httpClient.post(API.auth.linkGoogle(token), { credential });
        finish(res.data.data as AuthSession);
      } catch (err) {
        const code = errorCode(err);
        if (code === "link_used" || code === "link_revoked" || code === "link_expired") {
          setInfo(null);
          setLoadError(getApiErrorMessage(err, "Link ini sudah tidak berlaku."));
          return;
        }
        setError(getApiErrorMessage(err, "Undangan belum berhasil diterima. Silakan coba lagi."));
      } finally {
        setBusy(false);
      }
    },
    [token, finish],
  );

  async function savePassword(e: FormEvent) {
    e.preventDefault();
    if (password.length < MIN_PASSWORD) {
      setFieldError(`Kata sandi minimal ${MIN_PASSWORD} karakter`);
      return;
    }
    if (password !== confirm) {
      setFieldError("Kedua kata sandi belum sama");
      return;
    }
    setFieldError(null);
    setError(null);
    setBusy(true);
    try {
      const res = await httpClient.post(API.auth.linkPassword(token), { password });
      finish(res.data.data as AuthSession);
    } catch (err) {
      const code = errorCode(err);
      if (code === "link_used" || code === "link_revoked" || code === "link_expired") {
        setInfo(null);
        setLoadError(getApiErrorMessage(err, "Link ini sudah tidak berlaku."));
        return;
      }
      if (axios.isAxiosError(err) && err.response?.status === 422) {
        const fields = (err.response.data as { errors?: Record<string, string[]> }).errors ?? {};
        setFieldError(fields.password?.[0] ?? getApiErrorMessage(err, "Kata sandi belum memenuhi syarat"));
        return;
      }
      setError(getApiErrorMessage(err, "Kata sandi belum tersimpan. Silakan coba lagi."));
    } finally {
      setBusy(false);
    }
  }

  const isStaff = info?.purpose === "staff_invite";
  const isReset = info?.purpose === "client_reset";

  return (
    <AuthSplitLayout
      aside={
        <>
          <h1 className="text-[36px] font-bold leading-[1.15] text-white">
            {isStaff ? "Selamat bergabung." : "Every Step to Your Perfect Day."}
          </h1>
          <div className="mt-4 h-px w-12 bg-white/40" />
          <p className="mt-4 text-[15px] leading-relaxed text-white/70">
            {isStaff
              ? "Satu tempat untuk timeline, vendor, pembayaran, dan bukti kerja setiap pernikahan yang ditangani tim Anda."
              : "Pantau persiapan pernikahan Anda bersama WO: timeline, vendor, pembayaran, dan dokumen, kapan saja."}
          </p>
        </>
      }
    >
      <div className="rounded-xl border border-border bg-surface p-6 shadow-sm sm:p-10">
        {!info && !loadError && (
          <div className="flex items-center justify-center gap-2 py-10 text-sm text-text-secondary">
            <Loader2 className="h-4 w-4 animate-spin" /> Memeriksa link...
          </div>
        )}

        {loadError && (
          <>
            <h2 className="text-2xl font-bold text-text-primary">Link tidak dapat dipakai</h2>
            <p role="alert" className="mt-3 text-[14px] leading-relaxed text-text-secondary">
              {loadError}
            </p>
            <Link
              to={ROUTE_PATHS.login}
              className="mt-8 inline-flex h-9 w-full items-center justify-center rounded-md bg-slate-800 px-4 text-sm font-medium text-white hover:bg-slate-700"
            >
              Ke halaman masuk
            </Link>
          </>
        )}

        {info && (
          <>
            <p className="text-[13px] font-medium text-text-secondary">{info.tenantName || "ElProof"}</p>
            <h2 className="mt-1 text-2xl font-bold text-text-primary">
              {isStaff ? `Halo, ${info.displayName}` : isReset ? "Atur ulang kata sandi" : `Halo, ${info.displayName}`}
            </h2>
            <p className="mt-2 text-[13.5px] leading-relaxed text-text-secondary">
              {isStaff
                ? `Anda diundang bergabung di ${info.tenantName || "WO ini"}. Masuk dengan akun Google ${info.email} untuk menerima undangan.`
                : isReset
                  ? "Buat kata sandi baru untuk portal client Anda. Kata sandi lama berhenti berlaku setelah ini."
                  : "Buat kata sandi untuk masuk ke portal client dan memantau persiapan pernikahan Anda."}
            </p>
            <p className="mt-1 text-[12px] text-text-secondary">Link berlaku sampai {formatLinkExpiry(info.expiresAt)}.</p>

            <div className="mt-8 flex flex-col gap-5">
              {error && (
                <p role="alert" className="rounded-md border border-danger/30 bg-danger-soft px-3.5 py-2.5 text-[13px] font-medium text-danger-text">
                  {error}
                </p>
              )}

              {isStaff ? (
                <GoogleSignInButton text="continue_with" context="signin" onCredential={acceptWithGoogle} busy={busy} />
              ) : (
                <form onSubmit={savePassword} className="flex flex-col gap-5" noValidate>
                  {/* Lets password managers save the login under the right name. */}
                  <input type="text" name="username" autoComplete="username" value={info.username ?? ""} readOnly hidden />
                  <Field label="Nama pengguna">
                    <Input value={info.username ?? ""} readOnly className="bg-surface-muted" />
                  </Field>
                  <Field label="Kata sandi baru" htmlFor="new-password" required error={fieldError ?? undefined} hint={`Minimal ${MIN_PASSWORD} karakter`}>
                    <div className="relative">
                      <Input
                        id="new-password"
                        type={showPassword ? "text" : "password"}
                        autoComplete="new-password"
                        value={password}
                        onChange={(e) => {
                          setPassword(e.target.value);
                          setFieldError(null);
                        }}
                        className="pr-10"
                        autoFocus
                      />
                      <button
                        type="button"
                        onClick={() => setShowPassword((v) => !v)}
                        aria-label={showPassword ? "Sembunyikan kata sandi" : "Tampilkan kata sandi"}
                        className="absolute right-3 top-1/2 -translate-y-1/2 text-text-secondary hover:text-text-primary"
                      >
                        {showPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                      </button>
                    </div>
                  </Field>
                  <Field label="Ulangi kata sandi" htmlFor="confirm-password" required>
                    <Input
                      id="confirm-password"
                      type={showPassword ? "text" : "password"}
                      autoComplete="new-password"
                      value={confirm}
                      onChange={(e) => {
                        setConfirm(e.target.value);
                        setFieldError(null);
                      }}
                    />
                  </Field>
                  <Button type="submit" variant="neutral" className="w-full justify-center" disabled={busy}>
                    {busy ? "Menyimpan..." : "Simpan kata sandi dan masuk"}
                  </Button>
                </form>
              )}
            </div>
          </>
        )}
      </div>
    </AuthSplitLayout>
  );
}
