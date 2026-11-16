import { useEffect, useState } from "react";
import { EMPTY_PAGINATION_META, type PaginationMeta } from "@/shared/types/pagination";
import type { Page } from "@/modules/finance/stores/useFinanceStore";

/**
 * Loads one page of a server-side list whenever its query changes (`key` is the list state's paramsKey). A slower
 * answer to an older query never overwrites a newer one. The rows of the previous page stay on screen while the next
 * one loads (DataTable dims nothing; it shows the skeleton only when there is nothing to show).
 */
export function usePagedList<T>(load: () => Promise<Page<T>>, key: string) {
  const [rows, setRows] = useState<T[]>([]);
  const [meta, setMeta] = useState<PaginationMeta>(EMPTY_PAGINATION_META);
  const [status, setStatus] = useState<"loading" | "error" | "ready">("loading");
  const [attempt, setAttempt] = useState(0);

  useEffect(() => {
    let cancelled = false;
    setStatus("loading");
    load()
      .then((page) => {
        if (cancelled) return;
        setRows(page.rows);
        setMeta(page.meta);
        setStatus("ready");
      })
      .catch(() => !cancelled && setStatus("error"));
    return () => {
      cancelled = true;
    };
    // `load` is a new closure every render; the query is what `key` names.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [key, attempt]);

  return { rows, meta, status, retry: () => setAttempt((n) => n + 1) };
}
