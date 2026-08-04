import { z } from "zod";

export const ISSUE_IMPACT_OPTIONS = ["Low", "Medium", "High", "Critical"] as const;
export const ISSUE_STATUS_OPTIONS = ["Open", "In Review", "In Resolution", "Resolved", "Closed"] as const;

export const issueSchema = z.object({
  projectVendorId: z.string().min(1, "Vendor wajib dipilih"),
  // Empty string = "Umum" (general to the vendor, not tied to one milestone).
  vendorMilestoneId: z.string().optional().default(""),
  title: z.string().min(3, "Judul kendala wajib diisi"),
  description: z.string().min(3, "Deskripsi wajib diisi"),
  impact: z.enum(ISSUE_IMPACT_OPTIONS),
  foundDate: z.string().min(1, "Tanggal ditemukan wajib diisi"),
  resolutionPlan: z.string().optional().default(""),
  picStaffId: z.string().min(1, "PIC wajib dipilih"),
  targetResolutionDate: z.string().optional().default(""),
  // Not shown on the create form (always starts "Open" server-side) --
  // editable only from the edit form.
  status: z.enum(ISSUE_STATUS_OPTIONS).optional().default("Open"),
});

export type IssueFormValues = z.infer<typeof issueSchema>;
