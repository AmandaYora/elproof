// The 21 fixed brand-color presets a tenant can pick (see PLAN.md §3/§14) — no
// free-form hex is supported. Each preset supplies the same 4 shades
// theme.css already defines for the app's default "navy" look
// (--brand-navy-950/900/800 + --color-primary-soft), sourced from Tailwind's
// own default palette families so every choice is pre-tuned for
// contrast/accessibility. Keys here are the single source of truth and MUST
// match apps/api's domain.AllowedBrandColorPresets exactly.
//
// gold/orange/yellow/sky/pink/mustard use a lighter/more saturated shade
// band (family-700/600/500ish) than the darkest original presets
// (family-950/900/800) — a user flagged gold/orange as reading too
// brown/muddy at the 900-level Tailwind shade. An EARLIER version of this
// fix shifted those presets all the way to family-600/500/400, which look
// vivid but broke white-text contrast badly on "900" (every primary button
// app-wide) and "800" (hover/active-nav) — e.g. gold-900 measured ~2.15:1
// and orange-900 ~2.80:1 against white, both far under WCAG AA's 4.5:1 for
// normal text. Every preset below is re-picked so its "900" role clears
// ~4.5:1+ against white (verified via the WCAG relative-luminance formula),
// its "800"/hover role stays at least ~3:1 (acceptable for a momentary
// state), and "950" (large, persistent Sidebar background) stays safely
// dark. See PLAN.md §16 for the full per-preset contrast numbers.
export interface BrandColorShades {
  /** Darkest shade — sidebar/header background. */
  950: string;
  /** Primary shade — buttons, active states. */
  900: string;
  /** Hover/lighter-active shade. */
  800: string;
  /** Light tint — soft/tinted backgrounds. */
  soft: string;
}

export const BRAND_COLOR_PRESETS = {
  navy: { 950: "#172741", 900: "#1e3a5f", 800: "#24476f", soft: "#e3ebf3" },
  // amber-800/700/600 — 900 role (#b45309) measures ~5.02:1 against white.
  gold: { 950: "#92400e", 900: "#b45309", 800: "#d97706", soft: "#fef3c7" },
  // orange-800/700/600 — 900 role (#c2410c) measures ~5.18:1 against white.
  orange: { 950: "#9a3412", 900: "#c2410c", 800: "#ea580c", soft: "#ffedd5" },
  blue: { 950: "#172554", 900: "#1e3a8a", 800: "#1e40af", soft: "#dbeafe" },
  emerald: { 950: "#022c22", 900: "#064e3b", 800: "#065f46", soft: "#d1fae5" },
  red: { 950: "#450a0a", 900: "#7f1d1d", 800: "#991b1b", soft: "#fee2e2" },
  purple: { 950: "#2e1065", 900: "#4c1d95", 800: "#5b21b6", soft: "#ede9fe" },
  teal: { 950: "#042f2c", 900: "#134e4a", 800: "#115e59", soft: "#ccfbf1" },
  indigo: { 950: "#1e1b4b", 900: "#312e81", 800: "#3730a3", soft: "#e0e7ff" },
  rose: { 950: "#4c0519", 900: "#881337", 800: "#9f1239", soft: "#ffe4e6" },
  cyan: { 950: "#083344", 900: "#164e63", 800: "#155e75", soft: "#cffafe" },
  fuchsia: { 950: "#4a044e", 900: "#701a75", 800: "#86198f", soft: "#fae8ff" },
  lime: { 950: "#1a2e05", 900: "#365314", 800: "#3f6212", soft: "#ecfccb" },
  slate: { 950: "#020617", 900: "#0f172a", 800: "#1e293b", soft: "#f1f5f9" },
  stone: { 950: "#0c0a09", 900: "#1c1917", 800: "#292524", soft: "#f5f5f4" },
  // New in §14. The user's exact requested hex (#b3a500) only measures
  // ~2.53:1 against white -- unreadable on a primary button -- so per §16
  // this is darkened to a same-family olive/mustard tone that clears ~5:1
  // instead of the literal hex.
  mustard: { 950: "#4a4400", 900: "#7a7000", 800: "#9c8900", soft: "#f5f0c9" },
  // New in §14. 900 role (green-700, ~5.01:1) was already safe; only 800
  // (hover) is changed here (§16) -- green-500 measured ~2.28:1. Now uses
  // green-600, which happens to equal this app's --color-success hex
  // (#16a34a); accepted as a low-priority coincidence (different UI
  // contexts, a momentary hover state) rather than compromising contrast
  // further to dodge it.
  green: { 950: "#14532d", 900: "#15803d", 800: "#16a34a", soft: "#dcfce7" },
  // sky-800/700/600 — 900 role (#0369a1) measures ~5.93:1 against white.
  sky: { 950: "#075985", 900: "#0369a1", 800: "#0284c7", soft: "#e0f2fe" },
  // pink-700/600/500 — 900 role (#db2777) measures ~4.60:1 against white.
  pink: { 950: "#be185d", 900: "#db2777", 800: "#ec4899", soft: "#fce7f3" },
  // yellow-800/700/600 — 900 role (#a16207) measures ~4.92:1 against white.
  yellow: { 950: "#854d0e", 900: "#a16207", 800: "#ca8a04", soft: "#fef9c3" },
  // Requested as #B88A44/#C49A57 (hue ~36-37°, a muted bronze/antique-gold,
  // not a Tailwind family swatch) — both measured well under the 4.5:1/3:1
  // floors (~3.11:1 and ~2.59:1 against white) so, same treatment as
  // mustard above, darkened within the same hue to a tone that clears them:
  // 900 (#96692e) ~4.82:1, 800 (#ab8347) ~3.46:1.
  bronze: { 950: "#6b4a1f", 900: "#96692e", 800: "#ab8347", soft: "#f4e4cc" },
} as const satisfies Record<string, BrandColorShades>;

export type BrandColorPresetKey = keyof typeof BRAND_COLOR_PRESETS;

export const BRAND_COLOR_PRESET_LABELS: Record<BrandColorPresetKey, string> = {
  navy: "Navy (Default)",
  gold: "Emas",
  orange: "Oranye",
  blue: "Biru",
  emerald: "Hijau Emerald",
  red: "Merah",
  purple: "Ungu",
  teal: "Tosca",
  indigo: "Indigo",
  // Renamed from "Merah Muda" so the new, more vivid "pink" preset can use
  // that name instead — "rose"'s dusty/mauve tone reads more like "Mawar".
  rose: "Mawar",
  cyan: "Cyan",
  fuchsia: "Fuchsia",
  lime: "Hijau Lime",
  slate: "Abu Grafit",
  stone: "Cokelat Hangat",
  mustard: "Zaitun",
  green: "Hijau",
  sky: "Biru Langit",
  pink: "Merah Muda",
  yellow: "Kuning",
  bronze: "Perunggu",
};

export const BRAND_COLOR_PRESET_KEYS = Object.keys(BRAND_COLOR_PRESETS) as BrandColorPresetKey[];

export function isBrandColorPresetKey(value: string): value is BrandColorPresetKey {
  return Object.prototype.hasOwnProperty.call(BRAND_COLOR_PRESETS, value);
}

/** Overrides the app's brand CSS variables on the document root — reused by
 * useTenantBrandingStore.hydrate() and reset() (defaults = "navy"'s values,
 * i.e. theme.css's own hardcoded values, so reset() and "never configured"
 * look identical). */
export function applyBrandColorPreset(key: BrandColorPresetKey) {
  const shades = BRAND_COLOR_PRESETS[key];
  const root = document.documentElement.style;
  root.setProperty("--brand-navy-950", shades[950]);
  root.setProperty("--brand-navy-900", shades[900]);
  root.setProperty("--brand-navy-800", shades[800]);
  root.setProperty("--color-primary-soft", shades.soft);
}

export function resetBrandColorPreset() {
  const root = document.documentElement.style;
  root.removeProperty("--brand-navy-950");
  root.removeProperty("--brand-navy-900");
  root.removeProperty("--brand-navy-800");
  root.removeProperty("--color-primary-soft");
}
