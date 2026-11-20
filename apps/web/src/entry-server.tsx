// Build-time renderer for the public pages (ADR-0043). `vite build --ssr src/entry-server.tsx` bundles this file for Node and
// scripts/prerender.mjs calls it once per page; it never ships to the browser. It renders the same route table and the same
// StrictMode wrapper as main.tsx, so the HTML is what the browser's first render produces and hydrateRoot can adopt it.
import { StrictMode } from "react";
import { renderToString } from "react-dom/server";
import { prerender } from "react-dom/static";
import { createMemoryRouter, RouterProvider, type RouteObject } from "react-router-dom";
import { routes } from "@/app/routes/routes";
import { PUBLIC_PAGES, type PublicPageMeta } from "@/app/routes/public-pages";
import { APP_NAME } from "@/shared/constants/brand";
import { SITE_URL } from "@/shared/constants/site";

export { APP_NAME, SITE_URL };

/** Where the browser build sends API requests (empty: the page's own origin), for the prerender's API preloads. */
export const API_BASE_URL: string = import.meta.env.VITE_API_BASE_URL ?? "";

/** Every page scripts/prerender.mjs writes, with its head tags. */
export const pages: PublicPageMeta[] = Object.values(PUBLIC_PAGES);

function tree(path: string) {
  const router = createMemoryRouter(routes, { initialEntries: [path] });
  return (
    <StrictMode>
      <RouterProvider router={router} />
    </StrictMode>
  );
}

/**
 * The markup of one page, every lazy part in place. Two passes: `prerender` waits until every lazy chunk of the page has
 * loaded (React.lazy then answers synchronously), and `renderToString` writes the page in document order. A single
 * prerender pass would write Suspense content out of order, behind inline scripts that swap it in; those scripts are
 * blocked by a strict Content-Security-Policy and wait for requestAnimationFrame, so the page would stay on its fallbacks.
 */
export async function render(path: string): Promise<string> {
  await prerender(tree(path), {
    onError(error) {
      // A part that cannot render without a browser keeps its Suspense fallback and renders in the browser; the build
      // says which instead of failing.
      console.warn(`[prerender] ${path}: ${error instanceof Error ? error.message : String(error)}`);
    },
  });
  return renderToString(tree(path));
}

/**
 * Every path pattern the router answers ("/projects/:projectId/ringkasan", "/platform/tenant/:id/:tab?"), without the
 * catch-all "*" that draws the not-found page. The Go server answers 404 for a path that matches none of them.
 */
export function routePatterns(): string[] {
  const out = new Set<string>();
  const walk = (list: RouteObject[], base: string) => {
    for (const route of list) {
      let full = base;
      if (route.path) full = route.path.startsWith("/") ? route.path : `${base.replace(/\/$/, "")}/${route.path}`;
      if (route.path === "*" || full === "/*") continue;
      const normalized = full === "/" ? "/" : full.replace(/\/+/g, "/").replace(/\/$/, "");
      if (route.path || route.index) out.add(normalized);
      if (route.children) walk(route.children, full);
    }
  };
  walk(routes, "/");
  return [...out].sort();
}
