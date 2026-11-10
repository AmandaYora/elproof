import { useCallback, useEffect, useRef, useState } from "react";
import { Link, useLocation, useNavigate } from "react-router-dom";
import axios from "axios";
import { Eye, EyeOff, Lock } from "lucide-react";
import { Button } from "@/shared/components/ui/Button";
import { Input, Field } from "@/shared/components/ui/Input";
import { loginSchema, googleLinkSchema, mfaCodeSchema } from "@/modules/auth/schemas/login.schema";
import { useZodForm } from "@/shared/forms/useZodForm";
import { FormErrorSummary } from "@/shared/forms/FormErrorSummary";
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

/** What the person is told when a sign-in call fails: the server's own words, or a connection hint when it never answered. */
function loginErrorMessage(err: unknown, fallback: string) {
  if (axios.isAxiosError(err)) {
    if (!err.response) return "Tidak dapat terhubung ke server. Periksa koneksi internet Anda lalu coba lagi.";
    const message = (err.response.data as { message?: string })?.message;
    if (message) return message;
  }
  return fallback;
}

const LOGIN_FIELDS = (key: string) => key === "username" || key === "password";
const LINK_FIELDS = (key: string) => key === "password";
const MFA_FIELDS = (key: string) => key === "code";

export default function LoginPage() {
  usePageMeta({ title: "Masuk", description: "Masuk ke ElProof sebagai tim WO, klien, atau admin platform.", noindex: true });
  const navigate = useNavigate();
  // Three small forms, one per step; the page shows the message of the one on screen.
  const loginForm = useZodForm({ schema: loginSchema, initial: { username: "", password: "" } });
  const linkForm = useZodForm({ schema: googleLinkSchema, initial: { password: "" } });
  const mfaForm = useZodForm({ schema: mfaCodeSchema, initial: { code: "" } });
  const loginFormRef = useRef<HTMLFormElement>(null);
  const linkFormRef = useRef<HTMLFormElement>(null);
  const mfaFormRef = useRef<HTMLFormElement>(null);
  const setLoginError = loginForm.setFormError;
  const resetLink = linkForm.reset;
  const resetMfa = mfaForm.reset;
  const [showPassword, setShowPassword] = useState(false);
  // Second login step for Platform Console admins with two-factor on.
  const [mfaToken, setMfaToken] = useState<string | null>(null);
  // Why the app ended the previous session (e.g. tenant frozen), once.
  const [notice] = useState(() => peekLoginNotice());
  // Google account whose email matches an existing staff login: confirm
  // that login's password once to link them (or start a new workspace).
  const location = useLocation();
  const [googleLink, setGoogleLink] = useState<GoogleSignupState | null>(
    () => (location.state as { googleLink?: GoogleSignupState } | null)?.googleLink ?? null,
  );
  useEffect(() => {
    clearLoginNotice();
  }, []);

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
      setLoginError(null);
      setGoogleBusy(true);
      try {
        const res = await httpClient.post(API.auth.google, { credential });
        const data = res.data.data as AuthSession | GoogleAuthOutcome;
        if ("linkRequired" in data) {
          setGoogleLink({ credential, email: data.email, name: data.name });
          resetLink({ password: "" });
          return;
        }
        if ("signupRequired" in data) {
          const state: GoogleSignupState = { credential, email: data.email, name: data.name, invited: data.invited };
          navigate(ROUTE_PATHS.signupWorkspace, { state });
          return;
        }
        finishLogin(data);
      } catch (err) {
        setLoginError(loginErrorMessage(err, "Masuk dengan Google gagal. Silakan coba lagi."));
      } finally {
        setGoogleBusy(false);
      }
    },
    [finishLogin, navigate, setLoginError, resetLink],
  );

  // Wrong password, locked account, too many attempts: the server's own message
  // (or a connection hint) lands in the form's message, or on the field when the server pins it there.
  const submitLogin = loginForm.handleSubmit(
    async ({ username, password }) => {
      try {
        const res = await httpClient.post(API.auth.login, { username, password });
        const data = res.data.data as AuthSession | { mfaRequired: true; mfaToken: string };
        if ("mfaRequired" in data && data.mfaRequired) {
          setMfaToken(data.mfaToken);
          resetMfa({ code: "" });
          return;
        }
        finishLogin(data as AuthSession);
      } catch (err) {
        loginForm.applyServerError(err, loginErrorMessage(err, "Username atau kata sandi salah."), LOGIN_FIELDS);
      }
    },
    { root: () => loginFormRef.current },
  );

  const submitLink = linkForm.handleSubmit(
    async ({ password }) => {
      if (!googleLink) return;
      try {
        const res = await httpClient.post(API.auth.googleLink, { credential: googleLink.credential, password });
        finishLogin(res.data.data as AuthSession);
      } catch (err) {
        const status = axios.isAxiosError(err) ? err.response?.status : undefined;
        if (status === 401 && /google/i.test(loginErrorMessage(err, ""))) {
          // The Google credential expired: start over with the button, and say why there.
          setGoogleLink(null);
          setLoginError(loginErrorMessage(err, "Kata sandi salah."));
          return;
        }
        linkForm.applyServerError(err, loginErrorMessage(err, "Kata sandi salah."), LINK_FIELDS);
      }
    },
    { root: () => linkFormRef.current },
  );

  const submitMfa = mfaForm.handleSubmit(
    async ({ code }) => {
      if (!mfaToken) return;
      try {
        const res = await httpClient.post(API.auth.mfaVerify, { mfaToken, code });
        finishLogin(res.data.data as AuthSession);
      } catch (err) {
        const status = axios.isAxiosError(err) ? err.response?.status : undefined;
        const message = loginErrorMessage(err, "Kode verifikasi salah atau sudah kedaluwarsa.");
        // An expired challenge cannot be retried: go back to the password step, and say why there.
        if (status === 401 && /sesi verifikasi/i.test(message)) {
          setMfaToken(null);
          setLoginError(message);
          return;
        }
        mfaForm.applyServerError(err, message, MFA_FIELDS);
      }
    },
    { root: () => mfaFormRef.current },
  );

  // The message of the step on screen.
  const activeForm = googleLink && !mfaToken ? linkForm : mfaToken ? mfaForm : loginForm;
  const activeRoot = googleLink && !mfaToken ? linkFormRef : mfaToken ? mfaFormRef : loginFormRef;

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
            Pantau seluruh perjalanan menuju hari pernikahan Anda dalam satu portal. Mulai dari jadwal, desain,
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

          <h2 className="text-2xl font-bold text-text-primary">Selamat datang</h2>
          <p className="mt-1.5 text-[13.5px] text-text-secondary">
            {mfaToken
              ? "Verifikasi dua langkah untuk admin platform."
              : "Tim WO masuk dengan Google. Klien dan admin platform masuk dengan nama pengguna."}
          </p>

          <div className="mt-8 flex flex-col gap-5">
            {notice && !activeForm.formError && (
              <p role="status" className="rounded-md border border-warning/30 bg-warning-soft px-3.5 py-2.5 text-[13px] font-medium text-warning-text">
                {notice}
              </p>
            )}
            <FormErrorSummary errors={activeForm.errorList} formError={activeForm.formError} root={() => activeRoot.current} />

            {googleLink && !mfaToken ? (
              <>
                <form ref={linkFormRef} noValidate onSubmit={(e) => void submitLink(e)} className="flex flex-col gap-5">
                  <p className="rounded-md border border-border bg-surface-muted px-3.5 py-3 text-[13px] leading-relaxed text-text-secondary">
                    Email <span className="font-semibold text-text-primary">{googleLink.email}</span> sudah terdaftar sebagai akun anggota tim
                    di ElProof. Masukkan kata sandi ElProof akun tersebut sekali untuk menghubungkannya dengan Google. Setelah itu
                    cukup masuk dengan Google.
                  </p>
                  <Field label="Kata sandi ElProof" required name="password" error={linkForm.errors.password}>
                    <Input
                      type="password"
                      value={linkForm.values.password}
                      onChange={(e) => {
                        linkForm.set("password", e.target.value);
                        linkForm.setFormError(null);
                      }}
                      autoComplete="current-password"
                      autoFocus
                    />
                  </Field>
                  <Button type="submit" variant="neutral" className="mt-2 w-full justify-center" disabled={linkForm.submitting}>
                    {linkForm.submitting ? "Menghubungkan..." : "Hubungkan dan masuk"}
                  </Button>
                </form>
                <div className="flex flex-wrap justify-between gap-2 text-[13px]">
                  <button
                    type="button"
                    onClick={() => {
                      setGoogleLink(null);
                      linkForm.setFormError(null);
                      loginForm.setFormError(null);
                    }}
                    disabled={linkForm.submitting}
                    className="font-medium text-text-secondary hover:text-text-primary"
                  >
                    Batal
                  </button>
                  <button
                    type="button"
                    onClick={() => navigate(ROUTE_PATHS.signupWorkspace, { state: googleLink })}
                    disabled={linkForm.submitting}
                    className="font-medium text-text-secondary hover:text-text-primary"
                  >
                    Bukan akun saya, buat workspace baru
                  </button>
                </div>
              </>
            ) : mfaToken ? (
              <>
                <form ref={mfaFormRef} noValidate onSubmit={(e) => void submitMfa(e)} className="flex flex-col gap-5">
                  <Field label="Kode verifikasi" required name="code" error={mfaForm.errors.code} hint="Buka aplikasi authenticator Anda dan masukkan 6 digit kode untuk ElProof.">
                    <Input
                      value={mfaForm.values.code}
                      onChange={(e) => {
                        mfaForm.set("code", e.target.value.replace(/\D/g, "").slice(0, 6));
                        mfaForm.setFormError(null);
                      }}
                      inputMode="numeric"
                      autoComplete="one-time-code"
                      autoFocus
                      placeholder="123456"
                      className="font-mono tracking-[0.3em]"
                    />
                  </Field>
                  <Button type="submit" variant="neutral" className="mt-2 w-full justify-center" disabled={mfaForm.submitting}>
                    {mfaForm.submitting ? "Memeriksa kode..." : "Verifikasi kode"}
                  </Button>
                </form>
                <button
                  type="button"
                  onClick={() => {
                    setMfaToken(null);
                    mfaForm.setFormError(null);
                    loginForm.setFormError(null);
                  }}
                  disabled={mfaForm.submitting}
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
                <form ref={loginFormRef} className="flex flex-col gap-5" noValidate onSubmit={(e) => void submitLogin(e)}>
                <Field label="Nama pengguna atau email" htmlFor="login-username" required name="username" error={loginForm.errors.username}>
                  <Input
                    id="login-username"
                    name="username"
                    autoComplete="username"
                    autoCapitalize="none"
                    spellCheck={false}
                    value={loginForm.values.username}
                    onChange={(e) => {
                      loginForm.set("username", e.target.value);
                      loginForm.setFormError(null);
                    }}
                    onBlur={() => loginForm.blur("username")}
                    placeholder="Masukkan nama pengguna atau email Anda"
                  />
                </Field>

                <Field label="Kata sandi" htmlFor="login-password" required name="password" error={loginForm.errors.password}>
                  <div className="relative">
                    <Input
                      id="login-password"
                      name="password"
                      autoComplete="current-password"
                      type={showPassword ? "text" : "password"}
                      value={loginForm.values.password}
                      onChange={(e) => {
                        loginForm.set("password", e.target.value);
                        loginForm.setFormError(null);
                      }}
                      placeholder="Masukkan kata sandi Anda"
                      aria-invalid={loginForm.errors.password ? true : undefined}
                      className="pr-10"
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

                <Button
                  type="submit"
                  variant="neutral"
                  className="mt-2 w-full justify-center"
                  disabled={loginForm.submitting}
                >
                  {loginForm.submitting ? "Memproses..." : "Masuk"}
                </Button>
                <p className="text-[12.5px] leading-relaxed text-text-secondary">
                  <span className="font-medium text-text-primary">Lupa kata sandi?</span> Klien: minta link atur ulang ke WO Anda. Tim
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
