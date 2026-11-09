import { useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { ChevronRight, FileText, HeartHandshake, LogOut, Share, Smartphone } from "lucide-react";
import { Button } from "@/shared/components/ui/Button";
import { useCan } from "@/shared/stores/usePermissionStore";
import { logoutAndRedirect } from "@/shared/lib/auth-actions";
import { isIOS, isInstalledApp, useInstallPrompt } from "@/shared/lib/pwa";
import { ROUTE_PATHS, type ClientPortalTab } from "@/app/routes/route-paths";

function Row({ to, icon: Icon, label, hint }: { to: ClientPortalTab; icon: typeof FileText; label: string; hint: string }) {
  return (
    <Link
      to={ROUTE_PATHS.portal(to)}
      className="flex min-h-14 items-center gap-3 border-b border-border-light px-4 py-3 last:border-b-0 hover:bg-surface-muted focus-visible:outline focus-visible:outline-2 focus-visible:outline-navy-900"
    >
      <Icon className="h-5 w-5 shrink-0 text-navy-900" aria-hidden />
      <span className="min-w-0 flex-1">
        <span className="block text-[14px] font-semibold text-text-primary">{label}</span>
        <span className="block text-[12.5px] text-text-secondary">{hint}</span>
      </span>
      <ChevronRight className="h-4 w-4 shrink-0 text-text-secondary" aria-hidden />
    </Link>
  );
}

/** Everything that does not fit the bottom bar: documents, issues, install, sign out. */
export default function MoreTabPage() {
  const navigate = useNavigate();
  const canDocuments = useCan("portal.documents.view");
  const canIssues = useCan("portal.issues.view");
  const { event: installEvent, consume } = useInstallPrompt();
  const [installed, setInstalled] = useState(() => isInstalledApp());

  async function install() {
    if (!installEvent) return;
    await installEvent.prompt();
    const choice = await installEvent.userChoice;
    consume();
    if (choice.outcome === "accepted") setInstalled(true);
  }

  return (
    <div className="mx-auto flex w-full max-w-3xl flex-col gap-5">
      <h1 className="text-xl font-bold text-text-primary">Lainnya</h1>

      {(canDocuments || canIssues) && (
        <div className="overflow-hidden rounded-2xl border border-border bg-white">
          {canDocuments && <Row to="dokumen" icon={FileText} label="Dokumen" hint="Rundown, buku acara, dan dokumen yang dibagikan WO" />}
          {canIssues && <Row to="kendala" icon={HeartHandshake} label="Kendala" hint="Hal yang sedang kami tangani untuk Anda" />}
        </div>
      )}

      <section aria-labelledby="install-heading" className="rounded-2xl border border-border bg-white p-5">
        <h2 id="install-heading" className="flex items-center gap-2 text-[15px] font-semibold text-text-primary">
          <Smartphone className="h-4 w-4" aria-hidden /> Pasang di layar utama
        </h2>
        {installed ? (
          <p className="mt-2 text-[13.5px] text-text-secondary">Portal sudah terpasang di layar utama ponsel Anda.</p>
        ) : installEvent ? (
          <>
            <p className="mt-2 text-[13.5px] text-text-secondary">Buka portal langsung dari ikon di layar utama, tanpa mengetik alamatnya.</p>
            <Button className="mt-3" onClick={() => void install()}>Pasang sekarang</Button>
          </>
        ) : isIOS() ? (
          <ol className="mt-2 list-decimal space-y-1.5 pl-5 text-[13.5px] text-text-secondary">
            <li>Buka halaman ini di <strong className="font-semibold text-text-primary">Safari</strong>.</li>
            <li>Ketuk tombol Bagikan <Share className="inline h-3.5 w-3.5 align-text-bottom" aria-label="Bagikan" />.</li>
            <li>Pilih <strong className="font-semibold text-text-primary">Tambah ke Layar Utama</strong>, lalu ketuk Tambah.</li>
          </ol>
        ) : (
          <p className="mt-2 text-[13.5px] text-text-secondary">
            Di Chrome, buka menu di pojok kanan atas (tiga titik), lalu pilih <strong className="font-semibold text-text-primary">Tambahkan ke layar utama</strong>.
          </p>
        )}
      </section>

      <button
        type="button"
        onClick={() => void logoutAndRedirect(navigate)}
        className="flex min-h-12 items-center justify-center gap-2 rounded-2xl border border-border bg-white px-4 text-[14px] font-semibold text-text-secondary hover:bg-danger-soft hover:text-danger-text focus-visible:outline focus-visible:outline-2 focus-visible:outline-navy-900"
      >
        <LogOut className="h-4 w-4" aria-hidden /> Keluar
      </button>
    </div>
  );
}
