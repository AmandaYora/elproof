import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import axios from "axios";
import { Eye, EyeOff, Lock } from "lucide-react";
import { Button } from "@/shared/components/ui/Button";
import { Input, Field } from "@/shared/components/ui/Input";
import { loginSchema, type LoginFormValues } from "@/modules/auth/schemas/login.schema";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import { useAuthStore, type AuthSession } from "@/shared/stores/useAuthStore";
import { httpClient } from "@/shared/services/http-client";
import { API } from "@/shared/services/api-endpoints";
import { applyBrandColorPreset, resetBrandColorPreset, isBrandColorPresetKey } from "@/theme/brandPresets";
import { applyTabIdentity, resetTabIdentity } from "@/theme/tabIdentity";

interface DomainBranding {
  businessName: string;
  logoUrl: string | null;
}

// LoginPage's slate/neutral look is deliberately not tied to --brand-navy-*
// (ADR-0012 — no tenant is knowable pre-auth on the platform's own domain).
// Once ADR-0015 resolves a tenant from the request's Host header, that
// constraint no longer applies for *this specific request* — swapping these
// two className sets for the brand-tied ones below is the only visual
// difference a matched custom domain makes; the copy, layout, and structure
// stay identical either way.
// Every class below is written out in full (never built by string
// concatenation) since Tailwind generates CSS by statically scanning source
// text for whole utility tokens — a class assembled at runtime from pieces
// (e.g. `${"bg-navy-800"}/40`) never gets its CSS emitted.
const PANEL_CLASSES = {
  neutral: "bg-gradient-to-br from-slate-900 to-slate-800",
  branded: "bg-gradient-to-br from-navy-950 to-navy-900",
};
const BLUR_1_CLASSES = { neutral: "bg-slate-700/40", branded: "bg-navy-800/40" };
const BLUR_2_CLASSES = { neutral: "bg-slate-700/30", branded: "bg-navy-800/30" };
const ICON_BOX_CLASSES = { neutral: "bg-slate-800", branded: "bg-navy-900" };

// Fetches branding for the current request's Host header (ADR-0015) — only
// resolves to something when this page is loaded from a tenant's own custom
// domain; a 404 (the platform's own domain, localhost, anything unconfigured)
// is swallowed and the page keeps its default neutral look, same convention
// as useTenantBrandingStore.hydrate().
function useDomainBranding() {
  const [branding, setBranding] = useState<DomainBranding | null>(null);

  useEffect(() => {
    let cancelled = false;
    let logoObjectUrl: string | null = null;

    async function load() {
      try {
        const res = await httpClient.get(API.public.branding);
        const raw = res.data.data as { businessName: string; brandColorPreset: string; hasLogo: boolean };
        if (cancelled) return;

        let logoUrl: string | null = null;
        if (raw.hasLogo) {
          const fileRes = await httpClient.get(API.public.logo, { responseType: "blob" });
          if (cancelled) return;
          logoObjectUrl = URL.createObjectURL(fileRes.data as Blob);
          logoUrl = logoObjectUrl;
        }

        applyBrandColorPreset(isBrandColorPresetKey(raw.brandColorPreset) ? raw.brandColorPreset : "navy");
        applyTabIdentity(raw.businessName, logoUrl);
        setBranding({ businessName: raw.businessName, logoUrl });
      } catch {
        // No custom domain match, or a transient failure — keep the default look.
      }
    }
    void load();

    return () => {
      cancelled = true;
      resetBrandColorPreset();
      resetTabIdentity();
      if (logoObjectUrl) URL.revokeObjectURL(logoObjectUrl);
    };
  }, []);

  return branding;
}

export default function LoginPage() {
  const navigate = useNavigate();
  const branding = useDomainBranding();
  const [values, setValues] = useState<LoginFormValues>({ username: "", password: "" });
  const [errors, setErrors] = useState<Partial<Record<keyof LoginFormValues, string>>>({});
  const [authError, setAuthError] = useState<string | null>(null);
  const [showPassword, setShowPassword] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);

  function set<K extends keyof LoginFormValues>(key: K, value: LoginFormValues[K]) {
    setValues((prev) => ({ ...prev, [key]: value }));
    setAuthError(null);
  }

  async function attemptLogin(username: string, password: string) {
    setAuthError(null);
    setIsSubmitting(true);
    try {
      const res = await httpClient.post(API.auth.login, { username, password });
      const session = res.data.data as AuthSession;
      useAuthStore.getState().login(session);

      if (session.principalType === "staff") {
        navigate(ROUTE_PATHS.dashboard);
      } else if (session.principalType === "client") {
        navigate(ROUTE_PATHS.portal());
      } else {
        navigate(ROUTE_PATHS.platformDashboard);
      }
    } catch (err) {
      const message = axios.isAxiosError(err) ? (err.response?.data as { message?: string })?.message : undefined;
      setAuthError(message ?? "Username atau kata sandi salah.");
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
        className={`relative hidden overflow-hidden lg:flex lg:w-1/2 lg:flex-col lg:justify-center lg:px-16 ${branding ? PANEL_CLASSES.branded : PANEL_CLASSES.neutral}`}
      >
        <div
          className={`pointer-events-none absolute -right-24 -top-24 h-96 w-96 rounded-full blur-3xl ${branding ? BLUR_1_CLASSES.branded : BLUR_1_CLASSES.neutral}`}
        />
        <div
          className={`pointer-events-none absolute -bottom-32 -left-16 h-80 w-80 rounded-full blur-3xl ${branding ? BLUR_2_CLASSES.branded : BLUR_2_CLASSES.neutral}`}
        />
        <div className="relative">
          <h1 className="text-[40px] font-bold leading-[1.15] text-white">
            Transparansi Persiapan Pernikahan,
            <br />
            Kini Lebih Terjaga.
          </h1>
          <p className="mt-5 max-w-md text-[15px] leading-relaxed text-white/70">
            Satu pintu masuk untuk semua pihak — tim WO mendokumentasikan setiap progress, vendor, dan pembayaran,
            sementara pasangan client dapat memantau langsung persiapan hari bahagia mereka.
          </p>
        </div>
      </div>

      <div className="flex w-full items-center justify-center bg-background px-6 py-12 lg:w-1/2">
        <div className="w-full max-w-md rounded-xl border border-border bg-surface p-10 shadow-sm">
          <div
            className={`mb-6 flex h-12 w-12 items-center justify-center overflow-hidden rounded-xl text-white ${branding ? ICON_BOX_CLASSES.branded : ICON_BOX_CLASSES.neutral}`}
          >
            {branding?.logoUrl ? (
              <img src={branding.logoUrl} alt={branding.businessName} className="h-full w-full object-contain" />
            ) : (
              <Lock className="h-5 w-5" />
            )}
          </div>

          <h2 className="text-2xl font-bold text-text-primary">
            {branding ? `Selamat Datang di ${branding.businessName}` : "Selamat Datang"}
          </h2>
          <p className="mt-1.5 text-[13.5px] text-text-secondary">
            Masuk sebagai tim WO Console, client, atau admin platform.
          </p>

          <div className="mt-8 flex flex-col gap-5">
            {authError && (
              <p className="rounded-md border border-danger/30 bg-danger-soft px-3.5 py-2.5 text-[13px] font-medium text-danger">
                {authError}
              </p>
            )}

            <Field label="Nama Pengguna atau Email" required hint={errors.username}>
              <Input
                value={values.username}
                onChange={(e) => set("username", e.target.value)}
                placeholder="Masukkan nama pengguna atau email Anda"
                onKeyDown={(e) => e.key === "Enter" && handleSubmit()}
              />
            </Field>

            <Field label="Kata Sandi" required hint={errors.password}>
              <div className="relative">
                <Input
                  type={showPassword ? "text" : "password"}
                  value={values.password}
                  onChange={(e) => set("password", e.target.value)}
                  placeholder="Masukkan kata sandi Anda"
                  className="pr-10"
                  onKeyDown={(e) => e.key === "Enter" && handleSubmit()}
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
              variant={branding ? "primary" : "neutral"}
              className="mt-2 w-full justify-center"
              onClick={handleSubmit}
              disabled={isSubmitting}
            >
              {isSubmitting ? "Memproses..." : "Masuk"}
            </Button>
          </div>
        </div>
      </div>
    </div>
  );
}
