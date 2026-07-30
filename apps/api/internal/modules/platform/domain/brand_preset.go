package domain

// DefaultBrandColorPreset must match migration 000017's
// `brand_color_preset ... DEFAULT 'navy'` — used wherever a Tenant is
// constructed in Go without an explicit preset (e.g. Register), so the
// in-memory value agrees with what the DB will actually store.
const DefaultBrandColorPreset = "navy"

// AllowedBrandColorPresets is the fixed set of 21 brand colors a tenant may
// pick for its WO Console / Client Portal theme (see PLAN.md §3/§14) — free-form
// hex is deliberately not supported, so this is the single source of truth
// the application layer validates against. "navy" is the app's original,
// unbranded look and stays the default for every existing tenant. Hex values
// (and the mustard/green/sky/pink/yellow/bronze additions) live only on the
// frontend (apps/web/src/theme/brandPresets.ts) — this backend list only
// needs to agree on which keys are valid.
var AllowedBrandColorPresets = []string{
	"navy", "gold", "orange", "blue", "emerald", "red", "purple",
	"teal", "indigo", "rose", "cyan", "fuchsia", "lime", "slate", "stone",
	"mustard", "green", "sky", "pink", "yellow", "bronze",
}

func IsValidBrandColorPreset(preset string) bool {
	for _, p := range AllowedBrandColorPresets {
		if p == preset {
			return true
		}
	}
	return false
}
