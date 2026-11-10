import { useLayoutEffect } from "react";
import { useMatches } from "react-router-dom";

/** What a route says about itself. Put `handle: { title: "Proyek" }` on the route object; the deepest route with a title wins. */
export interface RouteHandle {
  title?: string;
}

/**
 * Tab title of a console page: "<page> — <site>". The route table names the page, so every page of a console tells a screen-reader
 * user (and the tab strip, and history) where it is, instead of all of them reading "WO Contoh — WO Console" (WCAG 2.4.2). A layout
 * effect, so a page that sets its own title with `usePageMeta` (a passive effect, which runs later) still wins.
 */
export function useRouteTitle(site: string | null | undefined) {
  const matches = useMatches();
  let title: string | undefined;
  for (let i = matches.length - 1; i >= 0 && !title; i--) title = (matches[i].handle as RouteHandle | undefined)?.title;

  useLayoutEffect(() => {
    if (!title) return;
    document.title = site ? `${title} — ${site}` : title;
  }, [title, site]);
}
