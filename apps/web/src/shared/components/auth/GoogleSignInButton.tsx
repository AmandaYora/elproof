import { useEffect, useRef, useState } from "react";
import { Loader2 } from "lucide-react";
import { renderGoogleButton, type GoogleButtonText } from "@/shared/lib/google-identity";

interface GoogleSignInButtonProps {
  text: GoogleButtonText;
  context: "signin" | "signup";
  onCredential: (credential: string) => void;
  /** Covers the button while the credential is being processed. */
  busy?: boolean;
}

/**
 * Google's branded "Sign in with Google" button (Google Identity Services,
 * popup mode). Shows a placeholder while loading and a plain message if
 * Google's script cannot load (blocked network, ad-blocker).
 */
export function GoogleSignInButton({ text, context, onCredential, busy = false }: GoogleSignInButtonProps) {
  const hostRef = useRef<HTMLDivElement>(null);
  const callbackRef = useRef(onCredential);
  const [state, setState] = useState<"loading" | "ready" | "error">("loading");

  useEffect(() => {
    callbackRef.current = onCredential;
  }, [onCredential]);

  useEffect(() => {
    let cancelled = false;
    const host = hostRef.current;
    if (!host) return;
    renderGoogleButton(host, { text, context, onCredential: (c) => callbackRef.current(c), isCancelled: () => cancelled })
      .then(() => !cancelled && setState("ready"))
      .catch(() => !cancelled && setState("error"));
    return () => {
      cancelled = true;
      host.innerHTML = "";
    };
  }, [text, context]);

  return (
    <div className="relative w-full">
      {state === "loading" && (
        <div className="flex h-11 w-full items-center justify-center rounded-md border border-border bg-surface text-[13px] text-text-secondary">
          <Loader2 className="mr-2 h-4 w-4 animate-spin" /> Memuat tombol Google...
        </div>
      )}
      {state === "error" && (
        <p role="alert" className="rounded-md border border-danger/30 bg-danger-soft px-3.5 py-2.5 text-[13px] font-medium text-danger-text">
          Tombol Google tidak dapat dimuat. Periksa koneksi internet atau nonaktifkan pemblokir iklan, lalu muat ulang halaman.
        </p>
      )}
      <div ref={hostRef} className={state === "ready" ? "flex min-h-11 w-full justify-center" : "h-0 overflow-hidden"} />
      {busy && state === "ready" && (
        <div className="absolute inset-0 flex items-center justify-center rounded-md bg-surface/85 text-[13px] font-medium text-text-secondary">
          <Loader2 className="mr-2 h-4 w-4 animate-spin" /> Memproses...
        </div>
      )}
    </div>
  );
}
