/** Tabs of the try-it preview on the Beranda; shared by the fallback and the lazy preview. */
export const PREVIEW_TABS = [
  { id: "proyek", label: "Proyek hari ini" },
  { id: "termin", label: "Termin & bukti" },
  { id: "portal", label: "Portal Klien" },
] as const;

/**
 * The tab row. On a phone the three tabs share the width (a label may wrap to two lines) instead of a row that scrolls
 * sideways and cuts the last tab at the screen edge (QA RT-05); from `sm` they sit side by side at their own width.
 */
export const PREVIEW_TABLIST_CLASS = "grid grid-cols-3 gap-1 py-2 sm:flex";

/** One tab (button or, in the fallback, plain text); the caller adds the border colour and state. */
export const PREVIEW_TAB_CLASS =
  "inline-flex min-h-11 min-w-0 items-center justify-center rounded-lg px-2 py-1 text-center text-[14px] leading-tight font-semibold sm:shrink-0 sm:px-3.5 sm:whitespace-nowrap";

/** The console menu drawn beside the preview on a computer (decoration only). */
export const PREVIEW_MENU = ["Beranda", "Proyek", "Jadwal", "Vendor", "Tagihan", "Kendala", "Tim"];
