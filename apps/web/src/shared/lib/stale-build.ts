// After a deploy, a tab still open on the previous build names that build's chunk files, which the server no longer has
// (each build names its files after their content, and the browser keeps them a year: apps/api/cmd/server/spa.go). The
// first screen that tab opens and has not loaded yet then fails. Vite reports it as "vite:preloadError"; the page reloads
// once and comes back on the current build. A second failure within a minute is not an old tab (offline, a real outage)
// and is left to the error screen, so the page never reloads in a loop.
const KEY = "elproof.staleBuildReloadAt";
const ONCE_WITHIN_MS = 60_000;

export function reloadOnStaleBuild() {
  window.addEventListener("vite:preloadError", (event) => {
    try {
      const last = Number(sessionStorage.getItem(KEY)) || 0;
      if (Date.now() - last < ONCE_WITHIN_MS) return;
      sessionStorage.setItem(KEY, String(Date.now()));
    } catch {
      // Without storage there is no guard against a loop: leave it to the error screen.
      return;
    }
    event.preventDefault();
    window.location.reload();
  });
}
