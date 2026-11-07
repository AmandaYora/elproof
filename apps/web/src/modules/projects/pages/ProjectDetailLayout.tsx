import axios from "axios";
import { Suspense, useEffect, useState } from "react";
import { Navigate, Outlet, useParams } from "react-router-dom";
import { TabNav } from "@/shared/components/ui/TabNav";
import { LoadError } from "@/shared/components/feedback/LoadError";
import { SkeletonRows } from "@/shared/components/ui/Skeleton";
import { ProjectHeaderCard } from "@/modules/projects/components/detail/ProjectHeaderCard";
import { ProjectSidePanel } from "@/modules/projects/components/detail/ProjectSidePanel";
import { useProjectStore } from "@/modules/projects/stores/useProjectStore";
import { ROUTE_PATHS } from "@/app/routes/route-paths";

export interface ProjectDetailContext {
  projectId: string;
}

const TABS = [
  { to: "ringkasan", label: "Ringkasan" },
  { to: "milestone", label: "Timeline" },
  { to: "vendor", label: "Vendor" },
  { to: "venue", label: "Venue" },
  { to: "pembayaran", label: "Keuangan" },
  { to: "kendala", label: "Kendala" },
  { to: "dokumen", label: "Dokumen" },
  { to: "client", label: "Klien" },
  { to: "aktivitas", label: "Aktivitas" },
];

export default function ProjectDetailLayout() {
  const { projectId } = useParams<{ projectId: string }>();
  const currentProject = useProjectStore((s) => s.currentProject);
  const fetchProjectDetail = useProjectStore((s) => s.fetchProjectDetail);
  const [status, setStatus] = useState<"loading" | "ready" | "not-found" | "error">("loading");
  const [attempt, setAttempt] = useState(0);

  useEffect(() => {
    if (!projectId) return;
    let cancelled = false;
    setStatus("loading");
    fetchProjectDetail(projectId)
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
  }, [projectId, fetchProjectDetail, attempt]);

  if (!projectId || status === "not-found") {
    return <Navigate to={ROUTE_PATHS.projects} replace />;
  }

  if (status === "error") {
    return <LoadError message="Project belum bisa dimuat. Periksa koneksi Anda, lalu coba lagi." onRetry={() => setAttempt((n) => n + 1)} />;
  }

  if (status === "loading" || !currentProject || currentProject.id !== projectId) {
    return <div className="rounded-lg border border-border bg-surface p-5"><SkeletonRows rows={5} label="Memuat project" /></div>;
  }

  return (
    <div className="flex flex-col gap-5">
      <ProjectHeaderCard projectId={projectId} />
      <TabNav items={TABS} />
      <div className="grid grid-cols-1 gap-5 lg:grid-cols-[minmax(0,1fr)_17rem]">
        <div className="min-w-0">
          <Suspense fallback={<div className="py-16 text-center text-sm text-text-secondary">Memuat...</div>}>
            <Outlet context={{ projectId } satisfies ProjectDetailContext} />
          </Suspense>
        </div>
        <ProjectSidePanel projectId={projectId} />
      </div>
    </div>
  );
}
