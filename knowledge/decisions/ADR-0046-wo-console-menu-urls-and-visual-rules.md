# ADR-0046: WO Console menu, Indonesian URLs, one banner slot, and visual rules

## Status
Accepted — 2026-11-16. Research and brief: `docs/plan/revamp-wo-console/RISET.md` (structure §3–§6, visual §7). Visual contract:
`knowledge/DESIGN.md` §4.5.

## Context
A review of the WO Console against SaaS navigation guidance (NN/g vertical navigation, tabs, settings IA, table row actions,
banner fatigue) and against modern in-app UIs (shadcn/ui blocks, Linear, Twenty, HoneyBook, Vercel) found: no cross-project home
for money and dates, a heavy project page (420 px header × 9 tabs), two stacked banners on every page, "Pengaturan" mixing workspace
settings with templates used daily, 3–4 inline icons per table row, cramped/wrapping tables, a flat type hierarchy, and URLs mixing
English and Indonesian. The owner asked to do all of it at once.

## Decision
* **Menu:** Beranda · Proyek · Kalender · Keuangan · Klien · Direktori (Vendor, Venue) · Templat (Jadwal bawaan, Kategori vendor) ·
  Pengaturan (Profil usaha, Tim, Peran dan akses, Langganan). Keuangan needs `finance.record`; Kalender is open to every staff role and
  scoped by the API to the role's projects. Sidebar shows the WO's logo (or initial) and name; the user is a menu (help on WhatsApp,
  Keluar) whose dropdown code loads on first click (entry-chunk budget).
* **Topbar:** search (⌘K) and "+ Proyek baru" (with `project.create`).
* **One banner slot:** a subscription problem (warning/danger) first, otherwise platform announcements; a calm trial (> 3 days) is a
  chip in the sidebar, not a banner. Maintenance announcements use a quiet card style.
* **URLs in Indonesian:** `/beranda`, `/proyek`, `/proyek/:id/...`, `/kalender`, `/keuangan`, `/klien`, `/vendor`, `/venue`,
  `/kategori-vendor`, `/tim`, `/jadwal-bawaan` (`/peran-akses`, `/profil-usaha`, `/langganan` unchanged). Old addresses redirect with
  the rest of the path, query and hash kept (`LegacyPathRedirect`), so links in chats and bookmarks keep working.
* **Shared building blocks:** `PageHeader`, `RowActions`, `ActionMenu` ghost, DataTable `truncate`/`card` columns and 44/36 px rows,
  KPI spacing in `StatCard`. Rules in DESIGN.md §4.5.
* Project detail (compact header, 6 tabs), Beranda (KPI → chart → actions), Keuangan and Kalender pages, and directory/settings
  polish are built on these (see the respective commits and API_CONTRACT entries).

## Consequences
* Every console screenshot changes; e2e golden snapshots are regenerated after review.
* e2e tests navigate the new URLs; old ones are covered by redirects.
