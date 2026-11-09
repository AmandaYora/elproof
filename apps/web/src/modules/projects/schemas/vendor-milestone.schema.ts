import { z } from "zod";

export const vendorMilestoneSchema = z.object({
  name: z.string().min(3, "Isi nama timeline, minimal 3 karakter"),
  description: z.string().optional().default(""),
  targetDate: z.string().min(1, "Pilih target tanggal"),
  picStaffId: z.string().min(1, "Pilih PIC"),
});

export type VendorMilestoneFormValues = z.infer<typeof vendorMilestoneSchema>;

export const vendorMilestoneEditSchema = z.object({
  status: z.enum(["Not Started", "In Progress", "Completed", "Blocked", "Cancelled"]),
  targetDate: z.string().min(1, "Pilih target tanggal"),
  completedDate: z.string().default(""),
  picStaffId: z.string().default(""),
  description: z.string().default(""),
  notes: z.string().default(""),
});
