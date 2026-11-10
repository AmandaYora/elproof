import { useCallback, useEffect, useRef, useState } from "react";
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
import { invitationPasswordSchema, MIN_PASSWORD } from "@/modules/auth/schemas/invitation.schema";
import { useZodForm } from "@/shared/forms/useZodForm";
import { FormErrorSummary } from "@/shared/forms/FormErrorSummary";

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

const FIELD_LABELS: Record<string, string> = { password: "Kata sandi baru", confirm: "Ulangi kata sandi" };
const FIELDS = (key: string) => key === "password";

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
  const form = useZodForm({ schema: invitationPasswordSchema, initial: { password: "", confirm: "" } });
  const { values, errors } = form;
  const formRef = useRef<HTMLFormElement>(null);
  const setFormError = form.setFormError;
  const [googleBusy, setGoogleBusy] = useState(false);
  const busy = googleBusy || form.submitting;
  const [showPassword, setShowPassword] = useState(false);

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
      setFormError(null);
      setGoogleBusy(true);
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
        setFormError(getApiErrorMessage(err, "Undangan belum berhasil diterima. Silakan coba lagi."));
      } finally {
        setGoogleBusy(false);
      }
    },
    [token, finish, setFormError],
  );

  const savePassword = form.handleSubmit(
    async ({ password }) => {
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
        throw err; // a 422 is shown on the password field, anything else above the form
      }
    },
    { root: () => formRef.current, fallback: "Kata sandi belum tersimpan. Silakan coba lagi.", known: FIELDS },
  );

  const isStaff = info?.purpose === "staff_invite";
  const isReset = info?.purpose === "client_reset";

  return (
    <AuthSplitLayout
      aside={
        <>
          <h1 className="text-[36px] font-bold leading-[1.15] text-white">
            {isStaff ? "Selamat bergabung." : "Setiap langkah menuju hari istimewa Anda."}
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
              <FormErrorSummary errors={form.errorList} formError={form.formError} labels={FIELD_LABELS} root={() => formRef.current} />

              {isStaff ? (
                <GoogleSignInButton text="continue_with" context="signin" onCredential={acceptWithGoogle} busy={busy} />
              ) : (
                <form ref={formRef} onSubmit={(e) => void savePassword(e)} className="flex flex-col gap-5" noValidate>
                  {/* Lets password managers save the login under the right name. */}
                  <input type="text" name="username" autoComplete="username" value={info.username ?? ""} readOnly hidden />
                  <Field label="Nama pengguna">
                    <Input value={info.username ?? ""} readOnly className="bg-surface-muted" />
                  </Field>
                  <Field label="Kata sandi baru" htmlFor="new-password" required name="password" error={errors.password} hint={`Minimal ${MIN_PASSWORD} karakter`}>
                    <div className="relative">
                      <Input
                        id="new-password"
                        type={showPassword ? "text" : "password"}
                        autoComplete="new-password"
                        value={values.password}
                        onChange={(e) => form.set("password", e.target.value)}
                        onBlur={() => form.blur("password")}
                        aria-invalid={errors.password ? true : undefined}
                        className="pr-10"
                        autoFocus
                      />
                      <button
                        type="button"
                        onClick={() => setShowPassword((v) => !v)}
                        aria-label={showPassword ? "Sembunyikan kata sandi" : "Tampilkan kata sandi"}
                        className="absolute right-1 top-1/2 flex size-8 -translate-y-1/2 items-center justify-center rounded-md text-text-secondary hover:text-text-primary focus-visible:outline-2 focus-visible:outline-offset-1 focus-visible:outline-foreground"
                      >
                        {showPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                      </button>
                    </div>
                  </Field>
                  <Field label="Ulangi kata sandi" htmlFor="confirm-password" required name="confirm" error={errors.confirm}>
                    <Input
                      id="confirm-password"
                      type={showPassword ? "text" : "password"}
                      autoComplete="new-password"
                      value={values.confirm}
                      onChange={(e) => form.set("confirm", e.target.value)}
                      onBlur={() => form.blur("confirm")}
                    />
                  </Field>
                  <Button type="submit" variant="neutral" className="w-full justify-center" disabled={form.submitting}>
                    {form.submitting ? "Menyimpan..." : "Simpan kata sandi dan masuk"}
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
