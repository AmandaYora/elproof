import { useCallback, useState } from "react";

/** How a list is laid out on a computer: rows to compare ("table") or cards to scan ("cards"). */
export type ListView = "table" | "cards";

/**
 * The layout a person chose for one list (`key`: "projects", "vendors"), remembered in this browser. A table stays the
 * default: it compares records column by column; cards suit scanning a few facts per record (NN/g, card vs list view).
 */
export function useListView(key: string): { view: ListView; setView: (v: ListView) => void } {
  const storageKey = `elproof.list.view.${key}`;
  const [view, setState] = useState<ListView>(() => {
    try {
      return window.localStorage.getItem(storageKey) === "cards" ? "cards" : "table";
    } catch {
      return "table";
    }
  });
  const setView = useCallback(
    (v: ListView) => {
      setState(v);
      try {
        window.localStorage.setItem(storageKey, v);
      } catch {
        // Not remembered: fine.
      }
    },
    [storageKey],
  );
  return { view, setView };
}
