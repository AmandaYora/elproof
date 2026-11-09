import { CheckCircle2, Info, X, XCircle } from "lucide-react";
import { cn } from "@/shared/lib/cn";
import { useToastStore, type ToastTone } from "@/shared/stores/useToastStore";

const TONE: Record<ToastTone, { icon: typeof Info; className: string }> = {
  success: { icon: CheckCircle2, className: "text-success-text" },
  error: { icon: XCircle, className: "text-danger-text" },
  info: { icon: Info, className: "text-info-text" },
};

/**
 * Renders toasts (mounted once per layout). Polite live region, so screen
 * readers announce them without stealing focus; errors stay a little longer.
 */
export function Toaster() {
  const toasts = useToastStore((s) => s.toasts);
  const dismiss = useToastStore((s) => s.dismiss);
  return (
    <div aria-live="polite" className="pointer-events-none fixed inset-x-0 bottom-0 z-[60] flex flex-col items-center gap-2 p-4 sm:items-end sm:p-6">
      {toasts.map((t) => {
        const { icon: Icon, className } = TONE[t.tone];
        return (
          <div
            key={t.id}
            role={t.tone === "error" ? "alert" : "status"}
            className="pointer-events-auto flex w-full max-w-sm items-start gap-3 rounded-lg border border-border bg-surface px-4 py-3 shadow-lg motion-safe:animate-[toast-in_180ms_ease-out]"
          >
            <Icon className={cn("mt-0.5 h-4 w-4 shrink-0", className)} aria-hidden />
            <p className="min-w-0 flex-1 text-[13.5px] leading-snug text-text-primary">{t.message}</p>
            <button
              type="button"
              onClick={() => dismiss(t.id)}
              aria-label="Tutup notifikasi"
              className="rounded p-0.5 text-text-secondary hover:bg-surface-muted"
            >
              <X className="h-3.5 w-3.5" />
            </button>
          </div>
        );
      })}
    </div>
  );
}
