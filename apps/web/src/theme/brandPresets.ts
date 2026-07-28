// The 15 fixed brand-color presets a tenant can pick (see PLAN.md §3) — no
// free-form hex is supported. Each preset supplies the same 4 shades
// theme.css already defines for the app's default "navy" look
// (--brand-navy-950/900/800 + --color-primary-soft), sourced from Tailwind's
// own default palette families for the 14 non-navy presets so every choice is
// pre-tuned for contrast/accessibility. Keys here are the single source of
// truth and MUST match apps/api's domain.AllowedBrandColorPresets exactly.
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
  gold: { 950: "#451a03", 900: "#78350f", 800: "#92400e", soft: "#fef3c7" },
  orange: { 950: "#431407", 900: "#7c2d12", 800: "#9a3412", soft: "#ffedd5" },
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
  rose: "Merah Muda",
  cyan: "Cyan",
  fuchsia: "Fuchsia",
  lime: "Hijau Lime",
  slate: "Abu Grafit",
  stone: "Cokelat Hangat",
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
