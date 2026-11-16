import { lazy, Suspense, useEffect, useState } from "react";
import { useOutletContext } from "react-router-dom";
import { FileText, FolderOpen, Eye } from "lucide-react";
import { Badge } from "@/shared/components/ui/Badge";
import { useProjectStore } from "@/modules/projects/stores/useProjectStore";
import type { Evidence } from "@/modules/projects/types";
import { formatDate } from "@/shared/lib/formatters";
import type { ClientPortalContext } from "@/modules/client-portal/layouts/ClientPortalLayout";
import { evidenceTypeLabel } from "@/shared/lib/labels";
import { PageIntro } from "@/modules/client-portal/components/PageIntro";

// The viewer is only needed once someone opens a document.
const EvidenceViewerModal = lazy(() => import("@/shared/components/ui/EvidenceViewerModal").then((m) => ({ default: m.EvidenceViewerModal })));

// Client Portal's own "Dokumen" tab — general project documents (rundown,
// buku acara, teks juru bicara, dll) with no other specific vendor/venue/
// timeline context. Backed by GET /projects/{id}/documents, which always
// only returns related_kind="general" AND is_client_visible=true documents
// (PLAN.md) — a document only ever shows up here after staff has explicitly
// opted it in from the WO Console's own Dokumen & Evidence section.
export default function DokumenTabPage() {
  const { projectId } = useOutletContext<ClientPortalContext>();
  const documents = useProjectStore((s) => s.documents);
  const fetchDocuments = useProjectStore((s) => s.fetchDocuments);
  const [viewingEvidence, setViewingEvidence] = useState<Evidence | null>(null);

  useEffect(() => {
    void fetchDocuments(projectId);
  }, [projectId, fetchDocuments]);

  const sortedDocuments = [...documents].sort((a, b) => (a.uploadedAt < b.uploadedAt ? 1 : -1));

  return (
    <div className="mx-auto flex w-full max-w-3xl flex-col gap-6">
      <PageIntro title="Dokumen">Dokumen acara yang kami bagikan untuk Anda: rundown, buku acara, dan lainnya.</PageIntro>

      {sortedDocuments.length === 0 ? (
        <div className="flex flex-col items-center gap-3 rounded-2xl border border-dashed border-border bg-card p-10 text-center">
          <FolderOpen className="size-8 text-muted-foreground" aria-hidden />
          <p className="text-base font-semibold text-foreground">Belum ada dokumen</p>
          <p className="max-w-sm text-[14px] text-muted-foreground">Dokumen acara Anda akan muncul di sini setelah dibagikan oleh tim kami.</p>
        </div>
      ) : (
        <ul className="overflow-hidden rounded-2xl border border-border bg-card">
          {sortedDocuments.map((doc) => (
            <li key={doc.id} className="border-b border-border-light last:border-b-0">
              <button
                type="button"
                onClick={() => setViewingEvidence(doc)}
                className="flex min-h-16 w-full items-start gap-3 px-4 py-3.5 text-left hover:bg-muted focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-solid focus-visible:outline-ring sm:px-5"
              >
                <FileText className="mt-0.5 size-5 shrink-0 text-primary" aria-hidden />
                <span className="min-w-0 flex-1">
                  <span className="block truncate text-[14px] font-semibold text-foreground">{doc.name}</span>
                  <span className="mt-1 flex flex-wrap items-center gap-1.5 text-[13px] text-muted-foreground">
                    <Badge tone="neutral" dot={false}>{evidenceTypeLabel(doc.type)}</Badge>
                    <span>{formatDate(doc.documentDate)}</span>
                  </span>
                  {doc.description && <span className="mt-1.5 line-clamp-2 block text-[13px] text-muted-foreground">{doc.description}</span>}
                </span>
                <Eye className="mt-0.5 size-4 shrink-0 text-muted-foreground" aria-hidden />
              </button>
            </li>
          ))}
        </ul>
      )}

      {viewingEvidence && (
        <Suspense fallback={null}>
          <EvidenceViewerModal open onClose={() => setViewingEvidence(null)} projectId={projectId} evidence={viewingEvidence} />
        </Suspense>
      )}
    </div>
  );
}
