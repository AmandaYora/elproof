// The 20 fixed brand-color presets a tenant can pick (see PLAN.md §3/§14) — no
// free-form hex is supported. Each preset supplies the same 4 shades
// theme.css already defines for the app's default "navy" look
// (--brand-navy-950/900/800 + --color-primary-soft), sourced from Tailwind's
// own default palette families so every choice is pre-tuned for
// contrast/accessibility. Keys here are the single source of truth and MUST
// match apps/api's domain.AllowedBrandColorPresets exactly.
//
// gold/orange (and every preset added in §14) deliberately use a
// lighter/more saturated shade band (family-600/500/400) than the other
// original presets (family-950/900/800) — a user flagged the original
// gold/orange as reading too brown/muddy at the 900-level Tailwind shade;
// shifting the whole ramp lighter keeps enough contrast for white text at
// the "950" sidebar-background role while looking genuinely vivid, not dark
// and desaturated like the rest of the original 15.
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
  // Vivid amber (family-600/500/400) — was family-900/800 (too brown/muddy).
  gold: { 950: "#d97706", 900: "#f59e0b", 800: "#fbbf24", soft: "#fef3c7" },
  // Vivid orange (family-600/500/400) — was family-900/800 (too brown/muddy).
  orange: { 950: "#ea580c", 900: "#f97316", 800: "#fb923c", soft: "#ffedd5" },
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
  // New in §14. 900 is exactly the hex the user asked for — #b3a500.
  mustard: { 950: "#6b6300", 900: "#b3a500", 800: "#d4c400", soft: "#f5f0c9" },
  // New in §14. Shifted off green-600 (#16a34a) on purpose — that hex is
  // this app's fixed --color-success and must stay visually distinct from
  // any brand preset.
  green: { 950: "#14532d", 900: "#15803d", 800: "#22c55e", soft: "#dcfce7" },
  sky: { 950: "#0284c7", 900: "#0ea5e9", 800: "#38bdf8", soft: "#e0f2fe" },
  pink: { 950: "#db2777", 900: "#ec4899", 800: "#f472b6", soft: "#fce7f3" },
  yellow: { 950: "#ca8a04", 900: "#eab308", 800: "#facc15", soft: "#fef9c3" },
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
