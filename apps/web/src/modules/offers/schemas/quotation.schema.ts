import { z } from "zod";
import { eventSchema } from "@/modules/rincian";

const WHATSAPP = /^$|^\+?[0-9][0-9 -]{6,22}$/;

/** "Penawaran baru" and "Ubah data penawaran": who it is for, who handles it, how long it stays valid. */
export const quotationHeaderSchema = z.object({
  coupleNames: z.string().trim().min(1, "Isi nama pasangan").max(150, "Nama pasangan maksimal 150 karakter"),
  contactWhatsapp: z.string().trim().regex(WHATSAPP, "Isi nomor WhatsApp yang benar, mis. 0812 3456 7890").optional().default(""),
  picStaffId: z.string().min(1, "Pilih PIC penawaran"),
  /** "" on create = the workspace's default validity. */
  validUntil: z.string().optional().default(""),
  notes: z.string().max(5000, "Catatan maksimal 5.000 karakter").optional().default(""),
});
export type QuotationHeaderValues = z.infer<typeof quotationHeaderSchema>;

/** A quotation's event may wait for its date (the couple has not picked the day yet). */
export const quotationEventSchema = eventSchema.extend({ eventDate: z.string().optional().default("") });

export const ACCEPT_CHANNEL_OPTIONS = ["WhatsApp", "InPerson", "Email", "Other"] as const;

/** "Tandai diterima": which revision, when, how; a quotation past its validity also gets a new date. */
export const acceptSchema = z
  .object({
    revisionN: z.string().min(1, "Pilih revisi yang diterima"),
    acceptedOn: z.string().min(1, "Pilih tanggal diterima"),
    channel: z.enum(ACCEPT_CHANNEL_OPTIONS, { message: "Pilih lewat apa penawaran diterima" }),
    expired: z.boolean(),
    extendUntil: z.string().optional().default(""),
  })
  .refine((v) => !v.expired || v.extendUntil !== "", { path: ["extendUntil"], message: "Isi tanggal perpanjangan masa berlaku" });
export type AcceptValues = z.infer<typeof acceptSchema>;

export const rejectSchema = z.object({
  reason: z.string().max(500, "Alasan maksimal 500 karakter").optional().default(""),
});

export const extendSchema = z.object({
  validUntil: z.string().min(1, "Pilih tanggal berlaku sampai"),
});

/** A vendor of the quotation (venues come in through its events). */
export const quotationSupplierSchema = z.object({
  vendorId: z.string().min(1, "Pilih vendor dari direktori"),
  paidBy: z.enum(["WO", "Client"]),
  showName: z.boolean(),
  agreedCost: z.union([z.literal(""), z.coerce.number().min(0, "Biaya rekanan tidak boleh negatif")]).optional().default(""),
  notes: z.string().max(2000, "Catatan maksimal 2.000 karakter").optional().default(""),
});
export type QuotationSupplierValues = z.infer<typeof quotationSupplierSchema>;

export const TERM_DUE_BASIS_OPTIONS = ["Accepted", "Event"] as const;

/** One row of "Termin pembayaran": a percentage or a fixed amount, due after acceptance or around an event. */
export const termRowSchema = z.object({
  label: z.string().trim().min(1, "Isi nama termin").max(60, "Nama termin maksimal 60 karakter"),
  mode: z.enum(["percent", "amount"]),
  percent: z.coerce.number().int("Isi persen tanpa koma").min(0).max(100, "Persen paling besar 100").optional().default(0),
  amount: z.coerce.number().min(0, "Jumlah tidak boleh negatif").optional().default(0),
  dueBasis: z.enum(TERM_DUE_BASIS_OPTIONS),
  /** Days, always typed as a positive number; `before` turns it negative for an event-based term. */
  days: z.coerce.number().int("Isi jumlah hari tanpa koma").min(0, "Jumlah hari tidak boleh negatif").max(730, "Paling lama 730 hari"),
  before: z.boolean(),
  dueEventId: z.string().optional().default(""),
});
export const termsSchema = z.object({ terms: z.array(termRowSchema).min(1, "Isi minimal satu termin") });
export type TermRowValues = z.infer<typeof termRowSchema>;

export const PAYMENT_METHOD_OPTIONS = ["Tunai", "Transfer Bank", "QRIS"] as const;

/** "Jadikan proyek": the project's own facts, the events' exact dates, and an optional down payment. */
export const convertSchema = z
  .object({
    name: z.string().trim().min(1, "Isi nama proyek").max(150, "Nama proyek maksimal 150 karakter"),
    brideName: z.string().trim().min(1, "Isi nama mempelai wanita"),
    groomName: z.string().trim().min(1, "Isi nama mempelai pria"),
    eventDates: z.array(z.string()),
    prepStartDate: z.string().min(1, "Pilih tanggal mulai persiapan"),
    picStaffId: z.string().optional().default(""),
    picSalesStaffId: z.string().optional().default(""),
    description: z.string().max(2000, "Catatan maksimal 2.000 karakter").optional().default(""),
    withDownPayment: z.boolean(),
    dpAmount: z.coerce.number().min(0, "Jumlah tidak boleh negatif").optional().default(0),
    dpDate: z.string().optional().default(""),
    dpMethod: z.string().optional().default("Transfer Bank"),
    dpReference: z.string().max(100, "Nomor referensi maksimal 100 karakter").optional().default(""),
    dpNotes: z.string().max(1000, "Catatan maksimal 1.000 karakter").optional().default(""),
  })
  .superRefine((v, ctx) => {
    if (!v.withDownPayment) return;
    if (!(v.dpAmount > 0)) ctx.addIssue({ code: "custom", path: ["dpAmount"], message: "Isi jumlah DP" });
    if (!v.dpDate) ctx.addIssue({ code: "custom", path: ["dpDate"], message: "Pilih tanggal bayar" });
    if (!v.dpMethod) ctx.addIssue({ code: "custom", path: ["dpMethod"], message: "Pilih metode bayar" });
  });
export type ConvertValues = z.infer<typeof convertSchema>;
