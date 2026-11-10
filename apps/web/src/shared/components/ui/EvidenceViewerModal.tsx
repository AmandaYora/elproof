import { useEffect, useState } from "react";
import axios from "axios";
import { Download, X, FileWarning, RefreshCw, ZoomIn, ZoomOut } from "lucide-react";
import { Badge } from "@/shared/components/ui/Badge";
import { Button } from "@/shared/components/kit/button";
import { Skeleton } from "@/shared/components/kit/skeleton";
import { Dialog, DialogClose, DialogContent, DialogDescription, DialogTitle } from "@/shared/components/kit/dialog";
import { httpClient } from "@/shared/services/http-client";
import { API } from "@/shared/services/api-endpoints";
import { getEvidenceFileKind } from "@/shared/lib/evidence-file";
import { formatDate } from "@/shared/lib/formatters";
import type { EvidenceType } from "@/modules/projects/types";
import { evidenceTypeLabel } from "@/shared/lib/labels";

interface EvidenceViewerModalProps {
  open: boolean;
  onClose: () => void;
  projectId: string;
  evidence: {
    id: string;
    name: string;
    type: EvidenceType;
    documentDate: string | null;
    fileName: string;
  };
  contextLabel?: string;
}

// Shows the real evidence file. The API hands out a direct, two-minute link to
// object storage, so the bytes never pass through this server and the browser
// loads the image or PDF itself. Where a deployment has no direct links (501),
// or the link cannot be had, the file is streamed through the authenticated
// download endpoint instead: that one needs the staff Bearer token (ADR-0004),
// so a bare <img>/<iframe> cannot call it and it is fetched as a blob.
export function EvidenceViewerModal({ open, onClose, projectId, evidence, contextLabel }: EvidenceViewerModalProps) {
  const [fileUrl, setFileUrl] = useState<string | null>(null);
  const [direct, setDirect] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [downloading, setDownloading] = useState(false);
  const [attempt, setAttempt] = useState(0);
  const [zoomed, setZoomed] = useState(false);

  useEffect(() => {
    if (!open) return;
    let objectUrl: string | null = null;
    let cancelled = false;
    setFileUrl(null);
    setDirect(false);
    setError(null);
    setZoomed(false);

    async function load() {
      try {
        const res = await httpClient.get(API.projects.evidenceLink(projectId, evidence.id));
        if (cancelled) return;
        setDirect(true);
        setFileUrl((res.data.data as { url: string }).url);
        return;
      } catch (err) {
        // A client who may not see the file gets the same 404 from the stream
        // endpoint, so falling back is harmless; anything else is worth a retry there.
        if (cancelled) return;
        if (axios.isAxiosError(err) && err.response?.status === 404) {
          setError("Dokumen belum bisa dimuat.");
          return;
        }
      }
      try {
        const res = await httpClient.get(API.projects.evidenceFile(projectId, evidence.id), { responseType: "blob" });
        if (cancelled) return;
        objectUrl = URL.createObjectURL(res.data as Blob);
        setFileUrl(objectUrl);
      } catch {
        if (!cancelled) setError("Dokumen belum bisa dimuat.");
      }
    }
    void load();

    return () => {
      cancelled = true;
      if (objectUrl) URL.revokeObjectURL(objectUrl);
    };
  }, [open, projectId, evidence.id, attempt]);

  // A direct link is only good for two minutes, so the download link is made
  // when the button is pressed, not when the viewer opens.
  async function downloadDirect() {
    setDownloading(true);
    try {
      const res = await httpClient.get(API.projects.evidenceLink(projectId, evidence.id), { params: { disposition: "attachment" } });
      window.location.assign((res.data.data as { url: string }).url);
    } catch {
      setError("Berkas belum bisa diunduh. Coba lagi.");
    } finally {
      setDownloading(false);
    }
  }

  const kind = getEvidenceFileKind(evidence.fileName);

  return (
    <Dialog open={open} onOpenChange={(next) => !next && onClose()}>
      <DialogContent
        showCloseButton={false}
        className="flex h-dvh max-h-dvh w-full max-w-none flex-col gap-0 overflow-hidden rounded-none p-0 sm:h-[88vh] sm:max-h-[88vh] sm:max-w-3xl sm:rounded-xl"
      >
        <div className="flex items-start justify-between gap-3 border-b px-4 py-3 sm:px-6 sm:py-4">
          <div className="min-w-0">
            <DialogTitle className="truncate text-[14.5px] leading-snug font-semibold sm:text-[15px]">{evidence.name}</DialogTitle>
            <DialogDescription render={<div />} className="mt-1 flex flex-wrap items-center gap-x-2 gap-y-1 text-[12px]">
              <Badge tone="neutral">{evidenceTypeLabel(evidence.type)}</Badge>
              <span>{formatDate(evidence.documentDate)}</span>
              {contextLabel && <span className="hidden sm:inline">· {contextLabel}</span>}
            </DialogDescription>
          </div>
          <div className="flex shrink-0 items-center gap-2">
            {fileUrl && kind === "image" && (
              <Button variant="outline" size="sm" onClick={() => setZoomed((z) => !z)} aria-pressed={zoomed}>
                {zoomed ? <ZoomOut /> : <ZoomIn />}
                <span className="hidden sm:inline">{zoomed ? "Sesuaikan layar" : "Ukuran asli"}</span>
                <span className="sr-only sm:hidden">{zoomed ? "Sesuaikan layar" : "Ukuran asli"}</span>
              </Button>
            )}
            {fileUrl && direct && (
              <Button variant="outline" size="sm" onClick={() => void downloadDirect()} disabled={downloading} aria-label="Unduh dokumen">
                <Download />
                <span className="hidden sm:inline">Unduh</span>
              </Button>
            )}
            {fileUrl && !direct && (
              <Button variant="outline" size="sm" nativeButton={false} render={<a href={fileUrl} download={evidence.fileName} />} aria-label="Unduh dokumen">
                <Download />
                <span className="hidden sm:inline">Unduh</span>
              </Button>
            )}
            <DialogClose render={<Button variant="ghost" size="icon-sm" />}>
              <X />
              <span className="sr-only">Tutup</span>
            </DialogClose>
          </div>
        </div>

        <div className="relative flex-1 overflow-hidden bg-surface-muted/50">
          {error ? (
            <div role="alert" className="flex h-full flex-col items-center justify-center gap-3 p-8 text-center">
              <FileWarning className="size-8 text-danger-text" aria-hidden />
              <p className="text-[13px] text-muted-foreground">{error}</p>
              <Button variant="outline" size="sm" onClick={() => setAttempt((n) => n + 1)}>
                <RefreshCw /> Coba lagi
              </Button>
            </div>
          ) : !fileUrl ? (
            <div role="status" aria-label="Memuat dokumen" className="flex h-full items-center justify-center p-8">
              <Skeleton className="h-full max-h-96 w-full max-w-md" />
            </div>
          ) : kind === "pdf" ? (
            <iframe src={fileUrl} title={evidence.name} referrerPolicy="no-referrer" className="h-full w-full border-0" />
          ) : kind === "image" ? (
            <div className={zoomed ? "h-full overflow-auto p-4 sm:p-8" : "flex h-full items-center justify-center p-4 sm:p-8"}>
              <img
                src={fileUrl}
                alt={evidence.name}
                referrerPolicy="no-referrer"
                className={zoomed ? "mx-auto max-w-none rounded-md border border-border-light bg-white p-2 shadow-sm" : "max-h-full max-w-full rounded-md border border-border-light bg-white p-2 object-contain shadow-sm"}
              />
            </div>
          ) : (
            <div className="flex h-full flex-col items-center justify-center gap-2 p-8 text-center text-[13px] text-text-secondary">
              Pratinjau tidak tersedia untuk jenis berkas ini. Gunakan tombol Unduh.
            </div>
          )}
        </div>
      </DialogContent>
    </Dialog>
  );
}
