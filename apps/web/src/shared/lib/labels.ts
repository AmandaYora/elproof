// One place for every status label shown to people (FLOW-IDEAL.md §4,
// PLAN.md Fase 3b). The database keeps English enum values; staff and
// clients each read their own wording. `null` for a client means "do not
// show this to the client" (internal detail).
import type {
  EngagementStatus,
  EventLayout,
  IssueImpact,
  IssueStatus,
  LineKind,
  MilestoneStatus,
  PaymentTermStatus,
  ProjectStatus,
  SupplierKind,
  SupplierPayer,
  SupplierPaymentStatus,
} from "@/modules/projects/types";
import type { AcceptChannel, QuotationDisplayStatus, TermDueBasis } from "@/modules/offers/types";

/** The price columns of a directory vendor. */
type VendorPricingTier = "Akad" | "AkadResepsi" | "Resepsi";

export type Audience = "staff" | "client";

interface Labels<T extends string> {
  staff: Record<T, string>;
  client: Record<T, string | null>;
}

export const PROJECT_STATUS_LABELS: Labels<ProjectStatus> = {
  staff: { Draft: "Draf", Preparation: "Persiapan", Ready: "Siap", Completed: "Selesai", Cancelled: "Dibatalkan" },
  client: { Draft: null, Preparation: "Sedang dipersiapkan", Ready: "Persiapan akhir", Completed: "Selesai, terima kasih", Cancelled: "Dibatalkan" },
};

export const ENGAGEMENT_STATUS_LABELS: Labels<EngagementStatus> = {
  staff: {
    Planned: "Direncanakan",
    Negotiation: "Negosiasi",
    Booked: "Dipesan",
    "DP Paid": "DP dibayar",
    "In Progress": "Dikerjakan",
    "Fully Paid": "Lunas",
    Ready: "Siap",
    Completed: "Selesai",
    Cancelled: "Dibatalkan",
  },
  client: {
    Planned: "Sedang dipilih WO",
    Negotiation: "Sedang dipilih WO",
    Booked: "Sudah dipesan",
    "DP Paid": "Sudah dipesan",
    "In Progress": "Sedang disiapkan",
    "Fully Paid": "Sudah dipesan",
    Ready: "Siap",
    Completed: "Selesai",
    Cancelled: "Diganti WO",
  },
};

export const MILESTONE_STATUS_LABELS: Labels<MilestoneStatus> = {
  staff: { "Not Started": "Belum mulai", "In Progress": "Berjalan", Completed: "Selesai", Blocked: "Terhambat", Cancelled: "Dibatalkan" },
  client: { "Not Started": "Akan datang", "In Progress": "Sedang berjalan", Completed: "Selesai", Blocked: "Sedang berjalan", Cancelled: null },
};

export const ISSUE_STATUS_LABELS: Labels<IssueStatus> = {
  staff: { Open: "Baru", "In Review": "Ditinjau", "In Resolution": "Ditangani", Resolved: "Teratasi", Closed: "Ditutup" },
  client: {
    Open: "Sedang kami periksa",
    "In Review": "Sedang kami periksa",
    "In Resolution": "Sedang ditangani WO",
    Resolved: "Sudah beres",
    Closed: "Sudah beres",
  },
};

export const ISSUE_IMPACT_LABELS: Labels<IssueImpact> = {
  staff: { Low: "Rendah", Medium: "Sedang", High: "Tinggi", Critical: "Kritis" },
  client: { Low: null, Medium: null, High: null, Critical: null },
};

/** A vendor's price package (DESIGN.md §7: "Akad", not "Pemberkatan"; "saja", not "Only"). The price fields carry the same words: "Harga Akad". */
export const PRICING_TIER_LABELS: Record<VendorPricingTier, string> = {
  Akad: "Akad",
  AkadResepsi: "Akad + Resepsi",
  Resepsi: "Resepsi saja",
};

/** Payment-schedule installment status (FLOW-IDEAL §4). */
export const TERM_STATUS_LABELS: Labels<PaymentTermStatus> = {
  staff: { paid: "Lunas", overdue: "Terlambat", partial: "Dibayar sebagian", due_soon: "Segera jatuh tempo", upcoming: "Belum jatuh tempo" },
  client: {
    paid: "Lunas",
    overdue: "Lewat jatuh tempo",
    partial: "Dibayar sebagian",
    due_soon: "Jatuh tempo segera",
    upcoming: "Menunggu pembayaran",
  },
};

/** Where an amount the WO owes a vendor or venue stands (GET /finance/payables). */
export const PAYABLE_STATUS_LABELS: Record<"paid" | "unpaid" | "overdue" | "incomplete", string> = {
  paid: "Lunas",
  unpaid: "Belum lunas",
  overdue: "Terlambat",
  // A vendor or venue the WO pays with no "Biaya rekanan" yet: what is owed is not known.
  incomplete: "Belum lengkap",
};

/** What a line of the rincian is (DESAIN §8: "Isi paket", "Potongan", never "Komponen" or "Diskon"). */
export const LINE_KIND_LABELS: Record<LineKind, string> = {
  Package: "Paket",
  Component: "Isi paket",
  ExtraGuests: "Tambahan tamu",
  Item: "Item",
  Discount: "Potongan",
};

/** Who an amount the WO owes goes to. */
export const PAYABLE_KIND_LABELS: Record<SupplierKind, string> = {
  vendor: "Vendor",
  venue: "Venue",
};

/** "Tata letak" of an event at its venue (DESAIN §8: Berdiri / Duduk, never Standing / Seated). */
export const EVENT_LAYOUT_LABELS: Record<EventLayout, string> = {
  Standing: "Berdiri",
  Seated: "Duduk",
};

/**
 * Where a payment to a vendor or venue the client pays directly stands (Paket WO Tahap 4, "Tagihan vendor"). The couple
 * sends proof from the portal; the WO marks it as matching ("Tandai sesuai") or refuses it with a reason. A payment the
 * WO recorded itself is just "Dicatat WO" to staff.
 */
export const SUPPLIER_PAYMENT_STATUS_LABELS: Labels<SupplierPaymentStatus> = {
  staff: { recorded: "Dicatat WO", pending: "Menunggu diperiksa", verified: "Sesuai", rejected: "Ditolak" },
  client: { recorded: "Dicatat wedding organizer Anda", pending: "Menunggu diperiksa wedding organizer Anda", verified: "Sesuai", rejected: "Ditolak" },
};

/** "Dibayar oleh" of a vendor or venue (DESAIN §8). */
export const SUPPLIER_PAYER_LABELS: Record<SupplierPayer, string> = {
  WO: "WO",
  Client: "Klien langsung",
};

// Activity entries are written by the server as text. Older ones used English words ("Project baru dibuat",
// "Pembayaran client dicatat", "Timeline vendor ...", raw project statuses); new ones are Indonesian at the source
// (QA EP-15). activityText rewrites the stored wording of old entries when they are shown, longest prefix first.
const LEGACY_ACTIVITY_PREFIXES: [string, string][] = [
  ["Project baru dibuat dari duplikasi project: ", "Proyek baru dibuat dari duplikat proyek: "],
  ["Project baru dibuat: ", "Proyek baru dibuat: "],
  ["Project diarsipkan", "Proyek diarsipkan"],
  ["Project dipulihkan dari arsip", "Proyek dipulihkan dari arsip"],
  ["Project dibatalkan", "Proyek dibatalkan"],
  ["Informasi project diperbarui: ", "Informasi proyek diperbarui: "],
  ["Status project diubah: ", "Status proyek diubah: "],
  ["Timeline project ditambahkan: ", "Tahapan proyek ditambahkan: "],
  ["Timeline project diperbarui: ", "Tahapan proyek diperbarui: "],
  ["Urutan timeline diubah", "Urutan tahapan diubah"],
  ["Vendor ditambahkan ke project", "Vendor ditambahkan ke proyek"],
  ["Timeline vendor ditambahkan: ", "Tahapan vendor ditambahkan: "],
  ["Timeline vendor diperbarui: ", "Tahapan vendor diperbarui: "],
  ["Evidence diunggah: ", "Dokumen diunggah: "],
  ["Evidence dihapus: ", "Dokumen dihapus: "],
  ["Jadwal pembayaran client diperbarui", "Jadwal termin klien diperbarui"],
  ["Verifikasi pembayaran client ", "Verifikasi pembayaran klien "],
  ["Pembayaran client ", "Pembayaran klien "],
];

/** An activity entry's text as a person reads it (see LEGACY_ACTIVITY_PREFIXES). */
export function activityText(description: string): string {
  let text = description;
  for (const [from, to] of LEGACY_ACTIVITY_PREFIXES) {
    if (text.startsWith(from)) {
      text = to + text.slice(from.length);
      break;
    }
  }
  if (text.startsWith("Status proyek diubah: ")) {
    const statuses = PROJECT_STATUS_LABELS.staff as Record<string, string>;
    text = text.replace(/: (\S+) → (\S+)$/, (whole, from: string, to: string) =>
      statuses[from] || statuses[to] ? `: ${statuses[from] ?? from} → ${statuses[to] ?? to}` : whole,
    );
  }
  // Field names inside "...diperbarui: a, b" summaries.
  return text.replace(/\btampil ke client\b/g, "tampil ke klien").replace(/\btimeline vendor\b/g, "tahapan vendor");
}

/** The label for one value, or null when the audience must not see it. */
export function labelFor<T extends string>(labels: Labels<T>, value: T, audience: Audience = "staff"): string | null {
  return audience === "client" ? labels.client[value] : labels.staff[value];
}

/** A voucher's standing today (the API computes it in WIB). Platform Console only. */
export const VOUCHER_STATE_LABELS: Record<"active" | "inactive" | "scheduled" | "expired" | "exhausted", string> = {
  active: "Aktif",
  inactive: "Nonaktif",
  scheduled: "Terjadwal",
  expired: "Kedaluwarsa",
  exhausted: "Habis",
};

/** One use of a voucher: held while its payment is pending, used once paid, released when the payment did not go through. */
export const VOUCHER_REDEMPTION_STATUS_LABELS: Record<"reserved" | "used" | "released", string> = {
  reserved: "Menunggu pembayaran",
  used: "Terpakai",
  released: "Dibatalkan",
};

// ---- Glosarium UI (DESIGN.md §7, D15) --------------------------------------
// One word per concept in everything a person reads. The database enums stay
// English; this is only the vocabulary of screens, buttons and messages.
export const GLOSARIUM = {
  proyek: "Proyek",
  klien: "Klien",
  tim: "Tim",
  termin: "Termin",
  tagihan: "Tagihan",
  bukti: "Bukti",
  dokumen: "Dokumen",
  kendala: "Kendala",
  jadwal: "Jadwal",
  tahapan: "Tahapan",
  vendor: "Vendor",
  venue: "Venue",
  tenant: "Tenant",
} as const;

/**
 * Wording that must not appear in UI text, with what to write instead. Used by
 * the language sweep (PLAN.md Tahap 7) to find strays; `Project` and `Client`
 * stay legal inside identifiers, so it is meant for string literals and JSX text.
 */
export const DISALLOWED_UI_TERMS: { pattern: RegExp; use: string }[] = [
  { pattern: /\bProject(s)?\b/, use: "Proyek" },
  { pattern: /\bClient(s)?\b/, use: "Klien" },
  { pattern: /\bInvoice\b/, use: "Tagihan" },
  { pattern: /\bEvidence\b/, use: "Bukti / Dokumen" },
  { pattern: /\bIssue(s)?\b/, use: "Kendala" },
  { pattern: /\bTimeline\b/, use: "Jadwal" },
  { pattern: /\bMilestone(s)?\b/, use: "Tahapan" },
  { pattern: /\bSubmit\b/, use: "kata kerja + objek, mis. \"Simpan pembayaran\"" },
  { pattern: /\bBills?\b/, use: "Tagihan vendor" },
  { pattern: /\b(Approve|Reject)(d|ed)?\b/, use: "Tandai sesuai / Tolak dengan alasan" },
];

/**
 * What a person reads for each document type. The enum value is what the API stores
 * and stays in English; only this label is shown (DESIGN.md §7).
 */
export const EVIDENCE_TYPE_LABELS: Record<string, string> = {
  Quotation: "Penawaran",
  Invoice: "Tagihan",
  Contract: "Kontrak",
  "Transfer Proof": "Bukti transfer",
  Receipt: "Kwitansi",
  "Purchase Order": "Pesanan pembelian",
  Photo: "Foto",
  Document: "Dokumen",
  Screenshot: "Tangkapan layar",
  "Minutes of Meeting": "Notulen rapat",
  Other: "Lainnya",
  "Booking Proof": "Bukti pemesanan",
};

export function evidenceTypeLabel(type: string): string {
  return EVIDENCE_TYPE_LABELS[type] ?? type;
}


// ---- Penawaran (paket WO Tahap 5) -------------------------------------------
// The stored status plus "Expired", which the API derives for a draft past its validity. Each badge also carries
// its own symbol (DESAIN §6.1): ◷ Draf · ⊘ Kedaluwarsa · ✓ Diterima · ✕ Ditolak · ⇢ Digantikan.
export const QUOTATION_STATUS_LABELS: Record<QuotationDisplayStatus, string> = {
  Draft: "Draf",
  Expired: "Kedaluwarsa",
  Accepted: "Diterima",
  Rejected: "Ditolak",
  Superseded: "Digantikan",
};

/** How the couple said yes to a quotation. */
export const ACCEPT_CHANNEL_LABELS: Record<AcceptChannel, string> = {
  WhatsApp: "WhatsApp",
  InPerson: "Tatap muka",
  Email: "Email",
  Other: "Lainnya",
};

/** What a term's due date counts from. */
export const TERM_DUE_BASIS_LABELS: Record<TermDueBasis, string> = {
  Accepted: "Setelah penawaran diterima",
  Event: "Dari tanggal acara",
};
