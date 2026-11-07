import { useCallback, useState } from "react";
import { Link, useLocation, useNavigate } from "react-router-dom";
import { Check } from "lucide-react";
import { GoogleSignInButton } from "@/shared/components/auth/GoogleSignInButton";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import { useAuthStore, type AuthSession } from "@/shared/stores/useAuthStore";
import { landingPathFor } from "@/shared/lib/auth-actions";
import { httpClient } from "@/shared/services/http-client";
import { API } from "@/shared/services/api-endpoints";
import { getApiErrorMessage } from "@/shared/lib/api-error";
import { AuthSplitLayout } from "@/modules/auth/components/AuthSplitLayout";
import type { GoogleAuthOutcome, GoogleSignupState } from "@/modules/auth/types";
import { usePageMeta } from "@/shared/hooks/usePageMeta";

const INCLUDED = [
  "Akses penuh semua fitur selama 14 hari",
  "Portal client untuk pasangan dan keluarga",
  "Tanpa kartu kredit, tanpa tagihan otomatis",
];

type GoogleAuthResponse = AuthSession | GoogleAuthOutcome;

export default function SignupPage() {
  usePageMeta({ title: "Daftar gratis", description: "Daftarkan wedding organizer Anda dan coba ElProof gratis selama masa percobaan, tanpa kartu kredit." });
  const navigate = useNavigate();
  const location = useLocation();
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(
    (location.state as { notice?: string } | null)?.notice ?? null,
  );

  const handleCredential = useCallback(
    async (credential: string) => {
      setError(null);
      setBusy(true);
      try {
        const res = await httpClient.post(API.auth.google, { credential });
        const data = res.data.data as GoogleAuthResponse;
        if ("signupRequired" in data || "linkRequired" in data) {
          const state: GoogleSignupState = {
            credential,
            email: data.email,
            name: data.name,
            invited: "signupRequired" in data ? data.invited : undefined,
          };
          // A staff login already uses this email: confirm it on the
          // login page (or choose to create a new workspace there).
          if ("linkRequired" in data) navigate(ROUTE_PATHS.login, { state: { googleLink: state } });
          else navigate(ROUTE_PATHS.signupWorkspace, { state });
          return;
        }
        // Already registered: just sign in.
        useAuthStore.getState().login(data);
        navigate(landingPathFor(data));
      } catch (err) {
        setError(getApiErrorMessage(err, "Pendaftaran dengan Google gagal. Silakan coba lagi."));
      } finally {
        setBusy(false);
      }
    },
    [navigate],
  );

  return (
    <AuthSplitLayout
      aside={
        <>
          <h1 className="text-[36px] font-bold leading-[1.15] text-white">Rapikan setiap pernikahan yang Anda tangani.</h1>
          <div className="mt-4 h-px w-12 bg-white/40" />
          <p className="mt-4 text-[15px] leading-relaxed text-white/70">
            Timeline, vendor, pembayaran, dan bukti kerja tercatat di satu tempat, sementara pasangan memantau
            persiapannya sendiri lewat portal client.
          </p>
        </>
      }
    >
      <div className="rounded-xl border border-border bg-surface p-6 shadow-sm sm:p-10">
        <h2 className="text-2xl font-bold text-text-primary">Coba ElProof gratis 14 hari</h2>
        <p className="mt-1.5 text-[13.5px] text-text-secondary">Daftar dengan akun Google, workspace WO Anda siap dalam satu menit.</p>

        <ul className="mt-6 flex flex-col gap-2.5">
          {INCLUDED.map((item) => (
            <li key={item} className="flex items-start gap-2.5 text-[14px] text-text-primary">
              <Check className="mt-0.5 h-4 w-4 shrink-0 text-success" aria-hidden />
              {item}
            </li>
          ))}
        </ul>

        <div className="mt-8 flex flex-col gap-4">
          {error && (
            <p role="alert" className="rounded-md border border-danger/30 bg-danger-soft px-3.5 py-2.5 text-[13px] font-medium text-danger">
              {error}
            </p>
          )}
          <GoogleSignInButton text="signup_with" context="signup" onCredential={handleCredential} busy={busy} />
          <p className="text-[12px] leading-relaxed text-text-secondary">
            ElProof hanya menerima nama dan alamat email akun Google Anda untuk membuat akun Owner. Baca{" "}
            <Link to={ROUTE_PATHS.homepagePrivacy} target="_blank" className="font-medium text-text-primary underline underline-offset-2">
              Kebijakan Privasi
            </Link>
            .
          </p>
        </div>

        <p className="mt-8 border-t border-border pt-5 text-center text-[13.5px] text-text-secondary">
          Sudah punya akun?{" "}
          <Link to={ROUTE_PATHS.login} className="font-semibold text-text-primary hover:underline">
            Masuk
          </Link>
        </p>
      </div>
    </AuthSplitLayout>
  );
}
