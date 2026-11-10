import { useCallback, useState } from "react";

export interface SavedView {
  name: string;
  /** The list's query string (search, filters, sort) without page. */
  query: string;
}

function read(key: string): SavedView[] {
  try {
    const raw = window.localStorage.getItem(key);
    const parsed: unknown = raw ? JSON.parse(raw) : [];
    return Array.isArray(parsed)
      ? parsed.filter((v): v is SavedView => !!v && typeof (v as SavedView).name === "string" && typeof (v as SavedView).query === "string")
      : [];
  } catch {
    return [];
  }
}

/**
 * Named filter+sort combinations a person keeps for a list ("Termin terlambat", "Vendor katering"). They live in this
 * browser only (localStorage), per list; nothing is sent to the server. Storage may be blocked: then views just are not kept.
 */
export function useSavedViews(listKey: string) {
  const storageKey = `elproof.views.${listKey}`;
  const [views, setViews] = useState<SavedView[]>(() => read(storageKey));

  const persist = useCallback(
    (next: SavedView[]) => {
      setViews(next);
      try {
        window.localStorage.setItem(storageKey, JSON.stringify(next));
      } catch {
        // Not kept across visits; still works for this one.
      }
    },
    [storageKey],
  );

  return {
    views,
    save: (name: string, query: string) => persist([...views.filter((v) => v.name !== name), { name, query }].slice(-12)),
    remove: (name: string) => persist(views.filter((v) => v.name !== name)),
  };
}

/** The part of a query string a saved view keeps: everything except the page. */
export function viewQuery(queryString: string): string {
  const params = new URLSearchParams(queryString);
  params.delete("page");
  return params.toString();
}
