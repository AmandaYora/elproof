import { createContext, useCallback, useContext, useEffect, useMemo, useState, type CSSProperties, type ReactNode } from "react";
import { useMediaQuery } from "@/shared/hooks/useMediaQuery";

const STORAGE_KEY = "elproof.sidebar.collapsed";

interface SidebarState {
  /** Phone: the menu is a sheet that opens over the page. */
  mobileOpen: boolean;
  setMobileOpen: (open: boolean) => void;
  /** Computer: the menu narrows to an icon rail. */
  collapsed: boolean;
  /** Opens the sheet on a phone, narrows or widens the rail on a computer: the same call for the button and the shortcut. */
  toggle: () => void;
}

const SidebarContext = createContext<SidebarState | null>(null);

export function useSidebar(): SidebarState {
  const ctx = useContext(SidebarContext);
  if (!ctx) throw new Error("useSidebar must be used inside <SidebarProvider>");
  return ctx;
}

/**
 * Menu state for a console layout. It also owns Cmd/Ctrl+B, and publishes the menu's width as `--sidebar-w` so the page
 * beside it can keep out of its way without knowing whether it is wide or narrow.
 */
export function SidebarProvider({ children }: { children: ReactNode }) {
  const isDesktop = useMediaQuery("(min-width: 1024px)");
  const [mobileOpen, setMobileOpen] = useState(false);
  const [collapsed, setCollapsed] = useState(() => {
    try {
      return window.localStorage.getItem(STORAGE_KEY) === "1";
    } catch {
      return false;
    }
  });

  const toggle = useCallback(() => {
    if (!isDesktop) {
      setMobileOpen((open) => !open);
      return;
    }
    setCollapsed((current) => {
      try {
        window.localStorage.setItem(STORAGE_KEY, current ? "0" : "1");
      } catch {
        // Not remembered across visits: fine.
      }
      return !current;
    });
  }, [isDesktop]);

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && !e.shiftKey && !e.altKey && e.key.toLowerCase() === "b") {
        e.preventDefault();
        toggle();
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [toggle]);

  // A sheet left open must not survive a resize to a computer.
  useEffect(() => {
    if (isDesktop) setMobileOpen(false);
  }, [isDesktop]);

  const value = useMemo(() => ({ mobileOpen, setMobileOpen, collapsed, toggle }), [mobileOpen, collapsed, toggle]);
  const style = { "--sidebar-w": collapsed ? "4rem" : "16rem" } as CSSProperties;
  return (
    <SidebarContext.Provider value={value}>
      <div style={style}>{children}</div>
    </SidebarContext.Provider>
  );
}
