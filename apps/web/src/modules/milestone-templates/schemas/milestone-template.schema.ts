import { z } from "zod";

export const milestoneTemplateSchema = z.object({
  name: z.string().min(3, "Nama timeline minimal 3 karakter"),
  daysBeforeEvent: z.number().int("Harus berupa angka bulat").min(0, "Tidak boleh negatif"),
});

export type MilestoneTemplateFormValues = z.infer<typeof milestoneTemplateSchema>;
