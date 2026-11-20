import { Link } from "react-router-dom";
import { ChevronRight } from "lucide-react";
import { Badge } from "@/shared/components/ui/Badge";
import { TERM_STATUS_LABELS } from "@/shared/lib/labels";
import { formatCurrency, formatDate } from "@/shared/lib/formatters";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import { daysUntil } from "@/modules/projects/lib/dates";
import type { NextSupplierBill } from "@/modules/client-portal/types";

/**
 * Beranda's "Tagihan vendor berikutnya" (Paket WO Tahap 4): the next bill of a vendor the couple pays directly, by the
 * vendor's real name, outside the contract above it. Amber when late (red stays for "Yang perlu Anda lakukan"). The
 * link opens Pembayaran › Tagihan vendor, where the proof is sent.
 */
export function NextVendorBillCard({ bill }: { bill: NextSupplierBill }) {
  const days = daysUntil(bill.dueDate);
  const late = bill.status === "overdue";
  const toPay = Math.max(bill.remaining - bill.pending, 0);

  return (
    <section aria-labelledby="vendor-bill-heading" className="rounded-2xl border border-border bg-card p-5 sm:p-6">
      <h2 id="vendor-bill-heading" className="text-base font-semibold text-foreground">Tagihan vendor berikutnya</h2>
      <p className="mt-1 text-[13px] text-muted-foreground">Anda bayar langsung ke vendor, di luar nilai kontrak.</p>

      <p className="mt-3 text-[14px] font-semibold text-foreground">
        {bill.supplierName} <span className="font-normal text-muted-foreground">· {bill.label}</span>
      </p>
      <p className="text-[26px] leading-tight font-extrabold tracking-[-0.02em] text-foreground tabular-nums">{formatCurrency(toPay)}</p>
      <div className="mt-1 flex flex-wrap items-center gap-x-3 gap-y-1.5 text-[14px] text-muted-foreground">
        <span>
          Jatuh tempo {formatDate(bill.dueDate)}
          {!late && days >= 0 && ` · ${days === 0 ? "hari ini" : `${days} hari lagi`}`}
        </span>
        {(late || bill.status === "partial") && <Badge tone="warning">{TERM_STATUS_LABELS.client[bill.status]}</Badge>}
      </div>
      {bill.pending > 0 && (
        <p className="mt-2 text-[13px] text-muted-foreground">
          {formatCurrency(bill.pending)} yang Anda kirim sedang diperiksa wedding organizer Anda.
        </p>
      )}

      <Link
        to={`${ROUTE_PATHS.portal("pembayaran")}#tagihan-vendor`}
        className="mt-4 inline-flex min-h-11 items-center gap-1 rounded-md text-[14px] font-semibold text-primary hover:underline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-solid focus-visible:outline-ring"
      >
        Lihat tagihan vendor
        <ChevronRight className="size-4" aria-hidden />
      </Link>
    </section>
  );
}
