import { Suspense, useEffect, useState } from "react";
import { Navigate, Outlet, useParams } from "react-router-dom";
import { TabNav } from "@/shared/components/ui/TabNav";
import { ProjectHeaderCard } from "@/modules/projects/components/detail/ProjectHeaderCard";
import { useProjectStore } from "@/modules/projects/stores/useProjectStore";
import { ROUTE_PATHS } from "@/app/routes/route-paths";

export interface ProjectDetailContext {
  projectId: string;
}

const TABS = [
  { to: "vendor", label: "Vendor" },
  { to: "milestone", label: "Milestone" },
  { to: "client", label: "Client" },
  { to: "pembayaran", label: "Pembayaran" },
  { to: "kendala", label: "Kendala" },
  { to: "dokumen", label: "Dokumen" },
  { to: "aktivitas", label: "Aktivitas" },
];

export default function ProjectDetailLayout() {
  const { projectId } = useParams<{ projectId: string }>();
  const currentProject = useProjectStore((s) => s.currentProject);
  const fetchProjectDetail = useProjectStore((s) => s.fetchProjectDetail);
  const [status, setStatus] = useState<"loading" | "ready" | "not-found">("loading");

  useEffect(() => {
    if (!projectId) return;
    let cancelled = false;
    setStatus("loading");
    fetchProjectDetail(projectId)
      .then(() => {
        if (!cancelled) setStatus("ready");
      })
      .catch(() => {
        if (!cancelled) setStatus("not-found");
      });
    return () => {
      cancelled = true;
    };
  }, [projectId, fetchProjectDetail]);

  if (!projectId || status === "not-found") {
    return <Navigate to={ROUTE_PATHS.projects} replace />;
  }

  if (status === "loading" || !currentProject || currentProject.id !== projectId) {
    return <div className="py-16 text-center text-sm text-text-secondary">Memuat...</div>;
  }

  return (
    <div className="flex flex-col gap-5">
      <ProjectHeaderCard projectId={projectId} />
      <TabNav items={TABS} />
      <Suspense fallback={<div className="py-16 text-center text-sm text-text-secondary">Memuat...</div>}>
        <Outlet context={{ projectId } satisfies ProjectDetailContext} />
      </Suspense>
    </div>
  );
}
