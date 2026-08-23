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
  "Booking Proof",
] as const;

export const EVIDENCE_RELATED_KIND_OPTIONS = [
  "vendorMilestone",
  "payment",
  "projectVendor",
  "issue",
  "clientPayment",
  "venuePayment",
  "general",
  "projectMilestone",
] as const;

export const evidenceUploadSchema = z
  .object({
    name: z.string().min(3, "Nama evidence wajib diisi"),
    type: z.enum(EVIDENCE_TYPE_OPTIONS),
    relatedKind: z.enum(EVIDENCE_RELATED_KIND_OPTIONS),
    relatedId: z.string().optional().default(""),
    documentDate: z.string().min(1, "Tanggal dokumen wajib diisi"),
    description: z.string().optional().default(""),
    isClientVisible: z.boolean().optional().default(false),
  })
  .superRefine((values, ctx) => {
    if (values.relatedKind !== "general" && values.relatedId.trim().length === 0) {
      ctx.addIssue({
        code: "custom",
        message: "Konteks terkait wajib dipilih",
        path: ["relatedId"],
      });
    }
  });

export type EvidenceUploadFormValues = z.infer<typeof evidenceUploadSchema>;
