# ADR-0038: Public site — scoped theme, one typeface, live prices, and CSS-only scroll motion

## Status
Accepted — implemented 2026-11-10 (task T-20261110-9). Extends ADR-0037 for the twelve pages a visitor sees before signing in; the consoles are unchanged.
The visual contract is `knowledge/DESIGN-PUBLIC.md`.

## Context
The owner judged the public pages (Beranda, Tentang Kami, FAQ, Kontak, four legal pages, Masuk, Daftar, Siapkan workspace, Undangan) "very ugly" and asked for a
full redesign after research. Measured problems: content hidden by a JavaScript reveal (`useRevealOnScroll`) stayed invisible on fast scrolls and full-page captures;
three different brand marks; template tells (eyebrow labels, identical icon cards, one coloured phrase in the headline); no price, no product preview, no FAQ on the
Beranda. The owner chose direction **B "Janji Terbukti"** (plum, seal gold) from three previews, with **all text in the typeface of direction A** (Plus Jakarta Sans,
bold and tight headings), "Paling hemat" computed from the data, and no testimonials.

Constraints from ADR-0037: semantic tokens only (no hex in components), tenant branding overrides `--brand-navy-*` inline on `<html>` at runtime, entry chunk ≤ 165 KB gzip,
and DESIGN.md forbids per-section entrance animation (written for the consoles).

## Decision

### A second token set, scoped to the public surface
* `src/theme/marketing.css` redefines the **same semantic names** (`--background`, `--foreground`, `--muted`, `--primary`, `--ring`, `--accent`, `--input`, …) under
  `:root[data-surface="marketing"]`, plus five public-only roles exposed as utilities: `seal`, `seal-text`, `seal-soft`, `highlight-text`, `on-primary-muted`.
  `--font-display` points at the sans family, so the public site uses one typeface.
* `useDocumentSurface("marketing")` sets the attribute on `<html>` from `MarketingLayout` and `AuthSplitLayout` and removes it on unmount. `<html>`, not a wrapper,
  because Base UI popups and the toaster render into `<body>`. Because `--primary`/`--ring`/`--accent` are defined directly, a tenant preset written inline to
  `--brand-navy-*` cannot leak into the public pages, and the consoles never see the public values (their e2e baselines are the guard).
* Sign-in screens use the public palette instead of the tenant-neutral stone button: still no tenant colour before sign-in (ADR-0012 holds).

### Prices from data
* `GET /api/v1/public/plans` (module `billing`, owner of `subscription_plans`): no session, active plans of ElProof's own catalog only (plans without an App
  mapping), ordered by duration then price, standard envelope, `{ id, name, durationMonths, price, features }`, `Cache-Control: public, max-age=300`.
* The Beranda parses it with Zod (`modules/homepage/schemas/plan.schema.ts`); per-month price, "Paling hemat" (lowest price per month, none on a tie) and
  "Hemat X%" (against the shortest plan) are computed in `modules/homepage/lib/planMath.ts`. No price is written by hand. Loading shows a skeleton without
  numbers; an error or an empty catalog asks the visitor to ask on WhatsApp.
* `perf-budget.json` route `marketing-beranda` goes from 0 to 1 API call, reason recorded there.

### Motion: a scoped exception
* Per-section reveal is allowed **only on Beranda and Tentang Kami**, opt-in with the `.reveal` class: CSS `animation-timeline: view()` inside
  `@supports` and `prefers-reduced-motion: no-preference`, opacity + ≤ 12 px over the first half of the element's entry. The animation is tied to the
  scroll position (it plays back if the element leaves through the bottom again); there is no timer and no state. No JavaScript hides content, so a browser
  without scroll-driven animations or a person who asked for less motion simply sees the page. Never on legal pages, forms, prices or FAQ.
* Motion that answers a person's action (a short fade when a "Cara kerja" step or a preview tab is chosen) is allowed, behind `motion-safe:`.
* One expressive moment per page: the seal pressed onto the verified payment in the hero (`.seal-press`, ≤ 700 ms, once, reduced-motion → static).
* `Reveal` and `useRevealOnScroll` are removed.
* Interactive parts below the fold (the clickable app preview, the scroll story) are lazy chunks; the entry budget does not change.

## Consequences
* A colour change on the public site is one file; the consoles cannot drift by accident.
* The public site needs the API for prices; without it the section degrades to a WhatsApp prompt, never to stale numbers.
* A cached price may lag an edit in Platform Console by up to five minutes.
* Aligning the consoles to the new palette is a separate decision (not taken).
