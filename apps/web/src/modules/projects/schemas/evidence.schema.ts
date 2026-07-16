import { z } from "zod";

export const EVIDENCE_TYPE_OPTIONS = [
  "Quotation",
  "Invoice",
  "Contract",
  "Transfer Proof",
  "Receipt",
  "Purchase Order",
  "Photo",
  "Document",
  "Screenshot",
  "Minutes of Meeting",
  "Other",
] as const;

export const EVIDENCE_RELATED_KIND_OPTIONS = ["vendorMilestone", "payment", "projectVendor", "issue"] as const;

export const evidenceUploadSchema = z.object({
  name: z.string().min(3, "Nama evidence wajib diisi"),
  type: z.enum(EVIDENCE_TYPE_OPTIONS),
  relatedKind: z.enum(EVIDENCE_RELATED_KIND_OPTIONS),
  relatedId: z.string().min(1, "Konteks terkait wajib dipilih"),
  documentDate: z.string().min(1, "Tanggal dokumen wajib diisi"),
  description: z.string().optional().default(""),
});

export type EvidenceUploadFormValues = z.infer<typeof evidenceUploadSchema>;
