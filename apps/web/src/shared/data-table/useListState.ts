import { useCallback, useMemo } from "react";
import { useSearchParams } from "react-router-dom";
import { useDebouncedValue } from "@/shared/hooks/useDebouncedValue";

export interface SortState {
  field: string;
  dir: "asc" | "desc";
}

export function parseSort(raw: string | null): SortState | null {
  if (!raw) return null;
  const [field, dir] = raw.split(":");
  if (!field) return null;
  return { field, dir: dir === "desc" ? "desc" : "asc" };
}

interface Options<F extends Record<string, string>> {
  /** Filter keys with their default (usually ""): a filter equal to its default is not written to the URL. */
  filters: F;
  /** Page size; the URL may override with ?limit=. */
  limit?: number;
  /** What this endpoint calls its search parameter ("search" for most lists, "q" for some). The URL always says ?q=. */
  searchParam?: string;
}

/**
 * The state of one list (page, sort, search, filters) lives in the URL, so a list can be bookmarked, shared, reloaded
 * and walked back to with the browser's Back button (PLAN.md Tahap 6). Anything equal to its default stays out of the
 * address. Changing a filter, the search or the sort returns to page 1. `params` is what the API call takes.
 */
export function useListState<F extends Record<string, string>>({ filters: defaults, limit = 10, searchParam = "search" }: Options<F>) {
  const [search, setSearch] = useSearchParams();

  const page = Math.max(1, Number(search.get("page")) || 1);
  const pageSize = Math.min(100, Math.max(1, Number(search.get("limit")) || limit));
  const sort = parseSort(search.get("sort"));
  const q = search.get("q") ?? "";
  const debouncedQ = useDebouncedValue(q, 250);
  const filters = useMemo(() => {
    const out: Record<string, string> = {};
    for (const key of Object.keys(defaults)) out[key] = search.get(key) ?? defaults[key];
    return out as F;
  }, [search, defaults]);

  const update = useCallback(
    (patch: Record<string, string | null>, opts: { resetPage?: boolean; replace?: boolean } = { resetPage: true }) => {
      setSearch(
        (prev) => {
          const next = new URLSearchParams(prev);
          for (const [key, value] of Object.entries(patch)) {
            if (value === null || value === "" || value === defaults[key]) next.delete(key);
            else next.set(key, value);
          }
          if (opts.resetPage) next.delete("page");
          return next;
        },
        { replace: opts.replace ?? false },
      );
    },
    [setSearch, defaults],
  );

  const isFiltered = q !== "" || Object.keys(defaults).some((key) => filters[key] !== defaults[key]);

  const params = useMemo(() => {
    const out: Record<string, string | number> = { page, limit: pageSize };
    if (sort) out.sort = `${sort.field}:${sort.dir}`;
    if (debouncedQ) out[searchParam] = debouncedQ;
    for (const key of Object.keys(defaults)) if (filters[key] !== "") out[key] = filters[key];
    return out;
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [page, pageSize, sort?.field, sort?.dir, debouncedQ, filters]);

  return {
    page,
    pageSize,
    sort,
    q,
    filters,
    isFiltered,
    params,
    /** A stable key for "did the query change": use it as an effect dependency. */
    paramsKey: JSON.stringify(params),
    setPage: (p: number) => update({ page: p === 1 ? null : String(p) }, { resetPage: false }),
    setSort: (next: SortState | null) => update({ sort: next ? `${next.field}:${next.dir}` : null }),
    // Typing replaces the history entry instead of adding one per keystroke.
    setQ: (value: string) => update({ q: value }, { resetPage: true, replace: true }),
    setFilter: (key: keyof F & string, value: string) => update({ [key]: value }),
    /** Back to the plain list: no search, no filters, default sort. */
    reset: () => setSearch(new URLSearchParams(), { replace: false }),
    /** The current query as a string, for saved views. */
    queryString: search.toString(),
    applyQueryString: (qs: string) => setSearch(new URLSearchParams(qs), { replace: false }),
  };
}
