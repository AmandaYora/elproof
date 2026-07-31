// Real backend types for the `vendors` module — see docs/API_CONTRACT.md.
// Revisi (per the "DATA VENDOR" slide): email/address are now optional;
// socialMedia/city/priceAkad/priceAkadResepsi are new; a single attachment
// slot replaces what used to be no attachment at all.

export interface Vendor {
  id: string;
  name: string;
  categoryId: string;
  picName: string;
  phone: string;
  email: string | null;
  socialMedia: string | null;
  city: string | null;
  address: string | null;
  priceAkad: number | null;
  priceAkadResepsi: number | null;
  notes: string;
  hasAttachment: boolean;
  attachmentIsImage: boolean;
  isActive: boolean;
  createdAt: string;
}

// Public-safe subset (ADR-0016-style split) — {id, name} only, backs Client
// Portal's Vendor Progress tab. Never carries PIC/phone/email/price/
// attachment — those stay staff-only via the full `Vendor` shape above.
export interface VendorSummary {
  id: string;
  name: string;
}

// One row of a vendor's project engagement history — see
// `GET /vendors/{id}/project-history`.
export interface VendorProjectHistoryItem {
  projectId: string;
  projectName: string;
  eventDate: string;
  venue: string;
  engagementStatus: string;
}

export interface VendorImportRowError {
  row: number;
  message: string;
}

export interface VendorImportResult {
  insertedCount: number;
  updatedCount: number;
  errors: VendorImportRowError[];
}
