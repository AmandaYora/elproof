import { z } from "zod";

export const vendorMilestoneSchema = z.object({
  name: z.string().min(3, "Nama milestone minimal 3 karakter"),
  description: z.string().optional().default(""),
  targetDate: z.string().min(1, "Target tanggal wajib diisi"),
  picStaffId: z.string().min(1, "PIC wajib dipilih"),
});

export type VendorMilestoneFormValues = z.infer<typeof vendorMilestoneSchema>;
