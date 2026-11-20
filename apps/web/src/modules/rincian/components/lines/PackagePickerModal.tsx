import { useEffect, useState } from "react";
import { Check, Search } from "lucide-react";
import { Modal } from "@/shared/components/ui/Modal";
import { Button } from "@/shared/components/ui/Button";
import { Input } from "@/shared/components/ui/Input";
import { SkeletonRows } from "@/shared/components/ui/Skeleton";
import { LoadError } from "@/shared/components/feedback/LoadError";
import { cn } from "@/shared/lib/cn";
import { formatCurrency } from "@/shared/lib/formatters";
import type { PackageChoice } from "@/modules/rincian/types";

export interface PackagePickerModalProps {
  /** Active catalog packages matching the search ("" = the first page). */
  search: (q: string) => Promise<PackageChoice[]>;
  onPick: (pkg: PackageChoice) => void;
  onClose: () => void;
}

/** "Tambah › Paket": a package from the catalog, found by name. Its rule and isi are copied when it is added. */
export default function PackagePickerModal({ search, onPick, onClose }: PackagePickerModalProps) {
  const [q, setQ] = useState("");
  const [list, setList] = useState<PackageChoice[] | null>(null);
  const [failed, setFailed] = useState(false);
  const [attempt, setAttempt] = useState(0);
  const [chosen, setChosen] = useState<PackageChoice | null>(null);

  useEffect(() => {
    let alive = true;
    setFailed(false);
    const t = setTimeout(() => {
      search(q.trim())
        .then((found) => alive && setList(found))
        .catch(() => alive && setFailed(true));
    }, 250);
    return () => {
      alive = false;
      clearTimeout(t);
    };
  }, [q, search, attempt]);

  return (
    <Modal
      open
      onClose={onClose}
      title="Tambah paket"
      description="Harga, tamu yang termasuk, dan isi paket disalin dari katalog. Perubahan katalog nanti tidak mengubah salinan ini."
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>
            Batal
          </Button>
          <Button disabled={!chosen} onClick={() => chosen && onPick(chosen)}>
            Tambah paket
          </Button>
        </>
      }
    >
      <div className="flex flex-col gap-3">
        <label className="relative block">
          <span className="sr-only">Cari paket</span>
          <Search aria-hidden className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input value={q} onChange={(e) => setQ(e.target.value)} placeholder="Cari nama paket" className="pl-9" />
        </label>
        {failed ? (
          <LoadError message="Katalog paket belum bisa dimuat. Periksa koneksi Anda, lalu coba lagi." onRetry={() => setAttempt((n) => n + 1)} />
        ) : list === null ? (
          <SkeletonRows rows={3} label="Memuat katalog paket" />
        ) : list.length === 0 ? (
          <p className="rounded-md bg-muted px-3.5 py-3 text-[13px] text-muted-foreground">
            {q.trim() ? `Tidak ada paket aktif bernama "${q.trim()}".` : "Belum ada paket aktif di katalog. Buat paket dulu di menu Paket."}
          </p>
        ) : (
          <ul role="listbox" aria-label="Paket katalog" className="flex max-h-80 flex-col gap-1.5 overflow-y-auto">
            {list.map((p) => {
              const selected = chosen?.id === p.id;
              return (
                <li key={p.id} role="option" aria-selected={selected}>
                  <button
                    type="button"
                    onClick={() => setChosen(p)}
                    onDoubleClick={() => onPick(p)}
                    className={cn(
                      "flex w-full items-start justify-between gap-3 rounded-md border px-3.5 py-2.5 text-left transition-colors",
                      selected ? "border-primary bg-accent" : "border-border hover:bg-muted",
                    )}
                  >
                    <span className="min-w-0">
                      <span className="block text-[13.5px] font-semibold text-foreground">{p.name}</span>
                      <span className="mt-0.5 block text-[12.5px] text-muted-foreground">
                        {[
                          p.eventKindNames.length > 0 ? p.eventKindNames.join(", ") : "",
                          p.includedGuests > 0 ? `${p.includedGuests.toLocaleString("id-ID")} tamu` : "",
                          `${p.componentCount} isi`,
                        ]
                          .filter(Boolean)
                          .join(" · ")}
                      </span>
                    </span>
                    <span className="flex shrink-0 items-center gap-2 text-[13.5px] font-semibold tabular-nums text-foreground">
                      {formatCurrency(p.price)}
                      {selected && <Check aria-hidden className="size-4 text-primary" />}
                    </span>
                  </button>
                </li>
              );
            })}
          </ul>
        )}
      </div>
    </Modal>
  );
}
