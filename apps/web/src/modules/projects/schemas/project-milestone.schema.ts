import { z } from "zod";

export const projectMilestoneSchema = z.object({
  name: z.string().min(3, "Isi nama timeline, minimal 3 karakter"),
  targetDate: z.string().min(1, "Pilih target tanggal"),
});

export type ProjectMilestoneFormValues = z.infer<typeof projectMilestoneSchema>;

export const MILESTONE_STATUS_OPTIONS = ["Not Started", "In Progress", "Completed", "Blocked", "Cancelled"] as const;

export const projectMilestoneEditSchema = z.object({
  status: z.enum(MILESTONE_STATUS_OPTIONS),
  targetDate: z.string().min(1, "Pilih target tanggal"),
  completedDate: z.string().default(""),
});

export type ProjectMilestoneEditValues = z.infer<typeof projectMilestoneEditSchema>;
