import { useEffect, useState } from "react";
import { Download, ExternalLink } from "lucide-react";
import { Modal } from "@/shared/components/ui/Modal";
import { Button } from "@/shared/components/ui/Button";
import { Skeleton } from "@/shared/components/ui/Skeleton";
import { LoadError } from "@/shared/components/feedback/LoadError";
import { useQuotationStore } from "@/modules/offers/stores/useQuotationStore";

interface Props {
  title: string;
  description?: string;
  /** The PDF's API address (fetched with the session, shown from a local copy). */
  url: string;
  onClose: () => void;
  /** "Unduh PDF" of a revision; a DRAF preview has none. */
  onDownload?: () => void;
}

/**
 * A PDF inside the app (no pop-up to be blocked): "Pratinjau PDF" (DRAF, makes no revision) or a revision's PDF.
 * The file is fetched with the session and shown from a local copy that is freed when the dialog closes.
 */
export default function PdfPreviewModal({ title, description, url, onClose, onDownload }: Props) {
  const fetchPdf = useQuotationStore((s) => s.fetchPdf);
  const [objectUrl, setObjectUrl] = useState<string | null>(null);
  const [failed, setFailed] = useState(false);
  const [attempt, setAttempt] = useState(0);

  useEffect(() => {
    let cancelled = false;
    let made: string | null = null;
    setFailed(false);
    setObjectUrl(null);
    fetchPdf(url).then(
      (blob) => {
        if (cancelled) return;
        made = URL.createObjectURL(new Blob([blob], { type: "application/pdf" }));
        setObjectUrl(made);
      },
      () => !cancelled && setFailed(true),
    );
    return () => {
      cancelled = true;
      if (made) URL.revokeObjectURL(made);
    };
  }, [fetchPdf, url, attempt]);

  return (
    <Modal
      open
      onClose={onClose}
      size="xl"
      title={title}
      description={description}
      footer={
        <>
          {objectUrl && (
            <Button variant="ghost" icon={<ExternalLink className="h-4 w-4" />} onClick={() => window.open(objectUrl, "_blank", "noopener")}>
              Buka di tab baru
            </Button>
          )}
          {onDownload && (
            <Button variant="secondary" icon={<Download className="h-4 w-4" />} onClick={onDownload}>
              Unduh PDF
            </Button>
          )}
          <Button onClick={onClose}>Tutup</Button>
        </>
      }
    >
      {failed ? (
        <LoadError message="PDF belum bisa dibuat. Periksa koneksi Anda, lalu coba lagi." onRetry={() => setAttempt((n) => n + 1)} />
      ) : objectUrl ? (
        <iframe title={title} src={objectUrl} className="h-[70vh] w-full rounded-md border border-border bg-muted" />
      ) : (
        <div role="status" aria-label="Menyiapkan PDF" className="flex h-[70vh] flex-col gap-3">
          <Skeleton className="h-8 w-1/3" />
          <Skeleton className="h-full w-full" />
        </div>
      )}
    </Modal>
  );
}
