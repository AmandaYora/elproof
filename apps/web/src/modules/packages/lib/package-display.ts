import { formatCurrencyCompact } from "@/shared/lib/formatters";
import type { PackageFormValues } from "@/modules/packages/schemas/package.schema";
import type { EventKind } from "@/modules/event-kinds/types";

interface GuestRule {
  includedGuests: number | null;
  extraGuestStep: number | null;
  extraGuestStepPrice: number | null;
}

/** "+Rp4 jt per 50 tamu": the extra-guest rule in one phrase, or null when the package has none. */
export function extraGuestText(rule: GuestRule): string | null {
  if (rule.includedGuests === null || !rule.extraGuestStep || rule.extraGuestStepPrice === null) return null;
  const per = rule.extraGuestStep === 1 ? "per tamu" : `per ${rule.extraGuestStep.toLocaleString("id-ID")} tamu`;
  return `+${formatCurrencyCompact(rule.extraGuestStepPrice)} ${per}`;
}

/**
 * The guest count the "Contoh hitung" box starts with: a little over two extra steps above what is included, so the
 * example shows a charge (500 included, per 50 → 620). The person can change it.
 */
export function defaultPreviewGuests(included: number | null, step: number | null): number {
  if (included === null) return 0;
  const s = step && step > 0 ? step : 50;
  return included + 2 * s + Math.round(s * 0.4);
}

let exampleRow = 0;
export const newRowKey = () => `row-${Date.now().toString(36)}-${(exampleRow++).toString(36)}`;

/**
 * "Isi dengan contoh" (DESAIN.md §6.6): fills the form with a typical reception package so a first-time Owner sees
 * what each field means. Nothing is saved until they press "Simpan paket".
 */
export function examplePackage(kinds: EventKind[]): PackageFormValues {
  const pick = (name: string) => kinds.find((k) => k.isActive && k.name.toLowerCase() === name)?.id;
  const kindIds = [pick("akad"), pick("resepsi")].filter((id): id is string => Boolean(id));
  const rows = [
    "Katering buffet 450 porsi + 6 gubukan",
    "Dekorasi pelaminan & area foto",
    "Rias & busana pengantin (akad dan resepsi)",
    "Dokumentasi foto & video",
    "MC & hiburan akustik",
  ];
  return {
    name: "Paket Resepsi 500 tamu",
    price: 120_000_000,
    eventKindIds: kindIds,
    includedGuests: "500",
    extraGuestStep: "50",
    extraGuestStepPrice: 4_000_000,
    defaultVenueId: "",
    components: rows.map((description) => ({ key: newRowKey(), id: "", description, categoryId: "", vendorId: "", estimatedCost: 0 })),
  };
}
