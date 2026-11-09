import { Button } from "@/shared/components/ui/Button";

/**
 * Shown when data could not be loaded (network or server trouble), so it is
 * never mistaken for "there is nothing here". Offers a retry.
 */
export function LoadError({ message = "Data belum bisa dimuat. Periksa koneksi Anda, lalu coba lagi.", onRetry }: { message?: string; onRetry: () => void }) {
  return (
    <div role="alert" className="flex flex-col items-center gap-3 rounded-lg border border-danger/30 bg-danger-soft px-6 py-10 text-center">
      <p className="max-w-sm text-[13.5px] font-medium text-danger-text">{message}</p>
      <Button size="sm" variant="secondary" onClick={onRetry}>Coba lagi</Button>
    </div>
  );
}
