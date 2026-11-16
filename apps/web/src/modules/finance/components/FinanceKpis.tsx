import { StatCard } from "@/shared/components/ui/StatCard";
import { Skeleton } from "@/shared/components/ui/Skeleton";
import { LoadError } from "@/shared/components/feedback/LoadError";
import { formatCurrency, formatCurrencyCompact } from "@/shared/lib/formatters";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import type { FinanceSummary } from "@/modules/finance/types";

/** A money figure short enough for a KPI ("Rp35 jt"), with the exact amount on hover and for screen readers. */
function Money({ amount }: { amount: number }) {
  return (
    <span title={formatCurrency(amount)}>
      <span aria-hidden>{formatCurrencyCompact(amount)}</span>
      <span className="sr-only">{formatCurrency(amount)}</span>
    </span>
  );
}

const at = (query: string) => `${ROUTE_PATHS.finance}?${query}`;

/**
 * The five numbers the page answers first (RISET WO Console §7.6): who is late, what falls due this month, what came
 * in, what the WO still owes, and how many proofs wait to be checked. Each one opens the list behind it.
 */
export function FinanceKpis({ summary, status, onRetry }: { summary: FinanceSummary | null; status: "loading" | "error" | "ready"; onRetry: () => void }) {
  if (status === "error" && !summary) {
    return <LoadError message="Ringkasan keuangan belum bisa dimuat. Periksa koneksi Anda, lalu coba lagi." onRetry={onRetry} />;
  }
  if (!summary) {
    return (
      <div role="status" aria-label="Memuat ringkasan keuangan" className="grid grid-cols-2 gap-3 lg:grid-cols-5">
        {Array.from({ length: 5 }, (_, i) => (
          <Skeleton key={i} className="h-[108px] rounded-lg" />
        ))}
      </div>
    );
  }
  const { overdue, dueSoon, received, payables, pendingVerification } = summary;
  return (
    <section aria-label="Ringkasan keuangan" className="grid grid-cols-2 gap-3 lg:grid-cols-5 [&>*:last-child]:col-span-2 lg:[&>*:last-child]:col-span-1">
      <StatCard
        label="Tagihan terlambat"
        value={<Money amount={overdue.amount} />}
        tone={overdue.count > 0 ? "danger" : "neutral"}
        sublabel={overdue.count > 0 ? `${overdue.count} termin dari ${overdue.projects} proyek` : "Tidak ada yang terlambat"}
        to={at("status=overdue")}
      />
      <StatCard
        label="Jatuh tempo 30 hari"
        value={<Money amount={dueSoon.amount} />}
        sublabel={dueSoon.count > 0 ? `${dueSoon.count} termin dari ${dueSoon.projects} proyek` : "Tidak ada termin"}
        to={at("status=due_30d")}
      />
      <StatCard
        label="Masuk 30 hari terakhir"
        value={<Money amount={received.amount} />}
        sublabel={received.count > 0 ? `${received.count} pembayaran klien` : "Belum ada pembayaran"}
      />
      <StatCard
        label="Sisa bayar vendor & venue"
        value={<Money amount={payables.amount} />}
        tone={payables.overdueCount > 0 ? "warning" : "neutral"}
        sublabel={
          payables.overdueCount > 0
            ? `${formatCurrencyCompact(payables.overdueAmount)} lewat jatuh tempo`
            : payables.count > 0
              ? `${payables.count} tagihan vendor dan venue`
              : "Semua sudah lunas"
        }
        to={at("tab=keluar")}
      />
      <StatCard
        label="Bukti menunggu verifikasi"
        value={pendingVerification.count}
        tone={pendingVerification.count > 0 ? "warning" : "neutral"}
        sublabel={pendingVerification.count > 0 ? `Senilai ${formatCurrencyCompact(pendingVerification.amount)}` : "Semua sudah diperiksa"}
        to={at("tab=verifikasi")}
      />
    </section>
  );
}
