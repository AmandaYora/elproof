import { useCallback, useEffect, useState } from "react";

export type LoadStatus = "loading" | "ready" | "error";

/**
 * Runs `load` when the page opens (and again on retry) and reports whether the
 * data is in, so a page shows a skeleton, the content, or "Coba lagi" and never
 * mistakes a failed call for "nothing here".
 */
export function useLoad(load: () => Promise<unknown>, deps: unknown[]): { status: LoadStatus; retry: () => void } {
  const [status, setStatus] = useState<LoadStatus>("loading");
  const [attempt, setAttempt] = useState(0);
  // eslint-disable-next-line react-hooks/exhaustive-deps
  const run = useCallback(load, deps);

  useEffect(() => {
    let cancelled = false;
    setStatus("loading");
    run()
      .then(() => !cancelled && setStatus("ready"))
      .catch(() => !cancelled && setStatus("error"));
    return () => {
      cancelled = true;
    };
  }, [run, attempt]);

  return { status, retry: () => setAttempt((n) => n + 1) };
}
