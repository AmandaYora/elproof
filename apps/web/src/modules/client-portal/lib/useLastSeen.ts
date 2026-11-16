import { useEffect, useState } from "react";

const KEY = (projectId: string) => `elproof.portal.lastSeen.${projectId}`;

/**
 * When the couple last opened Beranda (ms), read once as the page opens, then moved to now. Null on a first visit or
 * when storage is unavailable, so nothing is marked new rather than everything.
 */
export function useLastSeen(projectId: string): number | null {
  const [previous] = useState<number | null>(() => {
    try {
      const n = Number(window.localStorage.getItem(KEY(projectId)));
      return Number.isFinite(n) && n > 0 ? n : null;
    } catch {
      return null;
    }
  });
  useEffect(() => {
    try {
      window.localStorage.setItem(KEY(projectId), String(Date.now()));
    } catch {
      // Not remembered: the next visit marks nothing new, which is harmless.
    }
  }, [projectId]);
  return previous;
}
