import { useCallback, useEffect, useRef, useState } from "react";
import { ImagePlus } from "lucide-react";
import { Card, CardContent } from "@/shared/components/ui/Card";
import { Button } from "@/shared/components/ui/Button";
import { httpClient } from "@/shared/services/http-client";
import { API } from "@/shared/services/api-endpoints";
import { toast } from "@/shared/stores/useToastStore";
import { compressFileForUpload } from "@/shared/lib/image-compression";
import { getApiErrorMessage, getApiFieldErrors } from "@/shared/lib/api-error";
import { usePlatformCan } from "@/modules/platform-admin/lib/capabilities";

const SLOTS = [0, 1, 2];
const ACCEPT = "image/jpeg,image/png";
const MAX_BYTES = 12 * 1024 * 1024;

function SlideSlot({ index, canEdit }: { index: number; canEdit: boolean }) {
  const [src, setSrc] = useState<string | null>(null);
  const [version, setVersion] = useState(0);
  const [uploading, setUploading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const input = useRef<HTMLInputElement>(null);

  useEffect(() => {
    let cancelled = false;
    let objectUrl: string | null = null;
    httpClient
      .get(API.public.loginSlide(index), { responseType: "blob", params: { v: version } })
      .then((res) => {
        if (cancelled) return;
        objectUrl = URL.createObjectURL(res.data as Blob);
        setSrc(objectUrl);
      })
      .catch(() => {
        if (!cancelled) setSrc(null);
      });
    return () => {
      cancelled = true;
      if (objectUrl) URL.revokeObjectURL(objectUrl);
    };
  }, [index, version]);

  async function handleFile(file: File | undefined) {
    if (!file) return;
    setError(null);
    if (file.type !== "image/jpeg" && file.type !== "image/png") {
      setError("Gunakan foto JPG atau PNG.");
      return;
    }
    if (file.size > MAX_BYTES) {
      setError("Foto lebih dari 12 MB. Pilih foto yang lebih kecil.");
      return;
    }
    setUploading(true);
    try {
      const payload = await compressFileForUpload(file);
      await httpClient.put(API.console.loginSlide(index), { mimeType: payload.mimeType, base64Data: payload.base64Data });
      setVersion((v) => v + 1);
      toast.success(`Foto ${index + 1} diganti`);
    } catch (err) {
      setError(getApiFieldErrors(err).file ?? getApiErrorMessage(err, "Foto belum bisa diganti"));
    } finally {
      setUploading(false);
      if (input.current) input.current.value = "";
    }
  }

  return (
    <Card>
      <CardContent className="flex flex-col gap-3">
        <div className="aspect-[4/5] w-full overflow-hidden rounded-md bg-surface-muted">
          {src ? (
            <img src={src} alt={`Foto login ${index + 1}`} className="h-full w-full object-cover" />
          ) : (
            <div className="flex h-full items-center justify-center px-4 text-center text-[12.5px] text-text-secondary">Foto belum ada atau belum bisa dimuat</div>
          )}
        </div>
        <div className="flex items-center justify-between gap-2">
          <span className="text-[13.5px] font-semibold text-text-primary">Foto {index + 1}</span>
          {canEdit && (
            <>
              <input ref={input} type="file" accept={ACCEPT} className="sr-only" onChange={(e) => void handleFile(e.target.files?.[0])} aria-label={`Pilih foto pengganti untuk foto ${index + 1}`} />
              <Button size="sm" variant="secondary" icon={<ImagePlus className="h-3.5 w-3.5" />} disabled={uploading} onClick={() => input.current?.click()}>
                {uploading ? "Mengunggah..." : "Ganti foto"}
              </Button>
            </>
          )}
        </div>
        {error && (
          <p role="alert" className="text-[12.5px] font-medium text-danger">
            {error}
          </p>
        )}
      </CardContent>
    </Card>
  );
}

export default function LoginSlidesPage() {
  const canEdit = usePlatformCan("super-admin");
  const [count, setCount] = useState<number | null>(null);
  const load = useCallback(() => {
    httpClient
      .get(API.public.loginSlides)
      .then((res) => setCount((res.data.data as { count: number }).count))
      .catch(() => setCount(SLOTS.length));
  }, []);
  useEffect(() => {
    load();
  }, [load]);

  return (
    <div className="flex flex-col gap-5">
      <div>
        <h1 className="text-xl font-bold text-text-primary">Tampilan Login</h1>
        <p className="mt-1 max-w-2xl text-[13px] text-text-secondary">
          Tiga foto yang bergantian di halaman masuk. Pilih foto tegak (rasio sekitar 4:5), minimal 1200 piksel di sisi terpanjang. Foto baru langsung dipakai, tetapi browser pengunjung bisa
          menampilkan foto lama sampai 24 jam.
        </p>
      </div>
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
        {SLOTS.slice(0, count ?? SLOTS.length).map((i) => (
          <SlideSlot key={i} index={i} canEdit={canEdit} />
        ))}
      </div>
      {!canEdit && <p className="text-[13px] text-text-secondary">Hanya Super Admin yang dapat mengganti foto.</p>}
    </div>
  );
}
