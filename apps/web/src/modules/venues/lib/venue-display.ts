/** A venue's capacity per layout, as the API sends it (null = no number for that layout). */
interface VenueCapacity {
  capacityStanding: number | null;
  capacitySeated: number | null;
}

/** "600 berdiri · 400 duduk": only the layouts the venue has a number for; null when it has none. */
export function venueCapacityText(v: VenueCapacity): string | null {
  const parts = [
    v.capacityStanding ? `${v.capacityStanding.toLocaleString("id-ID")} berdiri` : null,
    v.capacitySeated ? `${v.capacitySeated.toLocaleString("id-ID")} duduk` : null,
  ].filter((p): p is string => p !== null);
  return parts.length > 0 ? parts.join(" · ") : null;
}

interface VenuePrices {
  priceAkad: number | null;
  priceResepsi: number | null;
  priceAkadResepsi: number | null;
}

/** The prices the venue has set, in the form's order. */
export function venuePriceLines(v: VenuePrices): [string, number][] {
  const lines: [string, number | null][] = [
    ["Akad", v.priceAkad],
    ["Resepsi", v.priceResepsi],
    ["Akad + Resepsi", v.priceAkadResepsi],
  ];
  return lines.filter((l): l is [string, number] => Boolean(l[1]));
}

/**
 * The price a project starts from when it picks this venue: Akad + Resepsi, else Resepsi, else Akad (0 when none).
 * The server derives the same value when it attaches a venue.
 */
export function venueStartingPrice(v: VenuePrices): number {
  return v.priceAkadResepsi ?? v.priceResepsi ?? v.priceAkad ?? 0;
}
