import { useCallback, useState } from "react";

export type ListDensity = "normal" | "padat";
const KEY = "elproof.list.density";

/** Row height of tables: "normal" 40 px, "padat" 32 px. Remembered in this browser for every list. */
export function useListDensity(): { density: ListDensity; setDensity: (d: ListDensity) => void } {
  const [density, setState] = useState<ListDensity>(() => {
    try {
      return window.localStorage.getItem(KEY) === "padat" ? "padat" : "normal";
    } catch {
      return "normal";
    }
  });
  const setDensity = useCallback((d: ListDensity) => {
    setState(d);
    try {
      window.localStorage.setItem(KEY, d);
    } catch {
      // Not remembered: fine.
    }
  }, []);
  return { density, setDensity };
}
