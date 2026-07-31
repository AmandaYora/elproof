// Real backend types for the `venues` directory (ADR-0016) — see
// docs/API_CONTRACT.md. Lives in its own module/menu, no longer a vendor
// category — see knowledge/decisions/ADR-0016-venue-extraction.md. Revisi:
// facilities/socialMedia are free text now (not arrays); phonePic/phoneVenue
// replace phone/phoneSecondary; city is a fixed-list field; there's no more
// photo gallery, just a single attachment (document or photo).

export interface Venue {
  id: string;
  name: string;
  picName: string;
  phonePic: string;
  phoneVenue: string | null;
  email: string | null;
  address: string | null;
  city: string | null;
  rentalPrice: number | null;
  charge: number | null;
  capacity: number | null;
  facilities: string | null;
  socialMedia: string | null;
  notes: string;
  hasAttachment: boolean;
  attachmentIsImage: boolean;
  isActive: boolean;
  createdAt: string;
}

export interface VenueImportRowError {
  row: number;
  message: string;
}

export interface VenueImportResult {
  insertedCount: number;
  updatedCount: number;
  errors: VenueImportRowError[];
}
