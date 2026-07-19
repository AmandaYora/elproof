import { Suspense, useEffect, useState } from "react";
import { Navigate, Outlet, useNavigate } from "react-router-dom";
import { Heart, LogOut } from "lucide-react";
import { TabNav } from "@/shared/components/ui/TabNav";
import { useProjectStore } from "@/modules/projects/stores/useProjectStore";
import { daysUntil } from "@/modules/projects/lib/dates";
import { APP_NAME } from "@/shared/constants/brand";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import { logoutAndRedirect } from "@/shared/lib/auth-actions";

export interface ClientPortalContext {
  projectId: string;
}

const TABS = [
  { to: ROUTE_PATHS.portal("ringkasan"), label: "Ringkasan" },
  { to: ROUTE_PATHS.portal("vendor"), label: "Vendor & Progress" },
  { to: ROUTE_PATHS.portal("pembayaran"), label: "Pembayaran" },
  { to: ROUTE_PATHS.portal("kendala"), label: "Kendala" },
];

export default function ClientPortalLayout() {
  const navigate = useNavigate();
  const project = useProjectStore((s) => s.currentProject);
  const fetchMyProject = useProjectStore((s) => s.fetchMyProject);
  const [status, setStatus] = useState<"loading" | "ready" | "denied">("loading");

  useEffect(() => {
    let cancelled = false;
    fetchMyProject()
      .then(() => {
        if (!cancelled) setStatus("ready");
      })
      .catch(() => {
        if (!cancelled) setStatus("denied");
      });
    return () => {
      cancelled = true;
    };
  }, [fetchMyProject]);

  if (status === "denied") {
    return <Navigate to={ROUTE_PATHS.login} replace />;
  }

  if (status === "loading" || !project) {
    return <div className="flex min-h-screen items-center justify-center text-sm text-text-secondary">Memuat...</div>;
  }

  const isOpenProject = project.status !== "Completed" && project.status !== "Cancelled";
  const d = daysUntil(project.eventDate);

  return (
    <div className="min-h-screen bg-background relative overflow-hidden">
      {/* Decorative grid background for depth */}
      <div className="fixed inset-0 pointer-events-none z-0">
        <div className="absolute inset-0 bg-[linear-gradient(to_right,#f1f5f9_1px,transparent_1px),linear-gradient(to_bottom,#f1f5f9_1px,transparent_1px)] bg-[size:4rem_4rem] [mask-image:radial-gradient(ellipse_60%_50%_at_50%_0%,#000_70%,transparent_100%)] opacity-60"></div>
      </div>

      <div className="sticky top-0 z-30 shadow-sm border-b border-border bg-white/80 backdrop-blur-xl">
        <div className="border-b border-border/50 bg-white/50">
          <div className="mx-auto flex max-w-5xl items-center justify-between gap-3 px-4 py-3 sm:px-6">
            <span className="flex shrink-0 items-center gap-2 text-[14px] font-bold text-navy-950">
              <div className="flex h-7 w-7 items-center justify-center rounded-lg bg-gradient-to-tr from-navy-900 to-navy-800 text-white shadow-sm">
                <Heart className="h-4 w-4" fill="currentColor" />
              </div>
              {APP_NAME}
            </span>

            <div className="flex min-w-0 flex-1 items-center justify-end gap-2.5 sm:justify-center sm:gap-4">
              <span className="hidden min-w-0 truncate text-[14px] font-semibold text-navy-950 sm:block">
                {project.brideName} &amp; {project.groomName}
              </span>
              {isOpenProject ? (
                <span className="flex shrink-0 items-center gap-1.5 rounded-full bg-blue-50 border border-blue-100 px-3 py-1 whitespace-nowrap shadow-sm">
                  <span className="text-[13px] font-bold tabular-nums text-blue-700">{d >= 0 ? d : 0}</span>
                  <span className="text-[11px] font-medium text-blue-600/80">hari lagi</span>
                </span>
              ) : (
                <span className="shrink-0 whitespace-nowrap rounded-full bg-surface-muted border border-border px-3 py-1 text-[11.5px] font-semibold text-text-secondary">
                  {project.status === "Completed" ? "Acara selesai" : "Dibatalkan"}
                </span>
              )}
            </div>

            <button
              onClick={() => void logoutAndRedirect(navigate)}
              aria-label="Keluar"
              className="flex shrink-0 items-center gap-1.5 rounded-full px-3 py-1.5 text-[12.5px] font-medium text-text-secondary transition-colors hover:bg-danger-soft hover:text-danger"
            >
              <LogOut className="h-3.5 w-3.5" /> <span className="hidden sm:inline">Keluar</span>
            </button>
          </div>
          <div className="mx-auto max-w-5xl px-4 pb-3 sm:hidden">
            <span className="block truncate text-[13.5px] font-semibold text-navy-950">
              {project.brideName} &amp; {project.groomName}
            </span>
          </div>
        </div>

        <div className="mx-auto max-w-5xl px-4 sm:px-6">
          <TabNav items={TABS} sticky={false} />
        </div>
      </div>

      <main className="mx-auto max-w-5xl px-4 py-8 sm:px-6 sm:py-10 relative z-10">
        <Suspense fallback={<div className="py-16 text-center text-sm text-text-secondary animate-pulse">Memuat...</div>}>
          <Outlet context={{ projectId: project.id } satisfies ClientPortalContext} />
        </Suspense>
      </main>
    </div>
  );
}
