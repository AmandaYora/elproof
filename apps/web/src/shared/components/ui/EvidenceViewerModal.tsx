import { useEffect, useState } from "react";
import axios from "axios";
import { Download, X, FileWarning } from "lucide-react";
import { Badge } from "@/shared/components/ui/Badge";
import { httpClient } from "@/shared/services/http-client";
import { API } from "@/shared/services/api-endpoints";
import { getEvidenceFileKind } from "@/shared/lib/evidence-file";
import { formatDate } from "@/shared/lib/formatters";
import type { EvidenceType } from "@/modules/projects/types";

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

  useEffect(() => {
    if (!open) return;
    let objectUrl: string | null = null;
    let cancelled = false;
    setFileUrl(null);
    setDirect(false);
    setError(null);

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
          setError("Gagal memuat berkas evidence.");
          return;
        }
      }
      try {
        const res = await httpClient.get(API.projects.evidenceFile(projectId, evidence.id), { responseType: "blob" });
        if (cancelled) return;
        objectUrl = URL.createObjectURL(res.data as Blob);
        setFileUrl(objectUrl);
      } catch {
        if (!cancelled) setError("Gagal memuat berkas evidence.");
      }
    }
    void load();

    return () => {
      cancelled = true;
      if (objectUrl) URL.revokeObjectURL(objectUrl);
    };
  }, [open, projectId, evidence.id]);

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

  useEffect(() => {
    if (!open) return;
    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    document.addEventListener("keydown", onKeyDown);
    return () => document.removeEventListener("keydown", onKeyDown);
  }, [open, onClose]);

  if (!open) return null;

  const kind = getEvidenceFileKind(evidence.fileName);

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-navy-950/60 p-0 sm:p-4 md:p-8">
      <div
        role="dialog"
        aria-modal="true"
        aria-label={evidence.name}
        className="flex h-full w-full flex-col overflow-hidden bg-surface shadow-xl sm:h-[88vh] sm:max-w-3xl sm:rounded-xl"
      >
        <div className="flex items-start justify-between gap-3 border-b border-border-light px-4 py-3 sm:px-6 sm:py-4">
          <div className="min-w-0">
            <p className="truncate text-[14.5px] font-semibold text-text-primary sm:text-[15px]">{evidence.name}</p>
            <div className="mt-1 flex flex-wrap items-center gap-x-2 gap-y-1 text-[12px] text-text-secondary">
              <Badge tone="neutral">{evidence.type}</Badge>
              <span>{formatDate(evidence.documentDate)}</span>
              {contextLabel && <span className="hidden sm:inline">· {contextLabel}</span>}
            </div>
          </div>
          <div className="flex shrink-0 items-center gap-2">
            {fileUrl && direct && (
              <button
                type="button"
                onClick={() => void downloadDirect()}
                disabled={downloading}
                className="flex items-center gap-1.5 rounded-md border border-border px-2.5 py-1.5 text-[12.5px] font-semibold text-navy-900 transition-colors hover:bg-surface-muted disabled:opacity-60"
                aria-label="Unduh dokumen"
              >
                <Download className="h-3.5 w-3.5" />
                <span className="hidden sm:inline">Unduh</span>
              </button>
            )}
            {fileUrl && !direct && (
              <a
                href={fileUrl}
                download={evidence.fileName}
                className="flex items-center gap-1.5 rounded-md border border-border px-2.5 py-1.5 text-[12.5px] font-semibold text-navy-900 transition-colors hover:bg-surface-muted"
                aria-label="Unduh dokumen"
              >
                <Download className="h-3.5 w-3.5" />
                <span className="hidden sm:inline">Unduh</span>
              </a>
            )}
            <button
              onClick={onClose}
              aria-label="Tutup"
              className="rounded-md p-1.5 text-text-secondary hover:bg-surface-muted"
            >
              <X className="h-4 w-4" />
            </button>
          </div>
        </div>

        <div className="relative flex-1 overflow-hidden bg-surface-muted/50">
          {error ? (
            <div className="flex h-full flex-col items-center justify-center gap-2 p-8 text-center">
              <FileWarning className="h-8 w-8 text-danger" />
              <p className="text-[13px] text-text-secondary">{error}</p>
            </div>
          ) : !fileUrl ? (
            <div className="flex h-full items-center justify-center text-[13px] text-text-secondary">Memuat berkas...</div>
          ) : kind === "pdf" ? (
            <iframe src={fileUrl} title={evidence.name} referrerPolicy="no-referrer" className="h-full w-full border-0" />
          ) : kind === "image" ? (
            <div className="flex h-full items-center justify-center p-4 sm:p-8">
              <img
                src={fileUrl}
                alt={evidence.name}
                referrerPolicy="no-referrer"
                className="max-h-full max-w-full rounded-md border border-border-light bg-white p-2 object-contain shadow-sm"
              />
            </div>
          ) : (
            <div className="flex h-full flex-col items-center justify-center gap-2 p-8 text-center text-[13px] text-text-secondary">
              Pratinjau tidak tersedia untuk jenis berkas ini — gunakan tombol Unduh.
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
