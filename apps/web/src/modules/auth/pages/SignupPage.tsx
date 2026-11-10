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
import { AuthAside, AuthCard, AuthSplitLayout } from "@/modules/auth/components/AuthSplitLayout";
import type { GoogleAuthOutcome, GoogleSignupState } from "@/modules/auth/types";
import { usePageMeta } from "@/shared/hooks/usePageMeta";

const INCLUDED = [
  "Akses penuh semua fitur selama 14 hari",
  "Portal klien untuk pasangan dan keluarga",
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
        <AuthAside
          title="Rapikan setiap pernikahan yang Anda tangani."
          body="Jadwal, vendor, pembayaran, dan buktinya tercatat di satu tempat, sementara pasangan memantau persiapannya sendiri lewat Portal Klien."
        />
      }
    >
      <AuthCard>
        <h1 className="text-[28px] leading-tight font-extrabold tracking-[-0.03em] text-foreground">Coba ElProof gratis 14 hari</h1>
        <p className="mt-2 text-[14.5px] leading-relaxed text-muted-foreground">Daftar dengan akun Google, workspace WO Anda siap dalam satu menit.</p>

        <ul className="mt-6 flex flex-col gap-2.5">
          {INCLUDED.map((item) => (
            <li key={item} className="flex items-start gap-2.5 text-[14px] text-text-primary">
              <Check className="mt-0.5 h-4 w-4 shrink-0 text-seal-text" aria-hidden />
              {item}
            </li>
          ))}
        </ul>

        <div className="mt-8 flex flex-col gap-4">
          {error && (
            <p role="alert" className="rounded-md border border-danger/30 bg-danger-soft px-3.5 py-2.5 text-[13px] font-medium text-danger-text">
              {error}
            </p>
          )}
          <GoogleSignInButton text="signup_with" context="signup" onCredential={handleCredential} busy={busy} />
          <p className="text-[12px] leading-relaxed text-text-secondary">
            ElProof hanya menerima nama dan alamat email akun Google Anda untuk membuat akun Owner. Baca{" "}
            <Link to={ROUTE_PATHS.homepagePrivacy} target="_blank" className="font-medium text-text-primary underline underline-offset-2">
              Kebijakan privasi
            </Link>
            .
          </p>
        </div>

        <p className="mt-8 border-t border-border pt-5 text-center text-[14px] text-muted-foreground">
          Sudah punya akun?{" "}
          <Link to={ROUTE_PATHS.login} className="font-semibold text-text-primary hover:underline">
            Masuk
          </Link>
        </p>
      </AuthCard>
    </AuthSplitLayout>
  );
}
