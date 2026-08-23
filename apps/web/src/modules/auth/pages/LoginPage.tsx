import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import axios from "axios";
import { Eye, EyeOff, Loader2, Lock } from "lucide-react";
import { Button } from "@/shared/components/ui/Button";
import { Input, Field } from "@/shared/components/ui/Input";
import { loginSchema, type LoginFormValues } from "@/modules/auth/schemas/login.schema";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import { useAuthStore, type AuthSession } from "@/shared/stores/useAuthStore";
import { httpClient } from "@/shared/services/http-client";
import { API } from "@/shared/services/api-endpoints";
import { applyBrandColorPreset, resetBrandColorPreset, isBrandColorPresetKey } from "@/theme/brandPresets";
import { applyTabIdentity, resetTabIdentity } from "@/theme/tabIdentity";
import { LoginPhotoSlider } from "@/modules/auth/components/LoginPhotoSlider";

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
const ICON_BOX_CLASSES = { neutral: "bg-slate-800", branded: "bg-navy-900" };
// The gradient scrim behind the headline/subtext over the photo slider —
// same neutral/branded split as PANEL_CLASSES and for the same reason
// (ADR-0012): a matched custom domain's scrim tints with the tenant's own
// --brand-navy-* colors (e.g. JWS Wedding's bronze preset), the platform's
// own domain keeps the original neutral slate.
const SCRIM_CLASSES = {
  neutral: "bg-gradient-to-t from-slate-950/95 via-slate-950/40 to-transparent",
  branded: "bg-gradient-to-t from-navy-950/95 via-navy-900/40 to-transparent",
};

// The platform's own hostname(s) — a request arriving on any of these can
// never resolve to a tenant (ADR-0015's Host-header lookup 404s), so there's
// nothing worth waiting for and the page can paint its neutral look on the
// very first render, exactly as before this fix. Every other hostname is a
// *candidate* custom domain (it might 404 too, e.g. an unconfigured or
// decommissioned domain) and is worth a brief wait to avoid painting the
// wrong identity first. "elproof.elcodelabs.com" is the same platform-domain
// string already used in contact.ts/tenant_handler.go.
const PLATFORM_HOSTNAMES = new Set(["elproof.elcodelabs.com", "localhost", "127.0.0.1"]);

// A hung/slow `/public/branding` call must never hold a custom-domain login
// page hostage forever — past this, fall back to the neutral look exactly as
// a 404 would. Same-origin/same-VPS in production (ADR-0015), so this is a
// generous ceiling, not an expected path.
const BRANDING_LOAD_TIMEOUT_MS = 4000;

// Fetches branding for the current request's Host header (ADR-0015) — only
// resolves to something when this page is loaded from a tenant's own custom
// domain; a 404 (the platform's own domain, localhost, anything unconfigured)
// is swallowed and the page keeps its default neutral look, same convention
// as useTenantBrandingStore.hydrate().
//
// `loading` starts `true` for any non-platform hostname and only flips to
// `false` once the lookup settles (match, 404, or timeout) — LoginPage holds
// off painting either the neutral or the branded identity until then, so a
// matched custom domain never flashes the wrong one first (see the "ngedip"
// production report this fixed). The platform's own hostname skips the wait
// entirely since it can never match a tenant.
function useDomainBranding() {
  const [loading, setLoading] = useState(() => !PLATFORM_HOSTNAMES.has(window.location.hostname));
  const [branding, setBranding] = useState<DomainBranding | null>(null);

  useEffect(() => {
    // The platform's own hostname never matches a tenant (ADR-0015) — skip
    // the request entirely rather than firing it just to watch it 404.
    if (PLATFORM_HOSTNAMES.has(window.location.hostname)) return;
    let cancelled = false;
    let logoObjectUrl: string | null = null;
    const timeoutId = window.setTimeout(() => {
      if (!cancelled) setLoading(false);
    }, BRANDING_LOAD_TIMEOUT_MS);

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
      } finally {
        if (!cancelled) setLoading(false);
      }
    }
    void load();

    return () => {
      cancelled = true;
      window.clearTimeout(timeoutId);
      resetBrandColorPreset();
      resetTabIdentity();
      if (logoObjectUrl) URL.revokeObjectURL(logoObjectUrl);
    };
  }, []);

  return { loading, branding };
}

export default function LoginPage() {
  const navigate = useNavigate();
  const { loading: brandingLoading, branding } = useDomainBranding();
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
        // Neither Wedding Planner ("Staff") nor Sales has Dashboard access
        // (see PLAN.md revisi-timeline-vendor-role-sales) — land them on the
        // one page they can actually reach instead of a page RequireRole
        // would just redirect away from anyway.
        navigate(session.role === "Staff" || session.role === "Sales" ? ROUTE_PATHS.projects : ROUTE_PATHS.dashboard);
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

  if (brandingLoading) {
    // Neither the neutral nor the branded identity is safe to paint yet —
    // showing either one first, then replacing it, is exactly the flash this
    // guards against (see useDomainBranding). Only reached on a non-platform
    // hostname, and only for the brief `/public/branding` round trip.
    return (
      <div className="flex min-h-screen items-center justify-center bg-slate-950">
        <Loader2 className="h-6 w-6 animate-spin text-white/60" />
      </div>
    );
  }

  return (
    <div className="flex min-h-screen">
      <div
        className={`relative hidden overflow-hidden lg:flex lg:w-1/2 lg:flex-col lg:justify-end lg:px-16 lg:pb-16 ${branding ? PANEL_CLASSES.branded : PANEL_CLASSES.neutral}`}
      >
        <LoginPhotoSlider />
        <div className={`pointer-events-none absolute inset-0 ${branding ? SCRIM_CLASSES.branded : SCRIM_CLASSES.neutral}`} />
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
