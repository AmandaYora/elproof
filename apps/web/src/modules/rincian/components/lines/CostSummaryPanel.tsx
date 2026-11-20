import type { ReactNode } from "react";
import { Link } from "react-router-dom";
import { CircleAlert } from "lucide-react";
import { cn } from "@/shared/lib/cn";
import { formatCurrency } from "@/shared/lib/formatters";
import { kindSums } from "@/modules/rincian/lib/editor";
import type { Rincian } from "@/modules/rincian/types";

export interface CostSummaryPanelProps {
  /** What the panel adds up: the latest preview while editing, the saved rincian otherwise. */
  rincian: Rincian | null;
  /** A preview is running: the total reads "Menghitung…". */
  computing?: boolean;
  /** The last preview failed: the last valid total stays, marked "belum diperbarui". */
  stale?: boolean;
  /** Where "Margin belum lengkap (n vendor)" leads (the vendor list filtered to those without a cost). */
  missingCostHref?: string;
  /** Something the screen adds under the numbers (e.g. the payment schedule no longer adding up). */
  children?: ReactNode;
  className?: string;
  id?: string;
}

/** A minus sign that reads as one ("−Rp5.000.000"), not a hyphen. */
export function negative(amount: number): string {
  return `−${formatCurrency(Math.abs(amount))}`;
}

/**
 * "Ringkasan biaya" (DESAIN §5.4): what the rincian comes to per kind of line, PPN when the WO is PKP, the contract
 * value, and, for a person who sees margins, Biaya rekanan and Margin. Every number comes from the server.
 */
export function CostSummaryPanel({ rincian, computing = false, stale = false, missingCostHref, children, className, id }: CostSummaryPanelProps) {
  const sums = kindSums(rincian);
  const totals = rincian?.totals;
  const margin = rincian?.margin;

  return (
    <section id={id} aria-labelledby="ringkasan-biaya-heading" className={cn("rounded-lg border border-border bg-card px-5 py-4", className)}>
      <h3 id="ringkasan-biaya-heading" className="text-[15px] font-semibold text-foreground">
        Ringkasan biaya
      </h3>
      {!totals || !sums ? (
        <p className="mt-3 text-[13px] text-muted-foreground">Nilai kontrak tampil setelah rincian dimuat.</p>
      ) : (
        <>
          <dl className="mt-3 flex flex-col gap-1.5 text-[13.5px]">
            <Row label="Paket" value={formatCurrency(sums.packages)} />
            {sums.extraGuests > 0 && <Row label="Tambahan tamu" value={formatCurrency(sums.extraGuests)} />}
            <Row label="Item" value={formatCurrency(sums.items)} />
            {sums.discount !== 0 && <Row label="Potongan" value={negative(sums.discount)} />}
            {rincian?.isPkp && (
              <>
                <Row label="Subtotal" value={formatCurrency(totals.subtotal)} muted />
                <Row label="PPN" value={formatCurrency(totals.tax)} />
              </>
            )}
          </dl>
          <div className="mt-3 border-t border-border-light pt-3">
            <p className="text-[13px] text-muted-foreground">Nilai kontrak</p>
            <p className="mt-0.5 text-[24px] leading-tight font-semibold tabular-nums text-foreground" data-testid="nilai-kontrak">
              {computing ? <span className="text-[17px] font-medium text-muted-foreground">Menghitung…</span> : formatCurrency(totals.total)}
            </p>
            {stale && !computing && <p className="mt-1 text-[12.5px] text-warning-text">Belum diperbarui. Periksa isian yang ditandai.</p>}
          </div>
          {margin && (
            <div className="mt-3 border-t border-border-light pt-3">
              <dl className="flex flex-col gap-1.5 text-[13.5px]">
                <Row label="Biaya rekanan" value={formatCurrency(margin.supplierCost)} />
                <Row label="Margin" value={formatCurrency(margin.margin)} strong />
              </dl>
              {/* Outside the <dl>: a definition list may only hold term/definition groups. */}
              {margin.incomplete && (
                <div className="mt-2.5 flex items-start gap-1.5 text-[13px] text-warning-text">
                  <CircleAlert aria-hidden className="mt-0.5 size-3.5 shrink-0" />
                  {missingCostHref ? (
                    <Link to={missingCostHref} className="font-medium underline underline-offset-2 hover:no-underline">
                      Margin belum lengkap ({margin.missingCost} vendor)
                    </Link>
                  ) : (
                    <span className="font-medium">Margin belum lengkap ({margin.missingCost} vendor)</span>
                  )}
                </div>
              )}
            </div>
          )}
        </>
      )}
      {children}
    </section>
  );
}

function Row({ label, value, muted = false, strong = false }: { label: string; value: string; muted?: boolean; strong?: boolean }) {
  return (
    <div className="flex items-baseline justify-between gap-3">
      <dt className={cn(muted ? "text-muted-foreground" : "text-foreground")}>{label}</dt>
      <dd className={cn("whitespace-nowrap tabular-nums", strong ? "font-semibold text-foreground" : "text-foreground")}>{value}</dd>
    </div>
  );
}
