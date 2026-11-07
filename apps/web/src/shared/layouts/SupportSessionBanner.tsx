import { useEffect, useState } from "react";
import { Eye } from "lucide-react";
import { useAuthStore } from "@/shared/stores/useAuthStore";
import { endSupportSession } from "@/shared/lib/impersonation";
import { Button } from "@/shared/components/ui/Button";
import { formatDateTime } from "@/shared/lib/formatters";

/**
 * Shown on every WO Console page during a support session, so the platform
 * admin never forgets whose workspace this is or that nothing can be changed.
 */
export function SupportSessionBanner() {
  const info = useAuthStore((s) => s.session?.impersonation);
  const [ending, setEnding] = useState(false);
  const [minutesLeft, setMinutesLeft] = useState(() => minutesUntil(info?.expiresAt));

  useEffect(() => {
    const tick = window.setInterval(() => {
      const left = minutesUntil(info?.expiresAt);
      setMinutesLeft(left);
      // The token is already dead at expiry: go back instead of waiting for the next failed request.
      if (info && left === 0 && new Date(info.expiresAt).getTime() <= Date.now()) void endSupportSession();
    }, 20_000);
    return () => window.clearInterval(tick);
  }, [info?.expiresAt]);

  if (!info) return null;
  return (
    <div
      role="status"
      className="print:hidden fixed inset-x-3 bottom-3 z-50 mx-auto flex max-w-3xl flex-wrap items-center justify-between gap-x-4 gap-y-2 rounded-lg bg-warning px-4 py-2.5 text-[13px] font-semibold text-white shadow-xl"
    >
      <span className="flex min-w-0 items-center gap-2">
        <Eye className="h-4 w-4 shrink-0" />
        <span className="min-w-0">
          Anda melihat {info.tenantName} sebagai Owner, hanya untuk dilihat.{" "}
          <span className="font-medium text-white/90">
            {minutesLeft > 0 ? `Berakhir ${formatDateTime(info.expiresAt)} (${minutesLeft} menit lagi).` : "Sesi hampir berakhir."}
          </span>
        </span>
      </span>
      <Button
        size="sm"
        variant="secondary"
        disabled={ending}
        onClick={() => {
          setEnding(true);
          void endSupportSession();
        }}
      >
        Akhiri sesi
      </Button>
    </div>
  );
}

function minutesUntil(iso: string | undefined): number {
  if (!iso) return 0;
  return Math.max(0, Math.ceil((new Date(iso).getTime() - Date.now()) / 60_000));
}
