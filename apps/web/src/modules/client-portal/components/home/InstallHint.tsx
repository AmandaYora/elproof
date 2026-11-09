import { useState } from "react";
import { Share, Smartphone, X } from "lucide-react";
import { Button } from "@/shared/components/kit/button";
import { isIOS, isInstalledApp, useInstallPrompt } from "@/shared/lib/pwa";

const DISMISSED = "elproof.portal.installHint.dismissed";

function wasDismissed(): boolean {
  try {
    return window.localStorage.getItem(DISMISSED) === "1";
  } catch {
    return false;
  }
}

/**
 * A small, dismissible nudge to put the portal on the home screen, shown only on
 * a phone that can do it and has not yet. Android with an install prompt gets a
 * button; iPhone gets the two taps it needs; anything else shows nothing.
 */
export function InstallHint() {
  const { event, consume } = useInstallPrompt();
  const [hidden, setHidden] = useState(() => isInstalledApp() || wasDismissed());
  const ios = isIOS();
  if (hidden || (!event && !ios)) return null;

  function dismiss() {
    setHidden(true);
    try {
      window.localStorage.setItem(DISMISSED, "1");
    } catch {
      // Not remembered: it shows again next visit, which is harmless.
    }
  }

  async function install() {
    if (!event) return;
    await event.prompt();
    const choice = await event.userChoice;
    consume();
    if (choice.outcome === "accepted") setHidden(true);
  }

  return (
    <section aria-labelledby="install-hint" className="relative rounded-2xl border border-border bg-card p-4 pr-12 sm:p-5 sm:pr-14">
      <h2 id="install-hint" className="flex items-center gap-2 text-[14px] font-semibold text-foreground">
        <Smartphone className="size-4" aria-hidden /> Pasang di layar utama
      </h2>
      {event ? (
        <>
          <p className="mt-1.5 text-[13px] text-muted-foreground">Buka portal langsung dari ikon di layar utama, tanpa mengetik alamatnya.</p>
          <Button className="mt-3" onClick={() => void install()}>Pasang sekarang</Button>
        </>
      ) : (
        <p className="mt-1.5 text-[13px] text-muted-foreground">
          Di Safari, ketuk Bagikan <Share className="inline size-3.5 align-text-bottom" aria-label="Bagikan" />, lalu pilih <strong className="font-semibold text-foreground">Tambah ke Layar Utama</strong>.
        </p>
      )}
      <Button variant="ghost" size="icon-sm" className="absolute top-2 right-2" onClick={dismiss}>
        <X aria-hidden />
        <span className="sr-only">Tutup petunjuk</span>
      </Button>
    </section>
  );
}
