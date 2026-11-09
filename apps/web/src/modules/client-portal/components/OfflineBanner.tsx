import { WifiOff } from "lucide-react";
import { useOnline } from "@/shared/hooks/useOnline";

/** Says so when the phone has no connection, so a stuck page is not mistaken for a broken portal. */
export function OfflineBanner() {
  const online = useOnline();
  if (online) return null;
  return (
    <div role="status" className="flex items-center justify-center gap-2 bg-warning-soft px-4 py-2 text-center text-[13px] font-medium text-warning-text">
      <WifiOff className="size-4 shrink-0" aria-hidden />
      Tidak ada koneksi internet. Data yang tampil mungkin belum yang terbaru.
    </div>
  );
}
