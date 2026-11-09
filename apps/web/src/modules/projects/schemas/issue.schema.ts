import { z } from "zod";

export const ISSUE_IMPACT_OPTIONS = ["Low", "Medium", "High", "Critical"] as const;
export const ISSUE_STATUS_OPTIONS = ["Open", "In Review", "In Resolution", "Resolved", "Closed"] as const;

export const issueSchema = z.object({
  projectVendorId: z.string().min(1, "Pilih vendor"),
  // Empty string = "Umum" (general to the vendor, not tied to one milestone).
  vendorMilestoneId: z.string().optional().default(""),
  title: z.string().min(3, "Isi judul kendala, minimal 3 karakter"),
  description: z.string().min(3, "Isi deskripsi kendala, minimal 3 karakter"),
  impact: z.enum(ISSUE_IMPACT_OPTIONS),
  foundDate: z.string().min(1, "Pilih tanggal ditemukan"),
  resolutionPlan: z.string().optional().default(""),
  picStaffId: z.string().min(1, "Pilih PIC"),
  targetResolutionDate: z.string().optional().default(""),
  // Not shown on the create form (always starts "Open" server-side) --
  // editable only from the edit form.
  status: z.enum(ISSUE_STATUS_OPTIONS).optional().default("Open"),
});

export type IssueFormValues = z.infer<typeof issueSchema>;
