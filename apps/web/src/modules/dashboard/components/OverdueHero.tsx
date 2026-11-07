import { Link } from "react-router-dom";
import { formatCurrency } from "@/shared/lib/formatters";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import type { DashboardTermRow } from "@/modules/dashboard/types";

/**
 * The one number a WO opens the dashboard for: how much clients owe past the
 * due date. Everything else on the page is quieter than this.
 */
export function OverdueHero({ total, terms }: { total: number; terms: DashboardTermRow[] }) {
  const projectCount = new Set(terms.map((t) => t.projectId)).size;
  const overdue = total > 0;
  return (
    <section
      aria-labelledby="overdue-heading"
      className={`rounded-lg border px-6 py-5 ${overdue ? "border-danger/30 bg-danger-soft" : "border-border bg-surface"}`}
    >
      <h2 id="overdue-heading" className="text-[13px] font-medium text-text-secondary">
        Tagihan klien yang terlambat
      </h2>
      <p className={`mt-1 text-[2.25rem] font-bold leading-tight tabular-nums ${overdue ? "text-danger" : "text-success"}`}>
        {formatCurrency(total)}
      </p>
      {overdue ? (
        <p className="mt-1 text-[13.5px] text-text-primary">
          {terms.length} termin di {projectCount} project melewati jatuh tempo.{" "}
          <Link to={ROUTE_PATHS.projectDetail(terms[0].projectId, "pembayaran/client")} className="font-semibold text-danger underline underline-offset-2">
            Lihat project yang terlambat
          </Link>
        </p>
      ) : (
        <p className="mt-1 text-[13.5px] text-text-secondary">Tidak ada termin yang lewat jatuh tempo. Jadwal termin diatur di tab Keuangan tiap project.</p>
      )}
    </section>
  );
}
