# ADR-0045: Navy + seal gold from the logo for every surface, and the public site redesign ("Bukti yang Terlihat")

## Status
Accepted — implemented 2026-11-16. Replaces the plum **values** of ADR-0039 (its rule "one palette, no per-WO colour" stays) and the
"Janji Terbukti" direction of ADR-0038 for the public site's look (its scoped surface, CSS-only motion and data-driven prices stay).
Research and brief: `docs/plan/redesign-situs-publik/BRIEF.md`. Visual contract: `knowledge/DESIGN.md` §4.1, `knowledge/DESIGN-PUBLIC.md`.

## Context
On 15 Nov 2026 the logo changed to a navy rounded square with a white heart-and-ring and gold check circles. The whole app was still plum
(#3E1F3B), so logo and product read as two brands. The owner also found the public site "kurang meyakinkan". A study of 14 SaaS sites
(Linear, Attio, Stripe, Ramp, Mercury, Notion, Cal.com, Superhuman, HoneyBook, Aisle Planner, Dubsado, Planning Pod, Mekari, Majoo) and
NN/g's credibility guidance showed what ours lacked: a large, realistic product picture in the hero, checkable proof right under it, feature
sections that show the product instead of text, and (for Indonesia) WhatsApp within reach. Asked on 16 Nov 2026, the owner chose the logo
palette for **the whole application**, not only the public site.

## Decision
* **Palette (all surfaces, `theme.css`):** `--primary`/`--foreground`/`--ring` navy #0B1B43 (the logo's), hover #16245A; navy-tinted neutrals
  (`--background` #F6F7FB, `--muted` #EDF0F6, text greys #4F5A78 / #646F8C, `--input` #7A84A0); seal gold `--seal` #B4853D on light
  surfaces (graphic only) and the new `--seal-on-primary` #E0B074 (the logo's gold, 8.49:1 on navy, may be text) on navy; `--highlight-text`
  #8A5A12; `--on-primary-muted` #C3CBE0. Status colours unchanged; navy and gold never mean status. Ratios: DESIGN.md §4.1.
* **Public surface (`marketing.css`):** white page, cool mist bands #EEF1F7. A warm cream band was considered and rejected: cream + serif +
  warm accent is the most common templated look, and the brand already has its warmth in the gold.
* **Type:** Plus Jakarta Sans stays everywhere (a display serif was considered for a "wedding premium" feel and rejected for the same reason).
* **Beranda:** wide headline + the two-screen hero (`HeroScreens`: WO Console in front of which sits the Client Portal phone; one payment,
  "Termin 2", carries the same gold seal on both: the brand promise drawn once), a proof strip of checkable facts (no customer counts, logos or
  ratings we do not have), a bento "Dari H-180 sampai hari H" (`Journey`, five cards of different sizes, each with a piece of the real screen)
  replacing the text timeline (`Story`, removed), the existing try-it preview, a navy band with CTAs, plan **cards** (one plan: card + "Setelah
  Anda mendaftar"), FAQ, closing CTA. WhatsApp: header link (xl), a CTA beside every "Coba gratis", and on phones an action bar that appears
  once the hero's own buttons have scrolled away (never in the first render, so prerendered HTML and hydration agree).
* Footer names the maker (Elcodelabs, PT Prayora Karya Pratama).

## Consequences
* Every console screenshot changes colour; the e2e golden snapshots were regenerated after review.
* Wedding photos (planned in the brief) are not used until the right to use the Tampilan Login photos is confirmed; a customer quote section is
  not built until a real WO agrees to be quoted.
* `theme-color` (index.html, manifest, offline page) is the logo navy.
