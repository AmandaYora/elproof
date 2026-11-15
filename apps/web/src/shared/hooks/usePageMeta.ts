import { useEffect } from "react";
import { APP_NAME } from "@/shared/constants/brand";
import { siteUrl } from "@/shared/constants/site";

interface PageMeta {
  /** Page name; the tab reads "<title> — ElProof". */
  title: string;
  /** One or two sentences for search results and link previews. */
  description?: string;
  /** Keeps the page out of search results (sign-in, invitations). */
  noindex?: boolean;
  /**
   * The page's own path, for the canonical link and og:url (on SITE_URL, ADR-0042). Public pages pass their entry of
   * PUBLIC_PAGES; without it, or on a noindex page, no canonical is set.
   */
  path?: string;
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

function canonicalTag(): HTMLLinkElement {
  let el = document.head.querySelector<HTMLLinkElement>('link[rel="canonical"]');
  if (!el) {
    el = document.createElement("link");
    el.setAttribute("rel", "canonical");
    document.head.appendChild(el);
  }
  return el;
}

/**
 * Sets the tab title and the search/share tags for the current page, and puts
 * everything back when the page unmounts so the next page starts clean. The
 * same values are in the prerendered HTML of a public page (scripts/prerender.mjs).
 */
export function usePageMeta({ title, description, noindex = false, path }: PageMeta) {
  useEffect(() => {
    const fullTitle = `${title} — ${APP_NAME}`;
    const previousTitle = document.title;
    document.title = fullTitle;

    const touched: { el: Element; attr: string; previous: string | null }[] = [];
    const setAttr = (el: Element, attr: string, value: string) => {
      touched.push({ el, attr, previous: el.getAttribute(attr) });
      el.setAttribute(attr, value);
    };
    const set = (attr: "name" | "property", key: string, value: string) => setAttr(metaTag(attr, key), "content", value);
    set("property", "og:title", fullTitle);
    set("name", "twitter:title", fullTitle);
    if (description) {
      set("name", "description", description);
      set("property", "og:description", description);
      set("name", "twitter:description", description);
    }
    if (noindex) set("name", "robots", "noindex");
    if (path && !noindex) {
      const url = siteUrl(path);
      setAttr(canonicalTag(), "href", url);
      set("property", "og:url", url);
    }

    return () => {
      document.title = previousTitle;
      // Restore in reverse so a tag touched twice ends at its original value.
      for (const { el, attr, previous } of touched.reverse()) {
        if (previous === null) el.remove();
        else el.setAttribute(attr, previous);
      }
    };
  }, [title, description, noindex, path]);
}
