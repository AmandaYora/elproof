import { useRef } from "react";
import { Link, Navigate, useLocation, useNavigate } from "react-router-dom";
import axios from "axios";
import { CircleAlert, Info } from "lucide-react";
import { Button } from "@/shared/components/ui/Button";
import { Field, Input } from "@/shared/components/ui/Input";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import { useAuthStore, type AuthSession } from "@/shared/stores/useAuthStore";
import { httpClient } from "@/shared/services/http-client";
import { API } from "@/shared/services/api-endpoints";
import { AuthSplitLayout } from "@/modules/auth/components/AuthSplitLayout";
import { workspaceSchema } from "@/modules/auth/schemas/workspace.schema";
import type { GoogleSignupState } from "@/modules/auth/types";
import { usePageMeta } from "@/shared/hooks/usePageMeta";
import { useZodForm } from "@/shared/forms/useZodForm";
import { FormErrorSummary } from "@/shared/forms/FormErrorSummary";

const FIELD_LABELS: Record<string, string> = { businessName: "Nama WO", acceptTerms: "Persetujuan" };
const FIELDS = (key: string) => key === "businessName" || key === "acceptTerms";

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
  usePageMeta({ title: "Siapkan workspace", description: "Lengkapi data WO Anda untuk mulai memakai ElProof.", noindex: true });
  const navigate = useNavigate();
  const signup = useLocation().state as GoogleSignupState | null;
  const form = useZodForm({ schema: workspaceSchema, initial: { businessName: "", acceptTerms: false, marketingOptIn: false } });
  const { values, errors } = form;
  const formRef = useRef<HTMLFormElement>(null);

  if (!signup?.credential) {
    // Opened directly (or the history entry was replaced): start again from
    // the Google step. The credential travels in router state only — it is
    // never written to storage — and expires with the ID token (~1 hour).
    return <Navigate to={ROUTE_PATHS.signup} replace />;
  }
  const { credential, email, name, invited } = signup;

  async function signInAndContinue() {
    const res = await httpClient.post(API.auth.google, { credential });
    const session = res.data.data as AuthSession | { signupRequired: true };
    if ("signupRequired" in session) throw new Error("workspace not ready");
    useAuthStore.getState().login(session);
    navigate(ROUTE_PATHS.dashboard, { replace: true, state: { welcome: true } });
  }

  const handleSubmit = form.handleSubmit(
    async (parsed) => {
      try {
        await httpClient.post(API.public.signupWorkspace, { credential, ...parsed });
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
        throw err; // a 422 is shown on its fields, anything else above the form
      }
    },
    { root: () => formRef.current, fallback: "Workspace belum berhasil dibuat. Silakan coba lagi.", known: FIELDS },
  );

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
      <form ref={formRef} onSubmit={(e) => void handleSubmit(e)} noValidate className="rounded-xl border border-border bg-surface p-6 shadow-sm sm:p-10">
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
            <p role="status" className="rounded-md border border-warning/30 bg-warning-soft px-3.5 py-3 text-[13px] leading-relaxed text-warning-text">
              Ada undangan staf untuk {email}. Jika Anda staf di sebuah WO, jangan buat workspace baru: buka link undangan yang
              dikirim WO Anda (lewat WhatsApp), lalu masuk dengan Google dari sana.
            </p>
          )}

          <FormErrorSummary errors={form.errorList} formError={form.formError} labels={FIELD_LABELS} root={() => formRef.current} />

          <Field label="Nama WO" htmlFor="businessName" required name="businessName" error={errors.businessName} hint="Tampil di WO Console dan portal client. Bisa diubah nanti.">
            <Input
              id="businessName"
              value={values.businessName}
              onChange={(e) => form.set("businessName", e.target.value)}
              onBlur={() => form.blur("businessName")}
              placeholder="Contoh: Kirana Wedding Organizer"
              autoComplete="organization"
              maxLength={100}
              autoFocus
            />
          </Field>

          <div data-field="acceptTerms" className="flex flex-col gap-1.5">
            <label className="flex items-start gap-3 text-[13.5px] leading-relaxed text-text-primary">
              <input
                type="checkbox"
                checked={values.acceptTerms}
                onChange={(e) => form.set("acceptTerms", e.target.checked)}
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
              <span role="alert" className="flex items-start gap-1 pl-7 text-xs font-medium text-danger-text">
                <CircleAlert aria-hidden className="mt-px size-3.5 shrink-0" />
                {errors.acceptTerms}
              </span>
            )}
          </div>

          <label className="flex items-start gap-3 text-[13.5px] leading-relaxed text-text-secondary">
            <input
              type="checkbox"
              checked={values.marketingOptIn}
              onChange={(e) => form.set("marketingOptIn", e.target.checked)}
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

          <Button type="submit" variant="neutral" className="mt-1 w-full justify-center" disabled={form.submitting}>
            {form.submitting ? "Menyiapkan workspace..." : "Buat workspace"}
          </Button>
        </div>
      </form>
    </AuthSplitLayout>
  );
}
