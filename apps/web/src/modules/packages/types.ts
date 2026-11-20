// The `offers` module's packages (docs/API_CONTRACT.md, ADR-0049): what a WO offers its clients — a price, the events it
// covers, the guests it includes, an optional default venue, and what it contains ("Isi paket").

export interface NamedRef {
  id: string;
  name: string;
}

export interface PackageComponent {
  id: string;
  description: string;
  categoryId: string | null;
  categoryName: string | null;
  vendorId: string | null;
  vendorName: string | null;
  /** Present only for a role with finance.margin.view; null when not filled in. */
  estimatedCost?: number | null;
}

export interface Package {
  id: string;
  name: string;
  price: number;
  /** null: the price does not depend on the number of guests. */
  includedGuests: number | null;
  /** Extra guests are charged per this many guests… */
  extraGuestStep: number | null;
  /** …at this price per started step. */
  extraGuestStepPrice: number | null;
  defaultVenue: NamedRef | null;
  eventKinds: NamedRef[];
  components: PackageComponent[];
  componentCount: number;
  isActive: boolean;
  createdAt: string;
  /** The version a save sends back as If-Match (ADR-0036). */
  updatedAt: string;
}

/** POST /packages/preview: the "Contoh hitung" box, priced by the same rule quotations use. */
export interface PackagePreview {
  price: number;
  extra: { guests: number; over: number; steps: number; amount: number };
  total: number;
}
