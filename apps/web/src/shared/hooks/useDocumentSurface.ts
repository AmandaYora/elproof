import { useLayoutEffect } from "react";

/**
 * Marks <html> with the surface whose tokens apply (theme/marketing.css). Set on
 * <html>, not on a wrapper, because Base UI dialogs, popovers and the toaster
 * render into <body>, outside any layout element. Removed again when the
 * layout unmounts, so the consoles never inherit the public palette.
 */
export function useDocumentSurface(surface: "marketing") {
  useLayoutEffect(() => {
    const root = document.documentElement;
    root.dataset.surface = surface;
    return () => {
      if (root.dataset.surface === surface) delete root.dataset.surface;
    };
  }, [surface]);
}
