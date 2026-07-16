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
    <div className="min-h-screen bg-background">
      <div className="sticky top-0 z-30 shadow-sm">
        <div className="bg-gradient-to-r from-navy-950 to-navy-900">
          <div className="mx-auto flex max-w-5xl items-center justify-between gap-3 px-4 py-2.5 sm:px-6">
            <span className="flex shrink-0 items-center gap-1.5 text-[13px] font-bold text-white">
              <Heart className="h-4 w-4 shrink-0" fill="currentColor" />
              {APP_NAME}
            </span>

            <div className="flex min-w-0 flex-1 items-center justify-end gap-2.5 sm:justify-center sm:gap-3">
              <span className="hidden min-w-0 truncate text-[13px] font-medium text-white/85 sm:block">
                {project.brideName} &amp; {project.groomName}
              </span>
              {isOpenProject ? (
                <span className="flex shrink-0 items-center gap-1.5 rounded-full bg-white/10 px-3 py-1 whitespace-nowrap">
                  <span className="text-[13px] font-bold tabular-nums text-white">{d >= 0 ? d : 0}</span>
                  <span className="text-[11px] text-white/70">hari lagi</span>
                </span>
              ) : (
                <span className="shrink-0 whitespace-nowrap rounded-full bg-white/10 px-3 py-1 text-[11.5px] font-medium text-white/85">
                  {project.status === "Completed" ? "Acara selesai" : "Dibatalkan"}
                </span>
              )}
            </div>

            <button
              onClick={() => void logoutAndRedirect(navigate)}
              aria-label="Keluar"
              className="flex shrink-0 items-center gap-1.5 text-[12.5px] font-medium text-white/70 hover:text-white"
            >
              <LogOut className="h-3.5 w-3.5" /> <span className="hidden sm:inline">Keluar</span>
            </button>
          </div>
          <div className="mx-auto max-w-5xl px-4 pb-2 sm:hidden">
            <span className="block truncate text-[12.5px] font-medium text-white/85">
              {project.brideName} &amp; {project.groomName}
            </span>
          </div>
        </div>

        <div className="mx-auto max-w-5xl bg-surface px-4 sm:px-6">
          <TabNav items={TABS} sticky={false} />
        </div>
      </div>

      <main className="mx-auto max-w-5xl px-4 py-6 sm:px-6 sm:py-8">
        <Suspense fallback={<div className="py-16 text-center text-sm text-text-secondary">Memuat...</div>}>
          <Outlet context={{ projectId: project.id } satisfies ClientPortalContext} />
        </Suspense>
      </main>
    </div>
  );
}
