// One place for every status label shown to people (FLOW-IDEAL.md §4,
// PLAN.md Fase 3b). The database keeps English enum values; staff and
// clients each read their own wording. `null` for a client means "do not
// show this to the client" (internal detail).
import type { EngagementStatus, IssueImpact, IssueStatus, MilestoneStatus, PaymentTermStatus, ProjectStatus, VendorPricingTier } from "@/modules/projects/types";

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
    overdue: "Lewat jatuh tempo — hubungi PIC",
    partial: "Dibayar sebagian",
    due_soon: "Jatuh tempo segera",
    upcoming: "Menunggu pembayaran",
  },
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

