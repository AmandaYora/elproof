import { useState } from "react";
import { FileText, Eye } from "lucide-react";
import { Modal } from "@/shared/components/ui/Modal";
import { Badge } from "@/shared/components/ui/Badge";
import { IconActionButton } from "@/shared/components/ui/IconActionButton";
import { EmptyState } from "@/shared/components/feedback/EmptyState";
import { EvidenceViewerModal } from "@/shared/components/ui/EvidenceViewerModal";
import { formatDate } from "@/shared/lib/formatters";
import type { Evidence } from "@/modules/projects/types";

interface EvidenceListModalProps {
  open: boolean;
  onClose: () => void;
  projectId: string;
  title: string;
  items: Evidence[];
}

// Domain-agnostic list-then-view picker for a small set of evidence rows
// tied to one entity (currently: one payment) -- deliberately not named
// after "payment" so it can be reused for any other entity that has
// evidence but no way to view it yet (see PLAN.md "Lihat Bukti"). Renders
// <Modal> and <EvidenceViewerModal> as siblings (never one inside the
// other's `children`) -- verified against Modal.tsx's own implementation:
// neither introduces a transform/filter/opacity that would create a new
// containing block, so both stay correctly stacked (same z-50, later DOM
// order wins) regardless of nesting.
export function EvidenceListModal({ open, onClose, projectId, title, items }: EvidenceListModalProps) {
  const [viewingEvidence, setViewingEvidence] = useState<Evidence | null>(null);

  return (
    <>
      <Modal open={open} onClose={onClose} title={title}>
        {items.length === 0 ? (
          <EmptyState
            icon={<FileText className="h-8 w-8" />}
            title="Belum ada bukti"
            description="Belum ada berkas evidence yang terlampir pada pembayaran ini."
          />
        ) : (
          <div className="flex flex-col gap-2">
            {items.map((item) => (
              <div
                key={item.id}
                className="flex items-center gap-3 rounded-md border border-border px-3.5 py-2.5"
              >
                <FileText className="h-4 w-4 shrink-0 text-text-secondary" />
                <div className="min-w-0 flex-1">
                  <p className="truncate text-[13px] font-medium text-text-primary">{item.name}</p>
                  <div className="mt-0.5 flex items-center gap-2 text-[12px] text-text-secondary">
                    <Badge tone="neutral">{item.type}</Badge>
                    <span>{formatDate(item.documentDate)}</span>
                  </div>
                </div>
                <IconActionButton icon={Eye} label="Lihat" tone="info" onClick={() => setViewingEvidence(item)} />
              </div>
            ))}
          </div>
        )}
      </Modal>

      {viewingEvidence && (
        <EvidenceViewerModal
          open
          onClose={() => setViewingEvidence(null)}
          projectId={projectId}
          evidence={viewingEvidence}
        />
      )}
    </>
  );
}
