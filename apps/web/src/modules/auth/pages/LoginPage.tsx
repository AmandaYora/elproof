import { useCallback, useEffect, useState } from "react";
import { Link, useLocation, useNavigate } from "react-router-dom";
import axios from "axios";
import { Eye, EyeOff, Lock } from "lucide-react";
import { Button } from "@/shared/components/ui/Button";
import { Input, Field } from "@/shared/components/ui/Input";
import { loginSchema, type LoginFormValues } from "@/modules/auth/schemas/login.schema";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import { clearLoginNotice, peekLoginNotice } from "@/shared/lib/session-notice";
import { useAuthStore, type AuthSession } from "@/shared/stores/useAuthStore";
import { httpClient } from "@/shared/services/http-client";
import { API } from "@/shared/services/api-endpoints";
import { LoginPhotoSlider } from "@/modules/auth/components/LoginPhotoSlider";
import { GoogleSignInButton } from "@/shared/components/auth/GoogleSignInButton";
import { landingPathFor } from "@/shared/lib/auth-actions";
import type { GoogleAuthOutcome, GoogleSignupState } from "@/modules/auth/types";
import { usePageMeta } from "@/shared/hooks/usePageMeta";

// The login page's slate look is deliberately not tied to --brand-navy-*
// (ADR-0012): no tenant is knowable before anyone has signed in. Every class is
// written out in full, never built by string concatenation, since Tailwind
// finds utilities by scanning the source for whole tokens.
const PANEL_CLASS = "bg-gradient-to-br from-slate-900 to-slate-800";
const ICON_BOX_CLASS = "bg-slate-800";
// The gradient scrim behind the headline over the photo slider.
const SCRIM_CLASS = "bg-gradient-to-t from-slate-950/95 via-slate-950/40 to-transparent";

export default function LoginPage() {
  usePageMeta({ title: "Masuk", description: "Masuk ke ElProof sebagai tim WO, klien, atau admin platform.", noindex: true });
  const navigate = useNavigate();
  const [values, setValues] = useState<LoginFormValues>({ username: "", password: "" });
  const [errors, setErrors] = useState<Partial<Record<keyof LoginFormValues, string>>>({});
  const [authError, setAuthError] = useState<string | null>(null);
  const [showPassword, setShowPassword] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);
  // Second login step for Platform Console admins with two-factor on.
  const [mfaToken, setMfaToken] = useState<string | null>(null);
  const [mfaCode, setMfaCode] = useState("");
  // Why the app ended the previous session (e.g. tenant frozen), once.
  const [notice] = useState(() => peekLoginNotice());
  // Google account whose email matches an existing staff login: confirm
  // that login's password once to link them (or start a new workspace).
  const location = useLocation();
  const [googleLink, setGoogleLink] = useState<GoogleSignupState | null>(
    () => (location.state as { googleLink?: GoogleSignupState } | null)?.googleLink ?? null,
  );
  const [linkPassword, setLinkPassword] = useState("");
  useEffect(() => {
    clearLoginNotice();
  }, []);

  function set<K extends keyof LoginFormValues>(key: K, value: LoginFormValues[K]) {
    setValues((prev) => ({ ...prev, [key]: value }));
    setAuthError(null);
  }

  const [googleBusy, setGoogleBusy] = useState(false);

  // Back to the page that sent the person here (an expired session, a
  // bookmarked deep link), when it belongs to their own console.
  const returnTo = (location.state as { from?: { pathname?: string; search?: string } } | null)?.from;
  const finishLogin = useCallback(
    (session: AuthSession) => {
      useAuthStore.getState().login(session);
      const path = returnTo?.pathname ?? "";
      const fits =
        session.principalType === "client"
          ? path.startsWith("/portal")
          : session.principalType === "platform_admin"
            ? session.mfa === true && path.startsWith("/platform/") && path !== ROUTE_PATHS.platformSecurity
            : path !== "" && !path.startsWith("/portal") && !path.startsWith("/platform");
      navigate(fits ? path + (returnTo?.search ?? "") : landingPathFor(session), { replace: true });
    },
    [navigate, returnTo],
  );

  // Sign in with Google (WO Owner and staff). A Google account with no
  // ElProof login yet continues to the workspace form.
  const handleGoogleCredential = useCallback(
    async (credential: string) => {
      setAuthError(null);
      setGoogleBusy(true);
      try {
        const res = await httpClient.post(API.auth.google, { credential });
        const data = res.data.data as AuthSession | GoogleAuthOutcome;
        if ("linkRequired" in data) {
          setGoogleLink({ credential, email: data.email, name: data.name });
          setLinkPassword("");
          return;
        }
        if ("signupRequired" in data) {
          const state: GoogleSignupState = { credential, email: data.email, name: data.name, invited: data.invited };
          navigate(ROUTE_PATHS.signupWorkspace, { state });
          return;
        }
        finishLogin(data);
      } catch (err) {
        setAuthError(loginErrorMessage(err, "Masuk dengan Google gagal. Silakan coba lagi."));
      } finally {
        setGoogleBusy(false);
      }
    },
    [finishLogin, navigate],
  );

  function loginErrorMessage(err: unknown, fallback: string) {
    if (axios.isAxiosError(err)) {
      if (!err.response) return "Tidak dapat terhubung ke server. Periksa koneksi internet Anda lalu coba lagi.";
      const message = (err.response.data as { message?: string })?.message;
      if (message) return message;
    }
    return fallback;
  }

  async function attemptLogin(username: string, password: string) {
    setAuthError(null);
    setIsSubmitting(true);
    try {
      const res = await httpClient.post(API.auth.login, { username, password });
      const data = res.data.data as AuthSession | { mfaRequired: true; mfaToken: string };
      if ("mfaRequired" in data && data.mfaRequired) {
        setMfaToken(data.mfaToken);
        setMfaCode("");
        return;
      }
      finishLogin(data as AuthSession);
    } catch (err) {
      setAuthError(loginErrorMessage(err, "Username atau kata sandi salah."));
    } finally {
      setIsSubmitting(false);
    }
  }

  async function linkGoogle() {
    if (!googleLink) return;
    if (!linkPassword) {
      setAuthError("Masukkan kata sandi ElProof akun ini.");
      return;
    }
    setAuthError(null);
    setIsSubmitting(true);
    try {
      const res = await httpClient.post(API.auth.googleLink, { credential: googleLink.credential, password: linkPassword });
      finishLogin(res.data.data as AuthSession);
    } catch (err) {
      const status = axios.isAxiosError(err) ? err.response?.status : undefined;
      if (status === 401 && /google/i.test(loginErrorMessage(err, ""))) {
        // The Google credential expired: start over with the button.
        setGoogleLink(null);
      }
      setAuthError(loginErrorMessage(err, "Kata sandi salah."));
    } finally {
      setIsSubmitting(false);
    }
  }

  async function verifyMfa() {
    if (!mfaToken) return;
    if (mfaCode.length !== 6) {
      setAuthError("Masukkan 6 digit kode dari aplikasi authenticator.");
      return;
    }
    setAuthError(null);
    setIsSubmitting(true);
    try {
      const res = await httpClient.post(API.auth.mfaVerify, { mfaToken, code: mfaCode });
      finishLogin(res.data.data as AuthSession);
    } catch (err) {
      const status = axios.isAxiosError(err) ? err.response?.status : undefined;
      setAuthError(loginErrorMessage(err, "Kode verifikasi salah atau sudah kedaluwarsa."));
      // An expired challenge cannot be retried: go back to the password step.
      if (status === 401 && /sesi verifikasi/i.test(loginErrorMessage(err, ""))) setMfaToken(null);
    } finally {
      setIsSubmitting(false);
    }
  }

  function handleSubmit() {
    const result = loginSchema.safeParse(values);
    if (!result.success) {
      const fieldErrors: Partial<Record<keyof LoginFormValues, string>> = {};
      for (const issue of result.error.issues) {
        fieldErrors[issue.path[0] as keyof LoginFormValues] = issue.message;
      }
      setErrors(fieldErrors);
      return;
    }
    setErrors({});
    void attemptLogin(values.username, values.password);
  }

  return (
    <div className="flex min-h-screen">
      <div
        className={`relative hidden overflow-hidden lg:flex lg:w-1/2 lg:flex-col lg:justify-end lg:px-16 lg:pb-16 ${PANEL_CLASS}`}
      >
        <LoginPhotoSlider />
        <div className={`pointer-events-none absolute inset-0 ${SCRIM_CLASS}`} />
        <div className="relative z-10">
          <h1 className="text-[40px] font-bold leading-[1.15] text-white">Every Step to Your Perfect Day.</h1>
          <div className="mt-4 h-px w-12 bg-white/40" />
          <p className="mt-4 max-w-md text-[15px] leading-relaxed text-white/70">
            Pantau seluruh perjalanan menuju hari pernikahan Anda dalam satu portal. Mulai dari timeline, desain,
            pembayaran, hingga progres produksi, semuanya tersusun rapi, transparan, dan dapat diakses kapan saja.
          </p>
        </div>
      </div>

      <div className="flex w-full items-center justify-center bg-background px-6 py-12 lg:w-1/2">
        <div className="w-full max-w-md rounded-xl border border-border bg-surface p-10 shadow-sm">
          <div
            className={`mb-6 flex h-12 w-12 items-center justify-center overflow-hidden rounded-xl text-white ${ICON_BOX_CLASS}`}
          >
            <Lock className="h-5 w-5" />
          </div>

          <h2 className="text-2xl font-bold text-text-primary">Selamat Datang</h2>
          <p className="mt-1.5 text-[13.5px] text-text-secondary">
            {mfaToken
              ? "Verifikasi dua langkah untuk admin platform."
              : "Tim WO masuk dengan Google. Client dan admin platform masuk dengan nama pengguna."}
          </p>

          <div className="mt-8 flex flex-col gap-5">
            {notice && !authError && (
              <p role="status" className="rounded-md border border-warning/30 bg-warning-soft px-3.5 py-2.5 text-[13px] font-medium text-warning-text">
                {notice}
              </p>
            )}
            {authError && (
              <p className="rounded-md border border-danger/30 bg-danger-soft px-3.5 py-2.5 text-[13px] font-medium text-danger-text">
                {authError}
              </p>
            )}

            {googleLink && !mfaToken ? (
              <>
                <p className="rounded-md border border-border bg-surface-muted px-3.5 py-3 text-[13px] leading-relaxed text-text-secondary">
                  Email <span className="font-semibold text-text-primary">{googleLink.email}</span> sudah terdaftar sebagai akun staf
                  di ElProof. Masukkan kata sandi ElProof akun tersebut sekali untuk menghubungkannya dengan Google. Setelah itu
                  cukup masuk dengan Google.
                </p>
                <Field label="Kata sandi ElProof" required>
                  <Input
                    type="password"
                    value={linkPassword}
                    onChange={(e) => {
                      setLinkPassword(e.target.value);
                      setAuthError(null);
                    }}
                    autoComplete="current-password"
                    autoFocus
                    onKeyDown={(e) => e.key === "Enter" && void linkGoogle()}
                  />
                </Field>
                <Button variant="neutral" className="mt-2 w-full justify-center" onClick={() => void linkGoogle()} disabled={isSubmitting}>
                  {isSubmitting ? "Menghubungkan..." : "Hubungkan dan masuk"}
                </Button>
                <div className="flex flex-wrap justify-between gap-2 text-[13px]">
                  <button
                    type="button"
                    onClick={() => {
                      setGoogleLink(null);
                      setAuthError(null);
                    }}
                    className="font-medium text-text-secondary hover:text-text-primary"
                  >
                    Batal
                  </button>
                  <button
                    type="button"
                    onClick={() => navigate(ROUTE_PATHS.signupWorkspace, { state: googleLink })}
                    className="font-medium text-text-secondary hover:text-text-primary"
                  >
                    Bukan akun saya, buat workspace baru
                  </button>
                </div>
              </>
            ) : mfaToken ? (
              <>
                <Field label="Kode verifikasi" required hint="Buka aplikasi authenticator Anda dan masukkan 6 digit kode untuk ElProof.">
                  <Input
                    value={mfaCode}
                    onChange={(e) => {
                      setMfaCode(e.target.value.replace(/\D/g, "").slice(0, 6));
                      setAuthError(null);
                    }}
                    inputMode="numeric"
                    autoComplete="one-time-code"
                    autoFocus
                    placeholder="123456"
                    className="font-mono tracking-[0.3em]"
                    onKeyDown={(e) => e.key === "Enter" && void verifyMfa()}
                  />
                </Field>
                <Button variant="neutral" className="mt-2 w-full justify-center" onClick={() => void verifyMfa()} disabled={isSubmitting}>
                  {isSubmitting ? "Memeriksa kode..." : "Verifikasi"}
                </Button>
                <button
                  type="button"
                  onClick={() => {
                    setMfaToken(null);
                    setAuthError(null);
                  }}
                  className="text-[13px] font-medium text-text-secondary hover:text-text-primary"
                >
                  Kembali ke kata sandi
                </button>
              </>
            ) : (
              <>
                <GoogleSignInButton text="signin_with" context="signin" onCredential={handleGoogleCredential} busy={googleBusy} />
                <div className="flex items-center gap-3 text-[12px] text-text-secondary" aria-hidden>
                  <span className="h-px flex-1 bg-border" />
                  atau dengan nama pengguna
                  <span className="h-px flex-1 bg-border" />
                </div>
                <form
                  className="flex flex-col gap-5"
                  noValidate
                  onSubmit={(e) => {
                    e.preventDefault();
                    handleSubmit();
                  }}
                >
                <Field label="Nama Pengguna atau Email" htmlFor="login-username" required error={errors.username}>
                  <Input
                    id="login-username"
                    name="username"
                    autoComplete="username"
                    autoCapitalize="none"
                    spellCheck={false}
                    value={values.username}
                    onChange={(e) => set("username", e.target.value)}
                    placeholder="Masukkan nama pengguna atau email Anda"
                  />
                </Field>

                <Field label="Kata Sandi" htmlFor="login-password" required error={errors.password}>
                  <div className="relative">
                    <Input
                      id="login-password"
                      name="password"
                      autoComplete="current-password"
                      type={showPassword ? "text" : "password"}
                      value={values.password}
                      onChange={(e) => set("password", e.target.value)}
                      placeholder="Masukkan kata sandi Anda"
                      className="pr-10"
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

                <Button
                  type="submit"
                  variant="neutral"
                  className="mt-2 w-full justify-center"
                  disabled={isSubmitting}
                >
                  {isSubmitting ? "Memproses..." : "Masuk"}
                </Button>
                <p className="text-[12.5px] leading-relaxed text-text-secondary">
                  <span className="font-medium text-text-primary">Lupa kata sandi?</span> Client: minta link atur ulang ke WO Anda. Tim
                  WO: masuk dengan tombol Google di atas.
                </p>
                </form>
              </>
            )}
          </div>

          {!mfaToken && !googleLink && (
            <p className="mt-8 border-t border-border pt-5 text-center text-[13.5px] text-text-secondary">
              Belum memakai ElProof?{" "}
              <Link to={ROUTE_PATHS.signup} className="font-semibold text-text-primary hover:underline">
                Coba gratis 14 hari
              </Link>
            </p>
          )}
        </div>
      </div>
    </div>
  );
}
