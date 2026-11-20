import { useEffect, useId, useState } from "react";
import { Input } from "@/shared/components/ui/Input";
import { formatCurrency } from "@/shared/lib/formatters";
import { cn } from "@/shared/lib/cn";
import { usePackageStore, type PreviewRequest } from "@/modules/packages/stores/usePackageStore";
import type { PackagePreview } from "@/modules/packages/types";

interface PackagePreviewCardProps {
  name: string;
  /** The rule as typed; null while it is incomplete (the form shows why). */
  rule: Omit<PreviewRequest, "guests"> | null;
  /** The descriptions of "Isi paket", in order: what the client reads under the package. */
  items: string[];
  /** Where the guest count starts (a little over the included number). */
  initialGuests: number;
}

/**
 * The "Contoh hitung" box (DESAIN.md §6.6, §10 A3): the extra-guest rule worked through for a number of guests the
 * person picks, priced by the server (POST /packages/preview, the same rule quotations use), and the package as the
 * client will read it. Recomputed 500 ms after the last change; a failed recompute keeps the last good result.
 */
export function PackagePreviewCard({ name, rule, items, initialGuests }: PackagePreviewCardProps) {
  const previewPackage = usePackageStore((s) => s.previewPackage);
  const guestsId = useId();
  const [guestsText, setGuestsText] = useState(String(initialGuests));
  const [result, setResult] = useState<PackagePreview | null>(null);
  const [state, setState] = useState<"idle" | "busy" | "stale">("idle");

  // A new included number (or step) moves the example along, until the person types their own.
  const [ownGuests, setOwnGuests] = useState(false);
  useEffect(() => {
    if (!ownGuests) setGuestsText(String(initialGuests));
  }, [initialGuests, ownGuests]);

  const guests = Number(guestsText) || 0;
  const key = rule ? JSON.stringify({ ...rule, guests }) : null;

  useEffect(() => {
    if (!rule || rule.includedGuests === null) {
      setResult(null);
      setState("idle");
      return;
    }
    const controller = new AbortController();
    setState("busy");
    const timer = setTimeout(() => {
      previewPackage({ ...rule, guests }, controller.signal)
        .then((r) => {
          setResult(r);
          setState("idle");
        })
        .catch(() => {
          if (!controller.signal.aborted) setState("stale");
        });
    }, 500);
    return () => {
      clearTimeout(timer);
      controller.abort();
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [key]);

  const price = rule?.price ?? 0;
  const fixedPrice = rule !== null && rule.includedGuests === null;

  return (
    <section aria-labelledby={`${guestsId}-title`} className="flex flex-col overflow-hidden rounded-xl border border-border bg-card">
      <div className="flex flex-col gap-3 p-4">
        <h2 id={`${guestsId}-title`} className="text-[15px] font-semibold text-foreground">
          Contoh hitung
        </h2>

        {rule === null ? (
          <p className="text-[13px] text-muted-foreground">Lengkapi harga dan aturan tamu untuk melihat contoh hitungnya.</p>
        ) : fixedPrice ? (
          <p className="text-[13px] text-muted-foreground">
            Harga tetap <strong className="font-semibold text-foreground">{formatCurrency(price)}</strong>, berapa pun jumlah tamunya.
          </p>
        ) : (
          <>
            <label htmlFor={guestsId} className="flex items-center gap-2 text-[13px] text-muted-foreground">
              Untuk
              <Input
                id={guestsId}
                inputMode="numeric"
                className="w-24 text-right tabular-nums"
                value={guestsText}
                onChange={(e) => {
                  setOwnGuests(true);
                  setGuestsText(e.target.value.replace(/\D/g, "").slice(0, 6));
                }}
              />
              tamu
            </label>

            {/* The sum as it is worked out on paper: each line is one step of the rule. */}
            <dl aria-live="polite" aria-busy={state === "busy"} className={cn("flex flex-col gap-1 font-medium tabular-nums transition-opacity", state === "busy" && "opacity-60")}>
              <div className="flex items-baseline justify-between gap-3 text-[13.5px]">
                <dt className="text-muted-foreground">Harga paket</dt>
                <dd className="text-foreground">{formatCurrency(price)}</dd>
              </div>
              {result && result.extra.steps > 0 ? (
                <div className="flex items-baseline justify-between gap-3 text-[13.5px]">
                  <dt className="text-muted-foreground">
                    + {result.extra.steps} × {formatCurrency(rule.extraGuestStepPrice ?? 0)}
                    <span className="block text-[12px] font-normal">
                      {result.extra.over.toLocaleString("id-ID")} tamu di atas {rule.includedGuests?.toLocaleString("id-ID")}
                      {rule.extraGuestStep && rule.extraGuestStep > 1 ? `, per ${rule.extraGuestStep.toLocaleString("id-ID")} tamu` : ""}
                    </span>
                  </dt>
                  <dd className="text-foreground">{formatCurrency(result.extra.amount)}</dd>
                </div>
              ) : (
                result && (
                  <p className="text-[12.5px] font-normal text-muted-foreground">
                    {guests.toLocaleString("id-ID")} tamu masih termasuk paket: tidak ada biaya tambahan.
                  </p>
                )
              )}
              <div className="mt-1 flex items-baseline justify-between gap-3 border-t border-border pt-2">
                <dt className="text-[13.5px] text-foreground">Total</dt>
                <dd className="text-[20px] font-semibold tracking-tight text-foreground">{result ? formatCurrency(result.total) : "Menghitung..."}</dd>
              </div>
            </dl>
            {state === "stale" && <p className="text-[12.5px] text-warning-text">Contoh belum diperbarui. Periksa koneksi; angka di atas hasil hitungan terakhir.</p>}
          </>
        )}
      </div>

      {/* What the client reads in a quotation: the package and what it contains, never costs or vendors. */}
      <div className="flex flex-col gap-2 border-t border-border bg-muted/40 p-4">
        <p className="text-[12.5px] text-muted-foreground">Seperti ini klien melihatnya</p>
        <div className="rounded-lg border border-border bg-card p-3">
          <div className="flex items-baseline justify-between gap-3">
            <p className="min-w-0 break-words text-[13.5px] font-semibold text-foreground">{name.trim() || "Nama paket"}</p>
            <p className="shrink-0 text-[13.5px] font-semibold tabular-nums text-foreground">{formatCurrency(price)}</p>
          </div>
          {items.length > 0 ? (
            <ul className="mt-2 flex flex-col gap-1 text-[13px] text-muted-foreground">
              {items.map((text, i) => (
                <li key={i} className="flex gap-2">
                  <span aria-hidden>•</span>
                  <span className="min-w-0 break-words">{text}</span>
                </li>
              ))}
            </ul>
          ) : (
            <p className="mt-2 text-[12.5px] text-muted-foreground">Isi paket yang Anda tambahkan tampil di sini.</p>
          )}
        </div>
      </div>
    </section>
  );
}
