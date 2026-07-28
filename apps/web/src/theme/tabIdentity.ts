// Browser tab title + favicon are the last "ElProof" touchpoints inside an
// authenticated session (Sidebar/Client Portal headers already show the
// tenant's own logo/name — see brandPresets.ts) — this makes those two match.
// Captured once at module load, before any tenant ever overrides it, so
// resetTabIdentity() has the real default to revert to without hardcoding a
// second copy of index.html's <title> text.
const DEFAULT_TITLE = document.title;

let faviconLink: HTMLLinkElement | null = null;

/** logoUrl reuses the same object URL already fetched for the sidebar/header
 * image — no separate favicon upload needed. Pass undefined/null logoUrl to
 * mean "no logo yet" (falls back to no favicon at all, same as today). */
export function applyTabIdentity(businessName: string, logoUrl: string | null, consoleLabel?: string) {
  document.title = consoleLabel ? `${businessName} — ${consoleLabel}` : businessName;

  if (logoUrl) {
    if (!faviconLink) {
      faviconLink = document.createElement("link");
      faviconLink.rel = "icon";
      document.head.appendChild(faviconLink);
    }
    faviconLink.href = logoUrl;
  } else if (faviconLink) {
    faviconLink.remove();
    faviconLink = null;
  }
}

export function resetTabIdentity() {
  document.title = DEFAULT_TITLE;
  if (faviconLink) {
    faviconLink.remove();
    faviconLink = null;
  }
}
