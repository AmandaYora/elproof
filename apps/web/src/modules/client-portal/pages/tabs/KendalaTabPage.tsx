import { useOutletContext } from "react-router-dom";
import { CheckCircle2 } from "lucide-react";
import { LoadError } from "@/shared/components/feedback/LoadError";
import { SkeletonRows } from "@/shared/components/ui/Skeleton";
import { IssueCard } from "@/modules/client-portal/components/IssueCard";
import { useProofViewer } from "@/modules/client-portal/components/useProofViewer";
import { useLoad } from "@/modules/client-portal/lib/useLoad";
import { useProjectStore } from "@/modules/projects/stores/useProjectStore";
import type { ClientPortalContext } from "@/modules/client-portal/layouts/ClientPortalLayout";

/** Kendala: open ones first, each with what happened, what we are doing, and when to expect news. One call. */
export default function KendalaTabPage() {
  const { projectId } = useOutletContext<ClientPortalContext>();
  const issues = useProjectStore((s) => s.issues);
  const fetchIssues = useProjectStore((s) => s.fetchIssues);
  const viewer = useProofViewer(projectId);
  const { status, retry } = useLoad(() => fetchIssues(projectId), [projectId, fetchIssues]);

  const sorted = [...issues].sort((a, b) => {
    const aOpen = a.status !== "Resolved" && a.status !== "Closed";
    const bOpen = b.status !== "Resolved" && b.status !== "Closed";
    if (aOpen !== bOpen) return aOpen ? -1 : 1;
    return a.foundDate < b.foundDate ? 1 : -1;
  });

  return (
    <div className="mx-auto flex w-full max-w-3xl flex-col gap-6">
      <header>
        <h1 className="text-xl font-semibold text-foreground">Kendala</h1>
        <p className="mt-1.5 max-w-prose text-[14px] leading-relaxed text-muted-foreground">
          Anda berhak tahu setiap kendala dan penanganannya, sekecil apa pun.
        </p>
      </header>

      {status === "loading" && <div className="rounded-2xl border border-border bg-card p-5"><SkeletonRows rows={3} label="Memuat kendala" /></div>}
      {status === "error" && <LoadError message="Kendala belum bisa dimuat. Periksa koneksi, lalu coba lagi." onRetry={retry} />}

      {status === "ready" && sorted.length === 0 && (
        <div className="flex flex-col items-center gap-3 rounded-2xl border border-border bg-card p-10 text-center">
          <span aria-hidden className="flex size-14 items-center justify-center rounded-full bg-success-soft text-success-text">
            <CheckCircle2 className="size-7" />
          </span>
          <p className="text-[15px] font-semibold text-foreground">Semuanya berjalan lancar</p>
          <p className="max-w-sm text-[14px] text-muted-foreground">Tidak ada kendala yang tercatat saat ini. Tim kami terus memantau persiapan pernikahan Anda.</p>
        </div>
      )}

      {status === "ready" && sorted.length > 0 && (
        <div className="flex flex-col gap-4">
          {sorted.map((issue) => (
            <IssueCard key={issue.id} issue={issue} onOpenProof={viewer.open} />
          ))}
        </div>
      )}
      {viewer.element}
    </div>
  );
}
