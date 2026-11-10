# ADR-0039: One palette for every surface — the per-WO colour choice is removed

## Status
Accepted — implemented 2026-11-10 (task T-20261110-11). Supersedes the colour half of ADR-0012 (the logo half stays) and the
"Tenant branding overrides only `--brand-navy-*`" line of ADR-0037; takes the decision ADR-0038 left open ("aligning the consoles
to the new palette is a separate decision"). Visual contract: `knowledge/DESIGN.md` §4–§5; contrast table in §4.1.

## Context
The public site moved to "Janji Terbukti" (plum, seal gold, Plus Jakarta Sans only; ADR-0038), scoped to `<html data-surface="marketing">`.
After sign-in the WO Console, Client Portal and Platform Console still used the old navy default, overridden at runtime by one of
21 colour presets a WO could choose (Profil Usaha, or ElProof's team in the tenant form), plus Fraunces for a few portal figures.

The owner decided (T-20261110-11): ElProof is a SaaS with one look for every WO — "fitur ubah warna itu dihilangkan total"; the stored
choice is dropped from the database; each WO keeps its logo.

Also found: Tailwind 4's `outline-none` sets `--tw-outline-style: none`, and `outline-2` reads that variable, so the kit button, input and
textarea drew **no** keyboard focus indicator at all (WCAG 2.4.7). Combobox and SearchInput used a 20%-opacity ring (about 1.3:1), and the
Combobox trigger's border used the decorative `--border` (1.3:1) instead of `--input`.

## Decision

### One palette in `theme.css`
* `:root` holds the whole palette for every surface: plum `--primary` #3E1F3B (actions, active navigation, focus ring), seal gold
  (`--seal` #B88A3E graphic only, `--seal-text` #7A5A1E, `--seal-soft` #F8F0E1), and plum-tinted neutrals from the same family as the public
  site's text greys. Values and computed WCAG ratios: DESIGN.md §4.1.
* Console neutrals are **not** the public blush: the page is a plum-tinted near-white (#FAF8F9) and quiet surfaces are #F4EFF2, because dense
  tables and forms need calm backgrounds; the text greys, input boundary, plum and gold are identical to the public site.
* `marketing.css` keeps only what differs on the public surface (white page, blush bands and borders) plus its scoped motion. The two files
  are not merged into one: the public surface is still a different surface, but it no longer carries its own copy of the palette.
* `--brand-navy-*`, `--color-primary-soft`, the `navy-*` utilities, `theme/colors.ts` and `theme/brandPresets.ts` are removed; components use
  semantic names (`bg-primary`, `hover:bg-primary-hover`, `outline-ring`, `text-foreground`). Badge/IconActionButton tone `navy` is now `brand`.
  The `neutral` button variant (`bg-stone-800`, kept so the sign-in page showed no tenant colour) is removed: there is no tenant colour left.
* WhatsApp green is the one third-party colour, exposed as `--whatsapp*` tokens instead of a hex in `AccessLinkDialog`; the QR code uses the
  library's default black on white instead of a hex.

### Where gold goes
Gold means **proof**, never status and never chrome: `ProofStamp` (seal-soft background, seal-text words and icon, a decorative seal outline) and
the thin full-strength `--seal` rule in the Portal home hero on plum (4.61:1). Warning stays amber with an icon and text; `--seal` is never drawn on `--background`
(2.95:1) and never used for text.

### One typeface
Plus Jakarta Sans everywhere. Fraunces and `--font-display` are removed (package, `@import`, token); the portal hero, the next amount due and
the H- counter use heavy, tight Plus Jakarta Sans like the public headings.

### The per-WO colour choice is gone
* API (module `platform`): `domain/brand_preset.go`, `UpdateOwnBrandColor`, `UpdateBrandColor`, `PATCH /tenants/me/brand-color`,
  `brandColorPreset` on the tenant update body, on `tenantResponse` and on `GET /tenants/me/branding` are removed. An old cached bundle that
  still sends `brandColorPreset` is harmless (unknown JSON fields are ignored); one that calls `PATCH /tenants/me/brand-color` gets not-found.
* Migration `000072_drop_tenant_brand_color`: `up` drops `tenants.brand_color_preset`; `down` re-adds it with 000017's definition
  (`VARCHAR(20) NOT NULL DEFAULT 'navy'`). The chosen colours are not recoverable after `up`.
* Web: the "Warna merek" card (Profil Usaha), the swatches in the tenant form and the row in Tenant 360° are removed;
  `useTenantBrandingStore` keeps the logo, business name, tab title and favicon, and its `hydrated` gate (so the logo and name are painted once).

### Keyboard focus
Every kit control that hides the browser outline adds `focus-visible:outline-solid` next to `outline-2` (button: 2 px, offset 2 px; input,
textarea, Combobox trigger, SearchInput: 2 px on the border). The ring is plum, ≥ 11.5:1 on every neutral surface (DESIGN.md §4.1). The
`keyboard.spec.ts` e2e asserts a solid outline of at least 2 px in `--ring` on the focused kit button and input.

## Consequences
* A colour change is one file for every surface; there is no runtime override, so no colour flash on sign-in and no per-preset contrast audit.
* A WO that had chosen a colour sees plum after the deploy; its logo and name stay. The data is dropped with the column (owner's decision).
* The image runs `migrate up` before serving, as before; the new code never reads the column. Rolling the code back needs `migrate down 1`
  first (the old code selects the column), which restores it with the default for every WO.
* `white`/`black` utilities remain only for modal scrims and text on solid status colours; any other palette class or hex in a component is
  a review failure (ADR-0037). The command palette's search box (kit `command.tsx`) shows its focus as a 2 px `--ring` ring on the group.
