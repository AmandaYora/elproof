/**
 * The public address of the site, without a trailing slash: the base of canonical links, og:url and sitemap.xml (ADR-0043).
 * Set VITE_SITE_URL at build time for another host (a staging copy); production is the default.
 */
export const SITE_URL = (import.meta.env.VITE_SITE_URL || "https://elproof.elcodelabs.com").replace(/\/+$/, "");

/** The absolute address of a path on the public site: "/" → "https://…/", "/faq" → "https://…/faq". */
export function siteUrl(path: string): string {
  return `${SITE_URL}${path.startsWith("/") ? path : `/${path}`}`;
}
