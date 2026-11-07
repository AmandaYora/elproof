import { useEffect } from "react";
import { APP_NAME } from "@/shared/constants/brand";

interface PageMeta {
  /** Page name; the tab reads "<title> — ElProof". */
  title: string;
  /** One or two sentences for search results and link previews. */
  description?: string;
  /** Keeps the page out of search results (sign-in, invitations). */
  noindex?: boolean;
}

function metaTag(attr: "name" | "property", key: string): HTMLMetaElement {
  let el = document.head.querySelector<HTMLMetaElement>(`meta[${attr}="${key}"]`);
  if (!el) {
    el = document.createElement("meta");
    el.setAttribute(attr, key);
    document.head.appendChild(el);
  }
  return el;
}

/**
 * Sets the tab title and the search/share tags for the current page, and puts
 * everything back when the page unmounts so the next page starts clean.
 */
export function usePageMeta({ title, description, noindex = false }: PageMeta) {
  useEffect(() => {
    const fullTitle = `${title} — ${APP_NAME}`;
    const previousTitle = document.title;
    document.title = fullTitle;

    const touched: { el: HTMLMetaElement; previous: string | null }[] = [];
    const set = (attr: "name" | "property", key: string, value: string) => {
      const el = metaTag(attr, key);
      touched.push({ el, previous: el.getAttribute("content") });
      el.setAttribute("content", value);
    };
    set("property", "og:title", fullTitle);
    if (description) {
      set("name", "description", description);
      set("property", "og:description", description);
    }
    if (noindex) set("name", "robots", "noindex");

    return () => {
      document.title = previousTitle;
      // Restore in reverse so a tag touched twice ends at its original value.
      for (const { el, previous } of touched.reverse()) {
        if (previous === null) el.remove();
        else el.setAttribute("content", previous);
      }
    };
  }, [title, description, noindex]);
}
