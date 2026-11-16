import { useRef, useState } from "react";
import { ImagePlus, Trash2 } from "lucide-react";
import { Card, CardContent, CardHeader } from "@/shared/components/ui/Card";
import { Button } from "@/shared/components/ui/Button";
import { useCan } from "@/shared/stores/usePermissionStore";
import { useAuthedImage } from "@/shared/hooks/useAuthedImage";
import { compressFileForUpload } from "@/shared/lib/image-compression";
import { getApiErrorMessage } from "@/shared/lib/api-error";
import { formatDate } from "@/shared/lib/formatters";
import { toast } from "@/shared/stores/useToastStore";
import { useProjectStore } from "@/modules/projects/stores/useProjectStore";

const ACCEPTED = ["image/jpeg", "image/png", "image/webp"];
const MAX_BYTES = 6 * 1024 * 1024;

/** Decoded size of a base64 string, without decoding it. */
function base64Bytes(b64: string): number {
  const padding = b64.endsWith("==") ? 2 : b64.endsWith("=") ? 1 : 0;
  return Math.floor((b64.length * 3) / 4) - padding;
}

/**
 * "Foto sampul portal" (ADR-0047): the photo behind the hero of the couple's Client Portal. The preview is a small copy
 * of that hero (the photo under a navy band carrying the names), so the WO sees what the couple will see; without a
 * photo it is the plain navy hero the portal shows today.
 */
export function ProjectCoverCard({ projectId }: { projectId: string }) {
  const project = useProjectStore((s) => s.currentProject);
  const saveCover = useProjectStore((s) => s.saveCover);
  const removeCover = useProjectStore((s) => s.removeCover);
  const canEdit = useCan("project.edit");
  const inputRef = useRef<HTMLInputElement>(null);
  const [busy, setBusy] = useState<"upload" | "remove" | null>(null);
  const [error, setError] = useState<string | null>(null);
  const image = useAuthedImage(project?.coverUrl ?? null);
  if (!project) return null;

  const hasCover = project.coverUrl !== null;
  const couple = [project.brideName, project.groomName].filter(Boolean).join(" & ") || project.name;

  async function onPick(file: File | undefined) {
    if (!file) return;
    setError(null);
    if (!ACCEPTED.includes(file.type)) {
      setError("Pilih foto JPG, PNG, atau WebP.");
      return;
    }
    setBusy("upload");
    try {
      const payload = await compressFileForUpload(file);
      if (base64Bytes(payload.base64Data) > MAX_BYTES) {
        setError("Ukuran foto maksimal 6 MB. Pilih foto yang lebih kecil.");
        return;
      }
      await saveCover(projectId, payload);
      toast.success(hasCover ? "Foto sampul diganti" : "Foto sampul disimpan");
    } catch (err) {
      setError(getApiErrorMessage(err, "Foto sampul belum tersimpan. Periksa koneksi lalu coba lagi."));
    } finally {
      setBusy(null);
    }
  }

  async function onRemove() {
    setError(null);
    setBusy("remove");
    try {
      await removeCover(projectId);
      toast.success("Foto sampul dihapus");
    } catch (err) {
      setError(getApiErrorMessage(err, "Foto sampul belum terhapus. Periksa koneksi lalu coba lagi."));
    } finally {
      setBusy(null);
    }
  }

  return (
    <Card role="region" aria-label="Foto sampul portal">
      <CardHeader title="Foto sampul portal" subtitle="Tampil di Portal Klien sebagai latar hero." />
      <CardContent className="flex flex-col gap-3">
        <figure
          aria-label={hasCover ? `Pratinjau hero portal dengan foto sampul ${couple}` : `Pratinjau hero portal ${couple}, tanpa foto sampul`}
          className="relative isolate flex aspect-[16/10] flex-col justify-end overflow-hidden rounded-lg bg-primary text-primary-foreground"
        >
          {image.status === "ready" && <img src={image.src} alt="" className="absolute inset-0 -z-10 size-full object-cover" />}
          {hasCover && image.status !== "ready" && (
            <span aria-hidden className="absolute inset-0 -z-10 animate-pulse bg-primary-hover motion-reduce:animate-none" />
          )}
          {/* Navy only under the text, as on the sign-in screen: the photo stays a photo. */}
          {hasCover && <span aria-hidden className="absolute inset-x-0 bottom-0 -z-10 h-3/5 bg-linear-to-t from-primary via-primary/75 to-transparent" />}
          <figcaption className="px-4 pb-3.5">
            <span className="block text-[11.5px] font-medium text-on-primary-muted">{formatDate(project.eventDate)}</span>
            <span className="block truncate text-[17px] leading-tight font-extrabold tracking-[-0.02em]">{couple}</span>
            <span aria-hidden className="mt-2 block h-px w-12 bg-seal-on-primary" />
          </figcaption>
        </figure>
        {image.status === "error" && <p className="text-[12.5px] text-text-secondary">Pratinjau foto belum bisa dimuat.</p>}
        {error && (
          <p role="alert" className="text-[12.5px] font-medium text-danger-text">
            {error}
          </p>
        )}
        {canEdit ? (
          <div className="flex flex-wrap items-center gap-2">
            <input
              ref={inputRef}
              type="file"
              accept={ACCEPTED.join(",")}
              className="sr-only"
              tabIndex={-1}
              aria-hidden
              onChange={(e) => {
                void onPick(e.target.files?.[0]);
                e.target.value = "";
              }}
            />
            <Button size="sm" variant="secondary" icon={<ImagePlus className="h-3.5 w-3.5" />} disabled={busy !== null} onClick={() => inputRef.current?.click()}>
              {busy === "upload" ? "Mengunggah..." : hasCover ? "Ganti foto" : "Unggah foto"}
            </Button>
            {hasCover && (
              <Button size="sm" variant="ghost" icon={<Trash2 className="h-3.5 w-3.5" />} disabled={busy !== null} onClick={() => void onRemove()}>
                {busy === "remove" ? "Menghapus..." : "Hapus foto"}
              </Button>
            )}
            <p className="w-full text-[12px] text-text-secondary">JPG, PNG, atau WebP, maksimal 6 MB. Foto mendatar paling pas.</p>
          </div>
        ) : (
          !hasCover && <p className="text-[12.5px] text-text-secondary">Belum ada foto sampul. Portal memakai latar navy.</p>
        )}
      </CardContent>
    </Card>
  );
}
