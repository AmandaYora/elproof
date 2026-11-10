import type { ReactNode } from "react";
import { PREVIEW_MENU, PREVIEW_TABS } from "@/modules/homepage/components/home/previewTabs";
import { SampleTag } from "@/modules/homepage/components/home/mock";
import { cn } from "@/shared/lib/cn";

/**
 * The frame of the try-it preview: a small WO Console window. The tab row and
 * the body come from the caller, so the static fallback and the interactive
 * preview share one shape (no jump when the chunk arrives).
 */
export function PreviewWindow({ tabs, children }: { tabs: ReactNode; children: ReactNode }) {
  return (
    <div className="overflow-hidden rounded-2xl border border-border bg-background shadow-xl shadow-primary/10">
      <div className="flex items-center justify-between gap-3 border-b border-border bg-muted px-4 pt-3 sm:px-5">
        <p className="pb-3 text-[14px] font-bold">WO Console</p>
        <SampleTag className="mb-3" />
      </div>
      <div className="border-b border-border bg-muted px-2 sm:px-3">{tabs}</div>
      <div className="grid md:grid-cols-[176px_1fr]">
        <div aria-hidden className="hidden border-r border-border px-3 py-4 md:block">
          {PREVIEW_MENU.map((item, i) => (
            <span
              key={item}
              className={cn(
                "block rounded-lg px-3 py-2 text-[13.5px]",
                i === 0 ? "bg-accent font-semibold text-accent-foreground" : "text-muted-foreground",
              )}
            >
              {item}
            </span>
          ))}
        </div>
        <div className="min-h-[420px] min-w-0">{children}</div>
      </div>
    </div>
  );
}

/** Shown until the interactive preview has loaded: the same window, tab names as plain text. */
export function PreviewFallback() {
  return (
    <PreviewWindow
      tabs={
        <div className="flex gap-1 overflow-x-auto py-2">
          {PREVIEW_TABS.map((tab, i) => (
            <span
              key={tab.id}
              className={cn(
                "inline-flex min-h-11 items-center rounded-lg px-3.5 text-[14px] font-semibold whitespace-nowrap",
                i === 0 ? "bg-background text-foreground" : "text-muted-foreground",
              )}
            >
              {tab.label}
            </span>
          ))}
        </div>
      }
    >
      <div aria-busy="true" className="p-5 sm:p-6">
        <p className="text-[20px] font-bold tracking-[-0.02em]">Raka &amp; Dinda</p>
        <p className="mt-1 text-[14px] text-muted-foreground">Sabtu, 12 Desember 2026, Gedung Sasana Kriya</p>
      </div>
    </PreviewWindow>
  );
}
