import type { ReactNode } from "react";
import { DataTable, type Column } from "@/shared/data-table/DataTable";
import { cn } from "@/shared/lib/cn";
import { Badge } from "@/shared/components/ui/Badge";

/**
 * Building blocks of the Keuangan tab's three ledgers (RISET WO Console §7.3): a section title with its action on the
 * right, one strip of totals, and the shared DataTable over the full width of the page.
 */
export function FinanceSection({ title, action, notice, children }: { title: string; action?: ReactNode; notice?: ReactNode; children: ReactNode }) {
  return (
    <section className="flex flex-col gap-4" aria-label={title}>
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h2 className="text-[16px] leading-snug font-semibold text-foreground">{title}</h2>
        {action}
      </div>
      {notice}
      {children}
    </section>
  );
}

export interface SummaryItem {
  label: string;
  value: string;
  /** One line under the number ("Lebih bayar Rp500.000"). */
  note?: string;
  /** "warning" (default) for a problem such as an overpayment, "muted" for plain context ("Rp2.000.000 menunggu verifikasi"). */
  noteTone?: "warning" | "muted";
  tone?: "neutral" | "warning";
}

/** The totals of a ledger in one strip: label 13 px, amount 20 px tabular, no capitals. */
export function FinanceSummary({ items }: { items: SummaryItem[] }) {
  return (
    <dl className="grid grid-cols-1 divide-y divide-border rounded-lg border border-border bg-card sm:grid-cols-3 sm:divide-x sm:divide-y-0">
      {items.map((item) => (
        <div key={item.label} className="flex items-baseline justify-between gap-3 px-5 py-3.5 sm:block">
          <dt className="text-[13px] text-muted-foreground">{item.label}</dt>
          <dd className="text-right sm:mt-1 sm:text-left">
            <span className={cn("text-[17px] font-bold tracking-[-0.01em] whitespace-nowrap tabular-nums sm:text-[20px]", item.tone === "warning" ? "text-warning-text" : "text-foreground")}>
              {item.value}
            </span>
            {item.note && <span className={cn("mt-0.5 block text-[12.5px]", item.noteTone === "muted" ? "text-muted-foreground" : "text-warning-text")}>{item.note}</span>}
          </dd>
        </div>
      ))}
    </dl>
  );
}

interface LedgerTableProps<T> {
  label: string;
  columns: Column<T>[];
  rows: T[];
  rowId: (row: T) => string;
  /** The page the browser shows (the ledger is loaded whole: a project has a handful of payments). */
  page: number;
  pageSize: number;
  totalPages: number;
  totalItems: number;
  onPageChange: (page: number) => void;
  empty: { title: string; description?: string; action?: ReactNode };
  actions?: (row: T) => ReactNode;
}

/** The shared DataTable over a ledger already in the browser: no sorting or filtering, paged locally. */
export function LedgerTable<T>({ label, columns, rows, rowId, page, pageSize, totalPages, totalItems, onPageChange, empty, actions }: LedgerTableProps<T>) {
  return (
    <DataTable
      label={label}
      columns={columns}
      rows={rows}
      rowId={rowId}
      status="ready"
      onRetry={() => undefined}
      meta={{ page, limit: pageSize, total: totalItems, totalPages }}
      onPageChange={onPageChange}
      sort={null}
      onSortChange={() => undefined}
      filtered={false}
      onResetFilters={() => undefined}
      empty={empty}
      actions={actions}
    />
  );
}

/** Whether a payment out has both its files (invoice and transfer proof). */
export function EvidenceBadge({ complete }: { complete: boolean }) {
  return complete ? <Badge tone="success">Bukti lengkap</Badge> : <Badge tone="warning">Belum lengkap</Badge>;
}
