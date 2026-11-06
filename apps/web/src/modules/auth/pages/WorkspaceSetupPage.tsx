import { useState, type FormEvent } from "react";
import { Link, Navigate, useLocation, useNavigate } from "react-router-dom";
import axios from "axios";
import { Info } from "lucide-react";
import { Button } from "@/shared/components/ui/Button";
import { Field, Input } from "@/shared/components/ui/Input";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import { useAuthStore, type AuthSession } from "@/shared/stores/useAuthStore";
import { httpClient } from "@/shared/services/http-client";
import { API } from "@/shared/services/api-endpoints";
import { getApiErrorMessage } from "@/shared/lib/api-error";
import { AuthSplitLayout } from "@/modules/auth/components/AuthSplitLayout";
import { workspaceSchema, type WorkspaceFormValues } from "@/modules/auth/schemas/workspace.schema";
import type { GoogleSignupState } from "@/modules/auth/types";

const EXPIRED_NOTICE = "Sesi Google Anda sudah berakhir. Silakan daftar ulang dengan Google.";

function apiErrorCode(err: unknown): string | undefined {
  if (!axios.isAxiosError(err)) return undefined;
  return (err.response?.data as { errors?: { code?: string } } | undefined)?.errors?.code;
}

/**
 * Step 2 of self-service signup (PLAN.md §3): one short form, then the
 * workspace is created and the Owner is signed in with the same Google
 * credential.
 */
export default function WorkspaceSetupPage() {
  const navigate = useNavigate();
  const signup = useLocation().state as GoogleSignupState | null;
  const [values, setValues] = useState<WorkspaceFormValues>({ businessName: "", acceptTerms: false, marketingOptIn: false });
  const [errors, setErrors] = useState<Partial<Record<keyof WorkspaceFormValues, string>>>({});
  const [formError, setFormError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  if (!signup?.credential) {
    // Opened directly (or the history entry was replaced): start again from
    // the Google step. The credential travels in router state only — it is
    // never written to storage — and expires with the ID token (~1 hour).
    return <Navigate to={ROUTE_PATHS.signup} replace />;
  }
  const { credential, email, name, invited } = signup;

  function set<K extends keyof WorkspaceFormValues>(key: K, value: WorkspaceFormValues[K]) {
    setValues((prev) => ({ ...prev, [key]: value }));
    setErrors((prev) => ({ ...prev, [key]: undefined }));
    setFormError(null);
  }

  async function signInAndContinue() {
    const res = await httpClient.post(API.auth.google, { credential });
    const session = res.data.data as AuthSession | { signupRequired: true };
    if ("signupRequired" in session) throw new Error("workspace not ready");
    useAuthStore.getState().login(session);
    navigate(ROUTE_PATHS.dashboard, { replace: true, state: { welcome: true } });
  }

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    const parsed = workspaceSchema.safeParse(values);
    if (!parsed.success) {
      const fieldErrors: Partial<Record<keyof WorkspaceFormValues, string>> = {};
      for (const issue of parsed.error.issues) fieldErrors[issue.path[0] as keyof WorkspaceFormValues] = issue.message;
      setErrors(fieldErrors);
      return;
    }
    setSubmitting(true);
    setFormError(null);
    try {
      await httpClient.post(API.public.signupWorkspace, { credential, ...parsed.data });
      await signInAndContinue();
    } catch (err) {
      const status = axios.isAxiosError(err) ? err.response?.status : undefined;
      const code = apiErrorCode(err);
      if (code === "google_account_exists") {
        // Registered meanwhile (another tab): sign in instead.
        try {
          await signInAndContinue();
          return;
        } catch {
          /* fall through to the message */
        }
      }
      if (status === 401) {
        navigate(ROUTE_PATHS.signup, { replace: true, state: { notice: EXPIRED_NOTICE } });
        return;
      }
      if (status === 422 && axios.isAxiosError(err)) {
        const fields = (err.response?.data as { errors?: Record<string, string[]> })?.errors ?? {};
        setErrors({
          businessName: fields.businessName?.[0],
          acceptTerms: fields.acceptTerms?.[0],
        });
      }
      setFormError(getApiErrorMessage(err, "Workspace belum berhasil dibuat. Silakan coba lagi."));
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <AuthSplitLayout
      aside={
        <>
          <h1 className="text-[36px] font-bold leading-[1.15] text-white">Satu langkah lagi.</h1>
          <div className="mt-4 h-px w-12 bg-white/40" />
          <p className="mt-4 text-[15px] leading-relaxed text-white/70">
            Setelah workspace dibuat, Anda langsung masuk ke WO Console dengan panduan singkat untuk memulai.
          </p>
        </>
      }
    >
      <form onSubmit={handleSubmit} noValidate className="rounded-xl border border-border bg-surface p-6 shadow-sm sm:p-10">
        <h2 className="text-2xl font-bold text-text-primary">Buat workspace WO</h2>
        <div className="mt-3 flex flex-wrap items-center gap-x-2 gap-y-1 text-[13.5px] text-text-secondary">
          <span>
            Daftar sebagai <span className="font-semibold text-text-primary">{name || email}</span>
            {name && <span> ({email})</span>}
          </span>
          <Link to={ROUTE_PATHS.signup} replace className="font-medium text-text-primary underline underline-offset-2">
            Ganti akun
          </Link>
        </div>

        <div className="mt-8 flex flex-col gap-5">
          {invited && (
            <p role="status" className="rounded-md border border-warning/30 bg-warning-soft px-3.5 py-3 text-[13px] leading-relaxed text-warning">
              Ada undangan staf untuk {email}. Jika Anda staf di sebuah WO, jangan buat workspace baru: buka link undangan yang
              dikirim WO Anda (lewat WhatsApp), lalu masuk dengan Google dari sana.
            </p>
          )}

          {formError && (
            <p role="alert" className="rounded-md border border-danger/30 bg-danger-soft px-3.5 py-2.5 text-[13px] font-medium text-danger">
              {formError}
            </p>
          )}

          <Field label="Nama WO" htmlFor="businessName" required error={errors.businessName} hint="Tampil di WO Console dan portal client. Bisa diubah nanti.">
            <Input
              id="businessName"
              value={values.businessName}
              onChange={(e) => set("businessName", e.target.value)}
              placeholder="Contoh: Kirana Wedding Organizer"
              autoComplete="organization"
              maxLength={100}
              autoFocus
            />
          </Field>

          <div className="flex flex-col gap-1.5">
            <label className="flex items-start gap-3 text-[13.5px] leading-relaxed text-text-primary">
              <input
                type="checkbox"
                checked={values.acceptTerms}
                onChange={(e) => set("acceptTerms", e.target.checked)}
                className="mt-1 h-4 w-4 shrink-0 accent-navy-900"
                aria-invalid={Boolean(errors.acceptTerms)}
              />
              <span>
                Saya menyetujui{" "}
                <Link to={ROUTE_PATHS.homepageTerms} target="_blank" className="font-medium underline underline-offset-2">
                  Syarat & Ketentuan
                </Link>{" "}
                dan{" "}
                <Link to={ROUTE_PATHS.homepageDataProcessing} target="_blank" className="font-medium underline underline-offset-2">
                  Perjanjian Pemrosesan Data
                </Link>
                , serta telah membaca{" "}
                <Link to={ROUTE_PATHS.homepagePrivacy} target="_blank" className="font-medium underline underline-offset-2">
                  Kebijakan Privasi
                </Link>{" "}
                ElProof.
              </span>
            </label>
            {errors.acceptTerms && (
              <span role="alert" className="pl-7 text-xs font-medium text-danger">
                {errors.acceptTerms}
              </span>
            )}
          </div>

          <label className="flex items-start gap-3 text-[13.5px] leading-relaxed text-text-secondary">
            <input
              type="checkbox"
              checked={values.marketingOptIn}
              onChange={(e) => set("marketingOptIn", e.target.checked)}
              className="mt-1 h-4 w-4 shrink-0 accent-navy-900"
            />
            <span>Kirimi saya kabar fitur baru dan promo ElProof (opsional).</span>
          </label>

          <p className="flex gap-2.5 rounded-md border border-border bg-surface-muted px-3.5 py-3 text-[12.5px] leading-relaxed text-text-secondary">
            <Info className="mt-0.5 h-4 w-4 shrink-0" aria-hidden />
            <span>
              Staf dari WO yang sudah memakai ElProof? Jangan buat workspace baru. Minta Owner WO Anda menambahkan email ini,
              lalu{" "}
              <Link to={ROUTE_PATHS.login} className="font-medium text-text-primary underline underline-offset-2">
                masuk dengan Google
              </Link>
              .
            </span>
          </p>

          <Button type="submit" variant="neutral" className="mt-1 w-full justify-center" disabled={submitting}>
            {submitting ? "Menyiapkan workspace..." : "Buat workspace"}
          </Button>
        </div>
      </form>
    </AuthSplitLayout>
  );
}
