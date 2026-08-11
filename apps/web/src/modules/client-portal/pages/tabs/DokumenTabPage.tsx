import { useEffect, useState } from "react";
import { useOutletContext } from "react-router-dom";
import { FileText, FolderOpen, Eye } from "lucide-react";
import { Badge } from "@/shared/components/ui/Badge";
import { EvidenceViewerModal } from "@/shared/components/ui/EvidenceViewerModal";
import { useProjectStore } from "@/modules/projects/stores/useProjectStore";
import type { Evidence } from "@/modules/projects/types";
import { formatDate } from "@/shared/lib/formatters";
import type { ClientPortalContext } from "@/modules/client-portal/layouts/ClientPortalLayout";

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
    <div className="flex flex-col gap-6 sm:gap-8">
      <section>
        <div className="mb-4 flex items-center gap-3 border-b border-border pb-4 sm:mb-6">
          <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-info-soft text-info">
            <FileText className="h-5 w-5 shrink-0" />
          </div>
          <div>
            <h2 className="text-[18px] font-bold text-navy-950 sm:text-[20px]">Dokumen</h2>
            <p className="mt-0.5 text-[13px] text-text-secondary sm:text-[14px]">
              Dokumen umum acara yang telah kami bagikan untuk Anda — rundown, buku acara, dan lainnya.
            </p>
          </div>
        </div>

        {sortedDocuments.length === 0 ? (
          <div className="flex flex-col items-center gap-4 rounded-3xl border border-border bg-white p-10 text-center shadow-sm sm:p-16">
            <div className="flex h-20 w-20 items-center justify-center rounded-full bg-info-soft text-info mb-2">
              <FolderOpen className="h-10 w-10" />
            </div>
            <p className="max-w-md text-[15px] font-bold text-navy-950 sm:text-[16px]">Belum Ada Dokumen</p>
            <p className="max-w-sm text-[14px] text-text-secondary mt-1">
              Dokumen umum acara Anda akan muncul di sini setelah dibagikan oleh tim kami.
            </p>
          </div>
        ) : (
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            {sortedDocuments.map((doc) => (
              <button
                key={doc.id}
                onClick={() => setViewingEvidence(doc)}
                className="flex items-start gap-3 rounded-2xl border border-border bg-white p-4 text-left shadow-sm transition-colors hover:border-navy-200 hover:bg-surface-muted/50"
              >
                <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-info-soft text-info">
                  <FileText className="h-5 w-5" />
                </div>
                <div className="min-w-0 flex-1">
                  <p className="truncate text-[14px] font-semibold text-navy-950">{doc.name}</p>
                  <div className="mt-1 flex flex-wrap items-center gap-1.5 text-[12px] text-text-secondary">
                    <Badge tone="neutral">{doc.type}</Badge>
                    <span>{formatDate(doc.documentDate)}</span>
                  </div>
                  {doc.description && (
                    <p className="mt-1.5 line-clamp-2 text-[12.5px] text-text-secondary">{doc.description}</p>
                  )}
                </div>
                <Eye className="h-4 w-4 shrink-0 text-text-secondary" />
              </button>
            ))}
          </div>
        )}
      </section>

      {viewingEvidence && (
        <EvidenceViewerModal open onClose={() => setViewingEvidence(null)} projectId={projectId} evidence={viewingEvidence} />
      )}
    </div>
  );
}
