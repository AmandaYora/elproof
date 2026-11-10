import { useEffect, useState, type ReactNode } from "react";
import { Link } from "react-router-dom";
import { ArrowDown, ArrowUp, ArrowUpDown } from "lucide-react";
import { cn } from "@/shared/lib/cn";
import { EmptyState } from "@/shared/components/feedback/EmptyState";
import { LoadError } from "@/shared/components/feedback/LoadError";
import { Skeleton } from "@/shared/components/ui/Skeleton";
import { Pagination } from "@/shared/components/ui/Pagination";
import { Button } from "@/shared/components/kit/button";
import type { PaginationMeta } from "@/shared/types/pagination";
import type { SortState } from "@/shared/data-table/useListState";
import type { ListDensity } from "@/shared/data-table/useListDensity";

export interface Column<T> {
  id: string;
  header: string;
  /** The API's sort field. A column with one is sortable. */
  sortKey?: string;
  /** Money and counts go right-aligned in tabular figures. */
  align?: "left" | "right";
  cell: (row: T) => ReactNode;
  className?: string;
  /** The line a phone card leads with (the first column when none is marked). */
  primary?: boolean;
  /** Left out of the phone card. */
  hideOnCard?: boolean;
}

export interface Selection {
  selected: ReadonlySet<string>;
  onChange: (next: Set<string>) => void;
}

interface Props<T> {
  /** Names the table for assistive technology ("Daftar vendor"). */
  label: string;
  columns: Column<T>[];
  rows: T[];
  rowId: (row: T) => string;
  status: "loading" | "error" | "ready";
  onRetry: () => void;
  meta: PaginationMeta;
  onPageChange: (page: number) => void;
  sort: SortState | null;
  onSortChange: (next: SortState | null) => void;
  /** Whether a search or filter is active: an empty result then says so and offers to reset. */
  filtered: boolean;
  onResetFilters: () => void;
  /** Shown when the list is empty and nothing is filtered: teach what this list is and offer one action. */
  empty: { title: string; description?: string; action?: ReactNode };
  density?: ListDensity;
  /** Row actions: the last column on a computer, the card footer on a phone. */
  actions?: (row: T) => ReactNode;
  selection?: Selection;
  /** Buttons for the bar that appears while rows are selected. */
  bulkActions?: (selectedIds: string[]) => ReactNode;
  /** Where the row's first cell links to (a route path). */
  rowHref?: (row: T) => string | undefined;
  /** Names a row for assistive technology ("Katering Rasa Nusantara"), used by its checkbox. */
  rowLabel?: (row: T) => string;
}

/** True once `on` has been true for `delayMs`: a skeleton that only appears when loading is not instant. */
function useAfter(on: boolean, delayMs: number): boolean {
  const [late, setLate] = useState(false);
  useEffect(() => {
    if (!on) {
      setLate(false);
      return;
    }
    const timer = setTimeout(() => setLate(true), delayMs);
    return () => clearTimeout(timer);
  }, [on, delayMs]);
  return late;
}

/**
 * The one list component of the app (PLAN.md Tahap 6): sortable headers (aria-sort, one button each), money right
 * aligned in tabular figures, two row heights, a sticky header that scrolls under itself so focus is never hidden,
 * four states that always look the same (loading skeleton after 400 ms, error with retry, empty that teaches, empty
 * because of a filter with "Reset filter"), a total in the footer, row selection with a bulk bar, and cards on a phone.
 */
export function DataTable<T>({
  label, columns, rows, rowId, status, onRetry, meta, onPageChange, sort, onSortChange, filtered, onResetFilters, empty,
  density = "normal", actions, selection, bulkActions, rowHref, rowLabel,
}: Props<T>) {
  const showSkeleton = useAfter(status === "loading" && rows.length === 0, 400);
  const ids = rows.map(rowId);
  const selectedIds = selection ? ids.filter((id) => selection.selected.has(id)) : [];
  const allSelected = ids.length > 0 && selectedIds.length === ids.length;
  const primary = columns.find((c) => c.primary) ?? columns[0];

  function toggleSort(col: Column<T>) {
    if (!col.sortKey) return;
    if (sort?.field !== col.sortKey) onSortChange({ field: col.sortKey, dir: "asc" });
    else if (sort.dir === "asc") onSortChange({ field: col.sortKey, dir: "desc" });
    else onSortChange(null); // third press: back to the list's own order
  }

  function toggleAll() {
    if (!selection) return;
    const next = new Set(selection.selected);
    if (allSelected) ids.forEach((id) => next.delete(id));
    else ids.forEach((id) => next.add(id));
    selection.onChange(next);
  }

  function toggleOne(id: string) {
    if (!selection) return;
    const next = new Set(selection.selected);
    if (next.has(id)) next.delete(id);
    else next.add(id);
    selection.onChange(next);
  }

  if (status === "error") return <LoadError message="Daftar belum bisa dimuat. Periksa koneksi Anda, lalu coba lagi." onRetry={onRetry} />;

  if (status === "loading" && rows.length === 0) {
    return (
      <div role="status" aria-label={`Memuat ${label.toLowerCase()}`} className="overflow-hidden rounded-lg border border-border bg-card">
        {showSkeleton && (
          <div className="flex flex-col gap-3 p-4">
            {Array.from({ length: 6 }, (_, i) => (
              <Skeleton key={i} className="h-8 w-full" />
            ))}
          </div>
        )}
        {!showSkeleton && <div className="h-40" />}
      </div>
    );
  }

  if (rows.length === 0) {
    return (
      <div className="rounded-lg border border-border bg-card">
        {filtered ? (
          <EmptyState
            title="Tidak ada hasil"
            description="Tidak ada data yang cocok dengan pencarian atau filter ini."
            action={<Button variant="outline" onClick={onResetFilters}>Reset filter</Button>}
          />
        ) : (
          <EmptyState title={empty.title} description={empty.description} action={empty.action} />
        )}
      </div>
    );
  }

  const rowClass = density === "padat" ? "h-8 text-[13px]" : "h-10";

  return (
    <div className="overflow-hidden rounded-lg border border-border bg-card" aria-busy={status === "loading" || undefined}>
      {selection && selectedIds.length > 0 && (
        <div role="region" aria-label="Aksi untuk baris terpilih" className="flex flex-wrap items-center gap-3 border-b bg-accent px-4 py-2 text-[13px]">
          <span className="font-semibold text-foreground">{selectedIds.length} dipilih</span>
          {bulkActions?.(selectedIds)}
          <Button variant="ghost" size="sm" onClick={() => selection.onChange(new Set())}>Batal pilih</Button>
        </div>
      )}

      {/* Computer: a real table. */}
      <div className="hidden max-h-[75vh] scroll-pt-10 overflow-auto md:block">
        <table aria-label={label} className="w-full border-collapse text-left text-sm">
          <thead className="sticky top-0 z-10 bg-card shadow-[0_1px_0_var(--border)]">
            <tr>
              {selection && (
                <th scope="col" className="w-10 px-3">
                  <input type="checkbox" checked={allSelected} onChange={toggleAll} aria-label="Pilih semua di halaman ini" className="size-4" />
                </th>
              )}
              {columns.map((col) => {
                const active = sort?.field === col.sortKey;
                const Icon = !active ? ArrowUpDown : sort?.dir === "asc" ? ArrowUp : ArrowDown;
                return (
                  <th
                    key={col.id}
                    scope="col"
                    aria-sort={col.sortKey ? (active ? (sort?.dir === "asc" ? "ascending" : "descending") : "none") : undefined}
                    className={cn("h-10 px-4 text-[12.5px] font-semibold text-muted-foreground", col.align === "right" && "text-right", col.className)}
                  >
                    {col.sortKey ? (
                      <button
                        type="button"
                        onClick={() => toggleSort(col)}
                        className={cn(
                          "-mx-1 inline-flex items-center gap-1 rounded px-1 py-1 hover:text-foreground focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-foreground",
                          col.align === "right" && "flex-row-reverse",
                          active && "text-foreground",
                        )}
                      >
                        {col.header}
                        <Icon aria-hidden className={cn("size-3.5", !active && "opacity-50")} />
                      </button>
                    ) : (
                      col.header
                    )}
                  </th>
                );
              })}
              {actions && <th scope="col" className="h-10 px-4 text-[12.5px] font-semibold text-muted-foreground">Aksi</th>}
            </tr>
          </thead>
          <tbody className="divide-y divide-border">
            {rows.map((row) => {
              const id = rowId(row);
              const href = rowHref?.(row);
              return (
                <tr key={id} className={cn(rowClass, "scroll-mt-10 hover:bg-muted/60", selection?.selected.has(id) && "bg-accent/60")}>
                  {selection && (
                    <td className="w-10 px-3">
                      <input
                        type="checkbox"
                        checked={selection.selected.has(id)}
                        onChange={() => toggleOne(id)}
                        aria-label={`Pilih ${rowLabel?.(row) ?? "baris ini"}`}
                        className="size-4"
                      />
                    </td>
                  )}
                  {columns.map((col, i) => (
                    <td key={col.id} className={cn("px-4 align-middle text-foreground", col.align === "right" && "text-right tabular-nums", col.className)}>
                      {href && i === 0 ? (
                        <Link to={href} className="font-medium underline-offset-2 hover:underline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-foreground">
                          {col.cell(row)}
                        </Link>
                      ) : (
                        col.cell(row)
                      )}
                    </td>
                  ))}
                  {actions && <td className="px-4 align-middle">{actions(row)}</td>}
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>

      {/* Phone: one card per row, led by the primary column. */}
      <ul className="divide-y divide-border md:hidden">
        {rows.map((row) => {
          const id = rowId(row);
          return (
            <li key={id} className="flex flex-col gap-2 px-4 py-3.5">
              <div className="flex items-start gap-3">
                {selection && (
                  <input
                    type="checkbox"
                    checked={selection.selected.has(id)}
                    onChange={() => toggleOne(id)}
                    aria-label={`Pilih ${rowLabel?.(row) ?? "baris ini"}`}
                    className="mt-1 size-5 shrink-0"
                  />
                )}
                <div className="min-w-0 flex-1 text-[14px] font-semibold text-foreground">
                  {rowHref?.(row) ? <Link to={rowHref(row)!}>{primary.cell(row)}</Link> : primary.cell(row)}
                </div>
              </div>
              {columns
                .filter((c) => c !== primary && !c.hideOnCard)
                .map((c) => (
                  <div key={c.id} className="flex items-center justify-between gap-3 text-[12.5px]">
                    <span className="shrink-0 text-muted-foreground">{c.header}</span>
                    <span className={cn("min-w-0 truncate text-right font-medium text-foreground", c.align === "right" && "tabular-nums")}>{c.cell(row)}</span>
                  </div>
                ))}
              {actions && <div className="flex items-center gap-2 pt-1">{actions(row)}</div>}
            </li>
          );
        })}
      </ul>

      {meta.totalPages > 1 ? (
        <Pagination page={meta.page} totalPages={meta.totalPages} totalItems={meta.total} pageSize={meta.limit} onPageChange={onPageChange} />
      ) : (
        <p className="border-t px-5 py-3 text-[12.5px] text-muted-foreground">
          Menampilkan <span className="font-medium text-foreground">{rows.length}</span> dari <span className="font-medium text-foreground">{meta.total}</span>
        </p>
      )}
    </div>
  );
}
