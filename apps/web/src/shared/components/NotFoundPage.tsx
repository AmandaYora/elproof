import { Link } from "react-router-dom";
import { AppLogoMark } from "@/shared/components/AppLogoMark";
import { buttonVariants } from "@/shared/components/kit/button";
import { usePageMeta } from "@/shared/hooks/usePageMeta";
import { cn } from "@/shared/lib/cn";
import { useAuthStore, type AuthSession } from "@/shared/stores/useAuthStore";
import { ROUTE_PATHS } from "@/app/routes/route-paths";

/** Where "back" leads for whoever is looking (QA RT-06): the start page of their own console, or the public site. */
function homeFor(session: AuthSession | null): { to: string; label: string } {
  if (!session) return { to: ROUTE_PATHS.home, label: "Ke halaman utama" };
  if (session.principalType === "staff") return { to: ROUTE_PATHS.dashboard, label: "Kembali ke Beranda" };
  if (session.principalType === "client") return { to: ROUTE_PATHS.portal(), label: "Kembali ke Portal Klien" };
  return { to: ROUTE_PATHS.platformDashboard, label: "Kembali ke Platform Console" };
}

/**
 * Any path no route answers. The Go server sends the same app shell with status 404 (ADR-0043), so this page names
 * itself in the tab and stays out of search results; the way back depends on whether, and as whom, the person is
 * signed in (a visitor cannot open Beranda).
 */
export function NotFoundPage() {
  usePageMeta({ title: "Halaman tidak ditemukan", noindex: true });
  const session = useAuthStore((s) => s.session);
  const home = homeFor(session);

  return (
    <main className="flex min-h-dvh flex-col items-center justify-center gap-6 bg-background px-4 py-12 text-center">
      <AppLogoMark className="size-10" />
      <div className="flex max-w-sm flex-col gap-2">
        <p className="text-[13px] font-semibold tabular-nums text-muted-foreground">404</p>
        <h1 className="text-[24px] leading-tight font-semibold tracking-[-0.02em] text-foreground">Halaman tidak ditemukan</h1>
        <p className="text-[14px] text-muted-foreground">Alamatnya mungkin salah ketik, atau halamannya sudah dipindah. Periksa alamatnya, atau kembali ke halaman awal.</p>
      </div>
      <div className="flex flex-wrap items-center justify-center gap-2">
        <Link to={home.to} className={cn(buttonVariants())}>
          {home.label}
        </Link>
        {!session && (
          <Link to={ROUTE_PATHS.login} className={cn(buttonVariants({ variant: "secondary" }))}>
            Masuk
          </Link>
        )}
      </div>
    </main>
  );
}
