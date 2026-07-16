import { z } from "zod";

export const projectMilestoneSchema = z.object({
  name: z.string().min(3, "Nama milestone minimal 3 karakter"),
  targetDate: z.string().min(1, "Target tanggal wajib diisi"),
});

export type ProjectMilestoneFormValues = z.infer<typeof projectMilestoneSchema>;
