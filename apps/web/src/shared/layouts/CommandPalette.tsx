import { lazy, Suspense, useEffect, useState, type ReactNode } from "react";
import { Search } from "lucide-react";
import type { NavItem } from "@/shared/layouts/nav/types";

// cmdk and its dialog are a sizeable chunk and nobody needs them until the palette is opened, so they load then.
const CommandPaletteDialog = lazy(() => import("@/shared/layouts/CommandPaletteDialog"));

/** The button in the top bar that opens the palette. */
export function CommandTrigger({ onOpen, placeholder = "Cari proyek, vendor, atau klien..." }: { onOpen: () => void; placeholder?: string }) {
  return (
    <button
      type="button"
      onClick={onOpen}
      className="flex h-9 w-full max-w-sm items-center gap-2 rounded-md border border-input bg-card px-3 text-left text-sm text-subtle-foreground hover:bg-muted focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-foreground"
    >
      <Search aria-hidden className="size-4 shrink-0" />
      <span className="min-w-0 flex-1 truncate">{placeholder}</span>
      <kbd aria-hidden className="hidden rounded border bg-muted px-1.5 py-0.5 font-sans text-[11px] font-medium text-muted-foreground sm:block">Ctrl K</kbd>
      <span className="sr-only">Pintasan: Ctrl atau Cmd dan K</span>
    </button>
  );
}

/**
 * Cmd/Ctrl+K: jump to a page or to a project, vendor or client (PLAN.md Tahap 7). The shortcut and the trigger are here;
 * the search itself (CommandPaletteDialog) is fetched the first time the palette opens.
 */
export function CommandPalette({ pages, searchRecords, trigger }: { pages: NavItem[]; searchRecords: boolean; trigger?: (open: () => void) => ReactNode }) {
  const [open, setOpen] = useState(false);
  const [wanted, setWanted] = useState(false);

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && !e.shiftKey && !e.altKey && e.key.toLowerCase() === "k") {
        e.preventDefault();
        setOpen((v) => !v);
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, []);
  useEffect(() => {
    if (open) setWanted(true);
  }, [open]);

  return (
    <>
      {trigger?.(() => setOpen(true))}
      {(wanted || open) && (
        <Suspense fallback={null}>
          <CommandPaletteDialog open={open} onOpenChange={setOpen} pages={pages} searchRecords={searchRecords} />
        </Suspense>
      )}
    </>
  );
}
