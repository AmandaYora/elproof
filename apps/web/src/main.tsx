import { StrictMode } from "react";
import { createRoot, hydrateRoot } from "react-dom/client";
import App from "@/app/App";
import "@/styles/globals.css";

const container = document.getElementById("root")!;
const app = (
  <StrictMode>
    <App />
  </StrictMode>
);

// A public page written at build time (ADR-0042) carries its own path on #root. Hydrate only when that is the page being
// opened; any other markup (a stale file, a path the server mapped loosely) is replaced by a fresh render.
const prerendered = container.dataset.prerendered;
const path = window.location.pathname.replace(/(.)\/+$/, "$1");

if (prerendered && prerendered === path) {
  hydrateRoot(container, app, {
    // Expected now and then: the sign-in page restored with history state, a notice kept in sessionStorage. React then
    // renders that part in the browser, which is the right result; say so without raising an error.
    onRecoverableError(error) {
      console.warn("[hydrate] rendered in the browser instead:", error instanceof Error ? error.message : error);
    },
  });
} else {
  createRoot(container).render(app);
}
