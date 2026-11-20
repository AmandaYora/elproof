import axios from "axios";
import { Suspense, useEffect, useState } from "react";
import { Navigate, Outlet, useParams } from "react-router-dom";
import { TabNav } from "@/shared/components/ui/TabNav";
import { LoadError } from "@/shared/components/feedback/LoadError";
import { SkeletonRows } from "@/shared/components/ui/Skeleton";
import { registerRevalidator } from "@/shared/stores/useConflictStore";
import { ProjectHeaderCard } from "@/modules/projects/components/detail/ProjectHeaderCard";
import { useProjectStore } from "@/modules/projects/stores/useProjectStore";
import { ROUTE_PATHS } from "@/app/routes/route-paths";

export interface ProjectDetailContext {
  projectId: string;
}

// Seven tabs that fit one row (RISET WO Console §3 #4, DESAIN §5). The clients and the history are panels on Ringkasan.
// Keuangan links its parent route, which opens "klien": the tab then stays marked on every Keuangan sub-tab.
const TABS = [
  { to: "ringkasan", label: "Ringkasan" },
  { to: "acara-rincian", label: "Acara & rincian" },
  { to: "jadwal", label: "Jadwal" },
  { to: "vendor", label: "Vendor & venue" },
  { to: "keuangan", label: "Keuangan" },
  { to: "kendala", label: "Kendala" },
  { to: "dokumen", label: "Dokumen" },
];

export default function ProjectDetailLayout() {
  const { projectId } = useParams<{ projectId: string }>();
  const currentProject = useProjectStore((s) => s.currentProject);
  const fetchOverview = useProjectStore((s) => s.fetchOverview);
  const [status, setStatus] = useState<"loading" | "ready" | "not-found" | "error">("loading");
  const [attempt, setAttempt] = useState(0);

  useEffect(() => {
    if (!projectId) return;
    let cancelled = false;
    setStatus("loading");
    fetchOverview(projectId)
      .then(() => {
        if (!cancelled) setStatus("ready");
      })
      .catch((err) => {
        if (cancelled) return;
        // Only a real "no such project / not yours" sends the user away; a
        // network or server failure offers a retry instead of losing the page.
        const code = axios.isAxiosError(err) ? err.response?.status : undefined;
        setStatus(code === 404 || code === 403 ? "not-found" : "error");
      });
    return () => {
      cancelled = true;
    };
  }, [projectId, fetchOverview, attempt]);

  // After "data diubah orang lain": load the project again. The page goes back to its loading state, so every tab (and any
  // form open in it, holding old values) is built again from the latest data.
  useEffect(() => registerRevalidator(() => setAttempt((n) => n + 1)), []);

  if (!projectId || status === "not-found") {
    return <Navigate to={ROUTE_PATHS.projects} replace />;
  }

  if (status === "error") {
    return <LoadError message="Proyek belum bisa dimuat. Periksa koneksi Anda, lalu coba lagi." onRetry={() => setAttempt((n) => n + 1)} />;
  }

  if (status === "loading" || !currentProject || currentProject.id !== projectId) {
    return <div className="rounded-lg border border-border bg-surface p-5"><SkeletonRows rows={5} label="Memuat proyek" /></div>;
  }

  return (
    <div className="flex flex-col gap-5">
      <ProjectHeaderCard projectId={projectId} />
      <TabNav items={TABS} aria-label="Bagian proyek" />
      <div className="min-w-0">
        <Suspense fallback={<div className="py-16 text-center text-sm text-text-secondary">Memuat...</div>}>
          <Outlet context={{ projectId } satisfies ProjectDetailContext} />
        </Suspense>
      </div>
    </div>
  );
}
