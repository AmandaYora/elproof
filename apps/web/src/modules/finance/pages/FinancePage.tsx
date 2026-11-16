import { useCallback, useEffect, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
import { PageHeader } from "@/shared/components/ui/PageHeader";
import { cn } from "@/shared/lib/cn";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import { useFinanceStore } from "@/modules/finance/stores/useFinanceStore";
import { FinanceKpis } from "@/modules/finance/components/FinanceKpis";
import { ReceivablesTab } from "@/modules/finance/components/ReceivablesTab";
import { PayablesTab } from "@/modules/finance/components/PayablesTab";
import { CashFlowTab } from "@/modules/finance/components/CashFlowTab";
import { VerificationsTab } from "@/modules/finance/components/VerificationsTab";

type Section = "tagihan" | "keluar" | "arus-kas" | "verifikasi";

const SECTIONS: { id: Section; label: string }[] = [
  { id: "tagihan", label: "Tagihan klien" },
  { id: "keluar", label: "Pembayaran keluar" },
  { id: "arus-kas", label: "Arus kas" },
  { id: "verifikasi", label: "Verifikasi bukti" },
];

/**
 * Keuangan: money across every project the person reaches (RISET WO Console §3 #1, brief §7.6). Five numbers first,
 * then one section at a time: client installments, money owed to vendors and venues, the 8-week cash flow, and the
 * transfer proofs waiting to be checked. Recording money stays in the project; every row leads to its Keuangan tab.
 * The section and each list's search, filters, sort and page live in the URL.
 */
export default function FinancePage() {
  const [search] = useSearchParams();
  const raw = search.get("tab");
  const section: Section = SECTIONS.some((s) => s.id === raw) ? (raw as Section) : "tagihan";

  const summary = useFinanceStore((s) => s.summary);
  const fetchSummary = useFinanceStore((s) => s.fetchSummary);
  const [summaryStatus, setSummaryStatus] = useState<"loading" | "error" | "ready">("loading");

  const loadSummary = useCallback(() => {
    setSummaryStatus("loading");
    fetchSummary()
      .then(() => setSummaryStatus("ready"))
      .catch(() => setSummaryStatus("error"));
  }, [fetchSummary]);

  useEffect(() => {
    loadSummary();
  }, [loadSummary]);

  const pending = summary?.pendingVerification.count ?? 0;

  return (
    <div className="flex flex-col gap-6">
      <PageHeader
        className="mb-0"
        title="Keuangan"
        description="Tagihan klien, pembayaran ke vendor dan venue, serta arus kas dari semua proyek Anda."
      />

      <FinanceKpis summary={summary} status={summaryStatus} onRetry={loadSummary} />

      <div className="flex flex-col gap-4">
        {/* `relative`: the badge's sr-only word is absolutely positioned; without a positioned ancestor inside the scrolling
            row it escapes it and widens the whole page on a phone (QA FQ-05). */}
        <nav aria-label="Bagian keuangan" className="relative -mx-1 flex gap-6 overflow-x-auto border-b border-border px-1">
          {SECTIONS.map((s) => {
            const active = s.id === section;
            return (
              <Link
                key={s.id}
                to={s.id === "tagihan" ? ROUTE_PATHS.finance : `${ROUTE_PATHS.finance}?tab=${s.id}`}
                aria-current={active ? "page" : undefined}
                className={cn(
                  "relative inline-flex shrink-0 items-center gap-1.5 border-b-2 py-3 text-[14px] font-medium transition-colors focus-visible:outline-2 focus-visible:outline-offset-[-2px] focus-visible:outline-solid focus-visible:outline-ring",
                  active ? "border-primary font-semibold text-foreground" : "border-transparent text-muted-foreground hover:text-foreground",
                )}
              >
                {s.label}
                {s.id === "verifikasi" && pending > 0 && (
                  <span className="rounded-full bg-warning-soft px-1.5 text-[12px] font-semibold tabular-nums text-warning-text">
                    {pending}
                    <span className="sr-only"> menunggu</span>
                  </span>
                )}
              </Link>
            );
          })}
        </nav>

        {section === "tagihan" && (
          <ReceivablesTab counts={summary ? { overdue: summary.overdue.count, due_30d: summary.dueSoon.count } : undefined} />
        )}
        {section === "keluar" && <PayablesTab />}
        {section === "arus-kas" && <CashFlowTab weeks={summary?.cashFlow ?? null} status={summaryStatus} onRetry={loadSummary} />}
        {section === "verifikasi" && <VerificationsTab />}
      </div>
    </div>
  );
}
