import { AlertCircle, Clock } from "lucide-react";
import { formatCurrency, formatDate } from "@/shared/lib/formatters";
import { daysUntil } from "@/modules/projects/lib/dates";
import type { PaymentTerm } from "@/modules/projects/types";

/**
 * "Yang perlu Anda lakukan": the only place the portal uses red, and only when
 * the client really has to act (a payment past its due date). A payment due
 * within a week is shown in amber, as a heads-up.
 */
export function ActionCard({ terms }: { terms: PaymentTerm[] }) {
  const items = terms.filter((t) => t.status === "overdue" || (t.status === "due_soon" && daysUntil(t.dueDate) <= 7)).slice(0, 3);
  if (items.length === 0) return null;
  return (
    <section aria-labelledby="todo-heading" className="rounded-2xl border border-border bg-white p-5 sm:p-6">
      <h2 id="todo-heading" className="text-[15px] font-semibold text-text-primary">Yang perlu Anda lakukan</h2>
      <ul className="mt-3 flex flex-col gap-2.5">
        {items.map((t) => {
          const overdue = t.status === "overdue";
          const Icon = overdue ? AlertCircle : Clock;
          return (
            <li key={t.id}>
              <a
                href="#tagihan"
                className={`flex items-start gap-3 rounded-xl border px-4 py-3 focus-visible:outline focus-visible:outline-2 focus-visible:outline-navy-900 ${
                  overdue ? "border-danger/30 bg-danger-soft" : "border-warning/30 bg-warning-soft"
                }`}
              >
                <Icon className={`mt-0.5 h-4 w-4 shrink-0 ${overdue ? "text-danger-text" : "text-warning-strong"}`} aria-hidden />
                <span className="min-w-0">
                  <span className="block text-[14px] font-semibold text-text-primary">
                    {overdue ? `Pembayaran ${t.label} sudah lewat jatuh tempo` : `Pembayaran ${t.label} jatuh tempo ${formatDate(t.dueDate)}`}
                  </span>
                  <span className="block text-[13px] text-text-secondary">
                    {formatCurrency(t.remaining)}
                    {overdue ? `, jatuh tempo ${formatDate(t.dueDate)}. Hubungi kami bila ada kendala.` : ""}
                  </span>
                </span>
              </a>
            </li>
          );
        })}
      </ul>
    </section>
  );
}
