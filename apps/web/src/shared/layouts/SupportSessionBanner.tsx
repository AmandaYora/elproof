import { useEffect, useState } from "react";
import { Eye } from "lucide-react";
import { useAuthStore } from "@/shared/stores/useAuthStore";
import { endSupportSession } from "@/shared/lib/impersonation";
import { Button } from "@/shared/components/ui/Button";
import { formatDateTime } from "@/shared/lib/formatters";

/**
 * Shown on every WO Console page during a support session, so the platform admin never forgets whose workspace this
 * is or that nothing can be changed. One line pinned to the bottom edge (beside the menu from `lg` up); AppLayout keeps
 * the same height free under the page (`pb-12`), so it never sits on top of content (QA FQ-15). On a phone it says only
 * what matters: support session, view only, minutes left.
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
      // From sm up z-70: above dialogs (50) and toasts (60) so the person never forgets whose workspace this is (DESIGN.md
      // §4.4). On a phone it stays under the menu sheet and dialogs (z-40), which would otherwise lose their last row.
      className="print:hidden fixed inset-x-0 bottom-0 z-40 flex h-12 items-center justify-between gap-3 border-t border-warning/50 bg-warning-soft px-4 text-[13px] font-semibold text-warning-text sm:z-[70] lg:left-(--sidebar-w)"
    >
      <span className="flex min-w-0 items-center gap-2">
        <Eye aria-hidden className="h-4 w-4 shrink-0" />
        <span className="truncate sm:hidden">
          Sesi dukungan · hanya lihat · {minutesLeft > 0 ? `${minutesLeft} menit` : "hampir berakhir"}
        </span>
        <span className="hidden truncate sm:inline">
          Anda melihat {info.tenantName} sebagai Owner, hanya untuk dilihat.{" "}
          <span className="font-medium">
            {minutesLeft > 0 ? `Berakhir ${formatDateTime(info.expiresAt)} (${minutesLeft} menit lagi).` : "Sesi hampir berakhir."}
          </span>
        </span>
      </span>
      <Button
        size="sm"
        variant="secondary"
        disabled={ending}
        aria-label="Akhiri sesi dukungan"
        onClick={() => {
          setEnding(true);
          void endSupportSession();
        }}
      >
        <span className="sm:hidden">Akhiri</span>
        <span className="hidden sm:inline">Akhiri sesi</span>
      </Button>
    </div>
  );
}

function minutesUntil(iso: string | undefined): number {
  if (!iso) return 0;
  return Math.max(0, Math.ceil((new Date(iso).getTime() - Date.now()) / 60_000));
}
